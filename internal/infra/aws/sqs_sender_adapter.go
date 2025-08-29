package aws

import (
	"go-api/internal/domain/gateway/queue"
	"go-api/pkg/sqs"

	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

// SQSSenderAdapter adapts the pkg/sqs.Sender to implement domain queue.Sender interface
type SQSSenderAdapter struct {
	sqsSender *sqs.Sender
}

// NewSQSSenderAdapter creates a new SQS sender adapter that implements domain interface
func NewSQSSenderAdapter(sqsClient sqsiface.SQSAPI) queue.Sender {
	return &SQSSenderAdapter{
		sqsSender: sqs.NewSender(sqsClient),
	}
}

// SendMessage implements the domain interface
func (adapter *SQSSenderAdapter) SendMessage(queueName string, body any) error {
	return adapter.sqsSender.SendMessage(queueName, body)
}

// SendMessageBatch implements the domain interface by converting types
func (adapter *SQSSenderAdapter) SendMessageBatch(queueName string, messages []queue.BatchMessage) (*queue.BatchResult, error) {
	// Convert domain types to SQS types
	sqsMessages := make([]sqs.BatchMessage, len(messages))
	for i, msg := range messages {
		sqsMessages[i] = sqs.BatchMessage{
			MessageID: msg.MessageID,
			Body:      msg.Body,
		}
	}

	// Call the SQS-specific method
	result, err := adapter.sqsSender.SendMessageBatch(queueName, sqsMessages)
	if err != nil {
		return nil, err
	}

	// Convert SQS result back to domain type
	return &queue.BatchResult{
		Successful: result.Successful,
		Failed:     result.Failed,
	}, nil
}
