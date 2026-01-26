package db

import (
	"go-api/internal/domain/model"
	"gorm.io/gorm"
)

type GormHealthDBGateway struct {
	DB *gorm.DB
}

var _ HealthDBGateway = (*GormHealthDBGateway)(nil)

func NewGormHealthDBGateway(db *gorm.DB) *GormHealthDBGateway {
	return &GormHealthDBGateway{DB: db}
}

func (gateway *GormHealthDBGateway) Health() model.ComponentHealthStatus {
	sqlDB, err := gateway.DB.DB()

	if err != nil {
		return model.ComponentHealthStatus{
			Status: model.StatusDown,
			Details: map[string]string{
				"message": err.Error(),
			},
		}
	}

	err = sqlDB.Ping()
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
