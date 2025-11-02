# GitHub Traffic Tracking

Automated system to track GitHub repository traffic statistics over time, overcoming GitHub's 14-day API limitation.

## Overview

GitHub's API only provides traffic data for the last 14 days. This system automatically collects and stores traffic statistics daily, allowing you to:

- **Track long-term trends** - View traffic data beyond the 14-day limit
- **Historical analysis** - Compare performance over weeks, months, or years
- **Multiple repositories** - Track all your public repos in one place
- **Automated collection** - No manual work required

## How It Works

```
┌─────────────────────┐
│ GitHub Actions      │  Runs daily at 00:00 UTC
│ (traffic-tracker)   │
└──────────┬──────────┘
           │
           ↓
┌─────────────────────┐
│ Collect traffic     │  Fetches views, clones, stars, forks
│ via GitHub API      │  for all repositories
└──────────┬──────────┘
           │
           ↓
┌─────────────────────┐
│ Store as JSON       │  traffic-data/YYYY-MM-DD.json
│ in traffic-stats    │  (separate branch)
│ branch              │
└─────────────────────┘
```

## Setup

The system is already configured and will start collecting data automatically. No setup required!

### Files Created

- **`.github/workflows/traffic-tracker.yml`** - GitHub Actions workflow
- **`scripts/view-traffic-stats.sh`** - Script to view historical data
- **`traffic-data/` directory** - Stored in `traffic-stats` branch (created automatically)

## Tracked Repositories

1. **cloud-deploy**
2. **homebrew-tap**
3. **mcp-memory-server**
4. **api-rate-limiter-rust**
5. **api-rate-limiter-go**
6. **api-rate-limiter-py**

## Viewing Traffic Data

### Option 1: Using the Script (Recommended)

```bash
# View all repositories (last 30 days)
./scripts/view-traffic-stats.sh

# View specific repository
./scripts/view-traffic-stats.sh cloud-deploy

# View last 7 days
./scripts/view-traffic-stats.sh cloud-deploy 7

# View all repos, last 90 days
./scripts/view-traffic-stats.sh all 90
```

### Option 2: View Raw Data

```bash
# Checkout the traffic-stats branch
git checkout traffic-stats

# View latest summary
cat traffic-data/latest-summary.md

# View specific date
cat traffic-data/2025-11-02.json | jq '.'

# List all collected data
ls -la traffic-data/
```

### Option 3: GitHub Web Interface

1. Go to your repository on GitHub
2. Switch to the `traffic-stats` branch
3. Browse the `traffic-data/` directory
4. View `latest-summary.md` for current stats

## Manual Trigger

To collect data immediately (instead of waiting for the daily schedule):

```bash
# Trigger via GitHub CLI
gh workflow run traffic-tracker.yml

# Or via GitHub web interface
# Actions → GitHub Traffic Tracker → Run workflow
```

## Data Format

Each daily file (`traffic-data/YYYY-MM-DD.json`) contains:

```json
{
  "date": "2025-11-02",
  "timestamp": "2025-11-02T00:00:00Z",
  "repositories": {
    "cloud-deploy": {
      "stars": 0,
      "forks": 0,
      "views_14d": 35,
      "unique_visitors_14d": 3,
      "clones_14d": 221,
      "unique_cloners_14d": 75
    }
  }
}
```

**Note:** Views and clones represent 14-day rolling windows from GitHub's API.

## Metrics Tracked

For each repository, daily:

| Metric | Description |
|--------|-------------|
| **Stars** | Total GitHub stars |
| **Forks** | Total forks |
| **Views (14d)** | Page views in last 14 days |
| **Unique Visitors (14d)** | Unique viewers in last 14 days |
| **Clones (14d)** | Git clones in last 14 days |
| **Unique Cloners (14d)** | Unique cloners in last 14 days |

## Understanding the Data

### Views vs Clones

- **Views** - People visiting the repository page on GitHub
- **Clones** - People/systems running `git clone` on your repository

### Why More Clones Than Views?

High clone-to-view ratios often indicate:
- ✅ CI/CD systems cloning your repo
- ✅ Developers using your project directly
- ✅ GitHub Actions workflows
- ✅ Package managers (Homebrew, etc.)

### 14-Day Rolling Window

GitHub provides a 14-day rolling window. Our system snapshots this daily, so:
- **Single day data** = That day's 14-day window
- **Historical trends** = Compare how the 14-day window changes over time

## Analyzing Trends

Example queries using the collected data:

```bash
# Count total data points collected
ls traffic-data/*.json | wc -l

# Find peak clone days for cloud-deploy
jq -r '.repositories["cloud-deploy"].clones_14d' traffic-data/*.json | sort -rn | head -5

# Calculate average views across all repos
jq -r '.repositories[].views_14d' traffic-data/*.json | awk '{sum+=$1; count++} END {print sum/count}'

# Get growth trends
for file in traffic-data/*.json; do
  date=$(basename "$file" .json)
  stars=$(jq -r '.repositories["cloud-deploy"].stars' "$file")
  echo "$date: $stars stars"
done
```

## Limitations

1. **14-day window** - Raw data from GitHub is always a 14-day rolling window
2. **API rate limits** - Workflow respects GitHub API limits
3. **Accuracy** - Depends on GitHub's traffic tracking accuracy
4. **Missing repos** - New repos need to be manually added to the workflow

## Adding More Repositories

Edit `.github/workflows/traffic-tracker.yml` and add to the `REPOS` array:

```yaml
REPOS=(
  "cloud-deploy"
  "your-new-repo"
  # Add more here
)
```

## Troubleshooting

### No data showing up?

1. Check if workflow ran: `gh run list --workflow=traffic-tracker.yml`
2. Check workflow logs: `gh run view <run-id>`
3. Verify branch exists: `git ls-remote --heads origin traffic-stats`

### Script errors?

```bash
# Fetch latest data
git fetch origin traffic-stats:traffic-stats

# Try running manually
./scripts/view-traffic-stats.sh
```

### Workflow not running?

- Verify it's enabled: Actions → Traffic Tracker → Enable workflow
- Check schedule: Should run daily at 00:00 UTC
- Trigger manually: `gh workflow run traffic-tracker.yml`

## Privacy & Security

- ✅ Only collects **public** traffic data available via GitHub API
- ✅ Uses GitHub Actions bot token (no personal credentials)
- ✅ Data stored in your own repository
- ✅ No external services or data sharing

## Future Enhancements

Potential improvements:

- 📊 Visualization dashboard (GitHub Pages)
- 📈 Trend analysis and reports
- 🔔 Notifications for traffic spikes
- 📉 Compare multiple repos side-by-side
- 💾 Export to CSV/Excel

## License

Part of the cloud-deploy project - MIT License

---

**Questions?** Check the script comments or open an issue!
