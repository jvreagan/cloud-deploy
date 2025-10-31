// Package vault provides integration with HashiCorp Vault for secret management.
// It supports multiple authentication methods (token, AppRole, AWS IAM, GCP IAM)
// and fetches secrets from Vault's KV v2 secrets engine.
package vault

import (
	"context"
	"fmt"

	vault "github.com/hashicorp/vault/api"
)

// Config holds Vault configuration including address and authentication details.
type Config struct {
	// Address is the Vault server address (e.g., "http://127.0.0.1:8200")
	Address string

	// Auth holds authentication configuration
	Auth AuthConfig

	// TLSSkipVerify skips TLS certificate verification (not recommended for production)
	TLSSkipVerify bool
}

// AuthConfig specifies the authentication method and credentials.
type AuthConfig struct {
	// Method is the auth method: "token", "approle", "aws-iam", "gcp-iam"
	Method string

	// Token for token authentication
	Token string

	// RoleID for AppRole authentication
	RoleID string

	// SecretID for AppRole authentication
	SecretID string

	// Role for AWS IAM or GCP IAM authentication
	Role string
}

// Client wraps the Vault API client and provides secret retrieval methods.
type Client struct {
	client *vault.Client
	config *Config
}

// NewClient creates a new Vault client with the given configuration.
// It initializes the client but does not authenticate yet.
//
// Example:
//
//	config := &vault.Config{
//	    Address: "http://127.0.0.1:8200",
//	    Auth: vault.AuthConfig{
//	        Method: "token",
//	        Token: "hvs.xxx",
//	    },
//	}
//	client, err := vault.NewClient(config)
func NewClient(config *Config) (*Client, error) {
	if config.Address == "" {
		return nil, fmt.Errorf("vault address is required")
	}

	// Create Vault client config
	vaultConfig := vault.DefaultConfig()
	vaultConfig.Address = config.Address

	if config.TLSSkipVerify {
		tlsConfig := &vault.TLSConfig{
			Insecure: true,
		}
		if err := vaultConfig.ConfigureTLS(tlsConfig); err != nil {
			return nil, fmt.Errorf("failed to configure TLS: %w", err)
		}
	}

	// Create Vault client
	client, err := vault.NewClient(vaultConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	return &Client{
		client: client,
		config: config,
	}, nil
}

// Authenticate authenticates to Vault using the configured auth method.
// This must be called before fetching secrets.
//
// Supported authentication methods:
//   - token: Uses a Vault token directly
//   - approle: Uses AppRole role_id and secret_id
//   - aws-iam: Uses AWS IAM credentials (future implementation)
//   - gcp-iam: Uses GCP IAM credentials (future implementation)
func (c *Client) Authenticate(ctx context.Context) error {
	switch c.config.Auth.Method {
	case "token":
		return c.authenticateWithToken()

	case "approle":
		return c.authenticateWithAppRole(ctx)

	case "aws-iam":
		return fmt.Errorf("aws-iam authentication not yet implemented")

	case "gcp-iam":
		return fmt.Errorf("gcp-iam authentication not yet implemented")

	default:
		return fmt.Errorf("unsupported auth method: %s", c.config.Auth.Method)
	}
}

// authenticateWithToken sets the token directly on the client.
func (c *Client) authenticateWithToken() error {
	if c.config.Auth.Token == "" {
		return fmt.Errorf("vault token is required for token authentication")
	}

	c.client.SetToken(c.config.Auth.Token)
	return nil
}

// authenticateWithAppRole authenticates using AppRole role_id and secret_id.
func (c *Client) authenticateWithAppRole(ctx context.Context) error {
	if c.config.Auth.RoleID == "" {
		return fmt.Errorf("role_id is required for approle authentication")
	}
	if c.config.Auth.SecretID == "" {
		return fmt.Errorf("secret_id is required for approle authentication")
	}

	// Prepare login data
	data := map[string]interface{}{
		"role_id":   c.config.Auth.RoleID,
		"secret_id": c.config.Auth.SecretID,
	}

	// Login via AppRole
	resp, err := c.client.Logical().WriteWithContext(ctx, "auth/approle/login", data)
	if err != nil {
		return fmt.Errorf("approle login failed: %w", err)
	}

	if resp == nil || resp.Auth == nil {
		return fmt.Errorf("approle login returned no auth token")
	}

	// Set the token from the response
	c.client.SetToken(resp.Auth.ClientToken)
	return nil
}

// GetSecret fetches a secret from Vault's KV v2 secrets engine.
//
// Parameters:
//   - path: The full path to the secret (e.g., "secret/data/myapp/database")
//   - key: The key within the secret data (e.g., "url")
//
// Returns the secret value as a string.
//
// Example:
//
//	value, err := client.GetSecret(ctx, "secret/data/myapp/database", "url")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("Database URL:", value)
//
// Note: For KV v2, the path must include "/data/" after the mount point.
// For example: "secret/data/myapp/database" not "secret/myapp/database"
func (c *Client) GetSecret(ctx context.Context, path, key string) (string, error) {
	// Read secret from Vault
	secret, err := c.client.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return "", fmt.Errorf("failed to read secret at %s: %w", path, err)
	}

	if secret == nil {
		return "", fmt.Errorf("secret not found at path: %s", path)
	}

	// For KV v2, secrets are nested under "data"
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected secret format at path: %s", path)
	}

	// Get the specific key
	value, ok := data[key]
	if !ok {
		return "", fmt.Errorf("key %s not found in secret at path: %s", key, path)
	}

	// Convert to string
	valueStr, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("value for key %s is not a string at path: %s", key, path)
	}

	return valueStr, nil
}

// GetSecrets fetches multiple secrets at once.
// This is a convenience method that calls GetSecret multiple times.
//
// Parameters:
//   - secrets: A map of environment variable names to SecretRef
//
// Returns a map of environment variable names to secret values.
//
// Example:
//
//	secrets := map[string]SecretRef{
//	    "DATABASE_URL": {Path: "secret/data/myapp/database", Key: "url"},
//	    "API_KEY": {Path: "secret/data/myapp/api", Key: "key"},
//	}
//	values, err := client.GetSecrets(ctx, secrets)
func (c *Client) GetSecrets(ctx context.Context, secrets map[string]SecretRef) (map[string]string, error) {
	values := make(map[string]string)

	for name, ref := range secrets {
		value, err := c.GetSecret(ctx, ref.Path, ref.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch secret %s: %w", name, err)
		}
		values[name] = value
	}

	return values, nil
}

// SecretRef references a specific secret key in Vault.
type SecretRef struct {
	// Path is the full Vault path (e.g., "secret/data/myapp/database")
	Path string

	// Key is the key within the secret (e.g., "url")
	Key string
}

// Close closes the Vault client.
// Currently a no-op but provided for future cleanup needs.
func (c *Client) Close() error {
	return nil
}
