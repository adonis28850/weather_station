-- Migration 001: Initialize database schema
-- This migration creates the main tables for storing weather data

-- Create the raw readings table
-- This table stores every 5-second reading from the weather sensor
-- Data is retained for 30 days, then deleted by the retention function
CREATE TABLE IF NOT EXISTS readings (
    -- Primary key: auto-incrementing unique identifier
    id BIGSERIAL PRIMARY KEY,

    -- Sensor identifier (allows multiple sensors in the future)
    sensor_id INTEGER NOT NULL,

    -- Timestamp when the reading was captured (with timezone)
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Temperature in Celsius (e.g., 22.5)
    temperature_c REAL NOT NULL,

    -- Humidity percentage (0-100)
    humidity INTEGER NOT NULL,

    -- UV Index (0-12, optional field) - matches rtl_433 JSON field 'uv'
    uv REAL,

    -- Light intensity in lux (optional field) - matches rtl_433 JSON field 'light_lux'
    light_lux INTEGER,

    -- Wind speed in meters per second
    wind_speed_m_s REAL NOT NULL,

    -- Wind gust speed in meters per second (peak wind speed)
    wind_gust_m_s REAL NOT NULL,

    -- Wind direction in degrees (0-360, where 0=N, 90=E, 180=S, 270=W)
    wind_dir_deg INTEGER NOT NULL,

    -- Rainfall in millimeters (cumulative or per reading)
    rain_mm REAL NOT NULL,

    -- Battery status: 0.0 to 1.0 (where 1.0 = 100% battery) - matches rtl_433 JSON field 'battery_ok'
    battery REAL NOT NULL,

    -- Sensor model (e.g., "Fineoffset-WS90") - matches rtl_433 JSON field 'model'
    model TEXT
);

-- Create index on timestamp for time-series queries
-- DESC order makes queries for recent data faster
CREATE INDEX IF NOT EXISTS idx_readings_timestamp ON readings(timestamp DESC);

-- Create composite index on sensor_id and timestamp
-- Optimizes queries for specific sensors over time ranges
CREATE INDEX IF NOT EXISTS idx_readings_sensor_timestamp ON readings(sensor_id, timestamp DESC);

-- Create the daily rollups table
-- This table stores aggregated data for each day (permanent storage)
-- One row per day with summary statistics
CREATE TABLE IF NOT EXISTS daily_weather (
    -- The date this rollup represents (primary key)
    day_date DATE PRIMARY KEY,

    -- Highest temperature recorded during the day
    temp_high_c REAL NOT NULL,

    -- Lowest temperature recorded during the day
    temp_low_c REAL NOT NULL,

    -- Highest humidity percentage during the day
    humidity_high INTEGER NOT NULL,

    -- Lowest humidity percentage during the day
    humidity_low INTEGER NOT NULL,

    -- Total rainfall in millimeters for the day
    rain_mm REAL NOT NULL,

    -- Maximum wind gust speed recorded during the day
    wind_max_gust_m_s REAL NOT NULL,

    -- Average wind speed for the day (mean of all readings)
    wind_mean_m_s REAL NOT NULL,

    -- Number of wind speed readings used to calculate the mean
    wind_sample_count INTEGER NOT NULL,

    -- Total number of raw readings for this day
    readings_count INTEGER NOT NULL,

    -- Timestamp of the first reading of the day
    first_reading_ts TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Timestamp of the last reading of the day
    last_reading_ts TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Maximum UV index recorded during the day
    uv_max REAL,

    -- Maximum light level in lux recorded during the day
    light_max INTEGER
);

-- Create index on date for historical queries
-- DESC order makes queries for recent days faster
CREATE INDEX IF NOT EXISTS idx_daily_weather_date ON daily_weather(day_date DESC);