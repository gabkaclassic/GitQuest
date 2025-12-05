package job

import (
	"context"
	"log/slog"
	"time"

	"github.com/gabkaclassic/GitQuest/internal/cache"
	"github.com/gabkaclassic/GitQuest/internal/config"
	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/gabkaclassic/GitQuest/internal/service"
)

type BackgroundJobs struct {
	eventCacheClient   cache.EventCacheClient
	achievementService service.AchievementService
	rules              []dto.Rule
	cfg                config.Jobs
	cleanupTicker      *time.Ticker
	calculateTicker    *time.Ticker
}

func NewBackgroundJobs(
	eventCacheClient cache.EventCacheClient,
	achievementService service.AchievementService,
	rules []dto.Rule,
	cfg config.Jobs,
) *BackgroundJobs {
	return &BackgroundJobs{
		eventCacheClient:   eventCacheClient,
		achievementService: achievementService,
		rules:              rules,
		cfg:                cfg,
	}
}

func StartBackgroundJobs(
	ctx context.Context,
	eventCacheClient cache.EventCacheClient,
	achievementService service.AchievementService,
	rules []dto.Rule,
	cfg config.Jobs,
) {
	jobs := NewBackgroundJobs(eventCacheClient, achievementService, rules, cfg)
	jobs.Run(ctx)
}

func (j *BackgroundJobs) Run(ctx context.Context) {
	j.cleanupTicker = time.NewTicker(j.cfg.Cleanup.Interval)
	j.calculateTicker = time.NewTicker(j.cfg.Calculate.Interval)
	defer j.Stop()

	for {
		select {
		case <-j.cleanupTicker.C:
			j.runCleanup(ctx)

		case <-j.calculateTicker.C:
			j.runCalculation(ctx)

		case <-ctx.Done():
			slog.Info("Background jobs shutting down...")
			return
		}
	}
}

func (j *BackgroundJobs) Stop() {
	if j.cleanupTicker != nil {
		j.cleanupTicker.Stop()
	}
	if j.calculateTicker != nil {
		j.calculateTicker.Stop()
	}
}

func (j *BackgroundJobs) runCleanup(ctx context.Context) {
	cleanupCtx, cancel := context.WithTimeout(ctx, j.cfg.Cleanup.Timeout)
	defer cancel()

	err := j.eventCacheClient.CleanupOldEventsFromCache(cleanupCtx)
	if err != nil {
		slog.Error("Cleanup old events error", slog.Any("error", err))
		return
	}
	slog.Info("Cleanup completed")
}

func (j *BackgroundJobs) runCalculation(ctx context.Context) {
	calculationCtx, cancel := context.WithTimeout(ctx, j.cfg.Calculate.Timeout)
	defer cancel()

	err := j.achievementService.CheckForNewAchievements(calculationCtx, j.rules)
	if err != nil {
		slog.Error("Calculate new achievements error", slog.Any("error", err))
		return
	}
	slog.Info("Calculate completed")
}
