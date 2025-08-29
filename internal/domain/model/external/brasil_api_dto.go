package external

// CitySearchResponse represents the response from the city search API
type CitySearchResponse struct {
	Nome   string `json:"nome"`
	Estado string `json:"estado"`
	ID     int    `json:"id"`
}

// WeatherForecastResponse represents the response from the weather forecast API
type WeatherForecastResponse struct {
	Cidade       string                `json:"cidade"`
	Estado       string                `json:"estado"`
	AtualizadoEm string                `json:"atualizado_em"`
	Clima        []WeatherConditionDTO `json:"clima"`
}

// WeatherConditionDTO represents a single weather condition
type WeatherConditionDTO struct {
	Data         string `json:"data"`
	Condicao     string `json:"condicao"`
	Min          int    `json:"min"`
	Max          int    `json:"max"`
	IndiceUV     int    `json:"indice_uv"`
	CondicaoDesc string `json:"condicao_desc"`
}

// WaveConditionResponse represents the response from the wave condition API
type WaveConditionResponse struct {
	Cidade       string             `json:"cidade"`
	Estado       string             `json:"estado"`
	AtualizadoEm string             `json:"atualizado_em"`
	Ondas        []WaveDayCondition `json:"ondas"`
}

// WaveDayCondition represents wave conditions for a specific day
type WaveDayCondition struct {
	Data       string        `json:"data"`
	DadosOndas []WaveDataDTO `json:"dados_ondas"`
}

// WaveDataDTO represents wave data for a specific time
type WaveDataDTO struct {
	Vento            float64 `json:"vento"`
	DirecaoVento     string  `json:"direcao_vento"`
	DirecaoVentoDesc string  `json:"direcao_vento_desc"`
	AlturaOnda       float64 `json:"altura_onda"`
	DirecaoOnda      string  `json:"direcao_onda"`
	DirecaoOndaDesc  string  `json:"direcao_onda_desc"`
	Agitacao         string  `json:"agitacao"`
	Hora             string  `json:"hora"`
}

// APIErrorResponse represents error responses from the Brasil API
type APIErrorResponse struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}
