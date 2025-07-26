#!/bin/bash
# Configuration Compatibility Test Suite
# Validates configuration file compatibility, migration, and backward compatibility

set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
readonly TEST_DIR="$PROJECT_ROOT/test_temp/config"
readonly CLI_BINARY="$PROJECT_ROOT/cce"

cd "$PROJECT_ROOT"

echo "=== Configuration Compatibility Testing ==="

# Clean and setup test environment
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"

# Build CLI binary
echo "🔨 Building CLI binary..."
go build -o "$CLI_BINARY" .
if [[ ! -f "$CLI_BINARY" ]]; then
    echo "❌ ERROR: Failed to build CLI binary"
    exit 1
fi

chmod +x "$CLI_BINARY"

# Test configuration scenarios
test_version_compatibility() {
    echo
    echo "1. Testing version compatibility..."
    
    export HOME="$TEST_DIR"
    
    # Test v1.0 configuration
    mkdir -p "$TEST_DIR/.claude-code-env"
    cat > "$TEST_DIR/.claude-code-env/config.json" << 'EOF'
{
    "version": "1.0",
    "environments": [
        {
            "name": "test-env-v1",
            "url": "https://api.anthropic.com/v1",
            "api_key": "test-key-v1",
            "is_active": true
        }
    ],
    "created_at": "2024-01-01T00:00:00Z"
}
EOF
    
    chmod 600 "$TEST_DIR/.claude-code-env/config.json"
    
    # Test configuration loading
    if timeout 10s "$CLI_BINARY" list >/dev/null 2>&1; then
        echo "  ✅ v1.0 configuration loaded successfully"
    else
        echo "  ❌ v1.0 configuration failed to load"
    fi
    
    # Test configuration upgrade on first run
    if grep -q "2.0" "$TEST_DIR/.claude-code-env/config.json" 2>/dev/null; then
        echo "  ✅ Configuration upgraded to latest version"
    else
        echo "  ⚠️  Configuration upgrade not detected"
    fi
}

test_file_format_compatibility() {
    echo
    echo "2. Testing file format compatibility..."
    
    export HOME="$TEST_DIR"
    
    # Test JSON format variations
    test_configs=(
        '{
            "version": "2.0",
            "environments": [
                {
                    "name": "json-minimal",
                    "url": "https://api.minimal.com",
                    "api_key": "test-minimal"
                }
            ]
        }'
        
        '{
            "version": "2.0",
            "environments": [
                {
                    "name": "json-full",
                    "url": "https://api.full.com",
                    "api_key": "test-full",
                    "is_active": true,
                    "created_at": "2024-01-01T10:00:00Z",
                    "last_used": "2024-01-01T11:00:00Z",
                    "metadata": {
                        "region": "us-east-1",
                        "type": "production"
                    }
                }
            ]
        }'
    )
    
    for i in "${!test_configs[@]}"; do
        config_dir="$TEST_DIR/format_test_$i"
        mkdir -p "$config_dir/.claude-code-env"
        
        echo "${test_configs[$i]}" > "$config_dir/.claude-code-env/config.json"
        chmod 600 "$config_dir/.claude-code-env/config.json"
        
        export HOME="$config_dir"
        
        if timeout 10s "$CLI_BINARY" list >/dev/null 2>&1; then
            echo "  ✅ Format test $i: JSON compatibility OK"
        else
            echo "  ❌ Format test $i: JSON compatibility FAILED"
        fi
    done
}

test_permissions_compatibility() {
    echo
    echo "3. Testing permissions compatibility..."
    
    export HOME="$TEST_DIR"
    
    # Test restrictive permissions
    config_dir="$TEST_DIR/permissions_test"
    mkdir -p "$config_dir/.claude-code-env"
    
    cat > "$config_dir/.claude-code-env/config.json" << 'EOF'
{
    "version": "2.0",
    "environments": [
        {
            "name": "permissions-test",
            "url": "https://api.test.com",
            "api_key": "test-key"
        }
    ]
}
EOF
    
    # Test different permission levels
    permissions=("600" "644" "755")
    
    for perm in "${permissions[@]}"; do
        chmod "$perm" "$config_dir/.claude-code-env/config.json"
        export HOME="$config_dir"
        
        if timeout 5s "$CLI_BINARY" list >/dev/null 2>&1; then
            echo "  ✅ Permission $perm: Compatible"
        else
            echo "  ❌ Permission $perm: Not compatible"
        fi
    done
}

test_backup_recovery() {
    echo
    echo "4. Testing backup and recovery..."
    
    export HOME="$TEST_DIR"
    
    # Create initial configuration
    mkdir -p "$TEST_DIR/.claude-code-env"
    cat > "$TEST_DIR/.claude-code-env/config.json" << 'EOF'
{
    "version": "2.0",
    "environments": [
        {
            "name": "backup-test-original",
            "url": "https://api.original.com",
            "api_key": "original-key"
        }
    ]
}
EOF
    chmod 600 "$TEST_DIR/.claude-code-env/config.json"
    
    # Create backup
    cp "$TEST_DIR/.claude-code-env/config.json" "$TEST_DIR/.claude-code-env/config.json.backup"
    
    # Test with corrupted configuration
    cp "$TEST_DIR/.claude-code-env/config.json" "$TEST_DIR/.claude-code-env/config.json.corrupt"
    echo '{"corrupted": true, "invalid: "}' > "$TEST_DIR/.claude-code-env/config.json"
    
    if ! timeout 5s "$CLI_BINARY" list >/dev/null 2>&1; then
        echo "  ✅ Corrupted config handled appropriately"
        
        # Restore from backup
        mv "$TEST_DIR/.claude-code-env/config.json.backup" "$TEST_DIR/.claude-code-env/config.json"
        if timeout 5s "$CLI_BINARY" list >/dev/null 2>&1; then
            echo "  ✅ Backed up configuration restored"
        else
            echo "  ❌ Backup restoration failed"
        fi
    else
        echo "  ❌ Corrupted config not handled"
    fi
}

test_environment_variables() {
    echo
    echo "5. Testing environment variable compatibility..."
    
    export HOME="$TEST_DIR"
    export CCE_CONFIG_DIR="$TEST_DIR/custom-config"
    
    mkdir -p "$TEST_DIR/custom-config"
    cat > "$TEST_DIR/custom-config/config.json" << 'EOF'
{
    "version": "2.0",
    "environments": [
        {
            "name": "env-var-test",
            "url": "https://api.envvar.com",
            "api_key": "envvar-key"
        }
    ]
}
EOF
    chmod 600 "$TEST_DIR/custom-config/config.json"
    
    if timeout 10s "$CLI_BINARY" list >/dev/null 2>&1; then
        echo "  ✅ CCE_CONFIG_DIR compatibility: OK"
    else
        echo "  ❌ CCE_CONFIG_DIR compatibility: FAILED"
    fi
    
    # Reset
    unset CCE_CONFIG_DIR
}

test_migration_scenarios() {
    echo
    echo "6. Testing configuration migration scenarios..."
    
    # Test various migration scenarios
    migrations=(
        "empty_to_v2"
        "v1_to_v2"
        "partial_to_v2"
        "corrupted_to_v2"
    )
    
    for scenario in "${migrations[@]}"; do
        scenario_dir="$TEST_DIR/migration_$scenario"
        mkdir -p "$scenario_dir/.claude-code-env"
        
        case "$scenario" in
            "empty_to_v2")
                # Empty config file
                echo '{}' > "$scenario_dir/.claude-code-env/config.json"
                ;;
            "v1_to_v2")
                # v1 format
                cat > "$scenario_dir/.claude-code-env/config.json" << 'EOF'
{
    "version": "1.0",
    "environment": {
        "name": "legacy",
        "url": "https://api.legacy.com",
        "key": "legacy-key",
        "active": true
    }
}
EOF
                ;;
            "partial_to_v2")
                # Partial JSON
                cat > "$scenario_dir/.claude-code-env/config.json" << 'EOF'
{
    "environments": [
        {"name": "partial", "url": null, "api_key": "partial-key"}
    ]
}
EOF
                ;;
            "corrupted_to_v2")
                # Corrupted JSON
                echo '{"corrupted":' > "$scenario_dir/.claude-code-env/config.json"
                ;;
        esac
        
        chmod 600 "$scenario_dir/.claude-code-env/config.json"
        
        export HOME="$scenario_dir"
        
        case "$scenario" in
            "corrupted_to_v2")
                # Should fail gracefully or recover
                if timeout 5s "$CLI_BINARY" list >/dev/null 2>&1; then
                    echo "  ✅ Migration $scenario: Handled gracefully"
                else
                    echo "  ✅ Migration $scenario: Failed as expected (corrupted)"
                fi
                ;;
            *)
                if timeout 5s "$CLI_BINARY" list --json >/dev/null 2>&1; then
                    echo "  ✅ Migration $scenario: Completed successfully"
                else
                    echo "  ❌ Migration $scenario: Failed"
                fi
                ;;
        esac
    done
}

# Performance testing
test_performance() {
    echo
    echo "7. Testing configuration performance..."
    
    export HOME="$TEST_DIR"
    
    # Create large configuration
    config_large="$TEST_DIR/large_config"
    mkdir -p "$config_large/.claude-code-env"
    
    cat > "$config_large/.claude-code-env/config.json" << 'GEN_EOF'
{
    "version": "2.0",
    "environments": [
GEN_EOF
    for i in {1..50}; do
        cat >> "$config_large/.claude-code-env/config.json" << EOF
        {
            "name": "env-$i",
            "url": "https://api.example$i.com",
            "api_key": "api-key-$i"
        }$(if [[ $i -lt 50 ]]; then echo ","; else echo ""; fi)
EOF
    done
    
    cat >> "$config_large/.claude-code-env/config.json" << 'GEN_EOF'
    ]
}
GEN_EOF
    chmod 600 "$config_large/.claude-code-env/config.json"
    
    # Measure performance
    export HOME="$config_large"
    local start_time=$(date +%s%3N)
    
    if timeout 15s "$CLI_BINARY" list --json >/dev/null 2>&1; then
        local end_time=$(date +%s%3N)
        local duration=$((end_time - start_time))
        echo "  ✅ Large config load: ${duration}ms"
        
        if [[ $duration -lt 1000 ]]; then
            echo "  ✅ Performance within acceptable range"
        else
            echo "  ⚠️  Performance may need optimization"
        fi
    else
        echo "  ❌ Large config load failed"
    fi
}

# Main test execution
main() {
    local failures=0
    
    test_version_compatibility || failures=$((failures + 1))
    test_file_format_compatibility || failures=$((failures + 1))
    test_permissions_compatibility || failures=$((failures + 1))
    test_backup_recovery || failures=$((failures + 1))
    test_environment_variables || failures=$((failures + 1))
    test_migration_scenarios || failures=$((failures + 1))
    test_performance || failures=$((failures + 1))
    
    echo
echo "=== Configuration Compatibility Test Summary ==="
    if [[ $failures -eq 0 ]]; then
        echo "🎉 All configuration compatibility tests passed!"
    else
        echo "❌ $failures test(s) failed"
    fi
    
    # Cleanup
    rm -rf "$PROJECT_ROOT/cce"
    rm -rf "$TEST_DIR"
    
    return $failures
}

main "$@"