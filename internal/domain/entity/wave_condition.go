package entity

type WaveCondition struct {
	ID                       string  `json:"id"`
	Day                      string  `json:"day"`
	CreatedAt                string  `json:"createdDate"`
	UpdatedAt                string  `json:"updatedDate"`
	Wind                     float64 `json:"wind"`
	WindDirection            string  `json:"windDirection"`
	WindDirectionDescription string  `json:"windDirectionDescription"`
	WaveHeight               float64 `json:"waveHeight"`
	WaveDirection            string  `json:"waveDirection"`
	WaveDirectionDescription string  `json:"waveDirectionDescription"`
	Agitation                string  `json:"agitation"`
	Hour                     int     `json:"hour"`
	CityID                   string  `json:"cityId"`
}
