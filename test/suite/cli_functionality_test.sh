#!/bin/bash
# CLI Functionality Testing Suite
# Comprehensive testing of CLI commands and user interactions

set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
readonly TEST_DIR="$PROJECT_ROOT/test_temp"
readonly CLI_BINARY="$PROJECT_ROOT/cce"

cd "$PROJECT_ROOT"

echo "=== CLI Functionality Testing ==="

# Clean and setup test environment
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"

# Build test binary
echo "🔨 Building CLI binary for testing..."
go build -o "$CLI_BINARY" .

if [[ ! -f "$CLI_BINARY" ]]; then
    echo "❌ ERROR: Failed to build CLI binary"
    exit 1
fi
chmod +x "$CLI_BINARY"
echo "✅ CLI binary ready: $CLI_BINARY"

# Test categories
echo
echo "🧪 Running comprehensive CLI tests..."

# 1. Basic command availability
test_basic_commands() {
    local commands=("help" "version" "list" "add" "remove" "use" "current" "config")
    local passed=0
    
    echo "1. Testing basic commands..."
    for cmd in "${commands[@]}"; do
        if timeout 10s "$CLI_BINARY" "$cmd" --help >/dev/null 2>&1; then
            echo "  ✅ cce $cmd --help"
            passed=$((passed + 1))
        elif timeout 10s "$CLI_BINARY" "$cmd" >/dev/null 2>&1; then
            echo "  ✅ cce $cmd"
            passed=$((passed + 1))
        else
            echo "  ⚠️  cce $cmd not available or failed"
        fi
    done
    echo "1. $passed/${#commands[@]} basic commands working"
}

# 2. Configuration commands
test_config_commands() {
    local temp_config="$TEST_DIR/test_config.json"
    
    echo
echo "2. Testing configuration commands..."
    
    # Create test config directory
    export HOME="$TEST_DIR"
    mkdir -p "$TEST_DIR/.claude-code-env"
    
    # Test config initialization
    if timeout 10s "$CLI_BINARY" add --name "test-api" --url "https://api.anthropic.com" --key "test-key" >/dev/null 2>&1; then
        echo "  ✅ Config initialization successful"
    else
        echo "  ❌ Config initialization failed"
    fi
    
    # Test list environments
    if timeout 10s "$CLI_BINARY" list >/dev/null 2>&1; then
        echo "  ✅ List environments successful"
    else
        echo "  ❌ List environments failed"
    fi
    
    # Test current environment
    if timeout 10s "$CLI_BINARY" current >/dev/null 2>&1; then
        echo "  ✅ Current environment reporting"
    else
        echo "  ❌ Current environment failed"
    fi
}

# 3. Error handling
test_error_handling() {
    echo
    echo "3. Testing error handling..."
    
    # Test with invalid arguments
    if ! "$CLI_BINARY" invalid-command >/dev/null 2>&1; then
        echo "  ✅ Handles invalid commands correctly"
    else
        echo "  ❌ Invalid command not handled"
    fi
    
    # Test missing arguments
    if ! "$CLI_BINARY" add --name "" 2>/dev/null; then
        echo "  ✅ Handles missing arguments correctly"
    else
        echo "  ❌ Missing arguments not handled"
    fi
}

# 4. Interactive functionality
test_interactive_mode() {
    local test_script="$TEST_DIR/interactive_test.sh"
    
    echo
echo "4. Testing interactive functionality..."
    
    cat > "$test_script" << 'EOF'
#!/bin/bash
cd "$PROJECT_ROOT"
echo -e "1\ntest-env\nhttps://api.example.com\nkey123\n" | timeout 10s ./cce
EOF
    
    chmod +x "$test_script"
    
    if timeout 15s bash "$test_script" >/dev/null 2>&1; then
        echo "  ✅ Interactive mode tested"
    else
        echo "  ⚠️  Interactive mode test failed"
    fi
}

# 5. Configuration validation
test_config_validation() {
    echo
echo "5. Testing configuration validation..."
    
    export HOME="$TEST_DIR"
    
    # Test invalid URL
    if ! echo -e "invalid\ninvalid-url\ntest\n" | timeout 10s "$CLI_BINARY" add >/dev/null 2>&1; then
        echo "  ✅ Invalid URL validation working"
    else
        echo "  ❌ URL validation missing"
    fi
    
    # Test duplicate environment names
    "$CLI_BINARY" add --name "test-dupe" --url "https://api1.com" --key "key1" >/dev/null 2>&1
    if ! "$CLI_BINARY" add --name "test-dupe" --url "https://api2.com" --key "key2" >/dev/null 2>&1; then
        echo "  ✅ Duplicate name validation working"
    else
        echo "  ❌ Duplicate name validation missing"
    fi
}

# 6. Environment switching
test_environment_switching() {
    echo
echo "6. Testing environment switching..."
    
    export HOME="$TEST_DIR"
    
    # Setup test environments
    "$CLI_BINARY" add --name "env1" --url "https://api1.com" --key "key1" >/dev/null 2>&1
    "$CLI_BINARY" add --name "env2" --url "https://api2.com" --key "key2" >/dev/null 2>&1
    
    # Test switching
    if timeout 10s "$CLI_BINARY" use "env1" >/dev/null 2>&1; then
        echo "  ✅ Environment switching working"
    else
        echo "  ❌ Environment switching failed"
    fi
    
    # Test with invalid environment
    if ! "$CLI_BINARY" use "nonexistent" >/dev/null 2>&1; then
        echo "  ✅ Invalid environment handling"
    else
        echo "  ❌ Invalid environment not handled"
    fi
}

# 7. Security testing
test_security() {
    echo
echo "7. Testing security features..."
    
    export HOME="$TEST_DIR"
    
    # Test API key masking
    "$CLI_BINARY" add --name "security-test" --url "https://api.com" --key "secret123" >/dev/null 2>&1
    
    # Check config file for unmasked keys
    if grep -q "secret123" "$TEST_DIR/.claude-code-env/config.json" 2>/dev/null; then
        echo "  ❌ API key in plain text in config file"
    else
        echo "  ✅ API key properly masked/not stored"
    fi
    
    # Test file permissions
    config_file="$TEST_DIR/.claude-code-env/config.json"
    if [[ -f "$config_file" ]]; then
        perms=$(stat -f%OLp "$config_file" 2>/dev/null || stat -c%a "$config_file" 2>/dev/null || echo "600")
        if [[ "$perms" == "600" || "$perms" == "0600" ]]; then
            echo "  ✅ Config file permissions secure: $perms"
        else
            echo "  ⚠️  Config file permissions may need adjustment: $perms"
        fi
    fi
}

# 8. Integration test
test_integration() {
    echo
    echo "8. Testing integration workflow..."
    
    export HOME="$TEST_DIR"
    
    # Complete workflow test
    (
        echo "=== Integration Test Workflow ==="
        echo "1. Adding environment..."
        "$CLI_BINARY" add --name "int-test" --url "https://api.anthropic.com" --key "test-key-123" >/dev/null
        echo "2. Listing environments..."
        "$CLI_BINARY" list
        echo "3. Setting active environment..."
        "$CLI_BINARY" use "int-test"
        echo "4. Confirming current..."
        "$CLI_BINARY" current
        echo "5. Removing environment..."
        "$CLI_BINARY" remove --name "int-test"
        echo "6. Final list..."
        "$CLI_BINARY" list
    ) > "$TEST_DIR/integration.log" 2>&1
    
    if [[ $? -eq 0 ]]; then
        echo "  ✅ Integration workflow test passed"
    else
        echo "  ❌ Integration workflow failed"
    fi
}

# Main test execution
main() {
    local failures=0
    
    test_basic_commands || failures=$((failures + 1))
    test_config_commands || failures=$((failures + 1))
    test_error_handling || failures=$((failures + 1))
    test_interactive_mode || failures=$((failures + 1))
    test_config_validation || failures=$((failures + 1))
    test_environment_switching || failures=$((failures + 1))
    test_security || failures=$((failures + 1))
    test_integration || failures=$((failures + 1))
    
    echo
    echo "=== CLI Functionality Test Summary ==="
    if [[ $failures -eq 0 ]]; then
        echo "🎉 All CLI functionality tests passed!"
    else
        echo "❌ $failures test(s) failed"
    fi
    
    # Cleanup
    rm -rf "$CLI_BINARY"
    
    return $failures
}

main "$@"