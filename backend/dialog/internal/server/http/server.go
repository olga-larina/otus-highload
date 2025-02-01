package internalhttp

import (
	"context"
	"errors"

	"github.com/olga-larina/otus-highload/dialog/internal/model"
	"github.com/olga-larina/otus-highload/pkg/logger"
	err_model "github.com/olga-larina/otus-highload/pkg/model"
)

type Server struct {
	authService   AuthService
	dialogService DialogService
}

type AuthService interface {
	GetUserId(ctx context.Context) (string, error)
}

type DialogService interface {
	CreateMessage(ctx context.Context, fromUserId *model.UserId, toUserId *model.UserId, postText *model.DialogMessageText) (*model.DialogMessageExtended, error)
	GetMessagesBetween(ctx context.Context, firstUserId *model.UserId, secondUserId *model.UserId) ([]*model.DialogMessageExtended, error)
}

func NewServer(
	authService AuthService,
	dialogService DialogService,
) *Server {
	return &Server{
		authService:   authService,
		dialogService: dialogService,
	}
}

// (GET /dialog/{user_id}/list)
func (s *Server) GetDialogUserIdList(ctx context.Context, request GetDialogUserIdListRequestObject) (GetDialogUserIdListResponseObject, error) {
	userId, err := s.authService.GetUserId(ctx)
	if err != nil {
		logger.Error(ctx, err, "not authorized")
		return GetDialogUserIdList401Response{}, nil
	}
	messages, err := s.dialogService.GetMessagesBetween(ctx, &userId, &request.UserId)
	if err != nil {
		response := GetDialogUserIdList500JSONResponse{}
		response.Body.Message = err.Error()
		return response, nil
	}
	response := GetDialogUserIdList200JSONResponse{}
	for _, message := range messages {
		response = append(response, message.DialogMessage)
	}
	return response, nil
}

// (POST /dialog/{user_id}/send)
func (s *Server) PostDialogUserIdSend(ctx context.Context, request PostDialogUserIdSendRequestObject) (PostDialogUserIdSendResponseObject, error) {
	userId, err := s.authService.GetUserId(ctx)
	if err != nil {
		logger.Error(ctx, err, "not authorized")
		return PostDialogUserIdSend401Response{}, nil
	}
	_, err = s.dialogService.CreateMessage(ctx, &userId, &request.UserId, &request.Body.Text)
	if err != nil {
		if errors.Is(err, err_model.ErrUserNotFound) {
			logger.Error(ctx, err, "failed to create dialog message")
			return PostDialogUserIdSend404Response{}, nil
		}
		response := PostDialogUserIdSend500JSONResponse{}
		response.Body.Message = err.Error()
		return response, nil
	}
	return PostDialogUserIdSend200Response{}, nil
}
