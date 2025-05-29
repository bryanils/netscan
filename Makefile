# Go Network Tool Makefile (Windows-compatible)

# Project settings
APP_NAME = netscan
SOURCE = main.go
BUILD_DIR = builds

# Active target
GOOS = windows
GOARCH = amd64
BINARY_NAME = $(APP_NAME)-windows-amd64.exe

# Build settings
BUILD_PATH = $(BUILD_DIR)/$(BINARY_NAME)
BUILD_FLAGS = -ldflags="-s -w"

# Default target
.PHONY: build
build: clean
	@echo Building $(BINARY_NAME) for $(GOOS)/$(GOARCH)...
	@if not exist $(BUILD_DIR) mkdir $(BUILD_DIR)
	@set GOOS=$(GOOS) && set GOARCH=$(GOARCH) && go build $(BUILD_FLAGS) -o $(BUILD_PATH) $(SOURCE)
	@echo Build complete: $(BUILD_PATH)
	@if exist $(BUILD_PATH) dir $(BUILD_PATH)

# Clean build directory
.PHONY: clean
clean:
	@echo Cleaning build directory...
	@if exist $(BUILD_DIR) rmdir /s /q $(BUILD_DIR)

.PHONY: build-all
build-all: clean
	@echo Building for all platforms...
	@if not exist $(BUILD_DIR) mkdir $(BUILD_DIR)
	@cmd /c "set GOOS=windows&& set GOARCH=amd64&& go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe $(SOURCE)"
	@cmd /c "set GOOS=linux&& set GOARCH=amd64&& go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(SOURCE)"
	@cmd /c "set GOOS=darwin&& set GOARCH=amd64&& go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-macos-amd64 $(SOURCE)"
	@cmd /c "set GOOS=darwin&& set GOARCH=arm64&& go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-macos-arm64 $(SOURCE)"
	@cmd /c "set GOOS=linux&& set GOARCH=arm64&& go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 $(SOURCE)"
	@cmd /c "set GOOS=windows&& set GOARCH=386&& go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-386.exe $(SOURCE)"
	@echo All builds complete!
	@if exist $(BUILD_DIR) dir $(BUILD_DIR)

# PowerShell version of build-all (more reliable on Windows)
.PHONY: build-all-ps
build-all-ps: clean
	@echo Building for all platforms using PowerShell...
	@if not exist $(BUILD_DIR) mkdir $(BUILD_DIR)
	@powershell -Command "$$env:GOOS='windows'; $$env:GOARCH='amd64'; go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe $(SOURCE)"
	@powershell -Command "$$env:GOOS='linux'; $$env:GOARCH='amd64'; go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(SOURCE)"
	@powershell -Command "$$env:GOOS='darwin'; $$env:GOARCH='amd64'; go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-macos-amd64 $(SOURCE)"
	@powershell -Command "$$env:GOOS='darwin'; $$env:GOARCH='arm64'; go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-macos-arm64 $(SOURCE)"
	@powershell -Command "$$env:GOOS='linux'; $$env:GOARCH='arm64'; go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 $(SOURCE)"
	@powershell -Command "$$env:GOOS='windows'; $$env:GOARCH='386'; go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-386.exe $(SOURCE)"
	@echo All builds complete!
	@if exist $(BUILD_DIR) dir $(BUILD_DIR)

# Run the application
.PHONY: run
run:
	@echo Running $(APP_NAME)...
	@go run $(SOURCE)

# Install dependencies
.PHONY: deps
deps:
	@echo Installing dependencies...
	@go mod tidy
	@go mod download

# Development build (current platform)
.PHONY: dev
dev:
	@echo Building development version...
	@go build -o $(APP_NAME).exe $(SOURCE)
	@echo Development build complete: ./$(APP_NAME).exe

# Show build info
.PHONY: info
info:
	@echo Build Information
	@echo ==================
	@echo App Name: $(APP_NAME)
	@echo Source: $(SOURCE)
	@echo Target OS: $(GOOS)
	@echo Target Arch: $(GOARCH)
	@echo Binary Name: $(BINARY_NAME)
	@echo Build Path: $(BUILD_PATH)
	@echo Build Flags: $(BUILD_FLAGS)

# Help
.PHONY: help
help:
	@echo Go Network Tool Build System
	@echo ===============================
	@echo.
	@echo Available targets:
	@echo   build        - Build for the configured target platform
	@echo   build-all    - Build for all supported platforms (cmd version)
	@echo   build-all-ps - Build for all supported platforms (PowerShell version)
	@echo   clean        - Remove build directory
	@echo   run          - Run the application directly
	@echo   dev          - Quick development build for current platform
	@echo   deps         - Install/update dependencies
	@echo   info         - Show current build configuration
	@echo   help         - Show this help message