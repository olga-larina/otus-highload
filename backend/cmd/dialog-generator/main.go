package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pckilgore/combuuid"
	"github.com/tarantool/go-tarantool/v2"

	"github.com/olga-larina/otus-highload/backend/internal/logger"
	"github.com/olga-larina/otus-highload/backend/internal/model"
	"github.com/olga-larina/otus-highload/backend/internal/service/shard"
)

const (
	DB_URI             = "postgres://otus:password@localhost:5432/backend"
	TARANTOOL_URI      = "localhost:3301"
	TARANTOOL_USER     = "otus"
	TARANTOOL_PASSWORD = "secretpassword"
	DIALOGS_CNT        = 10_000
	MESSAGES_CNT       = 100
)

func main() {
	var err error

	err = logger.New("DEBUG")
	if err != nil {
		log.Fatalf("failed building logger %v", err)
		return
	}

	ctx := context.Background()

	// подключение к БД
	cfg, err := pgxpool.ParseConfig(DB_URI)
	if err != nil {
		logger.Error(ctx, err, "failed to parse db uri")
		return
	}
	dbPool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		logger.Error(ctx, err, "failed to create db")
		return
	}
	defer dbPool.Close()

	// подключение к tarantool
	tarantoolDialer := tarantool.NetDialer{
		Address:  TARANTOOL_URI,
		User:     TARANTOOL_USER,
		Password: TARANTOOL_PASSWORD,
	}
	opts := tarantool.Opts{}
	tarantoolConn, err := tarantool.Connect(ctx, tarantoolDialer, opts)
	if err != nil {
		logger.Error(ctx, err, "failed to connect to tarantool")
		return
	}
	defer tarantoolConn.Close()

	// dialog id obtainer
	dialogIdObtainer := shard.NewDialogIdObtainer()

	// файл со значениями dialogId-userId1-userId2
	csvFile, err := os.Create("dialogs.csv")
	if err != nil {
		logger.Error(ctx, err, "failed to create csv file")
		return
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	// Генерация диалогов с сообщениями
	for i := 0; i < DIALOGS_CNT; i++ {
		// Выбираем случайных пользователей
		firstUser, secondUser, err := selectRandomUsers(ctx, dbPool)
		if err != nil {
			logger.Error(ctx, err, "failed to select random users")
			return
		}

		// Генерируем dialog_id
		dialogId, err := dialogIdObtainer.ObtainDialogId(ctx, &firstUser, &secondUser)
		if err != nil {
			logger.Error(ctx, err, "failed to generate dialogId", "userId1", firstUser, "userId2", secondUser)
			return
		}

		// Записываем в CSV
		if err := writer.Write([]string{firstUser, secondUser, *dialogId}); err != nil {
			logger.Error(ctx, err, "failed to write dialog to csv", "dialogId", *dialogId, "userId1", firstUser, "userId2", secondUser)
			return
		}

		// Создаём сообщения для этого диалога
		if err := createMessages(ctx, dbPool, tarantoolConn, *dialogId, firstUser, secondUser); err != nil {
			logger.Error(ctx, err, "failed to create messages for dialogId", "dialogId", *dialogId, "userId1", firstUser, "userId2", secondUser)
			return
		}
	}
}

func selectRandomUsers(ctx context.Context, dbPool *pgxpool.Pool) (model.UserId, model.UserId, error) {
	query := `
		SELECT id
		FROM users
		ORDER BY RANDOM()
		LIMIT 2;
	`
	rows, err := dbPool.Query(ctx, query)
	if err != nil {
		return "", "", err
	}
	defer rows.Close()

	var users []model.UserId
	for rows.Next() {
		var id model.UserId
		if err := rows.Scan(&id); err != nil {
			return "", "", err
		}
		users = append(users, id)
	}

	if len(users) != 2 {
		return "", "", fmt.Errorf("failed to select two users")
	}

	return users[0], users[1], nil
}

func createMessages(
	ctx context.Context,
	dbPool *pgxpool.Pool,
	tarantoolConn *tarantool.Connection,
	dialogId model.DialogId,
	firstUser model.UserId,
	secondUser model.UserId,
) error {
	sql := `
		INSERT INTO messages (dialog_id, message_id, content, from_user_id, to_user_id, send_time)
		VALUES ($1, $2, $3, $4, $5, now())
	`
	dbBatch := &pgx.Batch{}
	var tarantoolBatch []map[string]interface{}

	for i := 0; i < MESSAGES_CNT; i++ {
		fromUser := firstUser
		toUser := secondUser
		if i%2 == 1 { // чередуем отправителей
			fromUser, toUser = secondUser, firstUser
		}

		content := fmt.Sprintf("Message %d in dialog %s", i+1, dialogId)
		messageId := combuuid.NewUuid().String()

		dbBatch.Queue(sql, dialogId, messageId, content, fromUser, toUser)
		tarantoolBatch = append(tarantoolBatch, map[string]interface{}{
			"dialog_id":    dialogId,
			"message_id":   messageId,
			"content":      content,
			"from_user_id": fromUser,
			"to_user_id":   toUser,
		})
	}

	// добавляем в postgres
	br := dbPool.SendBatch(ctx, dbBatch)
	defer br.Close()
	if _, err := br.Exec(); err != nil {
		return fmt.Errorf("postgres batch execution failed: %w", err)
	}

	// добавляем в tarantool
	req := tarantool.NewCallRequest("batch_insert_messages")
	req = req.Args([]interface{}{tarantoolBatch})
	_, err := tarantoolConn.Do(req).Get()
	if err != nil {
		return fmt.Errorf("tarantool batch execution failed: %w", err)
	}

	return nil
}
