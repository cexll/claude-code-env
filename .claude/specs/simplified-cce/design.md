# Design Document for Simplified Claude Code Environment Switcher

## Overview

The Simplified Claude Code Environment Switcher (CCE) is designed following KISS principles to solve one specific problem: managing multiple Claude Code API configurations and launching Claude Code with the correct environment variables. The design eliminates all unnecessary complexity while maintaining reliability and usability.

**CRITICAL DESIGN CONSTRAINTS**: This implementation must address security vulnerabilities and reliability issues identified in validation (69/100 score) to achieve production readiness (95% threshold).

**Design Goals:**
- Under 300 lines of Go code total
- Use only Go standard library (except for minimal CLI parsing)
- Single binary with no external dependencies
- Simple, flat architecture with no interfaces or abstractions
- Direct, straightforward implementation
- **MANDATORY**: All error returns must be checked and handled
- **MANDATORY**: Secure (hidden) API key input implementation
- **MANDATORY**: Comprehensive unit tests with 80%+ coverage

## Architecture

### High-Level Architecture

The tool follows a simple linear architecture with three main phases:
1. **Configuration Management**: Load/save JSON configurations with full error handling
2. **Environment Selection**: Interactive or flag-based selection with input validation
3. **Process Execution**: Launch Claude Code with environment variables and error handling

### Core Components

#### 1. Configuration Structure
```go
type Environment struct {
    Name   string `json:"name"`
    URL    string `json:"url"`
    APIKey string `json:"api_key"`
}

type Config struct {
    Environments []Environment `json:"environments"`
}
```

#### 2. File Structure
```
main.go                 // Main entry point and CLI parsing (~80 lines)
config.go              // Configuration management (~100 lines)  
launcher.go            // Claude Code launching (~60 lines)
ui.go                  // Simple user interaction (~60 lines)
main_test.go           // Unit tests for main functionality
config_test.go         // Unit tests for configuration management
launcher_test.go       // Unit tests for launcher functionality
ui_test.go             // Unit tests for user interface
```

### Data Models

#### Configuration File Format
The configuration is stored as a simple JSON file at `~/.claude-code-env/config.json`:

```json
{
  "environments": [
    {
      "name": "production",
      "url": "https://api.anthropic.com",
      "api_key": "sk-ant-..."
    },
    {
      "name": "staging", 
      "url": "https://staging-api.anthropic.com",
      "api_key": "sk-ant-..."
    }
  ]
}
```

#### Environment Variables
The tool sets these environment variables before launching Claude Code:
- `ANTHROPIC_BASE_URL`: The base URL from the selected environment
- `ANTHROPIC_API_KEY`: The API key from the selected environment

## Components and Interfaces

### 1. Main Entry Point (main.go)
**Responsibilities:**
- Parse command line arguments using `flag` package
- Route to appropriate command handlers
- Coordinate between configuration, UI, and launcher components
- **Handle all error returns from called functions**

**Key Functions:**
- `main()`: Entry point with command routing and error handling
- `runDefault()`: Handle default behavior (interactive selection + launch) with full error checking
- `runList()`: Display all configured environments with error handling
- `runAdd()`: Add new environment configuration with input validation and error handling
- `runRemove()`: Remove environment configuration with confirmation and error handling

**Error Handling Requirements:**
- Check all function return values for errors
- Use appropriate exit codes (0=success, 1=general, 2=config, 3=launcher)
- Provide actionable error messages with context

### 2. Configuration Manager (config.go)
**Responsibilities:**
- Load and save configuration files with full error checking
- Validate configuration data comprehensively
- Manage configuration directory and file permissions
- **Handle ALL file operation errors explicitly**

**Key Functions:**
- `loadConfig() (Config, error)`: Read configuration from file, handle file not found, permission errors, JSON parsing errors
- `saveConfig(Config) error`: Write configuration to file with proper permissions, handle write errors, atomic operations
- `ensureConfigDir() error`: Create configuration directory if needed, handle permission errors
- `validateEnvironment(Environment) error`: Comprehensive validation of environment data
- `validateURL(string) error`: URL validation using net/url.Parse()
- `validateAPIKey(string) error`: Basic API key format validation

**Error Handling Strategy:**
- Return wrapped errors with context using fmt.Errorf()
- Check and handle os.Stat(), os.MkdirAll(), ioutil.ReadFile(), ioutil.WriteFile() errors
- Validate JSON parsing errors from json.Unmarshal()
- Handle atomic file operations (temp file + rename pattern)

### 3. User Interface (ui.go)
**Responsibilities:**
- Interactive environment selection with error handling
- **Secure API key input (hidden/masked characters)**
- Display formatted output with error checking
- Input validation and sanitization

**Key Functions:**
- `selectEnvironment([]Environment) (Environment, error)`: Interactive menu with input validation
- `promptForEnvironment() (Environment, error)`: Collect new environment details with validation
- `secureInput(prompt string) (string, error)`: **CRITICAL - Hidden input for API keys using termios/console API**
- `displayEnvironments([]Environment) error`: Format and display environment list with error handling
- `validateInput(string) error`: Input sanitization and validation

**Security Requirements:**
- Implement platform-specific secure input (Unix: termios, Windows: console API)
- Never echo sensitive characters to terminal
- Handle secure input errors gracefully
- Clear sensitive data from memory where possible

### 4. Claude Code Launcher (launcher.go)
**Responsibilities:**
- Set environment variables with validation
- Execute Claude Code process with full error handling
- Handle process exit codes and errors

**Key Functions:**
- `launchClaudeCode(Environment, []string) error`: Execute Claude Code with environment and error handling
- `checkClaudeCodeExists() error`: Verify Claude Code is in PATH using exec.LookPath()
- `prepareEnvironment(Environment) ([]string, error)`: Set up environment variables with validation
- `executeProcess(string, []string, []string) error`: Process execution with error handling

**Error Handling Requirements:**
- Check exec.LookPath() return value for claude-code existence
- Handle exec.Command() errors, process start failures, exit code capture
- Validate environment variable setting
- Provide detailed error messages for process execution failures

## Error Handling

### Error Strategy
Comprehensive error handling with descriptive messages and appropriate exit codes:
- Exit code 0: Success
- Exit code 1: General errors (file not found, invalid input)
- Exit code 2: Configuration errors
- Exit code 3: Claude Code execution errors

### Error Types and Patterns
Systematic error handling using Go standard patterns:
```go
func configError(operation, details string) error {
    return fmt.Errorf("configuration %s failed: %s", operation, details)
}

func validationError(field, reason string) error {
    return fmt.Errorf("validation failed for %s: %s", field, reason)
}

func launcherError(operation, details string) error {
    return fmt.Errorf("launcher %s failed: %s", operation, details)
}
```

### Mandatory Error Checks
ALL functions must check and handle error returns from:
- File operations: os.Open(), os.Create(), ioutil.ReadFile(), ioutil.WriteFile()
- JSON operations: json.Marshal(), json.Unmarshal()
- Process operations: exec.LookPath(), exec.Command().Start(), exec.Command().Wait()
- Network operations: url.Parse()
- System operations: os.MkdirAll(), os.Chmod()

## Testing Strategy

### Unit Testing Approach
**MANDATORY**: Comprehensive unit tests with minimum 80% coverage
- Test each function independently using standard `testing` package
- Use temporary directories for file system operations with proper cleanup
- Mock environment variables using `os.Setenv/os.Unsetenv` with cleanup
- Test ALL error conditions and edge cases
- Test both success and failure paths

### Test Coverage Requirements
**Critical Test Areas:**
- Configuration loading/saving with various file states (missing, corrupted, permission denied)
- Environment validation with malformed data (invalid URLs, empty fields, injection attempts)
- Command line argument parsing with invalid inputs
- Process execution scenarios (claude-code not found, execution failures)
- **Secure input functionality (API key masking)**
- File permission validation and creation
- Error propagation and handling

### Test Structure
```go
// Example test structure for each component
func TestLoadConfig_Success(t *testing.T) { /* test successful loading */ }
func TestLoadConfig_FileNotFound(t *testing.T) { /* test missing file */ }
func TestLoadConfig_PermissionDenied(t *testing.T) { /* test permission error */ }
func TestLoadConfig_InvalidJSON(t *testing.T) { /* test JSON parsing error */ }
```

### Integration Testing
- End-to-end test of full workflow with temporary directories
- Configuration file creation and modification with permission validation
- Environment variable setting and process launching (mocked)
- Error recovery and cleanup testing

## Security Considerations

### File Permissions
- Configuration directory: 0700 (owner read/write/execute only)
- Configuration file: 0600 (owner read/write only)
- **Validate permissions after creation and provide errors if setting fails**

### Secure Input Implementation
**CRITICAL SECURITY REQUIREMENT**: API key input must be completely hidden
```go
// Unix implementation using termios
func secureInputUnix(prompt string) (string, error) {
    fmt.Print(prompt)
    oldState, err := terminal.MakeRaw(int(syscall.Stdin))
    if err != nil {
        return "", fmt.Errorf("failed to set terminal raw mode: %w", err)
    }
    defer terminal.Restore(int(syscall.Stdin), oldState)
    
    // Read character by character without echo
    var input []byte
    buffer := make([]byte, 1)
    for {
        _, err := os.Stdin.Read(buffer)
        if err != nil {
            return "", fmt.Errorf("failed to read input: %w", err)
        }
        if buffer[0] == '\n' || buffer[0] == '\r' {
            break
        }
        if buffer[0] == 127 || buffer[0] == 8 { // backspace
            if len(input) > 0 {
                input = input[:len(input)-1]
            }
            continue
        }
        input = append(input, buffer[0])
    }
    fmt.Println() // newline after hidden input
    return string(input), nil
}
```

### Input Validation
- URL validation using `url.Parse()` with comprehensive error checking
- Environment name validation (alphanumeric, length limits, no special characters)
- API key format validation (length, prefix checks, basic structure)
- Input sanitization to prevent injection attacks

### Sensitive Data Handling
- Clear API keys from memory after use where possible
- Never log or display API keys in error messages
- Mask API keys in display output (show only first/last few characters)

## Dependencies

### Standard Library Only
- `encoding/json`: Configuration file parsing with error handling
- `flag`: Command line argument parsing with validation
- `fmt`: Formatted output and error formatting
- `io/ioutil`: File operations with comprehensive error checking
- `net/url`: URL validation with error handling
- `os`: Environment variables and process execution with error checking
- `os/exec`: Claude Code process launching with error handling
- `path/filepath`: File path operations
- `syscall`: File permissions and terminal control
- `golang.org/x/term`: Terminal control for secure input (standard extended library)

### Testing Dependencies
- `testing`: Standard testing framework
- `os`: Temporary directory creation and cleanup
- `path/filepath`: Test file path operations

## Implementation Simplifications

### Maintained Simplifications
- No external CLI framework (use standard `flag` package)
- No complex configuration library (use standard `encoding/json`)
- No advanced interactive UI (implement simple terminal control)
- No network validation (rely on Claude Code for connectivity)
- No configuration versioning or migration
- No caching or performance optimizations

### Critical Additions for Production Readiness
- **Comprehensive error handling for ALL operations**
- **Secure terminal input implementation**
- **Complete unit test coverage (80%+ minimum)**
- **Input validation and sanitization**
- **Proper file permission management**
- **Cross-platform secure input support**

### Trade-offs Maintained
- Limited network validation (basic URL parsing only)
- Simple error messages without advanced diagnostics
- No configuration backup or recovery mechanisms
- Basic interactive UI without advanced features
- No performance monitoring or analytics

This enhanced design maintains the 300-line constraint while ensuring production-level reliability, security, and testability to achieve the required 95% quality threshold.