#!/bin/bash
# Import Path Verification Test
# Validates all Go source files use correct import paths

set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
readonly OLD_MODULE="github.com/claude-code/env-switcher"
readonly NEW_MODULE="github.com/cexll/claude-code-env"

cd "$PROJECT_ROOT"

echo "=== Import Path Verification ==="

# 1. Verify all Go files use new import path
echo "🔍 Scanning Go source files for import paths..."

# Find all Go files
mapfile -t go_files < <(find . -type f -name "*.go" -not -path "./vendor/*" -not -path "*/.*/*")

TOTAL_FILES=0
CORRECT_IMPORTS=0
ERROR_FILES=()

for go_file in "${go_files[@]}"; do
    TOTAL_FILES=$((TOTAL_FILES + 1))
    
    # Check for old module reference
    if grep -q "$OLD_MODULE" "$go_file"; then
        echo "❌ $go_file: Contains old module reference"
        ERROR_FILES+=("$go_file")
    else
        # Count new module references
        local refs=$(grep -c "$NEW_MODULE" "$go_file" 2>/dev/null || echo 0)
        if [[ $refs -gt 0 ]]; then
            echo "✅ $go_file: Uses correct module ($refs references)"
            CORRECT_IMPORTS=$((CORRECT_IMPORTS + 1))
        else
            echo "ℹ️  $go_file: No module references (this may be OK)")
        fi
    fi
done

echo

echo "=== Summary ==="
echo "Total Go files scanned: $TOTAL_FILES"
echo "Files with correct imports: $CORRECT_IMPORTS"
echo "Files with errors: ${#ERROR_FILES[@]}"

if [[ ${#ERROR_FILES[@]} -gt 0 ]]; then
    echo
    echo "❌ ERROR: Found files with incorrect imports:"
    for error_file in "${ERROR_FILES[@]}"; do
        echo "  - $error_file"
    done
    exit 1
fi

# 2. Verify package structure integrity
echo "🔍 Validating package structure..."

expected_packages=(
    "cmd"
    "internal/config"
    "internal/network"
    "internal/ui"
    "internal/launcher"
    "pkg/types"
    "test"
)

for pkg in "${expected_packages[@]}"; do
    if [[ ! -d "$pkg" ]]; then
        echo "❌ ERROR: Expected package directory missing: $pkg"
        exit 1
    fi
done
echo "✅ All expected package directories exist"

# 3. Validate go.mod replaces
echo "🔍 Checking go.mod for replace directives..."
if grep -q "replace" go.mod; then
    echo "⚠️  WARN: Found replace directives in go.mod"
    grep "replace" go.mod
else
    echo "✅ No replace directives found"
fi

# 4. Verify internal imports structure
echo "🔍 Analyzing internal import structure..."

# Check internal packages reference pattern
for pkg_dir in internal/*; do
    if [[ -d "$pkg_dir" ]]; then
        for go_file in $(find "$pkg_dir" -name "*.go"); do
            # Ensure internal packages reference other internal modules correctly
            if grep -q "$NEW_MODULE/internal/" "$go_file"; then
                echo "✅ $go_file: Uses correct internal package reference"
            fi        done
    fi
done

echo "🎉 All import paths are correctly configured!"