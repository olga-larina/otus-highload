package feed

import (
	"context"

	err_model "github.com/olga-larina/otus-highload/pkg/model"
	"github.com/olga-larina/otus-highload/social/internal/model"
)

type PostFeedService struct {
	postFeedCache PostFeedCache
	maxFeedLength int
}

type PostFeedCache interface {
	GetOrLoad(ctx context.Context, userId *model.UserId) ([]model.PostExtended, error)
}

func NewPostFeedService(postFeedCache PostFeedCache, maxFeedLength int) *PostFeedService {
	return &PostFeedService{postFeedCache: postFeedCache, maxFeedLength: maxFeedLength}
}

func (s *PostFeedService) GetUserPosts(ctx context.Context, userId *model.UserId, limit int, offset int) ([]*model.PostExtended, error) {
	if offset+limit > s.maxFeedLength {
		return nil, err_model.ErrPostFeedLenNotValid
	}
	posts, err := s.postFeedCache.GetOrLoad(ctx, userId)
	if err != nil {
		return nil, err
	}
	postsResult := make([]*model.PostExtended, 0)
	for i := offset; i < offset+limit && i < len(posts); i++ {
		postsResult = append(postsResult, &posts[i])
	}
	return postsResult, nil
}
