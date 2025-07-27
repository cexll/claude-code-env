# Simplified Build for Claude Code Environment Switcher

.PHONY: build test clean help

# Default target
all: build

# Build the binary
build:
	go build -o cce .

# Run tests
test:
	go test -v ./...

# Test with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Run security tests
test-security:
	go test -v -run TestSecurity ./...

# Quality checks (format, vet, test)
quality: fmt vet test

# Clean build artifacts
clean:
	rm -f cce coverage.out coverage.html

# Install to system PATH
install: build
	sudo mv cce /usr/local/bin/

# Show help
help:
	@echo "Available targets:"
	@echo "  build         Build the CCE binary"
	@echo "  test          Run all tests"
	@echo "  test-coverage Generate test coverage report"
	@echo "  bench         Run performance benchmarks"
	@echo "  fmt           Format Go code"
	@echo "  vet           Run Go vet analysis"
	@echo "  test-security Run security-specific tests"
	@echo "  quality       Run format, vet, and test"
	@echo "  clean         Clean build artifacts"
	@echo "  install       Install to /usr/local/bin"
	@echo "  help          Show this help message"