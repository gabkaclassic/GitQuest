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
	SaveNewEvents(context.Context, []dto.Event) ([]string, error)
	GetUserEventsTimestampsByTypeAndRange(context.Context, string, dto.EventType, time.Time, time.Time) ([]time.Time, error)
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
	var keys []string
	iter := client.storage.Scan(ctx, 0, fmt.Sprintf("%s:*:*", eventKeyPrefix), 0).Iterator()

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return err
	}

	for _, key := range keys {
		_, err := client.storage.ZRemRangeByScore(
			ctx, key,
			"0", strconv.FormatInt(time.Now().Add(-dto.MaxTimeRange).Unix(), 10),
		).Result()
		if err != nil {
			return err
		}
	}

	return nil
}

func (client *eventCacheClient) SaveNewEvents(ctx context.Context, events []dto.Event) ([]string, error) {

	if events == nil {
		return nil, errors.New("events cannot be nil")
	}

	usersSet := make(map[string]bool)
	users := make([]string, 0)
	pipeline := client.storage.TxPipeline()
	for _, event := range events {
		pipeline.ZAdd(
			ctx, fmt.Sprintf("%s:%s:%s", eventKeyPrefix, event.Actor, event.EventType),
			redis.Z{
				Score:  float64(event.Timestamp.Unix()),
				Member: float64(event.Timestamp.Unix()),
			},
		)
		if _, exists := usersSet[event.Actor]; !exists {
			users = append(users, event.Actor)
			usersSet[event.Actor] = true
		}
	}

	_, err := pipeline.Exec(ctx)

	return users, err
}

func (client *eventCacheClient) GetUserEventsTimestampsByTypeAndRange(ctx context.Context, user string, eventType dto.EventType, startRange, endRange time.Time) ([]time.Time, error) {
	var timestamps []time.Time

	var minScore, maxScore string
	if startRange.Equal(endRange) {
		minScore = "-inf"
		maxScore = "+inf"
	} else {
		minScore = strconv.FormatInt(startRange.Unix(), 10)
		maxScore = strconv.FormatInt(endRange.Unix(), 10)
	}

	timestampStrs, err := client.storage.ZRangeByScore(
		ctx,
		fmt.Sprintf("%s:%s:%s", eventKeyPrefix, user, eventType),
		&redis.ZRangeBy{
			Min: minScore,
			Max: maxScore,
		},
	).Result()
	if err != nil {
		return nil, err
	}

	for _, timestampStr := range timestampStrs {
		unixTime, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse timestamp: %w", err)
		}
		timestamps = append(timestamps, time.Unix(unixTime, 0))
	}

	return timestamps, nil
}
