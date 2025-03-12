package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/olga-larina/otus-highload/pkg/logger"
	"github.com/olga-larina/otus-highload/pkg/model"
	"github.com/olga-larina/otus-highload/pkg/queue"
)

type VerifierProcessorService struct {
	verifierPublisher  queue.QueueSender // отправка событий о результате верификации пользователя
	verifierRoutingKey string            // routingKey для отправки событий о результате верификации пользователя
	done               chan struct{}
}

func NewVerifierProcessorService(
	verifierQueue queue.Queue,
	verifierRoutingKey string,
	serviceId string,
) *VerifierProcessorService {
	return &VerifierProcessorService{
		verifierPublisher:  verifierQueue.NewPublisher(),
		verifierRoutingKey: verifierRoutingKey,
		done:               make(chan struct{}),
	}
}

func (s *VerifierProcessorService) Start(ctx context.Context) error {
	logger.Info(ctx, "starting verifierProcessorService")

	err := s.verifierPublisher.Start(ctx)
	if err != nil {
		return err
	}

	logger.Info(ctx, "started verifierProcessorService")
	return nil
}

func (s *VerifierProcessorService) Stop(ctx context.Context) error {
	logger.Info(ctx, "stopping verifierProcessorService")

	err := s.verifierPublisher.Stop(ctx)
	if err != nil {
		logger.Error(ctx, err, "failed to stop verifierProcessorService success publisher")
	}

	<-ctx.Done()
	<-s.done

	logger.Info(ctx, "stopped verifierProcessorService")
	return nil
}

func (s *VerifierProcessorService) ProcessEvent(ctx context.Context, eventBytes []byte) {
	var event model.SagaEventJson
	err := json.Unmarshal(eventBytes, &event)
	if err != nil {
		logger.Error(ctx, err, "failed to read event")
		s.processSagaEvent(ctx, model.SagaEvent{Type: model.UserVerificationFailedSagaEventType})
	} else {
		logger.Info(ctx, "received event", "event", event)

		// имитируем ошибку в 50% случаев, в payload копируем payload входящего события
		if time.Now().Unix()%2 == 0 {
			logger.Error(ctx, errors.New("failed processing event"), "failed processing event", "event", event)
			s.processSagaEvent(ctx, model.SagaEvent{Type: model.UserVerificationFailedSagaEventType, Payload: event.Payload})
		} else {
			logger.Debug(ctx, "succeeded processing event", "event", event)
			s.processSagaEvent(ctx, model.SagaEvent{Type: model.UserVerifiedSagaEventType, Payload: event.Payload})
		}
	}
}

func (s *VerifierProcessorService) processSagaEvent(ctx context.Context, event model.SagaEvent) {
	eventStr, err := json.Marshal(event)
	if err != nil {
		logger.Error(
			ctx, err, "failed processing event",
			"stage", "marshal",
			"event", &event,
		)
		return
	}
	err = s.verifierPublisher.SendData(ctx, s.verifierRoutingKey, eventStr)
	if err != nil {
		logger.Error(
			ctx, err, "failed notifying event",
			"stage", "send",
			"event", &event,
		)
		return
	}
	logger.Debug(ctx, "sent verifier saga event", "event", event)
}
