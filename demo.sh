#!/bin/bash

# CCE Demo Script - Shows how to use the Claude Code Environment Switcher

echo "=== Claude Code Environment Switcher Demo ==="
echo

# Build the application
echo "1. Building CCE..."
make build
echo

# Show help
echo "2. CCE Help:"
./cce --help
echo

# Show current environments (should be empty)
echo "3. Current environments:"
./cce env list
echo

# Demo of adding an environment (commented out since it requires interactive input)
echo "4. To add a new environment, run:"
echo "   ./cce env add production"
echo "   Then follow the prompts to enter:"
echo "   - Description: Production Claude API"
echo "   - Base URL: https://api.anthropic.com/v1"
echo "   - API Key: sk-ant-api03-xxxxx"
echo

echo "5. After adding environments, you can:"
echo "   - List them: ./cce env list"
echo "   - Launch Claude Code: ./cce"
echo "   - Use specific environment: ./cce --env production"
echo "   - Edit environment: ./cce env edit production"
echo "   - Remove environment: ./cce env remove production"
echo

echo "6. Configuration is stored securely at:"
echo "   ~/.claude-code-env/config.json (with 600 permissions)"
echo

echo "=== Demo Complete ==="