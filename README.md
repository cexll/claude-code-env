# Claude Code Environment Switcher (CCE)

A Go CLI tool that manages multiple Claude Code API endpoint configurations, allowing seamless switching between different environments (production, staging, custom API providers, etc.). CCE acts as a wrapper around Claude Code, injecting appropriate environment variables before launching.

## Features

- **Environment Management**: Add, list, edit, and remove Claude Code environment configurations
- **Model Support**: Configure specific models for each environment (GPT-4, Claude 3.5 Sonnet, etc.)
- **Pass-through Support**: Seamlessly forwards Claude CLI arguments while managing environments
- **Interactive Selection**: Choose environments through an intuitive menu interface using promptui
- **Network Validation**: Real-time connectivity testing with SSL certificate validation and caching
- **Secure Storage**: Configuration stored with proper file permissions (600) in `~/.claude-code-env/config.json`
- **Argument Analysis**: Intelligent routing between CCE commands and Claude CLI pass-through
- **Cross-Platform**: Works on macOS, Linux, and Windows
- **Comprehensive Testing**: Unit tests, integration tests, security tests, and performance benchmarks

## Installation

### Using Make (Recommended)

```bash
git clone https://github.com/cexll/claude-code-env.git
cd claude-code-env-switch
make build
```

### Manual Build

```bash
go build -o cce .
```

### Install to System PATH

```bash
make install  # Installs to /usr/local/bin/
```

### Cross-Platform Builds

```bash
make build-all  # Builds for macOS, Linux, and Windows
```

## Usage

### Basic Usage

#### Launch with Environment Selection
```bash
cce  # Interactive environment selection menu
```

#### Drop-in Replacement for Claude CLI
```bash
# All Claude CLI commands work through CCE
cce -r "You are a helpful assistant"  # Pass-through to Claude CLI
cce --env production -r "Debug this code"  # With specific environment
```

#### Launch Directly
If no environments are configured, Claude Code launches directly. With multiple environments, an interactive selection menu appears.

### Environment Management

#### Add a new environment:
```bash
cce env add production
```

#### List all environments:
```bash
cce env list
```

#### Edit an environment:
```bash
cce env edit production
```

#### Remove an environment:
```bash
cce env remove production
```

### Flags and Options

```bash
cce [flags] [claude-code-args...]

CCE-specific flags:
  --env, -e string      Environment name to use
  --config string       Config file path (default: ~/.claude-code-env/config.json)
  --verbose, -v         Verbose output
  --no-interactive      Disable interactive mode
  --help, -h            Show help
  --version             Show version

# All other flags are passed through to Claude CLI
```

## Configuration

Environments are stored in `~/.claude-code-env/config.json` with the following structure:

```json
{
  "version": "1.0.0",
  "default_env": "production",
  "environments": {
    "production": {
      "name": "production",
      "description": "Production Claude API",
      "base_url": "https://api.anthropic.com/v1",
      "api_key": "sk-ant-api03-xxxxx",
      "model": "claude-3-5-sonnet-20241022",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z",
      "validation_status": "valid",
      "ssl_info": {
        "valid": true,
        "expires_at": "2024-12-31T23:59:59Z"
      }
    },
    "staging": {
      "name": "staging",
      "description": "Staging Environment",
      "base_url": "https://staging-api.anthropic.com/v1",
      "api_key": "sk-ant-staging-xxxxx",
      "model": "claude-3-haiku-20240307",
      "created_at": "2024-01-15T11:00:00Z",
      "updated_at": "2024-01-15T11:00:00Z"
    }
  }
}
```

### Configuration Fields

- **base_url**: API endpoint URL
- **api_key**: Authentication key
- **model**: Specific model to use (optional, defaults to Claude CLI default)
- **description**: Human-readable description
- **validation_status**: Network validation result (valid/invalid/unknown)
- **ssl_info**: SSL certificate validation details

## Security

- Configuration files are created with 600 permissions (owner read/write only)
- Configuration directory uses 700 permissions
- API keys are masked during input and not displayed in plain text
- Sensitive data is not logged

## Development

### Build Commands

```bash
make build        # Build the binary
make build-all    # Cross-platform builds
make test         # Run all tests
make test-coverage # Run tests with coverage report
make quality      # Run all quality checks (format, vet, lint, test)
make dev          # Run in development mode with verbose output
```

### Project Structure

```
cmd/                    # Cobra CLI commands
├── root.go            # Main command with argument analysis
└── env.go             # Environment management commands

internal/              # Private packages
├── config/            # Configuration management
├── launcher/          # Claude Code process launching
├── network/           # Network validation with SSL
├── parser/            # Argument analysis and delegation
└── ui/                # Terminal UI with promptui

pkg/types/             # Public interfaces and types
test/                  # Comprehensive test suite
├── integration/       # Integration tests
├── security/          # Security validation tests
├── performance/       # Benchmark tests
└── suite/             # Shell script test suite
```

### Testing

```bash
make test                           # All tests
go test ./internal/config/...       # Config tests
go test ./internal/network/...      # Network validation tests
go test ./test/integration/...      # Integration tests
go test ./test/security/...         # Security tests
go test ./test/performance/...      # Performance benchmarks
```

## Architecture

**Interface-Driven Design**: Uses dependency injection with interfaces defined in `pkg/types/types.go`:
- `ConfigManager`: Configuration operations
- `NetworkValidator`: API connectivity testing  
- `InteractiveUI`: Terminal interactions
- `ClaudeCodeLauncher`: Process execution

**Key Features**:
- Structured error handling with actionable suggestions
- Network validation with SSL certificate checking and caching
- Atomic configuration file operations
- Argument analysis for intelligent command routing
- Pass-through support for seamless Claude CLI integration

## Requirements

- Go 1.24+ (for building from source)
- Claude Code must be installed and available in PATH

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes following the existing code patterns
4. Add tests for new functionality
5. Run `make quality` to ensure all checks pass
6. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.