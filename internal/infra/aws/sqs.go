package aws

import "github.com/aws/aws-sdk-go-v2/service/sqs"

func NewSqsClient() *sqs.Client {
	return sqs.NewFromConfig(Config)
}
