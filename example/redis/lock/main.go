package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"go-api/pkg/redis"

	"github.com/go-co-op/gocron/v2"
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

	fmt.Println("Example Redis Distributed Lock Scenarios...")

	// Example Scenario: Single attempt lock acquisition
	fmt.Println("\nExample Scenario: Single attempt lock acquisition...")
	exampleScenarioSimpleAttemptLockWithNoRetries(ctx, client)

	// Example Scenario: Lock with manual refresh during processing and retries to acquire lock
	// One instance will get the lock and execute and one will fail and stop
	fmt.Println("\nExample Scenario: Lock with manual refresh during processing...")
	exampleScenarioManualRefreshAndRetriesToAcquire(ctx, client)

	// Example Scenario: Persistent lock with automatic refresh
	// Three instances will start, one will get the lock, processing a list manual refreshing the lock
	// One will get the lock after the first concludes via retries mechanism
	// One will fail, because get exhaust retries before the two firsts conclude the work
	fmt.Println("\nExample Scenario: Persistent lock with automatic refresh...")
	exampleScenarioAutoRefreshLongTaskProcessing(ctx, client)

	// Example failure scenario - instance dies while holding lock
	fmt.Println("\nExample failure scenario - instance dies while holding lock...")
	testFailureScenario(ctx, client)

	// Example panic failure scenario - instance panics while holding lock
	fmt.Println("\nExample panic failure scenario - instance panics while holding lock...")
	testPanicFailureScenario(ctx, client)

	// Example Scenario: Cron job with distributed lock
	fmt.Println("\nExample Scenario: Cron job with distributed lock...")
	exampleScenarioCronJobWithDistributedLock(ctx, client)

	// Show final health check
	fmt.Println("\nFinal health check after all scenarios:")
	testHealthCheck(client)

	fmt.Println("\n All distributed lock scenarios completed successfully!")
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

// exampleScenarioSimpleAttemptLockWithNoRetries demonstrates single attempt lock acquisition with immediate failure
func exampleScenarioSimpleAttemptLockWithNoRetries(ctx context.Context, client *redis.Client) {
	var wg sync.WaitGroup

	// Both instances run the same code
	for i := 1; i <= 2; i++ {
		wg.Add(1)
		go func(instanceID int) {
			defer wg.Done()

			// Same configuration for all instances
			lock := redis.NewSingleAttemptLock(client, "scenario1_lock", 10*time.Second, "scenario1")

			// Add small delay to simulate different startup times
			time.Sleep(time.Duration(instanceID-1) * 100 * time.Millisecond)

			err := lock.Lock(ctx)
			if err != nil {
				fmt.Printf("Instance %d:  Failed to acquire lock as expected: %v\n", instanceID, err)
				return
			}

			fmt.Printf("Instance %d:  Acquired lock successfully\n", instanceID)

			// Simulate work
			time.Sleep(1 * time.Second)

			err = lock.Unlock(ctx)
			if err != nil {
				fmt.Printf("Instance %d: Failed to release lock: %v\n", instanceID, err)
			} else {
				fmt.Printf("Instance %d:  Released lock successfully\n", instanceID)
			}
		}(i)
	}

	wg.Wait()
}

// exampleScenarioManualRefreshAndRetriesToAcquire demonstrates lock with manual refresh during processing
func exampleScenarioManualRefreshAndRetriesToAcquire(ctx context.Context, client *redis.Client) {
	var wg sync.WaitGroup

	// Both instances run the same code
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go func(instanceID int) {
			defer wg.Done()

			// Same configuration for all instances
			lock := redis.NewRetryLock(client, "scenario2_lock", 5*time.Second, 1*time.Second, 4, "scenario2")

			// Add small delay to simulate different startup times
			time.Sleep(time.Duration(instanceID-1) * 500 * time.Millisecond)

			err := lock.Lock(ctx)
			if err != nil {
				fmt.Printf("Instance %d:  Failed to acquire lock after retries as expected: %v\n", instanceID, err)
				return
			}

			fmt.Printf("Instance %d:  Acquired lock successfully\n", instanceID)

			// Simulate processing a list with manual refresh
			for j := 1; j <= 3; j++ {
				fmt.Printf("Instance %d: Processing item %d/3\n", instanceID, j)
				time.Sleep(1 * time.Second)

				// Manual refresh after each item
				err = lock.Refresh(ctx)
				if err != nil {
					fmt.Printf("Instance %d: Failed to refresh lock: %v\n", instanceID, err)
				} else {
					fmt.Printf("Instance %d:  Refreshed lock after item %d\n", instanceID, j)
				}
			}

			err = lock.Unlock(ctx)
			if err != nil {
				fmt.Printf("Instance %d: Failed to release lock: %v\n", instanceID, err)
			} else {
				fmt.Printf("Instance %d:  Released lock successfully\n", instanceID)
			}
		}(i)
	}

	wg.Wait()
}

// exampleScenarioAutoRefreshLongTaskProcessing demonstrates persistent lock with automatic refresh
func exampleScenarioAutoRefreshLongTaskProcessing(ctx context.Context, client *redis.Client) {
	var wg sync.WaitGroup

	// Both instances run the same code
	for i := 1; i <= 2; i++ {
		wg.Add(1)
		go func(instanceID int) {
			defer wg.Done()

			// Same configuration for all instances
			lock := redis.NewPersistentLock(client, "scenario3_lock", 5*time.Second, 2*time.Second, "scenario3")

			// Add small delay to simulate different startup times
			time.Sleep(time.Duration(instanceID-1) * 1 * time.Second)

			fmt.Printf("Instance %d: Attempting to acquire persistent lock...\n", instanceID)
			err := lock.Lock(ctx)
			if err != nil {
				fmt.Printf("Instance %d: Failed to acquire persistent lock: %v\n", instanceID, err)
				return
			}

			fmt.Printf("Instance %d:  Acquired persistent lock successfully\n", instanceID)

			// Start auto-refresh with original context (no timeout)
			// Auto-refresh will continue until Unlock() is called or context is cancelled
			refreshErrChan := lock.AutoRefresh(ctx)

			// Simulate long-running work
			fmt.Printf("Instance %d: Starting long-running work...\n", instanceID)

			// Show health check while lock is active (only for first instance)
			if instanceID == 1 {
				showHealthCheck(client)
			}

			time.Sleep(10 * time.Second)
			fmt.Printf("Instance %d: Long-running work completed\n", instanceID)

			// Release lock
			err = lock.Unlock(ctx)
			if err != nil {
				fmt.Printf("Instance %d: Failed to release persistent lock: %v\n", instanceID, err)
			} else {
				fmt.Printf("Instance %d:  Released persistent lock successfully\n", instanceID)
			}

			// Wait for refresh to complete
			err = <-refreshErrChan
			if err != nil {
				fmt.Printf("Instance %d: Auto-refresh error: %v\n", instanceID, err)
			} else {
				fmt.Printf("Instance %d:  Auto-refresh completed\n", instanceID)
			}
		}(i)
	}

	wg.Wait()
}

// showHealthCheck shows a compact health check focused on lock status
func showHealthCheck(client *redis.Client) {
	healthChecker := redis.NewHealthChecker(client.GetClient(), client.GetConfig())
	healthCheck := healthChecker.HealthCheck()

	fmt.Printf("Redis Status: %s | ", healthCheck.Status)

	if len(healthCheck.LockStatus) == 0 {
		fmt.Println("No locks registered")
	} else {
		fmt.Print("Locks: ")
		first := true
		for cacheName, isAcquired := range healthCheck.LockStatus {
			if !first {
				fmt.Print(", ")
			}
			status := "NOT_ACQUIRED"
			if isAcquired {
				status = "ACQUIRED"
			}
			fmt.Printf("%s[%s]", cacheName, status)
			first = false
		}
		fmt.Println()
	}
}

// testFailureScenario demonstrates what happens when ALL instances die while holding a lock
func testFailureScenario(ctx context.Context, client *redis.Client) {

	var wg sync.WaitGroup

	// Both instances run the same code
	for i := 1; i <= 2; i++ {
		wg.Add(1)
		go func(instanceID int) {
			defer wg.Done()

			// Same configuration for all instances
			lock := redis.NewPersistentLock(client, "critical_task", 10*time.Second, 2*time.Second, "critical")

			// Add small delay to simulate different startup times
			time.Sleep(time.Duration(instanceID-1) * 1 * time.Second)

			fmt.Printf("Instance %d: Attempting to acquire critical task lock...\n", instanceID)
			err := lock.Lock(ctx)
			if err != nil {
				fmt.Printf("Instance %d: Failed to acquire lock: %v\n", instanceID, err)
				return
			}

			fmt.Printf("Instance %d:  Acquired critical task lock successfully\n", instanceID)

			// Create a cancellable context for auto-refresh
			refreshCtx, cancelRefresh := context.WithCancel(ctx)
			defer cancelRefresh()

			// Start auto-refresh with cancellable context
			// Auto-refresh will continue until context is cancelled or Unlock() is called
			refreshErrChan := lock.AutoRefresh(refreshCtx)

			// Show health check while lock is active (only for first instance)
			if instanceID == 1 {
				showHealthCheck(client)
			}

			// Simulate critical work
			fmt.Printf("Instance %d: Starting critical task...\n", instanceID)
			time.Sleep(3 * time.Second)

			// SIMULATE TOTAL SYSTEM FAILURE - ALL instances die
			fmt.Printf("Instance %d:  SIMULATING INSTANCE %d DEATH (no unlock, no refresh)\n", instanceID, instanceID)

			// Cancel the refresh context to simulate instance death
			// This will stop the auto-refresh goroutine immediately
			cancelRefresh()

			time.Sleep(2 * time.Second)
			fmt.Printf("Instance %d: Instance died - lock will expire in 10 seconds due to no refresh\n", instanceID)

			// Wait for refresh goroutine to stop (this simulates the death)
			err = <-refreshErrChan
			if err != nil {
				fmt.Printf("Instance %d: Auto-refresh stopped due to death: %v\n", instanceID, err)
			} else {
				fmt.Printf("Instance %d: Auto-refresh stopped due to death\n", instanceID)
			}

			// NO UNLOCK CALL - This simulates complete instance failure
			fmt.Printf("Instance %d:  Instance completely dead - no cleanup performed\n", instanceID)
		}(i)
	}

	wg.Wait()
	fmt.Println("Total system failure scenario completed - All instances died without proper cleanup")
	fmt.Println("Lock will expire in Redis after 5 seconds due to no refresh...")

	// Wait for lock to expire and show that it's available again
	fmt.Println("\nWaiting for lock to expire in Redis...")
	time.Sleep(6 * time.Second)

	showHealthCheck(client)

	// Demonstrate that a new instance can now acquire the lock
	fmt.Println("\nDemonstrating that a new instance can acquire the expired lock...")
	testLockAcquisitionAfterFailure(ctx, client)
}

// testLockAcquisitionAfterFailure demonstrates that a new instance can acquire the lock after all previous instances failed
func testLockAcquisitionAfterFailure(ctx context.Context, client *redis.Client) {
	lock := redis.NewPersistentLock(client, "critical_task", 5*time.Second, 2*time.Second, "critical")

	fmt.Println("New Instance: Attempting to acquire critical task lock after system failure...")
	err := lock.Lock(ctx)
	if err != nil {
		fmt.Printf("New Instance: Failed to acquire lock: %v\n", err)
		return
	}

	fmt.Println("New Instance:  Successfully acquired lock after system failure recovery!")

	// Start auto-refresh
	refreshErrChan := lock.AutoRefresh(ctx)

	// Show health check
	showHealthCheck(client)

	// Simulate completing the critical task
	fmt.Println("New Instance: Completing the critical task that was interrupted by system failure...")
	time.Sleep(3 * time.Second)

	// Release lock properly
	err = lock.Unlock(ctx)
	if err != nil {
		fmt.Printf("New Instance: Failed to release lock: %v\n", err)
	} else {
		fmt.Println("New Instance:  Released critical task lock successfully")
	}

	// Wait for auto-refresh to complete
	err = <-refreshErrChan
	if err != nil {
		fmt.Printf("New Instance: Auto-refresh error: %v\n", err)
	} else {
		fmt.Println("New Instance:  Auto-refresh completed")
	}

	fmt.Println("System recovery scenario completed - New instance successfully took over after total system failure")
}

// testPanicFailureScenario demonstrates what happens when an instance panics while holding a lock
func testPanicFailureScenario(ctx context.Context, client *redis.Client) {
	fmt.Println("Simulating panic failure scenario - instance panics while holding lock...")

	var wg sync.WaitGroup

	// Both instances run the same code
	for i := 1; i <= 2; i++ {
		wg.Add(1)
		go func(instanceID int) {
			// Create a cancellable context for this instance
			instanceCtx, cancelInstance := context.WithCancel(ctx)
			defer func() {
				// Recover from panic and automatically cancel the context
				if r := recover(); r != nil {
					fmt.Printf("Instance %d:  PANIC RECOVERED: %v\n", instanceID, r)
					fmt.Printf("Instance %d:  Automatically canceling context due to panic...\n", instanceID)
					// Cancel the context to stop auto-refresh and other operations
					cancelInstance()
				}
				wg.Done()
			}()

			// Same configuration for all instances
			lock := redis.NewPersistentLock(client, "panic_task", 10*time.Second, 2*time.Second, "panic")

			// Add small delay to simulate different startup times
			time.Sleep(time.Duration(instanceID-1) * 1 * time.Second)

			fmt.Printf("Instance %d: Attempting to acquire panic task lock...\n", instanceID)
			err := lock.Lock(instanceCtx)
			if err != nil {
				fmt.Printf("Instance %d: Failed to acquire lock: %v\n", instanceID, err)
				return
			}

			fmt.Printf("Instance %d:  Acquired panic task lock successfully\n", instanceID)

			// Start auto-refresh with the instance-specific context
			lock.AutoRefresh(instanceCtx)

			// Show health check while lock is active (only for first instance)
			if instanceID == 1 {
				showHealthCheck(client)
			}

			// Simulate some work before panic
			fmt.Printf("Instance %d: Starting critical work before panic...\n", instanceID)
			time.Sleep(5 * time.Second)

			// SIMULATE PANIC - This will cause the instance to crash
			fmt.Printf("Instance %d:  SIMULATING FATAL PANIC - Instance will crash!\n", instanceID)
			panic("FATAL ERROR: Critical system failure - instance must die immediately!")
		}(i)
	}

	wg.Wait()
	fmt.Println("Panic failure scenario completed - Instance 1 panicked and died, Instance 2 continued normally")

	// Wait for lock to expire and show that it's available again
	fmt.Println("\nWaiting for lock to expire in Redis after panic...")
	time.Sleep(3 * time.Second)

	showHealthCheck(client)

	// Demonstrate that a new instance can now acquire the lock after panic
	fmt.Println("\nDemonstrating that a new instance can acquire the lock after panic failure...")
	testLockAcquisitionAfterPanic(ctx, client)
}

// testLockAcquisitionAfterPanic demonstrates that a new instance can acquire the lock after panic failure
func testLockAcquisitionAfterPanic(ctx context.Context, client *redis.Client) {
	lock := redis.NewPersistentLock(client, "panic_task", 5*time.Second, 2*time.Second, "panic")

	fmt.Println("New Instance: Attempting to acquire panic task lock after panic failure...")
	err := lock.Lock(ctx)
	if err != nil {
		fmt.Printf("New Instance: Failed to acquire lock: %v\n", err)
		return
	}

	fmt.Println("New Instance:  Successfully acquired lock after panic failure recovery!")

	// Start auto-refresh
	refreshErrChan := lock.AutoRefresh(ctx)

	// Show health check
	showHealthCheck(client)

	// Simulate completing the task that was interrupted by panic
	fmt.Println("New Instance: Completing the task that was interrupted by panic...")
	time.Sleep(2 * time.Second)

	// Release lock properly
	err = lock.Unlock(ctx)
	if err != nil {
		fmt.Printf("New Instance: Failed to release lock: %v\n", err)
	} else {
		fmt.Println("New Instance:  Released panic task lock successfully")
	}

	// Wait for auto-refresh to complete
	err = <-refreshErrChan
	if err != nil {
		fmt.Printf("New Instance: Auto-refresh error: %v\n", err)
	} else {
		fmt.Println("New Instance:  Auto-refresh completed")
	}

	fmt.Println("Panic recovery scenario completed - New instance successfully took over after panic failure")
}

// exampleScenarioCronJobWithDistributedLock demonstrates cron job scheduling with distributed lock
func exampleScenarioCronJobWithDistributedLock(ctx context.Context, client *redis.Client) {
	fmt.Println("Simulating 2 instances running cron jobs with distributed lock...")

	var wg sync.WaitGroup

	// Both instances run the same code
	for i := 1; i <= 2; i++ {
		wg.Add(1)
		go func(instanceID int) {
			defer wg.Done()

			// Create a cancellable context for this instance
			instanceCtx, cancelInstance := context.WithCancel(ctx)
			defer func() {
				// Recover from panic and automatically cancel the context
				if r := recover(); r != nil {
					fmt.Printf("Instance %d:  PANIC RECOVERED: %v\n", instanceID, r)
					fmt.Printf("Instance %d:  Automatically canceling context due to panic...\n", instanceID)
				}
				cancelInstance()
			}()

			// Same configuration for all instances
			lock := redis.NewScheduledTaskLock(client, "cron_job_lock", 10*time.Second, 2*time.Second, "cron_scheduler")

			fmt.Printf("Instance %d: Attempting to acquire cron job lock...\n", instanceID)
			err := lock.Lock(instanceCtx)
			if err != nil {
				fmt.Printf("Instance %d: Failed to acquire cron job lock: %v\n", instanceID, err)
				return
			}

			fmt.Printf("Instance %d:  Acquired cron job lock - Starting cron scheduler...\n", instanceID)

			// Start auto-refresh for cron scheduler (keeps lock indefinitely)
			refreshErrChan := lock.AutoRefresh(instanceCtx)

			// Initialize and start cron scheduler (non-blocking)
			fmt.Printf("Instance %d: Initializing cron scheduler...\n", instanceID)
			cronScheduler := NewCronJobScheduler(instanceID, client)

			// Start the scheduler (non-blocking)
			err = cronScheduler.Start(instanceCtx)
			if err != nil {
				fmt.Printf("Instance %d: Failed to start cron scheduler: %v\n", instanceID, err)
				// Release lock on error
				err := lock.Unlock(instanceCtx)
				if err != nil {
					return
				}
				return
			}

			fmt.Printf("Instance %d: Cron scheduler started successfully (will run for 15 seconds)...\n", instanceID)

			// Simulate context cancellation on error
			// In real code do not do this
			go func() {
				time.Sleep(15 * time.Second)
				fmt.Printf("Instance %d: Timeout reached, stopping cron scheduler...\n", instanceID)
				cancelInstance()
			}()

			// Wait for context cancellation
			select {
			case <-instanceCtx.Done():
				fmt.Printf("Instance %d: Context cancelled, stopping cron scheduler...\n", instanceID)
			case err = <-refreshErrChan:
				if err != nil {
					fmt.Printf("Instance %d: Auto-refresh error: %v\n", instanceID, err)
				} else {
					fmt.Printf("Instance %d:  Auto-refresh completed\n", instanceID)
				}
			}

			// Stop the scheduler gracefully
			cronScheduler.Stop()
			fmt.Printf("Instance %d: Cron scheduler stopped, releasing lock...\n", instanceID)
			// Release lock
			err = lock.Unlock(instanceCtx)
			if err != nil {
				fmt.Printf("Instance %d: Failed to release cron job lock: %v\n", instanceID, err)
			} else {
				fmt.Printf("Instance %d:  Released cron job lock successfully\n", instanceID)
			}
		}(i)
	}

	wg.Wait()
	fmt.Println("Cron job with distributed lock scenario completed - Only one instance ran the cron jobs")
}

// CronJobScheduler simulates a real cron job scheduler using go-co-op/gocron/v2
type CronJobScheduler struct {
	instanceID int
	scheduler  gocron.Scheduler
	stopChan   chan struct{}
	client     *redis.Client
	running    bool
}

// NewCronJobScheduler creates a new cron job scheduler
func NewCronJobScheduler(instanceID int, client *redis.Client) *CronJobScheduler {
	scheduler, _ := gocron.NewScheduler()
	return &CronJobScheduler{
		instanceID: instanceID,
		scheduler:  scheduler,
		stopChan:   make(chan struct{}),
		client:     client,
		running:    false,
	}
}

// Start initializes and starts the cron job scheduler in a non-blocking way
func (c *CronJobScheduler) Start(ctx context.Context) error {
	if c.running {
		return fmt.Errorf("cron scheduler %d is already running", c.instanceID)
	}

	fmt.Printf("Cron Scheduler %d: Initializing cron jobs...\n", c.instanceID)

	// Schedule different cron jobs with various expressions
	// Every 3 seconds using CronJob with 6-field format (seconds included)
	_, err := c.scheduler.NewJob(
		gocron.CronJob("*/3 * * * * *", true), // Every 3 seconds
		gocron.NewTask(func(ctx context.Context) {
			fmt.Printf("Cron Scheduler %d: [Every 3s] Executing data cleanup job\n", c.instanceID)
			c.simulateJobExecution("data_cleanup", 500*time.Millisecond)
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to schedule data cleanup job: %w", err)
	}

	// Every 5 seconds using CronJob with 6-field format (seconds included)
	_, err = c.scheduler.NewJob(
		gocron.CronJob("*/5 * * * * *", true), // Every 5 seconds
		gocron.NewTask(func(ctx context.Context) {
			fmt.Printf("Cron Scheduler %d: [Every 5s] Executing report generation job\n", c.instanceID)
			c.simulateJobExecution("report_generation", 800*time.Millisecond)
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to schedule report generation job: %w", err)
	}

	// Start the scheduler
	c.scheduler.Start()
	c.running = true

	fmt.Printf("Cron Scheduler %d: Started successfully\n", c.instanceID)

	// Start the monitoring goroutine
	go c.monitor(ctx)

	return nil
}

// monitor monitors the context and stop channel for shutdown signals
func (c *CronJobScheduler) monitor(ctx context.Context) {
	select {
	case <-ctx.Done():
		fmt.Printf("Cron Scheduler %d: Context cancelled, stopping cron jobs\n", c.instanceID)
		c.stop()
	case <-c.stopChan:
		fmt.Printf("Cron Scheduler %d: Stop signal received, stopping cron jobs\n", c.instanceID)
		c.stop()
	}
}

// stop stops the cron job scheduler
func (c *CronJobScheduler) stop() {
	if !c.running {
		return
	}

	err := c.scheduler.Shutdown()
	if err != nil {
		c.running = false
		return
	}
	c.running = false
	fmt.Printf("Cron Scheduler %d: All cron jobs stopped\n", c.instanceID)
}

// Stop stops the cron job scheduler gracefully
func (c *CronJobScheduler) Stop() {
	select {
	case <-c.stopChan:
		// Already closed
	default:
		close(c.stopChan)
	}
}

// IsRunning returns whether the scheduler is currently running
func (c *CronJobScheduler) IsRunning() bool {
	return c.running
}

// simulateJobExecution simulates job execution with distributed lock protection
func (c *CronJobScheduler) simulateJobExecution(jobName string, duration time.Duration) {
	// Create a lock for this specific job to prevent concurrent execution
	lock := redis.NewSingleAttemptLock(c.client, fmt.Sprintf("job_%s", jobName), 30*time.Second, "cron_jobs")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to acquire lock for this job
	err := lock.Lock(ctx)
	if err != nil {
		fmt.Printf("Cron Scheduler %d: Job '%s' skipped - another instance is running it: %v\n", c.instanceID, jobName, err)
		return
	}

	// Simulate job work
	time.Sleep(duration)
	fmt.Printf("Cron Scheduler %d: Job '%s' completed successfully\n", c.instanceID, jobName)

	// Release lock
	err = lock.Unlock(ctx)
	if err != nil {
		fmt.Printf("Cron Scheduler %d: Failed to release job lock for '%s': %v\n", c.instanceID, jobName, err)
	}
}

// testHealthCheck demonstrates health check with lock status
func testHealthCheck(client *redis.Client) {
	healthChecker := redis.NewHealthChecker(client.GetClient(), client.GetConfig())
	healthCheck := healthChecker.HealthCheck()

	fmt.Printf("=== HEALTH CHECK DETAILS ===\n")
	fmt.Printf("Status: %s\n", healthCheck.Status)

	fmt.Printf("\n--- Redis Connection Details ---\n")
	for key, value := range healthCheck.Details {
		fmt.Printf("%s: %s\n", key, value)
	}

	fmt.Printf("\n--- Lock Status ---\n")
	if len(healthCheck.LockStatus) == 0 {
		fmt.Println("No locks currently registered")
	} else {
		for cacheName, isAcquired := range healthCheck.LockStatus {
			status := "NOT ACQUIRED"
			if isAcquired {
				status = "ACQUIRED"
			}
			fmt.Printf("Lock '%s': %s\n", cacheName, status)
		}
	}
}
