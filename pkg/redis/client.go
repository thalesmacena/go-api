package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps the Redis client with additional functionality
type Client struct {
	rdb    *redis.Client
	config *Config
}

// NewClient creates a new Redis client with the given configuration
func NewClient(config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		panic(fmt.Sprintf("invalid Redis configuration: %v", err))
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:           fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password:       config.Password,
		DB:             config.Database,
		MinIdleConns:   config.MinIdleConns,
		MaxIdleConns:   config.MaxIdleConns,
		MaxActiveConns: config.MaxActive,
		MaxRetries:     config.MaxRetries,
		DialTimeout:    config.DialTimeout,
		ReadTimeout:    config.ReadTimeout,
		WriteTimeout:   config.WriteTimeout,
		PoolTimeout:    config.PoolTimeout,
	})

	return &Client{
		rdb:    rdb,
		config: config,
	}
}

// Ping tests the connection to Redis
func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// Close closes the Redis client connection
func (c *Client) Close() error {
	return c.rdb.Close()
}

// GetClient returns the underlying Redis client for advanced operations
func (c *Client) GetClient() *redis.Client {
	return c.rdb
}

// GetConfig returns the Redis configuration
func (c *Client) GetConfig() *Config {
	return c.config
}

// Set stores a key-value pair with optional expiration
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.rdb.Set(ctx, key, value, expiration).Err()
}

// Get retrieves a value by key
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	result, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		// Check if it's a "key not found" error (redis.Nil) or a real error
		if errors.Is(err, redis.Nil) {
			// Key doesn't exist - return empty string (this is not an error)
			return "", nil
		}
		// Real error (connection, etc.) - return it
		return "", err
	}
	return result, nil
}

// GetBytes retrieves a value by key as bytes
func (c *Client) GetBytes(ctx context.Context, key string) ([]byte, error) {
	result, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		// Check if it's a "key not found" error (redis.Nil) or a real error
		if errors.Is(err, redis.Nil) {
			// Key doesn't exist - return empty bytes (this is not an error)
			return []byte{}, nil
		}
		// Real error (connection, etc.) - return it
		return nil, err
	}
	return result, nil
}

// GetJSON retrieves a JSON value by key and unmarshals it into the provided interface
func (c *Client) GetJSON(ctx context.Context, key string, dest interface{}) error {
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		// Check if it's a "key not found" error (redis.Nil) or a real error
		if errors.Is(err, redis.Nil) {
			// Key doesn't exist - this is not an error, just return nil
			return nil
		}
		// Real error (connection, etc.) - return it
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

// SetJSON marshals the provided value to JSON and stores it with optional expiration
func (c *Client) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value to JSON: %w", err)
	}
	return c.rdb.Set(ctx, key, jsonData, expiration).Err()
}

// SetInt stores an integer value with optional expiration
func (c *Client) SetInt(ctx context.Context, key string, value int64, expiration time.Duration) error {
	return c.rdb.Set(ctx, key, strconv.FormatInt(value, 10), expiration).Err()
}

// GetInt retrieves an integer value by key
func (c *Client) GetInt(ctx context.Context, key string) (int64, error) {
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		// Check if it's a "key not found" error (redis.Nil) or a real error
		if errors.Is(err, redis.Nil) {
			// Key doesn't exist - return 0 (this is not an error)
			return 0, nil
		}
		// Real error (connection, etc.) - return it
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

// SetFloat stores a float64 value with optional expiration
func (c *Client) SetFloat(ctx context.Context, key string, value float64, expiration time.Duration) error {
	return c.rdb.Set(ctx, key, strconv.FormatFloat(value, 'f', -1, 64), expiration).Err()
}

// GetFloat retrieves a float64 value by key
func (c *Client) GetFloat(ctx context.Context, key string) (float64, error) {
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		// Check if it's a "key not found" error (redis.Nil) or a real error
		if errors.Is(err, redis.Nil) {
			// Key doesn't exist - return 0 (this is not an error)
			return 0, nil
		}
		// Real error (connection, etc.) - return it
		return 0, err
	}
	return strconv.ParseFloat(val, 64)
}

// SetBool stores a boolean value with optional expiration
func (c *Client) SetBool(ctx context.Context, key string, value bool, expiration time.Duration) error {
	return c.rdb.Set(ctx, key, strconv.FormatBool(value), expiration).Err()
}

// GetBool retrieves a boolean value by key
func (c *Client) GetBool(ctx context.Context, key string) (bool, error) {
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		// Check if it's a "key not found" error (redis.Nil) or a real error
		if errors.Is(err, redis.Nil) {
			// Key doesn't exist - return false (this is not an error)
			return false, nil
		}
		// Real error (connection, etc.) - return it
		return false, err
	}
	return strconv.ParseBool(val)
}

// Delete removes one or more keys
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

// Exists checks if one or more keys exist
func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	return c.rdb.Exists(ctx, keys...).Result()
}

// Expire sets an expiration on a key
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.rdb.Expire(ctx, key, expiration).Err()
}

// TTL returns the time to live of a key
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.rdb.TTL(ctx, key).Result()
}

// Keys returns all keys matching a pattern
func (c *Client) Keys(ctx context.Context, pattern string) ([]string, error) {
	return c.rdb.Keys(ctx, pattern).Result()
}

// Scan iterates over keys matching a pattern
func (c *Client) Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	return c.rdb.Scan(ctx, cursor, match, count).Result()
}

// HSet sets field in the hash stored at key to value
func (c *Client) HSet(ctx context.Context, key string, values ...interface{}) error {
	return c.rdb.HSet(ctx, key, values...).Err()
}

// HGet returns the value associated with field in the hash stored at key
func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	return c.rdb.HGet(ctx, key, field).Result()
}

// HGetAll returns all fields and values of the hash stored at key
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.rdb.HGetAll(ctx, key).Result()
}

// HDel removes one or more fields from a hash
func (c *Client) HDel(ctx context.Context, key string, fields ...string) error {
	return c.rdb.HDel(ctx, key, fields...).Err()
}

// HExists checks if a field exists in a hash
func (c *Client) HExists(ctx context.Context, key, field string) (bool, error) {
	return c.rdb.HExists(ctx, key, field).Result()
}

// LPush prepends one or more values to a list
func (c *Client) LPush(ctx context.Context, key string, values ...interface{}) error {
	return c.rdb.LPush(ctx, key, values...).Err()
}

// RPush appends one or more values to a list
func (c *Client) RPush(ctx context.Context, key string, values ...interface{}) error {
	return c.rdb.RPush(ctx, key, values...).Err()
}

// LPop removes and returns the first element of a list
func (c *Client) LPop(ctx context.Context, key string) (string, error) {
	return c.rdb.LPop(ctx, key).Result()
}

// RPop removes and returns the last element of a list
func (c *Client) RPop(ctx context.Context, key string) (string, error) {
	return c.rdb.RPop(ctx, key).Result()
}

// LLen returns the length of a list
func (c *Client) LLen(ctx context.Context, key string) (int64, error) {
	return c.rdb.LLen(ctx, key).Result()
}

// LRange returns a range of elements from a list
func (c *Client) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.rdb.LRange(ctx, key, start, stop).Result()
}

// SAdd adds one or more members to a set
func (c *Client) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return c.rdb.SAdd(ctx, key, members...).Err()
}

// SMembers returns all members of a set
func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	return c.rdb.SMembers(ctx, key).Result()
}

// SRem removes one or more members from a set
func (c *Client) SRem(ctx context.Context, key string, members ...interface{}) error {
	return c.rdb.SRem(ctx, key, members...).Err()
}

// SIsMember checks if a value is a member of a set
func (c *Client) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return c.rdb.SIsMember(ctx, key, member).Result()
}

// ZAdd adds one or more members to a sorted set
func (c *Client) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	return c.rdb.ZAdd(ctx, key, members...).Err()
}

// ZRange returns a range of members in a sorted set
func (c *Client) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.rdb.ZRange(ctx, key, start, stop).Result()
}

// ZRangeWithScores returns a range of members with scores in a sorted set
func (c *Client) ZRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	return c.rdb.ZRangeWithScores(ctx, key, start, stop).Result()
}

// ZRem removes one or more members from a sorted set
func (c *Client) ZRem(ctx context.Context, key string, members ...interface{}) error {
	return c.rdb.ZRem(ctx, key, members...).Err()
}

// Incr increments the integer value of a key by 1
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	return c.rdb.Incr(ctx, key).Result()
}

// IncrBy increments the integer value of a key by the given amount
func (c *Client) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.rdb.IncrBy(ctx, key, value).Result()
}

// Decr decrements the integer value of a key by 1
func (c *Client) Decr(ctx context.Context, key string) (int64, error) {
	return c.rdb.Decr(ctx, key).Result()
}

// DecrBy decrements the integer value of a key by the given amount
func (c *Client) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.rdb.DecrBy(ctx, key, value).Result()
}

// Pipeline creates a new pipeline for batch operations
func (c *Client) Pipeline() redis.Pipeliner {
	return c.rdb.Pipeline()
}

// TxPipeline creates a new transaction pipeline
func (c *Client) TxPipeline() redis.Pipeliner {
	return c.rdb.TxPipeline()
}

// Watch watches the given keys for modifications during a transaction
func (c *Client) Watch(ctx context.Context, fn func(*redis.Tx) error, keys ...string) error {
	return c.rdb.Watch(ctx, fn, keys...)
}

// Publish publishes a message to a channel
func (c *Client) Publish(ctx context.Context, channel string, message interface{}) error {
	return c.rdb.Publish(ctx, channel, message).Err()
}

// Subscribe subscribes to one or more channels
func (c *Client) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return c.rdb.Subscribe(ctx, channels...)
}

// PSubscribe subscribes to one or more patterns
func (c *Client) PSubscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return c.rdb.PSubscribe(ctx, channels...)
}

// FlushDB removes all keys from the current database
func (c *Client) FlushDB(ctx context.Context) error {
	return c.rdb.FlushDB(ctx).Err()
}

// FlushAll removes all keys from all databases
func (c *Client) FlushAll(ctx context.Context) error {
	return c.rdb.FlushAll(ctx).Err()
}

// FlushDBAsync removes all keys from the current database asynchronously
func (c *Client) FlushDBAsync(ctx context.Context) error {
	return c.rdb.FlushDBAsync(ctx).Err()
}

// FlushAllAsync removes all keys from all databases asynchronously
func (c *Client) FlushAllAsync(ctx context.Context) error {
	return c.rdb.FlushAllAsync(ctx).Err()
}

// FlushDBWithFallback removes all keys from the current database, with fallback to manual deletion
func (c *Client) FlushDBWithFallback(ctx context.Context) error {
	// Try the native FlushDB command first
	err := c.FlushDB(ctx)
	if err == nil {
		return nil
	}

	// If FlushDB fails, fall back to manual deletion
	fmt.Printf("FlushDB command failed (%v), falling back to manual key deletion...\n", err)
	return c.flushDBManual(ctx)
}

// FlushAllWithFallback removes all keys from all databases, with fallback to manual deletion
func (c *Client) FlushAllWithFallback(ctx context.Context) error {
	// Try the native FlushAll command first
	err := c.FlushAll(ctx)
	if err == nil {
		return nil
	}

	// If FlushAll fails, fall back to manual deletion
	fmt.Printf("FlushAll command failed (%v), falling back to manual key deletion...\n", err)
	return c.flushAllManual(ctx)
}

// flushDBManual manually deletes all keys from the current database
func (c *Client) flushDBManual(ctx context.Context) error {
	// Get all keys
	keys, err := c.Keys(ctx, "*")
	if err != nil {
		return fmt.Errorf("failed to get keys: %w", err)
	}

	if len(keys) == 0 {
		return nil // No keys to delete
	}

	// Delete all keys
	return c.Delete(ctx, keys...)
}

// flushAllManual manually deletes all keys from all databases
func (c *Client) flushAllManual(ctx context.Context) error {
	// For manual flush all, we can only flush the current database
	// since we can't easily switch databases with the go-redis client
	fmt.Println("Note: Manual FlushAll only clears the current database")
	return c.flushDBManual(ctx)
}

// Info returns information and statistics about the server
func (c *Client) Info(ctx context.Context, section ...string) (string, error) {
	return c.rdb.Info(ctx, section...).Result()
}

// Stats returns the client pool statistics
func (c *Client) Stats() *redis.PoolStats {
	return c.rdb.PoolStats()
}

// GetDBSize returns the number of keys in the current database
func (c *Client) GetDBSize(ctx context.Context) (int64, error) {
	return c.rdb.DBSize(ctx).Result()
}

// GetInfo returns Redis server information
func (c *Client) GetInfo(ctx context.Context, section ...string) (string, error) {
	return c.rdb.Info(ctx, section...).Result()
}

// CheckFlushCommandsAvailability checks if flush commands are available
func (c *Client) CheckFlushCommandsAvailability(ctx context.Context) (map[string]bool, error) {
	result := make(map[string]bool)

	// Test FlushDB
	err := c.FlushDB(ctx)
	result["FlushDB"] = err == nil

	// Test FlushAll
	err = c.FlushAll(ctx)
	result["FlushAll"] = err == nil

	return result, nil
}
