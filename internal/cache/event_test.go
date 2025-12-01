package cache

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestNewEventCacheClient(t *testing.T) {
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
			client, err := NewEventCacheClient(storage)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.IsType(t, &eventCacheClient{}, client)
				assert.Equal(t, storage, client.(*eventCacheClient).storage)
			}
		})
	}
}

func TestEventCacheClient_CleanupOldEventsFromCache(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &eventCacheClient{storage: db}
	ctx := context.Background()

	now := time.Now()
	maxTimeRangeTimestamp := strconv.FormatInt(now.Add(-dto.MaxTimeRange).Unix(), 10)

	tests := []struct {
		name        string
		mockFn      func()
		expectError bool
	}{
		{
			name: "success - multiple keys",
			mockFn: func() {
				mock.ExpectScan(0, "event:*:*", 0).SetVal([]string{
					"event:user1:rule1",
					"event:user2:rule2",
				}, 0)

				mock.ExpectZRemRangeByScore("event:user1:rule1", "0", maxTimeRangeTimestamp).SetVal(2)
				mock.ExpectZRemRangeByScore("event:user2:rule2", "0", maxTimeRangeTimestamp).SetVal(1)
			},
			expectError: false,
		},
		{
			name: "success - single key",
			mockFn: func() {
				mock.ExpectScan(0, "event:*:*", 0).SetVal([]string{"event:user1:rule1"}, 0)
				mock.ExpectZRemRangeByScore("event:user1:rule1", "0", maxTimeRangeTimestamp).SetVal(1)
			},
			expectError: false,
		},
		{
			name: "success - no keys found",
			mockFn: func() {
				mock.ExpectScan(0, "event:*:*", 0).SetVal([]string{}, 0)
			},
			expectError: false,
		},
		{
			name: "scan failure",
			mockFn: func() {
				mock.ExpectScan(0, "event:*:*", 0).SetErr(errors.New("scan error"))
			},
			expectError: true,
		},
		{
			name: "iterator error",
			mockFn: func() {
				mock.ExpectScan(0, "event:*:*", 0).SetVal([]string{"event:user1:rule1"}, 0)
				redis.NewScanCmdResult([]string{"event:user1:rule1"}, 0, nil)
			},
			expectError: true,
		},
		{
			name: "scan with pagination - multiple iterations",
			mockFn: func() {
				mock.ExpectScan(0, "event:*:*", 0).SetVal([]string{"event:user1:rule1", "event:user2:rule2"}, 42)
				mock.ExpectScan(42, "event:*:*", 0).SetVal([]string{"event:user3:rule3"}, 0)

				mock.ExpectZRemRangeByScore("event:user1:rule1", "0", maxTimeRangeTimestamp).SetVal(1)
				mock.ExpectZRemRangeByScore("event:user2:rule2", "0", maxTimeRangeTimestamp).SetVal(2)
				mock.ExpectZRemRangeByScore("event:user3:rule3", "0", maxTimeRangeTimestamp).SetVal(1)
			},
			expectError: false,
		},
		{
			name: "zremrangebyscore failure",
			mockFn: func() {
				mock.ExpectScan(0, "event:*:*", 0).SetVal([]string{"event:user1:rule1", "event:user2:rule2"}, 0)
				mock.ExpectZRemRangeByScore("event:user1:rule1", "0", maxTimeRangeTimestamp).SetErr(errors.New("redis error"))
			},
			expectError: true,
		},
		{
			name: "success - remove old events only",
			mockFn: func() {
				mock.ExpectScan(0, "event:*:*", 0).SetVal([]string{"event:user1:rule1"}, 0)
				mock.ExpectZRemRangeByScore("event:user1:rule1", "0", maxTimeRangeTimestamp).SetVal(5)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			err := client.CleanupOldEventsFromCache(ctx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestEventCacheClient_SaveNewEvents(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &eventCacheClient{storage: db}
	ctx := context.Background()

	tests := []struct {
		name        string
		mockFn      func()
		input       *[]dto.Event
		expected    *[]string
		expectError bool
	}{
		{
			name: "success - multiple events",
			mockFn: func() {
				mock.ExpectTxPipeline()
				mock.ExpectZAdd("event:user1:PushEvent", redis.Z{
					Score:  100,
					Member: float64(100),
				}).SetVal(1)
				mock.ExpectZAdd("event:user1:IssueEvent", redis.Z{
					Score:  200,
					Member: float64(200),
				}).SetVal(1)
				mock.ExpectZAdd("event:user2:PushEvent", redis.Z{
					Score:  300,
					Member: float64(300),
				}).SetVal(1)
				mock.ExpectTxPipelineExec()
			},
			input: &[]dto.Event{
				{Actor: "user1", EventType: "PushEvent", Timestamp: time.Unix(100, 0)},
				{Actor: "user1", EventType: "IssueEvent", Timestamp: time.Unix(200, 0)},
				{Actor: "user2", EventType: "PushEvent", Timestamp: time.Unix(300, 0)},
			},
			expected:    &[]string{"user1", "user2"},
			expectError: false,
		},
		{
			name: "success - single event",
			mockFn: func() {
				mock.ExpectTxPipeline()
				mock.ExpectZAdd("event:user1:PushEvent", redis.Z{
					Score:  100,
					Member: float64(100),
				}).SetVal(1)
				mock.ExpectTxPipelineExec()
			},
			input: &[]dto.Event{
				{Actor: "user1", EventType: "PushEvent", Timestamp: time.Unix(100, 0)},
			},
			expected:    &[]string{"user1"},
			expectError: false,
		},
		{
			name: "pipeline exec failure",
			mockFn: func() {
				mock.ExpectTxPipeline()
				mock.ExpectZAdd("event:user1:PushEvent", redis.Z{
					Score:  100,
					Member: float64(100),
				}).SetVal(1)
				mock.ExpectTxPipelineExec().SetErr(errors.New("pipeline error"))
			},
			input: &[]dto.Event{
				{Actor: "user1", EventType: "PushEvent", Timestamp: time.Unix(100, 0)},
			},
			expected:    &[]string{"user1"},
			expectError: true,
		},
		{
			name:        "nil events",
			mockFn:      func() {},
			input:       nil,
			expected:    nil,
			expectError: true,
		},
		{
			name:        "empty events",
			mockFn:      func() {},
			input:       &[]dto.Event{},
			expected:    &[]string{},
			expectError: false,
		},
		{
			name: "duplicate users",
			mockFn: func() {
				mock.ExpectTxPipeline()
				mock.ExpectZAdd("event:user1:PushEvent", redis.Z{
					Score:  100,
					Member: float64(100),
				}).SetVal(1)
				mock.ExpectZAdd("event:user1:IssueEvent", redis.Z{
					Score:  200,
					Member: float64(200),
				}).SetVal(1)
				mock.ExpectZAdd("event:user1:PullRequestEvent", redis.Z{
					Score:  300,
					Member: float64(300),
				}).SetVal(1)
				mock.ExpectTxPipelineExec()
			},
			input: &[]dto.Event{
				{Actor: "user1", EventType: "PushEvent", Timestamp: time.Unix(100, 0)},
				{Actor: "user1", EventType: "IssueEvent", Timestamp: time.Unix(200, 0)},
				{Actor: "user1", EventType: "PullRequestEvent", Timestamp: time.Unix(300, 0)},
			},
			expected:    &[]string{"user1"},
			expectError: false,
		},
		{
			name: "multiple events same user same type",
			mockFn: func() {
				mock.ExpectTxPipeline()
				mock.ExpectZAdd("event:user1:PushEvent", redis.Z{
					Score:  100,
					Member: float64(100),
				}).SetVal(1)
				mock.ExpectZAdd("event:user1:PushEvent", redis.Z{
					Score:  200,
					Member: float64(200),
				}).SetVal(1)
				mock.ExpectTxPipelineExec()
			},
			input: &[]dto.Event{
				{Actor: "user1", EventType: "PushEvent", Timestamp: time.Unix(100, 0)},
				{Actor: "user1", EventType: "PushEvent", Timestamp: time.Unix(200, 0)},
			},
			expected:    &[]string{"user1"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			users, err := client.SaveNewEvents(ctx, tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expected, users)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestEventCacheClient_GetUserEventsTimestampsByTypeAndRange(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &eventCacheClient{storage: db}
	ctx := context.Background()

	tests := []struct {
		name        string
		mockFn      func()
		user        string
		eventType   dto.EventType
		startRange  time.Time
		endRange    time.Time
		expected    *[]time.Time
		expectError bool
	}{
		{
			name: "success specific range",
			mockFn: func() {
				mock.ExpectZRangeByScore("event:user1:PushEvent", &redis.ZRangeBy{
					Min: "100",
					Max: "200",
				}).SetVal([]string{"100", "150", "200"})
			},
			user:       "user1",
			eventType:  "PushEvent",
			startRange: time.Unix(100, 0),
			endRange:   time.Unix(200, 0),
			expected: &[]time.Time{
				time.Unix(100, 0),
				time.Unix(150, 0),
				time.Unix(200, 0),
			},
			expectError: false,
		},
		{
			name: "success equal times with inf range",
			mockFn: func() {
				mock.ExpectZRangeByScore("event:user2:IssueEvent", &redis.ZRangeBy{
					Min: "-inf",
					Max: "+inf",
				}).SetVal([]string{"300", "400"})
			},
			user:       "user2",
			eventType:  "IssueEvent",
			startRange: time.Unix(500, 0),
			endRange:   time.Unix(500, 0),
			expected: &[]time.Time{
				time.Unix(300, 0),
				time.Unix(400, 0),
			},
			expectError: false,
		},
		{
			name: "success empty result",
			mockFn: func() {
				mock.ExpectZRangeByScore("event:user3:PullRequestEvent", &redis.ZRangeBy{
					Min: "1000",
					Max: "2000",
				}).SetVal([]string{})
			},
			user:        "user3",
			eventType:   "PullRequestEvent",
			startRange:  time.Unix(1000, 0),
			endRange:    time.Unix(2000, 0),
			expected:    nil,
			expectError: false,
		},
		{
			name: "zrangebyscore failure",
			mockFn: func() {
				mock.ExpectZRangeByScore("event:user1:PushEvent", &redis.ZRangeBy{
					Min: "100",
					Max: "200",
				}).SetErr(errors.New("redis error"))
			},
			user:        "user1",
			eventType:   "PushEvent",
			startRange:  time.Unix(100, 0),
			endRange:    time.Unix(200, 0),
			expected:    nil,
			expectError: true,
		},
		{
			name: "parse timestamp failure",
			mockFn: func() {
				mock.ExpectZRangeByScore("event:user4:PushEvent", &redis.ZRangeBy{
					Min: "100",
					Max: "200",
				}).SetVal([]string{"100", "invalid", "200"})
			},
			user:        "user4",
			eventType:   "PushEvent",
			startRange:  time.Unix(100, 0),
			endRange:    time.Unix(200, 0),
			expected:    nil,
			expectError: true,
		},
		{
			name: "success - single timestamp",
			mockFn: func() {
				mock.ExpectZRangeByScore("event:user5:PushEvent", &redis.ZRangeBy{
					Min: "500",
					Max: "600",
				}).SetVal([]string{"550"})
			},
			user:       "user5",
			eventType:  "PushEvent",
			startRange: time.Unix(500, 0),
			endRange:   time.Unix(600, 0),
			expected: &[]time.Time{
				time.Unix(550, 0),
			},
			expectError: false,
		},
		{
			name: "success - negative infinity range",
			mockFn: func() {
				mock.ExpectZRangeByScore("event:user6:IssueEvent", &redis.ZRangeBy{
					Min: "-inf",
					Max: "+inf",
				}).SetVal([]string{"1", "2", "3"})
			},
			user:       "user6",
			eventType:  "IssueEvent",
			startRange: time.Unix(10, 0),
			endRange:   time.Unix(10, 0),
			expected: &[]time.Time{
				time.Unix(1, 0),
				time.Unix(2, 0),
				time.Unix(3, 0),
			},
			expectError: false,
		},
		{
			name: "success zero to infinity range",
			mockFn: func() {
				mock.ExpectZRangeByScore("event:user7:PullRequestEvent", &redis.ZRangeBy{
					Min: "-inf",
					Max: "+inf",
				}).SetVal([]string{"100", "200"})
			},
			user:       "user7",
			eventType:  "PullRequestEvent",
			startRange: time.Unix(0, 0),
			endRange:   time.Unix(0, 0),
			expected: &[]time.Time{
				time.Unix(100, 0),
				time.Unix(200, 0),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			timestamps, err := client.GetUserEventsTimestampsByTypeAndRange(ctx, tt.user, tt.eventType, tt.startRange, tt.endRange)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tt.expected != nil {
				assert.Equal(t, tt.expected, timestamps)
			} else {
				assert.Empty(t, timestamps)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
