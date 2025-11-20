package processor

import (
	"encoding/json"
	"fmt"
	"go-api/internal/domain/entity"
	"go-api/internal/domain/usecase/weather"
	"go-api/pkg/log"

	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type WeatherProcessor struct {
	weatherUseCase weather.UseCase
}

func NewWeatherProcessor(weatherUseCase weather.UseCase) *WeatherProcessor {
	return &WeatherProcessor{
		weatherUseCase: weatherUseCase,
	}
}

// HandleMessage implements the sqs.Handler interface
func (p *WeatherProcessor) HandleMessage(msg *types.Message) error {
	if msg == nil || msg.Body == nil {
		return fmt.Errorf("received nil message or message body")
	}

	log.Infof("Processing weather message: %s", *msg.MessageId)

	// Parse the message body as a City entity
	var city entity.City
	if err := json.Unmarshal([]byte(*msg.Body), &city); err != nil {
		return fmt.Errorf("failed to unmarshal message body: %w", err)
	}

	// Update city monitoring using the weather use case
	if err := p.weatherUseCase.UpdateCityMonitoring(city); err != nil {
		return fmt.Errorf("failed to update city monitoring for %s: %w", city.Name, err)
	}

	log.Infof("Successfully processed weather update for city: %s", city.Name)
	return nil
}
