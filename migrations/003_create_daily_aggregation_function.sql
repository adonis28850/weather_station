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
    v_first_rain_mm REAL;
    v_last_rain_mm REAL;
    v_prev_rain_mm REAL;
    v_total_rain REAL;
    v_rain_start_count INTEGER;
    reading_record RECORD;
BEGIN
    -- SAFEGUARD: Check if there are readings for the target date
    -- If no readings exist, don't update anything (protects historical data)
    SELECT COUNT(*) INTO v_readings_count 
    FROM readings 
    WHERE DATE(timestamp) = target_date;

    IF v_readings_count = 0 THEN
        RETURN;
    END IF;

    -- Calculate statistics for the target date
    -- DATE(timestamp) = target_date filters readings for that specific day
    SELECT
        MAX(temperature_c),           -- Highest temperature
        MIN(temperature_c),           -- Lowest temperature
        MAX(humidity),                -- Highest humidity
        MIN(humidity),                -- Lowest humidity
        MAX(wind_gust_m_s),           -- Maximum wind gust
        AVG(wind_speed_m_s),          -- Average wind speed
        COUNT(wind_speed_m_s),        -- Count of wind readings
        COUNT(*),                     -- Total readings count
        MIN(timestamp),               -- First reading timestamp
        MAX(timestamp),               -- Last reading timestamp
        MAX(uv),                      -- Maximum UV index
        MAX(light_lux),               -- Maximum light level
        -- Get first rain_mm value when rain_start = 1
        (SELECT rain_mm FROM readings WHERE DATE(timestamp) = target_date AND rain_start = 1 ORDER BY timestamp ASC LIMIT 1),
        -- Get last rain_mm value when rain_start = 1
        (SELECT rain_mm FROM readings WHERE DATE(timestamp) = target_date AND rain_start = 1 ORDER BY timestamp DESC LIMIT 1)
    INTO
        v_temp_high, v_temp_low, v_humidity_high, v_humidity_low,
        v_wind_max_gust, v_wind_mean, v_wind_count,
        v_readings_count, v_first_ts, v_last_ts, v_uv_max, v_light_max,
        v_first_rain_mm, v_last_rain_mm
    FROM readings
    WHERE DATE(timestamp) = target_date;

    -- SAFEGUARD: Check if we have new-style readings with rain_start = 1
    SELECT COUNT(*) INTO v_rain_start_count 
    FROM readings 
    WHERE DATE(timestamp) = target_date AND rain_start = 1;

    -- Calculate rain based on available data type
    IF v_rain_start_count > 0 THEN
        -- Use new delta calculation for readings with rain_start = 1
        -- Handle sensor reset: if counter was reset during the day, sum positive deltas instead
        IF v_first_rain_mm IS NOT NULL AND v_last_rain_mm IS NOT NULL THEN
            IF v_last_rain_mm >= v_first_rain_mm THEN
                -- No reset detected, use simple delta
                v_rain_mm := v_last_rain_mm - v_first_rain_mm;
            ELSE
                -- Counter was reset during the day (last < first)
                -- Calculate rain by summing positive deltas between consecutive readings
                v_total_rain := 0;
                v_prev_rain_mm := NULL;

                FOR reading_record IN
                    SELECT rain_mm FROM readings
                    WHERE DATE(timestamp) = target_date AND rain_start = 1
                    ORDER BY timestamp ASC
                LOOP
                    IF v_prev_rain_mm IS NOT NULL AND reading_record.rain_mm > v_prev_rain_mm THEN
                        -- Only add positive deltas (ignore resets and decreases)
                        v_total_rain := v_total_rain + (reading_record.rain_mm - v_prev_rain_mm);
                    END IF;
                    v_prev_rain_mm := reading_record.rain_mm;
                END LOOP;

                v_rain_mm := v_total_rain;
            END IF;
        ELSE
            v_rain_mm := 0;
        END IF;
    ELSE
        -- Use old SUM logic for historical data (pre-rain_start implementation)
        SELECT COALESCE(SUM(rain_mm), 0) INTO v_rain_mm 
        FROM readings 
        WHERE DATE(timestamp) = target_date;
    END IF;

    -- Insert the aggregated data into the daily_weather table
    -- We know v_readings_count > 0 at this point
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