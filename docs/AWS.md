# AWS Deployment Guide

Complete guide for deploying applications to AWS Elastic Beanstalk using cloud-deploy.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Authentication](#authentication)
- [Manifest Configuration](#manifest-configuration)
- [Required Fields](#required-fields)
- [Optional Fields](#optional-fields)
- [Advanced Configuration](#advanced-configuration)
- [Examples](#examples)
- [Monitoring & Logging](#monitoring--logging)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

cloud-deploy uses **AWS Elastic Beanstalk** to deploy containerized applications. It provides:

- ✅ Automatic application and environment creation
- ✅ Docker container support
- ✅ Auto-detection of solution stacks
- ✅ CloudWatch metrics and logging integration
- ✅ Enhanced health reporting
- ✅ Load balancer and auto-scaling support
- ✅ Rolling deployments with zero downtime
- ✅ Environment variable management
- ✅ IAM role integration

## Prerequisites

1. **AWS Account** with appropriate permissions
2. **AWS CLI configured** (optional but recommended)
3. **Dockerized application** with a `Dockerfile` in your project
4. **Credentials** with the following permissions:
   - `elasticbeanstalk:*`
   - `s3:CreateBucket`, `s3:PutObject`, `s3:GetObject`
   - `ec2:DescribeInstances`, `ec2:DescribeSecurityGroups`
   - `cloudformation:*` (Elastic Beanstalk uses CloudFormation)
   - `iam:PassRole` (if using instance profiles)
   - `cloudwatch:PutMetricData` (if using monitoring)

## Authentication

cloud-deploy supports three authentication methods (in order of preference):

### 1. AWS CLI Credentials (Recommended)

```bash
aws configure
```

This is the **recommended approach** for development. Your credentials are stored securely at `~/.aws/credentials`.

### 2. Environment Variables

```bash
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key
export AWS_REGION=us-east-2  # optional
```

Great for **CI/CD pipelines** where credentials are injected as secrets.

### 3. Manifest File Credentials

```yaml
provider:
  name: aws
  region: us-east-2
  credentials:
    access_key_id: AKIAIOSFODNN7EXAMPLE
    secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

⚠️ **Security Warning**: Never commit credentials to version control! Use this only for testing or when credentials are injected at runtime.

## Manifest Configuration

### Minimal Example

The simplest possible AWS deployment manifest:

```yaml
version: "1.0"

provider:
  name: aws
  region: us-east-2

application:
  name: my-app

environment:
  name: my-app-env

deployment:
  platform: docker
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

This will:
1. Create an application called "my-app"
2. Create a single-instance environment
3. Upload and deploy your Docker container
4. Set up basic health checking

## Required Fields

### Provider Configuration

```yaml
provider:
  name: aws           # Must be "aws"
  region: us-east-2   # AWS region to deploy to
```

**Supported Regions**: Any AWS region that supports Elastic Beanstalk
- Popular: `us-east-1`, `us-east-2`, `us-west-1`, `us-west-2`, `eu-west-1`, `ap-southeast-1`

### Application Configuration

```yaml
application:
  name: my-app        # Unique name for your application (3-100 characters)
  description: "My Application"  # Optional but recommended
```

**Naming Rules**:
- Use lowercase letters, numbers, and hyphens
- Must be unique within your AWS account and region
- Cannot start or end with a hyphen

### Environment Configuration

```yaml
environment:
  name: my-app-env    # Unique environment name (4-40 characters)
  cname: my-app       # Optional: subdomain prefix (creates my-app.us-east-2.elasticbeanstalk.com)
```

**About Environments**:
- Each application can have multiple environments (e.g., dev, staging, prod)
- Environments run independently with their own resources
- The `cname` must be globally unique across all AWS accounts

### Deployment Configuration

```yaml
deployment:
  platform: docker                    # Platform type: docker, nodejs, python, ruby, go, java
  solution_stack: "64bit Amazon Linux 2023 v4.7.2 running Docker"  # Optional: auto-detected if omitted
  source:
    type: local                       # Source type: local, s3, or git
    path: "."                         # Path to your application code
```

**Platform Support**:
- `docker` - Requires a `Dockerfile` in your source directory
- Other platforms require appropriate runtime files (package.json, requirements.txt, etc.)

**Solution Stack Auto-Detection**:
If you omit `solution_stack`, cloud-deploy will automatically select the latest stable stack for your platform on Amazon Linux 2023.

### Instance Configuration

```yaml
instance:
  type: t3.micro                      # EC2 instance type
  environment_type: SingleInstance    # SingleInstance or LoadBalanced
```

**Instance Types**:
- **Development/Testing**: `t3.micro`, `t3.small`, `t3.medium`
- **Production**: `t3.large`, `t3.xlarge`, `m5.large`, `c5.large`
- See [AWS EC2 Instance Types](https://aws.amazon.com/ec2/instance-types/) for full list

**Environment Types**:
- **SingleInstance**: One EC2 instance, no load balancer (cheapest, for dev/test)
- **LoadBalanced**: Auto-scaling group with load balancer (production)

### Health Check Configuration

```yaml
health_check:
  type: basic          # basic or enhanced
  path: /health        # HTTP endpoint for health checks
```

**Health Check Types**:
- **basic**: Simple HTTP health checks (free)
- **enhanced**: Detailed health reporting with CloudWatch metrics (recommended for production)

## Optional Fields

### Environment Variables

Pass configuration to your application:

```yaml
environment_variables:
  NODE_ENV: production
  LOG_LEVEL: info
  DATABASE_URL: postgres://user:pass@host:5432/db
  API_KEY: your-api-key
```

**Access in Your Application**:
```javascript
// Node.js
const nodeEnv = process.env.NODE_ENV;

// Python
import os
node_env = os.environ.get('NODE_ENV')

// Go
nodeEnv := os.Getenv("NODE_ENV")
```

### Resource Tags

Add tags to AWS resources for organization and cost tracking:

```yaml
tags:
  Environment: production
  Project: myapp
  Team: backend
  CostCenter: engineering
```

Tags appear in:
- AWS Console
- Cost Explorer
- CloudWatch dashboards
- Resource Groups

### IAM Configuration

Assign IAM roles to your application instances:

```yaml
iam:
  instance_profile: my-ec2-instance-profile    # EC2 instance profile name
  service_role: aws-elasticbeanstalk-service-role  # Elastic Beanstalk service role
```

**When to Use**:
- Accessing S3 buckets without hardcoded credentials
- Connecting to RDS databases
- Publishing to SNS/SQS
- Reading from Secrets Manager
- Any AWS API calls from your application

**Setup**:
1. Create IAM role in AWS Console
2. Attach policies (e.g., `AmazonS3ReadOnlyAccess`)
3. Create instance profile from the role
4. Reference in manifest

## Advanced Configuration

### Monitoring & CloudWatch Integration

Enable comprehensive monitoring and logging:

```yaml
monitoring:
  # Enable enhanced health reporting (detailed metrics)
  enhanced_health: true

  # Enable CloudWatch custom metrics collection
  cloudwatch_metrics: true

  # Configure CloudWatch Logs streaming
  cloudwatch_logs:
    enabled: true
    retention_days: 7        # 1, 3, 5, 7, 14, 30, 60, 90, 120, 150, 180, 365, 400, 545, 731, 1827, 3653
    stream_logs: true
```

**Enhanced Health Reporting Provides**:
- Overall environment health status
- Request count and latency metrics (p10, p50, p75, p85, p90, p95, p99, p99.9)
- HTTP status code breakdown (2xx, 3xx, 4xx, 5xx)
- Instance health and status

**CloudWatch Metrics Include**:
- `ApplicationRequests2xx`, `ApplicationRequests4xx`, `ApplicationRequests5xx`
- `ApplicationLatencyP50`, `ApplicationLatencyP90`, `ApplicationLatencyP99`
- `EnvironmentHealth` status

**CloudWatch Logs Streams**:
- `/aws/elasticbeanstalk/<env-name>/var/log/eb-docker/containers/eb-current-app/stdouterr.log`
- Application logs from your Docker container
- Elastic Beanstalk platform logs

**Cost Note**: CloudWatch Logs incurs charges based on ingestion and storage. See [CloudWatch Pricing](https://aws.amazon.com/cloudwatch/pricing/).

See [MONITORING.md](MONITORING.md) for complete monitoring documentation.

### Load Balanced Environments

Configure auto-scaling and load balancing for production:

```yaml
instance:
  type: t3.small
  environment_type: LoadBalanced

# Optional: Advanced load balancer settings via environment properties
# Note: Use AWS Console or EB CLI for fine-grained auto-scaling rules
```

**Load Balanced Environments Include**:
- Application Load Balancer (ALB) or Classic Load Balancer
- Auto Scaling Group with min/max instance counts
- Health checks via load balancer
- SSL/TLS termination support
- Multiple availability zones

**Default Auto-Scaling**:
- Min instances: 1
- Max instances: 4
- Scale up: CPU > 80% for 5 minutes
- Scale down: CPU < 20% for 5 minutes

### Credentials in Manifest

For automated deployments where credentials must be in the manifest:

```yaml
provider:
  name: aws
  region: us-east-2
  credentials:
    access_key_id: ${AWS_ACCESS_KEY_ID}        # Use env var substitution
    secret_access_key: ${AWS_SECRET_ACCESS_KEY}
```

⚠️ **Best Practices**:
1. Never commit real credentials
2. Use environment variable substitution
3. Add manifest files with credentials to `.gitignore`
4. Rotate credentials regularly
5. Use IAM roles instead when possible

## Examples

### Example 1: Simple Single-Instance App

```yaml
version: "1.0"

provider:
  name: aws
  region: us-east-2

application:
  name: simple-api
  description: "Simple REST API"

environment:
  name: simple-api-dev
  cname: simple-api-dev

deployment:
  platform: docker
  source:
    type: local
    path: "."

instance:
  type: t3.micro
  environment_type: SingleInstance

health_check:
  type: basic
  path: /health

environment_variables:
  NODE_ENV: development
  PORT: "8080"
```

**Use Case**: Development environment, low traffic, cost-conscious.

### Example 2: Production with Monitoring

```yaml
version: "1.0"

provider:
  name: aws
  region: us-east-2

application:
  name: production-api
  description: "Production REST API"

environment:
  name: production-api-prod
  cname: api

deployment:
  platform: docker
  source:
    type: local
    path: "."

instance:
  type: t3.small
  environment_type: LoadBalanced

health_check:
  type: enhanced
  path: /api/health

monitoring:
  enhanced_health: true
  cloudwatch_metrics: true
  cloudwatch_logs:
    enabled: true
    retention_days: 30
    stream_logs: true

iam:
  instance_profile: myapp-ec2-profile

environment_variables:
  NODE_ENV: production
  LOG_LEVEL: warn
  DATABASE_URL: postgres://prod-db.example.com:5432/prod

tags:
  Environment: production
  Project: myapp
  ManagedBy: cloud-deploy
```

**Use Case**: Production environment with monitoring, auto-scaling, and IAM roles.

### Example 3: Staging Environment

```yaml
version: "1.0"

provider:
  name: aws
  region: us-west-2

application:
  name: myapp
  description: "My Application"

environment:
  name: myapp-staging
  cname: myapp-staging

deployment:
  platform: docker
  solution_stack: "64bit Amazon Linux 2023 v4.7.2 running Docker"
  source:
    type: local
    path: "."

instance:
  type: t3.small
  environment_type: LoadBalanced

health_check:
  type: enhanced
  path: /health

monitoring:
  enhanced_health: true
  cloudwatch_logs:
    enabled: true
    retention_days: 7

environment_variables:
  NODE_ENV: staging
  API_ENDPOINT: https://staging-api.example.com

tags:
  Environment: staging
  Team: qa
```

**Use Case**: Pre-production testing with enhanced monitoring but shorter log retention.

## Best Practices

### 1. Environment Naming Strategy

```
<app-name>-<environment>
Examples:
  - myapp-dev
  - myapp-staging
  - myapp-production
```

### 2. Resource Sizing

| Environment | Instance Type | Environment Type | When to Use |
|-------------|---------------|------------------|-------------|
| Development | t3.micro | SingleInstance | Local testing, low traffic |
| Staging | t3.small | LoadBalanced | Pre-production testing |
| Production | t3.medium+ | LoadBalanced | High traffic, HA required |

### 3. Health Check Endpoints

Create a dedicated health check endpoint in your application:

```javascript
// Node.js/Express example
app.get('/health', (req, res) => {
  res.status(200).json({ status: 'healthy', timestamp: Date.now() });
});
```

Requirements:
- Return 200 status code when healthy
- Response time < 2 seconds
- Don't perform expensive operations
- Check critical dependencies (database, cache, etc.)

### 4. Environment Variables

**Do**:
- ✅ Use environment variables for configuration
- ✅ Store secrets in AWS Secrets Manager, reference via IAM role
- ✅ Use different values per environment

**Don't**:
- ❌ Store sensitive data directly in manifest (use Secrets Manager)
- ❌ Hardcode values in application code
- ❌ Use the same values for dev and prod

### 5. Monitoring Setup

**Development**:
```yaml
monitoring:
  enhanced_health: false  # Save costs
```

**Production**:
```yaml
monitoring:
  enhanced_health: true
  cloudwatch_metrics: true
  cloudwatch_logs:
    enabled: true
    retention_days: 30
```

### 6. Deployment Workflow

```bash
# 1. Test locally with Docker
docker build -t myapp .
docker run -p 8080:8080 myapp

# 2. Deploy to development
cloud-deploy -command deploy -manifest manifest-dev.yaml

# 3. Test development deployment
cloud-deploy -command status -manifest manifest-dev.yaml
curl https://myapp-dev.us-east-2.elasticbeanstalk.com/health

# 4. Deploy to staging
cloud-deploy -command deploy -manifest manifest-staging.yaml

# 5. Run integration tests against staging

# 6. Deploy to production
cloud-deploy -command deploy -manifest manifest-prod.yaml
```

### 7. Cost Optimization

**Development**:
- Use `t3.micro` instances
- Use `SingleInstance` environment
- Disable enhanced monitoring
- Short log retention (3-7 days)
- **Stop** when not in use: `cloud-deploy -command stop`

**Production**:
- Right-size instances based on metrics
- Use Reserved Instances for steady workloads
- Enable auto-scaling to handle variable traffic
- Set appropriate min/max instance counts

## Troubleshooting

### Deployment Fails with "Service:AmazonElasticLoadBalancing"

**Problem**: Load balancer creation failed.

**Solution**: Check VPC and subnet configuration. Elastic Beanstalk requires at least two subnets in different availability zones for load-balanced environments.

### Application Not Responding to Health Checks

**Problem**: Environment shows "Degraded" status.

**Causes**:
1. Application not listening on correct port
2. Health check path returns non-200 status
3. Application takes too long to start

**Solutions**:
```yaml
# Ensure your app listens on port 80 (for Docker single-container)
# Or configure the correct port in Dockerfile
EXPOSE 8080

# Verify health check endpoint works
curl http://localhost:8080/health

# Check application logs
cloud-deploy -command status -manifest manifest.yaml
# Then view logs in AWS Console
```

### "Timeout Waiting for Environment to be Ready"

**Problem**: Deployment times out after 15 minutes.

**Causes**:
1. Docker image build is slow (large image, many layers)
2. Application startup is slow
3. Health check failing

**Solutions**:
1. Optimize Docker image:
   ```dockerfile
   # Use multi-stage builds
   FROM node:18 AS builder
   WORKDIR /app
   COPY package*.json ./
   RUN npm ci --production

   FROM node:18-slim
   COPY --from=builder /app/node_modules ./node_modules
   COPY . .
   CMD ["node", "server.js"]
   ```

2. Check CloudWatch Logs for errors
3. Increase timeout (requires code modification)

### "InvalidParameterValue: Namespace 'aws:autoscaling:launchconfiguration'"

**Problem**: Invalid instance type or configuration option.

**Solution**:
- Verify instance type is available in your region
- Check solution stack supports your configuration
- Update to latest solution stack

### S3 Bucket Already Exists

**Problem**: Deployment fails with "BucketAlreadyExists" error.

**Solution**: Elastic Beanstalk creates an S3 bucket per application. If reusing an application name, ensure the old bucket is deleted or the application was properly cleaned up.

### Insufficient Permissions

**Problem**: Access denied errors during deployment.

**Solution**: Ensure IAM user/role has these permissions:
```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": [
      "elasticbeanstalk:*",
      "s3:*",
      "ec2:*",
      "cloudformation:*",
      "autoscaling:*",
      "cloudwatch:*",
      "logs:*",
      "iam:PassRole"
    ],
    "Resource": "*"
  }]
}
```

### Environment Variables Not Available

**Problem**: Application can't access environment variables.

**Verification**:
1. SSH into instance: `eb ssh <environment-name>`
2. Check environment variables: `sudo /opt/elasticbeanstalk/bin/get-config environment`

**Solutions**:
- Verify variables are in manifest under `environment_variables`
- Check for typos in variable names
- Redeploy after changing variables

## Additional Resources

- [AWS Elastic Beanstalk Documentation](https://docs.aws.amazon.com/elasticbeanstalk/)
- [Supported Platforms](https://docs.aws.amazon.com/elasticbeanstalk/latest/platforms/platforms-supported.html)
- [CloudWatch Monitoring Guide](./MONITORING.md)
- [AWS Pricing Calculator](https://calculator.aws/)
- [Elastic Beanstalk Pricing](https://aws.amazon.com/elasticbeanstalk/pricing/)

## Need Help?

- Check the [main README](../README.md) for general information
- Review [example manifests](../examples/) for reference configurations
- Report issues at: https://github.com/jvreagan/cloud-deploy/issues
