# Module Path Migration Design

## Overview

This document outlines the architectural design and implementation strategy for migrating the Go module path from `github.com/claude-code/env-switcher` to `github.com/cexll/claude-code-env`. The migration follows a conservative three-phase approach that prioritizes safety and minimal disruption.

## Architecture

### Phase 1: Pre-Migration Validation
```
┌─────────────────────────────────────────────────────────────┐
│                   Pre-Migration State                      │
├─────────────────────────────────────────────────────────────┤
│ Current Module Path: github.com/claude-code/env-switcher  │
│ Current Package Structure:                                 │
│   /cmd                                                  │
│   /internal/config                                      │
│   /internal/launcher                                    │
│   /internal/network                                    │
│   /internal/parser                                     │
│   /internal/ui                                        │
│   /pkg/types                                         │
│   /test/*                                             │
└─────────────────────────────────────────────────────────────┘
```

### Phase 2: Migration Execution
```
┌─────────────────────────────────────────────────────────────┐
│                  Migration Pipeline                       │
├─────────────────────────────────────────────────────────────┤
│ 1. Module Path Update                                   │
│    - go mod edit -module github.com/cexll/claude-code-env │
│                                                     │
│ 2. Import Refactoring                                   │
│    - sed/gofmt automated replacement                  │
│    - git pre-commit hooks validation                   │
│                                                     │
│ 3. Build Validation                                   │
│    - Cross-platform compilation                       │
│    - CI/CD pipeline update                          │
│                                                     │
│ 4. Tests Verification                               │
│    - Run all test suites                            │
│    - Integration testing                           │
└─────────────────────────────────────────────────────────────┘
```

### Phase 3: Post-Migration Validation
```
┌─────────────────────────────────────────────────────────────┐
│                  Post-Migration State                   │
├─────────────────────────────────────────────────────────────┤
│ New Module Path: github.com/cexll/claude-code-env       │
│ Preserved Interface:                                  │
│   - Binary Name: cce                                │
│   - Config Location: ~/.claude-code-env/              │
│   - API: Identical                                   │
│   - CL Interface: Unchanged                        │
└─────────────────────────────────────────────────────────────┘
```

## Components and Interfaces

### Module Management Component
```go
type ModuleManager interface {
    UpdateModulePath(old, new string) error
    ValidateModuleIntegrity() error
    CleanModCache() error
    Rollback() error
}

// Implementation handles:
// - go.mod file modification
// - go.sum cleanup and regeneration
// - Module cache management
// - Dependency validation
```

### Import Path Refactorer
```go
type ImportRefactorer interface {
    ScanAndReplace(root string) error
    GeneratePathMap() map[string]string
    ValidateRefactoring() error
    CreatePatch(diff bool) string
}

// Implementation handles:
// - Recursive file scanning
// - AST-based import path replacement
// - Cross-platform path handling
// - Unicode-aware string replacement
```

### Build System Adapter
```go
type BuildSystemAdapter interface {
    UpdateBuildConfiguration() error
    ValidateCrossCompilation() error
    UpdateCIConfig() error
    ArchiveArtifacts() []BuildArtifact
}

// Adapts:
// - Makefile targets
// - GoReleaser configuration (if applicable)
// - Container image references
// - Documentation generation
```

## Data Models

### Migration State Model
```go
type MigrationState struct {
    CurrentVersion    string    `json:"currentVersion"`
    TargetVersion     string    `json:"targetVersion"`
    MigrationDate     time.Time `json:"migrationDate"`
    IsSuccessful      bool      `json:"isSuccessful"`
    RollbackAvailable bool      `json:"rollbackAvailable"`
    ValidationChecks  []ValidationResult `json:"validationChecks"`
}

type ValidationResult struct {
    CheckType string    `json:"checkType"`
    Status    string    `json:"status"`
    Message   string    `json:"message"`
    Timestamp time.Time `json:"timestamp"`
}
```

### Import Path Mapping
```go
type ImportPathMapping struct {
    OldPath string `mapstructure:"old_path"`
    NewPath string `mapstructure:"new_path"`
    File    string `mapstructure:"file_path"`
    Action  string `mapstructure:"action"` // replace|add|remove
}
```

## Error Handling

### Structured Error Hierarchy
```go
var (
    ErrModuleUpdateFailed   = errors.New("module path update failed")
    ErrImportRefactorFailed = errors.New("import refactoring failed")
    ErrBuildValidationFail  = errors.New("build validation failed")
    ErrRollbackIncomplete   = errors.New("rollback incomplete")
)

type MigrationError struct {
    Phase   string // "MODULE", "IMPORT", "BUILD", "TEST", "DEPLOY"
    Error   error
    Context map[string]string
}

func (me *MigrationError) Error() string {
    return fmt.Sprintf("migration phase %s: %v", me.Phase, me.Error)
}
```

### Recovery Mechanisms

#### Automatic Recovery Pipeline
1. **Pre-validation**: Check current state before any changes
2. **Checkpointing**: Create checkpoints after each major change
3. **Atomic operations**: Use filesystem atomic operations for critical changes
4. **Rollback validation**: Verify rollback procedure works correctly

#### Manual Recovery Procedures
- **Module state rollback**: Use git reset/revert if repository state involved
- **Cache manipulation**: Clear Go module cache across platforms
- **Cross-platform rollback**: Platform-specific rollback scripts
- **Documentation recovery**: Restore documentation state for module path

## Testing Strategy

### Test Levels

#### Unit Tests
```bash
# Test individual component functions
go test ./internal/module/...
go test ./internal/refactor/...
go test ./internal/validation/...
```

#### Integration Tests
```bash
# Test complete migration workflow
go test ./test/migration/... -tags=integration

# Cross-platform simulation tests
GOOS=darwin go test ./test/platform/...
GOOS=linux go test ./test/platform/...
GOOS=windows go test ./test/platform/...
```

#### End-to-End Tests
```bash
# Test complete migration including build pipeline
./test/migration/e2e_test.sh

# Verify CLI functionality post-migration
./test/migration/cli_test.sh
```

### Test Scenarios

#### Scenario 1: Clean Migration
- **Description**: Migration on fresh clone
- **Pre-conditions**: Clean repo, no local changes
- **Actions**: Run complete migration path
- **Expected**: Clean migration with no issues
- **Validation**: All tests pass, all builds succeed

#### Scenario 2: Staged Changes Migration
- **Description**: Migration with local uncommitted changes
- **Pre-conditions**: Dirty working tree with pending changes
- **Actions**: Migration should preserve local changes
- **Expected**: Clean merge of migration changes
- **Validation**: `git status` shows clean state

#### Scenario 3: Rollback Validation
- **Description**: Verify rollback capability
- **Pre-conditions**: Migration committed but not pushed
- **Actions**: Execute rollback procedure
- **Expected**: Complete restoration to pre-migration state
- **Validation**: Module build succeeds with old path

#### Scenario 4: Cross-Repository Compatibility
- **Description**: Ensure dependent packages still work
- **Pre-conditions**: Test packages import this module
- **Actions**: Update import paths in dependent code
- **Expected**: Dependent code builds successfully
- **Validation**: Integration test suite passes

### Testing Infrastructure

#### Git Pre-commit Hooks
```yaml
# .git/hooks/pre-commit
#!/bin/sh
# Validate module path migration
if [ -n "$MIGRATION_IN_PROGRESS" ]; then
    go mod verify
    go mod tidy
    go test ./...
fi
```

#### CI/CD Verification
```yaml
# .github/workflows/migration-verify.yml
name: Module Migration Validation
on:
  push:
    branches: [feature/module-migration]

jobs:
  test-migration:
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        go-version: [1.19, 1.20, 1.21]
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: ${{ matrix.go-version }}
      - run: ./scripts/validate-migration.sh
```

## Configuration Management

### Migration Configuration File
```yaml
# .migration-config.yaml
module_path:
  old: "github.com/claude-code/env-switcher"
  new: "github.com/cexll/claude-code-env"
  version_bump: "patch"

build_configuration:
  update_makefile_references: false  # Makefile doesn't reference module path
  update_container_references: true  
  update_documentation: true

validation:
  run_integration_tests: true
  validate_cross_platform: true
  test_dependency_resolution: true

rollback:
  create_backup_branch: true
  tag_pre_migration: true
  preserve_git_history: true
```

### Environment Variables
```bash
# Migration configuration
MIGRATION_DRY_RUN=true
MIGRATION_AUTO_COMMIT=false
MIGRATION_CREATE_TAG=true
MIGRATION_BACKUP_BRANCH=feature/pre-migration-backup
```

This design ensures that the module path migration is executed safely, thoroughly validated, and can be rolled back if necessary. The separation of concerns across different components allows for isolated testing and verification at each stage.