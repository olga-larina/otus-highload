package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/olga-larina/otus-highload/pkg/logger"
	"github.com/olga-larina/otus-highload/pkg/queue/rabbit"
	"github.com/olga-larina/otus-highload/pkg/tracing"
	"github.com/olga-larina/otus-highload/verifier/internal/service"
	"github.com/pckilgore/combuuid"
)

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "/etc/backend/server/config.yaml", "Path to configuration file")
}

func main() {
	serviceId := combuuid.NewUuid().String()

	flag.Parse()

	config, err := NewConfig(configFile)
	if err != nil {
		log.Fatalf("failed reading config %v", err)
		return
	}

	location, err := time.LoadLocation(config.Timezone)
	if err != nil {
		log.Fatalf("failed loading location %v", err)
		return
	}
	time.Local = location

	err = logger.New(config.Logger.Level)
	if err != nil {
		log.Fatalf("failed building logger %v", err)
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

	// tracer
	err = tracing.InitTracer(ctx)
	if err != nil {
		log.Fatalf("failed initialting tracer %v", err)
		return
	}

	// user created queue
	userCreatedQueue := rabbit.NewQueue(
		config.UserCreatedQueue.URI,
		config.UserCreatedQueue.ExchangeName,
		config.UserCreatedQueue.ExchangeType,
	)
	if err = userCreatedQueue.Start(ctx); err != nil {
		logger.Error(ctx, err, "user created queue failed to start")
		return
	}

	// verifier status queue
	verifierStatusQueue := rabbit.NewQueue(
		config.VerifierStatusQueue.URI,
		config.VerifierStatusQueue.ExchangeName,
		config.VerifierStatusQueue.ExchangeType,
	)
	if err = verifierStatusQueue.Start(ctx); err != nil {
		logger.Error(ctx, err, "verifier status queue failed to start")
		return
	}

	// verifier processor service
	verifierProcessorService := service.NewVerifierProcessorService(
		verifierStatusQueue,
		config.VerifierStatusQueue.RoutingKey,
		serviceId,
	)

	if err := verifierProcessorService.Start(ctx); err != nil {
		logger.Error(ctx, err, "verifierProcessorService failed to start")
		return
	}

	// verifier service
	verifierService := service.NewVerifierService(
		userCreatedQueue,
		config.UserCreatedQueue.QueueName,
		config.UserCreatedQueue.ConsumerTag,
		config.UserCreatedQueue.RoutingKey,
		verifierProcessorService,
		serviceId,
	)

	if err := verifierService.Start(ctx); err != nil {
		logger.Error(ctx, err, "verifierService failed to start")
		return
	}

	logger.Info(ctx, "server is running...")

	<-ctx.Done()

	if err := verifierService.Stop(ctx); err != nil {
		logger.Error(ctx, err, "failed to stop verifier service")
	}

	if err := verifierProcessorService.Stop(ctx); err != nil {
		logger.Error(ctx, err, "failed to stop verifier processor service")
	}

	if err := userCreatedQueue.Stop(ctx); err != nil {
		logger.Error(ctx, err, "failed to stop user created queue")
	}

	if err := verifierStatusQueue.Stop(ctx); err != nil {
		logger.Error(ctx, err, "failed to stop verifier status queue")
	}
}
