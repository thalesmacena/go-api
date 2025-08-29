package queue

import (
	"go-api/internal/domain/model"
	"go-api/pkg/sqs"
	"strconv"
	"sync"
)

type QueueHealthGateway struct {
	workers map[string]*sqs.Worker
	mutex   sync.RWMutex
}

func NewQueueHealthGateway() *QueueHealthGateway {
	return &QueueHealthGateway{
		workers: make(map[string]*sqs.Worker),
		mutex:   sync.RWMutex{},
	}
}

func (gateway *QueueHealthGateway) RegisterWorker(name string, worker *sqs.Worker) {
	gateway.mutex.Lock()
	defer gateway.mutex.Unlock()
	gateway.workers[name] = worker
}

func (gateway *QueueHealthGateway) UnregisterWorker(name string) {
	gateway.mutex.Lock()
	defer gateway.mutex.Unlock()
	delete(gateway.workers, name)
}

func (gateway *QueueHealthGateway) Health() model.ComponentHealthStatus {
	gateway.mutex.RLock()
	defer gateway.mutex.RUnlock()

	if len(gateway.workers) == 0 {
		return model.ComponentHealthStatus{
			Status: model.StatusUnknown,
			Details: map[string]string{
				"message":       "No workers registered",
				"workers_count": "0",
			},
		}
	}

	overallStatus := model.StatusUp
	details := make(map[string]string)
	workersUp := 0
	workersDown := 0

	for name, worker := range gateway.workers {
		workerHealth := worker.HealthCheck()

		if workerHealth.Status == sqs.StatusUp {
			workersUp++
			details[name+"_status"] = "UP"
		} else {
			workersDown++
			overallStatus = model.StatusDown
			details[name+"_status"] = "DOWN"
		}

		// Add worker details with prefix
		for key, value := range workerHealth.Details {
			details[name+"_"+key] = value
		}
	}

	details["workers_total"] = strconv.Itoa(len(gateway.workers))
	details["workers_up"] = strconv.Itoa(workersUp)
	details["workers_down"] = strconv.Itoa(workersDown)

	return model.ComponentHealthStatus{
		Status:  overallStatus,
		Details: details,
	}
}
