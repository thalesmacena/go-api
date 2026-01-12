package db

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"go-api/internal/domain/entity"

	"github.com/google/uuid"
)

const cityTimeLayout = "2006-01-02 15:04:05"

type SQLCCityGateway struct {
	DB *sql.DB
}

func NewSQLCCityGateway(db *sql.DB) *SQLCCityGateway {
	return &SQLCCityGateway{DB: db}
}

// FindAll retrieves all cities with pagination
func (gateway *SQLCCityGateway) FindAll(page int, size int) ([]entity.City, error) {
	return gateway.FindAllWithFilters(page, size, "", "", "")
}

// FindAllWithKeysetPagination retrieves cities using key-set pagination by ID
func (gateway *SQLCCityGateway) FindAllWithKeysetPagination(lastID string, size int) ([]entity.City, error) {
	query := `
		SELECT c.id, c.name, c.code, c.state, c.created_at, c.updated_at
		FROM cities c
		WHERE 1=1`

	args := []interface{}{}
	argCount := 0

	if lastID != "" {
		argCount++
		query += fmt.Sprintf(" AND c.id > $%d", argCount)
		args = append(args, lastID)
	}

	query += " ORDER BY c.id ASC"

	argCount++
	query += fmt.Sprintf(" LIMIT $%d", argCount)
	args = append(args, size)

	rows, err := gateway.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cities := make([]entity.City, 0)
	for rows.Next() {
		var city entity.City
		if err := rows.Scan(&city.ID, &city.Name, &city.Code, &city.State, &city.CreatedAt, &city.UpdatedAt); err != nil {
			return nil, err
		}

		var weatherErr, waveErr error
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			weatherErr = gateway.loadWeatherForecasts(&city, "")
		}()

		go func() {
			defer wg.Done()
			waveErr = gateway.loadWaveConditions(&city, "")
		}()

		wg.Wait()

		if weatherErr != nil {
			return nil, weatherErr
		}
		if waveErr != nil {
			return nil, waveErr
		}

		cities = append(cities, city)
	}

	return cities, nil
}

// FindAllWithFilters retrieves cities with filters and pagination
func (gateway *SQLCCityGateway) FindAllWithFilters(page int, size int, namePrefix string, state string, fromDate string) ([]entity.City, error) {
	// Ensure page is not negative (0-based pagination)
	if page < 0 {
		page = 0
	}
	offset := page * size

	// Build base query
	query := `
		SELECT c.id, c.name, c.code, c.state, c.created_at, c.updated_at
		FROM cities c
		WHERE 1=1`

	args := []interface{}{}
	argCount := 0

	// Add filters
	if namePrefix != "" {
		argCount++
		query += fmt.Sprintf(" AND c.name ILIKE $%d", argCount)
		args = append(args, namePrefix+"%")
	}

	if state != "" {
		argCount++
		query += fmt.Sprintf(" AND c.state = $%d", argCount)
		args = append(args, state)
	}

	// Add pagination
	argCount++
	query += fmt.Sprintf(" ORDER BY c.created_at DESC OFFSET $%d", argCount)
	args = append(args, offset)

	argCount++
	query += fmt.Sprintf(" LIMIT $%d", argCount)
	args = append(args, size)

	rows, err := gateway.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cities := make([]entity.City, 0)
	for rows.Next() {
		var city entity.City
		if err := rows.Scan(&city.ID, &city.Name, &city.Code, &city.State, &city.CreatedAt, &city.UpdatedAt); err != nil {
			return nil, err
		}

		// Load weather forecasts and wave conditions in parallel
		var weatherErr, waveErr error
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			weatherErr = gateway.loadWeatherForecasts(&city, fromDate)
		}()

		go func() {
			defer wg.Done()
			waveErr = gateway.loadWaveConditions(&city, fromDate)
		}()

		wg.Wait()

		if weatherErr != nil {
			return nil, weatherErr
		}
		if waveErr != nil {
			return nil, waveErr
		}

		cities = append(cities, city)
	}

	return cities, nil
}

// CountAll returns total count of cities
func (gateway *SQLCCityGateway) CountAll() (int64, error) {
	return gateway.CountWithFilters("", "", "")
}

// CountWithFilters returns count of cities with filters
func (gateway *SQLCCityGateway) CountWithFilters(namePrefix string, state string, fromDate string) (int64, error) {
	query := "SELECT COUNT(*) FROM cities c WHERE 1=1"
	args := []interface{}{}
	argCount := 0

	if namePrefix != "" {
		argCount++
		query += fmt.Sprintf(" AND c.name ILIKE $%d", argCount)
		args = append(args, namePrefix+"%")
	}

	if state != "" {
		argCount++
		query += fmt.Sprintf(" AND c.state = $%d", argCount)
		args = append(args, state)
	}

	var count int64
	err := gateway.DB.QueryRow(query, args...).Scan(&count)
	return count, err
}

// FindByID finds a city by ID
func (gateway *SQLCCityGateway) FindByID(id string) (*entity.City, error) {
	var city entity.City
	err := gateway.DB.QueryRow(`
		SELECT id, name, code, state, created_at, updated_at
		FROM cities
		WHERE id = $1`, id).
		Scan(&city.ID, &city.Name, &city.Code, &city.State, &city.CreatedAt, &city.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Load weather forecasts and wave conditions in parallel
	var weatherErr, waveErr error
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		weatherErr = gateway.loadWeatherForecasts(&city, "")
	}()

	go func() {
		defer wg.Done()
		waveErr = gateway.loadWaveConditions(&city, "")
	}()

	wg.Wait()

	if weatherErr != nil {
		return nil, weatherErr
	}
	if waveErr != nil {
		return nil, waveErr
	}

	return &city, nil
}

// FindByNameAndState finds a city by name and state
func (gateway *SQLCCityGateway) FindByNameAndState(name string, state string, fromDate string) (*entity.City, error) {
	var city entity.City
	err := gateway.DB.QueryRow(`
		SELECT id, name, code, state, created_at, updated_at
		FROM cities
		WHERE name = $1 AND state = $2`, name, state).
		Scan(&city.ID, &city.Name, &city.Code, &city.State, &city.CreatedAt, &city.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Load weather forecasts and wave conditions in parallel
	var weatherErr, waveErr error
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		weatherErr = gateway.loadWeatherForecasts(&city, fromDate)
	}()

	go func() {
		defer wg.Done()
		waveErr = gateway.loadWaveConditions(&city, fromDate)
	}()

	wg.Wait()

	if weatherErr != nil {
		return nil, weatherErr
	}
	if waveErr != nil {
		return nil, waveErr
	}

	return &city, nil
}

// Create creates a new city
func (gateway *SQLCCityGateway) Create(city entity.City) (*entity.City, error) {
	city.ID = uuid.New().String()
	now := time.Now().UTC().Format(cityTimeLayout)
	city.CreatedAt = now
	city.UpdatedAt = now

	_, err := gateway.DB.Exec(`
		INSERT INTO cities (id, name, code, state, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		city.ID, city.Name, city.Code, city.State, city.CreatedAt, city.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &city, nil
}

// UpdateByID updates a city by ID
func (gateway *SQLCCityGateway) UpdateByID(id string, updated entity.City) (*entity.City, error) {
	updated.UpdatedAt = time.Now().UTC().Format(cityTimeLayout)

	_, err := gateway.DB.Exec(`
		UPDATE cities
		SET name = $1, code = $2, state = $3, updated_at = $4
		WHERE id = $5`,
		updated.Name, updated.Code, updated.State, updated.UpdatedAt, id)
	if err != nil {
		return nil, err
	}

	updated.ID = id
	return &updated, nil
}

// UpdateByName updates a city by name and state
func (gateway *SQLCCityGateway) UpdateByName(name string, state string, updated entity.City) (*entity.City, error) {
	updated.UpdatedAt = time.Now().UTC().Format(cityTimeLayout)

	_, err := gateway.DB.Exec(`
		UPDATE cities
		SET name = $1, code = $2, state = $3, updated_at = $4
		WHERE name = $5 AND state = $6`,
		updated.Name, updated.Code, updated.State, updated.UpdatedAt, name, state)
	if err != nil {
		return nil, err
	}

	return &updated, nil
}

// DeleteByID deletes a city by ID
func (gateway *SQLCCityGateway) DeleteByID(id string) error {
	// Delete related weather forecasts and wave conditions first
	if err := gateway.DeleteWeatherForecastsByCityID(id); err != nil {
		return err
	}
	if err := gateway.DeleteWaveConditionsByCityID(id); err != nil {
		return err
	}

	_, err := gateway.DB.Exec(`DELETE FROM cities WHERE id = $1`, id)
	return err
}

// DeleteByNameAndState deletes a city by name and state
func (gateway *SQLCCityGateway) DeleteByNameAndState(name string, state string) error {
	// First get the city ID
	var cityID string
	err := gateway.DB.QueryRow(`SELECT id FROM cities WHERE name = $1 AND state = $2`, name, state).Scan(&cityID)
	if err != nil {
		return err
	}

	// Delete related data and city
	return gateway.DeleteByID(cityID)
}

// Weather Forecast operations

// CreateWeatherForecast creates a weather forecast for a city
func (gateway *SQLCCityGateway) CreateWeatherForecast(cityID string, weather entity.WeatherForecast) (*entity.WeatherForecast, error) {
	weather.ID = uuid.New().String()
	weather.CityID = cityID
	now := time.Now().UTC().Format(cityTimeLayout)
	weather.CreatedAt = now
	weather.UpdatedAt = now

	_, err := gateway.DB.Exec(`
		INSERT INTO weather_forecasts (id, day, condition, condition_description, min, max, uv_index, city_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		weather.ID, weather.Day, weather.Condition, weather.ConditionDescription,
		weather.Min, weather.Max, weather.UltraVioletIndex, weather.CityID,
		weather.CreatedAt, weather.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &weather, nil
}

// UpdateWeatherForecast updates a weather forecast
func (gateway *SQLCCityGateway) UpdateWeatherForecast(weatherID string, updated entity.WeatherForecast) (*entity.WeatherForecast, error) {
	updated.UpdatedAt = time.Now().UTC().Format(cityTimeLayout)

	_, err := gateway.DB.Exec(`
		UPDATE weather_forecasts
		SET day = $1, condition = $2, condition_description = $3, min = $4, max = $5, uv_index = $6, updated_at = $7
		WHERE id = $8`,
		updated.Day, updated.Condition, updated.ConditionDescription,
		updated.Min, updated.Max, updated.UltraVioletIndex, updated.UpdatedAt, weatherID)
	if err != nil {
		return nil, err
	}

	updated.ID = weatherID
	return &updated, nil
}

// DeleteWeatherForecast deletes a weather forecast
func (gateway *SQLCCityGateway) DeleteWeatherForecast(weatherID string) error {
	_, err := gateway.DB.Exec(`DELETE FROM weather_forecasts WHERE id = $1`, weatherID)
	return err
}

// DeleteWeatherForecastsByCityID deletes all weather forecasts for a city
func (gateway *SQLCCityGateway) DeleteWeatherForecastsByCityID(cityID string) error {
	_, err := gateway.DB.Exec(`DELETE FROM weather_forecasts WHERE city_id = $1`, cityID)
	return err
}

// Wave Condition operations

// CreateWaveCondition creates a wave condition for a city
func (gateway *SQLCCityGateway) CreateWaveCondition(cityID string, wave entity.WaveCondition) (*entity.WaveCondition, error) {
	wave.ID = uuid.New().String()
	wave.CityID = cityID
	now := time.Now().UTC().Format(cityTimeLayout)
	wave.CreatedAt = now
	wave.UpdatedAt = now

	_, err := gateway.DB.Exec(`
		INSERT INTO wave_conditions (id, day, wind, wind_direction, wind_direction_desc, wave_height, wave_direction, wave_direction_desc, agitation, hour, city_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		wave.ID, wave.Day, wave.Wind, wave.WindDirection, wave.WindDirectionDescription,
		wave.WaveHeight, wave.WaveDirection, wave.WaveDirectionDescription,
		wave.Agitation, wave.Hour, wave.CityID, wave.CreatedAt, wave.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &wave, nil
}

// UpdateWaveCondition updates a wave condition
func (gateway *SQLCCityGateway) UpdateWaveCondition(waveID string, updated entity.WaveCondition) (*entity.WaveCondition, error) {
	updated.UpdatedAt = time.Now().UTC().Format(cityTimeLayout)

	_, err := gateway.DB.Exec(`
		UPDATE wave_conditions
		SET day = $1, wind = $2, wind_direction = $3, wind_direction_desc = $4, wave_height = $5, wave_direction = $6, wave_direction_desc = $7, agitation = $8, hour = $9, updated_at = $10
		WHERE id = $11`,
		updated.Day, updated.Wind, updated.WindDirection, updated.WindDirectionDescription,
		updated.WaveHeight, updated.WaveDirection, updated.WaveDirectionDescription,
		updated.Agitation, updated.Hour, updated.UpdatedAt, waveID)
	if err != nil {
		return nil, err
	}

	updated.ID = waveID
	return &updated, nil
}

// DeleteWaveCondition deletes a wave condition
func (gateway *SQLCCityGateway) DeleteWaveCondition(waveID string) error {
	_, err := gateway.DB.Exec(`DELETE FROM wave_conditions WHERE id = $1`, waveID)
	return err
}

// DeleteWaveConditionsByCityID deletes all wave conditions for a city
func (gateway *SQLCCityGateway) DeleteWaveConditionsByCityID(cityID string) error {
	_, err := gateway.DB.Exec(`DELETE FROM wave_conditions WHERE city_id = $1`, cityID)
	return err
}

// Helper functions

// loadWeatherForecasts loads weather forecasts for a city
func (gateway *SQLCCityGateway) loadWeatherForecasts(city *entity.City, fromDate string) error {
	query := `
		SELECT id, day, condition, condition_description, min, max, uv_index, city_id, created_at, updated_at
		FROM weather_forecasts
		WHERE city_id = $1`

	args := []interface{}{city.ID}

	if fromDate != "" {
		query += " AND day >= $2"
		args = append(args, fromDate)
	} else {
		// Default to today if no date specified
		today := time.Now().Format("2006-01-02")
		query += " AND day >= $2"
		args = append(args, today)
	}

	query += " ORDER BY day ASC"

	rows, err := gateway.DB.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	forecasts := make([]entity.WeatherForecast, 0)
	for rows.Next() {
		var forecast entity.WeatherForecast
		if err := rows.Scan(&forecast.ID, &forecast.Day, &forecast.Condition, &forecast.ConditionDescription,
			&forecast.Min, &forecast.Max, &forecast.UltraVioletIndex, &forecast.CityID,
			&forecast.CreatedAt, &forecast.UpdatedAt); err != nil {
			return err
		}
		forecasts = append(forecasts, forecast)
	}

	city.WeatherForecasts = forecasts
	return nil
}

// loadWaveConditions loads wave conditions for a city
func (gateway *SQLCCityGateway) loadWaveConditions(city *entity.City, fromDate string) error {
	query := `
		SELECT id, day, wind, wind_direction, wind_direction_desc, wave_height, wave_direction, wave_direction_desc, agitation, hour, city_id, created_at, updated_at
		FROM wave_conditions
		WHERE city_id = $1`

	args := []interface{}{city.ID}

	if fromDate != "" {
		query += " AND day >= $2"
		args = append(args, fromDate)
	} else {
		// Default to today if no date specified
		today := time.Now().Format("2006-01-02")
		query += " AND day >= $2"
		args = append(args, today)
	}

	query += " ORDER BY day ASC, hour ASC"

	rows, err := gateway.DB.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	conditions := make([]entity.WaveCondition, 0)
	for rows.Next() {
		var condition entity.WaveCondition
		if err := rows.Scan(&condition.ID, &condition.Day, &condition.Wind, &condition.WindDirection,
			&condition.WindDirectionDescription, &condition.WaveHeight, &condition.WaveDirection,
			&condition.WaveDirectionDescription, &condition.Agitation, &condition.Hour,
			&condition.CityID, &condition.CreatedAt, &condition.UpdatedAt); err != nil {
			return err
		}
		conditions = append(conditions, condition)
	}

	city.WaveConditions = conditions
	return nil
}

// Upsert operations

// UpsertWeatherForecast inserts or updates a weather forecast based on city_id + day
func (gateway *SQLCCityGateway) UpsertWeatherForecast(cityID string, weather entity.WeatherForecast) (*entity.WeatherForecast, error) {
	// First, try to find existing weather forecast for this city and day
	var existingID string
	err := gateway.DB.QueryRow(`
		SELECT id FROM weather_forecasts 
		WHERE city_id = $1 AND day = $2`, cityID, weather.Day).Scan(&existingID)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// If exists, update it
	if existingID != "" {
		return gateway.UpdateWeatherForecast(existingID, weather)
	}

	// If doesn't exist, create new one
	return gateway.CreateWeatherForecast(cityID, weather)
}

// UpsertWaveCondition inserts or updates a wave condition based on city_id + day + hour
func (gateway *SQLCCityGateway) UpsertWaveCondition(cityID string, wave entity.WaveCondition) (*entity.WaveCondition, error) {
	// First, try to find existing wave condition for this city, day and hour
	var existingID string
	err := gateway.DB.QueryRow(`
		SELECT id FROM wave_conditions 
		WHERE city_id = $1 AND day = $2 AND hour = $3`, cityID, wave.Day, wave.Hour).Scan(&existingID)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// If exists, update it
	if existingID != "" {
		return gateway.UpdateWaveCondition(existingID, wave)
	}

	// If doesn't exist, create new one
	return gateway.CreateWaveCondition(cityID, wave)
}

// UpsertWeatherForecasts batch upserts a list of weather forecasts using transaction for better performance
func (gateway *SQLCCityGateway) UpsertWeatherForecasts(cityID string, forecasts []entity.WeatherForecast) ([]entity.WeatherForecast, error) {
	if len(forecasts) == 0 {
		return []entity.WeatherForecast{}, nil
	}

	// Start transaction for batch operation
	tx, err := gateway.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	results := make([]entity.WeatherForecast, 0, len(forecasts))

	for _, forecast := range forecasts {
		result, err := gateway.upsertWeatherForecastInTx(tx, cityID, forecast)
		if err != nil {
			return nil, err
		}
		results = append(results, *result)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return results, nil
}

// UpsertWaveConditions batch upserts a list of wave conditions using transaction for better performance
func (gateway *SQLCCityGateway) UpsertWaveConditions(cityID string, conditions []entity.WaveCondition) ([]entity.WaveCondition, error) {
	if len(conditions) == 0 {
		return []entity.WaveCondition{}, nil
	}

	// Start transaction for batch operation
	tx, err := gateway.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	results := make([]entity.WaveCondition, 0, len(conditions))

	for _, condition := range conditions {
		result, err := gateway.upsertWaveConditionInTx(tx, cityID, condition)
		if err != nil {
			return nil, err
		}
		results = append(results, *result)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return results, nil
}

// Helper functions for transaction-based upserts

// upsertWeatherForecastInTx performs upsert within a transaction
func (gateway *SQLCCityGateway) upsertWeatherForecastInTx(tx *sql.Tx, cityID string, weather entity.WeatherForecast) (*entity.WeatherForecast, error) {
	// Check if exists
	var existingID string
	err := tx.QueryRow(`
		SELECT id FROM weather_forecasts 
		WHERE city_id = $1 AND day = $2`, cityID, weather.Day).Scan(&existingID)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	now := time.Now().UTC().Format(cityTimeLayout)

	// If exists, update
	if existingID != "" {
		weather.ID = existingID
		weather.CityID = cityID
		weather.UpdatedAt = now

		_, err := tx.Exec(`
			UPDATE weather_forecasts
			SET day = $1, condition = $2, condition_description = $3, min = $4, max = $5, uv_index = $6, updated_at = $7
			WHERE id = $8`,
			weather.Day, weather.Condition, weather.ConditionDescription,
			weather.Min, weather.Max, weather.UltraVioletIndex, weather.UpdatedAt, existingID)
		if err != nil {
			return nil, err
		}

		return &weather, nil
	}

	// If doesn't exist, create
	weather.ID = uuid.New().String()
	weather.CityID = cityID
	weather.CreatedAt = now
	weather.UpdatedAt = now

	_, err = tx.Exec(`
		INSERT INTO weather_forecasts (id, day, condition, condition_description, min, max, uv_index, city_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		weather.ID, weather.Day, weather.Condition, weather.ConditionDescription,
		weather.Min, weather.Max, weather.UltraVioletIndex, weather.CityID,
		weather.CreatedAt, weather.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &weather, nil
}

// upsertWaveConditionInTx performs upsert within a transaction
func (gateway *SQLCCityGateway) upsertWaveConditionInTx(tx *sql.Tx, cityID string, wave entity.WaveCondition) (*entity.WaveCondition, error) {
	// Check if exists
	var existingID string
	err := tx.QueryRow(`
		SELECT id FROM wave_conditions 
		WHERE city_id = $1 AND day = $2 AND hour = $3`, cityID, wave.Day, wave.Hour).Scan(&existingID)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	now := time.Now().UTC().Format(cityTimeLayout)

	// If exists, update
	if existingID != "" {
		wave.ID = existingID
		wave.CityID = cityID
		wave.UpdatedAt = now

		_, err := tx.Exec(`
			UPDATE wave_conditions
			SET day = $1, wind = $2, wind_direction = $3, wind_direction_desc = $4, wave_height = $5, wave_direction = $6, wave_direction_desc = $7, agitation = $8, hour = $9, updated_at = $10
			WHERE id = $11`,
			wave.Day, wave.Wind, wave.WindDirection, wave.WindDirectionDescription,
			wave.WaveHeight, wave.WaveDirection, wave.WaveDirectionDescription,
			wave.Agitation, wave.Hour, wave.UpdatedAt, existingID)
		if err != nil {
			return nil, err
		}

		return &wave, nil
	}

	// If doesn't exist, create
	wave.ID = uuid.New().String()
	wave.CityID = cityID
	wave.CreatedAt = now
	wave.UpdatedAt = now

	_, err = tx.Exec(`
		INSERT INTO wave_conditions (id, day, wind, wind_direction, wind_direction_desc, wave_height, wave_direction, wave_direction_desc, agitation, hour, city_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		wave.ID, wave.Day, wave.Wind, wave.WindDirection, wave.WindDirectionDescription,
		wave.WaveHeight, wave.WaveDirection, wave.WaveDirectionDescription,
		wave.Agitation, wave.Hour, wave.CityID, wave.CreatedAt, wave.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &wave, nil
}
