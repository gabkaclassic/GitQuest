package repository

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewUserRepository(t *testing.T) {
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
			repo, err := NewUserRepository(tt.storage)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, repo)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, repo)
				assert.IsType(t, &userRepository{}, repo)
				assert.Equal(t, tt.storage, repo.(*userRepository).storage)
			}
		})
	}
}

func TestUserRepository_Save(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &userRepository{storage: db}

	tests := []struct {
		name        string
		input       *dto.User
		mockFn      func()
		expectError bool
	}{
		{
			name:  "success",
			input: &dto.User{Email: "test@example.com", Password: "pass"},
			mockFn: func() {
				mock.ExpectQuery("INSERT INTO users \\(email, password\\) VALUES").
					WithArgs("test@example.com", "pass").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
			},
			expectError: false,
		},
		{
			name:        "nil user",
			input:       nil,
			mockFn:      func() {},
			expectError: true,
		},
		{
			name:  "query error",
			input: &dto.User{Email: "fail@example.com", Password: "pass"},
			mockFn: func() {
				mock.ExpectQuery("INSERT INTO users \\(email, password\\) VALUES").
					WithArgs("fail@example.com", "pass").
					WillReturnError(errors.New("db fail"))
			},
			expectError: true,
		},
		{
			name:  "scan error",
			input: &dto.User{Email: "test2@example.com", Password: "pass"},
			mockFn: func() {
				mock.ExpectQuery("INSERT INTO users \\(email, password\\) VALUES").
					WithArgs("test2@example.com", "pass").
					WillReturnRows(sqlmock.NewRows(nil))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			result, err := repo.Save(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_ExistsByEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &userRepository{storage: db}

	tests := []struct {
		name        string
		email       string
		mockFn      func()
		expectExist bool
		expectError bool
	}{
		{
			name:  "user exists",
			email: "exist@example.com",
			mockFn: func() {
				mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM users WHERE email = \\$1\\)").
					WithArgs("exist@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
			},
			expectExist: true,
			expectError: false,
		},
		{
			name:  "user does not exist",
			email: "noexist@example.com",
			mockFn: func() {
				mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM users WHERE email = \\$1\\)").
					WithArgs("noexist@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
			},
			expectExist: false,
			expectError: false,
		},
		{
			name:  "query error",
			email: "fail@example.com",
			mockFn: func() {
				mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM users WHERE email = \\$1\\)").
					WithArgs("fail@example.com").
					WillReturnError(errors.New("db fail"))
			},
			expectExist: false,
			expectError: true,
		},
		{
			name:  "scan error",
			email: "scanfail@example.com",
			mockFn: func() {
				mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM users WHERE email = \\$1\\)").
					WithArgs("scanfail@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(nil))
			},
			expectExist: false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			exists, err := repo.ExistsByEmail(tt.email)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectExist, exists)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_GetPasswordByEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &userRepository{storage: db}

	tests := []struct {
		name        string
		email       string
		mockFn      func()
		expectPass  string
		expectError bool
	}{
		{
			name:  "success",
			email: "user@example.com",
			mockFn: func() {
				mock.ExpectQuery("SELECT password FROM users WHERE email = \\$1").
					WithArgs("user@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"password"}).AddRow("hashedpass"))
			},
			expectPass:  "hashedpass",
			expectError: false,
		},
		{
			name:  "query error",
			email: "fail@example.com",
			mockFn: func() {
				mock.ExpectQuery("SELECT password FROM users WHERE email = \\$1").
					WithArgs("fail@example.com").
					WillReturnError(errors.New("db fail"))
			},
			expectPass:  "",
			expectError: true,
		},
		{
			name:  "scan error",
			email: "scanfail@example.com",
			mockFn: func() {
				mock.ExpectQuery("SELECT password FROM users WHERE email = \\$1").
					WithArgs("scanfail@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"password"}).AddRow(nil))
			},
			expectPass:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			pass, err := repo.GetPasswordByEmail(tt.email)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, "", pass)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectPass, pass)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
