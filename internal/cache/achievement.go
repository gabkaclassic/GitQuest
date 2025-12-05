package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/redis/go-redis/v9"
)

const (
	achievementKeyPrefix = "achievement"
	summaryKeyPrefix     = "summary"

	userSummaryTTL = 5 * time.Minute
)

type AchievementCacheClient interface {
	SaveAll(context.Context, []dto.Achievement) error
	AchievementExists(context.Context, *dto.Achievement) (bool, error)
	GetUserAchievementsSummary(context.Context, string) (*dto.AchievementsSummary, error)
	SetUserAchievementsSummary(context.Context, string, *dto.AchievementsSummary) error
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

func (client *achievementCacheClient) SetUserAchievementsSummary(ctx context.Context, user string, summary *dto.AchievementsSummary) error {
	key := fmt.Sprintf("%s:%s", summaryKeyPrefix, user)

	data, err := json.Marshal(summary)

	if err != nil {
		return err
	}

	_, err = client.storage.SetEx(ctx, key, data, userSummaryTTL).Result()

	if err != nil {
		return err
	}

	return nil
}

func (client *achievementCacheClient) GetUserAchievementsSummary(ctx context.Context, user string) (*dto.AchievementsSummary, error) {

	key := fmt.Sprintf("%s:%s", summaryKeyPrefix, user)

	data, err := client.storage.Get(ctx, key).Result()

	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}

		return nil, err
	}

	var summary dto.AchievementsSummary
	err = json.Unmarshal([]byte(data), &summary)

	if err != nil {
		return nil, err
	}

	return &summary, nil
}

func (client *achievementCacheClient) SaveAll(ctx context.Context, achievements []dto.Achievement) error {

	if achievements == nil {
		return errors.New("achievements cannot be nil")
	}

	pipeline := client.storage.TxPipeline()

	for _, achievement := range achievements {
		achievementKey := fmt.Sprintf("%s:%s:%s", achievementKeyPrefix, achievement.User, achievement.RuleName)

		pipeline.ZAdd(
			ctx, achievementKey, redis.Z{
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

	key := fmt.Sprintf("%s:%s:%s", achievementKeyPrefix, achievement.User, achievement.RuleName)

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
