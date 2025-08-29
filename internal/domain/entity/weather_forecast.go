package entity

type WeatherForecast struct {
	ID                   string `json:"id"`
	Day                  string `json:"day"`
	CreatedAt            string `json:"createdDate"`
	UpdatedAt            string `json:"updatedDate"`
	Condition            string `json:"condition"`
	ConditionDescription string `json:"conditionDescription"`
	Min                  int    `json:"min"`
	Max                  int    `json:"max"`
	UltraVioletIndex     int    `json:"ultraVioletIndex"`
	CityID               string `json:"cityId"`
}
