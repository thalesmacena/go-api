package sqs

import (
	"context"
	"errors"
	"fmt"
	"go-api/pkg/log"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// HandlerFunc defines a function that handles a SQS Message
type HandlerFunc func(msg *types.Message) error

// HandleMessage implements the Handler interface for HandlerFunc
func (f HandlerFunc) HandleMessage(msg *types.Message) error {
	return f(msg)
}

// Handler defines an interface that processes a SQS Message
type Handler interface {
	HandleMessage(msg *types.Message) error
}

// LogLevel represents the logging level for the Worker
type LogLevel int

const (
	// Silent disables all logs
	Silent LogLevel = iota
	// ErrorLevel logs only errors
	ErrorLevel
	// InfoLevel logs informational and error messages
	InfoLevel
)

// HealthStatus represents the health status of the SQS worker
type HealthStatus string

const (
	// StatusUp indicates the worker is healthy and running
	StatusUp HealthStatus = "UP"
	// StatusDown indicates the worker is not healthy or not running
	StatusDown HealthStatus = "DOWN"
	// StatusUnknown indicates the worker status cannot be determined
	StatusUnknown HealthStatus = "UNKNOWN"
)

// WorkerHealthCheck represents the health check response for a SQS worker
type WorkerHealthCheck struct {
	Status  HealthStatus      `json:"status"`
	Details map[string]string `json:"details"`
}

// SQSWorkerClient defines the interface for SQS worker operations
type SQSWorkerClient interface {
	GetQueueUrl(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error)
	ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
	GetQueueAttributes(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error)
}

// WorkerConfig defines the configuration options for a Worker
type WorkerConfig struct {
	MaxNumberOfMessages int64
	WaitTimeSeconds     int64
	PoolSize            int64
	LogLevel            LogLevel
}

// Worker polls and processes messages from a SQS queue
type Worker struct {
	sqsClient           SQSWorkerClient
	queueName           string
	queueURL            string
	maxNumberOfMessages int32
	waitTimeSeconds     int32
	poolSize            int64
	logLevel            LogLevel
	handler             Handler
	isRunning           int32 // atomic flag to track if worker is running
	messagesProcessed   int64 // atomic counter for processed messages
}

// NewWorker creates and returns a new Worker.
//
// If the provided WorkerConfig is nil or its fields are zero,
// the following defaults will be used:
//   - MaxNumberOfMessages: 10
//   - WaitTimeSeconds: 20
//   - PoolSize: 1
//   - LogLevel: Silent
//
// Validations:
//   - MaxNumberOfMessages must be between 1 and 10.
//   - WaitTimeSeconds must be between 1 and 20.
//   - PoolSize must be greater than 0.
func NewWorker(sqsClient SQSWorkerClient, queueName string, handler Handler, config *WorkerConfig) (*Worker, error) {
	var maxMessages int64 = 10
	var waitTime int64 = 20
	var poolSize int64 = 1
	var logLevel LogLevel = Silent

	if config != nil {
		if config.MaxNumberOfMessages != 0 {
			maxMessages = config.MaxNumberOfMessages
		}
		if config.WaitTimeSeconds != 0 {
			waitTime = config.WaitTimeSeconds
		}
		if config.PoolSize != 0 {
			poolSize = config.PoolSize
		}
		logLevel = config.LogLevel
	}

	if maxMessages < 1 || maxMessages > 10 {
		return nil, errors.New("maxNumberOfMessages must be between 1 and 10")
	}
	if waitTime < 1 || waitTime > 20 {
		return nil, errors.New("waitTimeSeconds must be between 1 and 20")
	}
	if poolSize < 1 {
		return nil, errors.New("poolSize must be greater than 0")
	}

	ctx := context.Background()
	result, err := sqsClient.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: &queueName,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to get queue URL: %w", err)
	}

	if result.QueueUrl == nil {
		return nil, fmt.Errorf("queue URL is nil for queue %s", queueName)
	}

	return &Worker{
		sqsClient:           sqsClient,
		queueName:           queueName,
		queueURL:            *result.QueueUrl,
		maxNumberOfMessages: int32(maxMessages),
		waitTimeSeconds:     int32(waitTime),
		poolSize:            poolSize,
		logLevel:            logLevel,
		handler:             handler,
	}, nil
}

// Start begins polling messages and processing them concurrently.
// It will spawn PoolSize number of workers that keep polling messages
// until the provided context is canceled.
func (w *Worker) Start(ctx context.Context) {
	atomic.StoreInt32(&w.isRunning, 1)
	defer atomic.StoreInt32(&w.isRunning, 0)

	var wg sync.WaitGroup

	for i := int64(0); i < w.poolSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.pollMessages(ctx)
		}()
	}

	wg.Wait()
}

func (w *Worker) pollMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			output, err := w.sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
				QueueUrl:            &w.queueURL,
				MaxNumberOfMessages: w.maxNumberOfMessages,
				WaitTimeSeconds:     w.waitTimeSeconds,
			})
			if err != nil {
				w.logf(ErrorLevel, "failed to receive messages: %v", err)
				continue
			}

			for _, msg := range output.Messages {
				msgCopy := msg
				go w.handleMessage(ctx, &msgCopy)
			}
		}
	}
}

func (w *Worker) handleMessage(ctx context.Context, msg *types.Message) {
	if msg == nil {
		return
	}

	err := w.handler.HandleMessage(msg)
	if err != nil {
		w.logf(ErrorLevel, "error processing message ID %s: %v", safeMessageID(msg), err)
		return
	}

	_, err = w.sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      &w.queueURL,
		ReceiptHandle: msg.ReceiptHandle,
	})
	if err != nil {
		w.logf(ErrorLevel, "failed to delete message ID %s: %v", safeMessageID(msg), err)
	} else {
		w.logf(InfoLevel, "successfully deleted message ID %s", safeMessageID(msg))
		atomic.AddInt64(&w.messagesProcessed, 1)
	}
}

func (w *Worker) logf(level LogLevel, format string, v ...interface{}) {
	if w.logLevel == Silent {
		log.Debugf(format, v...)
	}
	if level == ErrorLevel && (w.logLevel == ErrorLevel || w.logLevel == InfoLevel) {
		log.Errorf(format, v...)
	}
	if level == InfoLevel && w.logLevel == InfoLevel {
		log.Infof(format, v...)
	}
}

func safeMessageID(msg *types.Message) string {
	if msg == nil || msg.MessageId == nil {
		return ""
	}
	return *msg.MessageId
}

// ParseLogLevel converts string log level to sqs.LogLevel
func ParseLogLevel(level string) LogLevel {
	switch level {
	case "silent":
		return Silent
	case "error":
		return ErrorLevel
	case "info":
		return InfoLevel
	default:
		return InfoLevel
	}
}

// HealthCheck returns the health status and details of the SQS worker
func (w *Worker) HealthCheck() WorkerHealthCheck {
	isRunning := atomic.LoadInt32(&w.isRunning) == 1
	messagesProcessed := atomic.LoadInt64(&w.messagesProcessed)

	var status HealthStatus
	if isRunning {
		status = StatusUp
	} else {
		status = StatusDown
	}

	// Test queue connectivity by attempting to get queue attributes
	queueAvailable := w.testQueueConnectivity()
	if !queueAvailable {
		status = StatusDown
	}

	details := map[string]string{
		"queue_name":             w.queueName,
		"queue_url":              w.queueURL,
		"pool_size":              strconv.FormatInt(w.poolSize, 10),
		"max_number_of_messages": strconv.FormatInt(int64(w.maxNumberOfMessages), 10),
		"wait_time_seconds":      strconv.FormatInt(int64(w.waitTimeSeconds), 10),
		"log_level":              w.getLogLevelString(),
		"is_running":             strconv.FormatBool(isRunning),
		"messages_processed":     strconv.FormatInt(messagesProcessed, 10),
		"queue_available":        strconv.FormatBool(queueAvailable),
	}

	return WorkerHealthCheck{
		Status:  status,
		Details: details,
	}
}

// testQueueConnectivity tests if the queue is accessible
func (w *Worker) testQueueConnectivity() bool {
	ctx := context.Background()
	_, err := w.sqsClient.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl: &w.queueURL,
		AttributeNames: []types.QueueAttributeName{
			types.QueueAttributeNameApproximateNumberOfMessages,
		},
	})
	return err == nil
}

// getLogLevelString returns the string representation of the log level
func (w *Worker) getLogLevelString() string {
	switch w.logLevel {
	case Silent:
		return "silent"
	case ErrorLevel:
		return "error"
	case InfoLevel:
		return "info"
	default:
		return "unknown"
	}
}
