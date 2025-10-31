//go:build integration

package aws

import (
	"context"
	"os"
	"testing"

	"github.com/jvreagan/cloud-deploy/pkg/manifest"
)

// TestAWSIntegration tests actual deployment to AWS Elastic Beanstalk.
// This test requires valid AWS credentials to be set in the environment.
//
// Required environment variables:
//   - AWS_ACCESS_KEY_ID
//   - AWS_SECRET_ACCESS_KEY
//   - AWS_REGION (optional, defaults to us-east-1)
//
// Run with: go test -tags=integration ./pkg/providers/aws -v
func TestAWSIntegration(t *testing.T) {
	// Skip if credentials are not available
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		t.Skip("Skipping AWS integration test: credentials not available")
	}

	ctx := context.Background()
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	// Create a test manifest
	m := &manifest.Manifest{
		Provider: manifest.ProviderConfig{
			Name:   "aws",
			Region: region,
		},
		Application: manifest.ApplicationConfig{
			Name:        "cloud-deploy-integration-test",
			Description: "Integration test application",
		},
		Environment: manifest.EnvironmentConfig{
			Name:  "integration-test-env",
			CName: "cloud-deploy-integration-test",
		},
		Deployment: manifest.DeploymentConfig{
			Platform: "docker",
			Source: manifest.SourceConfig{
				Type: "local",
				Path: "../../../examples/hello-world",
			},
		},
		Instance: manifest.InstanceConfig{
			Type:            "t3.micro",
			EnvironmentType: "SingleInstance",
		},
		HealthCheck: manifest.HealthCheckConfig{
			Type: "basic",
			Path: "/",
		},
	}

	// Create provider
	provider, err := New(ctx, region, nil)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	// Test deployment
	t.Run("Deploy", func(t *testing.T) {
		t.Log("Starting deployment to AWS Elastic Beanstalk...")
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

// TestAWSProviderCreation tests that we can create an AWS provider with credentials.
func TestAWSProviderCreation(t *testing.T) {
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		t.Skip("Skipping AWS provider creation test: credentials not available")
	}

	ctx := context.Background()
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	provider, err := New(ctx, region, nil)
	if err != nil {
		t.Fatalf("Failed to create AWS provider: %v", err)
	}

	if provider == nil {
		t.Fatal("Provider is nil")
	}

	if provider.Name() != "aws" {
		t.Errorf("Expected provider name 'aws', got '%s'", provider.Name())
	}

	t.Logf("Successfully created AWS provider for region: %s", region)
}
