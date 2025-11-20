package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"go-api/pkg/redis"
)

// createStandardRedisConfig creates a standard Redis configuration
func createStandardRedisConfig(host string, port int, password string) *redis.Config {
	return redis.NewRedisConfig().
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
	fmt.Println("Testing Redis Cache Package with Comprehensive Examples...")

	// Get Redis configuration from environment variables
	redisHost := getEnvOrDefault("REDIS_HOST", "localhost")
	redisPort := getEnvOrDefaultInt("REDIS_PORT", 6379)
	redisPassword := getEnvOrDefault("REDIS_PASSWORD", "")

	fmt.Printf("Using Redis configuration: Host=%s, Port=%d, Password=%s\n",
		redisHost, redisPort, maskPassword(redisPassword))

	// Create Redis configuration and client for all tests
	config := createStandardRedisConfig(redisHost, redisPort, redisPassword)
	client := redis.NewClient(config)
	defer func(client *redis.Client) {
		err := client.Close()
		if err != nil {
			fmt.Printf("Failed to close client: %v\n", err)
		}
	}(client)

	ctx := context.Background()

	// =============================================================================
	// BASIC CACHE OPERATIONS
	// =============================================================================
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("BASIC CACHE OPERATIONS")
	fmt.Println(strings.Repeat("=", 60))

	// Basic cache operations
	fmt.Println("\nBasic Set and Get operations:")
	cache := redis.NewCache(client, nil)
	err := cache.Set(ctx, "cache_key", "cache_value")
	if err != nil {
		fmt.Printf("Failed to set cache: %v", err)
		return
	}
	fmt.Println("✓ Set cache value successfully")

	var cacheValue string
	err = cache.Get(ctx, "cache_key", &cacheValue)
	if err != nil {
		fmt.Printf("Failed to get cache: %v", err)
		return
	}
	fmt.Printf("✓ Got cache value: %s\n", cacheValue)

	// =============================================================================
	// CACHE WITH JSON DATA
	// =============================================================================
	fmt.Println("\nCache with JSON data structures:")
	type User struct {
		ID       int                    `json:"id"`
		Name     string                 `json:"name"`
		Email    string                 `json:"email"`
		Active   bool                   `json:"active"`
		Metadata map[string]interface{} `json:"metadata"`
	}

	user := User{
		ID:     123,
		Name:   "John Doe",
		Email:  "john@example.com",
		Active: true,
		Metadata: map[string]interface{}{
			"last_login": "2024-01-15T10:30:00Z",
			"role":       "admin",
			"preferences": map[string]interface{}{
				"theme": "dark",
				"lang":  "en",
			},
		},
	}

	err = cache.Set(ctx, "user_123", user)
	if err != nil {
		fmt.Printf("Failed to set user cache: %v", err)
		return
	}
	fmt.Println("✓ Set user JSON data successfully")

	var retrievedUser User
	err = cache.Get(ctx, "user_123", &retrievedUser)
	if err != nil {
		fmt.Printf("Failed to get user cache: %v", err)
		return
	}
	fmt.Printf("✓ Got user JSON data: ID=%d, Name=%s, Email=%s, Active=%t\n",
		retrievedUser.ID, retrievedUser.Name, retrievedUser.Email, retrievedUser.Active)
	fmt.Printf("  Metadata: %+v\n", retrievedUser.Metadata)

	// =============================================================================
	// CACHE EXISTS OPERATION
	// =============================================================================
	fmt.Println("\nCache Exists operation:")
	exists, err := cache.Exists(ctx, "user_123")
	if err != nil {
		fmt.Printf("Failed to check cache existence: %v", err)
		return
	}
	fmt.Printf("✓ Cache key 'user_123' exists: %t\n", exists)

	exists, err = cache.Exists(ctx, "non_existent_key")
	if err != nil {
		fmt.Printf("Failed to check cache existence: %v", err)
		return
	}
	fmt.Printf("✓ Cache key 'non_existent_key' exists: %t\n", exists)

	// =============================================================================
	// CACHE GETORSET PATTERN
	// =============================================================================
	fmt.Println("\nCache GetOrSet pattern:")
	type Product struct {
		ID          int     `json:"id"`
		Name        string  `json:"name"`
		Price       float64 `json:"price"`
		Description string  `json:"description"`
		InStock     bool    `json:"in_stock"`
	}

	var expensiveProduct Product
	err = cache.GetOrSet(ctx, "product_456", &expensiveProduct, func() (interface{}, error) {
		// Simulate expensive database operation
		time.Sleep(100 * time.Millisecond)
		return Product{
			ID:          456,
			Name:        "Premium Widget",
			Price:       99.99,
			Description: "A high-quality widget with advanced features",
			InStock:     true,
		}, nil
	})
	if err != nil {
		fmt.Printf("Failed to get or set product cache: %v", err)
		return
	}
	fmt.Printf("✓ Got or set product data: ID=%d, Name=%s, Price=$%.2f\n",
		expensiveProduct.ID, expensiveProduct.Name, expensiveProduct.Price)

	// Try to get the same data again (should come from cache)
	var cachedProduct Product
	err = cache.GetOrSet(ctx, "product_456", &cachedProduct, func() (interface{}, error) {
		// This should not be called since data is in cache
		return Product{}, fmt.Errorf("this should not be called")
	})
	if err != nil {
		fmt.Printf("Failed to get cached product: %v", err)
		return
	}
	fmt.Printf("✓ Got cached product data: ID=%d, Name=%s, Price=$%.2f\n",
		cachedProduct.ID, cachedProduct.Name, cachedProduct.Price)

	// =============================================================================
	// CACHE WITH ARRAYS AND SLICES
	// =============================================================================
	fmt.Println("\nCache with arrays and slices:")
	type Order struct {
		ID         int      `json:"id"`
		Items      []string `json:"items"`
		Quantities []int    `json:"quantities"`
		Total      float64  `json:"total"`
	}

	order := Order{
		ID:         789,
		Items:      []string{"Widget A", "Widget B", "Widget C"},
		Quantities: []int{2, 1, 3},
		Total:      149.97,
	}

	err = cache.Set(ctx, "order_789", order)
	if err != nil {
		fmt.Printf("Failed to set order cache: %v", err)
		return
	}
	fmt.Println("✓ Set order with arrays successfully")

	var retrievedOrder Order
	err = cache.Get(ctx, "order_789", &retrievedOrder)
	if err != nil {
		fmt.Printf("Failed to get order cache: %v", err)
		return
	}
	fmt.Printf("✓ Got order data: ID=%d, Items=%v, Quantities=%v, Total=$%.2f\n",
		retrievedOrder.ID, retrievedOrder.Items, retrievedOrder.Quantities, retrievedOrder.Total)

	// =============================================================================
	// MULTIPLE CACHE OPERATIONS
	// =============================================================================
	fmt.Println("\nMultiple cache operations (MSet/MGet):")
	cacheDataMap := map[string]interface{}{
		"item1": "value1",
		"item2": "value2",
		"item3": "value3",
		"item4": "value4",
	}

	err = cache.MSet(ctx, cacheDataMap)
	if err != nil {
		fmt.Printf("Failed to set multiple cache items: %v", err)
		return
	}
	fmt.Println("✓ Set multiple cache items successfully")

	keys := []string{"item1", "item2", "item3", "item4"}
	results, err := cache.MGet(ctx, keys)
	if err != nil {
		fmt.Printf("Failed to get multiple cache items: %v", err)
		return
	}
	fmt.Printf("✓ Got %d cache items from multiple get\n", len(results))
	for key, data := range results {
		fmt.Printf("  - %s: %s\n", key, string(data))
	}

	// =============================================================================
	// CACHE WITH DIFFERENT DATA TYPES
	// =============================================================================
	fmt.Println("\nCache with different data types:")

	// Integer
	err = cache.Set(ctx, "int_data", 42)
	if err != nil {
		fmt.Printf("Failed to set int data: %v", err)
		return
	}
	var intData int
	err = cache.Get(ctx, "int_data", &intData)
	if err != nil {
		fmt.Printf("Failed to get int data: %v", err)
		return
	}
	fmt.Printf("✓ Integer data: %d\n", intData)

	// Float
	err = cache.Set(ctx, "float_data", 3.14159)
	if err != nil {
		fmt.Printf("Failed to set float data: %v", err)
		return
	}
	var floatData float64
	err = cache.Get(ctx, "float_data", &floatData)
	if err != nil {
		fmt.Printf("Failed to get float data: %v", err)
		return
	}
	fmt.Printf("✓ Float data: %f\n", floatData)

	// Boolean
	err = cache.Set(ctx, "bool_data", true)
	if err != nil {
		fmt.Printf("Failed to set bool data: %v", err)
		return
	}
	var boolData bool
	err = cache.Get(ctx, "bool_data", &boolData)
	if err != nil {
		fmt.Printf("Failed to get bool data: %v", err)
		return
	}
	fmt.Printf("✓ Boolean data: %t\n", boolData)

	// Array
	err = cache.Set(ctx, "array_data", []string{"item1", "item2", "item3"})
	if err != nil {
		fmt.Printf("Failed to set array data: %v", err)
		return
	}
	var arrayData []string
	err = cache.Get(ctx, "array_data", &arrayData)
	if err != nil {
		fmt.Printf("Failed to get array data: %v", err)
		return
	}
	fmt.Printf("✓ Array data: %v\n", arrayData)

	// =============================================================================
	// CACHE WITH CUSTOM SERIALIZER/DESERIALIZER
	// =============================================================================
	fmt.Println("\nCache with custom serializer/deserializer:")

	// Custom serializer that adds prefix
	customSerializer := func(v interface{}) ([]byte, error) {
		str := fmt.Sprintf("PREFIX:%v", v)
		return []byte(str), nil
	}

	// Custom deserializer that removes prefix
	customDeserializer := func(data []byte, v interface{}) error {
		str := string(data)
		if strings.HasPrefix(str, "PREFIX:") {
			// Remove prefix and set the value
			cleanStr := strings.TrimPrefix(str, "PREFIX:")
			if ptr, ok := v.(*string); ok {
				*ptr = cleanStr
			}
		}
		return nil
	}

	customSerialCache := redis.NewCache(client, redis.NewCacheOptions().
		WithTTL(time.Hour).
		WithSerializer(customSerializer).
		WithDeserializer(customDeserializer).
		WithCacheName("custom_serial_cache"))

	err = customSerialCache.Set(ctx, "custom_data", "original_value")
	if err != nil {
		fmt.Printf("Failed to set custom serial cache: %v", err)
		return
	}
	fmt.Println("✓ Set custom serial cache")

	var customValue string
	err = customSerialCache.Get(ctx, "custom_data", &customValue)
	if err != nil {
		fmt.Printf("Failed to get custom serial cache: %v", err)
		return
	}
	fmt.Printf("✓ Got custom serial cache value: %s\n", customValue)

	// =============================================================================
	// CACHE PERFORMANCE TESTING
	// =============================================================================
	fmt.Println("\nCache performance testing:")
	perfCache := redis.NewCache(client, redis.NewCacheOptions().
		WithTTL(time.Hour).
		WithCacheName("perf_cache"))

	// Performance test with multiple operations
	fmt.Println("\nPerformance test with 100 operations:")
	start := time.Now()

	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("perf_key_%d", i)
		value := fmt.Sprintf("perf_value_%d", i)

		err = perfCache.Set(ctx, key, value)
		if err != nil {
			fmt.Printf("Failed to set perf key %s: %v", key, err)
		}

		var retrievedValue string
		err = perfCache.Get(ctx, key, &retrievedValue)
		if err != nil {
			fmt.Printf("Failed to get perf key %s: %v", key, err)
		}
	}

	duration := time.Since(start)
	fmt.Printf("✓ Completed 100 set/get operations in %v\n", duration)
	fmt.Printf("✓ Average time per operation: %v\n", duration/200) // 200 total operations (100 set + 100 get)

	// =============================================================================
	// CACHE TTL CONFIGURATION AND HIERARCHY
	// =============================================================================
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("CACHE TTL CONFIGURATION AND HIERARCHY")
	fmt.Println(strings.Repeat("=", 60))

	// Create a new client with specific cache TTL configurations
	clientConfig := redis.NewRedisConfig().
		WithHost(redisHost).
		WithPort(redisPort).
		WithPassword(redisPassword).
		WithDatabase(0).
		WithMinIdleConns(5).
		WithMaxIdleConns(10).
		WithMaxActive(100).
		WithDialTimeout(5*time.Second).
		WithReadTimeout(3*time.Second).
		WithWriteTimeout(3*time.Second).
		WithPoolTimeout(4*time.Second).
		// Configure specific TTLs for different cache names
		WithCacheTTL("user_cache", 30*time.Minute).
		WithCacheTTL("session_cache", 2*time.Hour).
		WithCacheTTL("temp_cache", 5*time.Minute).
		// Set default TTL for caches not specified above
		WithDefaultCacheTTL(1 * time.Hour)

	clientWithConfig := redis.NewClient(clientConfig)
	defer func(client *redis.Client) {
		err := client.Close()
		if err != nil {
			fmt.Printf("Failed to close client with config: %v\n", err)
		}
	}(clientWithConfig)

	fmt.Println("\n✓ Created client with specific cache TTL configurations:")
	fmt.Println("  - user_cache: 30 minutes")
	fmt.Println("  - session_cache: 2 hours")
	fmt.Println("  - temp_cache: 5 minutes")
	fmt.Println("  - default TTL: 1 hour")

	// =============================================================================
	// CACHE WITH DIFFERENT CACHE NAMES USING CLIENT CONFIG
	// =============================================================================
	fmt.Println("\nCache with different cache names using client config:")

	// Session cache (2 hours TTL from client config)
	sessionCache := redis.NewCache(clientWithConfig, redis.NewCacheOptions().
		WithCacheName("session_cache"))

	err = sessionCache.Set(ctx, "session_123", "session_data")
	if err != nil {
		fmt.Printf("Failed to set session cache: %v", err)
		return
	}
	fmt.Println("✓ Set session cache (TTL: 2 hours from client config)")

	// Temp cache (5 minutes TTL from client config)
	tempCache := redis.NewCache(clientWithConfig, redis.NewCacheOptions().
		WithCacheName("temp_cache"))

	err = tempCache.Set(ctx, "temp_data", "temporary_data")
	if err != nil {
		fmt.Printf("Failed to set temp cache: %v", err)
		return
	}
	fmt.Println("✓ Set temp cache (TTL: 5 minutes from client config)")

	// Unknown cache name (uses default TTL from client config)
	unknownCache := redis.NewCache(clientWithConfig, redis.NewCacheOptions().
		WithCacheName("unknown_cache"))

	err = unknownCache.Set(ctx, "unknown_data", "unknown_data")
	if err != nil {
		fmt.Printf("Failed to set unknown cache: %v", err)
		return
	}
	fmt.Println("✓ Set unknown cache (TTL: 1 hour default from client config)")

	// Check TTLs to verify they match client configuration
	sessionTTL, err := sessionCache.GetTTL(ctx, "session_123")
	if err != nil {
		fmt.Printf("Failed to get session TTL: %v", err)
	} else {
		fmt.Printf("✓ Session TTL remaining: %v (expected ~2 hours)\n", sessionTTL)
	}

	tempTTL, err := tempCache.GetTTL(ctx, "temp_data")
	if err != nil {
		fmt.Printf("Failed to get temp TTL: %v", err)
	} else {
		fmt.Printf("✓ Temp TTL remaining: %v (expected ~5 minutes)\n", tempTTL)
	}

	unknownTTL, err := unknownCache.GetTTL(ctx, "unknown_data")
	if err != nil {
		fmt.Printf("Failed to get unknown TTL: %v", err)
	} else {
		fmt.Printf("✓ Unknown TTL remaining: %v (expected ~1 hour default)\n", unknownTTL)
	}

	// =============================================================================
	// CACHE TTL HIERARCHY DEMONSTRATION
	// =============================================================================
	fmt.Println("\nCache TTL hierarchy demonstration:")
	fmt.Println("Priority: Duration Parameter > CacheOptions > Client.Config")

	// Test 1: Duration parameter (highest priority)
	fmt.Println("\nDuration Parameter (Highest Priority):")
	durationCache := redis.NewCache(clientWithConfig, redis.NewCacheOptions().
		WithTTL(10*time.Minute).     // This should be overridden by duration parameter
		WithCacheName("user_cache")) // Client config has 30 min for this cache

	err = durationCache.SetWithTTL(ctx, "duration_data", "TTL from duration parameter", 5*time.Minute)
	if err != nil {
		fmt.Printf("Failed to set duration cache: %v", err)
		return
	}
	fmt.Println("✓ Set cache with duration parameter (5 min)")
	fmt.Println("  - Duration parameter: 5 minutes")
	fmt.Println("  - CacheOptions TTL: 10 minutes")
	fmt.Println("  - Client config TTL for 'user_cache': 30 minutes")
	fmt.Println("  - Expected result: Duration parameter (5 minutes) should win")

	durationTTL, err := durationCache.GetTTL(ctx, "duration_data")
	if err != nil {
		fmt.Printf("Failed to get duration TTL: %v", err)
	} else {
		fmt.Printf("✓ Duration TTL remaining: %v (should be ~5 minutes)\n", durationTTL)
	}

	// Test 2: CacheOptions TTL (medium priority)
	fmt.Println("\nCacheOptions TTL (Medium Priority):")
	cacheOptionsCache := redis.NewCache(clientWithConfig, redis.NewCacheOptions().
		WithTTL(15*time.Minute).     // This should override client config
		WithCacheName("user_cache")) // Client config has 30 min for this cache

	err = cacheOptionsCache.Set(ctx, "cache_options_data", "TTL from CacheOptions")
	if err != nil {
		fmt.Printf("Failed to set cache options cache: %v", err)
		return
	}
	fmt.Println("✓ Set cache with CacheOptions TTL (15 min)")
	fmt.Println("  - CacheOptions TTL: 15 minutes")
	fmt.Println("  - Client config TTL for 'user_cache': 30 minutes")
	fmt.Println("  - Expected result: CacheOptions TTL (15 minutes) should win")

	cacheOptionsTTL, err := cacheOptionsCache.GetTTL(ctx, "cache_options_data")
	if err != nil {
		fmt.Printf("Failed to get cache options TTL: %v", err)
	} else {
		fmt.Printf("✓ CacheOptions TTL remaining: %v (should be ~15 minutes)\n", cacheOptionsTTL)
	}

	// Test 3: Client.Config TTL (lowest priority - fallback)
	fmt.Println("\nClient.Config TTL (Lowest Priority - Fallback):")
	clientConfigCache := redis.NewCache(clientWithConfig, redis.NewCacheOptions().
		WithCacheName("user_cache")) // No TTL in CacheOptions, should use client config

	err = clientConfigCache.Set(ctx, "client_config_data", "TTL from Client.Config")
	if err != nil {
		fmt.Printf("Failed to set client config cache: %v", err)
		return
	}
	fmt.Println("✓ Set cache without CacheOptions TTL")
	fmt.Println("  - CacheOptions TTL: 0 (not set)")
	fmt.Println("  - Client config TTL for 'user_cache': 30 minutes")
	fmt.Println("  - Expected result: Client config TTL (30 minutes) should be used")

	clientConfigTTL, err := clientConfigCache.GetTTL(ctx, "client_config_data")
	if err != nil {
		fmt.Printf("Failed to get client config TTL: %v", err)
	} else {
		fmt.Printf("✓ Client config TTL remaining: %v (should be ~30 minutes)\n", clientConfigTTL)
	}

	// Test 4: Default TTL (final fallback)
	fmt.Println("\nDefault TTL (Final Fallback):")
	defaultCache := redis.NewCache(clientWithConfig, redis.NewCacheOptions().
		WithCacheName("unknown_cache")) // No specific config for this cache name

	err = defaultCache.Set(ctx, "default_data", "TTL from default config")
	if err != nil {
		fmt.Printf("Failed to set default cache: %v", err)
		return
	}
	fmt.Println("✓ Set cache with unknown cache name")
	fmt.Println("  - CacheOptions TTL: 0 (not set)")
	fmt.Println("  - Client config TTL for 'unknown_cache': not configured")
	fmt.Println("  - Client config default TTL: 1 hour")
	fmt.Println("  - Expected result: Default TTL (1 hour) should be used")

	defaultTTL, err := defaultCache.GetTTL(ctx, "default_data")
	if err != nil {
		fmt.Printf("Failed to get default TTL: %v", err)
	} else {
		fmt.Printf("✓ Default TTL remaining: %v (should be ~1 hour)\n", defaultTTL)
	}

	// =============================================================================
	// CACHE WITH REFRESH TTL
	// =============================================================================
	fmt.Println("\nCache with refresh TTL:")
	refreshCache := redis.NewCache(client, redis.NewCacheOptions().
		WithTTL(5*time.Second).
		WithRefreshTTL(true).
		WithCacheName("refresh_cache"))

	err = refreshCache.Set(ctx, "refresh_data", "This refreshes TTL on access")
	if err != nil {
		fmt.Printf("Failed to set refresh cache: %v", err)
		return
	}
	fmt.Println("✓ Set refresh cache")

	// Access the data multiple times to refresh TTL
	for i := 0; i < 3; i++ {
		var refreshValue string
		err = refreshCache.Get(ctx, "refresh_data", &refreshValue)
		if err != nil {
			fmt.Printf("Failed to get refresh cache: %v", err)
			return
		}
		fmt.Printf("✓ Access %d: %s\n", i+1, refreshValue)
		time.Sleep(1 * time.Second)
	}

	// =============================================================================
	// CACHE WITH SETWITH TTL
	// =============================================================================
	fmt.Println("\nCache with SetWithTTL:")
	customCache := redis.NewCache(clientWithConfig, redis.NewCacheOptions().
		WithTTL(30*time.Minute).
		WithRefreshTTL(true).
		WithCacheName("user_cache"))

	err = customCache.SetWithTTL(ctx, "custom_ttl_data", "This has custom TTL", 15*time.Second)
	if err != nil {
		fmt.Printf("Failed to set with custom TTL: %v", err)
		return
	}
	fmt.Println("✓ Set with custom TTL")

	customTTL, err := customCache.GetTTL(ctx, "custom_ttl_data")
	if err != nil {
		fmt.Printf("Failed to get custom TTL: %v", err)
	} else {
		fmt.Printf("✓ Custom TTL remaining: %v\n", customTTL)
	}

	// =============================================================================
	// CACHE EXTEND TTL
	// =============================================================================
	fmt.Println("\nCache extend TTL:")
	err = customCache.ExtendTTL(ctx, "user_123", 1*time.Hour)
	if err != nil {
		fmt.Printf("Failed to extend TTL: %v", err)
		return
	}
	fmt.Println("✓ Extended TTL")

	extendedTTL, err := customCache.GetTTL(ctx, "user_123")
	if err != nil {
		fmt.Printf("Failed to get extended TTL: %v", err)
	} else {
		fmt.Printf("✓ Extended TTL remaining: %v\n", extendedTTL)
	}

	// =============================================================================
	// CACHE CLEAR OPERATIONS
	// =============================================================================
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("CACHE CLEAR OPERATIONS")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Println("\nCache clear operations:")

	// Clear specific pattern
	err = customCache.Clear(ctx, "item*")
	if err != nil {
		fmt.Printf("Failed to clear pattern: %v", err)
		return
	}
	fmt.Println("✓ Cleared items matching pattern 'item*'")

	// Clear entire cache name
	err = customCache.ClearCacheName(ctx)
	if err != nil {
		fmt.Printf("Failed to clear cache name: %v", err)
		return
	}
	fmt.Println("✓ Cleared entire cache name")

	// =============================================================================
	// CLEANUP
	// =============================================================================
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("CLEANUP")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Println("\nCleaning up test keys...")
	keysToDelete := []string{
		// Basic cache
		"cache_key",

		// JSON data
		"user_123", "product_456", "order_789",

		// Multiple operations
		"item1", "item2", "item3", "item4",

		// Different data types
		"int_data", "float_data", "bool_data", "array_data",

		// Custom serial cache
		"custom_serial_cache::custom_data",

		// Client config caches
		"user_cache::duration_data", "user_cache::cache_options_data", "user_cache::client_config_data",
		"session_cache::session_123",
		"temp_cache::temp_data",
		"unknown_cache::unknown_data", "unknown_cache::default_data",

		// Refresh cache
		"refresh_cache::refresh_data",

		// Custom TTL data
		"user_cache::custom_ttl_data",

		// Performance cache
		"perf_cache::int_data", "perf_cache::float_data", "perf_cache::bool_data", "perf_cache::array_data",
	}

	// Add performance test keys
	for i := 0; i < 100; i++ {
		keysToDelete = append(keysToDelete, fmt.Sprintf("perf_cache::perf_key_%d", i))
	}

	deletedCount := 0
	for _, key := range keysToDelete {
		err := client.Delete(ctx, key)
		if err != nil {
			fmt.Printf("Failed to delete key %s: %v\n", key, err)
		} else {
			deletedCount++
		}
	}
	fmt.Printf("✓ Cleaned up %d test keys\n", deletedCount)

	fmt.Println("\n✓ All cache tests completed successfully!")
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
