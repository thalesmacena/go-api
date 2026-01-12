package weather

import (
	"go-api/internal/domain/entity"
	"go-api/internal/domain/model"
)

type UseCase interface {
	// FindAllCities returns a paginated list of cities with filters
	FindAllCities(page int, size int, namePrefix string, state string, fromDate string) (*model.Page[entity.City], error)

	// FindCityByNameAndState searches for a single city by name, state and optional date
	FindCityByNameAndState(name string, state string, fromDate string) (*entity.City, error)

	// CreateCityMonitoring searches for a city in the API, saves it and enqueues it
	CreateCityMonitoring(cityName string, state string) error

	// UpdateAllCitiesMonitoring enqueues all cities in batches using pagination
	UpdateAllCitiesMonitoring()

	// UpdateAllCitiesMonitoringScheduled enqueues all cities for update monitoring
	UpdateAllCitiesMonitoringScheduled(requestID string) error

	// UpdateCityMonitoring updates weather and wave conditions for a city in parallel
	UpdateCityMonitoring(city entity.City) error

	// RemoveCityMonitoring deletes a city and all its related weather and wave conditions
	RemoveCityMonitoring(name string, state string) error
}
