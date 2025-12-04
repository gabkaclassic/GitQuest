package service

import (
	"context"
	"errors"
	api "github.com/gabkaclassic/metrics/pkg/error"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"

	"github.com/gabkaclassic/GitQuest/internal/cache"
	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/gabkaclassic/GitQuest/internal/repository"
	"github.com/stretchr/testify/assert"
)

func TestNewAchievementService(t *testing.T) {
	mockRepo := &repository.MockAchievementRepository{}
	mockUserRepo := &repository.MockUserRepository{}
	mockEventCache := &cache.MockEventCacheClient{}
	mockUserCache := &cache.MockUserCacheClient{}
	mockAchievementCache := &cache.MockAchievementCacheClient{}
	mockNotification := &MockNotificationService{}

	tests := []struct {
		name        string
		repo        repository.AchievementRepository
		userRepo    repository.UserRepository
		eventCache  cache.EventCacheClient
		userCache   cache.UserCacheClient
		achCache    cache.AchievementCacheClient
		notif       NotificationService
		expectedErr bool
		errMsg      string
	}{
		{
			name:        "success",
			repo:        mockRepo,
			userRepo:    mockUserRepo,
			eventCache:  mockEventCache,
			userCache:   mockUserCache,
			achCache:    mockAchievementCache,
			notif:       mockNotification,
			expectedErr: false,
		},
		{
			name:        "nil achievement repository",
			repo:        nil,
			userRepo:    mockUserRepo,
			eventCache:  mockEventCache,
			userCache:   mockUserCache,
			achCache:    mockAchievementCache,
			notif:       mockNotification,
			expectedErr: true,
			errMsg:      "create new achievement service failed: achievement repository is nil",
		},
		{
			name:        "nil user repository",
			repo:        mockRepo,
			userRepo:    nil,
			eventCache:  mockEventCache,
			userCache:   mockUserCache,
			achCache:    mockAchievementCache,
			notif:       mockNotification,
			expectedErr: true,
			errMsg:      "create new achievement service failed: user repository is nil",
		},
		{
			name:        "nil event cache client",
			repo:        mockRepo,
			userRepo:    mockUserRepo,
			eventCache:  nil,
			userCache:   mockUserCache,
			achCache:    mockAchievementCache,
			notif:       mockNotification,
			expectedErr: true,
			errMsg:      "create new achievement service failed: event cache client is nil",
		},
		{
			name:        "nil user cache client",
			repo:        mockRepo,
			userRepo:    mockUserRepo,
			eventCache:  mockEventCache,
			userCache:   nil,
			achCache:    mockAchievementCache,
			notif:       mockNotification,
			expectedErr: true,
			errMsg:      "create new achievement service failed: user cache client is nil",
		},
		{
			name:        "nil achievement cache client",
			repo:        mockRepo,
			userRepo:    mockUserRepo,
			eventCache:  mockEventCache,
			userCache:   mockUserCache,
			achCache:    nil,
			notif:       mockNotification,
			expectedErr: true,
			errMsg:      "create new achievement service failed: achievement cache client is nil",
		},
		{
			name:        "nil notification service",
			repo:        mockRepo,
			userRepo:    mockUserRepo,
			eventCache:  mockEventCache,
			userCache:   mockUserCache,
			achCache:    mockAchievementCache,
			notif:       nil,
			expectedErr: true,
			errMsg:      "create new achievement service failed: notifications service is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewAchievementService(
				tt.repo,
				tt.userRepo,
				tt.eventCache,
				tt.userCache,
				tt.achCache,
				tt.notif,
			)

			if tt.expectedErr {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.errMsg)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
			}
		})
	}
}

func TestAchievementService_CheckForNewAchievements(t *testing.T) {
	user := "user1"
	rule := dto.Rule{
		Name:      "rule1",
		Version:   "1",
		EventType: "event",
		Count:     1,
		Reward:    10,
		Window:    3600,
		Streak:    false,
		Condition: dto.GTE,
	}

	type mocks struct {
		userCache        *cache.MockUserCacheClient
		repo             *repository.MockAchievementRepository
		eventCache       *cache.MockEventCacheClient
		achievementCache *cache.MockAchievementCacheClient
		notificationSvc  *MockNotificationService
	}

	tests := []struct {
		name      string
		setup     func(m mocks)
		rules     []dto.Rule
		expectErr string
	}{
		{
			name: "get users from cache error",
			setup: func(m mocks) {
				m.userCache.On("GetAll", mock.Anything).Return(nil, errors.New("cache error"))
			},
			rules:     []dto.Rule{},
			expectErr: "get users from cache error",
		},
		{
			name: "no achievements generated",
			setup: func(m mocks) {
				m.userCache.On("GetAll", mock.Anything).Return([]string{user}, nil)
				m.eventCache.On("GetUserEventsTimestampsByTypeAndRange", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]time.Time{}, nil)
			},
			rules:     []dto.Rule{rule},
			expectErr: "",
		},
		{
			name: "repository save error",
			setup: func(m mocks) {
				m.userCache.On("GetAll", mock.Anything).Return([]string{user}, nil)
				m.eventCache.On("GetUserEventsTimestampsByTypeAndRange", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]time.Time{time.Now()}, nil)
				m.achievementCache.On("AchievementExists", mock.Anything, mock.Anything).Return(false, nil)
				m.repo.On("SaveAll", mock.Anything).Return(errors.New("db error"))
			},
			rules:     []dto.Rule{rule},
			expectErr: "save new achievements to DB error",
		},
		{
			name: "achievement cache save error",
			setup: func(m mocks) {
				m.userCache.On("GetAll", mock.Anything).Return([]string{user}, nil)
				m.eventCache.On("GetUserEventsTimestampsByTypeAndRange", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]time.Time{time.Now()}, nil)
				m.achievementCache.On("AchievementExists", mock.Anything, mock.Anything).Return(false, nil)
				m.repo.On("SaveAll", mock.Anything).Return(nil)
				m.achievementCache.On("SaveAll", mock.Anything, mock.Anything).Return(errors.New("cache error"))
			},
			rules:     []dto.Rule{rule},
			expectErr: "save new achievements to cache error",
		},
		{
			name: "notification send error",
			setup: func(m mocks) {
				m.userCache.On("GetAll", mock.Anything).Return([]string{user}, nil)
				m.eventCache.On("GetUserEventsTimestampsByTypeAndRange", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]time.Time{time.Now()}, nil)
				m.achievementCache.On("AchievementExists", mock.Anything, mock.Anything).Return(false, nil)
				m.repo.On("SaveAll", mock.Anything).Return(nil)
				m.achievementCache.On("SaveAll", mock.Anything, mock.Anything).Return(nil)
				m.notificationSvc.On("Notify", mock.Anything).Return(errors.New("notify error"))
			},
			rules:     []dto.Rule{rule},
			expectErr: "send notification error",
		},
		{
			name: "successful path",
			setup: func(m mocks) {
				m.userCache.On("GetAll", mock.Anything).Return([]string{user}, nil)
				m.eventCache.On("GetUserEventsTimestampsByTypeAndRange", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]time.Time{time.Now()}, nil)
				m.achievementCache.On("AchievementExists", mock.Anything, mock.Anything).Return(false, nil)
				m.repo.On("SaveAll", mock.Anything).Return(nil)
				m.achievementCache.On("SaveAll", mock.Anything, mock.Anything).Return(nil)
				m.notificationSvc.On("Notify", mock.Anything).Return(nil)
			},
			rules:     []dto.Rule{rule},
			expectErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockObjs := mocks{
				userCache:        new(cache.MockUserCacheClient),
				repo:             new(repository.MockAchievementRepository),
				eventCache:       new(cache.MockEventCacheClient),
				achievementCache: new(cache.MockAchievementCacheClient),
				notificationSvc:  new(MockNotificationService),
			}

			tt.setup(mockObjs)

			service := &achievementService{
				userCacheClient:        mockObjs.userCache,
				repository:             mockObjs.repo,
				eventCacheClient:       mockObjs.eventCache,
				achievementCacheClient: mockObjs.achievementCache,
				notificationService:    mockObjs.notificationSvc,
			}

			err := service.CheckForNewAchievements(context.Background(), tt.rules)

			if tt.expectErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectErr)
			}

			mockObjs.userCache.AssertExpectations(t)
			mockObjs.repo.AssertExpectations(t)
			mockObjs.eventCache.AssertExpectations(t)
			mockObjs.achievementCache.AssertExpectations(t)
			mockObjs.notificationSvc.AssertExpectations(t)
		})
	}
}

func TestAchievementService_processUserEvents(t *testing.T) {
	user := "user1"
	rule := dto.Rule{
		Name:      "rule1",
		Version:   "1",
		EventType: "event",
		Count:     1,
		Reward:    10,
		Window:    0,
		Streak:    false,
		Condition: dto.GTE,
	}

	type mocks struct {
		eventCache       *cache.MockEventCacheClient
		achievementCache *cache.MockAchievementCacheClient
		repo             *repository.MockAchievementRepository
	}

	tests := []struct {
		name        string
		setup       func(m mocks)
		rules       []dto.Rule
		expectedOut int
	}{
		{
			name: "events cache error",
			setup: func(m mocks) {
				m.eventCache.On("GetUserEventsTimestampsByTypeAndRange", mock.Anything, user, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("cache error"))
			},
			rules:       []dto.Rule{rule},
			expectedOut: 0,
		},
		{
			name: "achievement exists in cache",
			setup: func(m mocks) {
				m.eventCache.On("GetUserEventsTimestampsByTypeAndRange", mock.Anything, user, mock.Anything, mock.Anything, mock.Anything).
					Return([]time.Time{time.Now()}, nil)
				m.achievementCache.On("AchievementExists", mock.Anything, mock.Anything).Return(true, nil)
			},
			rules:       []dto.Rule{rule},
			expectedOut: 0,
		},
		{
			name: "exists in all time",
			setup: func(m mocks) {
				m.eventCache.On("GetUserEventsTimestampsByTypeAndRange", mock.Anything, user, mock.Anything, mock.Anything, mock.Anything).
					Return([]time.Time{time.Now()}, nil)
				m.achievementCache.On("AchievementExists", mock.Anything, mock.Anything).Return(false, nil)
				m.repo.On("ExistsInAllTime", mock.Anything).Return(true, nil)
			},
			rules:       []dto.Rule{rule},
			expectedOut: 0,
		},
		{
			name: "successful achievement generation",
			setup: func(m mocks) {
				m.eventCache.On("GetUserEventsTimestampsByTypeAndRange", mock.Anything, user, mock.Anything, mock.Anything, mock.Anything).
					Return([]time.Time{time.Now()}, nil)
				m.achievementCache.On("AchievementExists", mock.Anything, mock.Anything).Return(false, nil)
				m.repo.On("ExistsInAllTime", mock.Anything).Return(false, nil)
			},
			rules:       []dto.Rule{rule},
			expectedOut: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockObjs := mocks{
				eventCache:       new(cache.MockEventCacheClient),
				achievementCache: new(cache.MockAchievementCacheClient),
				repo:             new(repository.MockAchievementRepository),
			}

			tt.setup(mockObjs)

			service := &achievementService{
				eventCacheClient:       mockObjs.eventCache,
				achievementCacheClient: mockObjs.achievementCache,
				repository:             mockObjs.repo,
			}

			out := make(chan dto.Achievement, 10)
			err := service.processUserEvents(context.Background(), user, tt.rules, out)
			assert.NoError(t, err)
			close(out)

			count := 0
			for range out {
				count++
			}
			assert.Equal(t, tt.expectedOut, count)

			mockObjs.eventCache.AssertExpectations(t)
			mockObjs.achievementCache.AssertExpectations(t)
			mockObjs.repo.AssertExpectations(t)
		})
	}
}

func TestAchievementService_reevalByDiff(t *testing.T) {
	diff := &dto.RuleDiff{
		Name:       "rule1",
		RewardDiff: 10,
	}

	type mocks struct {
		repo            *repository.MockAchievementRepository
		notificationSvc *MockNotificationService
	}

	tests := []struct {
		name      string
		setup     func(m mocks)
		expectErr string
	}{
		{
			name: "repository error",
			setup: func(m mocks) {
				m.repo.On("ReevalByRuleDiff", diff).Return(nil, errors.New("db error"))
			},
			expectErr: "db error",
		},
		{
			name: "notify error",
			setup: func(m mocks) {
				m.repo.On("ReevalByRuleDiff", diff).Return([]string{"user1", "user2"}, nil)
				m.notificationSvc.On("Notify", mock.Anything).Return(errors.New("notify error"))
			},
			expectErr: "notify error",
		},
		{
			name: "successful path",
			setup: func(m mocks) {
				m.repo.On("ReevalByRuleDiff", diff).Return([]string{"user1"}, nil)
				m.notificationSvc.On("Notify", mock.Anything).Return(nil)
			},
			expectErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockObjs := mocks{
				repo:            new(repository.MockAchievementRepository),
				notificationSvc: new(MockNotificationService),
			}

			tt.setup(mockObjs)

			service := &achievementService{
				repository:          mockObjs.repo,
				notificationService: mockObjs.notificationSvc,
			}

			err := service.reevalByDiff(diff)

			if tt.expectErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectErr)
			}

			mockObjs.repo.AssertExpectations(t)
			mockObjs.notificationSvc.AssertExpectations(t)
		})
	}
}

func TestCheckStreak(t *testing.T) {
	now := time.Now().Truncate(24 * time.Hour)

	tests := []struct {
		name       string
		timestamps []time.Time
		days       int
		expected   bool
	}{
		{
			name:       "empty timestamps",
			timestamps: []time.Time{},
			days:       3,
			expected:   false,
		},
		{
			name: "less than days",
			timestamps: []time.Time{
				now.Add(-1 * 24 * time.Hour),
			},
			days:     3,
			expected: false,
		},
		{
			name: "exact streak",
			timestamps: []time.Time{
				now.Add(-2 * 24 * time.Hour),
				now.Add(-1 * 24 * time.Hour),
				now,
			},
			days:     3,
			expected: true,
		},
		{
			name: "streak with gap",
			timestamps: []time.Time{
				now.Add(-3 * 24 * time.Hour),
				now.Add(-1 * 24 * time.Hour),
				now,
			},
			days:     3,
			expected: false,
		},
		{
			name: "long streak",
			timestamps: []time.Time{
				now.Add(-4 * 24 * time.Hour),
				now.Add(-3 * 24 * time.Hour),
				now.Add(-2 * 24 * time.Hour),
				now.Add(-1 * 24 * time.Hour),
				now,
			},
			days:     5,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkStreak(tt.timestamps, tt.days)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAchievementService_GetSummaryByUser(t *testing.T) {
	user := "u"

	type mocks struct {
		repo *repository.MockAchievementRepository
	}

	tests := []struct {
		name        string
		setup       func(m mocks)
		expected    *dto.AchievementsSummary
		expectedErr *api.APIError
	}{
		{
			name: "ok",
			setup: func(m mocks) {
				m.repo.
					On("GetByUser", mock.Anything, user).
					Return(&dto.AchievementsSummary{
						Achievements: []dto.AchievementInfo{
							{RuleName: "r1", Reward: 1},
							{RuleName: "r2", Reward: 2},
						},
					}, nil)
			},
			expected: &dto.AchievementsSummary{
				Achievements: []dto.AchievementInfo{
					{RuleName: "r1", Reward: 1},
					{RuleName: "r2", Reward: 2},
				},
			},
		},
		{
			name: "internal",
			setup: func(m mocks) {
				m.repo.
					On("GetByUser", mock.Anything, user).
					Return(nil, errors.New("db err"))
			},
			expectedErr: api.Internal("Get achievements for user error", errors.New("db err")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(repository.MockAchievementRepository)

			tt.setup(mocks{repo: mockRepo})

			svc := &achievementService{repository: mockRepo}

			out, err := svc.GetSummaryByUser(t.Context(), user)

			if tt.expectedErr != nil {
				assert.Nil(t, out)
				assert.NotNil(t, err)
				mockRepo.AssertExpectations(t)
				return
			}

			assert.Nil(t, err)
			assert.Equal(t, tt.expected, out)

			mockRepo.AssertExpectations(t)
		})
	}
}
