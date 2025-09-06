package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// LockOptions represents options for distributed locking
type LockOptions struct {
	// TTL is the lock expiration time
	TTL time.Duration
	// RetryDelay is the delay between retry attempts
	RetryDelay time.Duration
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int
	// RefreshInterval is the interval for refreshing the lock
	RefreshInterval time.Duration
	// LockNamespace is the namespace for organizing locks
	LockNamespace string
}

// NewLockOptions creates a new lock options with default values
func NewLockOptions() *LockOptions {
	return &LockOptions{
		TTL:             30 * time.Second,
		RetryDelay:      100 * time.Millisecond,
		MaxRetries:      10,
		RefreshInterval: 10 * time.Second,
		LockNamespace:   "",
	}
}

// WithTTL sets the lock expiration time
func (lo *LockOptions) WithTTL(ttl time.Duration) *LockOptions {
	if ttl < 0 {
		panic(fmt.Sprintf("invalid TTL: %v, must be non-negative", ttl))
	}
	lo.TTL = ttl
	return lo
}

// WithRetryDelay sets the delay between retry attempts
func (lo *LockOptions) WithRetryDelay(delay time.Duration) *LockOptions {
	if delay < 0 {
		panic(fmt.Sprintf("invalid retry delay: %v, must be non-negative", delay))
	}
	lo.RetryDelay = delay
	return lo
}

// WithMaxRetries sets the maximum number of retry attempts
func (lo *LockOptions) WithMaxRetries(maxRetries int) *LockOptions {
	if maxRetries < 0 {
		panic(fmt.Sprintf("invalid max retries: %d, must be non-negative", maxRetries))
	}
	lo.MaxRetries = maxRetries
	return lo
}

// WithRefreshInterval sets the interval for refreshing the lock
func (lo *LockOptions) WithRefreshInterval(interval time.Duration) *LockOptions {
	if interval < 0 {
		panic(fmt.Sprintf("invalid refresh interval: %v, must be non-negative", interval))
	}
	lo.RefreshInterval = interval
	return lo
}

// WithLockNamespace sets the namespace for organizing locks
func (lo *LockOptions) WithLockNamespace(namespace string) *LockOptions {
	lo.LockNamespace = namespace
	return lo
}

// DefaultLockOptions returns default lock options
func DefaultLockOptions() *LockOptions {
	return NewLockOptions()
}

// Lock represents a distributed lock
type Lock struct {
	client *Client
	key    string
	value  string
	opts   *LockOptions
}

// NewLock creates a new distributed lock
func NewLock(client *Client, key string, opts *LockOptions) *Lock {
	if opts == nil {
		opts = DefaultLockOptions()
	}
	return &Lock{
		client: client,
		key:    key,
		value:  generateLockValue(),
		opts:   opts,
	}
}

// buildLockKey constructs the full lock key using LockNamespace::lockKey format
func (l *Lock) buildLockKey() string {
	if l.opts.LockNamespace != "" {
		return l.opts.LockNamespace + "::" + l.key
	}
	return l.key
}

// Lock attempts to acquire the lock
func (l *Lock) Lock(ctx context.Context) error {
	fullKey := l.buildLockKey()
	for attempt := 0; attempt <= l.opts.MaxRetries; attempt++ {
		// Try to acquire the lock using SET with NX and EX options
		result, err := l.client.GetClient().SetNX(ctx, fullKey, l.value, l.opts.TTL).Result()
		if err != nil {
			return fmt.Errorf("failed to acquire lock: %w", err)
		}

		if result {
			return nil
		}

		// If this is the last attempt, return error
		if attempt == l.opts.MaxRetries {
			return fmt.Errorf("failed to acquire lock after %d attempts", l.opts.MaxRetries+1)
		}

		// Wait before retrying
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(l.opts.RetryDelay):
			continue
		}
	}

	return fmt.Errorf("failed to acquire lock")
}

// Unlock releases the lock
func (l *Lock) Unlock(ctx context.Context) error {
	// Use Lua script to ensure we only delete our own lock
	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`

	fullKey := l.buildLockKey()
	result, err := l.client.GetClient().Eval(ctx, script, []string{fullKey}, l.value).Result()
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	if result.(int64) == 0 {
		return fmt.Errorf("lock was not held by this client")
	}

	return nil
}

// Refresh extends the lock's TTL
func (l *Lock) Refresh(ctx context.Context) error {
	// Use Lua script to ensure we only refresh our own lock
	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("EXPIRE", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	fullKey := l.buildLockKey()
	result, err := l.client.GetClient().Eval(ctx, script, []string{fullKey}, l.value, int(l.opts.TTL.Seconds())).Result()
	if err != nil {
		return fmt.Errorf("failed to refresh lock: %w", err)
	}

	if result.(int64) == 0 {
		return fmt.Errorf("lock was not held by this client")
	}

	return nil
}

// IsLocked checks if the lock is currently held
func (l *Lock) IsLocked(ctx context.Context) (bool, error) {
	fullKey := l.buildLockKey()
	value, err := l.client.Get(ctx, fullKey)
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}

	return value == l.value, nil
}

// AutoRefresh starts a goroutine that automatically refreshes the lock
func (l *Lock) AutoRefresh(ctx context.Context) <-chan error {
	errChan := make(chan error, 1)

	go func() {
		ticker := time.NewTicker(l.opts.RefreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			case <-ticker.C:
				if err := l.Refresh(ctx); err != nil {
					errChan <- err
					return
				}
			}
		}
	}()

	return errChan
}

// LockWithFunc executes a function while holding a lock
func LockWithFunc(ctx context.Context, client *Client, key string, opts *LockOptions, fn func() error) error {
	lock := NewLock(client, key, opts)

	// Acquire lock
	if err := lock.Lock(ctx); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	// Ensure lock is released
	defer func() {
		if err := lock.Unlock(ctx); err != nil {
			// Log error but don't fail the function
			// In a real implementation, you might want to use a logger here
		}
	}()

	// Execute function
	return fn()
}

// LockWithTimeout executes a function while holding a lock with timeout
func LockWithTimeout(ctx context.Context, client *Client, key string, opts *LockOptions, timeout time.Duration, fn func() error) error {
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return LockWithFunc(timeoutCtx, client, key, opts, fn)
}

// generateLockValue generates a unique value for the lock
func generateLockValue() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
