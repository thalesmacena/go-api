package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// MessageHandler defines an interface that processes Redis pub/sub messages
type MessageHandler interface {
	HandleMessage(ctx context.Context, channel string, message string) error
}

// HandlerFunc defines a function that handles Redis pub/sub messages
type HandlerFunc func(ctx context.Context, channel string, message string) error

var _ MessageHandler = HandlerFunc(nil)

// HandleMessage implements the MessageHandler interface for HandlerFunc
func (f HandlerFunc) HandleMessage(ctx context.Context, channel string, message string) error {
	return f(ctx, channel, message)
}

// LogLevel represents the logging level for the PubSub
type LogLevel int

const (
	// Silent disables all logs
	Silent LogLevel = iota
	// ErrorLevel logs only errors
	ErrorLevel
	// InfoLevel logs informational and error messages
	InfoLevel
)

// SubscriberHealthCheck represents the health check response for Redis subscriber
type SubscriberHealthCheck struct {
	Status  HealthStatus      `json:"status"`
	Details map[string]string `json:"details"`
}

// PubSubConfig defines the configuration options for Redis pub/sub
type PubSubConfig struct {
	// PoolSize is the number of concurrent message handlers
	PoolSize int
	// LogLevel controls the logging verbosity
	LogLevel LogLevel
	// ReconnectDelay is the delay between reconnection attempts
	ReconnectDelay time.Duration
	// MaxReconnectAttempts is the maximum number of reconnection attempts
	MaxReconnectAttempts int
	// ChannelNamespace is the namespace for organizing channels
	ChannelNamespace string
}

// NewPubSubConfig creates a new pub/sub configuration with default values
func NewPubSubConfig() *PubSubConfig {
	return &PubSubConfig{
		PoolSize:             1,
		LogLevel:             InfoLevel,
		ReconnectDelay:       1 * time.Second,
		MaxReconnectAttempts: 10,
		ChannelNamespace:     "",
	}
}

// WithPoolSize sets the number of concurrent message handlers
func (psc *PubSubConfig) WithPoolSize(poolSize int) *PubSubConfig {
	if poolSize < 1 {
		panic(fmt.Sprintf("invalid pool size: %d, must be greater than 0", poolSize))
	}
	psc.PoolSize = poolSize
	return psc
}

// WithLogLevel sets the logging verbosity
func (psc *PubSubConfig) WithLogLevel(logLevel LogLevel) *PubSubConfig {
	psc.LogLevel = logLevel
	return psc
}

// WithReconnectDelay sets the delay between reconnection attempts
func (psc *PubSubConfig) WithReconnectDelay(delay time.Duration) *PubSubConfig {
	if delay < 0 {
		panic(fmt.Sprintf("invalid reconnect delay: %v, must be non-negative", delay))
	}
	psc.ReconnectDelay = delay
	return psc
}

// WithMaxReconnectAttempts sets the maximum number of reconnection attempts
func (psc *PubSubConfig) WithMaxReconnectAttempts(maxAttempts int) *PubSubConfig {
	if maxAttempts < 0 {
		panic(fmt.Sprintf("invalid max reconnect attempts: %d, must be non-negative", maxAttempts))
	}
	psc.MaxReconnectAttempts = maxAttempts
	return psc
}

// WithChannelNamespace sets the namespace for organizing channels
func (psc *PubSubConfig) WithChannelNamespace(namespace string) *PubSubConfig {
	psc.ChannelNamespace = namespace
	return psc
}

// Publisher handles Redis publishing operations
type Publisher struct {
	client *redis.Client
	config *PubSubConfig
}

// NewPublisher creates a new publisher
func NewPublisher(client *redis.Client, config *PubSubConfig) *Publisher {
	if config == nil {
		config = NewPubSubConfig()
	}
	return &Publisher{
		client: client,
		config: config,
	}
}

// buildChannelName constructs the full channel name using ChannelNamespace::channelName format
func (p *Publisher) buildChannelName(channel string) string {
	if p.config.ChannelNamespace != "" {
		return p.config.ChannelNamespace + "::" + channel
	}
	return channel
}

// Publish publishes a message to a channel
func (p *Publisher) Publish(ctx context.Context, channel string, message interface{}) error {
	fullChannel := p.buildChannelName(channel)
	return p.client.Publish(ctx, fullChannel, message).Err()
}

// PublishJSON publishes a JSON message to a channel
func (p *Publisher) PublishJSON(ctx context.Context, channel string, message interface{}) error {
	fullChannel := p.buildChannelName(channel)
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message to JSON: %w", err)
	}
	return p.client.Publish(ctx, fullChannel, jsonData).Err()
}

// Subscriber polls and processes messages from Redis pub/sub channels
type Subscriber struct {
	client               *redis.Client
	channels             []string
	patterns             []string
	poolSize             int
	logLevel             LogLevel
	reconnectDelay       time.Duration
	maxReconnectAttempts int
	channelNamespace     string
	handler              MessageHandler
	isRunning            int32 // atomic flag to track if subscriber is running
	messagesProcessed    int64 // atomic counter for processed messages
	reconnectAttempts    int32 // atomic counter for reconnect attempts
	mu                   sync.RWMutex
	sub                  *redis.PubSub
	ctx                  context.Context    // internal context for lifecycle management
	cancel               context.CancelFunc // cancel function to stop subscriber
}

// NewSubscriber creates and returns a new Subscriber.
//
// If the provided PubSubConfig is nil or its fields are zero,
// the following defaults will be used:
//   - PoolSize: 1
//   - LogLevel: Silent
//   - ReconnectDelay: 1 second
//   - MaxReconnectAttempts: 10
//
// Validations:
//   - PoolSize must be greater than 0.
func NewSubscriber(client *redis.Client, handler MessageHandler, config *PubSubConfig) (*Subscriber, error) {
	var poolSize = 1
	var logLevel LogLevel = Silent
	var reconnectDelay = 1 * time.Second
	var maxReconnectAttempts = 10

	var channelNamespace string
	if config != nil {
		if config.PoolSize != 0 {
			poolSize = config.PoolSize
		}
		logLevel = config.LogLevel
		if config.ReconnectDelay != 0 {
			reconnectDelay = config.ReconnectDelay
		}
		if config.MaxReconnectAttempts != 0 {
			maxReconnectAttempts = config.MaxReconnectAttempts
		}
		channelNamespace = config.ChannelNamespace
	}

	if poolSize < 1 {
		return nil, fmt.Errorf("pool size must be greater than 0")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Subscriber{
		client:               client,
		poolSize:             poolSize,
		logLevel:             logLevel,
		reconnectDelay:       reconnectDelay,
		maxReconnectAttempts: maxReconnectAttempts,
		channelNamespace:     channelNamespace,
		handler:              handler,
		ctx:                  ctx,
		cancel:               cancel,
	}, nil
}

// buildChannelName constructs the full channel name using ChannelNamespace::channelName format
func (s *Subscriber) buildChannelName(channel string) string {
	if s.channelNamespace != "" {
		return s.channelNamespace + "::" + channel
	}
	return channel
}

// Subscribe subscribes to one or more channels
func (s *Subscriber) Subscribe(ctx context.Context, channels ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Apply namespace to channels
	namespacedChannels := make([]string, len(channels))
	for i, channel := range channels {
		namespacedChannels[i] = s.buildChannelName(channel)
	}

	s.channels = namespacedChannels
	s.patterns = nil // Clear patterns when subscribing to channels

	if s.sub != nil {
		s.sub.Close()
	}

	s.sub = s.client.Subscribe(ctx, namespacedChannels...)
	return nil
}

// PSubscribe subscribes to one or more patterns
func (s *Subscriber) PSubscribe(ctx context.Context, patterns ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Apply namespace to patterns
	namespacedPatterns := make([]string, len(patterns))
	for i, pattern := range patterns {
		namespacedPatterns[i] = s.buildChannelName(pattern)
	}

	s.patterns = namespacedPatterns
	s.channels = nil // Clear channels when subscribing to patterns

	if s.sub != nil {
		s.sub.Close()
	}

	s.sub = s.client.PSubscribe(ctx, namespacedPatterns...)
	return nil
}

// Start begins listening for messages and processing them concurrently.
// It will spawn PoolSize number of workers that keep listening for messages
// until the provided context is canceled or Stop() is called.
func (s *Subscriber) Start(ctx context.Context) {
	if s.sub == nil {
		s.logf(ErrorLevel, "not subscribed to any channels or patterns")
		return
	}

	atomic.StoreInt32(&s.isRunning, 1)
	defer atomic.StoreInt32(&s.isRunning, 0)

	// Create a combined context that responds to both the provided context and internal cancellation
	combinedCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Monitor internal context for Stop() calls
	go func() {
		select {
		case <-s.ctx.Done():
			cancel()
		case <-combinedCtx.Done():
		}
	}()

	var wg sync.WaitGroup

	for i := 0; i < s.poolSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.listenMessages(combinedCtx)
		}()
	}

	wg.Wait()
}

// listenMessages listens for messages from Redis pub/sub
func (s *Subscriber) listenMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if s.sub == nil {
				s.logf(ErrorLevel, "subscriber is not initialized")
				time.Sleep(s.reconnectDelay)
				continue
			}

			ch := s.sub.Channel()
			for msg := range ch {
				go s.handleMessage(ctx, msg)
			}

			// If we get here, the channel was closed
			// Check if context was cancelled before attempting reconnect
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Try to reconnect if not cancelled
			if atomic.LoadInt32(&s.reconnectAttempts) < int32(s.maxReconnectAttempts) {
				atomic.AddInt32(&s.reconnectAttempts, 1)
				s.logf(ErrorLevel, "channel closed, attempting to reconnect (attempt %d/%d)",
					atomic.LoadInt32(&s.reconnectAttempts), s.maxReconnectAttempts)

				if err := s.reconnect(ctx); err != nil {
					s.logf(ErrorLevel, "failed to reconnect: %v", err)

					// Check again if context was cancelled
					select {
					case <-ctx.Done():
						return
					default:
						time.Sleep(s.reconnectDelay)
						continue
					}
				}

				atomic.StoreInt32(&s.reconnectAttempts, 0) // Reset on successful reconnect
			} else {
				s.logf(ErrorLevel, "max reconnection attempts reached, stopping subscriber")
				return
			}
		}
	}
}

// handleMessage processes a received message
func (s *Subscriber) handleMessage(ctx context.Context, msg *redis.Message) {
	if msg == nil {
		return
	}

	err := s.handler.HandleMessage(ctx, msg.Channel, msg.Payload)
	if err != nil {
		s.logf(ErrorLevel, "error processing message from channel %s: %v", msg.Channel, err)
		return
	}

	s.logf(InfoLevel, "successfully processed message from channel %s", msg.Channel)
	atomic.AddInt64(&s.messagesProcessed, 1)
}

// reconnect attempts to reconnect to Redis pub/sub
func (s *Subscriber) reconnect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sub != nil {
		s.sub.Close()
	}

	if len(s.channels) > 0 {
		s.sub = s.client.Subscribe(ctx, s.channels...)
	} else if len(s.patterns) > 0 {
		s.sub = s.client.PSubscribe(ctx, s.patterns...)
	} else {
		return fmt.Errorf("no channels or patterns to subscribe to")
	}

	return nil
}

// Stop gracefully stops the subscriber by canceling its internal context.
// This method should be called before Close() to ensure all goroutines are stopped cleanly.
// It's safe to call Stop() multiple times.
func (s *Subscriber) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

// Close closes the subscriber and releases resources.
// It automatically calls Stop() to cancel any running goroutines before closing the connection.
// It's recommended to call Stop() explicitly before Close() with a small delay to allow graceful shutdown.
func (s *Subscriber) Close() error {
	// Stop the subscriber first to cancel goroutines
	s.Stop()

	// Wait for goroutines to finish processing and exit cleanly
	// This prevents "channel closed" errors during shutdown
	time.Sleep(150 * time.Millisecond)

	s.mu.Lock()
	defer s.mu.Unlock()

	atomic.StoreInt32(&s.isRunning, 0)

	if s.sub != nil {
		return s.sub.Close()
	}
	return nil
}

// logf logs messages based on the configured log level
func (s *Subscriber) logf(level LogLevel, format string, v ...interface{}) {
	if s.logLevel == Silent {
		return
	}
	if level == ErrorLevel && (s.logLevel == ErrorLevel || s.logLevel == InfoLevel) {
		fmt.Printf("[ERROR] "+format+"\n", v...)
	}
	if level == InfoLevel && s.logLevel == InfoLevel {
		fmt.Printf("[INFO] "+format+"\n", v...)
	}
}

// ParseLogLevel converts string log level to LogLevel
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

// HealthCheck returns the health status and details of the Redis pub/sub
func (s *Subscriber) HealthCheck() SubscriberHealthCheck {
	isRunning := atomic.LoadInt32(&s.isRunning) == 1
	messagesProcessed := atomic.LoadInt64(&s.messagesProcessed)
	reconnectAttempts := atomic.LoadInt32(&s.reconnectAttempts)

	// Test Redis connectivity
	redisAvailable := s.testRedisConnectivity()

	// Determine status based on both running state and Redis connectivity
	var status HealthStatus
	if isRunning && redisAvailable {
		status = StatusUp
	} else if !isRunning {
		status = StatusDown
	} else if isRunning && !redisAvailable {
		status = StatusDown // Running but can't connect to Redis is a problem
	} else {
		status = StatusUnknown
	}

	details := map[string]string{
		"pool_size":              fmt.Sprintf("%d", s.poolSize),
		"log_level":              s.getLogLevelString(),
		"reconnect_delay":        s.reconnectDelay.String(),
		"max_reconnect_attempts": fmt.Sprintf("%d", s.maxReconnectAttempts),
		"is_running":             fmt.Sprintf("%t", isRunning),
		"messages_processed":     fmt.Sprintf("%d", messagesProcessed),
		"reconnect_attempts":     fmt.Sprintf("%d", reconnectAttempts),
		"redis_available":        fmt.Sprintf("%t", redisAvailable),
		"channels":               fmt.Sprintf("%v", s.channels),
		"patterns":               fmt.Sprintf("%v", s.patterns),
	}

	return SubscriberHealthCheck{
		Status:  status,
		Details: details,
	}
}

// testRedisConnectivity tests if Redis is accessible
func (s *Subscriber) testRedisConnectivity() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.client.Ping(ctx).Err()
	return err == nil
}

// getLogLevelString returns the string representation of the log level
func (s *Subscriber) getLogLevelString() string {
	switch s.logLevel {
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
