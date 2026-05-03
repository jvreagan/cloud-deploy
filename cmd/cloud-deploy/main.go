package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

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
		timeout      = flag.Duration("timeout", 30*time.Minute, "Maximum time for the operation to complete")
		showVersion  = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	if *showVersion {
		logging.Infof("cloud-deploy version %s", version)
		logging.Infof("  commit: %s", commit)
		logging.Infof("  built: %s", date)
		os.Exit(0)
	}

	// Load and parse manifest
	m, err := manifest.Load(*manifestFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading manifest: %v\n", err)
		os.Exit(1)
	}

	// Set up context with timeout and signal handling
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	sigCtx, sigCancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer sigCancel()
	ctx = sigCtx

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
		logging.Info("✓ Deployment successful!")
		logging.Infof("  Application: %s", result.ApplicationName)
		logging.Infof("  Environment: %s", result.EnvironmentName)
		logging.Infof("  URL: %s", result.URL)
		logging.Infof("  Status: %s", result.Status)

	case "stop":
		logging.Info("Stopping deployment...")
		if err := p.Stop(ctx, m); err != nil {
			fmt.Fprintf(os.Stderr, "Stop failed: %v\n", err)
			os.Exit(1)
		}
		logging.Info("✓ Deployment stopped successfully")

	case "destroy":
		logging.Info("Destroying deployment...")
		if err := p.Destroy(ctx, m); err != nil {
			fmt.Fprintf(os.Stderr, "Destroy failed: %v\n", err)
			os.Exit(1)
		}
		logging.Info("✓ Deployment destroyed successfully")

	case "status":
		status, err := p.Status(ctx, m)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get status: %v\n", err)
			os.Exit(1)
		}
		logging.Info("Deployment Status:")
		logging.Infof("  Application: %s", status.ApplicationName)
		logging.Infof("  Environment: %s", status.EnvironmentName)
		logging.Infof("  Status: %s", status.Status)
		logging.Infof("  Health: %s", status.Health)
		logging.Infof("  URL: %s", status.URL)
		logging.Infof("  Last Updated: %s", status.LastUpdated)

	case "rollback":
		logging.Info("Rolling back deployment...")
		result, err := p.Rollback(ctx, m)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Rollback failed: %v\n", err)
			os.Exit(1)
		}
		logging.Info("✓ Rollback successful!")
		logging.Infof("  Application: %s", result.ApplicationName)
		logging.Infof("  Environment: %s", result.EnvironmentName)
		logging.Infof("  URL: %s", result.URL)
		logging.Infof("  Status: %s", result.Status)
		logging.Infof("  Message: %s", result.Message)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", *command)
		fmt.Fprintf(os.Stderr, "Valid commands: deploy, stop, destroy, status, rollback\n")
		os.Exit(1)
	}
}
