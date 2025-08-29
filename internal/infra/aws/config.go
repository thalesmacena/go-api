package aws

import (
	"go-api/pkg/resource"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
)

var Session *session.Session

func init() {
	config := &aws.Config{
		Region: aws.String(resource.GetString("app.cloud.aws-region")),
	}

	// Check if LocalStack endpoint is configured
	if endpoint := resource.GetString("app.cloud.aws-endpoint"); endpoint != "" {
		config.Endpoint = aws.String(endpoint)
		config.DisableSSL = aws.Bool(!resource.GetBool("app.cloud.aws-use-ssl"))
		config.S3ForcePathStyle = aws.Bool(true) // Required for LocalStack
	}

	// Check if custom credentials are provided
	if accessKey := resource.GetString("app.cloud.aws-access-key-id"); accessKey != "" {
		secretKey := resource.GetString("app.cloud.aws-secret-access-key")
		if secretKey != "" {
			config.Credentials = credentials.NewStaticCredentials(accessKey, secretKey, "")
		}
	}
	// If no custom credentials are provided, AWS SDK will use default credential chain
	// (environment variables, IAM roles, etc.)

	sess, err := session.NewSession(config)

	if err != nil {
		log.Fatalf("Failed to create AWS session: %v", err)
	}

	Session = sess
}
