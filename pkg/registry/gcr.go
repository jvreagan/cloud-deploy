package registry

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/option"
	artifactregistry "google.golang.org/api/artifactregistry/v1"
)

// GCRRegistry represents a Google Container Registry (Artifact Registry)
type GCRRegistry struct {
	projectID      string
	region         string
	repositoryName string
	imageTag       string
	registryURL    string
	imageURI       string
	credentialsJSON string
}

// NewGCRRegistry creates a new GCR registry handler
func NewGCRRegistry(projectID, region, repositoryName, imageTag, credentialsJSON string) (*GCRRegistry, error) {
	return &GCRRegistry{
		projectID:       projectID,
		region:          region,
		repositoryName:  repositoryName,
		imageTag:        imageTag,
		credentialsJSON: credentialsJSON,
	}, nil
}

// GetRegistryURL returns the GCR registry URL
func (g *GCRRegistry) GetRegistryURL() string {
	return g.registryURL
}

// GetImageURI returns the full image URI in GCR
func (g *GCRRegistry) GetImageURI() string {
	return g.imageURI
}

// Authenticate authenticates Docker with GCR
func (g *GCRRegistry) Authenticate(ctx context.Context) error {
	// Build registry URL
	// Format: REGION-docker.pkg.dev/PROJECT_ID/REPOSITORY_NAME
	g.registryURL = fmt.Sprintf("%s-docker.pkg.dev/%s/%s", g.region, g.projectID, g.repositoryName)

	// Create Artifact Registry client
	var client *artifactregistry.Service
	var err error

	if g.credentialsJSON != "" {
		client, err = artifactregistry.NewService(ctx, option.WithCredentialsJSON([]byte(g.credentialsJSON)))
	} else {
		client, err = artifactregistry.NewService(ctx)
	}
	if err != nil {
		return fmt.Errorf("failed to create Artifact Registry client: %w", err)
	}

	// Create repository if it doesn't exist
	fmt.Printf("Ensuring Artifact Registry repository exists: %s\n", g.repositoryName)

	parent := fmt.Sprintf("projects/%s/locations/%s", g.projectID, g.region)
	repoName := fmt.Sprintf("%s/repositories/%s", parent, g.repositoryName)

	// Try to get the repository
	_, err = client.Projects.Locations.Repositories.Get(repoName).Context(ctx).Do()
	if err != nil {
		// Repository doesn't exist, create it
		fmt.Printf("Creating Artifact Registry repository: %s\n", g.repositoryName)

		repo := &artifactregistry.Repository{
			Format:      "DOCKER",
			Description: fmt.Sprintf("Repository for %s", g.repositoryName),
		}

		_, err = client.Projects.Locations.Repositories.Create(parent, repo).
			RepositoryId(g.repositoryName).
			Context(ctx).
			Do()
		if err != nil {
			// Ignore if already exists
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("failed to create Artifact Registry repository: %w", err)
			}
			fmt.Printf("Repository %s already exists\n", g.repositoryName)
		} else {
			fmt.Printf("Created Artifact Registry repository: %s\n", g.repositoryName)
		}
	} else {
		fmt.Printf("Artifact Registry repository %s already exists\n", g.repositoryName)
	}

	// Configure Docker to use gcloud credentials
	fmt.Println("Configuring Docker authentication for GCR...")

	// Use gcloud auth configure-docker for the region
	registryHost := fmt.Sprintf("%s-docker.pkg.dev", g.region)
	_, err = execCommand(ctx, "gcloud", "auth", "configure-docker", registryHost, "--quiet")
	if err != nil {
		return fmt.Errorf("failed to configure Docker for GCR: %w", err)
	}

	fmt.Println("Successfully authenticated with GCR")
	return nil
}

// TagImage tags the source image for GCR
func (g *GCRRegistry) TagImage(ctx context.Context, sourceImage string) (string, error) {
	// Extract image name from source
	parts := strings.Split(sourceImage, "/")
	imageName := parts[len(parts)-1]

	// Remove tag if present
	if idx := strings.Index(imageName, ":"); idx != -1 {
		imageName = imageName[:idx]
	}

	// Build target image URI
	// Format: REGION-docker.pkg.dev/PROJECT_ID/REPOSITORY_NAME/IMAGE_NAME:TAG
	g.imageURI = fmt.Sprintf("%s/%s:%s", g.registryURL, imageName, g.imageTag)

	// Tag the image
	if err := dockerTag(ctx, sourceImage, g.imageURI); err != nil {
		return "", fmt.Errorf("failed to tag image for GCR: %w", err)
	}

	return g.imageURI, nil
}

// PushImage pushes the image to GCR
func (g *GCRRegistry) PushImage(ctx context.Context, taggedImage string) error {
	if err := dockerPush(ctx, taggedImage); err != nil {
		return fmt.Errorf("failed to push image to GCR: %w", err)
	}
	return nil
}
