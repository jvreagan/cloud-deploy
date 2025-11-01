package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// Manager handles credential retrieval from various sources
type Manager struct {
	Source  string            // "environment", "secrets-manager", "vault", "encrypted-file"
	Secrets map[string]string // Maps credential keys to their source identifiers
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
		return nil, fmt.Errorf("HashiCorp Vault integration not yet implemented")
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
