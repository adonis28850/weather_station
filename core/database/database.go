package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"weather-station/core/config"
	"weather-station/shared/logger"
	"weather-station/shared/types"
)

const (
	// QueryTimeout is the timeout for database queries
	// This prevents slow queries from blocking goroutines indefinitely
	// 5 seconds is sufficient for most queries while protecting against database overload
	QueryTimeout = 5 * time.Second
)

// PrepareInsertStatement prepares the INSERT statement for readings
func Connect(cfg config.Config) (*sql.DB, error) {
	connectionString := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=disable password=%s",
		cfg.DBHost(),
		cfg.DBPort(),
		cfg.DBUser(),
		cfg.DBName(),
		cfg.DBPassword())

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool using config values
	db.SetMaxOpenConns(cfg.DBMaxOpenConns())       // Maximum open connections
	db.SetMaxIdleConns(cfg.DBMaxIdleConns())       // Maximum idle connections
	db.SetConnMaxLifetime(cfg.DBConnMaxLifetime()) // Connection lifetime

	// Verify connection with Ping() with timeout
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// ConnectWithRetry attempts to connect to database with retry logic
func ConnectWithRetry(cfg config.Config) (*sql.DB, error) {
	var lastErr error

	for attempt := 0; attempt < cfg.DBMaxRetries(); attempt++ {
		db, err := Connect(cfg)
		if err == nil {
			return db, nil
		}

		lastErr = err
		logger.Error("Database connection attempt %d/%d failed: %v", attempt+1, cfg.DBMaxRetries(), err)

		if attempt < cfg.DBMaxRetries()-1 {
			time.Sleep(cfg.DBRetryInterval())
		}
	}

	return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", cfg.DBMaxRetries(), lastErr)
}

// PrepareInsertStatement prepares the INSERT statement for readings
// This should be called once at startup and reused for all insertions
func PrepareInsertStatement(db *sql.DB) (*sql.Stmt, error) {
	query := `INSERT INTO readings 
		(sensor_id, timestamp, temperature_c, humidity, uv, light_lux, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model, rain_start, firmware)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`
	
	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	
	return stmt, nil
}

// InsertReading inserts a single reading into the database using a prepared statement
func InsertReading(insertStmt *sql.Stmt, resultChan chan<- error, reading types.Reading) {
	// Parse timestamp - handle both ISO 8601 and rtl_433 format
	var timestampParsed time.Time
	var err error
	
	// Try ISO 8601 format first (RFC3339Nano)
	timestampParsed, err = time.Parse(time.RFC3339Nano, reading.Time)
	if err != nil {
		// Fall back to rtl_433 format: "2006-01-02 15:04:05"
		timestampParsed, err = time.Parse("2006-01-02 15:04:05", reading.Time)
		if err != nil {
			logger.Error("Failed to parse timestamp: %v", err)
			resultChan <- err
			return
		}
	}

	// Execute insert with timeout using prepared statement
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()
	_, err = insertStmt.ExecContext(ctx, reading.SensorID,
		timestampParsed,
		reading.TemperatureC,
		reading.Humidity,
		reading.UVIndex,
		reading.Lux,
		reading.WindSpeedMS,
		reading.WindGustMS,
		reading.WindDirDeg,
		reading.RainMM,
		reading.BatteryOK,
		reading.Model,
		reading.RainStart,
		reading.Firmware)
	if err != nil {
		logger.Error("Failed to insert reading: %v", err)
		resultChan <- err
		return
	}

	logger.Info("Successfully inserted reading with sensor_id %d at %s", reading.SensorID, reading.Time)
	resultChan <- nil // Success
}

// ComputeDailyRollup triggers the daily rollup for a specific date
func ComputeDailyRollup(db *sql.DB, date time.Time) error {
	// Call the PostgreSQL function to compute the daily rollup for the specified date with timeout
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()
	_, err := db.ExecContext(ctx, "SELECT compute_daily_rollup($1)", date)

	if err != nil {
		logger.Error("Failed to compute daily rollup for %s: %v", date.Format("2006-01-02"), err)
		return err
	}

	logger.Info("Successfully computed daily rollup for %s", date.Format("2006-01-02"))
	return nil
}

// CleanOldReadings deletes readings older than the retention period
func CleanOldReadings(db *sql.DB, retentionDays int) error {
	// Call the PostgreSQL function to delete old readings based on retention policy with timeout
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()
	result, err := db.ExecContext(ctx, "SELECT delete_old_readings($1)", retentionDays)
	if err != nil {
		logger.Error("Failed to clean old readings: %v", err)
		return err
	}

	// Get the number of rows affected by the delete operation
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logger.Error("Failed to get rows affected: %v", err)
		return err
	}

	logger.Info("Deleted %d old readings", rowsAffected)
	return nil
}