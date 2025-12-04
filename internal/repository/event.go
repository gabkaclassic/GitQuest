package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"

	"github.com/gabkaclassic/GitQuest/internal/dto"
)

type EventRepository interface {
	SaveAll(context.Context, []dto.Event) error
	GetAllUsersWithEvents(context.Context) ([]string, error)
}

type eventRepository struct {
	storage *sql.DB
}

func NewEventRepository(storage *sql.DB) (EventRepository, error) {

	if storage == nil {
		return nil, errors.New("create new event repository failed: storage is nil")
	}

	return &eventRepository{
		storage: storage,
	}, nil
}

func (repository *eventRepository) SaveAll(ctx context.Context, events []dto.Event) error {

	if events == nil {
		return errors.New("events cannot be nil")
	}

	return executeWithRetry(func() error {
		tx, err := repository.storage.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		stmt, err := tx.PrepareContext(ctx, pq.CopyIn("events", "id", "actor", "timestamp", "type"))

		if err != nil {
			return err
		}

		for _, event := range events {
			_, err = stmt.ExecContext(ctx, event.ID, event.Actor, event.Timestamp, event.EventType)
			if err != nil {
				return err
			}
		}

		_, err = stmt.ExecContext(ctx)

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

func (repository *eventRepository) GetAllUsersWithEvents(ctx context.Context) ([]string, error) {
	users := make([]string, 0)
	err := executeWithRetry(func() error {
		rows, err := repository.storage.QueryContext(ctx, "SELECT actor DISTINCT FROM events")
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var user string
			if err = rows.Scan(&user); err != nil {
				return err
			}
			users = append(users, user)
		}

		if err = rows.Err(); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return users, nil
}
