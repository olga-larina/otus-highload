package sqlstorage

import (
	"context"
	"time"

	"github.com/olga-larina/otus-highload/dialog/internal/model"
	db "github.com/olga-larina/otus-highload/pkg/storage/sql"
)

type DialogStorage struct {
	db db.Db
}

func NewDialogStorage(db db.Db) *DialogStorage {
	return &DialogStorage{
		db: db,
	}
}

const createMessageSQL = `
INSERT INTO messages (message_id, content, dialog_id, from_user_id, to_user_id, send_time)
VALUES ($1, $2, $3, $4, $5, now())
RETURNING send_time
`

func (s *DialogStorage) CreateMessage(ctx context.Context, message *model.DialogMessageExtended) (*model.DialogMessageExtended, error) {
	row := s.db.WriteReturn(ctx, createMessageSQL, message.Id, message.Text, message.DialogId, message.From, message.To)

	var sendTime time.Time
	err := row.Scan(&sendTime)
	if err != nil {
		return nil, err
	}

	message.SendTime = sendTime
	return message, nil
}

const getMessagesInDialogSQL = `
SELECT message_id, content, dialog_id, from_user_id, to_user_id, send_time
FROM messages
WHERE dialog_id = $1
ORDER BY send_time DESC
`

func (s *DialogStorage) GetMessagesInDialog(ctx context.Context, dialogId *model.DialogId) ([]*model.DialogMessageExtended, error) {
	rows, err := s.db.QueryRows(ctx, getMessagesInDialogSQL, &dialogId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]*model.DialogMessageExtended, 0)
	for rows.Next() {
		var message model.DialogMessageExtended
		err = rows.Scan(&message.Id, &message.Text, &message.DialogId, &message.From, &message.To, &message.SendTime)
		if err != nil {
			return nil, err
		}
		messages = append(messages, &message)
	}
	return messages, nil
}
