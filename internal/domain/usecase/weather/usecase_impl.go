package weather

import (
	"errors"
	"fmt"
	"go-api/internal/domain/entity"
	"go-api/internal/domain/gateway/api"
	"go-api/internal/domain/gateway/db"
	"go-api/internal/domain/gateway/queue"
	"go-api/internal/domain/model"
	"go-api/internal/domain/model/external"
	"go-api/pkg/log"
	"strconv"
	"sync"

	"go.uber.org/zap"
)

type weatherUseCase struct {
	queueName   string
	batchSize   int
	apiGateway  api.WeatherGateway
	dbGateway   db.CityGateway
	queueSender queue.Sender
}

func NewWeatherUseCase(queueName string, batchSize int, queueSender queue.Sender, apiGateway api.WeatherGateway, dbGateway db.CityGateway) UseCase {
	return &weatherUseCase{
		queueName:   queueName,
		batchSize:   batchSize,
		queueSender: queueSender,
		apiGateway:  apiGateway,
		dbGateway:   dbGateway,
	}
}

// FindAllCities returns a paginated list of cities with filters
func (uc *weatherUseCase) FindAllCities(page int, size int, namePrefix string, state string, fromDate string) (*model.Page[entity.City], error) {
	cities, totalElements, err := uc.fetchCitiesAndCountInParallel(page, size, namePrefix, state, fromDate)
	if err != nil {
		return nil, err
	}

	return model.NewPage(cities, page, size, totalElements), nil
}

// fetchCitiesAndCountInParallel fetches cities and count in parallel for pagination
func (uc *weatherUseCase) fetchCitiesAndCountInParallel(page int, size int, namePrefix string, state string, fromDate string) ([]entity.City, int64, error) {
	var wg sync.WaitGroup
	var cities []entity.City
	var totalElements int64
	var citiesErr, countErr error

	// Get cities with filters in parallel
	wg.Add(1)
	go func() {
		defer wg.Done()
		cities, citiesErr = uc.dbGateway.FindAllWithFilters(page, size, namePrefix, state, fromDate)
	}()

	// Get total count with same filters in parallel
	wg.Add(1)
	go func() {
		defer wg.Done()
		totalElements, countErr = uc.dbGateway.CountWithFilters(namePrefix, state, fromDate)
	}()

	// Wait for both operations to complete
	wg.Wait()

	// Check for errors
	if citiesErr != nil {
		return nil, 0, fmt.Errorf("failed to find cities with filters: %w", citiesErr)
	}
	if countErr != nil {
		return nil, 0, fmt.Errorf("failed to count cities with filters: %w", countErr)
	}

	return cities, totalElements, nil
}

// FindCityByNameAndState searches for a single city by name, state and optional date
func (uc *weatherUseCase) FindCityByNameAndState(name string, state string, fromDate string) (*entity.City, error) {
	if name == "" || state == "" {
		return nil, errors.New("name and state are required")
	}

	city, err := uc.dbGateway.FindByNameAndState(name, state, fromDate)
	if err != nil {
		return nil, fmt.Errorf("failed to find city by name and state: %w", err)
	}

	if city == nil {
		return nil, errors.New("city not found")
	}

	return city, nil
}

// CreateCityMonitoring searches for a city in the API, saves it and enqueues it
func (uc *weatherUseCase) CreateCityMonitoring(cityName string, state string) error {
	if cityName == "" || state == "" {
		return errors.New("cityName and state are required")
	}

	// Search cities in the API
	searchResults, err := uc.apiGateway.SearchCities(cityName)
	if err != nil {
		return fmt.Errorf("failed to search cities in API: %w", err)
	}

	// Filter by state and get first result
	var selectedCity *entity.City
	for _, result := range searchResults {
		if result.Estado == state {
			selectedCity = &entity.City{
				Name:  result.Nome,
				Code:  strconv.Itoa(result.ID),
				State: result.Estado,
			}
			break
		}
	}

	if selectedCity == nil {
		return fmt.Errorf("no city found for name '%s' and state '%s'", cityName, state)
	}

	// Save city to database
	savedCity, err := uc.dbGateway.Create(*selectedCity)
	if err != nil {
		return fmt.Errorf("failed to save city to database: %w", err)
	}

	// Enqueue the saved city
	err = uc.queueSender.SendMessage(uc.queueName, savedCity)
	if err != nil {
		return fmt.Errorf("failed to enqueue saved city: %w", err)
	}

	log.Infof("City '%s' from state '%s' saved and enqueued successfully", savedCity.Name, savedCity.State)
	return nil
}

// UpdateAllCitiesMonitoring enqueues all cities in batches using pagination
func (uc *weatherUseCase) UpdateAllCitiesMonitoring() {
	page := 0

	for {
		// Get cities for current page
		cities, err := uc.dbGateway.FindAll(page, uc.batchSize)
		if err != nil {
			log.Warnf("Failed to fetch cities for page %d: %v", page, err)
			break
		}

		// If no cities found, we're done
		if len(cities) == 0 {
			break
		}

		// Prepare batch messages
		messages := make([]queue.BatchMessage, len(cities))
		for i, city := range cities {
			messages[i] = queue.BatchMessage{
				MessageID: fmt.Sprintf("city-%s-%d", city.ID, page),
				Body:      city,
			}
		}

		// Send batch
		result, err := uc.queueSender.SendMessageBatch(uc.queueName, messages)
		if err != nil {
			log.Warnf("Failed to send batch for page %d: %v", page, err)
			// Log all cities in this batch as failed
			for _, city := range cities {
				log.Warnf("Failed to enqueue city: %s (ID: %s, State: %s)", city.Name, city.ID, city.State)
			}
		} else {
			// Log individual failed cities
			for _, failedID := range result.Failed {
				for _, city := range cities {
					if fmt.Sprintf("city-%s-%d", city.ID, page) == failedID {
						log.Warnf("Failed to enqueue city: %s (ID: %s, State: %s)", city.Name, city.ID, city.State)
						break
					}
				}
			}
			log.Infof("Successfully enqueued %d cities, failed %d cities for page %d",
				len(result.Successful), len(result.Failed), page)
		}

		page++
	}

	log.Infof("Completed batch enqueuing all cities. Total pages processed: %d", page)
}

// UpdateAllCitiesMonitoringScheduled enqueues all cities for update monitoring
func (uc *weatherUseCase) UpdateAllCitiesMonitoringScheduled(requestID string) error {
	log.Info("Starting scheduled city monitoring update with key-set pagination", zap.String("request_id", requestID))

	var lastID string
	totalProcessed := 0
	totalEnqueued := 0
	totalFailed := 0

	for {
		// Get cities using key-set pagination
		cities, err := uc.dbGateway.FindAllWithKeysetPagination(lastID, uc.batchSize)
		if err != nil {
			log.Error("Failed to fetch cities with key-set pagination",
				zap.String("request_id", requestID),
				zap.String("last_id", lastID),
				zap.Error(err))
			return fmt.Errorf("failed to fetch cities with key-set pagination (lastID: %s): %w", lastID, err)
		}

		// If no cities found, we're done
		if len(cities) == 0 {
			log.Info("No more cities to process", zap.String("request_id", requestID))
			break
		}

		totalProcessed += len(cities)
		log.Info("Processing batch",
			zap.String("request_id", requestID),
			zap.Int("batch_size", len(cities)),
			zap.String("last_id", lastID))

		// Prepare batch messages
		messages := make([]queue.BatchMessage, len(cities))
		for i, city := range cities {
			messages[i] = queue.BatchMessage{
				MessageID: fmt.Sprintf("scheduled-%s-city-%s", requestID, city.ID),
				Body:      city,
			}
		}

		// Send batch
		result, err := uc.queueSender.SendMessageBatch(uc.queueName, messages)
		if err != nil {
			log.Warn("Failed to send batch",
				zap.String("request_id", requestID),
				zap.String("starting_city_id", lastID),
				zap.Error(err))
			// Log all cities in this batch as failed
			for _, city := range cities {
				log.Warn("Failed to enqueue city",
					zap.String("request_id", requestID),
					zap.String("city_name", city.Name),
					zap.String("city_id", city.ID),
					zap.String("state", city.State))
			}
			totalFailed += len(cities)
		} else {
			// Log individual failed cities
			for _, failedID := range result.Failed {
				for _, city := range cities {
					if fmt.Sprintf("scheduled-%s-city-%s", requestID, city.ID) == failedID {
						log.Warn("Failed to enqueue city",
							zap.String("request_id", requestID),
							zap.String("city_name", city.Name),
							zap.String("city_id", city.ID),
							zap.String("state", city.State))
						totalFailed++
						break
					}
				}
			}
			totalEnqueued += len(result.Successful)
			log.Info("Batch processed",
				zap.String("request_id", requestID),
				zap.Int("enqueued", len(result.Successful)),
				zap.Int("failed", len(result.Failed)))
		}

		// Update lastID for next iteration (use the last city's ID from this batch)
		lastID = cities[len(cities)-1].ID
	}

	log.Info("Completed scheduled city monitoring update",
		zap.String("request_id", requestID),
		zap.Int("total_processed", totalProcessed),
		zap.Int("total_enqueued", totalEnqueued),
		zap.Int("total_failed", totalFailed))
	return nil
}

// UpdateCityMonitoring updates weather and wave conditions for a city in parallel
func (uc *weatherUseCase) UpdateCityMonitoring(city entity.City) error {
	if city.Code == "" {
		return errors.New("city code is required")
	}

	cityCode, err := strconv.Atoi(city.Code)
	if err != nil {
		return fmt.Errorf("invalid city code '%s': %w", city.Code, err)
	}

	weatherErr, waveErr := uc.updateWeatherAndWaveInParallel(city, cityCode)

	// Weather is mandatory, wave conditions are optional
	if weatherErr != nil {
		return fmt.Errorf("weather update failed: %w", weatherErr)
	}

	// Wave conditions are optional - log warning but don't fail
	if waveErr != nil {
		log.Warnf("Wave conditions not available for city %s (likely inland city): %v", city.Name, waveErr)
		log.Infof("Successfully updated weather conditions for city: %s (wave conditions not available)", city.Name)
	} else {
		log.Infof("Successfully updated weather and wave conditions for city: %s", city.Name)
	}

	return nil
}

// updateWeatherAndWaveInParallel updates weather and wave conditions in parallel
func (uc *weatherUseCase) updateWeatherAndWaveInParallel(city entity.City, cityCode int) (error, error) {
	var wg sync.WaitGroup
	var weatherErr, waveErr error

	// Update weather conditions in parallel
	wg.Add(1)
	go func() {
		defer wg.Done()
		weatherErr = uc.updateWeatherConditions(city, cityCode)
	}()

	// Update wave conditions in parallel
	wg.Add(1)
	go func() {
		defer wg.Done()
		waveErr = uc.updateWaveConditions(city, cityCode)
	}()

	// Wait for both operations to complete
	wg.Wait()

	return weatherErr, waveErr
}

// updateWeatherConditions fetches and updates weather conditions for a city
func (uc *weatherUseCase) updateWeatherConditions(city entity.City, cityCode int) error {
	// Get weather forecast for 6 days
	weatherResponse, err := uc.apiGateway.GetWeatherForecast(cityCode, 6)
	if err != nil {
		return fmt.Errorf("failed to get weather forecast: %w", err)
	}

	// Convert API response to entities
	weatherForecasts := uc.convertWeatherResponse(weatherResponse, city.ID)

	// Upsert weather forecasts
	_, err = uc.dbGateway.UpsertWeatherForecasts(city.ID, weatherForecasts)
	if err != nil {
		return fmt.Errorf("failed to upsert weather forecasts: %w", err)
	}

	return nil
}

// convertWeatherResponse converts weather API response to entity list
func (uc *weatherUseCase) convertWeatherResponse(response *external.WeatherForecastResponse, cityID string) []entity.WeatherForecast {
	var weatherForecasts []entity.WeatherForecast

	for _, weatherData := range response.Clima {
		forecast := entity.WeatherForecast{
			Day:                  weatherData.Data,
			Condition:            weatherData.Condicao,
			ConditionDescription: weatherData.CondicaoDesc,
			Min:                  weatherData.Min,
			Max:                  weatherData.Max,
			UltraVioletIndex:     weatherData.IndiceUV,
			CityID:               cityID,
		}
		weatherForecasts = append(weatherForecasts, forecast)
	}

	return weatherForecasts
}

// updateWaveConditions fetches and updates wave conditions for a city
func (uc *weatherUseCase) updateWaveConditions(city entity.City, cityCode int) error {
	// Get wave conditions for 6 days
	waveResponse, err := uc.apiGateway.GetWaveConditions(cityCode, 6)
	if err != nil {
		// Wave conditions may not be available for inland cities
		return fmt.Errorf("wave conditions not available for city %s (state: %s): %w", city.Name, city.State, err)
	}

	// Check if response has valid wave data
	if waveResponse == nil || len(waveResponse.Ondas) == 0 {
		return fmt.Errorf("no wave data available for city %s (state: %s)", city.Name, city.State)
	}

	// Convert API response to entities
	waveConditions := uc.convertWaveResponse(waveResponse, city)

	// If no valid wave conditions were converted, don't proceed
	if len(waveConditions) == 0 {
		return fmt.Errorf("no valid wave conditions found for city %s (state: %s)", city.Name, city.State)
	}

	// Upsert wave conditions
	_, err = uc.dbGateway.UpsertWaveConditions(city.ID, waveConditions)
	if err != nil {
		return fmt.Errorf("failed to upsert wave conditions for city %s: %w", city.Name, err)
	}

	return nil
}

// convertWaveResponse converts wave API response to entity list
func (uc *weatherUseCase) convertWaveResponse(response *external.WaveConditionResponse, city entity.City) []entity.WaveCondition {
	var waveConditions []entity.WaveCondition

	for _, dayData := range response.Ondas {
		for _, waveData := range dayData.DadosOndas {
			// Extract hour from format like "00h Z", "03h Z"
			hour := uc.extractHourFromString(waveData.Hora)
			if hour == -1 {
				log.Warnf("Invalid hour format '%s' for city %s on %s", waveData.Hora, city.Name, dayData.Data)
				continue
			}

			condition := entity.WaveCondition{
				Day:                      dayData.Data,
				Wind:                     waveData.Vento,
				WindDirection:            waveData.DirecaoVento,
				WindDirectionDescription: waveData.DirecaoVentoDesc,
				WaveHeight:               waveData.AlturaOnda,
				WaveDirection:            waveData.DirecaoOnda,
				WaveDirectionDescription: waveData.DirecaoOndaDesc,
				Agitation:                waveData.Agitacao,
				Hour:                     hour,
				CityID:                   city.ID,
			}
			waveConditions = append(waveConditions, condition)
		}
	}

	return waveConditions
}

// extractHourFromString extracts the first numbers from a string like "00h Z", "03h Z"
func (uc *weatherUseCase) extractHourFromString(hourStr string) int {
	var numStr string

	// Extract only the first consecutive numbers
	for _, char := range hourStr {
		if char >= '0' && char <= '9' {
			numStr += string(char)
		} else if numStr != "" {
			// Stop at first non-digit after we found some digits
			break
		}
	}

	if numStr == "" {
		return -1
	}

	hour, err := strconv.Atoi(numStr)
	if err != nil {
		return -1
	}

	return hour
}

// RemoveCityMonitoring deletes a city and all its related weather and wave conditions
func (uc *weatherUseCase) RemoveCityMonitoring(name string, state string) error {
	if name == "" || state == "" {
		return errors.New("name and state are required")
	}

	// Find the city first
	city, err := uc.dbGateway.FindByNameAndState(name, state, "")
	if err != nil {
		return fmt.Errorf("failed to find city: %w", err)
	}

	if city == nil {
		return fmt.Errorf("city with name '%s' and state '%s' not found", name, state)
	}

	// Delete all weather forecasts for this city
	err = uc.dbGateway.DeleteWeatherForecastsByCityID(city.ID)
	if err != nil {
		return fmt.Errorf("failed to delete weather forecasts: %w", err)
	}

	// Delete all wave conditions for this city (optional - may not exist for inland cities)
	err = uc.dbGateway.DeleteWaveConditionsByCityID(city.ID)
	if err != nil {
		log.Warnf("Failed to delete wave conditions for city %s (may not have wave data): %v", city.Name, err)
	}

	// Delete the city itself
	err = uc.dbGateway.DeleteByNameAndState(name, state)
	if err != nil {
		return fmt.Errorf("failed to delete city: %w", err)
	}

	log.Infof("Successfully deleted city '%s' from state '%s' and all related data", name, state)
	return nil
}
