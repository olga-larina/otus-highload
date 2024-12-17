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
	queueName    string
	routingKey   string

	connection *amqp.Connection
	channel    *amqp.Channel
}

func NewQueue(uri string, exchangeName string, exchangeType string, queueName string, routingKey string) *Queue {
	return &Queue{
		uri:          uri,
		exchangeName: exchangeName,
		exchangeType: exchangeType,
		queueName:    queueName,
		routingKey:   routingKey,
	}
}

func (q *Queue) Start(ctx context.Context) error {
	logger.Info(ctx, "starting rabbit queue")

	var err error

	q.connection, err = amqp.Dial(q.uri)
	if err != nil {
		return err
	}
	logger.Info(ctx, "got rabbit queue connection, getting channel")

	q.channel, err = q.connection.Channel()
	if err != nil {
		return err
	}
	logger.Info(ctx, "got rabbit queue channel, declaring exchange", "exchangeName", q.exchangeName, "exchangeType", q.exchangeType)

	err = q.channel.ExchangeDeclare(
		q.exchangeName, // name
		q.exchangeType, // type
		true,           // durable
		false,          // autoDelete
		false,          // internal
		false,          // noWait
		nil,            // arguments
	)
	if err != nil {
		return err
	}
	logger.Info(ctx, "rabbit exchange declared, declaring queue", "queueName", q.queueName)

	queue, err := q.channel.QueueDeclare(
		q.queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // noWait
		nil,   // arguments
	)
	if err != nil {
		return err
	}
	logger.Info(ctx, "declared new rabbit queue, declaring binding", "routingKey", q.routingKey)

	err = q.channel.QueueBind(
		queue.Name,
		q.routingKey,
		q.exchangeName,
		false, // noWait
		nil,   // arguments
	)
	if err != nil {
		return err
	}

	logger.Info(ctx, "queue bound to exchange, rabbit queue started")
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
