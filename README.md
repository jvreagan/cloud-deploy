# cloud-deploy

**Manifest-driven multi-cloud deployment tool**

`cloud-deploy` is a command-line tool that simplifies deploying containerized applications to multiple cloud providers using a single declarative manifest file. Think of it as "docker-compose for cloud deployments."

## Features

- 📝 **Declarative Configuration** - Define your entire deployment in a single YAML manifest
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

```bash
# Clone the repository
git clone https://github.com/jvreagan/cloud-deploy
cd cloud-deploy

# Build from source
go build -o cloud-deploy cmd/cloud-deploy/main.go

# Or install directly
go install github.com/jvreagan/cloud-deploy/cmd/cloud-deploy@latest
```

## Quick Start

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
cloud-deploy -command deploy -manifest deploy-manifest.yaml
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

## Manifest Reference

See [deploy-manifest.example.yaml](deploy-manifest.example.yaml) for a complete example with all available options.

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
- `cloud_run.cpu` - CPU allocation: "1", "2", "4" (default: "1")
- `cloud_run.memory` - Memory: "256Mi", "512Mi", "1Gi", "2Gi", "4Gi" (default: "512Mi")
- `cloud_run.max_concurrency` - Max concurrent requests per container (default: 80)
- `cloud_run.min_instances` - Minimum instances (0 = scale to zero, default: 0)
- `cloud_run.max_instances` - Maximum instances (default: 100)
- `cloud_run.timeout_seconds` - Request timeout in seconds (default: 300, max: 3600)
- `monitoring.cloudwatch_logs` - Cloud Logging configuration (same format as AWS)

## Authentication

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

## Comparison

| Tool | Multi-Cloud | Declarative | Docker Support | Learning Curve |
|------|-------------|-------------|----------------|----------------|
| cloud-deploy | ✅ | ✅ | ✅ | Low |
| Terraform | ✅ | ✅ | ⚠️ | High |
| EB CLI | ❌ (AWS only) | ❌ | ✅ | Medium |
| gcloud | ❌ (GCP only) | ⚠️ | ✅ | Medium |

## Development

```bash
# Run tests
go test ./...

# Build
go build -o cloud-deploy cmd/cloud-deploy/main.go

# Run locally
./cloud-deploy -command deploy -manifest examples/aws-simple.yaml
```

## Examples

The `examples/` directory contains several deployment manifests:

- `aws-simple.yaml` - Minimal AWS deployment (uses default credentials)
- `aws-loadbalanced.yaml` - AWS with load balancer configuration
- `aws-with-credentials.yaml` - AWS with explicit credentials in manifest
- `aws-monitoring.yaml` - AWS with enhanced health reporting and CloudWatch monitoring
- `gcp-example.yaml` - Google Cloud Run deployment
- `deploy-manifest.example.yaml` - Complete reference with all options

## Documentation

- [Monitoring Guide](docs/MONITORING.md) - CloudWatch metrics, enhanced health, and logging configuration
- [Architecture Overview](docs/ARCHITECTURE.md) - System design and implementation details
- [Contributing Guide](CONTRIBUTING.md) - How to contribute to the project

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
