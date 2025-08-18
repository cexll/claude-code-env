# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Claude Code Environment Switcher (CCE) is a lightweight Go CLI tool that manages multiple Claude Code API endpoint configurations, allowing seamless switching between different environments (production, staging, custom API providers, etc.). The tool acts as a wrapper around Claude Code, injecting appropriate environment variables before launching.

## Commands

### Build and Test
```bash
# Build binary
make build              # or: go build -o cce .

# Run all tests
make test               # or: go test -v ./...

# Run specific test
go test -v -run TestValidateName ./...      # Run single test by name
go test -v -run "TestSecurity" ./...        # Run all security tests
go test -v -run "TestFlagPassthrough" ./...  # Run flag passthrough tests

# Test coverage
make test-coverage      # Generate HTML coverage report
go test -coverprofile=coverage.out ./...    # Generate coverage profile

# Performance benchmarks
make bench              # or: go test -bench=. -benchmem ./...

# Code quality
make quality            # Runs fmt + vet + test
make fmt                # Format all Go files
make vet                # Run static analysis

# Security tests
make test-security      # or: go test -v -run TestSecurity ./...

# Clean build artifacts
make clean              # Remove cce binary and coverage files
```

### Installation
```bash
# Install to system PATH
make install            # Builds and moves to /usr/local/bin/

# Development build
go build -o cce .       # Build binary in current directory
```

## Architecture

The project uses a minimalist 4-file architecture following KISS principles:

### Core Components

**`main.go`** (580+ lines)
- CLI entry point and command routing
- Two-phase argument parser for flag passthrough system
- Model validation with configurable patterns
- Environment validation (name, URL, API key)
- Help text generation with flag passthrough examples

**`config.go`** (367 lines)
- Atomic file operations with temp file + rename pattern
- Automatic backup creation before modifications
- Corruption recovery with `.backup` files
- JSON marshaling with proper indentation
- File permission management (0600 files, 0700 directories)

**`ui.go`** (1000+ lines)
- ANSI-free display core using carriage return and padding
- 4-tier progressive fallback system
- DisplayState tracking for stateful rendering
- TextPositioner for universal cursor control
- LineRenderer with differential updates
- Terminal width detection and responsive formatting

**`launcher.go`** (174 lines)
- Process execution with `exec.Command`
- Clean environment variable injection
- Comprehensive error handling with exit code preservation
- Signal forwarding for graceful shutdown

### Key Design Patterns

**Flag Passthrough System**
- Phase 1: Parse CCE-specific flags (`--env`, `-e`, `add`, `list`, `remove`)
- Phase 2: Collect remaining arguments for Claude Code
- Security validation prevents shell injection
- Supports `--` separator for explicit boundary

**ANSI-Free Display Management**
- Core functionality works without ANSI escape codes
- Uses carriage return (`\r`) and space padding for updates
- Progressive enhancement for capable terminals
- Stateful rendering prevents display accumulation

**4-Tier Terminal Fallback**
1. Full interactive: Arrow keys + ANSI enhancements
2. Basic interactive: Arrow keys without ANSI
3. Numbered selection: Simple numbered menu
4. Headless mode: Auto-select for CI/CD

**Configuration Atomicity**
- Write to temp file first
- Validate JSON structure
- Atomic rename to target
- Automatic backup before changes
- Recovery from corrupted configs

### Recent Enhancements

**Per-Environment API Key Variable** (2024)
- Choose between `ANTHROPIC_API_KEY` (default) and `ANTHROPIC_AUTH_TOKEN` per environment
- Runtime override with `-k` or `--key-var` flag
- Backward compatible with existing configurations

**Flag Passthrough System**
- Two-phase argument parsing separating CCE flags from Claude arguments
- Support for `--` separator for explicit argument separation
- Security validation preventing command injection

**ANSI-Free Display Management**
- DisplayState tracking with differential updates
- TextPositioner using carriage return and padding (no ANSI codes)
- LineRenderer for stateful menu rendering
- Smart truncation preserving essential information

**Additional Environment Variables**
- Configure custom variables per environment (e.g., `ANTHROPIC_SMALL_FAST_MODEL`)
- Interactive configuration during `cce add`
- Automatic injection when launching Claude Code

## Configuration

### File Location and Structure

Environments stored in `~/.claude-code-env/config.json`:
```json
{
  "environments": [
    {
      "name": "production",
      "url": "https://api.anthropic.com",
      "api_key": "sk-ant-api03-xxxxx",
      "api_key_env": "ANTHROPIC_API_KEY",
      "model": "claude-3-5-sonnet-20241022",
      "env_vars": {
        "ANTHROPIC_SMALL_FAST_MODEL": "claude-3-haiku-20240307"
      }
    }
  ]
}
```

### Environment Variables

**Model Validation**
- `CCE_MODEL_PATTERNS`: Comma-separated custom regex patterns
- `CCE_MODEL_STRICT`: Set to "false" for permissive mode with warnings

**Custom Variables per Environment**
- `ANTHROPIC_SMALL_FAST_MODEL`: Faster model for quick operations
- `ANTHROPIC_TIMEOUT`: Custom timeout values
- `ANTHROPIC_RETRY_COUNT`: Retry behavior configuration

## Testing

The project maintains 95%+ test coverage across 20+ test files:

**Test Categories**
- Unit tests for each core component
- Integration tests for end-to-end workflows
- Security tests for injection prevention and permissions
- Performance benchmarks for critical paths
- Terminal compatibility tests across different environments
- Error recovery and corrupted config handling

**Running Tests**
```bash
# All tests
go test -v ./...

# Specific test function
go test -v -run TestValidateName

# Test category (by naming pattern)
go test -v -run "TestSecurity"
go test -v -run "TestFlagPassthrough"
go test -v -run "TestUI"

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Security

**API Key Protection**
- Terminal raw mode for hidden input
- Masked display (first 6 + last 4 chars only)
- Never logged or exposed in process arguments

**File Security**
- Config files: 0600 permissions
- Config directory: 0700 permissions
- Atomic writes with backup

**Input Validation**
- URL format validation
- API key minimum length (10 chars)
- Name sanitization (alphanumeric + dash/underscore)
- Shell metacharacter detection in arguments

**Process Isolation**
- Clean environment variable injection
- No shell interpretation of arguments
- Secure command execution with `exec.Command`

## Dependencies

- `golang.org/x/term` v0.33.0: Secure terminal input
- `golang.org/x/sys` v0.34.0: System calls (indirect)
- Go 1.23.0+ required
- No external CLI frameworks (uses standard `flag` package)

## Troubleshooting

**Common Issues**

1. **"claude Code not found in PATH"**
   - Verify: `which claude`
   - Ensure Claude Code is installed

2. **Permission denied errors**
   - Check: `ls -la ~/.claude-code-env/`
   - Fix: `chmod 700 ~/.claude-code-env/`

3. **Display issues in terminal**
   - Try: Different terminal emulator
   - Check: `echo $TERM`
   - Fallback: Use `--env` flag for non-interactive

4. **Flag not recognized**
   - Use `--` to separate CCE and Claude flags
   - Example: `cce -- --help`

5. **API key validation fails**
   - Minimum 10 characters required
   - Check for trailing spaces