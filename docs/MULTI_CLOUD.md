# Multi-Cloud Active/Active Deployment Guide

This guide shows you how to deploy the same application to both AWS and GCP simultaneously, with automatic failover using Cloudflare as your front door.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [What You'll Build](#what-youll-build)
- [Prerequisites](#prerequisites)
- [Step 1: Create Your Application](#step-1-create-your-application)
- [Step 2: Create Deployment Manifests](#step-2-create-deployment-manifests)
- [Step 3: Deploy to AWS](#step-3-deploy-to-aws)
- [Step 4: Deploy to GCP](#step-4-deploy-to-gcp)
- [Step 5: Set Up Cloudflare Load Balancing](#step-5-set-up-cloudflare-load-balancing)
- [Step 6: Test Failover](#step-6-test-failover)
- [Step 7: Automate with GitHub Actions](#step-7-automate-with-github-actions)
- [Monitoring and Operations](#monitoring-and-operations)
- [Cost Breakdown](#cost-breakdown)
- [Troubleshooting](#troubleshooting)

## Architecture Overview

```
                         Internet Users
                               ↓
                    ┌──────────────────────┐
                    │   myapp.com          │
                    │   (Cloudflare)       │
                    │   Health Checks      │
                    │   Auto Failover      │
                    └──────────────────────┘
                         ↙           ↘
              ┌──────────┐         ┌──────────┐
              │   AWS    │         │   GCP    │
              │ Primary  │         │ Secondary│
              │ Active   │         │ Active   │
              └──────────┘         └──────────┘
           Elastic Beanstalk    Cloud Run
         us-east-1              us-central1
```

**How it works:**
1. Users access your app at `myapp.com`
2. Cloudflare performs health checks every 60 seconds
3. Traffic routes to healthy cloud provider
4. If AWS goes down, traffic automatically fails over to GCP
5. When AWS recovers, traffic can fail back automatically

## What You'll Build

By the end of this guide, you'll have:

- ✅ Same Docker application running on AWS and GCP
- ✅ Health check endpoint for monitoring
- ✅ Cloudflare load balancer with automatic failover
- ✅ Custom domain routing to both clouds
- ✅ GitHub Actions workflow for automated multi-cloud deployments
- ✅ Real-time health monitoring

**Total setup time:** ~30 minutes
**Monthly cost:** ~$5-10 (can run on free tiers + $5 Cloudflare)

## Prerequisites

### Required:
- ✅ AWS account with credentials configured
- ✅ GCP account with billing enabled
- ✅ Domain name (for Cloudflare)
- ✅ cloud-deploy installed locally

### Install cloud-deploy:
```bash
# macOS/Linux
brew tap jvreagan/tap
brew install cloud-deploy

# Or download from releases
curl -L https://github.com/jvreagan/cloud-deploy/releases/latest/download/cloud-deploy_Linux_x86_64.tar.gz | tar -xz
sudo mv cloud-deploy /usr/local/bin/
```

### Verify installation:
```bash
cloud-deploy -version
```

## Step 1: Create Your Application

Let's create a simple Node.js application with a health check endpoint.

### Create project directory:
```bash
mkdir multi-cloud-app
cd multi-cloud-app
```

### Create `package.json`:
```json
{
  "name": "multi-cloud-app",
  "version": "1.0.0",
  "description": "Multi-cloud demo application",
  "main": "server.js",
  "scripts": {
    "start": "node server.js"
  },
  "dependencies": {
    "express": "^4.18.2"
  }
}
```

### Create `server.js`:
```javascript
const express = require('express');
const app = express();
const PORT = process.env.PORT || 8080;

// Cloud provider identifier (set via environment variable)
const CLOUD_PROVIDER = process.env.CLOUD_PROVIDER || 'unknown';
const REGION = process.env.REGION || 'unknown';

// Store startup time
const startTime = new Date();

// Health check endpoint (required for load balancer)
app.get('/health', (req, res) => {
  const uptime = Math.floor((new Date() - startTime) / 1000);

  res.status(200).json({
    status: 'healthy',
    cloud: CLOUD_PROVIDER,
    region: REGION,
    uptime: `${uptime} seconds`,
    timestamp: new Date().toISOString(),
    version: '1.0.0'
  });
});

// Root endpoint
app.get('/', (req, res) => {
  res.json({
    message: 'Hello from Multi-Cloud!',
    cloud: CLOUD_PROVIDER,
    region: REGION,
    uptime: Math.floor((new Date() - startTime) / 1000),
    endpoints: {
      health: '/health',
      info: '/info'
    }
  });
});

// Info endpoint with detailed information
app.get('/info', (req, res) => {
  res.json({
    application: 'multi-cloud-app',
    version: '1.0.0',
    cloud: CLOUD_PROVIDER,
    region: REGION,
    environment: process.env.NODE_ENV || 'production',
    nodeVersion: process.version,
    platform: process.platform,
    uptime: Math.floor((new Date() - startTime) / 1000),
    memory: {
      rss: `${Math.round(process.memoryUsage().rss / 1024 / 1024)}MB`,
      heapTotal: `${Math.round(process.memoryUsage().heapTotal / 1024 / 1024)}MB`,
      heapUsed: `${Math.round(process.memoryUsage().heapUsed / 1024 / 1024)}MB`
    }
  });
});

// Start server
app.listen(PORT, () => {
  console.log(`Server running on port ${PORT}`);
  console.log(`Cloud: ${CLOUD_PROVIDER}`);
  console.log(`Region: ${REGION}`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('SIGTERM received, shutting down gracefully...');
  process.exit(0);
});
```

### Create `Dockerfile`:
```dockerfile
FROM node:18-alpine

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci --only=production

# Copy application code
COPY server.js ./

# Expose port
EXPOSE 8080

# Set default environment variables
ENV PORT=8080
ENV NODE_ENV=production

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD node -e "require('http').get('http://localhost:8080/health', (r) => { process.exit(r.statusCode === 200 ? 0 : 1); });"

# Start application
CMD ["npm", "start"]
```

### Create `.dockerignore`:
```
node_modules
npm-debug.log
.git
.gitignore
README.md
.env
```

### Test locally:
```bash
# Install dependencies
npm install

# Run locally
node server.js

# Test health check (in another terminal)
curl http://localhost:8080/health
```

## Step 2: Create Deployment Manifests

### Create manifests directory:
```bash
mkdir manifests
```

### Create `manifests/aws-production.yaml`:
```yaml
version: "1.0"

provider:
  name: aws
  region: us-east-1

application:
  name: multi-cloud-app
  description: "Multi-cloud demo application - AWS"

environment:
  name: multi-cloud-app-aws
  cname: multi-cloud-app-aws  # Creates: multi-cloud-app-aws.us-east-1.elasticbeanstalk.com

deployment:
  platform: docker
  solution_stack: "64bit Amazon Linux 2023 v4.7.2 running Docker"
  source:
    type: local
    path: .

instance:
  type: t3.micro
  environment_type: SingleInstance

health_check:
  type: enhanced
  path: /health
  interval: 30
  timeout: 5
  healthy_threshold: 3
  unhealthy_threshold: 5

# Set cloud provider environment variable
environment_variables:
  CLOUD_PROVIDER: "aws"
  REGION: "us-east-1"
  NODE_ENV: "production"

monitoring:
  enhanced_health: true
  cloudwatch_metrics: true

tags:
  Environment: production
  Cloud: aws
  Application: multi-cloud-app
```

### Create `manifests/gcp-production.yaml`:
```yaml
version: "1.0"

provider:
  name: gcp
  region: us-central1

  # IMPORTANT: Update these with your GCP details
  project_id: YOUR_GCP_PROJECT_ID
  billing_account_id: XXXXXX-XXXXXX-XXXXXX

  credentials:
    service_account_key_path: "/path/to/your-service-account-key.json"

  public_access: true

application:
  name: multi-cloud-app
  description: "Multi-cloud demo application - GCP"

deployment:
  platform: docker
  source:
    type: local
    path: .

health_check:
  type: basic
  path: /health

# Cloud Run resource configuration
cloud_run:
  cpu: 1
  memory: "512Mi"
  max_instances: 10
  min_instances: 1  # Keep 1 instance always warm
  timeout: 300
  concurrency: 80

# Set cloud provider environment variable
environment_variables:
  CLOUD_PROVIDER: "gcp"
  REGION: "us-central1"
  NODE_ENV: "production"

tags:
  environment: production
  cloud: gcp
  application: multi-cloud-app
```

**Important:** Update the GCP manifest with your:
- `project_id` - Your GCP project ID
- `billing_account_id` - Your GCP billing account
- `service_account_key_path` - Path to your service account JSON key

## Step 3: Deploy to AWS

### Configure AWS credentials:
```bash
# Option 1: Using AWS CLI
aws configure

# Option 2: Using environment variables
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"
```

### Deploy to AWS:
```bash
cloud-deploy -manifest manifests/aws-production.yaml -command deploy
```

**What happens:**
1. ✅ Creates application in Elastic Beanstalk
2. ✅ Creates environment with t3.micro instance
3. ✅ Packages your code into a Docker container
4. ✅ Deploys to AWS
5. ✅ Configures health checks
6. ✅ Returns URL (e.g., `multi-cloud-app-aws.us-east-1.elasticbeanstalk.com`)

**Save your AWS URL!** You'll need it for Cloudflare setup.

### Test AWS deployment:
```bash
# Replace with your actual URL
AWS_URL="multi-cloud-app-aws.us-east-1.elasticbeanstalk.com"

# Test health endpoint
curl https://$AWS_URL/health

# Expected response:
# {
#   "status": "healthy",
#   "cloud": "aws",
#   "region": "us-east-1",
#   "uptime": "42 seconds",
#   "timestamp": "2025-01-15T10:30:00.000Z"
# }
```

## Step 4: Deploy to GCP

### Set up GCP credentials:

If you haven't already, follow the [GCP Authentication Guide](GCP.md#authentication) to:
1. Create a service account
2. Download the JSON key file
3. Get your billing account ID

### Update GCP manifest:

Edit `manifests/gcp-production.yaml` and update:
```yaml
provider:
  project_id: my-multi-cloud-project  # Your project ID
  billing_account_id: 123456-123456-123456  # Your billing ID
  credentials:
    service_account_key_path: "./gcp-service-account-key.json"
```

### Deploy to GCP:
```bash
cloud-deploy -manifest manifests/gcp-production.yaml -command deploy
```

**What happens:**
1. ✅ Creates GCP project (if it doesn't exist)
2. ✅ Links billing account
3. ✅ Enables Cloud Run and Cloud Build APIs
4. ✅ Builds Docker image using Cloud Build
5. ✅ Deploys to Cloud Run
6. ✅ Returns URL (e.g., `multi-cloud-app-xxxxx-uc.a.run.app`)

**Save your GCP URL!** You'll need it for Cloudflare setup.

### Test GCP deployment:
```bash
# Replace with your actual URL
GCP_URL="multi-cloud-app-xxxxx-uc.a.run.app"

# Test health endpoint
curl https://$GCP_URL/health

# Expected response:
# {
#   "status": "healthy",
#   "cloud": "gcp",
#   "region": "us-central1",
#   "uptime": "15 seconds",
#   "timestamp": "2025-01-15T10:30:00.000Z"
# }
```

**You now have the same app running on both AWS and GCP!** ✅

## Step 5: Set Up Cloudflare Load Balancing

Now let's set up Cloudflare to route traffic between AWS and GCP with automatic failover.

### 5.1: Add Your Domain to Cloudflare

1. **Sign up for Cloudflare:** https://dash.cloudflare.com/sign-up
2. **Add your domain:**
   - Click "Add a Site"
   - Enter your domain (e.g., `myapp.com`)
   - Choose the Free plan
   - Follow instructions to update nameservers at your domain registrar

3. **Wait for activation** (usually 5-30 minutes)

### 5.2: Upgrade to Cloudflare Load Balancing

Load Balancing requires a paid add-on:

1. Go to **Traffic** → **Load Balancing**
2. Click **Purchase Load Balancing**
3. Choose **$5/month plan** (500,000 queries/month, 2 origin pools)
4. Complete purchase

### 5.3: Create Origin Pools

**Create AWS Origin Pool:**

1. Go to **Traffic** → **Load Balancing** → **Manage Pools**
2. Click **Create**
3. Configure:
   - **Pool Name:** `aws-us-east-1`
   - **Origin Name:** `aws-primary`
   - **Origin Address:** `multi-cloud-app-aws.us-east-1.elasticbeanstalk.com`
   - **Weight:** 1
   - **Health Check Region:** All regions
4. Configure Health Check:
   - **Monitor Type:** HTTPS
   - **Path:** `/health`
   - **Interval:** 60 seconds
   - **Timeout:** 5 seconds
   - **Retries:** 2
   - **Expected Status Code:** 200
5. Click **Save**

**Create GCP Origin Pool:**

1. Click **Create** again
2. Configure:
   - **Pool Name:** `gcp-us-central1`
   - **Origin Name:** `gcp-secondary`
   - **Origin Address:** `multi-cloud-app-xxxxx-uc.a.run.app`
   - **Weight:** 1
   - **Health Check Region:** All regions
3. Use same health check settings as AWS
4. Click **Save**

### 5.4: Create Load Balancer

1. Go to **Traffic** → **Load Balancing** → **Create Load Balancer**
2. Configure:
   - **Hostname:** `app.myapp.com` (or just `myapp.com`)
   - **Enable:** Toggle ON (orange cloud)

3. **Add Origin Pools:**
   - **Region:** Default (serves all traffic)
   - **Origin Pools:**
     - Add `aws-us-east-1` (Priority 0)
     - Add `gcp-us-central1` (Priority 1)

4. **Traffic Steering:**
   - Choose **Failover** (routes to AWS, fails over to GCP if unhealthy)
   - Or choose **Random** for true active/active load balancing

5. **Session Affinity:** Off (or configure based on your needs)

6. **Configure Health Check:**
   - Inherit from origin pools (already configured)

7. **Review and Deploy:**
   - Review settings
   - Click **Save and Deploy**

### 5.5: Verify DNS

Your load balancer is now live at `app.myapp.com`!

Test it:
```bash
# Check DNS resolution
dig app.myapp.com

# Test your application
curl https://app.myapp.com/health

# You should see response from AWS (primary)
# {
#   "status": "healthy",
#   "cloud": "aws",
#   ...
# }
```

## Step 6: Test Failover

Let's test that failover actually works!

### Test 1: Manual Failover via Cloudflare

1. Go to **Traffic** → **Load Balancing** → **Manage Pools**
2. Click on **aws-us-east-1** pool
3. Click **Disable** to simulate AWS being down
4. Wait 60-90 seconds for health checks to detect failure
5. Test your app:
   ```bash
   curl https://app.myapp.com/health
   # Should now show "cloud": "gcp"
   ```
6. Re-enable AWS pool and traffic will fail back

### Test 2: Stop AWS Environment

```bash
# Stop AWS environment
cloud-deploy -manifest manifests/aws-production.yaml -command stop

# Wait 60-90 seconds
sleep 90

# Test - should now route to GCP
curl https://app.myapp.com/health
# Response: "cloud": "gcp"

# Restart AWS
cloud-deploy -manifest manifests/aws-production.yaml -command deploy
```

### Monitor Health Checks

View real-time health check status:

1. Go to **Traffic** → **Load Balancing**
2. Click on your load balancer
3. View **Pool Health** dashboard
4. See status of both AWS and GCP origins

## Step 7: Automate with GitHub Actions

Let's automate deployments to both clouds with a single workflow.

### Create `.github/workflows/deploy-multi-cloud.yml`:

```yaml
name: Deploy Multi-Cloud

on:
  push:
    branches: [main]
  workflow_dispatch:
    inputs:
      deploy_target:
        description: 'Deploy to'
        required: true
        type: choice
        options:
          - both
          - aws
          - gcp

jobs:
  deploy-aws:
    name: Deploy to AWS
    if: github.event_name == 'push' || github.event.inputs.deploy_target == 'both' || github.event.inputs.deploy_target == 'aws'
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Install cloud-deploy
        run: |
          curl -L https://github.com/jvreagan/cloud-deploy/releases/latest/download/cloud-deploy_Linux_x86_64.tar.gz | tar -xz
          sudo mv cloud-deploy /usr/local/bin/
          cloud-deploy -version

      - name: Deploy to AWS
        run: |
          cloud-deploy \
            -manifest manifests/aws-production.yaml \
            -command deploy
        env:
          AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
          AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          AWS_REGION: us-east-1

      - name: Test AWS deployment
        run: |
          echo "Testing AWS deployment..."
          sleep 30
          curl -f https://multi-cloud-app-aws.us-east-1.elasticbeanstalk.com/health

  deploy-gcp:
    name: Deploy to GCP
    if: github.event_name == 'push' || github.event.inputs.deploy_target == 'both' || github.event.inputs.deploy_target == 'gcp'
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Install cloud-deploy
        run: |
          curl -L https://github.com/jvreagan/cloud-deploy/releases/latest/download/cloud-deploy_Linux_x86_64.tar.gz | tar -xz
          sudo mv cloud-deploy /usr/local/bin/
          cloud-deploy -version

      - name: Deploy to GCP
        run: |
          cloud-deploy \
            -manifest manifests/gcp-production.yaml \
            -command deploy
        env:
          GCP_PROJECT_ID: ${{ secrets.GCP_PROJECT_ID }}
          GCP_CREDENTIALS: ${{ secrets.GCP_CREDENTIALS }}
          GCP_BILLING_ACCOUNT_ID: ${{ secrets.GCP_BILLING_ACCOUNT_ID }}

      - name: Test GCP deployment
        run: |
          echo "Testing GCP deployment..."
          sleep 30
          # Update with your actual GCP URL
          curl -f https://multi-cloud-app-xxxxx-uc.a.run.app/health

  notify:
    name: Notify Deployment Complete
    needs: [deploy-aws, deploy-gcp]
    if: always()
    runs-on: ubuntu-latest

    steps:
      - name: Deployment summary
        run: |
          echo "Multi-cloud deployment complete!"
          echo "AWS Status: ${{ needs.deploy-aws.result }}"
          echo "GCP Status: ${{ needs.deploy-gcp.result }}"
```

### Configure GitHub Secrets:

Go to your repository: **Settings** → **Secrets and variables** → **Actions**

Add these secrets:
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `GCP_PROJECT_ID`
- `GCP_CREDENTIALS` (entire JSON content)
- `GCP_BILLING_ACCOUNT_ID`

### Deploy!

```bash
git add .
git commit -m "Add multi-cloud deployment"
git push origin main
```

The workflow will automatically deploy to both AWS and GCP!

## Monitoring and Operations

### Check Deployment Status

**AWS:**
```bash
cloud-deploy -manifest manifests/aws-production.yaml -command status
```

**GCP:**
```bash
cloud-deploy -manifest manifests/gcp-production.yaml -command status
```

### View Cloudflare Analytics

1. Go to **Traffic** → **Load Balancing**
2. Click on your load balancer
3. View:
   - Request distribution (AWS vs GCP)
   - Health check history
   - Failover events
   - Response times by origin

### Set Up Alerts

**Cloudflare Notifications:**

1. Go to **Notifications**
2. Create alert for "Load Balancing - Pool Toggle Alert"
3. Get notified when a pool becomes unhealthy

### Logs

**AWS Logs:**
```bash
# View recent logs in AWS Console
# Or use AWS CLI:
aws logs tail /aws/elasticbeanstalk/multi-cloud-app-aws/var/log/eb-docker/containers/eb-current-app --follow
```

**GCP Logs:**
```bash
# In GCP Console: Cloud Run → Service → Logs
# Or use gcloud:
gcloud run services logs read multi-cloud-app --region=us-central1 --limit=50
```

## Cost Breakdown

### Monthly Costs:

| Service | Cost | Notes |
|---------|------|-------|
| **AWS Elastic Beanstalk** | ~$15 | t3.micro instance (always on) |
| **GCP Cloud Run** | ~$5 | Min 1 instance + requests |
| **Cloudflare Load Balancing** | $5 | 500K queries/month |
| **Total** | **~$25/month** | |

### Cost Optimization:

**Use Free Tiers for Testing:**
- AWS: 750 hours/month free tier (first year)
- GCP: 2 million requests/month free, 360K GiB-seconds free
- Cloudflare: Free tier for DNS (manual failover)

**Production Optimization:**
- Scale down GCP min_instances to 0 during off-hours
- Use AWS t3.nano instead of t3.micro ($3.80/month vs $8/month)
- Implement request caching to reduce load

## Troubleshooting

### Issue: Health checks failing

**Check health endpoint:**
```bash
curl -v https://your-app-url/health
```

**Common causes:**
- Health endpoint returning wrong status code
- Timeout too short (increase to 10 seconds)
- SSL certificate issues
- Firewall blocking health check IPs

**Fix:** Ensure `/health` returns 200 status within timeout

### Issue: Traffic not failing over

**Verify:**
1. Origin pools are configured correctly
2. Health checks are passing
3. Failover is set to "Priority" mode
4. Wait at least 2-3 health check intervals

**Test manually:** Disable primary pool in Cloudflare dashboard

### Issue: One cloud not deploying

**AWS deployment fails:**
- Check AWS credentials
- Verify region is correct
- Check AWS service quotas

**GCP deployment fails:**
- Verify billing is enabled
- Check service account permissions
- Ensure project ID is unique

### Issue: High costs

**Check:**
- GCP min_instances (set to 0 for lower cost)
- AWS instance type (use t3.nano for testing)
- Cloudflare request volume (may need higher tier)

**Monitor:** Set up billing alerts in AWS and GCP consoles

## Next Steps

### Add More Regions

Deploy to multiple regions in each cloud:

```bash
# AWS us-west-2
manifests/aws-us-west-2.yaml

# GCP europe-west1
manifests/gcp-europe-west1.yaml

# Add all to Cloudflare pools
```

### Add Geographic Routing

Configure Cloudflare to route by user location:
- US traffic → AWS us-east-1
- Europe traffic → GCP europe-west1
- Asia traffic → AWS ap-southeast-1

### Add More Clouds

- Azure Container Instances
- Oracle Cloud Container Instances
- DigitalOcean App Platform

### Implement Blue/Green Deployments

Deploy new versions to secondary cloud first, test, then promote to primary.

### Add Monitoring

- Datadog / New Relic for APM
- PagerDuty for alerts
- Grafana for dashboards

## Summary

You now have:

✅ **Same application running on AWS and GCP**
✅ **Cloudflare load balancing with automatic failover**
✅ **Health monitoring and alerts**
✅ **Automated deployments via GitHub Actions**
✅ **Protection against cloud provider outages**
✅ **Single URL for users (app.myapp.com)**

Your application will automatically failover between clouds with zero downtime!

**Questions?** Open an issue at https://github.com/jvreagan/cloud-deploy/issues

---

**Built with ❤️ using cloud-deploy**
