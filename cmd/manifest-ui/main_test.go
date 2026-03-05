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

	"github.com/jvreagan/cloud-deploy/pkg/manifest"
	"gopkg.in/yaml.v3"
)

func TestGenerateManifestHandler_AWS(t *testing.T) {
	// Prepare test request
	reqData := ManifestRequest{
		Version: "1.0",
		Providers: []UIProviderConfig{
			{
				Name:   "aws",
				Region: "us-east-2",
				Instance: &manifest.InstanceConfig{
					Type:            "t3.micro",
					EnvironmentType: "SingleInstance",
				},
				HealthCheck: &manifest.HealthCheckConfig{
					Type: "enhanced",
					Path: "/health",
				},
				Monitoring: &manifest.MonitoringConfig{
					EnhancedHealth:    true,
					CloudWatchMetrics: true,
				},
			},
		},
		Application: manifest.ApplicationConfig{
			Name:        "test-app",
			Description: "Test application",
		},
		Environment: manifest.EnvironmentConfig{
			Name:  "test-env",
			CName: "test-app",
		},
		Deployment: manifest.DeploymentConfig{
			Platform:      "docker",
			SolutionStack: "64bit Amazon Linux 2023 v4.7.2 running Docker",
			Source: manifest.SourceConfig{
				Type: "local",
				Path: ".",
			},
		},
		Credentials: &CredentialsManager{
			Source: "environment",
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

		var m map[string]interface{}
		if err := yaml.Unmarshal(data, &m); err != nil {
			t.Fatalf("Generated manifest is not valid YAML: %v", err)
		}

		// Verify key fields
		if m["version"] != "1.0" {
			t.Errorf("Version mismatch in generated manifest")
		}

		// Verify CLI-compatible: singular "provider" (not "providers" array)
		provider, ok := m["provider"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected singular 'provider' key in manifest, got keys: %v", keys(m))
		}
		if provider["name"] != "aws" {
			t.Errorf("Provider name mismatch")
		}

		// Verify no "providers" array
		if _, hasProviders := m["providers"]; hasProviders {
			t.Errorf("Generated manifest should not contain 'providers' array")
		}

		// Cleanup
		os.Remove(filePath)
	}
}

func TestGenerateManifestHandler_GCP(t *testing.T) {
	publicAccess := true
	reqData := ManifestRequest{
		Version: "1.0",
		Providers: []UIProviderConfig{
			{
				Name:             "gcp",
				Region:           "us-central1",
				ProjectID:        "test-project",
				BillingAccountID: "123456-123456-123456",
				PublicAccess:     &publicAccess,
				CloudRun: &manifest.CloudRunConfig{
					CPU:            "2",
					Memory:         "1Gi",
					MaxConcurrency: 100,
					MinInstances:   1,
					MaxInstances:   10,
					TimeoutSeconds: 600,
				},
				Monitoring: &manifest.MonitoringConfig{
					CloudWatchLogs: &manifest.CloudWatchLogsConfig{
						Enabled:       true,
						RetentionDays: 7,
						StreamLogs:    true,
					},
				},
			},
		},
		Application: manifest.ApplicationConfig{
			Name:        "test-gcp-app",
			Description: "Test GCP application",
		},
		Environment: manifest.EnvironmentConfig{
			Name: "test-gcp-env",
		},
		Deployment: manifest.DeploymentConfig{
			Platform: "docker",
			Source: manifest.SourceConfig{
				Type: "local",
				Path: ".",
			},
		},
		Credentials: &CredentialsManager{
			Source: "environment",
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

		var m map[string]interface{}
		if err := yaml.Unmarshal(data, &m); err != nil {
			t.Fatalf("Generated manifest is not valid YAML: %v", err)
		}

		// Verify CLI-compatible singular provider
		provider, ok := m["provider"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected singular 'provider' key in manifest")
		}
		if provider["name"] != "gcp" {
			t.Errorf("Provider name mismatch")
		}
		if provider["project_id"] != "test-project" {
			t.Errorf("Project ID mismatch")
		}

		// Verify Cloud Run config
		cloudRun, ok := m["cloud_run"].(map[string]interface{})
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

func TestGenerateManifestHandler_MultiCloud(t *testing.T) {
	// Test multi-cloud deployment generates separate files per provider
	reqData := ManifestRequest{
		Version: "1.0",
		Providers: []UIProviderConfig{
			{
				Name:   "aws",
				Region: "us-east-1",
				Instance: &manifest.InstanceConfig{
					Type:            "t3.micro",
					EnvironmentType: "SingleInstance",
				},
			},
			{
				Name:      "gcp",
				Region:    "us-central1",
				ProjectID: "test-project",
				CloudRun: &manifest.CloudRunConfig{
					CPU:    "1",
					Memory: "512Mi",
				},
			},
			{
				Name:          "azure",
				Region:        "eastus",
				ResourceGroup: "test-rg",
				Container: &AzureContainerConfig{
					CPU:    1.0,
					Memory: 1.5,
				},
			},
		},
		Application: manifest.ApplicationConfig{
			Name:        "test-multicloud-app",
			Description: "Multi-cloud test application",
		},
		Environment: manifest.EnvironmentConfig{
			Name: "test-multicloud-env",
		},
		Deployment: manifest.DeploymentConfig{
			Platform: "docker",
			Source: manifest.SourceConfig{
				Type: "docker",
				Path: "myapp:latest",
			},
		},
		Credentials: &CredentialsManager{
			Source: "secrets-manager",
			Secrets: map[string]string{
				"aws":        "arn:aws:secretsmanager:us-east-1:123456789012:secret:aws-creds",
				"gcp":        "arn:aws:secretsmanager:us-east-1:123456789012:secret:gcp-creds",
				"azure":      "arn:aws:secretsmanager:us-east-1:123456789012:secret:azure-creds",
				"cloudflare": "arn:aws:secretsmanager:us-east-1:123456789012:secret:cf-creds",
			},
		},
		Cloudflare: &CloudflareConfig{
			Enabled: true,
			ZoneID:  "test-zone-id",
			Domain:  "app.example.com",
			LoadBalancer: CloudflareLoadBalancer{
				Name:           "multicloud-lb",
				SteeringPolicy: "dynamic_latency",
				Proxied:        true,
			},
			Monitors: []CloudflareMonitor{
				{
					Name:          "health-check",
					Type:          "https",
					Path:          "/health",
					Interval:      60,
					Retries:       2,
					ExpectedCodes: "200",
				},
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

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify multiple files were generated
	filenames, ok := response["filenames"].([]interface{})
	if !ok {
		t.Fatalf("Expected filenames array in response")
	}

	// Should have 3 provider files + 1 cloudflare config = 4 files
	if len(filenames) != 4 {
		t.Errorf("Expected 4 files (3 providers + cloudflare), got %d: %v", len(filenames), filenames)
	}

	// Verify each provider file is CLI-compatible (singular provider key)
	manifestsDir := "generated-manifests"
	for _, fn := range filenames[:3] { // Skip cloudflare config
		filename := fn.(string)
		filePath := filepath.Join(manifestsDir, filename)

		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read manifest file %s: %v", filename, err)
		}

		var m map[string]interface{}
		if err := yaml.Unmarshal(data, &m); err != nil {
			t.Fatalf("Generated manifest %s is not valid YAML: %v", filename, err)
		}

		// Each file must have singular "provider" not "providers"
		if _, hasProvider := m["provider"]; !hasProvider {
			t.Errorf("File %s missing singular 'provider' key", filename)
		}
		if _, hasProviders := m["providers"]; hasProviders {
			t.Errorf("File %s should not contain 'providers' array", filename)
		}

		os.Remove(filePath)
	}

	// Clean up cloudflare config file
	if len(filenames) >= 4 {
		cfFile := filenames[3].(string)
		os.Remove(filepath.Join(manifestsDir, cfFile))
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
		Providers: []UIProviderConfig{
			{
				Name:   "aws",
				Region: "us-east-1",
			},
		},
		Application: manifest.ApplicationConfig{
			Name: "minimal-app",
		},
		Environment: manifest.EnvironmentConfig{
			Name: "minimal-env",
		},
		Deployment: manifest.DeploymentConfig{
			Source: manifest.SourceConfig{
				Type: "local",
				Path: ".",
			},
		},
		Credentials: &CredentialsManager{
			Source: "environment",
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
	// Test that generated YAML uses singular "provider" key
	reqData := ManifestRequest{
		Version: "1.0",
		Providers: []UIProviderConfig{
			{
				Name:   "aws",
				Region: "us-west-2",
			},
		},
		Application: manifest.ApplicationConfig{
			Name: "format-test",
		},
		Environment: manifest.EnvironmentConfig{
			Name: "format-test-env",
		},
		Deployment: manifest.DeploymentConfig{
			Platform: "docker",
			Source: manifest.SourceConfig{
				Type: "local",
				Path: ".",
			},
		},
	}

	m := reqData.toManifest(0)

	yamlData, err := yaml.Marshal(&m)
	if err != nil {
		t.Fatalf("Failed to marshal to YAML: %v", err)
	}

	// Verify the YAML contains "provider:" (singular), not "providers:"
	yamlStr := string(yamlData)
	if !strings.Contains(yamlStr, "provider:") {
		t.Errorf("YAML should contain 'provider:' key")
	}
	if strings.Contains(yamlStr, "providers:") {
		t.Errorf("YAML should not contain 'providers:' key")
	}

	// Unmarshal back to verify round-trip
	var roundTrip manifest.Manifest
	if err := yaml.Unmarshal(yamlData, &roundTrip); err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	// Verify data integrity
	if roundTrip.Provider.Name != reqData.Providers[0].Name {
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
		// Also clean up cloudflare config files
		cfFiles, _ := filepath.Glob(filepath.Join(manifestsDir, "cloudflare-config-*.yaml"))
		for _, file := range cfFiles {
			os.Remove(file)
			t.Logf("Cleaned up cloudflare config: %s", file)
		}
	}
}

// keys is a test helper that returns the keys of a map.
func keys(m map[string]interface{}) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}
