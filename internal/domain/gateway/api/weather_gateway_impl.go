package api

import (
	"fmt"
	"go-api/internal/domain/model/external"
	"go-api/pkg/http"
)

// weatherGatewayImpl implements the WeatherGateway interface
type weatherGatewayImpl struct {
	httpClient *http.Client
}

// NewWeatherGateway creates a new instance of WeatherGateway with HTTP client
func NewWeatherGateway(baseUrl string, clientOptions http.ClientOptions) WeatherGateway {
	httpClient := http.NewHttpClient(baseUrl, clientOptions)

	return &weatherGatewayImpl{
		httpClient: httpClient,
	}
}

// SearchCities searches for cities by name
func (w *weatherGatewayImpl) SearchCities(cityName string) ([]external.CitySearchResponse, error) {
	path := fmt.Sprintf("/cptec/v1/cidade/%s", cityName)

	successResp, errResp, _, err := w.httpClient.Request().
		WithMethod(http.GET).
		WithPath(path).
		WithSuccessResp(&[]external.CitySearchResponse{}).
		WithErrorResp(&external.APIErrorResponse{}).
		Execute()

	if err == nil {
		response := successResp.(*[]external.CitySearchResponse)
		return *response, nil
	}

	if errResp != nil {
		errorResponse := errResp.(*external.APIErrorResponse)
		return nil, fmt.Errorf(errorResponse.Message)
	}

	return nil, err
}

// GetWeatherForecast gets weather forecast for a city
func (w *weatherGatewayImpl) GetWeatherForecast(cityCode int, days int) (*external.WeatherForecastResponse, error) {
	path := fmt.Sprintf("/cptec/v1/clima/previsao/%d/%d", cityCode, days)

	successResponse, errResp, _, err := w.httpClient.Request().
		WithMethod(http.GET).
		WithPath(path).
		WithSuccessResp(&external.WeatherForecastResponse{}).
		WithErrorResp(&external.APIErrorResponse{}).
		Execute()

	if err == nil {
		response := successResponse.(*external.WeatherForecastResponse)
		return response, nil
	}

	if errResp != nil {
		errorResponse := errResp.(*external.APIErrorResponse)
		return nil, fmt.Errorf(errorResponse.Message)
	}

	return nil, err
}

// GetWaveConditions gets wave conditions for a city
func (w *weatherGatewayImpl) GetWaveConditions(cityCode int, days int) (*external.WaveConditionResponse, error) {
	path := fmt.Sprintf("/cptec/v1/ondas/%d/%d", cityCode, days)

	successResponse, errResp, _, err := w.httpClient.Request().
		WithMethod(http.GET).
		WithPath(path).
		WithSuccessResp(&external.WaveConditionResponse{}).
		WithErrorResp(&external.APIErrorResponse{}).
		Execute()

	if err == nil {
		response := successResponse.(*external.WaveConditionResponse)
		return response, nil
	}

	if errResp != nil {
		errorResponse := errResp.(*external.APIErrorResponse)
		return nil, fmt.Errorf(errorResponse.Message)
	}

	return nil, err
}
