package api

import (
	"go-api/internal/domain/model/external"
)

// WeatherGateway defines the interface for weather-related external API calls
type WeatherGateway interface {
	// SearchCities searches for cities by name
	// Returns a list of cities matching the search criteria
	SearchCities(cityName string) ([]external.CitySearchResponse, error)

	// GetWeatherForecast gets weather forecast for a city
	// cityCode: the city code from the search API
	// days: number of days (1-6)
	GetWeatherForecast(cityCode int, days int) (*external.WeatherForecastResponse, error)

	// GetWaveConditions gets wave conditions for a city
	// cityCode: the city code from the search API
	// days: number of days (1-6)
	GetWaveConditions(cityCode int, days int) (*external.WaveConditionResponse, error)
}
