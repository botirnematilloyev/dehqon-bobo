package cache

import (
	"context"
	"dehqon-bobo/internal/platform/config"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func New(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("cache: cannot connect: %w", err)
	}

	return client, nil
}
