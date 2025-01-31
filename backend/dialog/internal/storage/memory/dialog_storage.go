package memory

import (
	"context"
	"errors"
	"time"

	"github.com/olga-larina/otus-highload/dialog/internal/model"
	"github.com/olga-larina/otus-highload/pkg/logger"
	"github.com/tarantool/go-tarantool/v2"
	"go.opentelemetry.io/otel"
)

const (
	TIME_FORMAT = "2006-01-02T15:04:05"
)

type DialogStorage struct {
	conn *tarantool.Connection
}

func NewDialogStorage(conn *tarantool.Connection) *DialogStorage {
	return &DialogStorage{conn: conn}
}

func (s *DialogStorage) CreateMessage(ctx context.Context, message *model.DialogMessageExtended) (*model.DialogMessageExtended, error) {
	_, span := otel.Tracer("default").Start(ctx, "tarantool write")
	defer span.End()
	req := tarantool.NewCallRequest("insert_message")
	req = req.Args([]interface{}{
		message.DialogId, message.Id, message.Text, message.From, message.To,
	})
	resp, err := s.conn.Do(req).Get()
	if err != nil {
		return nil, err
	}
	if len(resp) > 0 {
		sendTimeStr, ok := resp[0].(string)
		if !ok {
			return nil, err
		}
		sendTime, err := time.Parse(TIME_FORMAT, sendTimeStr)
		if err != nil {
			return nil, err
		}
		message.SendTime = sendTime

		return message, nil
	} else {
		return nil, errors.New("failed to create message")
	}
}

func (s *DialogStorage) GetMessagesInDialog(ctx context.Context, dialogId *model.DialogId) ([]*model.DialogMessageExtended, error) {
	ctxWithSpan, span := otel.Tracer("default").Start(ctx, "tarantool read")
	defer span.End()
	req := tarantool.NewCallRequest("get_messages_by_dialog")
	req = req.Args([]interface{}{&dialogId})
	resp, err := s.conn.Do(req).Get()
	if err != nil {
		return nil, err
	}

	if len(resp) != 1 {
		return nil, errors.New("failed to parse messages")
	}
	respArr := resp[0].([]interface{})

	messages := make([]*model.DialogMessageExtended, 0, len(respArr))
	for _, item := range respArr {
		row, ok := item.([]interface{})
		if !ok {
			return nil, errors.New("failed to parse messages")
		}

		message := &model.DialogMessageExtended{
			Id:       row[1].(string),
			DialogId: row[0].(string),
			DialogMessage: model.DialogMessage{
				Text: row[2].(string),
				From: row[3].(string),
				To:   row[4].(string),
			},
		}

		sendTimeStr := row[5].(string)
		sendTime, err := time.Parse(TIME_FORMAT, sendTimeStr)
		if err == nil {
			message.SendTime = sendTime
		} else {
			logger.Error(ctxWithSpan, err, "failed to convert sendTime", "messageId", message.Id, "sendTime", sendTimeStr)
		}

		messages = append(messages, message)
	}
	return messages, nil
}
