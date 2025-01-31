package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/olga-larina/otus-highload/pkg/logger"
	"github.com/olga-larina/otus-highload/social/internal/model"
	"github.com/olga-larina/otus-highload/social/internal/queue"
	"github.com/pckilgore/combuuid"
	"go.opentelemetry.io/otel"
)

type PostFeedUserSubscriber struct {
	postsQueue                    queue.Queue
	postFeedUserUpdateQueueName   string
	postFeedUserUpdateConsumerTag string
	postFeedUserUpdateRoutingKey  string
	serviceId                     string
	consumers                     map[model.UserId]queue.QueueConsumer
	activeSubscribers             map[model.UserId]map[string]chan []model.Post
	mu                            *sync.Mutex
}

func NewPostFeedUserSubscriber(
	postsQueue queue.Queue,
	postFeedUserUpdateQueueName string,
	postFeedUserUpdateConsumerTag string,
	postFeedUserUpdateRoutingKey string,
	serviceId string,
) *PostFeedUserSubscriber {
	return &PostFeedUserSubscriber{
		postsQueue:                    postsQueue,
		postFeedUserUpdateQueueName:   postFeedUserUpdateQueueName,
		postFeedUserUpdateConsumerTag: postFeedUserUpdateConsumerTag,
		postFeedUserUpdateRoutingKey:  postFeedUserUpdateRoutingKey,
		serviceId:                     serviceId,
		consumers:                     make(map[model.UserId]queue.QueueConsumer),
		activeSubscribers:             make(map[model.UserId]map[string]chan []model.Post),
		mu:                            &sync.Mutex{},
	}
}

// сюда нужно передавать сигнал о завершении!
func (s *PostFeedUserSubscriber) SubscribePostFeed(ctx context.Context, userId model.UserId) (string, <-chan []model.Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// получаем консьюмера
	// если его нет, то запускаем
	_, consumerExists := s.consumers[userId]
	if !consumerExists {
		consumer, err := s.prepareConsumer(ctx, userId)
		if err != nil {
			return "", nil, err
		}
		s.consumers[userId] = consumer
	}

	// создаём новый канал и подписку
	subscriptionId := combuuid.NewUuid().String()
	postChan := make(chan []model.Post)
	activeSubscribersByUserId, ok := s.activeSubscribers[userId]
	if !ok {
		activeSubscribersByUserId = make(map[string]chan []model.Post)
		s.activeSubscribers[userId] = activeSubscribersByUserId
	}
	activeSubscribersByUserId[subscriptionId] = postChan

	logger.Debug(ctx, "subsribed post feed", "userId", userId, "subscriptionId", subscriptionId)

	return subscriptionId, postChan, nil
}

func (s *PostFeedUserSubscriber) prepareConsumer(pctx context.Context, userId model.UserId) (queue.QueueConsumer, error) {
	var err error

	// создаём и запускаем консьюмера
	consumer := s.postsQueue.NewConsumer(
		fmt.Sprintf(s.postFeedUserUpdateQueueName, userId, s.serviceId),
		fmt.Sprintf(s.postFeedUserUpdateConsumerTag, userId, s.serviceId),
		fmt.Sprintf(s.postFeedUserUpdateRoutingKey, userId),
	)
	err = consumer.Start(pctx)
	if err != nil {
		return nil, err
	}

	// получаем данные из консьюмера
	data, err := consumer.ReceiveData(pctx)
	if err != nil {
		if err := consumer.Stop(pctx); err != nil {
			logger.Error(pctx, err, "failed stopping consumer")
		}
		return nil, err
	}

	go func() {
		defer func() {
			s.mu.Lock()
			defer s.mu.Unlock()

			// проходимся по всем активным подписчикам, закрываем и удаляем их каналы
			activeSubscribersByUserId, ok := s.activeSubscribers[userId]
			if ok {
				for _, subscriber := range activeSubscribersByUserId {
					close(subscriber)
				}
				delete(s.activeSubscribers, userId)
			}

			// проверяем, не удалили ли уже консьюмера
			_, ok = s.consumers[userId]
			if !ok {
				return
			}

			// останавливаем и удаляем консьюмера
			if err := consumer.Stop(pctx); err != nil {
				logger.Error(pctx, err, "failed stopping consumer")
			}
			delete(s.consumers, userId)
		}()

		for msg := range data {
			ctxInitial := queue.GetContext(pctx, msg)

			ctx, span := otel.Tracer("default").Start(ctxInitial, "consume queue "+"postFeed")

			// получаем и преобразуем данные
			var notification model.UserPostFeedNotification
			err := json.Unmarshal(msg.Body, &notification)
			if err != nil {
				logger.Error(ctx, err, "failed unmarshalling notification", "notification", string(msg.Body))
			} else {
				// проходимся по всем активным подписчикам, направляем в их каналы информацию
				activeSubscribersByUserId, ok := s.activeSubscribers[userId]
				if ok {
					for _, subscriber := range activeSubscribersByUserId {
						posts := make([]model.Post, len(notification.Posts))
						for i, post := range notification.Posts {
							posts[i] = post.Post
						}
						subscriber <- posts
					}
				}
			}

			span.End()
		}
	}()

	logger.Debug(pctx, "consumer created", "userId", userId)
	return consumer, nil
}

func (s *PostFeedUserSubscriber) UnsubscribePostFeed(ctx context.Context, userId model.UserId, subscriptionId string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// ищем подписку, удаляем при наличии
	activeSubscribersByUserId, ok := s.activeSubscribers[userId]
	if !ok {
		return nil
	}
	postChan, ok := activeSubscribersByUserId[subscriptionId]
	if !ok {
		return nil
	}
	close(postChan)
	delete(activeSubscribersByUserId, subscriptionId)

	// проверяем, остались ли ещё подписчики
	// если нет, то останавливаем консьюмера
	if len(activeSubscribersByUserId) > 0 {
		return nil
	}

	delete(s.activeSubscribers, userId)

	consumer := s.consumers[userId]
	delete(s.consumers, userId)

	logger.Debug(ctx, "unsubsribed post feed", "userId", userId, "subscriptionId", subscriptionId)

	return consumer.Stop(ctx)
}
