package entity

type City struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Code             string            `json:"code"`
	State            string            `json:"state"`
	CreatedAt        string            `json:"createdDate"`
	UpdatedAt        string            `json:"updatedDate"`
	WeatherForecasts []WeatherForecast `json:"weatherForecasts"`
	WaveConditions   []WaveCondition   `json:"waveConditions"`
}
