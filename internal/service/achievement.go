package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gabkaclassic/GitQuest/internal/cache"
	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/gabkaclassic/GitQuest/internal/repository"
	"github.com/gabkaclassic/GitQuest/internal/storage"
	api "github.com/gabkaclassic/metrics/pkg/error"
	"github.com/google/uuid"
)

const (
	workers = 8
)

type AchievementService interface {
	CheckForNewAchievements(context.Context, []dto.Rule) error
	ReevalByDiffs([]dto.RuleDiff) error
	GetSummaryByUser(context.Context, string) (*dto.AchievementsSummary, *api.APIError)
}

type achievementService struct {
	repository             repository.AchievementRepository
	userRepository         repository.UserRepository
	userCacheClient        cache.UserCacheClient
	eventCacheClient       cache.EventCacheClient
	achievementCacheClient cache.AchievementCacheClient
	notificationService    NotificationService
}

func NewAchievementService(
	repository repository.AchievementRepository,
	userRepository repository.UserRepository,
	eventCacheClient cache.EventCacheClient,
	userCacheClient cache.UserCacheClient,
	achievementCacheClient cache.AchievementCacheClient,
	notificationService NotificationService,
) (AchievementService, error) {

	if repository == nil {
		return nil, errors.New("create new achievement service failed: achievement repository is nil")
	}

	if userRepository == nil {
		return nil, errors.New("create new achievement service failed: user repository is nil")
	}

	if eventCacheClient == nil {
		return nil, errors.New("create new achievement service failed: event cache client is nil")
	}

	if userCacheClient == nil {
		return nil, errors.New("create new achievement service failed: user cache client is nil")
	}

	if achievementCacheClient == nil {
		return nil, errors.New("create new achievement service failed: achievement cache client is nil")
	}

	if notificationService == nil {
		return nil, errors.New("create new achievement service failed: notifications service is nil")
	}

	return &achievementService{
		repository:             repository,
		userRepository:         userRepository,
		eventCacheClient:       eventCacheClient,
		userCacheClient:        userCacheClient,
		achievementCacheClient: achievementCacheClient,
		notificationService:    notificationService,
	}, nil
}

func (service *achievementService) GetSummaryByUser(ctx context.Context, user string) (*dto.AchievementsSummary, *api.APIError) {

	summary, err := service.repository.GetByUser(ctx, user)

	if err != nil {
		if storage.IsNotFoundError(err) {
			return &dto.AchievementsSummary{
				Achievements: []dto.AchievementInfo{},
			}, nil
		}

		return nil, api.Internal("Get achievements for user error", err)
	}

	return summary, nil
}

func (service *achievementService) CheckForNewAchievements(
	ctx context.Context,
	rules []dto.Rule,
) error {
	users, err := service.userCacheClient.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("get users from cache error %w", err)
	}
	tasks := make(chan string, len(users))
	results := make(chan dto.Achievement)

	wg := sync.WaitGroup{}
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for user := range tasks {
				service.processUserEvents(ctx, user, rules, results)
			}
		}()
	}

	for _, user := range users {
		tasks <- user
	}
	close(tasks)

	go func() {
		wg.Wait()
		close(results)
	}()

	achievements := make([]dto.Achievement, 0)
	for achievement := range results {
		achievements = append(achievements, achievement)
	}

	if len(achievements) == 0 {
		return nil
	}

	if err := service.repository.SaveAll(achievements); err != nil {
		return fmt.Errorf("save new achievements to DB error %w", err)
	}

	if err := service.achievementCacheClient.SaveAll(ctx, achievements); err != nil {
		return fmt.Errorf("save new achievements to cache error %w", err)
	}

	notifications := make([]dto.Notification, len(achievements))

	for ind, achievement := range achievements {
		notificationID, err := uuid.NewUUID()

		if err != nil {
			return fmt.Errorf("generate new notification ID error: %w", err)
		}

		notifications[ind] = dto.Notification{
			ID:              notificationID,
			User:            achievement.User,
			Reward:          achievement.Reward,
			RuleName:        achievement.RuleName,
			RuleDescription: achievement.RuleDescription,
		}
	}

	if err := service.notificationService.Notify(notifications); err != nil {
		return fmt.Errorf("send notification error: %w", err)
	}

	return nil
}

func (service *achievementService) processUserEvents(
	ctx context.Context,
	user string,
	rules []dto.Rule,
	out chan<- dto.Achievement,
) error {
	now := time.Now()

	for _, rule := range rules {
		startRange := now.Add(-time.Duration(rule.Window))

		eventsTimestamps, err := service.eventCacheClient.GetUserEventsTimestampsByTypeAndRange(ctx, user, rule.EventType, startRange, now)
		if err != nil {
			slog.Error("Failed get users events from cache operation", slog.String("user", user), slog.Any("rule", rule), slog.Any("error", err))
			continue
		}

		if rule.Streak && !checkStreak(eventsTimestamps, rule.Count) {
			continue
		}

		if !rule.Condition.Compare(len(eventsTimestamps), rule.Count) {
			continue
		}

		achievement := dto.Achievement{
			User:            user,
			RuleName:        rule.Name,
			RuleVersion:     rule.Version,
			RuleDescription: rule.Description,
			Reward:          rule.Reward,
			StartRange:      startRange,
			EndRange:        now,
		}

		exists, err := service.achievementCacheClient.AchievementExists(ctx, &achievement)

		if err != nil {
			slog.Error("Failed check achievement exists in cache", slog.String("user", user), slog.Any("rule", rule), slog.Any("achievement", achievement), slog.Any("error", err))
			continue
		}

		if exists {
			continue
		}

		if achievement.StartRange.Equal(achievement.EndRange) {
			exists, err = service.repository.ExistsInAllTime(&achievement)
			if err != nil {
				slog.Error("Failed check achievement exists in DB", slog.String("user", user), slog.Any("rule", rule), slog.Any("achievement", achievement), slog.Any("error", err))
				continue
			}

			if exists {
				continue
			}
		}

		out <- achievement
	}

	return nil
}

func (service *achievementService) ReevalByDiffs(diffs []dto.RuleDiff) error {

	var wg sync.WaitGroup
	errCh := make(chan error)

	for _, diff := range diffs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errCh <- service.reevalByDiff(&diff)
		}()
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	for err := range errCh {
		return err
	}

	return nil
}

func (service *achievementService) reevalByDiff(diff *dto.RuleDiff) error {
	users, err := service.repository.ReevalByRuleDiff(diff)

	if err != nil {
		slog.Error("Error reeval by rule diff", slog.Any("diff", diff), slog.Any("error", err))
		return err
	}

	notifications := make([]dto.Notification, len(users))

	for ind, user := range users {
		notificationID, err := uuid.NewUUID()

		if err != nil {
			slog.Error("Generate new notification ID error", slog.Any("error", err))
			return err
		}

		notifications[ind] = dto.Notification{
			ID:              notificationID,
			User:            user,
			Reward:          diff.RewardDiff,
			RuleName:        diff.Name,
			RuleDescription: fmt.Sprintf("Reward changed by %d", diff.RewardDiff),
		}
	}

	err = service.notificationService.Notify(notifications)

	if err != nil {
		slog.Error("Send notification error", slog.Any("error", err))
		return err
	}

	return nil
}

func checkStreak(timestamps []time.Time, days int) bool {
	if len(timestamps) == 0 {
		return false
	}

	seen := make(map[int]struct{})
	start := time.Now().Add(time.Duration(-(days - 1)) * 24 * time.Hour).Truncate(24 * time.Hour)

	for _, t := range timestamps {
		day := t.Truncate(24 * time.Hour)
		if day.Before(start) {
			continue
		}
		seen[int(day.Unix())] = struct{}{}
	}

	return len(seen) == days
}
