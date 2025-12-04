ITERATION ?= 1
COVERAGE_FILE ?= coverage.out
COVERAGE_FILTERED ?= coverage_filtered.out

DB_URL = postgres://gitquest-user:password@localhost:5432/gitquest?sslmode=disable
MIGRATIONS_DIR = ./migrations

.PHONY: build test help

build:
	go build -o build/server cmd/server/main.go
	go build -o build/token cmd/token/main.go

test:
	@echo "==> Running tests with coverage..."
	@go clean -testcache
	@go test ./... -coverprofile=$(COVERAGE_FILE)
	@grep -v -E '(mocks\.gen\.go)|(main\.go)|(config\.go)|(app\.go)|(job\.go)' $(COVERAGE_FILE) > $(COVERAGE_FILTERED)
	@go tool cover -func=$(COVERAGE_FILTERED)
	@rm $(COVERAGE_FILTERED)

migrate-create:
    @read -p "Migration name: " name; \
    migrate create -seq -ext sql -dir $(MIGRATIONS_DIR) $$name

migrate-up:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" down 1

migrate-version:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" version