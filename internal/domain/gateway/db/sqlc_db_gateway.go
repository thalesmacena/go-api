package db

import (
	"context"
	"database/sql"
	"go-api/internal/domain/model"
	"time"
)

type SQLCHealthDBGateway struct {
	DB *sql.DB
}

var _ HealthDBGateway = (*SQLCHealthDBGateway)(nil)

func NewSQLCHealthDBGateway(db *sql.DB) *SQLCHealthDBGateway {
	return &SQLCHealthDBGateway{DB: db}
}

func (gateway *SQLCHealthDBGateway) Health() model.ComponentHealthStatus {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := gateway.DB.PingContext(ctx)

	if err != nil {
		return model.ComponentHealthStatus{
			Status: model.StatusDown,
			Details: map[string]string{
				"message": err.Error(),
			},
		}
	}

	return model.ComponentHealthStatus{
		Status: model.StatusUp,
		Details: map[string]string{
			"message": string(model.StatusUp),
		},
	}
}
