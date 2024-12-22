package service

import (
	"context"

	"github.com/olga-larina/otus-highload/backend/internal/model"
	"github.com/pckilgore/combuuid"
)

type PostService struct {
	storage  PostStorage
	notifier PostNotifier
}

type PostStorage interface {
	CreatePost(ctx context.Context, post *model.Post) (*model.PostExtended, error)
	UpdatePost(ctx context.Context, post *model.Post) (*model.PostExtended, error)
	DeletePost(ctx context.Context, postId *model.PostId, userId *model.UserId) error
	GetPostById(ctx context.Context, postId *model.PostId, userId *model.UserId) (*model.PostExtended, error)
}

type PostNotifier interface {
	NotifyCreatePost(ctx context.Context, post *model.PostExtended) error
	NotifyUpdatePost(ctx context.Context, post *model.PostExtended) error
	NotifyDeletePost(ctx context.Context, postId *model.PostId, userId *model.UserId) error
}

func NewPostService(storage PostStorage, notifier PostNotifier) *PostService {
	return &PostService{storage: storage, notifier: notifier}
}

func (s *PostService) CreatePost(ctx context.Context, userId *model.UserId, postText *model.PostText) (*model.PostExtended, error) {
	postId := combuuid.NewUuid().String() // sequential guid
	post, err := s.storage.CreatePost(ctx, &model.Post{Id: &postId, AuthorUserId: userId, Text: postText})
	if err != nil {
		return nil, err
	}
	_ = s.notifier.NotifyCreatePost(ctx, post)
	return post, nil
}

func (s *PostService) UpdatePost(ctx context.Context, userId *model.UserId, postId *model.PostId, postText *model.PostText) (*model.PostExtended, error) {
	post, err := s.storage.UpdatePost(ctx, &model.Post{Id: postId, AuthorUserId: userId, Text: postText})
	if err != nil {
		return nil, err
	}
	_ = s.notifier.NotifyUpdatePost(ctx, post)
	return post, nil
}

func (s *PostService) DeletePost(ctx context.Context, userId *model.UserId, postId *model.PostId) error {
	if err := s.storage.DeletePost(ctx, postId, userId); err != nil {
		return err
	}
	_ = s.notifier.NotifyDeletePost(ctx, postId, userId)
	return nil
}

func (s *PostService) GetPostById(ctx context.Context, userId *model.UserId, postId *model.PostId) (*model.PostExtended, error) {
	return s.storage.GetPostById(ctx, postId, userId)
}
