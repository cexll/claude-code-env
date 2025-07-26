# Module Path Migration Implementation Plan

This document provides the detailed technical checklist for executing the module path migration from `github.com/claude-code/env-switcher` to `github.com/cexll/claude-code-env`.

## Phase 1: Pre-Migration Setup & Validation

### 1.1 Environment Preparation
- [ ] **1.1.1** Create migration branch: `git checkout -b feature/module-path-migration`
  - Ensure clean working directory before starting migration
  - Verify current branch is up-to-date with remote
- [ ] **1.1.2** Backup current state: `git tag pre-module-migration-rollback-point`
  - Create annotated tag with migration metadata
  - Document rollback commit hash: `git rev-parse HEAD`
- [ ] **1.1.3** Install Go migration tools: `go install golang.org/x/tools/cmd/goimports@latest`
  - Verify goimports version compatibility
  - Install additional tooling for cross-platform validation

### 1.2 Pre-Validation Checks
- [ ] **1.2.1** Run complete test suite: `go test -v ./...`
  - Ensure ALL tests pass before migration
  - Document any flaky test failures
- [ ] **1.2.2** Build cross-platform validation: `make build-all`
  - Verify all platform builds succeed
  - Check build artifacts sizes and integrity
- [ ] **1.2.3** Clean module cache: `go clean -modcache -i -c -x`
  - Document current dependencies for rollback verification
  - Cache cleaning ensures clean dependency resolution

## Phase 2: Module Path Update Execution

### 2.1 Module Declaration Update
- [ ] **2.1.1** Update go.mod module declaration:
  ```bash
  go mod edit -module github.com/cexll/claude-code-env
  ```
  - Verify change with `cat go.mod | head -n 1`
  - Confirm module name shows `module github.com/cexll/claude-code-env`

### 2.2 Import Path Refactoring
- [ ] **2.2.1** Bulk import path replacement across all Go files:
  ```bash
  # Find and replace all import references
  find . -type f -name "*.go" -exec sed -i '' 's|github.com/claude-code/env-switcher|github.com/cexll/claude-code-env|g' {} \;
  ```
- [ ] **2.2.2** Format and organize imports:
  ```bash
  gofmt -w -s ./internal/... ./cmd/... ./pkg/...
  goimports -w ./internal/... ./cmd/... ./pkg/... ./test/...
  ```
- [ ] **2.2.3** Validate import path consistency:
  ```bash
  grep -r "github.com/claude-code/env-switcher" . --include="*.go" || echo "No old imports found"
  ```

### 2.3 Dependency Synchronization
- [ ] **2.3.1** Update go.mod and go.sum:
  ```bash
  go mod tidy -compat=1.19
  go mod verify
  ```
- [ ] **2.3.2** Clean and update dependencies:
  ```bash
  go get -u ./...
  go mod tidy
  ```

### 2.4 Configuration Validation
- [ ] **2.4.1** Update any configuration files referencing old module path:
  - Check `.golangci.yml` for module-specific configurations
  - Search for module path in documentation files
- [ ] **2.4.2** Validate configuration schema continuity:
  - Ensure configuration parsing still works correctly
  - Verify no schema changes from dependency updates

## Phase 3: Cross-Platform Build System Continuity

### 3.1 Build Configuration Validation
- [ ] **3.1.1** Validate Makefile compatibility (no module path references needed)
  ```bash
  make clean && make build
  ```
- [ ] **3.1.2** Test cross-platform compilation:
  ```bash
  make build-all
  tree dist/
  ```
- [ ] **3.1.3** Verify build artifacts:
  - Check that binary names remain:
    - `cce` (macOS/Linux)
    - `cce.exe` (Windows)
  - Verify executable permissions and sizes

### 3.2 Build Flags Validation
- [ ] **3.2.1** Test custom build flags compatibility:
  ```bash
  go build -ldflags "-X main.version=1.1.0-migration-test" -o cce-test .
  ./cce-test version
  ```
- [ ] **3.2.2** Validate embedding configurations still work:
  - Check that any embed.FS references work correctly
  - Verify configuration file loading mechanisms

## Phase 4: Comprehensive Testing & Validation

### 4.1 Regression Testing
- [ ] **4.1.1** Run complete unit test suite:
  ```bash
  go test -v ./internal/... -timeout=10m
  ```
- [ ] **4.1.2** Integration and E2E testing:
  ```bash
  go test ./test/integration/...
  go test ./test/e2e/...
  ```
- [ ] **4.1.3** Security test validation:
  ```bash
  make security
  ```

### 4.2 Backward Compatibility Testing
- [ ] **4.2.1** Test configuration file persistence:
  ```bash
  go run . --config ~/.claude-code-env/config.json version
  ```
- [ ] **4.2.2** Validate environment variable handling unchanged:
  ```bash
  ANTHROPIC_API_KEY=test ANTHROPIC_BASE_URL=https://api.example.com ./cce version
  ```
- [ ] **4.2.3** Test interactive mode functionality:
  ```bash
  expect test/interactive_test.exp
  ```

### 4.3 Cross-Platform Validation Matrix
- [ ] **4.3.1** macOS validation:
  - Intel: `GOOS=darwin GOARCH=amd64 go build -o cce-darwin-amd64`
  - Apple Silicon: `GOOS=darwin GOARCH=arm64 go build -o cce-darwin-arm64`
- [ ] **4.3.2** Linux validation:
  - x64: `GOOS=linux GOARCH=amd64 go build`
  - ARM64: `GOOS=linux GOARCH=arm64 go build`
- [ ] **4.3.3** Windows validation:
  - Win64: `GOOS=windows GOARCH=amd64 go build -o cce-windows-amd64.exe`

## Phase 5: Documentation & Release Preparation

### 5.1 Documentation Updates
- [ ] **5.1.1** Update README.md module references:
  - Change `import "github.com/claude-code/env-switcher"` examples
  - Update installation instructions with new module path
- [ ] **5.1.2** Update godoc package documentation:
  - Run `go doc github.com/cexll/claude-code-env/cmd`
  - Verify documentation shows correct package paths
- [ ] **5.1.3** Update contributing guidelines regarding module path

### 5.2 Release Tagging
- [ ] **5.2.1** Create migration-complete tag:
  ```bash
  git tag v1.1.0-module-migration
  git push origin v1.1.0-module-migration
  ```
- [ ] **5.2.2** Update go.mod version comment if needed:
  ```bash
  go mod edit -require=github.com/cexll/claude-code-env@latest
  ```

## Phase 6: Advanced Validation & Final Verification

### 6.1 Dependency Graph Validation
- [ ] **6.1.1** Validate dependency resolution:
  ```bash
  go list -m -u all > deps.txt
  go mod graph > deps-graph.txt
  ```
- [ ] **6.1.2** Compare before/after dependency trees for verification
- [ ] **6.1.3** Check for any circular dependency issues introduced during migration

### 6.2 Security Validation
- [ ] **6.2.1** Run security scanning tools:
  ```bash
  gosec ./...
  govulncheck ./...
  ```
- [ ] **6.2.2** Verify checksum database validation:
  ```bash
  go mod verify
  ```

### 6.3 Environment Regeneration Testing
- [ ] **6.3.1** Test module in fresh environment:
  ```bash
  cd /tmp
  mkdir test-module-migration
  cd test-module-migration
  go mod init test-module-test
  go get github.com/cexll/claude-code-env@latest
  ```
- [ ] **6.3.2** Verify CLI functionality in new environment

## Phase 7: Rollback & Recovery Validation

### 7.1 Rollback Testing
- [ ] **7.1.1** Test rollback procedure using saved tag:
  ```bash
  git reset --hard pre-module-migration-rollback-point
  go mod edit -module github.com/claude-code/env-switcher
  go mod tidy
  ```
- [ ] **7.1.2** Validate rollback state:
  - Confirm module path reverted
  - All imports restored to original state
  - Tests continue to pass post-rollback

### 7.2 Recovery Procedure Documentation
- [ ] **7.2.1** Document recovery steps in MIGRATION.md file
- [ ] **7.2.2** Create recovery script for emergency rollback
- [ ] **7.2.3** Validate recovery procedure works across Git workflows

## Phase 8: Final Integration & Release

### 8.1 Git Integration
- [ ] **8.1.1** Create comprehensive commit with detailed message:
  ```bash
  git add -u
  git commit -m "feat: migrate module path to github.com/cexll/claude-code-env

  - Updated go.mod module declaration
  - Refactored all import paths consistently  
  - Updated all documentation references
  - Validated cross-platform compatibility
  - Preserved backward compatibility for configs
  - Added module migration rollback capability

  BREAKING CHANGE: Module path changed from github.com/claude-code/env-switcher to github.com/cexll/claude-code-env"
  ```
- [ ] **8.1.2** Create PR with migration documentation

### 8.2 Release Pipeline
- [ ] **8.2.1** Push migration changes and create release:
  ```bash
  git push origin feature/module-path-migration
  # Create PR and get approval
  ```
- [ ] **8.2.2** Verify GitHub Actions/workflows use new module path
- [ ] **8.2.3** Create GitHub release with migration notes

## Validation Checklist Completion Sign-off

### ✅ Pre-completion Verification
Verify ALL checkboxes above are completed before marking successful:
- [ ] All automated tests pass (`go test ./...`)
- [ ] Cross-platform builds work (`make build-all`)
- [ ] Configuration compatibility preserved
- [ ] Rollback procedure tested and documented
- [ ] Documentation updated with new module path
- [ ] CI/CD pipeline updated and validated

### 🔍 Quality Gates
- [ ] No compilation warnings or errors
- [ ] All security scans pass
- [ ] Binary sizes are within 1% of pre-migration
- [ ] Performance benchmarks show no regression
- [ ] User-facing commands and interfaces unchanged

### 📋 Release Checklist
- [ ] Migration branch reviewed and approved
- [ ] Rollback tag created: `pre-module-migration-rollback-point`
- [ ] Release tag created: `v1.1.0-module-migration`
- [ ] Documentation distributed to team members
- [ ] Migration completed successfully!

## Post-Migration Support

### For Repository Consumers
After migration, existing consumers should update their go.mod:
```
go get github.com/cexll/claude-code-env@latest
# Or specific version:
go get github.com/cexll/claude-code-env@v1.1.0
```

### For Development Team
- Module path is now consistently `github.com/cexll/claude-code-env`
- All imports should use the new path
- The CLI command remains `cce` with identical interface
- Configuration files are fully backward compatible
- The rollback tag `pre-module-migration-rollback-point` is available for emergency rollback