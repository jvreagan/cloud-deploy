# Release Process

Quick reference for creating new releases of cloud-deploy.

## Prerequisites

1. **Create homebrew-tap repository** (one-time setup):
   ```bash
   # On GitHub, create: jvreagan/homebrew-tap
   git clone https://github.com/jvreagan/homebrew-tap
   cd homebrew-tap
   mkdir -p Formula
   echo "# Homebrew Tap for cloud-deploy" > README.md
   git add . && git commit -m "Initial tap" && git push
   ```

2. **Create GitHub Personal Access Token** (one-time setup):
   - Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
   - Generate new token with `repo` and `write:packages` scopes
   - Add as repository secret: `TAP_GITHUB_TOKEN`

3. **Uncomment in `.github/workflows/release.yml`**:
   ```yaml
   env:
     TAP_GITHUB_TOKEN: ${{ secrets.TAP_GITHUB_TOKEN }}
   ```

## Creating a Release

### 1. Update Version

Decide on the new version number (following [semantic versioning](https://semver.org/)):
- **Patch** (0.1.x): Bug fixes, small changes
- **Minor** (0.x.0): New features, backward compatible
- **Major** (x.0.0): Breaking changes

### 2. Update CHANGELOG (optional but recommended)

Create/update `CHANGELOG.md`:
```markdown
## [0.2.0] - 2024-10-29

### Added
- Web UI for manifest generation
- Homebrew installation support

### Changed
- Improved GCP provider with resource limits

### Fixed
- Bug in manifest parsing
```

### 3. Commit Changes

```bash
git add .
git commit -m "Prepare release v0.2.0"
git push
```

### 4. Create and Push Tag

```bash
# Create tag
git tag -a v0.2.0 -m "Release v0.2.0"

# Push tag to GitHub
git push origin v0.2.0
```

### 5. GitHub Actions Takes Over

The workflow will automatically:
1. ✅ Run tests (`go test ./...`)
2. ✅ Build binaries for all platforms (macOS, Linux, Windows × amd64/arm64)
3. ✅ Create archives (tar.gz for Unix, zip for Windows)
4. ✅ Calculate SHA256 checksums
5. ✅ Create GitHub release with all artifacts
6. ✅ Generate Homebrew formula
7. ✅ Push formula to homebrew-tap repository

### 6. Verify Release

1. Check GitHub releases: https://github.com/jvreagan/cloud-deploy/releases
2. Check homebrew-tap: https://github.com/jvreagan/homebrew-tap/tree/main/Formula
3. Test installation:
   ```bash
   brew uninstall cloud-deploy  # if already installed
   brew untap jvreagan/tap
   brew tap jvreagan/tap
   brew install cloud-deploy
   cloud-deploy -version
   ```

## Testing Before Release

### Test GoReleaser Locally

```bash
# Install goreleaser
brew install goreleaser

# Build snapshot (doesn't push anything)
goreleaser release --snapshot --clean

# Check artifacts
ls -la dist/
```

### Test Homebrew Formula

```bash
# After goreleaser snapshot
brew install --build-from-source ./dist/homebrew/Formula/cloud-deploy.rb

# Or test directly
brew install --build-from-source homebrew-formula/cloud-deploy.rb
```

## Troubleshooting

### Build Failed

Check GitHub Actions logs:
1. Go to repository → Actions tab
2. Click on the failed workflow
3. Check the logs for errors

Common issues:
- Tests failing → Fix tests before tagging
- Missing permissions → Check PAT token has correct scopes
- Formula errors → Test goreleaser locally first

### Wrong Version Number

If you tagged the wrong version:
```bash
# Delete local tag
git tag -d v0.2.0

# Delete remote tag
git push origin :refs/tags/v0.2.0

# Create correct tag
git tag -a v0.2.1 -m "Release v0.2.1"
git push origin v0.2.1
```

### Homebrew Formula Not Updated

Check:
1. `TAP_GITHUB_TOKEN` secret is set
2. Token has correct permissions
3. homebrew-tap repository exists
4. Check GitHub Actions logs for errors

Manual formula update:
```bash
cd ../homebrew-tap
# Edit Formula/cloud-deploy.rb manually
git add Formula/cloud-deploy.rb
git commit -m "Update formula to v0.2.0"
git push
```

## Release Checklist

- [ ] All tests passing
- [ ] Version numbers updated
- [ ] CHANGELOG updated (optional)
- [ ] Changes committed and pushed
- [ ] Tag created and pushed
- [ ] GitHub Actions workflow completed successfully
- [ ] GitHub release created with all artifacts
- [ ] Homebrew formula updated in tap repository
- [ ] Installation tested with Homebrew
- [ ] Documentation updated if needed

## Rollback

If a release has issues:

1. **Delete the release on GitHub** (keeps the tag)
2. **Fix the issues**
3. **Create a new patch release** (e.g., v0.2.1)

Don't reuse the same tag - create a new one.

## Version History

Track releases:
- v0.1.0 - Initial release
- v0.2.0 - Added Web UI and Homebrew support
- (future versions...)

## Resources

- [GoReleaser Documentation](https://goreleaser.com/intro/)
- [Semantic Versioning](https://semver.org/)
- [GitHub Actions](https://docs.github.com/en/actions)
- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
