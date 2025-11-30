package repository

import (
	"database/sql"
	"errors"
	"github.com/lib/pq"

	"github.com/gabkaclassic/GitQuest/internal/dto"
)

type AchievementRepository interface {
	SaveAll(*[]dto.Achievement) error
	ExistsInAllTime(*dto.Achievement) (bool, error)
	ReevalByRuleDiff(*dto.RuleDiff) (*[]string, error)
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

func (repository *achievementRepository) ExistsInAllTime(achievement *dto.Achievement) (bool, error) {
	var exists bool

	err := repository.storage.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM achievements WHERE rule_name = $1 AND rule_version = $2 AND user = $3)",
		achievement.RuleName, achievement.RuleVersion, achievement.User,
	).Scan(&exists)

	return exists, err
}

func (repository *achievementRepository) ReevalByRuleDiff(diff *dto.RuleDiff) (*[]string, error) {
	var users []string
	err := executeWithRetry(func() error {
		rows, err := repository.storage.Query(`
			WITH updated_achievements AS (
				UPDATE achievements 
				SET reward = reward + $1,
					rule_version = $2
				WHERE rule_name = $3 AND rule_version = $4
				RETURNING "user"
			)
			SELECT array_agg("user")
			FROM updated_achievements;
		`, diff.RewardDiff, diff.NewVersion, diff.Name, diff.OldVersion)

		if err != nil {
			return err
		}
		defer rows.Close()

		if rows.Next() {
			if err := rows.Scan(pq.Array(&users)); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &users, nil
}
