# Multi-Cloud Example Application

This is a complete example of a multi-cloud active/active deployment using cloud-deploy and Cloudflare.

## What This Demonstrates

- Same Docker application deployed to AWS and GCP
- Health check endpoint for load balancer monitoring
- Automatic failover between cloud providers
- Environment-aware application (knows which cloud it's running on)

## Quick Start

### 1. Deploy to AWS

```bash
# From this directory
cloud-deploy -manifest manifests/aws-production.yaml -command deploy
```

### 2. Deploy to GCP

Update `manifests/gcp-production.yaml` with your GCP project details, then:

```bash
cloud-deploy -manifest manifests/gcp-production.yaml -command deploy
```

### 3. Set Up Cloudflare

Follow the complete guide in [docs/MULTI_CLOUD.md](../../docs/MULTI_CLOUD.md#step-5-set-up-cloudflare-load-balancing)

## Testing Locally

```bash
# Install dependencies
npm install

# Run locally
npm start

# Test health endpoint
curl http://localhost:8080/health
```

## Application Endpoints

- `GET /` - Welcome message with cloud info
- `GET /health` - Health check (returns 200 with cloud info)
- `GET /info` - Detailed application information

## Files

- `server.js` - Node.js application with Express
- `Dockerfile` - Docker configuration
- `package.json` - Node.js dependencies
- `manifests/aws-production.yaml` - AWS deployment manifest
- `manifests/gcp-production.yaml` - GCP deployment manifest

## Full Documentation

See [Multi-Cloud Deployment Guide](../../docs/MULTI_CLOUD.md) for complete setup instructions.
