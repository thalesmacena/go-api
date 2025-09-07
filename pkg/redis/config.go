package redis

import (
	"fmt"
	"time"
)

// Config represents Redis configuration options
type Config struct {
	// Host is the Redis server host
	Host string
	// Port is the Redis server port
	Port int
	// Password is the Redis server password
	Password string
	// Database is the Redis database number
	Database int
	// MinIdleConns is the minimum number of idle connections, idle (unused but open) connections
	MinIdleConns int
	// MaxIdleConns is the maximum number of idle (unused but open) connections to keep in the pool.
	MaxIdleConns int
	// MaxActive is the maximum number of active connections that can be established
	MaxActive int
	// MaxRetries is the maximum number of retries for failed commands
	MaxRetries int
	// DialTimeout is the timeout for establishing connections
	DialTimeout time.Duration
	// ReadTimeout is the timeout for socket reads
	ReadTimeout time.Duration
	// WriteTimeout is the timeout for socket writes
	WriteTimeout time.Duration
	// PoolTimeout is the timeout for getting connection from pool
	PoolTimeout time.Duration
	// CacheTTLs is a map of cache names to their TTL durations
	CacheTTLs map[string]time.Duration
	// DefaultCacheTTL is the default TTL for caches when not specified in CacheTTLs
	DefaultCacheTTL time.Duration
}

// NewRedisConfig creates a new Redis configuration with default values
func NewRedisConfig() *Config {
	return &Config{
		Host:            "localhost",
		Port:            6379,
		Password:        "",
		Database:        0,
		MinIdleConns:    5,
		MaxIdleConns:    10,
		MaxActive:       100,
		MaxRetries:      3,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolTimeout:     4 * time.Second,
		CacheTTLs:       make(map[string]time.Duration),
		DefaultCacheTTL: 1 * time.Hour,
	}
}

// WithHost sets the Redis server host
func (c *Config) WithHost(host string) *Config {
	c.Host = host
	return c
}

// WithPort sets the Redis server port
func (c *Config) WithPort(port int) *Config {
	if port < 1 || port > 65535 {
		panic(fmt.Sprintf("invalid port: %d, must be between 1 and 65535", port))
	}
	c.Port = port
	return c
}

// WithPassword sets the Redis server password
func (c *Config) WithPassword(password string) *Config {
	c.Password = password
	return c
}

// WithDatabase sets the Redis database number
func (c *Config) WithDatabase(database int) *Config {
	if database < 0 || database > 15 {
		panic(fmt.Sprintf("invalid database: %d, must be between 0 and 15", database))
	}
	c.Database = database
	return c
}

// WithMinIdleConns sets the minimum number of idle connections
func (c *Config) WithMinIdleConns(minIdleConns int) *Config {
	if minIdleConns < 0 {
		panic(fmt.Sprintf("invalid min idle connections: %d, must be non-negative", minIdleConns))
	}
	c.MinIdleConns = minIdleConns
	return c
}

// WithMaxIdleConns sets the maximum number of idle connections
func (c *Config) WithMaxIdleConns(maxIdleConns int) *Config {
	if maxIdleConns < 0 {
		panic(fmt.Sprintf("invalid max idle connections: %d, must be non-negative", maxIdleConns))
	}
	c.MaxIdleConns = maxIdleConns
	return c
}

// WithMaxActive sets the maximum number of active connections
func (c *Config) WithMaxActive(maxActive int) *Config {
	if maxActive < 0 {
		panic(fmt.Sprintf("invalid max active connections: %d, must be non-negative", maxActive))
	}
	c.MaxActive = maxActive
	return c
}

// WithMaxRetries sets the maximum number of retries for failed commands
func (c *Config) WithMaxRetries(maxRetries int) *Config {
	if maxRetries < 0 {
		panic(fmt.Sprintf("invalid max retries: %d, must be non-negative", maxRetries))
	}
	c.MaxRetries = maxRetries
	return c
}

// WithDialTimeout sets the timeout for establishing connections
func (c *Config) WithDialTimeout(dialTimeout time.Duration) *Config {
	if dialTimeout < 0 {
		panic(fmt.Sprintf("invalid dial timeout: %v, must be non-negative", dialTimeout))
	}
	c.DialTimeout = dialTimeout
	return c
}

// WithReadTimeout sets the timeout for socket reads
func (c *Config) WithReadTimeout(readTimeout time.Duration) *Config {
	if readTimeout < 0 {
		panic(fmt.Sprintf("invalid read timeout: %v, must be non-negative", readTimeout))
	}
	c.ReadTimeout = readTimeout
	return c
}

// WithWriteTimeout sets the timeout for socket writes
func (c *Config) WithWriteTimeout(writeTimeout time.Duration) *Config {
	if writeTimeout < 0 {
		panic(fmt.Sprintf("invalid write timeout: %v, must be non-negative", writeTimeout))
	}
	c.WriteTimeout = writeTimeout
	return c
}

// WithPoolTimeout sets the timeout for getting connection from pool
func (c *Config) WithPoolTimeout(poolTimeout time.Duration) *Config {
	if poolTimeout < 0 {
		panic(fmt.Sprintf("invalid pool timeout: %v, must be non-negative", poolTimeout))
	}
	c.PoolTimeout = poolTimeout
	return c
}

// WithCacheTTL sets the TTL for a specific cache name
func (c *Config) WithCacheTTL(cacheName string, ttl time.Duration) *Config {
	if ttl < 0 {
		panic(fmt.Sprintf("invalid cache TTL: %v, must be non-negative", ttl))
	}
	if c.CacheTTLs == nil {
		c.CacheTTLs = make(map[string]time.Duration)
	}
	c.CacheTTLs[cacheName] = ttl
	return c
}

// WithDefaultCacheTTL sets the default TTL for caches
func (c *Config) WithDefaultCacheTTL(defaultTTL time.Duration) *Config {
	if defaultTTL < 0 {
		panic(fmt.Sprintf("invalid default cache TTL: %v, must be non-negative", defaultTTL))
	}
	c.DefaultCacheTTL = defaultTTL
	return c
}

// DefaultConfig returns a default Redis configuration
func DefaultConfig() *Config {
	return NewRedisConfig()
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("host cannot be empty")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d, must be between 1 and 65535", c.Port)
	}
	if c.Database < 0 || c.Database > 15 {
		return fmt.Errorf("invalid database: %d, must be between 0 and 15", c.Database)
	}
	if c.MinIdleConns < 0 {
		return fmt.Errorf("invalid min idle connections: %d, must be non-negative", c.MinIdleConns)
	}
	if c.MaxIdleConns < 0 {
		return fmt.Errorf("invalid max idle connections: %d, must be non-negative", c.MaxIdleConns)
	}
	if c.MaxActive < 0 {
		return fmt.Errorf("invalid max active connections: %d, must be non-negative", c.MaxActive)
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("invalid max retries: %d, must be non-negative", c.MaxRetries)
	}
	if c.DialTimeout < 0 {
		return fmt.Errorf("invalid dial timeout: %v, must be non-negative", c.DialTimeout)
	}
	if c.ReadTimeout < 0 {
		return fmt.Errorf("invalid read timeout: %v, must be non-negative", c.ReadTimeout)
	}
	if c.WriteTimeout < 0 {
		return fmt.Errorf("invalid write timeout: %v, must be non-negative", c.WriteTimeout)
	}
	if c.PoolTimeout < 0 {
		return fmt.Errorf("invalid pool timeout: %v, must be non-negative", c.PoolTimeout)
	}
	return nil
}
