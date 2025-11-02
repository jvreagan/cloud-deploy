// Package aws provides an AWS Elastic Beanstalk provider implementation
// that uses the AWS SDK for Go to deploy containerized applications.
package aws

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk"
	ebtypes "github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/jvreagan/cloud-deploy/pkg/manifest"
	"github.com/jvreagan/cloud-deploy/pkg/registry"
	"github.com/jvreagan/cloud-deploy/pkg/types"
)

// Provider implements the provider.Provider interface for AWS Elastic Beanstalk.
type Provider struct {
	ebClient *elasticbeanstalk.Client
	s3Client *s3.Client
	region   string
	config   aws.Config
}

// New creates a new AWS provider instance with the specified region and optional credentials.
// If credentials are provided in the manifest, they will be used.
// Otherwise, it falls back to the AWS SDK default credential chain (environment variables,
// shared credentials file, or IAM role).
func New(ctx context.Context, region string, creds *manifest.CredentialsConfig) (*Provider, error) {
	var cfg aws.Config
	var err error

	// If credentials are provided in the manifest, use them
	if creds != nil && creds.AccessKeyID != "" && creds.SecretAccessKey != "" {
		fmt.Println("Using credentials from manifest")
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				creds.AccessKeyID,
				creds.SecretAccessKey,
				"", // session token (optional)
			)),
		)
	} else {
		// Fall back to default credential chain
		fmt.Println("Using AWS default credential chain")
		cfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(region))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &Provider{
		ebClient: elasticbeanstalk.NewFromConfig(cfg),
		s3Client: s3.NewFromConfig(cfg),
		region:   region,
		config:   cfg,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "aws"
}

// Deploy deploys an application to AWS Elastic Beanstalk.
func (p *Provider) Deploy(ctx context.Context, m *manifest.Manifest) (*types.DeploymentResult, error) {
	fmt.Println("Starting AWS Elastic Beanstalk deployment...")

	// Step 0: Auto-detect solution stack if not specified
	if err := p.ensureSolutionStack(ctx, m); err != nil {
		return nil, fmt.Errorf("failed to determine solution stack: %w", err)
	}

	// Step 0.5: Fetch secrets from Vault if configured
	if m.Vault != nil && len(m.Secrets) > 0 {
		vaultSecrets, err := m.FetchVaultSecrets(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch vault secrets: %w", err)
		}

		// Merge Vault secrets with environment variables
		// Vault secrets take precedence over manifest environment variables
		if m.EnvironmentVariables == nil {
			m.EnvironmentVariables = make(map[string]string)
		}
		for key, value := range vaultSecrets {
			m.EnvironmentVariables[key] = value
		}
	}

	// Step 1: Create or verify application exists
	if err := p.ensureApplication(ctx, m); err != nil {
		return nil, fmt.Errorf("failed to ensure application: %w", err)
	}

	// Step 2: Push image to ECR
	fmt.Println("\n=== Pushing image to ECR ===")
	ecrRegistry, err := registry.NewECRRegistry(p.config, p.region, m.Application.Name, "latest")
	if err != nil {
		return nil, fmt.Errorf("failed to create ECR registry: %w", err)
	}

	if err := ecrRegistry.Authenticate(ctx); err != nil {
		return nil, fmt.Errorf("failed to authenticate with ECR: %w", err)
	}

	taggedImage, err := ecrRegistry.TagImage(ctx, m.Image)
	if err != nil {
		return nil, fmt.Errorf("failed to tag image for ECR: %w", err)
	}

	if err := ecrRegistry.PushImage(ctx, taggedImage); err != nil {
		return nil, fmt.Errorf("failed to push image to ECR: %w", err)
	}

	imageURI := ecrRegistry.GetImageURI()
	fmt.Printf("Image pushed to ECR: %s\n", imageURI)

	// Step 3: Create S3 bucket for application versions
	bucketName := fmt.Sprintf("elasticbeanstalk-%s-%s", p.region, m.Application.Name)
	if err := p.ensureBucket(ctx, bucketName); err != nil {
		return nil, fmt.Errorf("failed to ensure S3 bucket: %w", err)
	}

	// Step 4: Create and upload Dockerrun.aws.json
	versionLabel := fmt.Sprintf("v-%d", time.Now().Unix())
	s3Key := fmt.Sprintf("%s/%s.zip", m.Application.Name, versionLabel)

	if err := p.uploadDockerrun(ctx, imageURI, bucketName, s3Key); err != nil {
		return nil, fmt.Errorf("failed to upload Dockerrun.aws.json: %w", err)
	}

	// Step 4: Create application version
	if err := p.createApplicationVersion(ctx, m, versionLabel, bucketName, s3Key); err != nil {
		return nil, fmt.Errorf("failed to create application version: %w", err)
	}

	// Step 5: Create or update environment
	envExists, err := p.environmentExists(ctx, m.Application.Name, m.Environment.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check environment: %w", err)
	}

	if envExists {
		fmt.Printf("Updating existing environment: %s\n", m.Environment.Name)
		if err := p.updateEnvironment(ctx, m, versionLabel); err != nil {
			return nil, fmt.Errorf("failed to update environment: %w", err)
		}
	} else {
		fmt.Printf("Creating new environment: %s\n", m.Environment.Name)
		if err := p.createEnvironment(ctx, m, versionLabel); err != nil {
			return nil, fmt.Errorf("failed to create environment: %w", err)
		}
	}

	// Step 6: Wait for environment to be ready
	fmt.Println("Waiting for environment to be ready...")
	url, err := p.waitForEnvironment(ctx, m.Application.Name, m.Environment.Name)
	if err != nil {
		return nil, fmt.Errorf("environment deployment failed: %w", err)
	}

	return &types.DeploymentResult{
		ApplicationName: m.Application.Name,
		EnvironmentName: m.Environment.Name,
		URL:             url,
		Status:          "Ready",
		Message:         "Deployment successful",
	}, nil
}

// Destroy terminates an AWS Elastic Beanstalk environment and optionally the application.
func (p *Provider) Destroy(ctx context.Context, m *manifest.Manifest) error {
	fmt.Printf("Terminating environment: %s\n", m.Environment.Name)

	_, err := p.ebClient.TerminateEnvironment(ctx, &elasticbeanstalk.TerminateEnvironmentInput{
		EnvironmentName: aws.String(m.Environment.Name),
	})
	if err != nil {
		return fmt.Errorf("failed to terminate environment: %w", err)
	}

	fmt.Println("Waiting for environment termination...")
	if err := p.waitForEnvironmentTermination(ctx, m.Application.Name, m.Environment.Name); err != nil {
		return fmt.Errorf("failed to wait for termination: %w", err)
	}

	fmt.Println("Environment terminated successfully")
	return nil
}

// Stop stops the AWS Elastic Beanstalk environment but preserves the application and versions.
// This terminates all running resources (EC2 instances, load balancers, etc.) to stop costs,
// but keeps the application definition and version artifacts in S3 for fast redeployment.
func (p *Provider) Stop(ctx context.Context, m *manifest.Manifest) error {
	fmt.Printf("Stopping environment: %s\n", m.Environment.Name)
	fmt.Println("This will terminate all resources but preserve the application for fast restart.")

	_, err := p.ebClient.TerminateEnvironment(ctx, &elasticbeanstalk.TerminateEnvironmentInput{
		EnvironmentName: aws.String(m.Environment.Name),
	})
	if err != nil {
		return fmt.Errorf("failed to terminate environment: %w", err)
	}

	fmt.Println("Waiting for environment termination...")
	if err := p.waitForEnvironmentTermination(ctx, m.Application.Name, m.Environment.Name); err != nil {
		return fmt.Errorf("failed to wait for termination: %w", err)
	}

	fmt.Println("Environment stopped successfully")
	fmt.Printf("Application '%s' and versions are preserved in S3\n", m.Application.Name)
	fmt.Println("Run 'cloud-deploy -command deploy' to restart")
	return nil
}

// Status retrieves the current status of an AWS Elastic Beanstalk deployment.
func (p *Provider) Status(ctx context.Context, m *manifest.Manifest) (*types.DeploymentStatus, error) {
	result, err := p.ebClient.DescribeEnvironments(ctx, &elasticbeanstalk.DescribeEnvironmentsInput{
		ApplicationName:  aws.String(m.Application.Name),
		EnvironmentNames: []string{m.Environment.Name},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe environment: %w", err)
	}

	if len(result.Environments) == 0 {
		return nil, fmt.Errorf("environment not found: %s", m.Environment.Name)
	}

	env := result.Environments[0]
	url := ""
	if env.CNAME != nil {
		url = fmt.Sprintf("http://%s", *env.CNAME)
	}

	return &types.DeploymentStatus{
		ApplicationName: m.Application.Name,
		EnvironmentName: m.Environment.Name,
		Status:          string(env.Status),
		Health:          string(env.Health),
		URL:             url,
		LastUpdated:     env.DateUpdated.String(),
	}, nil
}

// ensureSolutionStack determines the solution stack to use.
// If already specified in manifest, it validates it exists.
// If not specified, it auto-detects the latest stack for the platform.
func (p *Provider) ensureSolutionStack(ctx context.Context, m *manifest.Manifest) error {
	// If already specified, validate and use it
	if m.Deployment.SolutionStack != "" {
		fmt.Printf("Using specified solution stack: %s\n", m.Deployment.SolutionStack)
		return nil
	}

	// Auto-detect based on platform
	fmt.Printf("Auto-detecting solution stack for platform: %s\n", m.Deployment.Platform)

	result, err := p.ebClient.ListAvailableSolutionStacks(ctx, &elasticbeanstalk.ListAvailableSolutionStacksInput{})
	if err != nil {
		return fmt.Errorf("failed to list solution stacks: %w", err)
	}

	// Filter for matching platform
	var candidates []string
	platformLower := strings.ToLower(m.Deployment.Platform)

	for _, stack := range result.SolutionStacks {
		stackLower := strings.ToLower(stack)
		// Look for stacks matching the platform on Amazon Linux 2023
		if strings.Contains(stackLower, platformLower) && strings.Contains(stackLower, "amazon linux 2023") {
			candidates = append(candidates, stack)
		}
	}

	if len(candidates) == 0 {
		return fmt.Errorf("no solution stack found for platform: %s", m.Deployment.Platform)
	}

	// Select the first one (AWS returns them in descending version order, so first = latest)
	m.Deployment.SolutionStack = candidates[0]
	fmt.Printf("Auto-selected solution stack: %s\n", m.Deployment.SolutionStack)

	return nil
}

// ensureApplication creates the application if it doesn't exist.
func (p *Provider) ensureApplication(ctx context.Context, m *manifest.Manifest) error {
	// Check if application exists
	result, err := p.ebClient.DescribeApplications(ctx, &elasticbeanstalk.DescribeApplicationsInput{
		ApplicationNames: []string{m.Application.Name},
	})
	if err != nil {
		return err
	}

	if len(result.Applications) > 0 {
		fmt.Printf("Application already exists: %s\n", m.Application.Name)
		return nil
	}

	// Create application
	fmt.Printf("Creating application: %s\n", m.Application.Name)
	_, err = p.ebClient.CreateApplication(ctx, &elasticbeanstalk.CreateApplicationInput{
		ApplicationName: aws.String(m.Application.Name),
		Description:     aws.String(m.Application.Description),
	})
	return err
}

// ensureBucket creates an S3 bucket if it doesn't exist.
func (p *Provider) ensureBucket(ctx context.Context, bucketName string) error {
	// Check if bucket exists
	_, err := p.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil {
		fmt.Printf("S3 bucket already exists: %s\n", bucketName)
		return nil
	}

	// Create bucket
	fmt.Printf("Creating S3 bucket: %s\n", bucketName)

	// For regions other than us-east-1, we need to specify LocationConstraint
	createBucketInput := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	if p.region != "us-east-1" {
		createBucketInput.CreateBucketConfiguration = &s3types.CreateBucketConfiguration{
			LocationConstraint: s3types.BucketLocationConstraint(p.region),
		}
	}

	_, err = p.s3Client.CreateBucket(ctx, createBucketInput)
	return err
}

// uploadSource zips the source directory and uploads it to S3.
func (p *Provider) uploadSource(ctx context.Context, sourcePath, bucketName, s3Key string) error {
	fmt.Println("Zipping source code...")

	// Create temporary zip file
	zipFile, err := os.CreateTemp("", "cloud-deploy-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(zipFile.Name())
	defer zipFile.Close()

	// Zip the source directory
	if err := zipDirectory(sourcePath, zipFile); err != nil {
		return fmt.Errorf("failed to zip directory: %w", err)
	}

	// Rewind to beginning of file
	if _, err := zipFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}

	// Upload to S3
	fmt.Printf("Uploading to S3: s3://%s/%s\n", bucketName, s3Key)
	_, err = p.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(s3Key),
		Body:   zipFile,
	})
	return err
}

// uploadDockerrun creates a Dockerrun.aws.json file for the ECR image and uploads it to S3.
func (p *Provider) uploadDockerrun(ctx context.Context, imageURI, bucketName, s3Key string) error {
	fmt.Println("Creating Dockerrun.aws.json...")

	// Create Dockerrun.aws.json structure
	dockerrun := map[string]interface{}{
		"AWSEBDockerrunVersion": "1",
		"Image": map[string]interface{}{
			"Name":   imageURI,
			"Update": "true",
		},
		"Ports": []map[string]interface{}{
			{
				"ContainerPort": 80,
				"HostPort":      80,
			},
			{
				"ContainerPort": 443,
				"HostPort":      443,
			},
		},
	}

	// Marshal to JSON
	dockerrunJSON, err := json.MarshalIndent(dockerrun, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal Dockerrun.aws.json: %w", err)
	}

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "cloud-deploy-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write Dockerrun.aws.json file
	dockerrunPath := filepath.Join(tmpDir, "Dockerrun.aws.json")
	if err := os.WriteFile(dockerrunPath, dockerrunJSON, 0644); err != nil {
		return fmt.Errorf("failed to write Dockerrun.aws.json: %w", err)
	}

	fmt.Printf("Dockerrun.aws.json created with image: %s\n", imageURI)

	// Create temporary zip file
	zipFile, err := os.CreateTemp("", "cloud-deploy-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(zipFile.Name())
	defer zipFile.Close()

	// Zip the Dockerrun.aws.json file
	if err := zipDirectory(tmpDir, zipFile); err != nil {
		return fmt.Errorf("failed to zip Dockerrun.aws.json: %w", err)
	}

	// Rewind to beginning of file
	if _, err := zipFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}

	// Upload to S3
	fmt.Printf("Uploading Dockerrun.aws.json to S3: s3://%s/%s\n", bucketName, s3Key)
	_, err = p.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(s3Key),
		Body:   zipFile,
	})
	return err
}

// createApplicationVersion creates a new application version.
func (p *Provider) createApplicationVersion(ctx context.Context, m *manifest.Manifest, versionLabel, bucketName, s3Key string) error {
	fmt.Printf("Creating application version: %s\n", versionLabel)

	_, err := p.ebClient.CreateApplicationVersion(ctx, &elasticbeanstalk.CreateApplicationVersionInput{
		ApplicationName: aws.String(m.Application.Name),
		VersionLabel:    aws.String(versionLabel),
		Description:     aws.String(fmt.Sprintf("Deployed by cloud-deploy at %s", time.Now().Format(time.RFC3339))),
		SourceBundle: &ebtypes.S3Location{
			S3Bucket: aws.String(bucketName),
			S3Key:    aws.String(s3Key),
		},
	})
	return err
}

// environmentExists checks if an environment exists.
func (p *Provider) environmentExists(ctx context.Context, appName, envName string) (bool, error) {
	result, err := p.ebClient.DescribeEnvironments(ctx, &elasticbeanstalk.DescribeEnvironmentsInput{
		ApplicationName:  aws.String(appName),
		EnvironmentNames: []string{envName},
	})
	if err != nil {
		return false, err
	}

	// Check if environment exists and is not terminated
	for _, env := range result.Environments {
		if *env.EnvironmentName == envName && env.Status != ebtypes.EnvironmentStatusTerminated {
			return true, nil
		}
	}

	return false, nil
}

// createEnvironment creates a new Elastic Beanstalk environment.
func (p *Provider) createEnvironment(ctx context.Context, m *manifest.Manifest, versionLabel string) error {
	optionSettings := p.buildOptionSettings(m)

	_, err := p.ebClient.CreateEnvironment(ctx, &elasticbeanstalk.CreateEnvironmentInput{
		ApplicationName:   aws.String(m.Application.Name),
		EnvironmentName:   aws.String(m.Environment.Name),
		VersionLabel:      aws.String(versionLabel),
		SolutionStackName: aws.String(m.Deployment.SolutionStack),
		CNAMEPrefix:       aws.String(m.Environment.CName),
		OptionSettings:    optionSettings,
	})
	return err
}

// updateEnvironment updates an existing environment with a new version and configuration.
func (p *Provider) updateEnvironment(ctx context.Context, m *manifest.Manifest, versionLabel string) error {
	// Build option settings from manifest to apply configuration changes
	optionSettings := p.buildOptionSettings(m)

	_, err := p.ebClient.UpdateEnvironment(ctx, &elasticbeanstalk.UpdateEnvironmentInput{
		EnvironmentName: aws.String(m.Environment.Name),
		VersionLabel:    aws.String(versionLabel),
		OptionSettings:  optionSettings,
	})
	return err
}

// buildOptionSettings constructs the Elastic Beanstalk option settings from the manifest.
func (p *Provider) buildOptionSettings(m *manifest.Manifest) []ebtypes.ConfigurationOptionSetting {
	settings := []ebtypes.ConfigurationOptionSetting{
		{
			Namespace:  aws.String("aws:autoscaling:launchconfiguration"),
			OptionName: aws.String("InstanceType"),
			Value:      aws.String(m.Instance.Type),
		},
		{
			Namespace:  aws.String("aws:elasticbeanstalk:environment"),
			OptionName: aws.String("EnvironmentType"),
			Value:      aws.String(m.Instance.EnvironmentType),
		},
	}

	// Add IAM instance profile if specified
	if m.IAM.InstanceProfile != "" {
		settings = append(settings, ebtypes.ConfigurationOptionSetting{
			Namespace:  aws.String("aws:autoscaling:launchconfiguration"),
			OptionName: aws.String("IamInstanceProfile"),
			Value:      aws.String(m.IAM.InstanceProfile),
		})
	}

	// Add health check settings
	if m.HealthCheck.Path != "" {
		settings = append(settings, ebtypes.ConfigurationOptionSetting{
			Namespace:  aws.String("aws:elasticbeanstalk:application"),
			OptionName: aws.String("Application Healthcheck URL"),
			Value:      aws.String(m.HealthCheck.Path),
		})
	}

	// Add enhanced health reporting if enabled (or if health check type is "enhanced")
	if m.Monitoring.EnhancedHealth || m.HealthCheck.Type == "enhanced" {
		settings = append(settings, ebtypes.ConfigurationOptionSetting{
			Namespace:  aws.String("aws:elasticbeanstalk:healthreporting:system"),
			OptionName: aws.String("SystemType"),
			Value:      aws.String("enhanced"),
		})
	}

	// Add CloudWatch metrics collection if enabled
	if m.Monitoring.CloudWatchMetrics {
		// Enable detailed CloudWatch monitoring for instances
		settings = append(settings, ebtypes.ConfigurationOptionSetting{
			Namespace:  aws.String("aws:autoscaling:launchconfiguration"),
			OptionName: aws.String("MonitoringInterval"),
			Value:      aws.String("1 minute"),
		})
	}

	// Add CloudWatch Logs streaming if configured
	if m.Monitoring.CloudWatchLogs != nil && m.Monitoring.CloudWatchLogs.Enabled {
		settings = append(settings, ebtypes.ConfigurationOptionSetting{
			Namespace:  aws.String("aws:elasticbeanstalk:cloudwatch:logs"),
			OptionName: aws.String("StreamLogs"),
			Value:      aws.String("true"),
		})

		// Set log retention if specified
		if m.Monitoring.CloudWatchLogs.RetentionDays > 0 {
			settings = append(settings, ebtypes.ConfigurationOptionSetting{
				Namespace:  aws.String("aws:elasticbeanstalk:cloudwatch:logs"),
				OptionName: aws.String("RetentionInDays"),
				Value:      aws.String(fmt.Sprintf("%d", m.Monitoring.CloudWatchLogs.RetentionDays)),
			})
		}

		// Stream application logs if configured (default is true when logs are enabled)
		if m.Monitoring.CloudWatchLogs.StreamLogs {
			settings = append(settings, ebtypes.ConfigurationOptionSetting{
				Namespace:  aws.String("aws:elasticbeanstalk:cloudwatch:logs:health"),
				OptionName: aws.String("HealthStreamingEnabled"),
				Value:      aws.String("true"),
			})
		}
	}

	// Add environment variables
	for key, value := range m.EnvironmentVariables {
		settings = append(settings, ebtypes.ConfigurationOptionSetting{
			Namespace:  aws.String("aws:elasticbeanstalk:application:environment"),
			OptionName: aws.String(key),
			Value:      aws.String(value),
		})
	}

	return settings
}

// waitForEnvironment waits for the environment to become ready and returns its URL.
func (p *Provider) waitForEnvironment(ctx context.Context, appName, envName string) (string, error) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	timeout := time.After(15 * time.Minute)

	for {
		select {
		case <-timeout:
			return "", fmt.Errorf("timeout waiting for environment to be ready")
		case <-ticker.C:
			result, err := p.ebClient.DescribeEnvironments(ctx, &elasticbeanstalk.DescribeEnvironmentsInput{
				ApplicationName:  aws.String(appName),
				EnvironmentNames: []string{envName},
			})
			if err != nil {
				return "", err
			}

			if len(result.Environments) == 0 {
				return "", fmt.Errorf("environment disappeared")
			}

			env := result.Environments[0]
			fmt.Printf("Environment status: %s, Health: %s\n", env.Status, env.Health)

			if env.Status == ebtypes.EnvironmentStatusReady {
				if env.CNAME != nil {
					return fmt.Sprintf("http://%s", *env.CNAME), nil
				}
				return "", fmt.Errorf("environment ready but no CNAME")
			}

			if env.Status == ebtypes.EnvironmentStatusTerminated || env.Status == ebtypes.EnvironmentStatusTerminating {
				return "", fmt.Errorf("environment failed: status=%s", env.Status)
			}
		}
	}
}

// waitForEnvironmentTermination waits for the environment to be terminated.
func (p *Provider) waitForEnvironmentTermination(ctx context.Context, appName, envName string) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	timeout := time.After(10 * time.Minute)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for environment termination")
		case <-ticker.C:
			result, err := p.ebClient.DescribeEnvironments(ctx, &elasticbeanstalk.DescribeEnvironmentsInput{
				ApplicationName:  aws.String(appName),
				EnvironmentNames: []string{envName},
			})
			if err != nil {
				return err
			}

			if len(result.Environments) == 0 {
				fmt.Println("Environment terminated")
				return nil
			}

			env := result.Environments[0]
			if env.Status == ebtypes.EnvironmentStatusTerminated {
				fmt.Println("Environment terminated")
				return nil
			}

			fmt.Printf("Termination status: %s\n", env.Status)
		}
	}
}

// zipDirectory creates a zip archive of a directory.
func zipDirectory(sourceDir string, zipFile *os.File) error {
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and certain files
		if info.IsDir() {
			return nil
		}

		// Skip hidden files and common excludes
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Create zip entry
		writer, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		// Copy file content
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}

// Rollback rolls back the AWS Elastic Beanstalk environment to the previous application version.
func (p *Provider) Rollback(ctx context.Context, m *manifest.Manifest) (*types.DeploymentResult, error) {
	fmt.Println("Starting AWS Elastic Beanstalk rollback...")

	// Step 1: Get current environment to find the deployed version
	envResult, err := p.ebClient.DescribeEnvironments(ctx, &elasticbeanstalk.DescribeEnvironmentsInput{
		ApplicationName:  aws.String(m.Application.Name),
		EnvironmentNames: []string{m.Environment.Name},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe environment: %w", err)
	}

	if len(envResult.Environments) == 0 {
		return nil, fmt.Errorf("environment not found: %s", m.Environment.Name)
	}

	currentVersion := envResult.Environments[0].VersionLabel
	if currentVersion == nil {
		return nil, fmt.Errorf("current environment has no version label")
	}

	fmt.Printf("Current version: %s\n", *currentVersion)

	// Step 2: List all application versions (sorted by creation date)
	versionsResult, err := p.ebClient.DescribeApplicationVersions(ctx, &elasticbeanstalk.DescribeApplicationVersionsInput{
		ApplicationName: aws.String(m.Application.Name),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list application versions: %w", err)
	}

	if len(versionsResult.ApplicationVersions) < 2 {
		return nil, fmt.Errorf("no previous version available to rollback to (only %d version(s) exist)", len(versionsResult.ApplicationVersions))
	}

	// Step 3: Find the previous version (the one before the current)
	var previousVersion *string
	var currentVersionDate *time.Time

	// First, find the current version's creation date
	for _, version := range versionsResult.ApplicationVersions {
		if version.VersionLabel != nil && *version.VersionLabel == *currentVersion {
			currentVersionDate = version.DateCreated
			break
		}
	}

	if currentVersionDate == nil {
		return nil, fmt.Errorf("could not find current version in version list")
	}

	// Find the most recent version that was created before the current version
	for _, version := range versionsResult.ApplicationVersions {
		if version.VersionLabel == nil || version.DateCreated == nil {
			continue
		}

		// Skip the current version
		if *version.VersionLabel == *currentVersion {
			continue
		}

		// Find versions created before the current one
		if version.DateCreated.Before(*currentVersionDate) {
			// If we haven't found a previous version yet, or this one is more recent than what we found
			if previousVersion == nil {
				previousVersion = version.VersionLabel
				currentVersionDate = version.DateCreated
			} else {
				// Find the most recent version before current
				for _, v := range versionsResult.ApplicationVersions {
					if v.VersionLabel != nil && *v.VersionLabel == *previousVersion {
						if version.DateCreated.After(*v.DateCreated) {
							previousVersion = version.VersionLabel
						}
						break
					}
				}
			}
		}
	}

	if previousVersion == nil {
		return nil, fmt.Errorf("no previous version found to rollback to")
	}

	fmt.Printf("Rolling back to previous version: %s\n", *previousVersion)

	// Step 4: Update environment to use the previous version
	_, err = p.ebClient.UpdateEnvironment(ctx, &elasticbeanstalk.UpdateEnvironmentInput{
		EnvironmentName: aws.String(m.Environment.Name),
		VersionLabel:    previousVersion,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to rollback environment: %w", err)
	}

	// Step 5: Wait for environment to be ready
	fmt.Println("Waiting for rollback to complete...")
	url, err := p.waitForEnvironment(ctx, m.Application.Name, m.Environment.Name)
	if err != nil {
		return nil, fmt.Errorf("rollback failed: %w", err)
	}

	return &types.DeploymentResult{
		ApplicationName: m.Application.Name,
		EnvironmentName: m.Environment.Name,
		URL:             url,
		Status:          "Ready",
		Message:         fmt.Sprintf("Rolled back to version %s", *previousVersion),
	}, nil
}
