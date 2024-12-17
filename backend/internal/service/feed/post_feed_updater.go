package feed

import (
	"context"
	"encoding/json"

	"github.com/olga-larina/otus-highload/backend/internal/logger"
	"github.com/olga-larina/otus-highload/backend/internal/model"
)

type PostFeedUpdater struct {
	postFeedCache    PostFeedCacher
	subscribersCache SubscribersCacher
	queue            QueueConsumer
	done             chan struct{}
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

type QueueConsumer interface {
	ReceiveData(ctx context.Context) (<-chan []byte, error)
}

func NewPostFeedUpdater(
	postFeedCache PostFeedCacher,
	subscribersCache SubscribersCacher,
	queue QueueConsumer,
) *PostFeedUpdater {
	return &PostFeedUpdater{
		postFeedCache:    postFeedCache,
		subscribersCache: subscribersCache,
		queue:            queue,
		done:             make(chan struct{}),
	}
}

func (s *PostFeedUpdater) Start(ctx context.Context) error {
	logger.Info(ctx, "starting postFeedUpdater")

	err := s.processEvents(ctx)
	if err != nil {
		return err
	}

	logger.Info(ctx, "started postFeedUpdater")
	return nil
}

func (s *PostFeedUpdater) Stop(ctx context.Context) error {
	logger.Info(ctx, "stopping postFeedUpdater")

	<-ctx.Done()
	<-s.done

	logger.Info(ctx, "stopped postFeedUpdater")
	return nil
}

/*
 * Получение событий из очереди.
 */
func (s *PostFeedUpdater) processEvents(ctx context.Context) error {
	data, err := s.queue.ReceiveData(ctx)
	if err != nil {
		defer close(s.done)
		return err
	}

	go func() {
		defer close(s.done)

		for d := range data {
			var notification model.NotificationJson

			err := json.Unmarshal(d, &notification)
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
	_, err = s.postFeedCache.Load(ctx, &userId)
	if err != nil {
		logger.Error(ctx, err, "failed loading postFeed", "userId", userId)
	}
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
	var errSubscriber error
	for _, subscriberId := range subscribers {
		_, errSubscriber = s.postFeedCache.Load(ctx, &subscriberId)
		if errSubscriber != nil {
			logger.Error(ctx, err, "failed loading postFeed", "userId", subscriberId)
			err = errSubscriber
		}
	}
	return err
}
