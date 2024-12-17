package rabbit

import (
	"context"

	"github.com/olga-larina/otus-highload/backend/internal/logger"
)

type Consumer struct {
	queue       *Queue
	consumerTag string
}

func (q *Queue) NewConsumer(consumerTag string) *Consumer {
	return &Consumer{
		queue:       q,
		consumerTag: consumerTag,
	}
}

func (c *Consumer) Stop(ctx context.Context) error {
	logger.Info(ctx, "stopping rabbit consumer", "consumerTag", c.consumerTag)

	err := c.queue.channel.Cancel(c.consumerTag, true)
	if err != nil {
		return err
	}

	logger.Info(ctx, "stopped rabbit consumer")
	return nil
}

func (c *Consumer) ReceiveData(_ context.Context) (<-chan []byte, error) {
	deliveries, err := c.queue.channel.Consume(
		c.queue.queueName,
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

	result := make(chan []byte)
	go func() {
		defer close(result)
		for d := range deliveries {
			result <- d.Body
		}
	}()
	return result, nil
}
