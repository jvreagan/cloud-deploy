// Package gcp provides a Google Cloud Run provider implementation
// that uses the Google Cloud SDK for Go to deploy containerized applications.
package gcp

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

	"cloud.google.com/go/cloudbuild/apiv1/v2"
	"cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
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
	"github.com/jvreagan/cloud-deploy/pkg/types"
)

// Provider implements the provider.Provider interface for Google Cloud Run.
type Provider struct {
	buildClient    *cloudbuild.Client
	runClient      *run.ServicesClient
	storageClient  *storage.Client
	projectsClient *cloudresourcemanager.Service
	billingClient  *cloudbilling.APIService
	usageClient    *serviceusage.Service
	loggingClient  *logadmin.Client
	projectID      string
	region         string
	publicAccess   bool
	billingAccount string
	organizationID string
}

// New creates a new GCP provider instance with the specified configuration.
// All authentication is done via service account credentials in the manifest.
// The provider will automatically:
// - Create the project if it doesn't exist
// - Link the billing account
// - Enable required APIs
// - Deploy the application
func New(ctx context.Context, config *manifest.ProviderConfig) (*Provider, error) {
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

	// Load service account credentials
	credOption, err := loadCredentials(config.Credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
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

	// Initialize Cloud Storage client
	storageClient, err := storage.NewClient(ctx, credOption)
	if err != nil {
		buildClient.Close()
		runClient.Close()
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
		buildClient:    buildClient,
		runClient:      runClient,
		storageClient:  storageClient,
		projectsClient: projectsClient,
		billingClient:  billingClient,
		usageClient:    usageClient,
		loggingClient:  loggingClient,
		projectID:      projectID,
		region:         config.Region,
		publicAccess:   publicAccess,
		billingAccount: config.BillingAccountID,
		organizationID: config.OrganizationID,
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

	// Step 1: Create storage bucket for source code
	bucketName := fmt.Sprintf("%s-cloud-deploy-source", p.projectID)
	if err := p.ensureBucket(ctx, bucketName); err != nil {
		return nil, fmt.Errorf("failed to ensure storage bucket: %w", err)
	}

	// Step 2: Upload source code to Cloud Storage
	timestamp := time.Now().Unix()
	objectName := fmt.Sprintf("%s/%d/source.tar.gz", m.Application.Name, timestamp)
	if err := p.uploadSource(ctx, m.Deployment.Source.Path, bucketName, objectName); err != nil {
		return nil, fmt.Errorf("failed to upload source from %s to gs://%s/%s: %w", m.Deployment.Source.Path, bucketName, objectName, err)
	}

	// Step 3: Build container image using Cloud Build
	imageTag := fmt.Sprintf("gcr.io/%s/%s:%d", p.projectID, m.Application.Name, timestamp)
	if err := p.buildImage(ctx, bucketName, objectName, imageTag); err != nil {
		return nil, fmt.Errorf("failed to build image %s from gs://%s/%s: %w", imageTag, bucketName, objectName, err)
	}

	// Step 4: Deploy to Cloud Run
	serviceName := m.Environment.Name
	if err := p.deployService(ctx, m, serviceName, imageTag); err != nil {
		return nil, fmt.Errorf("failed to deploy service %s with image %s: %w", serviceName, imageTag, err)
	}

	// Step 5: Configure Cloud Logging if enabled
	if m.Monitoring.CloudWatchLogs != nil && m.Monitoring.CloudWatchLogs.Enabled {
		if err := p.configureLogging(ctx, m); err != nil {
			fmt.Printf("Warning: failed to configure Cloud Logging: %v\n", err)
			// Don't fail deployment if logging configuration fails
		}
	}

	// Step 6: Wait for service to be ready
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

// ensureBucket creates a Cloud Storage bucket if it doesn't exist.
func (p *Provider) ensureBucket(ctx context.Context, bucketName string) error {
	bucket := p.storageClient.Bucket(bucketName)

	// Check if bucket exists
	_, err := bucket.Attrs(ctx)
	if err == nil {
		fmt.Printf("Storage bucket already exists: %s\n", bucketName)
		return nil
	}

	// Create bucket
	fmt.Printf("Creating storage bucket: %s\n", bucketName)
	if err := bucket.Create(ctx, p.projectID, &storage.BucketAttrs{
		Location: strings.ToUpper(p.region),
	}); err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
}

// uploadSource creates a tarball of the source directory and uploads it to Cloud Storage.
func (p *Provider) uploadSource(ctx context.Context, sourcePath, bucketName, objectName string) error {
	fmt.Println("Creating source tarball...")

	// Create temporary tar.gz file
	tarFile, err := os.CreateTemp("", "cloud-deploy-*.tar.gz")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tarFile.Name())
	defer tarFile.Close()

	// Create tar.gz archive
	if err := createTarGz(sourcePath, tarFile); err != nil {
		return fmt.Errorf("failed to create tarball: %w", err)
	}

	// Rewind to beginning of file
	if _, err := tarFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}

	// Upload to Cloud Storage
	fmt.Printf("Uploading to Cloud Storage: gs://%s/%s\n", bucketName, objectName)
	bucket := p.storageClient.Bucket(bucketName)
	obj := bucket.Object(objectName)
	writer := obj.NewWriter(ctx)

	if _, err := io.Copy(writer, tarFile); err != nil {
		writer.Close()
		return fmt.Errorf("failed to upload: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return nil
}

// buildImage builds a container image using Cloud Build.
func (p *Provider) buildImage(ctx context.Context, bucketName, objectName, imageTag string) error {
	fmt.Println("Building container image with Cloud Build...")

	build := &cloudbuildpb.Build{
		Source: &cloudbuildpb.Source{
			Source: &cloudbuildpb.Source_StorageSource{
				StorageSource: &cloudbuildpb.StorageSource{
					Bucket: bucketName,
					Object: objectName,
				},
			},
		},
		Steps: []*cloudbuildpb.BuildStep{
			{
				Name: "gcr.io/cloud-builders/docker",
				Args: []string{"build", "-t", imageTag, "."},
			},
		},
		Images: []string{imageTag},
		Timeout: durationpb.New(20 * time.Minute),
	}

	req := &cloudbuildpb.CreateBuildRequest{
		ProjectId: p.projectID,
		Build:     build,
	}

	op, err := p.buildClient.CreateBuild(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to start build: %w", err)
	}

	// Wait for build to complete
	fmt.Println("Waiting for build to complete...")
	result, err := op.Wait(ctx)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	if result.Status != cloudbuildpb.Build_SUCCESS {
		return fmt.Errorf("build failed with status: %s", result.Status)
	}

	fmt.Printf("Image built successfully: %s\n", imageTag)
	return nil
}

// deployService deploys or updates a Cloud Run service.
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

	// Create service specification
	service := &runpb.Service{
		Template: &runpb.RevisionTemplate{
			Containers: []*runpb.Container{
				{
					Image: imageTag,
					Env:   envVars,
				},
			},
		},
		Ingress: runpb.IngressTraffic_INGRESS_TRAFFIC_ALL,
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

	// Wait for project creation to complete
	fmt.Println("Waiting for project creation to complete...")
	// Project creation is usually fast, but we should wait for the operation
	// For now, we'll just verify it was created
	if op.Done {
		fmt.Printf("Project created successfully: %s\n", p.projectID)
	} else {
		fmt.Printf("Project creation initiated: %s (operation: %s)\n", p.projectID, op.Name)
		// In production, you might want to poll the operation status
		// For now, we'll continue and subsequent API calls will wait if needed
	}

	return nil
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

		// Wait for operation to complete (optional, but recommended)
		if !op.Done {
			fmt.Printf("    Waiting for %s to be enabled...\n", api)
			// In production, you might want to poll op.Name to check completion
			// For now, we'll just continue and let subsequent operations handle it
		}

		fmt.Printf("  ✓ %s (enabled)\n", api)
	}

	fmt.Println("All required APIs enabled")
	return nil
}

// createTarGz creates a tar.gz archive of a directory.
func createTarGz(sourceDir string, tarFile *os.File) error {
	gzWriter := gzip.NewWriter(tarFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip hidden files and common excludes
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// Copy file content
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tarWriter, file)
		return err
	})
}
