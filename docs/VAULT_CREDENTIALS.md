# Vault Credential Storage

**NEW FEATURE:** Store your AWS, GCP, and Azure credentials in HashiCorp Vault for true multi-cloud credential management.

## Overview

Instead of managing cloud credentials in multiple locations (`~/.aws/credentials`, service account JSON files, Azure CLI config), you can now store ALL your cloud provider credentials in Vault and have cloud-deploy fetch them automatically.

## Benefits

✅ **Single Source of Truth** - All cloud credentials in one place
✅ **Multi-Cloud Simplicity** - Same credential management for AWS, GCP, Azure
✅ **Credential Rotation** - Update credentials in Vault, all deployments use new creds
✅ **Access Control** - Use Vault policies to control who can deploy to which cloud
✅ **Audit Trail** - Vault logs all credential access
✅ **No Files on Disk** - Credentials never stored in local files

## Quick Start

### 1. Store Credentials in Vault

Use the migration script to move your existing credentials:

```bash
./scripts/migrate-credentials-to-vault.sh
```

This will:
- ✅ Find your AWS credentials in `~/.aws/credentials`
- ✅ Find your GCP service account keys
- ✅ Prompt for Azure credentials
- ✅ Store everything in Vault at standard paths

**Vault Paths Created:**
```
secret/cloud-deploy/aws/credentials
secret/cloud-deploy/gcp/credentials
secret/cloud-deploy/azure/credentials
```

### 2. Update Your Manifest

Tell cloud-deploy to use Vault for credentials:

```yaml
version: "1.0"
image: "my-app:latest"

# Vault configuration
vault:
  address: "http://127.0.0.1:8200"
  auth:
    method: token
    token: "${VAULT_TOKEN}"

# Provider configuration
provider:
  name: aws
  region: us-east-2

  # NEW: Tell cloud-deploy to load credentials from Vault!
  credentials:
    source: vault  # Options: vault, environment, manifest, cli

# Rest of manifest...
application:
  name: my-app
# ...
```

### 3. Deploy!

```bash
export VAULT_TOKEN=myroot
cloud-deploy -command deploy -manifest examples/aws-vault-credentials.yaml
```

Output:
```
📦 Loading aws credentials from Vault...
Loading AWS credentials from Vault...
✅ Successfully loaded AWS credentials from Vault
Starting deployment...
```

## Credential Source Options

You can choose where cloud-deploy loads credentials from:

| Source | Description | Use Case |
|--------|-------------|----------|
| **`vault`** | Fetch from HashiCorp Vault | ✅ Production, multi-cloud, teams |
| **`cli`** | Use cloud CLI credentials (default) | Development, single developer |
| **`environment`** | Load from env vars | CI/CD pipelines |
| **`manifest`** | Hardcoded in manifest | ❌ Not recommended (insecure) |

**Example:**

```yaml
# Different credential sources for different scenarios

# Production: Use Vault
provider:
  credentials:
    source: vault

# Development: Use AWS CLI
provider:
  credentials:
    source: cli  # or omit, this is the default

# CI/CD: Use environment variables
provider:
  credentials:
    source: environment
```

## How It Works

### Architecture

```
┌──────────────────────┐
│  deploy-manifest     │
│  credentials:        │
│    source: vault     │
└──────────┬───────────┘
           │
           ↓
┌──────────────────────┐      ┌────────────────┐
│  cloud-deploy CLI    │─────→│ Vault Server   │
│  1. Read manifest    │      │ (port 8200)    │
│  2. Check source     │      └────────────────┘
│  3. Fetch from Vault │
└──────────┬───────────┘
           │ AWS Credentials
           ↓
┌──────────────────────┐
│  AWS API             │
│  Elastic Beanstalk   │
│  (authenticated)     │
└──────────────────────┘
```

### Data Flow

1. **Manifest Parsing** - cloud-deploy reads `provider.credentials.source: vault`
2. **Vault Authentication** - Connects to Vault using token/AppRole
3. **Credential Fetch** - Gets `secret/cloud-deploy/aws/credentials`
4. **AWS Authentication** - Uses fetched credentials to auth to AWS
5. **Deployment** - Proceeds with normal deployment

## Vault Secret Structure

### AWS Credentials

**Path:** `secret/cloud-deploy/aws/credentials`

```json
{
  "access_key_id": "AKIAIOSFODNN7EXAMPLE",
  "secret_access_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
}
```

**Store manually:**
```bash
docker exec -e VAULT_ADDR='http://127.0.0.1:8200' -e VAULT_TOKEN='myroot' \
  vault vault kv put secret/cloud-deploy/aws/credentials \
  access_key_id="AKIAIOSFODNN7EXAMPLE" \
  secret_access_key="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
```

### GCP Credentials

**Path:** `secret/cloud-deploy/gcp/credentials`

```json
{
  "project_id": "my-gcp-project",
  "service_account_key": "{\"type\":\"service_account\",\"project_id\":\"my-project\",...}"
}
```

**Store manually:**
```bash
GCP_KEY=$(cat ~/my-service-account.json)
docker exec -i -e VAULT_ADDR='http://127.0.0.1:8200' -e VAULT_TOKEN='myroot' \
  vault vault kv put secret/cloud-deploy/gcp/credentials \
  project_id="my-gcp-project" \
  service_account_key="$GCP_KEY"
```

### Azure Credentials

**Path:** `secret/cloud-deploy/azure/credentials`

```json
{
  "subscription_id": "12345678-1234-1234-1234-123456789012",
  "client_id": "12345678-1234-1234-1234-123456789012",
  "client_secret": "my-client-secret",
  "tenant_id": "12345678-1234-1234-1234-123456789012"
}
```

## Multi-Cloud Example

The power of Vault credentials: **deploy to any cloud with the same manifest structure!**

```yaml
# deploy-to-aws.yaml
vault:
  address: "http://127.0.0.1:8200"
  auth:
    method: token
    token: "${VAULT_TOKEN}"

provider:
  name: aws  # ← Change this to gcp or azure
  region: us-east-2
  credentials:
    source: vault  # ← Same credential source!

application:
  name: my-app
# ...
```

**Deploy to AWS:**
```bash
cloud-deploy -command deploy -manifest deploy-to-aws.yaml
```

**Deploy to GCP:** Just change `name: gcp`
```bash
# Edit manifest: provider.name: gcp
cloud-deploy -command deploy -manifest deploy-to-gcp.yaml
```

All credentials managed in Vault! 🎉

## Security Best Practices

### Development (Local Vault)

✅ Use token authentication
✅ Store Vault token in environment variable
✅ Use dev mode for testing

### Production

✅ **Use AppRole authentication** (not tokens)
```yaml
vault:
  address: "https://vault.company.com"
  auth:
    method: approle
    role_id: "${VAULT_ROLE_ID}"
    secret_id: "${VAULT_SECRET_ID}"
```

✅ **Use TLS/HTTPS** for Vault
✅ **Enable Vault policies** to restrict access
✅ **Enable audit logging**
✅ **Rotate credentials** regularly
✅ **Use persistent Vault storage** (not dev mode)

### Credential Rotation

When you rotate cloud credentials:

1. **Update in Vault:**
```bash
vault kv put secret/cloud-deploy/aws/credentials \
  access_key_id="NEW_KEY" \
  secret_access_key="NEW_SECRET"
```

2. **Deploy again:**
```bash
cloud-deploy -command deploy -manifest app.yaml
```

3. **Done!** All deployments now use new credentials

No need to update config files, environment variables, or CI/CD secrets!

## Troubleshooting

### Error: "vault configuration required when credentials.source is 'vault'"

**Solution:** Add `vault:` section to manifest

```yaml
vault:
  address: "http://127.0.0.1:8200"
  auth:
    method: token
    token: "${VAULT_TOKEN}"
```

### Error: "failed to authenticate to vault: permission denied"

**Solution:** Check your Vault token

```bash
echo $VAULT_TOKEN  # Should show: myroot (or your token)
export VAULT_TOKEN=myroot
```

### Error: "failed to fetch AWS access_key_id from vault: key not found"

**Solution:** Credentials not in Vault yet

```bash
# Run migration script
./scripts/migrate-credentials-to-vault.sh

# Or store manually
docker exec -e VAULT_ADDR='http://127.0.0.1:8200' -e VAULT_TOKEN='myroot' \
  vault vault kv put secret/cloud-deploy/aws/credentials \
  access_key_id="YOUR_KEY" \
  secret_access_key="YOUR_SECRET"
```

### Error: "connection refused" to Vault

**Solution:** Vault not running

```bash
# Check if Vault is running
docker ps | grep vault

# Start Vault if not running
./vault-setup.sh
```

## Comparison: Before vs After

### Before (Without Vault)

```
Managing credentials in multiple locations:

AWS:
  ~/.aws/credentials

GCP:
  ~/gcp-service-account-key.json

Azure:
  ~/.azure/config

Problems:
  ❌ Credentials scattered across filesystem
  ❌ Different format for each cloud
  ❌ Hard to rotate
  ❌ No centralized access control
  ❌ No audit trail
```

### After (With Vault)

```
All credentials in Vault:

Vault:
  secret/cloud-deploy/aws/credentials
  secret/cloud-deploy/gcp/credentials
  secret/cloud-deploy/azure/credentials

Benefits:
  ✅ Single source of truth
  ✅ Consistent format
  ✅ Easy rotation
  ✅ Centralized access control
  ✅ Full audit trail
  ✅ Works across all clouds
```

## Related Documentation

- [VAULT_QUICKSTART.md](VAULT_QUICKSTART.md) - Set up Vault locally
- [docs/VAULT_INTEGRATION.md](docs/VAULT_INTEGRATION.md) - Full Vault integration guide
- [examples/aws-vault-credentials.yaml](examples/aws-vault-credentials.yaml) - Example manifest

---

**Summary:** With Vault credential storage, cloud-deploy becomes a truly unified multi-cloud deployment tool with enterprise-grade credential management. 🚀
