package service

import (
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/gabkaclassic/GitQuest/internal/repository"
	"github.com/stretchr/testify/mock"
)

func TestNewUserService(t *testing.T) {
	tests := []struct {
		name        string
		repository  repository.UserRepository
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid repository",
			repository:  &repository.MockUserRepository{},
			expectError: false,
		},
		{
			name:        "nil repository",
			repository:  nil,
			expectError: true,
			errorMsg:    "repository is nil",
		},
		{
			name:        "mock repository implementation",
			repository:  &repository.MockUserRepository{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewUserService(tt.repository)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, service)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
				assert.IsType(t, &userService{}, service)

				svc := service.(*userService)
				assert.Equal(t, tt.repository, svc.repository)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		expectErr bool
	}{
		{"valid", "test@example.com", false},
		{"invalid no @", "testexample.com", true},
		{"invalid empty", "", true},
		{"invalid garbage", "lol@lol@lol", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEmail(tt.email)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		expectErr string
	}{
		{"ok", "12345678", ""},
		{"too short", "123", "minimum password length is 8"},
		{"empty", "", "minimum password length is 8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassword(tt.password)
			if tt.expectErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectErr)
			}
		})
	}
}

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		expectErr bool
	}{
		{"ok", "strongpass", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := hashPassword(tt.password)

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, hash)
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	hashed, err := hashPassword("mypassword")

	assert.NoError(t, err)

	tests := []struct {
		name      string
		hash      string
		password  string
		expectErr bool
	}{
		{"match", hashed, "mypassword", false},
		{"wrong password", hashed, "lolnope", true},
		{"hash empty", "", "mypassword", true},
		{"password empty", hashed, "", true},
		{"broken hash", "abcdef", "mypassword", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifyPassword(tt.hash, tt.password)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUserService_Authenticate(t *testing.T) {
	type mocks struct {
		repo *repository.MockUserRepository
	}

	tests := []struct {
		name      string
		email     string
		password  string
		setup     func(m mocks)
		expectErr string
	}{
		{
			name:     "repo unexpected error",
			email:    "user@example.com",
			password: "pass",
			setup: func(m mocks) {
				m.repo.On("GetPasswordByEmail", "user@example.com").
					Return("", errors.New("db error"))
			},
			expectErr: "authorization failed",
		},
		{
			name:     "wrong password",
			email:    "user@example.com",
			password: "badpass",
			setup: func(m mocks) {
				hash, _ := hashPassword("goodpass")
				m.repo.On("GetPasswordByEmail", "user@example.com").
					Return(hash, nil)
			},
			expectErr: "authorization failed",
		},
		{
			name:     "success",
			email:    "user@example.com",
			password: "mypassword",
			setup: func(m mocks) {
				hash, _ := hashPassword("mypassword")
				m.repo.On("GetPasswordByEmail", "user@example.com").
					Return(hash, nil)
			},
			expectErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockObjs := mocks{
				repo: new(repository.MockUserRepository),
			}

			tt.setup(mockObjs)

			service := &userService{
				repository: mockObjs.repo,
			}

			err := service.Authenticate(tt.email, tt.password)

			if tt.expectErr == "" {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
				assert.Contains(t, err.Message, tt.expectErr)
			}

			mockObjs.repo.AssertExpectations(t)
		})
	}
}

func TestUserService_Register(t *testing.T) {
	type mocks struct {
		repo *repository.MockUserRepository
	}

	validEmail := "user@example.com"
	validPass := "mypassword"

	createdID := uuid.New()

	tests := []struct {
		name      string
		email     string
		password  string
		setup     func(m mocks)
		expectErr string
		expectID  *uuid.UUID
	}{
		{
			name:      "invalid email",
			email:     "bad",
			password:  validPass,
			setup:     func(m mocks) {},
			expectErr: "mail: missing '@' or angle-addr",
		},
		{
			name:      "invalid password",
			email:     validEmail,
			password:  "123",
			setup:     func(m mocks) {},
			expectErr: "minimum password length is 8",
		},
		{
			name:     "unexpected repo error",
			email:    validEmail,
			password: validPass,
			setup: func(m mocks) {
				m.repo.On("Save", mock.Anything).
					Return(nil, errors.New("db error"))
			},
			expectErr: "Invalid user data",
		},
		{
			name:     "success",
			email:    validEmail,
			password: validPass,
			setup: func(m mocks) {
				m.repo.On("Save", mock.Anything).
					Return(&createdID, nil)
			},
			expectID: &createdID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockObjs := mocks{
				repo: new(repository.MockUserRepository),
			}

			tt.setup(mockObjs)

			service := &userService{
				repository: mockObjs.repo,
			}

			id, err := service.Register(tt.email, tt.password)

			if tt.expectErr == "" {
				assert.Nil(t, err)
				assert.Equal(t, tt.expectID, id)
			} else {
				assert.NotNil(t, err)
				assert.Contains(t, err.Message, tt.expectErr)
				assert.Nil(t, id)
			}

			mockObjs.repo.AssertExpectations(t)
		})
	}
}
