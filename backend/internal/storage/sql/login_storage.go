package sqlstorage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/olga-larina/otus-highload/backend/internal/model"
)

type LoginStorage struct {
	db Db
}

func NewLoginStorage(db Db) *LoginStorage {
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
			return nil, model.ErrUserNotFound
		}
		return nil, err
	}
	return passwordHash, nil
}
