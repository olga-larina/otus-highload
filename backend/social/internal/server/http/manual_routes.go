package internalhttp

import (
	"context"
	"net/http"
	"sync"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	pkg_http "github.com/olga-larina/otus-highload/pkg/http/server"
	"github.com/olga-larina/otus-highload/pkg/logger"
	"github.com/olga-larina/otus-highload/social/internal/model"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	CACHE_INVALIDATE_ROUTE = "/internal/cache/invalidate"
	POST_FEED_ROUTE        = "/post/feed/posted"
)

// WebSocketUpgrader обновляет HTTP соединение до WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// принимаем любые запросы
		return true
	},
}

type PostNotifier interface {
	NotifyInvalidateAll(ctx context.Context) error
}

type PostFeedSubsriber interface {
	SubscribePostFeed(ctx context.Context, userId model.UserId) (string, <-chan []model.Post, error)
	UnsubscribePostFeed(ctx context.Context, userId model.UserId, subscriptionId string) error
}

func NewManualRouter(
	postNotifier PostNotifier,
	postFeedSubscriber PostFeedSubsriber,
	authFunc openapi3filter.AuthenticationFunc,
	authService AuthService,
) *mux.Router {
	router := mux.NewRouter()

	// роут с метриками
	router.Handle(pkg_http.METRICS_ROUTE, promhttp.Handler()).Methods("GET")

	// роут с инвалидацией кеша
	router.HandleFunc(CACHE_INVALIDATE_ROUTE, cacheInvalidateHandler(postNotifier)).Methods("POST")

	// роут с веб-сокетом для ленты постов
	router.HandleFunc(POST_FEED_ROUTE, postFeedWebSocketHandler(postFeedSubscriber, authFunc, authService))

	return router
}

func cacheInvalidateHandler(postNotifier PostNotifier) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		err := postNotifier.NotifyInvalidateAll(ctx)
		if err != nil {
			logger.Error(ctx, err, "failed notifying cache invalidation")
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			logger.Info(ctx, "succeeded notifying cache invalidation")
			w.WriteHeader(http.StatusOK)
		}
	}
}

func postFeedWebSocketHandler(
	postFeedSubscriber PostFeedSubsriber,
	authFunc openapi3filter.AuthenticationFunc,
	authService AuthService,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		// авторизуемся
		r, err = authenticate(authFunc, r)
		if err != nil {
			http.Error(w, "not authorized", http.StatusUnauthorized)
			return
		}

		// создаём контекст с отменой, чтобы отменять подписчиков очереди
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		userId, err := authService.GetUserId(ctx)
		if err != nil {
			http.Error(w, "not authorized", http.StatusUnauthorized)
			return
		}

		// обновляем соединение до WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "not valid connection", http.StatusBadRequest)
			return
		}
		defer conn.Close()

		done := make(chan struct{})
		// для безопасного закрытия канала
		var onceDone sync.Once

		// обработчик закрытия соединения
		conn.SetCloseHandler(func(code int, text string) error {
			logger.Info(ctx, "websocket connection was closed", "code", code, "message", text)
			onceDone.Do(func() { close(done) })
			return nil
		})

		// читаем сообщения от клиента
		go func() {
			defer onceDone.Do(func() { close(done) })

			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
						logger.Error(ctx, err, "websocket closed")
					} else {
						logger.Error(ctx, err, "websocket read error")
					}
					break
				}
			}
		}()

		// подписка
		subscriptionId, posts, err := postFeedSubscriber.SubscribePostFeed(ctx, userId)
		if err != nil {
			conn.WriteMessage(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, err.Error()),
			)
			return
		}
		defer postFeedSubscriber.UnsubscribePostFeed(ctx, userId, subscriptionId)

		for {
			select {
			case <-done:
				conn.WriteMessage(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, "connection closed"),
				)
				logger.Debug(ctx, "request done")
				return
			case postFeed, ok := <-posts:
				if !ok {
					conn.WriteMessage(
						websocket.CloseMessage,
						websocket.FormatCloseMessage(websocket.CloseNormalClosure, "connection closed"),
					)
					logger.Debug(ctx, "posts channel done")
					return
				}
				writeResponse(ctx, conn, postFeed)
			}
		}
	}
}

func authenticate(authFunc openapi3filter.AuthenticationFunc, r *http.Request) (*http.Request, error) {
	input := &openapi3filter.AuthenticationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{Request: r},
		SecuritySchemeName:     pkg_http.SECURITY_SCHEME,
		Scopes:                 make([]string, 0),
	}
	err := authFunc(r.Context(), input)
	if err != nil {
		return r, err
	}
	return input.RequestValidationInput.Request, nil
}

func writeResponse(ctx context.Context, conn *websocket.Conn, resp any) {
	if err := conn.WriteJSON(resp); err != nil {
		logger.Error(ctx, err, "error writing response")
	}

	logger.Debug(ctx, "response sent", "response", resp)
}
