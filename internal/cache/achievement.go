package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/redis/go-redis/v9"
)

const (
	achievementKeyPrefix = "achievement"
)

type AchievementCacheClient interface {
	SaveAll(context.Context, []dto.Achievement) error
	AchievementExists(context.Context, *dto.Achievement) (bool, error)
}

type achievementCacheClient struct {
	storage *redis.Client
}

func NewAchievementCacheClient(storage *redis.Client) (AchievementCacheClient, error) {

	if storage == nil {
		return nil, errors.New("create new achievement cache client failed: cache connection is nil")
	}

	return &achievementCacheClient{
		storage: storage,
	}, nil
}

func (client *achievementCacheClient) SaveAll(ctx context.Context, achievements []dto.Achievement) error {

	if achievements == nil {
		return errors.New("achievements cannot be nil")
	}

	pipeline := client.storage.TxPipeline()

	for _, achievement := range achievements {
		key := fmt.Sprintf("%s:%s:%s:%s", achievementKeyPrefix, achievement.User, achievement.RuleName, achievement.RuleVersion)
		pipeline.ZAdd(
			ctx, key, redis.Z{
				Score:  float64(achievement.EndRange.Unix()),
				Member: achievement.EndRange.Unix(),
			},
		)
	}
	_, err := pipeline.Exec(ctx)

	return err
}

func (client *achievementCacheClient) AchievementExists(ctx context.Context, achievement *dto.Achievement) (bool, error) {

	if achievement == nil {
		return false, errors.New("achievement cannot be nil")
	}

	key := fmt.Sprintf("%s:%s:%s:%s", achievementKeyPrefix, achievement.User, achievement.RuleName, achievement.RuleVersion)

	var minScore, maxScore string

	if achievement.StartRange.Equal(achievement.EndRange) {
		minScore = "-inf"
		maxScore = "+inf"
	} else {
		minScore = strconv.FormatInt(achievement.StartRange.Unix(), 10)
		maxScore = strconv.FormatInt(achievement.EndRange.Unix(), 10)
	}

	vals, err := client.storage.ZRangeByScoreWithScores(
		ctx, key, &redis.ZRangeBy{
			Min: minScore,
			Max: maxScore,
		},
	).Result()

	if err == redis.Nil {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return len(vals) > 0, nil
}
