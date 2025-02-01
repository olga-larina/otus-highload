package feed

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/olga-larina/otus-highload/pkg/logger"
	"github.com/olga-larina/otus-highload/social/internal/model"
	"github.com/olga-larina/otus-highload/social/internal/queue"
	"go.opentelemetry.io/otel"
)

type PostFeedUpdater struct {
	postFeedCache        PostFeedCacher
	subscribersCache     SubscribersCacher
	updatesConsumer      queue.QueueConsumer // получение событий об обновлении ленты (для обновления кеша)
	userPublisher        queue.QueueSender   // рассылка событий об обновлениях лент в каналы пользователей
	userUpdateRoutingKey string              // шаблон routingKey для рассылки событий об обновлениях лент
	done                 chan struct{}
}

type PostFeedCacher interface {
	InvalidateAll(ctx context.Context) error
	Load(ctx context.Context, userId *model.UserId) ([]model.PostExtended, error)
}

type SubscribersCacher interface {
	InvalidateAll(ctx context.Context) error
	Load(ctx context.Context, userId *model.UserId) ([]model.UserId, error)
	GetOrLoad(ctx context.Context, userId *model.UserId) ([]model.UserId, error)
}

func NewPostFeedUpdater(
	postFeedCache PostFeedCacher,
	subscribersCache SubscribersCacher,
	queue queue.Queue,
	postFeedCacheQueueName string,
	postFeedCacheConsumerTag string,
	postFeedCacheConsumerRoutingKey string,
	postFeedUserUpdateRoutingKey string,
	serviceId string,
) *PostFeedUpdater {
	return &PostFeedUpdater{
		postFeedCache:    postFeedCache,
		subscribersCache: subscribersCache,
		updatesConsumer: queue.NewConsumer(
			postFeedCacheQueueName,
			fmt.Sprintf(postFeedCacheConsumerTag, serviceId),
			postFeedCacheConsumerRoutingKey,
		),
		userPublisher:        queue.NewPublisher(),
		userUpdateRoutingKey: postFeedUserUpdateRoutingKey,
		done:                 make(chan struct{}),
	}
}

func (s *PostFeedUpdater) Start(ctx context.Context) error {
	logger.Info(ctx, "starting postFeedUpdater")

	err := s.updatesConsumer.Start(ctx)
	if err != nil {
		return err
	}

	err = s.userPublisher.Start(ctx)
	if err != nil {
		return err
	}

	err = s.processEvents(ctx)
	if err != nil {
		return err
	}

	logger.Info(ctx, "started postFeedUpdater")
	return nil
}

func (s *PostFeedUpdater) Stop(ctx context.Context) error {
	logger.Info(ctx, "stopping postFeedUpdater")

	err := s.userPublisher.Stop(ctx)
	if err != nil {
		logger.Error(ctx, err, "failed to stop postFeedUpdater publisher")
	}

	err = s.updatesConsumer.Stop(ctx)
	if err != nil {
		logger.Error(ctx, err, "failed to stop postFeedUpdater consumer")
	}

	<-ctx.Done()
	<-s.done

	logger.Info(ctx, "stopped postFeedUpdater")
	return nil
}

/*
 * Получение событий из очереди.
 */
func (s *PostFeedUpdater) processEvents(pctx context.Context) error {
	data, err := s.updatesConsumer.ReceiveData(pctx)
	if err != nil {
		defer close(s.done)
		return err
	}

	go func() {
		defer close(s.done)

		for msg := range data {
			ctxInitial := queue.GetContext(pctx, msg)

			ctx, span := otel.Tracer("default").Start(ctxInitial, "consume queue "+"postFeed")

			var notification model.NotificationJson
			err := json.Unmarshal(msg.Body, &notification)
			if err != nil {
				logger.Error(ctx, err, "failed to read notification")
			} else {
				logger.Info(ctx, "received notification", "notification", notification)

				switch notification.Type {
				case model.InvalidateAllNotificationType:
					err = s.processInvalidateAll(ctx)
				case model.AddFriendNotificationType:
					var payload model.AddFriendNotification
					err = json.Unmarshal([]byte(notification.Payload), &payload)
					if err == nil {
						err = s.processAddFriend(ctx, payload)
					}
				case model.DeleteFriendNotificationType:
					var payload model.DeleteFriendNotification
					err = json.Unmarshal([]byte(notification.Payload), &payload)
					if err == nil {
						err = s.processDeleteFriend(ctx, payload)
					}
				case model.CreatePostNotificationType:
					var payload model.CreatePostNotification
					err = json.Unmarshal([]byte(notification.Payload), &payload)
					if err == nil {
						err = s.processCreatePost(ctx, payload)
					}
				case model.UpdatePostNotificationType:
					var payload model.UpdatePostNotification
					err = json.Unmarshal([]byte(notification.Payload), &payload)
					if err == nil {
						err = s.processUpdatePost(ctx, payload)
					}
				case model.DeletePostNotificationType:
					var payload model.DeletePostNotification
					err = json.Unmarshal([]byte(notification.Payload), &payload)
					if err == nil {
						err = s.processDeletePost(ctx, payload)
					}
				}
				if err != nil {
					logger.Error(ctx, err, "failed processing notification", "notification", notification)
				}
			}

			span.End()
		}
	}()

	return nil
}

func (s *PostFeedUpdater) processInvalidateAll(ctx context.Context) error {
	if err := s.subscribersCache.InvalidateAll(ctx); err != nil {
		return err
	}
	return s.postFeedCache.InvalidateAll(ctx)
}

func (s *PostFeedUpdater) processAddFriend(ctx context.Context, notification model.AddFriendNotification) error {
	return s.processUpdateFriend(ctx, notification.UserId, notification.FriendId)
}

func (s *PostFeedUpdater) processDeleteFriend(ctx context.Context, notification model.DeleteFriendNotification) error {
	return s.processUpdateFriend(ctx, notification.UserId, notification.FriendId)
}

func (s *PostFeedUpdater) processUpdateFriend(ctx context.Context, userId model.UserId, friendId model.UserId) error {
	// обновляем подписчиков друга (т.к. изменились пользователи, которые на него подписаны)
	_, err := s.subscribersCache.Load(ctx, &friendId)
	if err != nil {
		logger.Error(ctx, err, "failed loading subscribers", "userId", friendId)
		return err
	}
	// перезагружаем ленту пользователя, т.к. у него добавился новый друг, за постами которого он следит
	posts, err := s.postFeedCache.Load(ctx, &userId)
	if err != nil {
		logger.Error(ctx, err, "failed loading postFeed", "userId", userId)
		return err
	}
	s.sendPostFeedUpdate(ctx, userId, posts)
	return err
}

func (s *PostFeedUpdater) processCreatePost(ctx context.Context, notification model.CreatePostNotification) error {
	return s.processUpdateFeed(ctx, notification.UserId)
}

func (s *PostFeedUpdater) processUpdatePost(ctx context.Context, notification model.UpdatePostNotification) error {
	return s.processUpdateFeed(ctx, notification.UserId)
}

func (s *PostFeedUpdater) processDeletePost(ctx context.Context, notification model.DeletePostNotification) error {
	return s.processUpdateFeed(ctx, notification.UserId)
}

func (s *PostFeedUpdater) processUpdateFeed(ctx context.Context, userId model.UserId) error {
	// получаем подписчиков текущего пользователя
	subscribers, err := s.subscribersCache.GetOrLoad(ctx, &userId)
	if err != nil {
		logger.Error(ctx, err, "failed loading subscribers", "userId", userId)
		return err
	}
	// у каждого подписчика обновляем ленту; если у кого-то не получилось, то не прекращаем
	// также пытаемся отправить обновление ленты в канал каждого пользователя
	var errSubscriber error
	var posts []model.PostExtended
	for _, subscriberId := range subscribers {
		posts, errSubscriber = s.postFeedCache.Load(ctx, &subscriberId)
		if errSubscriber != nil {
			logger.Error(ctx, err, "failed loading postFeed", "userId", subscriberId)
			err = errSubscriber
		} else {
			s.sendPostFeedUpdate(ctx, subscriberId, posts)
		}
	}
	return err
}

func (s *PostFeedUpdater) sendPostFeedUpdate(ctx context.Context, userId model.UserId, posts []model.PostExtended) {
	notification := model.UserPostFeedNotification{
		Posts: posts,
	}
	notificationStr, err := json.Marshal(notification)
	if err != nil {
		logger.Error(
			ctx, err, "failed notifying",
			"stage", "marshal",
			"userId", userId,
			"event", &notification,
		)
		return
	}
	err = s.userPublisher.SendData(ctx, fmt.Sprintf(s.userUpdateRoutingKey, userId), notificationStr)
	if err != nil {
		logger.Error(
			ctx, err, "failed notifying",
			"stage", "send",
			"userId", userId,
			"event", &notification,
		)
	}
	logger.Debug(ctx, "sent user notification", "userId", userId, "notification", notification)
}
