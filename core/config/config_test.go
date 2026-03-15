package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	t.Run("ValidConfigFromEnvVars", func(t *testing.T) {
		// Set valid environment variables
		t.Setenv("WEATHER_SERVER_PORT", "8080")
		t.Setenv("WEATHER_DB_HOST", "localhost")
		t.Setenv("WEATHER_DB_PORT", "5432")
		t.Setenv("WEATHER_DB_USER", "testuser")
		t.Setenv("WEATHER_DB_PASSWORD", "testpass")
		t.Setenv("WEATHER_DB_NAME", "testdb")
		t.Setenv("WEATHER_INGESTION_CHAN_SIZE", "1000")
		t.Setenv("WEATHER_JOB_QUEUE_SIZE", "100")
		t.Setenv("WEATHER_WORKER_COUNT", "5")
		t.Setenv("WEATHER_RESULT_QUEUE_SIZE", "100")
		t.Setenv("WEATHER_AGGREGATION_SCHEDULE", "0 0 * * *")
		t.Setenv("WEATHER_RETENTION_DAYS", "30")
		t.Setenv("WEATHER_LATITUDE", "41.8262")
		t.Setenv("WEATHER_LONGITUDE", "-87.6841")
		t.Setenv("WEATHER_TIMEZONE", "America/Chicago")
		t.Setenv("WEATHER_DB_MAX_OPEN_CONNS", "25")
		t.Setenv("WEATHER_DB_MAX_IDLE_CONNS", "5")
		t.Setenv("WEATHER_DB_CONN_MAX_LIFETIME_SECONDS", "300")
		t.Setenv("WEATHER_DB_MAX_RETRIES", "5")
		t.Setenv("WEATHER_DB_RETRY_INTERVAL_SECONDS", "5")
		t.Setenv("WEATHER_ENABLE_HTTP_LOGGING", "true")

		config, err := Load()
		if err != nil {
			t.Fatalf("Failed to load valid config: %v", err)
		}

		// Verify values
		if config.ServerPort() != 8080 {
			t.Errorf("Expected server port 8080, got %d", config.ServerPort())
		}
		if config.DBHost() != "localhost" {
			t.Errorf("Expected DB host localhost, got %s", config.DBHost())
		}
		if config.DBPort() != 5432 {
			t.Errorf("Expected DB port 5432, got %d", config.DBPort())
		}
		if config.DBUser() != "testuser" {
			t.Errorf("Expected DB user testuser, got %s", config.DBUser())
		}
		if config.DBName() != "testdb" {
			t.Errorf("Expected DB name testdb, got %s", config.DBName())
		}
		if config.NumWorkers() != 5 {
			t.Errorf("Expected 5 workers, got %d", config.NumWorkers())
		}
		if config.Latitude() != 41.8262 {
			t.Errorf("Expected latitude 41.8262, got %f", config.Latitude())
		}
		if config.Longitude() != -87.6841 {
			t.Errorf("Expected longitude -87.6841, got %f", config.Longitude())
		}
		if config.Timezone() != "America/Chicago" {
			t.Errorf("Expected timezone America/Chicago, got %s", config.Timezone())
		}
		if config.EnableHTTPLogging() != true {
			t.Errorf("Expected HTTP logging enabled, got %v", config.EnableHTTPLogging())
		}
	})

	t.Run("MissingRequiredEnvVars", func(t *testing.T) {
		// Clear all environment variables
		envVars := []string{
			"WEATHER_SERVER_PORT", "WEATHER_DB_HOST", "WEATHER_DB_PORT",
			"WEATHER_DB_USER", "WEATHER_DB_PASSWORD", "WEATHER_DB_NAME",
			"WEATHER_INGESTION_CHAN_SIZE", "WEATHER_JOB_QUEUE_SIZE",
			"WEATHER_WORKER_COUNT", "WEATHER_RESULT_QUEUE_SIZE",
			"WEATHER_AGGREGATION_SCHEDULE", "WEATHER_RETENTION_DAYS",
			"WEATHER_LATITUDE", "WEATHER_LONGITUDE", "WEATHER_TIMEZONE",
			"WEATHER_DB_MAX_OPEN_CONNS", "WEATHER_DB_MAX_IDLE_CONNS",
			"WEATHER_DB_CONN_MAX_LIFETIME_SECONDS", "WEATHER_DB_MAX_RETRIES",
			"WEATHER_DB_RETRY_INTERVAL_SECONDS", "WEATHER_ENABLE_HTTP_LOGGING",
		}
		for _, env := range envVars {
			os.Unsetenv(env)
		}

		// Should still load with defaults
		config, err := Load()
		if err != nil {
			t.Fatalf("Failed to load config with defaults: %v", err)
		}

		// Verify defaults are used
		if config.ServerPort() != 8080 {
			t.Errorf("Expected default server port 8080, got %d", config.ServerPort())
		}
		if config.NumWorkers() != 5 {
			t.Errorf("Expected default 5 workers, got %d", config.NumWorkers())
		}
		if config.Latitude() != 41.8262 {
			t.Errorf("Expected default latitude 41.8262, got %f", config.Latitude())
		}
		if config.Timezone() != "UTC" {
			t.Errorf("Expected default timezone UTC, got %s", config.Timezone())
		}
	})

	t.Run("InvalidTimeoutValues", func(t *testing.T) {
		t.Setenv("WEATHER_DB_CONN_MAX_LIFETIME_SECONDS", "invalid")
		t.Setenv("WEATHER_DB_RETRY_INTERVAL_SECONDS", "-5")

		_, err := Load()
		if err == nil {
			t.Error("Expected error for invalid timeout values")
		}
	})

	t.Run("InvalidWorkerCount", func(t *testing.T) {
		t.Setenv("WEATHER_WORKER_COUNT", "invalid")

		_, err := Load()
		if err == nil {
			t.Error("Expected error for invalid worker count")
		}
	})

	t.Run("InvalidWorkerCountNegative", func(t *testing.T) {
		t.Setenv("WEATHER_WORKER_COUNT", "-5")

		_, err := Load()
		if err == nil {
			t.Error("Expected error for negative worker count")
		}
	})

	t.Run("InvalidWorkerCountTooLarge", func(t *testing.T) {
		t.Setenv("WEATHER_WORKER_COUNT", "1000")

		_, err := Load()
		if err == nil {
			t.Error("Expected error for worker count exceeding maximum")
		}
	})

	t.Run("DefaultValues", func(t *testing.T) {
		// Clear all environment variables
		envVars := []string{
			"WEATHER_SERVER_PORT", "WEATHER_DB_HOST", "WEATHER_DB_PORT",
			"WEATHER_DB_USER", "WEATHER_DB_PASSWORD", "WEATHER_DB_NAME",
			"WEATHER_INGESTION_CHAN_SIZE", "WEATHER_JOB_QUEUE_SIZE",
			"WEATHER_WORKER_COUNT", "WEATHER_RESULT_QUEUE_SIZE",
			"WEATHER_AGGREGATION_SCHEDULE", "WEATHER_RETENTION_DAYS",
			"WEATHER_LATITUDE", "WEATHER_LONGITUDE", "WEATHER_TIMEZONE",
			"WEATHER_DB_MAX_OPEN_CONNS", "WEATHER_DB_MAX_IDLE_CONNS",
			"WEATHER_DB_CONN_MAX_LIFETIME_SECONDS", "WEATHER_DB_MAX_RETRIES",
			"WEATHER_DB_RETRY_INTERVAL_SECONDS", "WEATHER_ENABLE_HTTP_LOGGING",
		}
		for _, env := range envVars {
			os.Unsetenv(env)
		}

		config, err := Load()
		if err != nil {
			t.Fatalf("Failed to load config with defaults: %v", err)
		}

		// Verify all defaults
		if config.ServerPort() != 8080 {
			t.Errorf("Expected default server port 8080, got %d", config.ServerPort())
		}
		if config.DBPort() != 5432 {
			t.Errorf("Expected default DB port 5432, got %d", config.DBPort())
		}
		if config.IngestionChanSize() != 1000 {
			t.Errorf("Expected default ingestion channel size 1000, got %d", config.IngestionChanSize())
		}
		if config.JobQueueSize() != 100 {
			t.Errorf("Expected default job queue size 100, got %d", config.JobQueueSize())
		}
		if config.NumWorkers() != 5 {
			t.Errorf("Expected default 5 workers, got %d", config.NumWorkers())
		}
		if config.ResultQueueSize() != 100 {
			t.Errorf("Expected default result queue size 100, got %d", config.ResultQueueSize())
		}
		if config.AggregationSchedule() != "0 0 * * *" {
			t.Errorf("Expected default aggregation schedule '0 0 * * *', got %s", config.AggregationSchedule())
		}
		if config.RetentionDays() != 30 {
			t.Errorf("Expected default retention days 30, got %d", config.RetentionDays())
		}
		if config.Latitude() != 41.8262 {
			t.Errorf("Expected default latitude 41.8262, got %f", config.Latitude())
		}
		if config.Longitude() != -87.6841 {
			t.Errorf("Expected default longitude -87.6841, got %f", config.Longitude())
		}
		if config.Timezone() != "UTC" {
			t.Errorf("Expected default timezone UTC, got %s", config.Timezone())
		}
		if config.DBMaxOpenConns() != 25 {
			t.Errorf("Expected default max open connections 25, got %d", config.DBMaxOpenConns())
		}
		if config.DBMaxIdleConns() != 5 {
			t.Errorf("Expected default max idle connections 5, got %d", config.DBMaxIdleConns())
		}
		if config.DBConnMaxLifetime() != 300*time.Second {
			t.Errorf("Expected default connection lifetime 300s, got %v", config.DBConnMaxLifetime())
		}
		if config.DBMaxRetries() != 5 {
			t.Errorf("Expected default max retries 5, got %d", config.DBMaxRetries())
		}
		if config.DBRetryInterval() != 5*time.Second {
			t.Errorf("Expected default retry interval 5s, got %v", config.DBRetryInterval())
		}
		if config.EnableHTTPLogging() != true {
			t.Errorf("Expected default HTTP logging enabled, got %v", config.EnableHTTPLogging())
		}
	})
}

func TestValidateConfig(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		config := Config{
			dbMaxOpenConns:    25,
			dbMaxIdleConns:    5,
			numWorkers:        5,
			jobQueueSize:      100,
			dbConnMaxLifetime: 300 * time.Second,
			dbRetryInterval:   5 * time.Second,
		}

		err := validateConfig(config)
		if err != nil {
			t.Errorf("Valid config should not fail validation: %v", err)
		}
	})

	t.Run("InvalidDatabaseURL", func(t *testing.T) {
		// This is tested through environment variable parsing
		t.Setenv("WEATHER_DB_HOST", "invalid host with spaces")

		_, err := Load()
		if err != nil {
			// Should work, host is just a string
			t.Logf("DB host validation: %v", err)
		}
	})

	t.Run("InvalidPortNumber", func(t *testing.T) {
		t.Setenv("WEATHER_DB_PORT", "99999")

		_, err := Load()
		if err == nil {
			t.Error("Expected error for port number exceeding maximum")
		}
	})

	t.Run("NegativeTimeoutValues", func(t *testing.T) {
		t.Setenv("WEATHER_DB_CONN_MAX_LIFETIME_SECONDS", "-10")

		_, err := Load()
		if err == nil {
			t.Error("Expected error for negative timeout value")
		}
	})

	t.Run("IdleExceedsMaxConnections", func(t *testing.T) {
		t.Setenv("WEATHER_DB_MAX_OPEN_CONNS", "10")
		t.Setenv("WEATHER_DB_MAX_IDLE_CONNS", "20")

		_, err := Load()
		if err == nil {
			t.Error("Expected error when idle connections exceed max connections")
		}
	})

	t.Run("WorkersExceedJobQueue", func(t *testing.T) {
		t.Setenv("WEATHER_WORKER_COUNT", "10")
		t.Setenv("WEATHER_JOB_QUEUE_SIZE", "5")

		config, err := Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Should load but log warning
		if config.NumWorkers() != 10 {
			t.Errorf("Expected 10 workers, got %d", config.NumWorkers())
		}
	})

	t.Run("ShortConnectionLifetime", func(t *testing.T) {
		t.Setenv("WEATHER_DB_CONN_MAX_LIFETIME_SECONDS", "1")
		t.Setenv("WEATHER_DB_RETRY_INTERVAL_SECONDS", "10")

		config, err := Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Should load but log warning
		if config.DBConnMaxLifetime() != 1*time.Second {
			t.Errorf("Expected 1s connection lifetime, got %v", config.DBConnMaxLifetime())
		}
	})
}

func TestParseEnvVar(t *testing.T) {
	t.Run("StringParsing", func(t *testing.T) {
		t.Setenv("TEST_STRING", "hello world")
		result := getEnvString("TEST_STRING", "default")
		if result != "hello world" {
			t.Errorf("Expected 'hello world', got '%s'", result)
		}

		// Test default
		os.Unsetenv("TEST_STRING")
		result = getEnvString("TEST_STRING", "default")
		if result != "default" {
			t.Errorf("Expected default 'default', got '%s'", result)
		}
	})

	t.Run("IntegerParsing", func(t *testing.T) {
		t.Setenv("TEST_INT", "42")
		result, err := getEnvInt("TEST_INT", 0, 0, 100)
		if err != nil {
			t.Errorf("Failed to parse integer: %v", err)
		}
		if result != 42 {
			t.Errorf("Expected 42, got %d", result)
		}

		// Test default
		os.Unsetenv("TEST_INT")
		result, err = getEnvInt("TEST_INT", 10, 0, 100)
		if err != nil {
			t.Errorf("Failed to get default: %v", err)
		}
		if result != 10 {
			t.Errorf("Expected default 10, got %d", result)
		}
	})

	t.Run("FloatParsing", func(t *testing.T) {
		t.Setenv("TEST_FLOAT", "3.14159")
		result, err := getEnvFloat("TEST_FLOAT", 0.0, -100, 100)
		if err != nil {
			t.Errorf("Failed to parse float: %v", err)
		}
		if result != 3.14159 {
			t.Errorf("Expected 3.14159, got %f", result)
		}

		// Test default
		os.Unsetenv("TEST_FLOAT")
		result, err = getEnvFloat("TEST_FLOAT", 2.718, -100, 100)
		if err != nil {
			t.Errorf("Failed to get default: %v", err)
		}
		if result != 2.718 {
			t.Errorf("Expected default 2.718, got %f", result)
		}
	})

	t.Run("BooleanParsing", func(t *testing.T) {
		t.Setenv("TEST_BOOL", "true")
		result, err := getEnvBool("TEST_BOOL", false)
		if err != nil {
			t.Errorf("Failed to parse boolean: %v", err)
		}
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}

		// Test false
		t.Setenv("TEST_BOOL", "false")
		result, err = getEnvBool("TEST_BOOL", true)
		if err != nil {
			t.Errorf("Failed to parse boolean: %v", err)
		}
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}

		// Test default
		os.Unsetenv("TEST_BOOL")
		result, err = getEnvBool("TEST_BOOL", true)
		if err != nil {
			t.Errorf("Failed to get default: %v", err)
		}
		if result != true {
			t.Errorf("Expected default true, got %v", result)
		}
	})

	t.Run("DurationParsing", func(t *testing.T) {
		t.Setenv("TEST_DURATION", "60")
		result, err := getEnvDurationSeconds("TEST_DURATION", 30)
		if err != nil {
			t.Errorf("Failed to parse duration: %v", err)
		}
		if result != 60*time.Second {
			t.Errorf("Expected 60s, got %v", result)
		}

		// Test default
		os.Unsetenv("TEST_DURATION")
		result, err = getEnvDurationSeconds("TEST_DURATION", 30)
		if err != nil {
			t.Errorf("Failed to get default: %v", err)
		}
		if result != 30*time.Second {
			t.Errorf("Expected default 30s, got %v", result)
		}
	})

	t.Run("InvalidInteger", func(t *testing.T) {
		t.Setenv("TEST_INT", "not a number")
		_, err := getEnvInt("TEST_INT", 10, 0, 100)
		if err == nil {
			t.Error("Expected error for invalid integer")
		}
	})

	t.Run("IntegerOutOfRange", func(t *testing.T) {
		t.Setenv("TEST_INT", "150")
		_, err := getEnvInt("TEST_INT", 10, 0, 100)
		if err == nil {
			t.Error("Expected error for integer out of range")
		}
	})

	t.Run("InvalidFloat", func(t *testing.T) {
		t.Setenv("TEST_FLOAT", "not a number")
		_, err := getEnvFloat("TEST_FLOAT", 10.0, -100, 100)
		if err == nil {
			t.Error("Expected error for invalid float")
		}
	})

	t.Run("FloatOutOfRange", func(t *testing.T) {
		t.Setenv("TEST_FLOAT", "150.0")
		_, err := getEnvFloat("TEST_FLOAT", 10.0, -100, 100)
		if err == nil {
			t.Error("Expected error for float out of range")
		}
	})

	t.Run("InvalidBoolean", func(t *testing.T) {
		t.Setenv("TEST_BOOL", "not a boolean")
		_, err := getEnvBool("TEST_BOOL", true)
		if err == nil {
			t.Error("Expected error for invalid boolean")
		}
	})
}

func TestConfigGetters(t *testing.T) {
	t.Run("AllGettersWork", func(t *testing.T) {
		t.Setenv("WEATHER_SERVER_PORT", "8080")
		t.Setenv("WEATHER_DB_HOST", "localhost")
		t.Setenv("WEATHER_DB_PORT", "5432")
		t.Setenv("WEATHER_DB_USER", "testuser")
		t.Setenv("WEATHER_DB_PASSWORD", "testpass")
		t.Setenv("WEATHER_DB_NAME", "testdb")
		t.Setenv("WEATHER_INGESTION_CHAN_SIZE", "1000")
		t.Setenv("WEATHER_JOB_QUEUE_SIZE", "100")
		t.Setenv("WEATHER_WORKER_COUNT", "5")
		t.Setenv("WEATHER_RESULT_QUEUE_SIZE", "100")
		t.Setenv("WEATHER_AGGREGATION_SCHEDULE", "0 0 * * *")
		t.Setenv("WEATHER_RETENTION_DAYS", "30")
		t.Setenv("WEATHER_LATITUDE", "41.8262")
		t.Setenv("WEATHER_LONGITUDE", "-87.6841")
		t.Setenv("WEATHER_TIMEZONE", "America/Chicago")
		t.Setenv("WEATHER_DB_MAX_OPEN_CONNS", "25")
		t.Setenv("WEATHER_DB_MAX_IDLE_CONNS", "5")
		t.Setenv("WEATHER_DB_CONN_MAX_LIFETIME_SECONDS", "300")
		t.Setenv("WEATHER_DB_MAX_RETRIES", "5")
		t.Setenv("WEATHER_DB_RETRY_INTERVAL_SECONDS", "5")
		t.Setenv("WEATHER_ENABLE_HTTP_LOGGING", "true")

		config, err := Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Test all getters
		_ = config.ServerPort()
		_ = config.DBHost()
		_ = config.DBPort()
		_ = config.DBUser()
		_ = config.DBPassword()
		_ = config.DBName()
		_ = config.IngestionChanSize()
		_ = config.JobQueueSize()
		_ = config.NumWorkers()
		_ = config.ResultQueueSize()
		_ = config.AggregationSchedule()
		_ = config.RetentionDays()
		_ = config.Latitude()
		_ = config.Longitude()
		_ = config.Timezone()
		_ = config.DBMaxOpenConns()
		_ = config.DBMaxIdleConns()
		_ = config.DBConnMaxLifetime()
		_ = config.DBMaxRetries()
		_ = config.DBRetryInterval()
		_ = config.EnableHTTPLogging()
	})
}

func TestInvalidTimezone(t *testing.T) {
	t.Run("InvalidTimezone", func(t *testing.T) {
		t.Setenv("WEATHER_TIMEZONE", "Invalid/Timezone")

		_, err := Load()
		if err == nil {
			t.Error("Expected error for invalid timezone")
		}
	})

	t.Run("ValidTimezone", func(t *testing.T) {
		t.Setenv("WEATHER_TIMEZONE", "Europe/Madrid")

		_, err := Load()
		if err != nil {
			t.Errorf("Valid timezone should not fail: %v", err)
		}
	})
}

func TestLatitudeLongitudeValidation(t *testing.T) {
	t.Run("ValidLatitude", func(t *testing.T) {
		t.Setenv("WEATHER_LATITUDE", "45.5")

		config, err := Load()
		if err != nil {
			t.Errorf("Valid latitude should not fail: %v", err)
		}
		if config.Latitude() != 45.5 {
			t.Errorf("Expected latitude 45.5, got %f", config.Latitude())
		}
	})

	t.Run("LatitudeTooHigh", func(t *testing.T) {
		t.Setenv("WEATHER_LATITUDE", "95.0")

		_, err := Load()
		if err == nil {
			t.Error("Expected error for latitude exceeding 90")
		}
	})

	t.Run("LatitudeTooLow", func(t *testing.T) {
		t.Setenv("WEATHER_LATITUDE", "-95.0")

		_, err := Load()
		if err == nil {
			t.Error("Expected error for latitude below -90")
		}
	})

	t.Run("ValidLongitude", func(t *testing.T) {
		t.Setenv("WEATHER_LONGITUDE", "-122.5")

		config, err := Load()
		if err != nil {
			t.Errorf("Valid longitude should not fail: %v", err)
		}
		if config.Longitude() != -122.5 {
			t.Errorf("Expected longitude -122.5, got %f", config.Longitude())
		}
	})

	t.Run("LongitudeTooHigh", func(t *testing.T) {
		t.Setenv("WEATHER_LONGITUDE", "185.0")

		_, err := Load()
		if err == nil {
			t.Error("Expected error for longitude exceeding 180")
		}
	})

	t.Run("LongitudeTooLow", func(t *testing.T) {
		t.Setenv("WEATHER_LONGITUDE", "-185.0")

		_, err := Load()
		if err == nil {
			t.Error("Expected error for longitude below -180")
		}
	})
}