package redis

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisHealthCheck represents the health check response for Redis
type RedisHealthCheck struct {
	Status     HealthStatus      `json:"status"`
	Details    map[string]string `json:"details"`
	LockStatus map[string]bool   `json:"lock_status,omitempty"`
}

// HealthChecker provides Redis health checking functionality
type HealthChecker struct {
	client        *redis.Client
	config        *Config
	lastCheck     time.Time
	checkInterval time.Duration
	isHealthy     int32 // atomic flag to track health status
	lastError     string
	mu            chan struct{} // mutex for thread-safe operations
}

// NewHealthChecker creates a new Redis health checker
func NewHealthChecker(client *redis.Client, config *Config) *HealthChecker {
	return &HealthChecker{
		client:        client,
		config:        config,
		checkInterval: 30 * time.Second,
		mu:            make(chan struct{}, 1),
	}
}

// HealthCheck performs a comprehensive health check on the Redis connection
func (h *HealthChecker) HealthCheck() RedisHealthCheck {
	h.mu <- struct{}{}        // acquire lock
	defer func() { <-h.mu }() // release lock

	// Test basic connectivity
	pingResult := h.testPing()

	// Test basic operations
	operationResult := h.testBasicOperations()

	// Test connection pool
	poolResult := h.testConnectionPool()

	// Test memory usage
	memoryResult := h.testMemoryUsage()

	// Determine overall status
	var status HealthStatus
	if pingResult && operationResult && poolResult {
		status = StatusUp
		atomic.StoreInt32(&h.isHealthy, 1)
		h.lastError = ""
	} else {
		status = StatusDown
		atomic.StoreInt32(&h.isHealthy, 0)
	}

	h.lastCheck = time.Now()

	details := map[string]string{
		"host":                  h.config.Host,
		"port":                  strconv.Itoa(h.config.Port),
		"database":              strconv.Itoa(h.config.Database),
		"min_idle_conns":        strconv.Itoa(h.config.MinIdleConns),
		"max_retries":           strconv.Itoa(h.config.MaxRetries),
		"dial_timeout":          h.config.DialTimeout.String(),
		"read_timeout":          h.config.ReadTimeout.String(),
		"write_timeout":         h.config.WriteTimeout.String(),
		"pool_timeout":          h.config.PoolTimeout.String(),
		"ping_successful":       strconv.FormatBool(pingResult),
		"operations_successful": strconv.FormatBool(operationResult),
		"pool_healthy":          strconv.FormatBool(poolResult),
		"memory_accessible":     strconv.FormatBool(memoryResult),
		"last_check":            h.lastCheck.Format(time.RFC3339),
		"check_interval":        h.checkInterval.String(),
		"last_error":            h.lastError,
	}

	// Get lock status without affecting health status
	lockStatus := GetLockStatus()

	return RedisHealthCheck{
		Status:     status,
		Details:    details,
		LockStatus: lockStatus,
	}
}

// testPing tests basic connectivity to Redis
func (h *HealthChecker) testPing() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := h.client.Ping(ctx).Err()
	if err != nil {
		h.lastError = fmt.Sprintf("ping failed: %v", err)
		return false
	}
	return true
}

// testBasicOperations tests basic Redis operations
func (h *HealthChecker) testBasicOperations() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testKey := "health_check_test"
	testValue := "test_value"

	// Set a test value
	err := h.client.Set(ctx, testKey, testValue, time.Minute).Err()
	if err != nil {
		h.lastError = fmt.Sprintf("set operation failed: %v", err)
		return false
	}

	// Get the test value
	value, err := h.client.Get(ctx, testKey).Result()
	if err != nil {
		h.lastError = fmt.Sprintf("get operation failed: %v", err)
		return false
	}

	if value != testValue {
		h.lastError = fmt.Sprintf("value mismatch: expected %s, got %s", testValue, value)
		return false
	}

	// Clean up
	err = h.client.Del(ctx, testKey).Err()
	if err != nil {
		h.lastError = fmt.Sprintf("delete operation failed: %v", err)
		return false
	}

	return true
}

// testConnectionPool tests the connection pool health
func (h *HealthChecker) testConnectionPool() bool {
	stats := h.client.PoolStats()

	// Check if pool is accessible
	if stats.TotalConns == 0 && stats.IdleConns == 0 {
		h.lastError = "connection pool is not accessible"
		return false
	}

	return true
}

// testMemoryUsage tests if Redis memory operations are accessible
func (h *HealthChecker) testMemoryUsage() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := h.client.Info(ctx, "memory").Result()
	if err != nil {
		h.lastError = fmt.Sprintf("memory info failed: %v", err)
		return false
	}

	return true
}

// IsHealthy returns the current health status
func (h *HealthChecker) IsHealthy() bool {
	return atomic.LoadInt32(&h.isHealthy) == 1
}

// GetLastError returns the last error encountered during health checks
func (h *HealthChecker) GetLastError() string {
	return h.lastError
}

// GetLastCheck returns the time of the last health check
func (h *HealthChecker) GetLastCheck() time.Time {
	return h.lastCheck
}

// SetCheckInterval sets the interval for periodic health checks
func (h *HealthChecker) SetCheckInterval(interval time.Duration) {
	h.checkInterval = interval
}

// StartPeriodicHealthCheck starts a goroutine that performs periodic health checks
func (h *HealthChecker) StartPeriodicHealthCheck(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(h.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.HealthCheck()
			}
		}
	}()
}

// QuickHealthCheck performs a quick health check (ping only)
func (h *HealthChecker) QuickHealthCheck() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := h.client.Ping(ctx).Err()
	if err != nil {
		h.lastError = fmt.Sprintf("quick ping failed: %v", err)
		atomic.StoreInt32(&h.isHealthy, 0)
		return false
	}

	atomic.StoreInt32(&h.isHealthy, 1)
	h.lastError = ""
	return true
}

// GetPoolStats returns the current connection pool statistics
func (h *HealthChecker) GetPoolStats() *redis.PoolStats {
	return h.client.PoolStats()
}

// TestConnection tests the Redis connection with a custom timeout
func (h *HealthChecker) TestConnection(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return h.client.Ping(ctx).Err()
}
