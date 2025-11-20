package main

import (
	"context"
	"go-api/pkg/log"
	sqslib "go-api/pkg/sqs"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

func main() {
	ctx := context.Background()

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-west-2"),
	)
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	// create SQS Client
	sqsClient := sqs.NewFromConfig(cfg)

	// Set Queue Name
	queueName := "test-queue"

	handler := sqslib.HandlerFunc(func(msg *types.Message) error {
		var messageID, body string
		if msg.MessageId != nil {
			messageID = *msg.MessageId
		}
		if msg.Body != nil {
			body = *msg.Body
		}
		log.Infof("Received Message ID: %s, Value: %s", messageID, body)

		// Process Logic

		return nil
	})

	// Optional Configs
	workerConfig := &sqslib.WorkerConfig{
		MaxNumberOfMessages: 10,
		WaitTimeSeconds:     20,
		PoolSize:            5,
		LogLevel:            sqslib.InfoLevel,
	}

	// Create Worker
	worker, err := sqslib.NewWorker(sqsClient, queueName, handler, workerConfig)
	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	worker.Start(ctx)
}
