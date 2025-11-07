# Working Datadog APM Configuration

**Last Updated:** November 4, 2025
**Status:** ✅ OPERATIONAL - Both HTTP and HTTPS traces working

## Overview

This document describes the **working multi-container Datadog APM configuration** for AWS Elastic Beanstalk deployments using cloud-deploy with Datadog Orchestrion for Go applications.

## Architecture

**Deployment Pattern:** Multi-container sidecar
**Application:** testdd1 (Go HTTP/HTTPS server)
**APM Instrumentation:** Datadog Orchestrion (compile-time automatic instrumentation)
**Cloud Provider:** AWS Elastic Beanstalk
**Datadog Site:** us5.datadoghq.com

**Containers:**
1. **testdd1-app** - Application container with Orchestrion-instrumented Go binary
2. **datadog-agent** - Datadog agent sidecar for trace/metric/log collection

## Working Configuration

### Deployment Manifest

File: `deploy-manifest-multi-aws-datadog.yaml`

```yaml
version: "1.0"

# Multi-container deployment with Datadog agent sidecar on AWS Elastic Beanstalk
containers:
  - name: testdd1-app
    image: "testdd1:latest"
    ports:
      - container: 80
      - container: 443
    environment:
      ENV: production
      # Native Datadog APM tracing (Orchestrion)
      DD_SERVICE: "testdd1"
      DD_ENV: "production"
      DD_TRACE_AGENT_URL: "http://datadog-agent:8126"
      DD_TRACE_SAMPLE_RATE: "1.0"  # Send 100% of traces (no sampling)
      DD_TRACE_DEBUG: "true"        # Enable verbose trace logging for debugging

  - name: datadog-agent
    image: "datadog/agent:latest"
    environment:
      # Datadog API credentials
      DD_API_KEY: "${DD_API_KEY}"
      DD_SITE: "us5.datadoghq.com"

      # Cloud platform identification tags
      DD_TAGS: "cloud:aws,platform:elastic-beanstalk,region:us-east-1"

      # Enable APM, logs, and infrastructure metrics
      DD_APM_ENABLED: "true"
      DD_LOGS_ENABLED: "true"
      DD_DOGSTATSD_NON_LOCAL_TRAFFIC: "true"
      DD_PROCESS_AGENT_ENABLED: "true"
      DD_CONTAINER_INCLUDE: "name:testdd1-app"
      DD_CONTAINER_EXCLUDE: "name:datadog-agent"
      DD_LOG_LEVEL: "info"

provider:
  name: aws
  region: us-east-1
  credentials:
    source: environment

application:
  name: testdd1-multi-aws-datadog
  description: "Test DD1 Multi-Container with Datadog Agent - AWS Elastic Beanstalk"

environment:
  name: testdd1-multi-aws-datadog-env
  cname: jvr-testdd1-datadog

deployment:
  platform: docker
  solution_stack: "64bit Amazon Linux 2023 v4.7.2 running Docker"
  source:
    type: local
    path: "."

instance:
  type: t3.small  # Increased for Datadog agent
  environment_type: LoadBalanced

health_check:
  type: enhanced
  path: /health

monitoring:
  enhanced_health: true
  cloudwatch_metrics: true
  cloudwatch_logs:
    enabled: true
    retention_days: 7
    stream_logs: true

iam:
  instance_profile: cloud-hello-world-eb-ec2

tags:
  Project: cloud-deploy-test
  ManagedBy: cloud-deploy
  Application: testdd1-multi-aws-datadog
  Cloud: aws
  Type: multi-container-datadog

ssl:
  certificate_arn: arn:aws:acm:us-east-1:163436765630:certificate/ead022d8-eecb-4ab9-9cd5-460e93334488
```

### Generated Docker Compose Configuration

cloud-deploy generates this `docker-compose.yml` which gets deployed to Elastic Beanstalk:

```yaml
version: "3.8"
services:
  testdd1-app:
    image: 163436765630.dkr.ecr.us-east-1.amazonaws.com/testdd1-multi-aws-datadog:testdd1-app
    ports:
      - "80:80"
      - "443:443"
    environment:
      ENV: production
      DD_SERVICE: testdd1
      DD_ENV: production
      DD_TRACE_AGENT_URL: http://datadog-agent:8126
      DD_TRACE_SAMPLE_RATE: "1.0"
      DD_TRACE_DEBUG: "true"
    restart: unless-stopped

  datadog-agent:
    image: 163436765630.dkr.ecr.us-east-1.amazonaws.com/testdd1-multi-aws-datadog:datadog-agent
    environment:
      DD_API_KEY: 96445a08f1bb330929dad3dd470f9cdf
      DD_SITE: us5.datadoghq.com
      DD_TAGS: cloud:aws,platform:elastic-beanstalk,region:us-east-1
      DD_APM_ENABLED: "true"
      DD_LOGS_ENABLED: "true"
      DD_DOGSTATSD_NON_LOCAL_TRAFFIC: "true"
      DD_PROCESS_AGENT_ENABLED: "true"
      DD_CONTAINER_INCLUDE: name:testdd1-app
      DD_CONTAINER_EXCLUDE: name:datadog-agent
      DD_LOG_LEVEL: info
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - /proc/:/host/proc/:ro
      - /sys/fs/cgroup/:/host/sys/fs/cgroup:ro
    restart: unless-stopped
```

## Application Setup

### Dockerfile

The application uses a multi-stage build with Orchestrion instrumentation:

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Copy Go module files
COPY go.mod go.sum ./
COPY orchestrion.tool.go ./

# Download dependencies
RUN go mod download

# Install Orchestrion
RUN go install github.com/DataDog/orchestrion@latest

# Copy source code
COPY . .

# Build with Orchestrion instrumentation for x86_64 (AWS EC2)
RUN CGO_ENABLED=0 GOOS=linux /go/bin/orchestrion go build -o main main.go

FROM alpine:latest
WORKDIR /app

# Copy instrumented binary
COPY --from=builder /app/main .

# Expose HTTP and HTTPS ports
EXPOSE 80
EXPOSE 443

CMD ["./main"]
```

### orchestrion.tool.go

```go
//go:build tools

package main

import (
    _ "github.com/DataDog/orchestrion"
)
```

### go.mod Dependencies

```go
require (
    github.com/DataDog/dd-trace-go/v2 v2.x.x
    github.com/DataDog/orchestrion v0.x.x
)
```

## Building and Deploying

### Step 1: Set Environment Variables

```bash
export DD_API_KEY="your-datadog-api-key"
export AWS_ACCESS_KEY_ID="your-aws-access-key"
export AWS_SECRET_ACCESS_KEY="your-aws-secret-key"
export AWS_REGION="us-east-1"
```

### Step 2: Build Docker Image for x86_64 (AWS EC2 Architecture)

**CRITICAL:** AWS EC2 instances use x86_64 architecture. If building on Mac (ARM64), use `--platform linux/amd64`:

```bash
cd /path/to/testdd1

# Build application image for x86_64
docker build --platform linux/amd64 -t testdd1:latest .

# Pull Datadog agent image for x86_64
docker pull --platform linux/amd64 datadog/agent:latest
```

**Verify architecture:**

```bash
docker inspect testdd1:latest | grep Architecture
# Should show: "Architecture": "amd64"

docker inspect datadog/agent:latest | grep Architecture
# Should show: "Architecture": "amd64"
```

### Step 3: Deploy with cloud-deploy

```bash
cd /path/to/testdd1

# Deploy multi-container application
/path/to/cloud-deploy --command deploy --manifest deploy-manifest-multi-aws-datadog.yaml
```

**Deployment process:**
1. Pushes `testdd1:latest` to ECR as `testdd1-multi-aws-datadog:testdd1-app`
2. Pushes `datadog/agent:latest` to ECR as `testdd1-multi-aws-datadog:datadog-agent`
3. Generates `docker-compose.yml` with all environment variables
4. Creates Elastic Beanstalk application version
5. Configures ELB listeners for HTTP (port 80) and HTTPS (port 443)
6. Deploys to environment

### Step 4: Verify Deployment

```bash
# Test HTTP endpoint
curl -s http://jvr-testdd1-datadog.us-east-1.elasticbeanstalk.com/testdd1

# Test HTTPS endpoint
curl -k -s https://jvr-testdd1-datadog.us-east-1.elasticbeanstalk.com/testdd1
```

### Step 5: Generate Test Traffic

```bash
# Generate HTTP traffic
for i in {1..20}; do
  curl -s http://jvr-testdd1-datadog.us-east-1.elasticbeanstalk.com/random > /dev/null
  sleep 0.5
done

# Generate HTTPS traffic
for i in {1..20}; do
  curl -k -s https://jvr-testdd1-datadog.us-east-1.elasticbeanstalk.com/random > /dev/null
  sleep 0.5
done
```

### Step 6: Verify Traces in Datadog

1. Go to https://us5.datadoghq.com/apm/services
2. Look for the `testdd1` service
3. Verify traces for both HTTP and HTTPS requests
4. Check that all endpoints (`/testdd1`, `/random`, `/health`) are being traced

## Key Configuration Details

### Environment Variables (Application Container)

| Variable | Value | Purpose |
|----------|-------|---------|
| `DD_SERVICE` | `testdd1` | Service name in Datadog APM |
| `DD_ENV` | `production` | Environment tag |
| `DD_TRACE_AGENT_URL` | `http://datadog-agent:8126` | Agent endpoint for trace submission |
| `DD_TRACE_SAMPLE_RATE` | `1.0` | 100% sampling (send all traces) |
| `DD_TRACE_DEBUG` | `true` | Enable verbose trace logging |

### Environment Variables (Datadog Agent Container)

| Variable | Value | Purpose |
|----------|-------|---------|
| `DD_API_KEY` | `${DD_API_KEY}` | Datadog API key (from environment) |
| `DD_SITE` | `us5.datadoghq.com` | Datadog site (US5 region) |
| `DD_APM_ENABLED` | `true` | Enable APM trace collection |
| `DD_LOGS_ENABLED` | `true` | Enable log collection |
| `DD_DOGSTATSD_NON_LOCAL_TRAFFIC` | `true` | Accept metrics from other containers |
| `DD_PROCESS_AGENT_ENABLED` | `true` | Enable process monitoring |
| `DD_CONTAINER_INCLUDE` | `name:testdd1-app` | Only monitor testdd1-app container |
| `DD_CONTAINER_EXCLUDE` | `name:datadog-agent` | Exclude agent from monitoring |
| `DD_LOG_LEVEL` | `info` | Agent log verbosity |

### Docker Volumes (Datadog Agent)

| Host Path | Container Path | Purpose |
|-----------|----------------|---------|
| `/var/run/docker.sock` | `/var/run/docker.sock:ro` | Docker API access for container metrics |
| `/proc/` | `/host/proc/:ro` | Host process information |
| `/sys/fs/cgroup/` | `/host/sys/fs/cgroup:ro` | cgroup metrics |

## Architecture Notes

### Why Multi-Container Sidecar Pattern?

1. **Separation of Concerns:** Application and monitoring are separate containers
2. **Independent Lifecycle:** Can update agent without redeploying application
3. **Resource Control:** Separate CPU/memory limits for app and agent
4. **Industry Best Practice:** Recommended by Datadog for all platforms in 2025
5. **Full Feature Support:** Enables all Datadog features (APM, logs, metrics, processes)

### How Orchestrion Instrumentation Works

1. **Compile-time:** Orchestrion modifies Go source code during compilation
2. **Automatic:** No code changes required - imports are auto-added
3. **AST-level:** Operates at Abstract Syntax Tree level, verified by Go compiler
4. **net/http Support:** Instruments both `http.ListenAndServe` (HTTP) and `http.Server.ListenAndServeTLS` (HTTPS)
5. **Shared Mux:** Both HTTP and HTTPS servers use same `mux`, so both get instrumented

### Container Networking

- Containers share the same Docker network created by docker-compose
- Application container can reach agent via `http://datadog-agent:8126`
- Docker DNS resolves `datadog-agent` to the agent container's IP
- No host network mode needed - standard bridge networking works

## Troubleshooting

### No Traces Appearing in Datadog

**Check container status:**
```bash
cd /path/to/testdd1
eb ssh testdd1-multi-aws-datadog-env --command "docker ps"
```

**Check agent logs:**
```bash
eb logs testdd1-multi-aws-datadog-env --all
```

**Verify DD_API_KEY is set:**
```bash
echo $DD_API_KEY
```

**Verify agent connectivity:**
- Agent should connect to `https://us5.datadoghq.com`
- Check firewall rules allow outbound HTTPS to Datadog

### Architecture Mismatch Errors

**Symptom:** Containers crash with `exec format error`

**Cause:** Docker image built for ARM64 (Mac) but AWS EC2 uses x86_64

**Fix:** Rebuild with `--platform linux/amd64`:
```bash
docker build --platform linux/amd64 -t testdd1:latest .
```

### HTTPS Not Working

**Symptom:** HTTPS endpoint returns timeout or connection refused

**Cause:** ELB listeners not configured for port 443

**Fix:** Ensure manifest has port 443 in containers configuration and redeploy with cloud-deploy (not `eb deploy`)

## Verified Working

**Date:** November 4, 2025
**Deployment URL:** http://jvr-testdd1-datadog.us-east-1.elasticbeanstalk.com
**Datadog Site:** https://us5.datadoghq.com

**Test Results:**
- ✅ HTTP endpoint responding (200 OK)
- ✅ HTTPS endpoint responding (200 OK)
- ✅ HTTP traces appearing in Datadog
- ✅ HTTPS traces appearing in Datadog
- ✅ All endpoints traced (`/testdd1`, `/random`, `/health`)
- ✅ Both containers running healthy
- ✅ Infrastructure metrics available
- ✅ Log collection working

## References

- [Datadog Orchestrion Documentation](https://docs.datadoghq.com/tracing/trace_collection/automatic_instrumentation/dd_libraries/go/)
- [Datadog Orchestrion GitHub](https://github.com/DataDog/orchestrion)
- [AWS Elastic Beanstalk Multi-Container Docker](https://docs.aws.amazon.com/elasticbeanstalk/latest/dg/create_deploy_docker_v2config.html)
- [cloud-deploy Multi-Container Implementation](pkg/providers/aws/aws.go)
