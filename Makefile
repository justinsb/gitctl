.PHONY: build clean run-backend run-frontend build-macos run-macos install-macos build-ipad run-ipad test

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
	@rm -rf bin/ $(IPAD_DERIVED_DATA)
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

# Build macOS app bundle (includes Go backend)
build-macos:
	@echo "Building Go backend..."
	@go build -o bin/gitctl-backend ./cmd/gitctl-backend
	@echo "Building macOS app..."
	@cd cmd/gitctl-macos && swift build
	@mkdir -p bin/GitCtl.app/Contents/MacOS
	@cp cmd/gitctl-macos/.build/debug/GitCtl bin/GitCtl.app/Contents/MacOS/
	@cp bin/gitctl-backend bin/GitCtl.app/Contents/MacOS/
	@cp cmd/gitctl-macos/Info.plist bin/GitCtl.app/Contents/
	@echo "Built bin/GitCtl.app"

# Build and run macOS native app
run-macos: build-macos
	@bin/GitCtl.app/Contents/MacOS/GitCtl

# Install macOS app to /Applications
install-macos: build-macos
	@echo "Installing to /Applications..."
	@rm -rf /Applications/GitCtl.app
	@cp -R bin/GitCtl.app /Applications/
	@echo "Installed /Applications/GitCtl.app"

IPAD_SIMULATOR ?= iPad Pro 11-inch (M4)
IPAD_DERIVED_DATA = bin/DerivedData-ipad

# Build iPad app for simulator
build-ipad:
	@echo "Building iPad app for simulator..."
	@cd cmd/gitctl-macos && xcodebuild -scheme GitCtl \
		-destination 'platform=iOS Simulator,name=$(IPAD_SIMULATOR)' \
		-derivedDataPath ../../$(IPAD_DERIVED_DATA) \
		build 2>&1 | tail -1
	@mkdir -p bin/GitCtl-iPad.app
	@cp $(IPAD_DERIVED_DATA)/Build/Products/Debug-iphonesimulator/GitCtl bin/GitCtl-iPad.app/
	@cp cmd/gitctl-macos/Info-iOS.plist bin/GitCtl-iPad.app/Info.plist
	@echo "Built bin/GitCtl-iPad.app"

# Build and run iPad app in simulator
run-ipad: build-ipad
	@echo "Booting simulator..."
	@xcrun simctl boot '$(IPAD_SIMULATOR)' 2>/dev/null || true
	@open -a Simulator
	@echo "Installing app..."
	@xcrun simctl install '$(IPAD_SIMULATOR)' bin/GitCtl-iPad.app
	@echo "Launching app..."
	@xcrun simctl launch '$(IPAD_SIMULATOR)' com.justinsb.gitctl

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
