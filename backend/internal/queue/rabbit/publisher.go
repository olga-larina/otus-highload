package rabbit

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	queue *Queue
}

func (q *Queue) NewPublisher() *Publisher {
	return &Publisher{
		queue: q,
	}
}

func (q *Publisher) SendData(ctx context.Context, data []byte) error {
	return q.queue.channel.PublishWithContext(
		ctx,
		q.queue.exchangeName,
		q.queue.routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         data,
			DeliveryMode: amqp.Persistent,
		},
	)
}
