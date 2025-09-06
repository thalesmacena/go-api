package redis

// HealthStatus represents the health status
type HealthStatus string

const (
	// StatusUp indicates the service is healthy and running
	StatusUp HealthStatus = "UP"
	// StatusDown indicates the service is not healthy or not running
	StatusDown HealthStatus = "DOWN"
	// StatusUnknown indicates the service status cannot be determined
	StatusUnknown HealthStatus = "UNKNOWN"
)
