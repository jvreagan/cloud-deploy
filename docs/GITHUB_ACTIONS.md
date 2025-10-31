# GitHub Actions Integration Guide

This guide shows you how to integrate cloud-deploy into your GitHub Actions CI/CD pipeline.

## Table of Contents

- [Overview](#overview)
- [Deployment Workflows](#deployment-workflows)
- [Setup Instructions](#setup-instructions)
- [Usage Examples](#usage-examples)
- [Advanced Configuration](#advanced-configuration)
- [Troubleshooting](#troubleshooting)

## Overview

cloud-deploy provides three flexible ways to trigger deployments in GitHub Actions:

| Method | Trigger | Use Case | Approval Required |
|--------|---------|----------|-------------------|
| **Manual Dispatch** | GitHub UI button | On-demand deployments | Optional (via environments) |
| **PR Labels** | Adding labels to PRs | Review app deployments | Optional |
| **Commit Message** | `[deploy]` in commit | Automatic after merge | Optional (via environments) |

## Deployment Workflows

### 1. Manual Deployment Workflow

**File:** `.github/workflows/manual-deploy.yml`

Deploy on-demand from the GitHub Actions tab with full control over provider, environment, and command.

**Features:**
- ✅ Manual trigger from GitHub UI
- ✅ Choose provider (AWS or GCP)
- ✅ Choose environment (staging or production)
- ✅ Choose command (deploy, status, stop, destroy)
- ✅ Production safety confirmation required
- ✅ Environment-based approvals

**How to use:**
1. Go to **Actions** tab in GitHub
2. Select **"Manual Deployment"** workflow
3. Click **"Run workflow"**
4. Fill in the parameters:
   - **Provider:** `aws` or `gcp`
   - **Environment:** `staging` or `production`
   - **Manifest path:** Path to your manifest file
   - **Command:** `deploy`, `status`, `stop`, or `destroy`
   - **Confirm production:** Type `CONFIRM` for production/destroy
5. Click **"Run workflow"**

### 2. Label-based Deployment Workflow

**File:** `.github/workflows/label-deploy.yml`

Automatically deploy when you add specific labels to pull requests.

**Features:**
- ✅ Deploys when PR is labeled
- ✅ Supports review apps for testing
- ✅ Auto-removes label after deployment
- ✅ Useful for QA and staging environments

**Available labels:**
- `deploy:aws:staging` - Deploy to AWS staging
- `deploy:aws:production` - Deploy to AWS production
- `deploy:gcp:staging` - Deploy to GCP staging
- `deploy:gcp:production` - Deploy to GCP production

**How to use:**
1. Create a pull request
2. Add one of the deployment labels
3. GitHub Actions automatically deploys
4. Label is removed after deployment

**Create the labels:**
```bash
# In your repository settings or via GitHub CLI
gh label create "deploy:aws:staging" --color "0e8a16" --description "Deploy to AWS staging"
gh label create "deploy:aws:production" --color "b60205" --description "Deploy to AWS production"
gh label create "deploy:gcp:staging" --color "0e8a16" --description "Deploy to GCP staging"
gh label create "deploy:gcp:production" --color "b60205" --description "Deploy to GCP production"
```

### 3. Reusable Deployment Workflow

**File:** `.github/workflows/deploy.yml`

A reusable workflow that handles the actual deployment. Called by other workflows.

**Features:**
- ✅ Reusable across multiple workflows
- ✅ Consistent deployment logic
- ✅ Automatic artifact uploads
- ✅ PR comments with deployment status
- ✅ Environment-based approvals

## Setup Instructions

### Step 1: Copy Workflows to Your Repository

If you're using cloud-deploy for your own application:

```bash
# In your application repository
mkdir -p .github/workflows

# Copy the workflows you want to use
cp /path/to/cloud-deploy/.github/workflows/deploy.yml .github/workflows/
cp /path/to/cloud-deploy/.github/workflows/manual-deploy.yml .github/workflows/
cp /path/to/cloud-deploy/.github/workflows/label-deploy.yml .github/workflows/
```

### Step 2: Configure GitHub Secrets

Add your cloud provider credentials as repository secrets:

**Settings → Secrets and variables → Actions → New repository secret**

#### For AWS Deployments:

```
AWS_ACCESS_KEY_ID = AKIAIOSFODNN7EXAMPLE
AWS_SECRET_ACCESS_KEY = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

#### For GCP Deployments:

```
GCP_PROJECT_ID = my-project-id
GCP_BILLING_ACCOUNT_ID = 123456-123456-123456
GCP_CREDENTIALS = {"type":"service_account","project_id":"..."}
```

**Getting GCP credentials:**
```bash
# Create and download service account key
gcloud iam service-accounts keys create key.json \
  --iam-account=my-sa@my-project.iam.gserviceaccount.com

# Copy the JSON content and paste into GCP_CREDENTIALS secret
cat key.json
```

### Step 3: Create GitHub Environments (Optional)

Environments provide additional controls like required reviewers and secrets scoping.

**Settings → Environments → New environment**

Create environments:
- `staging` (no approvals required)
- `production` (require approvals from team leads)

**Environment protection rules:**
- Required reviewers: Add team members who must approve
- Wait timer: Add a delay before deployment
- Deployment branches: Restrict which branches can deploy

### Step 4: Create Deployment Manifests

Create manifest files for each environment:

```
your-repo/
├── manifests/
│   ├── staging-aws.yaml
│   ├── staging-gcp.yaml
│   ├── production-aws.yaml
│   └── production-gcp.yaml
└── .github/
    └── workflows/
        ├── deploy.yml
        ├── manual-deploy.yml
        └── label-deploy.yml
```

**Example: `manifests/staging-aws.yaml`**
```yaml
version: "1.0"

provider:
  name: aws
  region: us-east-1

application:
  name: my-app-staging
  description: "Staging environment"

environment:
  name: my-app-staging-env
  cname: my-app-staging

deployment:
  platform: docker
  source:
    type: local
    path: .

instance:
  type: t3.micro
  environment_type: SingleInstance

health_check:
  type: basic
  path: /health
```

## Usage Examples

### Example 1: Manual Deployment to Staging

1. Go to **Actions** → **Manual Deployment**
2. Click **Run workflow**
3. Select:
   - Provider: `aws`
   - Environment: `staging`
   - Manifest: `manifests/staging-aws.yaml`
   - Command: `deploy`
4. Click **Run workflow**

### Example 2: Deploy PR for Testing

1. Create a PR with your changes
2. Add label: `deploy:aws:staging`
3. Wait for deployment to complete
4. Test your changes at the deployment URL
5. Label is automatically removed

### Example 3: Production Deployment with Approval

1. Merge PR to `main`
2. Go to **Actions** → **Manual Deployment**
3. Click **Run workflow**
4. Select:
   - Provider: `gcp`
   - Environment: `production`
   - Manifest: `manifests/production-gcp.yaml`
   - Command: `deploy`
   - Confirm: `CONFIRM`
5. Approval request sent to reviewers
6. After approval, deployment proceeds

### Example 4: Check Deployment Status

1. Go to **Actions** → **Manual Deployment**
2. Select:
   - Command: `status`
   - Environment: `production`
3. View deployment status in logs

### Example 5: Emergency Stop

1. Go to **Actions** → **Manual Deployment**
2. Select:
   - Command: `stop`
   - Environment: `staging`
   - Confirm: `CONFIRM`
3. Resources are stopped but not destroyed

## Advanced Configuration

### Custom Manifest Paths

Modify workflows to use custom paths:

```yaml
# In label-deploy.yml
manifest_path: 'deploy/configs/staging.yaml'  # Custom path
```

### Multiple Providers

Deploy to multiple providers in parallel:

```yaml
jobs:
  deploy-aws:
    uses: ./.github/workflows/deploy.yml
    with:
      provider: aws
      environment: staging
      manifest_path: manifests/staging-aws.yaml
    secrets: inherit

  deploy-gcp:
    uses: ./.github/workflows/deploy.yml
    with:
      provider: gcp
      environment: staging
      manifest_path: manifests/staging-gcp.yaml
    secrets: inherit
```

### Automated Rollback

Add rollback on deployment failure:

```yaml
- name: Deploy
  id: deploy
  run: cloud-deploy -manifest ${{ inputs.manifest_path }} -command deploy

- name: Rollback on failure
  if: failure()
  run: |
    echo "Deployment failed, rolling back..."
    # Deploy previous version
    cloud-deploy -manifest manifests/previous.yaml -command deploy
```

### Slack Notifications

Add Slack notifications:

```yaml
- name: Notify Slack
  if: always()
  uses: slackapi/slack-github-action@v1
  with:
    payload: |
      {
        "text": "Deployment to ${{ inputs.environment }} completed",
        "status": "${{ job.status }}"
      }
  env:
    SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK }}
```

### Integration with Existing CI/CD

See `examples/workflows/app-ci-cd.yml` for a complete example showing:
- Build and test
- Automatic staging deployment
- Manual production deployment
- Smoke tests
- Rollback capabilities

## Troubleshooting

### Workflow not appearing in Actions tab

- Ensure workflows are in `.github/workflows/` directory
- Check YAML syntax with a validator
- Workflows must be on the default branch to appear

### Secrets not available

- Verify secrets are set in **Settings → Secrets and variables → Actions**
- Check secret names match exactly (case-sensitive)
- Secrets are not available in forked repositories (for security)

### Deployment fails with credentials error

```
❌ AWS credentials not configured in GitHub secrets
```

**Solution:** Add missing secrets as described in [Step 2](#step-2-configure-github-secrets)

### Environment approval required but none configured

- Go to **Settings → Environments**
- Click on the environment (e.g., `production`)
- Add required reviewers

### Label deployment not triggering

- Ensure `label-deploy.yml` is on the default branch
- Check label format: `deploy:provider:environment`
- Verify labels are created in repository

### Manual workflow "Run workflow" button disabled

- Workflows must be on the default branch (usually `main`)
- Push the workflow file to `main` first
- Refresh the Actions page

## Best Practices

1. **Use Environments**: Configure GitHub environments for staging and production with appropriate protections

2. **Require Approvals**: Always require manual approval for production deployments

3. **Test in Staging**: Deploy to staging first, test, then promote to production

4. **Monitor Deployments**: Set up alerts and notifications for deployment status

5. **Document Manifests**: Keep manifest files well-documented and version controlled

6. **Rotate Credentials**: Regularly rotate cloud provider credentials

7. **Use Branch Protection**: Protect `main` branch and require PR reviews

8. **Audit Deployments**: Review deployment logs and maintain audit trail

## Next Steps

- Review the [complete CI/CD example](../examples/workflows/app-ci-cd.yml)
- Set up [GitHub Environments](https://docs.github.com/en/actions/deployment/targeting-different-environments/using-environments-for-deployment)
- Configure [required reviewers](https://docs.github.com/en/actions/deployment/targeting-different-environments/using-environments-for-deployment#required-reviewers)
- Explore [cloud-deploy manifest options](../README.md#manifest-reference)

## Questions?

- Open an issue: https://github.com/jvreagan/cloud-deploy/issues
- Check the docs: https://github.com/jvreagan/cloud-deploy/tree/main/docs
- See examples: https://github.com/jvreagan/cloud-deploy/tree/main/examples
