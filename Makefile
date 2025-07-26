# Claude Code Environment Switcher - Makefile

# Build variables
BINARY_NAME=cce
GO_VERSION=1.19
VERSION?=1.0.0
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT_HASH=$(shell git rev-parse HEAD 2>/dev/null || echo "unknown")

# Go build flags
LDFLAGS=-ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.commitHash=${COMMIT_HASH}"

# Directories
BUILD_DIR=build
DIST_DIR=dist

.PHONY: all build clean test install uninstall run help

# Default target
all: build

# Build the binary
build:
	@echo "Building ${BINARY_NAME}..."
	@go build ${LDFLAGS} -o ${BINARY_NAME} .
	@echo "Build complete: ${BINARY_NAME}"

# Build for multiple platforms
build-all: clean
	@echo "Building for multiple platforms..."
	@mkdir -p ${DIST_DIR}
	@GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${DIST_DIR}/${BINARY_NAME}-darwin-amd64 .
	@GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${DIST_DIR}/${BINARY_NAME}-darwin-arm64 .
	@GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${DIST_DIR}/${BINARY_NAME}-linux-amd64 .
	@GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o ${DIST_DIR}/${BINARY_NAME}-linux-arm64 .
	@GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${DIST_DIR}/${BINARY_NAME}-windows-amd64.exe .
	@echo "Cross-platform builds complete in ${DIST_DIR}/"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f ${BINARY_NAME}
	@rm -rf ${BUILD_DIR}
	@rm -rf ${DIST_DIR}
	@rm -f coverage.out coverage.html

# Install binary to system PATH
install: build
	@echo "Installing ${BINARY_NAME} to /usr/local/bin/"
	@sudo cp ${BINARY_NAME} /usr/local/bin/
	@echo "Installation complete"

# Uninstall binary from system PATH
uninstall:
	@echo "Uninstalling ${BINARY_NAME} from /usr/local/bin/"
	@sudo rm -f /usr/local/bin/${BINARY_NAME}
	@echo "Uninstallation complete"

# Run the application
run: build
	@./${BINARY_NAME}

# Development mode - run with verbose output
dev: build
	@./${BINARY_NAME} --verbose

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	@golangci-lint run

# Vet code
vet:
	@echo "Vetting code..."
	@go vet ./...

# Security scan
security:
	@echo "Running security scan..."
	@gosec ./...

# Full quality check
quality: fmt vet lint test
	@echo "Quality checks complete"

# Show help
help:
	@echo "Available commands:"
	@echo "  build        - Build the binary"
	@echo "  build-all    - Build for multiple platforms"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  deps         - Install dependencies"
	@echo "  clean        - Clean build artifacts"
	@echo "  install      - Install binary to system PATH"
	@echo "  uninstall    - Remove binary from system PATH"
	@echo "  run          - Build and run the application"
	@echo "  dev          - Run in development mode (verbose)"
	@echo "  fmt          - Format code"
	@echo "  lint         - Lint code"
	@echo "  vet          - Vet code"
	@echo "  security     - Run security scan"
	@echo "  quality      - Run all quality checks"
	@echo "  help         - Show this help message"