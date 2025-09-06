package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"go-api/pkg/redis"
)

func main() {
	fmt.Println("Testing Redis Package with Comprehensive Examples...")

	// =============================================================================
	// CONFIGURATION EXAMPLES
	// =============================================================================
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("CONFIGURATION EXAMPLES")
	fmt.Println(strings.Repeat("=", 60))

	// Example 1: Default configuration
	fmt.Println("\n1. Using default configuration:")
	defaultConfig := redis.NewRedisConfig()
	client1 := redis.NewClient(defaultConfig)
	defer client1.Close()

	ctx := context.Background()
	err := client1.Set(ctx, "default_test", "default_value", time.Minute)
	if err != nil {
		log.Printf("Default config test failed: %v", err)
	} else {
		fmt.Println("✓ Default configuration works")
	}

	// Example 2: Custom configuration with fluent API
	fmt.Println("\n2. Using custom configuration with fluent API:")
	customConfig := redis.NewRedisConfig().
		WithHost("localhost").
		WithPort(6379).
		WithPassword("").
		WithDatabase(0).
		WithPoolSize(20).
		WithMinIdleConns(10).
		WithMaxRetries(5).
		WithDialTimeout(10 * time.Second).
		WithReadTimeout(5 * time.Second).
		WithWriteTimeout(5 * time.Second).
		WithPoolTimeout(6 * time.Second)

	client2 := redis.NewClient(customConfig)
	defer client2.Close()

	err = client2.Set(ctx, "custom_test", "custom_value", time.Minute)
	if err != nil {
		log.Printf("Custom config test failed: %v", err)
	} else {
		fmt.Println("✓ Custom configuration works")
	}

	// Example 3: Partial configuration (only changing some values)
	fmt.Println("\n3. Using partial configuration:")
	partialConfig := redis.NewRedisConfig().
		WithHost("localhost").
		WithPort(6379).
		WithPoolSize(15)

	client3 := redis.NewClient(partialConfig)
	defer client3.Close()

	err = client3.Set(ctx, "partial_test", "partial_value", time.Minute)
	if err != nil {
		log.Printf("Partial config test failed: %v", err)
	} else {
		fmt.Println("✓ Partial configuration works")
	}

	// Example 4: Configuration validation
	fmt.Println("\n4. Testing configuration validation:")

	// Valid configuration
	validConfig := redis.NewRedisConfig().WithPort(6379)
	err = validConfig.Validate()
	if err != nil {
		log.Printf("Valid config validation failed: %v", err)
	} else {
		fmt.Println("✓ Valid configuration passes validation")
	}

	// Invalid configuration (this will panic when creating client)
	fmt.Println("\n5. Testing invalid configuration (will panic):")
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("✓ Invalid configuration properly panicked: %v\n", r)
		}
	}()

	invalidConfig := redis.NewRedisConfig().WithPort(99999) // Invalid port
	redis.NewClient(invalidConfig)                          // This should panic

	// =============================================================================
	// BASIC OPERATIONS EXAMPLES
	// =============================================================================
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("BASIC OPERATIONS EXAMPLES")
	fmt.Println(strings.Repeat("=", 60))

	// Create Redis configuration using fluent API
	config := redis.NewRedisConfig().
		WithHost("localhost").
		WithPort(6379).
		WithPassword("").
		WithDatabase(0).
		WithPoolSize(10).
		WithMinIdleConns(5).
		WithDialTimeout(5 * time.Second).
		WithReadTimeout(3 * time.Second).
		WithWriteTimeout(3 * time.Second).
		WithPoolTimeout(4 * time.Second)

	// Create Redis client
	client := redis.NewClient(config)
	defer client.Close()

	// Test basic operations
	fmt.Println("\n1. Testing basic Redis operations...")

	// Set a value
	err = client.Set(ctx, "test_key", "test_value", time.Hour)
	if err != nil {
		log.Fatalf("Failed to set value: %v", err)
	}
	fmt.Println("✓ Set value successfully")

	// Get a value
	value, err := client.Get(ctx, "test_key")
	if err != nil {
		log.Fatalf("Failed to get value: %v", err)
	}
	fmt.Printf("✓ Got value: %s\n", value)

	// Test JSON operations
	fmt.Println("\n2. Testing JSON operations...")
	type User struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	user := User{
		ID:    1,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	err = client.SetJSON(ctx, "user:1", user, time.Hour)
	if err != nil {
		log.Fatalf("Failed to set JSON: %v", err)
	}
	fmt.Println("✓ Set JSON value successfully")

	var retrievedUser User
	err = client.GetJSON(ctx, "user:1", &retrievedUser)
	if err != nil {
		log.Fatalf("Failed to get JSON: %v", err)
	}
	fmt.Printf("✓ Got JSON value: %+v\n", retrievedUser)

	// Test cache operations
	fmt.Println("\n3. Testing cache operations...")

	// Cache with default options
	cache := redis.NewCache(client, nil)
	err = cache.Set(ctx, "cache_key", "cache_value")
	if err != nil {
		log.Fatalf("Failed to set cache: %v", err)
	}
	fmt.Println("✓ Set cache value successfully")

	var cacheValue string
	err = cache.Get(ctx, "cache_key", &cacheValue)
	if err != nil {
		log.Fatalf("Failed to get cache: %v", err)
	}
	fmt.Printf("✓ Got cache value: %s\n", cacheValue)

	// Cache with custom options using builder
	cacheOptions := redis.NewCacheOptions().
		WithTTL(30 * time.Minute).
		WithRefreshTTL(true).
		WithCacheName("user_cache")

	customCache := redis.NewCache(client, cacheOptions)
	err = customCache.Set(ctx, "user_123", "user_data")
	if err != nil {
		log.Fatalf("Failed to set custom cache: %v", err)
	}
	fmt.Println("✓ Set custom cache value successfully")
	fmt.Println("  - Cache key format: user_cache::user_123")

	// Test health check
	fmt.Println("\n4. Testing health check...")
	healthChecker := redis.NewHealthChecker(client.GetClient(), config)
	healthCheck := healthChecker.HealthCheck()
	fmt.Printf("✓ Health check status: %s\n", healthCheck.Status)
	fmt.Printf("✓ Health check details: %+v\n", healthCheck.Details)

	// Test batch operations
	fmt.Println("\n5. Testing batch operations...")
	operations := []redis.BatchOperation{
		{Operation: "SET", Key: "batch_key1", Value: "value1", TTL: time.Hour},
		{Operation: "SET", Key: "batch_key2", Value: "value2", TTL: time.Hour},
		{Operation: "GET", Key: "batch_key1"},
		{Operation: "GET", Key: "batch_key2"},
	}

	result, err := redis.ExecuteBatch(ctx, client, operations)
	if err != nil {
		log.Fatalf("Batch operation failed: %v", err)
	}
	fmt.Printf("✓ Batch operation completed: %d successful, %d failed\n",
		len(result.Successful), len(result.Failed))

	// Test key scanning
	fmt.Println("\n6. Testing key scanning...")
	keys, err := redis.ScanKeys(ctx, client, "batch_*", 10)
	if err != nil {
		log.Fatalf("Key scanning failed: %v", err)
	}
	fmt.Printf("✓ Found %d keys matching pattern: %v\n", len(keys), keys)

	// =============================================================================
	// BUILDERS EXAMPLES
	// =============================================================================
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("BUILDERS EXAMPLES")
	fmt.Println(strings.Repeat("=", 60))

	// Cache builder example
	fmt.Println("\n1. Cache Builder Example:")
	cacheBuilder := redis.NewCacheOptions().
		WithTTL(2 * time.Hour).
		WithRefreshTTL(true).
		WithCacheName("session_cache")

	cacheWithBuilder := redis.NewCache(client, cacheBuilder)
	err = cacheWithBuilder.Set(ctx, "session_456", "session_data")
	if err != nil {
		log.Printf("Cache builder test failed: %v", err)
	} else {
		fmt.Println("✓ Cache builder works")
		fmt.Println("  - Cache key format: session_cache::session_456")
	}

	// Lock builder example
	fmt.Println("\n2. Lock Builder Example:")
	lockBuilder := redis.NewLockOptions().
		WithTTL(60 * time.Second).
		WithRetryDelay(200 * time.Millisecond).
		WithMaxRetries(5).
		WithRefreshInterval(15 * time.Second).
		WithLockNamespace("user_service")

	lockWithBuilder := redis.NewLock(client, "test_lock", lockBuilder)
	err = lockWithBuilder.Lock(ctx)
	if err != nil {
		log.Printf("Lock builder test failed: %v", err)
	} else {
		fmt.Println("✓ Lock builder works - lock acquired")
		fmt.Println("  - Lock key format: user_service::test_lock")
		err = lockWithBuilder.Unlock(ctx)
		if err != nil {
			log.Printf("Failed to unlock: %v", err)
		}
	}

	// PubSub builder example
	fmt.Println("\n3. PubSub Builder Example:")
	pubsubBuilder := redis.NewPubSubConfig().
		WithPoolSize(3).
		WithLogLevel(redis.InfoLevel).
		WithReconnectDelay(2 * time.Second).
		WithMaxReconnectAttempts(5).
		WithChannelNamespace("notification_service")

	fmt.Printf("✓ PubSub builder works - PoolSize: %d, LogLevel: %d\n",
		pubsubBuilder.PoolSize, pubsubBuilder.LogLevel)
	fmt.Println("  - Channel format: notification_service::channel_name")

	// =============================================================================
	// HEALTH CHECK WITH CUSTOM CONFIGURATION
	// =============================================================================
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("HEALTH CHECK WITH CUSTOM CONFIGURATION")
	fmt.Println(strings.Repeat("=", 60))

	healthConfig := redis.NewRedisConfig().
		WithHost("localhost").
		WithPort(6379).
		WithPoolSize(5)

	healthClient := redis.NewClient(healthConfig)
	defer healthClient.Close()

	healthChecker2 := redis.NewHealthChecker(healthClient.GetClient(), healthConfig)
	healthCheck2 := healthChecker2.HealthCheck()

	fmt.Printf("✓ Health check status: %s\n", healthCheck2.Status)
	fmt.Printf("✓ Pool size: %s\n", healthCheck2.Details["pool_size"])
	fmt.Printf("✓ Host: %s\n", healthCheck2.Details["host"])
	fmt.Printf("✓ Port: %s\n", healthCheck2.Details["port"])

	// =============================================================================
	// CLEANUP
	// =============================================================================
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("CLEANUP")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Println("\nCleaning up test keys...")
	keysToDelete := []string{"default_test", "custom_test", "partial_test", "test_key", "user:1", "cache_key", "user_cache::user_123", "session_cache::session_456", "batch_key1", "batch_key2"}
	for _, key := range keysToDelete {
		err := healthClient.Delete(ctx, key)
		if err != nil {
			log.Printf("Failed to delete key %s: %v", key, err)
		} else {
			fmt.Printf("✓ Deleted key: %s\n", key)
		}
	}

	// =============================================================================
	// SUMMARY
	// =============================================================================
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("SUMMARY")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Println("\n✓ All Redis package tests completed successfully!")
	fmt.Println("\nConfiguration API Summary:")
	fmt.Println("- redis.NewRedisConfig() creates a config with defaults")
	fmt.Println("- Use With* methods to customize configuration")
	fmt.Println("- Configuration is validated automatically")
	fmt.Println("- Fluent API allows chaining methods")
	fmt.Println("- Default values are preserved when not specified")
	fmt.Println("- Cache TTLs can be configured per cache name")
	fmt.Println("- Cache keys use format: CacheName::cacheKey")

	fmt.Println("\nBuilder APIs Summary:")
	fmt.Println("- redis.NewCacheOptions() - Cache configuration builder")
	fmt.Println("- redis.NewLockOptions() - Lock configuration builder")
	fmt.Println("- redis.NewPubSubConfig() - PubSub configuration builder")
	fmt.Println("- All builders support fluent API with With* methods")
	fmt.Println("- All components support namespacing with '::' separator")
	fmt.Println("- Cache keys: CacheName::cacheKey")
	fmt.Println("- Lock keys: LockNamespace::lockKey")
	fmt.Println("- Channel names: ChannelNamespace::channelName")

	fmt.Println("\nRun the following examples for specific functionality:")
	fmt.Println("- go run example/redis/lock/main.go")
	fmt.Println("- go run example/redis/pubsub/main.go")
}
