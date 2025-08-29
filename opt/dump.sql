-- PostgreSQL DDL Script for GO-API Database
-- Schema: go

-- Create the schema
CREATE SCHEMA IF NOT EXISTS go;

-- Set search_path to the go schema
SET search_path TO go;

-- Create cities table
CREATE TABLE IF NOT EXISTS cities (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) NOT NULL,
    state VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create weather_forecasts table
CREATE TABLE IF NOT EXISTS weather_forecasts (
    id VARCHAR(36) PRIMARY KEY,
    day DATE NOT NULL,
    condition VARCHAR(100) NOT NULL,
    condition_description TEXT,
    min INTEGER NOT NULL,
    max INTEGER NOT NULL,
    uv_index INTEGER NOT NULL DEFAULT 0,
    city_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_weather_forecasts_city_id FOREIGN KEY (city_id) REFERENCES cities(id) ON DELETE CASCADE
);

-- Create wave_conditions table
CREATE TABLE IF NOT EXISTS wave_conditions (
    id VARCHAR(36) PRIMARY KEY,
    day DATE NOT NULL,
    wind DECIMAL(5,2) NOT NULL DEFAULT 0.0,
    wind_direction VARCHAR(10) NOT NULL,
    wind_direction_desc VARCHAR(50),
    wave_height DECIMAL(5,2) NOT NULL DEFAULT 0.0,
    wave_direction VARCHAR(10) NOT NULL,
    wave_direction_desc VARCHAR(50),
    agitation VARCHAR(50) NOT NULL,
    hour INTEGER NOT NULL CHECK (hour >= 0 AND hour <= 23),
    city_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_wave_conditions_city_id FOREIGN KEY (city_id) REFERENCES cities(id) ON DELETE CASCADE
);

-- Create short_urls table
CREATE TABLE IF NOT EXISTS short_urls (
    id VARCHAR(36) PRIMARY KEY,
    hash VARCHAR(255) UNIQUE NOT NULL,
    url TEXT NOT NULL,
    expiration TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance

-- Cities indexes
CREATE INDEX IF NOT EXISTS idx_cities_name ON cities(name);
CREATE INDEX IF NOT EXISTS idx_cities_state ON cities(state);
CREATE INDEX IF NOT EXISTS idx_cities_name_state ON cities(name, state);
CREATE INDEX IF NOT EXISTS idx_cities_created_at ON cities(created_at DESC);

-- Weather forecasts indexes
CREATE INDEX IF NOT EXISTS idx_weather_forecasts_city_id ON weather_forecasts(city_id);
CREATE INDEX IF NOT EXISTS idx_weather_forecasts_day ON weather_forecasts(day);
CREATE INDEX IF NOT EXISTS idx_weather_forecasts_city_day ON weather_forecasts(city_id, day);
CREATE INDEX IF NOT EXISTS idx_weather_forecasts_created_at ON weather_forecasts(created_at DESC);

-- Wave conditions indexes
CREATE INDEX IF NOT EXISTS idx_wave_conditions_city_id ON wave_conditions(city_id);
CREATE INDEX IF NOT EXISTS idx_wave_conditions_day ON wave_conditions(day);
CREATE INDEX IF NOT EXISTS idx_wave_conditions_hour ON wave_conditions(hour);
CREATE INDEX IF NOT EXISTS idx_wave_conditions_city_day_hour ON wave_conditions(city_id, day, hour);
CREATE INDEX IF NOT EXISTS idx_wave_conditions_created_at ON wave_conditions(created_at DESC);

-- Short URLs indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_short_urls_hash ON short_urls(hash);
CREATE INDEX IF NOT EXISTS idx_short_urls_expiration ON short_urls(expiration);
CREATE INDEX IF NOT EXISTS idx_short_urls_created_at ON short_urls(created_at DESC);

-- Enable trigram extension for better text search performance (if not already enabled)
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Add comments to tables for documentation
COMMENT ON TABLE cities IS 'Cities table storing city information for weather and wave data';
COMMENT ON TABLE weather_forecasts IS 'Weather forecast data for cities';
COMMENT ON TABLE wave_conditions IS 'Wave condition data for cities by day and hour';
COMMENT ON TABLE short_urls IS 'Short URL mappings with expiration dates';

-- Add comments to important columns
COMMENT ON COLUMN cities.code IS 'City code identifier';
COMMENT ON COLUMN cities.state IS 'State or region where the city is located';
COMMENT ON COLUMN weather_forecasts.day IS 'Date for the weather forecast';
COMMENT ON COLUMN weather_forecasts.uv_index IS 'UV index value for the day';
COMMENT ON COLUMN wave_conditions.day IS 'Date for the wave conditions';
COMMENT ON COLUMN wave_conditions.hour IS 'Hour of the day (0-23) for wave conditions';
COMMENT ON COLUMN short_urls.hash IS 'Unique hash identifier for the shortened URL';
COMMENT ON COLUMN short_urls.expiration IS 'Expiration timestamp for the short URL';
