package model

type CreateCityMonitoringDTO struct {
	CityName string `json:"cityName" validate:"required"`
	State    string `json:"state" validate:"required"`
}
