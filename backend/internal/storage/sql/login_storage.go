package sqlstorage

import (
	"context"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // for postgres
	"github.com/olga-larina/otus-highload/backend/internal/model"
)

type LoginStorage struct {
	db *Db
}

func NewLoginStorage(db *Db) *LoginStorage {
	return &LoginStorage{
		db: db,
	}
}

const getPasswordHashSQL = `
SELECT password_hash
FROM users
WHERE id = :id
`

func (s *LoginStorage) GetPasswordHash(ctx context.Context, id *model.UserId) ([]byte, error) {
	stmt, err := s.db.sqlDb.PrepareNamedContext(ctx, getPasswordHashSQL)
	if err != nil {
		return nil, fmt.Errorf("cannot prepare context for getting password hash by userId: %w", err)
	}

	rows, err := stmt.QueryxContext(ctx, map[string]interface{}{
		"id": &id,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot query context for getting password hash by userId: %w", err)
	}

	if !rows.Next() {
		return nil, model.ErrUserNotFound
	}

	var passwordHash []byte
	err = rows.Scan(&passwordHash)
	return passwordHash, err
}
