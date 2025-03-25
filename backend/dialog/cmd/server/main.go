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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	tarantool "github.com/tarantool/go-tarantool/v2"

	internalhttp "github.com/olga-larina/otus-highload/dialog/internal/server/http"
	"github.com/olga-larina/otus-highload/dialog/internal/service"
	"github.com/olga-larina/otus-highload/dialog/internal/service/shard"
	"github.com/olga-larina/otus-highload/dialog/internal/storage/memory"
	sqlstorage "github.com/olga-larina/otus-highload/dialog/internal/storage/sql"
	pkg_http "github.com/olga-larina/otus-highload/pkg/http/server"
	"github.com/olga-larina/otus-highload/pkg/logger"
	"github.com/olga-larina/otus-highload/pkg/service/auth"
	pkg_sqlstorage "github.com/olga-larina/otus-highload/pkg/storage/sql"
	"github.com/olga-larina/otus-highload/pkg/tracing"
	"github.com/olga-larina/otus-highload/pkg/zabbix"
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

	// tracer
	err = tracing.InitTracer(ctx)
	if err != nil {
		log.Fatalf("failed initialting tracer %v", err)
		return
	}

	// storages
	var dialogStorage service.DialogStorage
	// dialogue storage
	if config.Dialogue.DbType == "SQL" {
		// sql database
		var storageReplicaConfigs []*pkg_sqlstorage.DbConfig
		if len(config.Database.Replicas.URI) > 0 {
			replicasUri := strings.Split(config.Database.Replicas.URI, ",")
			storageReplicaConfigs = make([]*pkg_sqlstorage.DbConfig, len(replicasUri))
			for i, replicaUri := range replicasUri {
				storageReplicaConfigs[i] = parseDbConfig(replicaUri, config.Database.Replicas.ConnectParams)
			}
		}
		db, err := pkg_sqlstorage.NewReplicatedDb(ctx, parseDbConfig(config.Database.Master.URI, config.Database.Master.ConnectParams), storageReplicaConfigs)
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
		dialogStorage = sqlstorage.NewDialogStorage(db)
	} else {
		// tarantool
		tarantoolDialer := tarantool.NetDialer{
			Address:  config.InMemoryDatabase.URI,
			User:     config.InMemoryDatabase.User,
			Password: config.InMemoryDatabase.Password,
		}
		opts := tarantool.Opts{
			Concurrency: uint32(config.InMemoryDatabase.Concurrency),
		}
		tarantoolConn, err := tarantool.Connect(ctx, tarantoolDialer, opts)
		if err != nil {
			logger.Error(ctx, err, "failed to connect to tarantool")
			return
		}
		defer tarantoolConn.Close()

		dialogStorage = memory.NewDialogStorage(tarantoolConn)
	}

	// dialog id obtainer
	dialogIdObtainer := shard.NewDialogIdObtainer()

	// services
	authenticator, err := auth.NewFakeAuthenticator(config.Auth.PrivateKey)
	if err != nil {
		logger.Error(ctx, err, "failed to create authenticator")
		return
	}
	authService := auth.NewAuthService()
	dialogService := service.NewDialogService(dialogStorage, dialogIdObtainer)

	// zabbix observer
	zabbixObserver := zabbix.NewZabbixObserver(config.Zabbix.Host, config.Zabbix.Port, config.Zabbix.Period, config.Zabbix.Name)
	if err := zabbixObserver.Start(ctx); err != nil {
		logger.Error(ctx, err, "zabbixObserver failed to start")
		return
	}

	// authentication function
	authFunc := pkg_http.NewAuthenticator(authenticator, authenticator)

	// http server
	httpServerAddr := fmt.Sprintf("%s:%s", config.HTTPServer.Host, config.HTTPServer.Port)
	server := internalhttp.NewServer(
		authService,
		dialogService,
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
				AuthenticationFunc: authFunc,
			},
		})
	router.Use(pkg_http.TracingMiddleware, pkg_http.MetricsMiddleware, pkg_http.LoggingMiddleware)
	router.Use(func(next http.Handler) http.Handler {
		return pkg_http.SkipValidatorForManualRoutes(validator, next, []string{pkg_http.METRICS_ROUTE})
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

	if err := zabbixObserver.Stop(ctx); err != nil {
		logger.Error(ctx, err, "failed to stop zabbixObserver")
	}
}

func parseDbConfig(uri string, cfg DatabaseConnectConfig) *pkg_sqlstorage.DbConfig {
	return &pkg_sqlstorage.DbConfig{
		Uri:             uri,
		MaxConns:        cfg.MaxConns,
		MaxConnLifetime: cfg.MaxConnLifetime,
		MaxConnIdleTime: cfg.MaxConnIdleTime,
	}
}
