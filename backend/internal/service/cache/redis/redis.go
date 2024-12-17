package redis_cache

import (
	"context"
	"time"

	"github.com/go-redsync/redsync/v4"
	redsyncRedis "github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/olga-larina/otus-highload/backend/internal/logger"
	"github.com/redis/go-redis/v9"
)

// redis кэш
type RedisCache struct {
	// клиент redis
	redisClient *redis.Client
	// для реализации блокировок
	rs *redsync.Redsync
	// время жизни
	ttl time.Duration
	// преобразование значения в строку и обратно
	valueConverter ValueConverter
	cacheName      string
}

// преобразование значения в строку и обратно
type ValueConverter interface {
	ConvertToString(value any) (string, error)
	ConvertFromString(valueStr string) (any, error)
}

func NewRedisCache(opt *redis.Options, ttl time.Duration, valueConverter ValueConverter, cacheName string) (*RedisCache, error) {
	redisClient := redis.NewClient(opt)
	pool := redsyncRedis.NewPool(redisClient)
	rs := redsync.New(pool)
	return &RedisCache{
		redisClient:    redisClient,
		rs:             rs,
		ttl:            ttl,
		valueConverter: valueConverter,
		cacheName:      cacheName,
	}, nil
}

func (r *RedisCache) InvalidateAll(ctx context.Context) error {
	_, err := r.redisClient.FlushDB(ctx).Result()
	return err
}

func (r *RedisCache) Load(ctx context.Context, key string, loadFunc LoadFunc) (any, error) {
	var err error
	// блокировка доступа к определённому ключу
	mx := r.rs.NewMutex(key+"-lock",
		redsync.WithExpiry(5*time.Second),     // Срок действия блокировки
		redsync.WithTries(3),                  // Количество попыток захвата блокировки
		redsync.WithRetryDelay(2*time.Second), // Задержка между попытками
	)
	if err = mx.LockContext(ctx); err != nil {
		logger.Error(ctx, err, "failed obtaining lock", "cacheName", r.cacheName, "key", key)
		return nil, err
	}
	defer func() {
		_, err := mx.UnlockContext(ctx)
		if err != nil {
			logger.Error(ctx, err, "failed releasing lock", "cacheName", r.cacheName, "key", key)
		}
	}()

	logger.Debug(ctx, "loading value", "cacheName", r.cacheName, "key", key)
	var valueLoaded any
	valueLoaded, err = loadFunc(ctx, key)
	if err != nil {
		logger.Error(ctx, err, "failed loading value", "cacheName", r.cacheName, "key", key)
		return nil, err
	}
	err = r.add(ctx, key, valueLoaded)
	if err != nil {
		logger.Error(ctx, err, "failed saving value to cache", "cacheName", r.cacheName, "key", key)
	}

	return valueLoaded, err
}

func (r *RedisCache) GetOrLoad(ctx context.Context, key string, loadFunc LoadFunc) (any, error) {
	value, ok, err := r.get(ctx, key)
	if err != nil {
		return nil, err
	}
	if ok {
		logger.Debug(ctx, "returning value from cache", "cacheName", r.cacheName, "key", key)
		return value, nil
	}
	return r.Load(ctx, key, loadFunc)
}

// получить данные
func (r *RedisCache) get(ctx context.Context, key string) (any, bool, error) {
	logger.Debug(ctx, "obtaining value from cache", "cacheName", r.cacheName, "key", key)

	valueStr, err := r.redisClient.Get(ctx, key).Result()
	if err == redis.Nil { // значения нет в кэше
		return nil, false, nil
	} else if err != nil { // ошибка
		logger.Error(ctx, err, "failed returning value from cache", "cacheName", r.cacheName, "key", key)
		return nil, false, err
	} else { // значение есть, пытаемся преобразовать
		value, err := r.valueConverter.ConvertFromString(valueStr)
		if err != nil {
			logger.Error(ctx, err, "failed parsing value from cache", "cacheName", r.cacheName, "key", key)
			return nil, true, err
		}
		return value, true, nil
	}
}

// записать значение
func (r *RedisCache) add(ctx context.Context, key string, value any) error {
	logger.Debug(ctx, "adding value to cache", "cacheName", r.cacheName, "key", key)

	valueStr, err := r.valueConverter.ConvertToString(value)
	if err != nil {
		logger.Error(ctx, err, "failed converting value to string", "cacheName", r.cacheName, "key", key)
		return err
	}

	_, err = r.redisClient.Set(ctx, key, valueStr, r.ttl).Result()
	return err
}
