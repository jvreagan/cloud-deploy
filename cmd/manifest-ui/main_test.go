package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGenerateManifestHandler_AWS(t *testing.T) {
	// Prepare test request
	reqData := ManifestRequest{
		Version: "1.0",
		Provider: ProviderConfig{
			Name:   "aws",
			Region: "us-east-2",
		},
		Application: ApplicationConfig{
			Name:        "test-app",
			Description: "Test application",
		},
		Environment: EnvironmentConfig{
			Name:  "test-env",
			Cname: "test-app",
		},
		Deployment: DeploymentConfig{
			Platform:      "docker",
			SolutionStack: "64bit Amazon Linux 2023 v4.7.2 running Docker",
			Source: SourceConfig{
				Type: "local",
				Path: ".",
			},
		},
		Instance: InstanceConfig{
			Type:            "t3.micro",
			EnvironmentType: "SingleInstance",
		},
		HealthCheck: HealthCheckConfig{
			Type: "enhanced",
			Path: "/health",
		},
		Monitoring: MonitoringConfig{
			EnhancedHealth:    true,
			CloudWatchMetrics: true,
		},
		EnvironmentVars: map[string]string{
			"NODE_ENV": "production",
		},
		Tags: map[string]string{
			"Environment": "test",
		},
	}

	// Convert to JSON
	reqBody, err := json.Marshal(reqData)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/api/generate", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Record response
	rr := httptest.NewRecorder()

	// Call handler
	generateManifestHandler(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		t.Logf("Response body: %s", rr.Body.String())
	}

	// Parse response
	var response map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check response fields
	if response["message"] != "Manifest generated successfully" {
		t.Errorf("Unexpected message: %s", response["message"])
	}

	if !strings.HasPrefix(response["filename"], "aws-manifest-") {
		t.Errorf("Unexpected filename format: %s", response["filename"])
	}

	// Verify file was created
	filePath := response["path"]
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("Manifest file was not created: %s", filePath)
	} else {
		// Read and validate YAML
		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read manifest file: %v", err)
		}

		var manifest map[string]interface{}
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			t.Fatalf("Generated manifest is not valid YAML: %v", err)
		}

		// Verify key fields
		if manifest["version"] != "1.0" {
			t.Errorf("Version mismatch in generated manifest")
		}

		provider, ok := manifest["provider"].(map[string]interface{})
		if !ok {
			t.Fatalf("Provider not found in manifest")
		}
		if provider["name"] != "aws" {
			t.Errorf("Provider name mismatch")
		}

		// Cleanup
		os.Remove(filePath)
	}
}

func TestGenerateManifestHandler_GCP(t *testing.T) {
	publicAccess := true
	reqData := ManifestRequest{
		Version: "1.0",
		Provider: ProviderConfig{
			Name:             "gcp",
			Region:           "us-central1",
			ProjectID:        "test-project",
			BillingAccountID: "123456-123456-123456",
			PublicAccess:     &publicAccess,
			Credentials: CredentialsConfig{
				ServiceAccountKeyPath: "/path/to/key.json",
			},
		},
		Application: ApplicationConfig{
			Name:        "test-gcp-app",
			Description: "Test GCP application",
		},
		Environment: EnvironmentConfig{
			Name: "test-gcp-env",
		},
		Deployment: DeploymentConfig{
			Platform: "docker",
			Source: SourceConfig{
				Type: "local",
				Path: ".",
			},
		},
		CloudRun: &CloudRunConfig{
			CPU:            "2",
			Memory:         "1Gi",
			MaxConcurrency: 100,
			MinInstances:   1,
			MaxInstances:   10,
			TimeoutSeconds: 600,
		},
		Monitoring: MonitoringConfig{
			CloudWatchLogs: &CloudWatchLogsConfig{
				Enabled:       true,
				RetentionDays: 7,
				StreamLogs:    true,
			},
		},
		EnvironmentVars: map[string]string{
			"ENV": "production",
		},
	}

	reqBody, err := json.Marshal(reqData)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/generate", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	generateManifestHandler(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		t.Logf("Response body: %s", rr.Body.String())
	}

	var response map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !strings.HasPrefix(response["filename"], "gcp-manifest-") {
		t.Errorf("Unexpected filename format: %s", response["filename"])
	}

	// Verify file
	filePath := response["path"]
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("Manifest file was not created: %s", filePath)
	} else {
		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read manifest file: %v", err)
		}

		var manifest map[string]interface{}
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			t.Fatalf("Generated manifest is not valid YAML: %v", err)
		}

		// Verify GCP-specific fields
		provider, ok := manifest["provider"].(map[string]interface{})
		if !ok {
			t.Fatalf("Provider not found in manifest")
		}
		if provider["name"] != "gcp" {
			t.Errorf("Provider name mismatch")
		}
		if provider["project_id"] != "test-project" {
			t.Errorf("Project ID mismatch")
		}

		// Verify Cloud Run config
		cloudRun, ok := manifest["cloud_run"].(map[string]interface{})
		if !ok {
			t.Fatalf("Cloud Run config not found in manifest")
		}
		if cloudRun["cpu"] != "2" {
			t.Errorf("CPU mismatch in Cloud Run config")
		}

		// Cleanup
		os.Remove(filePath)
	}
}

func TestGenerateManifestHandler_InvalidMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/generate", nil)
	rr := httptest.NewRecorder()

	generateManifestHandler(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestGenerateManifestHandler_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/generate", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	generateManifestHandler(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestGenerateManifestHandler_MinimalConfig(t *testing.T) {
	// Test with minimal required fields only
	reqData := ManifestRequest{
		Provider: ProviderConfig{
			Name:   "aws",
			Region: "us-east-1",
		},
		Application: ApplicationConfig{
			Name: "minimal-app",
		},
		Environment: EnvironmentConfig{
			Name: "minimal-env",
		},
		Deployment: DeploymentConfig{
			Source: SourceConfig{
				Type: "local",
				Path: ".",
			},
		},
	}

	reqBody, err := json.Marshal(reqData)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/generate", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	generateManifestHandler(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		t.Logf("Response body: %s", rr.Body.String())
	}

	var response map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Cleanup
	os.Remove(response["path"])
}

func TestManifestYAMLFormat(t *testing.T) {
	// Test that generated YAML is valid and well-formatted
	reqData := ManifestRequest{
		Version: "1.0",
		Provider: ProviderConfig{
			Name:   "aws",
			Region: "us-west-2",
		},
		Application: ApplicationConfig{
			Name: "format-test",
		},
		Environment: EnvironmentConfig{
			Name: "format-test-env",
		},
		Deployment: DeploymentConfig{
			Platform: "docker",
			Source: SourceConfig{
				Type: "local",
				Path: ".",
			},
		},
	}

	yamlData, err := yaml.Marshal(&reqData)
	if err != nil {
		t.Fatalf("Failed to marshal to YAML: %v", err)
	}

	// Unmarshal back to verify round-trip
	var roundTrip ManifestRequest
	if err := yaml.Unmarshal(yamlData, &roundTrip); err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	// Verify data integrity
	if roundTrip.Provider.Name != reqData.Provider.Name {
		t.Errorf("Provider name mismatch after round-trip")
	}
	if roundTrip.Application.Name != reqData.Application.Name {
		t.Errorf("Application name mismatch after round-trip")
	}
}

// TestCleanup ensures the generated-manifests directory is cleaned up after tests
func TestCleanup(t *testing.T) {
	manifestsDir := "generated-manifests"
	if _, err := os.Stat(manifestsDir); !os.IsNotExist(err) {
		files, err := filepath.Glob(filepath.Join(manifestsDir, "*-manifest-*.yaml"))
		if err != nil {
			t.Logf("Error finding test files: %v", err)
			return
		}
		for _, file := range files {
			os.Remove(file)
			t.Logf("Cleaned up test file: %s", file)
		}
	}
}
