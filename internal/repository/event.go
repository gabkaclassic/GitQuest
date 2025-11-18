package repository

import (
	"database/sql"
	"errors"
	"github.com/lib/pq"

	"github.com/gabkaclassic/GitQuest/internal/dto"
)

type EventRepository interface {
	SaveAll(events *[]dto.Event) error
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

func (repository *eventRepository) SaveAll(events *[]dto.Event) error {
	return executeWithRetry(func() error {
		tx, err := repository.storage.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		stmt, err := tx.Prepare(pq.CopyIn("events", "id", "actor", "timestamp", "type"))

		if err != nil {
			return err
		}

		for _, event := range *events {
			_, err = stmt.Exec(event.ID, event.Actor, event.Timestamp, event.EventType)
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
