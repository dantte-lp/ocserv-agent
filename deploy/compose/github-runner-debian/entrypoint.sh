#!/bin/bash
set -e

# Configuration
REPO_URL="${REPO_URL:-https://github.com/dantte-lp/ocserv-agent}"
RUNNER_NAME="${RUNNER_NAME:-podman-runner}"
RUNNER_LABELS="${RUNNER_LABELS:-self-hosted,linux,x64}"
RUNNER_GROUP="${RUNNER_GROUP:-Default}"

echo "🏃 GitHub Actions Self-Hosted Runner"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Repository: $REPO_URL"
echo "Runner Name: $RUNNER_NAME"
echo "Labels: $RUNNER_LABELS"
echo ""

# Check if RUNNER_TOKEN is provided
if [ -z "$RUNNER_TOKEN" ]; then
    echo "❌ ERROR: RUNNER_TOKEN environment variable is required"
    echo ""
    echo "Get token with:"
    echo "  gh api --method POST -H 'Accept: application/vnd.github+json' \\"
    echo "    /repos/dantte-lp/ocserv-agent/actions/runners/registration-token \\"
    echo "    --jq '.token'"
    exit 1
fi

# Cleanup function
cleanup() {
    echo ""
    echo "🛑 Shutting down runner..."

    # Remove runner from GitHub
    if [ -f ".runner" ]; then
        echo "Removing runner from GitHub..."
        ./config.sh remove --token "$RUNNER_TOKEN" || true
    fi

    echo "✅ Cleanup complete"
}

trap cleanup EXIT SIGTERM SIGINT

# Check if already configured
if [ ! -f ".runner" ]; then
    echo "📝 Configuring runner..."

    # Run config.sh and capture output (allow warnings but catch real errors)
    set +e  # Temporarily disable exit on error
    OUTPUT=$(./config.sh \
        --url "$REPO_URL" \
        --token "$RUNNER_TOKEN" \
        --name "$RUNNER_NAME" \
        --labels "$RUNNER_LABELS" \
        --runnergroup "$RUNNER_GROUP" \
        --work "_work" \
        --unattended \
        --replace 2>&1)
    CONFIG_EXIT_CODE=$?
    set -e  # Re-enable exit on error

    # Show output
    echo "$OUTPUT"

    # Check if configuration actually succeeded (check for .runner file)
    if [ ! -f ".runner" ]; then
        echo "❌ ERROR: Runner configuration failed!"
        echo "Exit code: $CONFIG_EXIT_CODE"
        exit 1
    fi

    echo "✅ Runner configured"
else
    echo "✅ Runner already configured"
fi

# Start runner
echo ""
echo "🚀 Starting runner..."
exec ./run.sh
