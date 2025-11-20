package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// CacheOptions represents options for cache operations
type CacheOptions struct {
	// TTL is the time to live for the cached value
	TTL time.Duration
	// RefreshTTL indicates whether to refresh the TTL on access
	RefreshTTL bool
	// Serializer is a custom serializer function
	Serializer func(interface{}) ([]byte, error)
	// Deserializer is a custom deserializer function
	Deserializer func([]byte, interface{}) error
	// CacheName is the name of the cache for TTL lookup
	CacheName string
	// ttlSet indicates whether TTL was explicitly set
	ttlSet bool
}

// NewCacheOptions creates a new cache options with default values
func NewCacheOptions() *CacheOptions {
	return &CacheOptions{
		TTL:          0, // Not set by default
		RefreshTTL:   false,
		Serializer:   json.Marshal,
		Deserializer: json.Unmarshal,
		CacheName:    "",
		ttlSet:       false, // TTL not explicitly set
	}
}

// WithTTL sets the TTL for cache operations
func (co *CacheOptions) WithTTL(ttl time.Duration) *CacheOptions {
	if ttl < 0 {
		panic(fmt.Sprintf("invalid TTL: %v, must be non-negative", ttl))
	}
	co.TTL = ttl
	co.ttlSet = true // Mark that TTL was explicitly set
	return co
}

// WithRefreshTTL enables TTL refresh on access
func (co *CacheOptions) WithRefreshTTL(refresh bool) *CacheOptions {
	co.RefreshTTL = refresh
	return co
}

// WithSerializer sets a custom serializer function
func (co *CacheOptions) WithSerializer(serializer func(interface{}) ([]byte, error)) *CacheOptions {
	co.Serializer = serializer
	return co
}

// WithDeserializer sets a custom deserializer function
func (co *CacheOptions) WithDeserializer(deserializer func([]byte, interface{}) error) *CacheOptions {
	co.Deserializer = deserializer
	return co
}

// WithCacheName sets the cache name for TTL lookup
func (co *CacheOptions) WithCacheName(cacheName string) *CacheOptions {
	co.CacheName = cacheName
	return co
}

// DefaultCacheOptions returns default cache options
func DefaultCacheOptions() *CacheOptions {
	return NewCacheOptions()
}

// Cache provides high-level caching operations
type Cache struct {
	client *Client
	opts   *CacheOptions
}

// NewCache creates a new cache instance
func NewCache(client *Client, opts *CacheOptions) *Cache {
	if opts == nil {
		opts = DefaultCacheOptions()
	}
	return &Cache{
		client: client,
		opts:   opts,
	}
}

// getTTL returns the TTL for the cache with proper hierarchy:
// 1. CacheOptions.TTL (highest priority) - only if explicitly set
// 2. Client.Config.CacheTTLs[cacheName] (medium priority)
// 3. Client.Config.DefaultCacheTTL (lowest priority - fallback)
func (c *Cache) getTTL() time.Duration {
	// Priority 1: CacheOptions TTL (most specific) - only if explicitly set
	if c.opts.ttlSet {
		return c.opts.TTL
	}

	// Priority 2: Client config TTL for specific cache name
	if c.opts.CacheName != "" {
		if clientTTL, exists := c.client.config.CacheTTLs[c.opts.CacheName]; exists {
			return clientTTL
		}
	}

	// Priority 3: Default TTL from client config (fallback)
	if c.client.config.DefaultCacheTTL > 0 {
		return c.client.config.DefaultCacheTTL
	}

	// Final fallback: default from CacheOptions (1 hour)
	return 1 * time.Hour
}

// buildCacheKey constructs the full cache key using CacheName::cacheKey format
func (c *Cache) buildCacheKey(key string) string {
	if c.opts.CacheName != "" {
		return c.opts.CacheName + "::" + key
	}
	return key
}

// Get retrieves a value from cache and deserializes it
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) error {
	fullKey := c.buildCacheKey(key)
	data, err := c.client.GetBytes(ctx, fullKey)
	if err != nil {
		// Real error (connection, etc.) - return it
		return err
	}

	// Check if data is empty (key doesn't exist)
	if len(data) == 0 {
		// Key doesn't exist - this is not an error, just return nil
		return nil
	}

	if c.opts.RefreshTTL {
		// Refresh TTL on access
		if err := c.client.Expire(ctx, fullKey, c.getTTL()); err != nil {
			// Log warning but don't fail the operation
			// In a real implementation, you might want to use a logger here
		}
	}

	return c.opts.Deserializer(data, dest)
}

// Set stores a value in cache with serialization
func (c *Cache) Set(ctx context.Context, key string, value interface{}) error {
	fullKey := c.buildCacheKey(key)
	data, err := c.opts.Serializer(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}

	return c.client.Set(ctx, fullKey, data, c.getTTL())
}

// SetWithTTL stores a value in cache with custom TTL (highest priority)
func (c *Cache) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	fullKey := c.buildCacheKey(key)
	data, err := c.opts.Serializer(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}

	return c.client.Set(ctx, fullKey, data, ttl)
}

// SetWithTTLAndRefresh stores a value in cache with custom TTL and refresh TTL on access
func (c *Cache) SetWithTTLAndRefresh(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	fullKey := c.buildCacheKey(key)
	data, err := c.opts.Serializer(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}

	// Store with custom TTL
	err = c.client.Set(ctx, fullKey, data, ttl)
	if err != nil {
		return err
	}

	// If refresh TTL is enabled, we'll handle it in the Get method
	return nil
}

// Delete removes a value from cache
func (c *Cache) Delete(ctx context.Context, key string) error {
	fullKey := c.buildCacheKey(key)
	return c.client.Delete(ctx, fullKey)
}

// Exists checks if a key exists in cache
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := c.buildCacheKey(key)
	count, err := c.client.Exists(ctx, fullKey)
	return count > 0, err
}

// GetOrSet retrieves a value from cache, or sets it if it doesn't exist
func (c *Cache) GetOrSet(ctx context.Context, key string, dest interface{}, setter func() (interface{}, error)) error {
	// Try to get from cache first
	err := c.Get(ctx, key, dest)
	if err == nil {
		return nil
	}

	// If not found or error, call setter function
	value, err := setter()
	if err != nil {
		return fmt.Errorf("setter function failed: %w", err)
	}

	// Set in cache
	if err := c.Set(ctx, key, value); err != nil {
		return fmt.Errorf("failed to set value in cache: %w", err)
	}

	// Serialize the value and deserialize it into dest
	data, err := c.opts.Serializer(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}
	return c.opts.Deserializer(data, dest)
}

// MGet retrieves multiple values from cache
func (c *Cache) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	result := make(map[string][]byte)

	for _, key := range keys {
		fullKey := c.buildCacheKey(key)
		data, err := c.client.GetBytes(ctx, fullKey)
		if err != nil {
			// Real error (connection, etc.) - return it
			return nil, err
		}

		// Only add non-empty data to result
		if len(data) > 0 {
			result[key] = data
		}
	}

	return result, nil
}

// MSet stores multiple values in cache
func (c *Cache) MSet(ctx context.Context, values map[string]interface{}) error {
	for key, value := range values {
		if err := c.Set(ctx, key, value); err != nil {
			return fmt.Errorf("failed to set key %s: %w", key, err)
		}
	}
	return nil
}

// Clear removes all keys matching a pattern
func (c *Cache) Clear(ctx context.Context, pattern string) error {
	// If cache name is specified, prefix the pattern
	if c.opts.CacheName != "" {
		pattern = c.opts.CacheName + "::" + pattern
	}

	keys, err := c.client.Keys(ctx, pattern)
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return c.client.Delete(ctx, keys...)
	}

	return nil
}

// GetTTL returns the time to live of a key
func (c *Cache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	fullKey := c.buildCacheKey(key)
	return c.client.TTL(ctx, fullKey)
}

// ExtendTTL extends the TTL of a key
func (c *Cache) ExtendTTL(ctx context.Context, key string, ttl time.Duration) error {
	fullKey := c.buildCacheKey(key)
	return c.client.Expire(ctx, fullKey, ttl)
}

// ClearCacheName removes all keys belonging to the cache name
func (c *Cache) ClearCacheName(ctx context.Context) error {
	if c.opts.CacheName == "" {
		return fmt.Errorf("cache name is not set")
	}

	// Use the cache name with wildcard to match all keys
	pattern := c.opts.CacheName + "::*"

	keys, err := c.client.Keys(ctx, pattern)
	if err != nil {
		return fmt.Errorf("failed to get keys for cache name %s: %w", c.opts.CacheName, err)
	}

	if len(keys) > 0 {
		if err := c.client.Delete(ctx, keys...); err != nil {
			return fmt.Errorf("failed to delete keys for cache name %s: %w", c.opts.CacheName, err)
		}
	}

	return nil
}
