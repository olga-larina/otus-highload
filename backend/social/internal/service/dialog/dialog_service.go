package dialog

import (
	"context"
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/olga-larina/otus-highload/social/internal/model"
)

const (
	createMessageUrl = "/dialog/{user_id}/send"
	getMessagesUrl   = "/dialog/{user_id}/list"
	userIdPathParam  = "user_id"
)

type DialogService struct {
	dialogHttpClient DialogHttpClient
}

type DialogHttpClient interface {
	Get(ctx context.Context, path string, pathParams map[string]string, request any, responseTemplate any) (*resty.Response, error)
	Post(ctx context.Context, path string, pathParams map[string]string, request any, responseTemplate any) (*resty.Response, error)
}

func NewDialogService(dialogHttpClient DialogHttpClient) *DialogService {
	return &DialogService{dialogHttpClient: dialogHttpClient}
}

func (s *DialogService) CreateMessage(ctx context.Context, fromUserId *model.UserId, toUserId *model.UserId, postText *model.DialogMessageText) error {
	resp, err := s.dialogHttpClient.Post(
		ctx,
		createMessageUrl,
		buildPathParams(*toUserId),
		CreateMessageRequest{Text: *postText},
		CreateMessageResponse{},
	)
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("request error: status=%d, body=%s", resp.StatusCode(), resp.String())
	}
	return nil
}

func (s *DialogService) GetMessagesBetween(ctx context.Context, firstUserId *model.UserId, secondUserId *model.UserId) ([]model.DialogMessage, error) {
	getMessagesResponse := &GetMessagesResponse{}
	resp, err := s.dialogHttpClient.Get(
		ctx,
		getMessagesUrl,
		buildPathParams(*secondUserId),
		GetMessagesRequest{},
		getMessagesResponse,
	)
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("request error: status=%d, body=%s", resp.StatusCode(), resp.String())
	}
	return *getMessagesResponse, nil
}

func buildPathParams(userId string) map[string]string {
	params := make(map[string]string)
	params[userIdPathParam] = userId
	return params
}
