package sqlstorage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/olga-larina/otus-highload/backend/internal/model"
)

type PostStorage struct {
	db Db
}

func NewPostStorage(db Db) *PostStorage {
	return &PostStorage{
		db: db,
	}
}

const createPostSQL = `
INSERT INTO posts (id, content, user_id, create_time, update_time)
VALUES ($1, $2, $3, now(), now())
RETURNING id, content, user_id, create_time, update_time
`

func (s *PostStorage) CreatePost(ctx context.Context, post *model.Post) (*model.PostExtended, error) {
	row := s.db.QueryRow(ctx, createPostSQL, &post.Id, &post.Text, &post.AuthorUserId)

	var postCreated model.PostExtended
	err := row.Scan(&postCreated.Id, &postCreated.Text, &postCreated.AuthorUserId, &postCreated.CreateTime, &postCreated.UpdateTime)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == ERR_CODE_FOREIGN_KEY_VIOLATION {
				return nil, model.ErrUserNotFound
			}
		}
		return nil, err
	}
	return &postCreated, nil
}

const updatePostSQL = `
UPDATE posts
SET content=$3
WHERE id=$1 AND user_id=$2
RETURNING id, content, user_id, create_time, update_time
`

func (s *PostStorage) UpdatePost(ctx context.Context, post *model.Post) (*model.PostExtended, error) {
	row := s.db.QueryRow(ctx, updatePostSQL, &post.Id, &post.AuthorUserId, &post.Text)

	var postUpdated model.PostExtended
	err := row.Scan(&postUpdated.Id, &postUpdated.Text, &postUpdated.AuthorUserId, &postUpdated.CreateTime, &postUpdated.UpdateTime)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrPostNotFound
		}
		return nil, err
	}
	return &postUpdated, nil
}

const deletePostByIdSQL = `
DELETE FROM posts WHERE id=$1 AND user_id=$2
RETURNING id
`

func (s *PostStorage) DeletePost(ctx context.Context, postId *model.PostId, userId *model.UserId) error {
	row := s.db.QueryRow(ctx, deletePostByIdSQL, &postId, &userId)

	var deletedPostId model.PostId
	err := row.Scan(&deletedPostId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ErrPostNotFound
		}
		return err
	}
	return nil
}

const getPostByIdSQL = `
SELECT id, content, user_id, create_time, update_time
FROM posts
WHERE id = $1 AND user_id=$2
`

func (s *PostStorage) GetPostById(ctx context.Context, postId *model.PostId, userId *model.UserId) (*model.PostExtended, error) {
	row := s.db.QueryRow(ctx, getPostByIdSQL, &postId, &userId)

	var post model.PostExtended
	err := row.Scan(&post.Id, &post.Text, &post.AuthorUserId, &post.CreateTime, &post.UpdateTime)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrPostNotFound
		}
		return nil, err
	}
	return &post, nil
}
