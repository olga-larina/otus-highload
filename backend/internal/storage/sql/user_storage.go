package sqlstorage

import (
	"context"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // for postgres
	"github.com/olga-larina/otus-highload/backend/internal/model"
)

type UserStorage struct {
	db *Db
}

func NewUserStorage(db *Db) *UserStorage {
	return &UserStorage{
		db: db,
	}
}

const createUserSQL = `
INSERT INTO users (id, first_name, second_name, city, gender, birthdate, biography, password_hash)
VALUES (:id, :first_name, :second_name, :city, :gender, :birthdate, :biography, :password_hash)
`

func (s *UserStorage) CreateUser(ctx context.Context, user *model.UserExtended) error {
	stmt, err := s.db.sqlDb.PrepareNamedContext(ctx, createUserSQL)
	if err != nil {
		return fmt.Errorf("cannot prepare context for creating user: %w", err)
	}

	_, err = stmt.ExecContext(ctx, user)
	return err
}

const getUserByIdSQL = `
SELECT id, first_name, second_name, city, gender, birthdate, biography
FROM users
WHERE id = :id
`

func (s *UserStorage) GetUserById(ctx context.Context, id *model.UserId) (*model.User, error) {
	stmt, err := s.db.sqlDb.PrepareNamedContext(ctx, getUserByIdSQL)
	if err != nil {
		return nil, fmt.Errorf("cannot prepare context for getting user by id: %w", err)
	}

	rows, err := stmt.QueryxContext(ctx, map[string]interface{}{
		"id": &id,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot query context for getting user by id: %w", err)
	}

	if !rows.Next() {
		return nil, model.ErrUserNotFound
	}

	var user model.User
	err = rows.StructScan(&user)
	return &user, err
}

const searchByNameSQL = `
SELECT id, first_name, second_name, city, gender, birthdate, biography
FROM users
WHERE first_name LIKE :first_name AND second_name LIKE :last_name
ORDER BY id
`

func (s *UserStorage) SearchUsersByName(ctx context.Context, firstNamePrefix string, lastNamePrefix string) ([]*model.User, error) {
	stmt, err := s.db.sqlDb.PrepareNamedContext(ctx, searchByNameSQL)
	if err != nil {
		return nil, fmt.Errorf("cannot prepare context for getting user by id: %w", err)
	}

	rows, err := stmt.QueryxContext(ctx, map[string]interface{}{
		"first_name": firstNamePrefix + "%",
		"last_name":  lastNamePrefix + "%",
	})
	if err != nil {
		return nil, fmt.Errorf("cannot query context for searching users by name: %w", err)
	}

	users := make([]*model.User, 0)
	for rows.Next() {
		var user model.User
		err = rows.StructScan(&user)
		if err != nil {
			return nil, fmt.Errorf("cannot get result for searching users by name: %w", err)
		}
		users = append(users, &user)
	}
	return users, nil
}
