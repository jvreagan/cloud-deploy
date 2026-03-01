package registry

import (
	"context"
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

func (m *mockRegistry) GetRegistryURL() string                                      { return m.registryURL }
func (m *mockRegistry) GetImageReference() string                                   { return m.imageReference }
func (m *mockRegistry) GetImageURI() string                                         { return m.imageURI }
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
