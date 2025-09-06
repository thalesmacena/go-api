package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go-api/pkg/redis"
)

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

	ctx := context.Background()

	fmt.Println("Testing Redis Distributed Lock functionality...")

	// Create lock options using builder
	lockOptions := redis.NewLockOptions().
		WithTTL(30 * time.Second).
		WithRetryDelay(100 * time.Millisecond).
		WithMaxRetries(10).
		WithRefreshInterval(10 * time.Second).
		WithLockNamespace("user_service")

	// Test 1: Basic lock acquisition and release
	fmt.Println("\n1. Testing basic lock acquisition and release...")
	lock := redis.NewLock(client, "test_lock", lockOptions)
	fmt.Println("  - Lock key format: user_service::test_lock")

	err := lock.Lock(ctx)
	if err != nil {
		log.Fatalf("Failed to acquire lock: %v", err)
	}
	fmt.Println("✓ Acquired lock successfully")

	// Simulate some work
	time.Sleep(100 * time.Millisecond)

	err = lock.Unlock(ctx)
	if err != nil {
		log.Fatalf("Failed to release lock: %v", err)
	}
	fmt.Println("✓ Released lock successfully")

	// Test 2: Lock with custom options
	fmt.Println("\n2. Testing lock with custom options...")
	customLockOptions := redis.NewLockOptions().
		WithTTL(30 * time.Second).
		WithRetryDelay(100 * time.Millisecond).
		WithMaxRetries(5).
		WithRefreshInterval(10 * time.Second).
		WithLockNamespace("payment_service")

	customLock := redis.NewLock(client, "custom_lock", customLockOptions)
	fmt.Println("  - Lock key format: payment_service::custom_lock")

	err = customLock.Lock(ctx)
	if err != nil {
		log.Fatalf("Failed to acquire custom lock: %v", err)
	}
	fmt.Println("✓ Acquired custom lock successfully")

	err = customLock.Unlock(ctx)
	if err != nil {
		log.Fatalf("Failed to release custom lock: %v", err)
	}
	fmt.Println("✓ Released custom lock successfully")

	// Test 3: Lock with function execution
	fmt.Println("\n3. Testing lock with function execution...")
	var counter int
	err = redis.LockWithFunc(ctx, client, "func_lock", nil, func() error {
		fmt.Println("✓ Executing critical section with lock")
		counter++
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to execute function with lock: %v", err)
	}
	fmt.Printf("✓ Function executed successfully, counter: %d\n", counter)

	// Test 4: Lock with timeout
	fmt.Println("\n4. Testing lock with timeout...")
	err = redis.LockWithTimeout(ctx, client, "timeout_lock", nil, 5*time.Second, func() error {
		fmt.Println("✓ Executing function with timeout lock")
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to execute function with timeout lock: %v", err)
	}
	fmt.Println("✓ Function executed successfully with timeout")

	// Test 5: Concurrent lock attempts
	fmt.Println("\n5. Testing concurrent lock attempts...")
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			lock := redis.NewLock(client, "concurrent_lock", nil)
			err := lock.Lock(ctx)
			if err != nil {
				fmt.Printf("✗ Goroutine %d failed to acquire lock: %v\n", id, err)
				return
			}

			mu.Lock()
			successCount++
			mu.Unlock()

			fmt.Printf("✓ Goroutine %d acquired lock\n", id)

			// Simulate work
			time.Sleep(50 * time.Millisecond)

			err = lock.Unlock(ctx)
			if err != nil {
				fmt.Printf("✗ Goroutine %d failed to release lock: %v\n", id, err)
			} else {
				fmt.Printf("✓ Goroutine %d released lock\n", id)
			}
		}(i)
	}

	wg.Wait()
	fmt.Printf("✓ Concurrent test completed, %d successful acquisitions\n", successCount)

	// Test 6: Lock refresh
	fmt.Println("\n6. Testing lock refresh...")
	refreshLock := redis.NewLock(client, "refresh_lock", &redis.LockOptions{
		TTL:             5 * time.Second,
		RefreshInterval: 2 * time.Second,
	})

	err = refreshLock.Lock(ctx)
	if err != nil {
		log.Fatalf("Failed to acquire refresh lock: %v", err)
	}
	fmt.Println("✓ Acquired refresh lock")

	// Start auto-refresh
	refreshCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	refreshErrChan := refreshLock.AutoRefresh(refreshCtx)

	// Simulate long-running work
	go func() {
		time.Sleep(6 * time.Second)
		err := refreshLock.Unlock(ctx)
		if err != nil {
			fmt.Printf("✗ Failed to release refresh lock: %v\n", err)
		} else {
			fmt.Println("✓ Released refresh lock")
		}
	}()

	// Wait for refresh errors or timeout
	select {
	case err := <-refreshErrChan:
		if err != nil {
			fmt.Printf("✗ Auto-refresh error: %v\n", err)
		}
	case <-refreshCtx.Done():
		fmt.Println("✓ Auto-refresh completed (timeout)")
	}

	// Test 7: Lock health check
	fmt.Println("\n7. Testing lock health check...")
	healthLock := redis.NewLock(client, "health_lock", nil)

	err = healthLock.Lock(ctx)
	if err != nil {
		log.Fatalf("Failed to acquire health lock: %v", err)
	}

	isLocked, err := healthLock.IsLocked(ctx)
	if err != nil {
		log.Fatalf("Failed to check lock status: %v", err)
	}
	fmt.Printf("✓ Lock status check: %t\n", isLocked)

	err = healthLock.Unlock(ctx)
	if err != nil {
		log.Fatalf("Failed to release health lock: %v", err)
	}

	isLocked, err = healthLock.IsLocked(ctx)
	if err != nil {
		log.Fatalf("Failed to check lock status after release: %v", err)
	}
	fmt.Printf("✓ Lock status after release: %t\n", isLocked)

	fmt.Println("\n✓ All distributed lock tests completed successfully!")
}
