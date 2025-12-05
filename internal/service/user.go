package service

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"log/slog"
	"net/mail"

	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/gabkaclassic/GitQuest/internal/repository"
	"github.com/gabkaclassic/GitQuest/internal/storage"
	api "github.com/gabkaclassic/metrics/pkg/error"
	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	Authenticate(email string, password string) *api.APIError
	Register(email string, password string) (*uuid.UUID, *api.APIError)
}

type userService struct {
	repository repository.UserRepository
}

func NewUserService(repository repository.UserRepository) (UserService, error) {

	if repository == nil {
		return nil, errors.New("create new user service failed: repository is nil")
	}

	return &userService{
		repository: repository,
	}, nil
}

func (service *userService) Authenticate(email string, password string) *api.APIError {

	hashedPassword, err := service.repository.GetPasswordByEmail(email)

	if err != nil {
		if !storage.IsNotFoundError(err) {
			slog.Error("Authenticate user error", slog.String("email", email), slog.Any("error", err))
		}
		return api.Unauthorized("authorization failed")
	}

	err = verifyPassword(hashedPassword, password)

	if err != nil {
		return api.Unauthorized("authorization failed")
	}

	return nil
}

func (service *userService) Register(email string, password string) (*uuid.UUID, *api.APIError) {

	if err := validateEmail(email); err != nil {
		return nil, api.BadRequest(err.Error())
	}
	if err := validatePassword(password); err != nil {
		return nil, api.BadRequest(err.Error())
	}

	hashedPassword, err := hashPassword(password)

	if err != nil {
		slog.Error(
			"Hashing password error",
			slog.String("password", password),
			slog.Any("error", err),
		)
	}

	createdUserID, err := service.repository.Save(&dto.User{
		Email: email, Password: hashedPassword,
	})

	if err != nil {
		if storage.IsUniqueConstraintViolation(err) {
			return nil, api.BadRequest("User alerady exists")
		} else if !storage.IsStringDataRightTruncation(err) &&
			!storage.IsNotNullViolation(err) {
			slog.Error(
				"Register user error",
				slog.String("email", email),
				slog.String("password", password),
				slog.Any("error", err),
			)
		}
		return nil, api.BadRequest("Invalid user data")
	}

	slog.Info("Created user", slog.String("email", email), slog.Any("id", createdUserID))

	return createdUserID, nil
}

func validateEmail(email string) error {
	_, err := mail.ParseAddress(email)

	return err
}

func validatePassword(password string) error {

	if len(password) < 8 {
		return errors.New("minimum password length is 8")
	}

	return nil
}

func hashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedPassword), nil
}

func verifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
