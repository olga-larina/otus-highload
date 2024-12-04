package service

import (
	"context"

	"github.com/olga-larina/otus-highload/backend/internal/logger"
	"github.com/olga-larina/otus-highload/backend/internal/model"
	"github.com/pckilgore/combuuid"
)

type UserService struct {
	storage        UserStorage
	passwordHasher PasswordHasher
}

type UserStorage interface {
	CreateUser(ctx context.Context, user *model.UserExtended) error
	GetUserById(ctx context.Context, id *model.UserId) (*model.User, error)
	SearchUsersByName(ctx context.Context, firstNamePrefix string, lastNamePrefix string) ([]*model.User, error)
}

type PasswordHasher interface {
	HashPassword(ctx context.Context, password string) ([]byte, error)
}

func NewUserService(storage UserStorage, passwordHasher PasswordHasher) *UserService {
	return &UserService{storage: storage, passwordHasher: passwordHasher}
}

func (s *UserService) GetUserById(ctx context.Context, id *model.UserId) (*model.User, error) {
	return s.storage.GetUserById(ctx, id)
}

func (s *UserService) GetMe(ctx context.Context) (*model.User, error) {
	userId := ctx.Value(model.UserIdContextKey).(string)
	return s.GetUserById(ctx, &userId)
}

func (s *UserService) RegisterUser(ctx context.Context, registerBody *model.PostUserRegisterJSONRequestBody) (*model.UserId, error) {
	passwordHash, err := s.passwordHasher.HashPassword(ctx, *registerBody.Password)
	if err != nil {
		logger.Error(ctx, err, "failed hashing password")
		return nil, model.ErrNotValidPassword
	}

	userId := combuuid.NewUuid().String() // sequential guid
	user := &model.UserExtended{
		User: model.User{
			Id:         &userId,
			FirstName:  registerBody.FirstName,
			SecondName: registerBody.SecondName,
			City:       registerBody.City,
			Gender:     registerBody.Gender,
			Birthdate:  registerBody.Birthdate,
			Biography:  registerBody.Biography,
		},
		PasswordHash: passwordHash,
	}

	if err := s.storage.CreateUser(ctx, user); err != nil {
		return nil, err
	}
	return &userId, nil
}

func (s *UserService) SearchByName(ctx context.Context, firstNamePrefix string, lastNamePrefix string) ([]*model.User, error) {
	return s.storage.SearchUsersByName(ctx, firstNamePrefix, lastNamePrefix)
}
