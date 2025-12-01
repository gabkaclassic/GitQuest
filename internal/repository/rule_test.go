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

func TestNewRuleRepository(t *testing.T) {
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
			repo, err := NewRuleRepository(tt.storage)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, repo)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, repo)
				assert.IsType(t, &ruleRepository{}, repo)
				assert.Equal(t, tt.storage, repo.(*ruleRepository).storage)
			}
		})
	}
}

func TestRuleRepository_SaveAll(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &ruleRepository{storage: db}

	tests := []struct {
		name        string
		mockFn      func()
		input       *[]dto.Rule
		expectError bool
	}{
		{
			name: "success",
			mockFn: func() {
				mock.ExpectBegin()
				mock.ExpectPrepare("INSERT INTO rules .*").
					ExpectExec().
					WithArgs(
						"rule1", "desc1", "event1", int64(1000),
						int64(5), "cond1", true, int64(10), "v1",
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			input: &[]dto.Rule{
				{
					Name:        "rule1",
					Description: "desc1",
					EventType:   "event1",
					Window:      dto.TimeRange(time.Microsecond),
					Count:       5,
					Condition:   dto.Condition("cond1"),
					Streak:      true,
					Reward:      10,
					Version:     "v1",
				},
			},
			expectError: false,
		},
		{
			name:        "nil rules",
			mockFn:      func() {},
			input:       nil,
			expectError: true,
		},
		{
			name: "begin failure",
			mockFn: func() {
				mock.ExpectBegin().WillReturnError(errors.New("fail"))
			},
			input:       &[]dto.Rule{},
			expectError: true,
		},
		{
			name: "prepare failure",
			mockFn: func() {
				mock.ExpectBegin()
				mock.ExpectPrepare("INSERT INTO rules .*").WillReturnError(errors.New("fail"))
				mock.ExpectRollback()
			},
			input:       &[]dto.Rule{},
			expectError: true,
		},
		{
			name: "exec failure",
			mockFn: func() {
				mock.ExpectBegin()
				mock.ExpectPrepare("INSERT INTO rules .*").
					ExpectExec().
					WithArgs(
						"rule1", "desc1", "event1", int64(1000),
						int64(5), "cond1", true, int64(10), "v1",
					).
					WillReturnError(errors.New("fail"))
				mock.ExpectRollback()
			},
			input: &[]dto.Rule{
				{
					Name:        "rule1",
					Description: "desc1",
					EventType:   "event1",
					Window:      dto.TimeRange(time.Microsecond),
					Count:       5,
					Condition:   dto.Condition("cond1"),
					Streak:      true,
					Reward:      10,
					Version:     "v1",
				},
			},
			expectError: true,
		},
		{
			name: "commit failure",
			mockFn: func() {
				mock.ExpectBegin()
				mock.ExpectPrepare("INSERT INTO rules .*").
					ExpectExec().
					WithArgs(
						"rule1", "desc1", "event1", int64(1000),
						int64(5), "cond1", true, int64(10), "v1",
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit().WillReturnError(errors.New("fail"))
			},
			input: &[]dto.Rule{
				{
					Name:        "rule1",
					Description: "desc1",
					EventType:   "event1",
					Window:      dto.TimeRange(time.Microsecond),
					Count:       5,
					Condition:   dto.Condition("cond1"),
					Streak:      true,
					Reward:      10,
					Version:     "v1",
				},
			},
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

func TestRuleRepository_GetRule(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &ruleRepository{storage: db}

	tests := []struct {
		name        string
		mockFn      func()
		inputName   string
		inputVer    string
		expectRule  *dto.Rule
		expectError bool
	}{
		{
			name: "success",
			mockFn: func() {
				mock.ExpectQuery("SELECT description, event_type, \"window\", count, condition, streak, reward FROM rules WHERE name = \\$1 AND version = \\$2").
					WithArgs("rule1", "v1").
					WillReturnRows(sqlmock.NewRows([]string{
						"description", "event_type", "window", "count", "condition", "streak", "reward",
					}).AddRow("desc1", "event1", int64(1000000), int64(5), "cond1", true, int64(10)))
			},
			inputName: "rule1",
			inputVer:  "v1",
			expectRule: &dto.Rule{
				Name:        "rule1",
				Version:     "v1",
				Description: "desc1",
				EventType:   "event1",
				Window:      dto.FromNanoseconds(1000000),
				Count:       5,
				Condition:   "cond1",
				Streak:      true,
				Reward:      10,
			},
			expectError: false,
		},
		{
			name: "not found",
			mockFn: func() {
				mock.ExpectQuery("SELECT description, event_type, \"window\", count, condition, streak, reward FROM rules WHERE name = \\$1 AND version = \\$2").
					WithArgs("ruleX", "vX").
					WillReturnError(sql.ErrNoRows)
			},
			inputName:   "ruleX",
			inputVer:    "vX",
			expectRule:  nil,
			expectError: true,
		},
		{
			name: "scan error",
			mockFn: func() {
				mock.ExpectQuery("SELECT description, event_type, \"window\", count, condition, streak, reward FROM rules WHERE name = \\$1 AND version = \\$2").
					WithArgs("rule1", "v1").
					WillReturnRows(sqlmock.NewRows([]string{
						"description", "event_type", "window", "count", "condition", "streak", "reward",
					}).AddRow(nil, nil, nil, nil, nil, nil, nil))
			},
			inputName:   "rule1",
			inputVer:    "v1",
			expectRule:  nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			result, err := repo.GetRule(tt.inputName, tt.inputVer)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectRule, result)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRuleRepository_GetLastRule(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &ruleRepository{storage: db}

	tests := []struct {
		name        string
		mockFn      func()
		inputName   string
		expectRule  *dto.Rule
		expectError bool
	}{
		{
			name: "success",
			mockFn: func() {
				mock.ExpectQuery(`SELECT version, description, event_type, "window", count, condition, streak, reward FROM rules WHERE name = \$1 ORDER BY "created_at" DESC LIMIT 1`).
					WithArgs("rule1").
					WillReturnRows(sqlmock.NewRows([]string{
						"version", "description", "event_type", "window", "count", "condition", "streak", "reward",
					}).AddRow("v2", "desc2", "event1", int64(1000000), int64(5), "cond2", true, int64(10)))
			},
			inputName: "rule1",
			expectRule: &dto.Rule{
				Name:        "rule1",
				Version:     "v2",
				Description: "desc2",
				EventType:   "event1",
				Window:      dto.FromNanoseconds(1000000),
				Count:       5,
				Condition:   "cond2",
				Streak:      true,
				Reward:      10,
			},
			expectError: false,
		},
		{
			name: "not found",
			mockFn: func() {
				mock.ExpectQuery(`SELECT version, description, event_type, "window", count, condition, streak, reward FROM rules WHERE name = \$1 ORDER BY "created_at" DESC LIMIT 1`).
					WithArgs("ruleX").
					WillReturnError(sql.ErrNoRows)
			},
			inputName:   "ruleX",
			expectRule:  nil,
			expectError: true,
		},
		{
			name: "scan error",
			mockFn: func() {
				mock.ExpectQuery(`SELECT version, description, event_type, "window", count, condition, streak, reward FROM rules WHERE name = \$1 ORDER BY "created_at" DESC LIMIT 1`).
					WithArgs("rule1").
					WillReturnRows(sqlmock.NewRows([]string{
						"version", "description", "event_type", "window", "count", "condition", "streak", "reward",
					}).AddRow(nil, nil, nil, nil, nil, nil, nil, nil))
			},
			inputName:   "rule1",
			expectRule:  nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			result, err := repo.GetLastRule(tt.inputName)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectRule, result)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
