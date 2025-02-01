package cacher

import (
	"context"

	err_model "github.com/olga-larina/otus-highload/pkg/model"
	"github.com/olga-larina/otus-highload/social/internal/model"
	redis_cache "github.com/olga-larina/otus-highload/social/internal/service/cache/redis"
)

type PostFeedCacher struct {
	redisCache  *redis_cache.RedisCache
	reloadCache redis_cache.LoadFunc
}

type PostFeedStorage interface {
	GetPostsFeedByUserId(ctx context.Context, userId *model.UserId, limit int, offset int) ([]*model.PostExtended, error)
}

func NewPostFeedCacher(redisCache *redis_cache.RedisCache, postFeedStorage PostFeedStorage, maxFeedLength int) *PostFeedCacher {
	return &PostFeedCacher{
		redisCache: redisCache,
		reloadCache: func(ctx context.Context, key string) (any, error) {
			posts, err := postFeedStorage.GetPostsFeedByUserId(ctx, &key, maxFeedLength, 0)
			if err != nil {
				return nil, err
			}
			postsResult := make([]model.PostExtended, 0)
			for _, post := range posts {
				postsResult = append(postsResult, *post)
			}
			return postsResult, nil
		},
	}
}

func (c *PostFeedCacher) InvalidateAll(ctx context.Context) error {
	return c.redisCache.InvalidateAll(ctx)
}

func (c *PostFeedCacher) GetOrLoad(ctx context.Context, userId *model.UserId) ([]model.PostExtended, error) {
	res, err := c.redisCache.GetOrLoad(ctx, *userId, c.reloadCache)
	if err != nil {
		return nil, err
	}
	return convertPostFeed(res)
}

func (c *PostFeedCacher) Load(ctx context.Context, userId *model.UserId) ([]model.PostExtended, error) {
	res, err := c.redisCache.Load(ctx, *userId, c.reloadCache)
	if err != nil {
		return nil, err
	}
	return convertPostFeed(res)
}

func convertPostFeed(res any) ([]model.PostExtended, error) {
	posts, ok := res.([]model.PostExtended)
	if !ok {
		return nil, err_model.ErrNotValidCache
	}
	return posts, nil
}
