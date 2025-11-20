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

func main() {
	// Get Redis configuration from environment variables
	redisHost := getEnvOrDefault("REDIS_HOST", "localhost")
	redisPort := getEnvOrDefaultInt("REDIS_PORT", 6379)
	redisPassword := getEnvOrDefault("REDIS_PASSWORD", "redis_password")

	fmt.Printf("Using Redis configuration: Host=%s, Port=%d, Password=%s\n",
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
			fmt.Printf("Error closing Redis client: %v\n", err)
		}
	}(client)

	ctx := context.Background()

	// Warm-up Redis connection
	fmt.Println("Warming up Redis connection...")
	warmupLimiter, _ := redis.NewRateLimiter(client, "warmup", redis.NewRateLimiterOptions().
		WithMaxActiveTransactions(1).
		WithNamespace("warmup").
		WithCacheName("warmup_test"))

	for range 5 {
		txID, err := warmupLimiter.Acquire(ctx)
		if err == nil {
			warmupLimiter.Release(ctx, txID)
		}
	}
	warmupLimiter.Cleanup(ctx)
	fmt.Println("Warm-up complete!")

	fmt.Println("Example Redis Distributed Rate Limiter Scenarios...")

	// Example Scenario: Max active transactions only
	fmt.Println("\n=== Scenario: Max Active Transactions Only ===")
	exampleScenarioMaxActiveTransactions(ctx, client)

	// Example Scenario: Max transactions per second (TPS)
	fmt.Println("\n=== Scenario: Max Transactions Per Second (TPS) ===")
	exampleScenarioMaxTPS(ctx, client)

	// Example Scenario: Max transactions per minute (TPM)
	fmt.Println("\n=== Scenario: Max Transactions Per Minute (TPM) ===")
	exampleScenarioMaxTPM(ctx, client)

	// Example Scenario: Combined TPS and TPM limits
	fmt.Println("\n=== Scenario: Combined TPS and TPM Limits ===")
	exampleScenarioTPSandTPM(ctx, client)

	// Example Scenario: Combined limits (Active + TPS + TPM)
	fmt.Println("\n=== Scenario: Combined Limits (Active + TPS + TPM) ===")
	exampleScenarioCombinedLimits(ctx, client)

	// Example Scenario: Wait on limit vs immediate error
	fmt.Println("\n=== Scenario: Wait On Limit vs Immediate Error ===")
	exampleScenarioWaitVsImmediate(ctx, client)

	// Example Scenario: Health check and metrics monitoring
	fmt.Println("\n=== Scenario: Health Check and Metrics Monitoring ===")
	exampleScenarioHealthCheckMetrics(ctx, client)

	fmt.Println("\n All distributed rate limiter scenarios completed successfully!")
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

// exampleScenarioMaxActiveTransactions demonstrates rate limiting with max active transactions
func exampleScenarioMaxActiveTransactions(ctx context.Context, client *redis.Client) {
	fmt.Println("Testing with max 5 active transactions and job duration 1000ms...")

	opts := redis.NewRateLimiterOptions().
		WithMaxActiveTransactions(5).
		WithWaitOnLimit(false).
		WithNamespace("max_active").
		WithCacheName("max_active_test")

	limiter, err := redis.NewRateLimiter(client, "api_endpoint", opts)
	if err != nil {
		fmt.Printf("Failed to create rate limiter: %v\n", err)
		return
	}
	defer limiter.Cleanup(ctx)

	var wg sync.WaitGroup
	successCount := 0
	failureCount := 0
	var mu sync.Mutex

	// Send 20 requests over 2 seconds (1 request every 100ms)
	fmt.Println("Sending 20 requests over 2 seconds (1 every 100ms, task duration 1000ms)...")
	startTime := time.Now()

	for i := range 20 {
		wg.Add(1)
		go func(requestID int) {
			defer wg.Done()

			transactionID, err := limiter.Acquire(ctx)
			elapsed := time.Since(startTime)

			if err != nil {
				mu.Lock()
				failureCount++
				mu.Unlock()
				fmt.Printf("Request %02d [%4dms]:  Failed to acquire: %v\n", requestID, elapsed.Milliseconds(), err)
				return
			}

			mu.Lock()
			successCount++
			mu.Unlock()
			fmt.Printf("Request %02d [%4dms]:  Acquired transaction %s\n", requestID, elapsed.Milliseconds(), transactionID[:16]+"...")

			// Show metrics while transactions are active
			if requestID == 3 || requestID == 5 || requestID == 11 {
				showMetrics(ctx, limiter, fmt.Sprintf("During Processing (Request %02d)", requestID))
			}

			// Simulate work (500ms task duration)
			time.Sleep(1000 * time.Millisecond)

			err = limiter.Release(ctx, transactionID)
			elapsedRelease := time.Since(startTime)
			if err != nil {
				fmt.Printf("Request %02d [%4dms]: Failed to release: %v\n", requestID, elapsedRelease.Milliseconds(), err)
			} else {
				fmt.Printf("Request %02d [%4dms]:  Released transaction\n", requestID, elapsedRelease.Milliseconds())
			}
		}(i + 1)

		// Wait 100ms before next request
		time.Sleep(100 * time.Millisecond)
	}

	wg.Wait()

	totalElapsed := time.Since(startTime)
	fmt.Printf("\nResults: %d successful, %d failed out of 20 requests\n", successCount, failureCount)
	fmt.Printf("Total time: %v\n", totalElapsed)
	showMetrics(ctx, limiter, "After All Completed")
}

// exampleScenarioMaxTPS demonstrates rate limiting with max transactions per second
func exampleScenarioMaxTPS(ctx context.Context, client *redis.Client) {
	fmt.Println("Testing with max 5 transactions per second...")

	opts := redis.NewRateLimiterOptions().
		WithMaxTransactionsPerSecond(5).
		WithWaitOnLimit(false).
		WithNamespace("tps_limit").
		WithCacheName("tps_test")

	limiter, err := redis.NewRateLimiter(client, "tps_endpoint", opts)
	if err != nil {
		fmt.Printf("Failed to create rate limiter: %v\n", err)
		return
	}
	defer limiter.Cleanup(ctx)

	var wg sync.WaitGroup
	results := make([]bool, 20)
	var mu sync.Mutex

	// Send 20 requests over 2 seconds (1 request every 100ms)
	fmt.Println("Sending 20 requests over 2 seconds (1 every 100ms)...")
	startTime := time.Now()

	for i := range 20 {
		wg.Add(1)
		go func(requestID int) {
			defer wg.Done()

			_, err := limiter.Acquire(ctx)
			elapsed := time.Since(startTime)

			if err != nil {
				mu.Lock()
				results[requestID] = false
				mu.Unlock()
				fmt.Printf("Request %02d [%4dms]:  Rejected (TPS limit)\n", requestID+1, elapsed.Milliseconds())
				return
			}

			mu.Lock()
			results[requestID] = true
			mu.Unlock()
			fmt.Printf("Request %02d [%4dms]:  Accepted\n", requestID+1, elapsed.Milliseconds())

			// No Release - testing TPS only, not active transactions
		}(i)

		// Wait 100ms before next request
		time.Sleep(100 * time.Millisecond)
	}

	wg.Wait()

	// Count successful requests
	successCount := 0
	for _, success := range results {
		if success {
			successCount++
		}
	}

	totalElapsed := time.Since(startTime)
	fmt.Printf("\nResults: %d accepted, %d rejected (limit: 5 TPS)\n", successCount, len(results)-successCount)
	fmt.Printf("Total time: %v\n", totalElapsed)
}

// exampleScenarioMaxTPM demonstrates rate limiting with max transactions per minute
func exampleScenarioMaxTPM(ctx context.Context, client *redis.Client) {
	fmt.Println("Testing with max 45 transactions per minute...")

	opts := redis.NewRateLimiterOptions().
		WithMaxTransactionsPerMinute(45).
		WithWaitOnLimit(false).
		WithNamespace("tpm_limit").
		WithCacheName("tpm_test")

	limiter, err := redis.NewRateLimiter(client, "tpm_endpoint", opts)
	if err != nil {
		fmt.Printf("Failed to create rate limiter: %v\n", err)
		return
	}
	defer limiter.Cleanup(ctx)

	var wg sync.WaitGroup
	successCount := 0
	failureCount := 0
	var mu sync.Mutex

	// Send 70 requests over 70 seconds (1 request every second)
	fmt.Println("Sending 70 requests over 70 seconds (1 every second)...")
	startTime := time.Now()

	for i := range 70 {
		wg.Add(1)
		go func(requestID int) {
			defer wg.Done()

			_, err := limiter.Acquire(ctx)
			elapsed := time.Since(startTime)

			if err != nil {
				mu.Lock()
				failureCount++
				mu.Unlock()
				fmt.Printf("Request %03d [%3ds]:  Rejected (TPM limit)\n", requestID, int(elapsed.Seconds()))
				return
			}

			mu.Lock()
			successCount++
			mu.Unlock()
			fmt.Printf("Request %03d [%3ds]:  Accepted\n", requestID, int(elapsed.Seconds()))

			// No Release - testing TPM only, not active transactions
		}(i + 1)

		// Show metrics at specific intervals
		if (i+1)%20 == 0 {
			time.Sleep(100 * time.Millisecond)
			showMetrics(ctx, limiter, fmt.Sprintf("After %d requests", i+1))
		}

		// Wait 1 second before next request
		time.Sleep(1 * time.Second)
	}

	wg.Wait()

	totalElapsed := time.Since(startTime)
	fmt.Printf("\nResults: %d accepted, %d rejected (limit: 45 TPM)\n", successCount, failureCount)
	fmt.Printf("Total time: %v\n", totalElapsed)
	showMetrics(ctx, limiter, "After All Requests")
}

// exampleScenarioTPSandTPM demonstrates TPS and TPM limits working together
func exampleScenarioTPSandTPM(ctx context.Context, client *redis.Client) {
	fmt.Println("Testing with combined TPS and TPM limits: 3 TPS, 120 TPM...")
	fmt.Println("350 requests over 70 seconds (5 per second, 1 every 200ms)")

	opts := redis.NewRateLimiterOptions().
		WithMaxTransactionsPerSecond(3).
		WithMaxTransactionsPerMinute(120).
		WithWaitOnLimit(false).
		WithNamespace("tps_tpm_combined").
		WithCacheName("tps_tpm_test")

	limiter, err := redis.NewRateLimiter(client, "tps_tpm_endpoint", opts)
	if err != nil {
		fmt.Printf("Failed to create rate limiter: %v\n", err)
		return
	}
	defer limiter.Cleanup(ctx)

	var wg sync.WaitGroup
	stats := struct {
		total   int
		success int
		tps     int
		tpm     int
		mu      sync.Mutex
	}{}

	startTime := time.Now()

	// Send 350 requests over 70 seconds (1 every 200ms = 5 per second)
	for i := range 350 {
		wg.Add(1)
		go func(requestID int) {
			defer wg.Done()

			elapsed := time.Since(startTime)

			stats.mu.Lock()
			stats.total++
			stats.mu.Unlock()

			_, err := limiter.Acquire(ctx)
			if err != nil {
				errMsg := err.Error()
				stats.mu.Lock()
				if contains(errMsg, "per second") {
					stats.tps++
					fmt.Printf("Request %03d [%3ds]:  Rejected (TPS limit)\n", requestID, int(elapsed.Seconds()))
				} else if contains(errMsg, "per minute") {
					stats.tpm++
					fmt.Printf("Request %03d [%3ds]:  Rejected (TPM limit)\n", requestID, int(elapsed.Seconds()))
				}
				stats.mu.Unlock()
				return
			}

			stats.mu.Lock()
			stats.success++
			stats.mu.Unlock()
			fmt.Printf("Request %03d [%3ds]:  Accepted\n", requestID, int(elapsed.Seconds()))

			// No Release - testing only TPS and TPM without Active limit
		}(i + 1)

		// Show metrics at intervals (every 60 requests = every 12 seconds)
		if (i+1)%60 == 0 {
			time.Sleep(50 * time.Millisecond)
			showMetrics(ctx, limiter, fmt.Sprintf("After %d requests (~%ds)", i+1, int(time.Since(startTime).Seconds())))
		}

		// Wait 200ms before next request (5 requests per second)
		time.Sleep(200 * time.Millisecond)
	}

	wg.Wait()

	totalElapsed := time.Since(startTime)
	fmt.Printf("\n=== RESULTS SUMMARY ===\n")
	fmt.Printf("Total duration: %v\n", totalElapsed)
	fmt.Printf("Total requests: %d\n", stats.total)
	fmt.Printf("Successful: %d\n", stats.success)
	fmt.Printf("Rejected by TPS limit: %d\n", stats.tps)
	fmt.Printf("Rejected by TPM limit: %d\n", stats.tpm)
	showMetrics(ctx, limiter, "Final State")
}

// exampleScenarioCombinedLimits demonstrates all three limits working together
func exampleScenarioCombinedLimits(ctx context.Context, client *redis.Client) {
	fmt.Println("Testing with combined limits: 5 active, 10 TPS, 100 TPM...")
	fmt.Println("Scenario runs for 2 minutes to demonstrate all three limit types")

	opts := redis.NewRateLimiterOptions().
		WithMaxActiveTransactions(5).
		WithMaxTransactionsPerSecond(10).
		WithMaxTransactionsPerMinute(100).
		WithWaitOnLimit(false).
		WithTransactionTTL(30 * time.Second).
		WithNamespace("combined_limits").
		WithCacheName("combined_test")

	limiter, err := redis.NewRateLimiter(client, "combined_endpoint", opts)
	if err != nil {
		fmt.Printf("Failed to create rate limiter: %v\n", err)
		return
	}
	defer limiter.Cleanup(ctx)

	var wg sync.WaitGroup
	stats := struct {
		total   int
		success int
		active  int
		tps     int
		tpm     int
		mu      sync.Mutex
	}{}

	startTime := time.Now()
	requestID := 0

	// Phase 1 (0-30s): Fast burst to hit TPS limit
	fmt.Println("\n[Phase 1: 0-30s] Fast bursts to demonstrate TPS limit...")
	for range 3 {
		for range 20 {
			requestID++
			currentID := requestID
			wg.Add(1)
			go func(reqID int) {
				defer wg.Done()
				elapsed := time.Since(startTime)

				stats.mu.Lock()
				stats.total++
				stats.mu.Unlock()

				transactionID, err := limiter.Acquire(ctx)
				if err != nil {
					errMsg := err.Error()
					stats.mu.Lock()
					if contains(errMsg, "active") {
						stats.active++
						fmt.Printf("Request %03d [%3ds]:  Rejected (Active limit)\n", reqID, int(elapsed.Seconds()))
					} else if contains(errMsg, "per second") {
						stats.tps++
						fmt.Printf("Request %03d [%3ds]:  Rejected (TPS limit)\n", reqID, int(elapsed.Seconds()))
					} else if contains(errMsg, "per minute") {
						stats.tpm++
						fmt.Printf("Request %03d [%3ds]:  Rejected (TPM limit)\n", reqID, int(elapsed.Seconds()))
					}
					stats.mu.Unlock()
					return
				}

				stats.mu.Lock()
				stats.success++
				stats.mu.Unlock()
				fmt.Printf("Request %03d [%3ds]:  Accepted\n", reqID, int(elapsed.Seconds()))

				// Simulate work (1 second to build up active transactions)
				time.Sleep(1000 * time.Millisecond)
				limiter.Release(ctx, transactionID)
			}(currentID)
			time.Sleep(50 * time.Millisecond)
		}

		showMetrics(ctx, limiter, fmt.Sprintf("After burst %.0fs", time.Since(startTime).Seconds()))
		time.Sleep(500 * time.Millisecond)
	}

	// Phase 2 (30-60s): Moderate pace with long tasks to hit Active limit
	fmt.Println("\n[Phase 2: 30-60s] Long-running tasks to demonstrate Active limit...")
	for range 40 {
		requestID++
		currentID := requestID
		wg.Add(1)
		go func(reqID int) {
			defer wg.Done()
			elapsed := time.Since(startTime)

			stats.mu.Lock()
			stats.total++
			stats.mu.Unlock()

			transactionID, err := limiter.Acquire(ctx)
			if err != nil {
				errMsg := err.Error()
				stats.mu.Lock()
				if contains(errMsg, "active") {
					stats.active++
					fmt.Printf("Request %03d [%3ds]:  Rejected (Active limit)\n", reqID, int(elapsed.Seconds()))
				} else if contains(errMsg, "per second") {
					stats.tps++
					fmt.Printf("Request %03d [%3ds]:  Rejected (TPS limit)\n", reqID, int(elapsed.Seconds()))
				} else if contains(errMsg, "per minute") {
					stats.tpm++
					fmt.Printf("Request %03d [%3ds]:  Rejected (TPM limit)\n", reqID, int(elapsed.Seconds()))
				}
				stats.mu.Unlock()
				return
			}

			stats.mu.Lock()
			stats.success++
			stats.mu.Unlock()
			fmt.Printf("Request %03d [%3ds]:  Accepted\n", reqID, int(elapsed.Seconds()))

			// Long task (3 seconds)
			time.Sleep(3000 * time.Millisecond)
			limiter.Release(ctx, transactionID)
		}(currentID)
		time.Sleep(200 * time.Millisecond)
	}

	showMetrics(ctx, limiter, "After Phase 2")

	// Phase 3 (60-120s): Continued requests to hit TPM limit
	fmt.Println("\n[Phase 3: 60-120s] Sustained load to demonstrate TPM limit...")
	for range 80 {
		requestID++
		currentID := requestID
		wg.Add(1)
		go func(reqID int) {
			defer wg.Done()
			elapsed := time.Since(startTime)

			stats.mu.Lock()
			stats.total++
			stats.mu.Unlock()

			transactionID, err := limiter.Acquire(ctx)
			if err != nil {
				errMsg := err.Error()
				stats.mu.Lock()
				if contains(errMsg, "active") {
					stats.active++
					fmt.Printf("Request %03d [%3ds]:  Rejected (Active limit)\n", reqID, int(elapsed.Seconds()))
				} else if contains(errMsg, "per second") {
					stats.tps++
					fmt.Printf("Request %03d [%3ds]:  Rejected (TPS limit)\n", reqID, int(elapsed.Seconds()))
				} else if contains(errMsg, "per minute") {
					stats.tpm++
					fmt.Printf("Request %03d [%3ds]:  Rejected (TPM limit)\n", reqID, int(elapsed.Seconds()))
				}
				stats.mu.Unlock()
				return
			}

			stats.mu.Lock()
			stats.success++
			stats.mu.Unlock()
			fmt.Printf("Request %03d [%3ds]:  Accepted\n", reqID, int(elapsed.Seconds()))

			// Short task
			time.Sleep(200 * time.Millisecond)
			limiter.Release(ctx, transactionID)
		}(currentID)
		time.Sleep(500 * time.Millisecond)
	}

	wg.Wait()

	totalElapsed := time.Since(startTime)
	fmt.Printf("\n=== RESULTS SUMMARY ===\n")
	fmt.Printf("Total duration: %v\n", totalElapsed)
	fmt.Printf("Total requests: %d\n", stats.total)
	fmt.Printf("Successful: %d\n", stats.success)
	fmt.Printf("Rejected by Active limit: %d\n", stats.active)
	fmt.Printf("Rejected by TPS limit: %d\n", stats.tps)
	fmt.Printf("Rejected by TPM limit: %d\n", stats.tpm)
	showMetrics(ctx, limiter, "Final State")
}

// exampleScenarioWaitVsImmediate demonstrates wait vs immediate error behavior
func exampleScenarioWaitVsImmediate(ctx context.Context, client *redis.Client) {
	fmt.Println("Comparing WaitOnLimit=false vs WaitOnLimit=true...")

	// Test with immediate error
	fmt.Println("\nTest A: WaitOnLimit=false (immediate error)")
	optsImmediate := redis.NewRateLimiterOptions().
		WithMaxActiveTransactions(2).
		WithWaitOnLimit(false).
		WithNamespace("wait_immediate_a").
		WithCacheName("immediate_test")

	limiterImmediate, err := redis.NewRateLimiter(client, "immediate_endpoint", optsImmediate)
	if err != nil {
		fmt.Printf("Failed to create rate limiter: %v\n", err)
		return
	}
	defer limiterImmediate.Cleanup(ctx)

	var wg1 sync.WaitGroup
	for i := range 5 {
		wg1.Add(1)
		go func(requestID int) {
			defer wg1.Done()

			start := time.Now()
			transactionID, err := limiterImmediate.Acquire(ctx)
			elapsed := time.Since(start)

			if err != nil {
				fmt.Printf("Request %d:  Failed immediately after %v: %v\n", requestID, elapsed, err)
				return
			}

			fmt.Printf("Request %d:  Acquired after %v\n", requestID, elapsed)
			time.Sleep(500 * time.Millisecond)
			limiterImmediate.Release(ctx, transactionID)
		}(i + 1)

		time.Sleep(50 * time.Millisecond)
	}
	wg1.Wait()

	// Wait a bit before next test
	time.Sleep(1 * time.Second)

	// Test with wait
	fmt.Println("\nTest B: WaitOnLimit=true (wait for availability)")
	optsWait := redis.NewRateLimiterOptions().
		WithMaxActiveTransactions(2).
		WithWaitOnLimit(true).
		WithWaitTimeout(5 * time.Second).
		WithRetryDelay(100 * time.Millisecond).
		WithNamespace("wait_immediate_b").
		WithCacheName("wait_test")

	limiterWait, err := redis.NewRateLimiter(client, "wait_endpoint", optsWait)
	if err != nil {
		fmt.Printf("Failed to create rate limiter: %v\n", err)
		return
	}
	defer limiterWait.Cleanup(ctx)

	var wg2 sync.WaitGroup
	for i := range 5 {
		wg2.Add(1)
		go func(requestID int) {
			defer wg2.Done()

			start := time.Now()
			transactionID, err := limiterWait.Acquire(ctx)
			elapsed := time.Since(start)

			if err != nil {
				fmt.Printf("Request %d:  Failed after waiting %v: %v\n", requestID, elapsed, err)
				return
			}

			fmt.Printf("Request %d:  Acquired after waiting %v\n", requestID, elapsed)
			time.Sleep(500 * time.Millisecond)
			limiterWait.Release(ctx, transactionID)
			fmt.Printf("Request %d:  Completed\n", requestID)
		}(i + 1)

		time.Sleep(50 * time.Millisecond)
	}
	wg2.Wait()
}

// exampleScenarioHealthCheckMetrics demonstrates health check and metrics monitoring
func exampleScenarioHealthCheckMetrics(ctx context.Context, client *redis.Client) {
	fmt.Println("Demonstrating health check, metrics monitoring, and cleanup...")

	// Rate Limiter 1: Active Transactions Only
	fmt.Println("\nCreating Rate Limiter 1 (Active Transactions Only)...")
	limiterActive, _ := redis.NewRateLimiter(client, "active_endpoint", redis.NewRateLimiterOptions().
		WithMaxActiveTransactions(3).
		WithNamespace("health_check").
		WithCacheName("active_limiter"))

	// Rate Limiter 2: TPS and TPM Only
	fmt.Println("Creating Rate Limiter 2 (TPS and TPM Only)...")
	limiterTPSTPM, _ := redis.NewRateLimiter(client, "tps_tpm_endpoint", redis.NewRateLimiterOptions().
		WithMaxTransactionsPerSecond(5).
		WithMaxTransactionsPerMinute(30).
		WithNamespace("health_check").
		WithCacheName("tps_tpm_limiter"))

	// Show initial state
	fmt.Println("\n=== INITIAL STATE ===")
	showMetricsRaw(ctx, limiterActive, "Active Limiter")
	showMetricsRaw(ctx, limiterTPSTPM, "TPS/TPM Limiter")

	var wg sync.WaitGroup

	// Simulate load on Active limiter
	fmt.Println("\n=== LOAD ON ACTIVE LIMITER ===")
	fmt.Println("Starting 10 concurrent tasks (limit: 3 active)...")
	for i := range 10 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			txID, err := limiterActive.Acquire(ctx)
			if err == nil {
				fmt.Printf("Active Request %02d: Acquired, processing for 2s...\n", id+1)
				time.Sleep(2 * time.Second)
				limiterActive.Release(ctx, txID)
				fmt.Printf("Active Request %02d: Released\n", id+1)
			} else {
				fmt.Printf("Active Request %02d: Rejected - %v\n", id+1, err)
			}
		}(i)
		time.Sleep(100 * time.Millisecond)
	}

	// Show metrics during active load
	time.Sleep(300 * time.Millisecond)
	fmt.Println("\n=== METRICS DURING ACTIVE LOAD ===")
	showMetricsRaw(ctx, limiterActive, "Active Limiter")

	// Simulate load on TPS/TPM limiter
	fmt.Println("\n=== LOAD ON TPS/TPM LIMITER ===")
	fmt.Println("Sending 20 requests rapidly (limits: 5 TPS, 30 TPM)...")
	for i := range 20 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := limiterTPSTPM.Acquire(ctx)
			if err == nil {
				fmt.Printf("TPS/TPM Request %02d: Accepted\n", id+1)
			} else {
				fmt.Printf("TPS/TPM Request %02d: Rejected - %v\n", id+1, err)
			}
		}(i)
		time.Sleep(100 * time.Millisecond)
	}

	// Show metrics during TPS/TPM load
	time.Sleep(300 * time.Millisecond)
	fmt.Println("\n=== METRICS DURING TPS/TPM LOAD ===")
	showMetricsRaw(ctx, limiterTPSTPM, "TPS/TPM Limiter")

	wg.Wait()

	// Show final metrics before cleanup
	fmt.Println("\n=== FINAL METRICS BEFORE CLEANUP ===")
	showMetricsRaw(ctx, limiterActive, "Active Limiter")
	showMetricsRaw(ctx, limiterTPSTPM, "TPS/TPM Limiter")

	// Show global metrics
	fmt.Println("\n=== GLOBAL RATE LIMITER HEALTH CHECK ===")
	allMetrics := redis.GetRateLimiterMetrics(ctx)
	for name, metrics := range allMetrics {
		fmt.Printf("\nRate Limiter: %s\n", name)
		fmt.Printf("  Metrics: %v\n", metrics)
	}

	// Cleanup rate limiters
	fmt.Println("\n=== CLEANING UP RATE LIMITERS ===")
	fmt.Println("Cleaning up Active Limiter...")
	limiterActive.Cleanup(ctx)
	showMetricsRaw(ctx, limiterActive, "Active Limiter (after cleanup)")

	fmt.Println("\nCleaning up TPS/TPM Limiter...")
	limiterTPSTPM.Cleanup(ctx)
	showMetricsRaw(ctx, limiterTPSTPM, "TPS/TPM Limiter (after cleanup)")

	// Show global metrics after cleanup
	fmt.Println("\n=== GLOBAL METRICS AFTER CLEANUP ===")
	allMetricsAfter := redis.GetRateLimiterMetrics(ctx)
	if len(allMetricsAfter) == 0 {
		fmt.Println("All rate limiters cleaned up successfully - no active limiters registered")
	} else {
		fmt.Printf("Remaining rate limiters: %d\n", len(allMetricsAfter))
		for name := range allMetricsAfter {
			fmt.Printf("  - %s\n", name)
		}
	}
}

// showMetrics displays the current metrics of a rate limiter
func showMetrics(ctx context.Context, limiter *redis.RateLimiter, label string) {
	metrics, err := limiter.GetMetrics(ctx)
	if err != nil {
		fmt.Printf("[%s] Failed to get metrics: %v\n", label, err)
		return
	}

	fmt.Printf("[%s] Metrics:\n", label)
	printMetrics(metrics)
}

// showMetricsRaw displays the raw metrics map
func showMetricsRaw(ctx context.Context, limiter *redis.RateLimiter, label string) {
	metrics, err := limiter.GetMetrics(ctx)
	if err != nil {
		fmt.Printf("[%s] Failed to get metrics: %v\n", label, err)
		return
	}

	fmt.Printf("[%s] Metrics: %v\n", label, metrics)
}

// printMetrics prints rate limiter metrics in a formatted way
func printMetrics(metrics redis.RateLimiterMetrics) {
	if active, ok := metrics["active_transactions"]; ok {
		fmt.Printf("  Active: %s/%s (%s utilization)\n",
			active,
			metrics["max_active_transactions"],
			metrics["active_utilization"])
	}

	if tps, ok := metrics["transactions_per_second"]; ok {
		fmt.Printf("  TPS: %s/%s (%s utilization)\n",
			tps,
			metrics["max_transactions_per_second"],
			metrics["tps_utilization"])
	}

	if tpm, ok := metrics["transactions_per_minute"]; ok {
		fmt.Printf("  TPM: %s/%s (%s utilization)\n",
			tpm,
			metrics["max_transactions_per_minute"],
			metrics["tpm_utilization"])
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

// containsSubstring checks if a string contains a substring
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
