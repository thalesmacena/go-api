package aws

import (
	"context"
	"go-api/pkg/resource"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

var Config aws.Config

func init() {
	ctx := context.Background()

	// Load default config
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(resource.GetString("app.cloud.aws-region")),
	)

	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	// Check if custom credentials are provided
	if accessKey := resource.GetString("app.cloud.aws-access-key-id"); accessKey != "" {
		secretKey := resource.GetString("app.cloud.aws-secret-access-key")
		if secretKey != "" {
			cfg.Credentials = credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")
		}
	}
	// If no custom credentials are provided, AWS SDK will use default credential chain
	// (environment variables, IAM roles, etc.)

	// Check if LocalStack endpoint is configured
	if endpoint := resource.GetString("app.cloud.aws-endpoint"); endpoint != "" {
		cfg.BaseEndpoint = &endpoint
		// For LocalStack, we need to customize the endpoint resolver
		// The DisableHTTPS option is handled via the endpoint URL scheme (http:// vs https://)
	}

	Config = cfg
}
