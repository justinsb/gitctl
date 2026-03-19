.PHONY: build clean run-backend run-frontend test

# Default target
all: build

# Build both binaries
build:
	@echo "Building backend..."
	@go build -o bin/gitctl-backend ./cmd/gitctl-backend
	@echo "Building frontend..."
	@go build -o bin/gitctl ./cmd/gitctl
	@echo "Build complete! Binaries in bin/"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f /tmp/gitctl.sock
	@echo "Clean complete"

# Run backend server
run-backend:
	@echo "Starting backend server..."
	@go run cmd/gitctl-backend/main.go

# Run frontend TUI
run-frontend:
	@echo "Starting frontend TUI..."
	@go run cmd/gitctl/main.go

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@echo "Dependencies downloaded"

# Tidy go.mod
tidy:
	@echo "Tidying go.mod..."
	@go mod tidy
	@echo "Tidy complete"
