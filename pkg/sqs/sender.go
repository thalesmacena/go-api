package sqs

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// BatchMessage represents a message to be sent in batch
type BatchMessage struct {
	MessageID string `json:"messageId"`
	Body      any    `json:"body"`
}

// BatchResult represents the result of a batch send operation
type BatchResult struct {
	Successful []string `json:"successful"`
	Failed     []string `json:"failed"`
}

// SQSClient defines the interface for SQS operations
type SQSClient interface {
	GetQueueUrl(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error)
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
	SendMessageBatch(ctx context.Context, params *sqs.SendMessageBatchInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageBatchOutput, error)
}

// Sender handles sending messages to SQS queues
type Sender struct {
	sqsClient SQSClient
}

// NewSender creates and returns a new Sender
func NewSender(sqsClient SQSClient) *Sender {
	return &Sender{
		sqsClient: sqsClient,
	}
}

// SendMessage serializes the provided body to JSON and sends it to the specified queue
func (s *Sender) SendMessage(queueName string, body any) error {
	ctx := context.Background()

	// Get queue URL
	queueURL, err := s.getQueueURL(ctx, queueName)
	if err != nil {
		return fmt.Errorf("failed to get queue URL for %s: %w", queueName, err)
	}

	// Serialize body to JSON
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to serialize message body to JSON: %w", err)
	}

	// Send message
	messageBody := string(jsonBody)
	_, err = s.sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    &queueURL,
		MessageBody: &messageBody,
	})
	if err != nil {
		return fmt.Errorf("failed to send message to queue %s: %w", queueName, err)
	}

	return nil
}

// SendMessageBatch sends multiple messages in batches of 10 to the specified queue using parallel processing
// Returns BatchResult with successful and failed message IDs
func (s *Sender) SendMessageBatch(queueName string, messages []BatchMessage) (*BatchResult, error) {
	if len(messages) == 0 {
		return &BatchResult{
			Successful: []string{},
			Failed:     []string{},
		}, nil
	}

	ctx := context.Background()

	// Get queue URL
	queueURL, err := s.getQueueURL(ctx, queueName)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue URL for %s: %w", queueName, err)
	}

	// Split messages into batches of 10 (SQS limit)
	const batchSize = 10
	var batches [][]BatchMessage
	for i := 0; i < len(messages); i += batchSize {
		end := i + batchSize
		if end > len(messages) {
			end = len(messages)
		}
		batches = append(batches, messages[i:end])
	}

	// Channel to collect results from parallel batch sends
	resultChan := make(chan *BatchResult, len(batches))
	var wg sync.WaitGroup

	// Send all batches in parallel
	for _, batch := range batches {
		wg.Add(1)
		go func(batchMessages []BatchMessage) {
			defer wg.Done()

			batchResult, err := s.sendBatch(ctx, queueURL, batchMessages)
			if err != nil {
				// If the entire batch fails, mark all messages as failed
				failedResult := &BatchResult{
					Successful: []string{},
					Failed:     make([]string, len(batchMessages)),
				}
				for i, msg := range batchMessages {
					failedResult.Failed[i] = msg.MessageID
				}
				resultChan <- failedResult
				return
			}

			resultChan <- batchResult
		}(batch)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(resultChan)

	// Collect all results
	finalResult := &BatchResult{
		Successful: []string{},
		Failed:     []string{},
	}

	for batchResult := range resultChan {
		finalResult.Successful = append(finalResult.Successful, batchResult.Successful...)
		finalResult.Failed = append(finalResult.Failed, batchResult.Failed...)
	}

	return finalResult, nil
}

// sendBatch sends a single batch of up to 10 messages
func (s *Sender) sendBatch(ctx context.Context, queueURL string, messages []BatchMessage) (*BatchResult, error) {
	if len(messages) > 10 {
		return nil, fmt.Errorf("batch size cannot exceed 10 messages, got %d", len(messages))
	}

	entries := make([]types.SendMessageBatchRequestEntry, 0, len(messages))
	serializationFailed := make([]string, 0)

	// Prepare batch entries
	for _, msg := range messages {
		// Serialize body to JSON
		jsonBody, err := json.Marshal(msg.Body)
		if err != nil {
			// Add to serialization failed list
			serializationFailed = append(serializationFailed, msg.MessageID)
			continue
		}

		messageBody := string(jsonBody)
		entries = append(entries, types.SendMessageBatchRequestEntry{
			Id:          &msg.MessageID,
			MessageBody: &messageBody,
		})
	}

	result := &BatchResult{
		Successful: []string{},
		Failed:     serializationFailed, // Start with serialization failures
	}

	// If no messages could be serialized, return early
	if len(entries) == 0 {
		return result, nil
	}

	// Send batch
	output, err := s.sqsClient.SendMessageBatch(ctx, &sqs.SendMessageBatchInput{
		QueueUrl: &queueURL,
		Entries:  entries,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send message batch: %w", err)
	}

	// Process successful messages
	for _, success := range output.Successful {
		if success.Id != nil {
			result.Successful = append(result.Successful, *success.Id)
		}
	}

	// Process failed messages from SQS
	for _, failed := range output.Failed {
		if failed.Id != nil {
			result.Failed = append(result.Failed, *failed.Id)
		}
	}

	return result, nil
}

// getQueueURL retrieves the URL for the specified queue name
func (s *Sender) getQueueURL(ctx context.Context, queueName string) (string, error) {
	result, err := s.sqsClient.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: &queueName,
	})
	if err != nil {
		return "", err
	}
	if result.QueueUrl == nil {
		return "", fmt.Errorf("queue URL is nil for queue %s", queueName)
	}
	return *result.QueueUrl, nil
}

// extractMessageIDs extracts message IDs from a slice of BatchMessage
func extractMessageIDs(messages []BatchMessage) []string {
	ids := make([]string, len(messages))
	for i, msg := range messages {
		ids[i] = msg.MessageID
	}
	return ids
}
