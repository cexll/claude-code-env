# Implementation Tasks for Simplified Claude Code Environment Switcher

## Core Implementation Tasks

**CRITICAL**: This implementation must address security vulnerabilities and reliability issues to achieve production readiness (95% threshold). ALL error returns must be checked and handled properly.

### 1. Project Setup and Structure
- [ ] 1.1 Create new directory structure for simplified implementation
  - Create `simple-cce/` directory
  - Initialize Go module with `go mod init simple-cce`
  - Create basic file structure: `main.go`, `config.go`, `launcher.go`, `ui.go`
  - Create test file structure: `main_test.go`, `config_test.go`, `launcher_test.go`, `ui_test.go`
  - References: Requirements 4.1-4.6 (CLI interface structure), 7.1-7.10 (testing requirements)

- [ ] 1.2 Define core data structures in main.go
  - Implement `Environment` struct with name, URL, and API key fields
  - Implement `Config` struct with environments slice
  - Add JSON tags for serialization
  - Add validation methods for each struct
  - References: Requirements 1.1-1.2 (configuration storage format), 8.8 (required fields validation)

### 2. Configuration Management Implementation
- [ ] 2.1 Implement configuration file loading in config.go with comprehensive error handling
  - Create `loadConfig() (Config, error)` function to read JSON from `~/.claude-code-env/config.json`
  - **CRITICAL**: Handle all error returns from os.Stat(), ioutil.ReadFile(), json.Unmarshal()
  - Handle missing file by returning empty configuration (not an error)
  - Handle permission denied, file corrupted, invalid JSON with proper error wrapping
  - Return wrapped errors with context using fmt.Errorf()
  - References: Requirements 1.3, 1.6-1.8, 6.7-6.8 (CRUD operations, error handling)

- [ ] 2.2 Implement configuration file saving in config.go with atomic operations and error handling
  - Create `saveConfig(Config) error` function to write Config struct to JSON file
  - **CRITICAL**: Handle all error returns from os.MkdirAll(), ioutil.WriteFile(), os.Chmod()
  - Create configuration directory with 0700 permissions if needed, handle permission errors
  - Write file with 0600 permissions for security, validate permission setting
  - Use atomic write pattern (temp file + rename) with full error checking
  - Validate JSON marshaling errors
  - References: Requirements 1.5, 5.1-5.2, 6.7-6.8 (directory creation, file permissions, error handling)

- [ ] 2.3 Add comprehensive configuration validation in config.go
  - Create `validateEnvironment(Environment) error` function for complete validation
  - Create `validateURL(string) error` using `net/url.Parse()` with error handling
  - Create `validateAPIKey(string) error` for basic format and length validation
  - Create `validateName(string) error` for environment name validation (alphanumeric, length)
  - Check for required fields (name, URL, API key) and return specific errors
  - Sanitize inputs to prevent injection attacks
  - References: Requirements 1.4, 2.5, 8.1-8.8 (URL validation, field validation, input sanitization)

- [ ] 2.4 Implement configuration directory management with error handling
  - Create `ensureConfigDir() error` function to create `~/.claude-code-env/` with proper permissions
  - **CRITICAL**: Handle all error returns from os.MkdirAll(), os.Stat(), os.Chmod()
  - Handle existing directory gracefully, validate permissions
  - Return descriptive errors for permission failures
  - Use os.MkdirAll() with 0700 permissions and validate success
  - References: Requirements 1.5, 5.1, 6.7-6.8 (directory creation, permissions, error handling)

### 3. User Interface Implementation with Security and Error Handling
- [ ] 3.1 Implement environment selection UI in ui.go with input validation
  - Create `selectEnvironment([]Environment) (Environment, error)` function for interactive menu
  - Display numbered list of available environments with proper formatting
  - **CRITICAL**: Handle all error returns from fmt.Scanf(), validate user input
  - Implement input range validation and retry on invalid input
  - Return descriptive errors for invalid selections
  - Handle empty environment list gracefully
  - References: Requirements 2.1, 2.3, 6.7-6.8 (interactive selection, error handling)

- [ ] 3.2 Implement environment input UI in ui.go with validation
  - Create `promptForEnvironment() (Environment, error)` function to collect new environment details
  - Prompt for name, URL, and API key with individual validation
  - **CRITICAL**: Use secure input for API key (call secureInput function)
  - Validate each input immediately and provide specific error messages
  - Handle input errors and allow retry with clear feedback
  - Prevent duplicate environment names
  - References: Requirements 4.2, 6.3, 8.1-8.8 (add command, input validation, sanitization)

- [ ] 3.3 **CRITICAL**: Implement secure API key input in ui.go
  - Create `secureInput(prompt string) (string, error)` function for hidden API key entry
  - **SECURITY CRITICAL**: Implement platform-specific hidden input (termios on Unix, console API on Windows)
  - **MANDATORY**: Characters must NOT be visible in terminal
  - Handle backspace, enter, and control characters properly
  - **CRITICAL**: Handle all error returns from terminal operations
  - Clear sensitive data from memory where possible
  - Test thoroughly to ensure no character echoing
  - References: Requirements 5.3, 5.6-5.8 (API key masking, secure input, error handling)

- [ ] 3.4 Implement environment display in ui.go with error handling
  - Create `displayEnvironments([]Environment) error` function to format and show environment list
  - Display name and URL, mask API key with asterisks (show only first 4 and last 4 characters)
  - **CRITICAL**: Handle all error returns from fmt.Printf() and output operations
  - Use consistent formatting for all output
  - Handle empty environment list gracefully
  - Never display full API keys in any output
  - References: Requirements 4.1, 5.4, 6.7-6.8 (list command, API key protection, error handling)

### 4. Claude Code Launcher Implementation with Full Error Handling
- [ ] 4.1 Implement environment variable setup in launcher.go with validation
  - Create `prepareEnvironment(Environment) ([]string, error)` function to set ANTHROPIC_BASE_URL and ANTHROPIC_API_KEY
  - Inherit current environment and override Anthropic variables
  - **CRITICAL**: Validate environment variable values before setting
  - Return environment slice for process execution with error handling
  - Handle environment variable formatting errors
  - References: Requirements 3.1-3.4, 3.8 (environment variable setting, validation)

- [ ] 4.2 Implement Claude Code execution in launcher.go with comprehensive error handling
  - Create `launchClaudeCode(Environment, []string) error` function to execute claude-code command
  - **CRITICAL**: Handle all error returns from exec.Command(), cmd.Start(), cmd.Wait()
  - Use `os/exec.Command()` to run claude-code with arguments
  - Pass through all original command line arguments with validation
  - Set prepared environment variables with error checking
  - Capture and propagate exit codes properly
  - References: Requirements 3.3, 3.6-3.7 (argument passing, exit codes, error handling)

- [ ] 4.3 Implement Claude Code validation in launcher.go with error handling
  - Create `checkClaudeCodeExists() error` function to verify claude-code in PATH
  - **CRITICAL**: Handle error return from exec.LookPath()
  - Use `exec.LookPath()` to check if command exists
  - Return descriptive error with suggestions if not found
  - Provide actionable error messages for PATH issues
  - References: Requirements 3.5, 6.6 (clear error messages, recovery suggestions)

- [ ] 4.4 Handle process execution and exit codes in launcher.go with full error checking
  - Capture exit code from claude-code process with error handling
  - Exit with same code as claude-code process
  - **CRITICAL**: Handle all process execution errors (start failures, signal handling)
  - Distinguish between process errors and exit code propagation
  - Provide clear error messages for execution failures
  - References: Requirements 3.6, 6.1-6.2, 6.7-6.8 (exit codes, error handling)

### 5. Main CLI Implementation with Complete Error Handling
- [ ] 5.1 Implement command line parsing in main.go with validation
  - Use `flag` package for basic argument parsing with error handling
  - Support subcommands: list, add, remove, run with validation
  - Support --env/-e flag for direct environment selection with validation
  - Add --help/-h flag support with proper error handling
  - **CRITICAL**: Handle all error returns from flag parsing operations
  - Validate all command line arguments before processing
  - References: Requirements 4.1-4.6, 4.7-4.8 (CLI interface, error handling)

- [ ] 5.2 Implement list command handler in main.go with error handling
  - Create `runList() error` function to display all environments
  - **CRITICAL**: Handle all error returns from loadConfig() and displayEnvironments()
  - Load configuration with proper error handling and user feedback
  - Handle empty configuration gracefully with informative message
  - Return appropriate exit codes based on operation success
  - References: Requirements 4.1, 6.7-6.8 (list subcommand, error handling)

- [ ] 5.3 Implement add command handler in main.go with validation and error handling
  - Create `runAdd() error` function to add new environment
  - **CRITICAL**: Handle all error returns from UI functions, validation, and config operations
  - Use UI functions to collect environment details with full validation
  - Validate input comprehensively and save updated configuration
  - Prevent duplicate environment names with clear error messages
  - Implement proper error recovery and cleanup
  - References: Requirements 4.2, 6.7-6.8 (add subcommand, error handling)

- [ ] 5.4 Implement remove command handler in main.go with confirmation and error handling
  - Create `runRemove() error` function to delete environment
  - **CRITICAL**: Handle all error returns from config operations and user input
  - Confirm deletion with user interaction and input validation
  - Handle non-existent environment gracefully with informative error
  - Update configuration file with proper error handling
  - Provide recovery suggestions for common errors
  - References: Requirements 4.3, 6.7-6.8 (remove subcommand, error handling)

- [ ] 5.5 Implement default run behavior in main.go with comprehensive error handling
  - Create `runDefault() error` function for environment selection and Claude Code launching
  - **CRITICAL**: Handle all error returns from configuration, selection, and launching operations
  - Handle both interactive selection and --env flag with validation
  - Integrate configuration loading, selection, and launching with full error propagation
  - Display selected environment before launching with error handling
  - Implement proper cleanup and error recovery
  - References: Requirements 2.1-2.2, 2.4, 4.4, 6.7-6.8 (selection, launching, display, error handling)

### 6. **CRITICAL**: Error Handling and Validation Implementation
- [ ] 6.1 Implement consistent error formatting and exit codes
  - Create helper functions for different error types with proper wrapping
  - Use descriptive error messages with context and actionable suggestions
  - Implement proper exit codes (0=success, 1=general, 2=config, 3=launcher)
  - **MANDATORY**: Never ignore error returns from any function call
  - Use fmt.Errorf() for error wrapping with context
  - References: Requirements 6.1, 6.5, 6.7-6.10 (error messages, exit codes, error handling)

- [ ] 6.2 Add comprehensive input validation and sanitization
  - Validate URLs using `net/url.Parse()` with comprehensive error checking
  - Check environment name format and uniqueness with specific validation rules
  - Validate API key basic format (length, prefix) with clear error messages
  - Sanitize user input to prevent basic injection with proper escaping
  - **CRITICAL**: Handle all validation errors and provide recovery guidance
  - References: Requirements 1.4, 5.5, 6.3, 8.1-8.8 (validation, security, user feedback)

- [ ] 6.3 Implement graceful error recovery and user guidance
  - Handle file system errors (permissions, disk space) with specific suggestions
  - Provide helpful suggestions for common errors with actionable guidance
  - Continue operation when non-critical errors occur with proper fallbacks
  - Implement error context preservation through the call stack
  - **CRITICAL**: All error returns must be checked and handled appropriately
  - References: Requirements 6.2, 6.4, 6.6, 6.7-6.10 (graceful handling, suggestions, error handling)

### 7. **MANDATORY**: Comprehensive Testing Implementation (80%+ Coverage)
- [ ] 7.1 Create unit tests for configuration management with complete error scenario coverage
  - Test configuration loading with missing, empty, valid, and corrupted files
  - Test configuration saving with proper permissions and atomic operations
  - Test validation functions with various input types and malformed data
  - **CRITICAL**: Test ALL error paths and error handling scenarios
  - Use temporary directories for file system tests with proper cleanup
  - Test permission validation and error handling
  - References: Requirements 7.1-7.10 (testing requirements)

- [ ] 7.2 Create unit tests for UI functions with security validation
  - Test environment selection with various scenarios and invalid inputs
  - Test input validation and error handling with edge cases
  - **CRITICAL**: Test secure input functionality thoroughly (API key masking)
  - Mock user input for automated testing with comprehensive scenarios
  - Test display formatting functions and error handling
  - Validate that API keys are never displayed in plain text
  - References: Requirements 7.1-7.10, 5.3 (testing requirements, secure input)

- [ ] 7.3 Create unit tests for launcher functionality with process mocking
  - Test environment variable preparation with validation scenarios
  - Test claude-code existence checking with various PATH conditions
  - Mock process execution for testing with error simulation
  - Test error handling scenarios (command not found, execution failures)
  - **CRITICAL**: Test all error paths and recovery mechanisms
  - Validate exit code propagation and error handling
  - References: Requirements 7.1-7.10 (testing requirements)

- [ ] 7.4 Create integration tests with end-to-end validation
  - Test complete workflow from command line to execution
  - Test configuration file creation and modification with permission validation
  - Test environment variable setting and process launching (mocked)
  - **CRITICAL**: Test error recovery and cleanup in failure scenarios
  - Verify proper cleanup and error propagation
  - Test cross-platform compatibility (basic level)
  - References: Requirements 7.1-7.10 (testing requirements)

- [ ] 7.5 **CRITICAL**: Validate 80%+ test coverage and error handling completeness
  - Run `go test -cover` to verify minimum 80% coverage requirement
  - Ensure ALL error returns are tested in both success and failure scenarios
  - Test secure input functionality across different platforms
  - Validate that no error returns are ignored in the codebase
  - Create test coverage report and validate critical paths are covered
  - References: Requirements 7.1, 7.7-7.8 (coverage requirements, error testing)

### 8. **CRITICAL**: Security Implementation and Validation
- [ ] 8.1 Implement and test secure API key input across platforms
  - Implement Unix version using golang.org/x/term package
  - Implement Windows version using console API
  - **SECURITY CRITICAL**: Ensure NO characters are echoed to terminal
  - Test thoroughly on both platforms to prevent character leakage
  - Handle terminal mode restoration errors properly
  - Clear sensitive data from memory after use
  - References: Requirements 5.3, 5.6-5.8 (secure input, platform support)

- [ ] 8.2 Validate file permissions and security measures
  - Test configuration directory creation with 0700 permissions
  - Test configuration file creation with 0600 permissions
  - Validate that permission setting errors are handled properly
  - Test API key masking in all display scenarios
  - Ensure no API keys appear in error messages or logs
  - References: Requirements 5.1-5.2, 5.4-5.5 (file permissions, data protection)

### 9. Final Integration and Production Readiness Validation
- [ ] 9.1 Integrate all components in main.go with complete error handling
  - Connect CLI parsing to appropriate handlers with error propagation
  - Ensure proper error propagation and exit codes throughout the application
  - Verify all command line options work correctly with error scenarios
  - Test with real claude-code installation and mocked scenarios
  - **CRITICAL**: Verify NO error returns are ignored anywhere in the codebase
  - References: Requirements 6.7-6.8 (error handling completeness)

- [ ] 9.2 Validate line count constraint and simplicity
  - Count total lines across all source files (excluding tests)
  - Ensure implementation stays under 300 lines while maintaining functionality
  - Remove any unnecessary code or comments that don't add value
  - Optimize for simplicity and readability without sacrificing error handling
  - Ensure KISS principles are maintained despite security requirements

- [ ] 9.3 **CRITICAL**: Perform final validation for production readiness (95% threshold)
  - Test all subcommands with various scenarios and error conditions
  - Verify security measures (file permissions, API key masking, secure input)
  - Test error conditions and edge cases comprehensively
  - **MANDATORY**: Verify that ALL error returns are checked and handled
  - Run complete test suite and validate 80%+ coverage
  - Test cross-platform compatibility (Unix and Windows)
  - Validate that the implementation addresses all critical issues from the 69/100 score:
    - API key input is completely hidden ✓
    - All error returns are checked and handled ✓
    - Comprehensive unit tests with 80%+ coverage ✓
    - Proper input validation and sanitization ✓

### 10. Documentation and Deployment Preparation
- [ ] 10.1 Create minimal but complete documentation
  - Write brief README with installation and usage instructions
  - Document command line interface and examples
  - Include security considerations and requirements
  - Document testing procedures and coverage validation

- [ ] 10.2 Prepare for deployment and distribution
  - Create build script for cross-platform compilation
  - Test binary on different platforms (macOS, Linux, Windows)
  - Validate all dependencies are properly handled
  - Ensure the tool works independently without development environment