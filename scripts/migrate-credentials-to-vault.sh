#!/bin/bash
# Script to migrate cloud provider credentials to Vault
# This securely reads credentials from standard locations and stores them in Vault

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🔐 Cloud Credentials → Vault Migration"
echo "======================================"
echo ""

# Check if Vault is running
if ! docker ps | grep -q vault; then
    echo -e "${RED}❌ Vault is not running. Please run ./vault-setup.sh first.${NC}"
    exit 1
fi

# Set Vault environment variables
export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='myroot'

echo "✅ Vault connection established"
echo ""

# Function to store secret in Vault
store_secret() {
    local path=$1
    shift
    docker exec -e VAULT_ADDR='http://127.0.0.1:8200' -e VAULT_TOKEN='myroot' \
        vault vault kv put "$path" "$@" > /dev/null 2>&1
}

# Migrate AWS Credentials
echo "🔍 Checking for AWS credentials..."
if [ -f ~/.aws/credentials ]; then
    # Parse AWS credentials file
    AWS_ACCESS_KEY=$(grep -A 2 '\[default\]' ~/.aws/credentials | grep aws_access_key_id | cut -d '=' -f 2 | tr -d ' ')
    AWS_SECRET_KEY=$(grep -A 2 '\[default\]' ~/.aws/credentials | grep aws_secret_access_key | cut -d '=' -f 2 | tr -d ' ')

    if [ -n "$AWS_ACCESS_KEY" ] && [ -n "$AWS_SECRET_KEY" ]; then
        store_secret "secret/cloud-deploy/aws/credentials" \
            access_key_id="$AWS_ACCESS_KEY" \
            secret_access_key="$AWS_SECRET_KEY"
        echo -e "${GREEN}✅ AWS credentials stored in Vault${NC}"
        echo "   Path: secret/cloud-deploy/aws/credentials"
        echo "   Keys: access_key_id, secret_access_key"
        AWS_MIGRATED=true
    else
        echo -e "${YELLOW}⚠️  AWS credentials file exists but couldn't parse keys${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  No AWS credentials found at ~/.aws/credentials${NC}"
fi
echo ""

# Migrate GCP Credentials
echo "🔍 Checking for GCP credentials..."
GCP_KEY_FILES=$(find ~ -maxdepth 3 -name "*gcp*.json" -o -name "*service-account*.json" 2>/dev/null | head -5)

if [ -n "$GCP_KEY_FILES" ]; then
    echo "Found potential GCP service account files:"
    echo "$GCP_KEY_FILES"
    echo ""
    echo "Please enter the path to your GCP service account JSON file:"
    read -r GCP_KEY_PATH

    if [ -f "$GCP_KEY_PATH" ]; then
        GCP_KEY_CONTENT=$(cat "$GCP_KEY_PATH")
        PROJECT_ID=$(echo "$GCP_KEY_CONTENT" | grep -o '"project_id": *"[^"]*"' | cut -d'"' -f4)

        # Store the entire JSON as a single field
        docker exec -i -e VAULT_ADDR='http://127.0.0.1:8200' -e VAULT_TOKEN='myroot' \
            vault vault kv put secret/cloud-deploy/gcp/credentials \
            service_account_key="$GCP_KEY_CONTENT" \
            project_id="$PROJECT_ID" > /dev/null 2>&1

        echo -e "${GREEN}✅ GCP credentials stored in Vault${NC}"
        echo "   Path: secret/cloud-deploy/gcp/credentials"
        echo "   Keys: service_account_key, project_id"
        GCP_MIGRATED=true
    else
        echo -e "${YELLOW}⚠️  File not found: $GCP_KEY_PATH${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  No GCP service account files found${NC}"
    echo "   You can manually store GCP credentials later"
fi
echo ""

# Migrate Azure Credentials
echo "🔍 Checking for Azure credentials..."
if command -v az &> /dev/null; then
    echo "Azure CLI is installed. To store Azure credentials, please provide:"
    echo "  1. Subscription ID"
    echo "  2. Client ID (Service Principal)"
    echo "  3. Client Secret"
    echo "  4. Tenant ID"
    echo ""
    echo "Enter Subscription ID (or press Enter to skip):"
    read -r AZURE_SUB_ID

    if [ -n "$AZURE_SUB_ID" ]; then
        echo "Enter Client ID:"
        read -r AZURE_CLIENT_ID
        echo "Enter Client Secret:"
        read -rs AZURE_CLIENT_SECRET
        echo ""
        echo "Enter Tenant ID:"
        read -r AZURE_TENANT_ID

        store_secret "secret/cloud-deploy/azure/credentials" \
            subscription_id="$AZURE_SUB_ID" \
            client_id="$AZURE_CLIENT_ID" \
            client_secret="$AZURE_CLIENT_SECRET" \
            tenant_id="$AZURE_TENANT_ID"

        echo -e "${GREEN}✅ Azure credentials stored in Vault${NC}"
        echo "   Path: secret/cloud-deploy/azure/credentials"
        echo "   Keys: subscription_id, client_id, client_secret, tenant_id"
        AZURE_MIGRATED=true
    fi
else
    echo -e "${YELLOW}⚠️  Azure CLI not installed${NC}"
    echo "   You can manually store Azure credentials later"
fi
echo ""

# Summary
echo "======================================"
echo "📊 Migration Summary"
echo "======================================"
[ "$AWS_MIGRATED" = true ] && echo -e "${GREEN}✅ AWS${NC}" || echo -e "${YELLOW}⚠️  AWS (not migrated)${NC}"
[ "$GCP_MIGRATED" = true ] && echo -e "${GREEN}✅ GCP${NC}" || echo -e "${YELLOW}⚠️  GCP (not migrated)${NC}"
[ "$AZURE_MIGRATED" = true ] && echo -e "${GREEN}✅ Azure${NC}" || echo -e "${YELLOW}⚠️  Azure (not migrated)${NC}"
echo ""

echo "📝 Vault Secret Paths:"
echo "   AWS:   secret/cloud-deploy/aws/credentials"
echo "   GCP:   secret/cloud-deploy/gcp/credentials"
echo "   Azure: secret/cloud-deploy/azure/credentials"
echo ""

echo "🔒 Security Note:"
echo "   Your credentials are now in Vault. You can optionally remove them from:"
echo "   - ~/.aws/credentials (AWS)"
echo "   - Service account JSON files (GCP)"
echo "   But keep backups just in case!"
echo ""

echo "✅ Migration complete! Next step: Update cloud-deploy code to use Vault credentials."
