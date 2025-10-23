#!/bin/bash
# Install git hooks for the project

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "📦 Installing git hooks..."

# Get the git root directory
GIT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null)

if [ -z "$GIT_ROOT" ]; then
    echo -e "${RED}Error: Not a git repository${NC}"
    exit 1
fi

HOOKS_DIR="$GIT_ROOT/.git/hooks"
SOURCE_DIR="$GIT_ROOT/scripts/hooks"

# Create hooks directory if it doesn't exist
mkdir -p "$HOOKS_DIR"

# Check if source hooks directory exists
if [ ! -d "$SOURCE_DIR" ]; then
    echo -e "${YELLOW}Warning: Hooks source directory not found at $SOURCE_DIR${NC}"
    echo "Creating hooks from template..."
    mkdir -p "$SOURCE_DIR"
fi

# Install pre-commit hook
echo -n "Installing pre-commit hook... "
cat > "$HOOKS_DIR/pre-commit" <<'EOF'
#!/bin/bash
# Pre-commit hook: Auto-format Go code with gofmt

# Format all Go files that are staged for commit
STAGED_GO_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' | grep -v '.pb.go$')

if [ -n "$STAGED_GO_FILES" ]; then
    echo "🔧 Running gofmt on staged Go files..."

    for file in $STAGED_GO_FILES; do
        # Format the file
        gofmt -s -w "$file"

        # Re-add the file to staging (since we modified it)
        git add "$file"
    done

    echo "✅ All Go files formatted"
fi

exit 0
EOF

chmod +x "$HOOKS_DIR/pre-commit"
echo -e "${GREEN}OK${NC}"

# Install pre-push hook (optional - runs quick checks)
echo -n "Installing pre-push hook... "
cat > "$HOOKS_DIR/pre-push" <<'EOF'
#!/bin/bash
# Pre-push hook: Run quick checks before pushing

echo "🚀 Running pre-push checks..."
echo ""

# Run quick checks
if [ -f "./scripts/quick-check.sh" ]; then
    ./scripts/quick-check.sh
else
    echo "⚠️  Warning: quick-check.sh not found, skipping checks"
fi

exit 0
EOF

chmod +x "$HOOKS_DIR/pre-push"
echo -e "${GREEN}OK${NC}"

echo ""
echo -e "${GREEN}✓ Git hooks installed successfully!${NC}"
echo ""
echo "Installed hooks:"
echo "  - pre-commit: Auto-formats Go code with gofmt"
echo "  - pre-push: Runs quick-check.sh before pushing"
echo ""
echo "To skip hooks temporarily, use:"
echo "  git commit --no-verify"
echo "  git push --no-verify"
echo ""
