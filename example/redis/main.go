package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	redisPkg "go-api/pkg/redis"

	"github.com/redis/go-redis/v9"
)

// createStandardRedisConfig creates a standard Redis configuration
func createStandardRedisConfig(host string, port int, password string) *redisPkg.Config {
	return redisPkg.NewRedisConfig().
		WithHost(host).
		WithPort(port).
		WithPassword(password).
		WithDatabase(0).
		WithMinIdleConns(5).
		WithMaxIdleConns(10).
		WithMaxActive(100).
		WithDialTimeout(5 * time.Second).
		WithReadTimeout(3 * time.Second).
		WithWriteTimeout(3 * time.Second).
		WithPoolTimeout(4 * time.Second)
}

func main() {
	fmt.Println("Testing Redis Package with Comprehensive Examples...")

	// Get Redis configuration from environment variables
	redisHost := getEnvOrDefault("REDIS_HOST", "localhost")
	redisPort := getEnvOrDefaultInt("REDIS_PORT", 6379)
	redisPassword := getEnvOrDefault("REDIS_PASSWORD", "")

	fmt.Printf("Using Redis configuration: Host=%s, Port=%d, Password=%s\n",
		redisHost, redisPort, maskPassword(redisPassword))

	// =============================================================================
	// CONFIGURATION EXAMPLES
	// =============================================================================
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("CONFIGURATION EXAMPLES")
	fmt.Println(strings.Repeat("=", 60))

	// Example 1: Default configuration
	fmt.Println("\nUsing default configuration:")
	defaultConfig := redisPkg.NewRedisConfig()
	if redisPassword != "" {
		defaultConfig.Password = redisPassword
	}
	fmt.Println("✓ Default configuration created successfully")

	// Example 2: Custom configuration with fluent API
	fmt.Println("\nUsing custom configuration with fluent API:")
	_ = redisPkg.NewRedisConfig().
		WithHost(redisHost).
		WithPort(redisPort).
		WithPassword(redisPassword).
		WithDatabase(0).
		WithMinIdleConns(10).
		WithMaxIdleConns(15).
		WithMaxActive(150).
		WithMaxRetries(5).
		WithDialTimeout(10 * time.Second).
		WithReadTimeout(5 * time.Second).
		WithWriteTimeout(5 * time.Second).
		WithPoolTimeout(6 * time.Second)
	fmt.Println("✓ Custom configuration created successfully")

	// Example 3: Partial configuration (only changing some values)
	fmt.Println("\nUsing partial configuration:")
	_ = redisPkg.NewRedisConfig().
		WithHost(redisHost).
		WithPort(redisPort).
		WithPassword(redisPassword)
	fmt.Println("✓ Partial configuration created successfully")

	// Example 4: Configuration validation
	fmt.Println("\nTesting configuration validation:")
	validConfig := redisPkg.NewRedisConfig().WithPort(6379)
	err := validConfig.Validate()
	if err != nil {
		fmt.Println("Valid config validation failed: ", err)
	} else {
		fmt.Println("✓ Valid configuration passes validation")
	}

	// Invalid configuration test
	testInvalidConfiguration()

	// =============================================================================
	// COMPREHENSIVE SET/GET OPERATIONS EXAMPLES
	// =============================================================================
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("COMPREHENSIVE SET/GET OPERATIONS EXAMPLES")
	fmt.Println(strings.Repeat("=", 60))

	// Create Redis configuration and client for all tests
	config := createStandardRedisConfig(redisHost, redisPort, redisPassword)
	var client *redisPkg.Client = redisPkg.NewClient(config)
	defer func(client *redisPkg.Client) {
		err := client.Close()
		if err != nil {

		}
	}(client)

	ctx := context.Background()

	// Test the main client with a simple operation
	fmt.Println("\nTesting main client configuration:")
	err = client.Set(ctx, "config_test", "config_works", time.Minute)
	if err != nil {
		fmt.Printf("Main client test failed: %v\n", err)
	} else {
		value, err := client.Get(ctx, "config_test")
		if err != nil {
			fmt.Printf("Main client get test failed: %v\n", err)
		} else {
			fmt.Printf("✓ Main client configuration works: %s\n", value)
		}
	}

	// =============================================================================
	// STRING OPERATIONS (RAW STRING STORAGE)
	// =============================================================================
	fmt.Println("\nSTRING OPERATIONS (RAW STRING STORAGE)")
	fmt.Println(strings.Repeat("-", 40))

	// Basic string set/get (raw string storage - no type conversion)
	fmt.Println("\nBasic string operations (raw string storage):")
	err = client.Set(ctx, "string_key", "Hello Redis!", time.Hour)
	if err != nil {
		fmt.Printf("Failed to set string: %v\n", err)
		return
	}
	fmt.Println("✓ Set string value successfully")

	value, err := client.Get(ctx, "string_key")
	if err != nil {
		fmt.Printf("Failed to get string: %v\n", err)
		return
	}
	fmt.Printf("✓ Got string value: %s\n", value)

	// String with different TTLs
	fmt.Println("\nString with different TTLs:")
	err = client.Set(ctx, "short_ttl", "This expires in 10 seconds", 10*time.Second)
	if err != nil {
		fmt.Printf("Failed to set short TTL string: %v\n", err)
		return
	}
	fmt.Println("✓ Set string with 10s TTL")

	err = client.Set(ctx, "long_ttl", "This expires in 24 hours", 24*time.Hour)
	if err != nil {
		fmt.Printf("Failed to set long TTL string: %v", err)
	}
	fmt.Println("✓ Set string with 24h TTL")

	// Check TTL
	ttl, err := client.TTL(ctx, "short_ttl")
	if err != nil {
		fmt.Printf("Failed to get TTL: %v", err)
	}
	fmt.Printf("✓ Short TTL remaining: %v\n", ttl)

	// String operations with error handling
	fmt.Println("\nString operations with error handling:")

	// Try to get non-existent key (should not return error)
	nonExistentValue, err := client.Get(ctx, "non_existent_key")
	if err != nil {
		fmt.Printf("❌ Unexpected error for non-existent key: %v\n", err)
	} else {
		fmt.Printf("✓ Correctly handled non-existent key (no error): '%s'\n", nonExistentValue)
	}

	// Set empty string
	err = client.Set(ctx, "empty_string", "", time.Hour)
	if err != nil {
		fmt.Printf("Failed to set empty string: %v", err)
	}
	fmt.Println("✓ Set empty string successfully")

	emptyValue, err := client.Get(ctx, "empty_string")
	if err != nil {
		fmt.Printf("Failed to get empty string: %v", err)
	}
	fmt.Printf("✓ Got empty string: '%s' (length: %d)\n", emptyValue, len(emptyValue))

	// =============================================================================
	// CONDITIONAL OPERATIONS
	// =============================================================================
	// Conditional operations allow you to check if keys exist before performing
	// operations, or execute operations only under certain conditions.
	// These are essential for implementing atomic operations, avoiding race
	// conditions, and ensuring data consistency in concurrent environments.
	fmt.Println("\nCONDITIONAL OPERATIONS")
	fmt.Println(strings.Repeat("-", 40))

	// Check if key exists before setting
	fmt.Println("\nConditional set operations:")
	existsCount, err := client.Exists(ctx, "conditional_key")
	if err != nil {
		fmt.Printf("Failed to check if key exists: %v", err)
	}
	fmt.Printf("✓ Key 'conditional_key' exists: %t\n", existsCount > 0)

	// Set only if key doesn't exist
	err = client.Set(ctx, "conditional_key", "conditional_value", time.Hour)
	if err != nil {
		fmt.Printf("Failed to set conditional key: %v", err)
	}
	fmt.Println("✓ Set conditional key")

	// Check existence again
	existsCount, err = client.Exists(ctx, "conditional_key")
	if err != nil {
		fmt.Printf("Failed to check if key exists: %v", err)
	}
	fmt.Printf("✓ Key 'conditional_key' exists after set: %t\n", existsCount > 0)

	// =============================================================================
	// NUMERIC OPERATIONS
	// =============================================================================
	fmt.Println("\nNUMERIC OPERATIONS")
	fmt.Println(strings.Repeat("-", 40))

	// Integer operations
	fmt.Println("\nInteger operations:")
	intValue := int64(42)
	err = client.SetInt(ctx, "int_key", intValue, time.Hour)
	if err != nil {
		fmt.Printf("Failed to set integer: %v", err)
	}
	fmt.Println("✓ Set integer value")

	retrievedInt, err := client.GetInt(ctx, "int_key")
	if err != nil {
		fmt.Printf("Failed to get integer: %v", err)
	}
	fmt.Printf("✓ Got integer value: %d (type: %T)\n", retrievedInt, retrievedInt)

	// Float operations
	fmt.Println("\nFloat operations:")
	floatValue := 3.14159
	err = client.SetFloat(ctx, "float_key", floatValue, time.Hour)
	if err != nil {
		fmt.Printf("Failed to set float: %v", err)
	}
	fmt.Println("✓ Set float value")

	retrievedFloat, err := client.GetFloat(ctx, "float_key")
	if err != nil {
		fmt.Printf("Failed to get float: %v", err)
	}
	fmt.Printf("✓ Got float value: %f (type: %T)\n", retrievedFloat, retrievedFloat)

	// =============================================================================
	// BOOLEAN OPERATIONS
	// =============================================================================
	fmt.Println("\nBOOLEAN OPERATIONS")
	fmt.Println(strings.Repeat("-", 40))

	// Boolean operations
	fmt.Println("\nBoolean operations:")
	err = client.SetBool(ctx, "bool_true", true, time.Hour)
	if err != nil {
		fmt.Printf("Failed to set boolean true: %v", err)
	}
	fmt.Println("✓ Set boolean true")

	err = client.SetBool(ctx, "bool_false", false, time.Hour)
	if err != nil {
		fmt.Printf("Failed to set boolean false: %v", err)
	}
	fmt.Println("✓ Set boolean false")

	retrievedBoolTrue, err := client.GetBool(ctx, "bool_true")
	if err != nil {
		fmt.Printf("Failed to get boolean true: %v", err)
	}
	fmt.Printf("✓ Got boolean true: %t (type: %T)\n", retrievedBoolTrue, retrievedBoolTrue)

	retrievedBoolFalse, err := client.GetBool(ctx, "bool_false")
	if err != nil {
		fmt.Printf("Failed to get boolean false: %v", err)
	}
	fmt.Printf("✓ Got boolean false: %t (type: %T)\n", retrievedBoolFalse, retrievedBoolFalse)

	// =============================================================================
	// JSON OPERATIONS
	// =============================================================================
	fmt.Println("\nJSON OPERATIONS")
	fmt.Println(strings.Repeat("-", 40))

	// Complex structures with JSON
	fmt.Println("\nComplex structures (JSON is appropriate here):")
	type ComplexProduct struct {
		ID         int                    `json:"id"`
		Name       string                 `json:"name"`
		Price      float64                `json:"price"`
		Quantity   int                    `json:"quantity"`
		Active     bool                   `json:"active"`
		Discount   float64                `json:"discount"`
		Categories []string               `json:"categories"`
		Metadata   map[string]interface{} `json:"metadata"`
	}

	product := ComplexProduct{
		ID:         1001,
		Name:       "Laptop Pro",
		Price:      99.99,
		Quantity:   5,
		Active:     true,
		Discount:   0.15,
		Categories: []string{"electronics", "computers", "laptops"},
		Metadata: map[string]interface{}{
			"brand":    "TechCorp",
			"warranty": "2 years",
			"weight":   2.5,
			"features": []string{"SSD", "16GB RAM", "Touch Screen"},
		},
	}

	err = client.SetJSON(ctx, "product_complex", product, time.Hour)
	if err != nil {
		fmt.Printf("Failed to set complex product: %v", err)
	}
	fmt.Println("✓ Set complex product structure with JSON")

	var retrievedProduct ComplexProduct
	err = client.GetJSON(ctx, "product_complex", &retrievedProduct)
	if err != nil {
		fmt.Printf("Failed to get complex product: %v", err)
	}
	fmt.Printf("✓ Got complex product: %s (ID: %d, Price: $%.2f, Categories: %v)\n",
		retrievedProduct.Name, retrievedProduct.ID, retrievedProduct.Price, retrievedProduct.Categories)

	// =============================================================================
	// HASH OPERATIONS
	// =============================================================================
	// Hash operations allow you to store field-value pairs in a single key.
	// Think of it like a JSON object or a map where you can store multiple
	// attributes for a single entity. Perfect for user profiles, product
	// details, or any structured data that belongs together.
	fmt.Println("\nHASH OPERATIONS")
	fmt.Println(strings.Repeat("-", 40))

	// Hash operations
	fmt.Println("\nHash operations:")
	err = client.HSet(ctx, "user:1001", "name", "John Doe", "email", "john@example.com", "age", "30")
	if err != nil {
		fmt.Printf("Failed to set hash: %v", err)
	}
	fmt.Println("✓ Set hash fields successfully")

	name, err := client.HGet(ctx, "user:1001", "name")
	if err != nil {
		fmt.Printf("Failed to get hash field: %v", err)
	}
	fmt.Printf("✓ Got hash field 'name': %s\n", name)

	email, err := client.HGet(ctx, "user:1001", "email")
	if err != nil {
		fmt.Printf("Failed to get hash field: %v", err)
	}
	fmt.Printf("✓ Got hash field 'email': %s\n", email)

	// Get all hash fields
	allFields, err := client.HGetAll(ctx, "user:1001")
	if err != nil {
		fmt.Printf("Failed to get all hash fields: %v", err)
	}
	fmt.Printf("✓ Got all hash fields: %+v\n", allFields)

	// Check if field exists
	exists, err := client.HExists(ctx, "user:1001", "age")
	if err != nil {
		fmt.Printf("Failed to check hash field existence: %v", err)
	}
	fmt.Printf("✓ Hash field 'age' exists: %t\n", exists)

	// =============================================================================
	// LIST OPERATIONS
	// =============================================================================
	// List operations provide ordered collections of strings, similar to arrays.
	// You can add elements to the beginning (LPUSH) or end (RPUSH) of the list,
	// and retrieve elements by position. Perfect for queues, stacks, timelines,
	// or any ordered sequence of items.
	fmt.Println("\nLIST OPERATIONS")
	fmt.Println(strings.Repeat("-", 40))

	// List operations
	fmt.Println("\nList operations:")
	err = client.LPush(ctx, "tasks", "task1", "task2", "task3")
	if err != nil {
		fmt.Printf("Failed to push to list: %v", err)
	}
	fmt.Println("✓ Pushed items to list")

	err = client.RPush(ctx, "tasks", "task4", "task5")
	if err != nil {
		fmt.Printf("Failed to append to list: %v", err)
	}
	fmt.Println("✓ Appended items to list")

	// Get list length
	length, err := client.LLen(ctx, "tasks")
	if err != nil {
		fmt.Printf("Failed to get list length: %v", err)
	}
	fmt.Printf("✓ List length: %d\n", length)

	// Get list range
	items, err := client.LRange(ctx, "tasks", 0, -1)
	if err != nil {
		fmt.Printf("Failed to get list range: %v", err)
	}
	fmt.Printf("✓ List items: %v\n", items)

	// Pop from list
	popped, err := client.LPop(ctx, "tasks")
	if err != nil {
		fmt.Printf("Failed to pop from list: %v", err)
	}
	fmt.Printf("✓ Popped from left: %s\n", popped)

	// =============================================================================
	// SET OPERATIONS
	// =============================================================================
	// Set operations provide unordered collections of unique strings.
	// Each element appears only once in a set, making them perfect for
	// tags, categories, unique user IDs, or any collection where
	// duplicates are not allowed and order doesn't matter.
	fmt.Println("\nSET OPERATIONS")
	fmt.Println(strings.Repeat("-", 40))

	// Set operations
	fmt.Println("\nSet operations:")
	err = client.SAdd(ctx, "tags", "golang", "redis", "database", "cache")
	if err != nil {
		fmt.Printf("Failed to add to set: %v", err)
	}
	fmt.Println("✓ Added items to set")

	// Get all set members
	members, err := client.SMembers(ctx, "tags")
	if err != nil {
		fmt.Printf("Failed to get set members: %v", err)
	}
	fmt.Printf("✓ Set members: %v\n", members)

	// Check if member exists
	isMember, err := client.SIsMember(ctx, "tags", "golang")
	if err != nil {
		fmt.Printf("Failed to check set membership: %v", err)
	}
	fmt.Printf("✓ 'golang' is member of set: %t\n", isMember)

	// Remove member
	err = client.SRem(ctx, "tags", "cache")
	if err != nil {
		fmt.Printf("Failed to remove from set: %v", err)
	}
	fmt.Println("✓ Removed 'cache' from set")

	// =============================================================================
	// SORTED SET OPERATIONS
	// =============================================================================
	// Sorted set operations combine the uniqueness of sets with ordering by score.
	// Each member has an associated score (float) that determines its position.
	// Perfect for leaderboards, rankings, priority queues, or any collection
	// where you need both uniqueness and automatic sorting.
	fmt.Println("\nSORTED SET OPERATIONS")
	fmt.Println(strings.Repeat("-", 40))

	// Sorted set operations
	fmt.Println("\nSorted set operations:")
	err = client.ZAdd(ctx, "leaderboard", redis.Z{Score: 100, Member: "player1"}, redis.Z{Score: 200, Member: "player2"}, redis.Z{Score: 150, Member: "player3"})
	if err != nil {
		fmt.Printf("Failed to add to sorted set: %v", err)
	}
	fmt.Println("✓ Added items to sorted set")

	// Get sorted set range
	rankings, err := client.ZRange(ctx, "leaderboard", 0, -1)
	if err != nil {
		fmt.Printf("Failed to get sorted set range: %v", err)
	}
	fmt.Printf("✓ Rankings: %v\n", rankings)

	// Get sorted set with scores
	rankingsWithScores, err := client.ZRangeWithScores(ctx, "leaderboard", 0, -1)
	if err != nil {
		fmt.Printf("Failed to get sorted set with scores: %v", err)
	}
	fmt.Printf("✓ Rankings with scores: %v\n", rankingsWithScores)

	// =============================================================================
	// INCREMENT OPERATIONS
	// =============================================================================
	// Increment operations allow you to atomically increase or decrease numeric values.
	// These operations are thread-safe and perfect for counters, statistics,
	// rate limiting, or any scenario where you need to track numeric changes
	// without race conditions.
	fmt.Println("\nINCREMENT OPERATIONS")
	fmt.Println(strings.Repeat("-", 40))

	// Increment operations
	fmt.Println("\nIncrement operations:")
	err = client.SetInt(ctx, "counter", 0, time.Hour)
	if err != nil {
		fmt.Printf("Failed to set initial counter: %v", err)
	}

	// Increment by 1
	newValue, err := client.Incr(ctx, "counter")
	if err != nil {
		fmt.Printf("Failed to increment: %v", err)
	}
	fmt.Printf("✓ Incremented counter to: %d\n", newValue)

	// Increment by specific amount
	newValue, err = client.IncrBy(ctx, "counter", 5)
	if err != nil {
		fmt.Printf("Failed to increment by 5: %v", err)
	}
	fmt.Printf("✓ Incremented counter by 5 to: %d\n", newValue)

	// Decrement by 1
	newValue, err = client.Decr(ctx, "counter")
	if err != nil {
		fmt.Printf("Failed to decrement: %v", err)
	}
	fmt.Printf("✓ Decremented counter to: %d\n", newValue)

	// Decrement by specific amount
	newValue, err = client.DecrBy(ctx, "counter", 3)
	if err != nil {
		fmt.Printf("Failed to decrement by 3: %v", err)
	}
	fmt.Printf("✓ Decremented counter by 3 to: %d\n", newValue)

	// =============================================================================
	// PIPELINE OPERATIONS
	// =============================================================================
	// Pipeline operations allow you to batch multiple Redis commands together
	// and execute them in a single round-trip to the server. This significantly
	// improves performance when you need to execute many commands sequentially.
	// Perfect for bulk operations, data migration, or any high-throughput scenario.
	fmt.Println("\nPIPELINE OPERATIONS")
	fmt.Println(strings.Repeat("-", 40))

	// Pipeline operations
	fmt.Println("\nPipeline operations:")
	pipe := client.Pipeline()

	// Add multiple operations to pipeline
	pipe.Set(ctx, "pipeline_key1", "value1", time.Hour)
	pipe.Set(ctx, "pipeline_key2", "value2", time.Hour)
	pipe.Set(ctx, "pipeline_key3", "value3", time.Hour)
	pipe.Get(ctx, "pipeline_key1")
	pipe.Get(ctx, "pipeline_key2")
	pipe.Get(ctx, "pipeline_key3")

	// Execute pipeline
	cmders, err := pipe.Exec(ctx)
	if err != nil {
		fmt.Printf("Pipeline execution failed: %v", err)
	}
	fmt.Printf("✓ Pipeline executed successfully with %d commands\n", len(cmders))

	// Get results from pipeline
	for i, cmder := range cmders {
		if i >= 3 { // Only get commands (first 3 are sets, last 3 are gets)
			if getCmd, ok := cmder.(*redis.StringCmd); ok {
				value, err := getCmd.Result()
				if err != nil {
					fmt.Printf("Failed to get pipeline result: %v", err)
				} else {
					fmt.Printf("✓ Pipeline result %d: %s\n", i-2, value)
				}
			}
		}
	}

	// =============================================================================
	// PERFORMANCE TESTING
	// =============================================================================
	fmt.Println("\nPERFORMANCE TESTING")
	fmt.Println(strings.Repeat("-", 40))

	// Performance test with multiple operations
	fmt.Println("\nPerformance test with 100 operations:")
	start := time.Now()

	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("perf_key_%d", i)
		value := fmt.Sprintf("perf_value_%d", i)

		err = client.Set(ctx, key, value, time.Hour)
		if err != nil {
			fmt.Printf("Failed to set perf key %s: %v", key, err)
		}

		_, err = client.Get(ctx, key)
		if err != nil {
			fmt.Printf("Failed to get perf key %s: %v", key, err)
		}
	}

	duration := time.Since(start)
	fmt.Printf("✓ Completed 100 set/get operations in %v\n", duration)
	fmt.Printf("✓ Average time per operation: %v\n", duration/200) // 200 total operations (100 set + 100 get)

	// Test health check
	fmt.Println("\nHEALTH CHECK")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Println("\nTesting health check...")
	healthChecker := redisPkg.NewHealthChecker(client.GetClient(), config)
	healthCheck := healthChecker.HealthCheck()
	fmt.Printf("✓ Health check status: %s\n", healthCheck.Status)
	fmt.Printf("✓ Health check details: %+v\n", healthCheck.Details)

	// Test key scanning
	fmt.Println("\nTesting key scanning...")
	scannedKeys, err := redisPkg.ScanKeys(ctx, client, "pipeline_*", 10)
	if err != nil {
		fmt.Printf("Key scanning failed: %v", err)
	}
	fmt.Printf("✓ Found %d keys matching pattern: %v\n", len(scannedKeys), scannedKeys)

	// Use the same client for cleanup
	healthClient := client

	// =============================================================================
	// CLEANUP
	// =============================================================================
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("CLEANUP")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Println("\nCleaning up test keys...")
	keysToDelete := []string{
		// Configuration test keys
		"config_test",

		// String operations
		"string_key", "short_ttl", "long_ttl", "empty_string",

		// Numeric operations
		"int_key", "float_key",

		// Boolean operations
		"bool_true", "bool_false",

		// JSON operations
		"product_complex",

		// Hash operations
		"user:1001",

		// List operations
		"tasks",

		// Set operations
		"tags",

		// Sorted set operations
		"leaderboard",

		// Increment operations
		"counter",

		// Pipeline operations
		"pipeline_key1", "pipeline_key2", "pipeline_key3",

		// Conditional operations
		"conditional_key",

		// Performance test keys
		"perf_key_0", "perf_key_1", "perf_key_2", "perf_key_3", "perf_key_4", "perf_key_5", "perf_key_6", "perf_key_7", "perf_key_8", "perf_key_9",
		"perf_key_10", "perf_key_11", "perf_key_12", "perf_key_13", "perf_key_14", "perf_key_15", "perf_key_16", "perf_key_17", "perf_key_18", "perf_key_19",
		"perf_key_20", "perf_key_21", "perf_key_22", "perf_key_23", "perf_key_24", "perf_key_25", "perf_key_26", "perf_key_27", "perf_key_28", "perf_key_29",
		"perf_key_30", "perf_key_31", "perf_key_32", "perf_key_33", "perf_key_34", "perf_key_35", "perf_key_36", "perf_key_37", "perf_key_38", "perf_key_39",
		"perf_key_40", "perf_key_41", "perf_key_42", "perf_key_43", "perf_key_44", "perf_key_45", "perf_key_46", "perf_key_47", "perf_key_48", "perf_key_49",
		"perf_key_50", "perf_key_51", "perf_key_52", "perf_key_53", "perf_key_54", "perf_key_55", "perf_key_56", "perf_key_57", "perf_key_58", "perf_key_59",
		"perf_key_60", "perf_key_61", "perf_key_62", "perf_key_63", "perf_key_64", "perf_key_65", "perf_key_66", "perf_key_67", "perf_key_68", "perf_key_69",
		"perf_key_70", "perf_key_71", "perf_key_72", "perf_key_73", "perf_key_74", "perf_key_75", "perf_key_76", "perf_key_77", "perf_key_78", "perf_key_79",
		"perf_key_80", "perf_key_81", "perf_key_82", "perf_key_83", "perf_key_84", "perf_key_85", "perf_key_86", "perf_key_87", "perf_key_88", "perf_key_89",
		"perf_key_90", "perf_key_91", "perf_key_92", "perf_key_93", "perf_key_94", "perf_key_95", "perf_key_96", "perf_key_97", "perf_key_98", "perf_key_99",
	}

	deletedCount := 0
	for _, key := range keysToDelete {
		err := healthClient.Delete(ctx, key)
		if err != nil {
			fmt.Printf("Failed to delete key %s: %v\n", key, err)
		} else {
			deletedCount++
		}
	}
	fmt.Printf("✓ Cleaned up %d test keys\n", deletedCount)

	// =============================================================================
	// FLUSH OPERATIONS (DANGEROUS - EXECUTES ACTUAL FLUSH COMMANDS!)
	// =============================================================================
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("FLUSH OPERATIONS (DANGEROUS - WILL CLEAR ALL DATA!)")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("⚠️  WARNING: These operations will permanently delete ALL data!")
	fmt.Println("⚠️  Only run these on test/development environments!")
	fmt.Println("⚠️  These operations are IRREVERSIBLE!")
	fmt.Println()

	// Check which flush commands are available
	fmt.Println("Checking flush commands availability...")
	flushAvailability, err := client.CheckFlushCommandsAvailability(ctx)
	if err != nil {
		fmt.Printf("Failed to check flush commands availability: %v\n", err)
	} else {
		fmt.Printf("✓ FlushDB available: %t\n", flushAvailability["FlushDB"])
		fmt.Printf("✓ FlushAll available: %t\n", flushAvailability["FlushAll"])
		if !flushAvailability["FlushDB"] || !flushAvailability["FlushAll"] {
			fmt.Println("⚠️  Some flush commands are not available, will use fallback methods")
		}
	}
	fmt.Println()

	// Add test data for flush demonstration
	fmt.Println("Adding test data for flush operations demonstration...")
	flushTestKeys := []string{"flush_demo_1", "flush_demo_2", "flush_demo_3", "flush_demo_4", "flush_demo_5"}
	for i, key := range flushTestKeys {
		err = client.Set(ctx, key, fmt.Sprintf("flush_test_value_%d", i+1), time.Hour)
		if err != nil {
			fmt.Printf("Failed to set flush test key %s: %v\n", key, err)
		} else {
			fmt.Printf("✓ Set flush test key: %s\n", key)
		}
	}

	// Count keys before flush operations
	dbSizeBeforeFlush, err := client.GetDBSize(ctx)
	if err != nil {
		fmt.Printf("Failed to get database size before flush: %v\n", err)
	} else {
		fmt.Printf("✓ Total keys before flush operations: %d\n", dbSizeBeforeFlush)
	}

	// Execute FlushDB - removes all keys from current database
	fmt.Println("\nExecuting FlushDB (removes all keys from current database)...")
	err = client.FlushDBWithFallback(ctx)
	if err != nil {
		fmt.Printf("❌ FlushDB failed: %v\n", err)
	} else {
		fmt.Println("✓ FlushDB executed successfully!")
	}

	// Verify FlushDB results
	dbSizeAfterFlushDB, err := client.GetDBSize(ctx)
	if err != nil {
		fmt.Printf("Failed to get database size after FlushDB: %v\n", err)
	} else {
		fmt.Printf("✓ Keys remaining after FlushDB: %d\n", dbSizeAfterFlushDB)
		if dbSizeAfterFlushDB == 0 {
			fmt.Println("✓ All keys successfully removed by FlushDB!")
		}
	}

	// Add more test data for FlushAll demonstration
	fmt.Println("\nAdding more test data for FlushAll demonstration...")
	for i, key := range flushTestKeys {
		err = client.Set(ctx, key, fmt.Sprintf("flushall_test_value_%d", i+1), time.Hour)
		if err != nil {
			fmt.Printf("Failed to set FlushAll test key %s: %v\n", key, err)
		} else {
			fmt.Printf("✓ Set FlushAll test key: %s\n", key)
		}
	}

	// Count keys before FlushAll
	dbSizeBeforeFlushAll, err := client.GetDBSize(ctx)
	if err != nil {
		fmt.Printf("Failed to get database size before FlushAll: %v\n", err)
	} else {
		fmt.Printf("✓ Total keys before FlushAll: %d\n", dbSizeBeforeFlushAll)
	}

	// Execute FlushAll - removes all keys from all databases
	fmt.Println("\nExecuting FlushAll (removes all keys from all databases)...")
	err = client.FlushAllWithFallback(ctx)
	if err != nil {
		fmt.Printf("❌ FlushAll failed: %v\n", err)
	} else {
		fmt.Println("✓ FlushAll executed successfully!")
	}

	// Verify FlushAll results
	dbSizeAfterFlushAll, err := client.GetDBSize(ctx)
	if err != nil {
		fmt.Printf("Failed to get database size after FlushAll: %v\n", err)
	} else {
		fmt.Printf("✓ Keys remaining after FlushAll: %d\n", dbSizeAfterFlushAll)
		if dbSizeAfterFlushAll == 0 {
			fmt.Println("✓ All keys successfully removed by FlushAll!")
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

// testInvalidConfiguration tests invalid configuration and recovers from panic
func testInvalidConfiguration() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("✓ Invalid configuration properly panicked: %v\n", r)
		}
	}()

	fmt.Println("\nTesting invalid configuration (will panic):")
	invalidConfig := redisPkg.NewRedisConfig().WithPort(99999) // Invalid port
	redisPkg.NewClient(invalidConfig)                          // This should panic
}
