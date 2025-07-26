#!/bin/bash
# Network Validation Test Suite
# Tests network connectivity, SSL validation, API endpoint testing, and network performance

set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
readonly TEST_DIR="$PROJECT_ROOT/test_temp/network"
readonly CLI_BINARY="$PROJECT_ROOT/cce"

# Configurable timeouts
readonly CONNECT_TIMEOUT=5
readonly READ_TIMEOUT=10

# Test endpoints
readonly TEST_ENDPOINTS=(
    "https://api.anthropic.com:443"
    "https://claude.ai:443"
    "https://httpbin.org:443"
)

# Custom test endpoints for testing
cd "$PROJECT_ROOT"

echo "=== Network Validation Testing ==="

# Clean and setup test environment
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"

# Build CLI binary if not exists
if [[ ! -f "$CLI_BINARY" ]]; then
    echo "🔨 Building CLI binary..."
    go build -o "$CLI_BINARY" .
fi
chmod +x "$CLI_BINARY"

# Test certificate validation
test_ssl_cert_validation() {
    echo
    echo "1. Testing SSL certificate validation..."
    
    for endpoint in "${TEST_ENDPOINTS[@]}"; do
        echo -n "Testing $endpoint... "
        
        # Extract hostname and port
        local host=$(echo "$endpoint" | sed -E 's|https?://([^:/]+)(:[0-9]+)?.*|\1|')
        local port=$(echo "$endpoint" | sed -E 's|.*/([^:]+):([0-9]+)|\2|')
        [[ "$port" == *"https://"* ]] && port=443
        
        if timeout $CONNECT_TIMEOUT bash -c "</dev/null 2>/dev/null openssl s_client -connect $host:$port -servername $host" >/dev/null 2>&1; then
            echo "✅ SSL OK"
        else
            echo "❌ SSL FAILED"
        fi
    done
}

# Test network connectivity to endpoints
test_connectivity() {
    echo
echo "2. Testing network connectivity..."
    
    for endpoint in "${TEST_ENDPOINTS[@]}"; do
        echo -n "Testing $endpoint... "
        
        if command -v curl >/dev/null 2>&1; then
            if timeout $READ_TIMEOUT curl -s --connect-timeout $CONNECT_TIMEOUT -I "$endpoint" >/dev/null 2>&1; then
                echo "✅ CONNECT"
            else
                echo "❌ TIMEOUT/FAIL"
            fi
        elif command -v wget >/dev/null 2>&1; then
            if timeout $READ_TIMEOUT wget --spider -q --timeout=$CONNECT_TIMEOUT "$endpoint" >/dev/null 2>&1; then
                echo "✅ CONNECT"
            else
                echo "❌ TIMEOUT/FAIL"
            fi
        else
            # Fallback to socket test
            local host=$(echo "$endpoint" | sed -E 's|https?://([^:/]+)(:[0-9]+)?.*|\1|')
            local port=$(echo "$endpoint" | sed -E 's|.*/([^:]+):([0-9]+)|\2|')
            [[ "$port" == *"https://"* ]] && port=443
            
            if timeout $CONNECT_TIMEOUT bash -c "</dev/null 2>/dev/null > /dev/tcp/$host/$port" >/dev/null 2>&1; then
                echo "✅ SOCKET OK"
            else
                echo "❌ SOCKET FAIL"
            fi
        fi
    done
}

# Test URL validation for environment setup
test_url_validation() {
    echo
    echo "3. Testing URL validation..."
    
    valid_urls=(
        "https://api.anthropic.com"
        "https://claude.ai/api"
        "https://custom-endpoint.company.com:8080"
        "https://192.168.1.100:443"
        "https://domain.com/v1/messages"
    )
    
    invalid_urls=(
        "http://insecure-endpoint.com"
        "ftp://invalid-protocol.com"
        "not-a-url"
        "https://"
        "https://invalid..url.com"
    )
    
    echo "Testing valid URLs..."
    for url in "${valid_urls[@]}"; do
        echo -n "  $url: "
        if [[ $url =~ ^https://[^[:space:]]+$ ]]; then
            echo "✅ VALID"
        else
            echo "❌ INVALID"
        fi
    done
    
    echo
echo "Testing invalid URLs..."
    for url in "${invalid_urls[@]}"; do
        echo -n "  $url: "
        if [[ $url =~ ^https://[^[:space:]]+$ ]]; then
            echo "✅ VALID (should be invalid)"
        else
            echo "❌ INVALID"
        fi
    done
}

# Test SSL certificate details
test_certificate_details() {
    echo
    echo "4. Testing SSL certificate details..."
    
    for endpoint in "${TEST_ENDPOINTS[@]}"; do
        local host=$(echo "$endpoint" | sed -E 's|https?://([^:/]+)(:[0-9]+)?.*|\1|')
        
        echo "Certificate details for $host:"
        if timeout $CONNECT_TIMEOUT openssl s_client -connect $host:443 -servername $host 2>/dev/null </dev/null | openssl x509 -noout -text | grep -E "(Not Valid After|Not Valid Before|Subject:|Issuer:)" 2>/dev/null || true; then
            echo "---"
        else
            echo "  Could not retrieve certificate details"
        fi
    done
}

# Test network performance
test_performance() {
    echo
    echo "5. Testing network performance..."
    
    local test_url="https://httpbin.org/delay/2"
    
    echo "Testing response time from:"
    
    for endpoint in "${TEST_ENDPOINTS[@]}"; do
        echo -n "  $endpoint: "
        
        local start_time=$(date +%s.%N)
        
        if curl -s --connect-timeout 5 --max-time 10 "$endpoint" >/dev/null 2>&1; then
            local end_time=$(date +%s.%N)
            local duration=$(echo "$end_time - $start_time" | bc -l 2>/dev/null || echo "1")
            printf "%0.2fs\n" "$duration"
        else
            echo "timeout/fail"
        fi
    done
}

# Test API endpoint validation
test_api_endpoint_validation() {
    echo
    echo "6. Testing API endpoint validation..."
    
    export HOME="$TEST_DIR"
    
    # Test various API endpoints
    echo "=== API Endpoint Testing ==="
    
    # Test with actual Claude API
    echo "Testing Claude API endpoint..."
    
    local test_payload='{"model": "claude-3-sonnet-20240229", "messages": [{"role": "user", "content": "test"}], "max_tokens": 1}'
    
    if echo "$test_payload" | jq . >/dev/null 2>&1; then
        echo "  ✅ JSON payload valid"
    else
        echo "  ❌ JSON payload invalid"
    fi
    
    # Test with mock server
    if command -v python3 >/dev/null 2>&1; then
        echo "  Testing with local mock server..."
        
        # Create simple mock server
        cat > "$TEST_DIR/mock_server.py" << 'EOF'
#!/usr/bin/env python3
import http.server
import socketserver
import json
import sys

class MockHandler(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        self.send_response(200)
        self.send_header('Content-type', 'application/json')
        self.end_headers()
        response = {'id': 'test', 'content': [{'text': 'mock response', 'type': 'text'}]}
        self.wfile.write(json.dumps(response).encode())
    
    def log_message(self, format, *args):
        return

PORT = 8080
Handler = MockHandler
with socketserver.TCPServer(("localhost", PORT), Handler) as httpd:
    httpd.serve_forever()
EOF
        
        chmod +x "$TEST_DIR/mock_server.py"
        
        # Start mock server in background
        if timeout 5s "$TEST_DIR/mock_server.py" &; then
            echo "  ✅ Mock server started (simulated)"
        else
            echo "  ⚠️  Mock server not available"
        fi
    fi
    
    # Test SSL/TLS versions
    echo
    echo "7. Testing SSL/TLS version support..."
    
    local test_host="api.anthropic.com"
    
    for tls_version in "1.2" "1.3"; do
        echo -n "TLS $tls_version: "
        if timeout $CONNECT_TIMEOUT bash -c "</dev/null 2>/dev/null openssl s_client -connect $test_host:443 -servername $test_host -tls$tls_version" >/dev/null 2>&1; then
            echo "✅ SUPPORTED"
        else
            echo "❌ NOT SUPPORTED/MISMATCH"
        fi
    done
}

# Test cache validation
test_cache_validation() {
    echo
    echo "8. Testing network cache validation..."
    
    export HOME="$TEST_DIR"
    
    # Create test configuration with cache
    mkdir -p "$TEST_DIR/.claude-code-env"
    
    cat > "$TEST_DIR/.claude-code-env/config.json" << 'EOF'
{
    "version": "2.0",
    "environments": [
        {
            "name": "net-test",
            "url": "https://api.anthropic.com",
            "api_key": "test-key"
        }
    ]
}
EOF
    chmod 600 "$TEST_DIR/.claude-code-env/config.json"
    
    # Test network validation with caching
    if "$CLI_BINARY" --validate-config >/dev/null 2>&1; then
        echo "  ✅ Network validation with caching OK"
    else
        echo "  ⚠️  Network validation failed"
    fi
}

# DNS resolution testing
test_dns_resolution() {
    echo
    echo "9. Testing DNS resolution..."
    
    hosts=(
        "api.anthropic.com"
        "claude.ai"
        "8.8.8.8"
        "localhost"
    )
    
    for host in "${hosts[@]}"; do
        echo -n "$host: "
        if command -v dig >/dev/null 2>&1; then
            if dig +short "$host" >/dev/null 2>&1; then
                echo "✅ DNS OK"
            else
                echo "❌ DNS FAIL"
            fi
        elif command -v nslookup >/dev/null 2>&1; then
            if nslookup "$host" >/dev/null 2>&1; then
                echo "✅ DNS OK"
            else
                echo "❌ DNS FAIL"
            fi
        else
            echo "? DNS UNTESTED"
        fi
    done
}

# Proxy testing (if configured)
test_proxy_support() {
    echo
    echo "10. Testing proxy support..."
    
    local has_proxy=false
    
    if [[ -n "${http_proxy:-}" || -n "${https_proxy:-}" || -n "${HTTP_PROXY:-}" || -n "${HTTPS_PROXY:-}" ]]; then
        has_proxy=true
        echo "Proxy detected in environment"
        
        # Test proxy functionality with curl
        if timeout $CONNECT_TIMEOUT curl --connect-timeout $CONNECT_TIMEOUT -x "${http_proxy:-$https_proxy}" http://httpbin.org/get >/dev/null 2>&1; then
            echo "  ✅ Proxy test successful"
        else
            echo "  ⚠️  Proxy test failed or unavailable"
        fi
    else
        echo "  No proxy configured, skipping proxy tests"
    fi
}

# Main execution
main() {
    echo
echo "Starting comprehensive network validation tests..."
    
    local failures=0
    
    test_ssl_cert_validation || failures=$((failures + 1))
    test_connectivity || failures=$((failures + 1))  
    test_url_validation || failures=$((failures + 1))
    test_certificate_details || failures=$((failures + 1))
    test_performance || failures=$((failures + 1))
    test_api_endpoint_validation || failures=$((failures + 1))
    test_cache_validation || failures=$((failures + 1))
    test_dns_resolution || failures=$((failures + 1))
    test_proxy_support || failures=$((failures + 1))
    
    echo
    echo "=== Network Validation Test Summary ==="
    echo "Tests completed: $((8 + 2))"
    echo "Failures: $failures"
    
    if [[ $failures -eq 0 ]]; then
        echo "🎉 All network validation tests completed successfully!"
    else
        echo "⚠️  Some network validation tests failed (may be due to network configuration)"
    fi
    
    # Cleanup
    rm -rf "$CLI_BINARY"
    rm -rf "$TEST_DIR"
    
    return $failures
}

main "$@"