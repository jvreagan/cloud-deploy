#!/bin/bash
# View historical GitHub traffic statistics
# Usage: ./scripts/view-traffic-stats.sh [repo-name] [days]

set -e

REPO_NAME="${1:-all}"
DAYS="${2:-30}"

echo "📊 GitHub Traffic Statistics"
echo "=============================="
echo ""

# Check if traffic-stats branch exists
if ! git ls-remote --heads origin traffic-stats | grep -q traffic-stats; then
    echo "❌ Traffic stats branch doesn't exist yet."
    echo "The automated tracker will create it on first run."
    echo ""
    echo "You can manually trigger it with:"
    echo "  gh workflow run traffic-tracker.yml"
    exit 1
fi

# Fetch the traffic-stats branch
echo "Fetching latest traffic data..."
git fetch origin traffic-stats:traffic-stats 2>/dev/null || true

# Checkout traffic-stats branch (detached)
git checkout traffic-stats -- traffic-data/ 2>/dev/null || {
    echo "❌ No traffic data found."
    echo "Run the workflow first: gh workflow run traffic-tracker.yml"
    exit 1
}

echo ""

if [ "$REPO_NAME" = "all" ]; then
    echo "📈 Summary for ALL repositories (last $DAYS days)"
    echo ""

    # Get list of all data files
    FILES=$(ls -1 traffic-data/*.json 2>/dev/null | tail -n $DAYS)

    if [ -z "$FILES" ]; then
        echo "No data files found"
        exit 1
    fi

    # Display latest summary
    if [ -f "traffic-data/latest-summary.md" ]; then
        cat "traffic-data/latest-summary.md"
        echo ""
    fi

    # Show trend data
    echo "## Historical Trend (Last $DAYS Days)"
    echo ""
    echo "Date       | Repository | Views | Unique | Clones | Unique"
    echo "-----------|------------|-------|--------|--------|-------"

    for file in $FILES; do
        DATE=$(basename "$file" .json)
        jq -r --arg date "$DATE" '
            .repositories | to_entries[] |
            "\($date) | \(.key) | \(.value.views_14d) | \(.value.unique_visitors_14d) | \(.value.clones_14d) | \(.value.unique_cloners_14d)"
        ' "$file"
    done | column -t -s '|'

else
    echo "📈 Statistics for: $REPO_NAME (last $DAYS days)"
    echo ""

    FILES=$(ls -1 traffic-data/*.json 2>/dev/null | tail -n $DAYS)

    if [ -z "$FILES" ]; then
        echo "No data files found"
        exit 1
    fi

    echo "Date       | Stars | Forks | Views | Unique Visitors | Clones | Unique Cloners"
    echo "-----------|-------|-------|-------|-----------------|--------|---------------"

    for file in $FILES; do
        DATE=$(basename "$file" .json)
        jq -r --arg repo "$REPO_NAME" --arg date "$DATE" '
            if .repositories[$repo] then
                "\($date) | \(.repositories[$repo].stars) | \(.repositories[$repo].forks) | \(.repositories[$repo].views_14d) | \(.repositories[$repo].unique_visitors_14d) | \(.repositories[$repo].clones_14d) | \(.repositories[$repo].unique_cloners_14d)"
            else
                empty
            end
        ' "$file"
    done | column -t -s '|'
fi

echo ""

# Clean up
git restore traffic-data/ 2>/dev/null || true

echo ""
echo "💡 Tips:"
echo "  - View specific repo: ./scripts/view-traffic-stats.sh cloud-deploy"
echo "  - Change days: ./scripts/view-traffic-stats.sh cloud-deploy 7"
echo "  - View all repos: ./scripts/view-traffic-stats.sh all 30"
echo ""
echo "📊 Raw data stored in traffic-stats branch: traffic-data/"
