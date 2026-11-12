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

	summaryVersionKeyPrefix = "summary_version"

	userSummaryTTL = 5 * time.Minute
)

type AchievementCacheClient interface {
	SaveAll(context.Context, []dto.Achievement) error
	AchievementExists(context.Context, *dto.Achievement) (bool, error)
	GetUserAchievementsSummary(context.Context, string) (*dto.AchievementsSummary, error)
	GetUserAchievementsSummaryVersion(ctx context.Context, user string) (int64, error)
	SetUserAchievementsSummary(ctx context.Context, user string, summary *dto.AchievementsSummary, version int64) error
	InvalidateUserAchievementsSummaries(ctx context.Context, users []string) error
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

const setSummaryIfVersionScript = `if (redis.call('GET', KEYS[2]) or '0') == ARGV[2] then redis.call('SET', KEYS[1], ARGV[1], 'EX', ARGV[3]) return 1 end return 0`

func (client *achievementCacheClient) SetUserAchievementsSummary(ctx context.Context, user string, summary *dto.AchievementsSummary, version int64) error {
	data, err := json.Marshal(summary)

	if err != nil {
		return err
	}

	keys := []string{
		fmt.Sprintf("%s:%s", summaryKeyPrefix, user),
		fmt.Sprintf("%s:%s", summaryVersionKeyPrefix, user),
	}

	err = client.storage.Eval(
		ctx,
		setSummaryIfVersionScript,
		keys,
		data,
		strconv.FormatInt(version, 10),
		int64(userSummaryTTL/time.Second),
	).Err()

	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}

	return nil
}

func (client *achievementCacheClient) GetUserAchievementsSummaryVersion(ctx context.Context, user string) (int64, error) {
	key := fmt.Sprintf("%s:%s", summaryVersionKeyPrefix, user)

	value, err := client.storage.Get(ctx, key).Result()

	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}

		return 0, err
	}

	return strconv.ParseInt(value, 10, 64)
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

func (client *achievementCacheClient) InvalidateUserAchievementsSummaries(ctx context.Context, users []string) error {
	if len(users) == 0 {
		return nil
	}

	pipeline := client.storage.TxPipeline()

	for _, user := range users {
		pipeline.Incr(ctx, fmt.Sprintf("%s:%s", summaryVersionKeyPrefix, user))
		pipeline.Del(ctx, fmt.Sprintf("%s:%s", summaryKeyPrefix, user))
	}

	_, err := pipeline.Exec(ctx)

	return err
}
