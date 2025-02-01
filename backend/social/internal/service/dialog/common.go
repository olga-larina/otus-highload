package dialog

import "github.com/olga-larina/otus-highload/social/internal/model"

type CreateMessageRequest struct {
	Text model.DialogMessageText `json:"text"`
}

type CreateMessageResponse struct {
}

type GetMessagesRequest struct {
}

type GetMessagesResponse []model.DialogMessage
