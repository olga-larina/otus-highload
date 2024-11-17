package service

import (
	"context"

	"github.com/olga-larina/otus-highload/backend/internal/logger"
	"github.com/olga-larina/otus-highload/backend/internal/model"
	"golang.org/x/crypto/bcrypt"
)

type PasswordService struct {
}

func NewPasswordService() *PasswordService {
	return &PasswordService{}
}

func (s *PasswordService) HashPassword(_ context.Context, password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

func (s *PasswordService) CompareHashAndPassword(ctx context.Context, passwordHash []byte, password string) error {
	if err := bcrypt.CompareHashAndPassword(passwordHash, []byte(password)); err != nil {
		logger.Error(ctx, err, "failed comparing passwords")
		return model.ErrNotValidCredentials
	}
	return nil
}
