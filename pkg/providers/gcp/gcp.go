// Package gcp provides a Google Cloud Run provider implementation
// that uses the Google Cloud SDK for Go to deploy containerized applications.
package gcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/cloudbuild/apiv1/v2"
	"cloud.google.com/go/iam/apiv1/iampb"
	"cloud.google.com/go/logging/logadmin"
	run "cloud.google.com/go/run/apiv2"
	"cloud.google.com/go/run/apiv2/runpb"
	"cloud.google.com/go/storage"
	"google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/serviceusage/v1"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/jvreagan/cloud-deploy/pkg/manifest"
	"github.com/jvreagan/cloud-deploy/pkg/registry"
	"github.com/jvreagan/cloud-deploy/pkg/types"
)

// Provider implements the provider.Provider interface for Google Cloud Run.
type Provider struct {
	buildClient     *cloudbuild.Client
	runClient       *run.ServicesClient
	revisionsClient *run.RevisionsClient
	storageClient   *storage.Client
	projectsClient  *cloudresourcemanager.Service
	billingClient   *cloudbilling.APIService
	usageClient     *serviceusage.Service
	loggingClient   *logadmin.Client
	projectID       string
	region          string
	publicAccess    bool
	billingAccount  string
	organizationID  string
}

// New creates a new GCP provider instance with the specified configuration and manifest.
// Credentials can be loaded from:
// 1. Vault (if credentials.source == "vault")
// 2. Environment variables (if credentials.source == "environment")
// 3. Manifest (if credentials contain service_account_key)
// 4. Default application credentials (fallback)
//
// The provider will automatically:
// - Create the project if it doesn't exist
// - Link the billing account
// - Enable required APIs
// - Deploy the application
func New(ctx context.Context, config *manifest.ProviderConfig, m *manifest.Manifest) (*Provider, error) {
	projectID := config.ProjectID
	if projectID == "" {
		return nil, fmt.Errorf("provider.project_id is required in manifest for GCP deployments")
	}

	// Determine public access setting (default: true)
	publicAccess := true
	if config.PublicAccess != nil {
		publicAccess = *config.PublicAccess
	}

	fmt.Printf("Initializing GCP provider for project: %s\n", projectID)

	// Check if credentials should be loaded from Vault
	var credOption option.ClientOption
	var err error

	if config.Credentials != nil && config.Credentials.Source == "vault" {
		fmt.Println("Loading GCP credentials from Vault...")

		// Get credentials from Vault using manifest helper
		vaultCreds, err := m.GetCloudCredentials(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load GCP credentials from Vault: %w", err)
		}

		if vaultCreds != nil && vaultCreds.GCP.ServiceAccountKey != "" {
			fmt.Println("✅ Successfully loaded GCP credentials from Vault")
			credOption = option.WithCredentialsJSON([]byte(vaultCreds.GCP.ServiceAccountKey))
		}
	} else {
		// Load service account credentials from manifest (existing behavior)
		credOption, err = loadCredentials(config.Credentials)
		if err != nil {
			return nil, fmt.Errorf("failed to load credentials: %w", err)
		}
	}

	// Initialize Cloud Resource Manager client (for project management)
	projectsClient, err := cloudresourcemanager.NewService(ctx, credOption)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloud Resource Manager client: %w", err)
	}

	// Initialize Cloud Billing client
	billingClient, err := cloudbilling.NewService(ctx, credOption)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloud Billing client: %w", err)
	}

	// Initialize Service Usage client (for enabling APIs)
	usageClient, err := serviceusage.NewService(ctx, credOption)
	if err != nil {
		return nil, fmt.Errorf("failed to create Service Usage client: %w", err)
	}

	// Initialize Cloud Build client
	buildClient, err := cloudbuild.NewClient(ctx, credOption)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloud Build client: %w", err)
	}

	// Initialize Cloud Run client
	runClient, err := run.NewServicesClient(ctx, credOption)
	if err != nil {
		buildClient.Close()
		return nil, fmt.Errorf("failed to create Cloud Run client: %w", err)
	}

	// Initialize Cloud Run Revisions client
	revisionsClient, err := run.NewRevisionsClient(ctx, credOption)
	if err != nil {
		buildClient.Close()
		runClient.Close()
		return nil, fmt.Errorf("failed to create Cloud Run Revisions client: %w", err)
	}

	// Initialize Cloud Storage client
	storageClient, err := storage.NewClient(ctx, credOption)
	if err != nil {
		buildClient.Close()
		runClient.Close()
		revisionsClient.Close()
		return nil, fmt.Errorf("failed to create Storage client: %w", err)
	}

	// Initialize Cloud Logging client (will be configured after project is ready)
	loggingClient, err := logadmin.NewClient(ctx, projectID, credOption)
	if err != nil {
		buildClient.Close()
		runClient.Close()
		storageClient.Close()
		return nil, fmt.Errorf("failed to create Logging client: %w", err)
	}

	provider := &Provider{
		buildClient:     buildClient,
		runClient:       runClient,
		revisionsClient: revisionsClient,
		storageClient:   storageClient,
		projectsClient:  projectsClient,
		billingClient:   billingClient,
		usageClient:     usageClient,
		loggingClient:   loggingClient,
		projectID:       projectID,
		region:          config.Region,
		publicAccess:    publicAccess,
		billingAccount:  config.BillingAccountID,
		organizationID:  config.OrganizationID,
	}

	// Ensure project exists and is properly configured
	if err := provider.ensureProject(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure project: %w", err)
	}

	// Link billing account
	if err := provider.ensureBillingLinked(ctx); err != nil {
		return nil, fmt.Errorf("failed to link billing account: %w", err)
	}

	// Enable required APIs
	if err := provider.ensureAPIsEnabled(ctx); err != nil {
		return nil, fmt.Errorf("failed to enable required APIs: %w", err)
	}

	fmt.Println("GCP provider initialized successfully")
	return provider, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "gcp"
}

// Deploy deploys an application to Google Cloud Run.
func (p *Provider) Deploy(ctx context.Context, m *manifest.Manifest) (*types.DeploymentResult, error) {
	fmt.Println("Starting Google Cloud Run deployment...")

	// Step 0.5: Fetch secrets from Vault if configured
	if m.Vault != nil && len(m.Secrets) > 0 {
		vaultSecrets, err := m.FetchVaultSecrets(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch vault secrets: %w", err)
		}

		// Merge Vault secrets with environment variables
		// Vault secrets take precedence over manifest environment variables
		if m.EnvironmentVariables == nil {
			m.EnvironmentVariables = make(map[string]string)
		}
		for key, value := range vaultSecrets {
			m.EnvironmentVariables[key] = value
		}
	}

	// Step 1: Push image to GCR (Artifact Registry)
	fmt.Println("\n=== Distributing image to GCR ===")

	// Get credentials JSON for GCR authentication
	var credsJSON string
	if m.Provider.Credentials != nil {
		if m.Provider.Credentials.ServiceAccountKeyJSON != "" {
			credsJSON = m.Provider.Credentials.ServiceAccountKeyJSON
		}
	}

	repositoryName := m.Application.Name
	gcrRegistry, err := registry.NewGCRRegistry(p.projectID, p.region, repositoryName, "latest", credsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCR registry handler: %w", err)
	}

	// Use Distributor to push image to registry
	distributor := registry.NewDistributor(m.Image)
	distributor.AddRegistry(gcrRegistry)

	imageURIs, err := distributor.Distribute(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to distribute image to GCR: %w", err)
	}

	imageURI := imageURIs[gcrRegistry.GetRegistryURL()]
	fmt.Printf("Successfully pushed image to GCR: %s\n", imageURI)

	// Step 2: Deploy to Cloud Run
	serviceName := m.Environment.Name
	if err := p.deployService(ctx, m, serviceName, imageURI); err != nil {
		return nil, fmt.Errorf("failed to deploy service %s with image %s: %w", serviceName, imageURI, err)
	}

	// Step 3: Configure Cloud Logging if enabled
	if m.Monitoring.CloudWatchLogs != nil && m.Monitoring.CloudWatchLogs.Enabled {
		if err := p.configureLogging(ctx, m); err != nil {
			fmt.Printf("Warning: failed to configure Cloud Logging: %v\n", err)
			// Don't fail deployment if logging configuration fails
		}
	}

	// Step 4: Wait for service to be ready
	fmt.Println("Waiting for service to be ready...")
	url, err := p.waitForService(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("service deployment failed for %s: %w", serviceName, err)
	}

	return &types.DeploymentResult{
		ApplicationName: m.Application.Name,
		EnvironmentName: m.Environment.Name,
		URL:             url,
		Status:          "Ready",
		Message:         "Deployment successful",
	}, nil
}

// Destroy removes a Google Cloud Run service.
func (p *Provider) Destroy(ctx context.Context, m *manifest.Manifest) error {
	serviceName := m.Environment.Name
	parent := fmt.Sprintf("projects/%s/locations/%s/services/%s", p.projectID, p.region, serviceName)

	fmt.Printf("Deleting Cloud Run service: %s\n", serviceName)

	req := &runpb.DeleteServiceRequest{
		Name: parent,
	}

	op, err := p.runClient.DeleteService(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	// Wait for deletion to complete
	if _, err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for service deletion: %w", err)
	}

	fmt.Println("Service deleted successfully")
	return nil
}

// Stop stops the Cloud Run service but preserves container images for fast redeployment.
// For Cloud Run (serverless), this deletes the service but keeps the container images
// in Artifact Registry. Cloud Run automatically scales to zero when idle, so stopping
// primarily helps clean up unused services while preserving build artifacts.
func (p *Provider) Stop(ctx context.Context, m *manifest.Manifest) error {
	serviceName := m.Environment.Name
	parent := fmt.Sprintf("projects/%s/locations/%s/services/%s", p.projectID, p.region, serviceName)

	fmt.Printf("Stopping Cloud Run service: %s\n", serviceName)
	fmt.Println("This will delete the service but preserve container images for fast restart.")

	req := &runpb.DeleteServiceRequest{
		Name: parent,
	}

	op, err := p.runClient.DeleteService(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	// Wait for deletion to complete
	if _, err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for service deletion: %w", err)
	}

	fmt.Println("Service stopped successfully")
	fmt.Println("Container images are preserved in Artifact Registry")
	fmt.Println("Run 'cloud-deploy -command deploy' to restart")
	return nil
}

// Status retrieves the current status of a Google Cloud Run deployment.
func (p *Provider) Status(ctx context.Context, m *manifest.Manifest) (*types.DeploymentStatus, error) {
	serviceName := m.Environment.Name
	parent := fmt.Sprintf("projects/%s/locations/%s/services/%s", p.projectID, p.region, serviceName)

	req := &runpb.GetServiceRequest{
		Name: parent,
	}

	service, err := p.runClient.GetService(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	status := "Unknown"
	health := "Unknown"

	// Check if service is ready
	if service.TerminalCondition != nil {
		status = service.TerminalCondition.State.String()
		if service.TerminalCondition.State == runpb.Condition_CONDITION_SUCCEEDED {
			health = "Healthy"
		} else {
			health = "Unhealthy"
		}
	}

	url := service.Uri
	lastUpdated := ""
	if service.UpdateTime != nil {
		lastUpdated = service.UpdateTime.AsTime().Format(time.RFC3339)
	}

	return &types.DeploymentStatus{
		ApplicationName: m.Application.Name,
		EnvironmentName: m.Environment.Name,
		Status:          status,
		Health:          health,
		URL:             url,
		LastUpdated:     lastUpdated,
	}, nil
}

// deployService deploys or updates a Cloud Run service with resource limits and scaling configuration.
func (p *Provider) deployService(ctx context.Context, m *manifest.Manifest, serviceName, imageTag string) error {
	parent := fmt.Sprintf("projects/%s/locations/%s", p.projectID, p.region)
	serviceFullName := fmt.Sprintf("%s/services/%s", parent, serviceName)

	// Check if service exists
	getReq := &runpb.GetServiceRequest{
		Name: serviceFullName,
	}
	existingService, err := p.runClient.GetService(ctx, getReq)
	serviceExists := err == nil

	// Build environment variables
	envVars := make([]*runpb.EnvVar, 0, len(m.EnvironmentVariables))
	for key, value := range m.EnvironmentVariables {
		envVars = append(envVars, &runpb.EnvVar{
			Name: key,
			Values: &runpb.EnvVar_Value{
				Value: value,
			},
		})
	}

	// Build container resources from manifest configuration
	container := &runpb.Container{
		Image: imageTag,
		Env:   envVars,
	}

	// Apply Cloud Run configuration if specified
	if m.CloudRun != nil {
		// Set CPU and memory limits
		resources := &runpb.ResourceRequirements{
			Limits: make(map[string]string),
		}
		if m.CloudRun.CPU != "" {
			resources.Limits["cpu"] = m.CloudRun.CPU
		} else {
			resources.Limits["cpu"] = "1" // Default
		}
		if m.CloudRun.Memory != "" {
			resources.Limits["memory"] = m.CloudRun.Memory
		} else {
			resources.Limits["memory"] = "512Mi" // Default
		}
		container.Resources = resources
	}

	// Create revision template
	revisionTemplate := &runpb.RevisionTemplate{
		Containers: []*runpb.Container{container},
	}

	// Apply scaling configuration
	if m.CloudRun != nil {
		scaling := &runpb.RevisionScaling{}
		if m.CloudRun.MinInstances > 0 {
			scaling.MinInstanceCount = m.CloudRun.MinInstances
		}
		if m.CloudRun.MaxInstances > 0 {
			scaling.MaxInstanceCount = m.CloudRun.MaxInstances
		}
		revisionTemplate.Scaling = scaling

		// Set max concurrency
		if m.CloudRun.MaxConcurrency > 0 {
			revisionTemplate.MaxInstanceRequestConcurrency = m.CloudRun.MaxConcurrency
		}

		// Set timeout
		if m.CloudRun.TimeoutSeconds > 0 {
			revisionTemplate.Timeout = durationpb.New(time.Duration(m.CloudRun.TimeoutSeconds) * time.Second)
		}
	}

	// Create service specification
	service := &runpb.Service{
		Template: revisionTemplate,
		Ingress:  runpb.IngressTraffic_INGRESS_TRAFFIC_ALL,
	}

	if serviceExists {
		fmt.Printf("Updating existing service: %s\n", serviceName)

		// For updates, set the name
		service.Name = serviceFullName

		// Preserve existing settings
		service.Template = existingService.Template
		service.Template.Containers[0].Image = imageTag
		service.Template.Containers[0].Env = envVars

		req := &runpb.UpdateServiceRequest{
			Service: service,
		}

		op, err := p.runClient.UpdateService(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to update service: %w", err)
		}

		_, err = op.Wait(ctx)
		if err != nil {
			return fmt.Errorf("failed to wait for service update: %w", err)
		}
	} else {
		fmt.Printf("Creating new service: %s\n", serviceName)

		req := &runpb.CreateServiceRequest{
			Parent:    parent,
			Service:   service,
			ServiceId: serviceName,
		}

		op, err := p.runClient.CreateService(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to create service: %w", err)
		}

		_, err = op.Wait(ctx)
		if err != nil {
			return fmt.Errorf("failed to wait for service creation: %w", err)
		}
	}

	// Make service publicly accessible (optional, based on manifest settings)
	if err := p.setServiceIAMPolicy(ctx, serviceFullName); err != nil {
		return fmt.Errorf("failed to set IAM policy: %w", err)
	}

	fmt.Println("Service deployed successfully")
	return nil
}

// setServiceIAMPolicy configures the Cloud Run service's IAM policy.
// If publicAccess is true, makes the service publicly accessible (unauthenticated access).
func (p *Provider) setServiceIAMPolicy(ctx context.Context, serviceName string) error {
	if !p.publicAccess {
		fmt.Println("Public access disabled - service requires authentication")
		return nil
	}

	fmt.Println("Configuring service for public access...")

	// Get current IAM policy
	getPolicyReq := &iampb.GetIamPolicyRequest{
		Resource: serviceName,
	}

	policy, err := p.runClient.GetIamPolicy(ctx, getPolicyReq)
	if err != nil {
		return fmt.Errorf("failed to get IAM policy: %w", err)
	}

	// Add binding for allUsers to invoke the service
	binding := &iampb.Binding{
		Role:    "roles/run.invoker",
		Members: []string{"allUsers"},
	}

	// Check if binding already exists
	bindingExists := false
	for _, b := range policy.Bindings {
		if b.Role == "roles/run.invoker" {
			for _, member := range b.Members {
				if member == "allUsers" {
					bindingExists = true
					break
				}
			}
			if !bindingExists {
				b.Members = append(b.Members, "allUsers")
				bindingExists = true
			}
			break
		}
	}

	if !bindingExists {
		policy.Bindings = append(policy.Bindings, binding)
	}

	// Set the updated policy
	setPolicyReq := &iampb.SetIamPolicyRequest{
		Resource: serviceName,
		Policy:   policy,
	}

	_, err = p.runClient.SetIamPolicy(ctx, setPolicyReq)
	if err != nil {
		return fmt.Errorf("failed to set IAM policy: %w", err)
	}

	fmt.Println("Service configured for public access")
	return nil
}

// getServiceURL retrieves the public URL of a Cloud Run service.
func (p *Provider) getServiceURL(ctx context.Context, serviceName string) (string, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s/services/%s", p.projectID, p.region, serviceName)

	req := &runpb.GetServiceRequest{
		Name: parent,
	}

	service, err := p.runClient.GetService(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to get service: %w", err)
	}

	return service.Uri, nil
}

// loadCredentials loads GCP service account credentials from the manifest.
func loadCredentials(creds *manifest.CredentialsConfig) (option.ClientOption, error) {
	if creds == nil {
		return nil, fmt.Errorf("credentials are required for GCP deployments")
	}

	// Option 1: Load from file path
	if creds.ServiceAccountKeyPath != "" {
		fmt.Printf("Loading credentials from: %s\n", creds.ServiceAccountKeyPath)
		return option.WithCredentialsFile(creds.ServiceAccountKeyPath), nil
	}

	// Option 2: Load from JSON string
	if creds.ServiceAccountKeyJSON != "" {
		fmt.Println("Loading credentials from manifest JSON")
		return option.WithCredentialsJSON([]byte(creds.ServiceAccountKeyJSON)), nil
	}

	return nil, fmt.Errorf("either service_account_key_path or service_account_key_json is required")
}

// ensureProject creates the GCP project if it doesn't exist.
func (p *Provider) ensureProject(ctx context.Context) error {
	fmt.Printf("Checking if project exists: %s\n", p.projectID)

	// Check if project exists
	project, err := p.projectsClient.Projects.Get(p.projectID).Context(ctx).Do()
	if err == nil && project != nil {
		fmt.Printf("Project already exists: %s (state: %s)\n", p.projectID, project.LifecycleState)
		return nil
	}

	// Project doesn't exist, create it
	fmt.Printf("Creating project: %s\n", p.projectID)

	newProject := &cloudresourcemanager.Project{
		ProjectId: p.projectID,
		Name:      p.projectID,
	}

	// If organization ID is specified, create under organization
	if p.organizationID != "" {
		newProject.Parent = &cloudresourcemanager.ResourceId{
			Type: "organization",
			Id:   p.organizationID,
		}
	}

	op, err := p.projectsClient.Projects.Create(newProject).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	// Wait for project creation to complete with polling
	fmt.Println("Waiting for project creation to complete...")
	return p.waitForProjectCreation(ctx, op.Name)
}

// ensureBillingLinked links the billing account to the project.
func (p *Provider) ensureBillingLinked(ctx context.Context) error {
	fmt.Println("Checking billing account linkage...")

	projectName := fmt.Sprintf("projects/%s", p.projectID)

	// Check current billing info
	billingInfo, err := p.billingClient.Projects.GetBillingInfo(projectName).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to get billing info: %w", err)
	}

	// Check if billing is already enabled
	if billingInfo.BillingEnabled {
		fmt.Printf("Billing already enabled for project (account: %s)\n", billingInfo.BillingAccountName)
		return nil
	}

	// Link billing account
	fmt.Printf("Linking billing account: %s\n", p.billingAccount)

	billingAccountName := fmt.Sprintf("billingAccounts/%s", p.billingAccount)
	updateReq := &cloudbilling.ProjectBillingInfo{
		BillingAccountName: billingAccountName,
	}

	_, err = p.billingClient.Projects.UpdateBillingInfo(projectName, updateReq).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to link billing account: %w", err)
	}

	fmt.Println("Billing account linked successfully")
	return nil
}

// ensureAPIsEnabled enables required APIs for Cloud Run deployment.
func (p *Provider) ensureAPIsEnabled(ctx context.Context) error {
	requiredAPIs := []string{
		"cloudbuild.googleapis.com",
		"run.googleapis.com",
		"storage.googleapis.com",
		"containerregistry.googleapis.com",
		"serviceusage.googleapis.com",
	}

	fmt.Println("Enabling required GCP APIs...")

	for _, api := range requiredAPIs {
		serviceName := fmt.Sprintf("projects/%s/services/%s", p.projectID, api)

		// Check if API is already enabled
		service, err := p.usageClient.Services.Get(serviceName).Context(ctx).Do()
		if err == nil && service.State == "ENABLED" {
			fmt.Printf("  ✓ %s (already enabled)\n", api)
			continue
		}

		// Enable the API
		fmt.Printf("  → Enabling %s...\n", api)
		enableReq := &serviceusage.EnableServiceRequest{}
		op, err := p.usageClient.Services.Enable(serviceName, enableReq).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("failed to enable API %s: %w", api, err)
		}

		// Wait for API enablement to complete with polling
		if !op.Done {
			if err := p.waitForAPIEnablement(ctx, op.Name, api); err != nil {
				return fmt.Errorf("failed to wait for API %s enablement: %w", api, err)
			}
		}

		fmt.Printf("  ✓ %s (enabled)\n", api)
	}

	fmt.Println("All required APIs enabled")
	return nil
}

// waitForProjectCreation polls the project creation operation until it completes.
func (p *Provider) waitForProjectCreation(ctx context.Context, operationName string) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(3 * time.Minute)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for project %s creation (3 minutes elapsed)", p.projectID)
		case <-ticker.C:
			op, err := p.projectsClient.Operations.Get(operationName).Context(ctx).Do()
			if err != nil {
				return fmt.Errorf("failed to check project creation status: %w", err)
			}

			if op.Done {
				if op.Error != nil {
					return fmt.Errorf("project creation failed: %s", op.Error.Message)
				}
				fmt.Printf("Project created successfully: %s\n", p.projectID)
				return nil
			}

			fmt.Println("  Still creating project...")
		}
	}
}

// waitForAPIEnablement polls the API enablement operation until it completes.
func (p *Provider) waitForAPIEnablement(ctx context.Context, operationName, apiName string) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for API %s enablement (5 minutes elapsed)", apiName)
		case <-ticker.C:
			op, err := p.usageClient.Operations.Get(operationName).Context(ctx).Do()
			if err != nil {
				return fmt.Errorf("failed to check API enablement status: %w", err)
			}

			if op.Done {
				if op.Error != nil {
					return fmt.Errorf("API enablement failed: %s", op.Error.Message)
				}
				fmt.Printf("    API %s enabled successfully\n", apiName)
				return nil
			}

			fmt.Printf("    Waiting for %s to be enabled...\n", apiName)
		}
	}
}

// waitForService waits for the Cloud Run service to become ready and returns its URL.
func (p *Provider) waitForService(ctx context.Context, serviceName string) (string, error) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	timeout := time.After(10 * time.Minute)
	serviceFullName := fmt.Sprintf("projects/%s/locations/%s/services/%s", p.projectID, p.region, serviceName)

	for {
		select {
		case <-timeout:
			return "", fmt.Errorf("timeout waiting for service %s to be ready (10 minutes elapsed)", serviceName)
		case <-ticker.C:
			req := &runpb.GetServiceRequest{
				Name: serviceFullName,
			}

			service, err := p.runClient.GetService(ctx, req)
			if err != nil {
				return "", fmt.Errorf("failed to get service status: %w", err)
			}

			// Check terminal condition
			if service.TerminalCondition != nil {
				status := service.TerminalCondition.State.String()
				fmt.Printf("Service status: %s\n", status)

				if service.TerminalCondition.State == runpb.Condition_CONDITION_SUCCEEDED {
					fmt.Println("Service is ready!")
					return service.Uri, nil
				}

				if service.TerminalCondition.State == runpb.Condition_CONDITION_FAILED {
					message := "unknown error"
					if service.TerminalCondition.Message != "" {
						message = service.TerminalCondition.Message
					}
					return "", fmt.Errorf("service deployment failed: %s", message)
				}
			}

			fmt.Println("  Service is still deploying...")
		}
	}
}

// configureLogging sets up Cloud Logging for the Cloud Run service.
func (p *Provider) configureLogging(ctx context.Context, m *manifest.Manifest) error {
	fmt.Println("Configuring Cloud Logging...")

	// Cloud Run automatically sends logs to Cloud Logging
	// We just need to configure the log retention if specified
	if m.Monitoring.CloudWatchLogs != nil && m.Monitoring.CloudWatchLogs.RetentionDays > 0 {
		// Note: Log retention is set at the bucket level in Cloud Logging
		// For now, we'll just log that logging is configured
		fmt.Printf("Cloud Logging configured for service %s\n", m.Environment.Name)
		fmt.Printf("Logs will be available at: https://console.cloud.google.com/logs/query;query=resource.type%%3D%%22cloud_run_revision%%22%%0Aresource.labels.service_name%%3D%%22%s%%22?project=%s\n",
			m.Environment.Name, p.projectID)

		// If retention is specified, inform the user they need to configure it in Cloud Console
		if m.Monitoring.CloudWatchLogs.RetentionDays > 0 {
			fmt.Printf("Note: To set log retention to %d days, configure it in Cloud Logging settings:\n", m.Monitoring.CloudWatchLogs.RetentionDays)
			fmt.Printf("  https://console.cloud.google.com/logs/storage?project=%s\n", p.projectID)
		}
	}

	// Log the direct log viewing URL
	fmt.Printf("View logs: gcloud logging read 'resource.type=cloud_run_revision AND resource.labels.service_name=%s' --limit 50 --project=%s\n",
		m.Environment.Name, p.projectID)

	return nil
}

// Rollback rolls back the GCP Cloud Run service to the previous revision.
func (p *Provider) Rollback(ctx context.Context, m *manifest.Manifest) (*types.DeploymentResult, error) {
	fmt.Println("Starting Google Cloud Run rollback...")

	serviceName := m.Environment.Name
	parent := fmt.Sprintf("projects/%s/locations/%s/services/%s", p.projectID, p.region, serviceName)

	// Step 1: Get the current service to find active revision
	getReq := &runpb.GetServiceRequest{
		Name: parent,
	}

	service, err := p.runClient.GetService(ctx, getReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	// Find the current active revision
	var currentRevision string
	if service.Traffic != nil && len(service.Traffic) > 0 {
		// Find the revision serving 100% traffic
		for _, traffic := range service.Traffic {
			if traffic.Percent == 100 {
				currentRevision = traffic.Revision
				break
			}
		}
	}

	if currentRevision == "" {
		return nil, fmt.Errorf("could not determine current active revision")
	}

	fmt.Printf("Current revision: %s\n", currentRevision)

	// Step 2: List all revisions for this service
	listReq := &runpb.ListRevisionsRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", p.projectID, p.region),
	}

	revisions := p.revisionsClient.ListRevisions(ctx, listReq)

	// Filter revisions for this service and sort by creation time
	var serviceRevisions []*runpb.Revision
	for {
		revision, err := revisions.Next()
		if err != nil {
			break
		}

		// Check if this revision belongs to our service
		if strings.Contains(revision.Name, serviceName) {
			serviceRevisions = append(serviceRevisions, revision)
		}
	}

	if len(serviceRevisions) < 2 {
		return nil, fmt.Errorf("no previous revision available to rollback to (only %d revision(s) exist)", len(serviceRevisions))
	}

	// Step 3: Find the previous revision (most recent one before current)
	var previousRevision *runpb.Revision
	var currentRevisionTime *time.Time

	// Find the current revision's creation time
	for _, rev := range serviceRevisions {
		if strings.Contains(rev.Name, currentRevision) {
			if rev.CreateTime != nil {
				t := rev.CreateTime.AsTime()
				currentRevisionTime = &t
			}
			break
		}
	}

	if currentRevisionTime == nil {
		return nil, fmt.Errorf("could not find current revision creation time")
	}

	// Find the most recent revision created before the current one
	for _, rev := range serviceRevisions {
		// Skip current revision
		if strings.Contains(rev.Name, currentRevision) {
			continue
		}

		if rev.CreateTime != nil {
			revTime := rev.CreateTime.AsTime()
			if revTime.Before(*currentRevisionTime) {
				if previousRevision == nil || revTime.After(previousRevision.CreateTime.AsTime()) {
					previousRevision = rev
				}
			}
		}
	}

	if previousRevision == nil {
		return nil, fmt.Errorf("no previous revision found to rollback to")
	}

	// Extract just the revision name (last part of the full name)
	prevRevisionName := previousRevision.Name[strings.LastIndex(previousRevision.Name, "/")+1:]
	fmt.Printf("Rolling back to previous revision: %s\n", prevRevisionName)

	// Step 4: Update service traffic to route to previous revision
	service.Traffic = []*runpb.TrafficTarget{
		{
			Type:     runpb.TrafficTargetAllocationType_TRAFFIC_TARGET_ALLOCATION_TYPE_REVISION,
			Revision: prevRevisionName,
			Percent:  100,
		},
	}

	updateReq := &runpb.UpdateServiceRequest{
		Service: service,
	}

	op, err := p.runClient.UpdateService(ctx, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to rollback service: %w", err)
	}

	// Wait for rollback to complete
	fmt.Println("Waiting for rollback to complete...")
	_, err = op.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("rollback failed: %w", err)
	}

	// Get the updated service to retrieve the URL
	service, err = p.runClient.GetService(ctx, getReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get service after rollback: %w", err)
	}

	return &types.DeploymentResult{
		ApplicationName: m.Application.Name,
		EnvironmentName: m.Environment.Name,
		URL:             service.Uri,
		Status:          "Ready",
		Message:         fmt.Sprintf("Rolled back to revision %s", prevRevisionName),
	}, nil
}
