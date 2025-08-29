package controller

import (
	"go-api/internal/domain/model"
	"go-api/internal/domain/usecase/weather"
	"go-api/pkg/util/numberutils"
	"net/http"

	"github.com/labstack/echo/v4"
)

type WeatherController struct {
	api     *echo.Group
	useCase weather.UseCase
}

func NewWeatherController(api *echo.Group, useCase weather.UseCase) *WeatherController {
	return &WeatherController{api: api, useCase: useCase}
}

// InitWeatherRoutes initializes weather routes
func (controller *WeatherController) InitWeatherRoutes() {
	controller.api.GET("/weather", controller.FindAllCities)
	controller.api.GET("/weather/state/:state/city/:city", controller.FindCityByNameAndState)
	controller.api.GET("/weather/schedule", controller.UpdateAllCitiesMonitoring)
	controller.api.POST("/weather", controller.CreateCityMonitoring)
	controller.api.DELETE("/weather/state/:state/city/:city", controller.RemoveCityMonitoring)
}

// FindAllCities godoc
// @Summary Get all monitored cities
// @Description Retrieve all monitored cities with pagination and filtering options
// @Tags weather
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(0)
// @Param size query int false "Page size" default(10)
// @Param namePrefix query string false "City name prefix to filter by"
// @Param state query string false "State to filter by"
// @Param fromDate query string false "Date to filter from (YYYY-MM-DD)"
// @Success 200 {array} entity.City "Paginated list of cities"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /weather [get]
func (controller *WeatherController) FindAllCities(c echo.Context) error {
	var page int = numberutils.ToIntWithDefault(c.QueryParam("page"), 0)
	var size int = numberutils.ToIntWithDefault(c.QueryParam("size"), 10)
	var namePrefix string = c.QueryParam("namePrefix")
	var state string = c.QueryParam("state")
	var fromDate string = c.QueryParam("fromDate")

	cities, err := controller.useCase.FindAllCities(page, size, namePrefix, state, fromDate)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, cities)
}

// FindCityByNameAndState godoc
// @Summary Get city by name and state
// @Description Find a specific city by its name and state with weather data
// @Tags weather
// @Accept json
// @Produce json
// @Param city path string true "City name"
// @Param state path string true "State name"
// @Param fromDate query string false "Date to filter from (YYYY-MM-DD)"
// @Success 200 {object} entity.City "City data with weather information"
// @Failure 404 {object} map[string]string "City not found"
// @Router /weather/state/{state}/city/{city} [get]
func (controller *WeatherController) FindCityByNameAndState(c echo.Context) error {
	city := c.Param("city")
	state := c.Param("state")
	fromDate := c.QueryParam("fromDate")

	cityData, err := controller.useCase.FindCityByNameAndState(city, state, fromDate)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "City not found"})
	}
	return c.JSON(http.StatusOK, cityData)
}

// CreateCityMonitoring godoc
// @Summary Create city monitoring
// @Description Add a new city to the weather monitoring system
// @Tags weather
// @Accept json
// @Produce json
// @Param city body model.CreateCityMonitoringDTO true "City monitoring data"
// @Success 201 {object} map[string]string "City monitoring created successfully"
// @Failure 400 {object} map[string]string "Invalid request body or missing required fields"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /weather [post]
func (controller *WeatherController) CreateCityMonitoring(c echo.Context) error {
	var dto model.CreateCityMonitoringDTO
	if err := c.Bind(&dto); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if dto.CityName == "" || dto.State == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "cityName and state are required"})
	}

	err := controller.useCase.CreateCityMonitoring(dto.CityName, dto.State)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, map[string]string{"message": "City monitoring created successfully"})
}

// UpdateAllCitiesMonitoring godoc
// @Summary Schedule weather update for all cities
// @Description Schedule a weather monitoring update for all cities in the system
// @Tags weather
// @Accept json
// @Produce json
// @Success 202 {object} map[string]string "Cities monitoring update scheduled successfully"
// @Router /weather/schedule [get]
func (controller *WeatherController) UpdateAllCitiesMonitoring(c echo.Context) error {
	// Execute in a separate goroutine to avoid blocking the request
	go func() {
		controller.useCase.UpdateAllCitiesMonitoring()
	}()

	return c.JSON(http.StatusAccepted, map[string]string{"message": "Cities monitoring update scheduled successfully"})
}

// RemoveCityMonitoring godoc
// @Summary Remove city monitoring
// @Description Remove a city from the weather monitoring system
// @Tags weather
// @Accept json
// @Produce json
// @Param city path string true "City name"
// @Param state path string true "State name"
// @Success 204 "City monitoring removed successfully"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /weather/state/{state}/city/{city} [delete]
func (controller *WeatherController) RemoveCityMonitoring(c echo.Context) error {
	city := c.Param("city")
	state := c.Param("state")

	if err := controller.useCase.RemoveCityMonitoring(city, state); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}
