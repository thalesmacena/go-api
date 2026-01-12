package redis

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// LockRegistry tracks active locks for health check
type LockRegistry struct {
	locks map[string]*Lock
	mu    sync.RWMutex
}

// Global lock registry
var lockRegistry = &LockRegistry{
	locks: make(map[string]*Lock),
}

// RegisterLock registers a lock with the registry
func (lr *LockRegistry) RegisterLock(lock *Lock) {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	if lock.opts.CacheName != "" {
		lr.locks[lock.opts.CacheName] = lock
	}
}

// UnregisterLock removes a lock from the registry
func (lr *LockRegistry) UnregisterLock(cacheName string) {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	delete(lr.locks, cacheName)
}

// GetLockStatus returns the status of all registered locks
func (lr *LockRegistry) GetLockStatus() map[string]bool {
	lr.mu.RLock()
	defer lr.mu.RUnlock()

	status := make(map[string]bool)
	for cacheName, lock := range lr.locks {
		status[cacheName] = lock.IsAcquired()
	}
	return status
}

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
	// CacheName is used for health check identification
	CacheName string
	// PersistentRefresh indicates if the lock should be refreshed indefinitely
	PersistentRefresh bool
	// InfiniteRetry indicates if the lock should retry indefinitely
	InfiniteRetry bool
}

// NewLockOptions creates a new lock options with default values
func NewLockOptions() *LockOptions {
	return &LockOptions{
		TTL:               30 * time.Second,
		RetryDelay:        100 * time.Millisecond,
		MaxRetries:        10,
		RefreshInterval:   10 * time.Second,
		LockNamespace:     "",
		CacheName:         "",
		PersistentRefresh: false,
		InfiniteRetry:     false,
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

// WithCacheName sets the cache name for health check identification
func (lo *LockOptions) WithCacheName(cacheName string) *LockOptions {
	lo.CacheName = cacheName
	return lo
}

// WithPersistentRefresh sets whether the lock should be refreshed indefinitely
func (lo *LockOptions) WithPersistentRefresh(persistent bool) *LockOptions {
	lo.PersistentRefresh = persistent
	return lo
}

// WithInfiniteRetry sets whether the lock should retry indefinitely
func (lo *LockOptions) WithInfiniteRetry(infinite bool) *LockOptions {
	lo.InfiniteRetry = infinite
	return lo
}

// DefaultLockOptions returns default lock options
func DefaultLockOptions() *LockOptions {
	return NewLockOptions()
}

// Lock represents a distributed lock
type Lock struct {
	client      *Client
	key         string
	value       string
	opts        *LockOptions
	refreshStop chan struct{}
	refreshing  bool
	acquired    bool
}

// NewLock creates a new distributed lock
func NewLock(client *Client, key string, opts *LockOptions) *Lock {
	if opts == nil {
		opts = DefaultLockOptions()
	}
	lock := &Lock{
		client:      client,
		key:         key,
		value:       generateLockValue(),
		opts:        opts,
		refreshStop: make(chan struct{}),
	}

	// Register lock if it has a cache name
	if opts.CacheName != "" {
		lockRegistry.RegisterLock(lock)
	}

	return lock
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

	// Handle infinite retry scenario
	if l.opts.InfiniteRetry {
		for {
			// Try to acquire the lock using SET with NX and EX options
			result, err := l.client.GetClient().SetNX(ctx, fullKey, l.value, l.opts.TTL).Result()
			if err != nil {
				return fmt.Errorf("failed to acquire lock: %w", err)
			}

			if result {
				l.acquired = true
				return nil
			}

			// Wait before retrying
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(l.opts.RetryDelay):
				continue
			}
		}
	}

	// Handle finite retry scenario
	for attempt := 0; attempt < l.opts.MaxRetries; attempt++ {
		// Try to acquire the lock using SET with NX and EX options
		result, err := l.client.GetClient().SetNX(ctx, fullKey, l.value, l.opts.TTL).Result()
		if err != nil {
			return fmt.Errorf("failed to acquire lock: %w", err)
		}

		if result {
			l.acquired = true
			return nil
		}

		// Wait before retrying
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(l.opts.RetryDelay):
			continue
		}
	}

	// Final attempt
	result, err := l.client.GetClient().SetNX(ctx, fullKey, l.value, l.opts.TTL).Result()
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	if result {
		l.acquired = true
		return nil
	}

	return fmt.Errorf("failed to acquire lock after %d attempts", l.opts.MaxRetries)
}

// Unlock releases the lock
func (l *Lock) Unlock(ctx context.Context) error {
	// Stop auto-refresh if it's running
	l.StopAutoRefresh()

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

	l.acquired = false

	// Unregister lock from registry
	if l.opts.CacheName != "" {
		lockRegistry.UnregisterLock(l.opts.CacheName)
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
		return fmt.Errorf("lock was not held by this client or has expired")
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
// If the context is cancelled, the auto-refresh will stop
func (l *Lock) AutoRefresh(ctx context.Context) <-chan error {
	errChan := make(chan error, 1)

	if l.refreshing {
		errChan <- fmt.Errorf("auto-refresh is already running")
		return errChan
	}

	// Create new channel for this refresh session
	l.refreshStop = make(chan struct{})
	l.refreshing = true

	go func() {
		defer func() {
			l.refreshing = false
			// Always send a completion signal when the goroutine exits
			select {
			case errChan <- nil:
			default:
				// Channel might be closed, ignore
			}
		}()

		ticker := time.NewTicker(l.opts.RefreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				// Always respect context cancellation, even for persistent refresh
				// This allows the Graceful shutdown
				select {
				case errChan <- ctx.Err():
				default:
					// Channel might be closed, ignore
				}
				return
			case <-l.refreshStop:
				// Lock was released, stop refreshing
				// Completion signal will be sent by defer
				return
			case <-ticker.C:
				// For persistent refresh, use background context to avoid cancellation
				// For non-persistent refresh, use the provided context
				refreshCtx := ctx
				if l.opts.PersistentRefresh {
					refreshCtx = context.Background()
				}
				if err := l.Refresh(refreshCtx); err != nil {
					if !l.opts.PersistentRefresh {
						// For non-persistent refresh, send the refresh error
						select {
						case errChan <- err:
						default:
							// Channel might be closed, ignore
						}
						return
					}
					// For persistent refresh, continue trying to refresh
					// This ensures the lock is maintained even if individual refreshes fail
				}
			}
		}
	}()

	return errChan
}

// StopAutoRefresh stops the auto-refresh goroutine
func (l *Lock) StopAutoRefresh() {
	if l.refreshing {
		select {
		case <-l.refreshStop:
			// Channel already closed, nothing to do
		default:
			close(l.refreshStop)
		}
		l.refreshing = false
	}
}

// IsAutoRefreshing returns true if auto-refresh is currently running
func (l *Lock) IsAutoRefreshing() bool {
	return l.refreshing
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

// IsAcquired returns true if the lock is currently acquired by this instance
func (l *Lock) IsAcquired() bool {
	return l.acquired
}

// GetCacheName returns the cache name for health check identification
func (l *Lock) GetCacheName() string {
	return l.opts.CacheName
}

// GetKey returns the lock key
func (l *Lock) GetKey() string {
	return l.key
}

// GetFullKey returns the full lock key with namespace
func (l *Lock) GetFullKey() string {
	return l.buildLockKey()
}

// NewSingleAttemptLock creates a lock for scenario 1 (single attempt, no retry)
func NewSingleAttemptLock(client *Client, key string, ttl time.Duration, namespace string) *Lock {
	opts := NewLockOptions().
		WithTTL(ttl).
		WithMaxRetries(0).
		WithLockNamespace(namespace).
		WithCacheName(key)
	return NewLock(client, key, opts)
}

// NewRetryLock creates a lock for scenario 2 (retry with manual refresh)
func NewRetryLock(client *Client, key string, ttl time.Duration, retryDelay time.Duration, maxRetries int, namespace string) *Lock {
	opts := NewLockOptions().
		WithTTL(ttl).
		WithRetryDelay(retryDelay).
		WithMaxRetries(maxRetries).
		WithLockNamespace(namespace).
		WithCacheName(key)
	return NewLock(client, key, opts)
}

// NewPersistentLock creates a lock for scenario 3 (persistent with auto refresh)
func NewPersistentLock(client *Client, key string, ttl time.Duration, refreshInterval time.Duration, namespace string) *Lock {
	opts := NewLockOptions().
		WithTTL(ttl).
		WithRefreshInterval(refreshInterval).
		WithPersistentRefresh(true).
		WithInfiniteRetry(true).
		WithLockNamespace(namespace).
		WithCacheName(key)
	return NewLock(client, key, opts)
}

// NewScheduledTaskLock creates a lock for scenario 4 (scheduled task with persistent refresh)
func NewScheduledTaskLock(client *Client, key string, ttl time.Duration, refreshInterval time.Duration, namespace string) *Lock {
	opts := NewLockOptions().
		WithTTL(ttl).
		WithRefreshInterval(refreshInterval).
		WithPersistentRefresh(true).
		WithInfiniteRetry(true).
		WithLockNamespace(namespace).
		WithCacheName(key)
	return NewLock(client, key, opts)
}

// GetLockStatus returns the status of all registered locks for health check
func GetLockStatus() map[string]bool {
	return lockRegistry.GetLockStatus()
}

// generateLockValue generates a unique value for the lock
func generateLockValue() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}
