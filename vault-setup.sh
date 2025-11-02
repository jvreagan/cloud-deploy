#!/bin/bash
# HashiCorp Vault Local Setup Script
# This sets up Vault in dev mode for testing cloud-deploy

set -e

echo "🔐 Setting up HashiCorp Vault locally..."

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Error: Docker is not running. Please start Docker Desktop and try again."
    exit 1
fi

# Stop and remove existing vault container if it exists
if docker ps -a | grep -q vault; then
    echo "📦 Removing existing vault container..."
    docker stop vault 2>/dev/null || true
    docker rm vault 2>/dev/null || true
fi

# Start Vault in dev mode
echo "🚀 Starting Vault in dev mode..."
docker run -d \
  --name vault \
  -p 8200:8200 \
  -e VAULT_DEV_ROOT_TOKEN_ID=myroot \
  -e VAULT_DEV_LISTEN_ADDRESS=0.0.0.0:8200 \
  --cap-add=IPC_LOCK \
  hashicorp/vault:latest

# Wait for Vault to be ready
echo "⏳ Waiting for Vault to start..."
sleep 3

# Check if Vault is running
if docker ps | grep -q vault; then
    echo "✅ Vault is running!"
else
    echo "❌ Failed to start Vault"
    exit 1
fi

# Export environment variables
export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='myroot'

echo ""
echo "✅ Vault Setup Complete!"
echo ""
echo "📋 Environment Variables (add to your ~/.zshrc or ~/.bashrc):"
echo "   export VAULT_ADDR='http://127.0.0.1:8200'"
echo "   export VAULT_TOKEN='myroot'"
echo ""
echo "🔑 Root Token: myroot"
echo "🌐 Vault UI: http://127.0.0.1:8200/ui"
echo ""
echo "📝 Next Steps:"
echo "   1. Set environment variables in your current shell:"
echo "      export VAULT_ADDR='http://127.0.0.1:8200'"
echo "      export VAULT_TOKEN='myroot'"
echo ""
echo "   2. Enable KV secrets engine:"
echo "      docker exec vault vault secrets enable -path=secret kv-v2"
echo ""
echo "   3. Store a test secret:"
echo "      docker exec vault vault kv put secret/myapp/database url=\"postgresql://localhost:5432/db\""
echo ""
echo "   4. Verify the secret:"
echo "      docker exec vault vault kv get secret/myapp/database"
echo ""
echo "⚠️  WARNING: This is DEV MODE - DO NOT use in production!"
echo "   Dev mode stores data in memory and is lost when container stops."
echo ""
