package service

import (
	"context"

	"github.com/olga-larina/otus-highload/backend/internal/model"
)

type FriendService struct {
	storage  FriendStorage
	notifier FriendNotifier
}

type FriendStorage interface {
	AddFriend(ctx context.Context, userId *model.UserId, friendId *model.UserId) error
	DeleteFriend(ctx context.Context, userId *model.UserId, friendId *model.UserId) error
}

type FriendNotifier interface {
	NotifyAddFriend(ctx context.Context, userId *model.UserId, friendId *model.UserId) error
	NotifyDeleteFriend(ctx context.Context, userId *model.UserId, friendId *model.UserId) error
}

func NewFriendService(storage FriendStorage, notifier FriendNotifier) *FriendService {
	return &FriendService{storage: storage, notifier: notifier}
}

func (s *FriendService) AddFriend(ctx context.Context, userId *model.UserId, friendId *model.UserId) error {
	if err := s.storage.AddFriend(ctx, userId, friendId); err != nil {
		return err
	}
	_ = s.notifier.NotifyAddFriend(ctx, userId, friendId)
	return nil
}

func (s *FriendService) DeleteFriend(ctx context.Context, userId *model.UserId, friendId *model.UserId) error {
	if err := s.storage.DeleteFriend(ctx, userId, friendId); err != nil {
		return err
	}
	_ = s.notifier.NotifyDeleteFriend(ctx, userId, friendId)
	return nil
}
