# GitHub Actions Integration Guide

This guide shows you how to integrate cloud-deploy into your GitHub Actions CI/CD pipeline.

## Table of Contents

- [Overview](#overview)
- [How It Works: Installing cloud-deploy in Your Workflows](#how-it-works-installing-cloud-deploy-in-your-workflows)
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

## How It Works: Installing cloud-deploy in Your Workflows

> **Important:** This guide is for using cloud-deploy in YOUR APPLICATION repositories, not the cloud-deploy repository itself.

### The Big Picture

When you want to deploy your application using cloud-deploy via GitHub Actions:

1. **Your application repository** contains your app code and deployment workflows
2. **GitHub Actions runner** starts fresh for each workflow run (no pre-installed tools)
3. **cloud-deploy is installed** during the workflow (just like `aws-cli`, `kubectl`, or `terraform`)
4. **cloud-deploy runs** to deploy your application to AWS/GCP
5. **Runner is destroyed** after workflow completes

### Installation Methods

cloud-deploy must be installed in your workflow before you can use it. Here are three ways to install it:

#### Method 1: From GitHub Releases (Recommended)

This is the fastest and most reliable method. Downloads the pre-built binary from the latest release.

```yaml
- name: Install cloud-deploy
  run: |
    # Download from GitHub releases
    curl -L https://github.com/jvreagan/cloud-deploy/releases/latest/download/cloud-deploy_Linux_x86_64.tar.gz | tar -xz

    # Move to system path
    sudo mv cloud-deploy /usr/local/bin/

    # Verify installation
    cloud-deploy -version
```

**Pros:**
- ✅ Fast (pre-built binary)
- ✅ Reliable (pinned release version)
- ✅ No compilation needed

#### Method 2: Using `go install`

Install directly from source using Go. Good for getting the latest code.

```yaml
- name: Set up Go
  uses: actions/setup-go@v5
  with:
    go-version: '1.23'

- name: Install cloud-deploy
  run: go install github.com/jvreagan/cloud-deploy/cmd/cloud-deploy@latest
```

**Pros:**
- ✅ Always gets latest code
- ✅ Simple one-liner

**Cons:**
- ⚠️ Slower (compiles from source)
- ⚠️ May get unreleased changes

#### Method 3: Build from Source

Clone and build the repository. Useful for development or custom builds.

```yaml
- name: Set up Go
  uses: actions/setup-go@v5
  with:
    go-version: '1.23'

- name: Build cloud-deploy from source
  run: |
    git clone https://github.com/jvreagan/cloud-deploy.git
    cd cloud-deploy
    go build -o /usr/local/bin/cloud-deploy ./cmd/cloud-deploy
    cloud-deploy -version
```

**Pros:**
- ✅ Full control over build process
- ✅ Can build specific commits/branches

**Cons:**
- ⚠️ Slowest method
- ⚠️ Requires Go setup

### Complete Workflow Example

Here's what a complete deployment workflow looks like in your application repository:

```yaml
# .github/workflows/deploy.yml (in YOUR APPLICATION repo)
name: Deploy Application

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      # 1. Checkout YOUR application code
      - uses: actions/checkout@v4

      # 2. Build YOUR application
      - name: Build application
        run: |
          docker build -t my-app:${{ github.sha }} .

      # 3. Install cloud-deploy
      - name: Install cloud-deploy
        run: |
          curl -L https://github.com/jvreagan/cloud-deploy/releases/latest/download/cloud-deploy_Linux_x86_64.tar.gz | tar -xz
          sudo mv cloud-deploy /usr/local/bin/
          cloud-deploy -version

      # 4. Deploy using cloud-deploy
      - name: Deploy to AWS
        run: |
          cloud-deploy \
            -manifest manifests/production-aws.yaml \
            -command deploy
        env:
          AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
          AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          AWS_REGION: us-east-1
```

### Key Concepts

**Q: Is cloud-deploy pre-installed on GitHub runners?**
A: No. You must install it in your workflow, just like any other CLI tool.

**Q: Do I need to install it every time?**
A: Yes. GitHub Actions runners are ephemeral (fresh for each run). The installation is fast (5-10 seconds).

**Q: Where do I put the workflow file?**
A: In YOUR APPLICATION repository, in `.github/workflows/deploy.yml`

**Q: Where do I put the manifest file?**
A: In YOUR APPLICATION repository, typically in `manifests/` directory or root

**Q: Can I cache the cloud-deploy binary?**
A: Yes, but it's usually not worth it. The download is fast and caching adds complexity.

### Repository Structure

Your application repository should look like this:

```
your-app-repo/
├── .github/
│   └── workflows/
│       ├── deploy.yml          # Workflow that installs cloud-deploy
│       └── manual-deploy.yml   # Optional: manual deployment workflow
├── manifests/
│   ├── staging-aws.yaml        # Deployment manifest for staging
│   └── production-gcp.yaml     # Deployment manifest for production
├── src/                        # Your application code
├── Dockerfile                  # Your application's Dockerfile
└── README.md
```

### What About the cloud-deploy Repo?

The `cloud-deploy` repository:
- ✅ Contains the cloud-deploy tool source code
- ✅ Has its own CI/CD for testing and releasing cloud-deploy itself
- ❌ Does NOT deploy using cloud-deploy (that would be circular!)
- ❌ Is NOT where you put your application's deployment workflows

Your application repositories:
- ✅ Install and use cloud-deploy to deploy YOUR applications
- ✅ Contain deployment manifests for YOUR applications
- ✅ Run cloud-deploy commands in workflows

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
