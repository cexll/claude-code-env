# Claude Code Environment Switcher (CCE)

A command-line tool for managing multiple Claude Code API endpoint configurations and seamlessly switching between them.

## Features

- **Environment Management**: Add, list, edit, and remove Claude Code environment configurations
- **Interactive Selection**: Choose environments through an intuitive menu interface  
- **Secure Storage**: Configuration stored with proper file permissions (600) in `~/.claude-code-env/config.json`
- **Seamless Integration**: Automatically launches Claude Code with the selected environment
- **Cross-Platform**: Works on macOS, Linux, and Windows

## Installation

### Build from Source

```bash
git clone <repository-url>
cd claude-code-env-switch
go build -o cce .
```

Move the `cce` binary to a directory in your PATH (e.g., `/usr/local/bin/`).

## Usage

### Basic Usage

Run CCE without arguments to launch Claude Code:
```bash
cce
```

If no environments are configured, Claude Code launches directly. If multiple environments exist, an interactive selection menu appears.

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

### Advanced Usage

#### Specify environment directly:
```bash
cce --env production
```

#### Pass arguments to Claude Code:
```bash
cce --env staging --verbose file.txt
```

#### Non-interactive mode:
```bash
cce --no-interactive --env production
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
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  }
}
```

## Security

- Configuration files are created with 600 permissions (owner read/write only)
- Configuration directory uses 700 permissions
- API keys are masked during input and not displayed in plain text
- Sensitive data is not logged

## Requirements

- Go 1.19+ (for building from source)
- Claude Code must be installed and available in PATH

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## License

[Add your license here]