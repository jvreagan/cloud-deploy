// Package azure provides deployment functionality for Azure Container Instances (ACI).
// It implements the Provider interface for deploying containerized applications to Azure.
package azure

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerinstance/armcontainerinstance"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/jvreagan/cloud-deploy/pkg/manifest"
	"github.com/jvreagan/cloud-deploy/pkg/registry"
	"github.com/jvreagan/cloud-deploy/pkg/types"
)

// Provider implements the provider.Provider interface for Azure.
type Provider struct {
	subscriptionID      string
	location            string
	resourceGroup       string
	credential          azcore.TokenCredential
	containerClient     *armcontainerinstance.ContainerGroupsClient
	registryClient      *armcontainerregistry.RegistriesClient
	resourceGroupClient *armresources.ResourceGroupsClient
	blobServiceClient   *azblob.Client
}

// New creates a new Azure provider instance.
// It initializes all necessary Azure clients for managing resources.
//
// Parameters:
//   - ctx: Context for the operation
//   - subscriptionID: Azure subscription ID
//   - location: Azure region (e.g., "eastus", "westus2")
//   - resourceGroup: Resource group name (will be created if it doesn't exist)
//   - credentials: Azure credentials configuration
//   - m: Full manifest (for Vault credential loading)
//
// Authentication methods:
//  1. Vault: Load from HashiCorp Vault (if credentials.source == "vault")
//  2. Service Principal: Provide client_id, client_secret, tenant_id
//  3. Default Azure credentials: Leave credentials nil to use Azure CLI/Managed Identity
func New(ctx context.Context, subscriptionID, location, resourceGroup string, credentials *manifest.AzureCredentialsConfig, credConfig *manifest.CredentialsConfig, m *manifest.Manifest) (*Provider, error) {
	if subscriptionID == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}
	if location == "" {
		return nil, fmt.Errorf("location is required")
	}
	if resourceGroup == "" {
		return nil, fmt.Errorf("resource group is required")
	}

	var cred azcore.TokenCredential
	var err error

	// Check if credentials should be loaded from Vault
	if credConfig != nil && credConfig.Source == "vault" {
		fmt.Println("Loading Azure credentials from Vault...")

		// Get credentials from Vault using manifest helper
		vaultCreds, err := m.GetCloudCredentials(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load Azure credentials from Vault: %w", err)
		}

		if vaultCreds != nil {
			fmt.Println("✅ Successfully loaded Azure credentials from Vault")
			cred, err = azidentity.NewClientSecretCredential(
				vaultCreds.Azure.TenantID,
				vaultCreds.Azure.ClientID,
				vaultCreds.Azure.ClientSecret,
				nil,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create service principal credential from Vault: %w", err)
			}
		}
	} else if credentials != nil && credentials.ClientID != "" && credentials.ClientSecret != "" && credentials.TenantID != "" {
		// Authenticate based on credentials provided in manifest
		fmt.Println("Using Service Principal authentication from manifest")
		cred, err = azidentity.NewClientSecretCredential(
			credentials.TenantID,
			credentials.ClientID,
			credentials.ClientSecret,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create service principal credential: %w", err)
		}
	} else {
		fmt.Println("Using Default Azure credentials (Azure CLI or Managed Identity)")
		cred, err = azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create default credential: %w", err)
		}
	}

	// Create Azure clients
	containerClient, err := armcontainerinstance.NewContainerGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create container groups client: %w", err)
	}

	registryClient, err := armcontainerregistry.NewRegistriesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry client: %w", err)
	}

	resourceGroupClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource groups client: %w", err)
	}

	return &Provider{
		subscriptionID:      subscriptionID,
		location:            location,
		resourceGroup:       resourceGroup,
		credential:          cred,
		containerClient:     containerClient,
		registryClient:      registryClient,
		resourceGroupClient: resourceGroupClient,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "azure"
}

// Deploy deploys an application to Azure Container Instances.
// This method:
// 1. Creates resource group if it doesn't exist
// 2. Creates Azure Container Registry (ACR) if it doesn't exist
// 3. Pushes pre-built Docker image to ACR
// 4. Deploys to Azure Container Instances
// 5. Fetches Vault secrets if configured
func (p *Provider) Deploy(ctx context.Context, m *manifest.Manifest) (*types.DeploymentResult, error) {
	fmt.Println("Starting Azure Container Instances deployment...")

	// Step 0.5: Fetch secrets from Vault if configured
	if m.Vault != nil && len(m.Secrets) > 0 {
		vaultSecrets, err := m.FetchVaultSecrets(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch vault secrets: %w", err)
		}

		// Merge Vault secrets with environment variables
		if m.EnvironmentVariables == nil {
			m.EnvironmentVariables = make(map[string]string)
		}
		for key, value := range vaultSecrets {
			m.EnvironmentVariables[key] = value
		}
	}

	// Step 1: Ensure resource group exists
	if err := p.ensureResourceGroup(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure resource group: %w", err)
	}

	// Step 2: Create Container Registry (ACR)
	registryName := p.generateRegistryName(m.Application.Name)
	_, registryPassword, err := p.ensureContainerRegistry(ctx, registryName)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure container registry: %w", err)
	}

	// Step 3: Push image to ACR
	fmt.Println("\n=== Distributing image to ACR ===")
	acrRegistry, err := registry.NewACRRegistry(p.credential, p.subscriptionID, p.resourceGroup, registryName, p.location, "latest")
	if err != nil {
		return nil, fmt.Errorf("failed to create ACR registry handler: %w", err)
	}

	// Use Distributor to push image to registry
	distributor := registry.NewDistributor(m.Image)
	distributor.AddRegistry(acrRegistry)

	imageURIs, err := distributor.Distribute(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to distribute image to ACR: %w", err)
	}

	imageURI := imageURIs[acrRegistry.GetRegistryURL()]
	fmt.Printf("Successfully pushed image to ACR: %s\n", imageURI)

	// Step 4: Deploy to Azure Container Instances
	containerGroupName := m.Environment.Name
	fqdn, err := p.deployContainerGroup(ctx, m, containerGroupName, imageURI, registryName, registryPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy container group: %w", err)
	}

	// Step 5: Wait for container to be running
	fmt.Println("Waiting for container to be ready...")
	if err := p.waitForContainerGroup(ctx, containerGroupName); err != nil {
		return nil, fmt.Errorf("container group deployment failed: %w", err)
	}

	url := fmt.Sprintf("http://%s", fqdn)

	return &types.DeploymentResult{
		ApplicationName: m.Application.Name,
		EnvironmentName: m.Environment.Name,
		URL:             url,
		Status:          "Running",
		Message:         "Deployment successful",
	}, nil
}

// Destroy removes the Azure Container Instance and associated resources.
// This includes:
// - Terminating the container group
// - Optionally removing the container registry
func (p *Provider) Destroy(ctx context.Context, m *manifest.Manifest) error {
	fmt.Printf("Terminating container group: %s\n", m.Environment.Name)

	poller, err := p.containerClient.BeginDelete(ctx, p.resourceGroup, m.Environment.Name, nil)
	if err != nil {
		return fmt.Errorf("failed to begin delete container group: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete container group: %w", err)
	}

	fmt.Println("Container group terminated successfully")
	return nil
}

// Stop stops the running container group.
// Azure Container Instances doesn't support "stop" - containers are either running or deleted.
// This method deletes the container group, which effectively stops it.
// You can restart by running Deploy again.
func (p *Provider) Stop(ctx context.Context, m *manifest.Manifest) error {
	fmt.Printf("Stopping container group: %s\n", m.Environment.Name)
	fmt.Println("Note: Azure Container Instances will be deleted (restart with 'deploy' command)")

	return p.Destroy(ctx, m)
}

// Status returns the current status of the deployment.
func (p *Provider) Status(ctx context.Context, m *manifest.Manifest) (*types.DeploymentStatus, error) {
	resp, err := p.containerClient.Get(ctx, p.resourceGroup, m.Environment.Name, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get container group: %w", err)
	}

	containerGroup := resp.ContainerGroup

	status := "Unknown"
	if containerGroup.Properties != nil && containerGroup.Properties.ProvisioningState != nil {
		status = *containerGroup.Properties.ProvisioningState
	}

	var url string
	if containerGroup.Properties != nil && containerGroup.Properties.IPAddress != nil && containerGroup.Properties.IPAddress.Fqdn != nil {
		url = fmt.Sprintf("http://%s", *containerGroup.Properties.IPAddress.Fqdn)
	}

	var lastUpdated string
	if containerGroup.Properties != nil && containerGroup.Properties.InstanceView != nil &&
		len(containerGroup.Properties.InstanceView.Events) > 0 {
		lastEvent := containerGroup.Properties.InstanceView.Events[len(containerGroup.Properties.InstanceView.Events)-1]
		if lastEvent.LastTimestamp != nil {
			lastUpdated = lastEvent.LastTimestamp.Format(time.RFC3339)
		}
	}

	return &types.DeploymentStatus{
		ApplicationName: m.Application.Name,
		EnvironmentName: m.Environment.Name,
		Status:          status,
		Health:          "N/A", // Azure Container Instances doesn't have built-in health checks
		URL:             url,
		LastUpdated:     lastUpdated,
	}, nil
}

// Rollback rolls back the Azure Container Instance to the previous image version.
// This is achieved by redeploying with the previous image tag from ACR.
func (p *Provider) Rollback(ctx context.Context, m *manifest.Manifest) (*types.DeploymentResult, error) {
	fmt.Println("Starting Azure Container Instances rollback...")

	// Step 1: Get current container group to find current image
	resp, err := p.containerClient.Get(ctx, p.resourceGroup, m.Environment.Name, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get container group: %w", err)
	}

	containerGroup := resp.ContainerGroup
	if containerGroup.Properties == nil || len(containerGroup.Properties.Containers) == 0 {
		return nil, fmt.Errorf("no containers found in container group")
	}

	currentImage := *containerGroup.Properties.Containers[0].Properties.Image
	fmt.Printf("Current image: %s\n", currentImage)

	// Step 2: List all images in ACR to find previous version
	registryName := p.generateRegistryName(m.Application.Name)
	repositoryName := m.Application.Name

	previousImage, err := p.findPreviousImage(ctx, registryName, repositoryName, currentImage)
	if err != nil {
		return nil, fmt.Errorf("failed to find previous image: %w", err)
	}

	fmt.Printf("Rolling back to previous image: %s\n", previousImage)

	// Step 3: Update container group with previous image
	containerGroup.Properties.Containers[0].Properties.Image = to.Ptr(previousImage)

	poller, err := p.containerClient.BeginCreateOrUpdate(ctx, p.resourceGroup, m.Environment.Name, containerGroup, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin rollback: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("rollback failed: %w", err)
	}

	// Wait for container to be ready
	if err := p.waitForContainerGroup(ctx, m.Environment.Name); err != nil {
		return nil, fmt.Errorf("container group rollback failed: %w", err)
	}

	var url string
	if containerGroup.Properties.IPAddress != nil && containerGroup.Properties.IPAddress.Fqdn != nil {
		url = fmt.Sprintf("http://%s", *containerGroup.Properties.IPAddress.Fqdn)
	}

	return &types.DeploymentResult{
		ApplicationName: m.Application.Name,
		EnvironmentName: m.Environment.Name,
		URL:             url,
		Status:          "Running",
		Message:         fmt.Sprintf("Rolled back to image %s", previousImage),
	}, nil
}

// ensureResourceGroup creates the resource group if it doesn't exist.
func (p *Provider) ensureResourceGroup(ctx context.Context) error {
	fmt.Printf("Ensuring resource group exists: %s\n", p.resourceGroup)

	_, err := p.resourceGroupClient.CreateOrUpdate(ctx, p.resourceGroup, armresources.ResourceGroup{
		Location: to.Ptr(p.location),
		Tags: map[string]*string{
			"ManagedBy": to.Ptr("cloud-deploy"),
		},
	}, nil)

	return err
}

// generateRegistryName generates a valid ACR name from the application name.
// ACR names must be alphanumeric only, 5-50 characters.
func (p *Provider) generateRegistryName(appName string) string {
	// Remove non-alphanumeric characters and convert to lowercase
	name := strings.ToLower(appName)
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, name)

	// Ensure it's at least 5 characters
	if len(name) < 5 {
		name = name + "registry"
	}

	// Truncate to 50 characters
	if len(name) > 50 {
		name = name[:50]
	}

	return name
}

// ensureContainerRegistry creates or gets an Azure Container Registry.
// Returns the login server URL and admin password.
func (p *Provider) ensureContainerRegistry(ctx context.Context, registryName string) (string, string, error) {
	fmt.Printf("Ensuring container registry exists: %s\n", registryName)

	// Check if registry exists
	_, err := p.registryClient.Get(ctx, p.resourceGroup, registryName, nil)
	if err == nil {
		// Registry exists, get credentials
		return p.getRegistryCredentials(ctx, registryName)
	}

	// Create registry
	fmt.Printf("Creating new container registry: %s\n", registryName)
	poller, err := p.registryClient.BeginCreate(ctx, p.resourceGroup, registryName, armcontainerregistry.Registry{
		Location: to.Ptr(p.location),
		SKU: &armcontainerregistry.SKU{
			Name: to.Ptr(armcontainerregistry.SKUNameBasic),
		},
		Properties: &armcontainerregistry.RegistryProperties{
			AdminUserEnabled: to.Ptr(true),
		},
	}, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to begin create registry: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create registry: %w", err)
	}

	return p.getRegistryCredentials(ctx, registryName)
}

// getRegistryCredentials retrieves the ACR login server and admin password.
func (p *Provider) getRegistryCredentials(ctx context.Context, registryName string) (string, string, error) {
	// Get registry details
	resp, err := p.registryClient.Get(ctx, p.resourceGroup, registryName, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to get registry: %w", err)
	}

	loginServer := *resp.Registry.Properties.LoginServer

	// Get admin credentials
	credsResp, err := p.registryClient.ListCredentials(ctx, p.resourceGroup, registryName, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to get registry credentials: %w", err)
	}

	if len(credsResp.Passwords) == 0 {
		return "", "", fmt.Errorf("no passwords found for registry")
	}

	password := *credsResp.Passwords[0].Value

	return loginServer, password, nil
}


// deployContainerGroup creates or updates an Azure Container Instance.
func (p *Provider) deployContainerGroup(ctx context.Context, m *manifest.Manifest, name, image, registryName, registryPassword string) (string, error) {
	fmt.Printf("Deploying container group: %s\n", name)

	// Build environment variables
	envVars := make([]*armcontainerinstance.EnvironmentVariable, 0, len(m.EnvironmentVariables))
	for key, value := range m.EnvironmentVariables {
		envVars = append(envVars, &armcontainerinstance.EnvironmentVariable{
			Name:  to.Ptr(key),
			Value: to.Ptr(value),
		})
	}

	// Get registry login server
	registryLoginServer, _, err := p.getRegistryCredentials(ctx, registryName)
	if err != nil {
		return "", fmt.Errorf("failed to get registry login server: %w", err)
	}

	// Configure resources
	cpu := 1.0
	memoryGB := 1.5

	if m.Azure != nil {
		if m.Azure.CPU > 0 {
			cpu = m.Azure.CPU
		}
		if m.Azure.MemoryGB > 0 {
			memoryGB = m.Azure.MemoryGB
		}
	}

	// Create DNS label from environment name
	dnsLabel := strings.ToLower(m.Environment.Name)
	dnsLabel = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '-'
	}, dnsLabel)

	containerGroup := armcontainerinstance.ContainerGroup{
		Location: to.Ptr(p.location),
		Properties: &armcontainerinstance.ContainerGroupProperties{
			Containers: []*armcontainerinstance.Container{
				{
					Name: to.Ptr(m.Application.Name),
					Properties: &armcontainerinstance.ContainerProperties{
						Image: to.Ptr(image),
						Resources: &armcontainerinstance.ResourceRequirements{
							Requests: &armcontainerinstance.ResourceRequests{
								CPU:        to.Ptr(cpu),
								MemoryInGB: to.Ptr(memoryGB),
							},
						},
						Ports: []*armcontainerinstance.ContainerPort{
							{
								Port:     to.Ptr[int32](80),
								Protocol: to.Ptr(armcontainerinstance.ContainerNetworkProtocolTCP),
						},
						{
							Port:     to.Ptr[int32](443),
							Protocol: to.Ptr(armcontainerinstance.ContainerNetworkProtocolTCP),
							},
						},
						EnvironmentVariables: envVars,
					},
				},
			},
			OSType: to.Ptr(armcontainerinstance.OperatingSystemTypesLinux),
			IPAddress: &armcontainerinstance.IPAddress{
				Type: to.Ptr(armcontainerinstance.ContainerGroupIPAddressTypePublic),
				Ports: []*armcontainerinstance.Port{
					{
						Port:     to.Ptr[int32](80),
						Protocol: to.Ptr(armcontainerinstance.ContainerGroupNetworkProtocolTCP),
					},
					{
						Port:     to.Ptr[int32](443),
						Protocol: to.Ptr(armcontainerinstance.ContainerGroupNetworkProtocolTCP),
					},
				},
				DNSNameLabel: to.Ptr(dnsLabel),
			},
			ImageRegistryCredentials: []*armcontainerinstance.ImageRegistryCredential{
				{
					Server:   to.Ptr(registryLoginServer),
					Username: to.Ptr(registryName),
					Password: to.Ptr(registryPassword),
				},
			},
			RestartPolicy: to.Ptr(armcontainerinstance.ContainerGroupRestartPolicyAlways),
		},
		Tags: map[string]*string{
			"ManagedBy":   to.Ptr("cloud-deploy"),
			"Application": to.Ptr(m.Application.Name),
		},
	}

	poller, err := p.containerClient.BeginCreateOrUpdate(ctx, p.resourceGroup, name, containerGroup, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin create container group: %w", err)
	}

	result, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create container group: %w", err)
	}

	fqdn := ""
	if result.Properties != nil && result.Properties.IPAddress != nil && result.Properties.IPAddress.Fqdn != nil {
		fqdn = *result.Properties.IPAddress.Fqdn
	}

	return fqdn, nil
}

// waitForContainerGroup waits for the container group to reach running state.
func (p *Provider) waitForContainerGroup(ctx context.Context, name string) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	timeout := time.After(10 * time.Minute)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for container group to be ready")
		case <-ticker.C:
			resp, err := p.containerClient.Get(ctx, p.resourceGroup, name, nil)
			if err != nil {
				return fmt.Errorf("failed to get container group status: %w", err)
			}

			if resp.Properties == nil {
				continue
			}

			state := "Unknown"
			if resp.Properties.ProvisioningState != nil {
				state = *resp.Properties.ProvisioningState
			}

			fmt.Printf("Container group status: %s\n", state)

			if state == "Succeeded" {
				// Check if container is running
				if resp.Properties.InstanceView != nil && resp.Properties.InstanceView.State != nil {
					containerState := *resp.Properties.InstanceView.State
					if containerState == "Running" {
						return nil
					}
				} else {
					// If InstanceView or State is not available yet, assume succeeded means running
					return nil
				}
			}

			if state == "Failed" {
				return fmt.Errorf("container group provisioning failed")
			}
		}
	}
}

// findPreviousImage finds the previous image tag in ACR.
func (p *Provider) findPreviousImage(ctx context.Context, registryName, repositoryName, currentImage string) (string, error) {
	// Extract timestamp from current image
	// Format: <registry>/<repo>:<timestamp>
	parts := strings.Split(currentImage, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid image format: %s", currentImage)
	}

	// For simplicity, we'll return a message that rollback requires manual specification
	// In production, you would:
	// 1. List all tags in the ACR repository
	// 2. Parse timestamps
	// 3. Find the most recent tag before the current one

	return "", fmt.Errorf("automatic rollback not yet implemented - please specify image tag manually")
}

// createTarGz creates a tar.gz archive of a directory.
// This is used for packaging source code before uploading to Azure.
func createTarGz(sourceDir, targetFile string) error {
	file, err := os.Create(targetFile)
	if err != nil {
		return fmt.Errorf("failed to create tar file: %w", err)
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden files and directories
		if strings.HasPrefix(filepath.Base(path), ".") && path != sourceDir {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tarWriter, file)
		return err
	})
}

