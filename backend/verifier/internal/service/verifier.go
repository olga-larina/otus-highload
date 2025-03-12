package service

import (
	"context"
	"fmt"

	"github.com/olga-larina/otus-highload/pkg/logger"
	"github.com/olga-larina/otus-highload/pkg/queue"
	"go.opentelemetry.io/otel"
)

type Processor interface {
	ProcessEvent(ctx context.Context, eventBytes []byte)
}

type VerifierService struct {
	userCreatedConsumer queue.QueueConsumer // получение событий о создании пользователей
	processor           Processor           // обработка событий о создании пользователей
	done                chan struct{}
}

func NewVerifierService(
	userCreatedQueue queue.Queue,
	userCreatedQueueName string,
	userCreatedConsumerTag string,
	userCreatedConsumerRoutingKey string,
	processor Processor,
	serviceId string,
) *VerifierService {
	return &VerifierService{
		userCreatedConsumer: userCreatedQueue.NewConsumer(
			userCreatedQueueName,
			fmt.Sprintf(userCreatedConsumerTag, serviceId),
			userCreatedConsumerRoutingKey,
		),
		processor: processor,
		done:      make(chan struct{}),
	}
}

func (s *VerifierService) Start(ctx context.Context) error {
	logger.Info(ctx, "starting verifierService")

	err := s.userCreatedConsumer.Start(ctx)
	if err != nil {
		return err
	}

	err = s.processEvents(ctx)
	if err != nil {
		return err
	}

	logger.Info(ctx, "started verifierService")
	return nil
}

func (s *VerifierService) Stop(ctx context.Context) error {
	logger.Info(ctx, "stopping verifierService")

	err := s.userCreatedConsumer.Stop(ctx)
	if err != nil {
		logger.Error(ctx, err, "failed to stop verifierService consumer")
	}

	<-ctx.Done()
	<-s.done

	logger.Info(ctx, "stopped verifierService")
	return nil
}

func (s *VerifierService) processEvents(pctx context.Context) error {
	data, err := s.userCreatedConsumer.ReceiveData(pctx)
	if err != nil {
		defer close(s.done)
		return err
	}

	go func() {
		defer close(s.done)

		for msg := range data {
			ctxInitial := queue.GetContext(pctx, msg)

			ctx, span := otel.Tracer("default").Start(ctxInitial, "consume userCreated queue")
			s.processor.ProcessEvent(ctx, msg.Body)
			span.End()
		}
	}()

	return nil
}
