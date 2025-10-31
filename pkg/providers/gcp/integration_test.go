//go:build integration

package gcp

import (
	"context"
	"os"
	"testing"

	"github.com/jvreagan/cloud-deploy/pkg/manifest"
)

// TestGCPIntegration tests actual deployment to Google Cloud Run.
// This test requires valid GCP credentials to be set in the environment.
//
// Required environment variables:
//   - GCP_PROJECT_ID: Your GCP project ID
//   - GCP_CREDENTIALS: Service account JSON credentials (as a string)
//   - GCP_BILLING_ACCOUNT_ID: Billing account ID (format: XXXXXX-XXXXXX-XXXXXX)
//
// Run with: go test -tags=integration ./pkg/providers/gcp -v
func TestGCPIntegration(t *testing.T) {
	// Skip if credentials are not available
	if os.Getenv("GCP_PROJECT_ID") == "" || os.Getenv("GCP_CREDENTIALS") == "" {
		t.Skip("Skipping GCP integration test: credentials not available")
	}

	ctx := context.Background()
	projectID := os.Getenv("GCP_PROJECT_ID")
	billingAccountID := os.Getenv("GCP_BILLING_ACCOUNT_ID")
	if billingAccountID == "" {
		t.Skip("Skipping GCP integration test: GCP_BILLING_ACCOUNT_ID not set")
	}

	// Create a test manifest
	m := &manifest.Manifest{
		Provider: manifest.ProviderConfig{
			Name:             "gcp",
			Region:           "us-central1",
			ProjectID:        projectID,
			BillingAccountID: billingAccountID,
			Credentials: &manifest.CredentialsConfig{
				ServiceAccountKeyJSON: os.Getenv("GCP_CREDENTIALS"),
			},
		},
		Application: manifest.ApplicationConfig{
			Name:        "cloud-deploy-integration-test",
			Description: "Integration test application",
		},
		Environment: manifest.EnvironmentConfig{
			Name: "integration-test-env",
		},
		Deployment: manifest.DeploymentConfig{
			Platform: "docker",
			Source: manifest.SourceConfig{
				Type: "local",
				Path: "../../../examples/hello-world",
			},
		},
		Instance: manifest.InstanceConfig{
			Type:            "cloud-run",
			EnvironmentType: "serverless",
		},
		CloudRun: &manifest.CloudRunConfig{
			CPU:            "1",
			Memory:         "512Mi",
			MaxConcurrency: 80,
			MinInstances:   0,
			MaxInstances:   10,
		},
		HealthCheck: manifest.HealthCheckConfig{
			Type: "basic",
			Path: "/",
		},
	}

	// Create provider
	provider, err := New(ctx, &m.Provider)
	if err != nil {
		t.Fatalf("Failed to create GCP provider: %v", err)
	}

	// Test deployment
	t.Run("Deploy", func(t *testing.T) {
		t.Log("Starting deployment to Google Cloud Run...")
		result, err := provider.Deploy(ctx, m)
		if err != nil {
			t.Fatalf("Deployment failed: %v", err)
		}

		if result == nil {
			t.Fatal("Deployment result is nil")
		}

		t.Logf("Deployment successful!")
		t.Logf("  Application: %s", result.ApplicationName)
		t.Logf("  Environment: %s", result.EnvironmentName)
		t.Logf("  URL: %s", result.URL)
		t.Logf("  Status: %s", result.Status)
	})

	// Test status check
	t.Run("Status", func(t *testing.T) {
		t.Log("Checking deployment status...")
		status, err := provider.Status(ctx, m)
		if err != nil {
			t.Fatalf("Status check failed: %v", err)
		}

		if status == nil {
			t.Fatal("Status is nil")
		}

		t.Logf("Status retrieved successfully!")
		t.Logf("  Status: %s", status.Status)
		t.Logf("  Health: %s", status.Health)
		t.Logf("  URL: %s", status.URL)
	})

	// Cleanup: destroy the deployment
	t.Run("Destroy", func(t *testing.T) {
		t.Log("Cleaning up: destroying deployment...")
		err := provider.Destroy(ctx, m)
		if err != nil {
			t.Fatalf("Destroy failed: %v", err)
		}
		t.Log("Deployment destroyed successfully")
	})
}

// TestGCPProviderCreation tests that we can create a GCP provider with credentials.
func TestGCPProviderCreation(t *testing.T) {
	if os.Getenv("GCP_PROJECT_ID") == "" || os.Getenv("GCP_CREDENTIALS") == "" {
		t.Skip("Skipping GCP provider creation test: credentials not available")
	}

	ctx := context.Background()
	projectID := os.Getenv("GCP_PROJECT_ID")
	billingAccountID := os.Getenv("GCP_BILLING_ACCOUNT_ID")
	if billingAccountID == "" {
		billingAccountID = "000000-000000-000000" // Placeholder for test
	}

	config := &manifest.ProviderConfig{
		Name:             "gcp",
		Region:           "us-central1",
		ProjectID:        projectID,
		BillingAccountID: billingAccountID,
		Credentials: &manifest.CredentialsConfig{
			ServiceAccountKeyJSON: os.Getenv("GCP_CREDENTIALS"),
		},
	}

	provider, err := New(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create GCP provider: %v", err)
	}

	if provider == nil {
		t.Fatal("Provider is nil")
	}

	if provider.Name() != "gcp" {
		t.Errorf("Expected provider name 'gcp', got '%s'", provider.Name())
	}

	t.Logf("Successfully created GCP provider for project: %s", projectID)
}
