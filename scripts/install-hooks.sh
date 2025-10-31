#!/bin/bash
# Install git hooks for cloud-deploy development

set -e

echo "Installing git hooks for cloud-deploy..."

# Get the git directory
GIT_DIR=$(git rev-parse --git-dir 2>/dev/null)

if [ -z "$GIT_DIR" ]; then
    echo "Error: Not a git repository"
    exit 1
fi

# Get the script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Install pre-commit hook
if [ -f "$SCRIPT_DIR/pre-commit" ]; then
    echo "Installing pre-commit hook..."
    cp "$SCRIPT_DIR/pre-commit" "$GIT_DIR/hooks/pre-commit"
    chmod +x "$GIT_DIR/hooks/pre-commit"
    echo "✓ Pre-commit hook installed"
else
    echo "Error: pre-commit hook not found at $SCRIPT_DIR/pre-commit"
    exit 1
fi

echo ""
echo "Git hooks installed successfully!"
echo ""
echo "The pre-commit hook will run:"
echo "  - Code formatting checks (go fmt)"
echo "  - Static analysis (go vet)"
echo "  - Unit tests (go test)"
echo "  - Dependency tidiness (go mod tidy)"
echo ""
echo "To bypass the hook (not recommended), use: git commit --no-verify"
