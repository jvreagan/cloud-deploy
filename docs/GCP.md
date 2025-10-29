# GCP Deployment Guide

Complete guide for deploying applications to Google Cloud Run using cloud-deploy.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Authentication](#authentication)
- [Manifest Configuration](#manifest-configuration)
- [Required Fields](#required-fields)
- [Optional Fields](#optional-fields)
- [Cloud Run Configuration](#cloud-run-configuration)
- [Advanced Configuration](#advanced-configuration)
- [Examples](#examples)
- [Monitoring & Logging](#monitoring--logging)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

cloud-deploy uses **Google Cloud Run** to deploy containerized applications. It's **completely self-sufficient** - no gcloud CLI required!

### What Makes This Special?

Unlike other deployment tools that require extensive GCP setup, cloud-deploy does **everything automatically**:

✅ **Project Management**: Creates GCP projects automatically
✅ **Billing Setup**: Links billing accounts
✅ **API Enablement**: Enables all required APIs (Cloud Build, Cloud Run, Storage, etc.)
✅ **Container Building**: Builds Docker images using Cloud Build
✅ **Serverless Deployment**: Deploys to fully managed Cloud Run
✅ **Auto-Scaling**: Scales from 0 to N instances based on traffic
✅ **Cloud Logging**: Integrated logging with configurable retention
✅ **Public Access**: Optional public access with IAM configuration

### Cloud Run Benefits

- 🚀 **Serverless**: No infrastructure management
- 💰 **Pay-per-use**: Only pay when handling requests
- 📈 **Auto-scaling**: Scales to zero when idle, scales up automatically
- 🌍 **Global**: Deploy to regions worldwide
- ⚡ **Fast**: Cold starts < 1 second for optimized containers
- 🔒 **Secure**: HTTPS by default, IAM-based access control

## Prerequisites

### 1. GCP Account Setup

**Create a GCP Account**:
- Go to [https://cloud.google.com](https://cloud.google.com)
- Sign up (new users get $300 free credits)
- Complete account verification

### 2. Enable Billing

**Set up a Billing Account**:
1. Go to [GCP Billing](https://console.cloud.google.com/billing)
2. Click "Create Account" or link existing billing account
3. Add payment method
4. Copy your **Billing Account ID** (format: `XXXXXX-XXXXXX-XXXXXX`)

You'll need this ID in your manifest!

### 3. Create Service Account

**Step-by-Step**:

1. Go to [GCP Service Accounts](https://console.cloud.google.com/iam-admin/serviceaccounts)
2. Click **"Create Service Account"**
3. Enter name: `cloud-deploy-admin`
4. Click **"Create and Continue"**
5. Grant roles (choose one approach):

   **Option A: Simple (Recommended for Getting Started)**
   - Add role: `Owner`

   **Option B: Granular (Recommended for Production)**
   - `Project Creator` - Create and manage projects
   - `Billing Account User` - Link billing to projects
   - `Service Usage Admin` - Enable APIs
   - `Cloud Run Admin` - Deploy Cloud Run services
   - `Cloud Build Editor` - Build container images
   - `Storage Admin` - Upload source code
   - `Logging Admin` - Configure Cloud Logging

6. Click **"Continue"** then **"Done"**
7. Click on the service account you just created
8. Go to **Keys** tab
9. Click **Add Key → Create new key → JSON**
10. Download the JSON key file
11. **Save it securely** (you'll reference it in your manifest)

⚠️ **Security**: This key file grants access to your GCP resources. Never commit it to version control!

### 4. Application Requirements

- **Dockerized application** with a `Dockerfile`
- Application must listen on the `PORT` environment variable (Cloud Run sets this)
- Health checks optional (Cloud Run has built-in health checking)

## Authentication

cloud-deploy uses **Service Account credentials** for GCP authentication. Unlike AWS, there's no CLI credential chain fallback - you must provide service account credentials.

### Method 1: File Path (Recommended)

Store the service account key file outside your repository:

```yaml
provider:
  name: gcp
  region: us-central1
  credentials:
    service_account_key_path: "/path/to/service-account-key.json"
```

**Best practice**: Store the key in:
- `~/.gcp/keys/my-service-account.json` (Linux/Mac)
- `C:\Users\YourName\.gcp\keys\my-service-account.json` (Windows)
- Add to `.gitignore`: `*.json` or `*-key.json`

### Method 2: Inline JSON

For CI/CD where credentials are injected as environment variables:

```yaml
provider:
  name: gcp
  region: us-central1
  credentials:
    service_account_key_json: |
      {
        "type": "service_account",
        "project_id": "my-project",
        "private_key_id": "...",
        "private_key": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n",
        "client_email": "cloud-deploy-admin@my-project.iam.gserviceaccount.com",
        "client_id": "...",
        "auth_uri": "https://accounts.google.com/o/oauth2/auth",
        "token_uri": "https://oauth2.googleapis.com/token",
        "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
        "client_x509_cert_url": "..."
      }
```

**For CI/CD**: Store JSON as secret, inject via environment variable substitution:

```yaml
provider:
  credentials:
    service_account_key_json: ${GCP_SERVICE_ACCOUNT_KEY}
```

## Manifest Configuration

### Minimal Example

The simplest possible GCP deployment:

```yaml
version: "1.0"

provider:
  name: gcp
  region: us-central1
  project_id: my-new-project
  billing_account_id: XXXXXX-XXXXXX-XXXXXX
  credentials:
    service_account_key_path: "/path/to/service-account-key.json"

application:
  name: my-app

environment:
  name: my-app-env

deployment:
  platform: docker
  source:
    type: local
    path: "."
```

This will:
1. ✅ Create a new GCP project called "my-new-project"
2. ✅ Link your billing account
3. ✅ Enable required APIs (Cloud Build, Cloud Run, Storage)
4. ✅ Build your Docker container using Cloud Build
5. ✅ Deploy to Cloud Run
6. ✅ Make the service publicly accessible (default)

## Required Fields

### Provider Configuration

```yaml
provider:
  name: gcp                              # Must be "gcp"
  region: us-central1                    # GCP region for Cloud Run
  project_id: my-project-id              # GCP project ID (created if doesn't exist)
  billing_account_id: XXXXXX-XXXXXX-XXXXXX  # Your billing account ID
  credentials:
    service_account_key_path: "/path/to/key.json"  # Path to service account key
```

**Supported Regions**:

| Region | Location | Latency | Tier |
|--------|----------|---------|------|
| `us-central1` | Iowa, USA | Low | 1 |
| `us-east1` | South Carolina, USA | Low | 1 |
| `us-west1` | Oregon, USA | Low | 1 |
| `europe-west1` | Belgium | Low | 1 |
| `europe-west4` | Netherlands | Low | 1 |
| `asia-east1` | Taiwan | Medium | 1 |
| `asia-northeast1` | Tokyo | Medium | 1 |
| `australia-southeast1` | Sydney | High | 2 |

See [Cloud Run Locations](https://cloud.google.com/run/docs/locations) for complete list.

**Project ID Rules**:
- 6-30 characters
- Lowercase letters, numbers, hyphens
- Must start with a letter
- Must be globally unique across all of GCP
- Cannot be changed after creation

**Finding Your Billing Account ID**:
1. Go to [console.cloud.google.com/billing](https://console.cloud.google.com/billing)
2. Click on your billing account name
3. Copy the ID from the "Billing Account ID" field (format: `XXXXXX-XXXXXX-XXXXXX`)

### Application Configuration

```yaml
application:
  name: my-app                           # Application name (used for container registry)
  description: "My Cloud Run Application"  # Optional description
```

**Naming Rules**:
- Lowercase letters, numbers, hyphens
- Must start with a letter
- 1-63 characters
- Used as part of container image name in GCR

### Environment Configuration

```yaml
environment:
  name: my-app-env                       # Cloud Run service name
```

**About Environments**:
- Each environment is a separate Cloud Run service
- Environments can be: `dev`, `staging`, `production`, etc.
- Each has its own URL: `https://<service>-<hash>-<region>.a.run.app`

### Deployment Configuration

```yaml
deployment:
  platform: docker                       # Must be "docker" for Cloud Run
  source:
    type: local                          # Source location: local, s3, or git
    path: "."                            # Path to directory containing Dockerfile
```

**Important**: Your application must:
1. Have a `Dockerfile` in the source path
2. Listen on the port specified by the `PORT` environment variable (Cloud Run sets this automatically)
3. Start up within 240 seconds (configurable via `cloud_run.timeout_seconds`)

**Dockerfile Example**:
```dockerfile
FROM node:18-slim

WORKDIR /app
COPY package*.json ./
RUN npm ci --production
COPY . .

# Cloud Run sets PORT environment variable
ENV PORT=8080
EXPOSE 8080

CMD ["node", "server.js"]
```

**Application Code**:
```javascript
// Listen on the port Cloud Run provides
const PORT = process.env.PORT || 8080;
app.listen(PORT, '0.0.0.0', () => {
  console.log(`Server running on port ${PORT}`);
});
```

## Optional Fields

### Public Access Control

```yaml
provider:
  public_access: true                    # Default: true
```

**Options**:
- `true`: Anyone can access your service (no authentication required)
- `false`: Requires IAM authentication (only authorized users/services)

**When to Use Private Access**:
- Internal microservices
- Backend APIs called by authenticated frontend
- Services behind Cloud Armor or API Gateway

**Testing Private Services**:
```bash
# Get an identity token
gcloud auth print-identity-token

# Call the service
curl -H "Authorization: Bearer $(gcloud auth print-identity-token)" \
  https://my-service-xyz-uc.a.run.app
```

### Organization Configuration

If creating projects under a GCP Organization:

```yaml
provider:
  organization_id: "123456789012"        # Optional: GCP organization ID
```

**Finding Organization ID**:
```bash
gcloud organizations list
```

### Environment Variables

Pass configuration to your Cloud Run service:

```yaml
environment_variables:
  NODE_ENV: production
  LOG_LEVEL: info
  DATABASE_URL: postgres://user:pass@host:5432/db
  API_KEY: ${API_KEY}                    # Can use substitution
```

**Best Practice**: Use Secret Manager for sensitive data:
1. Store secrets in [Secret Manager](https://console.cloud.google.com/security/secret-manager)
2. Grant service account access to secrets
3. Access from application using Secret Manager API

## Cloud Run Configuration

Configure Cloud Run resources, scaling, and timeouts:

```yaml
cloud_run:
  # CPU allocation per container instance
  cpu: "1"                               # Options: "1", "2", "4" | Default: "1"

  # Memory allocation per container instance
  memory: "512Mi"                        # Options: "256Mi", "512Mi", "1Gi", "2Gi", "4Gi" | Default: "512Mi"

  # Maximum concurrent requests per container
  max_concurrency: 80                    # 1-1000 | Default: 80

  # Minimum instances (0 = scale to zero)
  min_instances: 0                       # 0-1000 | Default: 0

  # Maximum instances
  max_instances: 10                      # 1-1000 | Default: 100

  # Request timeout in seconds
  timeout_seconds: 300                   # 1-3600 | Default: 300
```

### CPU Configuration

| CPU | Use Case | Memory Range | Cost Factor |
|-----|----------|--------------|-------------|
| `"1"` | Web apps, APIs, light workloads | 128Mi - 4Gi | 1x |
| `"2"` | CPU-intensive tasks, image processing | 512Mi - 8Gi | 2x |
| `"4"` | Heavy computation, video transcoding | 2Gi - 16Gi | 4x |

**Notes**:
- CPU is only allocated during request processing (unless always-on)
- More CPU = higher cost but faster response times
- Always-on CPU requires min_instances > 0

### Memory Configuration

| Memory | Use Case | Min CPU | Cost |
|--------|----------|---------|------|
| `256Mi` | Minimal APIs, static file serving | 1 | Lowest |
| `512Mi` | Standard web apps, REST APIs | 1 | Low |
| `1Gi` | Applications with caching, moderate data | 1 | Medium |
| `2Gi` | Large caches, data processing | 1-2 | Medium-High |
| `4Gi` | Heavy memory usage, ML inference | 2-4 | High |

### Concurrency Configuration

`max_concurrency` controls how many requests a single container can handle simultaneously.

**Low Concurrency (1-10)**:
- ✅ Better isolation between requests
- ✅ Predictable latency
- ❌ More container instances needed
- **Use for**: Long-running requests, stateful applications

**Medium Concurrency (10-80)** - *Recommended Default*:
- ✅ Good balance of efficiency and isolation
- ✅ Cost-effective for most workloads
- **Use for**: Standard web applications, REST APIs

**High Concurrency (80-1000)**:
- ✅ Very cost-effective (fewer instances)
- ❌ Potential latency variance under load
- ❌ One slow request can affect others
- **Use for**: Fast, stateless requests; lightweight APIs

### Scaling Configuration

**Scale to Zero** (min_instances: 0):
```yaml
cloud_run:
  min_instances: 0    # Default
  max_instances: 10
```

✅ **Pros**: Pay only when handling requests, extremely cost-effective
❌ **Cons**: Cold start latency (~1-3 seconds for first request after idle)

**Warm Pool** (min_instances: 1-N):
```yaml
cloud_run:
  min_instances: 2    # Keep 2 instances always running
  max_instances: 100
```

✅ **Pros**: No cold starts, consistent latency
❌ **Cons**: Pay for idle time (24/7 cost for min_instances)

**Scaling Calculation**:
```
Instances = CEILING(Concurrent Requests / max_concurrency)
```

Example: 200 concurrent requests with max_concurrency of 80:
```
200 / 80 = 2.5 → 3 instances needed
```

### Timeout Configuration

```yaml
cloud_run:
  timeout_seconds: 300    # 5 minutes (default)
```

**Timeout Limits**:
- Minimum: 1 second
- Maximum: 3600 seconds (1 hour)
- Default: 300 seconds (5 minutes)

**Use Cases**:
- **Short (< 60s)**: Fast APIs, web servers
- **Medium (60-300s)**: Report generation, data processing
- **Long (300-3600s)**: Batch jobs, video processing, ML inference

⚠️ **Note**: Long timeouts may incur higher costs. Consider Cloud Run Jobs for batch processing.

## Advanced Configuration

### Monitoring & Cloud Logging

Enable Cloud Logging integration:

```yaml
monitoring:
  cloudwatch_logs:                       # Name kept for AWS compatibility
    enabled: true                        # Enable Cloud Logging
    retention_days: 7                    # Log retention (1-3653 days)
    stream_logs: true                    # Stream all logs
```

**What Gets Logged**:
- ✅ Application stdout/stderr
- ✅ HTTP request logs (method, path, status, latency, user agent)
- ✅ System events (cold starts, container crashes, deployments)
- ✅ Cloud Run platform logs

**Viewing Logs**:

After deployment, cloud-deploy provides:
```
View logs: gcloud logging read 'resource.type=cloud_run_revision AND resource.labels.service_name=my-app-env' --limit 50 --project=my-project
```

Or use Cloud Console:
```
https://console.cloud.google.com/logs/query?project=my-project
```

**Log Retention**:
- Default: 30 days
- Configurable: 1-3653 days
- Longer retention = higher storage costs

**Cost**: Free tier includes 50 GB/month ingestion, 10 GB storage. See [Cloud Logging Pricing](https://cloud.google.com/logging/pricing).

### Complete Configuration Example

```yaml
version: "1.0"

provider:
  name: gcp
  region: us-central1
  project_id: my-production-project
  billing_account_id: XXXXXX-XXXXXX-XXXXXX
  public_access: true
  credentials:
    service_account_key_path: "~/.gcp/keys/prod-service-account.json"

application:
  name: production-api
  description: "Production REST API on Cloud Run"

environment:
  name: prod-api

deployment:
  platform: docker
  source:
    type: local
    path: "."

# Cloud Run resource configuration
cloud_run:
  cpu: "2"
  memory: "1Gi"
  max_concurrency: 80
  min_instances: 2              # Keep 2 warm for low latency
  max_instances: 100            # Scale up to 100
  timeout_seconds: 60           # 1 minute timeout

# Monitoring configuration
monitoring:
  cloudwatch_logs:
    enabled: true
    retention_days: 30
    stream_logs: true

# Environment variables
environment_variables:
  NODE_ENV: production
  LOG_LEVEL: warn
  API_VERSION: v2
  MAX_REQUEST_SIZE: "10mb"
```

## Examples

### Example 1: Simple Development Service

```yaml
version: "1.0"

provider:
  name: gcp
  region: us-central1
  project_id: my-dev-project
  billing_account_id: XXXXXX-XXXXXX-XXXXXX
  credentials:
    service_account_key_path: "~/.gcp/keys/dev-sa.json"

application:
  name: simple-api

environment:
  name: simple-api-dev

deployment:
  platform: docker
  source:
    type: local
    path: "."

# Minimal resources for development
cloud_run:
  cpu: "1"
  memory: "256Mi"
  min_instances: 0              # Scale to zero to save costs

environment_variables:
  NODE_ENV: development
  DEBUG: "true"
```

**Cost**: Near-zero when idle (scale-to-zero enabled)

### Example 2: Production API with High Traffic

```yaml
version: "1.0"

provider:
  name: gcp
  region: us-central1
  project_id: my-prod-project
  billing_account_id: XXXXXX-XXXXXX-XXXXXX
  public_access: true
  credentials:
    service_account_key_path: "/secure/location/prod-sa.json"

application:
  name: high-traffic-api
  description: "Production API with auto-scaling"

environment:
  name: api-prod

deployment:
  platform: docker
  source:
    type: local
    path: "."

cloud_run:
  cpu: "2"                      # More CPU for faster responses
  memory: "1Gi"                 # Enough for caching
  max_concurrency: 100          # Handle many concurrent requests
  min_instances: 5              # Always-on for zero cold starts
  max_instances: 500            # Scale to handle traffic spikes
  timeout_seconds: 30           # Fast API, short timeout

monitoring:
  cloudwatch_logs:
    enabled: true
    retention_days: 90          # Long retention for compliance
    stream_logs: true

environment_variables:
  NODE_ENV: production
  LOG_LEVEL: error
  CACHE_ENABLED: "true"
  MAX_CACHE_SIZE: "512MB"
```

**Cost**: Higher (5 always-on instances) but zero cold starts, consistent latency.

### Example 3: Batch Processing Service

```yaml
version: "1.0"

provider:
  name: gcp
  region: us-central1
  project_id: batch-processing-project
  billing_account_id: XXXXXX-XXXXXX-XXXXXX
  public_access: false          # Internal service only
  credentials:
    service_account_key_path: "/keys/batch-sa.json"

application:
  name: batch-processor
  description: "Long-running batch processing"

environment:
  name: batch-processor-prod

deployment:
  platform: docker
  source:
    type: local
    path: "."

cloud_run:
  cpu: "4"                      # Maximum CPU for processing
  memory: "4Gi"                 # Large memory for data processing
  max_concurrency: 1            # One job per container
  min_instances: 0              # Scale to zero when idle
  max_instances: 20             # Limit parallel jobs
  timeout_seconds: 3600         # 1 hour timeout for long jobs

environment_variables:
  WORKER_MODE: batch
  CHUNK_SIZE: "1000"
```

**Use Case**: Processing large datasets, video transcoding, ML batch inference.

### Example 4: Microservice with Secret Manager

```yaml
version: "1.0"

provider:
  name: gcp
  region: europe-west1
  project_id: microservices-prod
  billing_account_id: XXXXXX-XXXXXX-XXXXXX
  public_access: false          # Behind API Gateway
  credentials:
    service_account_key_path: "${GCP_SA_KEY_PATH}"

application:
  name: payment-service
  description: "Payment processing microservice"

environment:
  name: payment-svc-prod

deployment:
  platform: docker
  source:
    type: local
    path: "./services/payment"

cloud_run:
  cpu: "2"
  memory: "512Mi"
  max_concurrency: 50
  min_instances: 2              # Critical service, always available
  max_instances: 50
  timeout_seconds: 30

monitoring:
  cloudwatch_logs:
    enabled: true
    retention_days: 365         # Long retention for audit
    stream_logs: true

environment_variables:
  SERVICE_NAME: payment
  ENVIRONMENT: production
  # Secrets loaded via Secret Manager in application code
  # Do not put actual secrets here!
```

**Security**: Use Secret Manager for sensitive data:
```javascript
// In your application
const {SecretManagerServiceClient} = require('@google-cloud/secret-manager');
const client = new SecretManagerServiceClient();

async function getSecret(secretName) {
  const [version] = await client.accessSecretVersion({
    name: `projects/PROJECT_ID/secrets/${secretName}/versions/latest`,
  });
  return version.payload.data.toString();
}

const apiKey = await getSecret('payment-api-key');
```

## Best Practices

### 1. Project Organization

**Separate Projects per Environment**:
```
my-app-dev         → Development
my-app-staging     → Staging
my-app-prod        → Production
```

Benefits:
- ✅ Clear billing separation
- ✅ Isolated permissions
- ✅ Prevents accidental prod deployments

### 2. Service Naming Strategy

```
<app-name>-<environment>
Examples:
  - api-dev
  - api-staging
  - api-prod
  - worker-prod
  - frontend-dev
```

### 3. Resource Sizing Guidelines

| Use Case | CPU | Memory | Min Instances | Max Instances |
|----------|-----|--------|---------------|---------------|
| Dev/Test | 1 | 256Mi-512Mi | 0 | 10 |
| Low Traffic Prod | 1 | 512Mi | 0-1 | 50 |
| Medium Traffic Prod | 1-2 | 512Mi-1Gi | 1-2 | 100 |
| High Traffic Prod | 2-4 | 1Gi-4Gi | 5-10 | 500+ |
| Background Jobs | 2-4 | 1Gi-4Gi | 0 | 20 |

### 4. Cost Optimization

**Development**:
```yaml
cloud_run:
  cpu: "1"
  memory: "256Mi"
  min_instances: 0              # Scale to zero when idle
  max_instances: 10
```

**Estimated Cost**: < $5/month with minimal usage

**Production (Low Traffic)**:
```yaml
cloud_run:
  cpu: "1"
  memory: "512Mi"
  min_instances: 1              # One always-on instance
  max_instances: 50
```

**Estimated Cost**: ~$30-50/month (1 always-on instance + autoscaling)

**Production (High Traffic)**:
```yaml
cloud_run:
  cpu: "2"
  memory: "1Gi"
  min_instances: 5
  max_instances: 100
```

**Estimated Cost**: ~$150-300/month (5 always-on instances + autoscaling)

Use [GCP Pricing Calculator](https://cloud.google.com/products/calculator) for accurate estimates.

### 5. Security Best Practices

**Service Account Keys**:
- ✅ Store outside repository (e.g., `~/.gcp/keys/`)
- ✅ Use separate service accounts per environment
- ✅ Rotate keys regularly (every 90 days)
- ✅ Use minimal required permissions
- ❌ Never commit to version control
- ❌ Never store in public locations

**Secrets Management**:
- ✅ Use [Secret Manager](https://cloud.google.com/secret-manager) for API keys, passwords, tokens
- ✅ Grant service account access only to needed secrets
- ✅ Use secret versioning for rotation
- ❌ Don't put secrets in environment variables in manifest

**Network Security**:
- Use `public_access: false` for internal services
- Place behind Cloud Armor for DDoS protection
- Use API Gateway for rate limiting and authentication
- Enable Cloud Armor security policies

### 6. Monitoring & Observability

**Enable Logging**:
```yaml
monitoring:
  cloudwatch_logs:
    enabled: true
    retention_days: 30          # Balance cost and compliance needs
```

**Application Logging Best Practices**:
```javascript
// Use structured logging
console.log(JSON.stringify({
  severity: 'INFO',
  message: 'User logged in',
  userId: user.id,
  timestamp: new Date().toISOString()
}));

// Cloud Logging automatically parses JSON
```

**Monitor Key Metrics**:
- Request count
- Request latency (p50, p95, p99)
- Error rate
- Container instance count
- CPU and memory utilization

**Set Up Alerts** in Cloud Monitoring for:
- Error rate > 5%
- P95 latency > 1000ms
- Instance count approaching max_instances

### 7. Deployment Workflow

```bash
# 1. Build and test locally
docker build -t myapp .
docker run -p 8080:8080 -e PORT=8080 myapp
curl http://localhost:8080/health

# 2. Deploy to development
cloud-deploy -command deploy -manifest manifest-dev.yaml

# 3. Check status
cloud-deploy -command status -manifest manifest-dev.yaml

# 4. View logs
gcloud logging read 'resource.type=cloud_run_revision' --limit 50 --project=my-dev-project

# 5. Test development deployment
curl https://my-app-dev-xyz-uc.a.run.app/health

# 6. Deploy to production
cloud-deploy -command deploy -manifest manifest-prod.yaml

# 7. Monitor production deployment
# Watch logs and metrics in Cloud Console
```

### 8. Container Optimization

**Optimize Dockerfile for Cloud Run**:

```dockerfile
# Use specific version tags, not 'latest'
FROM node:18.19-slim

# Set working directory
WORKDIR /app

# Copy only package files first (better caching)
COPY package*.json ./

# Install production dependencies only
RUN npm ci --only=production && npm cache clean --force

# Copy application code
COPY . .

# Cloud Run sets PORT environment variable
ENV PORT=8080

# Use non-root user for security
RUN useradd -m appuser
USER appuser

# Expose port (documentation only, Cloud Run uses PORT env var)
EXPOSE 8080

# Start application
CMD ["node", "server.js"]
```

**Reduce Cold Start Time**:
1. Use smaller base images (`-slim`, `-alpine`)
2. Minimize layers in Dockerfile
3. Use multi-stage builds to reduce final image size
4. Optimize application startup time
5. Consider using `min_instances: 1` for critical services

**Example Multi-Stage Build**:
```dockerfile
# Build stage
FROM node:18 AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

# Production stage
FROM node:18-slim
WORKDIR /app
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/node_modules ./node_modules
ENV PORT=8080
EXPOSE 8080
CMD ["node", "dist/server.js"]
```

## Troubleshooting

### Project Creation Fails

**Error**: "Failed to create project: permission denied"

**Causes**:
1. Service account lacks "Project Creator" role
2. Organization policy restricts project creation
3. Billing account doesn't have permissions

**Solutions**:
1. Grant `roles/resourcemanager.projectCreator` to service account
2. Check organization policies
3. Verify billing account access

### Build Fails

**Error**: "Failed to build image: context exceeded deadline"

**Causes**:
1. Dockerfile build is too slow (large dependencies)
2. Network timeout downloading packages
3. Build timeout (default: 20 minutes)

**Solutions**:
```dockerfile
# 1. Use multi-stage builds
FROM node:18 AS builder
# ... build steps ...

FROM node:18-slim
COPY --from=builder /app ./app

# 2. Use .dockerignore to exclude unnecessary files
# .dockerignore:
node_modules
.git
*.md
.env
tests/

# 3. Layer caching - copy package.json first
COPY package*.json ./
RUN npm ci
COPY . .
```

### Container Crashes on Startup

**Error**: Service shows "Service is starting" then fails

**Causes**:
1. Application doesn't listen on `PORT` environment variable
2. Application crashes during startup
3. Missing dependencies
4. Insufficient memory

**Solutions**:

1. **Listen on PORT**:
```javascript
// Correct ✅
const PORT = process.env.PORT || 8080;
app.listen(PORT, '0.0.0.0');

// Wrong ❌
app.listen(3000, 'localhost');  // Must use PORT and bind to 0.0.0.0
```

2. **Check logs**:
```bash
gcloud logging read 'resource.type=cloud_run_revision AND resource.labels.service_name=my-app' --limit 100 --project=my-project
```

3. **Increase memory**:
```yaml
cloud_run:
  memory: "1Gi"  # Increase if seeing OOM errors
```

### Service Returns 404

**Error**: Accessing service URL returns 404

**Causes**:
1. Application not listening on correct port
2. Application routing issue
3. Service not fully deployed

**Solutions**:
1. Verify application listens on `process.env.PORT`
2. Check application logs for routing errors
3. Wait for deployment to complete (check status)

### "Permission Denied" Accessing Service

**Error**: 403 Forbidden when accessing public service

**Cause**: IAM permissions not set correctly

**Solution**:
```yaml
provider:
  public_access: true  # Ensure this is set to true
```

Or manually grant access:
```bash
gcloud run services add-iam-policy-binding my-service \
  --region=us-central1 \
  --member="allUsers" \
  --role="roles/run.invoker"
```

### Logs Not Appearing

**Error**: No logs in Cloud Logging

**Causes**:
1. Application not writing to stdout/stderr
2. Logging not enabled
3. Service account lacks logging permissions

**Solutions**:

1. **Write to stdout** (not files):
```javascript
// Correct ✅
console.log('Request received');

// Wrong ❌
fs.appendFileSync('/var/log/app.log', 'Request received');
```

2. **Enable logging in manifest**:
```yaml
monitoring:
  cloudwatch_logs:
    enabled: true
```

3. **Grant logging permissions** to service account:
```bash
gcloud projects add-iam-policy-binding my-project \
  --member="serviceAccount:SA_EMAIL" \
  --role="roles/logging.logWriter"
```

### High Costs

**Issue**: Unexpected high Cloud Run costs

**Common Causes**:
1. Too many `min_instances` (always-on instances)
2. Memory allocation too high
3. Long-running requests with high timeout
4. Too many container instances due to low `max_concurrency`

**Cost Optimization**:

```yaml
# Before (expensive 💰💰💰)
cloud_run:
  cpu: "4"
  memory: "4Gi"
  min_instances: 10
  max_concurrency: 10
  timeout_seconds: 3600

# After (optimized 💰)
cloud_run:
  cpu: "1"
  memory: "512Mi"
  min_instances: 0      # Scale to zero when idle
  max_concurrency: 80   # Higher concurrency = fewer instances
  timeout_seconds: 60   # Shorter timeout
```

**Monitor Costs**:
1. Set up billing alerts in GCP Console
2. Review [Cost Table](https://console.cloud.google.com/billing/linkedaccount)
3. Use [Cloud Billing Reports](https://console.cloud.google.com/billing/reports)

### Cold Start Latency

**Issue**: First request after idle is slow (2-5 seconds)

**Causes**:
- Service scaled to zero (`min_instances: 0`)
- Large container image
- Slow application startup

**Solutions**:

**Option 1: Keep Warm Instances** (costs more but eliminates cold starts)
```yaml
cloud_run:
  min_instances: 1  # At least one instance always running
```

**Option 2: Optimize Container** (reduce cold start time)
```dockerfile
# Use smaller base image
FROM node:18-slim  # Instead of node:18

# Optimize dependencies
RUN npm ci --only=production

# Reduce image layers
COPY . .  # Instead of copying files individually
```

**Option 3: Optimize Application Startup**
```javascript
// Lazy load heavy dependencies
const heavyModule = require('./heavy-module');  // ❌ Loaded at startup

// Better ✅
let heavyModule;
function getHeavyModule() {
  if (!heavyModule) heavyModule = require('./heavy-module');
  return heavyModule;
}
```

## Additional Resources

- [Cloud Run Documentation](https://cloud.google.com/run/docs)
- [Cloud Run Pricing](https://cloud.google.com/run/pricing)
- [Best Practices for Cloud Run](https://cloud.google.com/run/docs/best-practices)
- [Cloud Logging Documentation](https://cloud.google.com/logging/docs)
- [Secret Manager Documentation](https://cloud.google.com/secret-manager/docs)
- [GCP Pricing Calculator](https://cloud.google.com/products/calculator)
- [Cloud Run Samples](https://github.com/GoogleCloudPlatform/cloud-run-samples)

## Need Help?

- Check the [main README](../README.md) for general information
- Review [example manifests](../examples/) for reference configurations
- See [FEATURES.md](./FEATURES.md) for feature comparison
- Report issues at: https://github.com/jvreagan/cloud-deploy/issues
