package queue

import "context"

type Queue interface {
	NewConsumer(queueName string, consumerTag string, routingKey string) QueueConsumer
	NewPublisher() QueueSender
}

type QueueConsumer interface {
	ReceiveData(ctx context.Context) (<-chan []byte, error)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type QueueSender interface {
	SendData(ctx context.Context, routingKey string, data []byte) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
