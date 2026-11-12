package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestNewAchievementCacheClient(t *testing.T) {
	tests := []struct {
		name        string
		storage     func() *redis.Client
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid redis client",
			storage: func() *redis.Client {
				client, _ := redismock.NewClientMock()
				return client
			},
			expectError: false,
		},
		{
			name:        "nil redis client",
			storage:     func() *redis.Client { return nil },
			expectError: true,
			errorMsg:    "cache connection is nil",
		},
		{
			name: "connected redis client",
			storage: func() *redis.Client {
				client, mock := redismock.NewClientMock()
				mock.ExpectPing().SetVal("PONG")
				return client
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := tt.storage()
			client, err := NewAchievementCacheClient(storage)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.IsType(t, &achievementCacheClient{}, client)
				assert.Equal(t, storage, client.(*achievementCacheClient).storage)
			}
		})
	}
}

func TestAchievementCacheClient_SetUserAchievementsSummary(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &achievementCacheClient{storage: db}
	ctx := context.Background()

	user := "user1"
	summary := &dto.AchievementsSummary{
		Achievements: []dto.AchievementInfo{
			{RuleName: "rule1", Reward: 10},
			{RuleName: "rule2", Reward: 5},
		},
		RewardSum: 15,
	}

	data, _ := json.Marshal(summary)
	keys := []string{
		fmt.Sprintf("%s:%s", summaryKeyPrefix, user),
		fmt.Sprintf("%s:%s", summaryVersionKeyPrefix, user),
	}
	ttl := int64(userSummaryTTL / time.Second)

	tests := []struct {
		name        string
		mockFn      func()
		expectError bool
	}{
		{
			name: "success",
			mockFn: func() {
				mock.ExpectEval(setSummaryIfVersionScript, keys, data, "3", ttl).SetVal(int64(1))
			},
		},
		{
			name: "version moved",
			mockFn: func() {
				mock.ExpectEval(setSummaryIfVersionScript, keys, data, "3", ttl).SetVal(int64(0))
			},
		},
		{
			name: "eval failure",
			mockFn: func() {
				mock.ExpectEval(setSummaryIfVersionScript, keys, data, "3", ttl).SetErr(errors.New("redis error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			err := client.SetUserAchievementsSummary(ctx, user, summary, 3)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAchievementCacheClient_GetUserAchievementsSummaryVersion(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &achievementCacheClient{storage: db}
	ctx := context.Background()
	key := fmt.Sprintf("%s:%s", summaryVersionKeyPrefix, "u1")

	tests := []struct {
		name        string
		mockFn      func()
		expected    int64
		expectError bool
	}{
		{
			name:     "value",
			mockFn:   func() { mock.ExpectGet(key).SetVal("3") },
			expected: 3,
		},
		{
			name:   "missing",
			mockFn: func() { mock.ExpectGet(key).RedisNil() },
		},
		{
			name:        "redis error",
			mockFn:      func() { mock.ExpectGet(key).SetErr(errors.New("fail")) },
			expectError: true,
		},
		{
			name:        "not a number",
			mockFn:      func() { mock.ExpectGet(key).SetVal("x") },
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			version, err := client.GetUserAchievementsSummaryVersion(ctx, "u1")
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, version)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAchievementCacheClient_GetUserAchievementsSummary(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &achievementCacheClient{storage: db}
	ctx := context.Background()

	user := "user1"
	summary := &dto.AchievementsSummary{
		Achievements: []dto.AchievementInfo{
			{RuleName: "rule1", Reward: 10},
			{RuleName: "rule2", Reward: 20},
		},
		RewardSum: 30,
	}
	summaryJSON, _ := json.Marshal(summary)

	tests := []struct {
		name        string
		mockFn      func()
		expected    *dto.AchievementsSummary
		expectError bool
	}{
		{
			name: "success",
			mockFn: func() {
				mock.ExpectGet(fmt.Sprintf("%s:%s", summaryKeyPrefix, user)).
					SetVal(string(summaryJSON))
			},
			expected:    summary,
			expectError: false,
		},
		{
			name: "key does not exist",
			mockFn: func() {
				mock.ExpectGet(fmt.Sprintf("%s:%s", summaryKeyPrefix, user)).
					SetErr(redis.Nil)
			},
			expected:    nil,
			expectError: false,
		},
		{
			name: "redis error",
			mockFn: func() {
				mock.ExpectGet(fmt.Sprintf("%s:%s", summaryKeyPrefix, user)).
					SetErr(errors.New("redis error"))
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "invalid json",
			mockFn: func() {
				mock.ExpectGet(fmt.Sprintf("%s:%s", summaryKeyPrefix, user)).
					SetVal("invalid json")
			},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			result, err := client.GetUserAchievementsSummary(ctx, user)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAchievementCacheClient_SaveAll(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &achievementCacheClient{storage: db}
	ctx := context.Background()

	tests := []struct {
		name        string
		mockFn      func()
		input       []dto.Achievement
		expectError bool
	}{
		{
			name: "success",
			mockFn: func() {
				mock.ExpectTxPipeline()

				mock.ExpectZAdd("achievement:user1:rule1", redis.Z{
					Score:  100,
					Member: int64(100),
				}).SetVal(1)
				mock.ExpectZAdd("achievement:user2:rule2", redis.Z{
					Score:  200,
					Member: int64(200),
				}).SetVal(1)

				mock.ExpectTxPipelineExec()
			},
			input: []dto.Achievement{
				{User: "user1", RuleName: "rule1", RuleVersion: "v1", EndRange: time.Unix(100, 0)},
				{User: "user2", RuleName: "rule2", RuleVersion: "v2", EndRange: time.Unix(200, 0)},
			},
			expectError: false,
		},
		{
			name: "pipeline exec failure",
			mockFn: func() {
				mock.ExpectTxPipeline()
				mock.ExpectZAdd("achievement:user1:rule1", redis.Z{
					Score:  100,
					Member: int64(100),
				}).SetVal(1)
				mock.ExpectTxPipelineExec().SetErr(errors.New("fail"))
			},
			input: []dto.Achievement{
				{User: "user1", RuleName: "rule1", RuleVersion: "v1", EndRange: time.Unix(100, 0)},
			},
			expectError: true,
		},
		{
			name:        "nil achievements",
			mockFn:      func() {},
			input:       nil,
			expectError: true,
		},
		{
			name:        "empty achievements",
			mockFn:      func() {},
			input:       []dto.Achievement{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			err := client.SaveAll(ctx, tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAchievementCacheClient_AchievementExists(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &achievementCacheClient{storage: db}
	ctx := context.Background()

	now := time.Unix(100, 0)
	achievement := &dto.Achievement{
		User:        "user1",
		RuleName:    "rule1",
		RuleVersion: "v1",
		StartRange:  now,
		EndRange:    now,
	}

	tests := []struct {
		name        string
		mockFn      func()
		input       *dto.Achievement
		expected    bool
		expectError bool
	}{
		{
			name: "exists",
			mockFn: func() {
				mock.ExpectZRangeByScoreWithScores(
					"achievement:user1:rule1",
					&redis.ZRangeBy{Min: "-inf", Max: "+inf"},
				).SetVal([]redis.Z{{Score: float64(now.Unix()), Member: int64(now.Unix())}})
			},
			input:       achievement,
			expected:    true,
			expectError: false,
		},
		{
			name: "does not exist",
			mockFn: func() {
				mock.ExpectZRangeByScoreWithScores(
					"achievement:user1:rule1",
					&redis.ZRangeBy{Min: "-inf", Max: "+inf"},
				).SetVal([]redis.Z{})
			},
			input:       achievement,
			expected:    false,
			expectError: false,
		},
		{
			name: "redis error",
			mockFn: func() {
				mock.ExpectZRangeByScoreWithScores(
					"achievement:user1:rule1",
					&redis.ZRangeBy{Min: "-inf", Max: "+inf"},
				).SetErr(errors.New("fail"))
			},
			input:       achievement,
			expected:    false,
			expectError: true,
		},
		{
			name:        "nil achievement",
			mockFn:      func() {},
			input:       nil,
			expected:    false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			result, err := client.AchievementExists(ctx, tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAchievementCacheClient_InvalidateUserAchievementsSummaries(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &achievementCacheClient{storage: db}
	ctx := context.Background()

	tests := []struct {
		name        string
		mockFn      func()
		input       []string
		expectError bool
	}{
		{
			name: "success",
			mockFn: func() {
				mock.ExpectTxPipeline()
				mock.ExpectIncr("summary_version:u1").SetVal(1)
				mock.ExpectDel("summary:u1").SetVal(1)
				mock.ExpectIncr("summary_version:u2").SetVal(1)
				mock.ExpectDel("summary:u2").SetVal(1)
				mock.ExpectTxPipelineExec()
			},
			input: []string{"u1", "u2"},
		},
		{
			name: "redis error",
			mockFn: func() {
				mock.ExpectTxPipeline()
				mock.ExpectIncr("summary_version:u1").SetVal(1)
				mock.ExpectDel("summary:u1").SetVal(1)
				mock.ExpectTxPipelineExec().SetErr(errors.New("fail"))
			},
			input:       []string{"u1"},
			expectError: true,
		},
		{
			name:   "empty users",
			mockFn: func() {},
			input:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			err := client.InvalidateUserAchievementsSummaries(ctx, tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
