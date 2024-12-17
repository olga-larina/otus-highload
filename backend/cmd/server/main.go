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
	"github.com/redis/go-redis/v9"

	"github.com/olga-larina/otus-highload/backend/internal/logger"
	"github.com/olga-larina/otus-highload/backend/internal/queue/rabbit"
	internalhttp "github.com/olga-larina/otus-highload/backend/internal/server/http"
	"github.com/olga-larina/otus-highload/backend/internal/service"
	"github.com/olga-larina/otus-highload/backend/internal/service/auth"
	cacher "github.com/olga-larina/otus-highload/backend/internal/service/cache"
	"github.com/olga-larina/otus-highload/backend/internal/service/cache/converter"
	redis_cache "github.com/olga-larina/otus-highload/backend/internal/service/cache/redis"
	"github.com/olga-larina/otus-highload/backend/internal/service/feed"
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
	friendStorage := sqlstorage.NewFriendStorage(db)
	postStorage := sqlstorage.NewPostStorage(db)
	postFeedStorage := sqlstorage.NewPostFeedStorage(db)

	// queue
	queue := rabbit.NewQueue(
		config.Queue.URI,
		config.Queue.ExchangeName,
		config.Queue.ExchangeType,
		config.Queue.QueueName,
		config.Queue.RoutingKey,
	)
	consumer := queue.NewConsumer(config.Queue.ConsumerTag)
	if err := queue.Start(ctx); err != nil {
		logger.Error(ctx, err, "queue failed to start")
		return
	}
	publisher := queue.NewPublisher()
	if err := queue.Start(ctx); err != nil {
		logger.Error(ctx, err, "queue failed to start")
		return
	}

	// subscribers cache
	subscriberRedisOpts, err := redis.ParseURL(config.Cache.SubscribersCache.URI)
	if err != nil {
		logger.Error(ctx, err, "failed to parse subscribers cache url")
		return
	}
	subscribersConverter := converter.NewSubscriberStringConverter()
	subscriberRedisCache, err := redis_cache.NewRedisCache(subscriberRedisOpts, config.Cache.SubscribersCache.Ttl, subscribersConverter, "subscribers")
	if err != nil {
		logger.Error(ctx, err, "failed to create subscribers cache")
		return
	}
	subscriberCacher := cacher.NewSubscriberCacher(subscriberRedisCache, friendStorage)

	// post feed cache
	postFeedRedisOpts, err := redis.ParseURL(config.Cache.PostFeedCache.URI)
	if err != nil {
		logger.Error(ctx, err, "failed to parse post feed cache url")
		return
	}
	postFeedConverter := converter.NewPostFeedStringConverter()
	postFeedRedisCache, err := redis_cache.NewRedisCache(postFeedRedisOpts, config.Cache.PostFeedCache.Ttl, postFeedConverter, "postFeed")
	if err != nil {
		logger.Error(ctx, err, "failed to create post feed cache")
		return
	}
	postFeedCacher := cacher.NewPostFeedCacher(postFeedRedisCache, postFeedStorage, config.PostFeed.MaxSize)

	// post feed updater
	postFeedUpdater := feed.NewPostFeedUpdater(
		postFeedCacher,
		subscriberCacher,
		consumer,
	)

	if err := postFeedUpdater.Start(ctx); err != nil {
		logger.Error(ctx, err, "postFeedUpdater failed to start")
		return
	}

	// post feed notifications
	postFeedNotificationService := feed.NewPostFeedNotificationService(publisher)

	// services
	authenticator, err := auth.NewFakeAuthenticator(config.Auth.PrivateKey)
	if err != nil {
		logger.Error(ctx, err, "failed to create authenticator")
		return
	}
	authService := auth.NewAuthService()
	passwordService := service.NewPasswordService()
	loginService := service.NewLoginService(loginStorage, passwordService, authenticator)
	userService := service.NewUserService(userStorage, passwordService)
	friendService := service.NewFriendService(friendStorage, postFeedNotificationService)
	postService := service.NewPostService(postStorage, postFeedNotificationService)
	postFeedService := feed.NewPostFeedService(postFeedCacher, config.PostFeed.MaxSize)

	// http server
	httpServerAddr := fmt.Sprintf("%s:%s", config.HTTPServer.Host, config.HTTPServer.Port)
	server := internalhttp.NewServer(
		authService,
		loginService,
		userService,
		friendService,
		postService,
		postFeedService,
	)
	serverHandler := internalhttp.NewStrictHandler(
		server,
		[]internalhttp.StrictMiddlewareFunc{},
		// []internalhttp.StrictMiddlewareFunc{internalhttp.StrictLoggingMiddleware},
	)

	router := mux.NewRouter()
	// роут с метриками
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")
	// роут с инвалидацией кеша
	router.HandleFunc("/internal/cache/invalidate", func() func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			err := postFeedNotificationService.NotifyInvalidateAll(ctx)
			if err != nil {
				logger.Error(ctx, err, "failed notifying cache invalidation")
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				logger.Info(ctx, "succeeded notifying cache invalidation")
				w.WriteHeader(http.StatusOK)
			}
		}
	}()).Methods("POST")

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
		return internalhttp.SkipValidatorForMetricsAndInternal(validator, next)
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

	if err := consumer.Stop(ctx); err != nil {
		logger.Error(ctx, err, "failed to stop consumer")
	}

	if err := postFeedUpdater.Stop(ctx); err != nil {
		logger.Error(ctx, err, "failed to stop postFeedUpdater")
	}

	if err := queue.Stop(ctx); err != nil {
		logger.Error(ctx, err, "failed to stop queue")
	}
}

func parseDbConfig(uri string, cfg DatabaseConnectConfig) *sqlstorage.DbConfig {
	return &sqlstorage.DbConfig{
		Uri:             uri,
		MaxConns:        cfg.MaxConns,
		MaxConnLifetime: cfg.MaxConnLifetime,
		MaxConnIdleTime: cfg.MaxConnIdleTime,
	}
}
