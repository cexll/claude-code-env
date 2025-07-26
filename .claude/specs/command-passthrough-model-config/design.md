# Design Document: Command Pass-through Architecture and Model Configuration Support

## Overview

This document details the technical design for implementing command pass-through architecture and model configuration support in the Claude Code Environment Switcher (CCE). The solution transforms CCE from a simple environment switcher into a comprehensive Claude CLI wrapper while maintaining backward compatibility and existing performance characteristics.

### Key Design Goals

1. **Transparent Pass-through**: CCE becomes a drop-in replacement for Claude CLI
2. **Intelligent Command Routing**: Automatic detection and handling of CCE vs Claude CLI flags
3. **Environment-aware Model Configuration**: Per-environment model specifications with runtime injection
4. **Minimal Performance Overhead**: Sub-50ms delegation latency
5. **Backward Compatibility**: Seamless migration of existing configurations

## Architecture

### High-Level Component Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                           CCE CLI                               │
├─────────────────────────────────────────────────────────────────┤
│  Command Parser & Router                                        │
│  ├── Argument Analyzer                                          │
│  ├── Flag Classifier (CCE vs Claude)                           │
│  └── Delegation Decision Engine                                 │
├─────────────────────────────────────────────────────────────────┤
│  Environment Manager (Enhanced)                                 │
│  ├── Configuration Loader                                       │
│  ├── Model Configuration Handler                                │
│  ├── Environment Variable Injector                              │
│  └── Migration Manager                                          │
├─────────────────────────────────────────────────────────────────┤
│  Enhanced Launcher                                              │
│  ├── Pass-through Launcher                                      │
│  ├── Environment Injection Engine                               │
│  ├── Signal Forwarding Manager                                  │
│  └── Process Lifecycle Manager                                  │
├─────────────────────────────────────────────────────────────────┤
│  Interactive UI (Enhanced)                                      │
│  ├── Model Configuration Forms                                  │
│  ├── Enhanced Environment Display                               │
│  └── Combined Help System                                       │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow Architecture

```
User Command Input
       │
       ▼
┌─────────────────┐
│  Argument       │
│  Parser         │
└─────────────────┘
       │
       ▼
┌─────────────────┐    ┌─────────────────┐
│  Flag           │───▶│  CCE Flags      │
│  Classifier     │    │  (--env, etc.)  │
└─────────────────┘    └─────────────────┘
       │                       │
       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐
│  Claude Flags   │    │  Environment    │
│  (Pass-through) │    │  Selection      │
└─────────────────┘    └─────────────────┘
       │                       │
       │              ┌─────────────────┐
       │              │  Model Config   │
       │              │  Resolution     │
       │              └─────────────────┘
       │                       │
       ▼                       ▼
┌─────────────────────────────────────────┐
│  Environment Variable Injection         │
│  ├── ANTHROPIC_BASE_URL                 │
│  ├── ANTHROPIC_API_KEY                  │
│  └── ANTHROPIC_MODEL (if configured)    │
└─────────────────────────────────────────┘
       │
       ▼
┌─────────────────┐
│  Claude CLI     │
│  Process Launch │
└─────────────────┘
```

## Components and Interfaces

### 1. Command Parser & Router

#### 1.1 Argument Analyzer

**Purpose**: Parse and categorize command-line arguments to determine routing strategy.

```go
type ArgumentAnalyzer interface {
    AnalyzeArguments(args []string) (*ArgumentAnalysis, error)
    ClassifyFlags(args []string) (*FlagClassification, error)
    ExtractCCEFlags(args []string) (cceFlags *CCEFlags, remainingArgs []string, error)
}

type ArgumentAnalysis struct {
    HasCCEFlags    bool
    HasClaudeFlags bool
    RequiresPassthrough bool
    EnvironmentHints []string
}

type FlagClassification struct {
    CCEFlags    []string
    ClaudeFlags []string
    Conflicts   []FlagConflict
    Unknown     []string
}

type CCEFlags struct {
    Environment    string
    Config         string
    Verbose        bool
    NoInteractive  bool
    ShowVersion    bool
    ShowHelp       bool
}

type FlagConflict struct {
    Flag        string
    CCEValue    string
    ClaudeValue string
    Resolution  ConflictResolution
}
```

**Implementation Strategy**:
- Maintain a registry of known CCE and Claude CLI flags
- Use prefix matching and known patterns for flag classification
- Implement conflict resolution with precedence rules
- Cache flag classifications for performance

#### 1.2 Delegation Decision Engine

**Purpose**: Determine whether to handle commands internally or delegate to Claude CLI.

```go
type DelegationEngine interface {
    ShouldDelegate(analysis *ArgumentAnalysis) bool
    GetDelegationStrategy(args []string) DelegationStrategy
    PrepareDelegation(env *types.Environment, args []string) (*DelegationPlan, error)
}

type DelegationStrategy int

const (
    HandleInternally DelegationStrategy = iota
    DelegateWithEnvironment
    DelegateDirectly
    ShowCombinedHelp
)

type DelegationPlan struct {
    Strategy      DelegationStrategy
    Environment   *types.Environment
    ClaudeArgs    []string
    EnvVars       map[string]string
    WorkingDir    string
}
```

### 2. Enhanced Environment Management

#### 2.1 Model Configuration Handler

**Purpose**: Manage model specifications within environment configurations.

```go
type ModelConfigHandler interface {
    ValidateModelName(model string) error
    GetModelForEnvironment(env *types.Environment) string
    SetModelForEnvironment(env *types.Environment, model string) error
    GetSupportedModels() []string
}

// Extended Environment type
type Environment struct {
    Name        string            `json:"name"`
    Description string            `json:"description,omitempty"`
    BaseURL     string            `json:"base_url"`
    APIKey      string            `json:"api_key"`
    Model       string            `json:"model,omitempty"`           // NEW FIELD
    Headers     map[string]string `json:"headers,omitempty"`
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
    NetworkInfo *NetworkInfo      `json:"network_info,omitempty"`
}
```

#### 2.2 Configuration Migration Manager

**Purpose**: Handle configuration schema migrations for backward compatibility.

```go
type MigrationManager interface {
    GetConfigVersion(config *types.Config) string
    NeedsMigration(config *types.Config) bool
    MigrateConfig(config *types.Config) (*types.Config, error)
    CreateBackup(config *types.Config) error
}

type ConfigMigration struct {
    FromVersion string
    ToVersion   string
    Migrator    func(*types.Config) (*types.Config, error)
}
```

### 3. Enhanced Launcher System

#### 3.1 Pass-through Launcher

**Purpose**: Launch Claude CLI with proper environment injection and argument forwarding.

```go
type PassthroughLauncher interface {
    LaunchWithPassthrough(plan *DelegationPlan) error
    InjectEnvironmentVariables(env *types.Environment) map[string]string
    ForwardSignals(cmd *exec.Cmd) error
    PreservateExitCode(cmd *exec.Cmd) error
}

// Enhanced ClaudeCodeLauncher interface
type ClaudeCodeLauncher interface {
    Launch(env *types.Environment, args []string) error
    LaunchWithDelegation(plan *DelegationPlan) error     // NEW METHOD
    ValidateClaudeCode() error
    GetClaudeCodePath() (string, error)
    SetPassthroughMode(enabled bool)                     // NEW METHOD
}
```

#### 3.2 Environment Injection Engine

**Purpose**: Prepare and inject environment variables for Claude CLI execution.

```go
type EnvironmentInjector interface {
    PrepareEnvironment(env *types.Environment) map[string]string
    InjectBaseURL(env *types.Environment) string
    InjectAPIKey(env *types.Environment) string
    InjectModel(env *types.Environment) string
    InjectCustomHeaders(env *types.Environment) map[string]string
    ValidateInjection(envVars map[string]string) error
}

type InjectionResult struct {
    EnvVars     map[string]string
    Warnings    []string
    Masked      map[string]string  // For logging
}
```

### 4. Enhanced Interactive UI

#### 4.1 Model Configuration Forms

**Purpose**: Provide interactive model configuration during environment setup.

```go
// Enhanced InputField for model configuration
type ModelInputField struct {
    InputField
    ModelSuggestions []string
    ValidateModel    func(string) error
    ShowSuggestions  bool
}

// Enhanced InteractiveUI interface
type InteractiveUI interface {
    Select(label string, items []SelectItem) (int, string, error)
    Prompt(label string, validate func(string) error) (string, error)
    PromptPassword(label string, validate func(string) error) (string, error)
    PromptModel(label string, suggestions []string) (string, error)  // NEW METHOD
    Confirm(label string) (bool, error)
    MultiInput(fields []InputField) (map[string]string, error)
    ShowEnvironmentDetails(env *types.Environment, includeModel bool) // ENHANCED
}
```

## Data Models

### Enhanced Configuration Schema

```json
{
  "version": "1.1.0",
  "default_env": "production",
  "environments": {
    "production": {
      "name": "production",
      "description": "Production API endpoint",
      "base_url": "https://api.anthropic.com",
      "api_key": "sk-ant-...",
      "model": "claude-3-5-sonnet-20241022",
      "headers": {},
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z",
      "network_info": {
        "last_checked": "2024-01-01T00:00:00Z",
        "status": "success",
        "response_time_ms": 150,
        "ssl_valid": true
      }
    },
    "staging": {
      "name": "staging",
      "description": "Staging environment with fast model",
      "base_url": "https://staging-api.anthropic.com",
      "api_key": "sk-ant-staging-...",
      "model": "claude-3-haiku-20240307",
      "headers": {},
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  },
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### Command Routing Decision Tree

```
Input: cce [flags] [args]
│
├─ Contains --help/-h?
│  ├─ Yes → Show combined help (CCE + Claude CLI)
│  └─ No → Continue
│
├─ Contains CCE-specific flags? (--env, --config, etc.)
│  ├─ Yes → Parse CCE flags, determine environment
│  └─ No → Check if direct delegation needed
│
├─ Has non-CCE flags or arguments?
│  ├─ Yes → Prepare for delegation
│  ├─ No → Use current CCE behavior (environment selection)
│
├─ Environment selected/specified?
│  ├─ Yes → Inject environment variables
│  │   ├─ ANTHROPIC_BASE_URL
│  │   ├─ ANTHROPIC_API_KEY  
│  │   └─ ANTHROPIC_MODEL (if configured)
│  └─ No → Launch Claude CLI directly
│
└─ Launch Claude CLI with prepared environment and arguments
```

## Error Handling

### Enhanced Error Types

```go
// Enhanced error types for new functionality
type PassthroughError struct {
    Type        PassthroughErrorType
    Message     string
    Cause       error
    ClaudeArgs  []string
    Suggestions []string
}

type PassthroughErrorType int

const (
    ClaudeNotFoundError PassthroughErrorType = iota
    ArgumentParsingError
    EnvironmentInjectionError
    FlagConflictError
    DelegationError
)

type ModelConfigError struct {
    Type            ModelConfigErrorType
    Model           string
    Environment     string
    Message         string
    SuggestedModels []string
}

type ModelConfigErrorType int

const (
    InvalidModelName ModelConfigErrorType = iota
    ModelNotSupported
    ModelConfigMissing
    ModelValidationFailed
)
```

### Error Recovery Strategies

1. **Claude CLI Not Found**:
   - Attempt to find alternative Claude executables
   - Provide installation guidance with platform-specific instructions
   - Fall back to configuration-only mode

2. **Flag Conflicts**:
   - Apply precedence rules (CCE flags take priority)
   - Log conflicts in verbose mode
   - Provide conflict resolution suggestions

3. **Environment Injection Failures**:
   - Continue with partial environment injection
   - Log warnings for failed injections
   - Provide fallback to direct Claude CLI launch

4. **Model Configuration Errors**:
   - Suggest valid model names based on common patterns
   - Allow bypassing model specification
   - Provide model validation with helpful error messages

## Testing Strategy

### Unit Testing

1. **Argument Parsing Tests**:
   - Flag classification accuracy
   - Conflict resolution logic
   - Edge cases with complex argument patterns

2. **Environment Injection Tests**:
   - Variable preparation and validation
   - Model configuration handling
   - Security aspects (key masking, permissions)

3. **Configuration Migration Tests**:
   - Version detection and migration paths
   - Backup creation and restoration
   - Schema validation for new fields

### Integration Testing

1. **End-to-End Pass-through Tests**:
   - Command delegation with various Claude CLI flags
   - Environment variable injection verification
   - Signal handling and process lifecycle

2. **Configuration Management Tests**:
   - Model configuration CRUD operations
   - Interactive UI workflows for model setup
   - Backward compatibility with existing configs

3. **Cross-Platform Tests**:
   - Claude CLI discovery on different platforms
   - Path handling and argument escaping
   - Signal handling variations

### Performance Testing

1. **Command Routing Benchmarks**:
   - Argument parsing performance
   - Flag classification speed
   - Delegation overhead measurement

2. **Configuration Loading Benchmarks**:
   - Large configuration file handling
   - Model validation performance
   - Cache effectiveness testing

### Security Testing

1. **Environment Variable Security**:
   - API key masking in logs and error messages
   - Process environment isolation
   - Configuration file permissions

2. **Argument Injection Prevention**:
   - Shell injection attack vectors
   - Argument escaping validation
   - Process privilege isolation

## Performance Considerations

### Optimization Strategies

1. **Lazy Loading**:
   - Configuration files loaded only when needed
   - Claude CLI path resolution caching
   - Model validation caching

2. **Efficient Argument Processing**:
   - Single-pass argument parsing
   - Regex compilation caching
   - Flag classification memoization

3. **Minimal Memory Footprint**:
   - Stream processing for large configurations
   - Garbage collection optimization
   - Resource cleanup in error paths

### Performance Targets

- **Command routing latency**: < 50ms
- **Memory overhead**: < 10MB additional RAM
- **Configuration loading**: < 100ms for typical configs
- **Model validation**: < 10ms per model check

## Security Considerations

### Data Protection

1. **API Key Security**:
   - Environment variables scoped to child processes only
   - No plain-text logging of sensitive values
   - Secure cleanup of environment variable memory

2. **Configuration File Security**:
   - Maintain 600 permissions for config files
   - Atomic writes to prevent corruption
   - Backup files with same security restrictions

3. **Process Security**:
   - Minimal privilege principle for child processes
   - Signal handling without information leakage
   - Secure process termination and cleanup

### Input Validation

1. **Argument Sanitization**:
   - Prevent shell injection through argument crafting
   - Validate file paths and prevent directory traversal
   - Escape special characters appropriately

2. **Model Name Validation**:
   - Allowlist-based model name validation
   - Prevent injection through model specifications
   - Format validation for model identifiers

## Migration Strategy

### Configuration Schema Migration

1. **Automatic Detection**:
   - Version field comparison
   - Schema validation and feature detection
   - Graceful handling of unknown fields

2. **Migration Steps**:
   ```
   1. Detect configuration version
   2. Create backup with timestamp
   3. Apply version-specific migrations
   4. Validate migrated configuration
   5. Update version field
   6. Save with atomic write
   ```

3. **Rollback Capability**:
   - Automatic backup creation
   - Migration validation before commit
   - Error recovery with backup restoration

### Deployment Strategy

1. **Phased Rollout**:
   - Feature flags for new functionality
   - Gradual enablement of pass-through mode
   - Monitoring and rollback capabilities

2. **Backward Compatibility**:
   - Preserve existing CLI behavior by default
   - Opt-in activation for new features
   - Comprehensive regression testing

## Implementation Phases

### Phase 1: Core Pass-through Architecture
- Argument parsing and classification
- Basic delegation engine
- Environment variable injection
- Signal forwarding

### Phase 2: Model Configuration Support
- Configuration schema extension
- Model validation and management
- Interactive UI enhancements
- Migration system

### Phase 3: Enhanced Features
- Advanced conflict resolution
- Performance optimizations
- Comprehensive error handling
- Security hardening

### Phase 4: Polish and Documentation
- User experience improvements
- Comprehensive testing
- Documentation and examples
- Performance tuning