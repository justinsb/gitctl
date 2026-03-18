.PHONY: proto build clean run-backend run-frontend test

# Default target
all: proto build

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	@PATH="$$PATH:$$(go env GOPATH)/bin" protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/gitctl.proto
	@echo "Protobuf code generated successfully"

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

# Install protoc plugins
install-tools:
	@echo "Installing protoc plugins..."
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "Tools installed successfully"

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
