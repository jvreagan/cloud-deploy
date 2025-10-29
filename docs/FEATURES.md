# cloud-deploy Features Guide

Complete overview of cloud-deploy's features, capabilities, and what makes it different from other deployment tools.

## Table of Contents

- [What is cloud-deploy?](#what-is-cloud-deploy)
- [Core Philosophy](#core-philosophy)
- [Key Features](#key-features)
- [Provider Comparison](#provider-comparison)
- [Feature Comparison with Other Tools](#feature-comparison-with-other-tools)
- [Deployment Lifecycle](#deployment-lifecycle)
- [Supported Platforms](#supported-platforms)
- [Roadmap](#roadmap)

## What is cloud-deploy?

**cloud-deploy** is a manifest-driven, multi-cloud deployment tool that simplifies deploying containerized applications to major cloud providers using a single, declarative YAML configuration.

### The Problem It Solves

**Before cloud-deploy**:
```bash
# AWS deployment
eb init
eb create my-environment
eb deploy
eb setenv VAR1=value VAR2=value
# ... multiple commands, complex setup

# GCP deployment
gcloud projects create my-project
gcloud billing accounts projects link
gcloud services enable run.googleapis.com
gcloud builds submit --tag gcr.io/my-project/my-app
gcloud run deploy --image gcr.io/my-project/my-app
# ... many manual steps, CLI tools required

# Different commands, different workflows, different tools
```

**With cloud-deploy**:
```bash
# Any provider (AWS, GCP, Azure, OCI)
cloud-deploy -command deploy -manifest manifest.yaml

# One command, one manifest, any cloud
```

### Think of it as...

- **"docker-compose for cloud deployments"** - Declarative configuration for cloud infrastructure
- **"kubectl for multiple clouds"** - One tool to deploy anywhere
- **"Infrastructure as Config"** - Not full IaC like Terraform, but focused on application deployment

## Core Philosophy

### 1. Declarative Configuration

Everything is defined in a single YAML manifest:

```yaml
version: "1.0"

provider:
  name: aws
  region: us-east-2

application:
  name: my-app

environment:
  name: my-app-prod

deployment:
  platform: docker
  source:
    type: local
    path: "."

# Configuration, not code
```

You declare **what** you want, not **how** to achieve it.

### 2. Multi-Cloud Abstraction

Same manifest structure works across providers:

```yaml
# AWS Elastic Beanstalk
provider:
  name: aws
  region: us-east-2

# Google Cloud Run
provider:
  name: gcp
  region: us-central1
  project_id: my-project
  billing_account_id: XXXXXX

# Azure (coming soon)
provider:
  name: azure
  region: eastus

# Oracle Cloud (coming soon)
provider:
  name: oci
  region: us-phoenix-1
```

Change the provider block, keep everything else.

### 3. Zero Configuration Setup

**AWS**: No manual setup required
- Automatically creates applications
- Automatically creates environments
- Automatically uploads source code
- Auto-detects solution stacks

**GCP**: Completely self-sufficient
- Automatically creates projects
- Automatically links billing
- Automatically enables APIs
- Automatically builds containers

No console clicking, no CLI tools required (except cloud-deploy itself).

### 4. Production-Ready Features

Not just deployment - includes:
- ✅ Monitoring and logging
- ✅ Health checks
- ✅ Auto-scaling
- ✅ Environment variables
- ✅ IAM integration
- ✅ Resource tagging
- ✅ Rolling deployments
- ✅ Zero-downtime updates

## Key Features

### 🚀 Deployment Features

#### Automatic Resource Creation

**AWS**:
- Creates Elastic Beanstalk applications
- Creates environments (single-instance or load-balanced)
- Creates S3 buckets for application versions
- Uploads and versions application code
- Configures EC2 instances, security groups, load balancers

**GCP**:
- Creates GCP projects (if they don't exist)
- Links billing accounts to projects
- Enables required APIs (Cloud Build, Cloud Run, Storage, etc.)
- Creates Cloud Storage buckets
- Builds container images with Cloud Build
- Deploys to Cloud Run with auto-scaling

#### Solution Stack Auto-Detection (AWS)

Don't know which solution stack to use? cloud-deploy auto-detects:

```yaml
deployment:
  platform: docker  # cloud-deploy picks the latest Docker stack on Amazon Linux 2023
```

Automatically selects the latest stable stack for your platform.

#### Intelligent Polling

All long-running operations poll with proper timeouts:

**AWS**:
- Environment creation/updates (15-minute timeout)
- Environment termination (10-minute timeout)
- Real-time status updates

**GCP**:
- Project creation (3-minute timeout)
- API enablement (5-minute timeout per API)
- Service deployment (10-minute timeout)
- Detailed status messages at each stage

No more wondering if something is stuck or just slow.

#### Multiple Deployment Strategies

**AWS**:
- **SingleInstance**: One EC2 instance (dev/test)
- **LoadBalanced**: Auto-scaling with load balancer (production)
- Rolling deployments with zero downtime

**GCP**:
- **Serverless**: Cloud Run auto-scales from 0 to N
- **Always-On**: Keep minimum instances warm
- **Traffic Splitting**: Built into Cloud Run (via console)

### 📊 Monitoring & Logging

#### AWS CloudWatch Integration

```yaml
monitoring:
  # Enhanced health reporting
  enhanced_health: true

  # Custom metrics collection
  cloudwatch_metrics: true

  # Log streaming
  cloudwatch_logs:
    enabled: true
    retention_days: 30
    stream_logs: true
```

**Metrics Provided**:
- Request counts (2xx, 3xx, 4xx, 5xx)
- Latency percentiles (p50, p75, p90, p95, p99)
- Environment health status
- Instance health
- Auto-scaling metrics

**Logs Streamed**:
- Application stdout/stderr
- Docker container logs
- Elastic Beanstalk platform logs
- Web server access logs

#### GCP Cloud Logging Integration

```yaml
monitoring:
  cloudwatch_logs:  # Same config structure as AWS
    enabled: true
    retention_days: 7
    stream_logs: true
```

**Automatically Logs**:
- HTTP request logs (method, path, status, latency)
- Application stdout/stderr
- Container lifecycle events
- Cold start events
- Error traces

**Provides Direct Links** to:
- Cloud Console log viewer
- gcloud command to view logs
- Log retention configuration

### 🔐 Security Features

#### IAM Integration (AWS)

```yaml
iam:
  instance_profile: my-ec2-instance-profile
  service_role: aws-elasticbeanstalk-service-role
```

Your application can access AWS services without hardcoded credentials.

#### Service Account Authentication (GCP)

```yaml
provider:
  credentials:
    service_account_key_path: "/path/to/key.json"
```

Secure authentication using service account credentials.

#### Public/Private Access Control (GCP)

```yaml
provider:
  public_access: true   # Public internet access
  # or
  public_access: false  # Requires IAM authentication
```

Control who can access your services.

#### Environment Variable Management

```yaml
environment_variables:
  NODE_ENV: production
  DATABASE_URL: ${DATABASE_URL}  # From environment
  API_KEY: ${SECRET_API_KEY}     # Injected at deploy time
```

Never commit secrets - use environment variable substitution.

### ⚙️ Resource Configuration

#### AWS Instance Configuration

```yaml
instance:
  type: t3.micro                    # EC2 instance type
  environment_type: SingleInstance  # or LoadBalanced

health_check:
  type: enhanced                    # basic or enhanced
  path: /health                     # Health check endpoint
```

#### GCP Cloud Run Configuration

```yaml
cloud_run:
  cpu: "1"                          # CPU cores: 1, 2, 4
  memory: "512Mi"                   # Memory: 256Mi to 4Gi
  max_concurrency: 80               # Requests per container
  min_instances: 0                  # Scale to zero
  max_instances: 100                # Max scale out
  timeout_seconds: 300              # Request timeout
```

Fine-grained control over compute resources.

### 🔄 Environment Management

#### Create, Update, Deploy

```bash
# Create or update deployment
cloud-deploy -command deploy -manifest manifest.yaml
```

- **First run**: Creates everything (application, environment, resources)
- **Subsequent runs**: Updates existing deployment
- **Idempotent**: Safe to run multiple times

#### Stop Without Destroying

```bash
# Stop to save costs, preserves everything for fast restart
cloud-deploy -command stop -manifest manifest.yaml
```

**AWS**: Terminates environment but keeps application and versions
**GCP**: Deletes service but keeps container images

Resume later with `deploy` command - much faster than creating from scratch.

#### Complete Destruction

```bash
# Remove everything
cloud-deploy -command destroy -manifest manifest.yaml
```

**AWS**: Terminates environment (application preserved)
**GCP**: Deletes Cloud Run service

#### Status Checking

```bash
# Get deployment status
cloud-deploy -command status -manifest manifest.yaml
```

Returns:
- Application name
- Environment name
- Current status (Ready, Updating, Deploying, etc.)
- Health status (Healthy, Unhealthy, Unknown)
- Public URL
- Last update time

### 🏷️ Resource Tagging

```yaml
tags:
  Environment: production
  Project: myapp
  Team: backend
  CostCenter: engineering
  ManagedBy: cloud-deploy
```

**AWS**: Applied to all resources (EC2, load balancers, S3, etc.)
- Visible in AWS Console
- Used for cost allocation
- Searchable in Cost Explorer

**GCP**: Applied to projects and services
- Visible in Cloud Console
- Used for billing reports
- Filterable in resource lists

## Provider Comparison

### AWS Elastic Beanstalk vs GCP Cloud Run

| Feature | AWS Elastic Beanstalk | GCP Cloud Run |
|---------|----------------------|---------------|
| **Compute Model** | EC2 instances (VMs) | Containers (serverless) |
| **Scaling** | Auto-scaling groups (min-max instances) | Auto-scale 0-N containers |
| **Cold Starts** | No (instances always running) | Yes (if min_instances: 0) |
| **Cost Model** | Pay for EC2 instances 24/7 | Pay per request + instance time |
| **Load Balancing** | Application Load Balancer | Built-in (automatic) |
| **Health Checks** | Configurable (basic/enhanced) | Automatic |
| **HTTPS** | Requires certificate setup | Automatic (free) |
| **Best For** | Traditional apps, VMs needed, consistent load | Microservices, APIs, variable/spiky traffic |

### Cost Comparison (Approximate)

**Development Environment** (minimal traffic):

| Provider | Configuration | Monthly Cost |
|----------|--------------|--------------|
| AWS | t3.micro (SingleInstance) | ~$8-12 |
| GCP | Scale to zero (min_instances: 0) | < $1 |

**Production Environment** (moderate traffic):

| Provider | Configuration | Monthly Cost |
|----------|--------------|--------------|
| AWS | t3.small (LoadBalanced, 2-4 instances) | ~$50-100 |
| GCP | 2 min instances, scale to 50 max | ~$30-80 |

**Production Environment** (high traffic):

| Provider | Configuration | Monthly Cost |
|----------|--------------|--------------|
| AWS | t3.medium (LoadBalanced, 4-10 instances) | ~$200-500 |
| GCP | 5 min instances, scale to 500 max | ~$150-400 |

*Estimates include compute only, not data transfer or storage*

### When to Use Each Provider

**Use AWS Elastic Beanstalk when**:
- ✅ You need persistent instances (WebSockets, background workers)
- ✅ Your app requires local disk storage
- ✅ You need consistent, predictable latency (no cold starts)
- ✅ You're already using AWS services (RDS, ElastiCache, etc.)
- ✅ You need fine-grained control over infrastructure

**Use GCP Cloud Run when**:
- ✅ You have variable or spiky traffic (auto-scales better)
- ✅ You want to minimize costs (scale to zero when idle)
- ✅ You're building stateless microservices or APIs
- ✅ You want fastest time-to-deployment (fully managed)
- ✅ You don't need persistent local storage

## Feature Comparison with Other Tools

### cloud-deploy vs EB CLI (AWS)

| Feature | cloud-deploy | EB CLI |
|---------|-------------|--------|
| **Multi-cloud** | ✅ Yes (AWS, GCP, Azure*, OCI*) | ❌ AWS only |
| **Declarative** | ✅ YAML manifest | ⚠️ Partial (.ebextensions) |
| **Learning Curve** | Low (one manifest) | Medium (many commands) |
| **Auto-detection** | ✅ Solution stacks | ⚠️ Limited |
| **Setup Required** | ❌ None (uses AWS SDK) | ✅ EB CLI install |
| **CloudWatch Logs** | ✅ Built-in | ⚠️ Separate commands |
| **Resource Tagging** | ✅ In manifest | ⚠️ Via .ebextensions |

*In development

### cloud-deploy vs gcloud CLI (GCP)

| Feature | cloud-deploy | gcloud |
|---------|-------------|--------|
| **Multi-cloud** | ✅ Yes | ❌ GCP only |
| **Project Creation** | ✅ Automatic | ❌ Manual (gcloud projects create) |
| **API Enablement** | ✅ Automatic | ❌ Manual (gcloud services enable) |
| **Billing Link** | ✅ Automatic | ❌ Manual (gcloud billing) |
| **Declarative** | ✅ Full manifest | ❌ Imperative commands |
| **Resource Config** | ✅ In manifest (CPU, memory, scaling) | ⚠️ Via flags |
| **Setup Required** | ❌ None (service account key) | ✅ gcloud CLI install |

### cloud-deploy vs Terraform

| Feature | cloud-deploy | Terraform |
|---------|-------------|-----------|
| **Purpose** | Application deployment | Infrastructure as Code |
| **Scope** | App-focused | Everything (networks, DBs, etc.) |
| **Learning Curve** | Low | High (HCL language) |
| **Multi-cloud** | ✅ Same manifest format | ⚠️ Different providers/syntax |
| **State Management** | ❌ None (reads from cloud) | ✅ State files required |
| **Docker Support** | ✅ Native | ⚠️ Via providers |
| **Auto-scaling Config** | ✅ Simple | ⚠️ Complex (many resources) |
| **Best For** | Deploying apps | Managing infrastructure |

**When to use both**:
```
Terraform → Create VPC, RDS database, IAM roles
cloud-deploy → Deploy application to Elastic Beanstalk/Cloud Run
```

### cloud-deploy vs Kubernetes/Helm

| Feature | cloud-deploy | Kubernetes + Helm |
|---------|-------------|-------------------|
| **Infrastructure** | Fully managed (Beanstalk/Cloud Run) | Self-managed or GKE/EKS |
| **Complexity** | Low | High |
| **Manifest Format** | Simple YAML (one file) | Multiple YAMLs (services, deployments, etc.) |
| **Learning Curve** | Low | Very High |
| **Setup Time** | Minutes | Hours to days |
| **Best For** | Standard web apps, APIs, microservices | Complex applications, need full control |
| **Cost** | Lower (managed services) | Higher (cluster overhead) |

**Use K8s/Helm when**:
- You need advanced orchestration (sidecars, init containers, etc.)
- You're running 20+ microservices
- You need service mesh, custom networking
- You want full portability

**Use cloud-deploy when**:
- You want simple, fast deployments
- You have < 10 services
- You want minimal operational overhead
- Cost efficiency matters

## Deployment Lifecycle

### 1. Initial Deployment

```bash
cloud-deploy -command deploy -manifest manifest.yaml
```

**What happens**:

**AWS**:
1. ✅ Creates Elastic Beanstalk application (if doesn't exist)
2. ✅ Auto-detects solution stack (if not specified)
3. ✅ Creates S3 bucket for versions
4. ✅ Zips source code
5. ✅ Uploads to S3
6. ✅ Creates application version
7. ✅ Creates environment (EC2, load balancer, security groups)
8. ✅ Deploys application
9. ✅ Configures health checks and monitoring
10. ✅ Waits for environment to be ready (polls every 10s)
11. ✅ Returns public URL

**GCP**:
1. ✅ Creates project (if doesn't exist, polls until ready)
2. ✅ Links billing account
3. ✅ Enables APIs (Cloud Build, Run, Storage - polls each)
4. ✅ Creates storage bucket
5. ✅ Creates tarball of source
6. ✅ Uploads to Cloud Storage
7. ✅ Triggers Cloud Build (builds Docker image)
8. ✅ Waits for build to complete (polls until success)
9. ✅ Creates Cloud Run service with resource config
10. ✅ Configures public access (if enabled)
11. ✅ Configures Cloud Logging (if enabled)
12. ✅ Waits for service to be ready (polls until healthy)
13. ✅ Returns public URL

**Time**: 5-15 minutes (first deployment)

### 2. Update Deployment

```bash
cloud-deploy -command deploy -manifest manifest.yaml
```

**What happens**:

**AWS**:
1. ✅ Detects existing environment
2. ✅ Creates new application version
3. ✅ Updates environment with new version (rolling deployment)
4. ✅ Waits for update to complete
5. ✅ Returns updated URL

**GCP**:
1. ✅ Detects existing service
2. ✅ Builds new container image
3. ✅ Updates Cloud Run service (blue-green deployment)
4. ✅ Gradually shifts traffic to new revision
5. ✅ Waits for service to be ready
6. ✅ Returns URL

**Time**: 3-8 minutes (updates are faster)

### 3. Stop Deployment

```bash
cloud-deploy -command stop -manifest manifest.yaml
```

**AWS**:
- Terminates environment (EC2 instances, load balancer)
- Preserves application and versions in S3
- Next deploy is much faster (doesn't create from scratch)

**GCP**:
- Deletes Cloud Run service
- Preserves container images in GCR
- Next deploy is much faster (doesn't rebuild)

**Time**: 3-5 minutes

### 4. Destroy Deployment

```bash
cloud-deploy -command destroy -manifest manifest.yaml
```

**AWS**:
- Terminates environment completely
- Application definition preserved (not deleted)

**GCP**:
- Deletes Cloud Run service
- Project, storage, and images remain

**Time**: 3-5 minutes

### 5. Check Status

```bash
cloud-deploy -command status -manifest manifest.yaml
```

Returns current deployment status - fast, no modifications.

**Time**: < 5 seconds

## Supported Platforms

### Current Support

#### AWS Elastic Beanstalk

**Platforms**:
- ✅ Docker (single container)
- ✅ Node.js
- ✅ Python
- ✅ Ruby
- ✅ Go
- ✅ Java
- ✅ PHP
- ✅ .NET

**Environments**:
- ✅ Single Instance
- ✅ Load Balanced
- ✅ Worker (background processing)

**Regions**: All AWS regions

#### GCP Cloud Run

**Platforms**:
- ✅ Docker (any language with Dockerfile)

**Scaling**:
- ✅ Scale to zero
- ✅ Auto-scaling
- ✅ Always-on (min_instances)

**Regions**: All Cloud Run regions

### Coming Soon

#### Azure Container Instances

- ⏳ Planned for v0.3
- ⏳ Similar to Cloud Run (serverless containers)
- ⏳ Azure-specific features (ACR integration, VNet support)

#### Oracle Cloud Infrastructure

- ⏳ Planned for v0.4
- ⏳ Container Instances
- ⏳ OKE (managed Kubernetes) integration

## Roadmap

### v0.2 (Current)
- ✅ AWS Elastic Beanstalk support
- ✅ GCP Cloud Run support
- ✅ Operation polling
- ✅ CloudWatch/Cloud Logging integration
- ✅ Resource configuration
- ✅ Comprehensive documentation

### v0.3 (Next)
- ⏳ Azure Container Instances support
- ⏳ Environment variable templating
- ⏳ Multi-environment deployments (single command)
- ⏳ Deployment rollback
- ⏳ Blue-green deployments (AWS)
- ⏳ Traffic splitting (GCP)

### v0.4 (Future)
- ⏳ OCI Container Instances support
- ⏳ CI/CD integration guides
- ⏳ Secrets management integration
- ⏳ Cost estimation before deployment
- ⏳ Deployment history and audit logs

### v1.0 (Long-term)
- ⏳ All four providers (AWS, GCP, Azure, OCI)
- ⏳ Advanced networking configuration
- ⏳ Database provisioning
- ⏳ Service mesh integration
- ⏳ Comprehensive test coverage
- ⏳ Official Homebrew/APT packages

## Why Choose cloud-deploy?

### ✅ Use cloud-deploy if you want:

- Simple, declarative deployments
- Multi-cloud support with one tool
- Fast time-to-production
- Minimal operational overhead
- Built-in monitoring and logging
- Cost-effective deployments
- Docker container support
- Good defaults with flexibility

### ❌ Don't use cloud-deploy if you need:

- Full infrastructure control (use Terraform)
- Advanced Kubernetes features (use K8s/Helm)
- Complex multi-tier architectures (use Terraform + K8s)
- Database provisioning (coming in v0.4)
- Advanced networking (VPCs, peering, etc.)

## Additional Resources

- [AWS Deployment Guide](./AWS.md)
- [GCP Deployment Guide](./GCP.md)
- [Monitoring Guide](./MONITORING.md)
- [Main README](../README.md)
- [Example Manifests](../examples/)

## Contributing

We welcome contributions! Areas we need help:

- Azure Container Instances provider
- OCI Container Instances provider
- Documentation improvements
- Bug reports and fixes
- Feature requests

See [CONTRIBUTING.md](../CONTRIBUTING.md) for details.

---

**cloud-deploy** - Deploy once, run anywhere. 🚀
