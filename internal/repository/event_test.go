package repository

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/stretchr/testify/assert"
)

func TestNewEventRepository(t *testing.T) {
	tests := []struct {
		name        string
		storage     *sql.DB
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid storage",
			storage:     &sql.DB{},
			expectError: false,
		},
		{
			name:        "nil storage",
			storage:     nil,
			expectError: true,
			errorMsg:    "storage is nil",
		},
		{
			name:        "storage with mock connection",
			storage:     func() *sql.DB { db, _, _ := sqlmock.New(); return db }(),
			expectError: false,
		},
		{
			name:        "closed database connection",
			storage:     func() *sql.DB { db, _, _ := sqlmock.New(); db.Close(); return db }(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := NewEventRepository(tt.storage)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, repo)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, repo)
				assert.IsType(t, &eventRepository{}, repo)
				assert.Equal(t, tt.storage, repo.(*eventRepository).storage)
			}
		})
	}
}

func TestEventRepository_SaveAll(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &eventRepository{storage: db}

	tests := []struct {
		name        string
		mockFn      func()
		input       []dto.Event
		expectError bool
	}{
		{
			name: "success",
			mockFn: func() {
				mock.ExpectBegin()

				mock.ExpectPrepare("^COPY").
					ExpectExec().
					WithArgs("id1", "actor1", sqlmock.AnyArg(), "type1").
					WillReturnResult(sqlmock.NewResult(0, 1))

				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			},
			input: []dto.Event{
				{
					ID:        "id1",
					Actor:     "actor1",
					Timestamp: time.Now(),
					EventType: "type1",
				},
			},
			expectError: false,
		},
		{
			name: "begin failure",
			mockFn: func() {
				mock.ExpectBegin().WillReturnError(errors.New("fail"))
			},
			input:       []dto.Event{},
			expectError: true,
		},
		{
			name: "prepare failure",
			mockFn: func() {
				mock.ExpectBegin()
				mock.ExpectPrepare("^COPY").
					WillReturnError(errors.New("fail"))
				mock.ExpectRollback()
			},
			input:       []dto.Event{},
			expectError: true,
		},
		{
			name: "exec failure",
			mockFn: func() {
				mock.ExpectBegin()
				mock.ExpectPrepare("^COPY").
					ExpectExec().
					WithArgs("id1", "actor1", sqlmock.AnyArg(), "type1").
					WillReturnError(errors.New("fail"))
				mock.ExpectRollback()
			},
			input: []dto.Event{
				{
					ID:        "id1",
					Actor:     "actor1",
					Timestamp: time.Now(),
					EventType: "type1",
				},
			},
			expectError: true,
		},
		{
			name: "final exec failure",
			mockFn: func() {
				mock.ExpectBegin()
				mock.ExpectPrepare("^COPY").
					ExpectExec().
					WithArgs("id1", "actor1", sqlmock.AnyArg(), "type1").
					WillReturnResult(sqlmock.NewResult(0, 1))

				mock.ExpectExec(".*").WillReturnError(errors.New("fail"))
				mock.ExpectRollback()
			},
			input: []dto.Event{
				{
					ID:        "id1",
					Actor:     "actor1",
					Timestamp: time.Now(),
					EventType: "type1",
				},
			},
			expectError: true,
		},
		{
			name: "commit failure",
			mockFn: func() {
				mock.ExpectBegin()
				mock.ExpectPrepare("^COPY").
					ExpectExec().
					WithArgs("id1", "actor1", sqlmock.AnyArg(), "type1").
					WillReturnResult(sqlmock.NewResult(0, 1))

				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit().WillReturnError(errors.New("fail"))
			},
			input: []dto.Event{
				{
					ID:        "id1",
					Actor:     "actor1",
					Timestamp: time.Now(),
					EventType: "type1",
				},
			},
			expectError: true,
		},
		{
			name:        "events nil",
			mockFn:      func() {},
			input:       nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			err := repo.SaveAll(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestEventRepository_GetAllUsersWithEvents(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &eventRepository{storage: db}

	tests := []struct {
		name        string
		mockFn      func()
		expectData  []string
		expectError bool
	}{
		{
			name: "success with multiple users",
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"actor"}).
					AddRow("user1").
					AddRow("user2")
				mock.ExpectQuery("SELECT actor DISTINCT FROM events").WillReturnRows(rows)
			},
			expectData:  []string{"user1", "user2"},
			expectError: false,
		},
		{
			name: "query error",
			mockFn: func() {
				mock.ExpectQuery("SELECT actor DISTINCT FROM events").
					WillReturnError(errors.New("fail"))
			},
			expectData:  nil,
			expectError: true,
		},
		{
			name: "scan error",
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"actor"}).AddRow(nil)
				mock.ExpectQuery("SELECT actor DISTINCT FROM events").WillReturnRows(rows)
			},
			expectData:  nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			result, err := repo.GetAllUsersWithEvents()

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectData, result)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
