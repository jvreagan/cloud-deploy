# cloud-deploy Architecture

This document describes the architecture and design of cloud-deploy.

## Table of Contents

- [Overview](#overview)
- [Design Principles](#design-principles)
- [System Architecture](#system-architecture)
- [Component Details](#component-details)
- [Data Flow](#data-flow)
- [Provider Interface](#provider-interface)
- [Adding New Providers](#adding-new-providers)

## Overview

cloud-deploy is a manifest-driven multi-cloud deployment tool that abstracts away the differences between cloud providers, allowing users to deploy applications using a consistent interface.

### Goals

1. **Simplicity** - One manifest, one command to deploy
2. **Consistency** - Same interface for all cloud providers
3. **Extensibility** - Easy to add new providers
4. **Automation** - Fully automated via cloud APIs, no manual steps

### Non-Goals

- Infrastructure as Code (use Terraform/Pulumi for that)
- Container orchestration (use Kubernetes for that)
- CI/CD pipelines (use GitHub Actions/GitLab CI for that)

## Design Principles

### 1. Provider Abstraction

Each cloud provider implements the same `Provider` interface, allowing the core application to work with any provider without knowing provider-specific details.

### 2. Declarative Configuration

The manifest file is declarative - it describes **what** to deploy, not **how** to deploy it. The provider implementations handle the **how**.

### 3. Idempotency

Running the same deployment command multiple times should be safe and produce the same result.

### 4. Fail Fast

Validate the manifest and check for errors before making any API calls.

## System Architecture

```
┌─────────────────────────────────────────────────────────┐
│                         User                             │
└────────────────────┬────────────────────────────────────┘
                     │
                     │ create manifest.yaml
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│                  cloud-deploy CLI                        │
│                (cmd/cloud-deploy/main.go)                │
└────────────────────┬────────────────────────────────────┘
                     │
                     │ parse & validate
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│                 Manifest Parser                          │
│              (pkg/manifest/manifest.go)                  │
│                                                           │
│  • Loads YAML file                                       │
│  • Parses into Go structs                                │
│  • Validates required fields                             │
└────────────────────┬────────────────────────────────────┘
                     │
                     │ get provider
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│                Provider Factory                          │
│              (pkg/provider/provider.go)                  │
│                                                           │
│  • Creates provider based on manifest.provider.name      │
└────────────────────┬────────────────────────────────────┘
                     │
      ┌──────────────┼──────────────┬──────────────┐
      │              │               │              │
      ▼              ▼               ▼              ▼
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│   AWS   │    │   GCP   │    │  Azure  │    │   OCI   │
│Provider │    │Provider │    │Provider │    │Provider │
└────┬────┘    └─────────┘    └─────────┘    └─────────┘
     │
     │ Deploy/Destroy/Status
     │
     ▼
┌─────────────────────────────────────────────────────────┐
│              Cloud Provider APIs                         │
│                                                           │
│  AWS: Elastic Beanstalk API                              │
│  GCP: Cloud Run API                                      │
│  Azure: Container Instances API                          │
│  OCI: Container Instances API                            │
└─────────────────────────────────────────────────────────┘
```

## Component Details

### 1. CLI (cmd/cloud-deploy/main.go)

**Responsibility:** Entry point for the application

**Functions:**
- Parse command-line flags
- Load manifest file
- Create provider
- Execute command (deploy/destroy/status)
- Display results to user

**Input:** Command-line arguments
**Output:** Console output (success/error messages)

### 2. Manifest Parser (pkg/manifest/)

**Responsibility:** Parse and validate manifest files

**Components:**
- `Manifest` struct - Root configuration object
- `Load()` function - Reads and parses YAML file
- `Validate()` function - Checks required fields

**Input:** YAML manifest file
**Output:** `Manifest` struct or validation error

### 3. Provider Interface (pkg/provider/)

**Responsibility:** Define the contract all providers must implement

**Methods:**
- `Name() string` - Provider identifier
- `Deploy(ctx, manifest) (*DeploymentResult, error)` - Deploy application
- `Destroy(ctx, manifest) error` - Remove deployment
- `Status(ctx, manifest) (*DeploymentStatus, error)` - Get deployment status

**Factory Pattern:**
```go
provider, err := provider.Factory("aws")
```

### 4. Provider Implementations (pkg/providers/*)

**Responsibility:** Implement provider-specific deployment logic

**Example: AWS Provider**

```
pkg/providers/aws/
├── provider.go        # Main provider implementation
├── application.go     # Application management
├── environment.go     # Environment management
├── deployer.go        # Deployment logic
└── provider_test.go   # Tests
```

**Key responsibilities:**
- Create application if not exists
- Create environment if not exists
- Package source code
- Upload to cloud storage (S3, GCS, etc.)
- Create application version
- Deploy to environment
- Wait for deployment to complete
- Return deployment URL

## Data Flow

### Deploy Command

```
1. User runs: cloud-deploy -command deploy -manifest app.yaml

2. CLI loads and parses manifest
   manifest.Load("app.yaml")
   
3. CLI creates provider
   provider := provider.Factory(manifest.Provider.Name)
   
4. Provider.Deploy() is called
   
5. Provider implementation:
   a. Check if application exists
      - If not, create it
      
   b. Check if environment exists
      - If not, create it
      
   c. Package source code
      - Create ZIP/tarball of source directory
      
   d. Upload to cloud storage
      - AWS: S3 bucket
      - GCP: Cloud Storage
      - Azure: Blob Storage
      
   e. Create application version
      - Reference uploaded package
      
   f. Deploy to environment
      - Update environment with new version
      
   g. Wait for deployment
      - Poll until status is "Ready"
      
   h. Return result
      - Application name
      - Environment name
      - Public URL
      - Status

6. CLI displays result to user
```

### Destroy Command

```
1. User runs: cloud-deploy -command destroy -manifest app.yaml

2. CLI loads and parses manifest

3. Provider.Destroy() is called

4. Provider implementation:
   a. Terminate environment
      - Wait for termination to complete
      
   b. Delete application versions
      
   c. Delete application
      
   d. Clean up cloud storage

5. CLI confirms destruction
```

### Status Command

```
1. User runs: cloud-deploy -command status -manifest app.yaml

2. CLI loads and parses manifest

3. Provider.Status() is called

4. Provider queries cloud API

5. Returns:
   - Application status
   - Environment status
   - Health (Green/Yellow/Red)
   - Public URL
   - Last updated timestamp

6. CLI displays status
```

## Provider Interface

### Why an Interface?

The `Provider` interface allows:
1. **Polymorphism** - CLI works with any provider
2. **Testability** - Easy to mock providers for testing
3. **Extensibility** - Add new providers without changing core code

### Provider Lifecycle

```go
// 1. Create provider
provider := aws.NewProvider()

// 2. Deploy
result, err := provider.Deploy(ctx, manifest)

// 3. Check status
status, err := provider.Status(ctx, manifest)

// 4. Destroy
err := provider.Destroy(ctx, manifest)
```

## Adding New Providers

See [CONTRIBUTING.md](../CONTRIBUTING.md#adding-a-new-cloud-provider) for detailed instructions.

**Quick overview:**

1. Create `pkg/providers/yourprovider/provider.go`
2. Implement the `Provider` interface
3. Register in `provider.Factory()`
4. Add tests
5. Add example manifest
6. Update documentation

## Error Handling

### Validation Errors

Caught before any API calls:
- Missing required fields
- Invalid configuration values
- Unsupported provider

### API Errors

Handled gracefully:
- Authentication failures
- Permission denied
- Resource already exists (idempotent)
- Resource not found
- Quota exceeded

### Retry Logic

Providers should implement retry logic for:
- Transient network errors
- Rate limiting
- Service unavailable

## Future Enhancements

### Planned Features

1. **State Management** - Track deployments across runs
2. **Rollback** - Revert to previous version
3. **Blue/Green Deployments** - Zero-downtime updates
4. **Secrets Management** - Integrate with vault/secrets manager
5. **CI/CD Integration** - GitHub Actions, GitLab CI examples
6. **Dry Run Mode** - Preview changes without applying

### Provider Roadmap

- [x] AWS Elastic Beanstalk (In Progress)
- [ ] Google Cloud Run
- [ ] Azure Container Instances
- [ ] Oracle Cloud Container Instances
- [ ] DigitalOcean App Platform
- [ ] Heroku

## References

- [Provider Interface](../pkg/provider/provider.go)
- [Manifest Schema](../pkg/manifest/manifest.go)
- [Contributing Guide](../CONTRIBUTING.md)

---

Last updated: 2025-10-27
