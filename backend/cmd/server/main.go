package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gorilla/mux"
	middleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/olga-larina/otus-highload/backend/internal/logger"
	internalhttp "github.com/olga-larina/otus-highload/backend/internal/server/http"
	"github.com/olga-larina/otus-highload/backend/internal/service"
	"github.com/olga-larina/otus-highload/backend/internal/service/auth"
	sqlstorage "github.com/olga-larina/otus-highload/backend/internal/storage/sql"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
)

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "/etc/backend/server/config.yaml", "Path to configuration file")
}

func main() {
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

	// storages
	var storageReplicaConfigs []*sqlstorage.DbConfig
	if len(config.Database.Replicas.URI) > 0 {
		replicasUri := strings.Split(config.Database.Replicas.URI, ",")
		storageReplicaConfigs = make([]*sqlstorage.DbConfig, len(replicasUri))
		for i, replicaUri := range replicasUri {
			storageReplicaConfigs[i] = parseDbConfig(replicaUri, config.Database.Replicas.ConnectParams)
		}
	}
	db, err := sqlstorage.NewReplicatedDb(ctx, parseDbConfig(config.Database.Master.URI, config.Database.Master.ConnectParams), storageReplicaConfigs)
	if err != nil {
		logger.Error(ctx, err, "failed to create db")
		return
	}
	if err := db.Connect(ctx); err != nil {
		logger.Error(ctx, err, "failed to connect to db")
		return
	}
	defer func() {
		if err := db.Close(ctx); err != nil {
			logger.Error(ctx, err, "failed to close sql storage")
		}
	}()
	userStorage := sqlstorage.NewUserStorage(db)
	loginStorage := sqlstorage.NewLoginStorage(db)

	// services
	authenticator, err := auth.NewFakeAuthenticator(config.Auth.PrivateKey)
	if err != nil {
		logger.Error(ctx, err, "failed to create authenticator")
		return
	}
	passwordService := service.NewPasswordService()
	loginService := service.NewLoginService(loginStorage, passwordService, authenticator)
	userService := service.NewUserService(userStorage, passwordService)

	// http server
	httpServerAddr := fmt.Sprintf("%s:%s", config.HTTPServer.Host, config.HTTPServer.Port)
	server := internalhttp.NewServer(
		loginService,
		userService,
	)
	serverHandler := internalhttp.NewStrictHandler(
		server,
		[]internalhttp.StrictMiddlewareFunc{},
		// []internalhttp.StrictMiddlewareFunc{internalhttp.StrictLoggingMiddleware},
	)

	router := mux.NewRouter()
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	internalhttp.HandlerWithOptions(
		serverHandler,
		internalhttp.GorillaServerOptions{
			BaseRouter: router,
			// Middlewares: []internalhttp.MiddlewareFunc{internalhttp.LoggingMiddleware},
			ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
				logger.Error(ctx, err, "error in request")
				http.Error(w, err.Error(), http.StatusBadRequest)
			},
		},
	)

	spec, err := internalhttp.GetSwagger()
	if err != nil {
		logger.Error(ctx, err, "failed to load swagger spec")
		return
	}
	spec.Servers = nil

	validator := middleware.OapiRequestValidatorWithOptions(spec,
		&middleware.Options{
			Options: openapi3filter.Options{
				AuthenticationFunc: internalhttp.NewAuthenticator(authenticator, authenticator),
			},
		})
	router.Use(internalhttp.LoggingMiddleware)
	router.Use(func(next http.Handler) http.Handler {
		return internalhttp.SkipValidatorForMetrics(validator, next)
	})

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
	})
	httpServer := &http.Server{
		Handler: c.Handler(router),
		Addr:    httpServerAddr,
	}

	go func() {
		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			logger.Error(ctx, err, "failed to stop http server")
		}
	}()

	logger.Info(ctx, "server is running...")

	if err := httpServer.ListenAndServe(); err != nil {
		logger.Error(ctx, err, "http server stopped")
		cancel()
	}

	<-ctx.Done()
}

func parseDbConfig(uri string, cfg DatabaseConnectConfig) *sqlstorage.DbConfig {
	return &sqlstorage.DbConfig{
		Uri:             uri,
		MaxConns:        cfg.MaxConns,
		MaxConnLifetime: cfg.MaxConnLifetime,
		MaxConnIdleTime: cfg.MaxConnIdleTime,
	}
}
