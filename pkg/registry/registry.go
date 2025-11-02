// Package registry provides functionality for distributing Docker images
// to cloud provider container registries (ECR, ACR, GCR).
package registry

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Registry represents a cloud provider container registry
type Registry interface {
	// GetRegistryURL returns the full registry URL for the image
	GetRegistryURL() string

	// Authenticate authenticates with the registry
	Authenticate(ctx context.Context) error

	// TagImage tags a local Docker image for the registry
	TagImage(ctx context.Context, sourceImage string) (string, error)

	// PushImage pushes the tagged image to the registry
	PushImage(ctx context.Context, taggedImage string) error

	// GetImageURI returns the full image URI in the registry
	GetImageURI() string
}

// Distributor handles distributing a Docker image to multiple cloud registries
type Distributor struct {
	sourceImage string
	registries  []Registry
}

// NewDistributor creates a new image distributor
func NewDistributor(sourceImage string) *Distributor {
	return &Distributor{
		sourceImage: sourceImage,
		registries:  make([]Registry, 0),
	}
}

// AddRegistry adds a registry to distribute the image to
func (d *Distributor) AddRegistry(registry Registry) {
	d.registries = append(d.registries, registry)
}

// Distribute authenticates, tags, and pushes the image to all registered registries
func (d *Distributor) Distribute(ctx context.Context) (map[string]string, error) {
	imageURIs := make(map[string]string)

	for _, registry := range d.registries {
		fmt.Printf("\n=== Distributing to %s ===\n", registry.GetRegistryURL())

		// Authenticate
		fmt.Println("Authenticating with registry...")
		if err := registry.Authenticate(ctx); err != nil {
			return nil, fmt.Errorf("failed to authenticate with registry %s: %w", registry.GetRegistryURL(), err)
		}

		// Tag image
		fmt.Printf("Tagging image %s for registry...\n", d.sourceImage)
		taggedImage, err := registry.TagImage(ctx, d.sourceImage)
		if err != nil {
			return nil, fmt.Errorf("failed to tag image for registry %s: %w", registry.GetRegistryURL(), err)
		}
		fmt.Printf("Tagged as: %s\n", taggedImage)

		// Push image
		fmt.Printf("Pushing image to %s...\n", registry.GetRegistryURL())
		if err := registry.PushImage(ctx, taggedImage); err != nil {
			return nil, fmt.Errorf("failed to push image to registry %s: %w", registry.GetRegistryURL(), err)
		}
		fmt.Printf("Successfully pushed to %s\n", registry.GetRegistryURL())

		imageURIs[registry.GetRegistryURL()] = registry.GetImageURI()
	}

	return imageURIs, nil
}

// execCommand executes a shell command and returns the output
func execCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command failed: %s\nOutput: %s", err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

// dockerTag tags a Docker image
func dockerTag(ctx context.Context, sourceImage, targetImage string) error {
	fmt.Printf("Running: docker tag %s %s\n", sourceImage, targetImage)
	_, err := execCommand(ctx, "docker", "tag", sourceImage, targetImage)
	return err
}

// dockerPush pushes a Docker image to a registry
func dockerPush(ctx context.Context, image string) error {
	fmt.Printf("Running: docker push %s\n", image)
	_, err := execCommand(ctx, "docker", "push", image)
	return err
}
