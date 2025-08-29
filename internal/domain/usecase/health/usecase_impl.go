package health

import (
	"go-api/internal/domain/gateway/db"
	"go-api/internal/domain/gateway/queue"
	"go-api/internal/domain/model"
)

type healthUseCase struct {
	dbGateway    db.HealthDBGateway
	queueGateway queue.HealthGateway
}

func NewHealthUseCase(dbGateway db.HealthDBGateway, queueGateway queue.HealthGateway) UseCase {
	return &healthUseCase{
		dbGateway:    dbGateway,
		queueGateway: queueGateway,
	}
}

func (useCase *healthUseCase) CheckHealth() model.HealthResponse {
	dbHealth := useCase.dbGateway.Health()
	queueHealth := useCase.queueGateway.Health()

	overallStatus := model.StatusUp
	if dbHealth.Status != model.StatusUp || queueHealth.Status != model.StatusUp {
		overallStatus = model.StatusDown
	}

	return model.HealthResponse{
		Status:   overallStatus,
		Database: dbHealth,
		Queue:    queueHealth,
	}
}
