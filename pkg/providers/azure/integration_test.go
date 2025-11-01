//go:build integration

package azure

import (
	"context"
	"os"
	"testing"

	"github.com/jvreagan/cloud-deploy/pkg/manifest"
)

// TestAzureIntegration tests actual deployment to Azure Container Instances.
// This test requires valid Azure credentials to be set in the environment.
//
// Required environment variables:
//   - AZURE_SUBSCRIPTION_ID: Your Azure subscription ID
//   - AZURE_CLIENT_ID: Service principal client ID
//   - AZURE_CLIENT_SECRET: Service principal client secret
//   - AZURE_TENANT_ID: Azure tenant ID
//   - AZURE_RESOURCE_GROUP (optional, defaults to cloud-deploy-test-rg)
//   - AZURE_LOCATION (optional, defaults to eastus)
//
// Run with: go test -tags=integration ./pkg/providers/azure -v
func TestAzureIntegration(t *testing.T) {
	// Skip if credentials are not available
	if os.Getenv("AZURE_SUBSCRIPTION_ID") == "" ||
		os.Getenv("AZURE_CLIENT_ID") == "" ||
		os.Getenv("AZURE_CLIENT_SECRET") == "" ||
		os.Getenv("AZURE_TENANT_ID") == "" {
		t.Skip("Skipping Azure integration test: credentials not available")
	}

	ctx := context.Background()
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	resourceGroup := os.Getenv("AZURE_RESOURCE_GROUP")
	if resourceGroup == "" {
		resourceGroup = "cloud-deploy-test-rg"
	}
	location := os.Getenv("AZURE_LOCATION")
	if location == "" {
		location = "eastus"
	}

	// Create a test manifest
	m := &manifest.Manifest{
		Provider: manifest.ProviderConfig{
			Name:           "azure",
			Region:         location,
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Credentials: &manifest.CredentialsConfig{
				Azure: &manifest.AzureCredentialsConfig{
					ClientID:     os.Getenv("AZURE_CLIENT_ID"),
					ClientSecret: os.Getenv("AZURE_CLIENT_SECRET"),
					TenantID:     os.Getenv("AZURE_TENANT_ID"),
				},
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
		Azure: &manifest.AzureConfig{
			CPU:      1.0,
			MemoryGB: 1.5,
		},
		EnvironmentVariables: map[string]string{
			"ENV": "test",
		},
		HealthCheck: manifest.HealthCheckConfig{
			Type: "basic",
			Path: "/",
		},
	}

	// Create provider
	provider, err := New(ctx, subscriptionID, location, resourceGroup, m.Provider.Credentials.Azure)
	if err != nil {
		t.Fatalf("Failed to create Azure provider: %v", err)
	}

	// Test deployment
	t.Run("Deploy", func(t *testing.T) {
		t.Log("Starting deployment to Azure Container Instances...")
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

// TestAzureProviderCreation tests that we can create an Azure provider with credentials.
func TestAzureProviderCreation(t *testing.T) {
	if os.Getenv("AZURE_SUBSCRIPTION_ID") == "" ||
		os.Getenv("AZURE_CLIENT_ID") == "" ||
		os.Getenv("AZURE_CLIENT_SECRET") == "" ||
		os.Getenv("AZURE_TENANT_ID") == "" {
		t.Skip("Skipping Azure provider creation test: credentials not available")
	}

	ctx := context.Background()
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	location := os.Getenv("AZURE_LOCATION")
	if location == "" {
		location = "eastus"
	}
	resourceGroup := os.Getenv("AZURE_RESOURCE_GROUP")
	if resourceGroup == "" {
		resourceGroup = "cloud-deploy-test-rg"
	}

	creds := &manifest.AzureCredentialsConfig{
		ClientID:     os.Getenv("AZURE_CLIENT_ID"),
		ClientSecret: os.Getenv("AZURE_CLIENT_SECRET"),
		TenantID:     os.Getenv("AZURE_TENANT_ID"),
	}

	provider, err := New(ctx, subscriptionID, location, resourceGroup, creds)
	if err != nil {
		t.Fatalf("Failed to create Azure provider: %v", err)
	}

	if provider == nil {
		t.Fatal("Provider is nil")
	}

	if provider.Name() != "azure" {
		t.Errorf("Expected provider name 'azure', got '%s'", provider.Name())
	}

	t.Logf("Successfully created Azure provider for subscription: %s", subscriptionID)
}
