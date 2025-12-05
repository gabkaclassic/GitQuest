package cache

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
)

const (
	usersKey = "user"
)

type UserCacheClient interface {
	Save(context.Context, string) error
	SaveAll(context.Context, []string) error
	GetAll(context.Context) ([]string, error)
}

type userCacheClient struct {
	storage *redis.Client
}

func NewUserCacheClient(storage *redis.Client) (UserCacheClient, error) {

	if storage == nil {
		return nil, errors.New("create new user cache client failed: cache connection is nil")
	}

	return &userCacheClient{
		storage: storage,
	}, nil
}

func (client *userCacheClient) SaveAll(ctx context.Context, users []string) error {

	if users == nil {
		return errors.New("users cannot be nil")
	}

	pipeline := client.storage.TxPipeline()

	for _, user := range users {
		pipeline.ZAdd(ctx, usersKey, redis.Z{
			Member: user,
		})
	}
	_, err := pipeline.Exec(ctx)

	return err
}

func (client *userCacheClient) Save(ctx context.Context, user string) error {

	_, err := client.storage.ZAdd(ctx, usersKey, redis.Z{
		Member: user,
	}).Result()

	return err
}

func (client *userCacheClient) GetAll(ctx context.Context) ([]string, error) {

	users, err := client.storage.ZRange(ctx, usersKey, 0, -1).Result()

	if err != nil {
		return nil, err
	}

	return users, nil
}
