package aws

import "github.com/aws/aws-sdk-go/service/sqs"

func NewSqsClient() *sqs.SQS {
	return sqs.New(Session)
}
