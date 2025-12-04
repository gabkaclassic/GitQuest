package cache

import (
	"context"
	"errors"
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
