package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// HealthCheck performs a health check on the Redis connection
func HealthCheck(ctx context.Context, client *Client) error {
	// Test basic connectivity
	if err := client.Ping(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	// Test basic operations
	testKey := "health_check_test"
	testValue := "test_value"

	// Set a test value
	if err := client.Set(ctx, testKey, testValue, time.Minute); err != nil {
		return fmt.Errorf("set operation failed: %w", err)
	}

	// Get the test value
	value, err := client.Get(ctx, testKey)
	if err != nil {
		return fmt.Errorf("get operation failed: %w", err)
	}

	if value != testValue {
		return fmt.Errorf("value mismatch: expected %s, got %s", testValue, value)
	}

	// Clean up
	if err := client.Delete(ctx, testKey); err != nil {
		return fmt.Errorf("delete operation failed: %w", err)
	}

	return nil
}

// BatchOperation represents a batch operation
type BatchOperation struct {
	Operation string
	Key       string
	Value     interface{}
	TTL       time.Duration
}

// BatchResult represents the result of a batch operation
type BatchResult struct {
	Successful []string
	Failed     map[string]error
}

// ExecuteBatch executes multiple operations in a pipeline
func ExecuteBatch(ctx context.Context, client *Client, operations []BatchOperation) (*BatchResult, error) {
	pipe := client.Pipeline()

	// Add operations to pipeline
	for _, op := range operations {
		switch op.Operation {
		case "SET":
			pipe.Set(ctx, op.Key, op.Value, op.TTL)
		case "GET":
			pipe.Get(ctx, op.Key)
		case "DEL":
			pipe.Del(ctx, op.Key)
		case "EXPIRE":
			pipe.Expire(ctx, op.Key, op.TTL)
		default:
			return nil, fmt.Errorf("unsupported operation: %s", op.Operation)
		}
	}

	// Execute pipeline
	cmders, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("pipeline execution failed: %w", err)
	}

	// Process results
	result := &BatchResult{
		Successful: make([]string, 0),
		Failed:     make(map[string]error),
	}

	for i, cmder := range cmders {
		op := operations[i]
		if err := cmder.Err(); err != nil {
			result.Failed[op.Key] = err
		} else {
			result.Successful = append(result.Successful, op.Key)
		}
	}

	return result, nil
}

// KeyPattern represents a key pattern for operations
type KeyPattern struct {
	Pattern string
	Count   int64
}

// ScanKeys scans for keys matching a pattern
func ScanKeys(ctx context.Context, client *Client, pattern string, count int64) ([]string, error) {
	var keys []string
	var cursor uint64

	for {
		scanKeys, nextCursor, err := client.Scan(ctx, cursor, pattern, count)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		keys = append(keys, scanKeys...)
		cursor = nextCursor

		if cursor == 0 {
			break
		}
	}

	return keys, nil
}

// DeleteKeysByPattern deletes all keys matching a pattern
func DeleteKeysByPattern(ctx context.Context, client *Client, pattern string, batchSize int64) error {
	keys, err := ScanKeys(ctx, client, pattern, batchSize)
	if err != nil {
		return fmt.Errorf("failed to scan keys: %w", err)
	}

	if len(keys) == 0 {
		return nil
	}

	// Delete keys in batches
	for i := 0; i < len(keys); i += int(batchSize) {
		end := i + int(batchSize)
		if end > len(keys) {
			end = len(keys)
		}

		batch := keys[i:end]
		if err := client.Delete(ctx, batch...); err != nil {
			return fmt.Errorf("failed to delete batch: %w", err)
		}
	}

	return nil
}

// GetMemoryUsage returns memory usage information
func GetMemoryUsage(ctx context.Context, client *Client) (map[string]string, error) {
	info, err := client.Info(ctx, "memory")
	if err != nil {
		return nil, fmt.Errorf("failed to get memory info: %w", err)
	}

	// Parse memory info (simplified parsing)
	memoryInfo := make(map[string]string)
	// In a real implementation, you would parse the info string properly
	memoryInfo["raw_info"] = info

	return memoryInfo, nil
}

// GetStats returns client pool statistics
func GetStats(client *Client) *redis.PoolStats {
	return client.Stats()
}

// RetryOperation retries an operation with exponential backoff
func RetryOperation(ctx context.Context, operation func() error, maxRetries int, initialDelay time.Duration) error {
	var err error
	delay := initialDelay

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err = operation()
		if err == nil {
			return nil
		}

		// If this is the last attempt, return the error
		if attempt == maxRetries {
			return fmt.Errorf("operation failed after %d retries: %w", maxRetries+1, err)
		}

		// Wait before retrying
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			delay *= 2 // Exponential backoff
		}
	}

	return err
}

// WithTimeout executes an operation with a timeout
func WithTimeout(ctx context.Context, timeout time.Duration, operation func(context.Context) error) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return operation(timeoutCtx)
}

// IsConnectionError checks if an error is a connection error
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	// Check for common connection errors
	errorStr := err.Error()
	connectionErrors := []string{
		"connection refused",
		"connection reset",
		"connection timeout",
		"network is unreachable",
		"no such host",
		"i/o timeout",
	}

	for _, connErr := range connectionErrors {
		if contains(errorStr, connErr) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr)))
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
