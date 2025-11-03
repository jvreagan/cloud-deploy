package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/jvreagan/cloud-deploy/pkg/logging"
	"github.com/jvreagan/cloud-deploy/pkg/manifest"
	"github.com/jvreagan/cloud-deploy/pkg/provider"
)

// Version information (set via ldflags during build)
var (
	version = "0.1.0"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Parse command line flags
	var (
		manifestFile = flag.String("manifest", "deploy-manifest.yaml", "Path to deployment manifest file")
		command      = flag.String("command", "deploy", "Command to execute: deploy, stop, destroy, status, rollback")
		showVersion  = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	if *showVersion {
		logging.Info("cloud-deploy version %s\n", version)
		logging.Info("  commit: %s\n", commit)
		logging.Info("  built: %s\n", date)
		os.Exit(0)
	}

	// Load and parse manifest
	m, err := manifest.Load(*manifestFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading manifest: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Create provider
	p, err := provider.Factory(ctx, m)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating provider: %v\n", err)
		os.Exit(1)
	}

	// Execute command
	switch *command {
	case "deploy":
		result, err := p.Deploy(ctx, m)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Deployment failed: %v\n", err)
			os.Exit(1)
		}
		logging.Info("✓ Deployment successful!\n")
		logging.Info("  Application: %s\n", result.ApplicationName)
		logging.Info("  Environment: %s\n", result.EnvironmentName)
		logging.Info("  URL: %s\n", result.URL)
		logging.Info("  Status: %s\n", result.Status)

	case "stop":
		logging.Info("Stopping deployment...\n")
		if err := p.Stop(ctx, m); err != nil {
			fmt.Fprintf(os.Stderr, "Stop failed: %v\n", err)
			os.Exit(1)
		}
		logging.Info("✓ Deployment stopped successfully\n")

	case "destroy":
		logging.Info("Destroying deployment...\n")
		if err := p.Destroy(ctx, m); err != nil {
			fmt.Fprintf(os.Stderr, "Destroy failed: %v\n", err)
			os.Exit(1)
		}
		logging.Info("✓ Deployment destroyed successfully\n")

	case "status":
		status, err := p.Status(ctx, m)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get status: %v\n", err)
			os.Exit(1)
		}
		logging.Info("Deployment Status:\n")
		logging.Info("  Application: %s\n", status.ApplicationName)
		logging.Info("  Environment: %s\n", status.EnvironmentName)
		logging.Info("  Status: %s\n", status.Status)
		logging.Info("  Health: %s\n", status.Health)
		logging.Info("  URL: %s\n", status.URL)
		logging.Info("  Last Updated: %s\n", status.LastUpdated)

	case "rollback":
		logging.Info("Rolling back deployment...\n")
		result, err := p.Rollback(ctx, m)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Rollback failed: %v\n", err)
			os.Exit(1)
		}
		logging.Info("✓ Rollback successful!\n")
		logging.Info("  Application: %s\n", result.ApplicationName)
		logging.Info("  Environment: %s\n", result.EnvironmentName)
		logging.Info("  URL: %s\n", result.URL)
		logging.Info("  Status: %s\n", result.Status)
		logging.Info("  Message: %s\n", result.Message)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", *command)
		fmt.Fprintf(os.Stderr, "Valid commands: deploy, stop, destroy, status, rollback\n")
		os.Exit(1)
	}
}
