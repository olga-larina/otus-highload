package service

import (
	"context"
	"encoding/json"

	"github.com/olga-larina/otus-highload/pkg/logger"
	pkg_model "github.com/olga-larina/otus-highload/pkg/model"
	"github.com/olga-larina/otus-highload/pkg/queue"
	"github.com/olga-larina/otus-highload/social/internal/model"
	"github.com/pckilgore/combuuid"
)

type UserService struct {
	storage        UserStorage
	passwordHasher PasswordHasher
	userPublisher  queue.QueueSender
	userRoutingKey string
}

type UserStorage interface {
	CreateUser(ctx context.Context, user *model.UserExtended) error
	GetUserById(ctx context.Context, id *model.UserId) (*model.User, error)
	SearchUsersByName(ctx context.Context, firstNamePrefix string, lastNamePrefix string) ([]*model.User, error)
}

type PasswordHasher interface {
	HashPassword(ctx context.Context, password string) ([]byte, error)
}

func NewUserService(
	storage UserStorage,
	passwordHasher PasswordHasher,
	userPublisher queue.QueueSender,
	userRoutingKey string,
) *UserService {
	return &UserService{storage: storage, passwordHasher: passwordHasher, userPublisher: userPublisher, userRoutingKey: userRoutingKey}
}

func (s *UserService) GetUserById(ctx context.Context, id *model.UserId) (*model.User, error) {
	return s.storage.GetUserById(ctx, id)
}

func (s *UserService) RegisterUser(ctx context.Context, registerBody *model.PostUserRegisterJSONRequestBody) (*model.UserId, error) {
	passwordHash, err := s.passwordHasher.HashPassword(ctx, *registerBody.Password)
	if err != nil {
		logger.Error(ctx, err, "failed hashing password")
		return nil, pkg_model.ErrNotValidPassword
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
		Status:       model.UserPendingVerification,
	}

	if err := s.storage.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	s.sendUserCreated(ctx, user.User)

	return &userId, nil
}

func (s *UserService) SearchByName(ctx context.Context, firstNamePrefix string, lastNamePrefix string) ([]*model.User, error) {
	return s.storage.SearchUsersByName(ctx, firstNamePrefix, lastNamePrefix)
}

func (s *UserService) sendUserCreated(ctx context.Context, user model.User) {
	event := pkg_model.SagaEvent{Type: pkg_model.UserCreatedSagaEventType, Payload: user}
	eventStr, err := json.Marshal(event)
	if err != nil {
		logger.Error(
			ctx, err, "failed sending user created",
			"stage", "marshal",
			"event", &event,
		)
		return
	}
	err = s.userPublisher.SendData(ctx, s.userRoutingKey, eventStr)
	if err != nil {
		logger.Error(
			ctx, err, "failed sending user created",
			"stage", "send",
			"event", &event,
		)
		return
	}
	logger.Debug(ctx, "sent user created", "event", event)
}
