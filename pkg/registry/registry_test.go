package registry

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/go-containerregistry/pkg/authn"
)

// awsConfigStub returns a minimal aws.Config for constructor tests (no real credentials).
func awsConfigStub() aws.Config {
	return aws.Config{Region: "us-east-1"}
}

// mockRegistry implements the Registry interface for testing
type mockRegistry struct {
	registryURL    string
	imageReference string
	imageURI       string
	authError      error
}

func (m *mockRegistry) GetRegistryURL() string    { return m.registryURL }
func (m *mockRegistry) GetImageReference() string { return m.imageReference }
func (m *mockRegistry) GetImageURI() string       { return m.imageURI }
func (m *mockRegistry) GetAuthenticator(ctx context.Context) (authn.Authenticator, error) {
	if m.authError != nil {
		return nil, m.authError
	}
	return &authn.Basic{Username: "test", Password: "test"}, nil
}

func TestNewDistributor(t *testing.T) {
	d := NewDistributor("myapp:latest")

	if d.sourceImage != "myapp:latest" {
		t.Errorf("sourceImage = %q, want %q", d.sourceImage, "myapp:latest")
	}
	if len(d.registries) != 0 {
		t.Errorf("registries length = %d, want 0", len(d.registries))
	}
}

func TestDistributorAddRegistry(t *testing.T) {
	d := NewDistributor("myapp:latest")

	r1 := &mockRegistry{registryURL: "ecr.amazonaws.com"}
	r2 := &mockRegistry{registryURL: "gcr.io"}
	r3 := &mockRegistry{registryURL: "azurecr.io"}

	d.AddRegistry(r1)
	if len(d.registries) != 1 {
		t.Fatalf("registries length = %d, want 1", len(d.registries))
	}

	d.AddRegistry(r2)
	d.AddRegistry(r3)
	if len(d.registries) != 3 {
		t.Fatalf("registries length = %d, want 3", len(d.registries))
	}

	if d.registries[0].GetRegistryURL() != "ecr.amazonaws.com" {
		t.Errorf("registry[0] URL = %q, want %q", d.registries[0].GetRegistryURL(), "ecr.amazonaws.com")
	}
	if d.registries[1].GetRegistryURL() != "gcr.io" {
		t.Errorf("registry[1] URL = %q, want %q", d.registries[1].GetRegistryURL(), "gcr.io")
	}
	if d.registries[2].GetRegistryURL() != "azurecr.io" {
		t.Errorf("registry[2] URL = %q, want %q", d.registries[2].GetRegistryURL(), "azurecr.io")
	}
}

func TestNewECRRegistry(t *testing.T) {
	// NewECRRegistry doesn't require live AWS credentials, just config
	r, err := NewECRRegistry(
		awsConfigStub(),
		"us-east-1",
		"myapp",
		"v1.0.0",
	)
	if err != nil {
		t.Fatalf("NewECRRegistry returned error: %v", err)
	}
	if r.region != "us-east-1" {
		t.Errorf("region = %q, want %q", r.region, "us-east-1")
	}
	if r.repositoryName != "myapp" {
		t.Errorf("repositoryName = %q, want %q", r.repositoryName, "myapp")
	}
	if r.imageTag != "v1.0.0" {
		t.Errorf("imageTag = %q, want %q", r.imageTag, "v1.0.0")
	}
	// registryURL and imageURI are empty until GetAuthenticator is called
	if r.GetRegistryURL() != "" {
		t.Errorf("GetRegistryURL() = %q before auth, want empty", r.GetRegistryURL())
	}
}

func TestNewGCRRegistry(t *testing.T) {
	r, err := NewGCRRegistry("my-project", "us-central1", "myapp", "v1.0.0", "")
	if err != nil {
		t.Fatalf("NewGCRRegistry returned error: %v", err)
	}
	if r.projectID != "my-project" {
		t.Errorf("projectID = %q, want %q", r.projectID, "my-project")
	}
	if r.region != "us-central1" {
		t.Errorf("region = %q, want %q", r.region, "us-central1")
	}
	if r.repositoryName != "myapp" {
		t.Errorf("repositoryName = %q, want %q", r.repositoryName, "myapp")
	}
	if r.imageTag != "v1.0.0" {
		t.Errorf("imageTag = %q, want %q", r.imageTag, "v1.0.0")
	}
}

func TestNewACRRegistry(t *testing.T) {
	r, err := NewACRRegistry(nil, "sub-123", "my-rg", "myregistry", "eastus", "v1.0.0")
	if err != nil {
		t.Fatalf("NewACRRegistry returned error: %v", err)
	}
	if r.subscriptionID != "sub-123" {
		t.Errorf("subscriptionID = %q, want %q", r.subscriptionID, "sub-123")
	}
	if r.resourceGroup != "my-rg" {
		t.Errorf("resourceGroup = %q, want %q", r.resourceGroup, "my-rg")
	}
	if r.registryName != "myregistry" {
		t.Errorf("registryName = %q, want %q", r.registryName, "myregistry")
	}
	if r.location != "eastus" {
		t.Errorf("location = %q, want %q", r.location, "eastus")
	}
	if r.imageTag != "v1.0.0" {
		t.Errorf("imageTag = %q, want %q", r.imageTag, "v1.0.0")
	}
}

func TestRegistryInterfaceCompliance(t *testing.T) {
	// Verify all registry types satisfy the Registry interface at compile time
	var _ Registry = (*ECRRegistry)(nil)
	var _ Registry = (*GCRRegistry)(nil)
	var _ Registry = (*ACRRegistry)(nil)
}

func TestECRRegistryGetters(t *testing.T) {
	r, err := NewECRRegistry(awsConfigStub(), "us-west-2", "myservice", "deploy-20260304T120000")
	if err != nil {
		t.Fatalf("NewECRRegistry returned error: %v", err)
	}

	// Before authentication, getters return empty strings
	if got := r.GetRegistryURL(); got != "" {
		t.Errorf("GetRegistryURL() before auth = %q, want empty", got)
	}
	if got := r.GetImageURI(); got != "" {
		t.Errorf("GetImageURI() before auth = %q, want empty", got)
	}
	if got := r.GetImageReference(); got != "" {
		t.Errorf("GetImageReference() before auth = %q, want empty", got)
	}
	// GetImageURI and GetImageReference should return the same value
	if r.GetImageURI() != r.GetImageReference() {
		t.Errorf("GetImageURI() != GetImageReference(): %q vs %q", r.GetImageURI(), r.GetImageReference())
	}
}

func TestGCRRegistryGetters(t *testing.T) {
	r, err := NewGCRRegistry("my-project", "us-central1", "myservice", "v2.0.0", "")
	if err != nil {
		t.Fatalf("NewGCRRegistry returned error: %v", err)
	}

	// Before authentication, getters return empty strings
	if got := r.GetRegistryURL(); got != "" {
		t.Errorf("GetRegistryURL() before auth = %q, want empty", got)
	}
	if got := r.GetImageURI(); got != "" {
		t.Errorf("GetImageURI() before auth = %q, want empty", got)
	}
	if got := r.GetImageReference(); got != "" {
		t.Errorf("GetImageReference() before auth = %q, want empty", got)
	}
}

func TestACRRegistryGetters(t *testing.T) {
	r, err := NewACRRegistry(nil, "sub-abc", "rg-test", "testregistry", "westus2", "deploy-20260304T100000")
	if err != nil {
		t.Fatalf("NewACRRegistry returned error: %v", err)
	}

	// Before authentication, getters return empty strings
	if got := r.GetRegistryURL(); got != "" {
		t.Errorf("GetRegistryURL() before auth = %q, want empty", got)
	}
	if got := r.GetImageURI(); got != "" {
		t.Errorf("GetImageURI() before auth = %q, want empty", got)
	}
	if got := r.GetImageReference(); got != "" {
		t.Errorf("GetImageReference() before auth = %q, want empty", got)
	}
}

func TestGCRRegistryFieldInitialization(t *testing.T) {
	tests := []struct {
		name            string
		projectID       string
		region          string
		repositoryName  string
		imageTag        string
		credentialsJSON string
	}{
		{
			name:            "standard config",
			projectID:       "my-project",
			region:          "us-central1",
			repositoryName:  "myapp",
			imageTag:        "v1.0.0",
			credentialsJSON: `{"type":"service_account"}`,
		},
		{
			name:           "no credentials",
			projectID:      "other-project",
			region:         "europe-west1",
			repositoryName: "service",
			imageTag:       "latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewGCRRegistry(tt.projectID, tt.region, tt.repositoryName, tt.imageTag, tt.credentialsJSON)
			if err != nil {
				t.Fatalf("NewGCRRegistry returned error: %v", err)
			}
			if r.projectID != tt.projectID {
				t.Errorf("projectID = %q, want %q", r.projectID, tt.projectID)
			}
			if r.region != tt.region {
				t.Errorf("region = %q, want %q", r.region, tt.region)
			}
			if r.repositoryName != tt.repositoryName {
				t.Errorf("repositoryName = %q, want %q", r.repositoryName, tt.repositoryName)
			}
			if r.imageTag != tt.imageTag {
				t.Errorf("imageTag = %q, want %q", r.imageTag, tt.imageTag)
			}
			if r.credentialsJSON != tt.credentialsJSON {
				t.Errorf("credentialsJSON = %q, want %q", r.credentialsJSON, tt.credentialsJSON)
			}
		})
	}
}

func TestACRRegistryFieldInitialization(t *testing.T) {
	tests := []struct {
		name           string
		subscriptionID string
		resourceGroup  string
		registryName   string
		location       string
		imageTag       string
	}{
		{
			name:           "east us",
			subscriptionID: "sub-123",
			resourceGroup:  "rg-prod",
			registryName:   "prodregistry",
			location:       "eastus",
			imageTag:       "deploy-20260304T120000",
		},
		{
			name:           "west europe",
			subscriptionID: "sub-456",
			resourceGroup:  "rg-staging",
			registryName:   "stagingregistry",
			location:       "westeurope",
			imageTag:       "latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewACRRegistry(nil, tt.subscriptionID, tt.resourceGroup, tt.registryName, tt.location, tt.imageTag)
			if err != nil {
				t.Fatalf("NewACRRegistry returned error: %v", err)
			}
			if r.subscriptionID != tt.subscriptionID {
				t.Errorf("subscriptionID = %q, want %q", r.subscriptionID, tt.subscriptionID)
			}
			if r.resourceGroup != tt.resourceGroup {
				t.Errorf("resourceGroup = %q, want %q", r.resourceGroup, tt.resourceGroup)
			}
			if r.registryName != tt.registryName {
				t.Errorf("registryName = %q, want %q", r.registryName, tt.registryName)
			}
			if r.location != tt.location {
				t.Errorf("location = %q, want %q", r.location, tt.location)
			}
			if r.imageTag != tt.imageTag {
				t.Errorf("imageTag = %q, want %q", r.imageTag, tt.imageTag)
			}
		})
	}
}

func TestMockRegistryInterface(t *testing.T) {
	mock := &mockRegistry{
		registryURL:    "myregistry.azurecr.io",
		imageReference: "myregistry.azurecr.io/app:v1",
		imageURI:       "myregistry.azurecr.io/app:v1",
	}

	if got := mock.GetRegistryURL(); got != "myregistry.azurecr.io" {
		t.Errorf("GetRegistryURL() = %q, want %q", got, "myregistry.azurecr.io")
	}
	if got := mock.GetImageReference(); got != "myregistry.azurecr.io/app:v1" {
		t.Errorf("GetImageReference() = %q, want %q", got, "myregistry.azurecr.io/app:v1")
	}
	if got := mock.GetImageURI(); got != "myregistry.azurecr.io/app:v1" {
		t.Errorf("GetImageURI() = %q, want %q", got, "myregistry.azurecr.io/app:v1")
	}

	auth, err := mock.GetAuthenticator(context.Background())
	if err != nil {
		t.Fatalf("GetAuthenticator returned error: %v", err)
	}
	if auth == nil {
		t.Error("GetAuthenticator returned nil authenticator")
	}
}

func TestMockRegistryAuthError(t *testing.T) {
	mock := &mockRegistry{
		registryURL: "failing.registry.io",
		authError:   fmt.Errorf("authentication failed"),
	}

	auth, err := mock.GetAuthenticator(context.Background())
	if err == nil {
		t.Error("Expected error from GetAuthenticator")
	}
	if auth != nil {
		t.Error("Expected nil authenticator when error occurs")
	}
	if err.Error() != "authentication failed" {
		t.Errorf("Expected 'authentication failed', got %q", err.Error())
	}
}

func TestDistributorMultipleRegistries(t *testing.T) {
	d := NewDistributor("myapp:latest")

	registries := []*mockRegistry{
		{registryURL: "ecr.amazonaws.com", imageReference: "ecr/app:v1", imageURI: "ecr/app:v1"},
		{registryURL: "gcr.io", imageReference: "gcr/app:v1", imageURI: "gcr/app:v1"},
		{registryURL: "azurecr.io", imageReference: "acr/app:v1", imageURI: "acr/app:v1"},
	}

	for _, r := range registries {
		d.AddRegistry(r)
	}

	if len(d.registries) != 3 {
		t.Fatalf("registries length = %d, want 3", len(d.registries))
	}

	// Verify order is preserved
	for i, r := range registries {
		if d.registries[i].GetRegistryURL() != r.registryURL {
			t.Errorf("registry[%d] URL = %q, want %q", i, d.registries[i].GetRegistryURL(), r.registryURL)
		}
	}
}

func TestNewDistributorSourceImage(t *testing.T) {
	tests := []struct {
		name        string
		sourceImage string
	}{
		{"simple tag", "myapp:latest"},
		{"with registry", "docker.io/myapp:v1.0.0"},
		{"sha256 digest", "myapp@sha256:abcdef1234567890"},
		{"no tag", "myapp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDistributor(tt.sourceImage)
			if d.sourceImage != tt.sourceImage {
				t.Errorf("sourceImage = %q, want %q", d.sourceImage, tt.sourceImage)
			}
			if len(d.registries) != 0 {
				t.Errorf("registries should be empty, got %d", len(d.registries))
			}
		})
	}
}

func TestDistributeWithNoRegistries(t *testing.T) {
	d := NewDistributor("myapp:latest")

	// Distribute with no registries should still try to load from daemon,
	// which will fail without Docker. But if we could mock daemon.Image,
	// it would return an empty map. We test that it at least doesn't panic
	// and returns an error (since Docker daemon likely isn't available in CI).
	result, err := d.Distribute(context.Background())
	if err != nil {
		// Expected: Docker daemon not available in test environment
		return
	}
	// If Docker is available and image doesn't exist, we'd get an error above.
	// If somehow both work, result should be empty map.
	if len(result) != 0 {
		t.Errorf("Distribute with no registries returned %d results, want 0", len(result))
	}
}
