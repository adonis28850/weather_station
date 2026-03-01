package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration values for the weather station application
// All values are loaded from environment variables with sensible defaults
type Config struct {
	serverPort          int
	dbHost              string
	dbPort              int
	dbUser              string
	dbPassword          string
	dbName              string
	ingestionChanSize   int
	jobQueueSize        int
	numWorkers          int
	resultQueueSize     int
	aggregationSchedule string // e.g., "0 0 * * *" for daily at midnight
	retentionDays       int
	latitude            float64 // Geographic latitude for astronomical calculations
	longitude           float64 // Geographic longitude for astronomical calculations
	timezone            string  // Timezone for local time conversion (e.g., "Europe/Madrid")
	// Database connection pool settings
	dbMaxOpenConns    int           // Maximum open connections in the pool
	dbMaxIdleConns    int           // Maximum idle connections in the pool
	dbConnMaxLifetime time.Duration // Maximum lifetime of a connection
	// Database connection retry settings
	dbMaxRetries      int           // Maximum number of connection retry attempts
	dbRetryInterval   time.Duration // Interval between retry attempts
	enableHTTPLogging bool          // Flag to enable or disable HTTP request logging
}

// Getters for Config fields (to keep fields private)
func (c *Config) ServerPort() int                  { return c.serverPort }
func (c *Config) DBHost() string                   { return c.dbHost }
func (c *Config) DBPort() int                      { return c.dbPort }
func (c *Config) DBUser() string                   { return c.dbUser }
func (c *Config) DBPassword() string               { return c.dbPassword }
func (c *Config) DBName() string                   { return c.dbName }
func (c *Config) IngestionChanSize() int           { return c.ingestionChanSize }
func (c *Config) JobQueueSize() int                { return c.jobQueueSize }
func (c *Config) NumWorkers() int                  { return c.numWorkers }
func (c *Config) ResultQueueSize() int             { return c.resultQueueSize }
func (c *Config) AggregationSchedule() string      { return c.aggregationSchedule }
func (c *Config) RetentionDays() int               { return c.retentionDays }
func (c *Config) Latitude() float64                { return c.latitude }
func (c *Config) Longitude() float64               { return c.longitude }
func (c *Config) Timezone() string                 { return c.timezone }
func (c *Config) DBMaxOpenConns() int              { return c.dbMaxOpenConns }
func (c *Config) DBMaxIdleConns() int              { return c.dbMaxIdleConns }
func (c *Config) DBConnMaxLifetime() time.Duration { return c.dbConnMaxLifetime }
func (c *Config) DBMaxRetries() int                { return c.dbMaxRetries }
func (c *Config) DBRetryInterval() time.Duration   { return c.dbRetryInterval }
func (c *Config) EnableHTTPLogging() bool          { return c.enableHTTPLogging }

// Helper functions for configuration loading
// These reduce code duplication and make configuration parsing more maintainable

// getEnvInt retrieves an integer environment variable with optional range validation
func getEnvInt(name string, defaultValue int, min, max int) (int, error) {
	if val := os.Getenv(name); val != "" {
		parsed, err := strconv.Atoi(val)
		if err != nil {
			return defaultValue, fmt.Errorf("Invalid %s: must be a valid integer", name)
		}
		if parsed < min || parsed > max {
			return defaultValue, fmt.Errorf("Invalid %s: must be between %d and %d, got %d", name, min, max, parsed)
		}
		return parsed, nil
	}
	return defaultValue, nil
}

// getEnvFloat retrieves a float environment variable with optional range validation
func getEnvFloat(name string, defaultValue float64, min, max float64) (float64, error) {
	if val := os.Getenv(name); val != "" {
		parsed, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return defaultValue, fmt.Errorf("Invalid %s: must be a valid float", name)
		}
		if parsed < min || parsed > max {
			return defaultValue, fmt.Errorf("Invalid %s: must be between %.2f and %.2f, got %.2f", name, min, max, parsed)
		}
		return parsed, nil
	}
	return defaultValue, nil
}

// getEnvBool retrieves a boolean environment variable
func getEnvBool(name string, defaultValue bool) (bool, error) {
	if val := os.Getenv(name); val != "" {
		parsed, err := strconv.ParseBool(val)
		if err != nil {
			return defaultValue, fmt.Errorf("Invalid %s: must be a valid boolean (true/false)", name)
		}
		return parsed, nil
	}
	return defaultValue, nil
}

// getEnvString retrieves a string environment variable
func getEnvString(name string, defaultValue string) string {
	if val := os.Getenv(name); val != "" {
		return val
	}
	return defaultValue
}

// getEnvDurationSeconds retrieves a duration environment variable from seconds
func getEnvDurationSeconds(name string, defaultValueSeconds int) (time.Duration, error) {
	seconds, err := getEnvInt(name, defaultValueSeconds, 1, 86400)
	if err != nil {
		return time.Duration(defaultValueSeconds) * time.Second, err
	}
	return time.Duration(seconds) * time.Second, nil
}

// Load loads configuration from environment variables set up by Kubernetes ConfigMap
func Load() (Config, error) {
	// Server configuration
	serverPort, err := getEnvInt("WEATHER_SERVER_PORT", 8080, 1, 65535)
	if err != nil {
		return Config{}, err
	}

	// Database configuration
	dbHost := getEnvString("WEATHER_DB_HOST", "")
	dbPort, err := getEnvInt("WEATHER_DB_PORT", 5432, 1, 65535)
	if err != nil {
		return Config{}, err
	}
	dbUser := getEnvString("WEATHER_DB_USER", "")
	dbPassword := getEnvString("WEATHER_DB_PASSWORD", "")
	dbName := getEnvString("WEATHER_DB_NAME", "")

	// Worker pool configuration
	ingestionChanSize, err := getEnvInt("WEATHER_INGESTION_CHAN_SIZE", 1000, 1, 100000)
	if err != nil {
		return Config{}, err
	}
	jobQueueSize, err := getEnvInt("WEATHER_JOB_QUEUE_SIZE", 100, 1, 10000)
	if err != nil {
		return Config{}, err
	}
	numWorkers, err := getEnvInt("WEATHER_WORKER_COUNT", 5, 1, 100)
	if err != nil {
		return Config{}, err
	}
	resultQueueSize, err := getEnvInt("WEATHER_RESULT_QUEUE_SIZE", 100, 1, 10000)
	if err != nil {
		return Config{}, err
	}

	// Aggregation and retention
	aggregationSchedule := getEnvString("WEATHER_AGGREGATION_SCHEDULE", "0 0 * * *")
	retentionDays, err := getEnvInt("WEATHER_RETENTION_DAYS", 30, 1, 3650)
	if err != nil {
		return Config{}, err
	}

	// Astronomical configuration
	latitude, err := getEnvFloat("WEATHER_LATITUDE", 41.8262, -90, 90)
	if err != nil {
		return Config{}, err
	}
	longitude, err := getEnvFloat("WEATHER_LONGITUDE", -87.6841, -180, 180)
	if err != nil {
		return Config{}, err
	}
	timezone := getEnvString("WEATHER_TIMEZONE", "UTC")
	// Validate timezone by trying to load it
	if _, err := time.LoadLocation(timezone); err != nil {
		return Config{}, fmt.Errorf("Invalid WEATHER_TIMEZONE: must be a valid IANA timezone (e.g., 'Europe/Madrid'): %w", err)
	}

	// Database connection pool settings
	dbMaxOpenConns, err := getEnvInt("WEATHER_DB_MAX_OPEN_CONNS", 25, 1, 1000)
	if err != nil {
		return Config{}, err
	}
	dbMaxIdleConns, err := getEnvInt("WEATHER_DB_MAX_IDLE_CONNS", 5, 0, 1000)
	if err != nil {
		return Config{}, err
	}
	dbConnMaxLifetime, err := getEnvDurationSeconds("WEATHER_DB_CONN_MAX_LIFETIME_SECONDS", 300)
	if err != nil {
		return Config{}, err
	}

	// Database connection retry settings
	dbMaxRetries, err := getEnvInt("WEATHER_DB_MAX_RETRIES", 5, 1, 100)
	if err != nil {
		return Config{}, err
	}
	dbRetryInterval, err := getEnvDurationSeconds("WEATHER_DB_RETRY_INTERVAL_SECONDS", 5)
	if err != nil {
		return Config{}, err
	}

	// HTTP logging
	enableHTTPLogging, err := getEnvBool("WEATHER_ENABLE_HTTP_LOGGING", true)
	if err != nil {
		return Config{}, err
	}

	// Create Config struct
	config := Config{
		serverPort:          serverPort,
		dbHost:              dbHost,
		dbPort:              dbPort,
		dbUser:              dbUser,
		dbPassword:          dbPassword,
		dbName:              dbName,
		ingestionChanSize:   ingestionChanSize,
		jobQueueSize:        jobQueueSize,
		numWorkers:          numWorkers,
		resultQueueSize:     resultQueueSize,
		aggregationSchedule: aggregationSchedule,
		retentionDays:       retentionDays,
		latitude:            latitude,
		longitude:           longitude,
		timezone:            timezone,
		dbMaxOpenConns:      dbMaxOpenConns,
		dbMaxIdleConns:      dbMaxIdleConns,
		dbConnMaxLifetime:   dbConnMaxLifetime,
		dbMaxRetries:        dbMaxRetries,
		dbRetryInterval:     dbRetryInterval,
		enableHTTPLogging:   enableHTTPLogging,
	}

	// Validate interdependent configuration values
	if err := validateConfig(config); err != nil {
		return Config{}, err
	}

	return config, nil
}

// validateConfig performs cross-field validation of configuration values
// This ensures that interdependent settings are consistent and valid
func validateConfig(config Config) error {
	// Validate database connection pool settings
	// Idle connections should not exceed maximum open connections
	if config.dbMaxIdleConns > config.dbMaxOpenConns {
		return fmt.Errorf("Invalid configuration: WEATHER_DB_MAX_IDLE_CONNS (%d) cannot exceed WEATHER_DB_MAX_OPEN_CONNS (%d)",
			config.dbMaxIdleConns, config.dbMaxOpenConns)
	}

	// Validate worker pool configuration
	// Number of workers should be reasonable relative to job queue size
	if config.numWorkers > config.jobQueueSize {
		// Log warning but don't fail
		fmt.Printf("Warning: WEATHER_WORKER_COUNT (%d) exceeds WEATHER_JOB_QUEUE_SIZE (%d). Some workers may be idle.\n",
			config.numWorkers, config.jobQueueSize)
	}

	// Validate database connection lifetime vs retry interval
	// Connection lifetime should be significantly longer than retry interval
	if config.dbConnMaxLifetime < config.dbRetryInterval*10 {
		// Log warning but don't fail
		fmt.Printf("Warning: WEATHER_DB_CONN_MAX_LIFETIME (%v) is very short relative to WEATHER_DB_RETRY_INTERVAL (%v). Consider increasing lifetime.\n",
			config.dbConnMaxLifetime, config.dbRetryInterval)
	}

	return nil
}
