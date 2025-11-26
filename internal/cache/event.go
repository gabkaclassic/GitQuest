package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/gabkaclassic/GitQuest/internal/dto"
)

const (
	eventKeyPrefix = "event"
)

type EventCacheClient interface {
	CleanupOldEventsFromCache(context.Context) error
	SaveNewEvents(context.Context, *[]dto.Event) error
}

type eventCacheClient struct {
	storage *redis.Client
}

func NewEventCacheClient(storage *redis.Client) (EventCacheClient, error) {

	if storage == nil {
		return nil, errors.New("create new event cache client failed: cache connection is nil")
	}

	return &eventCacheClient{
		storage: storage,
	}, nil
}

func (client *eventCacheClient) CleanupOldEventsFromCache(ctx context.Context) error {

	_, err := client.storage.ZRemRangeByScore(
		ctx, fmt.Sprintf("%s:*:*", eventKeyPrefix),
		"0", strconv.FormatInt(time.Now().Add(-dto.MaxTimeRange).Unix(), 10),
	).Result()

	return err
}

func (client *eventCacheClient) SaveNewEvents(ctx context.Context, events *[]dto.Event) error {
	pipeline := client.storage.Pipeline()

	for _, event := range *events {
		pipeline.ZAdd(
			ctx, fmt.Sprintf("%s:%s:%s", eventKeyPrefix, event.Actor, event.EventType),
			redis.Z{
				Score:  float64(event.Timestamp.Unix()),
				Member: event.ID,
			},
		)
	}

	_, err := pipeline.Exec(ctx)

	return err
}
