package rabbit

import (
	"context"

	"github.com/olga-larina/otus-highload/backend/internal/logger"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Queue struct {
	uri          string
	exchangeName string
	exchangeType string

	connection *amqp.Connection
}

func NewQueue(uri string, exchangeName string, exchangeType string) *Queue {
	return &Queue{
		uri:          uri,
		exchangeName: exchangeName,
		exchangeType: exchangeType,
	}
}

func (q *Queue) Start(ctx context.Context) error {
	logger.Info(ctx, "starting rabbit queue")

	var err error

	q.connection, err = amqp.Dial(q.uri)
	if err != nil {
		return err
	}

	logger.Info(ctx, "got rabbit queue connection")
	return nil
}

func (q *Queue) Stop(ctx context.Context) error {
	logger.Info(ctx, "stopping rabbit queue")

	err := q.connection.Close()
	if err != nil {
		return err
	}

	logger.Info(ctx, "stopped rabbit queue")
	return nil
}
