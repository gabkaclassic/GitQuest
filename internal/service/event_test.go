package service

import (
	"context"
	"errors"
	"testing"

	"github.com/gabkaclassic/GitQuest/internal/cache"
	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/gabkaclassic/GitQuest/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewEventService(t *testing.T) {
	tests := []struct {
		name             string
		repository       repository.EventRepository
		eventCacheClient cache.EventCacheClient
		userCacheClient  cache.UserCacheClient
		expectError      bool
		errorMsg         string
	}{
		{
			name:             "all dependencies provided",
			repository:       &repository.MockEventRepository{},
			eventCacheClient: &cache.MockEventCacheClient{},
			userCacheClient:  &cache.MockUserCacheClient{},
			expectError:      false,
		},
		{
			name:             "nil repository",
			repository:       nil,
			eventCacheClient: &cache.MockEventCacheClient{},
			userCacheClient:  &cache.MockUserCacheClient{},
			expectError:      true,
			errorMsg:         "repository is nil",
		},
		{
			name:             "nil event cache client",
			repository:       &repository.MockEventRepository{},
			eventCacheClient: nil,
			userCacheClient:  &cache.MockUserCacheClient{},
			expectError:      true,
			errorMsg:         "event cache client is nil",
		},
		{
			name:             "nil user cache client",
			repository:       &repository.MockEventRepository{},
			eventCacheClient: &cache.MockEventCacheClient{},
			userCacheClient:  nil,
			expectError:      true,
			errorMsg:         "user cache client is nil",
		},
		{
			name:             "all nil dependencies",
			repository:       nil,
			eventCacheClient: nil,
			userCacheClient:  nil,
			expectError:      true,
			errorMsg:         "repository is nil",
		},
		{
			name:             "nil repository and event cache client",
			repository:       nil,
			eventCacheClient: nil,
			userCacheClient:  &cache.MockUserCacheClient{},
			expectError:      true,
			errorMsg:         "repository is nil",
		},
		{
			name:             "nil repository and user cache client",
			repository:       nil,
			eventCacheClient: &cache.MockEventCacheClient{},
			userCacheClient:  nil,
			expectError:      true,
			errorMsg:         "repository is nil",
		},
		{
			name:             "nil event and user cache clients",
			repository:       &repository.MockEventRepository{},
			eventCacheClient: nil,
			userCacheClient:  nil,
			expectError:      true,
			errorMsg:         "event cache client is nil",
		},
		{
			name:             "valid mock implementations",
			repository:       &repository.MockEventRepository{},
			eventCacheClient: &cache.MockEventCacheClient{},
			userCacheClient:  &cache.MockUserCacheClient{},
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewEventService(tt.repository, tt.eventCacheClient, tt.userCacheClient)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, service)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
				assert.IsType(t, &eventService{}, service)

				svc := service.(*eventService)
				assert.Equal(t, tt.repository, svc.repository)
				assert.Equal(t, tt.eventCacheClient, svc.eventCacheClient)
				assert.Equal(t, tt.userCacheClient, svc.userCacheClient)
			}
		})
	}
}

func TestEventService_SaveAll(t *testing.T) {
	events := []dto.Event{{Actor: "user1"}}

	type mocks struct {
		repo       *repository.MockEventRepository
		eventCache *cache.MockEventCacheClient
		userCache  *cache.MockUserCacheClient
	}

	tests := []struct {
		name      string
		setup     func(m mocks)
		expectErr string
	}{
		{
			name: "repository error",
			setup: func(m mocks) {
				m.repo.On("SaveAll", mock.Anything, events).Return(errors.New("db error"))
			},
			expectErr: "save events error",
		},
		{
			name: "event cache error",
			setup: func(m mocks) {
				m.repo.On("SaveAll", mock.Anything, events).Return(nil)
				m.eventCache.On("SaveNewEvents", mock.Anything, events).Return(nil, errors.New("cache error"))
			},
			expectErr: "cache operations error",
		},
		{
			name: "user cache error",
			setup: func(m mocks) {
				m.repo.On("SaveAll", mock.Anything, events).Return(nil)
				m.eventCache.On("SaveNewEvents", mock.Anything, events).Return([]string{"user1"}, nil)
				m.userCache.On("SaveAll", mock.Anything, []string{"user1"}).Return(errors.New("user cache error"))
			},
			expectErr: "cache operations error",
		},
		{
			name: "successful save",
			setup: func(m mocks) {
				m.repo.On("SaveAll", mock.Anything, events).Return(nil)
				m.eventCache.On("SaveNewEvents", mock.Anything, events).Return([]string{"user1"}, nil)
				m.userCache.On("SaveAll", mock.Anything, []string{"user1"}).Return(nil)
			},
			expectErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockObjs := mocks{
				repo:       new(repository.MockEventRepository),
				eventCache: new(cache.MockEventCacheClient),
				userCache:  new(cache.MockUserCacheClient),
			}

			tt.setup(mockObjs)

			service := &eventService{
				repository:       mockObjs.repo,
				eventCacheClient: mockObjs.eventCache,
				userCacheClient:  mockObjs.userCache,
			}

			err := service.SaveAll(context.Background(), events)

			if tt.expectErr == "" {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
				assert.Contains(t, err.Message, tt.expectErr)
			}

			mockObjs.repo.AssertExpectations(t)
			mockObjs.eventCache.AssertExpectations(t)
			mockObjs.userCache.AssertExpectations(t)
		})
	}
}

func TestEventService_LoadUsersToCache(t *testing.T) {
	type mocks struct {
		repo      *repository.MockEventRepository
		userCache *cache.MockUserCacheClient
	}

	tests := []struct {
		name      string
		setup     func(m mocks)
		expectErr string
	}{
		{
			name: "repository error",
			setup: func(m mocks) {
				m.repo.On("GetAllUsersWithEvents", mock.Anything).Return(nil, errors.New("db error"))
			},
			expectErr: "db error",
		},
		{
			name: "user cache error",
			setup: func(m mocks) {
				m.repo.On("GetAllUsersWithEvents", mock.Anything).Return([]string{"user1"}, nil)
				m.userCache.On("SaveAll", mock.Anything, []string{"user1"}).Return(errors.New("cache error"))
			},
			expectErr: "cache error",
		},
		{
			name: "successful path",
			setup: func(m mocks) {
				m.repo.On("GetAllUsersWithEvents", mock.Anything).Return([]string{"user1"}, nil)
				m.userCache.On("SaveAll", mock.Anything, []string{"user1"}).Return(nil)
			},
			expectErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockObjs := mocks{
				repo:      new(repository.MockEventRepository),
				userCache: new(cache.MockUserCacheClient),
			}

			tt.setup(mockObjs)

			service := &eventService{
				repository:      mockObjs.repo,
				userCacheClient: mockObjs.userCache,
			}

			err := service.LoadUsersToCache(context.Background())

			if tt.expectErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectErr)
			}

			mockObjs.repo.AssertExpectations(t)
			mockObjs.userCache.AssertExpectations(t)
		})
	}
}
