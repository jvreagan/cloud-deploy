# HashiCorp Vault Quick Start Guide

This guide helps you set up Vault locally and use it with cloud-deploy.

## Step 1: Start Vault

Run the setup script:

```bash
./vault-setup.sh
```

This will:
- Start Vault in Docker (dev mode)
- Run on http://127.0.0.1:8200
- Use root token: `myroot`

## Step 2: Set Environment Variables

```bash
export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='myroot'
```

**Make it permanent** (add to ~/.zshrc or ~/.bashrc):
```bash
echo 'export VAULT_ADDR="http://127.0.0.1:8200"' >> ~/.zshrc
echo 'export VAULT_TOKEN="myroot"' >> ~/.zshrc
source ~/.zshrc
```

## Step 3: Enable KV Secrets Engine

```bash
docker exec vault vault secrets enable -path=secret kv-v2
```

## Step 4: Store Some Secrets

```bash
# Database credentials
docker exec vault vault kv put secret/myapp/database \
  url="postgresql://user:pass@localhost:5432/mydb"

# Stripe API key
docker exec vault vault kv put secret/myapp/stripe \
  api_key="sk_test_abc123xyz"

# JWT signing key
docker exec vault vault kv put secret/myapp/jwt \
  signing_key="super-secret-key-change-me"
```

## Step 5: Verify Secrets

```bash
# List secrets
docker exec vault vault kv list secret/myapp

# Read a specific secret
docker exec vault vault kv get secret/myapp/database
```

## Step 6: Use with cloud-deploy

Now you can use the example manifest:

```bash
# Deploy with Vault secrets
cloud-deploy -command deploy -manifest examples/aws-with-vault.yaml
```

The deployment will:
1. Authenticate to Vault using token
2. Fetch DATABASE_URL, STRIPE_API_KEY, JWT_SECRET from Vault
3. Inject them as environment variables in your deployed application
4. Your app can access them via `os.Getenv("DATABASE_URL")`, etc.

## Access Vault UI

Open in your browser: http://127.0.0.1:8200/ui

**Login with token:** `myroot`

## Useful Commands

```bash
# Check Vault status
docker exec vault vault status

# View all secrets in a path
docker exec vault vault kv get -format=json secret/myapp/database

# Update a secret
docker exec vault vault kv put secret/myapp/database \
  url="postgresql://newuser:newpass@localhost:5432/mydb"

# Delete a secret
docker exec vault vault kv delete secret/myapp/database

# Stop Vault
docker stop vault

# Start Vault again
docker start vault

# Remove Vault completely
docker stop vault && docker rm vault
```

## Important Notes

### ⚠️ Dev Mode Limitations

This setup uses **dev mode** which is great for testing but:
- ✅ Easy to set up
- ✅ No configuration needed
- ❌ **Data is stored in memory** (lost when container stops)
- ❌ Not secure for production
- ❌ Uses a simple root token

### 🔐 For Production

For production deployments, you should:
1. Use Vault in server mode with persistent storage
2. Use AppRole authentication instead of token
3. Set up TLS/HTTPS
4. Configure proper policies
5. Enable audit logging

See the full guide: [docs/VAULT_INTEGRATION.md](docs/VAULT_INTEGRATION.md)

## Multi-Cloud Example

The beauty of Vault is using the same secrets across clouds:

```yaml
# Same Vault config works for AWS, GCP, Azure!
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
  name: aws  # Change to 'gcp' or 'azure' - same secrets!
  # ... provider config
```

## Troubleshooting

**Problem:** "connection refused" error

**Solution:** Make sure Vault is running:
```bash
docker ps | grep vault
```

If not running, start it:
```bash
docker start vault
# or run ./vault-setup.sh again
```

---

**Problem:** "permission denied" error

**Solution:** Check your token is set:
```bash
echo $VAULT_TOKEN
```

Should show: `myroot`

---

**Problem:** "path not found" error

**Solution:** Make sure KV engine is enabled:
```bash
docker exec vault vault secrets enable -path=secret kv-v2
```

## Next Steps

1. Read the [full Vault integration guide](docs/VAULT_INTEGRATION.md)
2. Try deploying to different clouds with the same secrets
3. Explore AppRole authentication for production use
4. Set up Vault policies for fine-grained access control
