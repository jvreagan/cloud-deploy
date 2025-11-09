# Manifest Reference

Complete reference for all cloud-deploy manifest options. This document describes every field available in the deployment manifest.

## Table of Contents

- [Manifest Structure](#manifest-structure)
- [Root Level Fields](#root-level-fields)
- [Provider Configuration](#provider-configuration)
- [Credentials Configuration](#credentials-configuration)
- [Application Configuration](#application-configuration)
- [Environment Configuration](#environment-configuration)
- [Deployment Configuration](#deployment-configuration)
- [Instance Configuration](#instance-configuration)
- [Container Configuration](#container-configuration)
- [Port Mapping](#port-mapping)
- [Health Check Configuration](#health-check-configuration)
- [Cloud Run Configuration (GCP)](#cloud-run-configuration-gcp)
- [Azure Configuration](#azure-configuration)
- [Monitoring Configuration](#monitoring-configuration)
- [IAM Configuration](#iam-configuration)
- [SSL Configuration](#ssl-configuration)
- [Environment Variables](#environment-variables)
- [Tags](#tags)
- [Complete Examples](#complete-examples)

---

## Manifest Structure

A cloud-deploy manifest is a YAML file that defines your complete deployment configuration.

**Minimal Example:**
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

---

## Root Level Fields

### `version`
**Type:** `string`
**Required:** Yes
**Default:** None
**Description:** Manifest schema version. Currently only `"1.0"` is supported.

**Example:**
```yaml
version: "1.0"
```

---

### `image`
**Type:** `string`
**Required:** No (either `image` or `containers` required)
**Default:** None
**Providers:** All
**Description:** Docker image to deploy for single-container deployments. Image must exist in your local Docker daemon. **Deprecated in favor of `containers` for multi-container deployments.**

**Example:**
```yaml
image: "myapp:latest"
```

**Notes:**
- For backward compatibility, if `image` is set and `containers` is empty, single-container mode is used
- Image can be a local tag or a fully qualified registry URL
- The image must be built before deployment

---

### `containers`
**Type:** `array[Container]`
**Required:** No (either `image` or `containers` required)
**Default:** None
**Providers:** AWS, GCP, Azure
**Description:** Array of containers for multi-container deployments. Each container runs as a separate process.

**Example:**
```yaml
containers:
  - name: app
    image: "myapp:latest"
    ports:
      - container: 80
      - container: 443
    environment:
      ENV: production

  - name: datadog-agent
    image: "datadog/agent:latest"
    environment:
      DD_API_KEY: "${DD_API_KEY}"
      DD_SITE: "us5.datadoghq.com"
```

**Notes:**
- If `containers` is set, it takes precedence over `image`
- See [Container Configuration](#container-configuration) for container field details
- AWS: Uses Docker Compose format
- GCP: Uses Cloud Run sidecars
- Azure: Uses Container Groups

---

### `provider`
**Type:** `ProviderConfig`
**Required:** Yes
**Default:** None
**Description:** Cloud provider configuration. See [Provider Configuration](#provider-configuration).

---

### `application`
**Type:** `ApplicationConfig`
**Required:** Yes
**Default:** None
**Description:** Application metadata. See [Application Configuration](#application-configuration).

---

### `environment`
**Type:** `EnvironmentConfig`
**Required:** Yes
**Default:** None
**Description:** Environment configuration. See [Environment Configuration](#environment-configuration).

---

### `deployment`
**Type:** `DeploymentConfig`
**Required:** Yes
**Default:** None
**Description:** Deployment settings. See [Deployment Configuration](#deployment-configuration).

---

### `instance`
**Type:** `InstanceConfig`
**Required:** Yes (AWS), No (GCP, Azure)
**Default:** None
**Description:** Compute instance configuration. See [Instance Configuration](#instance-configuration).

---

### `cloud_run`
**Type:** `CloudRunConfig`
**Required:** No
**Default:** None
**Providers:** GCP only
**Description:** GCP Cloud Run-specific configuration. See [Cloud Run Configuration](#cloud-run-configuration-gcp).

---

### `azure`
**Type:** `AzureConfig`
**Required:** No
**Default:** None
**Providers:** Azure only
**Description:** Azure Container Instances-specific configuration. See [Azure Configuration](#azure-configuration).

---

### `health_check`
**Type:** `HealthCheckConfig`
**Required:** Yes
**Default:** None
**Description:** Health check settings. See [Health Check Configuration](#health-check-configuration).

---

### `monitoring`
**Type:** `MonitoringConfig`
**Required:** No
**Default:** All monitoring features disabled
**Providers:** AWS (CloudWatch), GCP (Cloud Logging)
**Description:** Monitoring and metrics configuration. See [Monitoring Configuration](#monitoring-configuration).

---

### `iam`
**Type:** `IAMConfig`
**Required:** No
**Default:** None
**Providers:** AWS
**Description:** IAM roles and profiles. See [IAM Configuration](#iam-configuration).

---

### `environment_variables`
**Type:** `map[string]string`
**Required:** No
**Default:** Empty
**Providers:** All
**Description:** Environment variables to inject into the container(s). Supports environment variable expansion using `${VAR_NAME}` syntax.

**Example:**
```yaml
environment_variables:
  NODE_ENV: production
  LOG_LEVEL: info
  DATABASE_URL: "${DATABASE_URL}"
  API_KEY: "${API_KEY}"
```

**Notes:**
- Variables are expanded at manifest load time using the shell's environment
- For multi-container deployments, these apply to the primary container
- Use container-specific `environment` field for per-container variables

---

### `tags`
**Type:** `map[string]string`
**Required:** No
**Default:** Empty
**Providers:** AWS
**Description:** Tags to apply to cloud resources (applications, environments, etc.).

**Example:**
```yaml
tags:
  Project: MyProject
  Environment: Production
  Team: Platform
  CostCenter: Engineering
```

---

### `ports`
**Type:** `array[PortMapping]`
**Required:** No
**Default:** None
**Description:** Port mappings for single-container deployments. See [Port Mapping](#port-mapping).

**Example:**
```yaml
ports:
  - container: 80
  - container: 443
    host: 443
```

**Notes:**
- For multi-container, use per-container `ports` field instead
- If `host` is omitted, defaults to same as `container`

---

### `ssl`
**Type:** `SSLConfig`
**Required:** No
**Default:** None
**Providers:** AWS
**Description:** SSL/TLS certificate configuration. See [SSL Configuration](#ssl-configuration).

---

## Provider Configuration

Defines which cloud provider to use and how to authenticate.

### Fields

#### `name`
**Type:** `string`
**Required:** Yes
**Allowed Values:** `aws`, `gcp`, `azure`, `oci`
**Description:** Cloud provider name.

#### `region`
**Type:** `string`
**Required:** Yes
**Description:** Cloud region to deploy to.

**Examples:**
- AWS: `us-east-1`, `us-east-2`, `us-west-1`, `us-west-2`, `eu-west-1`
- GCP: `us-central1`, `us-east1`, `us-west1`, `europe-west1`
- Azure: `eastus`, `westus`, `centralus`, `northeurope`, `westeurope`

#### `credentials`
**Type:** `CredentialsConfig`
**Required:** No
**Default:** Uses cloud provider CLI credentials
**Description:** Credential configuration. See [Credentials Configuration](#credentials-configuration).

---

### GCP-Specific Fields

#### `project_id`
**Type:** `string`
**Required:** Yes (GCP only)
**Description:** GCP project ID. Will be created if it doesn't exist.

#### `billing_account_id`
**Type:** `string`
**Required:** Yes (GCP only, if project doesn't exist)
**Format:** `XXXXXX-XXXXXX-XXXXXX`
**Description:** GCP billing account ID. Required for creating new projects.

**How to find:**
1. Go to https://console.cloud.google.com/billing
2. Copy the billing account ID

#### `public_access`
**Type:** `boolean`
**Required:** No
**Default:** `true`
**Description:** Make Cloud Run service publicly accessible.

#### `organization_id`
**Type:** `string`
**Required:** No
**Description:** GCP organization ID for creating projects under an organization.

---

### Azure-Specific Fields

#### `subscription_id`
**Type:** `string`
**Required:** Yes (Azure only)
**Description:** Azure subscription ID.

#### `resource_group`
**Type:** `string`
**Required:** Yes (Azure only)
**Description:** Azure resource group name. Will be created if it doesn't exist.

---

### Example

```yaml
# AWS
provider:
  name: aws
  region: us-east-2
  credentials:
    source: environment

# GCP
provider:
  name: gcp
  region: us-central1
  project_id: my-project
  billing_account_id: "XXXXXX-XXXXXX-XXXXXX"
  public_access: true

# Azure
provider:
  name: azure
  region: eastus
  subscription_id: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  resource_group: my-resource-group
```

---

## Credentials Configuration

Defines how to authenticate with cloud providers.

### Fields

#### `source`
**Type:** `string`
**Required:** No
**Default:** `cli`
**Allowed Values:** `manifest`, `environment`, `cli`, `vault`
**Description:** Source of credentials.

**Values:**
- `manifest`: Use credentials in this manifest (not recommended for production)
- `environment`: Use environment variables (e.g., `AWS_ACCESS_KEY_ID`)
- `cli`: Use cloud provider CLI credentials (default)
- `vault`: Fetch from HashiCorp Vault

---

### AWS Credentials

#### `access_key_id`
**Type:** `string`
**Required:** Yes (if `source: manifest`)
**Description:** AWS access key ID.

#### `secret_access_key`
**Type:** `string`
**Required:** Yes (if `source: manifest`)
**Description:** AWS secret access key.

**Environment Variables (when `source: environment`):**
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `AWS_REGION`

---

### GCP Credentials

#### `service_account_key_path`
**Type:** `string`
**Required:** Yes (if `source: manifest`, option 1)
**Description:** Path to service account JSON key file.

#### `service_account_key_json`
**Type:** `string`
**Required:** Yes (if `source: manifest`, option 2)
**Description:** Service account JSON content directly embedded.

**Environment Variables (when `source: environment`):**
- `GOOGLE_APPLICATION_CREDENTIALS` (path to JSON key file)
- `GCP_PROJECT_ID`

---

### Azure Credentials

#### `azure`
**Type:** `AzureCredentialsConfig`
**Required:** Yes (if `source: manifest`)
**Description:** Azure service principal credentials.

**Fields:**
- `client_id`: Application (client) ID
- `client_secret`: Client secret
- `tenant_id`: Directory (tenant) ID

**Environment Variables (when `source: environment`):**
- `AZURE_CLIENT_ID`
- `AZURE_CLIENT_SECRET`
- `AZURE_TENANT_ID`
- `AZURE_SUBSCRIPTION_ID`

---

### Examples

```yaml
# Use environment variables (recommended)
credentials:
  source: environment

# Use cloud provider CLI
credentials:
  source: cli

# Vault (recommended for production)
credentials:
  source: vault

# Manifest (not recommended)
credentials:
  source: manifest
  access_key_id: "AKIAIOSFODNN7EXAMPLE"
  secret_access_key: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
```

---

## Application Configuration

Defines the application being deployed.

### Fields

#### `name`
**Type:** `string`
**Required:** Yes
**Pattern:** Alphanumeric, hyphens, underscores
**Description:** Application name. Must be unique within your cloud account.

#### `description`
**Type:** `string`
**Required:** No
**Description:** Human-readable description of the application.

### Example

```yaml
application:
  name: my-web-app
  description: "Production web application"
```

---

## Environment Configuration

Defines the environment (running instance) of the application.

### Fields

#### `name`
**Type:** `string`
**Required:** Yes
**Pattern:** Alphanumeric, hyphens, underscores
**Description:** Environment name. Must be unique within the application.

**Common Values:** `dev`, `staging`, `prod`, `production`, `test`

#### `cname`
**Type:** `string`
**Required:** No (AWS), Yes (others may vary)
**Description:** Subdomain/CNAME for the environment.

**AWS Result:** Creates `<cname>.<region>.elasticbeanstalk.com`

### Example

```yaml
environment:
  name: production
  cname: my-app-prod
```

**Result (AWS):** `my-app-prod.us-east-2.elasticbeanstalk.com`

---

## Deployment Configuration

Specifies how the application should be deployed.

### Fields

#### `platform`
**Type:** `string`
**Required:** Yes
**Allowed Values:** `docker`, `nodejs`, `python`, `java`, `go`, etc.
**Description:** Platform/runtime type.

**Common Values:**
- `docker` - Docker containers (most common)
- `nodejs` - Node.js applications
- `python` - Python applications
- `java` - Java applications
- `go` - Go applications

#### `solution_stack`
**Type:** `string`
**Required:** No
**Default:** Auto-detected based on platform
**Providers:** AWS
**Description:** Specific solution stack version.

**Example:** `"64bit Amazon Linux 2023 v4.0.0 running Docker"`

**Note:** If omitted, cloud-deploy will auto-detect the latest version.

#### `source`
**Type:** `SourceConfig`
**Required:** Yes
**Description:** Source code location.

**Fields:**
- `type`: Source type (`local`, `s3`, `git`)
- `path`: Path to source code

### Examples

```yaml
# Local source
deployment:
  platform: docker
  source:
    type: local
    path: "."

# Specific solution stack
deployment:
  platform: docker
  solution_stack: "64bit Amazon Linux 2023 v4.0.0 running Docker"
  source:
    type: local
    path: "."

# S3 source
deployment:
  platform: docker
  source:
    type: s3
    path: "s3://my-bucket/my-app.zip"
```

---

## Instance Configuration

Specifies compute resources for the deployment.

### Fields

#### `type`
**Type:** `string`
**Required:** Yes (AWS), No (GCP, Azure)
**Description:** Instance type/size.

**AWS Examples:**
- `t3.micro` - 2 vCPU, 1 GB RAM (free tier eligible)
- `t3.small` - 2 vCPU, 2 GB RAM
- `t3.medium` - 2 vCPU, 4 GB RAM
- `m5.large` - 2 vCPU, 8 GB RAM

**GCP:** Use `cloud_run` config instead
**Azure:** Use `azure` config instead

#### `environment_type`
**Type:** `string`
**Required:** Yes (AWS), No (GCP, Azure)
**Allowed Values:** `SingleInstance`, `LoadBalanced`
**Description:** Deployment architecture.

**Values:**
- `SingleInstance`: Single EC2 instance (no load balancer)
- `LoadBalanced`: Auto-scaling with load balancer

### Example

```yaml
instance:
  type: t3.micro
  environment_type: SingleInstance
```

---

## Container Configuration

Defines a single container in multi-container deployments.

### Fields

#### `name`
**Type:** `string`
**Required:** Yes
**Description:** Container name. Must be unique within the deployment.

#### `image`
**Type:** `string`
**Required:** Yes
**Description:** Docker image for this container.

#### `ports`
**Type:** `array[PortMapping]`
**Required:** No
**Description:** Port mappings for this container. See [Port Mapping](#port-mapping).

#### `environment`
**Type:** `map[string]string`
**Required:** No
**Description:** Environment variables for this container.

#### `command`
**Type:** `array[string]`
**Required:** No
**Description:** Override container's default command.

### Example

```yaml
containers:
  - name: web-app
    image: "myapp:latest"
    ports:
      - container: 80
      - container: 443
    environment:
      ENV: production
      LOG_LEVEL: info

  - name: sidecar
    image: "helper:latest"
    command: ["./helper", "--config", "/etc/config.yaml"]
    environment:
      HELPER_MODE: production
```

---

## Port Mapping

Defines container port mappings.

### Fields

#### `container`
**Type:** `integer`
**Required:** Yes
**Description:** Port number inside the container.

#### `host`
**Type:** `integer`
**Required:** No
**Default:** Same as `container`
**Description:** Port number on the host.

### Examples

```yaml
# Simple port exposure
ports:
  - container: 80

# Explicit host mapping
ports:
  - container: 80
    host: 80
  - container: 443
    host: 443

# YAML shorthand
ports:
  - container: 8080
```

---

## Health Check Configuration

Defines how the cloud provider should check application health.

### Fields

#### `type`
**Type:** `string`
**Required:** Yes
**Allowed Values:** `basic`, `enhanced`
**Description:** Health check type.

**Values:**
- `basic`: Simple HTTP health check
- `enhanced`: Enhanced health reporting with detailed metrics (AWS only)

#### `path`
**Type:** `string`
**Required:** Yes
**Description:** HTTP path to health check endpoint.

**Common Values:** `/health`, `/healthz`, `/api/health`, `/status`

#### `interval_seconds`
**Type:** `integer`
**Required:** No
**Default:** 30 (AWS), varies by provider
**Description:** Seconds between health checks.

**Recommended:**
- Development: 30-60 seconds
- Production: 300 seconds (5 minutes)

#### `timeout_seconds`
**Type:** `integer`
**Required:** No
**Default:** 5
**Description:** Health check timeout in seconds.

### Examples

```yaml
# Basic health check
health_check:
  type: basic
  path: /health

# Enhanced with custom interval
health_check:
  type: enhanced
  path: /api/status
  interval_seconds: 300
  timeout_seconds: 30
```

---

## Cloud Run Configuration (GCP)

GCP Cloud Run-specific resource and scaling configuration.

### Fields

#### `cpu`
**Type:** `string`
**Required:** No
**Default:** `"1"`
**Allowed Values:** `"1"`, `"2"`, `"4"`
**Description:** CPU allocation.

#### `memory`
**Type:** `string`
**Required:** No
**Default:** `"512Mi"`
**Allowed Values:** `"256Mi"`, `"512Mi"`, `"1Gi"`, `"2Gi"`, `"4Gi"`, `"8Gi"`
**Description:** Memory allocation.

#### `max_concurrency`
**Type:** `integer`
**Required:** No
**Default:** `80`
**Description:** Maximum concurrent requests per container.

#### `min_instances`
**Type:** `integer`
**Required:** No
**Default:** `0`
**Description:** Minimum number of instances (0 = scale to zero).

#### `max_instances`
**Type:** `integer`
**Required:** No
**Default:** `100`
**Description:** Maximum number of instances.

#### `timeout_seconds`
**Type:** `integer`
**Required:** No
**Default:** `300`
**Max:** `3600` (1st gen), `86400` (2nd gen)
**Description:** Request timeout in seconds.

### Example

```yaml
cloud_run:
  cpu: "2"
  memory: "1Gi"
  max_concurrency: 100
  min_instances: 1
  max_instances: 50
  timeout_seconds: 600
```

---

## Azure Configuration

Azure Container Instances-specific resource configuration.

### Fields

#### `cpu`
**Type:** `float`
**Required:** No
**Default:** `1.0`
**Description:** CPU allocation in cores.

**Common Values:** `1.0`, `2.0`, `4.0`

#### `memory_gb`
**Type:** `float`
**Required:** No
**Default:** `1.5`
**Description:** Memory allocation in GB.

**Common Values:** `1.5`, `2.0`, `4.0`, `8.0`

### Example

```yaml
azure:
  cpu: 2.0
  memory_gb: 4.0
```

---

## Monitoring Configuration

Monitoring, metrics, and logging configuration.

### Fields

#### `enhanced_health`
**Type:** `boolean`
**Required:** No
**Default:** `false`
**Providers:** AWS
**Description:** Enable enhanced health reporting with detailed metrics.

**Metrics Provided:**
- Application requests (2xx, 4xx, 5xx)
- Request latency (p50, p75, p90, p99)
- Instance health

#### `cloudwatch_metrics`
**Type:** `boolean`
**Required:** No
**Default:** `false`
**Providers:** AWS
**Description:** Enable CloudWatch custom metrics collection.

#### `cloudwatch_logs`
**Type:** `CloudWatchLogsConfig`
**Required:** No
**Description:** CloudWatch Logs streaming configuration.

**Fields:**
- `enabled` (boolean): Enable log streaming
- `retention_days` (integer): Log retention (1, 3, 5, 7, 14, 30, 60, 90, 120, 150, 180, 365, etc.)
- `stream_logs` (boolean): Stream application logs

### Examples

```yaml
# Enhanced monitoring
monitoring:
  enhanced_health: true
  cloudwatch_metrics: true

# With log streaming
monitoring:
  enhanced_health: true
  cloudwatch_logs:
    enabled: true
    retention_days: 30
    stream_logs: true
```

---

## IAM Configuration

IAM roles and profiles for accessing cloud resources.

### Fields

#### `instance_profile`
**Type:** `string`
**Required:** No
**Providers:** AWS
**Description:** EC2 instance profile name.

**Use Case:** Allow EC2 instances to access other AWS services (S3, DynamoDB, etc.)

#### `service_role`
**Type:** `string`
**Required:** No
**Providers:** AWS
**Description:** Service role for Elastic Beanstalk.

### Example

```yaml
iam:
  instance_profile: my-app-instance-profile
  service_role: aws-elasticbeanstalk-service-role
```

---

## SSL Configuration

SSL/TLS certificate configuration.

### Fields

#### `certificate_arn`
**Type:** `string`
**Required:** No
**Providers:** AWS
**Description:** AWS Certificate Manager (ACM) certificate ARN for HTTPS.

**Format:** `arn:aws:acm:region:account:certificate/certificate-id`

**How to get:**
1. Request certificate in AWS Certificate Manager
2. Validate domain ownership
3. Copy the ARN

### Example

```yaml
ssl:
  certificate_arn: "arn:aws:acm:us-east-2:123456789012:certificate/12345678-1234-1234-1234-123456789012"
```

**Result:** Configures load balancer to terminate HTTPS on port 443 using the ACM certificate.

---

## Environment Variables

Global environment variables apply to all containers (single-container) or the primary container (multi-container).

**Syntax:**
```yaml
environment_variables:
  KEY: value
  ANOTHER_KEY: "${ENV_VAR}"
```

**Environment Variable Expansion:**
Variables in the format `${VAR_NAME}` are expanded from the shell environment at manifest load time.

**Example:**
```yaml
environment_variables:
  NODE_ENV: production
  DATABASE_URL: "${DATABASE_URL}"
  API_KEY: "${API_KEY}"
  LOG_LEVEL: info
```

**Shell Environment:**
```bash
export DATABASE_URL="postgresql://localhost/mydb"
export API_KEY="secret-key-123"
cloud-deploy -command deploy -manifest manifest.yaml
```

---

## Tags

Resource tags for organization and cost tracking.

**Syntax:**
```yaml
tags:
  key: value
```

**Example:**
```yaml
tags:
  Project: MyApp
  Environment: Production
  Team: Platform
  CostCenter: Engineering
  Owner: john@example.com
```

**Providers:** AWS (applied to applications, environments, and resources)

---

## Complete Examples

### Minimal AWS Deployment

```yaml
version: "1.0"

provider:
  name: aws
  region: us-east-2

application:
  name: simple-app

environment:
  name: simple-app-env
  cname: my-app

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

---

### Production AWS with Monitoring

```yaml
version: "1.0"

provider:
  name: aws
  region: us-east-2
  credentials:
    source: environment

application:
  name: production-app
  description: "Production web application"

environment:
  name: production-env
  cname: myapp

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
  path: /health
  interval_seconds: 300
  timeout_seconds: 30

monitoring:
  enhanced_health: true
  cloudwatch_metrics: true
  cloudwatch_logs:
    enabled: true
    retention_days: 30
    stream_logs: true

iam:
  instance_profile: myapp-instance-profile

ssl:
  certificate_arn: "arn:aws:acm:us-east-2:123456789012:certificate/12345678-1234-1234-1234-123456789012"

environment_variables:
  NODE_ENV: production
  DATABASE_URL: "${DATABASE_URL}"
  API_KEY: "${API_KEY}"

tags:
  Project: MyApp
  Environment: Production
  Team: Platform
```

---

### GCP Cloud Run Deployment

```yaml
version: "1.0"

provider:
  name: gcp
  region: us-central1
  project_id: my-project-123
  billing_account_id: "XXXXXX-XXXXXX-XXXXXX"
  public_access: true
  credentials:
    source: environment

application:
  name: my-gcp-app
  description: "GCP Cloud Run application"

environment:
  name: production

deployment:
  platform: docker
  source:
    type: local
    path: "."

cloud_run:
  cpu: "2"
  memory: "1Gi"
  max_concurrency: 100
  min_instances: 1
  max_instances: 50
  timeout_seconds: 600

health_check:
  type: basic
  path: /health

environment_variables:
  NODE_ENV: production
  LOG_LEVEL: info
```

---

### Multi-Container with Datadog

```yaml
version: "1.0"

provider:
  name: aws
  region: us-east-2
  credentials:
    source: environment

application:
  name: monitored-app
  description: "Application with Datadog APM"

environment:
  name: production-env

deployment:
  platform: docker
  source:
    type: local
    path: "."

instance:
  type: t3.small
  environment_type: LoadBalanced

containers:
  - name: app
    image: "myapp:latest"
    ports:
      - container: 80
      - container: 443
    environment:
      ENV: production
      DD_SERVICE: myapp
      DD_ENV: production
      DD_TRACE_AGENT_URL: "http://datadog-agent:8126"

  - name: datadog-agent
    image: "datadog/agent:latest"
    environment:
      DD_API_KEY: "${DD_API_KEY}"
      DD_SITE: "us5.datadoghq.com"
      DD_APM_ENABLED: "true"
      DD_LOGS_ENABLED: "true"

health_check:
  type: basic
  path: /health
  interval_seconds: 300

tags:
  Project: MyApp
  Environment: Production
  Monitoring: Datadog
```

---

### Azure Container Instances

```yaml
version: "1.0"

provider:
  name: azure
  region: eastus
  subscription_id: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  resource_group: my-resource-group
  credentials:
    source: environment

application:
  name: azure-app
  description: "Azure Container Instances application"

environment:
  name: production

deployment:
  platform: docker
  source:
    type: local
    path: "."

azure:
  cpu: 2.0
  memory_gb: 4.0

health_check:
  type: basic
  path: /health
  interval_seconds: 300
  timeout_seconds: 30

environment_variables:
  ENV: production
  LOG_LEVEL: info
```

---

## See Also

- [AWS Deployment Guide](AWS.md)
- [GCP Deployment Guide](GCP.md)
- [Vault Integration](VAULT_INTEGRATION.md)
- [Multi-Cloud Deployments](MULTI_CLOUD.md)
- [Datadog Integration](DATADOG.md)
- [Features Guide](FEATURES.md)
