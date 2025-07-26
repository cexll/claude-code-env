#!/bin/bash
# Module Integrity Verification Test
# Validates module name, dependencies, and file structure

set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
readonly OLD_MODULE="github.com/claude-code/env-switcher"
readonly NEW_MODULE="github.com/cexll/claude-code-env"

cd "$PROJECT_ROOT"

echo "=== Module Integrity Validation ==="

# 1. Verify module name in go.mod
echo "🔍 Checking go.mod module declaration..."
if ! grep -q "module $NEW_MODULE" go.mod; then
    echo "❌ ERROR: Module name in go.mod is incorrect"
    echo "Expected: module $NEW_MODULE"
    echo "Found:"
    head -n 1 go.mod
    exit 1
fi
echo "✅ Module name correctly set to: $NEW_MODULE"

# 2. Verify module dependencies
echo "🔍 Validating go.mod dependencies..."
go mod validate || {
    echo "❌ ERROR: go.mod validation failed"
    go mod why -m all
    exit 1
}

# Check for any references to old module
echo "🔍 Scanning for old module references..."
if grep -r "$OLD_MODULE" . --include="*.go" --exclude-dir=vendor --exclude-dir=.git 2>/dev/null; then
    echo "❌ ERROR: Found references to old module $OLD_MODULE"
    exit 1
fi
echo "✅ No old module references found"

# 3. Validate module can be downloaded
echo "🔍 Testing module download..."
go mod download
go mod verify
echo "✅ Module dependencies downloaded and verified"

# 4. Check go.sum integrity
echo "🔍 Validating go.sum..."
if [ ! -f "go.sum" ]; then
    echo "❌ ERROR: go.sum file missing"
    exit 1
fi

if ! go mod verify; then
    echo "❌ ERROR: go.sum verification failed"
    exit 1
fi
echo "✅ go.sum verification passed"

# 5. Validate module can be imported
echo "🔍 Testing module import..."
mkdir -p /tmp/module_test
cd /tmp/module_test

cat > go.mod << EOF
module module_test

go 1.24

require $NEW_MODULE latest
EOF

if go mod tidy; then
    echo "✅ Module can be imported successfully"
    cd "$PROJECT_ROOT"
    rm -rf /tmp/module_test
else
    echo "❌ ERROR: Module cannot be imported"
    exit 1
fi

echo "🎉 Module integrity validation completed successfully!"