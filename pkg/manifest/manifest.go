// Package manifest provides types and functions for parsing and validating
// cloud-deploy manifest files. Manifests are YAML files that define the
// complete configuration for deploying an application to a cloud provider.
package manifest

import (
	"fmt"
	"os"

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
	
	// Health check configuration
	HealthCheck HealthCheckConfig `yaml:"health_check"`
	
	// IAM configuration (roles, profiles) - optional
	IAM IAMConfig `yaml:"iam,omitempty"`
	
	// Environment variables to set in the deployment - optional
	EnvironmentVariables map[string]string `yaml:"environment_variables,omitempty"`
	
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
}

// CredentialsConfig contains cloud provider credentials.
// Note: It's recommended to use CLI credentials or environment variables
// instead of storing credentials in the manifest.
type CredentialsConfig struct {
	// Access key ID or equivalent
	AccessKeyID string `yaml:"access_key_id,omitempty"`
	
	// Secret access key or equivalent
	SecretAccessKey string `yaml:"secret_access_key,omitempty"`
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
	
	// Solution stack or runtime version (provider-specific)
	SolutionStack string `yaml:"solution_stack"`
	
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

// HealthCheckConfig defines how the cloud provider should check application health.
type HealthCheckConfig struct {
	// Type of health check (basic or enhanced)
	Type string `yaml:"type"`
	
	// Path to health check endpoint (e.g., /health, /api/status)
	Path string `yaml:"path"`
}

// IAMConfig specifies IAM roles and profiles to use.
// This allows the application to access other cloud resources securely.
type IAMConfig struct {
	// Instance profile for EC2/compute instances - optional
	InstanceProfile string `yaml:"instance_profile,omitempty"`
	
	// Service role for the cloud service - optional
	ServiceRole string `yaml:"service_role,omitempty"`
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

	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
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
	if m.Provider.Name == "" {
		return fmt.Errorf("provider name is required")
	}
	if m.Application.Name == "" {
		return fmt.Errorf("application name is required")
	}
	if m.Environment.Name == "" {
		return fmt.Errorf("environment name is required")
	}
	return nil
}
