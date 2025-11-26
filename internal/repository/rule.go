package repository

import (
	"database/sql"
	"errors"

	"github.com/gabkaclassic/GitQuest/internal/dto"
)

type RuleRepository interface {
	SaveAll(rules *[]dto.Rule) error
	GetRule(name string, version string) (*dto.Rule, error)
}

type ruleRepository struct {
	storage *sql.DB
}

func NewRuleRepository(storage *sql.DB) (RuleRepository, error) {

	if storage == nil {
		return nil, errors.New("create new rule repository failed: storage is nil")
	}

	return &ruleRepository{
		storage: storage,
	}, nil
}

func (repository *ruleRepository) SaveAll(rules *[]dto.Rule) error {
	return executeWithRetry(func() error {
		tx, err := repository.storage.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		stmt, err := tx.Prepare(`
            INSERT INTO rules (name, description, event_type, "window", count, condition, streak, reward, version)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
            ON CONFLICT (name, version) DO NOTHING
        `)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, rule := range *rules {
			_, err = stmt.Exec(
				rule.Name, rule.Description, rule.EventType, rule.Window.ToNanoseconds(),
				rule.Count, string(rule.Condition), rule.Streak, rule.Reward, rule.Version,
			)
			if err != nil {
				return err
			}
		}

		return tx.Commit()
	})
}

func (repository *ruleRepository) GetRule(name string, version string) (*dto.Rule, error) {

	var rule dto.Rule
	var windowNs int64
	var conditionStr string

	err := repository.storage.QueryRow(
		`SELECT description, event_type, "window", count, condition, streak, reward FROM rules WHERE name = $1 AND version = $2`,
		name, version,
	).Scan(
		&rule.Description,
		&rule.EventType,
		&windowNs,
		&rule.Count,
		&conditionStr,
		&rule.Streak,
		&rule.Reward,
	)

	if err != nil {
		return nil, err
	}

	rule.Name = name
	rule.Version = version
	rule.Window = dto.FromNanoseconds(windowNs)
	rule.Condition = dto.Condition(conditionStr)

	return &rule, nil
}
