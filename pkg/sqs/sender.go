package sqs

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
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

// Sender handles sending messages to SQS queues
type Sender struct {
	sqsClient sqsiface.SQSAPI
}

// NewSender creates and returns a new Sender
func NewSender(sqsClient sqsiface.SQSAPI) *Sender {
	return &Sender{
		sqsClient: sqsClient,
	}
}

// SendMessage serializes the provided body to JSON and sends it to the specified queue
func (s *Sender) SendMessage(queueName string, body any) error {
	// Get queue URL
	queueURL, err := s.getQueueURL(queueName)
	if err != nil {
		return fmt.Errorf("failed to get queue URL for %s: %w", queueName, err)
	}

	// Serialize body to JSON
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to serialize message body to JSON: %w", err)
	}

	// Send message
	_, err = s.sqsClient.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(string(jsonBody)),
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

	// Get queue URL
	queueURL, err := s.getQueueURL(queueName)
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

			batchResult, err := s.sendBatch(queueURL, batchMessages)
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
func (s *Sender) sendBatch(queueURL string, messages []BatchMessage) (*BatchResult, error) {
	if len(messages) > 10 {
		return nil, fmt.Errorf("batch size cannot exceed 10 messages, got %d", len(messages))
	}

	entries := make([]*sqs.SendMessageBatchRequestEntry, 0, len(messages))
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

		entries = append(entries, &sqs.SendMessageBatchRequestEntry{
			Id:          aws.String(msg.MessageID),
			MessageBody: aws.String(string(jsonBody)),
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
	output, err := s.sqsClient.SendMessageBatch(&sqs.SendMessageBatchInput{
		QueueUrl: aws.String(queueURL),
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
func (s *Sender) getQueueURL(queueName string) (string, error) {
	result, err := s.sqsClient.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return "", err
	}
	return aws.StringValue(result.QueueUrl), nil
}

// extractMessageIDs extracts message IDs from a slice of BatchMessage
func extractMessageIDs(messages []BatchMessage) []string {
	ids := make([]string, len(messages))
	for i, msg := range messages {
		ids[i] = msg.MessageID
	}
	return ids
}
