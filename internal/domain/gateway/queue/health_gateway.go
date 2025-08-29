package queue

import (
	"go-api/internal/domain/model"
	"go-api/pkg/sqs"
)

type HealthGateway interface {
	Health() model.ComponentHealthStatus
	RegisterWorker(name string, worker *sqs.Worker)
	UnregisterWorker(name string)
}
