package registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"golang.org/x/oauth2/google"
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

// GetImageReference returns the full image reference for GCR
func (g *GCRRegistry) GetImageReference() string {
	return g.imageURI
}

// GetAuthenticator returns the authenticator for GCR using service account credentials
func (g *GCRRegistry) GetAuthenticator(ctx context.Context) (authn.Authenticator, error) {
	// Build registry URL
	// Format: REGION-docker.pkg.dev/PROJECT_ID/REPOSITORY_NAME
	g.registryURL = fmt.Sprintf("%s-docker.pkg.dev/%s/%s", g.region, g.projectID, g.repositoryName)

	// Build image URI using repository name as image
	g.imageURI = fmt.Sprintf("%s/%s:%s", g.registryURL, g.repositoryName, g.imageTag)

	// Create Artifact Registry client
	var client *artifactregistry.Service
	var err error

	if g.credentialsJSON != "" {
		client, err = artifactregistry.NewService(ctx, option.WithCredentialsJSON([]byte(g.credentialsJSON)))
	} else {
		client, err = artifactregistry.NewService(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create Artifact Registry client: %w", err)
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
				return nil, fmt.Errorf("failed to create Artifact Registry repository: %w", err)
			}
			fmt.Printf("Repository %s already exists\n", g.repositoryName)
		} else {
			fmt.Printf("Created Artifact Registry repository: %s\n", g.repositoryName)
		}
	} else {
		fmt.Printf("Artifact Registry repository %s already exists\n", g.repositoryName)
	}

	// Get OAuth2 token source from service account credentials
	var creds *google.Credentials
	if g.credentialsJSON != "" {
		creds, err = google.CredentialsFromJSON(ctx, []byte(g.credentialsJSON), "https://www.googleapis.com/auth/cloud-platform")
	} else {
		creds, err = google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get Google credentials: %w", err)
	}

	// Get an OAuth2 token
	token, err := creds.TokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth2 token: %w", err)
	}

	fmt.Println("Successfully retrieved GCR OAuth2 credentials")

	// Return authenticator with oauth2accesstoken as username and token as password
	return &authn.Basic{
		Username: "oauth2accesstoken",
		Password: token.AccessToken,
	}, nil
}
