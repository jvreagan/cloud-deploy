# cloud-deploy

[![CI](https://github.com/jvreagan/cloud-deploy/actions/workflows/ci.yml/badge.svg)](https://github.com/jvreagan/cloud-deploy/actions/workflows/ci.yml)
[![Release](https://github.com/jvreagan/cloud-deploy/actions/workflows/release.yml/badge.svg)](https://github.com/jvreagan/cloud-deploy/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jvreagan/cloud-deploy)](https://goreportcard.com/report/github.com/jvreagan/cloud-deploy)

**Manifest-driven multi-cloud deployment tool**

`cloud-deploy` is a command-line tool that simplifies deploying containerized applications to multiple cloud providers using a single declarative manifest file. Think of it as "docker-compose for cloud deployments."

## Quick Install

```bash
brew tap jvreagan/tap
brew install cloud-deploy
```

> **Note:** Homebrew works on macOS and Linux. See [Installation](#installation) section for other options.

## Features

- 📝 **Declarative Configuration** - Define your entire deployment in a single YAML manifest
- 🌐 **Web UI** - Generate manifests with a simple web interface
- ☁️ **Multi-Cloud Support** - Deploy to AWS, GCP, Azure, and OCI with the same tool
- 🚀 **Fully Automated** - Creates applications, environments, and deploys via cloud APIs - no console needed
- 🔄 **Idempotent** - Run the same command repeatedly safely
- 📦 **Docker Support** - Native support for containerized applications
- 📊 **Built-in Monitoring** - CloudWatch metrics, enhanced health reporting, and log streaming (AWS)

## Status

🚀 **Active Development** - AWS and GCP providers implemented

**Supported Providers:**
- [x] AWS Elastic Beanstalk
- [x] Google Cloud Run
- [ ] Azure Container Instances
- [ ] Oracle Cloud Container Instances

## Installation

### Homebrew (macOS and Linux) - Recommended

The easiest way to install `cloud-deploy` is via Homebrew. This works on both macOS and Linux.

```bash
# Step 1: Add the cloud-deploy tap
brew tap jvreagan/tap

# Step 2: Install cloud-deploy (includes both CLI and Web UI)
brew install cloud-deploy

# Step 3: Verify installation
cloud-deploy -version
manifest-ui -h
```

**What gets installed:**
- `cloud-deploy` - Main CLI tool for deploying to cloud providers
- `manifest-ui` - Web UI server for generating manifest files
- Web assets - Frontend files for the manifest generator UI

**Installation locations:**
- Apple Silicon Macs: `/opt/homebrew/bin/`
- Intel Macs: `/usr/local/bin/`
- Linux: `/home/linuxbrew/.linuxbrew/bin/`

All binaries are pre-built and ready to use - no compilation needed!

### From Source

#### CLI Tool

```bash
# Clone the repository
git clone https://github.com/jvreagan/cloud-deploy
cd cloud-deploy

# Build from source
go build -o cloud-deploy cmd/cloud-deploy/main.go

# Or install directly
go install github.com/jvreagan/cloud-deploy/cmd/cloud-deploy@latest
```

#### Web UI (Manifest Generator)

No build required! Run directly from source:

```bash
# After cloning the repository
cd cloud-deploy
go run cmd/manifest-ui/main.go
```

Then open your browser to **http://localhost:5001** to use the visual manifest generator.

See the [Web UI section](#web-ui---manifest-generator) below for more details.

## Quick Start

**First, install cloud-deploy:** (if you haven't already)
```bash
brew tap jvreagan/tap
brew install cloud-deploy
```

### Option 1: Using the Web UI (Recommended for beginners)

```bash
# Start the manifest generator
manifest-ui
```

Open http://localhost:5001 in your browser, fill out the form, and generate your manifest!

### Option 2: Manual YAML Creation

1. Create a deployment manifest (`deploy-manifest.yaml`):

```yaml
version: "1.0"

provider:
  name: aws
  region: us-east-2

application:
  name: my-app
  description: "My Application"
  
environment:
  name: my-app-env
  cname: my-app  # Creates: my-app.us-east-2.elasticbeanstalk.com
  
deployment:
  platform: docker
  solution_stack: "64bit Amazon Linux 2023 v4.7.2 running Docker"
  source:
    type: local
    path: "."
  
instance:
  type: t3.micro
  environment_type: SingleInstance
  
health_check:
  type: enhanced
  path: /health

# Optional: Enable CloudWatch monitoring
monitoring:
  enhanced_health: true
  cloudwatch_metrics: true
```

2. Deploy your application:

```bash
# If you created the manifest manually
cloud-deploy -command deploy -manifest deploy-manifest.yaml

# Or if you used the Web UI
cloud-deploy -command deploy -manifest generated-manifests/aws-manifest-20241029-123456.yaml
```

3. Check deployment status:

```bash
cloud-deploy -command status -manifest deploy-manifest.yaml
```

4. Stop when not in use (preserves application for fast restart):

```bash
cloud-deploy -command stop -manifest deploy-manifest.yaml
```

5. Destroy completely when done:

```bash
cloud-deploy -command destroy -manifest deploy-manifest.yaml
```

## Web UI - Manifest Generator

Prefer a visual interface? Use the built-in web UI to generate manifests without writing YAML!

### Start the Web UI

```bash
go run cmd/manifest-ui/main.go
```

Then open your browser to: **http://localhost:5001**

### Features

- ✨ Interactive form-based interface
- 🔄 Dynamic fields based on provider selection (AWS or GCP)
- ✅ Built-in validation for required fields
- 📋 Help text and examples for all options
- 💾 Auto-saves manifests to `generated-manifests/` directory
- 🧪 Fully tested with comprehensive test suite

### Using the Web UI

1. **Select provider** - Choose AWS or GCP
2. **Fill in the form** - Required fields are marked with *
3. **Generate manifest** - Click the button to create your YAML file
4. **Deploy** - Use the generated manifest with cloud-deploy:
   ```bash
   cloud-deploy -command deploy -manifest generated-manifests/aws-manifest-20241029-123456.yaml
   ```

See [cmd/manifest-ui/README.md](cmd/manifest-ui/README.md) for detailed documentation.

## Manifest Reference

**Quick Reference**: See [deploy-manifest.example.yaml](deploy-manifest.example.yaml) for a complete example with all available options.

**Detailed Documentation**:
- **AWS**: See the [AWS Deployment Guide](docs/AWS.md) for complete field documentation
- **GCP**: See the [GCP Deployment Guide](docs/GCP.md) for complete field documentation

### Required Fields

- `provider.name` - Cloud provider (aws, gcp, azure, oci)
- `provider.region` - Deployment region
- `application.name` - Application name
- `environment.name` - Environment name

### Optional Fields

- `application.description` - Application description
- `environment.cname` - Custom subdomain
- `deployment.platform` - Platform type (default: docker)
- `deployment.source.type` - Source type (local, s3, git)
- `instance.type` - Instance type (e.g., t3.micro)
- `instance.environment_type` - SingleInstance or LoadBalanced
- `health_check.type` - Health check type (basic or enhanced)
- `health_check.path` - Health check endpoint
- `monitoring.enhanced_health` - Enable enhanced health reporting (AWS)
- `monitoring.cloudwatch_metrics` - Enable CloudWatch metrics (AWS)
- `monitoring.cloudwatch_logs` - CloudWatch logs configuration (AWS)
- `iam.instance_profile` - IAM instance profile to use
- `environment_variables` - Environment variables map
- `tags` - Resource tags map

### GCP-Specific Fields

When using `provider.name: gcp`, these additional fields are required or available:

**Required:**
- `provider.project_id` - Your GCP project ID (cloud-deploy will create if it doesn't exist)
- `provider.billing_account_id` - Your GCP billing account ID (format: XXXXXX-XXXXXX-XXXXXX)
- `provider.credentials.service_account_key_path` - Path to service account JSON key file
  - OR `provider.credentials.service_account_key_json` - Service account JSON content

**Optional:**
- `provider.public_access` - Make Cloud Run service publicly accessible (default: true)
- `provider.organization_id` - Organization ID (if creating project under an organization)
- `cloud_run.*` - Cloud Run resource configuration (CPU, memory, scaling, timeout)
  - See [GCP Guide](docs/GCP.md#cloud-run-configuration) for complete configuration options
- `monitoring.cloudwatch_logs` - Cloud Logging configuration (same format as AWS)

**📖 Complete Field Reference**: [GCP Deployment Guide - Required & Optional Fields](docs/GCP.md#required-fields)

## Quick Start: Authentication

> 📖 **For detailed authentication guides, see:**
> - [AWS Authentication Guide](docs/AWS.md#authentication)
> - [GCP Authentication Guide](docs/GCP.md#authentication)

### AWS

Use AWS credentials for Elastic Beanstalk deployments. Credentials can be provided in three ways (in order of preference):

1. **AWS CLI credentials** (recommended)
   ```bash
   aws configure
   ```

2. **Environment variables**
   ```bash
   export AWS_ACCESS_KEY_ID=your_access_key
   export AWS_SECRET_ACCESS_KEY=your_secret_key
   ```

3. **Manifest file** (works but not recommended for production)
   ```yaml
   provider:
     name: aws
     credentials:
       access_key_id: YOUR_AWS_ACCESS_KEY_ID
       secret_access_key: YOUR_AWS_SECRET_ACCESS_KEY
   ```

   See `examples/aws-with-credentials.yaml` for a complete example.

   **Security Warning**: Never commit files with real credentials to version control!

### GCP

**No gcloud CLI required!** cloud-deploy is completely self-sufficient for GCP deployments.

> 📖 **For complete GCP setup instructions with screenshots and detailed steps, see the [GCP Deployment Guide](docs/GCP.md#authentication)**

#### Step 1: Create a Service Account

1. Go to [GCP Service Accounts](https://console.cloud.google.com/iam-admin/serviceaccounts)
2. Click **Create Service Account**
3. Give it a name (e.g., "cloud-deploy-admin")
4. Grant these roles:
   - **Owner** (simplest, gives all permissions)
   - OR for more granular control:
     - **Project Creator** (to create projects)
     - **Billing Account User** (to link billing)
     - **Service Usage Admin** (to enable APIs)
     - **Cloud Run Admin** (to deploy services)
     - **Cloud Build Editor** (to build images)
     - **Storage Admin** (to upload source code)
5. Click **Done**, then click the service account
6. Go to **Keys** tab → **Add Key** → **Create new key** → **JSON**
7. Download the JSON key file and save it securely

#### Step 2: Get Your Billing Account ID

1. Go to [GCP Billing](https://console.cloud.google.com/billing)
2. Copy your billing account ID (format: `XXXXXX-XXXXXX-XXXXXX`)

#### Step 3: Configure Manifest

That's it! Everything else is in the manifest:

```yaml
provider:
  name: gcp
  region: us-central1

  # cloud-deploy will create this project automatically
  project_id: my-new-project

  # Your billing account ID
  billing_account_id: XXXXXX-XXXXXX-XXXXXX

  # Path to the service account key you downloaded
  credentials:
    service_account_key_path: "/path/to/service-account-key.json"

  # Optional: Make service public (default: true)
  public_access: true
```

#### What cloud-deploy Does Automatically

When you run `cloud-deploy -command deploy`:

1. ✅ Creates the GCP project (if it doesn't exist) with polling until complete
2. ✅ Links your billing account
3. ✅ Enables all required APIs (Cloud Build, Cloud Run, Storage, etc.) with polling
4. ✅ Builds your Docker image using Cloud Build
5. ✅ Deploys to Cloud Run with resource limits and scaling configuration
6. ✅ Configures public access
7. ✅ Sets up Cloud Logging (if enabled)
8. ✅ Waits for service to be ready before returning

## Commands

- **deploy** - Create or update a deployment
- **stop** - Stop the environment/service but preserve the application and versions for fast restart
- **destroy** - Remove a deployment completely (application, environment, and versions)
- **status** - Check deployment status

## Why cloud-deploy?

**Problem:** Each cloud provider has different tools and workflows for deployment:
- AWS: EB CLI, CloudFormation, or console
- GCP: gcloud, Cloud Console
- Azure: az CLI, ARM templates
- Multiple commands required even for simple deployments

**Solution:** One tool, one manifest, any cloud

```bash
# Same command works for AWS, GCP, Azure, OCI
cloud-deploy -command deploy
```

> 📖 **For detailed comparison with other tools and feature breakdown, see the [Features Guide](docs/FEATURES.md)**

## Comparison

| Tool | Multi-Cloud | Declarative | Docker Support | Learning Curve |
|------|-------------|-------------|----------------|----------------|
| cloud-deploy | ✅ | ✅ | ✅ | Low |
| Terraform | ✅ | ✅ | ⚠️ | High |
| EB CLI | ❌ (AWS only) | ❌ | ✅ | Medium |
| gcloud | ❌ (GCP only) | ⚠️ | ✅ | Medium |

## Development

### Quick Start

```bash
# Clone the repository
git clone https://github.com/jvreagan/cloud-deploy.git
cd cloud-deploy

# Install dependencies
go mod download

# Install git hooks (optional but recommended)
./scripts/install-hooks.sh

# Run tests
go test ./...

# Build
go build -o cloud-deploy cmd/cloud-deploy/main.go

# Run locally
./cloud-deploy -command deploy -manifest examples/aws-simple.yaml
```

### Testing

cloud-deploy has comprehensive test coverage including unit tests, integration tests, and CI automation.

**Run unit tests:**
```bash
go test ./...                          # Run all unit tests
go test -v ./...                       # Verbose output
go test -cover ./...                   # With coverage
go test -race ./...                    # With race detection
```

**Run integration tests** (requires cloud credentials):
```bash
# AWS integration tests
export AWS_ACCESS_KEY_ID="your-key"
export AWS_SECRET_ACCESS_KEY="your-secret"
go test -tags=integration -v ./pkg/providers/aws

# GCP integration tests
export GCP_PROJECT_ID="your-project"
export GCP_CREDENTIALS='{"type":"service_account",...}'
go test -tags=integration -v ./pkg/providers/gcp
```

**Pre-commit hooks:** Install git hooks to automatically run tests before commits:
```bash
./scripts/install-hooks.sh
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed development guidelines.

## GitHub Actions Integration

> **Use cloud-deploy in YOUR application repositories** to deploy to AWS, GCP, and other clouds automatically via GitHub Actions.

Integrate cloud-deploy into your CI/CD pipeline with ready-to-use GitHub Actions workflows.

### How It Works

When using cloud-deploy with GitHub Actions:

1. **Install cloud-deploy** in your workflow (downloaded from releases)
2. **Run deployment commands** using your application's manifest file
3. **Deploy to AWS/GCP** automatically after builds/tests pass

```yaml
# Example: .github/workflows/deploy.yml in YOUR app repository
- name: Install cloud-deploy
  run: |
    curl -L https://github.com/jvreagan/cloud-deploy/releases/latest/download/cloud-deploy_Linux_x86_64.tar.gz | tar -xz
    sudo mv cloud-deploy /usr/local/bin/

- name: Deploy to AWS
  run: cloud-deploy -manifest manifests/production.yaml -command deploy
  env:
    AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
    AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
```

See [How It Works: Installing cloud-deploy in Your Workflows](docs/GITHUB_ACTIONS.md#how-it-works-installing-cloud-deploy-in-your-workflows) for complete details.

### Deployment Methods

| Method | Trigger | Best For |
|--------|---------|----------|
| **Manual Dispatch** | GitHub UI button | Production deployments, on-demand operations |
| **PR Labels** | Add `deploy:aws:staging` label | Review apps, testing changes |
| **Commit Message** | Include `[deploy]` in commit | Automatic staging/production |

### Quick Setup

**1. Copy workflows to your application repository:**
```bash
# In YOUR application repository (not cloud-deploy repo)
mkdir -p .github/workflows
cp cloud-deploy/.github/workflows/deploy.yml .github/workflows/
cp cloud-deploy/.github/workflows/manual-deploy.yml .github/workflows/
```

**2. Add cloud credentials as GitHub secrets:**
```
Settings → Secrets and variables → Actions
- AWS_ACCESS_KEY_ID
- AWS_SECRET_ACCESS_KEY
- GCP_PROJECT_ID
- GCP_CREDENTIALS
```

**3. Create deployment manifests:**
```bash
# In YOUR application repository
mkdir -p manifests
# Create manifests/staging-aws.yaml, manifests/production-gcp.yaml, etc.
```

**4. Deploy from GitHub UI:**
- Go to **Actions** tab
- Select **Manual Deployment**
- Choose provider, environment, and click **Run workflow**

### Features

✅ Manual deployments with UI controls
✅ Automatic PR-based deployments via labels
✅ Environment-based approvals for production
✅ Deployment logs as artifacts
✅ PR comments with deployment status
✅ Rollback capabilities
✅ Multi-cloud support (AWS + GCP)

**Complete documentation:** [docs/GITHUB_ACTIONS.md](docs/GITHUB_ACTIONS.md)

**Example workflows:** [examples/workflows/](examples/workflows/)

## Examples

The `examples/` directory contains several deployment manifests:

- `aws-simple.yaml` - Minimal AWS deployment (uses default credentials)
- `aws-loadbalanced.yaml` - AWS with load balancer configuration
- `aws-with-credentials.yaml` - AWS with explicit credentials in manifest
- `aws-monitoring.yaml` - AWS with enhanced health reporting and CloudWatch monitoring
- `gcp-example.yaml` - Google Cloud Run deployment
- `deploy-manifest.example.yaml` - Complete reference with all options

## Documentation

### 📚 Comprehensive Guides

- **[Features Overview](docs/FEATURES.md)** - Complete feature list, provider comparison, and what cloud-deploy offers
- **[Homebrew Installation](HOMEBREW.md)** - Complete guide for setting up Homebrew distribution
- **[AWS Deployment Guide](docs/AWS.md)** - Everything you need to deploy to AWS Elastic Beanstalk
  - Prerequisites and authentication
  - Required and optional fields
  - Monitoring and CloudWatch integration
  - Best practices and troubleshooting
  - Complete examples
- **[GCP Deployment Guide](docs/GCP.md)** - Complete guide for Google Cloud Run deployments
  - Self-sufficient setup (no gcloud CLI required)
  - Service account creation
  - Cloud Run configuration (CPU, memory, scaling)
  - Cloud Logging integration
  - Best practices and cost optimization
  - Troubleshooting guide
- **[Monitoring Guide](docs/MONITORING.md)** - CloudWatch metrics, enhanced health, and logging configuration

### 🎯 Quick Links

**Getting Started:**
1. Read [Features Overview](docs/FEATURES.md) to understand what cloud-deploy does
2. Choose your provider: [AWS Guide](docs/AWS.md) or [GCP Guide](docs/GCP.md)
3. Review [examples/](examples/) for reference manifests
4. Deploy your first application!

**Advanced Topics:**
- [Monitoring and Logging](docs/MONITORING.md)
- [Contributing Guide](CONTRIBUTING.md)

## Contributing

Contributions welcome! This project is in early development.

### Adding a New Provider

1. Implement the `Provider` interface in `pkg/provider/provider.go`
2. Add provider-specific logic in `pkg/providers/<provider-name>/`
3. Register in `provider.Factory()`
4. Add tests and documentation

## License

MIT License - see LICENSE file

## Roadmap

- [x] Project structure
- [x] Manifest schema
- [x] Provider interface
- [x] AWS Elastic Beanstalk provider
- [x] GCP Cloud Run provider
- [ ] Azure Container Instances provider
- [ ] OCI Container Instances provider
- [ ] CI/CD integration examples
- [ ] Comprehensive test coverage

## Related Projects

- [AWS EB CLI](https://docs.aws.amazon.com/elasticbeanstalk/latest/dg/eb-cli3.html) - AWS-specific deployment tool
- [gcloud CLI](https://cloud.google.com/sdk/gcloud) - GCP command-line tool
- [Terraform](https://www.terraform.io/) - Infrastructure as Code
- [Pulumi](https://www.pulumi.com/) - Modern IaC with programming languages

---

**Note:** This is an independent project and is not affiliated with or endorsed by AWS, Google, Microsoft, or Oracle.
