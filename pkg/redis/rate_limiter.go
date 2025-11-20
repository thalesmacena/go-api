package redis

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"
)

// RateLimiterRegistry tracks active rate limiters for health check
type RateLimiterRegistry struct {
	limiters map[string]*RateLimiter
	mu       sync.RWMutex
}

// Global rate limiter registry
var rateLimiterRegistry = &RateLimiterRegistry{
	limiters: make(map[string]*RateLimiter),
}

// RegisterRateLimiter registers a rate limiter with the registry
func (rlr *RateLimiterRegistry) RegisterRateLimiter(limiter *RateLimiter) {
	rlr.mu.Lock()
	defer rlr.mu.Unlock()

	if limiter.opts.CacheName != "" {
		rlr.limiters[limiter.opts.CacheName] = limiter
	}
}

// UnregisterRateLimiter removes a rate limiter from the registry
func (rlr *RateLimiterRegistry) UnregisterRateLimiter(cacheName string) {
	rlr.mu.Lock()
	defer rlr.mu.Unlock()

	delete(rlr.limiters, cacheName)
}

// GetRateLimiterMetrics returns the metrics of all registered rate limiters
func (rlr *RateLimiterRegistry) GetRateLimiterMetrics(ctx context.Context) map[string]map[string]string {
	rlr.mu.RLock()
	defer rlr.mu.RUnlock()

	metrics := make(map[string]map[string]string)
	for cacheName, limiter := range rlr.limiters {
		m, err := limiter.GetMetrics(ctx)
		if err == nil {
			metrics[cacheName] = m
		}
	}
	return metrics
}

// RateLimiterOptions represents options for rate limiting
type RateLimiterOptions struct {
	// MaxActiveTransactions is the maximum number of concurrent active transactions (optional)
	MaxActiveTransactions int
	// MaxTransactionsPerSecond is the maximum number of transactions per second (optional)
	MaxTransactionsPerSecond int
	// MaxTransactionsPerMinute is the maximum number of transactions per minute (optional)
	MaxTransactionsPerMinute int
	// WaitOnLimit indicates whether to wait when limit is reached (true) or return error immediately (false)
	WaitOnLimit bool
	// WaitTimeout is the maximum time to wait when WaitOnLimit is true
	WaitTimeout time.Duration
	// RetryDelay is the delay between retry attempts when waiting
	RetryDelay time.Duration
	// Namespace is the namespace for organizing rate limiters
	Namespace string
	// CacheName is used for health check identification
	CacheName string
	// TransactionTTL is the maximum time a transaction can be active before auto-release
	TransactionTTL time.Duration
}

// NewRateLimiterOptions creates a new rate limiter options with default values
func NewRateLimiterOptions() *RateLimiterOptions {
	return &RateLimiterOptions{
		MaxActiveTransactions:    0, // Unlimited by default
		MaxTransactionsPerSecond: 0, // Unlimited by default
		MaxTransactionsPerMinute: 0, // Unlimited by default
		WaitOnLimit:              false,
		WaitTimeout:              30 * time.Second,
		RetryDelay:               100 * time.Millisecond,
		Namespace:                "",
		CacheName:                "",
		TransactionTTL:           5 * time.Minute,
	}
}

// WithMaxActiveTransactions sets the maximum number of concurrent transactions
func (rlo *RateLimiterOptions) WithMaxActiveTransactions(max int) *RateLimiterOptions {
	if max < 0 {
		panic(fmt.Sprintf("invalid max active transactions: %d, must be non-negative", max))
	}
	rlo.MaxActiveTransactions = max
	return rlo
}

// WithMaxTransactionsPerSecond sets the maximum number of transactions per second
func (rlo *RateLimiterOptions) WithMaxTransactionsPerSecond(max int) *RateLimiterOptions {
	if max < 0 {
		panic(fmt.Sprintf("invalid max transactions per second: %d, must be non-negative", max))
	}
	rlo.MaxTransactionsPerSecond = max
	return rlo
}

// WithMaxTransactionsPerMinute sets the maximum number of transactions per minute
func (rlo *RateLimiterOptions) WithMaxTransactionsPerMinute(max int) *RateLimiterOptions {
	if max < 0 {
		panic(fmt.Sprintf("invalid max transactions per minute: %d, must be non-negative", max))
	}
	rlo.MaxTransactionsPerMinute = max
	return rlo
}

// WithWaitOnLimit sets whether to wait when limit is reached
func (rlo *RateLimiterOptions) WithWaitOnLimit(wait bool) *RateLimiterOptions {
	rlo.WaitOnLimit = wait
	return rlo
}

// WithWaitTimeout sets the maximum time to wait when WaitOnLimit is true
func (rlo *RateLimiterOptions) WithWaitTimeout(timeout time.Duration) *RateLimiterOptions {
	if timeout < 0 {
		panic(fmt.Sprintf("invalid wait timeout: %v, must be non-negative", timeout))
	}
	rlo.WaitTimeout = timeout
	return rlo
}

// WithRetryDelay sets the delay between retry attempts
func (rlo *RateLimiterOptions) WithRetryDelay(delay time.Duration) *RateLimiterOptions {
	if delay < 0 {
		panic(fmt.Sprintf("invalid retry delay: %v, must be non-negative", delay))
	}
	rlo.RetryDelay = delay
	return rlo
}

// WithNamespace sets the namespace for organizing rate limiters
func (rlo *RateLimiterOptions) WithNamespace(namespace string) *RateLimiterOptions {
	rlo.Namespace = namespace
	return rlo
}

// WithCacheName sets the cache name for health check identification
func (rlo *RateLimiterOptions) WithCacheName(cacheName string) *RateLimiterOptions {
	rlo.CacheName = cacheName
	return rlo
}

// WithTransactionTTL sets the maximum time a transaction can be active
func (rlo *RateLimiterOptions) WithTransactionTTL(ttl time.Duration) *RateLimiterOptions {
	if ttl < 0 {
		panic(fmt.Sprintf("invalid transaction TTL: %v, must be non-negative", ttl))
	}
	rlo.TransactionTTL = ttl
	return rlo
}

// Validate validates the rate limiter options
func (rlo *RateLimiterOptions) Validate() error {
	if rlo.MaxActiveTransactions == 0 && rlo.MaxTransactionsPerSecond == 0 && rlo.MaxTransactionsPerMinute == 0 {
		return fmt.Errorf("at least one limit must be configured (MaxActiveTransactions, MaxTransactionsPerSecond, or MaxTransactionsPerMinute)")
	}
	return nil
}

// DefaultRateLimiterOptions returns default rate limiter options
func DefaultRateLimiterOptions() *RateLimiterOptions {
	return NewRateLimiterOptions()
}

// RateLimiter represents a distributed rate limiter
type RateLimiter struct {
	client        *Client
	key           string
	opts          *RateLimiterOptions
	transactionID string
	activeKeyName string
	tpsKeyName    string
	tpmKeyName    string
}

// NewRateLimiter creates a new distributed rate limiter
func NewRateLimiter(client *Client, key string, opts *RateLimiterOptions) (*RateLimiter, error) {
	if opts == nil {
		opts = DefaultRateLimiterOptions()
	}

	if err := opts.Validate(); err != nil {
		return nil, err
	}

	limiter := &RateLimiter{
		client: client,
		key:    key,
		opts:   opts,
	}

	// Build key names
	limiter.activeKeyName = limiter.buildKey("active")
	limiter.tpsKeyName = limiter.buildKey("tps")
	limiter.tpmKeyName = limiter.buildKey("tpm")

	// Register rate limiter if it has a cache name
	if opts.CacheName != "" {
		rateLimiterRegistry.RegisterRateLimiter(limiter)
	}

	return limiter, nil
}

// buildKey constructs the full key using Namespace::key::suffix format
func (rl *RateLimiter) buildKey(suffix string) string {
	if rl.opts.Namespace != "" {
		return rl.opts.Namespace + "::" + rl.key + "::" + suffix
	}
	return rl.key + "::" + suffix
}

// Acquire attempts to acquire a transaction slot
func (rl *RateLimiter) Acquire(ctx context.Context) (string, error) {
	if rl.opts.WaitOnLimit {
		return rl.acquireWithWait(ctx)
	}
	return rl.acquireImmediate(ctx)
}

// acquireImmediate attempts to acquire immediately or returns error
func (rl *RateLimiter) acquireImmediate(ctx context.Context) (string, error) {
	// Check all limits using Lua script for atomicity
	script := rl.buildAcquireScript()

	now := time.Now()
	transactionID := fmt.Sprintf("%d", now.UnixNano())

	result, err := rl.client.GetClient().Eval(ctx, script, []string{
		rl.activeKeyName,
		rl.tpsKeyName,
		rl.tpmKeyName,
	},
		rl.opts.MaxActiveTransactions,
		rl.opts.MaxTransactionsPerSecond,
		rl.opts.MaxTransactionsPerMinute,
		transactionID,
		now.Unix(),
		now.UnixNano(),
		int(rl.opts.TransactionTTL.Seconds()),
	).Result()

	if err != nil {
		return "", fmt.Errorf("failed to acquire rate limiter: %w", err)
	}

	// Result: 1 = success, 0 = active limit, -1 = TPS limit, -2 = TPM limit
	resultCode := result.(int64)
	if resultCode == 1 {
		rl.transactionID = transactionID
		return transactionID, nil
	}

	switch resultCode {
	case 0:
		return "", fmt.Errorf("active transactions limit reached (%d/%d)", rl.opts.MaxActiveTransactions, rl.opts.MaxActiveTransactions)
	case -1:
		return "", fmt.Errorf("transactions per second limit reached (%d TPS)", rl.opts.MaxTransactionsPerSecond)
	case -2:
		return "", fmt.Errorf("transactions per minute limit reached (%d TPM)", rl.opts.MaxTransactionsPerMinute)
	default:
		return "", fmt.Errorf("unknown error code: %d", resultCode)
	}
}

// acquireWithWait attempts to acquire with retry/wait logic
func (rl *RateLimiter) acquireWithWait(ctx context.Context) (string, error) {
	deadline := time.Now().Add(rl.opts.WaitTimeout)

	for {
		transactionID, err := rl.acquireImmediate(ctx)
		if err == nil {
			return transactionID, nil
		}

		// Check if we've exceeded the timeout
		if time.Now().After(deadline) {
			return "", fmt.Errorf("timeout waiting for rate limiter: %w", err)
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(rl.opts.RetryDelay):
			// Continue retry
		}
	}
}

// buildAcquireScript builds the Lua script for atomic acquire operation
func (rl *RateLimiter) buildAcquireScript() string {
	return `
		local active_key = KEYS[1]
		local tps_key = KEYS[2]
		local tpm_key = KEYS[3]
		
		local max_active = tonumber(ARGV[1])
		local max_tps = tonumber(ARGV[2])
		local max_tpm = tonumber(ARGV[3])
		local transaction_id = ARGV[4]
		local now_seconds = tonumber(ARGV[5])
		local now_nanos = tonumber(ARGV[6])
		local transaction_ttl = tonumber(ARGV[7])
		
		-- Check active transactions limit
		if max_active > 0 then
			local active_count = tonumber(redis.call("GET", active_key)) or 0
			if active_count >= max_active then
				return 0
			end
		end
		
		-- Check TPS limit (sliding window of 1 second)
		if max_tps > 0 then
			-- Remove old entries (older than 1 second)
			local tps_cutoff_time = now_nanos - (1 * 1000000000)
			redis.call("ZREMRANGEBYSCORE", tps_key, "-inf", tps_cutoff_time)
			
			-- Count entries in the last second
			local tps_count = redis.call("ZCOUNT", tps_key, tps_cutoff_time, "+inf")
			if tps_count >= max_tps then
				return -1
			end
		end
		
		-- Check TPM limit (sliding window of 60 seconds)
		if max_tpm > 0 then
			-- Remove old entries (older than 60 seconds)
			local tpm_cutoff_time = now_nanos - (60 * 1000000000)
			redis.call("ZREMRANGEBYSCORE", tpm_key, "-inf", tpm_cutoff_time)
			
			-- Count entries in the last minute
			local tpm_count = redis.call("ZCOUNT", tpm_key, tpm_cutoff_time, "+inf")
			if tpm_count >= max_tpm then
				return -2
			end
		end
		
		-- All checks passed, acquire the transaction
		
		-- Increment active transactions
		if max_active > 0 then
			redis.call("INCR", active_key)
			redis.call("EXPIRE", active_key, transaction_ttl * 2)
		end
		
		-- Add to TPS sorted set (sliding window)
		if max_tps > 0 then
			redis.call("ZADD", tps_key, now_nanos, transaction_id .. ":tps")
			redis.call("EXPIRE", tps_key, 2)
		end
		
		-- Add to TPM sorted set (sliding window)
		if max_tpm > 0 then
			redis.call("ZADD", tpm_key, now_nanos, transaction_id)
			redis.call("EXPIRE", tpm_key, 60)
		end
		
		return 1
	`
}

// Release releases a transaction slot
func (rl *RateLimiter) Release(ctx context.Context, transactionID string) error {
	if transactionID == "" {
		return fmt.Errorf("transaction ID is required")
	}

	// Only decrement active transactions counter
	if rl.opts.MaxActiveTransactions > 0 {
		count, err := rl.client.Decr(ctx, rl.activeKeyName)
		if err != nil {
			return fmt.Errorf("failed to release transaction: %w", err)
		}

		// Ensure count doesn't go below zero
		if count < 0 {
			_ = rl.client.Set(ctx, rl.activeKeyName, 0, rl.opts.TransactionTTL*2)
		}
	}

	return nil
}

// RateLimiterMetrics represents the current metrics of the rate limiter as key-value pairs
type RateLimiterMetrics map[string]string

// GetMetrics returns the current metrics of the rate limiter
func (rl *RateLimiter) GetMetrics(ctx context.Context) (RateLimiterMetrics, error) {
	metrics := make(RateLimiterMetrics)

	now := time.Now()

	// Get active transactions count
	if rl.opts.MaxActiveTransactions > 0 {
		activeCount, err := rl.client.GetInt(ctx, rl.activeKeyName)
		if err != nil {
			activeCount = 0
		}
		metrics["active_transactions"] = strconv.Itoa(int(activeCount))
		metrics["max_active_transactions"] = strconv.Itoa(rl.opts.MaxActiveTransactions)
		utilization := float64(activeCount) / float64(rl.opts.MaxActiveTransactions) * 100
		metrics["active_utilization"] = fmt.Sprintf("%.1f%%", utilization)
	}

	// Get TPS count (sliding window of 1 second)
	if rl.opts.MaxTransactionsPerSecond > 0 {
		// Count entries in the last second
		tpsCutoffTime := now.Add(-1 * time.Second).UnixNano()

		tpsCount, err := rl.client.GetClient().ZCount(ctx, rl.tpsKeyName,
			strconv.FormatInt(tpsCutoffTime, 10),
			"+inf").Result()
		if err != nil {
			tpsCount = 0
		}
		metrics["transactions_per_second"] = strconv.FormatInt(tpsCount, 10)
		metrics["max_transactions_per_second"] = strconv.Itoa(rl.opts.MaxTransactionsPerSecond)
		utilization := float64(tpsCount) / float64(rl.opts.MaxTransactionsPerSecond) * 100
		metrics["tps_utilization"] = fmt.Sprintf("%.1f%%", utilization)
	}

	// Get TPM count (sliding window of 60 seconds)
	if rl.opts.MaxTransactionsPerMinute > 0 {
		// Count entries in the last minute
		tpmCutoffTime := now.Add(-60 * time.Second).UnixNano()

		tpmCount, err := rl.client.GetClient().ZCount(ctx, rl.tpmKeyName,
			strconv.FormatInt(tpmCutoffTime, 10),
			"+inf").Result()
		if err != nil {
			tpmCount = 0
		}
		metrics["transactions_per_minute"] = strconv.FormatInt(tpmCount, 10)
		metrics["max_transactions_per_minute"] = strconv.Itoa(rl.opts.MaxTransactionsPerMinute)
		utilization := float64(tpmCount) / float64(rl.opts.MaxTransactionsPerMinute) * 100
		metrics["tpm_utilization"] = fmt.Sprintf("%.1f%%", utilization)
	}

	return metrics, nil
}

// GetCacheName returns the cache name for health check identification
func (rl *RateLimiter) GetCacheName() string {
	return rl.opts.CacheName
}

// GetKey returns the rate limiter key
func (rl *RateLimiter) GetKey() string {
	return rl.key
}

// Cleanup removes all keys associated with this rate limiter
func (rl *RateLimiter) Cleanup(ctx context.Context) error {
	keys := []string{rl.activeKeyName, rl.tpsKeyName, rl.tpmKeyName}
	err := rl.client.Delete(ctx, keys...)

	// Unregister from registry
	if rl.opts.CacheName != "" {
		rateLimiterRegistry.UnregisterRateLimiter(rl.opts.CacheName)
	}

	return err
}

// GetRateLimiterMetrics returns the metrics of all registered rate limiters for health check
func GetRateLimiterMetrics(ctx context.Context) map[string]map[string]string {
	return rateLimiterRegistry.GetRateLimiterMetrics(ctx)
}

// WithTransaction executes a function with rate limiting
func (rl *RateLimiter) WithTransaction(ctx context.Context, fn func() error) error {
	// Acquire transaction
	transactionID, err := rl.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire rate limiter: %w", err)
	}

	// Ensure transaction is released
	defer func() {
		if err := rl.Release(ctx, transactionID); err != nil {
			// Log error but don't fail the function
			// In a real implementation, you might want to use a logger here
		}
	}()

	// Execute function
	return fn()
}
