# cloud-deploy

**Manifest-driven multi-cloud deployment tool**

`cloud-deploy` is a command-line tool that simplifies deploying containerized applications to multiple cloud providers using a single declarative manifest file. Think of it as "docker-compose for cloud deployments."

## Features

- 📝 **Declarative Configuration** - Define your entire deployment in a single YAML manifest
- ☁️ **Multi-Cloud Support** - Deploy to AWS, GCP, Azure, and OCI with the same tool
- 🚀 **Fully Automated** - Creates applications, environments, and deploys via cloud APIs - no console needed
- 🔄 **Idempotent** - Run the same command repeatedly safely
- 📦 **Docker Support** - Native support for containerized applications

## Status

🚧 **Early Development** - Currently implementing AWS Elastic Beanstalk support

**Supported Providers:**
- [ ] AWS Elastic Beanstalk (In Progress)
- [ ] Google Cloud Run
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
  type: basic
  path: /health
```

2. Deploy your application:

```bash
cloud-deploy -command deploy -manifest deploy-manifest.yaml
```

3. Check deployment status:

```bash
cloud-deploy -command status -manifest deploy-manifest.yaml
```

4. Destroy when done:

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
- `health_check.path` - Health check endpoint
- `iam.instance_profile` - IAM instance profile to use
- `environment_variables` - Environment variables map
- `tags` - Resource tags map

## Authentication

### AWS

Credentials can be provided in three ways (in order of preference):

1. **AWS CLI credentials** (recommended)
   ```bash
   aws configure
   ```

2. **Environment variables**
   ```bash
   export AWS_ACCESS_KEY_ID=your_access_key
   export AWS_SECRET_ACCESS_KEY=your_secret_key
   ```

3. **Manifest file** (not recommended for production)
   ```yaml
   provider:
     name: aws
     credentials:
       access_key_id: YOUR_KEY
       secret_access_key: YOUR_SECRET
   ```

## Commands

- **deploy** - Create or update a deployment
- **destroy** - Remove a deployment
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
| EB CLI | ❌ | ❌ | ✅ | Medium |
| gcloud | ❌ | ⚠️ | ✅ | Medium |

## Development

```bash
# Run tests
go test ./...

# Build
go build -o cloud-deploy cmd/cloud-deploy/main.go

# Run locally
./cloud-deploy -command deploy -manifest examples/aws-example.yaml
```

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
- [ ] AWS Elastic Beanstalk provider
- [ ] GCP Cloud Run provider
- [ ] Azure Container Instances provider
- [ ] OCI Container Instances provider
- [ ] CI/CD integration examples
- [ ] Comprehensive test coverage

## Related Projects

- [AWS EB CLI](https://docs.aws.amazon.com/elasticbeanstalk/latest/dg/eb-cli3.html) - AWS-specific deployment tool
- [Terraform](https://www.terraform.io/) - Infrastructure as Code
- [Pulumi](https://www.pulumi.com/) - Modern IaC with programming languages

---

**Note:** This is an independent project and is not affiliated with or endorsed by AWS, Google, Microsoft, or Oracle.
