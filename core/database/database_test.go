package database

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"weather-station/core/config"
	"weather-station/shared/types"
)

// getTestDB returns a database connection for testing, skipping if not available
func getTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=weather_station sslmode=disable")
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		t.Skipf("PostgreSQL database not available: %v", err)
		return nil
	}

	return db
}

// TestConnect tests database connection functionality
func TestConnect(t *testing.T) {
	t.Run("SuccessfulConnection", func(t *testing.T) {
		// Set environment variables for test
		t.Setenv("WEATHER_DB_HOST", "127.0.0.1")
		t.Setenv("WEATHER_DB_PORT", "5432")
		t.Setenv("WEATHER_DB_USER", "postgres")
		t.Setenv("WEATHER_DB_PASSWORD", "postgres")
		t.Setenv("WEATHER_DB_NAME", "weather_station")
		t.Setenv("WEATHER_DB_MAX_OPEN_CONNS", "25")
		t.Setenv("WEATHER_DB_MAX_IDLE_CONNS", "5")
		t.Setenv("WEATHER_DB_CONN_MAX_LIFETIME_SECONDS", "300")

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		db, err := Connect(cfg)
		if err != nil {
			t.Fatalf("Expected successful connection, got error: %v", err)
		}
		defer db.Close()

		// Verify connection works
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			t.Errorf("Expected successful ping, got error: %v", err)
		}
	})

	t.Run("InvalidHost", func(t *testing.T) {
		t.Setenv("WEATHER_DB_HOST", "invalid-host")
		t.Setenv("WEATHER_DB_PORT", "5432")
		t.Setenv("WEATHER_DB_USER", "postgres")
		t.Setenv("WEATHER_DB_PASSWORD", "postgres")
		t.Setenv("WEATHER_DB_NAME", "weather_station")

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		db, err := Connect(cfg)
		if err == nil {
			db.Close()
			t.Error("Expected error for invalid host, got nil")
		}
	})

	t.Run("InvalidCredentials", func(t *testing.T) {
		t.Setenv("WEATHER_DB_HOST", "127.0.0.1")
		t.Setenv("WEATHER_DB_PORT", "5432")
		t.Setenv("WEATHER_DB_USER", "invalid_user")
		t.Setenv("WEATHER_DB_PASSWORD", "invalid_password")
		t.Setenv("WEATHER_DB_NAME", "weather_station")

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		db, err := Connect(cfg)
		if err == nil {
			db.Close()
			t.Error("Expected error for invalid credentials, got nil")
		}
	})
}

// TestConnectWithRetry tests connection with retry logic
func TestConnectWithRetry(t *testing.T) {
	t.Run("SuccessfulConnection", func(t *testing.T) {
		t.Setenv("WEATHER_DB_HOST", "127.0.0.1")
		t.Setenv("WEATHER_DB_PORT", "5432")
		t.Setenv("WEATHER_DB_USER", "postgres")
		t.Setenv("WEATHER_DB_PASSWORD", "postgres")
		t.Setenv("WEATHER_DB_NAME", "weather_station")
		t.Setenv("WEATHER_DB_MAX_RETRIES", "3")
		t.Setenv("WEATHER_DB_RETRY_INTERVAL_SECONDS", "1")

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		db, err := ConnectWithRetry(cfg)
		if err != nil {
			t.Fatalf("Expected successful connection, got error: %v", err)
		}
		defer db.Close()

		// Verify connection works
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			t.Errorf("Expected successful ping, got error: %v", err)
		}
	})

	t.Run("FailedAfterRetries", func(t *testing.T) {
		t.Setenv("WEATHER_DB_HOST", "invalid-host")
		t.Setenv("WEATHER_DB_PORT", "5432")
		t.Setenv("WEATHER_DB_USER", "postgres")
		t.Setenv("WEATHER_DB_PASSWORD", "postgres")
		t.Setenv("WEATHER_DB_NAME", "weather_station")
		t.Setenv("WEATHER_DB_MAX_RETRIES", "2")
		t.Setenv("WEATHER_DB_RETRY_INTERVAL_SECONDS", "1")

		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		db, err := ConnectWithRetry(cfg)
		if err == nil {
			db.Close()
			t.Error("Expected error after retries, got nil")
		}
	})
}

// TestPrepareInsertStatement tests preparing the INSERT statement
func TestPrepareInsertStatement(t *testing.T) {
	db := getTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	t.Run("SuccessfulPrepare", func(t *testing.T) {
		stmt, err := PrepareInsertStatement(db)
		if err != nil {
			t.Fatalf("Expected successful prepare, got error: %v", err)
		}
		defer stmt.Close()

		if stmt == nil {
			t.Error("Expected non-nil statement")
		}
	})

	t.Run("NilDatabase", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for nil database, but did not panic")
			}
		}()
		PrepareInsertStatement(nil)
	})
}

// TestInsertReading tests inserting a reading into the database
func TestInsertReading(t *testing.T) {
	db := getTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	t.Run("SuccessfulInsert", func(t *testing.T) {
		stmt, err := PrepareInsertStatement(db)
		if err != nil {
			t.Fatalf("Failed to prepare statement: %v", err)
		}
		defer stmt.Close()

		resultChan := make(chan error, 1)
		reading := types.Reading{
			Time:          "2026-03-15T10:00:00Z",
			Model:         "TestModel",
			SensorID:      9999,
			TemperatureC:  25.5,
			Humidity:      70,
			UVIndex:       6.0,
			Lux:           50000.0,
			WindSpeedMS:   5.0,
			WindGustMS:    10.0,
			WindDirDeg:    90,
			RainMM:        0.5,
			BatteryOK:     1.0,
			RainStart:     0,
			Firmware:     160,
		}

		InsertReading(stmt, resultChan, reading)

		select {
		case err := <-resultChan:
			if err != nil {
				t.Errorf("Expected successful insert, got error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("Timeout waiting for insert result")
		}

		// Verify the reading was inserted
		var count int
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM readings WHERE sensor_id = 9999").Scan(&count)
		if err != nil {
			t.Errorf("Failed to verify insert: %v", err)
		}
		if count == 0 {
			t.Error("Expected reading to be inserted, but count is 0")
		}

		// Clean up
		db.ExecContext(ctx, "DELETE FROM readings WHERE sensor_id = 9999")
	})

	t.Run("RFC3339NanoFormat", func(t *testing.T) {
		stmt, err := PrepareInsertStatement(db)
		if err != nil {
			t.Fatalf("Failed to prepare statement: %v", err)
		}
		defer stmt.Close()

		resultChan := make(chan error, 1)
		reading := types.Reading{
			Time:          "2026-03-15T11:30:45.123456789Z",
			Model:         "TestModel",
			SensorID:      9998,
			TemperatureC:  20.0,
			Humidity:      65,
			UVIndex:       5.0,
			Lux:           40000.0,
			WindSpeedMS:   3.0,
			WindGustMS:    7.0,
			WindDirDeg:    180,
			RainMM:        0.0,
			BatteryOK:     1.0,
			RainStart:     0,
			Firmware:     160,
		}

		InsertReading(stmt, resultChan, reading)

		select {
		case err := <-resultChan:
			if err != nil {
				t.Errorf("Expected successful insert, got error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("Timeout waiting for insert result")
		}

		// Clean up
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		db.ExecContext(ctx, "DELETE FROM readings WHERE sensor_id = 9998")
	})

	t.Run("Rtl433Format", func(t *testing.T) {
		stmt, err := PrepareInsertStatement(db)
		if err != nil {
			t.Fatalf("Failed to prepare statement: %v", err)
		}
		defer stmt.Close()

		resultChan := make(chan error, 1)
		reading := types.Reading{
			Time:          "2026-03-15 12:15:30",
			Model:         "TestModel",
			SensorID:      9997,
			TemperatureC:  22.5,
			Humidity:      68,
			UVIndex:       5.5,
			Lux:           45000.0,
			WindSpeedMS:   4.0,
			WindGustMS:    8.0,
			WindDirDeg:    135,
			RainMM:        0.2,
			BatteryOK:     1.0,
			RainStart:     0,
			Firmware:     160,
		}

		InsertReading(stmt, resultChan, reading)

		select {
		case err := <-resultChan:
			if err != nil {
				t.Errorf("Expected successful insert, got error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("Timeout waiting for insert result")
		}

		// Clean up
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		db.ExecContext(ctx, "DELETE FROM readings WHERE sensor_id = 9997")
	})

	t.Run("InvalidTimestamp", func(t *testing.T) {
		stmt, err := PrepareInsertStatement(db)
		if err != nil {
			t.Fatalf("Failed to prepare statement: %v", err)
		}
		defer stmt.Close()

		resultChan := make(chan error, 1)
		reading := types.Reading{
			Time:          "invalid-timestamp",
			Model:         "TestModel",
			SensorID:      9996,
			TemperatureC:  20.0,
			Humidity:      65,
			UVIndex:       5.0,
			Lux:           40000.0,
			WindSpeedMS:   3.0,
			WindGustMS:    7.0,
			WindDirDeg:    180,
			RainMM:        0.0,
			BatteryOK:     1.0,
			RainStart:     0,
			Firmware:     160,
		}

		InsertReading(stmt, resultChan, reading)

		select {
		case err := <-resultChan:
			if err == nil {
				t.Error("Expected error for invalid timestamp, got nil")
			}
		case <-time.After(5 * time.Second):
			t.Error("Timeout waiting for insert result")
		}
	})
}

// TestComputeDailyRollup tests triggering the daily rollup function
func TestComputeDailyRollup(t *testing.T) {
	db := getTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	t.Run("SuccessfulRollup", func(t *testing.T) {
		// First, insert some test readings for today
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Clean up any existing test data
		db.ExecContext(ctx, "DELETE FROM readings WHERE sensor_id = 9995")
		db.ExecContext(ctx, "DELETE FROM daily_weather WHERE day_date = CURRENT_DATE")

		// Insert test readings
		_, err := db.ExecContext(ctx, `
			INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, uv, light_lux, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware)
			VALUES 
				(9995, NOW() - INTERVAL '2 hours', 20.0, 65, 5.0, 40000.0, 3.0, 7.0, 180, 0.0, 1.0, 'TestModel', 0, 160),
				(9995, NOW() - INTERVAL '1 hour', 22.0, 68, 6.0, 45000.0, 4.0, 8.0, 135, 0.1, 1.0, 'TestModel', 0, 160),
				(9995, NOW(), 25.0, 70, 7.0, 50000.0, 5.0, 10.0, 90, 0.2, 1.0, 'TestModel', 0, 160)
		`)
		if err != nil {
			t.Fatalf("Failed to insert test readings: %v", err)
		}

		// Compute daily rollup
		today := time.Now()
		err = ComputeDailyRollup(db, today)
		if err != nil {
			t.Errorf("Expected successful rollup, got error: %v", err)
		}

		// Verify rollup was created
		var count int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM daily_weather WHERE day_date = CURRENT_DATE").Scan(&count)
		if err != nil {
			t.Errorf("Failed to verify rollup: %v", err)
		}
		if count == 0 {
			t.Error("Expected daily rollup to be created, but count is 0")
		}

		// Clean up
		db.ExecContext(ctx, "DELETE FROM readings WHERE sensor_id = 9995")
		db.ExecContext(ctx, "DELETE FROM daily_weather WHERE day_date = CURRENT_DATE")
	})

	t.Run("NoDataForDate", func(t *testing.T) {
		// Try to compute rollup for a date with no data
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Clean up any existing test data
		db.ExecContext(ctx, "DELETE FROM daily_weather WHERE day_date = CURRENT_DATE - INTERVAL '10 days'")

		date := time.Now().AddDate(0, 0, -10)
		err := ComputeDailyRollup(db, date)
		if err != nil {
			t.Errorf("Expected successful rollup (even with no data), got error: %v", err)
		}
	})
}

// TestCleanOldReadings tests deleting old readings based on retention policy
func TestCleanOldReadings(t *testing.T) {
	db := getTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	t.Run("SuccessfulCleanup", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Clean up any existing test data
		db.ExecContext(ctx, "DELETE FROM readings WHERE sensor_id IN (9994, 9993)")

		// Insert old readings (older than retention period)
		_, err := db.ExecContext(ctx, `
			INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, uv, light_lux, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware)
			VALUES 
				(9994, NOW() - INTERVAL '10 days', 20.0, 65, 5.0, 40000.0, 3.0, 7.0, 180, 0.0, 1.0, 'TestModel', 0, 160),
				(9994, NOW() - INTERVAL '15 days', 22.0, 68, 6.0, 45000.0, 4.0, 8.0, 135, 0.1, 1.0, 'TestModel', 0, 160),
				(9993, NOW() - INTERVAL '5 days', 25.0, 70, 7.0, 50000.0, 5.0, 10.0, 90, 0.2, 1.0, 'TestModel', 0, 160)
		`)
		if err != nil {
			t.Fatalf("Failed to insert test readings: %v", err)
		}

		// Count readings before cleanup
		var countBefore int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM readings WHERE sensor_id IN (9994, 9993)").Scan(&countBefore)
		if err != nil {
			t.Fatalf("Failed to count readings: %v", err)
		}

		// Clean readings older than 7 days
		err = CleanOldReadings(db, 7)
		if err != nil {
			t.Errorf("Expected successful cleanup, got error: %v", err)
		}

		// Count readings after cleanup
		var countAfter int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM readings WHERE sensor_id IN (9994, 9993)").Scan(&countAfter)
		if err != nil {
			t.Errorf("Failed to count readings after cleanup: %v", err)
		}

		// Should have deleted 2 old readings (10 and 15 days old), kept 1 (5 days old)
		if countAfter != 1 {
			t.Errorf("Expected 1 reading after cleanup, got %d", countAfter)
		}

		// Clean up
		db.ExecContext(ctx, "DELETE FROM readings WHERE sensor_id IN (9994, 9993)")
	})

	t.Run("NoOldReadings", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Clean up any existing test data
		db.ExecContext(ctx, "DELETE FROM readings WHERE sensor_id = 9992")

		// Insert only recent readings (within retention period)
		_, err := db.ExecContext(ctx, `
			INSERT INTO readings (sensor_id, timestamp, temperature_c, humidity, uv, light_lux, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware)
			VALUES (9992, NOW() - INTERVAL '1 day', 20.0, 65, 5.0, 40000.0, 3.0, 7.0, 180, 0.0, 1.0, 'TestModel', 0, 160)
		`)
		if err != nil {
			t.Fatalf("Failed to insert test readings: %v", err)
		}

		var countBefore int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM readings WHERE sensor_id = 9992").Scan(&countBefore)
		if err != nil {
			t.Fatalf("Failed to count readings: %v", err)
		}

		// Try to clean readings older than 7 days
		err = CleanOldReadings(db, 7)
		if err != nil {
			t.Errorf("Expected successful cleanup, got error: %v", err)
		}

		// Verify no readings were deleted
		var countAfter int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM readings WHERE sensor_id = 9992").Scan(&countAfter)
		if err != nil {
			t.Errorf("Failed to count readings after cleanup: %v", err)
		}

		if countAfter != countBefore {
			t.Errorf("Expected %d readings after cleanup, got %d", countBefore, countAfter)
		}

		// Clean up
		db.ExecContext(ctx, "DELETE FROM readings WHERE sensor_id = 9992")
	})
}