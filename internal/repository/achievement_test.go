package repository

import (
	"errors"
	"testing"
	"time"

	"database/sql"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/lib/pq"

	"github.com/stretchr/testify/assert"
)

func TestNewAchievementRepository(t *testing.T) {
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
			repo, err := NewAchievementRepository(tt.storage)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, repo)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, repo)
				assert.IsType(t, &achievementRepository{}, repo)
				assert.Equal(t, tt.storage, repo.(*achievementRepository).storage)
			}
		})
	}
}

func TestAchievementRepository_SaveAll(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &achievementRepository{storage: db}

	tests := []struct {
		name        string
		mockFn      func()
		input       []dto.Achievement
		expectError bool
	}{
		{
			name: "success",
			mockFn: func() {
				mock.ExpectBegin()

				mock.ExpectPrepare("^COPY").
					ExpectExec().
					WithArgs("user1", "rule1", "1", int64(10), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(0, 1))

				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			},
			input: []dto.Achievement{
				{
					User:        "user1",
					RuleName:    "rule1",
					RuleVersion: "1",
					Reward:      10,
					StartRange:  time.Now(),
					EndRange:    time.Now(),
				},
			},
			expectError: false,
		},
		{
			name: "begin failure",
			mockFn: func() {
				mock.ExpectBegin().WillReturnError(errors.New("fail"))
			},
			input:       []dto.Achievement{},
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
			input:       []dto.Achievement{},
			expectError: true,
		},
		{
			name: "exec failure",
			mockFn: func() {
				mock.ExpectBegin()

				mock.ExpectPrepare("^COPY").
					ExpectExec().
					WithArgs("user1", "rule1", "1", int64(5), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnError(errors.New("fail"))

				mock.ExpectRollback()
			},
			input: []dto.Achievement{
				{
					User:        "user1",
					RuleName:    "rule1",
					RuleVersion: "1",
					Reward:      5,
					StartRange:  time.Now(),
					EndRange:    time.Now(),
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
					WithArgs("user1", "rule1", "1", int64(5), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(0, 1))

				mock.ExpectExec(".*").WillReturnError(errors.New("fail"))
				mock.ExpectRollback()
			},
			input: []dto.Achievement{
				{
					User:        "user1",
					RuleName:    "rule1",
					RuleVersion: "1",
					Reward:      5,
					StartRange:  time.Now(),
					EndRange:    time.Now(),
				},
			},
			expectError: true,
		},
		{
			name: "close failure",
			mockFn: func() {
				mock.ExpectBegin()

				mock.ExpectPrepare("^COPY").
					ExpectExec().
					WithArgs("user1", "rule1", "1", int64(5), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(0, 1))

				mock.ExpectExec(".*").WillReturnError(errors.New("fail"))

				mock.ExpectRollback()
			},
			input: []dto.Achievement{
				{
					User:        "user1",
					RuleName:    "rule1",
					RuleVersion: "1",
					Reward:      5,
					StartRange:  time.Now(),
					EndRange:    time.Now(),
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
					WithArgs("user1", "rule1", "1", int64(5), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(0, 1))

				mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit().WillReturnError(errors.New("fail"))
			},
			input: []dto.Achievement{
				{
					User:        "user1",
					RuleName:    "rule1",
					RuleVersion: "1",
					Reward:      5,
					StartRange:  time.Now(),
					EndRange:    time.Now(),
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

func TestAchievementRepository_ExistsInAllTime(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &achievementRepository{storage: db}

	tests := []struct {
		name           string
		input          *dto.Achievement
		mockFn         func()
		expectedExists bool
		expectError    bool
		errorMessage   string
	}{
		{
			name: "achievement exists",
			input: &dto.Achievement{
				User:        "user1",
				RuleName:    "rule1",
				RuleVersion: "v1",
			},
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
				mock.ExpectQuery("^SELECT EXISTS").
					WithArgs("rule1", "v1", "user1").
					WillReturnRows(rows)
			},
			expectedExists: true,
			expectError:    false,
		},
		{
			name: "achievement does not exist",
			input: &dto.Achievement{
				User:        "user2",
				RuleName:    "rule2",
				RuleVersion: "v2",
			},
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
				mock.ExpectQuery("^SELECT EXISTS").
					WithArgs("rule2", "v2", "user2").
					WillReturnRows(rows)
			},
			expectedExists: false,
			expectError:    false,
		},
		{
			name: "database query error",
			input: &dto.Achievement{
				User:        "user3",
				RuleName:    "rule3",
				RuleVersion: "v3",
			},
			mockFn: func() {
				mock.ExpectQuery("^SELECT EXISTS").
					WithArgs("rule3", "v3", "user3").
					WillReturnError(errors.New("database connection error"))
			},
			expectedExists: false,
			expectError:    true,
			errorMessage:   "database connection error",
		},
		{
			name: "scan error - wrong column type",
			input: &dto.Achievement{
				User:        "user4",
				RuleName:    "rule4",
				RuleVersion: "v4",
			},
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow("not-a-bool")
				mock.ExpectQuery("^SELECT EXISTS").
					WithArgs("rule4", "v4", "user4").
					WillReturnRows(rows)
			},
			expectedExists: false,
			expectError:    true,
		},
		{
			name: "empty rule name",
			input: &dto.Achievement{
				User:        "user5",
				RuleName:    "",
				RuleVersion: "v5",
			},
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
				mock.ExpectQuery("^SELECT EXISTS").
					WithArgs("", "v5", "user5").
					WillReturnRows(rows)
			},
			expectedExists: false,
			expectError:    false,
		},
		{
			name: "empty rule version",
			input: &dto.Achievement{
				User:        "user6",
				RuleName:    "rule6",
				RuleVersion: "",
			},
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
				mock.ExpectQuery("^SELECT EXISTS").
					WithArgs("rule6", "", "user6").
					WillReturnRows(rows)
			},
			expectedExists: false,
			expectError:    false,
		},
		{
			name: "empty user",
			input: &dto.Achievement{
				User:        "",
				RuleName:    "rule7",
				RuleVersion: "v7",
			},
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
				mock.ExpectQuery("^SELECT EXISTS").
					WithArgs("rule7", "v7", "").
					WillReturnRows(rows)
			},
			expectedExists: false,
			expectError:    false,
		},
		{
			name:  "nil achievement",
			input: nil,
			mockFn: func() {
			},
			expectedExists: false,
			expectError:    true,
			errorMessage:   "achievement cannot be nil",
		},
		{
			name: "sql injection attempt",
			input: &dto.Achievement{
				User:        "user'; DROP TABLE achievements; --",
				RuleName:    "rule' OR '1'='1",
				RuleVersion: "v1' UNION SELECT * FROM users --",
			},
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
				mock.ExpectQuery("^SELECT EXISTS").
					WithArgs("rule' OR '1'='1", "v1' UNION SELECT * FROM users --", "user'; DROP TABLE achievements; --").
					WillReturnRows(rows)
			},
			expectedExists: false,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			exists, err := repo.ExistsInAllTime(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedExists, exists)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAchievementRepository_ReevalByRuleDiff(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &achievementRepository{storage: db}

	tests := []struct {
		name          string
		input         *dto.RuleDiff
		mockFn        func()
		expectedUsers []string
		expectError   bool
		errorMessage  string
	}{
		{
			name: "successful reeval with updated users",
			input: &dto.RuleDiff{
				Name:       "commit-rule",
				OldVersion: "hash1",
				NewVersion: "hash2",
				RewardDiff: 50,
			},
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"array_agg"}).AddRow(pq.Array([]string{"user1", "user2", "user3"}))
				mock.ExpectQuery(`^WITH updated_achievements AS`).
					WithArgs(50, "hash2", "commit-rule", "hash1").
					WillReturnRows(rows)
			},
			expectedUsers: []string{"user1", "user2", "user3"},
			expectError:   false,
		},
		{
			name: "successful reeval with single user",
			input: &dto.RuleDiff{
				Name:       "mr-rule",
				OldVersion: "old_hash",
				NewVersion: "new_hash",
				RewardDiff: 25,
			},
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"array_agg"}).AddRow(pq.Array([]string{"user42"}))
				mock.ExpectQuery(`^WITH updated_achievements AS`).
					WithArgs(25, "new_hash", "mr-rule", "old_hash").
					WillReturnRows(rows)
			},
			expectedUsers: []string{"user42"},
			expectError:   false,
		},
		{
			name: "successful reeval with no users affected",
			input: &dto.RuleDiff{
				Name:       "issue-rule",
				OldVersion: "ver1",
				NewVersion: "ver2",
				RewardDiff: -10,
			},
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"array_agg"}).AddRow(nil)
				mock.ExpectQuery(`^WITH updated_achievements AS`).
					WithArgs(-10, "ver2", "issue-rule", "ver1").
					WillReturnRows(rows)
			},
			expectedUsers: nil,
			expectError:   false,
		},
		{
			name: "successful reeval with empty array",
			input: &dto.RuleDiff{
				Name:       "empty-rule",
				OldVersion: "old",
				NewVersion: "new",
				RewardDiff: 0,
			},
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"array_agg"}).AddRow(pq.Array([]string{}))
				mock.ExpectQuery(`^WITH updated_achievements AS`).
					WithArgs(0, "new", "empty-rule", "old").
					WillReturnRows(rows)
			},
			expectedUsers: []string{},
			expectError:   false,
		},
		{
			name: "database query error",
			input: &dto.RuleDiff{
				Name:       "error-rule",
				OldVersion: "old",
				NewVersion: "new",
				RewardDiff: 100,
			},
			mockFn: func() {
				mock.ExpectQuery(`^WITH updated_achievements AS`).
					WithArgs(100, "new", "error-rule", "old").
					WillReturnError(errors.New("connection timeout"))
			},
			expectedUsers: nil,
			expectError:   true,
			errorMessage:  "connection timeout",
		},
		{
			name: "scan error - wrong array type",
			input: &dto.RuleDiff{
				Name:       "scan-error-rule",
				OldVersion: "v1",
				NewVersion: "v2",
				RewardDiff: 30,
			},
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"array_agg"}).AddRow("not-an-array")
				mock.ExpectQuery(`^WITH updated_achievements AS`).
					WithArgs(30, "v2", "scan-error-rule", "v1").
					WillReturnRows(rows)
			},
			expectedUsers: nil,
			expectError:   true,
		},
		{
			name: "rows iteration error",
			input: &dto.RuleDiff{
				Name:       "rows-error-rule",
				OldVersion: "ov",
				NewVersion: "nv",
				RewardDiff: 40,
			},
			mockFn: func() {
				mock.ExpectQuery(`^WITH updated_achievements AS`).
					WithArgs(40, "nv", "rows-error-rule", "ov").
					WillReturnError(errors.New("row iteration error"))
			},
			expectedUsers: nil,
			expectError:   true,
			errorMessage:  "row iteration error",
		},
		{
			name: "negative reward diff",
			input: &dto.RuleDiff{
				Name:       "negative-reward-rule",
				OldVersion: "v1",
				NewVersion: "v2",
				RewardDiff: -100,
			},
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"array_agg"}).AddRow(pq.Array([]string{"user1", "user2"}))
				mock.ExpectQuery(`^WITH updated_achievements AS`).
					WithArgs(-100, "v2", "negative-reward-rule", "v1").
					WillReturnRows(rows)
			},
			expectedUsers: []string{"user1", "user2"},
			expectError:   false,
		},
		{
			name: "zero reward diff",
			input: &dto.RuleDiff{
				Name:       "zero-reward-rule",
				OldVersion: "v1",
				NewVersion: "v2",
				RewardDiff: 0,
			},
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"array_agg"}).AddRow(pq.Array([]string{"user1"}))
				mock.ExpectQuery(`^WITH updated_achievements AS`).
					WithArgs(0, "v2", "zero-reward-rule", "v1").
					WillReturnRows(rows)
			},
			expectedUsers: []string{"user1"},
			expectError:   false,
		},
		{
			name: "empty rule name",
			input: &dto.RuleDiff{
				Name:       "",
				OldVersion: "old",
				NewVersion: "new",
				RewardDiff: 50,
			},
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"array_agg"}).AddRow(nil)
				mock.ExpectQuery(`^WITH updated_achievements AS`).
					WithArgs(50, "new", "", "old").
					WillReturnRows(rows)
			},
			expectedUsers: nil,
			expectError:   false,
		},
		{
			name: "empty versions",
			input: &dto.RuleDiff{
				Name:       "rule",
				OldVersion: "",
				NewVersion: "",
				RewardDiff: 10,
			},
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"array_agg"}).AddRow(nil)
				mock.ExpectQuery(`^WITH updated_achievements AS`).
					WithArgs(10, "", "rule", "").
					WillReturnRows(rows)
			},
			expectedUsers: nil,
			expectError:   false,
		},
		{
			name:  "nil rule diff",
			input: nil,
			mockFn: func() {
			},
			expectedUsers: nil,
			expectError:   true,
			errorMessage:  "diff cannot be nil",
		},
		{
			name: "same old and new version",
			input: &dto.RuleDiff{
				Name:       "same-version-rule",
				OldVersion: "same",
				NewVersion: "same",
				RewardDiff: 100,
			},
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"array_agg"}).AddRow(nil)
				mock.ExpectQuery(`^WITH updated_achievements AS`).
					WithArgs(100, "same", "same-version-rule", "same").
					WillReturnRows(rows)
			},
			expectedUsers: nil,
			expectError:   false,
		},
		{
			name: "sql injection attempt in rule name",
			input: &dto.RuleDiff{
				Name:       "rule'; DROP TABLE achievements; --",
				OldVersion: "old",
				NewVersion: "new",
				RewardDiff: 999,
			},
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"array_agg"}).AddRow(nil)
				mock.ExpectQuery(`^WITH updated_achievements AS`).
					WithArgs(999, "new", "rule'; DROP TABLE achievements; --", "old").
					WillReturnRows(rows)
			},
			expectedUsers: nil,
			expectError:   false,
		},
		{
			name: "large positive reward diff",
			input: &dto.RuleDiff{
				Name:       "large-reward-rule",
				OldVersion: "v1",
				NewVersion: "v2",
				RewardDiff: 1000000,
			},
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"array_agg"}).AddRow(pq.Array([]string{"user1"}))
				mock.ExpectQuery(`^WITH updated_achievements AS`).
					WithArgs(1000000, "v2", "large-reward-rule", "v1").
					WillReturnRows(rows)
			},
			expectedUsers: []string{"user1"},
			expectError:   false,
		},
		{
			name: "large negative reward diff",
			input: &dto.RuleDiff{
				Name:       "large-negative-reward-rule",
				OldVersion: "v1",
				NewVersion: "v2",
				RewardDiff: -1000000,
			},
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"array_agg"}).AddRow(pq.Array([]string{"user1"}))
				mock.ExpectQuery(`^WITH updated_achievements AS`).
					WithArgs(-1000000, "v2", "large-negative-reward-rule", "v1").
					WillReturnRows(rows)
			},
			expectedUsers: []string{"user1"},
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			users, err := repo.ReevalByRuleDiff(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
				assert.Nil(t, users)
			} else {
				assert.NoError(t, err)
				if tt.expectedUsers == nil {
					assert.Empty(t, users)
				} else {
					assert.NotNil(t, users)
					assert.Equal(t, tt.expectedUsers, users)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAchievementRepository_GetByUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &achievementRepository{storage: db}

	user := "u"

	tests := []struct {
		name        string
		mockFn      func()
		expect      *dto.AchievementsSummary
		expectError bool
	}{
		{
			name: "success",
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"rule_name", "reward", "total"}).
					AddRow("r1", 1, 3).
					AddRow("r2", 2, 3)

				mock.
					ExpectQuery(`SELECT rule_name, reward, SUM\(reward\) OVER\(\) AS total FROM achievements WHERE "user" = \$1`).
					WithArgs(user).
					WillReturnRows(rows)
			},
			expect: &dto.AchievementsSummary{
				Achievements: []dto.AchievementInfo{
					{RuleName: "r1", Reward: 1},
					{RuleName: "r2", Reward: 2},
				},
				RewardSum: 3,
			},
		},
		{
			name: "query error",
			mockFn: func() {
				mock.
					ExpectQuery(`SELECT rule_name, reward, SUM\(reward\) OVER\(\) AS total FROM achievements WHERE "user" = \$1`).
					WithArgs(user).
					WillReturnError(errors.New("db fail"))
			},
			expectError: true,
		},
		{
			name: "scan error",
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"rule_name", "reward", "total"}).
					AddRow("r1", "x", 3)

				mock.
					ExpectQuery(`SELECT rule_name, reward, SUM\(reward\) OVER\(\) AS total FROM achievements WHERE "user" = \$1`).
					WithArgs(user).
					WillReturnRows(rows)
			},
			expectError: true,
		},
		{
			name: "empty result",
			mockFn: func() {
				rows := sqlmock.NewRows([]string{"rule_name", "reward", "total"})
				mock.
					ExpectQuery(`SELECT rule_name, reward, SUM\(reward\) OVER\(\) AS total FROM achievements WHERE "user" = \$1`).
					WithArgs(user).
					WillReturnRows(rows)
			},
			expect: &dto.AchievementsSummary{
				Achievements: []dto.AchievementInfo{},
				RewardSum:    0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()

			out, err := repo.GetByUser(t.Context(), user)

			if tt.expectError {
				assert.Error(t, err)
				assert.NoError(t, mock.ExpectationsWereMet())
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expect, out)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
