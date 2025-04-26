package health

import (
	"go-api/internal/domain/gateway/db"
	"go-api/internal/domain/model"
)

type healthUseCase struct {
	dbGateway db.HealthDBGateway
}

func NewHealthUseCase(dbGateway db.HealthDBGateway) UseCase {
	return &healthUseCase{
		dbGateway: dbGateway,
	}
}

func (useCase *healthUseCase) CheckHealth() model.HealthResponse {
	dbHealth := useCase.dbGateway.Health()

	overallStatus := model.StatusUp
	if dbHealth.Status != model.StatusUp {
		overallStatus = model.StatusDown
	}

	return model.HealthResponse{
		Status:   overallStatus,
		Database: dbHealth,
	}
}
