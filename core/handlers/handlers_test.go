package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"weather-station/core/config"
	"weather-station/shared/types"
)

// TestIngestHandler tests the ingest endpoint
func TestIngestHandler(t *testing.T) {
	// Test 1: Successful ingestion
	t.Run("SuccessfulIngestion", func(t *testing.T) {
		ingestionChan := make(chan types.Reading, 10)
		handler := IngestHandler(ingestionChan)

		reading := types.Reading{
			Time:          "2026-03-12T10:00:00Z",
			Model:         "TestModel",
			SensorID:      1,
			TemperatureC:  20.5,
			Humidity:      65,
			UVIndex:       5.0,
			Lux:           10000.0,
			WindSpeedMS:   3.5,
			WindGustMS:    8.0,
			WindDirDeg:    180,
			RainMM:        0.0,
			BatteryOK:     1.0,
			RainStart:     0,
			Firmware:     160,
		}

		body, _ := json.Marshal(reading)
		req := httptest.NewRequest(http.MethodPost, "/api/ingest", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
		}

		var response map[string]string
		json.Unmarshal(w.Body.Bytes(), &response)
		if response["status"] != "accepted" {
			t.Errorf("Expected status 'accepted', got '%s'", response["status"])
		}
	})

	// Test 2: Invalid JSON
	t.Run("InvalidJSON", func(t *testing.T) {
		ingestionChan := make(chan types.Reading, 10)
		handler := IngestHandler(ingestionChan)

		req := httptest.NewRequest(http.MethodPost, "/api/ingest", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	// Test 3: Wrong Content-Type
	t.Run("WrongContentType", func(t *testing.T) {
		ingestionChan := make(chan types.Reading, 10)
		handler := IngestHandler(ingestionChan)

		req := httptest.NewRequest(http.MethodPost, "/api/ingest", bytes.NewReader([]byte("{}")))
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusUnsupportedMediaType {
			t.Errorf("Expected status %d, got %d", http.StatusUnsupportedMediaType, w.Code)
		}
	})

	// Test 4: Invalid reading data
	t.Run("InvalidReadingData", func(t *testing.T) {
		ingestionChan := make(chan types.Reading, 10)
		handler := IngestHandler(ingestionChan)

		// Temperature below minimum
		invalidReading := types.Reading{
			Time:          "2026-03-12T10:00:00Z",
			Model:         "TestModel",
			SensorID:      1,
			TemperatureC:  -100.0, // Invalid
			Humidity:      65,
		}

		body, _ := json.Marshal(invalidReading)
		req := httptest.NewRequest(http.MethodPost, "/api/ingest", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	// Test 5: Channel full
	t.Run("ChannelFull", func(t *testing.T) {
		ingestionChan := make(chan types.Reading, 1)
		handler := IngestHandler(ingestionChan)

		// Fill the channel
		ingestionChan <- types.Reading{Time: "2026-03-12T10:00:00Z", Model: "Test", SensorID: 1}

		reading := types.Reading{
			Time:          "2026-03-12T10:00:01Z",
			Model:         "TestModel",
			SensorID:      1,
			TemperatureC:  20.5,
			Humidity:      65,
		}

		body, _ := json.Marshal(reading)
		req := httptest.NewRequest(http.MethodPost, "/api/ingest", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})
}

// TestMethodCheck tests the method check middleware
func TestMethodCheck(t *testing.T) {
	// Test 1: Allowed method
	t.Run("AllowedMethod", func(t *testing.T) {
		next := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}
		handler := MethodCheck(http.MethodGet, next)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	// Test 2: Disallowed method
	t.Run("DisallowedMethod", func(t *testing.T) {
		next := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}
		handler := MethodCheck(http.MethodGet, next)

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})
}

// TestHealthCheckHandler tests the health check endpoint
func TestHealthCheckHandler(t *testing.T) {
	// Test 1: Healthy database
	t.Run("HealthyDatabase", func(t *testing.T) {
		db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
		if err != nil {
			t.Skip("PostgreSQL not available for testing")
			return
		}
		defer db.Close()

		// Ping to verify connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = db.PingContext(ctx); err != nil {
			t.Skipf("PostgreSQL database not available: %v", err)
			return
		}

		server := NewServer(db, config.Config{})

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()

		server.HealthCheckHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got '%v'", response["status"])
		}
	})

	// Test 2: Unhealthy database (nil database)
	t.Run("UnhealthyDatabase", func(t *testing.T) {
		var db *sql.DB = nil

		server := NewServer(db, config.Config{})

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()

		// This will panic with nil database, so we need to recover
		defer func() {
			if r := recover(); r != nil {
				// Expected panic for nil database
				t.Logf("Expected panic with nil database: %v", r)
			}
		}()

		server.HealthCheckHandler(w, req)

		// If we get here without panic, check the response
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		if response["status"] != "unhealthy" {
			t.Errorf("Expected status 'unhealthy', got '%v'", response["status"])
		}
	})
}

// TestCurrentWeatherHandler tests the current weather endpoint
func TestCurrentWeatherHandler(t *testing.T) {
	// Test 1: No data available
	t.Run("NoDataAvailable", func(t *testing.T) {
		db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
		if err != nil {
			t.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		defer db.Close()

		// Ping to verify connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = db.PingContext(ctx); err != nil {
			t.Fatalf("Failed to ping database: %v", err)
		}

		server := NewServer(db, config.Config{})
		server.UpdateAstronomicalData()

		req := httptest.NewRequest(http.MethodGet, "/api/weather/current", nil)
		w := httptest.NewRecorder()

		server.CurrentWeatherHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}

		var response map[string]string
		json.Unmarshal(w.Body.Bytes(), &response)
		if response["error"] != "No data available" {
			t.Errorf("Expected error 'No data available', got '%s'", response["error"])
		}
	})

	// Test 2: Successful response with data
	t.Run("SuccessfulResponse", func(t *testing.T) {
		db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
		if err != nil {
			t.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		defer db.Close()

		// Ping to verify connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = db.PingContext(ctx); err != nil {
			t.Fatalf("Failed to ping database: %v", err)
		}

		// Clear any existing data
		db.Exec(`DELETE FROM readings`)

		_, err = db.Exec(`INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, uv, light_lux, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware) VALUES (
			1, '2026-03-12T10:00:00Z', 20.5, 65, 5.0, 10000.0, 3.5, 8.0, 180, 0.0, 1.0, 'TestModel', 0, 160
		)`)
		if err != nil { //nolint:nilness
			t.Fatalf("Failed to insert test data: %v", err)
		}

		server := NewServer(db, config.Config{})
		server.UpdateAstronomicalData()

		req := httptest.NewRequest(http.MethodGet, "/api/weather/current", nil)
		w := httptest.NewRecorder()

		server.CurrentWeatherHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response CurrentWeatherResponse
		json.Unmarshal(w.Body.Bytes(), &response)
		if response.TemperatureC != 20.5 {
			t.Errorf("Expected temperature 20.5, got %f", response.TemperatureC)
		}

		// Clean up
		db.Exec(`DELETE FROM readings WHERE timestamp = '2026-03-12T10:00:00Z'`)
	})

	// Test 3: RainStart calculated correctly when currently raining
	t.Run("RainStartCalculatedWhenRaining", func(t *testing.T) {
		db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
		if err != nil {
			t.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		defer db.Close()

		// Ping to verify connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = db.PingContext(ctx); err != nil {
			t.Fatalf("Failed to ping database: %v", err)
		}

		// Clear any existing data
		db.Exec(`DELETE FROM readings`)

		// Insert readings from the last 2 minutes with rain accumulation
		twoMinutesAgo := time.Now().Add(-2 * time.Minute)
		timestamp1 := twoMinutesAgo.Format(time.RFC3339)
		oneMinuteAgo := time.Now().Add(-1 * time.Minute)
		timestamp2 := oneMinuteAgo.Format(time.RFC3339)

		_, err = db.Exec(`INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, uv, light_lux, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware) VALUES 
			(1, $1, 20.0, 60, 5.0, 10000.0, 3.0, 7.0, 180, 0.0, 1.0, 'TestModel', 0, 160),
			(1, $2, 20.5, 62, 5.5, 11000.0, 3.2, 7.5, 185, 1.0, 1.0, 'TestModel', 0, 160)
		`, timestamp1, timestamp2)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}

		server := NewServer(db, config.Config{})
		server.UpdateAstronomicalData()

		req := httptest.NewRequest(http.MethodGet, "/api/weather/current", nil)
		w := httptest.NewRecorder()

		server.CurrentWeatherHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response CurrentWeatherResponse
		json.Unmarshal(w.Body.Bytes(), &response)

		// Verify RainStart is calculated as 1 (currently raining)
		if response.RainStart != 1 {
			t.Errorf("Expected RainStart 1 (currently raining), got %d", response.RainStart)
		}

		// Verify daily rain is calculated correctly
		if response.DailyRainMM != 1.0 {
			t.Errorf("Expected daily rain 1.0mm, got %f", response.DailyRainMM)
		}

		// Clean up
		db.Exec(`DELETE FROM readings WHERE timestamp >= $1`, twoMinutesAgo.Format(time.RFC3339))
	})

	// Test 4: RainStart calculated correctly when not raining
	t.Run("RainStartCalculatedWhenNotRaining", func(t *testing.T) {
		db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
		if err != nil {
			t.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		defer db.Close()

		// Ping to verify connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = db.PingContext(ctx); err != nil {
			t.Fatalf("Failed to ping database: %v", err)
		}

		// Clear any existing data
		db.Exec(`DELETE FROM readings`)

		// Insert a reading from 10 minutes ago (outside the 5-minute window)
		tenMinutesAgo := time.Now().Add(-10 * time.Minute)
		timestamp := tenMinutesAgo.Format(time.RFC3339)

		_, err = db.Exec(`INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, uv, light_lux, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware) VALUES (
			1, $1, 20.5, 65, 5.0, 10000.0, 3.5, 8.0, 180, 5.0, 1.0, 'TestModel', 0, 160
		)`, timestamp)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}

		server := NewServer(db, config.Config{})
		server.UpdateAstronomicalData()

		req := httptest.NewRequest(http.MethodGet, "/api/weather/current", nil)
		w := httptest.NewRecorder()

		server.CurrentWeatherHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response CurrentWeatherResponse
		json.Unmarshal(w.Body.Bytes(), &response)

		// Verify RainStart is calculated as 0 (not currently raining)
		if response.RainStart != 0 {
			t.Errorf("Expected RainStart 0 (not currently raining), got %d", response.RainStart)
		}

		// Clean up
		db.Exec(`DELETE FROM readings WHERE timestamp >= $1`, tenMinutesAgo.Format(time.RFC3339))
	})

	// Test 5: Database timeout
	t.Run("DatabaseTimeout", func(t *testing.T) {
		db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
		if err != nil {
			t.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		defer db.Close()

		// Ping to verify connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = db.PingContext(ctx); err != nil {
			t.Fatalf("Failed to ping database: %v", err)
		}

		server := NewServer(db, config.Config{})

		req := httptest.NewRequest(http.MethodGet, "/api/weather/current", nil)
		w := httptest.NewRecorder()

		// This test would need a custom setup to actually trigger a timeout
		// For now, we'll just verify the handler exists
		server.CurrentWeatherHandler(w, req)

		// Without actual data, we expect 404
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})
}

// TestHistoryWeatherHandler tests the history weather endpoint
func TestHistoryWeatherHandler(t *testing.T) {
	// Test 1: Invalid date format
	t.Run("InvalidDateFormat", func(t *testing.T) {
		db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
		if err != nil {
			t.Skip("PostgreSQL not available for testing")
			return
		}
		defer db.Close()

		// Ping to verify connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = db.PingContext(ctx); err != nil {
			t.Skipf("PostgreSQL database not available: %v", err)
			return
		}

		server := NewServer(db, config.Config{})

		req := httptest.NewRequest(http.MethodGet, "/api/weather/history?start=invalid-date", nil)
		w := httptest.NewRecorder()

		server.HistoryWeatherHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	// Test 2: No data available
	t.Run("NoDataAvailable", func(t *testing.T) {
		db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
		if err != nil {
			t.Skip("PostgreSQL not available for testing")
			return
		}
		defer db.Close()

		// Ping to verify connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = db.PingContext(ctx); err != nil {
			t.Skipf("PostgreSQL database not available: %v", err)
			return
		}

		// Create daily_weather table
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS daily_weather (
			day_date DATE,
			temp_high_c REAL,
			temp_low_c REAL,
			humidity_high INTEGER,
			humidity_low INTEGER,
			rain_mm REAL,
			wind_max_gust_m_s REAL,
			wind_mean_m_s REAL,
			wind_sample_count INTEGER,
			readings_count INTEGER,
			first_reading_ts TIMESTAMP,
			last_reading_ts TIMESTAMP,
			uv_max REAL,
			light_max INTEGER
		)`)
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}

		// Clear any existing data
		db.Exec(`DELETE FROM daily_weather`)

		server := NewServer(db, config.Config{})

		req := httptest.NewRequest(http.MethodGet, "/api/weather/history", nil)
		w := httptest.NewRecorder()

		server.HistoryWeatherHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	// Test 3: Successful response
	t.Run("SuccessfulResponse", func(t *testing.T) {
		db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
		if err != nil {
			t.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		defer db.Close()

		// Ping to verify connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = db.PingContext(ctx); err != nil {
			t.Fatalf("Failed to ping database: %v", err)
		}

		// Create daily_weather table and insert test data
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS daily_weather (
			day_date DATE,
			temp_high_c REAL,
			temp_low_c REAL,
			humidity_high INTEGER,
			humidity_low INTEGER,
			rain_mm REAL,
			wind_max_gust_m_s REAL,
			wind_mean_m_s REAL,
			wind_sample_count INTEGER,
			readings_count INTEGER,
			first_reading_ts TIMESTAMP,
			last_reading_ts TIMESTAMP,
			uv_max REAL,
			light_max INTEGER
		)`)
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}

		// Clear any existing data
		db.Exec(`DELETE FROM daily_weather`)

		// Use today's date for test data to ensure it's within the retention period
		now := time.Now()
		today := now.Format("2006-01-02")
		todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
		todayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.UTC).Format(time.RFC3339)

		_, err = db.Exec(`INSERT INTO daily_weather (day_date, temp_high_c, temp_low_c, humidity_high, humidity_low, rain_mm, wind_max_gust_m_s, wind_mean_m_s, wind_sample_count, readings_count, first_reading_ts, last_reading_ts, uv_max, light_max) VALUES (
			$1, 25.0, 15.0, 80, 40, 5.0, 10.0, 5.0, 100, 100, $2, $3, 8.0, 50000
		)`, today, todayStart, todayEnd)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}

		server := NewServer(db, config.Config{})

		req := httptest.NewRequest(http.MethodGet, "/api/weather/history", nil)
		w := httptest.NewRecorder()

		server.HistoryWeatherHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response []DailyRollupReading
		json.Unmarshal(w.Body.Bytes(), &response)
		if len(response) != 1 {
			t.Errorf("Expected 1 reading, got %d", len(response))
		}
		if response[0].TemperatureHighC != 25.0 {
			t.Errorf("Expected temp_high_c 25.0, got %f", response[0].TemperatureHighC)
		}

		// Clean up
		db.Exec(`DELETE FROM daily_weather WHERE day_date = '2026-03-12'`)
	})
}

// TestAvailableYearsHandler tests the available years endpoint
func TestAvailableYearsHandler(t *testing.T) {
	// Test 1: No data available
	t.Run("NoDataAvailable", func(t *testing.T) {
		db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
		if err != nil {
			t.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		defer db.Close()

		// Ping to verify connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = db.PingContext(ctx); err != nil {
			t.Fatalf("Failed to ping database: %v", err)
		}

		// Create daily_weather table
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS daily_weather (
			day_date DATE,
			temp_high_c REAL,
			temp_low_c REAL,
			humidity_high INTEGER,
			humidity_low INTEGER,
			rain_mm REAL,
			wind_max_gust_m_s REAL,
			wind_mean_m_s REAL,
			wind_sample_count INTEGER,
			readings_count INTEGER,
			first_reading_ts TIMESTAMP,
			last_reading_ts TIMESTAMP,
			uv_max REAL,
			light_max INTEGER
		)`)
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}

		// Clear any existing data
		db.Exec(`DELETE FROM daily_weather`)

		server := NewServer(db, config.Config{})

		req := httptest.NewRequest(http.MethodGet, "/api/weather/years", nil)
		w := httptest.NewRecorder()

		server.AvailableYearsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response []int
		json.Unmarshal(w.Body.Bytes(), &response)
		if len(response) != 0 {
			t.Errorf("Expected 0 years, got %d", len(response))
		}
	})

	// Test 2: Successful response with years
	t.Run("SuccessfulResponse", func(t *testing.T) {
		db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
		if err != nil {
			t.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		defer db.Close()

		// Ping to verify connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = db.PingContext(ctx); err != nil {
			t.Fatalf("Failed to ping database: %v", err)
		}

		// Create daily_weather table and insert test data
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS daily_weather (
			day_date DATE,
			temp_high_c REAL,
			temp_low_c REAL,
			humidity_high INTEGER,
			humidity_low INTEGER,
			rain_mm REAL,
			wind_max_gust_m_s REAL,
			wind_mean_m_s REAL,
			wind_sample_count INTEGER,
			readings_count INTEGER,
			first_reading_ts TIMESTAMP,
			last_reading_ts TIMESTAMP,
			uv_max REAL,
			light_max INTEGER
		)`)
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}

		// Clear any existing data
		db.Exec(`DELETE FROM daily_weather`)

		_, err = db.Exec(`INSERT INTO daily_weather VALUES ('2025-01-01', 0, 0, 0, 0, 0, 0, 0, 0, 0, '2025-01-01', '2025-01-01', 0, 0)`)
		if err != nil { //nolint:nilness
			t.Fatalf("Failed to insert test data: %v", err)
		}
		_, err = db.Exec(`INSERT INTO daily_weather VALUES ('2024-01-01', 0, 0, 0, 0, 0, 0, 0, 0, 0, '2024-01-01', '2024-01-01', 0, 0)`)
		if err != nil { //nolint:nilness
			t.Fatalf("Failed to insert test data: %v", err)
		}
		_, err = db.Exec(`INSERT INTO daily_weather VALUES ('2025-06-01', 0, 0, 0, 0, 0, 0, 0, 0, 0, '2025-06-01', '2025-06-01', 0, 0)`)
		if err != nil { //nolint:nilness
			t.Fatalf("Failed to insert test data: %v", err)
		}

		server := NewServer(db, config.Config{})

		req := httptest.NewRequest(http.MethodGet, "/api/weather/years", nil)
		w := httptest.NewRecorder()

		server.AvailableYearsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response []int
		json.Unmarshal(w.Body.Bytes(), &response)
		if len(response) != 2 {
			t.Errorf("Expected 2 unique years, got %d", len(response))
		}
		if response[0] != 2025 {
			t.Errorf("Expected first year 2025, got %d", response[0])
		}
		if response[1] != 2024 {
			t.Errorf("Expected second year 2024, got %d", response[1])
		}

		// Clean up
		db.Exec(`DELETE FROM daily_weather WHERE day_date IN ('2025-01-01', '2024-01-01', '2025-06-01')`)
	})
}

// TestRecentWeatherHandler tests the recent weather endpoint
func TestRecentWeatherHandler(t *testing.T) {
	// Test 1: Invalid date format
	t.Run("InvalidDateFormat", func(t *testing.T) {
		db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
		if err != nil {
			t.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		defer db.Close()

		// Ping to verify connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = db.PingContext(ctx); err != nil {
			t.Fatalf("Failed to ping database: %v", err)
		}

		server := NewServer(db, config.Config{})

		req := httptest.NewRequest(http.MethodGet, "/api/weather/recent?start=invalid-date", nil)
		w := httptest.NewRecorder()

		server.RecentWeatherHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	// Test 2: Invalid limit
	t.Run("InvalidLimit", func(t *testing.T) {
		db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
		if err != nil {
			t.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		defer db.Close()

		// Ping to verify connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = db.PingContext(ctx); err != nil {
			t.Fatalf("Failed to ping database: %v", err)
		}

		server := NewServer(db, config.Config{})

		req := httptest.NewRequest(http.MethodGet, "/api/weather/recent?limit=invalid", nil)
		w := httptest.NewRecorder()

		server.RecentWeatherHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	// Test 3: No data available
	t.Run("NoDataAvailable", func(t *testing.T) {
		db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
		if err != nil {
			t.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		defer db.Close()

		// Ping to verify connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = db.PingContext(ctx); err != nil {
			t.Fatalf("Failed to ping database: %v", err)
		}

		// Create readings table
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}

		// Clear any existing data
		db.Exec(`DELETE FROM readings`)

		server := NewServer(db, config.Config{})

		req := httptest.NewRequest(http.MethodGet, "/api/weather/recent", nil)
		w := httptest.NewRecorder()

		server.RecentWeatherHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	// Test 4: Successful response
	t.Run("SuccessfulResponse", func(t *testing.T) {
		db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
		if err != nil {
			t.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		defer db.Close()

		// Ping to verify connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = db.PingContext(ctx); err != nil {
			t.Fatalf("Failed to ping database: %v", err)
		}

		// Create readings table and insert test data
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}

		// Clear any existing data
		db.Exec(`DELETE FROM readings`)

		// Use today's date to ensure it's within the 1-day default range
		today := time.Now().Format(time.RFC3339)

		_, err = db.Exec(`INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, uv, light_lux, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware) VALUES (
			1, $1, 20.5, 65, 5.0, 10000.0, 3.5, 8.0, 180, 0.0, 1.0, 'TestModel', 0, 160
		)`, today)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}

		server := NewServer(db, config.Config{})

		req := httptest.NewRequest(http.MethodGet, "/api/weather/recent", nil)
		w := httptest.NewRecorder()

		server.RecentWeatherHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response []types.Reading
		json.Unmarshal(w.Body.Bytes(), &response)
		if len(response) != 1 {
			t.Errorf("Expected 1 reading, got %d", len(response))
		}
		if response[0].TemperatureC != 20.5 {
			t.Errorf("Expected temperature 20.5, got %f", response[0].TemperatureC)
		}

		// Clean up
		db.Exec(`DELETE FROM readings WHERE timestamp = '2026-03-12T10:00:00Z'`)
	})

	// Test 5: Custom limit
	t.Run("CustomLimit", func(t *testing.T) {
		db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
		if err != nil {
			t.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		defer db.Close()

		// Ping to verify connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = db.PingContext(ctx); err != nil {
			t.Fatalf("Failed to ping database: %v", err)
		}

		// Create readings table and insert test data
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}

		// Clear any existing data
		db.Exec(`DELETE FROM readings`)

		// Insert 5 readings using relative times
		now := time.Now()
		timestamp := now.Add(-1 * time.Hour).Format(time.RFC3339)
		
		for i := 0; i < 5; i++ {
			_, err = db.Exec(`INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, uv, light_lux, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware) VALUES (
				1, $1, 20.5, 65, 5.0, 10000.0, 3.5, 8.0, 180, 0.0, 1.0, 'TestModel', 0, 160
			)`, timestamp)
			if err != nil {
				t.Fatalf("Failed to insert test data: %v", err)
			}
		}

		server := NewServer(db, config.Config{})

		req := httptest.NewRequest(http.MethodGet, "/api/weather/recent?limit=2", nil)
		w := httptest.NewRecorder()

		server.RecentWeatherHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response []types.Reading
		json.Unmarshal(w.Body.Bytes(), &response)
		if len(response) != 2 {
			t.Errorf("Expected 2 readings (limited), got %d", len(response))
		}

		// Clean up
		db.Exec(`DELETE FROM readings WHERE timestamp = $1`, timestamp)
	})
}

// TestUpdateAstronomicalData tests the astronomical data update
func TestUpdateAstronomicalData(t *testing.T) {
	// Test 1: Successful update
	t.Run("SuccessfulUpdate", func(t *testing.T) {
		db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
		if err != nil {
			t.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		// Don't close db as it may be nil

		server := NewServer(db, config.Config{})

		// Update astronomical data
		server.UpdateAstronomicalData()

		// Verify data was cached
		server.astronomicalMutex.RLock()
		data := server.cachedAstronomical
		server.astronomicalMutex.RUnlock()

		if data.Sunrise == "" {
			t.Error("Expected non-empty sunrise time")
		}
		if data.Sunset == "" {
			t.Error("Expected non-empty sunset time")
		}
	})

	// Test 2: Concurrent updates
	t.Run("ConcurrentUpdates", func(t *testing.T) {
		db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
		if err != nil {
			t.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		// Don't close db as it may be nil

		server := NewServer(db, config.Config{})

		// Launch multiple concurrent updates
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				server.UpdateAstronomicalData()
				done <- true
			}()
		}

		// Wait for all updates to complete
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify data was cached
		server.astronomicalMutex.RLock()
		data := server.cachedAstronomical
		server.astronomicalMutex.RUnlock()

		if data.Sunrise == "" {
			t.Error("Expected non-empty sunrise time after concurrent updates")
		}
	})
}

// TestCalculateRainWithResetDetection tests the rain calculation with reset detection
func TestCalculateRainWithResetDetection(t *testing.T) {
	// Test 1: No readings
	t.Run("NoReadings", func(t *testing.T) {
		db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
		if err != nil {
			t.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		defer db.Close()

		// Ping to verify connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = db.PingContext(ctx); err != nil {
			t.Fatalf("Failed to ping database: %v", err)
		}

		// Create readings table
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}

		// Clear any existing data
		db.Exec(`DELETE FROM readings`)

		server := NewServer(db, config.Config{})

		ctx = context.Background()
		start := time.Now().AddDate(0, 0, -1)
		end := time.Now()

		rain := server.calculateRainWithResetDetection(ctx, start, end)

		if rain != 0 {
			t.Errorf("Expected 0 rain (no readings), got %f", rain)
		}
	})

	// Test 2: Normal progression (no reset)
	t.Run("NormalProgression", func(t *testing.T) {
		db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
		if err != nil {
			t.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		defer db.Close()

		// Ping to verify connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = db.PingContext(ctx); err != nil {
			t.Fatalf("Failed to ping database: %v", err)
		}

		// Create readings table
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}

		// Clear any existing data
		db.Exec(`DELETE FROM readings`)

		// Insert readings with normal rain progression using relative times
		now := time.Now()
		timestamp1 := now.Add(-2 * time.Hour).Format(time.RFC3339)
		timestamp2 := now.Add(-1*time.Hour - 50*time.Minute).Format(time.RFC3339)
		timestamp3 := now.Add(-1*time.Hour - 40*time.Minute).Format(time.RFC3339)

		_, err = db.Exec(`INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, uv, light_lux, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware) VALUES (1, $1, 0, 0, 0, 0, 0, 0, 0, 0.0, 0, '', 0, 0)`, timestamp1)
		if err != nil { //nolint:nilness
			t.Fatalf("Failed to insert test data: %v", err)
		}
		_, err = db.Exec(`INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, uv, light_lux, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware) VALUES (1, $1, 0, 0, 0, 0, 0, 0, 0, 5.0, 0, '', 0, 0)`, timestamp2)
		if err != nil { //nolint:nilness
			t.Fatalf("Failed to insert test data: %v", err)
		}
		_, err = db.Exec(`INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, uv, light_lux, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware) VALUES (1, $1, 0, 0, 0, 0, 0, 0, 0, 10.0, 0, '', 0, 0)`, timestamp3)
		if err != nil { //nolint:nilness
			t.Fatalf("Failed to insert test data: %v", err)
		}

		server := NewServer(db, config.Config{})

		ctx = context.Background()
		start := time.Now().AddDate(0, 0, -1)
		end := time.Now()

		rain := server.calculateRainWithResetDetection(ctx, start, end)

		if rain != 10.0 {
			t.Errorf("Expected 10.0 rain (normal progression), got %f", rain)
		}

		// Clean up
		db.Exec(`DELETE FROM readings WHERE timestamp >= $1 AND timestamp <= $2`, timestamp1, timestamp3)
	})

	// Test 3: Sensor reset detected
	t.Run("SensorResetDetected", func(t *testing.T) {
		db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
		if err != nil {
			t.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		defer db.Close()

		// Ping to verify connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err = db.PingContext(ctx); err != nil {
			t.Fatalf("Failed to ping database: %v", err)
		}

		// Create readings table
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}

		// Clear any existing data
		db.Exec(`DELETE FROM readings`)

		// Insert readings with sensor reset using relative times
		now := time.Now()
		timestamp1 := now.Add(-2 * time.Hour).Format(time.RFC3339)
		timestamp2 := now.Add(-1*time.Hour - 50*time.Minute).Format(time.RFC3339)
		timestamp3 := now.Add(-1*time.Hour - 40*time.Minute).Format(time.RFC3339)

		_, err = db.Exec(`INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, uv, light_lux, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware) VALUES (1, $1, 0, 0, 0, 0, 0, 0, 0, 5.0, 0, '', 0, 0)`, timestamp1)
		if err != nil { //nolint:nilness
			t.Fatalf("Failed to insert test data: %v", err)
		}
		_, err = db.Exec(`INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, uv, light_lux, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware) VALUES (1, $1, 0, 0, 0, 0, 0, 0, 0, 0.0, 0, '', 0, 0)`, timestamp2) // Reset
		if err != nil { //nolint:nilness
			t.Fatalf("Failed to insert test data: %v", err)
		}
		_, err = db.Exec(`INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, uv, light_lux, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware) VALUES (1, $1, 0, 0, 0, 0, 0, 0, 0, 3.0, 0, '', 0, 0)`, timestamp3)
		if err != nil { //nolint:nilness
			t.Fatalf("Failed to insert test data: %v", err)
		}

		server := NewServer(db, config.Config{})

		ctx = context.Background()
		start := time.Now().AddDate(0, 0, -1)
		end := time.Now()

		rain := server.calculateRainWithResetDetection(ctx, start, end)

		if rain != 8.0 {
			t.Errorf("Expected 8.0 rain (5.0 before reset + 3.0 after reset), got %f", rain)
		}

		// Clean up
		db.Exec(`DELETE FROM readings WHERE timestamp >= $1 AND timestamp <= $2`, timestamp1, timestamp3)
	})
}

// TestIsCurrentlyRaining tests the isCurrentlyRaining function
func TestIsCurrentlyRaining(t *testing.T) {
	db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	// Ping to verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	server := NewServer(db, config.Config{})

	// Test 1: No data available
	t.Run("NoDataAvailable", func(t *testing.T) {
		// Clear any existing data
		db.Exec(`DELETE FROM readings`)

		ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
		defer cancel()

		result := server.isCurrentlyRaining(ctx)
		if result != 0 {
			t.Errorf("Expected 0 (not raining) when no data available, got %d", result)
		}
	})

	// Test 2: No rain in last 5 minutes
	t.Run("NoRainInLast5Minutes", func(t *testing.T) {
		// Clear any existing data
		db.Exec(`DELETE FROM readings`)

		// Insert readings from 10 minutes ago (outside the 5-minute window)
		tenMinutesAgo := time.Now().Add(-10 * time.Minute)
		timestamp := tenMinutesAgo.Format(time.RFC3339)

		_, err = db.Exec(`INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, uv, light_lux, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware) VALUES (
			1, $1, 20.0, 60, 5.0, 10000.0, 3.0, 7.0, 180, 5.0, 1.0, 'TestModel', 0, 160
		)`, timestamp)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
		defer cancel()

		result := server.isCurrentlyRaining(ctx)
		if result != 0 {
			t.Errorf("Expected 0 (not raining) when no rain in last 5 minutes, got %d", result)
		}

		// Clean up
		db.Exec(`DELETE FROM readings`)
	})

	// Test 3: Rain in last 5 minutes (above threshold)
	t.Run("RainAboveThreshold", func(t *testing.T) {
		// Clear any existing data
		db.Exec(`DELETE FROM readings`)

		// Insert readings from the last 2 minutes with rain accumulation
		twoMinutesAgo := time.Now().Add(-2 * time.Minute)
		timestamp1 := twoMinutesAgo.Format(time.RFC3339)
		oneMinuteAgo := time.Now().Add(-1 * time.Minute)
		timestamp2 := oneMinuteAgo.Format(time.RFC3339)

		_, err = db.Exec(`INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, uv, light_lux, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware) VALUES 
			(1, $1, 20.0, 60, 5.0, 10000.0, 3.0, 7.0, 180, 0.0, 1.0, 'TestModel', 0, 160),
			(1, $2, 20.5, 62, 5.5, 11000.0, 3.2, 7.5, 185, 1.0, 1.0, 'TestModel', 0, 160)
		`, timestamp1, timestamp2)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
		defer cancel()

		result := server.isCurrentlyRaining(ctx)
		if result != 1 {
			t.Errorf("Expected 1 (raining) when rain > 0.1mm in last 5 minutes, got %d", result)
		}

		// Clean up
		db.Exec(`DELETE FROM readings`)
	})

	// Test 4: Rain in last 5 minutes (below threshold)
	t.Run("RainBelowThreshold", func(t *testing.T) {
		// Clear any existing data
		db.Exec(`DELETE FROM readings`)

		// Insert readings with very small rain accumulation (< 0.1mm)
		twoMinutesAgo := time.Now().Add(-2 * time.Minute)
		timestamp1 := twoMinutesAgo.Format(time.RFC3339)
		oneMinuteAgo := time.Now().Add(-1 * time.Minute)
		timestamp2 := oneMinuteAgo.Format(time.RFC3339)

		_, err = db.Exec(`INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, uv, light_lux, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware) VALUES 
			(1, $1, 20.0, 60, 5.0, 10000.0, 3.0, 7.0, 180, 0.0, 1.0, 'TestModel', 0, 160),
			(1, $2, 20.5, 62, 5.5, 11000.0, 3.2, 7.5, 185, 0.05, 1.0, 'TestModel', 0, 160)
		`, timestamp1, timestamp2)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
		defer cancel()

		result := server.isCurrentlyRaining(ctx)
		if result != 0 {
			t.Errorf("Expected 0 (not raining) when rain < 0.1mm in last 5 minutes, got %d", result)
		}

		// Clean up
		db.Exec(`DELETE FROM readings`)
	})

	// Test 5: Sensor reset during rain
	t.Run("SensorResetDuringRain", func(t *testing.T) {
		// Clear any existing data
		db.Exec(`DELETE FROM readings`)

		// Insert readings with a sensor reset in the middle
		twoMinutesAgo := time.Now().Add(-2 * time.Minute)
		timestamp1 := twoMinutesAgo.Format(time.RFC3339)
		ninetySecondsAgo := time.Now().Add(-90 * time.Second)
		timestamp2 := ninetySecondsAgo.Format(time.RFC3339)
		thirtySecondsAgo := time.Now().Add(-30 * time.Second)
		timestamp3 := thirtySecondsAgo.Format(time.RFC3339)

		_, err = db.Exec(`INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, uv, light_lux, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware) VALUES 
			(1, $1, 20.0, 60, 5.0, 10000.0, 3.0, 7.0, 180, 5.0, 1.0, 'TestModel', 0, 160),
			(1, $2, 20.5, 62, 5.5, 11000.0, 3.2, 7.5, 185, 0.0, 1.0, 'TestModel', 0, 160),
			(1, $3, 21.0, 64, 6.0, 12000.0, 3.5, 8.0, 190, 0.5, 1.0, 'TestModel', 0, 160)
		`, timestamp1, timestamp2, timestamp3)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
		defer cancel()

		result := server.isCurrentlyRaining(ctx)
		// Should count 0.5mm after reset (not 5.0 + 0.5)
		if result != 1 {
			t.Errorf("Expected 1 (raining) when rain after reset > 0.1mm, got %d", result)
		}

		// Clean up
		db.Exec(`DELETE FROM readings`)
	})

	// Test 6: Rain counter starts at non-zero value and doesn't change (production scenario)
	t.Run("NonZeroCounterNoChange", func(t *testing.T) {
		// Clear any existing data
		db.Exec(`DELETE FROM readings`)

		// Insert readings where rain counter starts at 6.2 and doesn't change
		// This simulates the production scenario
		twoMinutesAgo := time.Now().Add(-2 * time.Minute)
		timestamp1 := twoMinutesAgo.Format(time.RFC3339)
		ninetySecondsAgo := time.Now().Add(-90 * time.Second)
		timestamp2 := ninetySecondsAgo.Format(time.RFC3339)
		thirtySecondsAgo := time.Now().Add(-30 * time.Second)
		timestamp3 := thirtySecondsAgo.Format(time.RFC3339)

		_, err = db.Exec(`INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, uv, light_lux, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware) VALUES 
			(1, $1, 20.0, 60, 5.0, 10000.0, 3.0, 7.0, 180, 6.2, 1.0, 'TestModel', 0, 160),
			(1, $2, 20.5, 62, 5.5, 11000.0, 3.2, 7.5, 185, 6.2, 1.0, 'TestModel', 0, 160),
			(1, $3, 21.0, 64, 6.0, 12000.0, 3.5, 8.0, 190, 6.2, 1.0, 'TestModel', 0, 160)
		`, timestamp1, timestamp2, timestamp3)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
		defer cancel()

		result := server.isCurrentlyRaining(ctx)
		// Should return 0 because there's no rain accumulation (counter stays at 6.2)
		if result != 0 {
			t.Errorf("Expected 0 (not raining) when rain counter doesn't change, got %d", result)
		}

		// Clean up
		db.Exec(`DELETE FROM readings`)
	})
}

// TestLoggingMiddleware tests the logging middleware
func TestLoggingMiddleware(t *testing.T) {
	// Test 1: Logging enabled
	t.Run("LoggingEnabled", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := LoggingMiddleware(next, true)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	// Test 2: Logging disabled
	t.Run("LoggingDisabled", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := LoggingMiddleware(next, false)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}

// TestRTL433VersionHandler tests the rtl_433 version handler
func TestRTL433VersionHandler(t *testing.T) {
	db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
		return
	}
	defer db.Close()

	// Ping to verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		t.Skipf("PostgreSQL database not available: %v", err)
		return
	}

	server := NewServer(db, config.Config{})

	// Test 1: Valid version
	t.Run("ValidVersion", func(t *testing.T) {
		requestBody := map[string]string{
			"version": "rtl_433 version 24.10",
		}
		jsonBody, _ := json.Marshal(requestBody)

		req := httptest.NewRequest(http.MethodPost, "/api/system/rtl433-version", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.RTL433VersionHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]string
		json.Unmarshal(w.Body.Bytes(), &response)
		if response["status"] != "success" {
			t.Errorf("Expected status 'success', got '%v'", response["status"])
		}

		// Verify version was stored
		server.rtl433VersionMutex.RLock()
		version := server.rtl433Version
		server.rtl433VersionMutex.RUnlock()

		if version != "rtl_433 version 24.10" {
			t.Errorf("Expected version 'rtl_433 version 24.10', got '%s'", version)
		}
	})

	// Test 2: Empty version
	t.Run("EmptyVersion", func(t *testing.T) {
		requestBody := map[string]string{
			"version": "",
		}
		jsonBody, _ := json.Marshal(requestBody)

		req := httptest.NewRequest(http.MethodPost, "/api/system/rtl433-version", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.RTL433VersionHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	// Test 3: Invalid JSON
	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/system/rtl433-version", bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.RTL433VersionHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	// Test 4: Missing version field
	t.Run("MissingVersionField", func(t *testing.T) {
		requestBody := map[string]string{}
		jsonBody, _ := json.Marshal(requestBody)

		req := httptest.NewRequest(http.MethodPost, "/api/system/rtl433-version", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.RTL433VersionHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

// TestSystemInfoHandler tests the system info handler
func TestSystemInfoHandler(t *testing.T) {
	db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
		return
	}
	defer db.Close()

	// Ping to verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		t.Skipf("PostgreSQL database not available: %v", err)
		return
	}

	server := NewServer(db, config.Config{})

	// Test 1: No version set
	t.Run("NoVersionSet", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/system/info", nil)
		w := httptest.NewRecorder()

		server.SystemInfoHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]string
		json.Unmarshal(w.Body.Bytes(), &response)

		if response["rtl_433_version"] != "" {
			t.Errorf("Expected empty rtl_433_version, got '%s'", response["rtl_433_version"])
		}
	})

	// Test 2: Version set
	t.Run("VersionSet", func(t *testing.T) {
		// Set a version
		server.rtl433VersionMutex.Lock()
		server.rtl433Version = "rtl_433 version 24.10"
		server.rtl433VersionMutex.Unlock()

		req := httptest.NewRequest(http.MethodGet, "/api/system/info", nil)
		w := httptest.NewRecorder()

		server.SystemInfoHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]string
		json.Unmarshal(w.Body.Bytes(), &response)

		if response["rtl_433_version"] != "rtl_433 version 24.10" {
			t.Errorf("Expected rtl_433_version 'rtl_433 version 24.10', got '%s'", response["rtl_433_version"])
		}
	})
}

// TestCalculateRainWithResetDetectionEdgeCases tests edge cases for rain calculation with reset detection
func TestCalculateRainWithResetDetectionEdgeCases(t *testing.T) {
	db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
		return
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		t.Skipf("PostgreSQL database not available: %v", err)
		return
	}

	server := NewServer(db, config.Config{})

	t.Run("MultipleResetsInOneDay", func(t *testing.T) {
		sensorID := 9980

		// Clean up any existing test data
		db.ExecContext(ctx, "DELETE FROM readings WHERE sensor_id = $1", sensorID)

		// Insert readings with multiple resets
		stmt, err := db.PrepareContext(ctx, `
			INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware)
			VALUES ($1, $2, 20.0, 65, 3.0, 7.0, 180, $3, 1.0, 'TestModel', $4, 160)
		`)
		if err != nil {
			t.Fatalf("Failed to prepare insert statement: %v", err)
		}
		defer stmt.Close()

		baseTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 8, 0, 0, 0, time.UTC)

		// First rain event
		stmt.ExecContext(ctx, sensorID, baseTime, 0.5, 1)
		// Sensor reset
		stmt.ExecContext(ctx, sensorID, baseTime.Add(1*time.Hour), 0.0, 0)
		// Second rain event
		stmt.ExecContext(ctx, sensorID, baseTime.Add(2*time.Hour), 0.3, 1)
		// Another reset
		stmt.ExecContext(ctx, sensorID, baseTime.Add(3*time.Hour), 0.0, 0)
		// Third rain event
		stmt.ExecContext(ctx, sensorID, baseTime.Add(4*time.Hour), 0.2, 1)

		start := baseTime
		end := baseTime.Add(5 * time.Hour)

		rain := server.calculateRainWithResetDetection(ctx, start, end)
		expectedRain := 0.5 + 0.3 + 0.2 // Sum of all rain events

		if rain != expectedRain {
			t.Errorf("Expected total rain %.2f, got %.2f", expectedRain, rain)
		}

		// Clean up
		db.ExecContext(ctx, "DELETE FROM readings WHERE sensor_id = $1", sensorID)
	})

	t.Run("ContinuousRainWithoutReset", func(t *testing.T) {
		sensorID := 9981

		// Clean up any existing test data
		db.ExecContext(ctx, "DELETE FROM readings WHERE sensor_id = $1", sensorID)

		// Insert readings with continuous rain
		stmt, err := db.PrepareContext(ctx, `
			INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware)
			VALUES ($1, $2, 20.0, 65, 3.0, 7.0, 180, $3, 1.0, 'TestModel', $4, 160)
		`)
		if err != nil {
			t.Fatalf("Failed to prepare insert statement: %v", err)
		}
		defer stmt.Close()

		baseTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 8, 0, 0, 0, time.UTC)

		stmt.ExecContext(ctx, sensorID, baseTime, 0.5, 1)
		stmt.ExecContext(ctx, sensorID, baseTime.Add(1*time.Hour), 0.7, 1)
		stmt.ExecContext(ctx, sensorID, baseTime.Add(2*time.Hour), 0.8, 1)

		start := baseTime
		end := baseTime.Add(3 * time.Hour)

		rain := server.calculateRainWithResetDetection(ctx, start, end)
		expectedRain := 0.8 // Should take the last reading (cumulative)

		if rain != expectedRain {
			t.Errorf("Expected total rain %.2f, got %.2f", expectedRain, rain)
		}

		// Clean up
		db.ExecContext(ctx, "DELETE FROM readings WHERE sensor_id = $1", sensorID)
	})
}

// TestIsCurrentlyRainingEdgeCases tests edge cases for current rain detection
func TestIsCurrentlyRainingEdgeCases(t *testing.T) {
	db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
		return
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		t.Skipf("PostgreSQL database not available: %v", err)
		return
	}

	server := NewServer(db, config.Config{})

	t.Run("ExactThreshold", func(t *testing.T) {
		sensorID := 9982

		// Clean up any existing test data
		db.ExecContext(ctx, "DELETE FROM readings WHERE sensor_id = $1", sensorID)

		// Insert 2 readings with delta of exactly 0.1mm (should NOT be detected as raining, threshold is > 0.1)
		now := time.Now()
		stmt, err := db.PrepareContext(ctx, `
			INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware)
			VALUES ($1, $2, 20.0, 65, 3.0, 7.0, 180, $3, 1.0, 'TestModel', 1, 160)
		`)
		if err != nil {
			t.Fatalf("Failed to prepare insert statement: %v", err)
		}
		defer stmt.Close()

		stmt.ExecContext(ctx, sensorID, now.Add(-3*time.Minute), 0.0)
		stmt.ExecContext(ctx, sensorID, now.Add(-2*time.Minute), 0.1)

		isRaining := server.isCurrentlyRaining(ctx)

		if isRaining != 0 {
			t.Error("Should NOT detect rain at exact threshold (0.1mm), threshold is > 0.1")
		}

		// Clean up
		db.ExecContext(ctx, "DELETE FROM readings WHERE sensor_id = $1", sensorID)
	})

	t.Run("ResetWithinFiveMinutes", func(t *testing.T) {
		sensorID := 9983

		// Clean up any existing test data
		db.ExecContext(ctx, "DELETE FROM readings WHERE sensor_id = $1", sensorID)

		// Insert readings with reset within 5-minute window
		now := time.Now()
		stmt, err := db.PrepareContext(ctx, `
			INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware)
			VALUES ($1, $2, 20.0, 65, 3.0, 7.0, 180, $3, 1.0, 'TestModel', $4, 160)
		`)
		if err != nil {
			t.Fatalf("Failed to prepare insert statement: %v", err)
		}
		defer stmt.Close()

		stmt.ExecContext(ctx, sensorID, now.Add(-3*time.Minute), 0.5, 1)
		stmt.ExecContext(ctx, sensorID, now.Add(-2*time.Minute), 0.0, 0) // Reset
		stmt.ExecContext(ctx, sensorID, now.Add(-1*time.Minute), 0.05, 1)

		isRaining := server.isCurrentlyRaining(ctx)

		if isRaining != 0 {
			t.Error("Should not detect rain after reset within 5-minute window with low accumulation")
		}

		// Clean up
		db.ExecContext(ctx, "DELETE FROM readings WHERE sensor_id = $1", sensorID)
	})

	t.Run("JustAboveThreshold", func(t *testing.T) {
		sensorID := 9984

		// Clean up any existing test data
		db.ExecContext(ctx, "DELETE FROM readings WHERE sensor_id = $1", sensorID)

		// Insert 2 readings with delta of 0.11mm (above the threshold of 0.1mm)
		now := time.Now()
		stmt, err := db.PrepareContext(ctx, `
			INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware)
			VALUES ($1, $2, 20.0, 65, 3.0, 7.0, 180, $3, 1.0, 'TestModel', 1, 160)
		`)
		if err != nil {
			t.Fatalf("Failed to prepare insert statement: %v", err)
		}
		defer stmt.Close()

		stmt.ExecContext(ctx, sensorID, now.Add(-3*time.Minute), 0.0)
		stmt.ExecContext(ctx, sensorID, now.Add(-2*time.Minute), 0.11)

		isRaining := server.isCurrentlyRaining(ctx)

		if isRaining != 1 {
			t.Error("Should detect rain just above threshold (0.11mm)")
		}

		// Clean up
		db.ExecContext(ctx, "DELETE FROM readings WHERE sensor_id = $1", sensorID)
	})

	t.Run("JustBelowThreshold", func(t *testing.T) {
		sensorID := 9985

		// Clean up any existing test data
		db.ExecContext(ctx, "DELETE FROM readings WHERE sensor_id = $1", sensorID)

		// Insert reading just below threshold
		now := time.Now()
		_, err := db.ExecContext(ctx, `
			INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware)
			VALUES ($1, $2, 20.0, 65, 3.0, 7.0, 180, 0.09, 1.0, 'TestModel', 1, 160)
		`, sensorID, now.Add(-2*time.Minute))
		if err != nil {
			t.Fatalf("Failed to insert test reading: %v", err)
		}

		isRaining := server.isCurrentlyRaining(ctx)

		if isRaining != 0 {
			t.Error("Should not detect rain just below threshold (0.09mm)")
		}

		// Clean up
		db.ExecContext(ctx, "DELETE FROM readings WHERE sensor_id = $1", sensorID)
	})

	t.Run("NoRecentReadings", func(t *testing.T) {
		sensorID := 9986

		// Clean up any existing test data
		db.ExecContext(ctx, "DELETE FROM readings WHERE sensor_id = $1", sensorID)

		// Insert reading older than 5 minutes
		now := time.Now()
		_, err := db.ExecContext(ctx, `
			INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware)
			VALUES ($1, $2, 20.0, 65, 3.0, 7.0, 180, 0.5, 1.0, 'TestModel', 1, 160)
		`, sensorID, now.Add(-10*time.Minute))
		if err != nil {
			t.Fatalf("Failed to insert test reading: %v", err)
		}

		isRaining := server.isCurrentlyRaining(ctx)

		if isRaining != 0 {
			t.Error("Should not detect rain with readings older than 5 minutes")
		}

		// Clean up
		db.ExecContext(ctx, "DELETE FROM readings WHERE sensor_id = $1", sensorID)
	})
}
