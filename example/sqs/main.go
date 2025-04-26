package main

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"go-api/pkg/log"
	sqslib "go-api/pkg/sqs"
	"time"
)

func main() {
	// Create a session
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	}))

	// create SQS Client
	sqsClient := sqs.New(sess)

	// Set Queue Name
	queueName := "teste1-oregon"

	handler := sqslib.HandlerFunc(func(msg *sqs.Message) error {
		log.Infof("Received Message ID: %s, Value: %s", aws.StringValue(msg.MessageId), aws.StringValue(msg.Body))

		// Process Logic

		return nil
	})

	// Optional Configs
	config := &sqslib.WorkerConfig{
		MaxNumberOfMessages: 10,
		WaitTimeSeconds:     20,
		PoolSize:            5,
		LogLevel:            sqslib.InfoLevel,
	}

	// Create Worker
	worker, err := sqslib.NewWorker(sqsClient, queueName, handler, config)
	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	worker.Start(ctx)
}
