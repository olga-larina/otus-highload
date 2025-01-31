package sqlstorage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	err_model "github.com/olga-larina/otus-highload/pkg/model"
	db "github.com/olga-larina/otus-highload/pkg/storage/sql"
	"github.com/olga-larina/otus-highload/social/internal/model"
)

type LoginStorage struct {
	db db.Db
}

func NewLoginStorage(db db.Db) *LoginStorage {
	return &LoginStorage{
		db: db,
	}
}

const getPasswordHashSQL = `
SELECT password_hash
FROM users
WHERE id = $1
`

func (s *LoginStorage) GetPasswordHash(ctx context.Context, id *model.UserId) ([]byte, error) {
	row := s.db.QueryRow(ctx, getPasswordHashSQL, &id)

	var passwordHash []byte
	err := row.Scan(&passwordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err_model.ErrUserNotFound
		}
		return nil, err
	}
	return passwordHash, nil
}
