# Homebrew Installation Guide

This guide explains how to make cloud-deploy available via Homebrew and how users can install it.

## For Users: Installing via Homebrew

Once the tap is set up, users can install cloud-deploy with:

```bash
# Add the tap (one-time)
brew tap jvreagan/tap

# Install cloud-deploy
brew install cloud-deploy
```

Both binaries will be available:
```bash
# CLI tool
cloud-deploy -version

# Web UI
manifest-ui
```

### Running the Web UI after Homebrew install

```bash
# The manifest-ui binary will be in your PATH
manifest-ui

# Or run with full path
/usr/local/bin/manifest-ui  # Intel Mac
/opt/homebrew/bin/manifest-ui  # Apple Silicon Mac
```

Then open http://localhost:5001 in your browser.

## For Maintainers: Setting Up Homebrew Distribution

### Option 1: Automated with GoReleaser (Recommended)

This is the easiest approach. GoReleaser automatically:
- Builds binaries for all platforms
- Creates GitHub releases
- Generates and updates Homebrew formulas
- Calculates checksums

#### Step 1: Create a Homebrew Tap Repository

Create a new GitHub repository named `homebrew-tap`:

```bash
# On GitHub, create a new repo: jvreagan/homebrew-tap
# Then clone it locally
git clone https://github.com/jvreagan/homebrew-tap
cd homebrew-tap

# Create Formula directory
mkdir -p Formula
echo "# Homebrew Tap for cloud-deploy" > README.md
git add .
git commit -m "Initial tap setup"
git push
```

#### Step 2: Create a GitHub Personal Access Token (PAT)

1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Name it: `HOMEBREW_TAP_TOKEN`
4. Select scopes:
   - `repo` (all)
   - `write:packages`
5. Generate and copy the token

#### Step 3: Add Token to Repository Secrets

1. Go to your cloud-deploy repository
2. Settings → Secrets and variables → Actions
3. Click "New repository secret"
4. Name: `TAP_GITHUB_TOKEN`
5. Value: Paste the token you created
6. Click "Add secret"

#### Step 4: Uncomment the TAP_GITHUB_TOKEN in workflow

Edit `.github/workflows/release.yml` and uncomment:
```yaml
env:
  GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  TAP_GITHUB_TOKEN: ${{ secrets.TAP_GITHUB_TOKEN }}  # Uncomment this line
```

#### Step 5: Create a Release

```bash
# Tag a new version
git tag v0.2.0
git push origin v0.2.0
```

GitHub Actions will automatically:
1. Run tests
2. Build binaries for all platforms
3. Create a GitHub release
4. Update the Homebrew formula in your tap

#### Step 6: Test the Installation

```bash
brew tap jvreagan/tap
brew install cloud-deploy
cloud-deploy -version
```

### Option 2: Manual Formula Creation

If you prefer not to use GoReleaser:

#### Step 1: Create a Release Manually

```bash
# Build for different platforms
GOOS=darwin GOARCH=amd64 go build -o cloud-deploy-darwin-amd64 cmd/cloud-deploy/main.go
GOOS=darwin GOARCH=arm64 go build -o cloud-deploy-darwin-arm64 cmd/cloud-deploy/main.go
GOOS=linux GOARCH=amd64 go build -o cloud-deploy-linux-amd64 cmd/cloud-deploy/main.go

# Create archives
tar -czf cloud-deploy-darwin-amd64.tar.gz cloud-deploy-darwin-amd64
tar -czf cloud-deploy-darwin-arm64.tar.gz cloud-deploy-darwin-arm64
tar -czf cloud-deploy-linux-amd64.tar.gz cloud-deploy-linux-amd64

# Upload to GitHub releases manually
```

#### Step 2: Calculate SHA256 Checksums

```bash
shasum -a 256 cloud-deploy-darwin-amd64.tar.gz
shasum -a 256 cloud-deploy-darwin-arm64.tar.gz
shasum -a 256 cloud-deploy-linux-amd64.tar.gz
```

#### Step 3: Update the Formula

Edit `homebrew-formula/cloud-deploy.rb` with the correct URLs and SHA256 values.

#### Step 4: Copy to Tap Repository

```bash
cp homebrew-formula/cloud-deploy.rb ../homebrew-tap/Formula/
cd ../homebrew-tap
git add Formula/cloud-deploy.rb
git commit -m "Update cloud-deploy formula to v0.2.0"
git push
```

### Option 3: Submit to Homebrew Core (Advanced)

To get cloud-deploy into the official Homebrew repository:

1. Your project needs to be stable and well-maintained
2. Follow the [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
3. Submit a PR to [homebrew-core](https://github.com/Homebrew/homebrew-core)

Requirements:
- Stable version (not pre-release)
- 75+ GitHub stars (or notable project)
- 30-day waiting period for new formulae
- Must pass all Homebrew CI tests

## Testing the Formula Locally

Before publishing:

```bash
# Test the formula
brew install --build-from-source homebrew-formula/cloud-deploy.rb

# Or test from your tap
brew tap jvreagan/tap
brew install --build-from-source cloud-deploy

# Run audit
brew audit --strict cloud-deploy

# Test installation
brew test cloud-deploy
```

## Updating the Formula

When releasing a new version:

1. Update version in code
2. Create a new git tag: `git tag v0.3.0 && git push origin v0.3.0`
3. GoReleaser will automatically update the formula
4. Users update with: `brew upgrade cloud-deploy`

## Troubleshooting

### "No available formula with the name"

Make sure the tap is added:
```bash
brew tap jvreagan/tap
```

### "SHA256 mismatch"

The checksum in the formula doesn't match the binary. Regenerate with:
```bash
shasum -a 256 <file.tar.gz>
```

### "Formula not found"

Check that the formula is in the correct location:
```bash
ls ../homebrew-tap/Formula/cloud-deploy.rb
```

### Testing Different Architectures

Use Docker for Linux testing:
```bash
docker run -it --rm ubuntu:latest bash
apt-get update && apt-get install -y curl git
# Test installation
```

## Web UI Considerations

The Web UI (`manifest-ui`) requires the `web/static` directory. The Homebrew formula:
1. Installs the `manifest-ui` binary to `/usr/local/bin` or `/opt/homebrew/bin`
2. Installs web assets to the Homebrew Cellar

Users can run `manifest-ui` from any directory, and it will serve the bundled assets.

## Resources

- [GoReleaser Documentation](https://goreleaser.com/)
- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [Creating Homebrew Taps](https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap)
- [GitHub Actions for Go](https://github.com/actions/setup-go)
