package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/jvreagan/cloud-deploy/pkg/manifest"
)

func TestFactory(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		manifest     *manifest.Manifest
		expectError  bool
		errorMessage string
		providerName string
	}{
		{
			name: "AWS provider",
			manifest: &manifest.Manifest{
				Provider: manifest.ProviderConfig{
					Name:   "aws",
					Region: "us-east-1",
				},
			},
			expectError:  false,
			providerName: "aws",
		},
		{
			name: "AWS provider with credentials",
			manifest: &manifest.Manifest{
				Provider: manifest.ProviderConfig{
					Name:   "aws",
					Region: "us-west-2",
					Credentials: &manifest.CredentialsConfig{
						AccessKeyID:     "test-key",
						SecretAccessKey: "test-secret",
					},
				},
			},
			expectError:  false,
			providerName: "aws",
		},
		{
			name: "GCP provider - requires valid credentials",
			manifest: &manifest.Manifest{
				Provider: manifest.ProviderConfig{
					Name:             "gcp",
					Region:           "us-central1",
					ProjectID:        "test-project",
					BillingAccountID: "123456-123456-123456",
					Credentials: &manifest.CredentialsConfig{
						ServiceAccountKeyJSON: `{"type":"service_account","project_id":"test"}`,
					},
				},
			},
			expectError:  true,
			errorMessage: "failed to create Cloud Resource Manager client",
		},
		{
			name: "Azure provider",
			manifest: &manifest.Manifest{
				Provider: manifest.ProviderConfig{
					Name:           "azure",
					Region:         "eastus",
					SubscriptionID: "test-subscription-id",
					ResourceGroup:  "test-rg",
				},
			},
			expectError:  false,
			providerName: "azure",
		},
		{
			name: "OCI provider - not implemented",
			manifest: &manifest.Manifest{
				Provider: manifest.ProviderConfig{
					Name:   "oci",
					Region: "us-ashburn-1",
				},
			},
			expectError:  true,
			errorMessage: "OCI provider not yet implemented",
		},
		{
			name: "unknown provider",
			manifest: &manifest.Manifest{
				Provider: manifest.ProviderConfig{
					Name:   "unknown-provider",
					Region: "some-region",
				},
			},
			expectError:  true,
			errorMessage: "unknown provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := Factory(ctx, tt.manifest)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got none", tt.errorMessage)
					return
				}
				if !strings.Contains(err.Error(), tt.errorMessage) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMessage, err)
				}
				// Note: Provider may be partially initialized even when error occurs
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if provider == nil {
					t.Error("Expected provider to be non-nil")
					return
				}
				if provider.Name() != tt.providerName {
					t.Errorf("Expected provider name '%s', got '%s'", tt.providerName, provider.Name())
				}
			}
		})
	}
}

func TestFactoryAWSRegions(t *testing.T) {
	ctx := context.Background()

	regions := []string{
		"us-east-1",
		"us-east-2",
		"us-west-1",
		"us-west-2",
		"eu-west-1",
		"eu-central-1",
		"ap-southeast-1",
		"ap-northeast-1",
	}

	for _, region := range regions {
		t.Run("AWS-"+region, func(t *testing.T) {
			m := &manifest.Manifest{
				Provider: manifest.ProviderConfig{
					Name:   "aws",
					Region: region,
				},
			}

			provider, err := Factory(ctx, m)
			if err != nil {
				t.Errorf("Failed to create AWS provider for region %s: %v", region, err)
				return
			}
			if provider == nil {
				t.Errorf("Provider is nil for region %s", region)
				return
			}
			if provider.Name() != "aws" {
				t.Errorf("Expected provider name 'aws', got '%s'", provider.Name())
			}
		})
	}
}

func TestFactoryGCPConfiguration(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		config      manifest.ProviderConfig
		expectError bool
	}{
		{
			name: "GCP with service account key path",
			config: manifest.ProviderConfig{
				Name:             "gcp",
				Region:           "us-central1",
				ProjectID:        "test-project",
				BillingAccountID: "123456-123456-123456",
				Credentials: &manifest.CredentialsConfig{
					ServiceAccountKeyPath: "/path/to/key.json",
				},
			},
			expectError: true, // Will error because file doesn't exist, but that's expected
		},
		{
			name: "GCP with service account JSON",
			config: manifest.ProviderConfig{
				Name:             "gcp",
				Region:           "us-west1",
				ProjectID:        "another-project",
				BillingAccountID: "654321-654321-654321",
				Credentials: &manifest.CredentialsConfig{
					ServiceAccountKeyJSON: `{"type":"service_account","project_id":"test"}`,
				},
			},
			expectError: true, // Will error because credentials are incomplete
		},
		{
			name: "GCP with organization ID",
			config: manifest.ProviderConfig{
				Name:             "gcp",
				Region:           "europe-west1",
				ProjectID:        "org-project",
				BillingAccountID: "111111-222222-333333",
				OrganizationID:   "123456789",
				Credentials: &manifest.CredentialsConfig{
					ServiceAccountKeyJSON: `{"type":"service_account","project_id":"test"}`,
				},
			},
			expectError: true, // Will error because credentials are incomplete
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &manifest.Manifest{
				Provider: tt.config,
			}

			provider, err := Factory(ctx, m)

			if tt.expectError {
				if err == nil {
					t.Error("Expected an error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if provider == nil {
					t.Error("Expected provider to be non-nil")
					return
				}
				if provider.Name() != "gcp" {
					t.Errorf("Expected provider name 'gcp', got '%s'", provider.Name())
				}
			}
		})
	}
}

func TestFactoryEmptyManifest(t *testing.T) {
	ctx := context.Background()
	m := &manifest.Manifest{}

	provider, err := Factory(ctx, m)

	if err == nil {
		t.Error("Expected error for empty manifest")
	}
	if provider != nil {
		t.Error("Expected provider to be nil for empty manifest")
	}
}
