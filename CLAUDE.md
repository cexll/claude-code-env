# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Claude Code Environment Switcher (CCE) is a Go CLI tool that manages multiple Claude Code API endpoint configurations, allowing seamless switching between different environments (production, staging, custom API providers, etc.). The tool acts as a wrapper around Claude Code, injecting appropriate environment variables before launching.

## Common Development Commands

### Build and Run
```bash
# Build the binary
make build

# Build for all platforms (macOS, Linux, Windows)
make build-all

# Run the application
make run

# Development mode with verbose output
make dev
```

### Testing
```bash
# Run all tests
make test

# Run tests with coverage report (generates coverage.html)
make test-coverage

# Run specific test categories
go test ./internal/config/...          # Config manager tests
go test ./internal/network/...         # Network validation tests
go test ./test/integration/...         # Integration tests
go test ./test/security/...            # Security tests
go test ./test/performance/...         # Performance benchmarks

# Run a single test function
go test -run TestSpecificFunction ./internal/config/
```

### Code Quality
```bash
# Run all quality checks (format, vet, lint, test)
make quality

# Individual quality commands
make fmt        # Format code
make vet        # Go vet analysis
make lint       # golangci-lint (requires golangci-lint installed)
make security   # Security scan with gosec (requires gosec installed)
```

### Dependencies
```bash
# Install and clean dependencies
make deps
```

## Architecture Overview

### Core Components

**Interface-Driven Design**: The architecture uses dependency injection with clearly defined interfaces in `pkg/types/types.go`:
- `ConfigManager`: Configuration file operations and validation
- `NetworkValidator`: API endpoint connectivity testing
- `InteractiveUI`: Terminal-based user interactions
- `ClaudeCodeLauncher`: Process execution and environment injection

**Package Structure**:
- `cmd/`: Cobra CLI command definitions and orchestration
- `internal/config/`: Configuration file management with atomic operations
- `internal/network/`: Network validation with SSL certificate checking and caching
- `internal/ui/`: Interactive terminal UI using promptui
- `internal/launcher/`: Claude Code process launching with environment injection
- `pkg/types/`: Core interfaces, data structures, and structured error types
- `test/`: Comprehensive test suite with mocks, integration tests, and security validation

### Key Design Patterns

**Error Handling**: Structured error types (`ConfigError`, `EnvironmentError`, `NetworkError`, `LauncherError`) with actionable suggestions and recovery guidance.

**Network Validation**: Production-grade URL connectivity testing with SSL certificate validation, intelligent caching (TTL-based), and retry logic with exponential backoff.

**Security**: Configuration files stored with 600 permissions, API keys masked during input/display, no sensitive data in logs.

**Configuration Management**: Atomic file operations (temp file + rename pattern), automatic backup creation, validation with recovery mechanisms.

### Data Flow

1. **Environment Selection**: Interactive menu (promptui) or direct flag specification
2. **Network Validation**: Real-time connectivity testing with SSL verification
3. **Configuration Loading**: Secure config file loading with validation
4. **Environment Injection**: ANTHROPIC_BASE_URL and ANTHROPIC_API_KEY setup
5. **Process Launching**: Claude Code execution with argument forwarding

## Development Standards

### Function Organization
All functions maintain single responsibility and are under 50 lines. Complex operations are decomposed into focused helper functions with clear names and purposes.

### Documentation
Comprehensive godoc comments for all exported functions, types, and packages. Use `go doc` to view documentation locally.

### Testing Strategy
- Unit tests with mocks for all components
- Integration tests for complete workflows
- Security tests for file permissions and input validation
- Performance benchmarks for critical operations
- Cross-platform compatibility testing

### Network Operations
Network validation includes SSL certificate checking, connection timeouts, and caching with TTL. All network errors include diagnostic information and actionable suggestions.

## Configuration

Environments are stored in `~/.claude-code-env/config.json` with versioning support and migration capabilities. The configuration includes network validation status, SSL certificate information, and usage analytics.

## Dependencies

- **github.com/spf13/cobra**: CLI framework
- **github.com/manifoldco/promptui**: Interactive terminal UI
- **github.com/stretchr/testify**: Testing framework with mocks

## Security Considerations

- Configuration directory created with 700 permissions
- Configuration files created with 600 permissions  
- API keys masked during input and never displayed in plain text
- No sensitive data logged or exposed in error messages
- Input validation prevents injection attacks
- SSL certificate validation for HTTPS endpoints