package registry

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// ECRRegistry represents an AWS Elastic Container Registry
type ECRRegistry struct {
	config        aws.Config
	region        string
	repositoryName string
	accountID     string
	imageTag      string
	registryURL   string
	imageURI      string
}

// NewECRRegistry creates a new ECR registry handler
func NewECRRegistry(config aws.Config, region, repositoryName, imageTag string) (*ECRRegistry, error) {
	return &ECRRegistry{
		config:         config,
		region:         region,
		repositoryName: repositoryName,
		imageTag:       imageTag,
	}, nil
}

// GetRegistryURL returns the ECR registry URL
func (e *ECRRegistry) GetRegistryURL() string {
	return e.registryURL
}

// GetImageURI returns the full image URI in ECR
func (e *ECRRegistry) GetImageURI() string {
	return e.imageURI
}

// Authenticate authenticates Docker with ECR
func (e *ECRRegistry) Authenticate(ctx context.Context) error {
	// Get AWS account ID
	stsClient := sts.NewFromConfig(e.config)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("failed to get AWS account ID: %w", err)
	}
	e.accountID = *identity.Account

	// Build registry URL
	e.registryURL = fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", e.accountID, e.region)

	// Create ECR client
	ecrClient := ecr.NewFromConfig(e.config)

	// Create repository if it doesn't exist
	fmt.Printf("Ensuring ECR repository exists: %s\n", e.repositoryName)
	_, err = ecrClient.CreateRepository(ctx, &ecr.CreateRepositoryInput{
		RepositoryName: aws.String(e.repositoryName),
	})
	if err != nil {
		// Ignore error if repository already exists
		if !strings.Contains(err.Error(), "RepositoryAlreadyExistsException") {
			return fmt.Errorf("failed to create ECR repository: %w", err)
		}
		fmt.Printf("Repository %s already exists\n", e.repositoryName)
	} else {
		fmt.Printf("Created ECR repository: %s\n", e.repositoryName)
	}

	// Get authorization token
	authOutput, err := ecrClient.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return fmt.Errorf("failed to get ECR authorization token: %w", err)
	}

	if len(authOutput.AuthorizationData) == 0 {
		return fmt.Errorf("no authorization data returned from ECR")
	}

	// Decode the authorization token
	authToken := *authOutput.AuthorizationData[0].AuthorizationToken
	decodedToken, err := base64.StdEncoding.DecodeString(authToken)
	if err != nil {
		return fmt.Errorf("failed to decode ECR authorization token: %w", err)
	}

	// Token format is "AWS:password"
	parts := strings.SplitN(string(decodedToken), ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid ECR authorization token format")
	}

	username := parts[0]
	password := parts[1]

	// Login to Docker registry
	fmt.Printf("Logging into ECR registry: %s\n", e.registryURL)
	_, err = execCommand(ctx, "docker", "login", "-u", username, "-p", password, e.registryURL)
	if err != nil {
		return fmt.Errorf("failed to login to ECR: %w", err)
	}

	fmt.Println("Successfully authenticated with ECR")
	return nil
}

// TagImage tags the source image for ECR
func (e *ECRRegistry) TagImage(ctx context.Context, sourceImage string) (string, error) {
	// Build target image URI
	e.imageURI = fmt.Sprintf("%s/%s:%s", e.registryURL, e.repositoryName, e.imageTag)

	// Tag the image
	if err := dockerTag(ctx, sourceImage, e.imageURI); err != nil {
		return "", fmt.Errorf("failed to tag image for ECR: %w", err)
	}

	return e.imageURI, nil
}

// PushImage pushes the image to ECR
func (e *ECRRegistry) PushImage(ctx context.Context, taggedImage string) error {
	if err := dockerPush(ctx, taggedImage); err != nil {
		return fmt.Errorf("failed to push image to ECR: %w", err)
	}
	return nil
}
