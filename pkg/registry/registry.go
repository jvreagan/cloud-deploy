// Package registry provides functionality for distributing Docker images
// to cloud provider container registries (ECR, ACR, GCR).
package registry

import (
	"context"
	"fmt"

	"github.com/jvreagan/cloud-deploy/pkg/logging"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// Registry represents a cloud provider container registry
type Registry interface {
	// GetRegistryURL returns the full registry URL for the image
	GetRegistryURL() string

	// GetAuthenticator returns the authenticator for this registry
	GetAuthenticator(ctx context.Context) (authn.Authenticator, error)

	// GetImageReference returns the full image reference (with tag) for the registry
	GetImageReference() string

	// GetImageURI returns the full image URI in the registry (same as GetImageReference)
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

// Distribute reads the image from Docker daemon and pushes it to all registered registries
func (d *Distributor) Distribute(ctx context.Context) (map[string]string, error) {
	imageURIs := make(map[string]string)

	// Load image from Docker daemon once
	logging.Info("Loading image %s from Docker daemon...\n", d.sourceImage)
	sourceRef, err := name.ParseReference(d.sourceImage)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source image reference: %w", err)
	}

	img, err := daemon.Image(sourceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to load image from Docker daemon: %w", err)
	}
	logging.Info("Image loaded successfully")

	// Distribute to each registry
	for _, registry := range d.registries {
		logging.Info("\n=== Distributing to %s ===\n", registry.GetRegistryURL())

		// Get authenticator
		logging.Info("Preparing authentication...")
		auth, err := registry.GetAuthenticator(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get authenticator for registry %s: %w", registry.GetRegistryURL(), err)
		}

		// Parse target reference
		targetRef, err := name.ParseReference(registry.GetImageReference())
		if err != nil {
			return nil, fmt.Errorf("failed to parse target image reference: %w", err)
		}
		logging.Info("Target: %s\n", targetRef.Name())

		// Push image to registry using OCI Distribution API
		logging.Info("Pushing image to %s...\n", registry.GetRegistryURL())
		if err := remote.Write(targetRef, img, remote.WithAuth(auth), remote.WithContext(ctx)); err != nil {
			return nil, fmt.Errorf("failed to push image to registry %s: %w", registry.GetRegistryURL(), err)
		}
		logging.Info("Successfully pushed to %s\n", registry.GetRegistryURL())

		imageURIs[registry.GetRegistryURL()] = registry.GetImageURI()
	}

	return imageURIs, nil
}
