.PHONY: run build test clean deps

# Run the application
run:
	go run cmd/server/main.go

# Build the application
build:
	go build -o bin/vshazam cmd/server/main.go

# Run tests
test:
	go test ./...

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
		echo "Air is not installed. Install it with: go install github.com/cosmtrek/air@latest"; \
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