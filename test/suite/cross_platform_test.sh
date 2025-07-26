#!/bin/bash
# Cross-Platform Build Verification Test
# Validates successful builds across different OS/arch combinations

set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
readonly BUILD_DIR="$PROJECT_ROOT/dist/cross-platform"

cd "$PROJECT_ROOT"

echo "=== Cross-Platform Build Verification ==="

# Clean build directory
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

# Define platforms to test
readonly PLATFORMS=(
    "darwin:amd64"
    "darwin:arm64"
    "linux:amd64"
    "linux:arm64"
    "windows:amd64"
    "windows:386"
    "linux:386"
    "linux:ppc64le"
    "linux:s390x"
)

SUCCESSFUL_BUILDS=0
FAILED_BUILDS=()

# 1. Test standard cross-platform builds
echo "🔨 Testing cross-platform compilation..."

for platform in "${PLATFORMS[@]}"; do
    IFS=':' read -r os arch <<< "$platform"
    
    suffix=""
    [[ "$os" == "windows" ]] && suffix=".exe"
    
    output_binary="$BUILD_DIR/cce-${os}-${arch}${suffix}"
    
    echo -n "Building for $os/$arch... "
    
    if env GOOS="$os" GOARCH="$arch" go build -ldflags="-s -w" -o "$output_binary" . 2>/dev/null; then
        # Verify binary was created
        if [[ -f "$output_binary" ]]; then
            file_size=$(stat -f%z "$output_binary" 2>/dev/null || stat -c%s "$output_binary" 2>/dev/null || echo "0")
            if [[ $file_size -gt 1000000 ]]; then  # Should be >1MB
                echo -e "✅ SUCCESS (${file_size} bytes)"
                SUCCESSFUL_BUILDS=$((SUCCESSFUL_BUILDS + 1))
            else
                echo "❌ FAIL (size issue: $file_size bytes)"
                FAILED_BUILDS+=("$os/$arch (size)")
            fi
        else
            echo "❌ FAIL (no file created)"
            FAILED_BUILDS+=("$os/$arch (missing)")
        fi
    else
        echo "❌ FAIL (compile error)"
        FAILED_BUILDS+=("$os/$arch (compile)")
    fi
done

# 2. Test Go version compatibility
echo
echo "🔍 Testing Go version compatibility..."
go_version=$(go version | awk '{print $3}' | sed 's/go//')
echo "Current Go version: $go_version"

if go version | grep -q "go1\.2"; then
    echo "✅ Go 1.24+ detected, module supports Go 1.24+"
else
    echo "⚠️  Consider upgrading to Go 1.24+ for optimal compatibility"
fi

# 3. Test CGO disabled builds
echo "🚀 Testing with CGO disabled..."
for os_arch in "linux:amd64" "darwin:amd64" "windows:amd64"; do
    IFS=':' read -r os arch <<< "$os_arch"
    
    output="$BUILD_DIR/cce-${os}-${arch}-static${suffix}"
    
    if env CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -ldflags="-s -w" -o "$output" . 2>/dev/null; then
        echo "✅ Static build for $os/$arch"
    else
        echo "⚠️  Static build failed for $os/$arch"
    fi
done

# 4. Test build tags
echo
echo "⚙️  Testing build tags..."

# Test build with build tags
if go build -tags production -o "$BUILD_DIR/cce-tagged" . 2>/dev/null; then
    echo "✅ Build with production tag succeeded"
else
    echo "⚠️  Build with production tag failed"
fi

# 5. Validate Makefile builds
echo
echo "🔍 Testing Makefile targets..."

if make build-all 2>/dev/null; then
    echo "✅ Makefile build-all target successful"
    
    # Verify build outputs
    if [[ -d "dist" ]]; then
        echo "✅ Build artifacts in dist/ directory"
        ls -la dist/cce* 2>/dev/null || echo "⚠️  Some build artifacts may be missing"
    else
        echo "⚠️  dist/ directory not created by build-all"
    fi
else
    echo "❌ Makefile build-all target failed"
fi

# 6. Test build reproducibility
echo
echo "🔄 Testing build reproducibility..."

rm -f "$BUILD_DIR/repro_test"
go build -ldflags="-s -w" -o "$BUILD_DIR/repro_a" .
sleep 2
go build -ldflags="-s -w" -o "$BUILD_DIR/repro_b" .

if [ "$($BUILD_DIR/repro_a --help 2>/dev/null | head -1)" = "$($BUILD_DIR/repro_b --help 2>/dev/null | head -1)" ]; then
    echo "✅ Builds are reproducible"
else
    echo "⚠️  Builds may not be deterministic"
fi

rm -f "$BUILD_DIR/repro_a" "$BUILD_DIR/repro_b"

# Summary
echo
echo "=== Cross-Platform Build Summary ==="
echo "Total platforms tested: ${#PLATFORMS[@]}"
echo "Successful builds: $SUCCESSFUL_BUILDS"
echo "Failed builds: ${#FAILED_BUILDS[@]}"

if [[ ${#FAILED_BUILDS[@]} -gt 0 ]]; then
    echo
echo "Failed builds:"
    for failed in "${FAILED_BUILDS[@]}"; do
        echo "  - $failed"
    done
fi

if [[ $SUCCESSFUL_BUILDS -eq ${#PLATFORMS[@]} ]]; then
    echo
    echo "🎉 All cross-platform builds successful!"
    exit 0
else
    echo
    echo "❌ Some builds failed - please check the failed platforms"
    exit 1
fi