package queue

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

type Sender interface {
	SendMessage(queueName string, body any) error
	SendMessageBatch(queueName string, messages []BatchMessage) (*BatchResult, error)
}
