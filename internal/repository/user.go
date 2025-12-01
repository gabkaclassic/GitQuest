package repository

import (
	"database/sql"
	"errors"

	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/google/uuid"
)

type UserRepository interface {
	Save(user *dto.User) (*uuid.UUID, error)
	ExistsByEmail(email string) (bool, error)
	GetPasswordByEmail(email string) (string, error)
}

type userRepository struct {
	storage *sql.DB
}

func NewUserRepository(storage *sql.DB) (UserRepository, error) {

	if storage == nil {
		return nil, errors.New("create new user repository failed: storage is nil")
	}

	return &userRepository{
		storage: storage,
	}, nil
}

func (repository *userRepository) Save(user *dto.User) (*uuid.UUID, error) {

	if user == nil {
		return nil, errors.New("user cannot be nil")
	}

	createdUserID := uuid.UUID{}
	err := repository.storage.QueryRow(
		"INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id",
		user.Email, user.Password,
	).Scan(&createdUserID)

	if err != nil {
		return nil, err
	}

	return &createdUserID, nil
}

func (repository *userRepository) ExistsByEmail(email string) (bool, error) {

	var exists bool

	err := repository.storage.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email,
	).Scan(&exists)

	return exists, err
}

func (repository *userRepository) GetPasswordByEmail(email string) (string, error) {

	var password string

	err := repository.storage.QueryRow(
		"SELECT password FROM users WHERE email = $1", email,
	).Scan(&password)

	return password, err
}
