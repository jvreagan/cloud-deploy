# Development Scripts

This directory contains scripts to help with cloud-deploy development.

## Git Hooks

### Installing Hooks

To install the git hooks, run:

```bash
./scripts/install-hooks.sh
```

This will install the pre-commit hook that runs before each commit.

### Pre-commit Hook

The pre-commit hook automatically runs these checks before each commit:

1. **Code Formatting** (`go fmt`)
   - Ensures all Go code is properly formatted
   - Fails if any files need formatting

2. **Static Analysis** (`go vet`)
   - Checks for common Go programming errors
   - Detects suspicious constructs

3. **Unit Tests** (`go test -short`)
   - Runs all unit tests
   - Uses `-short` flag to skip long-running tests

4. **Dependency Management** (`go mod tidy`)
   - Verifies go.mod and go.sum are up to date
   - Ensures no missing or unused dependencies

### Bypassing the Hook

In rare cases where you need to bypass the pre-commit hook, use:

```bash
git commit --no-verify
```

**Note:** This is not recommended. Only use when absolutely necessary (e.g., work-in-progress commits to a feature branch).

### Manual Testing

You can manually run the pre-commit checks without committing:

```bash
./scripts/pre-commit
```

## Troubleshooting

### Hook not running

If the hook isn't running, ensure it's executable:

```bash
chmod +x .git/hooks/pre-commit
```

### Hook fails on fresh checkout

Make sure you've run:

```bash
go mod download
```

### Tests are slow

The pre-commit hook uses `go test -short` to skip long-running tests. If tests are still slow, consider optimizing test setup/teardown.
