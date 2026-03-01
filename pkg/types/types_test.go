package types

import (
	"testing"
)

func TestDeploymentResult(t *testing.T) {
	tests := []struct {
		name            string
		result          DeploymentResult
		wantApp         string
		wantEnv         string
		wantURL         string
		wantStatus      string
		wantMessage     string
	}{
		{
			name: "complete deployment result",
			result: DeploymentResult{
				ApplicationName: "test-app",
				EnvironmentName: "test-env",
				URL:             "https://test-app.example.com",
				Status:          "Ready",
				Message:         "Deployment successful",
			},
			wantApp:     "test-app",
			wantEnv:     "test-env",
			wantURL:     "https://test-app.example.com",
			wantStatus:  "Ready",
			wantMessage: "Deployment successful",
		},
		{
			name:        "empty deployment result",
			result:      DeploymentResult{},
			wantApp:     "",
			wantEnv:     "",
			wantURL:     "",
			wantStatus:  "",
			wantMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.ApplicationName != tt.wantApp {
				t.Errorf("ApplicationName = %q, want %q", tt.result.ApplicationName, tt.wantApp)
			}
			if tt.result.EnvironmentName != tt.wantEnv {
				t.Errorf("EnvironmentName = %q, want %q", tt.result.EnvironmentName, tt.wantEnv)
			}
			if tt.result.URL != tt.wantURL {
				t.Errorf("URL = %q, want %q", tt.result.URL, tt.wantURL)
			}
			if tt.result.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", tt.result.Status, tt.wantStatus)
			}
			if tt.result.Message != tt.wantMessage {
				t.Errorf("Message = %q, want %q", tt.result.Message, tt.wantMessage)
			}
		})
	}
}

func TestDeploymentStatus(t *testing.T) {
	tests := []struct {
		name            string
		status          DeploymentStatus
		wantApp         string
		wantEnv         string
		wantStatus      string
		wantHealth      string
		wantURL         string
		wantLastUpdated string
	}{
		{
			name: "complete deployment status",
			status: DeploymentStatus{
				ApplicationName: "test-app",
				EnvironmentName: "test-env",
				Status:          "Ready",
				Health:          "Green",
				URL:             "https://test-app.example.com",
				LastUpdated:     "2024-01-01T00:00:00Z",
			},
			wantApp:         "test-app",
			wantEnv:         "test-env",
			wantStatus:      "Ready",
			wantHealth:      "Green",
			wantURL:         "https://test-app.example.com",
			wantLastUpdated: "2024-01-01T00:00:00Z",
		},
		{
			name:            "empty deployment status",
			status:          DeploymentStatus{},
			wantApp:         "",
			wantEnv:         "",
			wantStatus:      "",
			wantHealth:      "",
			wantURL:         "",
			wantLastUpdated: "",
		},
		{
			name: "unhealthy status",
			status: DeploymentStatus{
				ApplicationName: "unhealthy-app",
				EnvironmentName: "prod-env",
				Status:          "Updating",
				Health:          "Red",
				URL:             "https://unhealthy-app.example.com",
				LastUpdated:     "2024-01-02T10:30:00Z",
			},
			wantApp:         "unhealthy-app",
			wantEnv:         "prod-env",
			wantStatus:      "Updating",
			wantHealth:      "Red",
			wantURL:         "https://unhealthy-app.example.com",
			wantLastUpdated: "2024-01-02T10:30:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.status.ApplicationName != tt.wantApp {
				t.Errorf("ApplicationName = %q, want %q", tt.status.ApplicationName, tt.wantApp)
			}
			if tt.status.EnvironmentName != tt.wantEnv {
				t.Errorf("EnvironmentName = %q, want %q", tt.status.EnvironmentName, tt.wantEnv)
			}
			if tt.status.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", tt.status.Status, tt.wantStatus)
			}
			if tt.status.Health != tt.wantHealth {
				t.Errorf("Health = %q, want %q", tt.status.Health, tt.wantHealth)
			}
			if tt.status.URL != tt.wantURL {
				t.Errorf("URL = %q, want %q", tt.status.URL, tt.wantURL)
			}
			if tt.status.LastUpdated != tt.wantLastUpdated {
				t.Errorf("LastUpdated = %q, want %q", tt.status.LastUpdated, tt.wantLastUpdated)
			}
		})
	}
}
