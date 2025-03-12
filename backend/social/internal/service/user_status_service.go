package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/olga-larina/otus-highload/pkg/logger"
	pkg_model "github.com/olga-larina/otus-highload/pkg/model"
	"github.com/olga-larina/otus-highload/pkg/queue"
	"github.com/olga-larina/otus-highload/social/internal/model"
	"go.opentelemetry.io/otel"
)

type UserStatusStorage interface {
	UpdateUserStatus(ctx context.Context, id *model.UserId, status *model.UserStatus) error
}

type UserStatusService struct {
	verifierStatusConsumer queue.QueueConsumer // получение событий о статусе верификации
	storage                UserStatusStorage
	done                   chan struct{}
}

func NewUserStatusService(
	sagaVerifierStatusQueue queue.Queue,
	sagaVerifierStatusQueueName string,
	sagaVerifierStatusConsumerTag string,
	sagaVerifierStatusConsumerRoutingKey string,
	storage UserStatusStorage,
	serviceId string,
) *UserStatusService {
	return &UserStatusService{
		verifierStatusConsumer: sagaVerifierStatusQueue.NewConsumer(
			sagaVerifierStatusQueueName,
			fmt.Sprintf(sagaVerifierStatusConsumerTag, serviceId),
			sagaVerifierStatusConsumerRoutingKey,
		),
		storage: storage,
		done:    make(chan struct{}),
	}
}

func (s *UserStatusService) Start(ctx context.Context) error {
	logger.Info(ctx, "starting userStatusService")

	err := s.verifierStatusConsumer.Start(ctx)
	if err != nil {
		return err
	}

	err = s.processEvents(ctx)
	if err != nil {
		return err
	}

	logger.Info(ctx, "started userStatusService")
	return nil
}

func (s *UserStatusService) Stop(ctx context.Context) error {
	logger.Info(ctx, "stopping userStatusService")

	err := s.verifierStatusConsumer.Stop(ctx)
	if err != nil {
		logger.Error(ctx, err, "failed to stop userStatusService consumer")
	}

	<-ctx.Done()
	<-s.done

	logger.Info(ctx, "stopped userStatusService")
	return nil
}

func (s *UserStatusService) processEvents(pctx context.Context) error {
	data, err := s.verifierStatusConsumer.ReceiveData(pctx)
	if err != nil {
		defer close(s.done)
		return err
	}

	go func() {
		defer close(s.done)

		for msg := range data {
			ctxInitial := queue.GetContext(pctx, msg)

			ctx, span := otel.Tracer("default").Start(ctxInitial, "consume verifierStatus queue")
			s.processEvent(ctx, msg.Body)
			span.End()
		}
	}()

	return nil
}

func (s *UserStatusService) processEvent(ctx context.Context, eventBytes []byte) {
	var event pkg_model.SagaEventJson
	err := json.Unmarshal(eventBytes, &event)
	if err != nil {
		logger.Error(ctx, err, "failed to read event")
	} else {
		logger.Info(ctx, "received event", "event", event)

		if event.Type == pkg_model.UserVerifiedSagaEventType || event.Type == pkg_model.UserVerificationFailedSagaEventType {
			var payload model.User
			err = json.Unmarshal([]byte(event.Payload), &payload)
			if err == nil {
				var userStatus model.UserStatus
				if event.Type == pkg_model.UserVerifiedSagaEventType {
					userStatus = model.UserVerificationSucceeded
				} else {
					userStatus = model.UserVerificationFailed
				}
				err = s.storage.UpdateUserStatus(ctx, payload.Id, &userStatus)
			}
		} else {
			err = fmt.Errorf("unknown event type %s", event.Type)
		}

		if err != nil {
			logger.Error(ctx, err, "failed processing saga event", "event", event)
			// для повторов при ошибке можно добавить очередь ретраев
		} else {
			logger.Debug(ctx, "succeeded processing saga event", "event", event)
		}
	}
}
