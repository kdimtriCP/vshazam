.PHONY: run build test clean deps

# Run the application
run:
	go run cmd/server/main.go

# Build the application
build:
	go build -o bin/vshazam cmd/server/main.go

# Run all tests
test:
	go test ./...

# Run unit tests only
test-unit:
	go test ./internal/...

# Run integration tests
test-integration:
	go test ./tests/integration/... -v

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/

# Download dependencies
deps:
	go mod download

# Tidy dependencies
tidy:
	go mod tidy

# Run with hot reload (requires air)
dev:
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Air is not installed. Install it with: go install github.com/air-verse/air@latest"; \
		echo "Running without hot reload..."; \
		go run cmd/server/main.go; \
	fi

# Format code
fmt:
	go fmt ./...

# Run linter (requires golangci-lint)
lint:
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint is not installed. Install it from https://golangci-lint.run/usage/install/"; \
	fi

# Docker commands
docker-build:
	docker-compose build

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

docker-clean:
	docker-compose down -v

# Data migration
migrate-data:
	go run scripts/migrate_sqlite_to_postgres.go