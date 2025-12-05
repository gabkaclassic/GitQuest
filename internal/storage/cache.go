package storage

import (
	"context"
	"github.com/gabkaclassic/GitQuest/internal/config"
	"github.com/redis/go-redis/v9"
)

func NewCacheStorage(cfg config.Cache) (*redis.Client, error) {

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		Username: cfg.Username,
		DB:       cfg.DB,
	})

	ctx := context.Background()
	_, err := client.Ping(ctx).Result()

	if err != nil {
		return nil, err
	}

	return client, err
}
