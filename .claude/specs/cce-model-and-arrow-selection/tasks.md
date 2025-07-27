# Claude Code Environment Switcher (CCE) Enhancement Implementation Tasks

## Implementation Tasks

### 1. Add Model Field to Environment Structure

- [ ] **1.1** Modify Environment struct in main.go to include optional Model field
  - Add `Model string \`json:"model,omitempty"\`` field
  - Place field after APIKey to maintain existing order
  - Addresses requirements 1.1, 3.1, 3.3 from requirements.md

- [ ] **1.2** Create model validation function following existing validation pattern
  - Implement `validateModel(model string) error` function in main.go
  - Support empty/missing models (return nil for empty strings)
  - Validate against Anthropic model naming patterns using regex
  - Provide clear error messages with examples of valid formats
  - Addresses requirements 1.3, 7.1, 7.5 from requirements.md

- [ ] **1.3** Integrate model validation into existing validateEnvironment function
  - Add call to validateModel in validateEnvironment function
  - Ensure validation follows existing error handling patterns
  - Maintain backward compatibility for environments without model field
  - Addresses requirements 1.1, 4.4 from requirements.md

### 2. Enhance Environment Configuration Input

- [ ] **2.1** Extend promptForEnvironment function in ui.go to include model prompting
  - Add model input prompt after API key input
  - Use regularInput function for model entry (not secure input)
  - Support empty input for default model behavior
  - Include validation loop with retry mechanism for invalid model names
  - Show examples of valid model formats in prompts
  - Addresses requirements 1.2, 7.1 from requirements.md

- [ ] **2.2** Update displayEnvironments function to show model information
  - Add model display line in environment listing
  - Show "default" when model field is empty or missing
  - Maintain existing format and API key masking
  - Addresses requirements 1.6 from requirements.md

### 3. Implement ANTHROPIC_MODEL Environment Variable Support

- [ ] **3.1** Modify prepareEnvironment function in launcher.go to set ANTHROPIC_MODEL
  - Add conditional ANTHROPIC_MODEL environment variable when model field is non-empty
  - Follow existing pattern for ANTHROPIC_BASE_URL and ANTHROPIC_API_KEY
  - Ensure proper environment variable filtering and setup
  - Test with existing and new environment configurations
  - Addresses requirements 1.5 from requirements.md

### 4. Implement Arrow Key Navigation System

- [ ] **4.1** Create key input parsing system in ui.go
  - Implement parseKeyInput function to handle raw key input
  - Support arrow keys (\x1b[A, \x1b[B, \x1b[C, \x1b[D)
  - Handle Enter (\n, \r), Escape (\x1b), and Ctrl+C (\x03)
  - Include cross-platform key code compatibility
  - Addresses requirements 2.2, 2.3, 2.7, 6.1 from requirements.md

- [ ] **4.2** Create interactive environment menu display function
  - Implement displayEnvironmentMenu function with visual selection indicator
  - Show current selection with "► " prefix
  - Include model information in menu display ("default" if not set)
  - Add clear instructions for arrow navigation and controls
  - Handle terminal screen clearing and cursor positioning
  - Addresses requirements 2.1, 4.5 from requirements.md

- [ ] **4.3** Implement arrow key navigation core logic
  - Create selectEnvironmentWithArrows function as main navigation controller
  - Handle up/down arrow navigation with wraparound (first ↔ last)
  - Implement Enter key selection and Escape/Ctrl+C cancellation
  - Include proper terminal state management (raw mode + restoration)
  - Add graceful fallback to numbered selection on terminal issues
  - Addresses requirements 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 7.2, 7.3 from requirements.md

- [ ] **4.4** Integrate arrow key navigation into selectEnvironment function
  - Modify selectEnvironment to detect terminal capability
  - Use arrow key navigation when terminal supports it
  - Fall back to existing numbered selection when needed
  - Maintain single environment auto-selection behavior
  - Handle no environments case with existing error message
  - Addresses requirements 2.8, 2.9, 2.10 from requirements.md

### 5. Enhance Error Handling and Terminal Management

- [ ] **5.1** Implement comprehensive terminal state management
  - Add proper terminal state restoration in all exit conditions
  - Handle terminal interrupts (Ctrl+C) gracefully during navigation
  - Include error handling for terminal capability detection
  - Ensure fallback mechanisms work across platforms
  - Addresses requirements 2.7, 6.2, 7.4 from requirements.md

- [ ] **5.2** Create fallback mechanism for non-compatible terminals
  - Implement fallbackToNumberedSelection function
  - Detect terminal compatibility issues and switch modes
  - Provide user feedback when falling back to numbered selection
  - Ensure existing numbered selection behavior is preserved
  - Addresses requirements 7.2, 7.3 from requirements.md

### 6. Update Help and User Interface Text

- [ ] **6.1** Update help text and usage information in main.go
  - Modify showHelp function to mention model support
  - Update examples to show model field usage
  - Include arrow key navigation information in help
  - Maintain existing help format and structure
  - Addresses requirements 4.5 from requirements.md

- [ ] **6.2** Enhance user prompts and feedback messages
  - Update success messages to include model information when relevant
  - Improve error messages for model validation failures
  - Add instructional text for arrow key navigation
  - Ensure all messages follow existing format patterns
  - Addresses requirements 7.1, 7.5 from requirements.md

### 7. Testing and Validation

- [ ] **7.1** Create comprehensive unit tests for model functionality
  - Test model validation with valid and invalid model names
  - Verify Environment struct JSON serialization with model field
  - Test ANTHROPIC_MODEL environment variable setting
  - Validate backward compatibility with existing configurations
  - Addresses requirements 1.7, 3.1, 3.2, 3.3 from requirements.md

- [ ] **7.2** Implement integration tests for arrow key navigation
  - Test key input parsing for all supported key combinations
  - Verify terminal state management and restoration
  - Test fallback behavior for non-compatible terminals
  - Validate menu display and selection highlighting
  - Addresses requirements 2.1-2.10, 6.1, 6.2, 7.2, 7.3, 7.4 from requirements.md

- [ ] **7.3** Validate cross-platform compatibility
  - Test arrow key codes on different operating systems
  - Verify terminal control works on macOS, Linux, and Windows
  - Ensure environment variable handling is platform-independent
  - Test configuration file handling across platforms
  - Addresses requirements 6.1, 6.2, 6.3, 6.4 from requirements.md

- [ ] **7.4** Perform backward compatibility verification
  - Load and test existing configuration files without model fields
  - Verify mixed configurations (some environments with/without models)
  - Test all existing CLI commands and options continue working
  - Ensure no breaking changes to existing user workflows
  - Addresses requirements 3.1, 3.2, 3.3, 3.4 from requirements.md

### 8. Code Quality and KISS Principle Compliance

- [ ] **8.1** Verify codebase size remains within constraints
  - Measure total line count across all Go files
  - Ensure enhancement adds ≤100 lines total
  - Verify no new dependencies beyond golang.org/x/term
  - Check that all enhancements extend existing functions appropriately
  - Addresses requirements 4.1, 4.2, 4.3, 4.4, 4.5 from requirements.md

- [ ] **8.2** Validate security and data protection standards
  - Ensure model field follows same file permission patterns (0600)
  - Verify no sensitive information exposure in new functionality
  - Test terminal state security (proper cleanup on all exits)
  - Validate model name input against injection attacks
  - Addresses requirements 5.1, 5.2, 5.3, 5.4, 5.5 from requirements.md

## Implementation Priority and Dependencies

**High Priority (Core Functionality):**
- Tasks 1.1-1.3: Model field infrastructure (foundation for other tasks)
- Task 3.1: ANTHROPIC_MODEL environment variable (essential feature)
- Tasks 4.1-4.4: Arrow key navigation system (primary UX enhancement)

**Medium Priority (User Experience):**
- Tasks 2.1-2.2: Enhanced configuration input and display
- Tasks 5.1-5.2: Terminal management and fallback handling
- Tasks 6.1-6.2: Help text and user interface improvements

**Standard Priority (Quality Assurance):**
- Tasks 7.1-7.4: Testing and validation
- Tasks 8.1-8.2: Code quality and security verification

**Task Dependencies:**
- Task 1.2 must complete before 1.3 (validation function before integration)
- Task 1.3 must complete before 2.1 (validation available for prompting)
- Tasks 4.1-4.2 must complete before 4.3 (input parsing and display before navigation)
- Task 4.3 must complete before 4.4 (navigation logic before integration)
- All core tasks (1-6) should complete before comprehensive testing (7-8)

## Code Size Budget per File

**main.go:** +25 lines
- Model field addition to Environment struct (+1 line)
- validateModel function implementation (+15 lines)
- validateEnvironment enhancement (+2 lines)
- showHelp updates (+4 lines)
- displayEnvironments enhancement (+3 lines)

**ui.go:** +50 lines
- parseKeyInput function (+15 lines)
- displayEnvironmentMenu function (+10 lines)
- selectEnvironmentWithArrows function (+20 lines)
- promptForEnvironment model input enhancement (+5 lines)

**launcher.go:** +10 lines
- prepareEnvironment ANTHROPIC_MODEL enhancement (+5 lines)
- Additional error handling and validation (+5 lines)

**config.go:** +5 lines
- Additional validation calls and error handling (+5 lines)

**Total Estimated Addition:** +90 lines (within 100 line constraint)