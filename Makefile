ITERATION ?= 1
COVERAGE_FILE ?= coverage.out
COVERAGE_FILTERED ?= coverage_filtered.out

.PHONY: build test help

build:
	go build -o build/server cmd/server/main.go

test:
	@echo "==> Running tests with coverage..."
	@go clean -testcache
	@go test ./... -coverprofile=$(COVERAGE_FILE)
	@grep -v -E '(mocks\.gen\.go)' $(COVERAGE_FILE) > $(COVERAGE_FILTERED)
	@go tool cover -func=$(COVERAGE_FILTERED)
	@rm $(COVERAGE_FILTERED)

