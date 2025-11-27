package repository

import (
	"database/sql"
	"errors"
	"github.com/lib/pq"

	"github.com/gabkaclassic/GitQuest/internal/dto"
)

type AchievementRepository interface {
	SaveAll(achievements *[]dto.Achievement) error
}

type achievementRepository struct {
	storage *sql.DB
}

func NewAchievementRepository(storage *sql.DB) (AchievementRepository, error) {

	if storage == nil {
		return nil, errors.New("create new achievement repository failed: storage is nil")
	}

	return &achievementRepository{
		storage: storage,
	}, nil
}

func (repository *achievementRepository) SaveAll(achievements *[]dto.Achievement) error {
	return executeWithRetry(func() error {
		tx, err := repository.storage.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		stmt, err := tx.Prepare(pq.CopyIn("achievements", "user", "rule_name", "rule_version", "reward", "period_start", "period_end"))

		if err != nil {
			return err
		}

		for _, achievement := range *achievements {
			_, err = stmt.Exec(achievement.User, achievement.RuleName, achievement.RuleVersion, achievement.Reward, achievement.StartRange, achievement.EndRange)
			if err != nil {
				return err
			}
		}

		_, err = stmt.Exec()

		if err != nil {
			return err
		}

		err = stmt.Close()
		if err != nil {
			return err
		}

		return tx.Commit()
	})
}
