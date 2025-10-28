// Package types provides shared types used across cloud-deploy packages.
package types

// DeploymentResult contains information about a successful deployment.
// This is returned by the Deploy method after a deployment completes.
type DeploymentResult struct {
	// Name of the deployed application
	ApplicationName string

	// Name of the deployed environment
	EnvironmentName string

	// Public URL where the application can be accessed
	URL string

	// Current status (e.g., "Launching", "Ready", "Updating")
	Status string

	// Human-readable message with deployment details
	Message string
}

// DeploymentStatus contains the current status of a deployment.
// This is returned by the Status method.
type DeploymentStatus struct {
	// Name of the application
	ApplicationName string

	// Name of the environment
	EnvironmentName string

	// Current status (e.g., "Ready", "Updating", "Terminating")
	Status string

	// Health status (e.g., "Green", "Yellow", "Red", "Grey")
	Health string

	// Public URL where the application can be accessed
	URL string

	// Timestamp of last update (format varies by provider)
	LastUpdated string
}
