package sqs

import (
	"context"
	"errors"
	"fmt"
	"go-api/pkg/log"
	"sync"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

// HandlerFunc defines a function that handles a SQS Message
type HandlerFunc func(msg *sqs.Message) error

// HandleMessage implements the Handler interface for HandlerFunc
func (f HandlerFunc) HandleMessage(msg *sqs.Message) error {
	return f(msg)
}

// Handler defines an interface that processes a SQS Message
type Handler interface {
	HandleMessage(msg *sqs.Message) error
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

// WorkerConfig defines the configuration options for a Worker
type WorkerConfig struct {
	MaxNumberOfMessages int64
	WaitTimeSeconds     int64
	PoolSize            int64
	LogLevel            LogLevel
}

// Worker polls and processes messages from a SQS queue
type Worker struct {
	sqsClient           sqsiface.SQSAPI
	queueName           string
	queueURL            string
	maxNumberOfMessages int64
	waitTimeSeconds     int64
	poolSize            int64
	logLevel            LogLevel
	handler             Handler
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
func NewWorker(sqsClient sqsiface.SQSAPI, queueName string, handler Handler, config *WorkerConfig) (*Worker, error) {
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

	result, err := sqsClient.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: &queueName,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to get queue URL: %w", err)
	}

	return &Worker{
		sqsClient:           sqsClient,
		queueName:           queueName,
		queueURL:            *result.QueueUrl,
		maxNumberOfMessages: maxMessages,
		waitTimeSeconds:     waitTime,
		poolSize:            poolSize,
		logLevel:            logLevel,
		handler:             handler,
	}, nil
}

// Start begins polling messages and processing them concurrently.
// It will spawn PoolSize number of workers that keep polling messages
// until the provided context is canceled.
func (w *Worker) Start(ctx context.Context) {
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
			output, err := w.sqsClient.ReceiveMessage(&sqs.ReceiveMessageInput{
				QueueUrl:            &w.queueURL,
				MaxNumberOfMessages: &w.maxNumberOfMessages,
				WaitTimeSeconds:     &w.waitTimeSeconds,
			})
			if err != nil {
				w.logf(ErrorLevel, "failed to receive messages: %v", err)
				continue
			}

			for _, msg := range output.Messages {
				go w.handleMessage(msg)
			}
		}
	}
}

func (w *Worker) handleMessage(msg *sqs.Message) {
	if msg == nil {
		return
	}

	err := w.handler.HandleMessage(msg)
	if err != nil {
		w.logf(ErrorLevel, "error processing message ID %s: %v", safeMessageID(msg), err)
		return
	}

	_, err = w.sqsClient.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      &w.queueURL,
		ReceiptHandle: msg.ReceiptHandle,
	})
	if err != nil {
		w.logf(ErrorLevel, "failed to delete message ID %s: %v", safeMessageID(msg), err)
	} else {
		w.logf(InfoLevel, "successfully deleted message ID %s", safeMessageID(msg))
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

func safeMessageID(msg *sqs.Message) string {
	if msg == nil || msg.MessageId == nil {
		return ""
	}
	return *msg.MessageId
}
