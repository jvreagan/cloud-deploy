package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/jvreagan/cloud-deploy/pkg/vault"
)

// Manager handles credential retrieval from various sources
type Manager struct {
	Source      string            // "environment", "secrets-manager", "vault", "encrypted-file"
	Secrets     map[string]string // Maps credential keys to their source identifiers
	VaultConfig *vault.Config     // Vault configuration if Source is "vault"
}

// ProviderCredentials represents the credentials needed for a cloud provider
type ProviderCredentials struct {
	AWS struct {
		AccessKeyID     string `json:"access_key_id"`
		SecretAccessKey string `json:"secret_access_key"`
		SessionToken    string `json:"session_token,omitempty"`
	} `json:"aws,omitempty"`
	GCP struct {
		ProjectID           string `json:"project_id"`
		ServiceAccountKey   string `json:"service_account_key"` // JSON key content
		ServiceAccountEmail string `json:"service_account_email,omitempty"`
	} `json:"gcp,omitempty"`
	Azure struct {
		TenantID       string `json:"tenant_id"`
		ClientID       string `json:"client_id"`
		ClientSecret   string `json:"client_secret"`
		SubscriptionID string `json:"subscription_id"`
	} `json:"azure,omitempty"`
	Cloudflare struct {
		APIToken  string `json:"api_token"`
		Email     string `json:"email,omitempty"`
		AccountID string `json:"account_id"`
	} `json:"cloudflare,omitempty"`
}

// GetCredentials retrieves credentials based on the configured source
func (m *Manager) GetCredentials(ctx context.Context, provider string) (*ProviderCredentials, error) {
	switch m.Source {
	case "environment":
		return m.getFromEnvironment(provider)
	case "secrets-manager":
		return m.getFromSecretsManager(ctx, provider)
	case "vault":
		return m.getFromVault(ctx, provider)
	case "encrypted-file":
		return nil, fmt.Errorf("encrypted file integration not yet implemented")
	default:
		return nil, fmt.Errorf("unknown credentials source: %s", m.Source)
	}
}

// getFromEnvironment retrieves credentials from environment variables
func (m *Manager) getFromEnvironment(provider string) (*ProviderCredentials, error) {
	creds := &ProviderCredentials{}

	switch provider {
	case "aws":
		creds.AWS.AccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
		creds.AWS.SecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		creds.AWS.SessionToken = os.Getenv("AWS_SESSION_TOKEN")

		if creds.AWS.AccessKeyID == "" || creds.AWS.SecretAccessKey == "" {
			return nil, fmt.Errorf("AWS credentials not found in environment")
		}

	case "gcp":
		creds.GCP.ProjectID = os.Getenv("GCP_PROJECT_ID")
		creds.GCP.ServiceAccountKey = os.Getenv("GCP_SERVICE_ACCOUNT_KEY")

		if creds.GCP.ProjectID == "" || creds.GCP.ServiceAccountKey == "" {
			return nil, fmt.Errorf("GCP credentials not found in environment")
		}

	case "azure":
		creds.Azure.TenantID = os.Getenv("AZURE_TENANT_ID")
		creds.Azure.ClientID = os.Getenv("AZURE_CLIENT_ID")
		creds.Azure.ClientSecret = os.Getenv("AZURE_CLIENT_SECRET")
		creds.Azure.SubscriptionID = os.Getenv("AZURE_SUBSCRIPTION_ID")

		if creds.Azure.TenantID == "" || creds.Azure.ClientID == "" || creds.Azure.ClientSecret == "" {
			return nil, fmt.Errorf("Azure credentials not found in environment")
		}

	case "cloudflare":
		creds.Cloudflare.APIToken = os.Getenv("CLOUDFLARE_API_TOKEN")
		creds.Cloudflare.AccountID = os.Getenv("CLOUDFLARE_ACCOUNT_ID")

		if creds.Cloudflare.APIToken == "" {
			return nil, fmt.Errorf("Cloudflare credentials not found in environment")
		}

	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}

	return creds, nil
}

// getFromSecretsManager retrieves credentials from AWS Secrets Manager
func (m *Manager) getFromSecretsManager(ctx context.Context, provider string) (*ProviderCredentials, error) {
	// Get the secret ARN or name from the secrets map
	secretID, ok := m.Secrets[provider]
	if !ok {
		return nil, fmt.Errorf("no secret configured for provider: %s", provider)
	}

	// Create AWS Secrets Manager client
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := secretsmanager.NewFromConfig(cfg)

	// Retrieve the secret value
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretID),
	}

	result, err := client.GetSecretValue(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve secret %s: %w", secretID, err)
	}

	// Parse the secret JSON
	creds := &ProviderCredentials{}
	if err := json.Unmarshal([]byte(*result.SecretString), creds); err != nil {
		return nil, fmt.Errorf("failed to parse secret JSON: %w", err)
	}

	return creds, nil
}

// getFromVault retrieves credentials from HashiCorp Vault
func (m *Manager) getFromVault(ctx context.Context, provider string) (*ProviderCredentials, error) {
	if m.VaultConfig == nil {
		return nil, fmt.Errorf("vault configuration is required when using vault credentials")
	}

	// Create Vault client
	client, err := vault.NewClient(m.VaultConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}
	defer client.Close()

	// Authenticate to Vault
	if err := client.Authenticate(ctx); err != nil {
		return nil, fmt.Errorf("failed to authenticate to vault: %w", err)
	}

	creds := &ProviderCredentials{}

	switch provider {
	case "aws":
		// Fetch AWS credentials from Vault
		// Expected path: secret/data/cloud-deploy/aws/credentials
		accessKeyID, err := client.GetSecret(ctx, "secret/data/cloud-deploy/aws/credentials", "access_key_id")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch AWS access_key_id from vault: %w", err)
		}

		secretAccessKey, err := client.GetSecret(ctx, "secret/data/cloud-deploy/aws/credentials", "secret_access_key")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch AWS secret_access_key from vault: %w", err)
		}

		creds.AWS.AccessKeyID = accessKeyID
		creds.AWS.SecretAccessKey = secretAccessKey

	case "gcp":
		// Fetch GCP credentials from Vault
		// Expected path: secret/data/cloud-deploy/gcp/credentials
		projectID, err := client.GetSecret(ctx, "secret/data/cloud-deploy/gcp/credentials", "project_id")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch GCP project_id from vault: %w", err)
		}

		serviceAccountKey, err := client.GetSecret(ctx, "secret/data/cloud-deploy/gcp/credentials", "service_account_key")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch GCP service_account_key from vault: %w", err)
		}

		creds.GCP.ProjectID = projectID
		creds.GCP.ServiceAccountKey = serviceAccountKey

	case "azure":
		// Fetch Azure credentials from Vault
		// Expected path: secret/data/cloud-deploy/azure/credentials
		subscriptionID, err := client.GetSecret(ctx, "secret/data/cloud-deploy/azure/credentials", "subscription_id")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch Azure subscription_id from vault: %w", err)
		}

		clientID, err := client.GetSecret(ctx, "secret/data/cloud-deploy/azure/credentials", "client_id")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch Azure client_id from vault: %w", err)
		}

		clientSecret, err := client.GetSecret(ctx, "secret/data/cloud-deploy/azure/credentials", "client_secret")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch Azure client_secret from vault: %w", err)
		}

		tenantID, err := client.GetSecret(ctx, "secret/data/cloud-deploy/azure/credentials", "tenant_id")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch Azure tenant_id from vault: %w", err)
		}

		creds.Azure.SubscriptionID = subscriptionID
		creds.Azure.ClientID = clientID
		creds.Azure.ClientSecret = clientSecret
		creds.Azure.TenantID = tenantID

	case "cloudflare":
		// Fetch Cloudflare credentials from Vault
		// Expected path: secret/data/cloud-deploy/cloudflare/credentials
		apiToken, err := client.GetSecret(ctx, "secret/data/cloud-deploy/cloudflare/credentials", "api_token")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch Cloudflare api_token from vault: %w", err)
		}

		accountID, err := client.GetSecret(ctx, "secret/data/cloud-deploy/cloudflare/credentials", "account_id")
		if err != nil {
			// Account ID might be optional
			accountID = ""
		}

		creds.Cloudflare.APIToken = apiToken
		creds.Cloudflare.AccountID = accountID

	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}

	return creds, nil
}

// ValidateCredentials checks if the credentials are valid for the given provider
func ValidateCredentials(creds *ProviderCredentials, provider string) error {
	switch provider {
	case "aws":
		if creds.AWS.AccessKeyID == "" || creds.AWS.SecretAccessKey == "" {
			return fmt.Errorf("AWS credentials are incomplete")
		}
	case "gcp":
		if creds.GCP.ProjectID == "" || creds.GCP.ServiceAccountKey == "" {
			return fmt.Errorf("GCP credentials are incomplete")
		}
	case "azure":
		if creds.Azure.TenantID == "" || creds.Azure.ClientID == "" || creds.Azure.ClientSecret == "" {
			return fmt.Errorf("Azure credentials are incomplete")
		}
	case "cloudflare":
		if creds.Cloudflare.APIToken == "" {
			return fmt.Errorf("Cloudflare credentials are incomplete")
		}
	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}
	return nil
}
