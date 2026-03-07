-- Migration 003: Create daily aggregation function
-- This function aggregates all readings for a specific date into a single daily rollup
-- It's called by the Go application's daily aggregator goroutine at midnight
--
-- IMPORTANT: rain_mm is a cumulative counter from the WS90 sensor
-- We calculate daily rain as the delta between last and first reading of the day
-- Only count readings where rain_start = 1 (sensor detects rain event)
--
-- SAFEGUARDS:
-- - Only updates dates that have actual readings (protects historical data)
-- - Falls back to SUM(rain_mm) for pre-rain_start readings
-- - Uses delta calculation for readings with rain_start = 1
CREATE OR REPLACE FUNCTION compute_daily_rollup(target_date DATE)
RETURNS void AS $$
DECLARE
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
    v_has_rain_start BOOLEAN;
BEGIN
    -- Get all statistics in a single query, using FILTER to check for rain_start
    SELECT
        MAX(temperature_c),
        MIN(temperature_c),
        MAX(humidity),
        MIN(humidity),
        MAX(wind_gust_m_s),
        AVG(wind_speed_m_s),
        COUNT(wind_speed_m_s),
        COUNT(*),
        MIN(timestamp),
        MAX(timestamp),
        MAX(uv),
        MAX(light_lux),
        COUNT(*) FILTER (WHERE rain_start = 1) > 0
    INTO
        v_temp_high, v_temp_low, v_humidity_high, v_humidity_low,
        v_wind_max_gust, v_wind_mean, v_wind_count, v_readings_count,
        v_first_ts, v_last_ts, v_uv_max, v_light_max, v_has_rain_start
    FROM readings
    WHERE DATE(timestamp) = target_date;

    -- SAFEGUARD: If no readings exist, don't update anything
    IF v_readings_count = 0 THEN
        RETURN;
    END IF;

    -- Calculate rain based on available data type
    IF v_has_rain_start THEN
        -- Use window function to calculate delta, handling sensor resets
        SELECT COALESCE(SUM(rain_delta), 0) INTO v_rain_mm
        FROM (
            SELECT
                CASE
                    WHEN rain_mm > LAG(rain_mm) OVER (ORDER BY timestamp)
                    THEN rain_mm - LAG(rain_mm) OVER (ORDER BY timestamp)
                    ELSE 0
                END as rain_delta
            FROM readings
            WHERE DATE(timestamp) = target_date AND rain_start = 1
        ) deltas;
    ELSE
        -- Use old SUM logic for historical data (pre-rain_start implementation)
        SELECT COALESCE(SUM(rain_mm), 0) INTO v_rain_mm
        FROM readings
        WHERE DATE(timestamp) = target_date;
    END IF;

    -- Insert the aggregated data into the daily_weather table
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
        temp_high_c = EXCLUDED.temp_high_c,
        temp_low_c = EXCLUDED.temp_low_c,
        humidity_high = EXCLUDED.humidity_high,
        humidity_low = EXCLUDED.humidity_low,
        rain_mm = EXCLUDED.rain_mm,
        wind_max_gust_m_s = EXCLUDED.wind_max_gust_m_s,
        wind_mean_m_s = EXCLUDED.wind_mean_m_s,
        wind_sample_count = EXCLUDED.wind_sample_count,
        readings_count = EXCLUDED.readings_count,
        first_reading_ts = EXCLUDED.first_reading_ts,
        last_reading_ts = EXCLUDED.last_reading_ts,
        uv_max = EXCLUDED.uv_max,
        light_max = EXCLUDED.light_max;
END;
$$ LANGUAGE plpgsql;

-- Note: This function is called by the Go application's daily aggregator goroutine