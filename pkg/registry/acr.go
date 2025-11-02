package registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry"
)

// ACRRegistry represents an Azure Container Registry
type ACRRegistry struct {
	cred             azcore.TokenCredential
	subscriptionID   string
	resourceGroup    string
	registryName     string
	location         string
	imageTag         string
	registryURL      string
	imageURI         string
	loginServer      string
}

// NewACRRegistry creates a new ACR registry handler
func NewACRRegistry(cred azcore.TokenCredential, subscriptionID, resourceGroup, registryName, location, imageTag string) (*ACRRegistry, error) {
	return &ACRRegistry{
		cred:           cred,
		subscriptionID: subscriptionID,
		resourceGroup:  resourceGroup,
		registryName:   registryName,
		location:       location,
		imageTag:       imageTag,
	}, nil
}

// GetRegistryURL returns the ACR registry URL
func (a *ACRRegistry) GetRegistryURL() string {
	return a.registryURL
}

// GetImageURI returns the full image URI in ACR
func (a *ACRRegistry) GetImageURI() string {
	return a.imageURI
}

// Authenticate authenticates Docker with ACR
func (a *ACRRegistry) Authenticate(ctx context.Context) error {
	// Create registries client
	client, err := armcontainerregistry.NewRegistriesClient(a.subscriptionID, a.cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create ACR client: %w", err)
	}

	// Create or get registry
	fmt.Printf("Ensuring ACR registry exists: %s\n", a.registryName)

	// Try to get existing registry first
	getResp, err := client.Get(ctx, a.resourceGroup, a.registryName, nil)
	var registry *armcontainerregistry.Registry

	if err != nil {
		// Registry doesn't exist, create it
		fmt.Printf("Creating ACR registry: %s\n", a.registryName)

		poller, err := client.BeginCreate(ctx, a.resourceGroup, a.registryName, armcontainerregistry.Registry{
			Location: to.Ptr(a.location),
			SKU: &armcontainerregistry.SKU{
				Name: to.Ptr(armcontainerregistry.SKUNameBasic),
			},
			Properties: &armcontainerregistry.RegistryProperties{
				AdminUserEnabled: to.Ptr(true),
			},
		}, nil)
		if err != nil {
			return fmt.Errorf("failed to begin creating ACR registry: %w", err)
		}

		resp, err := poller.PollUntilDone(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to create ACR registry: %w", err)
		}
		registry = &resp.Registry
		fmt.Printf("Created ACR registry: %s\n", a.registryName)
	} else {
		fmt.Printf("ACR registry %s already exists\n", a.registryName)
		registry = &getResp.Registry
	}

	// Get login server
	if registry.Properties.LoginServer == nil {
		return fmt.Errorf("registry login server is nil")
	}
	a.loginServer = *registry.Properties.LoginServer
	a.registryURL = a.loginServer

	// Get admin credentials
	creds, err := client.ListCredentials(ctx, a.resourceGroup, a.registryName, nil)
	if err != nil {
		return fmt.Errorf("failed to get ACR credentials: %w", err)
	}

	if creds.Username == nil || len(creds.Passwords) == 0 {
		return fmt.Errorf("no admin credentials available for ACR")
	}

	username := *creds.Username
	password := *creds.Passwords[0].Value

	// Login to Docker registry
	fmt.Printf("Logging into ACR registry: %s\n", a.registryURL)
	_, err = execCommand(ctx, "docker", "login", "-u", username, "-p", password, a.registryURL)
	if err != nil {
		return fmt.Errorf("failed to login to ACR: %w", err)
	}

	fmt.Println("Successfully authenticated with ACR")
	return nil
}

// TagImage tags the source image for ACR
func (a *ACRRegistry) TagImage(ctx context.Context, sourceImage string) (string, error) {
	// Extract repository name from source image
	// Source image format: "name:tag" or "registry/name:tag"
	parts := strings.Split(sourceImage, "/")
	imageName := parts[len(parts)-1]

	// Remove tag if present
	if idx := strings.Index(imageName, ":"); idx != -1 {
		imageName = imageName[:idx]
	}

	// Build target image URI
	a.imageURI = fmt.Sprintf("%s/%s:%s", a.loginServer, imageName, a.imageTag)

	// Tag the image
	if err := dockerTag(ctx, sourceImage, a.imageURI); err != nil {
		return "", fmt.Errorf("failed to tag image for ACR: %w", err)
	}

	return a.imageURI, nil
}

// PushImage pushes the image to ACR
func (a *ACRRegistry) PushImage(ctx context.Context, taggedImage string) error {
	if err := dockerPush(ctx, taggedImage); err != nil {
		return fmt.Errorf("failed to push image to ACR: %w", err)
	}
	return nil
}
