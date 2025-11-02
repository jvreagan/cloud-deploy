# HashiCorp Vault Integration

## Overview

cloud-deploy integrates with HashiCorp Vault (Open Source) to provide unified, multi-cloud secret management. This allows you to store secrets once in Vault and deploy them to any cloud provider (AWS, GCP, Azure, OCI).

## Two Ways to Use Vault

1. **Cloud Provider Credentials** - Store AWS, GCP, Azure credentials in Vault for multi-cloud deployments
   - 📖 **Complete guide:** [VAULT_CREDENTIALS.md](../VAULT_CREDENTIALS.md)
   - Recommended for production deployments
   - Centralized credential rotation and auditing

2. **Application Secrets** (this guide) - Store database passwords, API keys, and other application secrets
   - Inject secrets as environment variables during deployment
   - Works with all cloud providers
   - Covered in detail below

## Why Vault?

**Multi-Cloud Flexibility:**
- Single source of truth for secrets across all cloud providers
- Deploy the same application to AWS, GCP, or Azure using the same secrets
- No vendor lock-in to cloud-specific secret managers

**Security Benefits:**
- Centralized secret management and auditing
- Dynamic secret generation (auto-rotating credentials)
- Fine-grained access control policies
- Encryption at rest and in transit

**Open Source:**
- Free to use
- Self-hosted - you control your secrets
- No per-secret pricing

## Architecture

```
┌─────────────────┐
│ deploy-manifest │
│   (secrets:)    │
└────────┬────────┘
         │
         ↓
┌────────────────────┐      ┌──────────────┐
│  cloud-deploy CLI  │─────→│ Vault Server │
└────────┬───────────┘      └──────────────┘
         │                   (fetch secrets)
         ↓
┌────────────────────┐
│ Cloud Provider API │
│ (AWS/GCP/Azure)    │
│ - Inject secrets   │
│   as env vars      │
└────────────────────┘
```

**Flow:**
1. User defines secrets in manifest referencing Vault paths
2. cloud-deploy authenticates to Vault using token/AppRole
3. cloud-deploy fetches secrets from Vault
4. Secrets are injected as environment variables during deployment
5. Application reads secrets via `os.Getenv()`

## Manifest Configuration

### Basic Vault Configuration

```yaml
version: "1.0"

# Vault configuration
vault:
  # Vault server address
  address: "http://127.0.0.1:8200"  # or https://vault.yourcompany.com

  # Authentication method
  auth:
    method: token  # or approle, aws-iam, gcp-iam

    # Token auth (simplest - for dev/testing)
    token: "${VAULT_TOKEN}"  # Read from environment variable

    # AppRole auth (recommended for production)
    # role_id: "${VAULT_ROLE_ID}"
    # secret_id: "${VAULT_SECRET_ID}"

# Application secrets from Vault
secrets:
  - name: DATABASE_URL
    vault_path: secret/data/myapp/database
    vault_key: url

  - name: STRIPE_API_KEY
    vault_path: secret/data/myapp/stripe
    vault_key: api_key

  - name: JWT_SECRET
    vault_path: secret/data/myapp/jwt
    vault_key: signing_key

# Regular environment variables (non-secret)
environment_variables:
  ENV: production
  LOG_LEVEL: info

provider:
  name: aws
  region: us-east-1

application:
  name: myapp

# ... rest of manifest
```

### Authentication Methods

**1. Token Authentication (Simplest)**
```yaml
vault:
  address: "http://127.0.0.1:8200"
  auth:
    method: token
    token: "${VAULT_TOKEN}"  # Set: export VAULT_TOKEN=hvs.xxx
```

**2. AppRole Authentication (Production)**
```yaml
vault:
  address: "https://vault.yourcompany.com"
  auth:
    method: approle
    role_id: "${VAULT_ROLE_ID}"
    secret_id: "${VAULT_SECRET_ID}"
```

**3. AWS IAM Authentication (AWS Deployments)**
```yaml
vault:
  address: "https://vault.yourcompany.com"
  auth:
    method: aws-iam
    role: myapp-deployer
```

**4. GCP IAM Authentication (GCP Deployments)**
```yaml
vault:
  address: "https://vault.yourcompany.com"
  auth:
    method: gcp-iam
    role: myapp-deployer
```

## Vault Setup Guide

### Step 1: Install Vault

**Option A: Docker (Quickest for testing)**
```bash
# Start Vault in dev mode (NOT for production!)
docker run -d --name vault \
  -p 8200:8200 \
  -e VAULT_DEV_ROOT_TOKEN_ID=myroot \
  hashicorp/vault:latest

# Set environment variables
export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='myroot'
```

**Option B: Binary Installation**
```bash
# Download from https://www.vaultproject.io/downloads
# Or use package manager:
brew install vault  # macOS
apt-get install vault  # Ubuntu/Debian
yum install vault  # RHEL/CentOS
```

**Option C: Production Deployment**
- Deploy to AWS EC2, GCP Compute Engine, or Kubernetes
- Use auto-unseal with cloud KMS
- Run 3+ nodes for high availability
- See: https://learn.hashicorp.com/collections/vault/day-one-raft

### Step 2: Initialize Vault (Production)

```bash
# Initialize Vault (first time only)
vault operator init

# Save the unseal keys and root token securely!
# You'll get 5 unseal keys and 1 root token

# Unseal Vault (requires 3 of 5 keys by default)
vault operator unseal <key1>
vault operator unseal <key2>
vault operator unseal <key3>

# Login with root token
vault login <root-token>
```

### Step 3: Enable KV Secrets Engine

```bash
# Enable KV v2 secrets engine at path "secret"
vault secrets enable -path=secret kv-v2

# Verify
vault secrets list
```

### Step 4: Store Secrets

```bash
# Store application secrets
vault kv put secret/myapp/database \
  url="postgresql://user:pass@host:5432/dbname"

vault kv put secret/myapp/stripe \
  api_key="sk_live_xxx"

vault kv put secret/myapp/jwt \
  signing_key="supersecretkey123"

# Verify secrets are stored
vault kv get secret/myapp/database
```

### Step 5: Create Policy for cloud-deploy

```bash
# Create policy file
cat > cloud-deploy-policy.hcl <<EOF
# Allow reading secrets for myapp
path "secret/data/myapp/*" {
  capabilities = ["read"]
}
EOF

# Create policy in Vault
vault policy write cloud-deploy cloud-deploy-policy.hcl

# Verify
vault policy read cloud-deploy
```

### Step 6: Create Token for cloud-deploy

```bash
# Create a token with the cloud-deploy policy
vault token create -policy=cloud-deploy

# Save the token
export VAULT_TOKEN=hvs.xxxxxx
```

## Usage Examples

### Example 1: Deploy to AWS with Vault Secrets

```yaml
# deploy-manifest.yaml
version: "1.0"

vault:
  address: "http://127.0.0.1:8200"
  auth:
    method: token
    token: "${VAULT_TOKEN}"

secrets:
  - name: DATABASE_URL
    vault_path: secret/data/myapp/database
    vault_key: url

provider:
  name: aws
  region: us-east-1

application:
  name: myapp

environment:
  name: myapp-prod

deployment:
  platform: docker
  source:
    type: local
    path: "."
```

**Deploy:**
```bash
export VAULT_TOKEN=hvs.xxx
cloud-deploy -command deploy -manifest deploy-manifest.yaml
```

**What happens:**
1. cloud-deploy connects to Vault at `http://127.0.0.1:8200`
2. Authenticates using `$VAULT_TOKEN`
3. Fetches secret from `secret/data/myapp/database` (key: `url`)
4. Deploys to AWS with `DATABASE_URL` environment variable set
5. Your app reads it: `os.Getenv("DATABASE_URL")`

### Example 2: Multi-Cloud with Same Secrets

**AWS Deployment:**
```yaml
# aws-manifest.yaml
vault:
  address: "https://vault.company.com"
  auth:
    method: token
    token: "${VAULT_TOKEN}"

secrets:
  - name: API_KEY
    vault_path: secret/data/shared/api
    vault_key: key

provider:
  name: aws
  region: us-east-1
```

**GCP Deployment:**
```yaml
# gcp-manifest.yaml
vault:
  address: "https://vault.company.com"
  auth:
    method: token
    token: "${VAULT_TOKEN}"

secrets:
  - name: API_KEY
    vault_path: secret/data/shared/api  # Same path!
    vault_key: key

provider:
  name: gcp
  region: us-central1
```

**Result:** Both AWS and GCP deployments use the exact same secrets from Vault.

### Example 3: Multiple Secrets

```yaml
vault:
  address: "http://127.0.0.1:8200"
  auth:
    method: token
    token: "${VAULT_TOKEN}"

secrets:
  # Database
  - name: DB_HOST
    vault_path: secret/data/myapp/database
    vault_key: host
  - name: DB_PASSWORD
    vault_path: secret/data/myapp/database
    vault_key: password

  # External APIs
  - name: STRIPE_KEY
    vault_path: secret/data/myapp/stripe
    vault_key: api_key
  - name: SENDGRID_KEY
    vault_path: secret/data/myapp/sendgrid
    vault_key: api_key

  # App secrets
  - name: JWT_SECRET
    vault_path: secret/data/myapp/jwt
    vault_key: signing_key
  - name: SESSION_SECRET
    vault_path: secret/data/myapp/session
    vault_key: encryption_key
```

## Security Best Practices

### 1. Never Commit Tokens
```bash
# Set tokens in environment, not in manifest files
export VAULT_TOKEN=hvs.xxx
export VAULT_ROLE_ID=xxx
export VAULT_SECRET_ID=xxx
```

### 2. Use Short-Lived Tokens
```bash
# Create token with TTL
vault token create -policy=cloud-deploy -ttl=1h
```

### 3. Use AppRole in Production
```bash
# Create AppRole
vault auth enable approle
vault write auth/approle/role/cloud-deploy \
  token_policies="cloud-deploy" \
  token_ttl=1h \
  token_max_ttl=4h

# Get credentials
vault read auth/approle/role/cloud-deploy/role-id
vault write -f auth/approle/role/cloud-deploy/secret-id
```

### 4. Use TLS in Production
```yaml
vault:
  address: "https://vault.yourcompany.com"  # HTTPS!
  tls_skip_verify: false  # Verify certificates
```

### 5. Least Privilege Policies
```hcl
# Only allow reading specific paths
path "secret/data/myapp/*" {
  capabilities = ["read"]
}

# Deny everything else (implicit)
```

## Troubleshooting

### "connection refused"
- Check Vault is running: `vault status`
- Check address is correct: `echo $VAULT_ADDR`
- For Docker: Use `http://127.0.0.1:8200` not `http://localhost:8200`

### "permission denied"
- Check token is valid: `vault token lookup`
- Check policy allows reading: `vault policy read cloud-deploy`
- Verify path is correct: `vault kv get secret/myapp/database`

### "secret not found"
- Verify secret exists: `vault kv list secret/`
- Check path format: Use `secret/data/myapp/...` not `secret/myapp/...`
- KV v2 requires `/data/` in the path

### "vault is sealed"
```bash
vault operator unseal <key1>
vault operator unseal <key2>
vault operator unseal <key3>
```

## Migration from Cloud Secret Managers

If you're currently using AWS Secrets Manager or GCP Secret Manager, you can migrate:

### Step 1: Export Secrets
```bash
# AWS Secrets Manager
aws secretsmanager get-secret-value \
  --secret-id prod/myapp/database \
  --query SecretString --output text

# GCP Secret Manager
gcloud secrets versions access latest \
  --secret=database-url
```

### Step 2: Import to Vault
```bash
vault kv put secret/myapp/database \
  url="<value-from-aws-or-gcp>"
```

### Step 3: Update Manifest
Change from cloud-specific to Vault:
```yaml
# Before (AWS-specific)
secrets:
  - name: DATABASE_URL
    from_secret: prod/myapp/database-url

# After (Vault - works everywhere)
vault:
  address: "http://vault.company.com"
  auth:
    method: token
    token: "${VAULT_TOKEN}"

secrets:
  - name: DATABASE_URL
    vault_path: secret/data/myapp/database
    vault_key: url
```

## Next Steps

1. **Set up Vault** - Follow Step 1-6 in "Vault Setup Guide"
2. **Store your secrets** - Use `vault kv put` commands
3. **Update your manifest** - Add `vault:` and `secrets:` sections
4. **Deploy** - Run `cloud-deploy -command deploy`
5. **Verify** - Check your app has access to secrets

## References

- [Vault Documentation](https://www.vaultproject.io/docs)
- [Vault Getting Started](https://learn.hashicorp.com/collections/vault/getting-started)
- [Vault Production Hardening](https://learn.hashicorp.com/tutorials/vault/production-hardening)
- [KV Secrets Engine](https://www.vaultproject.io/docs/secrets/kv)
