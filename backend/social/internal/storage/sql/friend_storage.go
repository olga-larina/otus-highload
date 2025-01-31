package sqlstorage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	err_model "github.com/olga-larina/otus-highload/pkg/model"
	db "github.com/olga-larina/otus-highload/pkg/storage/sql"
	"github.com/olga-larina/otus-highload/social/internal/model"
)

type FriendStorage struct {
	db db.Db
}

func NewFriendStorage(db db.Db) *FriendStorage {
	return &FriendStorage{
		db: db,
	}
}

const addFriendSQL = `
INSERT INTO friends (user_id, friend_id)
VALUES ($1, $2)
`

func (s *FriendStorage) AddFriend(ctx context.Context, userId *model.UserId, friendId *model.UserId) error {
	_, err := s.db.Write(ctx, addFriendSQL, &userId, &friendId)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == db.ERR_CODE_FOREIGN_KEY_VIOLATION {
				return err_model.ErrUserNotFound
			}
			if pgErr.Code == db.ERR_CODE_UNIQUE_VIOLATION {
				return err_model.ErrUserAlreadyExists
			}
		}
	}
	return err
}

const deleteFriendSQL = `
DELETE FROM friends WHERE user_id=$1 AND friend_id=$2
RETURNING user_id
`

func (s *FriendStorage) DeleteFriend(ctx context.Context, userId *model.UserId, friendId *model.UserId) error {
	row := s.db.WriteReturn(ctx, deleteFriendSQL, &userId, &friendId)

	var deletedUserId model.UserId
	err := row.Scan(&deletedUserId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return err_model.ErrUserNotFound
		}
		return err
	}
	return nil
}

const getUserIdsWithFriend = `
SELECT user_id
FROM friends
where friend_id = $1
`

func (s *FriendStorage) GetUserIdsWithFriend(ctx context.Context, friendId *model.UserId) ([]*model.UserId, error) {
	rows, err := s.db.QueryRows(ctx, getUserIdsWithFriend, friendId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	userIds := make([]*model.UserId, 0)
	for rows.Next() {
		var userId model.UserId
		err = rows.Scan(&userId)
		if err != nil {
			return nil, err
		}
		userIds = append(userIds, &userId)
	}
	return userIds, nil
}
