package aws

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jvreagan/cloud-deploy/pkg/manifest"
)

func TestProviderName(t *testing.T) {
	provider := &Provider{
		region: "us-east-1",
	}

	if provider.Name() != "aws" {
		t.Errorf("Expected provider name 'aws', got '%s'", provider.Name())
	}
}

func TestZipDirectory(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"file1.txt":            "content1",
		"file2.txt":            "content2",
		"subdir/file3.txt":     "content3",
		"subdir/deep/file4.go": "package main",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create a temporary zip file
	zipFile, err := os.CreateTemp("", "test-*.zip")
	if err != nil {
		t.Fatalf("Failed to create temp zip file: %v", err)
	}
	defer os.Remove(zipFile.Name())
	defer zipFile.Close()

	// Test zipDirectory
	err = zipDirectory(tmpDir, zipFile)
	if err != nil {
		t.Fatalf("zipDirectory failed: %v", err)
	}

	// Rewind and verify zip contents
	if _, err := zipFile.Seek(0, 0); err != nil {
		t.Fatalf("Failed to seek: %v", err)
	}

	stat, err := zipFile.Stat()
	if err != nil {
		t.Fatalf("Failed to stat zip file: %v", err)
	}

	zipReader, err := zip.NewReader(zipFile, stat.Size())
	if err != nil {
		t.Fatalf("Failed to create zip reader: %v", err)
	}

	// Verify all files are in the zip
	foundFiles := make(map[string]bool)
	for _, f := range zipReader.File {
		foundFiles[f.Name] = true
	}

	expectedFiles := []string{
		"file1.txt",
		"file2.txt",
		"subdir/file3.txt",
		"subdir/deep/file4.go",
	}

	for _, expectedFile := range expectedFiles {
		if !foundFiles[expectedFile] {
			t.Errorf("Expected file '%s' not found in zip", expectedFile)
		}
	}
}

func TestZipDirectorySkipsHiddenFiles(t *testing.T) {
	// Create a temporary directory with hidden files
	tmpDir := t.TempDir()

	// Create regular and hidden files
	files := []string{
		"regular.txt",
		".hidden",
		".git/config",
	}

	for _, file := range files {
		fullPath := filepath.Join(tmpDir, file)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create a temporary zip file
	zipFile, err := os.CreateTemp("", "test-*.zip")
	if err != nil {
		t.Fatalf("Failed to create temp zip file: %v", err)
	}
	defer os.Remove(zipFile.Name())
	defer zipFile.Close()

	// Test zipDirectory
	err = zipDirectory(tmpDir, zipFile)
	if err != nil {
		t.Fatalf("zipDirectory failed: %v", err)
	}

	// Rewind and verify zip contents
	if _, err := zipFile.Seek(0, 0); err != nil {
		t.Fatalf("Failed to seek: %v", err)
	}

	stat, err := zipFile.Stat()
	if err != nil {
		t.Fatalf("Failed to stat zip file: %v", err)
	}

	zipReader, err := zip.NewReader(zipFile, stat.Size())
	if err != nil {
		t.Fatalf("Failed to create zip reader: %v", err)
	}

	// Verify only regular files are included
	for _, f := range zipReader.File {
		if strings.HasPrefix(filepath.Base(f.Name), ".") {
			t.Errorf("Hidden file '%s' should not be in zip", f.Name)
		}
	}

	// Verify regular file is included
	foundRegular := false
	for _, f := range zipReader.File {
		if f.Name == "regular.txt" {
			foundRegular = true
			break
		}
	}
	if !foundRegular {
		t.Error("Regular file 'regular.txt' should be in zip")
	}
}

func TestBuildOptionSettings(t *testing.T) {
	provider := &Provider{
		region: "us-east-1",
	}

	tests := []struct {
		name            string
		manifest        *manifest.Manifest
		expectedOptions map[string]map[string]string
	}{
		{
			name: "basic settings",
			manifest: &manifest.Manifest{
				Instance: manifest.InstanceConfig{
					Type:            "t3.micro",
					EnvironmentType: "SingleInstance",
				},
				HealthCheck: manifest.HealthCheckConfig{},
				Monitoring:  manifest.MonitoringConfig{},
				IAM:         manifest.IAMConfig{},
			},
			expectedOptions: map[string]map[string]string{
				"aws:autoscaling:launchconfiguration": {
					"InstanceType": "t3.micro",
				},
				"aws:elasticbeanstalk:environment": {
					"EnvironmentType": "SingleInstance",
				},
			},
		},
		{
			name: "with IAM instance profile",
			manifest: &manifest.Manifest{
				Instance: manifest.InstanceConfig{
					Type:            "t3.small",
					EnvironmentType: "LoadBalanced",
				},
				IAM: manifest.IAMConfig{
					InstanceProfile: "my-instance-profile",
				},
				HealthCheck: manifest.HealthCheckConfig{},
				Monitoring:  manifest.MonitoringConfig{},
			},
			expectedOptions: map[string]map[string]string{
				"aws:autoscaling:launchconfiguration": {
					"InstanceType":      "t3.small",
					"IamInstanceProfile": "my-instance-profile",
				},
				"aws:elasticbeanstalk:environment": {
					"EnvironmentType": "LoadBalanced",
				},
			},
		},
		{
			name: "with health check path",
			manifest: &manifest.Manifest{
				Instance: manifest.InstanceConfig{
					Type:            "t3.medium",
					EnvironmentType: "SingleInstance",
				},
				HealthCheck: manifest.HealthCheckConfig{
					Type: "basic",
					Path: "/health",
				},
				Monitoring: manifest.MonitoringConfig{},
				IAM:        manifest.IAMConfig{},
			},
			expectedOptions: map[string]map[string]string{
				"aws:elasticbeanstalk:application": {
					"Application Healthcheck URL": "/health",
				},
			},
		},
		{
			name: "with enhanced health reporting",
			manifest: &manifest.Manifest{
				Instance: manifest.InstanceConfig{
					Type:            "t3.micro",
					EnvironmentType: "SingleInstance",
				},
				HealthCheck: manifest.HealthCheckConfig{
					Type: "enhanced",
				},
				Monitoring: manifest.MonitoringConfig{
					EnhancedHealth: true,
				},
				IAM: manifest.IAMConfig{},
			},
			expectedOptions: map[string]map[string]string{
				"aws:elasticbeanstalk:healthreporting:system": {
					"SystemType": "enhanced",
				},
			},
		},
		{
			name: "with CloudWatch metrics",
			manifest: &manifest.Manifest{
				Instance: manifest.InstanceConfig{
					Type:            "t3.micro",
					EnvironmentType: "SingleInstance",
				},
				HealthCheck: manifest.HealthCheckConfig{},
				Monitoring: manifest.MonitoringConfig{
					CloudWatchMetrics: true,
				},
				IAM: manifest.IAMConfig{},
			},
			expectedOptions: map[string]map[string]string{
				"aws:autoscaling:launchconfiguration": {
					"MonitoringInterval": "1 minute",
				},
			},
		},
		{
			name: "with CloudWatch Logs",
			manifest: &manifest.Manifest{
				Instance: manifest.InstanceConfig{
					Type:            "t3.micro",
					EnvironmentType: "SingleInstance",
				},
				HealthCheck: manifest.HealthCheckConfig{},
				Monitoring: manifest.MonitoringConfig{
					CloudWatchLogs: &manifest.CloudWatchLogsConfig{
						Enabled:       true,
						RetentionDays: 7,
						StreamLogs:    true,
					},
				},
				IAM: manifest.IAMConfig{},
			},
			expectedOptions: map[string]map[string]string{
				"aws:elasticbeanstalk:cloudwatch:logs": {
					"StreamLogs":      "true",
					"RetentionInDays": "7",
				},
				"aws:elasticbeanstalk:cloudwatch:logs:health": {
					"HealthStreamingEnabled": "true",
				},
			},
		},
		{
			name: "with environment variables",
			manifest: &manifest.Manifest{
				Instance: manifest.InstanceConfig{
					Type:            "t3.micro",
					EnvironmentType: "SingleInstance",
				},
				HealthCheck: manifest.HealthCheckConfig{},
				Monitoring:  manifest.MonitoringConfig{},
				IAM:         manifest.IAMConfig{},
				EnvironmentVariables: map[string]string{
					"ENV":     "production",
					"API_KEY": "secret-key",
					"DEBUG":   "false",
				},
			},
			expectedOptions: map[string]map[string]string{
				"aws:elasticbeanstalk:application:environment": {
					"ENV":     "production",
					"API_KEY": "secret-key",
					"DEBUG":   "false",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := provider.buildOptionSettings(tt.manifest)

			// Convert settings to a map for easier validation
			settingsMap := make(map[string]map[string]string)
			for _, setting := range settings {
				namespace := *setting.Namespace
				optionName := *setting.OptionName
				value := *setting.Value

				if settingsMap[namespace] == nil {
					settingsMap[namespace] = make(map[string]string)
				}
				settingsMap[namespace][optionName] = value
			}

			// Verify expected options are present
			for expectedNamespace, expectedOptions := range tt.expectedOptions {
				actualOptions, exists := settingsMap[expectedNamespace]
				if !exists {
					t.Errorf("Expected namespace '%s' not found in settings", expectedNamespace)
					continue
				}

				for expectedOption, expectedValue := range expectedOptions {
					actualValue, exists := actualOptions[expectedOption]
					if !exists {
						t.Errorf("Expected option '%s' not found in namespace '%s'", expectedOption, expectedNamespace)
						continue
					}
					if actualValue != expectedValue {
						t.Errorf("For namespace '%s', option '%s': expected value '%s', got '%s'",
							expectedNamespace, expectedOption, expectedValue, actualValue)
					}
				}
			}
		})
	}
}

func TestBuildOptionSettingsComplex(t *testing.T) {
	provider := &Provider{
		region: "us-west-2",
	}

	// Test a complex configuration with multiple features
	m := &manifest.Manifest{
		Instance: manifest.InstanceConfig{
			Type:            "t3.large",
			EnvironmentType: "LoadBalanced",
		},
		IAM: manifest.IAMConfig{
			InstanceProfile: "my-profile",
			ServiceRole:     "my-role",
		},
		HealthCheck: manifest.HealthCheckConfig{
			Type: "enhanced",
			Path: "/api/health",
		},
		Monitoring: manifest.MonitoringConfig{
			EnhancedHealth:    true,
			CloudWatchMetrics: true,
			CloudWatchLogs: &manifest.CloudWatchLogsConfig{
				Enabled:       true,
				RetentionDays: 30,
				StreamLogs:    true,
			},
		},
		EnvironmentVariables: map[string]string{
			"NODE_ENV": "production",
			"PORT":     "8080",
		},
	}

	settings := provider.buildOptionSettings(m)

	// Verify we have settings
	if len(settings) == 0 {
		t.Error("Expected settings to be generated")
	}

	// Verify critical settings are present
	foundInstanceType := false
	foundEnvType := false
	foundInstanceProfile := false
	foundHealthPath := false
	foundEnhancedHealth := false
	foundMonitoring := false
	foundLogs := false
	foundEnvVars := 0

	for _, setting := range settings {
		namespace := *setting.Namespace
		optionName := *setting.OptionName
		value := *setting.Value

		if namespace == "aws:autoscaling:launchconfiguration" && optionName == "InstanceType" {
			if value != "t3.large" {
				t.Errorf("Expected InstanceType 't3.large', got '%s'", value)
			}
			foundInstanceType = true
		}

		if namespace == "aws:elasticbeanstalk:environment" && optionName == "EnvironmentType" {
			if value != "LoadBalanced" {
				t.Errorf("Expected EnvironmentType 'LoadBalanced', got '%s'", value)
			}
			foundEnvType = true
		}

		if namespace == "aws:autoscaling:launchconfiguration" && optionName == "IamInstanceProfile" {
			if value != "my-profile" {
				t.Errorf("Expected IamInstanceProfile 'my-profile', got '%s'", value)
			}
			foundInstanceProfile = true
		}

		if namespace == "aws:elasticbeanstalk:application" && optionName == "Application Healthcheck URL" {
			if value != "/api/health" {
				t.Errorf("Expected health path '/api/health', got '%s'", value)
			}
			foundHealthPath = true
		}

		if namespace == "aws:elasticbeanstalk:healthreporting:system" && optionName == "SystemType" {
			if value != "enhanced" {
				t.Errorf("Expected SystemType 'enhanced', got '%s'", value)
			}
			foundEnhancedHealth = true
		}

		if namespace == "aws:autoscaling:launchconfiguration" && optionName == "MonitoringInterval" {
			foundMonitoring = true
		}

		if namespace == "aws:elasticbeanstalk:cloudwatch:logs" && optionName == "StreamLogs" {
			foundLogs = true
		}

		if namespace == "aws:elasticbeanstalk:application:environment" {
			foundEnvVars++
		}
	}

	if !foundInstanceType {
		t.Error("InstanceType setting not found")
	}
	if !foundEnvType {
		t.Error("EnvironmentType setting not found")
	}
	if !foundInstanceProfile {
		t.Error("IamInstanceProfile setting not found")
	}
	if !foundHealthPath {
		t.Error("Health check path setting not found")
	}
	if !foundEnhancedHealth {
		t.Error("Enhanced health setting not found")
	}
	if !foundMonitoring {
		t.Error("Monitoring setting not found")
	}
	if !foundLogs {
		t.Error("CloudWatch Logs setting not found")
	}
	if foundEnvVars != 2 {
		t.Errorf("Expected 2 environment variables, found %d", foundEnvVars)
	}
}
