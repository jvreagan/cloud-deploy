# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**cloud-deploy** is a manifest-driven multi-cloud deployment tool that simplifies deploying containerized applications to AWS, GCP, Azure, and OCI using a single declarative YAML manifest. The tool provides a unified interface across cloud providers, automatic resource creation, and features like one-command rollback, HashiCorp Vault integration, and a web UI for manifest generation.

## Build and Test Commands

### Building

```bash
# Build main CLI tool
go build -o cloud-deploy ./cmd/cloud-deploy

# Build manifest UI server
go build -o manifest-ui ./cmd/manifest-ui

# Build both
go build -v ./...
```

### Testing

```bash
# Run unit tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with race detection and coverage
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

# View coverage report
go tool cover -func=coverage.out
go tool cover -html=coverage.out

# Run tests for a specific package
go test ./pkg/manifest
go test ./pkg/providers/aws

# Run integration tests (requires cloud credentials)
go test -tags=integration -v ./pkg/providers/aws
go test -tags=integration -v ./pkg/providers/gcp
go test -tags=integration -v -timeout=30m ./...
```

### Linting and Formatting

```bash
# Format code
go fmt ./...

# Run go vet
go vet ./...

# Run staticcheck (install first: go install honnef.co/go/tools/cmd/staticcheck@latest)
staticcheck ./...
```

### Development Tools

```bash
# Install dependencies
go mod download

# Verify dependencies
go mod verify

# Tidy dependencies
go mod tidy

# Install git hooks for automatic formatting
./scripts/install-hooks.sh

# Run manifest UI locally
go run cmd/manifest-ui/main.go
# Then open http://localhost:5001
```

## Architecture

### Core Design Pattern: Provider Interface

The codebase uses a **provider abstraction pattern** where all cloud providers implement the same `Provider` interface defined in `pkg/provider/provider.go`. This allows the CLI to work with any cloud provider without knowing provider-specific details.

**Provider Interface Methods:**
- `Deploy(ctx, manifest) (*DeploymentResult, error)` - Deploy application
- `Destroy(ctx, manifest) error` - Remove deployment completely
- `Stop(ctx, manifest) error` - Stop environment but preserve application/versions
- `Status(ctx, manifest) (*DeploymentStatus, error)` - Get deployment status
- `Rollback(ctx, manifest) (*DeploymentResult, error)` - Rollback to previous version
- `Name() string` - Return provider name

### Component Architecture

```
User → CLI (cmd/cloud-deploy/main.go)
       ↓
       Manifest Parser (pkg/manifest/manifest.go)
       ↓
       Provider Factory (pkg/provider/provider.go)
       ↓
       Provider Implementation (pkg/providers/{aws,gcp,azure}/)
       ↓
       Cloud Provider APIs
```

### Key Packages

**pkg/manifest** - Manifest parsing and validation
- Loads YAML manifests
- Validates required fields
- Defines complete configuration structs

**pkg/provider** - Provider interface and factory
- Defines the `Provider` interface
- Factory pattern for creating providers based on manifest

**pkg/providers/{aws,gcp,azure}** - Provider implementations
- Each implements the Provider interface
- Handles provider-specific API calls and logic
- AWS: Elastic Beanstalk, S3
- GCP: Cloud Run, Cloud Build, Cloud Storage
- Azure: Container Instances, Container Registry

**pkg/registry** - Container registry abstraction (API-based, no CLI dependencies)
- Distributes Docker images to cloud registries (ECR, GCR, ACR)
- Uses go-containerregistry library for OCI Distribution Spec compliance
- Loads images from Docker daemon once, distributes to multiple registries
- Zero CLI dependencies (no docker login, docker tag, docker push, or gcloud auth)

**pkg/vault** - HashiCorp Vault integration
- Fetches secrets from Vault
- Supports multiple auth methods (token, AppRole, AWS IAM, GCP IAM)
- Injects secrets as environment variables

**pkg/types** - Shared types
- DeploymentResult, DeploymentStatus
- Common structs used across providers

**pkg/credentials** - Credential management
- Handles cloud provider authentication

**cmd/cloud-deploy** - Main CLI entry point
- Parses flags
- Loads manifest
- Creates provider
- Executes commands

**cmd/manifest-ui** - Web UI server
- Serves web interface on port 5001
- Generates manifest YAML files from form input
- Saves to generated-manifests/ directory

### Data Flow: Deploy Command

1. CLI parses command-line flags and loads manifest
2. Manifest parser validates YAML and creates Manifest struct
3. Provider factory creates appropriate provider (AWS/GCP/Azure)
4. Provider.Deploy() executes:
   - Check if application exists (create if not)
   - Check if environment exists (create if not)
   - Package source code or tag Docker image
   - Upload to cloud storage / push to registry
   - Create application version
   - Deploy to environment
   - Wait for deployment to complete
   - Return deployment result with URL
5. CLI displays result to user

## Adding a New Cloud Provider

1. **Create provider package:** `pkg/providers/newprovider/`
2. **Implement Provider interface:** See `pkg/provider/provider.go`
3. **Register in factory:** Update `provider.Factory()` in `pkg/provider/provider.go`
4. **Add tests:** Create `pkg/providers/newprovider/newprovider_test.go`
5. **Add integration tests:** Create `pkg/providers/newprovider/integration_test.go` with `//go:build integration` tag
6. **Add example manifest:** Create `examples/newprovider-example.yaml`
7. **Update documentation:** Add provider to README.md and create docs guide

## Integration Tests

Integration tests deploy real resources to cloud providers and are marked with build tag `integration` to prevent running in normal CI.

**Requirements:**
- Cloud provider credentials set as environment variables
- Tests will create and destroy real cloud resources
- May incur cloud provider costs
- Tests are skipped if credentials are not available

**AWS integration tests:**
```bash
export AWS_ACCESS_KEY_ID="..."
export AWS_SECRET_ACCESS_KEY="..."
export AWS_REGION="us-east-1"
go test -tags=integration -v ./pkg/providers/aws
```

**GCP integration tests:**
```bash
export GCP_PROJECT_ID="..."
export GCP_BILLING_ACCOUNT_ID="..."
export GCP_CREDENTIALS='{"type":"service_account",...}'
go test -tags=integration -v ./pkg/providers/gcp
```

## Release Process

Releases are automated via GoReleaser and GitHub Actions:

1. Push a git tag: `git tag v0.1.0 && git push origin v0.1.0`
2. GitHub Actions runs release workflow
3. GoReleaser builds binaries for multiple platforms (Linux, macOS, Windows)
4. Binaries uploaded to GitHub Releases
5. Homebrew formula automatically updated in homebrew-tap repository

**Manual release:**
```bash
goreleaser release --clean
```

## Important Design Principles

**Idempotency** - All operations are designed to be safe to run multiple times. Creating an application that already exists should succeed without error.

**Fail Fast** - Validate manifest and check for errors before making any API calls to avoid partial deployments.

**Provider Abstraction** - Core CLI should never contain provider-specific logic. All provider details must be encapsulated in provider implementations.

**Declarative Configuration** - Manifests describe **what** to deploy, not **how**. Provider implementations handle the **how**.

## CRITICAL: HTTPS/SSL Configuration Requirements

### End-to-End HTTPS for HTTPS-by-Design Applications

**NEVER assume SSL termination means forwarding to HTTP on port 80.**

Some applications (like helloworld3) are **HTTPS-by-design** and serve HTTPS traffic on port 443 at the origin. When deploying these applications with ACM certificates:

**CORRECT Configuration:**
- **Client → ELB:** HTTPS on port 443 using ACM certificate (eliminates browser warnings)
- **ELB → Instance:** HTTPS on port 443 (respects application architecture)
- Instance uses self-signed certificate (client never sees it, ACM cert shields it)

**WRONG Configuration (DO NOT DO THIS):**
- Client → ELB: HTTPS with ACM
- ELB → Instance: HTTP on port 80 ← WRONG! Application doesn't listen on port 80!

### AWS Elastic Beanstalk: HTTPS Backend Configuration

When configuring ELB listeners in `pkg/providers/aws/aws.go`:

```go
if port.ContainerPort == 443 {
    // Check if ACM certificate is configured
    if m.SSL != nil && m.SSL.CertificateArn != "" {
        // End-to-end HTTPS: ACM at ELB, self-signed at origin
        protocol = "HTTPS"
        instanceProtocol = "HTTPS"  // NOT "HTTP"!
        sslCertificateId = m.SSL.CertificateArn
        instancePort = 443  // NOT 80!
    } else {
        // Fall back to TCP passthrough for self-signed certs
        protocol = "TCP"
        instanceProtocol = "TCP"
    }
}
```

### Why This Matters

1. **Application Architecture:** HTTPS-by-design applications only listen on port 443 with TLS
2. **Security:** End-to-end encryption from client through to application
3. **ACM Purpose:** ACM certificate at ELB eliminates browser warnings (trusted CA)
4. **Backend Certificate:** Instance's self-signed cert is fine - client never sees it

### Implementation Notes

When `InstanceProtocol: HTTPS` is used with ELB, the load balancer will attempt to validate the backend SSL certificate. If using self-signed certificates at the origin:

**Option 1: Backend Authentication Policy (PREFERRED)**
Configure ELB to not validate backend certificates or use a custom CA policy.

**Option 2: TCP Passthrough (FALLBACK)**
Use `TCP` protocol on port 443, but this prevents using ACM certificate at ELB.

**Option 3: Valid Backend Certificate**
Install a valid certificate on the backend instance (adds complexity).

The goal is Option 1: HTTPS/HTTPS with ACM at frontend and relaxed backend validation.

## Vault Credential Storage Feature

cloud-deploy supports storing cloud provider credentials in HashiCorp Vault, enabling true multi-cloud credential management.

### Credential Sources

Credentials can be loaded from multiple sources (specified in `provider.credentials.source`):

1. **`vault`** - Fetch from HashiCorp Vault (NEW FEATURE)
2. **`environment`** - Load from environment variables
3. **`manifest`** - Use credentials specified in manifest (not recommended)
4. **`cli`** - Use cloud provider CLI credentials (default)

### Vault Credential Paths

When `credentials.source: vault`, cloud-deploy fetches credentials from these paths:

```
secret/cloud-deploy/aws/credentials
  ├── access_key_id
  └── secret_access_key

secret/cloud-deploy/gcp/credentials
  ├── project_id
  └── service_account_key

secret/cloud-deploy/azure/credentials
  ├── subscription_id
  ├── client_id
  ├── client_secret
  └── tenant_id
```

### Migration Script

Use `scripts/migrate-credentials-to-vault.sh` to migrate existing credentials to Vault.

### Example Manifest with Vault Credentials

```yaml
vault:
  address: "http://127.0.0.1:8200"
  auth:
    method: token
    token: "${VAULT_TOKEN}"

provider:
  name: aws
  region: us-east-2
  credentials:
    source: vault  # Load AWS credentials from Vault!
```

### Implementation Details

**Key files for Vault credential support:**
- `pkg/credentials/credentials.go` - Credential manager with Vault integration
- `pkg/manifest/manifest.go` - `GetCloudCredentials()` method
- `pkg/providers/aws/aws.go` - AWS provider with Vault support
- `pkg/vault/vault.go` - Vault client

## Common Development Tasks

When modifying provider implementations, ensure you:
- Maintain the Provider interface contract
- Update both unit tests and integration tests
- Handle errors gracefully with meaningful messages
- Support idempotent operations
- Add polling/waiting logic for async operations
- Return structured results (DeploymentResult, DeploymentStatus)

When adding new manifest fields:
- Update `pkg/manifest/manifest.go` structs
- Add YAML tags and comments
- Update validation logic if field is required
- Add to example manifests in `examples/`
- Document in relevant provider guide (docs/AWS.md, docs/GCP.md)

When adding Vault credential support to a new provider:
- Update `pkg/credentials/credentials.go` - Add provider case in `getFromVault()`
- Update provider's `New()` function to accept manifest parameter
- Add credential source check and Vault loading logic
- Update provider factory in `pkg/provider/provider.go`
- Test with `credentials.source: vault` in manifest

## Container Registry Architecture (No CLI Dependencies)

cloud-deploy eliminates all CLI dependencies for container registry operations by using the `go-containerregistry` library to interact directly with registry APIs.

### Design Pattern: Distributor

The registry package uses a **Distributor pattern** that loads a Docker image from the local daemon once and efficiently distributes it to multiple cloud registries using API calls.

**Old approach (CLI-based):**
```bash
# Required docker and gcloud CLI tools installed
docker login <registry>
docker tag <source> <target>
docker push <target>
```

**New approach (API-based):**
```go
// No CLI dependencies - uses go-containerregistry
distributor := registry.NewDistributor(sourceImage)
distributor.AddRegistry(ecrRegistry)
distributor.AddRegistry(gcrRegistry)
imageURIs, err := distributor.Distribute(ctx)
```

### Registry Interface

Each cloud registry (ECR, GCR, ACR) implements the `Registry` interface:

```go
type Registry interface {
    GetRegistryURL() string
    GetAuthenticator(ctx context.Context) (authn.Authenticator, error)
    GetImageReference() string
    GetImageURI() string
}
```

### Authentication Methods (API-based)

**AWS ECR:**
- Uses AWS SDK `GetAuthorizationToken` API
- Decodes base64 token to get username/password
- Returns `authn.Basic` authenticator

**Azure ACR:**
- Uses Azure SDK `ListCredentials` API
- Gets admin username and password
- Returns `authn.Basic` authenticator

**GCP Artifact Registry:**
- Uses OAuth2 token from service account credentials
- Generates access token via `google.CredentialsFromJSON`
- Returns `authn.Basic` with username "oauth2accesstoken"

### Image Distribution Flow

1. **Load Once:** Image loaded from Docker daemon via `daemon.Image()`
2. **Authenticate:** Each registry provides authenticator via `GetAuthenticator()`
3. **Push:** Image pushed to each registry via `remote.Write()` with OCI Distribution API
4. **Return URIs:** Map of registry URLs to pushed image URIs returned

### Benefits

- **Zero CLI dependencies** - Only Docker daemon required (for reading images)
- **Modern & maintainable** - Uses official cloud SDKs and OCI standards
- **Better for OSS adoption** - Fewer external dependencies
- **More efficient** - Load image once, push to multiple registries
- **Cross-platform** - Works anywhere Go runs (no shell commands)

### Key Files

- `pkg/registry/registry.go` - Distributor and Registry interface
- `pkg/registry/ecr.go` - AWS ECR implementation
- `pkg/registry/acr.go` - Azure ACR implementation
- `pkg/registry/gcr.go` - GCP Artifact Registry implementation

### Dependencies

- `github.com/google/go-containerregistry` - OCI registry operations
- Cloud provider SDKs (AWS, Azure, GCP) - Authentication only
