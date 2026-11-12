package service

import (
	"context"
	"errors"
	"testing"
	"time"

	api "github.com/gabkaclassic/metrics/pkg/error"
	"github.com/stretchr/testify/mock"

	"github.com/gabkaclassic/GitQuest/internal/cache"
	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/gabkaclassic/GitQuest/internal/repository"
	"github.com/stretchr/testify/assert"
)

func TestNewAchievementService(t *testing.T) {
	mockRepo := &repository.MockAchievementRepository{}
	mockUserRepo := &repository.MockUserRepository{}
	mockEventRepo := &repository.MockEventRepository{}
	mockEventCache := &cache.MockEventCacheClient{}
	mockUserCache := &cache.MockUserCacheClient{}
	mockAchievementCache := &cache.MockAchievementCacheClient{}
	mockNotification := &MockNotificationService{}

	tests := []struct {
		name        string
		repo        repository.AchievementRepository
		userRepo    repository.UserRepository
		eventRepo   repository.EventRepository
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
			eventRepo:   mockEventRepo,
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
			eventRepo:   mockEventRepo,
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
			eventRepo:   mockEventRepo,
			eventCache:  mockEventCache,
			userCache:   mockUserCache,
			achCache:    mockAchievementCache,
			notif:       mockNotification,
			expectedErr: true,
			errMsg:      "create new achievement service failed: user repository is nil",
		},
		{
			name:        "nil event repository",
			repo:        mockRepo,
			userRepo:    mockUserRepo,
			eventRepo:   nil,
			eventCache:  mockEventCache,
			userCache:   mockUserCache,
			achCache:    mockAchievementCache,
			notif:       mockNotification,
			expectedErr: true,
			errMsg:      "create new achievement service failed: event repository is nil",
		},
		{
			name:        "nil event cache client",
			repo:        mockRepo,
			userRepo:    mockUserRepo,
			eventRepo:   mockEventRepo,
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
			eventRepo:   mockEventRepo,
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
			eventRepo:   mockEventRepo,
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
			eventRepo:   mockEventRepo,
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
				tt.eventRepo,
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
				m.achievementCache.On("InvalidateUserAchievementsSummaries", mock.Anything, []string{user}).Return(nil)
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
				m.achievementCache.On("InvalidateUserAchievementsSummaries", mock.Anything, []string{user}).Return(nil)
				m.notificationSvc.On("Notify", mock.Anything).Return(nil)
			},
			rules:     []dto.Rule{rule},
			expectErr: "",
		},
		{
			name: "invalidate error does not abort notifications",
			setup: func(m mocks) {
				m.userCache.On("GetAll", mock.Anything).Return([]string{user}, nil)
				m.eventCache.On("GetUserEventsTimestampsByTypeAndRange", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return([]time.Time{time.Now()}, nil)
				m.achievementCache.On("AchievementExists", mock.Anything, mock.Anything).Return(false, nil)
				m.repo.On("SaveAll", mock.Anything).Return(nil)
				m.achievementCache.On("SaveAll", mock.Anything, mock.Anything).Return(nil)
				m.achievementCache.On("InvalidateUserAchievementsSummaries", mock.Anything, []string{user}).Return(errors.New("redis down"))
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
				eventRepository:        new(repository.MockEventRepository),
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

	windowed := rule
	windowed.Window = 3600

	streak := rule
	streak.Streak = true
	streak.Count = 1

	type mocks struct {
		eventCache       *cache.MockEventCacheClient
		achievementCache *cache.MockAchievementCacheClient
		repo             *repository.MockAchievementRepository
		eventRepo        *repository.MockEventRepository
	}

	tests := []struct {
		name        string
		setup       func(m mocks)
		rules       []dto.Rule
		expectedOut int
		notCalled   func(m mocks, t *testing.T)
	}{
		{
			name: "events cache error on windowed rule",
			setup: func(m mocks) {
				m.eventCache.On("GetUserEventsTimestampsByTypeAndRange", mock.Anything, user, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("cache error"))
			},
			rules:       []dto.Rule{windowed},
			expectedOut: 0,
		},
		{
			name: "count error",
			setup: func(m mocks) {
				m.achievementCache.On("AchievementExists", mock.Anything, mock.Anything).Return(false, nil)
				m.repo.On("ExistsInAllTime", mock.Anything).Return(false, nil)
				m.eventRepo.On("CountByActorAndType", mock.Anything, user, rule.EventType).Return(0, errors.New("db error"))
			},
			rules:       []dto.Rule{rule},
			expectedOut: 0,
		},
		{
			name: "count does not satisfy condition",
			setup: func(m mocks) {
				m.achievementCache.On("AchievementExists", mock.Anything, mock.Anything).Return(false, nil)
				m.repo.On("ExistsInAllTime", mock.Anything).Return(false, nil)
				m.eventRepo.On("CountByActorAndType", mock.Anything, user, rule.EventType).Return(0, nil)
			},
			rules:       []dto.Rule{rule},
			expectedOut: 0,
		},
		{
			name: "all-time cache error skips count",
			setup: func(m mocks) {
				m.achievementCache.On("AchievementExists", mock.Anything, mock.Anything).Return(false, errors.New("cache error"))
			},
			rules:       []dto.Rule{rule},
			expectedOut: 0,
		},
		{
			name: "all-time db error skips count",
			setup: func(m mocks) {
				m.achievementCache.On("AchievementExists", mock.Anything, mock.Anything).Return(false, nil)
				m.repo.On("ExistsInAllTime", mock.Anything).Return(false, errors.New("db error"))
			},
			rules:       []dto.Rule{rule},
			expectedOut: 0,
		},
		{
			name: "windowed rule uses cache",
			setup: func(m mocks) {
				m.eventCache.On("GetUserEventsTimestampsByTypeAndRange", mock.Anything, user, mock.Anything, mock.Anything, mock.Anything).
					Return([]time.Time{time.Now()}, nil)
				m.achievementCache.On("AchievementExists", mock.Anything, mock.Anything).Return(false, nil)
				m.repo.On("ExistsInAllTime", mock.Anything).Return(false, nil).Maybe()
			},
			rules:       []dto.Rule{windowed},
			expectedOut: 1,
		},
		{
			name: "window 0 streak uses cache",
			setup: func(m mocks) {
				m.eventCache.On("GetUserEventsTimestampsByTypeAndRange", mock.Anything, user, mock.Anything, mock.Anything, mock.Anything).
					Return([]time.Time{time.Now()}, nil)
				m.achievementCache.On("AchievementExists", mock.Anything, mock.Anything).Return(false, nil)
				m.repo.On("ExistsInAllTime", mock.Anything).Return(false, nil)
			},
			rules:       []dto.Rule{streak},
			expectedOut: 1,
		},
		{
			name: "all-time rule already granted in cache -> no DB count",
			setup: func(m mocks) {
				m.achievementCache.On("AchievementExists", mock.Anything, mock.Anything).Return(true, nil)
			},
			rules:       []dto.Rule{rule},
			expectedOut: 0,
			notCalled: func(m mocks, t *testing.T) {
				m.eventRepo.AssertNotCalled(t, "CountByActorAndType", mock.Anything, mock.Anything, mock.Anything)
				m.repo.AssertNotCalled(t, "ExistsInAllTime", mock.Anything)
			},
		},
		{
			name: "all-time rule granted in DB only -> no count",
			setup: func(m mocks) {
				m.achievementCache.On("AchievementExists", mock.Anything, mock.Anything).Return(false, nil)
				m.repo.On("ExistsInAllTime", mock.Anything).Return(true, nil)
			},
			rules:       []dto.Rule{rule},
			expectedOut: 0,
			notCalled: func(m mocks, t *testing.T) {
				m.eventRepo.AssertNotCalled(t, "CountByActorAndType", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name: "successful achievement generation",
			setup: func(m mocks) {
				m.eventRepo.On("CountByActorAndType", mock.Anything, user, rule.EventType).Return(1, nil)
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
				eventRepo:        new(repository.MockEventRepository),
			}

			tt.setup(mockObjs)

			service := &achievementService{
				eventCacheClient:       mockObjs.eventCache,
				achievementCacheClient: mockObjs.achievementCache,
				repository:             mockObjs.repo,
				eventRepository:        mockObjs.eventRepo,
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
			mockObjs.eventRepo.AssertExpectations(t)
			if tt.notCalled != nil {
				tt.notCalled(mockObjs, t)
			}
		})
	}
}

func TestAchievementService_reevalByDiff(t *testing.T) {
	diff := &dto.RuleDiff{
		Name:       "rule1",
		RewardDiff: 10,
	}

	type mocks struct {
		repo             *repository.MockAchievementRepository
		achievementCache *cache.MockAchievementCacheClient
		notificationSvc  *MockNotificationService
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
				m.achievementCache.On("InvalidateUserAchievementsSummaries", mock.Anything, []string{"user1", "user2"}).Return(nil)
				m.notificationSvc.On("Notify", mock.Anything).Return(errors.New("notify error"))
			},
			expectErr: "notify error",
		},
		{
			name: "successful path",
			setup: func(m mocks) {
				m.repo.On("ReevalByRuleDiff", diff).Return([]string{"user1"}, nil)
				m.achievementCache.On("InvalidateUserAchievementsSummaries", mock.Anything, []string{"user1"}).Return(nil)
				m.notificationSvc.On("Notify", mock.Anything).Return(nil)
			},
			expectErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockObjs := mocks{
				repo:             new(repository.MockAchievementRepository),
				achievementCache: new(cache.MockAchievementCacheClient),
				notificationSvc:  new(MockNotificationService),
			}

			tt.setup(mockObjs)

			service := &achievementService{
				repository:             mockObjs.repo,
				achievementCacheClient: mockObjs.achievementCache,
				notificationService:    mockObjs.notificationSvc,
			}

			err := service.reevalByDiff(diff)

			if tt.expectErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectErr)
			}

			mockObjs.repo.AssertExpectations(t)
			mockObjs.achievementCache.AssertExpectations(t)
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
		repo  *repository.MockAchievementRepository
		cache *cache.MockAchievementCacheClient
	}

	tests := []struct {
		name        string
		setup       func(m mocks)
		expected    *dto.AchievementsSummary
		expectedErr *api.APIError
	}{
		{
			name: "cache hit",
			setup: func(m mocks) {
				m.cache.On("GetUserAchievementsSummary", mock.Anything, user).
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
			expectedErr: nil,
		},
		{
			name: "cache miss repo hit",
			setup: func(m mocks) {
				m.cache.On("GetUserAchievementsSummary", mock.Anything, user).
					Return(nil, nil)
				m.repo.On("GetByUser", mock.Anything, user).
					Return(&dto.AchievementsSummary{
						Achievements: []dto.AchievementInfo{
							{RuleName: "r3", Reward: 3},
						},
					}, nil)
				m.cache.On("SetUserAchievementsSummary", mock.Anything, user,
					&dto.AchievementsSummary{
						Achievements: []dto.AchievementInfo{
							{RuleName: "r3", Reward: 3},
						},
					}).Return(nil)
			},
			expected: &dto.AchievementsSummary{
				Achievements: []dto.AchievementInfo{
					{RuleName: "r3", Reward: 3},
				},
			},
			expectedErr: nil,
		},
		{
			name: "cache miss, repo empty",
			setup: func(m mocks) {
				m.cache.On("GetUserAchievementsSummary", mock.Anything, user).
					Return(nil, nil)
				m.repo.On("GetByUser", mock.Anything, user).
					Return(&dto.AchievementsSummary{
						Achievements: []dto.AchievementInfo{},
						RewardSum:    0,
					}, nil)
				m.cache.On("SetUserAchievementsSummary", mock.Anything, user,
					&dto.AchievementsSummary{
						Achievements: []dto.AchievementInfo{},
						RewardSum:    0,
					}).Return(nil)
			},
			expected: &dto.AchievementsSummary{
				Achievements: []dto.AchievementInfo{},
				RewardSum:    0,
			},
			expectedErr: nil,
		},
		{
			name: "cache error",
			setup: func(m mocks) {
				m.cache.On("GetUserAchievementsSummary", mock.Anything, user).
					Return(nil, errors.New("cache err"))
			},
			expectedErr: api.Internal("Get achievements for user error", errors.New("cache err")),
		},
		{
			name: "repo error",
			setup: func(m mocks) {
				m.cache.On("GetUserAchievementsSummary", mock.Anything, user).
					Return(nil, nil)
				m.repo.On("GetByUser", mock.Anything, user).
					Return(nil, errors.New("db err"))
			},
			expectedErr: api.Internal("Get achievements for user error", errors.New("db err")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(repository.MockAchievementRepository)
			mockCache := new(cache.MockAchievementCacheClient)

			tt.setup(mocks{repo: mockRepo, cache: mockCache})

			svc := &achievementService{
				repository:             mockRepo,
				achievementCacheClient: mockCache,
			}

			out, err := svc.GetSummaryByUser(context.Background(), user)

			if tt.expectedErr != nil {
				assert.Nil(t, out)
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.expected, out)
			}

			mockRepo.AssertExpectations(t)
			mockCache.AssertExpectations(t)
		})
	}
}
