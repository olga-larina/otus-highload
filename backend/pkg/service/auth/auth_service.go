package auth

import (
	"context"

	"github.com/olga-larina/otus-highload/pkg/model"
)

type AuthService struct {
}

func NewAuthService() *AuthService {
	return &AuthService{}
}

func (s *AuthService) GetUserId(ctx context.Context) (string, error) {
	userId, ok := ctx.Value(model.UserIdContextKey).(string)
	if !ok || len(userId) == 0 {
		return "", model.ErrNotAuthorized
	}
	return userId, nil
}
