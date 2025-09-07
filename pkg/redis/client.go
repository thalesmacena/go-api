package redis

import (
	"context"
	"encoding/json"
	"fmt"
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

// Set stores a key-value pair with optional expiration
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.rdb.Set(ctx, key, value, expiration).Err()
}

// Get retrieves a value by key
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, key).Result()
}

// GetBytes retrieves a value by key as bytes
func (c *Client) GetBytes(ctx context.Context, key string) ([]byte, error) {
	return c.rdb.Get(ctx, key).Bytes()
}

// GetJSON retrieves a JSON value by key and unmarshals it into the provided interface
func (c *Client) GetJSON(ctx context.Context, key string, dest interface{}) error {
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
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

// Info returns information and statistics about the server
func (c *Client) Info(ctx context.Context, section ...string) (string, error) {
	return c.rdb.Info(ctx, section...).Result()
}

// Stats returns the client pool statistics
func (c *Client) Stats() *redis.PoolStats {
	return c.rdb.PoolStats()
}
