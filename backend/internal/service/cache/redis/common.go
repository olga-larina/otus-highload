package redis_cache

import (
	"context"
)

type LoadFunc func(ctx context.Context, key string) (any, error)
