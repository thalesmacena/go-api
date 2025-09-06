package main

import (
	"context"
	"fmt"
	"log"
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
	fmt.Printf("[%s] Received message on channel '%s': %s (total: %d)\n",
		h.name, channel, message, h.counter.GetCount())
	return nil
}

func main() {
	// Create Redis configuration using fluent API
	config := redis.NewRedisConfig().
		WithHost("localhost").
		WithPort(6379).
		WithPassword("").
		WithDatabase(0).
		WithMinIdleConns(5).
		WithMaxIdleConns(10).
		WithDialTimeout(5 * time.Second).
		WithReadTimeout(3 * time.Second).
		WithWriteTimeout(3 * time.Second).
		WithPoolTimeout(4 * time.Second)

	// Create Redis client
	client := redis.NewClient(config)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("Testing Redis Pub/Sub functionality...")

	// Create pub/sub configuration using builder
	pubsubConfig := redis.NewPubSubConfig().
		WithPoolSize(2).
		WithLogLevel(redis.InfoLevel).
		WithReconnectDelay(1 * time.Second).
		WithMaxReconnectAttempts(5).
		WithChannelNamespace("user_service")

	// Test 1: Basic publisher and subscriber
	fmt.Println("\n1. Testing basic publisher and subscriber...")

	counter1 := &MessageCounter{}
	handler1 := &CustomHandler{counter: counter1, name: "Handler1"}

	pubSubConfig := pubsubConfig

	subscriber1, err := redis.NewSubscriber(client.GetClient(), handler1, pubSubConfig)
	if err != nil {
		log.Fatalf("Failed to create subscriber1: %v", err)
	}
	publisher1 := redis.NewPublisher(client.GetClient(), pubSubConfig)

	// Subscribe to channels
	err = subscriber1.Subscribe(ctx, "test_channel", "another_channel")
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}
	fmt.Println("✓ Subscribed to channels")
	fmt.Println("  - Channel format: user_service::test_channel, user_service::another_channel")

	// Start subscriber in background
	go subscriber1.Start(ctx)

	// Wait for subscription to be ready
	time.Sleep(100 * time.Millisecond)

	// Publish messages
	err = publisher1.Publish(ctx, "test_channel", "Hello from test_channel!")
	if err != nil {
		log.Fatalf("Failed to publish: %v", err)
	}
	fmt.Println("✓ Published message to test_channel")

	err = publisher1.Publish(ctx, "another_channel", "Hello from another_channel!")
	if err != nil {
		log.Fatalf("Failed to publish: %v", err)
	}
	fmt.Println("✓ Published message to another_channel")

	// Wait for messages to be processed
	time.Sleep(200 * time.Millisecond)

	// Test 2: JSON message publishing
	fmt.Println("\n2. Testing JSON message publishing...")

	type Message struct {
		ID      int       `json:"id"`
		Content string    `json:"content"`
		Time    time.Time `json:"time"`
	}

	jsonMessage := Message{
		ID:      1,
		Content: "This is a JSON message",
		Time:    time.Now(),
	}

	err = publisher1.PublishJSON(ctx, "test_channel", jsonMessage)
	if err != nil {
		log.Fatalf("Failed to publish JSON: %v", err)
	}
	fmt.Println("✓ Published JSON message")

	// Wait for JSON message to be processed
	time.Sleep(200 * time.Millisecond)

	// Test 3: Pattern subscription
	fmt.Println("\n3. Testing pattern subscription...")

	counter2 := &MessageCounter{}
	handler2 := &CustomHandler{counter: counter2, name: "PatternHandler"}

	subscriber2, err := redis.NewSubscriber(client.GetClient(), handler2, pubSubConfig)
	if err != nil {
		log.Fatalf("Failed to create subscriber2: %v", err)
	}

	// Subscribe to patterns
	err = subscriber2.PSubscribe(ctx, "user:*", "event:*")
	if err != nil {
		log.Fatalf("Failed to subscribe to patterns: %v", err)
	}
	fmt.Println("✓ Subscribed to patterns")

	// Start pattern subscriber in background
	go subscriber2.Start(ctx)

	// Wait for subscription to be ready
	time.Sleep(100 * time.Millisecond)

	// Publish messages matching patterns
	err = publisher1.Publish(ctx, "user:123", "User 123 message")
	if err != nil {
		log.Fatalf("Failed to publish to user:123: %v", err)
	}
	fmt.Println("✓ Published message to user:123")

	err = publisher1.Publish(ctx, "event:login", "User logged in")
	if err != nil {
		log.Fatalf("Failed to publish to event:login: %v", err)
	}
	fmt.Println("✓ Published message to event:login")

	err = publisher1.Publish(ctx, "other:message", "This should not match")
	if err != nil {
		log.Fatalf("Failed to publish to other:message: %v", err)
	}
	fmt.Println("✓ Published message to other:message (should not match pattern)")

	// Wait for pattern messages to be processed
	time.Sleep(200 * time.Millisecond)

	// Test 4: Health check
	fmt.Println("\n4. Testing pub/sub health check...")

	healthCheck1 := subscriber1.HealthCheck()
	fmt.Printf("✓ Subscriber1 health: %s\n", healthCheck1.Status)
	fmt.Printf("✓ Subscriber1 details: %+v\n", healthCheck1.Details)

	healthCheck2 := subscriber2.HealthCheck()
	fmt.Printf("✓ Subscriber2 health: %s\n", healthCheck2.Status)
	fmt.Printf("✓ Subscriber2 details: %+v\n", healthCheck2.Details)

	// Test 5: High-frequency message publishing
	fmt.Println("\n5. Testing high-frequency message publishing...")

	counter3 := &MessageCounter{}
	handler3 := &CustomHandler{counter: counter3, name: "HighFreqHandler"}

	highFreqConfig := redis.NewPubSubConfig().
		WithPoolSize(3).
		WithLogLevel(redis.ErrorLevel). // Reduce logging for high frequency
		WithReconnectDelay(500 * time.Millisecond).
		WithMaxReconnectAttempts(3)

	subscriber3, err := redis.NewSubscriber(client.GetClient(), handler3, highFreqConfig)
	if err != nil {
		log.Fatalf("Failed to create subscriber3: %v", err)
	}

	err = subscriber3.Subscribe(ctx, "high_freq_channel")
	if err != nil {
		log.Fatalf("Failed to subscribe subscriber3: %v", err)
	}

	go subscriber3.Start(ctx)

	// Wait for subscription to be ready
	time.Sleep(100 * time.Millisecond)

	// Publish many messages quickly
	for i := 0; i < 10; i++ {
		err = publisher1.Publish(ctx, "high_freq_channel", fmt.Sprintf("High frequency message %d", i+1))
		if err != nil {
			log.Printf("Failed to publish high freq message %d: %v", i+1, err)
		}
	}
	fmt.Println("✓ Published 10 high-frequency messages")

	// Wait for high-frequency messages to be processed
	time.Sleep(500 * time.Millisecond)

	// Test 6: Error handling
	fmt.Println("\n6. Testing error handling...")

	counter4 := &MessageCounter{}
	errorHandler := &CustomHandler{counter: counter4, name: "ErrorHandler"}

	// Create a handler that returns an error for testing
	errorHandlerFunc := redis.HandlerFunc(func(ctx context.Context, channel string, message string) error {
		if message == "error_message" {
			return fmt.Errorf("intentional error for testing")
		}
		errorHandler.HandleMessage(ctx, channel, message)
		return nil
	})

	errorSubscriber, err := redis.NewSubscriber(client.GetClient(), errorHandlerFunc, pubSubConfig)
	if err != nil {
		log.Fatalf("Failed to create error subscriber: %v", err)
	}

	err = errorSubscriber.Subscribe(ctx, "error_test_channel")
	if err != nil {
		log.Fatalf("Failed to subscribe error subscriber: %v", err)
	}

	go errorSubscriber.Start(ctx)

	// Wait for subscription to be ready
	time.Sleep(100 * time.Millisecond)

	// Publish normal message
	err = publisher1.Publish(ctx, "error_test_channel", "normal_message")
	if err != nil {
		log.Fatalf("Failed to publish normal message: %v", err)
	}
	fmt.Println("✓ Published normal message (should be processed)")

	// Publish error message
	err = publisher1.Publish(ctx, "error_test_channel", "error_message")
	if err != nil {
		log.Fatalf("Failed to publish error message: %v", err)
	}
	fmt.Println("✓ Published error message (should cause error)")

	// Wait for error messages to be processed
	time.Sleep(200 * time.Millisecond)

	// Print final statistics
	fmt.Println("\n7. Final statistics:")
	fmt.Printf("✓ Handler1 received %d messages\n", counter1.GetCount())
	fmt.Printf("✓ PatternHandler received %d messages\n", counter2.GetCount())
	fmt.Printf("✓ HighFreqHandler received %d messages\n", counter3.GetCount())
	fmt.Printf("✓ ErrorHandler received %d messages\n", counter4.GetCount())

	// Close subscribers
	fmt.Println("\n8. Closing subscribers...")
	err = subscriber1.Close()
	if err != nil {
		log.Printf("Failed to close subscriber1: %v", err)
	} else {
		fmt.Println("✓ Closed subscriber1")
	}

	err = subscriber2.Close()
	if err != nil {
		log.Printf("Failed to close subscriber2: %v", err)
	} else {
		fmt.Println("✓ Closed subscriber2")
	}

	err = subscriber3.Close()
	if err != nil {
		log.Printf("Failed to close subscriber3: %v", err)
	} else {
		fmt.Println("✓ Closed subscriber3")
	}

	err = errorSubscriber.Close()
	if err != nil {
		log.Printf("Failed to close error subscriber: %v", err)
	} else {
		fmt.Println("✓ Closed error subscriber")
	}

	fmt.Println("\n✓ All pub/sub tests completed successfully!")
}
