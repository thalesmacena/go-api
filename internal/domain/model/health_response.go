package model

// HealthStatus represents the possible health status values
type HealthStatus string

const (
	StatusUp      HealthStatus = "UP"
	StatusDown    HealthStatus = "DOWN"
	StatusUnknown HealthStatus = "UNKNOWN"
)

// ComponentHealthStatus represents the health check structure of a application component
type ComponentHealthStatus struct {
	Status  HealthStatus      `json:"status"`
	Details map[string]string `json:"details"`
}

// HealthResponse represents the health check response of all application
type HealthResponse struct {
	Status   HealthStatus          `json:"status"`
	Database ComponentHealthStatus `json:"database"`
	Queue    ComponentHealthStatus `json:"queue"`
}
