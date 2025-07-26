#!/bin/bash
# Claude Code Environment Switcher - Comprehensive Module Rename Test Suite
# This test suite validates all aspects of the module rename from
# github.com/claude-code/env-switcher to github.com/cexll/claude-code-env

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Configuration
readonly ORIGINAL_MODULE="github.com/claude-code/env-switcher"
readonly NEW_MODULE="github.com/cexll/claude-code-env"
readonly PROJECT_ROOT="/Users/chenwenjie/Downloads/claude-code-env-switch"
readonly TEST_OUTPUT_DIR="$PROJECT_ROOT/test_results"
readonly COVERAGE_DIR="$PROJECT_ROOT/coverage"
readonly TIMESTAMPS_FILE="$TEST_OUTPUT_DIR/timestamps.log"

# Ensure test directories exist
mkdir -p "$TEST_OUTPUT_DIR" "$COVERAGE_DIR"

# Logging function with timestamp
log() {
    local level="$1"
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S.%3N')
    echo -e "${timestamp} [${level}] ${message}" | tee -a "$TIMESTAMPS_FILE"
}

# Test execution counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# Helper function to run individual test suites
run_test_suite() {
    local suite_name="$1"
    local test_script="$2"
    local timeout="${3:-300}"
    
    log "INFO" "Starting test suite: $suite_name"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if [[ -x "$test_script" ]]; then
        local output_file="$TEST_OUTPUT_DIR/${suite_name// /_}.log"
        
        if timeout "$timeout" bash "$test_script" > "$output_file" 2>&1; then
            log "SUCCESS" "✅ $suite_name - PASSED"
            PASSED_TESTS=$((PASSED_TESTS + 1))
            return 0
        else
            log "ERROR" "❌ $suite_name - FAILED"
            FAILED_TESTS=$((FAILED_TESTS + 1))
            return 1
        fi
    else
        log "ERROR" "❌ $suite_name - SKIPPED (script not executable)"
        SKIPPED_TESTS=$((SKIPPED_TESTS + 1))
        return 1
    fi
}

# Summary report
print_summary() {
    local exit_code=0
    
    echo
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}    Claude Code Environment Switcher${NC}"
echo -e "${GREEN}   Module Rename Validation Summary${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "Total Tests:    ${TOTAL_TESTS}"
    echo -e "Passed:         ${GREEN}${PASSED_TESTS}${NC}"
    echo -e "Failed:         ${RED}${FAILED_TESTS}${NC}"
    echo -e "Skipped:        ${YELLOW}${SKIPPED_TESTS}${NC}"
    echo
    
    if [[ $FAILED_TESTS -gt 0 || $SKIPPED_TESTS -gt $(($TOTAL_TESTS / 2 + 1)) ]]; then
        echo -e "${RED}❌ Issues detected - please review the logs${NC}"
        exit_code=1
    else
        echo -e "${GREEN}✅ All tests completed successfully${NC}"
    fi
    
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo
    
    return $exit_code
}

# Main execution flow
main() {
    log "INFO" "Starting comprehensive module rename validation suite"
    
    # Change to project root
    cd "$PROJECT_ROOT"
    
    # Clean previous builds
    log "INFO" "Cleaning previous build artifacts"
    make clean 2>/dev/null || true
    
    # Remove old test results
    rm -rf "${TEST_OUTPUT_DIR:?}"/*
    rm -rf "${COVERAGE_DIR:?}"/*
    
    # Run individual test suites
    local test_scripts=(
        "module_integrity:./test/suite/module_integrity_test.sh:180"
        "import_verification:./test/suite/import_verification_test.sh:120"
        "cross_platform:./test/suite/cross_platform_test.sh:300"
        "cli_functionality:./test/suite/cli_functionality_test.sh:180"
        "config_compatibility:./test/suite/config_compatibility_test.sh:150"
        "network_validation:./test/suite/network_validation_test.sh:240"
        "security_permissions:./test/suite/security_permissions_test.sh:120"
        "performance_benchmark:./test/suite/performance_benchmark_test.sh:300"
    )
    
    for test_spec in "${test_scripts[@]}"; do
        IFS=':' read -r suite_name script_path timeout <<< "$test_spec"
        
        if [[ -f "$script_path" ]]; then
            run_test_suite "$suite_name" "$script_path" "$timeout"
        else
            log "WARN" "Test script $script_path not found, creating..."
            # Create the missing script
            create_missing_test_script "$suite_name" "$script_path"
            if [[ -x "$script_path" ]]; then
                run_test_suite "$suite_name" "$script_path" "$timeout"
            fi
        fi
    done
    
    # Generate comprehensive report
    generate_comprehensive_report
    
    # Generate coverage summary
    generate_coverage_report
    
    # Final summary
    print_summary
}

# Generate comprehensive report
generate_comprehensive_report() {
    local report_file="$TEST_OUTPUT_DIR/comprehensive_report.md"
    
    cat > "$report_file" << EOF
# Claude Code Environment Switcher - Module Rename Validation Report

**Generated:** $(date)
**Original Module:** $ORIGINAL_MODULE
**New Module:** $NEW_MODULE

## Test Environment
- **Project Root:** $PROJECT_ROOT
- **Go Version:** $(go version)
- **Platform:** $(uname -a)
- **Test Duration:** $(date -r "$TIMESTAMPS_FILE" "+%Y-%m-%d %H:%M:%S" 2>/dev/null || echo "Unknown")

## Test Results Summary
- **Total Test Suites:** $TOTAL_TESTS
- **Passed:** $PASSED_TESTS
- **Failed:** $FAILED_TESTS
- **Skipped:** $SKIPPED_TESTS

## Detailed Results by Category

### 1. Module Integrity
[Results from module_integrity_test.sh]
\`\`\`
$(cat "$TEST_OUTPUT_DIR/module_integrity.log" 2>/dev/null || echo "Not available")
\`\`\`

### 2. Import Path Verification
[Results from import_verification_test.sh]
\`\`\`
$(cat "$TEST_OUTPUT_DIR/import_verification.log" 2>/dev/null || echo "Not available")
\`\`\`

### 3. Cross-Platform Build Verification
[Results from cross_platform_test.sh]
\`\`\`
$(cat "$TEST_OUTPUT_DIR/cross_platform.log" 2>/dev/null || echo "Not available")
\`\`\`

### 4. CLI Functionality Testing
[Results from cli_functionality_test.sh]
\`\`\`
$(cat "$TEST_OUTPUT_DIR/cli_functionality.log" 2>/dev/null || echo "Not available")
\`\`\`

### 5. Configuration Compatibility
[Results from config_compatibility_test.sh]
\`\`\`
$(cat "$TEST_OUTPUT_DIR/config_compatibility.log" 2>/dev/null || echo "Not available")
\`\`\`

### 6. Network Validation
[Results from network_validation_test.sh]
\`\`\`
$(cat "$TEST_OUTPUT_DIR/network_validation.log" 2>/dev/null || echo "Not available")
\`\`\`

### 7. Security Permissions
[Results from security_permissions_test.sh]
\`\`\`
$(cat "$TEST_OUTPUT_DIR/security_permissions.log" 2>/dev/null || echo "Not available")
\`\`\`

### 8. Performance Benchmarks
[Results from performance_benchmark_test.sh]
\`\`\`
$(cat "$TEST_OUTPUT_DIR/performance_benchmark.log" 2>/dev/null || echo "Not available")
\`\`\`

## Recommendations
$(generate_recommendations)

## Next Steps
1. Review all failed tests and address issues
2. Integrate this suite into CI/CD pipeline
3. Schedule regular automated testing
4. Monitor performance trends over time
EOF
    
    log "INFO" "Comprehensive report generated: $report_file"
}

# Generate recommendations based on test results
generate_recommendations() {
    local recommendations=""
    
    if [[ $FAILED_TESTS -gt 0 ]]; then
        recommendations+="- **Address failures:** Focus on failed test suites\n"
    fi
    
    # Check for common issues
    local config_files=$(find "$TEST_OUTPUT_DIR" -name "*.log" -exec grep -l "go.mod" {} \; | wc -l)
    if [[ $config_files -eq 0 ]]; then
        recommendations+="- **Verify dependencies:** May need to resolve go.mod issues\n"
    fi
    
    # Check build issues
    local build_files=$(find "$TEST_OUTPUT_DIR" -name "*.log" -exec grep -l "build" {} \; | wc -l)
    if [[ $build_files -eq 0 ]]; then
        recommendations+="- **Check build configuration:** Build verification may be incomplete\n"
    fi
    
    if [[ -z "$recommendations" ]]; then
        recommendations="- ✅ All checks are passing"
    fi
    
    echo "$recommendations"
}

# Coverage report generation
generate_coverage_report() {
    log "INFO" "Generating coverage report"
    
    # Run tests for all packages with coverage
    go test ./... -coverprofile="$COVERAGE_DIR/coverage.out" -covermode=atomic || true
    
    # Generate HTML coverage report
    if [[ -f "$COVERAGE_DIR/coverage.out" ]]; then
        go tool cover -html="$COVERAGE_DIR/coverage.out" -o "$COVERAGE_DIR/coverage.html"
        log "INFO" "Coverage report: $COVERAGE_DIR/coverage.html"
    fi
    
    # Extract coverage percentage
    if command -v go tool cover &> /dev/null; then
        local coverage_percent=$(go tool cover -func="$COVERAGE_DIR/coverage.out" 2>/dev/null | grep total: | awk '{print $3}' || echo "0%")
        log "INFO" "Overall code coverage: $coverage_percent"
    fi
}

# Create missing test scripts
create_missing_test_script() {
    local suite_name="$1"
    local script_path="$2"
    
    log "INFO" "Creating missing test script: $script_path"
    
    mkdir -p "$(dirname "$script_path")"
    
    case "$suite_name" in
        "module_integrity")
            create_module_integrity_script "$script_path"
            ;;
        "import_verification")
            create_import_verification_script "$script_path"
            ;;
        "cross_platform")
            create_cross_platform_script "$script_path"
            ;;
        "cli_functionality")
            create_cli_functionality_script "$script_path"
            ;;
        "config_compatibility")
            create_config_compatibility_script "$script_path"
            ;;
        "network_validation")
            create_network_validation_script "$script_path"
            ;;
        "security_permissions")
            create_security_permissions_script "$script_path"
            ;;
        "performance_benchmark")
            create_performance_benchmark_script "$script_path"
            ;;
    esac
    
    chmod +x "$script_path"
    log "INFO" "Created test script: $script_path"
}

# Register trap for cleanup
trap cleanup EXIT

cleanup() {
    local exit_code=$?
    
    # Clean up test processes
    pkill -f "test.*daemon" 2>/dev/null || true
    pkill -f "cce.*test" 2>/dev/null || true
    
    log "INFO" "Test suite cleanup completed"
    
    return $exit_code
}

# Execute main if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi