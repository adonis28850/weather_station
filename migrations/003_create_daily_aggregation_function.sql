-- Migration 003: Create daily aggregation function
-- This function aggregates all readings for a specific date into a single daily rollup
-- It's called by the Go application's daily aggregator goroutine at midnight
CREATE OR REPLACE FUNCTION compute_daily_rollup(target_date DATE)
RETURNS void AS $$
DECLARE
    -- Variables to store aggregated values
    v_temp_high REAL;
    v_temp_low REAL;
    v_humidity_high INTEGER;
    v_humidity_low INTEGER;
    v_rain_mm REAL;
    v_wind_max_gust REAL;
    v_wind_mean REAL;
    v_wind_count INTEGER;
    v_readings_count INTEGER;
    v_first_ts TIMESTAMP WITH TIME ZONE;
    v_last_ts TIMESTAMP WITH TIME ZONE;
    v_uv_max REAL;
    v_light_max INTEGER;
BEGIN
    -- Calculate statistics for the target date
    -- DATE(timestamp) = target_date filters readings for that specific day
    SELECT
        MAX(temperature_c),           -- Highest temperature
        MIN(temperature_c),           -- Lowest temperature
        MAX(humidity),                -- Highest humidity
        MIN(humidity),                -- Lowest humidity
        COALESCE(SUM(rain_mm), 0),    -- Total rain (0 if no rain)
        MAX(wind_gust_m_s),           -- Maximum wind gust
        AVG(wind_speed_m_s),          -- Average wind speed
        COUNT(wind_speed_m_s),        -- Count of wind readings
        COUNT(*),                     -- Total readings count
        MIN(timestamp),               -- First reading timestamp
        MAX(timestamp),               -- Last reading timestamp
        MAX(uv),                      -- Maximum UV index
        MAX(light_lux)                -- Maximum light level
    INTO
        v_temp_high, v_temp_low, v_humidity_high, v_humidity_low,
        v_rain_mm, v_wind_max_gust, v_wind_mean, v_wind_count,
        v_readings_count, v_first_ts, v_last_ts, v_uv_max, v_light_max
    FROM readings
    WHERE DATE(timestamp) = target_date;

    -- Only insert if we have readings for this date
    -- This prevents NULL constraint violations when there's no data
    IF v_readings_count > 0 THEN
        -- Insert the aggregated data into the daily_weather table
        -- ON CONFLICT DO NOTHING prevents duplicate rollups for the same date
        INSERT INTO daily_weather (
            day_date, temp_high_c, temp_low_c, humidity_high, humidity_low,
            rain_mm, wind_max_gust_m_s, wind_mean_m_s, wind_sample_count,
            readings_count, first_reading_ts, last_reading_ts, uv_max, light_max
        ) VALUES (
            target_date, v_temp_high, v_temp_low, v_humidity_high, v_humidity_low,
            v_rain_mm, v_wind_max_gust, v_wind_mean, v_wind_count,
            v_readings_count, v_first_ts, v_last_ts, v_uv_max, v_light_max
        )
        ON CONFLICT (day_date) DO UPDATE SET
            uv_max = EXCLUDED.uv_max,
            light_max = EXCLUDED.light_max;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Note: This function is called by the Go application's daily aggregator goroutine