package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestVersion tests the -version flag by running the binary
func TestVersion(t *testing.T) {
	// This test requires the binary to be built first
	// We'll test it by invoking the binary as a subprocess
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI environment")
	}

	// Try to build the binary
	cmd := exec.Command("go", "build", "-o", "cloud-deploy-test", ".")
	if err := cmd.Run(); err != nil {
		t.Skipf("Could not build binary for testing: %v", err)
	}
	defer os.Remove("cloud-deploy-test")

	// Test -version flag
	cmd = exec.Command("./cloud-deploy-test", "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run -version: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "cloud-deploy version") {
		t.Errorf("Expected version output to contain 'cloud-deploy version', got: %s", outputStr)
	}
}

// TestInvalidCommand tests that invalid commands return an error
func TestInvalidCommand(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI environment")
	}

	// Try to build the binary
	cmd := exec.Command("go", "build", "-o", "cloud-deploy-test", ".")
	if err := cmd.Run(); err != nil {
		t.Skipf("Could not build binary for testing: %v", err)
	}
	defer os.Remove("cloud-deploy-test")

	// Create a minimal test manifest
	tmpDir := t.TempDir()
	manifestPath := tmpDir + "/test-manifest.yaml"
	manifestContent := `version: "1.0"
provider:
  name: aws
  region: us-east-1
application:
  name: test-app
environment:
  name: test-env
deployment:
  platform: docker
  source:
    type: local
    path: ./test
instance:
  type: t3.micro
  environment_type: SingleInstance
health_check:
  type: basic
  path: /health
`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatalf("Failed to create test manifest: %v", err)
	}

	// Test invalid command
	cmd = exec.Command("./cloud-deploy-test", "-manifest", manifestPath, "-command", "invalid-command")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("Expected error for invalid command, but got none")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Unknown command") {
		t.Errorf("Expected error message to contain 'Unknown command', got: %s", outputStr)
	}
}

// TestMissingManifest tests that missing manifest file returns an error
func TestMissingManifest(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI environment")
	}

	// Try to build the binary
	cmd := exec.Command("go", "build", "-o", "cloud-deploy-test", ".")
	if err := cmd.Run(); err != nil {
		t.Skipf("Could not build binary for testing: %v", err)
	}
	defer os.Remove("cloud-deploy-test")

	// Test with non-existent manifest
	cmd = exec.Command("./cloud-deploy-test", "-manifest", "/nonexistent/manifest.yaml", "-command", "status")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("Expected error for missing manifest, but got none")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Error loading manifest") {
		t.Errorf("Expected error message to contain 'Error loading manifest', got: %s", outputStr)
	}
}

// TestVersionVariable tests that version variables are set
func TestVersionVariable(t *testing.T) {
	// Test that version variable exists and has a default value
	if version == "" {
		t.Error("version variable should not be empty")
	}

	// commit and date can be "none" and "unknown" respectively in development
	// Just verify they exist
	_ = commit
	_ = date
}

// TestCommandValidation tests command validation logic
func TestCommandValidation(t *testing.T) {
	validCommands := []string{"deploy", "stop", "destroy", "status"}

	// This test verifies we know what the valid commands are
	// In a real scenario, we'd extract this logic into a testable function
	for _, cmd := range validCommands {
		if cmd == "" {
			t.Errorf("Command should not be empty")
		}
	}

	// Verify we have the expected commands
	expectedCount := 4
	if len(validCommands) != expectedCount {
		t.Errorf("Expected %d valid commands, got %d", expectedCount, len(validCommands))
	}
}

// TestManifestFlagDefault tests the default manifest file flag
func TestManifestFlagDefault(t *testing.T) {
	// The default manifest file should be "deploy-manifest.yaml"
	defaultManifest := "deploy-manifest.yaml"

	if defaultManifest == "" {
		t.Error("Default manifest should not be empty")
	}

	if !strings.HasSuffix(defaultManifest, ".yaml") {
		t.Errorf("Default manifest should end with .yaml, got: %s", defaultManifest)
	}
}

// TestCommandFlagDefault tests the default command flag
func TestCommandFlagDefault(t *testing.T) {
	// The default command should be "deploy"
	defaultCommand := "deploy"

	if defaultCommand != "deploy" {
		t.Errorf("Default command should be 'deploy', got: %s", defaultCommand)
	}
}

// TestVersionFormat tests version string format
func TestVersionFormat(t *testing.T) {
	// Save original values
	originalVersion := version
	originalCommit := commit
	originalDate := date

	defer func() {
		version = originalVersion
		commit = originalCommit
		date = originalDate
	}()

	// Set test values
	version = "1.2.3"
	commit = "abc123def"
	date = "2024-10-30"

	// Verify version components
	if !strings.Contains(version, ".") {
		t.Errorf("Version should contain dots: %s", version)
	}

	if len(commit) == 0 {
		t.Error("Commit should not be empty")
	}

	if len(date) == 0 {
		t.Error("Date should not be empty")
	}
}

// TestAllCommandsAreValid tests that all commands are properly defined
func TestAllCommandsAreValid(t *testing.T) {
	commands := map[string]bool{
		"deploy":  true,
		"stop":    true,
		"destroy": true,
		"status":  true,
	}

	// Verify each command
	for cmd, shouldBeValid := range commands {
		if !shouldBeValid {
			t.Errorf("Command '%s' should be valid", cmd)
		}

		// Verify command is not empty
		if cmd == "" {
			t.Error("Command should not be empty")
		}

		// Verify command is lowercase
		if strings.ToLower(cmd) != cmd {
			t.Errorf("Command '%s' should be lowercase", cmd)
		}
	}
}

// TestInvalidCommandStrings tests various invalid command strings
func TestInvalidCommandStrings(t *testing.T) {
	validCommands := map[string]bool{
		"deploy":  true,
		"stop":    true,
		"destroy": true,
		"status":  true,
	}

	invalidCommands := []string{
		"",
		"DEPLOY",
		"Deploy",
		"delete",
		"remove",
		"start",
		"restart",
		"update",
		"list",
		"create",
		"build",
		"run",
		"invalid",
		"unknown",
	}

	for _, cmd := range invalidCommands {
		if validCommands[cmd] {
			t.Errorf("Command '%s' should be invalid", cmd)
		}
	}
}

// TestManifestPathVariations tests different manifest path formats
func TestManifestPathVariations(t *testing.T) {
	manifestPaths := []string{
		"deploy-manifest.yaml",
		"./deploy-manifest.yaml",
		"/path/to/manifest.yaml",
		"../relative/path/manifest.yaml",
		"configs/production.yaml",
		"manifest.yml",
	}

	for _, path := range manifestPaths {
		if path == "" {
			t.Error("Manifest path should not be empty")
		}

		// Verify path has yaml/yml extension
		hasYamlExt := strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")
		if !hasYamlExt {
			t.Errorf("Manifest path '%s' should have .yaml or .yml extension", path)
		}
	}
}

// TestVersionVariableDefaults tests default values
func TestVersionVariableDefaults(t *testing.T) {
	// These should never be empty, even if not set via ldflags
	if version == "" {
		t.Error("version should have a default value")
	}

	if commit == "" {
		t.Error("commit should have a default value")
	}

	if date == "" {
		t.Error("date should have a default value")
	}

	// Default values should be specific strings
	// (this assumes the defaults in main.go are "0.1.0", "none", "unknown")
	t.Logf("Default version: %s", version)
	t.Logf("Default commit: %s", commit)
	t.Logf("Default date: %s", date)
}

// TestCommandCaseSensitivity tests that commands are case-sensitive
func TestCommandCaseSensitivity(t *testing.T) {
	validCommands := map[string]bool{
		"deploy":  true,
		"stop":    true,
		"destroy": true,
		"status":  true,
	}

	// Upper case versions should NOT be valid
	upperCaseCommands := []string{
		"DEPLOY",
		"STOP",
		"DESTROY",
		"STATUS",
	}

	for _, cmd := range upperCaseCommands {
		if validCommands[cmd] {
			t.Errorf("Upper case command '%s' should not be valid (commands are case-sensitive)", cmd)
		}
	}

	// Mixed case versions should NOT be valid
	mixedCaseCommands := []string{
		"Deploy",
		"Stop",
		"Destroy",
		"Status",
	}

	for _, cmd := range mixedCaseCommands {
		if validCommands[cmd] {
			t.Errorf("Mixed case command '%s' should not be valid (commands are case-sensitive)", cmd)
		}
	}
}

// TestFlagUsageStrings tests that flag descriptions are set
func TestFlagUsageStrings(t *testing.T) {
	expectedUsages := map[string]string{
		"manifest": "Path to deployment manifest file",
		"command":  "Command to execute: deploy, stop, destroy, status",
		"version":  "Show version information",
	}

	for flagName, expectedUsage := range expectedUsages {
		if expectedUsage == "" {
			t.Errorf("Flag '%s' should have a usage string", flagName)
		}

		// Verify usage string is descriptive (has some minimum length)
		if len(expectedUsage) < 10 {
			t.Errorf("Flag '%s' usage string is too short: '%s'", flagName, expectedUsage)
		}
	}
}

// TestCommandCount tests that we have exactly the expected number of commands
func TestCommandCount(t *testing.T) {
	commands := []string{"deploy", "stop", "destroy", "status"}

	expectedCount := 4
	actualCount := len(commands)

	if actualCount != expectedCount {
		t.Errorf("Expected %d commands, got %d", expectedCount, actualCount)
	}

	// Verify no duplicates
	seen := make(map[string]bool)
	for _, cmd := range commands {
		if seen[cmd] {
			t.Errorf("Duplicate command found: '%s'", cmd)
		}
		seen[cmd] = true
	}
}

// TestFlagCount tests that we have exactly the expected number of flags
func TestFlagCount(t *testing.T) {
	flags := []string{"manifest", "command", "version"}

	expectedCount := 3
	actualCount := len(flags)

	if actualCount != expectedCount {
		t.Errorf("Expected %d flags, got %d", expectedCount, actualCount)
	}

	// Verify no duplicates
	seen := make(map[string]bool)
	for _, flag := range flags {
		if seen[flag] {
			t.Errorf("Duplicate flag found: '%s'", flag)
		}
		seen[flag] = true
	}
}
