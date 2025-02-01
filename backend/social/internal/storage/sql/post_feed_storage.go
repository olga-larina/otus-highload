package sqlstorage

import (
	"context"

	db "github.com/olga-larina/otus-highload/pkg/storage/sql"
	"github.com/olga-larina/otus-highload/social/internal/model"
)

type PostFeedStorage struct {
	db db.Db
}

func NewPostFeedStorage(db db.Db) *PostFeedStorage {
	return &PostFeedStorage{
		db: db,
	}
}

const getPostsFeedByUserIdSQL = `
SELECT p.id, p.content, p.user_id, p.create_time, p.update_time
FROM friends f
  JOIN posts p ON f.friend_id = p.user_id
WHERE f.user_id = $1
ORDER BY create_time DESC
LIMIT $2
OFFSET $3
`

func (s *PostFeedStorage) GetPostsFeedByUserId(ctx context.Context, userId *model.UserId, limit int, offset int) ([]*model.PostExtended, error) {
	rows, err := s.db.QueryRows(ctx, getPostsFeedByUserIdSQL, userId, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := make([]*model.PostExtended, 0)
	for rows.Next() {
		var post model.PostExtended
		err = rows.Scan(&post.Id, &post.Text, &post.AuthorUserId, &post.CreateTime, &post.UpdateTime)
		if err != nil {
			return nil, err
		}
		posts = append(posts, &post)
	}
	return posts, nil
}
