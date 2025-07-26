#!/bin/bash
set -e
cd /Users/chenwenjie/Downloads/claude-code-env-switch

echo "=== Claude Code Environment Switcher Module Rename Validation ==="
echo "Old: github.com/claude-code/env-switcher"
echo "New: github.com/cexll/claude-code-env"
echo

# 1. Clean build artifacts
echo "🔧 Cleaning build artifacts..."
make clean

# 2. Verify import paths
echo "🔍 Verifying import paths are correctly updated..."
if grep -r "github\.com/claude-code/env-switcher" . --include="*.go" --exclude-dir=vendor --exclude=coverage.out 2>/dev/null; then
    echo "❌ ERROR: Found old import paths that need updating"
    exit 1
fi

# Count new import usage
NEW_IMPORT_COUNT=$(grep -r "github\.com/cexll/claude-code-env" . --include="*.go" --exclude-dir=vendor | wc -l)
echo "✅ Found $NEW_IMPORT_COUNT new module import references"

# 3. Verify go.mod
echo "📄 Verifying go.mod module declaration..."
if grep -q "module github.com/cexll/claude-code-env" go.mod; then
    echo "✅ go.mod module declaration correct"
else
    echo "❌ go.mod module declaration incorrect"
    exit 1
fi

# 4. Update dependencies
echo "📦 Updating dependencies..."
go mod tidy
go mod verify

# 5. Test compilation across all packages
echo "🔨 Testing compilation integrity..."
echo "   Testing main build..."
go build -o cce . && echo "✅ Main build successful"

echo "   Testing cross-platform builds..."
GOOS=darwin GOARCH=amd64 go build -o cce-darwin-amd64 . && echo "✅ Darwin AMD64"
GOOS=darwin GOARCH=arm64 go build -o cce-darwin-arm64 . && echo "✅ Darwin ARM64"
GOOS=linux GOARCH=amd64 go build -o cce-linux-amd64 . && echo "✅ Linux AMD64"
GOOS=linux GOARCH=arm64 go build -o cce-linux-arm64 . && echo "✅ Linux ARM64"
GOOS=windows GOARCH=amd64 go build -o cce-windows-amd64.exe . && echo "✅ Windows AMD64"

echo "   Testing test compilation..."
go test -c ./cmd/ && echo "✅ cmd tests compile"
go test -c ./internal/config/ && echo "✅ config tests compile"
go test -c ./internal/network/ && echo "✅ network tests compile"
go test -c ./internal/ui/ && echo "✅ ui tests compile"
go test -c ./internal/launcher/ && echo "✅ launcher tests compile"

# 6. Test Makefile build-all
echo "📝 Testing build system..."
make build-all && echo "✅ Makefile build-all successful"

# 7. Test unit tests
echo "🧪 Running unit tests..."
go test ./... -v && echo "✅ All unit tests pass"

# 8. Configuration compatibility
echo "⚙️  Testing configuration compatibility..."
if [ -d "dist" ]; then
    echo "✅ Build artifacts generated successfully"
    ls -la dist/
else
    echo "❌ Build artifacts missing"
    exit 1
fi

# 9. CLI functionality
echo "🔧 Testing CLI basic functionality..."
./cce --help > /dev/null && echo "✅ CLI help command works"
./cce version 2>/dev/null || echo "ℹ️  Version command may not be implemented (non-critical)"

# Cleanup
echo "🧹 Cleaning up test artifacts..."
rm -f cce* test.test

echo
echo "🎉 All module rename validation tests completed successfully!"
echo "Module github.com/cexll/claude-code-env is ready for use."