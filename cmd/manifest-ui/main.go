package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jvreagan/cloud-deploy/pkg/manifest"
	"gopkg.in/yaml.v3"
)

// ManifestRequest represents the data sent from the frontend.
// It accepts multiple providers (the UI's multi-cloud view) and converts
// them to individual CLI-compatible manifests via toManifest().
type ManifestRequest struct {
	Version         string                     `json:"version"`
	Application     manifest.ApplicationConfig `json:"application"`
	Environment     manifest.EnvironmentConfig `json:"environment"`
	Deployment      manifest.DeploymentConfig  `json:"deployment"`
	Providers       []UIProviderConfig         `json:"providers"`
	Credentials     *CredentialsManager        `json:"credentials,omitempty"`
	Cloudflare      *CloudflareConfig          `json:"cloudflare,omitempty"`
	EnvironmentVars map[string]string          `json:"environment_variables,omitempty"`
	Tags            map[string]string          `json:"tags,omitempty"`
}

// UIProviderConfig extends manifest.ProviderConfig with UI-specific fields
// like Instance, CloudRun, Container, HealthCheck, Monitoring, and IAM
// that the UI groups under each provider.
type UIProviderConfig struct {
	Name             string                      `json:"name"`
	Region           string                      `json:"region"`
	ProjectID        string                      `json:"project_id,omitempty"`
	BillingAccountID string                      `json:"billing_account_id,omitempty"`
	OrganizationID   string                      `json:"organization_id,omitempty"`
	PublicAccess     *bool                       `json:"public_access,omitempty"`
	ResourceGroup    string                      `json:"resource_group,omitempty"`
	SubscriptionID   string                      `json:"subscription_id,omitempty"`
	Instance         *manifest.InstanceConfig    `json:"instance,omitempty"`
	CloudRun         *manifest.CloudRunConfig    `json:"cloud_run,omitempty"`
	Container        *AzureContainerConfig       `json:"container,omitempty"`
	HealthCheck      *manifest.HealthCheckConfig `json:"health_check,omitempty"`
	Monitoring       *manifest.MonitoringConfig  `json:"monitoring,omitempty"`
	IAM              *manifest.IAMConfig         `json:"iam,omitempty"`
}

// CredentialsManager is a UI-only type for credential source configuration.
type CredentialsManager struct {
	Source  string            `json:"source" yaml:"source"`
	Secrets map[string]string `json:"secrets,omitempty" yaml:"secrets,omitempty"`
}

// AzureContainerConfig is a UI-only type for Azure container resource configuration.
type AzureContainerConfig struct {
	CPU           float32 `json:"cpu,omitempty" yaml:"cpu,omitempty"`
	Memory        float32 `json:"memory,omitempty" yaml:"memory,omitempty"`
	Port          int32   `json:"port,omitempty" yaml:"port,omitempty"`
	RestartPolicy string  `json:"restart_policy,omitempty" yaml:"restart_policy,omitempty"`
}

// CloudflareConfig is a UI-only type for Cloudflare load balancer configuration.
type CloudflareConfig struct {
	Enabled      bool                   `json:"enabled" yaml:"enabled"`
	ZoneID       string                 `json:"zone_id" yaml:"zone_id"`
	AccountID    string                 `json:"account_id,omitempty" yaml:"account_id,omitempty"`
	Domain       string                 `json:"domain" yaml:"domain"`
	LoadBalancer CloudflareLoadBalancer `json:"load_balancer" yaml:"load_balancer"`
	Pools        []CloudflarePool       `json:"pools,omitempty" yaml:"pools,omitempty"`
	Monitors     []CloudflareMonitor    `json:"monitors,omitempty" yaml:"monitors,omitempty"`
}

type CloudflareLoadBalancer struct {
	Name           string `json:"name" yaml:"name"`
	SteeringPolicy string `json:"steering_policy,omitempty" yaml:"steering_policy,omitempty"`
	TTL            int32  `json:"ttl,omitempty" yaml:"ttl,omitempty"`
	Proxied        bool   `json:"proxied" yaml:"proxied"`
}

type CloudflarePool struct {
	Name        string             `json:"name" yaml:"name"`
	Description string             `json:"description,omitempty" yaml:"description,omitempty"`
	Provider    string             `json:"provider,omitempty" yaml:"provider,omitempty"`
	Origins     []CloudflareOrigin `json:"origins,omitempty" yaml:"origins,omitempty"`
}

type CloudflareOrigin struct {
	Name    string            `json:"name" yaml:"name"`
	Address string            `json:"address,omitempty" yaml:"address,omitempty"`
	Enabled bool              `json:"enabled" yaml:"enabled"`
	Weight  float32           `json:"weight,omitempty" yaml:"weight,omitempty"`
	Header  map[string]string `json:"header,omitempty" yaml:"header,omitempty"`
}

type CloudflareMonitor struct {
	Name          string `json:"name" yaml:"name"`
	Type          string `json:"type" yaml:"type"`
	Path          string `json:"path,omitempty" yaml:"path,omitempty"`
	Port          int32  `json:"port,omitempty" yaml:"port,omitempty"`
	Interval      int32  `json:"interval,omitempty" yaml:"interval,omitempty"`
	Retries       int32  `json:"retries,omitempty" yaml:"retries,omitempty"`
	Timeout       int32  `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	ExpectedCodes string `json:"expected_codes,omitempty" yaml:"expected_codes,omitempty"`
}

// toManifest converts the UI request into a CLI-compatible manifest.Manifest
// for the provider at the given index.
func (req *ManifestRequest) toManifest(providerIdx int) manifest.Manifest {
	p := req.Providers[providerIdx]

	m := manifest.Manifest{
		Version:              req.Version,
		Application:          req.Application,
		Environment:          req.Environment,
		Deployment:           req.Deployment,
		EnvironmentVariables: req.EnvironmentVars,
		Tags:                 req.Tags,
	}

	// Map UI provider to CLI provider (singular)
	m.Provider = manifest.ProviderConfig{
		Name:             p.Name,
		Region:           p.Region,
		ProjectID:        p.ProjectID,
		BillingAccountID: p.BillingAccountID,
		OrganizationID:   p.OrganizationID,
		PublicAccess:     p.PublicAccess,
		ResourceGroup:    p.ResourceGroup,
		SubscriptionID:   p.SubscriptionID,
	}

	// Map credentials source
	if req.Credentials != nil {
		m.Provider.Credentials = &manifest.CredentialsConfig{
			Source: req.Credentials.Source,
		}
	}

	// Map provider-specific config
	if p.Instance != nil {
		m.Instance = *p.Instance
	}
	if p.CloudRun != nil {
		m.CloudRun = p.CloudRun
	}
	if p.Container != nil {
		m.Azure = &manifest.AzureConfig{
			CPU:      float64(p.Container.CPU),
			MemoryGB: float64(p.Container.Memory),
		}
	}
	if p.HealthCheck != nil {
		m.HealthCheck = *p.HealthCheck
	}
	if p.Monitoring != nil {
		m.Monitoring = *p.Monitoring
	}
	if p.IAM != nil {
		m.IAM = *p.IAM
	}

	return m
}

func main() {
	// Serve static files
	fs := http.FileServer(http.Dir("web/static"))
	http.Handle("/", fs)

	// API endpoint for generating manifest
	http.HandleFunc("/api/generate", generateManifestHandler)

	port := ":5001"
	fmt.Printf("Starting manifest-ui server on http://localhost%s\n", port)
	fmt.Println("Open your browser and navigate to http://localhost:5001")

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}

func generateManifestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ManifestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Set default version if not provided
	if req.Version == "" {
		req.Version = "1.0"
	}

	// Create manifests directory if it doesn't exist
	manifestsDir := "generated-manifests"
	if err := os.MkdirAll(manifestsDir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create manifests directory: %v", err), http.StatusInternalServerError)
		return
	}

	timestamp := time.Now().Format("20060102-150405")

	if len(req.Providers) == 0 {
		http.Error(w, "At least one provider is required", http.StatusBadRequest)
		return
	}

	// Single provider: generate one CLI-compatible manifest
	if len(req.Providers) == 1 {
		m := req.toManifest(0)

		yamlData, err := yaml.Marshal(&m)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to generate YAML: %v", err), http.StatusInternalServerError)
			return
		}

		filename := fmt.Sprintf("%s-manifest-%s.yaml", req.Providers[0].Name, timestamp)
		filePath := filepath.Join(manifestsDir, filename)

		if err := os.WriteFile(filePath, yamlData, 0644); err != nil {
			http.Error(w, fmt.Sprintf("Failed to write manifest file: %v", err), http.StatusInternalServerError)
			return
		}

		response := map[string]string{
			"message":  "Manifest generated successfully",
			"filename": filename,
			"path":     filePath,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Multi-provider: generate one file per provider
	var filenames []string
	var lastPath string

	for i, p := range req.Providers {
		m := req.toManifest(i)

		yamlData, err := yaml.Marshal(&m)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to generate YAML for %s: %v", p.Name, err), http.StatusInternalServerError)
			return
		}

		filename := fmt.Sprintf("%s-manifest-%s.yaml", p.Name, timestamp)
		filePath := filepath.Join(manifestsDir, filename)

		if err := os.WriteFile(filePath, yamlData, 0644); err != nil {
			http.Error(w, fmt.Sprintf("Failed to write manifest file: %v", err), http.StatusInternalServerError)
			return
		}

		filenames = append(filenames, filename)
		lastPath = filePath
	}

	// Also write the Cloudflare config as a separate file if present
	if req.Cloudflare != nil && req.Cloudflare.Enabled {
		cfData, err := yaml.Marshal(req.Cloudflare)
		if err == nil {
			cfFilename := fmt.Sprintf("cloudflare-config-%s.yaml", timestamp)
			cfPath := filepath.Join(manifestsDir, cfFilename)
			os.WriteFile(cfPath, cfData, 0644)
			filenames = append(filenames, cfFilename)
		}
	}

	response := map[string]interface{}{
		"message":   fmt.Sprintf("Generated %d manifest files", len(req.Providers)),
		"filename":  fmt.Sprintf("multi-cloud-manifest-%s.yaml", timestamp),
		"filenames": filenames,
		"path":      lastPath,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
