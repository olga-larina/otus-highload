package service

import (
	"context"

	"github.com/olga-larina/otus-highload/social/internal/model"
)

type LoginService struct {
	storage            LoginStorage
	passwordComparator PasswordComparator
	tokenGenerator     TokenGenerator
}

type LoginStorage interface {
	GetPasswordHash(ctx context.Context, id *model.UserId) ([]byte, error)
}

type PasswordComparator interface {
	CompareHashAndPassword(ctx context.Context, passwordHash []byte, password string) error
}

type TokenGenerator interface {
	CreateJWS(userId string) (string, error)
}

func NewLoginService(storage LoginStorage, passwordComparator PasswordComparator, tokenGenerator TokenGenerator) *LoginService {
	return &LoginService{storage: storage, passwordComparator: passwordComparator, tokenGenerator: tokenGenerator}
}

func (s *LoginService) Login(ctx context.Context, request *model.PostLoginJSONRequestBody) (string, error) {
	passwordHash, err := s.storage.GetPasswordHash(ctx, request.Id)
	if err != nil {
		return "", err
	}
	if err = s.passwordComparator.CompareHashAndPassword(ctx, passwordHash, *request.Password); err != nil {
		return "", err
	}
	token, err := s.tokenGenerator.CreateJWS(*request.Id)
	if err != nil {
		return "", err
	}
	return token, nil
}
