package service

import (
	"errors"
	"github.com/redis/go-redis/v9"

	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/gabkaclassic/GitQuest/internal/repository"
)

type AchievementService interface {
	SaveAll(*[]dto.Achievement) error
}

type achievementService struct {
	repository repository.AchievementRepository
}

func NewAchievementService(repository repository.AchievementRepository, cache *redis.Client) (AchievementService, error) {

	if repository == nil {
		return nil, errors.New("create new achievement service failed: repository is nil")
	}

	return &achievementService{
		repository: repository,
	}, nil
}

func (service *achievementService) SaveAll(achievements *[]dto.Achievement) error {

	return service.repository.SaveAll(achievements)
}
