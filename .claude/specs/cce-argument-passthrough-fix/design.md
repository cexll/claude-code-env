# CCE Argument Passthrough Fix - Design Document

## Overview

This design document outlines the implementation approach for fixing the CCE argument passthrough issue while addressing code quality concerns identified in validation feedback. The solution eliminates code duplication between analyzer.go and preprocessor.go, simplifies complex control flow in cmd/root.go, and implements consistent error handling patterns to achieve 95%+ code quality.

## Problem Statement

**Current Issues (Validation Feedback)**:
1. **Code Duplication**: Flag extraction logic duplicated between analyzer.go and preprocessor.go
2. **Complex Control Flow**: cmd/root.go Execute() function has complex control flow (lines 49-92)
3. **Inconsistent Error Handling**: Mixed error return patterns (some return errors, others booleans)
4. **Missing Unit Tests**: No specific unit tests for new parser package components
5. **Function Length**: Several functions exceed the 50-line guideline
6. **Tight Coupling**: cmd/root.go tightly coupled to parser components

**Required Flow (Fixed)**:
```
User Input → Unified Flag Parser → Delegation Decision → Claude CLI Execution
             ↓ (if CCE-only)
             Cobra Command Parser → CCE Internal Handling
```

## Architecture

### High-Level Component Design

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   User Input    │───▶│ Unified Flag     │───▶│ Routing         │
│                 │    │ Parser           │    │ Controller      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │                          │
                              ▼                          ▼
                       ┌──────────────────┐    ┌─────────────────┐
                       │ Flag             │    │ Execution       │
                       │ Classification   │    │ Strategy        │
                       └──────────────────┘    └─────────────────┘
                              │                          │
                              ▼                          ▼
                       ┌──────────────────┐    ┌─────────────────┐
                       │ Delegation       │    │ Claude CLI      │
                       │ Decision         │    │ Launcher        │
                       └──────────────────┘    └─────────────────┘
```

### Key Design Principles

1. **Single Responsibility**: Each component has one clear purpose
2. **Dependency Injection**: Components depend on interfaces, not implementations
3. **Composition over Inheritance**: Build complex behavior through composition
4. **Fail Fast**: Validate inputs early and provide clear error messages
5. **Immutable Data**: Use immutable structures to prevent state corruption

## Components and Interfaces

### 1. Unified Flag Parser (New)

**File**: `internal/parser/flag_parser.go` (replaces duplicated logic)

**Purpose**: Single source of truth for all flag parsing operations.

**Interface**:
```go
type FlagParser interface {
    ParseFlags(args []string) (*ParseResult, error)
    ExtractCCEFlags(args []string) (*CCEFlags, []string, error)
    ClassifyFlags(args []string) (*FlagClassification, error)
    ValidateFlags(flags *CCEFlags) error
}

type ParseResult struct {
    CCEFlags       *CCEFlags
    RemainingArgs  []string
    Classification *FlagClassification
    Metadata       *ParseMetadata
}

type ParseMetadata struct {
    ProcessingTime time.Duration
    WarningsFound  []string
    DebugInfo      map[string]interface{}
}
```

**Implementation Strategy**:
- Extract common flag parsing logic from analyzer.go and preprocessor.go
- Implement single flag extraction algorithm with configurable output formats
- Provide helper methods for specific use cases (CCE-only, delegation, etc.)
- Use builder pattern for complex parsing configurations

### 2. Routing Controller (New)

**File**: `internal/routing/controller.go` (simplifies cmd/root.go)

**Purpose**: Orchestrate the execution flow and reduce cmd/root.go complexity.

**Interface**:
```go
type RoutingController interface {
    DetermineExecutionPath(parseResult *ParseResult) (ExecutionPath, error)
    ExecuteCommand(path ExecutionPath, config *Config) error
    HandleSpecialCases(flags *CCEFlags) (bool, error)
}

type ExecutionPath struct {
    Strategy      ExecutionStrategy
    Environment   *types.Environment
    Arguments     []string
    Metadata      map[string]interface{}
}

type ExecutionStrategy int
const (
    ExecuteInternally ExecutionStrategy = iota
    DelegateToClaudeCLI
    ShowHelp
    ShowVersion
    ExecuteSubcommand
)
```

**Key Methods**:
- `DetermineExecutionPath()`: Single decision point for execution strategy
- `ExecuteCommand()`: Orchestrate execution based on strategy
- `HandleSpecialCases()`: Process help, version, and other special flags

### 3. Enhanced Error System (New)

**File**: `internal/errors/command_errors.go`

**Purpose**: Provide consistent, structured error handling across all components.

**Interface**:
```go
type CommandError interface {
    error
    Code() ErrorCode
    Context() map[string]interface{}
    Suggestions() []string
    Wrap(error) CommandError
}

type ErrorCode int
const (
    ErrInvalidFlag ErrorCode = iota
    ErrEnvironmentNotFound
    ErrClaudeCLINotFound
    ErrDelegationFailed
    ErrInternalError
)

type StructuredError struct {
    code        ErrorCode
    message     string
    cause       error
    context     map[string]interface{}
    suggestions []string
}
```

**Error Handling Strategy**:
- All errors include actionable suggestions
- Consistent error wrapping preserves context
- Structured error codes enable programmatic handling
- Context includes debugging information without exposing sensitive data

### 4. Execution Strategy Factory (New)

**File**: `internal/execution/strategy_factory.go`

**Purpose**: Create execution strategies based on parsed arguments and configuration.

**Interface**:
```go
type StrategyFactory interface {
    CreateStrategy(path ExecutionPath) (ExecutionStrategy, error)
}

type ExecutionStrategy interface {
    Execute(ctx context.Context, params *ExecutionParams) error
    Validate(params *ExecutionParams) error
    EstimateResources() ResourceEstimate
}

type ExecutionParams struct {
    Environment   *types.Environment
    Arguments     []string
    ConfigManager types.ConfigManager
    UI           types.InteractiveUI
    Launcher     types.ClaudeCodeLauncher
}
```

**Strategy Implementations**:
- `InternalExecutionStrategy`: Handle CCE-specific commands
- `DelegationExecutionStrategy`: Forward to Claude CLI
- `HelpExecutionStrategy`: Display combined help
- `VersionExecutionStrategy`: Show version information

### 5. Refactored Root Command Handler

**File**: `cmd/root.go` (simplified)

**Current Issues**:
- 92-line Execute() function with complex control flow
- Tight coupling to parser components
- Mixed error handling patterns

**Design Changes**:

```go
// Simplified Execute function (<20 lines)
func Execute() {
    controller := createRoutingController()
    if err := controller.ProcessCommand(os.Args[1:]); err != nil {
        handleCommandError(err)
        os.Exit(1)
    }
}

// Extracted command processing logic
func (c *RoutingController) ProcessCommand(args []string) error {
    parseResult, err := c.flagParser.ParseFlags(args)
    if err != nil {
        return errors.WrapParsingError(err)
    }
    
    executionPath, err := c.DetermineExecutionPath(parseResult)
    if err != nil {
        return errors.WrapRoutingError(err)
    }
    
    return c.ExecuteCommand(executionPath, c.config)
}
```

**Key Improvements**:
- Single responsibility: cmd/root.go only handles CLI setup
- Delegation: Business logic moved to dedicated controllers
- Error handling: Consistent error wrapping and user feedback
- Testing: Each component can be tested in isolation

### 6. Shared Flag Operations (New)

**File**: `internal/parser/flag_operations.go`

**Purpose**: Eliminate code duplication between analyzer and preprocessor.

**Shared Operations**:
```go
type FlagOperations struct {
    registry *FlagRegistry
}

func (f *FlagOperations) ExtractFlagValue(args []string, index int) (string, int, error)
func (f *FlagOperations) ClassifyFlag(flag string) FlagType
func (f *FlagOperations) PreserveQuoting(arg string) string
func (f *FlagOperations) ValidateFlagSyntax(flag string) error
func (f *FlagOperations) NormalizeFlagName(flag string) string
```

**Benefits**:
- Single implementation of common flag operations
- Consistent behavior across all parsing components
- Easier testing and maintenance
- Reduced likelihood of bugs from duplicate implementations

## Data Models

### Enhanced Parsing Result

```go
type ParseResult struct {
    // Core parsing results
    CCEFlags       *CCEFlags
    RemainingArgs  []string
    Classification *FlagClassification
    
    // Quality and debugging information
    Metadata       *ParseMetadata
    Warnings       []ValidationWarning
    PerformanceLog *PerformanceMetrics
    
    // Execution guidance
    SuggestedPath  ExecutionPath
    RequiresAuth   bool
    RiskLevel     RiskLevel
}
```

### Structured Error Context

```go
type ErrorContext struct {
    Operation    string            // What was being attempted
    Input        []string          // Original arguments (sanitized)
    Environment  string            // Current environment context
    Suggestions  []string          // Actionable user guidance
    DebugInfo    map[string]interface{} // Technical details
    Timestamp    time.Time         // When error occurred
}
```

### Performance Metrics

```go
type PerformanceMetrics struct {
    ParsingTime    time.Duration
    ValidationTime time.Duration
    DecisionTime   time.Duration
    TotalTime      time.Duration
    MemoryUsage    int64
    CacheHits      int
    CacheMisses    int
}
```

## Architectural Patterns

### 1. Chain of Responsibility for Argument Processing

```go
type ArgumentProcessor interface {
    Process(args []string, context *ProcessingContext) (*ProcessingResult, error)
    CanHandle(args []string) bool
    SetNext(processor ArgumentProcessor)
}

// Chain: SpecialFlagsProcessor -> CCEFlagsProcessor -> ClaudeFlagsProcessor -> UnknownFlagsProcessor
```

### 2. Strategy Pattern for Execution

```go
type ExecutionStrategy interface {
    Execute(params *ExecutionParams) error
    Validate(params *ExecutionParams) error
    EstimateTime() time.Duration
}

// Strategies: InternalExecution, DelegatedExecution, HelpExecution, VersionExecution
```

### 3. Factory Pattern for Component Creation

```go
type ComponentFactory interface {
    CreateFlagParser(config *ParserConfig) FlagParser
    CreateRoutingController(dependencies *Dependencies) RoutingController
    CreateExecutionStrategy(path ExecutionPath) ExecutionStrategy
    CreateErrorHandler(config *ErrorConfig) ErrorHandler
}
```

### 4. Observer Pattern for Performance Monitoring

```go
type PerformanceObserver interface {
    OnParsingStart(args []string)
    OnParsingComplete(result *ParseResult, duration time.Duration)
    OnExecutionStart(strategy ExecutionStrategy)
    OnExecutionComplete(result *ExecutionResult, duration time.Duration)
}
```

## Error Handling Strategy

### 1. Error Hierarchy

```
CommandError (interface)
├── ParsingError
│   ├── InvalidFlagError
│   ├── MissingValueError
│   └── MalformedArgumentError
├── RoutingError
│   ├── AmbiguousPathError
│   └── UnsupportedCombinationError
├── ExecutionError
│   ├── EnvironmentError
│   ├── DelegationError
│   └── LauncherError
└── SystemError
    ├── ConfigurationError
    └── ResourceError
```

### 2. Error Context Enrichment

```go
func enrichError(err error, context *ErrorContext) CommandError {
    structured := &StructuredError{
        code:        determineErrorCode(err),
        message:     err.Error(),
        cause:       err,
        context:     context.ToMap(),
        suggestions: generateSuggestions(err, context),
    }
    return structured
}
```

### 3. Recovery Strategies

```go
type ErrorRecovery interface {
    CanRecover(err CommandError) bool
    Recover(err CommandError, context *ExecutionContext) error
    SuggestAlternatives(err CommandError) []string
}
```

## Testing Strategy

### 1. Component-Level Unit Tests

**FlagParser Tests**:
```go
func TestFlagParser_ExtractCCEFlags(t *testing.T) {
    tests := []struct {
        name     string
        args     []string
        expected *CCEFlags
        remaining []string
        wantErr  bool
    }{
        // Comprehensive test cases for all scenarios
    }
}
```

**RoutingController Tests**:
```go
func TestRoutingController_DetermineExecutionPath(t *testing.T) {
    // Test all routing decisions with mocked dependencies
}
```

### 2. Integration Tests with Mock Dependencies

```go
func TestCompleteWorkflow(t *testing.T) {
    // End-to-end test with mocked Claude CLI
    mockLauncher := &MockClaudeCodeLauncher{}
    controller := NewRoutingController(mockLauncher, ...)
    
    err := controller.ProcessCommand([]string{"-r", "test"})
    assert.NoError(t, err)
    assert.True(t, mockLauncher.WasCalled())
}
```

### 3. Performance Benchmarks

```go
func BenchmarkFlagParsing(b *testing.B) {
    parser := NewFlagParser()
    args := []string{"--env", "prod", "-r", "instruction"}
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := parser.ParseFlags(args)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### 4. Security Tests

```go
func TestArgumentSanitization(t *testing.T) {
    maliciousArgs := []string{
        "--env", "prod; rm -rf /",
        "-r", "$(malicious command)",
        "--config", "../../../etc/passwd",
    }
    // Verify proper sanitization and injection prevention
}
```

## Implementation Plan

### Phase 1: Foundation Refactoring (Priority: Critical)

1. **Create Unified Flag Parser**
   - Extract common logic from analyzer.go and preprocessor.go
   - Implement shared flag operations
   - Create comprehensive unit tests
   - **Success Criteria**: Zero code duplication in flag parsing

2. **Implement Structured Error System**
   - Create error hierarchy with consistent patterns
   - Add context enrichment and suggestion generation
   - Replace boolean returns with proper error types
   - **Success Criteria**: All functions return structured errors

3. **Refactor Root Command Handler**
   - Extract business logic to RoutingController
   - Simplify Execute() function to <20 lines
   - Implement proper dependency injection
   - **Success Criteria**: cmd/root.go complexity score <5

### Phase 2: Architecture Enhancement (Priority: High)

1. **Create Routing Controller**
   - Implement execution path determination
   - Add strategy factory for different execution types
   - Create proper abstraction layers
   - **Success Criteria**: Loose coupling between components

2. **Implement Execution Strategies**
   - Create strategy implementations for each execution type
   - Add proper validation and resource estimation
   - Implement performance monitoring
   - **Success Criteria**: All execution paths properly abstracted

3. **Add Comprehensive Testing**
   - Create unit tests for all new components
   - Add integration tests with proper mocking
   - Implement performance benchmarks
   - **Success Criteria**: >95% test coverage

### Phase 3: Quality and Performance (Priority: Medium)

1. **Performance Optimization**
   - Add caching for repeated operations
   - Implement lazy loading where appropriate
   - Add performance monitoring and metrics
   - **Success Criteria**: <10ms total processing time

2. **Security Hardening**
   - Implement proper input validation
   - Add argument sanitization
   - Ensure sensitive data masking
   - **Success Criteria**: Zero security vulnerabilities

3. **Documentation and Examples**
   - Create comprehensive API documentation
   - Add troubleshooting guides
   - Provide usage examples
   - **Success Criteria**: All components fully documented

## Quality Assurance

### Code Quality Metrics

- **Cyclomatic Complexity**: Max 10 per function
- **Function Length**: Max 50 lines per function
- **Code Duplication**: Zero blocks >6 lines duplicated
- **Test Coverage**: Minimum 95% line and branch coverage
- **Documentation**: 100% of public APIs documented

### Performance Targets

- **Flag Parsing**: <2ms for typical arguments
- **Execution Decision**: <1ms for routing determination
- **Total Overhead**: <10ms end-to-end processing
- **Memory Usage**: <5MB additional memory footprint

### Security Requirements

- **Input Validation**: 100% of user inputs validated
- **Data Sanitization**: All arguments properly escaped
- **Sensitive Data**: Zero exposure in logs or errors
- **Process Isolation**: Proper sandboxing maintained

## Migration Strategy

### Backward Compatibility

1. **Existing Commands**: All current CCE commands continue working
2. **Configuration**: No changes to config file format
3. **Environment Variables**: All existing env vars preserved
4. **Exit Codes**: Claude CLI exit codes properly forwarded

### Rollback Plan

1. **Feature Flags**: Enable/disable new parsing logic
2. **Fallback Mechanism**: Revert to original Cobra processing
3. **Monitoring**: Track success/failure rates
4. **Alerting**: Automatic notifications for issues

### Deployment Validation

1. **Smoke Tests**: Basic functionality validation
2. **Performance Tests**: Verify no regression
3. **Compatibility Tests**: All existing workflows work
4. **Security Tests**: No new vulnerabilities introduced

## Conclusion

This enhanced design eliminates the identified code quality issues while maintaining the core functionality improvements. Key achievements:

1. **Zero Code Duplication**: Unified flag parsing eliminates duplication
2. **Simplified Control Flow**: Routing controller reduces cmd/root.go complexity
3. **Consistent Error Handling**: Structured error system provides uniform patterns
4. **Comprehensive Testing**: Full test coverage for all components
5. **Performance Optimization**: Meets all timing requirements
6. **Security Hardening**: Proper input validation and data protection

The architecture follows SOLID principles, uses established design patterns, and provides clear separation of concerns while maintaining backward compatibility and achieving 95%+ code quality standards.