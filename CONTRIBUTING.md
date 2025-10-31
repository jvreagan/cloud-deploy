# Contributing to cloud-deploy

Thank you for your interest in contributing to cloud-deploy! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Making Changes](#making-changes)
- [Adding a New Cloud Provider](#adding-a-new-cloud-provider)
- [Testing](#testing)
- [Documentation](#documentation)
- [Pull Request Process](#pull-request-process)

## Code of Conduct

This project follows the standard open-source code of conduct. Please be respectful and constructive in all interactions.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/cloud-deploy.git`
3. Add upstream remote: `git remote add upstream https://github.com/jvreagan/cloud-deploy.git`
4. Create a feature branch: `git checkout -b feature/your-feature-name`

## Development Setup

**Prerequisites:**
- Go 1.25 or later
- Git
- Cloud provider CLI tools (AWS CLI, gcloud, az, etc.) for testing

**Setup:**

```bash
# Clone the repository
git clone https://github.com/jvreagan/cloud-deploy.git
cd cloud-deploy

# Install dependencies
go mod download

# Install git hooks (recommended)
./scripts/install-hooks.sh

# Build the project
go build -o cloud-deploy cmd/cloud-deploy/main.go

# Run tests
go test ./...
```

## Project Structure

```
cloud-deploy/
├── cmd/cloud-deploy/      # CLI entry point
│   └── main.go            # Main application
├── pkg/
│   ├── manifest/          # Manifest parsing and validation
│   │   └── manifest.go
│   ├── provider/          # Provider interface
│   │   └── provider.go
│   └── providers/         # Provider implementations
│       ├── aws/           # AWS Elastic Beanstalk
│       ├── gcp/           # Google Cloud Run
│       ├── azure/         # Azure Container Instances
│       └── oci/           # Oracle Cloud Container Instances
├── examples/              # Example manifests
├── docs/                  # Documentation
├── deploy-manifest.example.yaml
├── README.md
└── CONTRIBUTING.md (this file)
```

## Making Changes

### Code Style

- Follow standard Go conventions
- Run `go fmt` before committing
- Use meaningful variable and function names
- Add comments for exported types and functions (Godoc format)

**Example Godoc comment:**

```go
// Deploy deploys an application according to the manifest.
// This method creates the application, environment, and deploys the code.
//
// Example:
//
//	result, err := provider.Deploy(ctx, manifest)
//	if err != nil {
//	  return err
//	}
//
// Returns deployment information including the application URL.
func (p *Provider) Deploy(ctx context.Context, m *manifest.Manifest) (*DeploymentResult, error) {
    // Implementation
}
```

### Commit Messages

Use clear, descriptive commit messages:

```
feat: add GCP Cloud Run provider
fix: handle missing environment gracefully
docs: update AWS manifest examples
test: add provider interface tests
```

**Format:**
- `feat:` new feature
- `fix:` bug fix
- `docs:` documentation changes
- `test:` test changes
- `refactor:` code refactoring
- `chore:` maintenance tasks

## Adding a New Cloud Provider

To add support for a new cloud provider:

### 1. Create Provider Package

Create a new directory under `pkg/providers/`:

```bash
mkdir -p pkg/providers/yourprovider
```

### 2. Implement Provider Interface

Create `pkg/providers/yourprovider/provider.go`:

```go
package yourprovider

import (
    "context"
    "github.com/jvreagan/cloud-deploy/pkg/manifest"
    "github.com/jvreagan/cloud-deploy/pkg/provider"
)

// Provider implements the Provider interface for YourProvider
type Provider struct {
    // Provider-specific fields
}

// NewProvider creates a new YourProvider provider
func NewProvider(m *manifest.Manifest) (*Provider, error) {
    return &Provider{}, nil
}

// Name returns the provider name
func (p *Provider) Name() string {
    return "yourprovider"
}

// Deploy deploys an application
func (p *Provider) Deploy(ctx context.Context, m *manifest.Manifest) (*provider.DeploymentResult, error) {
    // Implement deployment logic
    return nil, nil
}

// Destroy removes the deployment
func (p *Provider) Destroy(ctx context.Context, m *manifest.Manifest) error {
    // Implement destroy logic
    return nil
}

// Status returns deployment status
func (p *Provider) Status(ctx context.Context, m *manifest.Manifest) (*provider.DeploymentStatus, error) {
    // Implement status check logic
    return nil, nil
}
```

### 3. Register Provider

Update `pkg/provider/provider.go`:

```go
func Factory(providerName string) (Provider, error) {
    switch providerName {
    case "aws":
        return aws.NewProvider()
    case "yourprovider":
        return yourprovider.NewProvider()
    // ... other providers
    }
}
```

### 4. Add Tests

Create `pkg/providers/yourprovider/provider_test.go`:

```go
package yourprovider

import (
    "context"
    "testing"
)

func TestDeploy(t *testing.T) {
    // Test deployment logic
}

func TestDestroy(t *testing.T) {
    // Test destroy logic
}

func TestStatus(t *testing.T) {
    // Test status logic
}
```

### 5. Add Example Manifest

Create `examples/yourprovider-example.yaml`:

```yaml
version: "1.0"

provider:
  name: yourprovider
  region: your-region

application:
  name: example-app
  description: "Example application"

# ... provider-specific configuration
```

### 6. Update Documentation

- Add provider to README.md
- Document provider-specific manifest fields
- Add authentication instructions

## Testing

### Unit Tests

```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./pkg/manifest

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Integration Tests

Integration tests require actual cloud credentials and deploy real resources. They are marked with build tags to prevent them from running in normal test runs.

**Integration tests are defined in:**
- `pkg/providers/aws/integration_test.go` - AWS Elastic Beanstalk tests
- `pkg/providers/gcp/integration_test.go` - Google Cloud Run tests

#### Running AWS Integration Tests

Set up AWS credentials:

```bash
export AWS_ACCESS_KEY_ID="your-access-key-id"
export AWS_SECRET_ACCESS_KEY="your-secret-access-key"
export AWS_REGION="us-east-1"  # Optional, defaults to us-east-1
```

Run the tests:

```bash
go test -tags=integration -v ./pkg/providers/aws
```

#### Running GCP Integration Tests

Set up GCP credentials:

```bash
export GCP_PROJECT_ID="your-project-id"
export GCP_BILLING_ACCOUNT_ID="XXXXXX-XXXXXX-XXXXXX"
export GCP_CREDENTIALS='{"type":"service_account","project_id":"..."}'
```

Run the tests:

```bash
go test -tags=integration -v ./pkg/providers/gcp
```

#### Running All Integration Tests

```bash
# Set up both AWS and GCP credentials, then:
go test -tags=integration -v ./...
```

**Important Notes:**
- Integration tests deploy real resources and may incur costs
- Tests automatically clean up resources after completion
- Tests are skipped if credentials are not available
- Integration tests take longer to run (10-30 minutes)

### Manual Testing

1. Create a test manifest
2. Build the binary: `go build -o cloud-deploy cmd/cloud-deploy/main.go`
3. Run: `./cloud-deploy -command deploy -manifest test-manifest.yaml`
4. Verify deployment
5. Clean up: `./cloud-deploy -command destroy -manifest test-manifest.yaml`

## Documentation

### Code Documentation

- Add Godoc comments to all exported types and functions
- Include examples in comments where helpful
- Keep comments up-to-date with code changes

### User Documentation

- Update README.md for user-facing changes
- Add examples for new features
- Update manifest reference for new fields

### Generate Documentation

```bash
# Generate godoc
godoc -http=:6060

# View at http://localhost:6060/pkg/github.com/jvreagan/cloud-deploy/
```

## Pull Request Process

1. **Update your fork**
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Make your changes**
   - Write code
   - Add tests
   - Update documentation

3. **Test your changes**
   ```bash
   go test ./...
   go fmt ./...
   ```

4. **Commit your changes**
   ```bash
   git add .
   git commit -m "feat: add your feature"
   ```

5. **Push to your fork**
   ```bash
   git push origin feature/your-feature-name
   ```

6. **Create Pull Request**
   - Go to GitHub and create a PR
   - Fill out the PR template
   - Link any related issues

### PR Requirements

- ✅ Tests pass (`go test ./...`)
- ✅ Code is formatted (`go fmt`)
- ✅ Documentation is updated
- ✅ Commit messages are clear
- ✅ No merge conflicts with main

### PR Review Process

1. Automated checks run (tests, linting)
2. Maintainer reviews code
3. Address feedback
4. Maintainer approves and merges

## Questions?

- Open an issue for bugs or feature requests
- Tag issues with appropriate labels
- Be specific and provide examples

## License

By contributing to cloud-deploy, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to cloud-deploy! 🎉
