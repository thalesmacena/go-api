package db

import "go-api/internal/domain/model"

type HealthDBGateway interface {
	Health() model.ComponentHealthStatus
}
