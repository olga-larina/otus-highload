package service

import (
	"context"

	"github.com/olga-larina/otus-highload/dialog/internal/model"
	"github.com/pckilgore/combuuid"
)

type DialogService struct {
	storage          DialogStorage
	dialogIdObtainer DialogIdObtainer
}

type DialogStorage interface {
	CreateMessage(ctx context.Context, message *model.DialogMessageExtended) (*model.DialogMessageExtended, error)
	GetMessagesInDialog(ctx context.Context, dialogId *model.DialogId) ([]*model.DialogMessageExtended, error)
}

type DialogIdObtainer interface {
	ObtainDialogId(ctx context.Context, firstUserId *model.UserId, secondUserId *model.UserId) (*model.DialogId, error)
}

func NewDialogService(storage DialogStorage, dialogIdObtainer DialogIdObtainer) *DialogService {
	return &DialogService{storage: storage, dialogIdObtainer: dialogIdObtainer}
}

func (s *DialogService) CreateMessage(ctx context.Context, fromUserId *model.UserId, toUserId *model.UserId, postText *model.DialogMessageText) (*model.DialogMessageExtended, error) {
	dialogId, err := s.dialogIdObtainer.ObtainDialogId(ctx, fromUserId, toUserId)
	if err != nil {
		return nil, err
	}
	messageId := combuuid.NewUuid().String() // sequential guid
	message, err := s.storage.CreateMessage(ctx, &model.DialogMessageExtended{
		Id:       messageId,
		DialogId: *dialogId,
		DialogMessage: model.DialogMessage{
			From: *fromUserId,
			To:   *toUserId,
			Text: *postText,
		},
	})
	if err != nil {
		return nil, err
	}
	return message, nil
}

func (s *DialogService) GetMessagesBetween(ctx context.Context, firstUserId *model.UserId, secondUserId *model.UserId) ([]*model.DialogMessageExtended, error) {
	dialogId, err := s.dialogIdObtainer.ObtainDialogId(ctx, firstUserId, secondUserId)
	if err != nil {
		return nil, err
	}
	return s.storage.GetMessagesInDialog(ctx, dialogId)
}
