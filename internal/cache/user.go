package cache

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const (
	userKeyPrefix = "user"
	allUsersKey   = "user:*"
)

type UserCacheClient interface {
	Save(context.Context, string) error
	SaveAll(context.Context, *[]string) error
	GetAll(context.Context) (*[]string, error)
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

func (client *userCacheClient) SaveAll(ctx context.Context, users *[]string) error {

	if users == nil {
		return errors.New("users cannot be nil")
	}

	pipeline := client.storage.TxPipeline()

	for _, user := range *users {
		key := fmt.Sprintf("%s:%s", userKeyPrefix, user)
		pipeline.Set(ctx, key, user, 0)
	}
	_, err := pipeline.Exec(ctx)

	return err
}

func (client *userCacheClient) Save(ctx context.Context, user string) error {

	_, err := client.storage.Set(
		ctx,
		fmt.Sprintf("%s:%s", userKeyPrefix, user),
		true, 0,
	).Result()

	return err
}

func (client *userCacheClient) GetAll(ctx context.Context) (*[]string, error) {
	var users []string

	keys, err := client.storage.Keys(ctx, allUsersKey).Result()
	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return &users, nil
	}

	values, err := client.storage.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	for _, value := range values {
		if str, ok := value.(string); ok {
			users = append(users, str)
		}
	}

	return &users, nil
}
