package registry

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/jvreagan/cloud-deploy/pkg/logging"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/google/go-containerregistry/pkg/authn"
)

// ECRRegistry represents an AWS Elastic Container Registry
type ECRRegistry struct {
	config         aws.Config
	region         string
	repositoryName string
	accountID      string
	imageTag       string
	registryURL    string
	imageURI       string
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

// GetImageReference returns the full image reference for ECR
func (e *ECRRegistry) GetImageReference() string {
	return e.imageURI
}

// GetAuthenticator returns the authenticator for ECR using AWS credentials
func (e *ECRRegistry) GetAuthenticator(ctx context.Context) (authn.Authenticator, error) {
	// Get AWS account ID
	stsClient := sts.NewFromConfig(e.config)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS account ID: %w", err)
	}
	e.accountID = *identity.Account

	// Build registry URL and image URI
	e.registryURL = fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", e.accountID, e.region)
	e.imageURI = fmt.Sprintf("%s/%s:%s", e.registryURL, e.repositoryName, e.imageTag)

	// Create ECR client
	ecrClient := ecr.NewFromConfig(e.config)

	// Create repository if it doesn't exist
	logging.Info("Ensuring ECR repository exists: %s\n", e.repositoryName)
	_, err = ecrClient.CreateRepository(ctx, &ecr.CreateRepositoryInput{
		RepositoryName: aws.String(e.repositoryName),
	})
	if err != nil {
		// Ignore error if repository already exists
		if !strings.Contains(err.Error(), "RepositoryAlreadyExistsException") {
			return nil, fmt.Errorf("failed to create ECR repository: %w", err)
		}
		logging.Info("Repository %s already exists\n", e.repositoryName)
	} else {
		logging.Info("Created ECR repository: %s\n", e.repositoryName)
	}

	// Get authorization token
	authOutput, err := ecrClient.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ECR authorization token: %w", err)
	}

	if len(authOutput.AuthorizationData) == 0 {
		return nil, fmt.Errorf("no authorization data returned from ECR")
	}

	// Decode the authorization token
	authToken := *authOutput.AuthorizationData[0].AuthorizationToken
	decodedToken, err := base64.StdEncoding.DecodeString(authToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ECR authorization token: %w", err)
	}

	// Token format is "AWS:password"
	parts := strings.SplitN(string(decodedToken), ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid ECR authorization token format")
	}

	username := parts[0]
	password := parts[1]

	logging.Info("Successfully retrieved ECR credentials")

	// Return authenticator with username and password
	return &authn.Basic{
		Username: username,
		Password: password,
	}, nil
}
