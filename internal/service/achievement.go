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
)

const (
	workers = 8
)

type AchievementService interface {
	CheckForNewAchievements(ctx context.Context, rules *[]dto.Rule) error
}

type achievementService struct {
	repository             repository.AchievementRepository
	userRepository         repository.UserRepository
	userCacheClient        cache.UserCacheClient
	eventCacheClient       cache.EventCacheClient
	achievementCacheClient cache.AchievementCacheClient
}

func NewAchievementService(
	repository repository.AchievementRepository,
	userRepository repository.UserRepository,
	eventCacheClient cache.EventCacheClient,
	userCacheClient cache.UserCacheClient,
	achievementCacheClient cache.AchievementCacheClient,
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

	return &achievementService{
		repository:             repository,
		userRepository:         userRepository,
		eventCacheClient:       eventCacheClient,
		userCacheClient:        userCacheClient,
		achievementCacheClient: achievementCacheClient,
	}, nil
}

func (service *achievementService) CheckForNewAchievements(
	ctx context.Context,
	rules *[]dto.Rule,
) error {
	users, err := service.userCacheClient.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("get users from cache error %w", err)
	}
	tasks := make(chan string, len(*users))
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

	for _, user := range *users {
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

	if err := service.repository.SaveAll(&achievements); err != nil {
		return fmt.Errorf("save new achievements to DB error %w", err)
	}

	if err := service.achievementCacheClient.SaveAll(ctx, &achievements); err != nil {
		return fmt.Errorf("save new achievements to cache error %w", err)
	}

	return nil
}

func (service *achievementService) processUserEvents(
	ctx context.Context,
	user string,
	rules *[]dto.Rule,
	out chan<- dto.Achievement,
) error {
	now := time.Now()

	for _, rule := range *rules {
		startRange := now.Add(-time.Duration(rule.Window))

		eventsTimestamps, err := service.eventCacheClient.GetUserEventsTimestampsByTypeAndRange(ctx, user, rule.EventType, startRange, now)
		if err != nil {
			slog.Error("Failed get users events from cache operation", slog.String("user", user), slog.Any("rule", rule), slog.Any("error", err))
			continue
		}

		if rule.Streak && !checkStreak(eventsTimestamps, rule.Count) {
			continue
		}

		if !rule.Condition.Compare(len(*eventsTimestamps), rule.Count) {
			continue
		}

		achievement := dto.Achievement{
			User:        user,
			RuleName:    rule.Name,
			RuleVersion: rule.Version,
			Reward:      rule.Reward,
			StartRange:  startRange,
			EndRange:    now,
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

func checkStreak(timestamps *[]time.Time, days int) bool {
	if len(*timestamps) == 0 {
		return false
	}

	seen := make(map[int]struct{})
	start := time.Now().Add(time.Duration(-(days - 1)) * 24 * time.Hour).Truncate(24 * time.Hour)

	for _, t := range *timestamps {
		day := t.Truncate(24 * time.Hour)
		if day.Before(start) {
			continue
		}
		seen[int(day.Unix())] = struct{}{}
	}

	return len(seen) == days
}
