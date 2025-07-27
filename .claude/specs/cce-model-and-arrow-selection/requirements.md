# Claude Code Environment Switcher (CCE) Enhancement Requirements

## Introduction

This document outlines the requirements for enhancing the Claude Code Environment Switcher (CCE) with two key features: ANTHROPIC_MODEL environment variable support and interactive arrow key navigation for environment selection. These enhancements will provide users with more precise model control and an improved user experience while maintaining the tool's KISS (Keep It Simple, Stupid) principle and ~300 line codebase limit.

## Requirements

### 1. ANTHROPIC_MODEL Environment Variable Support

**User Story:** As a Claude Code user, I want to specify different AI models per environment configuration so that I can use different model capabilities (Claude 3.5 Sonnet, Claude 3 Haiku, etc.) for different use cases without manually setting environment variables.

**Acceptance Criteria:**
1. **WHEN** the system loads an environment configuration, **THEN** it **SHALL** support an optional "model" field in the Environment struct
2. **WHEN** a user adds a new environment, **THEN** the system **SHALL** prompt for an optional model specification after API key input
3. **WHEN** the model field is provided, **THEN** the system **SHALL** validate that it follows Anthropic model naming conventions (e.g., "claude-3-5-sonnet-20241022", "claude-3-haiku-20240307")
4. **WHEN** the model field is empty or not provided, **THEN** the system **SHALL** NOT set the ANTHROPIC_MODEL environment variable, allowing Claude Code to use its default model selection
5. **WHEN** launching Claude Code with an environment that has a model specified, **THEN** the system **SHALL** set the ANTHROPIC_MODEL environment variable with the configured model value
6. **WHEN** displaying environment information (list command), **THEN** the system **SHALL** show the model field if present, or indicate "default" if not specified
7. **WHEN** loading existing configuration files without a model field, **THEN** the system **SHALL** maintain backward compatibility and treat missing model fields as optional

### 2. Interactive Arrow Key Navigation

**User Story:** As a CCE user, I want to navigate environment selections using arrow keys and Enter to confirm so that I can quickly select environments without typing numbers, providing a more intuitive and efficient selection experience.

**Acceptance Criteria:**
1. **WHEN** multiple environments are available for selection, **THEN** the system **SHALL** display a visual menu with the currently selected environment highlighted
2. **WHEN** the user presses the down arrow key, **THEN** the system **SHALL** move the selection highlight to the next environment in the list
3. **WHEN** the user presses the up arrow key, **THEN** the system **SHALL** move the selection highlight to the previous environment in the list
4. **WHEN** the user reaches the last environment and presses down arrow, **THEN** the system **SHALL** wrap to the first environment
5. **WHEN** the user is at the first environment and presses up arrow, **THEN** the system **SHALL** wrap to the last environment
6. **WHEN** the user presses Enter or Return, **THEN** the system **SHALL** select the currently highlighted environment and proceed with Claude Code launch
7. **WHEN** the user presses Escape or Ctrl+C, **THEN** the system **SHALL** cancel the selection and exit gracefully
8. **WHEN** only one environment is configured, **THEN** the system **SHALL** automatically select it without showing the interactive menu
9. **WHEN** no environments are configured, **THEN** the system **SHALL** display an appropriate error message directing users to the 'add' command
10. **WHEN** the arrow key navigation is active, **THEN** the system **SHALL** use the existing golang.org/x/term library for terminal control without adding new dependencies

### 3. Configuration Backward Compatibility

**User Story:** As an existing CCE user, I want my current environment configurations to continue working without modification so that I can upgrade to the enhanced version seamlessly.

**Acceptance Criteria:**
1. **WHEN** the system loads an existing configuration file without model fields, **THEN** it **SHALL** successfully parse and load all environments
2. **WHEN** saving a configuration with some environments having model fields and others not, **THEN** the system **SHALL** correctly serialize both types to JSON
3. **WHEN** validating environments during load or save operations, **THEN** the system **SHALL** treat the model field as optional and not require it for validation
4. **WHEN** displaying environments that don't have a model field, **THEN** the system **SHALL** show "default" or equivalent indication for the model

### 4. KISS Principle Compliance

**User Story:** As a developer maintaining CCE, I want the enhancements to maintain the tool's simplicity so that the codebase remains around 300 lines, easy to understand, and free from over-engineering.

**Acceptance Criteria:**
1. **WHEN** implementing the model field, **THEN** the system **SHALL** add it directly to the existing Environment struct without creating new interfaces or abstractions
2. **WHEN** implementing arrow key navigation, **THEN** the system **SHALL** use the existing golang.org/x/term dependency and standard library functions
3. **WHEN** measuring the total codebase size, **THEN** it **SHALL** remain under 400 lines across all Go files (allowing ~100 line increase for both features)
4. **WHEN** adding new validation logic, **THEN** it **SHALL** follow the existing validation pattern in the validateEnvironment function
5. **WHEN** enhancing the UI module, **THEN** it **SHALL** extend existing functions rather than creating parallel systems

### 5. Security and Data Protection

**User Story:** As a security-conscious user, I want the model field and new navigation features to maintain the same security standards as existing functionality so that no sensitive information is exposed or logged.

**Acceptance Criteria:**
1. **WHEN** storing the model field in configuration files, **THEN** the system **SHALL** use the same file permissions (0600) and atomic write operations as existing configuration management
2. **WHEN** displaying environment information including model data, **THEN** the system **SHALL** NOT expose any sensitive information beyond what is currently shown
3. **WHEN** handling terminal input for arrow key navigation, **THEN** the system **SHALL** properly manage terminal state restoration on all exit conditions (normal, interrupt, error)
4. **WHEN** validating model names, **THEN** the system **SHALL** prevent injection attacks by using strict format validation
5. **WHEN** setting the ANTHROPIC_MODEL environment variable, **THEN** the system **SHALL** follow the same secure environment preparation pattern as ANTHROPIC_API_KEY and ANTHROPIC_BASE_URL

### 6. Cross-Platform Compatibility

**User Story:** As a user on different operating systems, I want the arrow key navigation and model features to work consistently across macOS, Linux, and Windows so that I have the same experience regardless of platform.

**Acceptance Criteria:**
1. **WHEN** using arrow key navigation on any supported platform, **THEN** the system **SHALL** correctly interpret platform-specific key codes for arrow keys
2. **WHEN** managing terminal state for interactive navigation, **THEN** the system **SHALL** use platform-appropriate terminal control methods via golang.org/x/term
3. **WHEN** storing model field data in configuration files, **THEN** the system **SHALL** use platform-independent JSON serialization
4. **WHEN** setting the ANTHROPIC_MODEL environment variable, **THEN** it **SHALL** work correctly with platform-specific environment variable handling

### 7. Error Handling and User Experience

**User Story:** As a CCE user, I want clear error messages and graceful handling of edge cases for both model configuration and arrow key navigation so that I can quickly understand and resolve any issues.

**Acceptance Criteria:**
1. **WHEN** a user enters an invalid model name, **THEN** the system **SHALL** display a clear error message with examples of valid model names and allow retry
2. **WHEN** arrow key navigation encounters terminal compatibility issues, **THEN** the system **SHALL** fall back gracefully to the existing numbered selection menu
3. **WHEN** the terminal window is too small to display the environment list, **THEN** the system **SHALL** handle the display gracefully without crashing
4. **WHEN** the user interrupts arrow key navigation (Ctrl+C), **THEN** the system **SHALL** restore terminal state and exit with appropriate status code
5. **WHEN** configuration files contain invalid model field values, **THEN** the system **SHALL** provide specific error messages indicating which environment has the invalid model and what the valid format should be