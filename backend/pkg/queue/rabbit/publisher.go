package rabbit

import (
	"context"

	"github.com/olga-larina/otus-highload/pkg/logger"
	"github.com/olga-larina/otus-highload/pkg/queue"
	"github.com/olga-larina/otus-highload/pkg/tracing"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
)

type Publisher struct {
	queue   *Queue
	channel *amqp.Channel
}

func (q *Queue) NewPublisher() queue.QueueSender {
	return &Publisher{
		queue: q,
	}
}

func (p *Publisher) Start(ctx context.Context) error {
	logger.Info(ctx, "starting rabbit producer")

	var err error

	p.channel, err = p.queue.connection.Channel()
	if err != nil {
		return err
	}
	logger.Info(ctx, "got rabbit producer channel, declaring exchange", "exchangeName", p.queue.exchangeName, "exchangeType", p.queue.exchangeType)

	err = p.channel.ExchangeDeclare(
		p.queue.exchangeName, // name
		p.queue.exchangeType, // type
		true,                 // durable
		false,                // autoDelete
		false,                // internal
		false,                // noWait
		nil,                  // arguments
	)
	if err != nil {
		return err
	}
	logger.Info(ctx, "rabbit producer exchange declared")
	return nil
}

func (p *Publisher) Stop(ctx context.Context) error {
	logger.Info(ctx, "stopping rabbit producer")

	err := p.channel.Close()
	if err != nil {
		return err
	}

	logger.Info(ctx, "stopped rabbit producer")
	return nil
}

func (p *Publisher) SendData(ctx context.Context, routingKey string, data []byte) error {
	ctxWithSpan, span := otel.Tracer("default").Start(ctx, "publish queue "+routingKey)
	defer span.End()
	headers := make(amqp.Table)
	headers[queue.TraceIdHeader] = tracing.GetTraceId(ctxWithSpan)
	headers[queue.SpanIdHeader] = tracing.GetSpanId(ctxWithSpan)
	headers[queue.RequestIdHeader] = tracing.GetRequestId(ctxWithSpan)
	return p.channel.PublishWithContext(
		ctxWithSpan,
		p.queue.exchangeName,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         data,
			DeliveryMode: amqp.Persistent,
			Headers:      headers,
		},
	)
}
