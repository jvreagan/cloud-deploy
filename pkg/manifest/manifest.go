// Package manifest provides types and functions for parsing and validating
// cloud-deploy manifest files. Manifests are YAML files that define the
// complete configuration for deploying an application to a cloud provider.
package manifest

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/jvreagan/cloud-deploy/pkg/vault"
	"gopkg.in/yaml.v3"
)

// Manifest represents the complete deployment configuration.
// It defines all aspects of deploying an application to a cloud provider,
// including provider settings, application details, environment configuration,
// and deployment parameters.
//
// Example:
//
//	manifest := &Manifest{
//	  Provider: ProviderConfig{Name: "aws", Region: "us-east-2"},
//	  Application: ApplicationConfig{Name: "my-app"},
//	  Environment: EnvironmentConfig{Name: "my-app-env"},
//	}
type Manifest struct {
	// Version of the manifest schema (currently "1.0")
	Version string `yaml:"version"`

	// Image is the Docker image to deploy (e.g., "myapp:latest" or "docker.io/myapp:v1.0")
	// This should be a pre-built image available in your local Docker daemon or a registry
	Image string `yaml:"image"`

	// Provider configuration (cloud provider, region, credentials)
	Provider ProviderConfig `yaml:"provider"`

	// Application configuration (name, description)
	Application ApplicationConfig `yaml:"application"`

	// Environment configuration (name, subdomain)
	Environment EnvironmentConfig `yaml:"environment"`

	// Deployment configuration (platform, source code location)
	Deployment DeploymentConfig `yaml:"deployment"`

	// Instance configuration (type, scaling)
	Instance InstanceConfig `yaml:"instance"`

	// Cloud Run configuration (GCP-specific) - optional
	CloudRun *CloudRunConfig `yaml:"cloud_run,omitempty"`

	// Azure configuration (Azure-specific) - optional
	Azure *AzureConfig `yaml:"azure,omitempty"`

	// Health check configuration
	HealthCheck HealthCheckConfig `yaml:"health_check"`

	// Monitoring configuration (CloudWatch, metrics) - optional
	Monitoring MonitoringConfig `yaml:"monitoring,omitempty"`

	// IAM configuration (roles, profiles) - optional
	IAM IAMConfig `yaml:"iam,omitempty"`

	// Environment variables to set in the deployment - optional
	EnvironmentVariables map[string]string `yaml:"environment_variables,omitempty"`

	// Vault configuration for secret management - optional
	Vault *VaultConfig `yaml:"vault,omitempty"`

	// Secrets to fetch from Vault and inject as environment variables - optional
	Secrets []SecretConfig `yaml:"secrets,omitempty"`

	// Tags to apply to cloud resources - optional
	Tags map[string]string `yaml:"tags,omitempty"`
}

// ProviderConfig specifies which cloud provider to use and how to authenticate.
type ProviderConfig struct {
	// Name of the cloud provider (aws, gcp, azure, oci)
	Name string `yaml:"name"`

	// Region to deploy to (e.g., us-east-2, us-west-1)
	Region string `yaml:"region"`

	// Credentials for authentication - optional, can use CLI credentials instead
	Credentials *CredentialsConfig `yaml:"credentials,omitempty"`

	// GCP-specific: Project ID (required for GCP provider)
	// The provider will create this project if it doesn't exist
	ProjectID string `yaml:"project_id,omitempty"`

	// GCP-specific: Billing account ID (required for GCP project creation)
	// Format: "XXXXXX-XXXXXX-XXXXXX"
	// Find yours at: https://console.cloud.google.com/billing
	BillingAccountID string `yaml:"billing_account_id,omitempty"`

	// GCP-specific: Make Cloud Run service publicly accessible (default: true)
	PublicAccess *bool `yaml:"public_access,omitempty"`

	// GCP-specific: Organization ID (optional, for creating projects under an organization)
	OrganizationID string `yaml:"organization_id,omitempty"`

	// Azure-specific: Subscription ID (required for Azure provider)
	SubscriptionID string `yaml:"subscription_id,omitempty"`

	// Azure-specific: Resource Group name (required for Azure provider)
	// Will be created if it doesn't exist
	ResourceGroup string `yaml:"resource_group,omitempty"`
}

// CredentialsConfig contains cloud provider credentials.
// Note: It's recommended to use CLI credentials or environment variables
// instead of storing credentials in the manifest.
type CredentialsConfig struct {
	// AWS: Access key ID
	AccessKeyID string `yaml:"access_key_id,omitempty"`

	// AWS: Secret access key
	SecretAccessKey string `yaml:"secret_access_key,omitempty"`

	// GCP: Path to service account JSON key file
	ServiceAccountKeyPath string `yaml:"service_account_key_path,omitempty"`

	// GCP: Or provide service account JSON content directly (base64 encoded or raw JSON string)
	ServiceAccountKeyJSON string `yaml:"service_account_key_json,omitempty"`

	// Azure: Service Principal credentials (optional, can use Azure CLI credentials)
	Azure *AzureCredentialsConfig `yaml:"azure,omitempty"`
}

// AzureCredentialsConfig contains Azure Service Principal credentials.
type AzureCredentialsConfig struct {
	// Client ID (Application ID) of the Service Principal
	ClientID string `yaml:"client_id,omitempty"`

	// Client Secret of the Service Principal
	ClientSecret string `yaml:"client_secret,omitempty"`

	// Tenant ID (Directory ID)
	TenantID string `yaml:"tenant_id,omitempty"`
}

// ApplicationConfig defines the application being deployed.
type ApplicationConfig struct {
	// Name of the application (must be unique within the cloud account)
	Name string `yaml:"name"`

	// Description of the application - optional
	Description string `yaml:"description,omitempty"`
}

// EnvironmentConfig defines the environment for the application.
// An environment is a running instance of the application (e.g., dev, staging, prod).
type EnvironmentConfig struct {
	// Name of the environment (must be unique within the application)
	Name string `yaml:"name"`

	// CName/subdomain for the environment (creates: <cname>.<region>.<provider>.com)
	CName string `yaml:"cname"`
}

// DeploymentConfig specifies how the application should be deployed.
type DeploymentConfig struct {
	// Platform type (e.g., docker, nodejs, python)
	Platform string `yaml:"platform"`

	// Solution stack or runtime version (provider-specific, optional - will auto-detect if not specified)
	SolutionStack string `yaml:"solution_stack,omitempty"`

	// Source code location
	Source SourceConfig `yaml:"source"`
}

// SourceConfig specifies where the application source code is located.
type SourceConfig struct {
	// Type of source (local, s3, git)
	Type string `yaml:"type"`

	// Path to source code (file path, S3 URL, or git repository)
	Path string `yaml:"path"`
}

// InstanceConfig specifies the compute resources for the deployment.
type InstanceConfig struct {
	// Type of instance (e.g., t3.micro, t3.small)
	Type string `yaml:"type"`

	// Environment type: SingleInstance or LoadBalanced
	EnvironmentType string `yaml:"environment_type"`
}

// CloudRunConfig specifies GCP Cloud Run-specific configuration.
type CloudRunConfig struct {
	// CPU allocation (e.g., "1", "2", "4") - default: "1"
	CPU string `yaml:"cpu,omitempty"`

	// Memory allocation (e.g., "256Mi", "512Mi", "1Gi", "2Gi") - default: "512Mi"
	Memory string `yaml:"memory,omitempty"`

	// Maximum number of concurrent requests per container - default: 80
	MaxConcurrency int32 `yaml:"max_concurrency,omitempty"`

	// Minimum number of instances to keep running - default: 0 (scale to zero)
	MinInstances int32 `yaml:"min_instances,omitempty"`

	// Maximum number of instances to scale to - default: 100
	MaxInstances int32 `yaml:"max_instances,omitempty"`

	// Request timeout in seconds (max: 3600 for 1st gen, 86400 for 2nd gen) - default: 300
	TimeoutSeconds int32 `yaml:"timeout_seconds,omitempty"`
}

// AzureConfig specifies Azure Container Instances-specific configuration.
type AzureConfig struct {
	// CPU allocation in cores (e.g., 1.0, 2.0) - default: 1.0
	CPU float64 `yaml:"cpu,omitempty"`

	// Memory allocation in GB (e.g., 1.5, 2.0, 4.0) - default: 1.5
	MemoryGB float64 `yaml:"memory_gb,omitempty"`
}

// HealthCheckConfig defines how the cloud provider should check application health.
type HealthCheckConfig struct {
	// Type of health check (basic or enhanced)
	Type string `yaml:"type"`

	// Path to health check endpoint (e.g., /health, /api/status)
	Path string `yaml:"path"`
}

// MonitoringConfig defines monitoring and metrics collection settings.
type MonitoringConfig struct {
	// Enable enhanced health reporting (default: false)
	// Enhanced health provides detailed metrics like ApplicationRequests2xx, latency, etc.
	EnhancedHealth bool `yaml:"enhanced_health,omitempty"`

	// Enable CloudWatch custom metrics collection (default: false)
	// This enables application-level metrics beyond basic EC2 metrics
	CloudWatchMetrics bool `yaml:"cloudwatch_metrics,omitempty"`

	// CloudWatch Logs configuration (optional)
	CloudWatchLogs *CloudWatchLogsConfig `yaml:"cloudwatch_logs,omitempty"`
}

// CloudWatchLogsConfig defines CloudWatch Logs streaming settings.
type CloudWatchLogsConfig struct {
	// Enable streaming logs to CloudWatch (default: false)
	Enabled bool `yaml:"enabled,omitempty"`

	// Log retention in days (1, 3, 5, 7, 14, 30, 60, 90, 120, 150, 180, 365, 400, 545, 731, 1827, 3653)
	RetentionDays int `yaml:"retention_days,omitempty"`

	// Stream application logs (default: true if enabled)
	StreamLogs bool `yaml:"stream_logs,omitempty"`
}

// IAMConfig specifies IAM roles and profiles to use.
// This allows the application to access other cloud resources securely.
type IAMConfig struct {
	// Instance profile for EC2/compute instances - optional
	InstanceProfile string `yaml:"instance_profile,omitempty"`

	// Service role for the cloud service - optional
	ServiceRole string `yaml:"service_role,omitempty"`
}

// VaultConfig specifies HashiCorp Vault connection and authentication settings.
type VaultConfig struct {
	// Address is the Vault server URL (e.g., "http://127.0.0.1:8200")
	Address string `yaml:"address"`

	// Auth holds authentication configuration
	Auth VaultAuthConfig `yaml:"auth"`

	// TLSSkipVerify skips TLS certificate verification (not recommended for production)
	TLSSkipVerify bool `yaml:"tls_skip_verify,omitempty"`
}

// VaultAuthConfig specifies how to authenticate to Vault.
type VaultAuthConfig struct {
	// Method is the auth method: "token", "approle", "aws-iam", "gcp-iam"
	Method string `yaml:"method"`

	// Token for token authentication (can be environment variable reference like "${VAULT_TOKEN}")
	Token string `yaml:"token,omitempty"`

	// RoleID for AppRole authentication (can be environment variable reference)
	RoleID string `yaml:"role_id,omitempty"`

	// SecretID for AppRole authentication (can be environment variable reference)
	SecretID string `yaml:"secret_id,omitempty"`

	// Role for AWS IAM or GCP IAM authentication
	Role string `yaml:"role,omitempty"`
}

// SecretConfig defines a secret to fetch from Vault.
type SecretConfig struct {
	// Name is the environment variable name (e.g., "DATABASE_URL")
	Name string `yaml:"name"`

	// VaultPath is the full Vault path (e.g., "secret/data/myapp/database")
	VaultPath string `yaml:"vault_path"`

	// VaultKey is the key within the secret (e.g., "url")
	VaultKey string `yaml:"vault_key"`
}

// Load reads a manifest file from disk, parses it, and validates it.
// Returns an error if the file cannot be read, is invalid YAML, or fails validation.
//
// Example:
//
//	manifest, err := manifest.Load("deploy-manifest.yaml")
//	if err != nil {
//	  log.Fatal(err)
//	}
func Load(filename string) (*Manifest, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	// Expand environment variables in the YAML content
	expanded := os.ExpandEnv(string(data))

	var manifest Manifest
	if err := yaml.Unmarshal([]byte(expanded), &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	if err := manifest.Validate(); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}

	return &manifest, nil
}

// Validate checks if the manifest has all required fields and valid values.
// Returns an error describing what is invalid.
func (m *Manifest) Validate() error {
	if m.Image == "" {
		return fmt.Errorf("image is required (e.g., 'myapp:latest')")
	}
	if m.Provider.Name == "" {
		return fmt.Errorf("provider name is required")
	}
	if m.Application.Name == "" {
		return fmt.Errorf("application name is required")
	}
	if m.Environment.Name == "" {
		return fmt.Errorf("environment name is required")
	}

	// GCP-specific validation
	if m.Provider.Name == "gcp" {
		if m.Provider.ProjectID == "" {
			return fmt.Errorf("provider.project_id is required for GCP deployments")
		}
		if m.Provider.Credentials == nil ||
			(m.Provider.Credentials.ServiceAccountKeyPath == "" && m.Provider.Credentials.ServiceAccountKeyJSON == "") {
			return fmt.Errorf("provider.credentials.service_account_key_path or service_account_key_json is required for GCP deployments")
		}
		if m.Provider.BillingAccountID == "" {
			return fmt.Errorf("provider.billing_account_id is required for GCP deployments")
		}
	}

	// Azure-specific validation
	if m.Provider.Name == "azure" {
		if m.Provider.SubscriptionID == "" {
			return fmt.Errorf("provider.subscription_id is required for Azure deployments")
		}
		if m.Provider.ResourceGroup == "" {
			return fmt.Errorf("provider.resource_group is required for Azure deployments")
		}
	}

	// Vault validation
	if m.Vault != nil {
		if m.Vault.Address == "" {
			return fmt.Errorf("vault.address is required when vault is configured")
		}
		if m.Vault.Auth.Method == "" {
			return fmt.Errorf("vault.auth.method is required when vault is configured")
		}
	}

	return nil
}

// FetchVaultSecrets fetches secrets from Vault and returns them as a map.
// This is called during deployment to retrieve secrets and inject them as environment variables.
//
// Returns a map of environment variable names to secret values.
func (m *Manifest) FetchVaultSecrets(ctx context.Context) (map[string]string, error) {
	// If no Vault config or secrets, return empty map
	if m.Vault == nil || len(m.Secrets) == 0 {
		return make(map[string]string), nil
	}

	// Expand environment variables in Vault config
	vaultConfig := &vault.Config{
		Address:       m.Vault.Address,
		TLSSkipVerify: m.Vault.TLSSkipVerify,
		Auth: vault.AuthConfig{
			Method:   m.Vault.Auth.Method,
			Token:    expandEnvVars(m.Vault.Auth.Token),
			RoleID:   expandEnvVars(m.Vault.Auth.RoleID),
			SecretID: expandEnvVars(m.Vault.Auth.SecretID),
			Role:     m.Vault.Auth.Role,
		},
	}

	// Create Vault client
	client, err := vault.NewClient(vaultConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}
	defer client.Close()

	// Authenticate to Vault
	fmt.Println("Authenticating to Vault...")
	if err := client.Authenticate(ctx); err != nil {
		return nil, fmt.Errorf("failed to authenticate to vault: %w", err)
	}

	// Build secret refs map
	secretRefs := make(map[string]vault.SecretRef)
	for _, secret := range m.Secrets {
		secretRefs[secret.Name] = vault.SecretRef{
			Path: secret.VaultPath,
			Key:  secret.VaultKey,
		}
	}

	// Fetch all secrets
	fmt.Printf("Fetching %d secrets from Vault...\n", len(secretRefs))
	secrets, err := client.GetSecrets(ctx, secretRefs)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch secrets from vault: %w", err)
	}

	fmt.Printf("Successfully retrieved %d secrets from Vault\n", len(secrets))
	return secrets, nil
}

// expandEnvVars expands environment variable references in the format ${VAR_NAME}.
// For example: "${VAULT_TOKEN}" becomes the value of the VAULT_TOKEN environment variable.
func expandEnvVars(s string) string {
	if s == "" {
		return s
	}

	// Match ${VAR_NAME} pattern
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name (remove ${ and })
		varName := match[2 : len(match)-1]
		// Get environment variable value
		return os.Getenv(varName)
	})
}
