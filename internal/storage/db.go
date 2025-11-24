package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/gabkaclassic/GitQuest/internal/config"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/lib/pq"
)

func NewDBStorage(cfg config.DB) (*sql.DB, error) {
	var connectionString string

	if len(cfg.DSN) > 0 {
		connectionString = cfg.DSN
	} else {
		return nil, errors.New("DSN is required")
	}
	connection, err := sql.Open(
		cfg.Driver,
		connectionString,
	)

	if err != nil {
		return nil, err
	}

	err = connection.Ping()

	if err != nil {
		return nil, err
	}

	if err := runMigrations(connection, cfg); err != nil {
		connection.Close()
		return nil, fmt.Errorf("migrations failed: %w", err)
	}

	return connection, nil
}

func runMigrations(db *sql.DB, cfg config.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	migrationsPath := "file://migrations"
	if cfg.MigrationsPath != "" {
		migrationsPath = "file://" + cfg.MigrationsPath
	} else {
		return nil
	}

	m, err := migrate.NewWithDatabaseInstance(
		migrationsPath,
		"postgres",
		driver,
	)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return err
	}
	slog.Info("Migrations finished", slog.String("filepath", migrationsPath))

	return nil
}

func IsUniqueConstraintViolation(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23505" // unique_violation
	}
	return false
}

func GetUniqueConstraintName(err error) string {
	if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
		return pqErr.Constraint
	}
	return ""
}

func IsForeignKeyViolation(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23503" // foreign_key_violation
	}
	return false
}

func IsNotNullViolation(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23502" // not_null_violation
	}
	return false
}

func IsCheckConstraintViolation(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23514" // check_violation
	}
	return false
}

func IsStringDataRightTruncation(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "22001" // string_data_right_truncation
	}
	return false
}

func IsNotFoundError(err error) bool {
	return err == sql.ErrNoRows
}
