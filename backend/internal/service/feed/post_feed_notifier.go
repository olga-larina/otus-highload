package feed

import (
	"context"
	"encoding/json"

	"github.com/olga-larina/otus-highload/backend/internal/logger"
	"github.com/olga-larina/otus-highload/backend/internal/model"
	"github.com/olga-larina/otus-highload/backend/internal/queue"
)

type PostFeedNotificationService struct {
	queue      queue.QueueSender
	routingKey string
}

func NewPostFeedNotificationService(queue queue.QueueSender, routingKey string) *PostFeedNotificationService {
	return &PostFeedNotificationService{queue: queue, routingKey: routingKey}
}

func (s *PostFeedNotificationService) NotifyInvalidateAll(ctx context.Context) error {
	notification := &model.Notification{
		Type:    model.InvalidateAllNotificationType,
		Payload: model.InvalidateAllNotification{},
	}
	return s.notify(ctx, notification)
}

func (s *PostFeedNotificationService) NotifyAddFriend(ctx context.Context, userId *model.UserId, friendId *model.UserId) error {
	notification := &model.Notification{
		Type: model.AddFriendNotificationType,
		Payload: model.AddFriendNotification{
			UserId:   *userId,
			FriendId: *friendId,
		},
	}
	return s.notify(ctx, notification)
}

func (s *PostFeedNotificationService) NotifyDeleteFriend(ctx context.Context, userId *model.UserId, friendId *model.UserId) error {
	notification := &model.Notification{
		Type: model.DeleteFriendNotificationType,
		Payload: model.DeleteFriendNotification{
			UserId:   *userId,
			FriendId: *friendId,
		},
	}
	return s.notify(ctx, notification)
}

func (s *PostFeedNotificationService) NotifyCreatePost(ctx context.Context, post *model.PostExtended) error {
	notification := &model.Notification{
		Type: model.CreatePostNotificationType,
		Payload: model.CreatePostNotification{
			UserId: *post.AuthorUserId,
			Post:   *post,
		},
	}
	return s.notify(ctx, notification)
}

func (s *PostFeedNotificationService) NotifyUpdatePost(ctx context.Context, post *model.PostExtended) error {
	notification := &model.Notification{
		Type: model.UpdatePostNotificationType,
		Payload: model.UpdatePostNotification{
			UserId: *post.AuthorUserId,
			Post:   *post,
		},
	}
	return s.notify(ctx, notification)
}

func (s *PostFeedNotificationService) NotifyDeletePost(ctx context.Context, postId *model.PostId, userId *model.UserId) error {
	notification := &model.Notification{
		Type: model.DeletePostNotificationType,
		Payload: model.DeletePostNotification{
			UserId: *userId,
			PostId: *postId,
		},
	}
	return s.notify(ctx, notification)
}

func (s *PostFeedNotificationService) notify(ctx context.Context, notification *model.Notification) error {
	notificationStr, err := json.Marshal(notification)
	if err != nil {
		logger.Error(
			ctx, err, "failed notifying",
			"stage", "marshal",
			"eventType", &notification.Type,
			"event", &notification.Payload,
		)
		return err
	}
	err = s.queue.SendData(ctx, s.routingKey, notificationStr)
	if err != nil {
		logger.Error(
			ctx, err, "failed notifying",
			"stage", "send",
			"eventType", &notification.Type,
			"event", &notification.Payload,
		)
		return err
	}
	return nil
}
