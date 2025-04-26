package health

import "go-api/internal/domain/model"

type UseCase interface {
	CheckHealth() model.HealthResponse
}
