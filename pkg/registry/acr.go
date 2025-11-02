package registry

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry"
	"github.com/google/go-containerregistry/pkg/authn"
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

// GetImageReference returns the full image reference for ACR
func (a *ACRRegistry) GetImageReference() string {
	return a.imageURI
}

// GetAuthenticator returns the authenticator for ACR using admin credentials
func (a *ACRRegistry) GetAuthenticator(ctx context.Context) (authn.Authenticator, error) {
	// Create registries client
	client, err := armcontainerregistry.NewRegistriesClient(a.subscriptionID, a.cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create ACR client: %w", err)
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
			return nil, fmt.Errorf("failed to begin creating ACR registry: %w", err)
		}

		resp, err := poller.PollUntilDone(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create ACR registry: %w", err)
		}
		registry = &resp.Registry
		fmt.Printf("Created ACR registry: %s\n", a.registryName)
	} else {
		fmt.Printf("ACR registry %s already exists\n", a.registryName)
		registry = &getResp.Registry
	}

	// Get login server
	if registry.Properties.LoginServer == nil {
		return nil, fmt.Errorf("registry login server is nil")
	}
	a.loginServer = *registry.Properties.LoginServer
	a.registryURL = a.loginServer

	// Build image URI using registry name as repository
	a.imageURI = fmt.Sprintf("%s/%s:%s", a.loginServer, a.registryName, a.imageTag)

	// Get admin credentials
	creds, err := client.ListCredentials(ctx, a.resourceGroup, a.registryName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get ACR credentials: %w", err)
	}

	if creds.Username == nil || len(creds.Passwords) == 0 {
		return nil, fmt.Errorf("no admin credentials available for ACR")
	}

	username := *creds.Username
	password := *creds.Passwords[0].Value

	fmt.Println("Successfully retrieved ACR credentials")

	// Return authenticator with username and password
	return &authn.Basic{
		Username: username,
		Password: password,
	}, nil
}
