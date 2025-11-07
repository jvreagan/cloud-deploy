# Datadog APM Integration

This document explains how to integrate Datadog Application Performance Monitoring (APM) with your cloud-deploy applications across AWS, Azure, and GCP.

## Overview

cloud-deploy supports Datadog APM integration using the **serverless-init wrapper pattern**, which provides automatic trace collection with zero code changes. This approach works consistently across all supported cloud providers.

**Status:** ✅ Production Ready (as of v0.8.0)

## Table of Contents

- [Quick Start](#quick-start)
- [Serverless-Init Pattern](#serverless-init-pattern)
- [Configuration](#configuration)
- [Multi-Cloud Deployment](#multi-cloud-deployment)
- [Span-Based Metrics](#span-based-metrics)
- [Examples](#examples)
- [Troubleshooting](#troubleshooting)

## Quick Start

### 1. Build Application with Orchestrion

For Go applications, use Datadog Orchestrion for compile-time instrumentation:

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Copy dependencies
COPY go.mod go.sum ./
RUN go mod download

# Install Orchestrion
RUN go install github.com/DataDog/orchestrion@latest

# Copy source and build with instrumentation
COPY . .
RUN CGO_ENABLED=0 GOOS=linux /go/bin/orchestrion go build -o main main.go

# Use serverless-init wrapper
FROM datadog/serverless-init:latest
COPY --from=builder /app/main /app/main
CMD ["/app/main"]
```

### 2. Configure Deployment Manifest

Add Datadog configuration to your deployment manifest:

```yaml
version: "1.0"

containers:
  - name: myapp
    image: "myapp:latest"
    ports:
      - container: 80
      - container: 443
    environment:
      ENV: production

      # Datadog configuration
      DD_API_KEY: "${DD_API_KEY}"
      DD_SITE: "us5.datadoghq.com"
      DD_SERVICE: "myapp"
      DD_ENV: "production"
      DD_VERSION: "1.0.0"
      DD_TAGS: "cloud:aws,team:platform"

      # Enable APM tracing
      DD_TRACE_ENABLED: "true"
      DD_TRACE_SAMPLE_RATE: "1.0"

provider:
  name: aws
  region: us-east-1
  credentials:
    source: environment
```

### 3. Deploy

```bash
export DD_API_KEY="your-datadog-api-key"
cloud-deploy -command deploy -manifest deploy-manifest.yaml
```

Traces will automatically appear in your Datadog APM dashboard.

## Serverless-Init Pattern

### What is serverless-init?

`datadog/serverless-init` is Datadog's official wrapper for containerized and serverless environments. It wraps your application binary and provides a built-in trace agent.

**Benefits:**
- ✅ Single container (simpler deployment)
- ✅ Works across all clouds (AWS, Azure, GCP)
- ✅ Zero application code changes
- ✅ Automatic trace collection
- ✅ Lower resource usage than sidecar pattern
- ✅ Official Datadog solution

### How It Works

1. **Application Build:** Your app is compiled with Orchestrion for automatic instrumentation
2. **Container Wrapper:** serverless-init wraps your binary and starts a trace agent
3. **Trace Collection:** Traces are automatically sent to Datadog
4. **No Code Changes:** Application code remains unchanged

## Configuration

### Required Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `DD_API_KEY` | Datadog API key | `"96445a08..."` |
| `DD_SITE` | Datadog site | `"us5.datadoghq.com"` |

### Recommended Environment Variables

| Variable | Description | Example | Default |
|----------|-------------|---------|---------|
| `DD_SERVICE` | Service name | `"myapp"` | - |
| `DD_ENV` | Environment | `"production"` | - |
| `DD_VERSION` | App version | `"1.0.0"` | - |
| `DD_TAGS` | Custom tags | `"cloud:aws,team:platform"` | - |

### APM Configuration

| Variable | Description | Example | Default |
|----------|-------------|---------|---------|
| `DD_TRACE_ENABLED` | Enable tracing | `"true"` | `true` |
| `DD_TRACE_SAMPLE_RATE` | Sample rate | `"1.0"` (100%) | `1.0` |
| `DD_TRACE_DEBUG` | Debug logging | `"true"` | `false` |

### Datadog Sites

Choose the correct site based on your Datadog account region:

| Region | DD_SITE Value |
|--------|---------------|
| US1 | `datadoghq.com` |
| US3 | `us3.datadoghq.com` |
| US5 | `us5.datadoghq.com` |
| EU1 | `datadoghq.eu` |
| AP1 | `ap1.datadoghq.com` |

## Multi-Cloud Deployment

### Using Cloud Tags for Visibility

When deploying to multiple clouds, use the `DD_TAGS` environment variable to identify which cloud served each request:

**AWS Deployment:**
```yaml
environment:
  DD_TAGS: "cloud:aws,platform:elastic-beanstalk,region:us-east-1"
```

**Azure Deployment:**
```yaml
environment:
  DD_TAGS: "cloud:azure,platform:container-instances,region:eastus"
```

**GCP Deployment:**
```yaml
environment:
  DD_TAGS: "cloud:gcp,platform:cloud-run,region:us-central1"
```

### Viewing Multi-Cloud Traces

In Datadog APM, filter traces by cloud:
- `cloud:aws` - Traces from AWS deployments
- `cloud:azure` - Traces from Azure deployments
- `cloud:gcp` - Traces from GCP deployments

## Span-Based Metrics

Span tags are only available in Traces Explorer by default. To use them in dashboards and monitors, you need to create span-based metrics.

### Creating Span-Based Metrics

1. **Navigate to APM Settings:**
   - Go to APM → Setup & Configuration → Generate Metrics

2. **Create New Metric:**
   - Click "New Metric"
   - Filter: `service:your-service-name`
   - Group by: Add tags you want (e.g., `cloud`)
   - Metric name: `your_service.requests.by_cloud`

3. **Use in Dashboards:**
   ```
   sum:your_service.requests.by_cloud{*} by {cloud}.as_count()
   ```

### Example: Cloud Distribution Dashboard

Create a dashboard showing traffic distribution across clouds:

**Timeseries Widget:**
```
sum:myapp.requests.by_cloud{*} by {cloud}.as_count()
```

**Query Value Widgets:**
- AWS: `sum:myapp.requests.by_cloud{cloud:aws}.as_count()`
- Azure: `sum:myapp.requests.by_cloud{cloud:azure}.as_count()`
- GCP: `sum:myapp.requests.by_cloud{cloud:gcp}.as_count()`

## Examples

### Example 1: Simple AWS Deployment

```yaml
version: "1.0"

containers:
  - name: api
    image: "myapi:latest"
    ports:
      - container: 80
    environment:
      DD_API_KEY: "${DD_API_KEY}"
      DD_SITE: "us5.datadoghq.com"
      DD_SERVICE: "api"
      DD_ENV: "production"
      DD_TRACE_ENABLED: "true"

provider:
  name: aws
  region: us-east-1

application:
  name: myapi-prod
```

### Example 2: Multi-Cloud with Custom Tags

**AWS manifest (deploy-manifest-aws.yaml):**
```yaml
version: "1.0"

containers:
  - name: api
    image: "myapi-aws:latest"
    ports:
      - container: 80
      - container: 443
    environment:
      DD_API_KEY: "${DD_API_KEY}"
      DD_SITE: "us5.datadoghq.com"
      DD_SERVICE: "api"
      DD_ENV: "production"
      DD_VERSION: "1.0.0"
      DD_TAGS: "cloud:aws,platform:elastic-beanstalk,region:us-east-1"
      DD_TRACE_ENABLED: "true"
      DD_TRACE_SAMPLE_RATE: "1.0"

provider:
  name: aws
  region: us-east-1
```

**Azure manifest (deploy-manifest-azure.yaml):**
```yaml
version: "1.0"

containers:
  - name: api
    image: "myapi-azure:latest"
    ports:
      - container: 80
      - container: 443
    environment:
      DD_API_KEY: "${DD_API_KEY}"
      DD_SITE: "us5.datadoghq.com"
      DD_SERVICE: "api"
      DD_ENV: "production"
      DD_VERSION: "1.0.0"
      DD_TAGS: "cloud:azure,platform:container-instances,region:eastus"
      DD_TRACE_ENABLED: "true"
      DD_TRACE_SAMPLE_RATE: "1.0"

provider:
  name: azure
  region: eastus
```

**GCP manifest (deploy-manifest-gcp.yaml):**
```yaml
version: "1.0"

containers:
  - name: api
    image: "myapi-gcp:latest"
    ports:
      - container: 80
    environment:
      DD_API_KEY: "${DD_API_KEY}"
      DD_SITE: "us5.datadoghq.com"
      DD_SERVICE: "api"
      DD_ENV: "production"
      DD_VERSION: "1.0.0"
      DD_TAGS: "cloud:gcp,platform:cloud-run,region:us-central1"
      DD_TRACE_ENABLED: "true"
      DD_TRACE_SAMPLE_RATE: "1.0"

provider:
  name: gcp
  region: us-central1
```

### Example 3: Development Environment (Reduced Sampling)

```yaml
containers:
  - name: api
    image: "myapi:latest"
    environment:
      DD_API_KEY: "${DD_API_KEY}"
      DD_SITE: "us5.datadoghq.com"
      DD_SERVICE: "api"
      DD_ENV: "development"
      DD_TRACE_ENABLED: "true"
      DD_TRACE_SAMPLE_RATE: "0.1"  # Sample 10% of requests
```

## Troubleshooting

### Traces Not Appearing

**Problem:** No traces showing up in Datadog after deployment.

**Solutions:**

1. **Verify API Key:**
   ```bash
   # Check environment variable is set
   echo $DD_API_KEY
   ```

2. **Check DD_SITE:**
   - Ensure DD_SITE matches your Datadog account region
   - Common mistake: Using `datadoghq.com` when account is on `us5.datadoghq.com`

3. **Verify Container Logs:**
   ```bash
   # AWS
   aws elasticbeanstalk describe-environment-health --environment-name myapp-env

   # Azure
   az container logs --name myapp-env --resource-group myapp-rg

   # GCP
   gcloud run services logs read myapp-service --region us-central1
   ```

4. **Generate Test Traffic:**
   ```bash
   # Send requests to generate traces
   for i in {1..10}; do curl https://your-app-url.com/endpoint; done
   ```

### High Trace Volume

**Problem:** Too many traces, high Datadog costs.

**Solutions:**

1. **Reduce Sample Rate:**
   ```yaml
   DD_TRACE_SAMPLE_RATE: "0.1"  # Sample 10% instead of 100%
   ```

2. **Filter Health Check Traces:**
   - Ensure health check endpoint doesn't generate traces
   - Use Datadog trace filtering in APM settings

3. **Adjust Retention:**
   - Configure retention filters in Datadog APM settings
   - Keep only error traces or slow traces

### Span Tags Not in Dashboards

**Problem:** Span tags visible in Traces Explorer but not available for dashboards.

**Solution:** Create span-based metrics (see [Span-Based Metrics](#span-based-metrics) section above).

## Health Check Configuration

Reduce health check frequency to minimize trace noise:

```yaml
health_check:
  type: basic
  path: /health
  interval_seconds: 300    # Check every 5 minutes
  timeout_seconds: 30
```

**Benefits:**
- Fewer health check traces in APM
- Lower resource usage
- Cleaner trace visualization

## Performance Considerations

### Resource Usage

serverless-init adds minimal overhead:
- **Memory:** ~50MB additional
- **CPU:** <5% additional
- **Startup:** +1-2 seconds

### Sampling Recommendations

| Environment | Sample Rate | Reasoning |
|-------------|-------------|-----------|
| Production | 100% initially, then 10-50% | Start high, reduce after establishing baseline |
| Staging | 50% | Balance cost and visibility |
| Development | 10% | Minimize cost while testing |

## Alternative: Multi-Container Sidecar Pattern

For advanced use cases requiring full Datadog agent features (DogStatsD, custom configs), cloud-deploy supports multi-container deployments with a Datadog agent sidecar.

**When to use sidecar:**
- Need DogStatsD for custom metrics
- Require advanced agent configuration
- Running non-serverless workloads with persistent agent

**For most deployments, serverless-init is recommended** for its simplicity and lower resource usage.

See [CLAUDE.md](../CLAUDE.md) for multi-container implementation details.

## Related Documentation

- [Datadog Serverless Init](https://docs.datadoghq.com/serverless/aws_lambda/configuration/)
- [Datadog Orchestrion (Go)](https://github.com/DataDog/orchestrion)
- [Datadog APM](https://docs.datadoghq.com/tracing/)
- [Cloud-Deploy Monitoring](./MONITORING.md)
- [Cloud-Deploy Main Documentation](../README.md)

## Support

For issues or questions:
- GitHub Issues: https://github.com/jvreagan/cloud-deploy/issues
- Datadog Support: https://help.datadoghq.com/
