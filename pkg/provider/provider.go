// Package provider defines the interface that all cloud providers must implement.
// This abstraction allows cloud-deploy to support multiple cloud providers
// (AWS, GCP, Azure, OCI) with a consistent interface.
package provider

import (
	"context"
	"fmt"

	"github.com/jvreagan/cloud-deploy/pkg/manifest"
	"github.com/jvreagan/cloud-deploy/pkg/providers/aws"
	"github.com/jvreagan/cloud-deploy/pkg/providers/gcp"
	"github.com/jvreagan/cloud-deploy/pkg/types"
)

// Provider defines the interface that all cloud providers must implement.
// Each provider (AWS, GCP, Azure, OCI) implements these methods to handle
// provider-specific deployment logic.
//
// Example implementation:
//
//	type AWSProvider struct {}
//
//	func (p *AWSProvider) Name() string { return "aws" }
//	func (p *AWSProvider) Deploy(ctx, manifest) (*DeploymentResult, error) { ... }
type Provider interface {
	// Name returns the provider name (e.g., "aws", "gcp", "azure", "oci")
	Name() string
	
	// Deploy deploys an application according to the manifest.
	// This method:
	// 1. Creates the application if it doesn't exist
	// 2. Creates the environment if it doesn't exist
	// 3. Packages and uploads the source code
	// 4. Deploys the application to the environment
	//
	// Returns deployment information including the application URL.
	Deploy(ctx context.Context, m *manifest.Manifest) (*types.DeploymentResult, error)
	
	// Destroy removes the deployed application and all associated resources.
	// This includes:
	// - Terminating the environment
	// - Deleting application versions
	// - Removing the application
	//
	// Use with caution - this action is typically irreversible.
	Destroy(ctx context.Context, m *manifest.Manifest) error

	// Stop stops the running environment/service but preserves the application and versions.
	// This is useful for temporarily stopping resource consumption without destroying everything.
	// Provider-specific behavior:
	// - AWS: Terminates the Elastic Beanstalk environment (keeps application + versions)
	// - GCP: Deletes the Cloud Run service (keeps container images)
	//
	// After stopping, you can restart by running Deploy again.
	Stop(ctx context.Context, m *manifest.Manifest) error

	// Status returns the current status of the deployment.
	// This queries the cloud provider for:
	// - Application status
	// - Environment status
	// - Health status
	// - Deployment URL
	// - Last update time
	Status(ctx context.Context, m *manifest.Manifest) (*types.DeploymentStatus, error)
}

// Factory creates a provider based on the manifest configuration.
// It requires a context and manifest to properly initialize the provider
// with the correct region and settings.
//
// Supported providers: aws, gcp, azure, oci
//
// Example:
//
//	provider, err := provider.Factory(ctx, manifest)
//	if err != nil {
//	  log.Fatal(err)
//	}
//
// Returns an error if the provider is not supported or not yet implemented.
func Factory(ctx context.Context, m *manifest.Manifest) (Provider, error) {
	switch m.Provider.Name {
	case "aws":
		return aws.New(ctx, m.Provider.Region, m.Provider.Credentials)
	case "gcp":
		return gcp.New(ctx, &m.Provider)
	case "azure":
		return nil, fmt.Errorf("Azure provider not yet implemented")
	case "oci":
		return nil, fmt.Errorf("OCI provider not yet implemented")
	default:
		return nil, fmt.Errorf("unknown provider: %s", m.Provider.Name)
	}
}
