# CloudWatch Monitoring and Enhanced Health Configuration

This document explains how to configure CloudWatch monitoring and enhanced health reporting for your cloud-deploy applications.

## Overview

Cloud-deploy now supports comprehensive monitoring configuration through the manifest file:
- **Enhanced Health Reporting** - Detailed application-level health metrics
- **CloudWatch Metrics** - Custom application metrics beyond basic EC2 metrics
- **CloudWatch Logs** - Stream application logs to CloudWatch

## Table of Contents

- [Quick Start](#quick-start)
- [Manifest Configuration](#manifest-configuration)
- [Available Metrics](#available-metrics)
- [Cost Considerations](#cost-considerations)
- [Examples](#examples)
- [Troubleshooting](#troubleshooting)

## Quick Start

Add monitoring configuration to your `deploy-manifest.yaml`:

```yaml
monitoring:
  enhanced_health: true
  cloudwatch_metrics: true
  cloudwatch_logs:
    enabled: true
    retention_days: 7
    stream_logs: true
```

Deploy your application:

```bash
cloud-deploy -command deploy -manifest deploy-manifest.yaml
```

## Manifest Configuration

### Enhanced Health Reporting

Enhanced health reporting provides detailed metrics about your application's performance and health.

```yaml
monitoring:
  enhanced_health: true
```

**What it enables:**
- Application-level request metrics (2xx, 3xx, 4xx, 5xx)
- Detailed latency percentiles (P10, P50, P75, P85, P90, P95, P99, P99.9)
- Instance health status
- Environment health aggregation

**Alternative method:**
You can also enable enhanced health by setting the health check type to "enhanced":

```yaml
health_check:
  type: enhanced  # This also enables enhanced health reporting
  path: /health
```

### CloudWatch Metrics

Enable custom CloudWatch metrics collection for your auto-scaling group and application.

```yaml
monitoring:
  cloudwatch_metrics: true
```

**What it enables:**
- Detailed CloudWatch monitoring for EC2 instances
- Auto-scaling group metrics
- Application request and latency metrics

### CloudWatch Logs

Stream application logs to CloudWatch for centralized logging and analysis.

```yaml
monitoring:
  cloudwatch_logs:
    enabled: true           # Enable log streaming
    retention_days: 7       # How long to keep logs (1, 3, 5, 7, 14, 30, 60, 90, 120, 150, 180, 365, 400, 545, 731, 1827, 3653)
    stream_logs: true       # Stream health and application logs
```

**Configuration options:**

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | boolean | false | Enable CloudWatch Logs streaming |
| `retention_days` | integer | - | Log retention period in days (see valid values above) |
| `stream_logs` | boolean | true | Stream application and health logs |

## Available Metrics

When enhanced health reporting is enabled, the following metrics become available in CloudWatch:

### Request Metrics

| Metric Name | Description | Unit |
|-------------|-------------|------|
| `ApplicationRequests2xx` | Number of successful requests (200-299 status codes) | Count |
| `ApplicationRequests3xx` | Number of redirect requests (300-399 status codes) | Count |
| `ApplicationRequests4xx` | Number of client error requests (400-499 status codes) | Count |
| `ApplicationRequests5xx` | Number of server error requests (500-599 status codes) | Count |
| `ApplicationRequestsTotal` | Total number of requests | Count |

### Latency Metrics

| Metric Name | Description | Unit |
|-------------|-------------|------|
| `ApplicationLatencyP10` | 10th percentile latency | Seconds |
| `ApplicationLatencyP50` | 50th percentile (median) latency | Seconds |
| `ApplicationLatencyP75` | 75th percentile latency | Seconds |
| `ApplicationLatencyP85` | 85th percentile latency | Seconds |
| `ApplicationLatencyP90` | 90th percentile latency | Seconds |
| `ApplicationLatencyP95` | 95th percentile latency | Seconds |
| `ApplicationLatencyP99` | 99th percentile latency | Seconds |
| `ApplicationLatencyP99.9` | 99.9th percentile latency | Seconds |

### Health Metrics

| Metric Name | Description | Unit |
|-------------|-------------|------|
| `InstanceHealth` | Health status of individual instances | - |
| `EnvironmentHealth` | Overall environment health | - |
| `InstancesOk` | Number of instances with OK health | Count |
| `InstancesWarning` | Number of instances with Warning health | Count |
| `InstancesDegraded` | Number of instances with Degraded health | Count |
| `InstancesSevere` | Number of instances with Severe health | Count |

### System Metrics

| Metric Name | Description | Unit |
|-------------|-------------|------|
| `CPUIdle` | Percentage of time CPU is idle | Percent |
| `CPUUser` | Percentage of time CPU spent in user space | Percent |
| `CPUSystem` | Percentage of time CPU spent in kernel space | Percent |
| `LoadAverage1min` | 1-minute load average | - |
| `LoadAverage5min` | 5-minute load average | - |
| `RootFilesystemUtil` | Root filesystem utilization | Percent |

## Viewing Metrics

### AWS Console

1. Go to **CloudWatch** → **Metrics** → **All Metrics**
2. Select **AWS/ElasticBeanstalk** namespace
3. Filter by:
   - **Dimension:** `EnvironmentName = your-env-name`
   - **Metric Name:** Select desired metrics (ApplicationRequests2xx, etc.)

### AWS CLI

List all available metrics for your environment:

```bash
aws cloudwatch list-metrics \
  --namespace AWS/ElasticBeanstalk \
  --dimensions Name=EnvironmentName,Value=your-env-name \
  --region us-east-1
```

Get metric statistics (example: ApplicationRequests2xx):

```bash
aws cloudwatch get-metric-statistics \
  --namespace AWS/ElasticBeanstalk \
  --metric-name ApplicationRequests2xx \
  --dimensions Name=EnvironmentName,Value=your-env-name \
  --start-time 2024-10-28T00:00:00Z \
  --end-time 2024-10-28T23:59:59Z \
  --period 300 \
  --statistics Sum \
  --region us-east-1
```

## Cost Considerations

### Enhanced Health Reporting

**Cost:** ~$0.004/hour per environment = ~$3/month

- Charged per environment, not per instance
- Includes all enhanced metrics
- Worth it for production environments

### CloudWatch Metrics

**Cost:** First 10 metrics are free, then $0.30/metric/month

- Enhanced health reporting provides ~40 metrics
- After free tier: ~$9/month for all metrics
- Only charged for metrics that are published

### CloudWatch Logs

**Cost:**
- **Ingestion:** $0.50/GB ingested
- **Storage:** $0.03/GB/month (after free tier)
- **Data transfer:** Standard AWS data transfer rates

**Estimation:**
- Low-traffic app: ~0.1-0.5 GB/month = $0.05-0.25/month
- Medium-traffic app: ~1-5 GB/month = $0.50-2.50/month

### Total Estimated Monthly Cost

| Configuration | Monthly Cost (Estimated) |
|---------------|--------------------------|
| Basic (no monitoring) | $0 |
| Enhanced Health only | ~$3 |
| Enhanced Health + Metrics | ~$12 |
| Full monitoring (all features) | ~$15-20 |

**Note:** Costs are estimates and vary by region and usage.

## Examples

### Example 1: Minimal Monitoring (Enhanced Health Only)

Best for: Development/staging environments

```yaml
version: "1.0"

provider:
  name: aws
  region: us-east-1

application:
  name: my-app

environment:
  name: my-app-dev
  cname: my-app-dev

deployment:
  platform: docker
  source:
    type: local
    path: "."

instance:
  type: t3.micro
  environment_type: SingleInstance

health_check:
  type: enhanced
  path: /health

monitoring:
  enhanced_health: true  # Only enhanced health, no CloudWatch logs
```

### Example 2: Full Monitoring (Production)

Best for: Production environments requiring comprehensive observability

```yaml
version: "1.0"

provider:
  name: aws
  region: us-east-1

application:
  name: my-app

environment:
  name: my-app-prod
  cname: my-app-prod

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

monitoring:
  enhanced_health: true
  cloudwatch_metrics: true
  cloudwatch_logs:
    enabled: true
    retention_days: 30  # Keep logs for 30 days
    stream_logs: true

iam:
  instance_profile: aws-elasticbeanstalk-ec2-role  # Needs CloudWatch permissions
```

### Example 3: Cost-Optimized Monitoring

Best for: Production on a budget

```yaml
monitoring:
  enhanced_health: true          # Get essential metrics
  cloudwatch_metrics: false      # Skip extra metrics to reduce cost
  cloudwatch_logs:
    enabled: true
    retention_days: 7            # Keep logs for 1 week only
    stream_logs: true
```

## Backward Compatibility

The monitoring configuration is **optional**. Existing manifests without monitoring configuration will continue to work with basic health reporting.

**Legacy behavior:**
- `health_check.type: basic` → Basic health reporting (no enhanced metrics)
- `health_check.type: enhanced` → Automatically enables enhanced health reporting

**New behavior:**
- `monitoring.enhanced_health: true` → Explicitly enables enhanced health
- Both `health_check.type: enhanced` and `monitoring.enhanced_health: true` can be used together

## Troubleshooting

### Metrics Not Appearing

**Problem:** CloudWatch metrics not showing up after deployment.

**Solutions:**
1. **Wait 5-15 minutes** - Metrics can take time to appear
2. **Generate traffic** - Metrics won't appear without requests
3. **Check configuration:**
   ```bash
   aws elasticbeanstalk describe-configuration-settings \
     --environment-name your-env-name \
     --application-name your-app-name \
     --query 'ConfigurationSettings[0].OptionSettings[?Namespace==`aws:elasticbeanstalk:healthreporting:system`]'
   ```
4. **Verify SystemType is "enhanced"**

### Logs Not Streaming

**Problem:** Application logs not appearing in CloudWatch.

**Solutions:**
1. **Check IAM permissions** - Instance profile needs CloudWatch Logs permissions:
   - `logs:CreateLogGroup`
   - `logs:CreateLogStream`
   - `logs:PutLogEvents`
2. **Verify configuration:**
   ```bash
   aws elasticbeanstalk describe-configuration-settings \
     --environment-name your-env-name \
     --application-name your-app-name \
     --query 'ConfigurationSettings[0].OptionSettings[?Namespace==`aws:elasticbeanstalk:cloudwatch:logs`]'
   ```

### High Costs

**Problem:** CloudWatch costs are higher than expected.

**Solutions:**
1. **Reduce log retention** - Change `retention_days` to 7 or less
2. **Disable metrics for non-production** - Set `cloudwatch_metrics: false` for dev/staging
3. **Filter logs** - Configure your application to log less verbose output
4. **Use sampling** - Reduce metric publishing frequency

## IAM Permissions Required

Your instance profile needs these permissions for full monitoring:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "cloudwatch:PutMetricData",
        "cloudwatch:GetMetricStatistics",
        "cloudwatch:ListMetrics"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents",
        "logs:DescribeLogStreams"
      ],
      "Resource": "arn:aws:logs:*:*:log-group:/aws/elasticbeanstalk/*"
    }
  ]
}
```

## Best Practices

1. **Enable enhanced health for all environments** - Even basic enhanced health is valuable
2. **Use CloudWatch Logs for production** - Essential for debugging production issues
3. **Set appropriate retention** - Balance cost vs compliance requirements
4. **Monitor costs** - Set up CloudWatch billing alarms
5. **Test in dev first** - Verify monitoring configuration before production deployment

## Related Documentation

- [AWS Elastic Beanstalk Enhanced Health Reporting](https://docs.aws.amazon.com/elasticbeanstalk/latest/dg/health-enhanced.html)
- [AWS CloudWatch Metrics](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/working_with_metrics.html)
- [AWS CloudWatch Logs](https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/WhatIsCloudWatchLogs.html)
- [Cloud-Deploy Main Documentation](../README.md)

## Support

For issues or questions:
- GitHub Issues: https://github.com/jvreagan/cloud-deploy/issues
- Documentation: https://github.com/jvreagan/cloud-deploy/docs
