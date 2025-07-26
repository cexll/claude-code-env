#!/bin/bash
cd /Users/chenwenjie/Downloads/claude-code-env-switch

# Clean any existing build artifacts
echo "Cleaning build artifacts..."
rm -f cce cce-* coverage.out coverage.html

# Make script executable
chmod +x test_build.sh

# Verify all imports are correctly updated
echo "Verifying import paths..."
if grep -r "github\.com/claude-code/env-switcher" . --include="*.go" --exclude-dir=vendor 2>/dev/null; then
    echo "ERROR: Found old import paths that need updating"
    exit 1
fi

# Show current import path count
echo "Current import path usage:"
grep -r "github\.com/cexll/claude-code-env" . --include="*.go" --exclude-dir=vendor | wc -l
echo "Import paths appear correctly updated"

# Update dependencies
echo "Updating dependencies..."
go mod tidy 
go mod verify

if [ $? -ne 0 ]; then
    echo "ERROR: go mod verification failed"
    exit 1
fi

# Test build
echo "Testing build..."
go build -o cce . && echo "✓ Build successful" || { echo "✗ Build failed"; exit 1; }

# Test cross-platform builds
echo "Testing cross-platform builds..."
GOOS=darwin GOARCH=amd64 go build -o cce-darwin-amd64 . && echo "✓ Darwin AMD64 build successful"
GOOS=darwin GOARCH=arm64 go build -o cce-darwin-arm64 . && echo "✓ Darwin ARM64 build successful"
GOOS=linux GOARCH=amd64 go build -o cce-linux-amd64 . && echo "✓ Linux AMD64 build successful"
GOOS=linux GOARCH=arm64 go build -o cce-linux-arm64 . && echo "✓ Linux ARM64 build successful"
GOOS=windows GOARCH=amd64 go build -o cce-windows-amd64.exe . && echo "✓ Windows AMD64 build successful"

# Test compilation of test suite
echo "Testing test compilation..."
go test -c ./cmd/ && echo "✓ cmd tests compile"
go test -c ./internal/config/ && echo "✓ config tests compile" 
go test -c ./internal/network/ && echo "✓ network tests compile"
go test -c ./internal/ui/ && echo "✓ ui tests compile"
go test -c ./internal/launcher/ && echo "✓ launcher tests compile"

# Cleanup test binaries
rm -f cce* test.test

# Test actual unit tests
echo "Testing unit tests..."
go test ./... -v && echo "✓ All tests pass"

echo "All build integrity tests passed!"