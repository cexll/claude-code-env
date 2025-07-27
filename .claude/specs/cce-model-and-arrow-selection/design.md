# Claude Code Environment Switcher (CCE) Enhancement Design

## Overview

This design document outlines the architecture and implementation approach for enhancing CCE with ANTHROPIC_MODEL environment variable support and interactive arrow key navigation. The enhancements maintain the existing KISS principle, staying within ~400 lines total while leveraging the existing golang.org/x/term dependency and established patterns.

## Architecture

### Enhanced Data Model

The existing `Environment` struct will be extended with an optional `Model` field:

```go
type Environment struct {
    Name   string `json:"name"`
    URL    string `json:"url"`
    APIKey string `json:"api_key"`
    Model  string `json:"model,omitempty"` // New optional field
}
```

**Design Decisions:**
- Use `omitempty` JSON tag for backward compatibility
- Place Model field last to maintain existing field order
- No breaking changes to existing configuration files
- Model field treated as optional throughout the system

### Model Validation System

A new validation function will be added following the existing pattern:

```go
func validateModel(model string) error
```

**Validation Logic:**
- Empty/missing models are valid (optional field)
- Non-empty models must match Anthropic naming conventions
- Regex pattern: `^claude-(3-5-sonnet|3-haiku|opus|sonnet)-[0-9]{8}$` or simplified aliases
- Comprehensive error messages with examples

### Terminal UI Enhancement Architecture

The arrow key navigation will extend the existing `selectEnvironment` function in `ui.go`:

```go
func selectEnvironmentInteractive(config Config) (Environment, error)
func selectEnvironmentWithArrows(config Config) (Environment, error) // New function
```

**Terminal Control Design:**
- Leverage existing `golang.org/x/term` dependency
- Use raw mode for direct key input capture
- Implement fallback to numbered selection on terminal issues
- Proper terminal state restoration on all exit paths

### Key Input Handling

```go
type keyEvent struct {
    Key   rune
    Arrow ArrowKey
}

type ArrowKey int
const (
    None ArrowKey = iota
    Up
    Down
    Left
    Right
)
```

**Key Mapping:**
- Arrow keys: `\x1b[A` (up), `\x1b[B` (down), `\x1b[C` (right), `\x1b[D` (left)
- Enter: `\n`, `\r`
- Escape: `\x1b`
- Ctrl+C: `\x03`

## Components and Interfaces

### 1. Enhanced Configuration Management (config.go)

**Minimal Changes Required:**
- No changes to loading/saving logic (JSON handles optional fields automatically)
- `validateEnvironment` function extended to call `validateModel`
- Backward compatibility maintained through `omitempty` JSON tag

### 2. Enhanced Environment Launcher (launcher.go)

**New Environment Variable Setup:**
```go
func prepareEnvironment(env Environment) ([]string, error) {
    // Existing logic for ANTHROPIC_BASE_URL and ANTHROPIC_API_KEY
    
    // New: Add ANTHROPIC_MODEL if specified
    if env.Model != "" {
        newEnv = append(newEnv, fmt.Sprintf("ANTHROPIC_MODEL=%s", env.Model))
    }
    
    return newEnv, nil
}
```

### 3. Enhanced User Interface (ui.go)

**Interactive Selection Enhancement:**
```go
func selectEnvironmentWithArrows(config Config) (Environment, error) {
    selectedIndex := 0
    
    for {
        displayEnvironmentMenu(config.Environments, selectedIndex)
        
        key, err := readSingleKey()
        if err != nil {
            return fallbackToNumberedSelection(config)
        }
        
        switch key {
        case ArrowUp:
            selectedIndex = (selectedIndex - 1 + len(config.Environments)) % len(config.Environments)
        case ArrowDown:
            selectedIndex = (selectedIndex + 1) % len(config.Environments)
        case Enter:
            return config.Environments[selectedIndex], nil
        case Escape, CtrlC:
            return Environment{}, fmt.Errorf("selection cancelled")
        }
    }
}
```

**Visual Menu Display:**
```go
func displayEnvironmentMenu(environments []Environment, selectedIndex int) {
    clearScreen()
    fmt.Println("Select environment (use ↑↓ arrows, Enter to confirm, Esc to cancel):")
    
    for i, env := range environments {
        prefix := "  "
        if i == selectedIndex {
            prefix = "► " // Visual selection indicator
        }
        
        modelDisplay := "default"
        if env.Model != "" {
            modelDisplay = env.Model
        }
        
        fmt.Printf("%s%s (%s) [%s]\n", prefix, env.Name, env.URL, modelDisplay)
    }
}
```

**Enhanced Environment Prompting:**
```go
func promptForEnvironment(config Config) (Environment, error) {
    // Existing name, URL, API key logic...
    
    // New: Prompt for optional model
    for {
        fmt.Print("Model (optional, press Enter for default): ")
        modelInput, err := regularInput("")
        if err != nil {
            return Environment{}, err
        }
        
        if modelInput == "" {
            break // Optional field, empty is valid
        }
        
        if err := validateModel(modelInput); err != nil {
            fmt.Printf("Invalid model: %v\n", err)
            continue
        }
        
        env.Model = modelInput
        break
    }
    
    return env, nil
}
```

### 4. Enhanced CLI Interface (main.go)

**Display Enhancement:**
```go
func displayEnvironments(config Config) error {
    for _, env := range config.Environments {
        maskedKey := maskAPIKey(env.APIKey)
        modelDisplay := env.Model
        if modelDisplay == "" {
            modelDisplay = "default"
        }
        
        fmt.Printf("\n  Name:  %s\n", env.Name)
        fmt.Printf("  URL:   %s\n", env.URL)  
        fmt.Printf("  Model: %s\n", modelDisplay)
        fmt.Printf("  Key:   %s\n", maskedKey)
    }
    
    return nil
}
```

## Data Models

### Configuration File Format (Enhanced)

```json
{
  "environments": [
    {
      "name": "production",
      "url": "https://api.anthropic.com",
      "api_key": "sk-ant-api03-xxxxx"
    },
    {
      "name": "experimental", 
      "url": "https://api.anthropic.com",
      "api_key": "sk-ant-api03-yyyyy",
      "model": "claude-3-5-sonnet-20241022"
    }
  ]
}
```

**Backward Compatibility:**
- Existing configurations without `model` field load successfully
- Missing `model` field treated as empty string (default behavior)
- No migration required for existing users

### Environment Variable Mapping

| Environment Field | Environment Variable | Required |
|------------------|---------------------|----------|
| URL              | ANTHROPIC_BASE_URL  | Yes      |
| APIKey           | ANTHROPIC_API_KEY   | Yes      |
| Model            | ANTHROPIC_MODEL     | No       |

## Error Handling

### Model Validation Errors

```go
func validateModel(model string) error {
    if model == "" {
        return nil // Optional field
    }
    
    // Validate against known Anthropic model patterns
    validPatterns := []string{
        `^claude-3-5-sonnet-[0-9]{8}$`,
        `^claude-3-haiku-[0-9]{8}$`, 
        `^claude-opus-[0-9]{8}$`,
        `^claude-sonnet-[0-9]{8}$`,
        `^claude-(opus|sonnet)-4-[0-9]{8}$`,
    }
    
    for _, pattern := range validPatterns {
        if matched, _ := regexp.MatchString(pattern, model); matched {
            return nil
        }
    }
    
    return fmt.Errorf("invalid model format. Examples: claude-3-5-sonnet-20241022, claude-3-haiku-20240307")
}
```

### Terminal Control Error Handling

```go
func selectEnvironmentWithArrows(config Config) (Environment, error) {
    // Check terminal compatibility
    if !term.IsTerminal(int(syscall.Stdin)) {
        return fallbackToNumberedSelection(config)
    }
    
    // Set up raw mode with proper cleanup
    oldState, err := term.MakeRaw(int(syscall.Stdin))
    if err != nil {
        return fallbackToNumberedSelection(config)
    }
    
    defer func() {
        if err := term.Restore(int(syscall.Stdin), oldState); err != nil {
            fmt.Fprintf(os.Stderr, "Warning: failed to restore terminal: %v\n", err)
        }
    }()
    
    // Arrow key navigation logic with error handling
    // ...
}

func fallbackToNumberedSelection(config Config) (Environment, error) {
    fmt.Println("Arrow key navigation not supported, using numbered selection:")
    return selectEnvironment(config) // Use existing numbered selection
}
```

### Cross-Platform Terminal Key Codes

```go
func parseKeyInput(input []byte) (ArrowKey, rune, error) {
    if len(input) == 0 {
        return None, 0, fmt.Errorf("empty input")
    }
    
    // Single character keys
    if len(input) == 1 {
        switch input[0] {
        case '\n', '\r':
            return None, '\n', nil
        case '\x1b': // Escape
            return None, '\x1b', nil
        case '\x03': // Ctrl+C
            return None, '\x03', nil
        default:
            return None, rune(input[0]), nil
        }
    }
    
    // Arrow key sequences (cross-platform)
    if len(input) >= 3 && input[0] == '\x1b' && input[1] == '[' {
        switch input[2] {
        case 'A':
            return Up, 0, nil
        case 'B':
            return Down, 0, nil
        case 'C':
            return Right, 0, nil
        case 'D':
            return Left, 0, nil
        }
    }
    
    return None, 0, fmt.Errorf("unrecognized key sequence")
}
```

## Testing Strategy

### Model Field Testing

1. **Configuration Compatibility Tests:**
   - Load existing configs without model field
   - Save/load configs with mixed model field presence
   - JSON serialization/deserialization validation

2. **Model Validation Tests:**
   - Valid Anthropic model names (various formats)
   - Invalid model names and error messages
   - Empty/missing model handling

3. **Environment Variable Tests:**
   - ANTHROPIC_MODEL set when model specified
   - ANTHROPIC_MODEL not set when model empty
   - Environment variable precedence and isolation

### Arrow Key Navigation Testing

1. **Terminal Compatibility Tests:**
   - Raw mode initialization and cleanup
   - Terminal state restoration on interrupts
   - Fallback to numbered selection

2. **Key Input Tests:**
   - Arrow key detection (up/down/left/right)
   - Enter key confirmation
   - Escape and Ctrl+C cancellation
   - Cross-platform key code handling

3. **Menu Display Tests:**
   - Environment list rendering
   - Selection highlighting
   - Wraparound navigation (first ↔ last)

### Integration Testing

1. **End-to-End Workflow Tests:**
   - Add environment with model specification
   - Arrow key selection with model display
   - Claude Code launch with ANTHROPIC_MODEL set

2. **Backward Compatibility Tests:**
   - Existing configurations continue working
   - Mixed old/new format configurations
   - No regression in existing functionality

## Implementation Constraints

### KISS Principle Adherence

1. **Code Size Limit:** Total enhancement should add ≤100 lines across all files
2. **No New Dependencies:** Use only existing golang.org/x/term and standard library  
3. **No New Abstractions:** Extend existing structs and functions directly
4. **Simple Logic:** Prefer straightforward implementations over clever optimizations

### File-by-File Enhancement Limits

- `main.go`: +20 lines (model validation, enhanced display)
- `config.go`: +10 lines (model field handling) 
- `ui.go`: +50 lines (arrow key navigation, enhanced prompting)
- `launcher.go`: +10 lines (ANTHROPIC_MODEL environment variable)
- **Total:** ~90 lines additional code

### Performance Considerations

- Terminal operations must be responsive (<100ms for key detection)
- Configuration loading/saving performance unchanged
- Memory usage minimal (no caching of terminal state)
- Claude Code launch time unaffected

### Security Considerations

- Model field stored with same file permissions as other config data (0600)
- Terminal raw mode properly cleaned up to prevent security issues
- No logging of sensitive terminal input
- Model validation prevents injection attacks through regex validation

## Migration Strategy

### Existing User Experience

1. **No Action Required:** Existing configurations work without modification
2. **Gradual Adoption:** Users can add model fields to environments as needed
3. **Feature Discovery:** Help text and prompts guide users to new features
4. **Fallback Support:** Arrow key navigation falls back gracefully

### Configuration Evolution

```json
// Phase 1: Existing configuration (continues to work)
{
  "environments": [
    {
      "name": "prod",
      "url": "https://api.anthropic.com", 
      "api_key": "sk-ant-xxx"
    }
  ]
}

// Phase 2: Enhanced configuration (after user adds model)
{
  "environments": [
    {
      "name": "prod",
      "url": "https://api.anthropic.com",
      "api_key": "sk-ant-xxx",
      "model": "claude-3-5-sonnet-20241022"
    }
  ]
}
```

### Deployment Approach

1. **Drop-in Replacement:** New binary works with existing configurations
2. **Feature Announcement:** Users discover new features through prompts
3. **Documentation Update:** README and help text reflect new capabilities
4. **Version Compatibility:** No breaking changes to CLI interface or config format