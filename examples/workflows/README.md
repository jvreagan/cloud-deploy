# GitHub Actions Deployment Workflows

This directory contains example GitHub Actions workflows for integrating cloud-deploy into your CI/CD pipeline.

## Quick Start

### 1. Choose Your Deployment Method

| File | Method | When to Use |
|------|--------|-------------|
| `manual-deploy.yml` | Manual trigger via UI | Production deployments, on-demand operations |
| `label-deploy.yml` | PR labels | Review apps, staging deployments, testing |
| `app-ci-cd.yml` | Full CI/CD pipeline | Complete application deployment workflow |

### 2. Copy Workflows to Your Repository

**Option A: Manual deployment only**
```bash
# In your application repository
mkdir -p .github/workflows
cp manual-deploy.yml .github/workflows/
cp deploy.yml .github/workflows/  # Required by manual-deploy.yml
```

**Option B: Label-based deployment**
```bash
cp label-deploy.yml .github/workflows/
cp deploy.yml .github/workflows/  # Required by label-deploy.yml
```

**Option C: Complete CI/CD**
```bash
cp app-ci-cd.yml .github/workflows/
# Customize app-ci-cd.yml for your application
```

### 3. Configure Secrets

Add your cloud provider credentials in **Settings → Secrets and variables → Actions**:

**For AWS:**
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`

**For GCP:**
- `GCP_PROJECT_ID`
- `GCP_CREDENTIALS` (service account JSON)
- `GCP_BILLING_ACCOUNT_ID`

### 4. Create Deployment Manifests

Create manifest files in your repository:

```
your-repo/
├── manifests/
│   ├── staging-aws.yaml
│   └── production-gcp.yaml
└── .github/
    └── workflows/
        └── manual-deploy.yml
```

### 5. Deploy!

**Manual deployment:**
1. Go to **Actions** tab
2. Select **Manual Deployment**
3. Click **Run workflow**
4. Fill in parameters and deploy

**Label-based deployment:**
1. Create a PR
2. Add label: `deploy:aws:staging`
3. Automatic deployment triggered

## Workflow Files

### deploy.yml (Reusable Workflow)

Core deployment workflow used by other workflows. **Always include this file.**

**Features:**
- Reusable across workflows
- Supports AWS and GCP
- Environment-based approvals
- Automatic PR comments
- Deployment logs uploaded as artifacts

### manual-deploy.yml

Trigger deployments manually from GitHub Actions UI.

**Use cases:**
- Production deployments
- Emergency updates
- Status checks
- Stopping/destroying resources

**Safety features:**
- Production confirmation required
- Destroy confirmation required
- Environment-based approvals
- Validation step

**Example usage:**
```yaml
# Just copy to .github/workflows/manual-deploy.yml
# No modifications needed!
```

### label-deploy.yml

Automatically deploy when PR is labeled.

**Use cases:**
- Review apps
- Staging deployments from PRs
- QA environments
- Testing features before merge

**Label format:**
```
deploy:provider:environment

Examples:
- deploy:aws:staging
- deploy:gcp:production
```

**How to create labels:**
```bash
gh label create "deploy:aws:staging" --color "0e8a16"
gh label create "deploy:gcp:production" --color "b60205"
```

**Example usage:**
```yaml
# Copy to .github/workflows/label-deploy.yml
# Customize manifest_path if needed
```

### app-ci-cd.yml (Complete Example)

Full-featured CI/CD pipeline example for your application.

**Pipeline stages:**
1. **Build & Test**
   - Install dependencies
   - Run linter
   - Run tests
   - Build Docker image

2. **Deploy to Staging** (automatic on `develop` push)
   - Deploy to staging environment
   - Run smoke tests
   - Notify team

3. **Deploy to Production** (manual approval required)
   - Deploy to production
   - Verify deployment
   - Create release
   - Rollback on failure

**Customization required:**
```yaml
# Update these variables
env:
  APP_NAME: my-app              # Your app name
  DOCKER_IMAGE: my-app:${{ github.sha }}

# Update build steps for your language/framework
- name: Set up build environment
  uses: actions/setup-node@v4   # Change to your language

- name: Build application
  run: npm run build             # Change to your build command
```

## Common Patterns

### Pattern 1: Staging First, Then Production

```yaml
# Deploy to staging automatically
deploy-staging:
  if: github.ref == 'refs/heads/develop'
  uses: ./.github/workflows/deploy.yml
  with:
    provider: aws
    environment: staging
    manifest_path: manifests/staging.yaml

# Deploy to production manually with approval
deploy-production:
  if: github.ref == 'refs/heads/main' && contains(github.event.head_commit.message, '[deploy]')
  uses: ./.github/workflows/deploy.yml
  with:
    provider: gcp
    environment: production  # Requires approval
    manifest_path: manifests/production.yaml
```

### Pattern 2: Multi-Cloud Deployment

Deploy to both AWS and GCP in parallel:

```yaml
deploy:
  strategy:
    matrix:
      include:
        - provider: aws
          manifest: manifests/aws-staging.yaml
        - provider: gcp
          manifest: manifests/gcp-staging.yaml

  uses: ./.github/workflows/deploy.yml
  with:
    provider: ${{ matrix.provider }}
    environment: staging
    manifest_path: ${{ matrix.manifest }}
```

### Pattern 3: Canary Deployment

Deploy to a small subset first, then full deployment:

```yaml
deploy-canary:
  uses: ./.github/workflows/deploy.yml
  with:
    provider: gcp
    environment: canary
    manifest_path: manifests/canary.yaml  # MinInstances: 1

verify-canary:
  needs: deploy-canary
  runs-on: ubuntu-latest
  steps:
    - name: Run tests against canary
      run: npm run test:e2e

deploy-full:
  needs: verify-canary
  uses: ./.github/workflows/deploy.yml
  with:
    provider: gcp
    environment: production
    manifest_path: manifests/production.yaml
```

### Pattern 4: Scheduled Status Checks

Check deployment status on a schedule:

```yaml
name: Health Check

on:
  schedule:
    - cron: '0 */4 * * *'  # Every 4 hours

jobs:
  check-production:
    uses: ./.github/workflows/deploy.yml
    with:
      provider: gcp
      environment: production
      manifest_path: manifests/production.yaml
      command: status
```

## Environment Configuration

### Staging Environment

**Settings → Environments → New environment: "staging"**

- **Protection rules:** None (allow automatic deployment)
- **Deployment branches:** `develop`, `feature/*`
- **Secrets:** Use staging-specific credentials

### Production Environment

**Settings → Environments → New environment: "production"**

- **Protection rules:**
  - Required reviewers: Team leads, SRE team
  - Wait timer: 5 minutes (cooling off period)
- **Deployment branches:** `main` only
- **Secrets:** Use production credentials

## Troubleshooting

### Workflow not running

**Problem:** Manual deployment workflow doesn't appear

**Solution:** Ensure workflows are committed to the default branch (`main`)

```bash
git checkout main
git add .github/workflows/
git commit -m "Add deployment workflows"
git push origin main
```

### Label deployment not triggering

**Problem:** Adding label to PR doesn't trigger deployment

**Solution:**
1. Ensure `label-deploy.yml` is on the default branch
2. Check label format: `deploy:provider:environment`
3. Create labels if they don't exist:
   ```bash
   gh label create "deploy:aws:staging"
   ```

### Secrets not found

**Problem:** `AWS credentials not configured in GitHub secrets`

**Solution:**
1. Go to **Settings → Secrets and variables → Actions**
2. Click **New repository secret**
3. Add required secrets (names are case-sensitive)

### Deployment fails silently

**Problem:** Workflow succeeds but deployment failed

**Solution:**
1. Check deployment logs artifact
2. Review cloud-deploy output in workflow logs
3. Run `status` command to check actual deployment state

## Best Practices

1. **Always test in staging first**
   - Use `deploy:aws:staging` label on PRs
   - Verify staging before production

2. **Use environment protection**
   - Require approvals for production
   - Add wait timers for safety

3. **Monitor deployments**
   - Set up Slack/email notifications
   - Check deployment logs regularly

4. **Version your manifests**
   - Keep manifests in version control
   - Use separate manifests per environment

5. **Secure your credentials**
   - Use GitHub Secrets (never commit credentials)
   - Rotate credentials regularly
   - Use least-privilege IAM policies

6. **Document your process**
   - Add README in workflows directory
   - Document required secrets
   - Maintain runbooks for common operations

## Next Steps

- Read the [complete GitHub Actions guide](../../docs/GITHUB_ACTIONS.md)
- Configure [GitHub Environments](https://docs.github.com/en/actions/deployment/targeting-different-environments/using-environments-for-deployment)
- Set up [deployment protection rules](https://docs.github.com/en/actions/deployment/targeting-different-environments/using-environments-for-deployment#deployment-protection-rules)
- Explore [cloud-deploy features](../../README.md)

## Questions?

Open an issue: https://github.com/jvreagan/cloud-deploy/issues
