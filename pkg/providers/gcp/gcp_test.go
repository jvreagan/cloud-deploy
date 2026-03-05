package gcp

import (
	"strings"
	"testing"

	"github.com/jvreagan/cloud-deploy/pkg/manifest"
)

func TestProviderName(t *testing.T) {
	provider := &Provider{
		projectID: "test-project",
		region:    "us-central1",
	}

	if provider.Name() != "gcp" {
		t.Errorf("Expected provider name 'gcp', got '%s'", provider.Name())
	}
}

func TestLoadCredentials(t *testing.T) {
	tests := []struct {
		name        string
		creds       *manifest.CredentialsConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "with service account key path",
			creds: &manifest.CredentialsConfig{
				ServiceAccountKeyPath: "/path/to/key.json",
			},
			expectError: false,
		},
		{
			name: "with service account key JSON",
			creds: &manifest.CredentialsConfig{
				ServiceAccountKeyJSON: `{"type":"service_account","project_id":"test"}`,
			},
			expectError: false,
		},
		{
			name:        "with nil credentials",
			creds:       nil,
			expectError: true,
			errorMsg:    "credentials are required",
		},
		{
			name:        "with empty credentials",
			creds:       &manifest.CredentialsConfig{},
			expectError: true,
			errorMsg:    "either service_account_key_path, service_account_key_json, or source: environment is required",
		},
		{
			name: "with both path and JSON (path takes precedence)",
			creds: &manifest.CredentialsConfig{
				ServiceAccountKeyPath: "/path/to/key.json",
				ServiceAccountKeyJSON: `{"type":"service_account"}`,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			option, err := loadCredentials(tt.creds)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got none", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMsg, err)
				}
				if option != nil {
					t.Error("Expected option to be nil when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if option == nil {
					t.Error("Expected option to be non-nil")
				}
			}
		})
	}
}

func TestLoadCredentialsWithInvalidJSON(t *testing.T) {
	tests := []struct {
		name        string
		creds       *manifest.CredentialsConfig
		expectError bool
	}{
		{
			name: "with empty JSON",
			creds: &manifest.CredentialsConfig{
				ServiceAccountKeyJSON: `{}`,
			},
			expectError: false,
		},
		{
			name: "with valid JSON string",
			creds: &manifest.CredentialsConfig{
				ServiceAccountKeyJSON: `{"type":"service_account","project_id":"test"}`,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			option, err := loadCredentials(tt.creds)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if option != nil {
					t.Error("Expected option to be nil when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if option == nil {
					t.Error("Expected option to be non-nil")
				}
			}
		})
	}
}

func TestProviderRegionAndProject(t *testing.T) {
	tests := []struct {
		name      string
		projectID string
		region    string
	}{
		{"us-central1", "test-project-1", "us-central1"},
		{"us-east1", "test-project-2", "us-east1"},
		{"europe-west1", "test-project-3", "europe-west1"},
		{"asia-southeast1", "test-project-4", "asia-southeast1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &Provider{
				projectID: tt.projectID,
				region:    tt.region,
			}

			if provider.projectID != tt.projectID {
				t.Errorf("Expected projectID '%s', got '%s'", tt.projectID, provider.projectID)
			}
			if provider.region != tt.region {
				t.Errorf("Expected region '%s', got '%s'", tt.region, provider.region)
			}
		})
	}
}

func TestProviderPublicAccessSetting(t *testing.T) {
	tests := []struct {
		name         string
		publicAccess bool
	}{
		{"public access enabled", true},
		{"public access disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &Provider{
				projectID:    "test-project",
				region:       "us-central1",
				publicAccess: tt.publicAccess,
			}

			if provider.publicAccess != tt.publicAccess {
				t.Errorf("Expected publicAccess %v, got %v", tt.publicAccess, provider.publicAccess)
			}
		})
	}
}

func TestLoadCredentialsWithEnvironmentSource(t *testing.T) {
	creds := &manifest.CredentialsConfig{
		Source: "environment",
	}

	option, err := loadCredentials(creds)
	if err != nil {
		t.Errorf("Unexpected error for environment source: %v", err)
	}
	if option != nil {
		t.Error("Expected nil option for environment source (uses Application Default Credentials)")
	}
}

func TestLoadCredentialsSourcePrecedence(t *testing.T) {
	// When source is "environment", path and JSON should be ignored
	creds := &manifest.CredentialsConfig{
		Source:                "environment",
		ServiceAccountKeyPath: "/some/path.json",
		ServiceAccountKeyJSON: `{"type":"service_account"}`,
	}

	option, err := loadCredentials(creds)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if option != nil {
		t.Error("Expected nil option when source is environment, regardless of other fields")
	}
}

func TestLoadCredentialsPathPrecedenceOverJSON(t *testing.T) {
	// When both path and JSON are set (without environment source), path takes precedence
	creds := &manifest.CredentialsConfig{
		ServiceAccountKeyPath: "/path/to/key.json",
		ServiceAccountKeyJSON: `{"type":"service_account","project_id":"test"}`,
	}

	option, err := loadCredentials(creds)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if option == nil {
		t.Error("Expected non-nil option when path is specified")
	}
}

func TestProviderFields(t *testing.T) {
	tests := []struct {
		name           string
		projectID      string
		region         string
		publicAccess   bool
		billingAccount string
		organizationID string
	}{
		{
			name:           "full config",
			projectID:      "my-project",
			region:         "us-central1",
			publicAccess:   true,
			billingAccount: "012345-6789AB-CDEF01",
			organizationID: "123456789",
		},
		{
			name:      "minimal config",
			projectID: "minimal",
			region:    "europe-west1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{
				projectID:      tt.projectID,
				region:         tt.region,
				publicAccess:   tt.publicAccess,
				billingAccount: tt.billingAccount,
				organizationID: tt.organizationID,
			}

			if p.Name() != "gcp" {
				t.Errorf("Expected Name() = 'gcp', got '%s'", p.Name())
			}
			if p.projectID != tt.projectID {
				t.Errorf("Expected projectID '%s', got '%s'", tt.projectID, p.projectID)
			}
			if p.region != tt.region {
				t.Errorf("Expected region '%s', got '%s'", tt.region, p.region)
			}
			if p.publicAccess != tt.publicAccess {
				t.Errorf("Expected publicAccess %v, got %v", tt.publicAccess, p.publicAccess)
			}
			if p.billingAccount != tt.billingAccount {
				t.Errorf("Expected billingAccount '%s', got '%s'", tt.billingAccount, p.billingAccount)
			}
			if p.organizationID != tt.organizationID {
				t.Errorf("Expected organizationID '%s', got '%s'", tt.organizationID, p.organizationID)
			}
		})
	}
}
