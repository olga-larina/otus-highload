package queue

import (
	"context"

	"github.com/olga-larina/otus-highload/pkg/logger"
	"github.com/olga-larina/otus-highload/pkg/tracing"
)

const (
	TraceIdHeader   = "trace-id"
	SpanIdHeader    = "span-id"
	RequestIdHeader = "x-request-id"
)

type Queue interface {
	NewConsumer(queueName string, consumerTag string, routingKey string) QueueConsumer
	NewPublisher() QueueSender
}

type QueueConsumer interface {
	ReceiveData(ctx context.Context) (<-chan *ConsumerMessage, error)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type QueueSender interface {
	SendData(ctx context.Context, routingKey string, data []byte) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type ConsumerMessage struct {
	Body      []byte
	TraceId   string
	SpanId    string
	RequestId string
}

func GetContext(ctx context.Context, msg *ConsumerMessage) context.Context {
	ctx, err := tracing.GetContext(ctx, msg.TraceId, msg.SpanId, msg.RequestId)
	if err != nil {
		logger.Error(ctx, err, "error obtaining context by traceId and spanId")
	}
	return ctx
}
