package rabbit

import (
	"context"

	"github.com/olga-larina/otus-highload/pkg/logger"
	"github.com/olga-larina/otus-highload/social/internal/queue"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	queueName   string
	consumerTag string
	routingKey  string
	queue       *Queue
	channel     *amqp.Channel
}

func (q *Queue) NewConsumer(queueName string, consumerTag string, routingKey string) queue.QueueConsumer {
	return &Consumer{
		queueName:   queueName,
		consumerTag: consumerTag,
		routingKey:  routingKey,
		queue:       q,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	logger.Info(ctx, "starting rabbit consumer")

	var err error

	c.channel, err = c.queue.connection.Channel()
	if err != nil {
		return err
	}
	logger.Info(ctx, "got rabbit consumer channel, declaring queue")

	// создаём динамическую очередь
	queue, err := c.channel.QueueDeclare(
		c.queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // noWait
		nil,   // arguments
	)
	if err != nil {
		return err
	}
	logger.Info(ctx, "declared new rabbit queue, declaring binding", "queueName", queue.Name, "routingKey", c.routingKey)

	err = c.channel.QueueBind(
		queue.Name,
		c.routingKey,
		c.queue.exchangeName,
		false, // noWait
		nil,   // arguments
	)
	if err != nil {
		return err
	}

	logger.Info(ctx, "queue bound to exchange, rabbit queue started")
	return nil
}

func (c *Consumer) Stop(ctx context.Context) error {
	logger.Info(ctx, "stopping rabbit consumer", "consumerTag", c.consumerTag)

	err := c.channel.QueueUnbind(
		c.queueName,
		c.routingKey,
		c.queue.exchangeName,
		nil,
	)
	if err != nil {
		return err
	}

	err = c.channel.Cancel(c.consumerTag, true)
	if err != nil {
		return err
	}

	err = c.channel.Close()
	if err != nil {
		return err
	}

	logger.Info(ctx, "stopped rabbit consumer")
	return nil
}

func (c *Consumer) ReceiveData(ctx context.Context) (<-chan *queue.ConsumerMessage, error) {
	deliveries, err := c.channel.Consume(
		c.queueName,
		c.consumerTag,
		true,  // noAck
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,   // arguments
	)
	if err != nil {
		return nil, err
	}

	result := make(chan *queue.ConsumerMessage)
	go func() {
		defer close(result)
		for d := range deliveries {
			result <- GetMessage(ctx, &d)
		}
	}()
	return result, nil
}

func GetMessage(ctx context.Context, msg *amqp.Delivery) *queue.ConsumerMessage {
	return &queue.ConsumerMessage{
		Body:      msg.Body,
		TraceId:   GetHeader(msg, queue.TraceIdHeader),
		SpanId:    GetHeader(msg, queue.SpanIdHeader),
		RequestId: GetHeader(msg, queue.RequestIdHeader),
	}
}

func GetHeader(msg *amqp.Delivery, name string) string {
	for key, value := range msg.Headers {
		if string(key) == name {
			if val, ok := value.(string); ok {
				return val
			}
			return ""
		}
	}
	return ""
}
