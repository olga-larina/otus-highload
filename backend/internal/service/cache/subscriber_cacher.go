package cacher

import (
	"context"

	"github.com/olga-larina/otus-highload/backend/internal/model"
	redis_cache "github.com/olga-larina/otus-highload/backend/internal/service/cache/redis"
)

type SubscriberCacher struct {
	redisCache  *redis_cache.RedisCache
	reloadCache redis_cache.LoadFunc
}

type FriendStorage interface {
	// Получить ID пользователей, у которых в друзьях friendId (т.е. тех, кто подписан на него)
	GetUserIdsWithFriend(ctx context.Context, friendId *model.UserId) ([]*model.UserId, error)
}

func NewSubscriberCacher(redisCache *redis_cache.RedisCache, friendStorage FriendStorage) *SubscriberCacher {
	return &SubscriberCacher{
		redisCache: redisCache,
		reloadCache: func(ctx context.Context, key string) (any, error) {
			subscribers, err := friendStorage.GetUserIdsWithFriend(ctx, &key)
			if err != nil {
				return nil, err
			}
			subscribersResult := make([]model.UserId, 0)
			for _, subscriber := range subscribers {
				subscribersResult = append(subscribersResult, *subscriber)
			}
			return subscribersResult, nil
		},
	}
}

func (c *SubscriberCacher) InvalidateAll(ctx context.Context) error {
	return c.redisCache.InvalidateAll(ctx)
}

func (c *SubscriberCacher) GetOrLoad(ctx context.Context, userId *model.UserId) ([]model.UserId, error) {
	res, err := c.redisCache.GetOrLoad(ctx, *userId, c.reloadCache)
	if err != nil {
		return nil, err
	}
	return convertSubscribers(res)
}

func (c *SubscriberCacher) Load(ctx context.Context, userId *model.UserId) ([]model.UserId, error) {
	res, err := c.redisCache.Load(ctx, *userId, c.reloadCache)
	if err != nil {
		return nil, err
	}
	return convertSubscribers(res)
}

func convertSubscribers(res any) ([]model.UserId, error) {
	posts, ok := res.([]model.UserId)
	if !ok {
		return nil, model.ErrNotValidCache
	}
	return posts, nil
}
