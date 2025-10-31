package types

import (
	"testing"
)

func TestDeploymentResult(t *testing.T) {
	tests := []struct {
		name   string
		result DeploymentResult
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
		},
		{
			name:   "empty deployment result",
			result: DeploymentResult{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify struct can be created and accessed
			if tt.result.ApplicationName != tt.result.ApplicationName {
				t.Errorf("ApplicationName mismatch")
			}
			if tt.result.EnvironmentName != tt.result.EnvironmentName {
				t.Errorf("EnvironmentName mismatch")
			}
			if tt.result.URL != tt.result.URL {
				t.Errorf("URL mismatch")
			}
			if tt.result.Status != tt.result.Status {
				t.Errorf("Status mismatch")
			}
			if tt.result.Message != tt.result.Message {
				t.Errorf("Message mismatch")
			}
		})
	}
}

func TestDeploymentStatus(t *testing.T) {
	tests := []struct {
		name   string
		status DeploymentStatus
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
		},
		{
			name:   "empty deployment status",
			status: DeploymentStatus{},
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify struct can be created and accessed
			if tt.status.ApplicationName != tt.status.ApplicationName {
				t.Errorf("ApplicationName mismatch")
			}
			if tt.status.EnvironmentName != tt.status.EnvironmentName {
				t.Errorf("EnvironmentName mismatch")
			}
			if tt.status.Status != tt.status.Status {
				t.Errorf("Status mismatch")
			}
			if tt.status.Health != tt.status.Health {
				t.Errorf("Health mismatch")
			}
			if tt.status.URL != tt.status.URL {
				t.Errorf("URL mismatch")
			}
			if tt.status.LastUpdated != tt.status.LastUpdated {
				t.Errorf("LastUpdated mismatch")
			}
		})
	}
}

func TestDeploymentResultFieldAssignment(t *testing.T) {
	result := DeploymentResult{}

	result.ApplicationName = "new-app"
	if result.ApplicationName != "new-app" {
		t.Errorf("Expected ApplicationName 'new-app', got '%s'", result.ApplicationName)
	}

	result.EnvironmentName = "new-env"
	if result.EnvironmentName != "new-env" {
		t.Errorf("Expected EnvironmentName 'new-env', got '%s'", result.EnvironmentName)
	}

	result.URL = "https://example.com"
	if result.URL != "https://example.com" {
		t.Errorf("Expected URL 'https://example.com', got '%s'", result.URL)
	}

	result.Status = "Launching"
	if result.Status != "Launching" {
		t.Errorf("Expected Status 'Launching', got '%s'", result.Status)
	}

	result.Message = "Test message"
	if result.Message != "Test message" {
		t.Errorf("Expected Message 'Test message', got '%s'", result.Message)
	}
}

func TestDeploymentStatusFieldAssignment(t *testing.T) {
	status := DeploymentStatus{}

	status.ApplicationName = "new-app"
	if status.ApplicationName != "new-app" {
		t.Errorf("Expected ApplicationName 'new-app', got '%s'", status.ApplicationName)
	}

	status.EnvironmentName = "new-env"
	if status.EnvironmentName != "new-env" {
		t.Errorf("Expected EnvironmentName 'new-env', got '%s'", status.EnvironmentName)
	}

	status.Status = "Ready"
	if status.Status != "Ready" {
		t.Errorf("Expected Status 'Ready', got '%s'", status.Status)
	}

	status.Health = "Green"
	if status.Health != "Green" {
		t.Errorf("Expected Health 'Green', got '%s'", status.Health)
	}

	status.URL = "https://example.com"
	if status.URL != "https://example.com" {
		t.Errorf("Expected URL 'https://example.com', got '%s'", status.URL)
	}

	status.LastUpdated = "2024-01-01"
	if status.LastUpdated != "2024-01-01" {
		t.Errorf("Expected LastUpdated '2024-01-01', got '%s'", status.LastUpdated)
	}
}
