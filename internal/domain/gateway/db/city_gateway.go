package db

import (
	"go-api/internal/domain/entity"
)

type CityGateway interface {
	// City CRUD operations
	FindAll(page int, size int) ([]entity.City, error)
	FindAllWithFilters(page int, size int, namePrefix string, state string, fromDate string) ([]entity.City, error)
	FindAllWithKeysetPagination(lastID string, size int) ([]entity.City, error)
	CountAll() (int64, error)
	CountWithFilters(namePrefix string, state string, fromDate string) (int64, error)
	FindByID(id string) (*entity.City, error)
	FindByNameAndState(name string, state string, fromDate string) (*entity.City, error)

	Create(city entity.City) (*entity.City, error)
	UpdateByID(id string, updated entity.City) (*entity.City, error)
	UpdateByName(name string, state string, updated entity.City) (*entity.City, error)
	DeleteByID(id string) error
	DeleteByNameAndState(name string, state string) error

	// Weather forecast operations
	CreateWeatherForecast(cityID string, weather entity.WeatherForecast) (*entity.WeatherForecast, error)
	UpdateWeatherForecast(weatherID string, updated entity.WeatherForecast) (*entity.WeatherForecast, error)
	DeleteWeatherForecast(weatherID string) error
	DeleteWeatherForecastsByCityID(cityID string) error

	// Wave condition operations
	CreateWaveCondition(cityID string, wave entity.WaveCondition) (*entity.WaveCondition, error)
	UpdateWaveCondition(waveID string, updated entity.WaveCondition) (*entity.WaveCondition, error)
	DeleteWaveCondition(waveID string) error
	DeleteWaveConditionsByCityID(cityID string) error

	// Upsert operations (insert or update based on unique constraints)
	UpsertWeatherForecast(cityID string, weather entity.WeatherForecast) (*entity.WeatherForecast, error)
	UpsertWaveCondition(cityID string, wave entity.WaveCondition) (*entity.WaveCondition, error)

	// Batch upsert operations for lists
	UpsertWeatherForecasts(cityID string, forecasts []entity.WeatherForecast) ([]entity.WeatherForecast, error)
	UpsertWaveConditions(cityID string, conditions []entity.WaveCondition) ([]entity.WaveCondition, error)
}
