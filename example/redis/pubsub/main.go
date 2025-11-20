package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"go-api/pkg/redis"
)

// MessageCounter keeps track of received messages
type MessageCounter struct {
	count int
	mu    sync.Mutex
}

func (mc *MessageCounter) Increment() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.count++
}

func (mc *MessageCounter) GetCount() int {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return mc.count
}

// CustomHandler implements MessageHandler interface
type CustomHandler struct {
	counter *MessageCounter
	name    string
}

func (h *CustomHandler) HandleMessage(ctx context.Context, channel string, message string) error {
	h.counter.Increment()
	fmt.Printf("[%s] Received message on channel '%s': %s (total: %d)",
		h.name, channel, message, h.counter.GetCount())
	return nil
}

func main() {
	// Get Redis configuration from environment variables
	redisHost := getEnvOrDefault("REDIS_HOST", "localhost")
	redisPort := getEnvOrDefaultInt("REDIS_PORT", 6379)
	redisPassword := getEnvOrDefault("REDIS_PASSWORD", "redis_password")

	fmt.Printf("Using Redis configuration: Host=%s, Port=%d, Password=%s",
		redisHost, redisPort, maskPassword(redisPassword))

	// Create Redis configuration using fluent API
	config := redis.NewRedisConfig().
		WithHost(redisHost).
		WithPort(redisPort).
		WithPassword(redisPassword).
		WithDatabase(0).
		WithMinIdleConns(5).
		WithMaxIdleConns(10).
		WithMaxActive(100).
		WithDialTimeout(5 * time.Second).
		WithReadTimeout(3 * time.Second).
		WithWriteTimeout(3 * time.Second).
		WithPoolTimeout(4 * time.Second)

	// Create Redis client
	client := redis.NewClient(config)
	defer func(client *redis.Client) {
		err := client.Close()
		if err != nil {
			fmt.Printf("Error closing Redis client: %v", err)
		}
	}(client)

	ctx := context.Background()

	fmt.Println("Example Redis Pub/Sub Scenarios...")

	// Example Scenario: Basic publisher and subscriber
	fmt.Println("=== Scenario: Basic Publisher and Subscriber ===")
	exampleScenarioBasicPubSub(ctx, client)
	time.Sleep(200 * time.Millisecond) // Wait between scenarios

	// Example Scenario: JSON message publishing
	fmt.Println("=== Scenario: JSON Message Publishing ===")
	exampleScenarioJSONMessages(ctx, client)
	time.Sleep(200 * time.Millisecond) // Wait between scenarios

	// Example Scenario: Pattern subscription
	fmt.Println("=== Scenario: Pattern Subscription ===")
	exampleScenarioPatternSubscription(ctx, client)
	time.Sleep(200 * time.Millisecond) // Wait between scenarios

	// Example Scenario: Error handling
	fmt.Println("=== Scenario: Error Handling ===")
	exampleScenarioErrorHandling(ctx, client)
	time.Sleep(200 * time.Millisecond) // Wait between scenarios

	// Example Scenario: Health check
	fmt.Println("=== Scenario: Health Check and Monitoring ===")
	exampleScenarioHealthCheck(ctx, client)

	fmt.Println("All pub/sub scenarios completed successfully!")
}

// exampleScenarioBasicPubSub demonstrates basic publish and subscribe functionality
func exampleScenarioBasicPubSub(ctx context.Context, client *redis.Client) {
	fmt.Println("Testing basic publisher and subscriber with namespace...")

	counter := &MessageCounter{}
	handler := &CustomHandler{counter: counter, name: "BasicHandler"}

	// Create pub/sub configuration with namespace
	config := redis.NewPubSubConfig().
		WithPoolSize(2).
		WithLogLevel(redis.InfoLevel).
		WithReconnectDelay(1 * time.Second).
		WithMaxReconnectAttempts(5).
		WithChannelNamespace("user_service")

	subscriber, err := redis.NewSubscriber(client.GetClient(), handler, config)
	if err != nil {
		fmt.Printf("Failed to create subscriber: %v", err)
		return
	}
	defer subscriber.Close()

	publisher := redis.NewPublisher(client.GetClient(), config)

	// Subscribe to channels
	err = subscriber.Subscribe(ctx, "test_channel", "another_channel")
	if err != nil {
		fmt.Printf("Failed to subscribe: %v", err)
		return
	}
	fmt.Println("Subscribed to channels: test_channel, another_channel")
	fmt.Println("  - Actual channels: user_service::test_channel, user_service::another_channel")

	// Start subscriber in background
	go subscriber.Start(ctx)

	// Wait for subscription to be ready
	time.Sleep(100 * time.Millisecond)

	// Publish messages
	err = publisher.Publish(ctx, "test_channel", "Hello from test_channel!")
	if err != nil {
		fmt.Printf("Failed to publish: %v", err)
		return
	}
	fmt.Println("Published message to test_channel")

	err = publisher.Publish(ctx, "another_channel", "Hello from another_channel!")
	if err != nil {
		fmt.Printf("Failed to publish: %v", err)
		return
	}
	fmt.Println("Published message to another_channel")

	// Wait for messages to be processed
	time.Sleep(500 * time.Millisecond)

	fmt.Printf("Results: %d messages received", counter.GetCount())
}

// exampleScenarioJSONMessages demonstrates JSON message publishing and receiving
func exampleScenarioJSONMessages(ctx context.Context, client *redis.Client) {
	fmt.Println("Testing JSON message serialization and publishing...")

	counter := &MessageCounter{}
	handler := &CustomHandler{counter: counter, name: "JSONHandler"}

	config := redis.NewPubSubConfig().
		WithPoolSize(1).
		WithLogLevel(redis.InfoLevel)

	subscriber, err := redis.NewSubscriber(client.GetClient(), handler, config)
	if err != nil {
		fmt.Printf("Failed to create subscriber: %v", err)
		return
	}
	defer subscriber.Close()

	publisher := redis.NewPublisher(client.GetClient(), config)

	// Subscribe to channel
	err = subscriber.Subscribe(ctx, "json_channel")
	if err != nil {
		fmt.Printf("Failed to subscribe: %v", err)
		return
	}
	fmt.Println("Subscribed to json_channel")

	// Start subscriber
	go subscriber.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Define message structure
	type Message struct {
		ID      int       `json:"id"`
		Content string    `json:"content"`
		Time    time.Time `json:"time"`
	}

	// Publish JSON messages
	for i := 1; i <= 3; i++ {
		jsonMessage := Message{
			ID:      i,
			Content: fmt.Sprintf("JSON message number %d", i),
			Time:    time.Now(),
		}

		err = publisher.PublishJSON(ctx, "json_channel", jsonMessage)
		if err != nil {
			fmt.Printf("Failed to publish JSON message %d: %v", i, err)
			continue
		}
		fmt.Printf("Published JSON message %d", i)
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for messages to be processed
	time.Sleep(500 * time.Millisecond)

	fmt.Printf("Results: %d JSON messages received", counter.GetCount())
}

// exampleScenarioPatternSubscription demonstrates pattern-based subscription
func exampleScenarioPatternSubscription(ctx context.Context, client *redis.Client) {
	fmt.Println("Testing pattern-based subscription (user:*, event:*)...")

	counter := &MessageCounter{}
	handler := &CustomHandler{counter: counter, name: "PatternHandler"}

	config := redis.NewPubSubConfig().
		WithPoolSize(2).
		WithLogLevel(redis.InfoLevel)

	subscriber, err := redis.NewSubscriber(client.GetClient(), handler, config)
	if err != nil {
		fmt.Printf("Failed to create subscriber: %v", err)
		return
	}
	defer subscriber.Close()

	publisher := redis.NewPublisher(client.GetClient(), config)

	// Subscribe to patterns
	err = subscriber.PSubscribe(ctx, "user:*", "event:*")
	if err != nil {
		fmt.Printf("Failed to subscribe to patterns: %v", err)
		return
	}
	fmt.Println("Subscribed to patterns: user:*, event:*")

	// Start subscriber
	go subscriber.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Publish messages matching patterns
	testMessages := []struct {
		channel string
		message string
		matches bool
	}{
		{"user:123", "User 123 logged in", true},
		{"user:456", "User 456 updated profile", true},
		{"event:login", "Login event triggered", true},
		{"event:logout", "Logout event triggered", true},
		{"other:message", "This should not match", false},
	}

	for _, test := range testMessages {
		err = publisher.Publish(ctx, test.channel, test.message)
		if err != nil {
			fmt.Printf("Failed to publish to %s: %v", test.channel, err)
			continue
		}

		if test.matches {
			fmt.Printf("Published to %s (matches pattern)", test.channel)
		} else {
			fmt.Printf("Published to %s (should not match pattern)", test.channel)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for messages to be processed
	time.Sleep(500 * time.Millisecond)

	fmt.Printf("Results: %d messages received (expected: 4, non-matching: 1)", counter.GetCount())
}

// exampleScenarioErrorHandling demonstrates error handling in message handlers
func exampleScenarioErrorHandling(ctx context.Context, client *redis.Client) {
	fmt.Println("Testing error handling in message handlers...")

	counter := &MessageCounter{}
	errorCounter := 0
	var mu sync.Mutex

	// Create a handler that returns an error for specific messages
	errorHandler := redis.HandlerFunc(func(ctx context.Context, channel string, message string) error {
		if message == "error_trigger" {
			mu.Lock()
			errorCounter++
			mu.Unlock()
			return fmt.Errorf("intentional error for message: %s", message)
		}

		counter.Increment()
		fmt.Printf("[ErrorTestHandler] Received: %s", message)
		return nil
	})

	config := redis.NewPubSubConfig().
		WithPoolSize(1)

	subscriber, err := redis.NewSubscriber(client.GetClient(), errorHandler, config)
	if err != nil {
		fmt.Printf("Failed to create subscriber: %v", err)
		return
	}
	defer subscriber.Close()

	publisher := redis.NewPublisher(client.GetClient(), config)

	// Subscribe to channel
	err = subscriber.Subscribe(ctx, "error_test_channel")
	if err != nil {
		fmt.Printf("Failed to subscribe: %v", err)
		return
	}
	fmt.Println("Subscribed to error_test_channel")

	// Start subscriber
	go subscriber.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Publish test messages
	messages := []struct {
		content     string
		shouldError bool
	}{
		{"normal_message_1", false},
		{"normal_message_2", false},
		{"error_trigger", true},
		{"normal_message_3", false},
		{"error_trigger", true},
		{"normal_message_4", false},
	}

	for i, msg := range messages {
		err = publisher.Publish(ctx, "error_test_channel", msg.content)
		if err != nil {
			fmt.Printf("Failed to publish message %d: %v", i+1, err)
			continue
		}

		if msg.shouldError {
			fmt.Printf("Published error-triggering message %d", i+1)
		} else {
			fmt.Printf("Published normal message %d", i+1)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for messages to be processed
	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	errors := errorCounter
	mu.Unlock()

	fmt.Printf("=== RESULTS ===")
	fmt.Printf("Total messages: %d", len(messages))
	fmt.Printf("Successful: %d", counter.GetCount())
	fmt.Printf("Errors handled: %d", errors)
}

// exampleScenarioHealthCheck demonstrates health check and monitoring
func exampleScenarioHealthCheck(ctx context.Context, client *redis.Client) {
	fmt.Println("Demonstrating health check and subscriber monitoring...")

	counter := &MessageCounter{}
	handler := &CustomHandler{counter: counter, name: "HealthCheckHandler"}

	config := redis.NewPubSubConfig().
		WithPoolSize(3).
		WithLogLevel(redis.InfoLevel).
		WithReconnectDelay(1 * time.Second).
		WithMaxReconnectAttempts(5).
		WithChannelNamespace("monitoring")

	subscriber, err := redis.NewSubscriber(client.GetClient(), handler, config)
	if err != nil {
		fmt.Printf("Failed to create subscriber: %v", err)
		return
	}
	defer subscriber.Close()

	publisher := redis.NewPublisher(client.GetClient(), config)

	// Subscribe to channels
	err = subscriber.Subscribe(ctx, "health_channel_1", "health_channel_2")
	if err != nil {
		fmt.Printf("Failed to subscribe: %v", err)
		return
	}
	fmt.Println("Subscribed to health channels")

	// Start subscriber
	go subscriber.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Check health before publishing
	fmt.Println("--- Health Check: Before Publishing ---")
	showHealthCheck(subscriber)

	// Publish some messages
	for i := 1; i <= 5; i++ {
		channel := "health_channel_1"
		if i%2 == 0 {
			channel = "health_channel_2"
		}

		err = publisher.Publish(ctx, channel, fmt.Sprintf("Health check message %d", i))
		if err != nil {
			fmt.Printf("Failed to publish message %d: %v", i, err)
			continue
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for messages to be processed
	time.Sleep(500 * time.Millisecond)

	// Check health after publishing
	fmt.Println("--- Health Check: After Publishing ---")
	showHealthCheck(subscriber)

	fmt.Printf("Total messages received: %d", counter.GetCount())

	// Close subscriber and show final health check
	fmt.Println("--- Closing Subscriber ---")
	subscriber.Close()
	time.Sleep(100 * time.Millisecond) // Wait for graceful shutdown

	fmt.Println("--- Health Check: After Closing ---")
	showHealthCheck(subscriber)
}

// showHealthCheck displays the health status of a subscriber
func showHealthCheck(subscriber *redis.Subscriber) {
	healthCheck := subscriber.HealthCheck()

	fmt.Printf("Status: %s", healthCheck.Status)

	if len(healthCheck.Details) > 0 {
		fmt.Println("Details:")
		for key, value := range healthCheck.Details {
			fmt.Printf("  - %s: %s", key, value)
		}
	}
}

// getEnvOrDefault returns the value of an environment variable or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvOrDefaultInt returns the value of an environment variable as int or a default value
func getEnvOrDefaultInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// maskPassword masks the password for display purposes
func maskPassword(password string) string {
	if password == "" {
		return "(empty)"
	}
	return "***"
}
