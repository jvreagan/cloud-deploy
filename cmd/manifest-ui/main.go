package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// ManifestRequest represents the data sent from the frontend
type ManifestRequest struct {
	Version             string                 `json:"version" yaml:"version"`
	Provider            ProviderConfig         `json:"provider" yaml:"provider"`
	Application         ApplicationConfig      `json:"application" yaml:"application"`
	Environment         EnvironmentConfig      `json:"environment" yaml:"environment"`
	Deployment          DeploymentConfig       `json:"deployment" yaml:"deployment"`
	Instance            InstanceConfig         `json:"instance,omitempty" yaml:"instance,omitempty"`
	CloudRun            *CloudRunConfig        `json:"cloud_run,omitempty" yaml:"cloud_run,omitempty"`
	HealthCheck         HealthCheckConfig      `json:"health_check,omitempty" yaml:"health_check,omitempty"`
	Monitoring          MonitoringConfig       `json:"monitoring,omitempty" yaml:"monitoring,omitempty"`
	IAM                 IAMConfig              `json:"iam,omitempty" yaml:"iam,omitempty"`
	EnvironmentVars     map[string]string      `json:"environment_variables,omitempty" yaml:"environment_variables,omitempty"`
	Tags                map[string]string      `json:"tags,omitempty" yaml:"tags,omitempty"`
}

type ProviderConfig struct {
	Name              string            `json:"name" yaml:"name"`
	Region            string            `json:"region" yaml:"region"`
	ProjectID         string            `json:"project_id,omitempty" yaml:"project_id,omitempty"`
	BillingAccountID  string            `json:"billing_account_id,omitempty" yaml:"billing_account_id,omitempty"`
	OrganizationID    string            `json:"organization_id,omitempty" yaml:"organization_id,omitempty"`
	PublicAccess      *bool             `json:"public_access,omitempty" yaml:"public_access,omitempty"`
	Credentials       CredentialsConfig `json:"credentials,omitempty" yaml:"credentials,omitempty"`
}

type CredentialsConfig struct {
	AccessKeyID              string `json:"access_key_id,omitempty" yaml:"access_key_id,omitempty"`
	SecretAccessKey          string `json:"secret_access_key,omitempty" yaml:"secret_access_key,omitempty"`
	ServiceAccountKeyPath    string `json:"service_account_key_path,omitempty" yaml:"service_account_key_path,omitempty"`
	ServiceAccountKeyJSON    string `json:"service_account_key_json,omitempty" yaml:"service_account_key_json,omitempty"`
}

type ApplicationConfig struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type EnvironmentConfig struct {
	Name  string `json:"name" yaml:"name"`
	Cname string `json:"cname,omitempty" yaml:"cname,omitempty"`
}

type DeploymentConfig struct {
	Platform      string       `json:"platform,omitempty" yaml:"platform,omitempty"`
	SolutionStack string       `json:"solution_stack,omitempty" yaml:"solution_stack,omitempty"`
	Source        SourceConfig `json:"source" yaml:"source"`
}

type SourceConfig struct {
	Type string `json:"type" yaml:"type"`
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
}

type InstanceConfig struct {
	Type            string `json:"type,omitempty" yaml:"type,omitempty"`
	EnvironmentType string `json:"environment_type,omitempty" yaml:"environment_type,omitempty"`
}

type CloudRunConfig struct {
	CPU            string `json:"cpu,omitempty" yaml:"cpu,omitempty"`
	Memory         string `json:"memory,omitempty" yaml:"memory,omitempty"`
	MaxConcurrency int32  `json:"max_concurrency,omitempty" yaml:"max_concurrency,omitempty"`
	MinInstances   int32  `json:"min_instances,omitempty" yaml:"min_instances,omitempty"`
	MaxInstances   int32  `json:"max_instances,omitempty" yaml:"max_instances,omitempty"`
	TimeoutSeconds int32  `json:"timeout_seconds,omitempty" yaml:"timeout_seconds,omitempty"`
}

type HealthCheckConfig struct {
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
}

type MonitoringConfig struct {
	EnhancedHealth    bool                   `json:"enhanced_health,omitempty" yaml:"enhanced_health,omitempty"`
	CloudWatchMetrics bool                   `json:"cloudwatch_metrics,omitempty" yaml:"cloudwatch_metrics,omitempty"`
	CloudWatchLogs    *CloudWatchLogsConfig  `json:"cloudwatch_logs,omitempty" yaml:"cloudwatch_logs,omitempty"`
}

type CloudWatchLogsConfig struct {
	Enabled       bool   `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	RetentionDays int    `json:"retention_days,omitempty" yaml:"retention_days,omitempty"`
	StreamLogs    bool   `json:"stream_logs,omitempty" yaml:"stream_logs,omitempty"`
}

type IAMConfig struct {
	InstanceProfile string `json:"instance_profile,omitempty" yaml:"instance_profile,omitempty"`
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

	// Convert to YAML
	yamlData, err := yaml.Marshal(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate YAML: %v", err), http.StatusInternalServerError)
		return
	}

	// Create manifests directory if it doesn't exist
	manifestsDir := "generated-manifests"
	if err := os.MkdirAll(manifestsDir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create manifests directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s-manifest-%s.yaml", req.Provider.Name, timestamp)
	filepath := filepath.Join(manifestsDir, filename)

	// Write file
	if err := os.WriteFile(filepath, yamlData, 0644); err != nil {
		http.Error(w, fmt.Sprintf("Failed to write manifest file: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := map[string]string{
		"message":  "Manifest generated successfully",
		"filename": filename,
		"path":     filepath,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
