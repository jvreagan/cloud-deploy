package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid AWS manifest",
			content: `version: "1.0"
image: "test-app:latest"
provider:
  name: aws
  region: us-east-1
application:
  name: test-app
  description: Test application
environment:
  name: test-env
  cname: test
deployment:
  platform: docker
  source:
    type: local
    path: ./test
instance:
  type: t3.micro
  environment_type: SingleInstance
health_check:
  type: basic
  path: /health
`,
			shouldError: false,
		},
		{
			name: "valid GCP manifest",
			content: `version: "1.0"
image: "test-app:latest"
provider:
  name: gcp
  region: us-central1
  project_id: test-project
  billing_account_id: XXXXXX-XXXXXX-XXXXXX
  credentials:
    service_account_key_path: /path/to/key.json
application:
  name: test-app
environment:
  name: test-env
deployment:
  platform: docker
  source:
    type: local
    path: ./test
instance:
  type: n1-standard-1
  environment_type: SingleInstance
health_check:
  type: basic
  path: /health
`,
			shouldError: false,
		},
		{
			name: "invalid YAML",
			content: `invalid: yaml: content:
  - not: properly
  formatted
`,
			shouldError: true,
			errorMsg:    "failed to parse manifest",
		},
		{
			name: "missing provider name",
			content: `version: "1.0"
image: "test-app:latest"
provider:
  region: us-east-1
application:
  name: test-app
environment:
  name: test-env
`,
			shouldError: true,
			errorMsg:    "provider name is required",
		},
		{
			name: "missing application name",
			content: `version: "1.0"
image: "test-app:latest"
provider:
  name: aws
  region: us-east-1
environment:
  name: test-env
`,
			shouldError: true,
			errorMsg:    "application name is required",
		},
		{
			name: "missing environment name",
			content: `version: "1.0"
image: "test-app:latest"
provider:
  name: aws
  region: us-east-1
application:
  name: test-app
`,
			shouldError: true,
			errorMsg:    "environment name is required",
		},
		{
			name: "GCP missing project_id",
			content: `version: "1.0"
image: "test-app:latest"
provider:
  name: gcp
  region: us-central1
  billing_account_id: XXXXXX-XXXXXX-XXXXXX
  credentials:
    service_account_key_path: /path/to/key.json
application:
  name: test-app
environment:
  name: test-env
`,
			shouldError: true,
			errorMsg:    "provider.project_id is required for GCP deployments",
		},
		{
			name: "GCP missing credentials",
			content: `version: "1.0"
image: "test-app:latest"
provider:
  name: gcp
  region: us-central1
  project_id: test-project
  billing_account_id: XXXXXX-XXXXXX-XXXXXX
application:
  name: test-app
environment:
  name: test-env
`,
			shouldError: true,
			errorMsg:    "provider.credentials.service_account_key_path or service_account_key_json is required",
		},
		{
			name: "GCP missing billing account",
			content: `version: "1.0"
image: "test-app:latest"
provider:
  name: gcp
  region: us-central1
  project_id: test-project
  credentials:
    service_account_key_path: /path/to/key.json
application:
  name: test-app
environment:
  name: test-env
`,
			shouldError: true,
			errorMsg:    "provider.billing_account_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "manifest.yaml")

			err := os.WriteFile(tmpFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			// Test Load
			manifest, err := Load(tmpFile)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got none", tt.errorMsg)
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if manifest == nil {
					t.Error("Expected manifest to be non-nil")
				}
			}
		})
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	_, err := Load("/path/to/nonexistent/file.yaml")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
	if !contains(err.Error(), "failed to read manifest file") {
		t.Errorf("Expected 'failed to read manifest file' error, got: %v", err)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		manifest    *Manifest
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid AWS manifest",
			manifest: &Manifest{
				Image: "test-app:latest",
				Provider: ProviderConfig{
					Name:   "aws",
					Region: "us-east-1",
				},
				Application: ApplicationConfig{
					Name: "test-app",
				},
				Environment: EnvironmentConfig{
					Name: "test-env",
				},
			},
			shouldError: false,
		},
		{
			name: "valid GCP manifest",
			manifest: &Manifest{
				Image: "test-app:latest",
				Provider: ProviderConfig{
					Name:             "gcp",
					Region:           "us-central1",
					ProjectID:        "test-project",
					BillingAccountID: "XXXXXX-XXXXXX-XXXXXX",
					Credentials: &CredentialsConfig{
						ServiceAccountKeyPath: "/path/to/key.json",
					},
				},
				Application: ApplicationConfig{
					Name: "test-app",
				},
				Environment: EnvironmentConfig{
					Name: "test-env",
				},
			},
			shouldError: false,
		},
		{
			name: "missing provider name",
			manifest: &Manifest{
				Image: "test-app:latest",
				Provider: ProviderConfig{
					Region: "us-east-1",
				},
				Application: ApplicationConfig{
					Name: "test-app",
				},
				Environment: EnvironmentConfig{
					Name: "test-env",
				},
			},
			shouldError: true,
			errorMsg:    "provider name is required",
		},
		{
			name: "missing application name",
			manifest: &Manifest{
				Image: "test-app:latest",
				Provider: ProviderConfig{
					Name:   "aws",
					Region: "us-east-1",
				},
				Environment: EnvironmentConfig{
					Name: "test-env",
				},
			},
			shouldError: true,
			errorMsg:    "application name is required",
		},
		{
			name: "missing environment name",
			manifest: &Manifest{
				Image: "test-app:latest",
				Provider: ProviderConfig{
					Name:   "aws",
					Region: "us-east-1",
				},
				Application: ApplicationConfig{
					Name: "test-app",
				},
			},
			shouldError: true,
			errorMsg:    "environment name is required",
		},
		{
			name: "GCP missing project ID",
			manifest: &Manifest{
				Image: "test-app:latest",
				Provider: ProviderConfig{
					Name:             "gcp",
					Region:           "us-central1",
					BillingAccountID: "XXXXXX-XXXXXX-XXXXXX",
					Credentials: &CredentialsConfig{
						ServiceAccountKeyPath: "/path/to/key.json",
					},
				},
				Application: ApplicationConfig{
					Name: "test-app",
				},
				Environment: EnvironmentConfig{
					Name: "test-env",
				},
			},
			shouldError: true,
			errorMsg:    "provider.project_id is required for GCP deployments",
		},
		{
			name: "GCP missing credentials",
			manifest: &Manifest{
				Image: "test-app:latest",
				Provider: ProviderConfig{
					Name:             "gcp",
					Region:           "us-central1",
					ProjectID:        "test-project",
					BillingAccountID: "XXXXXX-XXXXXX-XXXXXX",
				},
				Application: ApplicationConfig{
					Name: "test-app",
				},
				Environment: EnvironmentConfig{
					Name: "test-env",
				},
			},
			shouldError: true,
			errorMsg:    "provider.credentials.service_account_key_path or service_account_key_json is required",
		},
		{
			name: "GCP with service_account_key_json",
			manifest: &Manifest{
				Image: "test-app:latest",
				Provider: ProviderConfig{
					Name:             "gcp",
					Region:           "us-central1",
					ProjectID:        "test-project",
					BillingAccountID: "XXXXXX-XXXXXX-XXXXXX",
					Credentials: &CredentialsConfig{
						ServiceAccountKeyJSON: `{"type":"service_account"}`,
					},
				},
				Application: ApplicationConfig{
					Name: "test-app",
				},
				Environment: EnvironmentConfig{
					Name: "test-env",
				},
			},
			shouldError: false,
		},
		{
			name: "GCP missing billing account",
			manifest: &Manifest{
				Image: "test-app:latest",
				Provider: ProviderConfig{
					Name:      "gcp",
					Region:    "us-central1",
					ProjectID: "test-project",
					Credentials: &CredentialsConfig{
						ServiceAccountKeyPath: "/path/to/key.json",
					},
				},
				Application: ApplicationConfig{
					Name: "test-app",
				},
				Environment: EnvironmentConfig{
					Name: "test-env",
				},
			},
			shouldError: true,
			errorMsg:    "provider.billing_account_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.manifest.Validate()

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got none", tt.errorMsg)
				} else if !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestManifestCompleteStructure(t *testing.T) {
	publicAccess := true
	manifest := &Manifest{
		Version: "1.0",
		Provider: ProviderConfig{
			Name:             "gcp",
			Region:           "us-central1",
			ProjectID:        "test-project",
			BillingAccountID: "123456-123456-123456",
			PublicAccess:     &publicAccess,
			OrganizationID:   "org-123",
			Credentials: &CredentialsConfig{
				AccessKeyID:           "aws-key",
				SecretAccessKey:       "aws-secret",
				ServiceAccountKeyPath: "/path/to/key.json",
				ServiceAccountKeyJSON: `{"type":"service_account"}`,
			},
		},
		Application: ApplicationConfig{
			Name:        "my-app",
			Description: "My application",
		},
		Environment: EnvironmentConfig{
			Name:  "prod",
			CName: "myapp",
		},
		Deployment: DeploymentConfig{
			Platform:      "docker",
			SolutionStack: "Docker running on Amazon Linux 2023",
			Source: SourceConfig{
				Type: "local",
				Path: "./app",
			},
		},
		Instance: InstanceConfig{
			Type:            "t3.micro",
			EnvironmentType: "SingleInstance",
		},
		CloudRun: &CloudRunConfig{
			CPU:            "2",
			Memory:         "1Gi",
			MaxConcurrency: 100,
			MinInstances:   1,
			MaxInstances:   10,
			TimeoutSeconds: 300,
		},
		HealthCheck: HealthCheckConfig{
			Type: "enhanced",
			Path: "/health",
		},
		Monitoring: MonitoringConfig{
			EnhancedHealth:    true,
			CloudWatchMetrics: true,
			CloudWatchLogs: &CloudWatchLogsConfig{
				Enabled:       true,
				RetentionDays: 7,
				StreamLogs:    true,
			},
		},
		IAM: IAMConfig{
			InstanceProfile: "my-instance-profile",
			ServiceRole:     "my-service-role",
		},
		EnvironmentVariables: map[string]string{
			"ENV": "production",
			"API": "https://api.example.com",
		},
		Tags: map[string]string{
			"Team":    "DevOps",
			"Project": "CloudDeploy",
		},
	}

	// Verify all fields are accessible
	if manifest.Version != "1.0" {
		t.Errorf("Expected version '1.0', got '%s'", manifest.Version)
	}
	if manifest.Provider.Name != "gcp" {
		t.Errorf("Expected provider 'gcp', got '%s'", manifest.Provider.Name)
	}
	if manifest.Application.Name != "my-app" {
		t.Errorf("Expected app name 'my-app', got '%s'", manifest.Application.Name)
	}
	if manifest.Environment.Name != "prod" {
		t.Errorf("Expected env name 'prod', got '%s'", manifest.Environment.Name)
	}
	if manifest.CloudRun.CPU != "2" {
		t.Errorf("Expected CPU '2', got '%s'", manifest.CloudRun.CPU)
	}
	if manifest.Monitoring.CloudWatchLogs.RetentionDays != 7 {
		t.Errorf("Expected retention 7, got %d", manifest.Monitoring.CloudWatchLogs.RetentionDays)
	}
	if manifest.EnvironmentVariables["ENV"] != "production" {
		t.Errorf("Expected ENV 'production', got '%s'", manifest.EnvironmentVariables["ENV"])
	}
	if manifest.Tags["Team"] != "DevOps" {
		t.Errorf("Expected Team 'DevOps', got '%s'", manifest.Tags["Team"])
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
