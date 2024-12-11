package sqlstorage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib" // for postgres
	"github.com/olga-larina/otus-highload/backend/internal/model"
)

type UserStorage struct {
	db Db
}

func NewUserStorage(db Db) *UserStorage {
	return &UserStorage{
		db: db,
	}
}

const createUserSQL = `
INSERT INTO users (id, first_name, second_name, city, gender, birthdate, biography, password_hash)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`

func (s *UserStorage) CreateUser(ctx context.Context, user *model.UserExtended) error {
	_, err := s.db.Write(
		ctx,
		createUserSQL,
		user.Id, user.FirstName, user.SecondName, user.City, user.Gender, user.Birthdate, user.Biography, user.PasswordHash,
	)
	return err
}

const getUserByIdSQL = `
SELECT id, first_name, second_name, city, gender, birthdate, biography
FROM users
WHERE id = $1
`

func (s *UserStorage) GetUserById(ctx context.Context, id *model.UserId) (*model.User, error) {
	row := s.db.QueryRow(ctx, getUserByIdSQL, &id)

	var user model.User
	err := row.Scan(&user.Id, &user.FirstName, &user.SecondName, &user.City, &user.Gender, &user.Birthdate, &user.Biography)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

const searchByNameSQL = `
SELECT id, first_name, second_name, city, gender, birthdate, biography
FROM users
WHERE first_name LIKE $1 AND second_name LIKE $2
ORDER BY id
`

func (s *UserStorage) SearchUsersByName(ctx context.Context, firstNamePrefix string, lastNamePrefix string) ([]*model.User, error) {
	rows, err := s.db.QueryRows(ctx, searchByNameSQL, firstNamePrefix+"%", lastNamePrefix+"%")
	if err != nil {
		return nil, fmt.Errorf("cannot query for searching users by name: %w", err)
	}
	defer rows.Close()

	users := make([]*model.User, 0)
	for rows.Next() {
		var user model.User
		err = rows.Scan(&user.Id, &user.FirstName, &user.SecondName, &user.City, &user.Gender, &user.Birthdate, &user.Biography)
		if err != nil {
			return nil, fmt.Errorf("cannot get result for searching users by name: %w", err)
		}
		users = append(users, &user)
	}
	return users, nil
}
