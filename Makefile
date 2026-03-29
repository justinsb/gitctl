.PHONY: build clean run-backend run-frontend run-macos test

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

# Build macOS app bundle
build-macos:
	@echo "Building macOS app..."
	@cd cmd/gitctl-macos && swift build
	@mkdir -p bin/GitCtl.app/Contents/MacOS
	@cp cmd/gitctl-macos/.build/debug/GitCtl bin/GitCtl.app/Contents/MacOS/
	@cp cmd/gitctl-macos/Info.plist bin/GitCtl.app/Contents/
	@echo "Built bin/GitCtl.app"

# Build and run macOS native app
run-macos: build-macos
	@open bin/GitCtl.app

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
