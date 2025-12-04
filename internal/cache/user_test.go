package cache

import (
	"context"
	"errors"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestNewUserCacheClient(t *testing.T) {
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
			client, err := NewUserCacheClient(storage)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.IsType(t, &userCacheClient{}, client)
				assert.Equal(t, storage, client.(*userCacheClient).storage)
			}
		})
	}
}

func TestUserCacheClient_SaveAll(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &userCacheClient{storage: db}
	ctx := context.Background()

	tests := []struct {
		name        string
		mockFn      func()
		input       []string
		expectError bool
	}{
		{
			name: "success multiple users",
			mockFn: func() {
				mock.ExpectTxPipeline()
				mock.ExpectZAdd(usersKey, redis.Z{Member: "user1"}).SetVal(1)
				mock.ExpectZAdd(usersKey, redis.Z{Member: "user2"}).SetVal(1)
				mock.ExpectZAdd(usersKey, redis.Z{Member: "user3"}).SetVal(1)
				mock.ExpectTxPipelineExec()
			},
			input:       []string{"user1", "user2", "user3"},
			expectError: false,
		},
		{
			name: "success single user",
			mockFn: func() {
				mock.ExpectTxPipeline()
				mock.ExpectZAdd(usersKey, redis.Z{Member: "testuser"}).SetVal(1)
				mock.ExpectTxPipelineExec()
			},
			input:       []string{"testuser"},
			expectError: false,
		},
		{
			name: "pipeline exec failure",
			mockFn: func() {
				mock.ExpectTxPipeline()
				mock.ExpectZAdd(usersKey, redis.Z{Member: "user1"}).SetErr(errors.New("pipeline error"))
			},
			input:       []string{"user1"},
			expectError: true,
		},
		{
			name:        "nil users",
			mockFn:      func() {},
			input:       nil,
			expectError: true,
		},
		{
			name:        "empty users",
			mockFn:      func() {},
			input:       []string{},
			expectError: false,
		},
		{
			name: "success duplicate users",
			mockFn: func() {
				mock.ExpectTxPipeline()
				mock.ExpectZAdd(usersKey, redis.Z{Member: "user1"}).SetVal(1)
				mock.ExpectZAdd(usersKey, redis.Z{Member: "user1"}).SetVal(1)
				mock.ExpectTxPipelineExec()
			},
			input:       []string{"user1", "user1"},
			expectError: false,
		},
		{
			name: "ZAdd command failure in pipeline",
			mockFn: func() {
				mock.ExpectTxPipeline()
				mock.ExpectZAdd(usersKey, redis.Z{Member: "user1"}).SetErr(errors.New("zadd error"))
			},
			input:       []string{"user1"},
			expectError: true,
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

func TestUserCacheClient_Save(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &userCacheClient{storage: db}
	ctx := context.Background()

	tests := []struct {
		name        string
		mockFn      func()
		input       string
		expectError bool
	}{
		{
			name: "success",
			mockFn: func() {
				mock.ExpectZAdd(usersKey, redis.Z{Member: "testuser"}).SetVal(1)
			},
			input:       "testuser",
			expectError: false,
		},
		{
			name: "ZAdd failure",
			mockFn: func() {
				mock.ExpectZAdd(usersKey, redis.Z{Member: "erroruser"}).SetErr(errors.New("redis error"))
			},
			input:       "erroruser",
			expectError: true,
		},
		{
			name: "success with special characters",
			mockFn: func() {
				mock.ExpectZAdd(usersKey, redis.Z{Member: "user-name_123"}).SetVal(1)
			},
			input:       "user-name_123",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			err := client.Save(ctx, tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserCacheClient_GetAll(t *testing.T) {
	db, mock := redismock.NewClientMock()
	client := &userCacheClient{storage: db}
	ctx := context.Background()

	tests := []struct {
		name        string
		mockFn      func()
		expected    []string
		expectError bool
	}{
		{
			name: "success multiple users",
			mockFn: func() {
				mock.ExpectZRange(usersKey, 0, -1).SetVal([]string{"user1", "user2", "user3"})
			},
			expected:    []string{"user1", "user2", "user3"},
			expectError: false,
		},
		{
			name: "success single user",
			mockFn: func() {
				mock.ExpectZRange(usersKey, 0, -1).SetVal([]string{"testuser"})
			},
			expected:    []string{"testuser"},
			expectError: false,
		},
		{
			name: "ZRange failure",
			mockFn: func() {
				mock.ExpectZRange(usersKey, 0, -1).SetErr(errors.New("zrange error"))
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "no users found",
			mockFn: func() {
				mock.ExpectZRange(usersKey, 0, -1).SetVal([]string{})
			},
			expected:    nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			users, err := client.GetAll(ctx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tt.expected != nil {
				assert.Equal(t, tt.expected, users)
			} else {
				assert.Empty(t, users)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
