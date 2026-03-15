# Calibration Feature Implementation

## Overview
Add calibration functionality to the Core application and dashboard to adjust sensor readings, similar to how Ecowitt gateways handle calibration.

## Rain Sensor Calibration - Critical Information

### WS90 Piezo Rain Sensor Working Principle
- Raindrops fall on sensor surface → produces mechanical vibration → sensor generates voltage corresponding to rainfall amount
- **Measurement accuracy is influenced by environmental factors**: wind, terrain, debris, raindrop size
- **Large rainfall intensity**: raindrops may have double impacts → over-reporting
- **Minimal rainfall intensity**: vertical momentum too small → under-reporting
- **Requires different calibration for different rainfall intensities**

### Ecowitt's 5-Level Rain Calibration System

Based on official Ecowitt documentation, the WS90 uses **5 separate calibration gains** for different rain rates:

| Parameter | Rain Rate Range | Description | Recommended Approach |
|-----------|----------------|-------------|----------------------|
| **Piezo Rain1** | < 4 mm/h | Light rain/drizzle | **Keep at 1.0** unless consistently produces same results |
| **Piezo Rain2** | < 10 mm/h | Light to moderate rain | Adjust based on reference data |
| **Piezo Rain3** | < 30 mm/h | Moderate rain | Adjust based on reference data |
| **Piezo Rain4** | < 60 mm/h | Heavy rain | Adjust based on reference data |
| **Piezo Rain5** | > 60 mm/h | Very heavy rain | Adjust based on reference data |

### Calibration Formula
```
Calculated Gain = (Reference Rainfall / WS90 Rainfall) / Current Gain
```

**Example from Ecowitt documentation:**
- WH40 (reference): 6.0mm with rain rate 36mm/hr
- WS90: 7.5mm recorded with gain 1.0
- Calculation: (6.0 / 7.5) = 0.8
- Set Rain4 gain to 0.8

### Dew and Humidity Issues

**Known Problems:**
- **Dew detection**: Haptic sensors frequently record dew as rain, especially at high humidity (90%+)
- **False readings**: Condensation on sensor triggers vibration signals similar to light rain
- **High humidity sensitivity**: Even without actual rain, high humidity can cause false positives

**Current WS90 Limitations:**
- No built-in dew filtering in the sensor hardware
- Default calibration is "almost useless" according to Ecowitt documentation
- Accuracy specifications: ±20% for <5mm/h and >50mm/h, ±10% for 5-50mm/h

### Software-Based Dew Detection

Since we cannot calibrate the WS90 hardware, implement **smart filtering** in the application:

**Primary Rain Detection Logic (using rain_start):**
```go
// PRIMARY FILTER: Use sensor's own rain detection
if reading.RainStart == 0 {
    // Sensor says no rain, ignore rain_mm completely
    // This is the most reliable filter as it uses the sensor's internal logic
    reading.RainMM = 0
    return reading
}

// SECONDARY FILTER: Humidity-based dew detection (backup)
// Only apply if rain_start = 1 (sensor thinks it's raining)
if reading.RainStart == 1 && humidity > 90 && reading.RainMM > 0 && reading.RainMM < 1.0 {
    // Sensor says raining but humidity is very high - likely dew
    // Reduce or filter the reading
    dewCorrectionFactor := 1.0 - ((humidity - 90) * 0.02)
    reading.RainMM = reading.RainMM * dewCorrectionFactor
}

// More aggressive filtering for extreme humidity
if reading.RainStart == 1 && humidity > 95 && reading.RainMM > 0 && reading.RainMM < 2.0 {
    // Sensor says raining but humidity is extremely high - very likely dew
    reading.RainMM = 0.0  // Filter out completely
}
```

**Benefits of using rain_start:**
- More accurate than humidity thresholds alone
- Uses sensor's internal rain detection algorithm
- Reduces false positives from dew, wind, and debris
- Only applies humidity filtering as a secondary check
- Based on real sensor behavior from your live data

**Alternative: Apply dew correction factor (secondary filter only)**
```go
// Only apply humidity correction when rain_start = 1 (sensor thinks it's raining)
if reading.RainStart == 1 && humidity > 90 {
    dewCorrectionFactor := 1.0 - ((humidity - 90) * 0.02)  // Reduces reading
    reading.RainMM = reading.RainMM * dewCorrectionFactor
}
```

### Recommended Calibration Strategy

**Phase 1: Basic Single-Gain Calibration**
1. Start with a single global rain gain (e.g., 0.6-0.8) to reduce overall sensitivity
2. Apply rain_start filtering as primary filter (ignore rain when rain_start = 0)
3. Apply humidity filtering as secondary backup only when rain_start = 1
4. Monitor and adjust based on actual rain events

**Phase 2: Advanced Multi-Rate Calibration**
1. Implement rain rate detection based on recent readings
2. Apply different gains based on detected rain rate
3. Use 5-level system matching Ecowitt's approach

**Phase 3: Smart Dew Filtering**
1. Add rain_start filtering (primary filter using sensor's own rain detection)
2. Add humidity-based dew detection as secondary backup (only when rain_start = 1)
3. Implement time-based filtering (dew typically occurs overnight/early morning)
4. Add wind speed correlation (real rain often has wind)
5. Consider temperature trends (dew occurs during cooling periods)

## Architecture

### 1. Database Schema

**Enhanced calibration table for multi-rate calibration:**

```sql
CREATE TABLE calibration_settings (
    id SERIAL PRIMARY KEY,
    sensor_type VARCHAR(50) NOT NULL,  -- 'rain', 'temperature', etc.
    calibration_value FLOAT NOT NULL DEFAULT 1.0,
    rain_rate_min FLOAT,  -- For multi-rate calibration
    rain_rate_max FLOAT,  -- For multi-rate calibration
    description TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Pre-populate with WS90 rain calibration levels
INSERT INTO calibration_settings (sensor_type, calibration_value, rain_rate_min, rain_rate_max, description) VALUES
('rain_piezo_1', 1.0, 0, 4, 'Light rain/drizzle - keep at 1.0 unless consistent results'),
('rain_piezo_2', 1.0, 4, 10, 'Light to moderate rain'),
('rain_piezo_3', 1.0, 10, 30, 'Moderate rain'),
('rain_piezo_4', 1.0, 30, 60, 'Heavy rain'),
('rain_piezo_5', 1.0, 60, 9999, 'Very heavy rain');
```

**Dew filtering settings:**

```sql
CREATE TABLE dew_filter_settings (
    id SERIAL PRIMARY KEY,
    humidity_threshold FLOAT DEFAULT 90.0,
    rain_threshold FLOAT DEFAULT 1.0,
    rain_start_enabled BOOLEAN DEFAULT true,  -- Enable rain_start primary filter
    humidity_filter_enabled BOOLEAN DEFAULT true,  -- Enable humidity secondary filter
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### 2. Core Application Integration

#### Data Storage Strategy:
- **Store raw sensor data** in the database
- **Apply calibration** when retrieving data for display
- **Preserve original readings** for debugging and recalibration

#### Configuration structure:

```go
type CalibrationConfig struct {
    // Rain calibration gains
    RainPiezo1Gain float64 `env:"RAIN_PIEZO_1_GAIN" default:"1.0"`
    RainPiezo2Gain float64 `env:"RAIN_PIEZO_2_GAIN" default:"1.0"`
    RainPiezo3Gain float64 `env:"RAIN_PIEZO_3_GAIN" default:"1.0"`
    RainPiezo4Gain float64 `env:"RAIN_PIEZO_4_GAIN" default:"1.0"`
    RainPiezo5Gain float64 `env:"RAIN_PIEZO_5_GAIN" default:"1.0"`
    
    // Dew filtering
    DewFilterEnabled bool    `env:"DEW_FILTER_ENABLED" default:"true"`
    RainStartEnabled bool   `env:"RAIN_START_ENABLED" default:"true"`  // Primary filter
    HumidityFilterEnabled bool   `env:"HUMIDITY_FILTER_ENABLED" default:"true"`  // Secondary filter
    DewHumidityThreshold float64 `env:"DEW_HUMIDITY_THRESHOLD" default:"90.0"`
    DewRainThreshold float64 `env:"DEW_RAIN_THRESHOLD" default:"1.0"`
}
```

#### Rain rate detection:

```go
func detectRainRate(recentReadings []types.Reading) float64 {
    // Calculate rain rate from recent readings
    // Return mm/h rate
}

func getRainGain(rate float64, config CalibrationConfig) float64 {
    switch {
    case rate < 4:
        return config.RainPiezo1Gain
    case rate < 10:
        return config.RainPiezo2Gain
    case rate < 30:
        return config.RainPiezo3Gain
    case rate < 60:
        return config.RainPiezo4Gain
    default:
        return config.RainPiezo5Gain
    }
}

func applyDewFilter(reading types.Reading, config CalibrationConfig) (types.Reading, error) {
    if !config.DewFilterEnabled {
        return reading, nil
    }
    
    // PRIMARY FILTER: Use sensor's own rain detection
    if config.RainStartEnabled && reading.RainStart == 0 {
        // Sensor says no rain, return with rain_mm = 0
        reading.RainMM = 0
        return reading, fmt.Errorf("filtered: rain_start = 0")
    }
    
    // SECONDARY FILTER: Humidity-based dew detection (only when rain_start = 1)
    if config.HumidityFilterEnabled && reading.RainStart == 1 &&
       reading.Humidity > config.DewHumidityThreshold && 
       reading.RainMM > 0 && 
       reading.RainMM < config.DewRainThreshold {
        // Sensor says raining but humidity is very high - likely dew
        dewCorrectionFactor := 1.0 - ((reading.Humidity - config.DewHumidityThreshold) * 0.02)
        reading.RainMM = reading.RainMM * dewCorrectionFactor
        return reading, fmt.Errorf("filtered: dew correction applied")
    }
    
    return reading, nil
}

func applyCalibration(reading types.Reading, config CalibrationConfig) types.Reading {
    // Store raw data, return calibrated version for display
    calibratedReading := reading
    
    // Detect rain rate
    rate := detectRainRate([]types.Reading{reading})
    
    // Get appropriate gain
    gain := getRainGain(rate, config)
    
    // Apply calibration to the display value
    calibratedReading.RainMM = reading.RainMM * gain
    
    return calibratedReading
}
```

#### Integration points:

**Data Retrieval (apply calibration for display):**
```go
func (s *Server) getCalibratedReadings(startTime, endTime time.Time) ([]types.Reading, error) {
    // Query raw data from database
    var rawReadings []types.Reading
    query := `SELECT * FROM readings WHERE timestamp BETWEEN $1 AND $2 ORDER BY timestamp ASC`
    
    // Apply calibration to each reading for display
    var calibratedReadings []types.Reading
    for _, raw := range rawReadings {
        // Apply rain_start filtering: ignore rain when sensor says it's not raining
        if raw.RainStart == 0 {
            // Sensor says no rain, set rain to 0
            raw.RainMM = 0
        }
        
        // Apply dew filtering (secondary check using humidity)
        filtered, err := applyDewFilter(raw, s.config)
        if err != nil {
            // Dew filtered, skip or add with rain_mm = 0
            filtered.RainMM = 0
        }
        
        // Apply calibration
        calibrated := applyCalibration(filtered, s.config)
        calibratedReadings = append(calibratedReadings, calibrated)
    }
    
    return calibratedReadings, nil
}
```

### 3. Dashboard UI

#### Calibration settings page structure:

**Rain Calibration Section:**
- 5 separate input fields for Piezo Rain 1-5 gains
- Range: 0.1 to 2.0 (default 1.0)
- Show current rain rate and which gain is being applied
- Real-time preview: show before/after calibration values

**Dew Filter Section:**
- Enable/disable toggle
- Rain Start Filter toggle (primary filter - uses sensor's rain detection)
- Humidity Filter toggle (secondary filter - humidity-based dew detection)
- Humidity threshold slider (80-100%)
- Rain threshold slider (0.1-5.0mm)
- Statistics: show how many readings filtered in last 24h

**Advanced Options:**
- Manual rain rate override for testing
- Calibration history log
- Reset to defaults button

#### API Endpoints:

**GET /api/calibration**
- **Purpose**: Retrieve all current calibration settings
- **Use cases**: 
  - Load calibration values into dashboard settings form
  - Display current calibration status to user
  - Verify calibration settings after updates
- **Response**: JSON object with all calibration parameters
  ```json
  {
    "rain_piezo_1": 1.0,
    "rain_piezo_2": 0.8,
    "rain_piezo_3": 0.8,
    "rain_piezo_4": 0.8,
    "rain_piezo_5": 0.8,
    "dew_filter_enabled": true,
    "rain_start_enabled": true,
    "humidity_filter_enabled": true,
    "dew_humidity_threshold": 90.0,
    "dew_rain_threshold": 1.0
  }
  ```

**PUT /api/calibration**
- **Purpose**: Update calibration settings
- **Use cases**:
  - User adjusts rain gain values in dashboard
  - User enables/disables dew filtering
  - User changes dew threshold parameters
  - Bulk update of multiple calibration values
- **Request Body**: JSON object with calibration parameters to update
- **Response**: Updated calibration settings
- **Validation**: Ensure values are within acceptable ranges (0.1-2.0 for gains, 80-100 for humidity threshold)

## Implementation Plan

### Phase 1: Basic Implementation
1. Add single rain gain to config
2. Implement rain_start filtering (primary filter)
3. Implement basic humidity-based dew filtering (secondary)
4. Add simple UI with one gain control
5. Deploy and monitor

### Phase 2: Multi-Rate Calibration
1. Implement 5-level rain rate detection
2. Add database support for multiple gains
3. Update UI with 5 gain controls
4. Add rain rate display

### Phase 3: Advanced Features
1. Add calibration history and logging
2. Implement smart dew detection with multiple factors
3. Add calibration statistics and reporting
4. Create calibration wizard for new users

## Best Practices

1. **Start conservative**: Begin with lower gains (0.6-0.8) to reduce false positives
2. **Monitor closely**: Track filtered readings vs. actual rain events
3. **Use reference data**: Compare with nearby weather stations when possible
4. **Document changes**: Keep a log of calibration adjustments and reasons
5. **Test in different conditions**: Verify calibration works for light, moderate, and heavy rain

## Data Flow
1. **Ingestor**: Receives raw data from WS90 → Stores in database (no calibration)
2. **Database**: Contains raw sensor readings
3. **Core API**: Retrieves raw data → Applies calibration & dew filtering → Returns to dashboard
4. **Dashboard**: Displays calibrated values to user
5. **Calibration changes**: Affect all data retrieval going forward (no need to modify stored data)

## Notes
- WS90's default calibration is poor and needs adjustment
- Dew detection is critical for accurate readings in humid climates
- Multi-rate calibration provides better accuracy across different rain intensities
- Software filtering can compensate for hardware limitations
- Calibration values are multipliers: 0.6 = 40% reduction, 1.5 = 50% increase
- Raw data preservation allows recalibration of historical data
- Performance impact is negligible on modern hardware
- Always validate calibration against real rain events when possible

---

# Testing Implementation Plan

## Overview
Implement comprehensive testing across the weather station project to ensure code quality, catch regressions, and enable safe refactoring.

## Difficulty Assessment

### Easy to Test
- Go utility functions (validation, conversions, helpers in shared)
- Business logic (rain calculations, weather icon logic)
- Configuration handling
- Logger functionality
- JavaScript utility functions (unit conversions, CSV export)

### Medium Difficulty
- HTTP handlers (need httptest package)
- Database queries (need test database setup)
- Worker pool logic
- API integration tests

### Hard to Test
- Database functions (compute_daily_rollup requires full PostgreSQL)
- Async goroutines (daily aggregator, concurrent operations)
- Frontend E2E (requires browser automation)
- Full system integration (end-to-end data flow)

## Phase 1: Unit Tests for Shared Utilities

### Target Files
- `shared/validation/validation.go`
- `shared/workers/pool.go`
- `shared/logger/logger.go`

### Test Cases to Implement

**Validation Tests (`shared/validation/validation_test.go`):**
```go
func TestValidateReading(t *testing.T) {
    // Test valid reading - DONE
    // Test invalid temperature (extreme values) - DONE
    // Test invalid humidity (0-100 range) - DONE
    // Test invalid wind speed (negative values) - DONE
    // Test missing required fields - DONE
}
```

**Status:** All 20 tests implemented and passing (100% coverage)

**Worker Pool Tests (`shared/workers/pool_test.go`):**
```go
func TestStartWorkerPool(t *testing.T) {
    // Test pool creation with valid config - DONE
    // Test pool with zero workers - DONE
    // Test pool with negative queue size - DONE
    // Test pool with single worker - DONE
    // Test pool with multiple workers - DONE
    // Test pool with zero queue size - DONE
}

func TestPoolGetJobChan(t *testing.T) {
    // Test job channel retrieval - DONE
}

func TestPoolGetResultChan(t *testing.T) {
    // Test result channel retrieval - DONE
    // Test channel buffer size - DONE
}

func TestPoolWait(t *testing.T) {
    // Test waiting for workers to finish - DONE
    // Test wait with no jobs - DONE
}

func TestPoolShutdown(t *testing.T) {
    // Test graceful shutdown - DONE
    // Test shutdown with active jobs - DONE
    // Test multiple shutdowns panic - DONE
}

func TestPoolJobProcessing(t *testing.T) {
    // Test successful job processing - DONE
    // Test job processing with errors - DONE
}

func TestPoolPanicRecovery(t *testing.T) {
    // Test worker panic recovery - DONE
}

func TestPoolConcurrency(t *testing.T) {
    // Test concurrent job submission - DONE
}
```

**Status:** All 17 tests implemented and passing (96.6% coverage)

**Logger Tests (`shared/logger/logger_test.go`):**
```go
func TestLoggerInfo(t *testing.T) {
    // Test INFO level logging - DONE
    // Test log format - DONE
    // Test with multiple arguments - DONE
    // Test with empty message - DONE
}

func TestLoggerError(t *testing.T) {
    // Test ERROR level logging - DONE
    // Test log format - DONE
    // Test with multiple arguments - DONE
    // Test with nil error - DONE
}

func TestLoggerPlain(t *testing.T) {
    // Test plain logging without tags - DONE
    // Test log format - DONE
    // Test with empty message - DONE
    // Test with special characters - DONE
}

func TestLoggerColorCodes(t *testing.T) {
    // Test Info contains green color - DONE
    // Test Error contains red color - DONE
    // Test Plain contains white color - DONE
}

func TestLoggerConcurrentLogging(t *testing.T) {
    // Test concurrent logging from multiple goroutines - DONE
}

func TestLoggerMixedLevels(t *testing.T) {
    // Test mix of Info, Error, and Plain logs - DONE
}
```

**Status:** All 17 tests implemented and passing (100% coverage)

**How to run tests:**
```bash
# Run all tests in the shared package
cd shared
go test -v ./...

# Run specific test function
go test -v ./workers/... -run TestStartWorkerPool

# Run specific test case (subtest)
go test -v ./workers/... -run "TestStartWorkerPool/ValidConfig"

# Run tests with coverage
go test -v -cover ./...
```

**Phase 1 Summary:** COMPLETED
- Validation: 20 tests, 100% coverage
- Worker Pool: 17 tests, 96.6% coverage
- Logger: 17 tests, 100% coverage
- Total: 54 tests, excellent coverage

---

## Phase 2: Handler Tests with httptest

### Target Files
- `core/handlers/handlers.go`

### Test Cases to Implement

**Ingest Handler:**
```go
func TestIngestHandler(t *testing.T) {
    // Test successful ingestion - DONE
    // Test invalid JSON - DONE
    // Test wrong Content-Type - DONE
    // Test invalid reading data - DONE
    // Test channel full - DONE
}
```

**Method Check Middleware:**
```go
func TestMethodCheck(t *testing.T) {
    // Test allowed method - DONE
    // Test disallowed method - DONE
}
```

**Health Check Handler:**
```go
func TestHealthCheckHandler(t *testing.T) {
    // Test healthy database - DONE (skipped without PostgreSQL)
    // Test unhealthy database - DONE
}
```

**Current Weather Handler:**
```go
func TestCurrentWeatherHandler(t *testing.T) {
    // Test successful response - DONE (skipped without PostgreSQL)
    // Test database error handling - DONE (skipped without PostgreSQL)
    // Test missing data scenario - DONE (skipped without PostgreSQL)
    // Test JSON response structure - DONE (skipped without PostgreSQL)
}
```

**History Weather Handler:**
```go
func TestHistoryWeatherHandler(t *testing.T) {
    // Test invalid date format - DONE (skipped without PostgreSQL)
    // Test no data available - DONE (skipped without PostgreSQL)
    // Test successful response - DONE (skipped without PostgreSQL)
}
```

**Available Years Handler:**
```go
func TestAvailableYearsHandler(t *testing.T) {
    // Test no data available - DONE (skipped without PostgreSQL)
    // Test successful response with years - DONE (skipped without PostgreSQL)
}
```

**Recent Weather Handler:**
```go
func TestRecentWeatherHandler(t *testing.T) {
    // Test invalid date format - DONE (skipped without PostgreSQL)
    // Test invalid limit - DONE (skipped without PostgreSQL)
    // Test no data available - DONE (skipped without PostgreSQL)
    // Test successful response - DONE (skipped without PostgreSQL)
    // Test custom limit - DONE (skipped without PostgreSQL)
}
```

**Update Astronomical Data:**
```go
func TestUpdateAstronomicalData(t *testing.T) {
    // Test successful update - DONE (skipped without PostgreSQL)
    // Test concurrent updates - DONE (skipped without PostgreSQL)
}
```

**Calculate Rain with Reset Detection:**
```go
func TestCalculateRainWithResetDetection(t *testing.T) {
    // Test no readings - DONE (skipped without PostgreSQL)
    // Test normal progression (no reset) - DONE (skipped without PostgreSQL)
    // Test sensor reset detected - DONE (skipped without PostgreSQL)
}
```

**Logging Middleware:**
```go
func TestLoggingMiddleware(t *testing.T) {
    // Test logging enabled - DONE
    // Test logging disabled - DONE
}
```

**Status:** All 23 tests implemented and passing (16% coverage due to PostgreSQL dependency)

### Setup Required
```go
// Create test server
func setupTestServer(t *testing.T) (*httptest.Server, *sql.DB) {
    // Setup mock database
    // Create test server with handlers
    // Return server and DB for cleanup
}
```

**Phase 2 Summary:** COMPLETED
- 42 tests implemented covering all handlers
- Tests use httptest for HTTP testing
- All tests passing with 77.4% coverage
- Added comprehensive tests for rain detection functionality:
  - TestIsCurrentlyRaining (6 subtests): Tests rain detection based on 5-minute window
    - NoDataAvailable: Returns 0 when no readings exist
    - NoRainInLast5Minutes: Returns 0 when readings are outside the 5-minute window
    - RainAboveThreshold: Returns 1 when rain > 0.1mm in last 5 minutes
    - RainBelowThreshold: Returns 0 when rain < 0.1mm (sensor noise threshold)
    - SensorResetDuringRain: Correctly handles sensor resets and only counts rain after reset
    - NonZeroCounterNoChange: Handles non-zero counter values with no change
  - TestCurrentWeatherHandler additions (2 subtests): Tests RainStart calculation integration
- Added system info functionality tests:
  - TestRTL433VersionHandler (4 subtests): Tests rtl_433 version endpoint
    - ValidVersion: Accepts and stores valid version string
    - EmptyVersion: Handles empty version gracefully
    - InvalidJSON: Handles malformed JSON
    - MissingVersionField: Handles missing version field
  - TestSystemInfoHandler (2 subtests): Tests system info endpoint
    - NoVersionSet: Returns empty string when no version set
    - VersionSet: Returns stored version string
- Rain detection logic correctly handles sensor resets and 0.1mm threshold
- Mutex protection for concurrent access to rtl_433 version

---

## Phase 3: Database Integration Tests

### Target Files
- `core/database/database.go`
- `migrations/*.sql`

### Test Cases to Implement

**Database Connection Tests:**
```go
func TestDatabaseConnection(t *testing.T) {
    // Test successful connection
    // Test connection failure
    // Test connection pool
}
```

**Insert Reading Tests:**
```go
func TestInsertReading(t *testing.T) {
    // Test successful insert
    // Test duplicate handling
    // Test NULL values
    // Test timestamp handling
}
```

**Query Tests:**
```go
func TestGetRecentReadings(t *testing.T) {
    // Test limit parameter
    // Test empty result
    // Test ordering
}
```

**Daily Rollup Function Tests:**
```go
func TestComputeDailyRollup(t *testing.T) {
    // Test with readings (rain_start = 1)
    // Test with readings (rain_start = 0)
    // Test with no readings (safeguard)
    // Test sensor reset handling
    // Test historical data preservation
}
```

### Setup Required
```go
// Test database setup
func setupTestDatabase(t *testing.T) *sql.DB {
    // Connect to test PostgreSQL instance
    // Run migrations
    // Seed test data
    // Return DB connection
}
```

### Test Database Configuration
Use local Postgres database (host=127.0.0.1:5432, user=postgres, password=postgres, dbname=weather_station).

### Status: COMPLETED

**Tests Implemented:** 15 subtests, all passing

**Test Coverage:** 83.1% of statements

**Summary:**
- TestConnect (3 subtests): Database connection with connection pool configuration, error handling for invalid host and credentials
- TestConnectWithRetry (2 subtests): Connection with retry logic, failure after max retries
- TestPrepareInsertStatement (2 subtests): Preparing INSERT statement, panic handling for nil database
- TestInsertReading (4 subtests): Inserting readings with different timestamp formats (RFC3339Nano, rtl_433), error handling for invalid timestamps
- TestComputeDailyRollup (2 subtests): Triggering daily rollup function, handling no data for specified date
- TestCleanOldReadings (2 subtests): Deleting old readings based on retention policy, handling no old readings

**Key Features:**
- Tests use local PostgreSQL database
- Graceful skip when database unavailable
- Comprehensive error handling testing
- Tests both success and failure cases
- Tests different timestamp formats
- Tests database functions (rollup, cleanup)
- Tests connection pool configuration
- Tests retry logic

---

## Phase 4: E2E Tests for Critical User Flows

### Target Flows

**1. Data Ingestion Flow:**
```go
func TestDataIngestionFlow(t *testing.T) {
    // Start ingestor
    // Send mock sensor data
    // Verify data stored in database
    // Verify API returns data
}
```

**2. Daily Rollup Flow:**
```go
func TestDailyRollupFlow(t *testing.T) {
    // Insert test readings for a day
    // Trigger daily rollup
    // Verify daily_weather table updated
    // Verify historical data preserved
}
```

**3. Dashboard Data Display:**
```go
func TestDashboardDataDisplay(t *testing.T) {
    // Seed test data
    // Start core server
    // Call current weather API
    // Verify response structure
    // Verify rain_start field included
}
```

### Ingestor Unit Tests ✅ COMPLETED

**Target Files:**
- `ingestor/main.go`

**Test Cases Implemented:**

**RTL433 Version Tests (4 subtests):**
```go
func TestSendRTL433Version(t *testing.T) {
    // Test successful version retrieval and sending - DONE
    // Test version parsing with regex - DONE
    // Test timeout handling (5-second timeout) - DONE
    // Test retry logic (3 attempts with 2-second delay) - DONE
    // Test error handling when rtl_433 command fails - DONE
    // Test empty version string handling - DONE
    // Test malformed version string handling - DONE
}
```

**Version Parsing Tests (5 subtests):**
```go
func TestParseRTL433Version(t *testing.T) {
    // Test parsing version with date (e.g., "25.12 (2025-12-12)") - DONE
    // Test parsing version without date (e.g., "25.12") - DONE
    // Test parsing full line fallback - DONE
    // Test empty string handling - DONE
    // Test invalid format handling - DONE
}
```

**Send to Core Tests (3 subtests):**
```go
func TestSendToCore(t *testing.T) {
    // Test successful send - DONE
    // Test retry logic - DONE
    // Test network errors - DONE
}
```

**Config Tests (4 subtests):**
```go
func TestLoadConfig(t *testing.T) {
    // Test valid config - DONE
    // Test invalid timeout - DONE
    // Test invalid worker count - DONE
    // Test missing required config - DONE (skipped due to os.Exit)
}
```

**Helper Function Tests (3 subtests):**
```go
func TestCalculateBackoff(t *testing.T) {
    // Test exponential backoff calculation - DONE
    // Test max backoff capping - DONE
}

func TestMin(t *testing.T) {
    // Test min helper function - DONE
}
```

**Processing Tests (2 subtests):**
```go
func TestProcessStdout(t *testing.T) {
    // Test valid JSON reading - DONE
    // Test invalid JSON reading - DONE
}
```

**Status: All 21 subtests passing, 33.0% coverage**

**Note:** Created `ingestor/main_test.go` file with comprehensive tests
```

### Core Astronomical Unit Tests ⏳ NOT STARTED

**Target Files:**
- `core/astronomical/astronomical.go`

**Test Cases to Implement:**

**Sunrise/Sunset Calculation Tests (6 subtests):**
```go
func TestCalculateSunriseSunset(t *testing.T) {
    // Test sunrise calculation for known date/location - TODO
    // Test sunset calculation for known date/location - TODO
    // Test calculation at equinox - TODO
    // Test calculation at solstice - TODO
    // Test calculation at different latitudes - TODO
    // Test calculation at different longitudes - TODO
}
```

**Day/Night Detection Tests (4 subtests):**
```go
func TestIsDaytime(t *testing.T) {
    // Test daytime detection - TODO
    // Test nighttime detection - TODO
    // Test edge case at sunrise - TODO
    // Test edge case at sunset - TODO
}
```

**Timezone Handling Tests (3 subtests):**
```go
func TestTimezoneConversion(t *testing.T) {
    // Test UTC to local conversion - TODO
    // Test local to UTC conversion - TODO
    // Test daylight saving time handling - TODO
}
```

**Note:** Core astronomical currently has 0% coverage. Need to create `core/astronomical/astronomical_test.go` file.

### Core Config Unit Tests ⏳ NOT STARTED

**Target Files:**
- `core/config/config.go`

**Test Cases to Implement:**

**Config Loading Tests (5 subtests):**
```go
func TestLoadConfig(t *testing.T) {
    // Test valid config from environment variables - TODO
    // Test missing required environment variables - TODO
    // Test invalid timeout values - TODO
    // Test invalid worker count values - TODO
    // Test default values for optional config - TODO
}
```

**Config Validation Tests (4 subtests):**
```go
func TestValidateConfig(t *testing.T) {
    // Test valid config - TODO
    // Test invalid database URL - TODO
    // Test invalid port number - TODO
    // Test negative timeout values - TODO
}
```

**Environment Variable Parsing Tests (3 subtests):**
```go
func TestParseEnvVar(t *testing.T) {
    // Test string parsing - TODO
    // Test integer parsing - TODO
    // Test boolean parsing - TODO
}
```

**Note:** Core config currently has 0% coverage. Need to create `core/config/config_test.go` file.

### Frontend E2E Tests (Optional)
```javascript
// Using Playwright or Cypress
describe('Weather Dashboard', () => {
    test('displays current weather', async () => {
        // Navigate to dashboard
        // Verify temperature displayed
        // Verify wind direction shows compass letters
        // Verify rain icon only shows when raining
    });

    test('exports CSV correctly', async () => {
        // Navigate to history tab
        // Select date range
        // Click export button
        // Verify CSV format and content
    });
});
```

### Setup Required
```go
// Test environment setup
func setupTestEnvironment(t *testing.T) (*sql.DB, *http.Client) {
    // Start test database
    // Run migrations
    // Seed test data
    // Start test server
    // Return DB and HTTP client
}
```

---

## Testing Infrastructure

### Required Tools
- Go testing framework (built-in)
- testify/assert for assertions
- httptest for HTTP handler tests
- testcontainers-go for Docker test databases
- Playwright or Cypress for frontend E2E tests

---

## Best Practices

1. **Start Small**: Begin with easy unit tests to build momentum
2. **Test Behavior, Not Implementation**: Focus on what code does, not how
3. **Use Table-Driven Tests**: For multiple test cases with same logic
4. **Mock External Dependencies**: Database, APIs, file system
5. **Keep Tests Fast**: Unit tests should run in milliseconds
6. **Test Edge Cases**: Null values, empty inputs, extreme values
7. **Use Descriptive Names**: Test names should explain what they test
8. **Avoid Test Interdependence**: Each test should be independent
9. **Clean Up Resources**: Close connections, delete temp files
10. **Run Tests Locally Before Pushing**: Catch issues early

---

# MeteoClimatic Integration

## Overview
Implement a module to send weather data from the WS90 ingestor to MeteoClimatic (https://wiki.meteoclimatic.net/wiki/P%c3%a1gina_principal), allowing users to optionally share their weather station data with the MeteoClimatic network.

## MeteoClimatic API Overview

MeteoClimatic supports **three protocols** for data submission:

1. **Ecowitt Protocol** - Used by Fine Offset devices (including WS90)
2. **Wunderground Protocol** - Used by various station brands
3. **WeatherLink Live Protocol** - Used by Davis stations

## Recommended Approach: Wunderground Protocol

Use the **Wunderground Protocol** for MeteoClimatic integration. This protocol is well-documented, widely supported, and provides all the necessary fields for weather data submission.

## Prerequisites

1. **Register your station** at MeteoClimatic (https://www.meteoclimatic.net)
2. **Obtain credentials** from your station profile:
   - Station Code (e.g., `ESMAD2800000028576Z`)
   - API Key (e.g., `xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`)
3. **Configure timezone** to UTC (important for daily rain accumulation)
4. **Verify altitude** in your station profile (used for pressure calculations)

## API Configuration Helper

Use https://api.m11c.net/ to generate the exact API endpoint and parameters for your station:
1. Select "Protocolo de envío" → "Wunderground"
2. Enter your Station Code
3. Enter your API Key
4. Enter Pressure Offset (or use *W to use station's pressure)
5. The tool will generate the exact endpoint URL and parameters

## Data Submission Format

Based on the Wunderground protocol, send HTTP **GET** requests with query parameters:

**Example URL:**
```
https://api.m11c.net/v2/wu?ID=ESMAD2800000028576Z*W&PASSWORD=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx&tempf=75.0&windspeedmph=5.5&winddir=180&baromin=29.92&absbaromin=29.95&humidity=55&solarradiation=500&UV=5&dailyrainin=0.00&monthlyrainin=2.50&yearlyrainin=15.75&softwaretype=weather_station_ingestor
```

**Protocol Details:**
- **HTTP Method:** GET
- **Endpoint:** `https://api.m11c.net/v2/wu`
- **Data Format:** Query parameters (URL-encoded)
- **Authentication:** ID and PASSWORD parameters

## Required Parameters

**Authentication:**
- `ID` (string) - Station code with optional pressure offset
  - Format: `STATIONCODE*OFFSET` or `STATIONCODE*W`
  - Example: `ESMAD2800000028576Z*W` (use station's pressure)
  - Example: `ESMAD2800000028576Z*70` (subtract 70 hPa from relative pressure)
- `PASSWORD` (string) - API key from MeteoClimatic

**Mandatory Weather Data:**
- `tempf` (float) - Outdoor temperature in Fahrenheit
- `humidity` (int) - Outdoor humidity percentage (0-100)
- `dateutc` (string) - Timestamp in UTC (format: "YYYY-MM-DD HH:MM:SS")

**Optional but Recommended:**
- `windspeedmph` (float) - Wind speed in mph
- `winddir` (int) - Wind direction in degrees (0-360)
- `baromin` (float) - Barometric pressure in inches of mercury (relative)
- `absbaromin` (float) - Absolute barometric pressure in inches of mercury
- `dailyrainin` (float) - Daily rain in inches
- `softwaretype` (string) - Software name/type identifier

## Extended Parameters (Ecowitt Extensions)

MeteoClimatic supports additional parameters beyond standard Wunderground:

**Solar & UV:**
- `solarradiation` (float) - Solar radiation (W/m²)
- `UV` (int) - UV index

**Rain Totals:**
- `monthlyrainin` (float) - Monthly rain in inches
- `yearlyrainin` (float) - Yearly rain in inches

**Additional Rain:**
- `rainratein` (float) - Rain rate in inches/hour
- `hourlyrainin` (float) - Hourly rain in inches

**Wind:**
- `windgustmph` (float) - Wind gust in mph

**Temperature:**
- `dewptf` (float) - Dew point in Fahrenheit

## Field Comparison: Reading Struct vs Wunderground Protocol

### Fields Available in Reading Struct ✅

| Reading Field | Wunderground Field | Type | Notes |
|---------------|-------------------|------|-------|
| `TemperatureC` | `tempf` | float (°F) | ✅ Available, needs °C → °F conversion |
| `Humidity` | `humidity` | int (%) | ✅ Direct match |
| `WindSpeedMS` | `windspeedmph` | float (mph) | ✅ Available, needs m/s → mph conversion |
| `WindDirDeg` | `winddir` | int (degrees) | ✅ Direct match |
| `RainMM` | `dailyrainin` | float (inches) | ✅ Available, needs mm → inches conversion |
| `UVIndex` | `UV` | int | ✅ Direct match |
| `Lux` | `solarradiation` | float (W/m²) | ✅ Available, needs lux → W/m² conversion |
| `Time` | `dateutc` | string | ✅ Available, needs format conversion |

### Missing from Reading Struct ❌

| Wunderground Field | Type | Status | Priority |
|--------------------|------|--------|----------|
| `absbaromin` | float (inches) | ❌ **MISSING** | **CRITICAL** - Required for MeteoClimatic (absolute pressure) |
| `monthlyrainin` | float (inches) | ❌ Missing | Optional - Monthly rain total |
| `yearlyrainin` | float (inches) | ❌ Missing | Optional - Yearly rain total |
| `windgustmph` | float (mph) | ❌ Missing | Optional - Wind gust |
| `rainratein` | float (inches/hour) | ❌ Missing | Optional - Rain rate |
| `hourlyrainin` | float (inches) | ❌ Missing | Optional - Hourly rain |
| `dewptf` | float (°F) | ❌ Missing | Optional - Dew point |

**Note:** MeteoClimatic PREFERS absolute pressure (`absbaromin`) and will calculate relative pressure itself based on altitude and temperature. Do NOT send `baromin` unless you have a specific reason.

### Additional Fields in Reading (Not Used by Wunderground)

- `Model` - Not needed
- `SensorID` - Not needed
- `RainStart` - Not needed (used for local dew filtering only)
- `BatteryOK` - Not needed
- `Firmware` - Not needed
- `WindGustMS` - Not directly used (would convert to `windgustmph`)

## Critical Issue: Missing Barometric Pressure

**The WS90 Reading struct does NOT include barometric pressure**, which is a **required field** for MeteoClimatic submission.

## Solution: Open-Meteo API for Barometric Pressure

### MeteoClimatic Pressure Requirements

**MeteoClimatic PREFERS absolute pressure** and will calculate relative pressure itself based on:
- Absolute pressure from the sensor
- Station altitude (configured in MeteoClimatic profile)
- Outdoor temperature

From the documentation: "MeteoClimatic calcula la presión atmosférica correcta para cada estación en función de la presión absoluta registrada por el sensor de la consola, la altitud de la consola y la temperatura exterior"

**Important:** Absolute pressure values should NOT be altered with calibration constants or offsets.

### Recommended Approach: Use Open-Meteo API

Fetch absolute barometric pressure from Open-Meteo API based on your coordinates:

**API Details:**
- **Free API**: No API key required
- **Endpoint:** `https://api.open-meteo.com/v1/forecast`
- **Parameters:** `latitude`, `longitude`, `current=surface_pressure`
- **Response:** Returns absolute surface pressure in hPa

**Example Request:**
```
GET https://api.open-meteo.com/v1/forecast?latitude=42.0625&longitude=-7.6875&current=surface_pressure
```

**Example Response:**
```json
{
  "current": {
    "time": "2026-03-13T22:15",
    "interval": 900,
    "surface_pressure": 985.9
  }
}
```

**Implementation:**
```go
// Returns absolute pressure in inches of mercury
func fetchPressureFromAPI(lat, lon float64) (absolute float64, err error) {
    url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%.6f&longitude=%.6f&current=surface_pressure", lat, lon)
    resp, err := http.Get(url)
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()
    
    var result struct {
        Current struct {
            SurfacePressure float64 `json:"surface_pressure"`
        } `json:"current"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return 0, err
    }
    
    // Convert hPa to inches of mercury (1 hPa = 0.02953 inHg)
    // Example: 985.9 hPa * 0.02953 = 29.10 inHg
    absolute = result.Current.SurfacePressure * 0.02953
    
    return absolute, nil
}
```

### ID Field Configuration

Use `*W` in the ID field to let MeteoClimatic calculate relative pressure from absolute pressure:
- **ID Format**: `STATIONCODE*W`
- **Example**: `ESMAD2800000028576Z*W`
- **Effect**: MeteoClimatic will use the absolute pressure you send and calculate relative pressure based on altitude

**Alternative**: If you know your pressure offset, use `*OFFSET`:
- **ID Format**: `STATIONCODE*OFFSET`
- **Example**: `ESMAD2800000028576Z*70` (subtract 70 hPa from relative to calculate absolute)

## Implementation Strategy for Ingestor

### 1. Add New Module

Create `ingestor/meteoclimatic.go` with:
- Function to transform Reading struct to Wunderground protocol format
- Function to send HTTP GET request to MeteoClimatic API endpoint
- Error handling and retry logic
- Pressure fetching from Open-Meteo API
- Rain totals tracking (monthly, yearly)
- **Rate limiter to enforce 60-second minimum interval between submissions**

### 2. Configuration

Add to config structure:
```go
type Config struct {
    // MeteoClimatic integration
    MeteoClimaticEnabled bool
    MeteoClimaticStationCode string
    MeteoClimaticAPIKey string
    MeteoClimaticLatitude float64
    MeteoClimaticLongitude float64
    MeteoClimaticPressureSource string // "api" or "calculated"
    MeteoClimaticAltitude float64 // For calculated pressure fallback
    MeteoClimaticSoftwareType string // e.g., "weather_station_ingestor"
    MeteoClimaticUpdateInterval time.Duration // Minimum interval between submissions (default: 60 seconds)
}
```

### 3. Integration Point

In `ingestor/main.go`, after processing a reading:
```go
if config.MeteoClimaticEnabled {
    go meteoclimatic.SendReading(reading, config)
}
```

**IMPORTANT**: Implement rate limiting to ensure data is only sent to MeteoClimatic every 60 seconds minimum. Do not send data more frequently than this.

### 4. Data Mapping

```go
func transformReadingToWunderground(reading types.Reading, absolutePressure float64, monthlyRain, yearlyRain float64) string {
    // Build query parameters
    params := url.Values{}
    
    // Authentication
    params.Add("ID", config.MeteoclimaticStationCode + "*W") // Use *W to let MeteoClimatic calculate relative pressure
    params.Add("PASSWORD", config.MeteoclimaticAPIKey)
    
    // Mandatory fields
    tempf := reading.TemperatureC * 9/5 + 32
    params.Add("tempf", fmt.Sprintf("%.1f", tempf))
    params.Add("humidity", fmt.Sprintf("%d", reading.Humidity))
    
    // Format timestamp as YYYY-MM-DD HH:MM:SS
    params.Add("dateutc", reading.Time)
    
    // Optional fields
    if reading.WindSpeedMS > 0 {
        windspeedmph := reading.WindSpeedMS * 2.23694 // m/s to mph
        params.Add("windspeedmph", fmt.Sprintf("%.1f", windspeedmph))
    }
    
    if reading.WindDirDeg >= 0 {
        params.Add("winddir", fmt.Sprintf("%d", reading.WindDirDeg))
    }
    
    // Send absolute pressure (MeteoClimatic will calculate relative pressure)
    if absolutePressure > 0 {
        params.Add("absbaromin", fmt.Sprintf("%.2f", absolutePressure))
    }
    
    if reading.RainMM > 0 {
        dailyrainin := reading.RainMM * 0.0393701 // mm to inches
        params.Add("dailyrainin", fmt.Sprintf("%.2f", dailyrainin))
    }
    
    if reading.UVIndex > 0 {
        params.Add("UV", fmt.Sprintf("%.1f", reading.UVIndex))
    }
    
    if reading.Lux > 0 {
        // Convert lux to W/m² (approximate: 1 lux ≈ 0.0079 W/m² for solar radiation)
        solarradiation := reading.Lux * 0.0079
        params.Add("solarradiation", fmt.Sprintf("%.0f", solarradiation))
    }
    
    if monthlyRain > 0 {
        params.Add("monthlyrainin", fmt.Sprintf("%.2f", monthlyRain))
    }
    
    if yearlyRain > 0 {
        params.Add("yearlyrainin", fmt.Sprintf("%.2f", yearlyRain))
    }
    
    params.Add("softwaretype", config.MeteoclimaticSoftwareType)
    
    return fmt.Sprintf("https://api.m11c.net/v2/wu?%s", params.Encode())
}
```

### 5. Rain Totals Tracking

Since Wunderground protocol requires monthly and yearly rain totals, implement a tracking mechanism:

```go
type RainTotals struct {
    Monthly float64
    Yearly float64
    LastUpdate time.Time
    CurrentMonth int
    CurrentYear int
}

func (r *RainTotals) Update(reading types.Reading) {
    now := time.Now()
    
    // Reset monthly total if month changed
    if now.Month() != r.CurrentMonth || now.Year() != r.CurrentYear {
        r.Monthly = 0
        r.CurrentMonth = int(now.Month())
        r.CurrentYear = now.Year()
    }
    
    // Add rain to totals (convert mm to inches)
    rainInches := reading.RainMM * 0.0393701
    r.Monthly += rainInches
    r.Yearly += rainInches
    r.LastUpdate = now
}
```

## Key Considerations

### Unit Conversions
- Temperature: °C → °F (multiply by 9/5, add 32)
- Wind Speed: m/s → mph (multiply by 2.23694)
- Rain: mm → inches (multiply by 0.0393701)
- Pressure: hPa → inches of mercury (multiply by 0.02953)
- Solar Radiation: lux → W/m² (multiply by 0.0079, approximate)

### Data Quality
- Use `*W` in ID field to let MeteoClimatic calculate pressure from altitude
- Or use `*OFFSET` with known pressure offset if you have it
- Skip optional fields if data is unavailable (don't send zeros)
- Always send mandatory fields (tempf, humidity, dateutc)

### Error Handling
- Implement retry logic for network failures (3-5 retries with exponential backoff)
- Log failed submissions for manual review
- Don't block local storage if MeteoClimatic fails
- Handle HTTP errors gracefully (4xx, 5xx responses)

### Rate Limiting
- **IMPORTANT**: Only send data to MeteoClimatic every 60 seconds minimum
- Do not send data more frequently than every 60 seconds
- Implement a rate limiter or timer to enforce 60-second minimum interval
- Consider batching if sending historical data (but still respect 60-second interval between batches)
- Violating rate limits may result in data rejection or IP blocking

### Performance
- Send data asynchronously (goroutine)
- Cache pressure data for 5-10 minutes to reduce API calls
- Implement timeout for HTTP requests (10-30 seconds)
- Track rain totals in memory (or database for persistence)

---

## Overall Testing Status

### Completed Phases

**Phase 1: Unit Tests for Shared Utilities** ✅ COMPLETED
- Validation: 20 tests, 100% coverage
- Worker Pool: 17 tests, 96.6% coverage
- Logger: 17 tests, 100% coverage
- Total: 54 tests, excellent coverage

**Phase 2: Handler Tests with httptest** ✅ COMPLETED
- 42 tests implemented covering all handlers
- All tests passing with 77.4% coverage
- Includes rain detection tests (6 subtests)
- Includes rtl_433 version tests (6 subtests)
- Includes system info tests (2 subtests)
- Fixed weather icon bug (incorrect day/night calculation)
- Added mutex protection for concurrent access

**Phase 3: Database Integration Tests** ✅ COMPLETED
- 15 tests implemented covering database functions
- All tests passing with 83.1% coverage
- Tests database connection, retry logic, prepared statements
- Tests insert operations with different timestamp formats
- Tests daily rollup and cleanup functions
- Tests error handling and edge cases

### Pending Phases

**Phase 4: E2E Tests for Critical User Flows** ⏳ NOT STARTED
- Target: Full system integration
- Tests needed: Data ingestion flow, daily rollup flow, dashboard display

### Test Coverage Summary

**Current Test Coverage:**
- Core handlers: 77.4% (42 tests)
- Core database: 83.1% (15 tests)
- Shared logger: 100% (17 tests)
- Shared validation: 100% (20 tests)
- Shared workers: 96.6% (17 tests)
- Ingestor: 0% (no tests)

**Total Tests:** 111 tests passing

**Missing Test Coverage:**
- Ingestor: 0% (needs 16+ tests)
- Core astronomical: 0% (no tests)
- Core config: 0% (no tests)

### Recent Changes (2026-03-15)

**New Functionality Added:**
- RTL433 version display in dashboard
- System info endpoints (`/api/system/rtl433-version`, `/api/system/info`)
- Ingestor version sending at startup
- Weather data logging for debugging (raw JSON from rtl_433)

**Tests Added:**
- TestConnect (3 subtests): Database connection tests
- TestConnectWithRetry (2 subtests): Retry logic tests
- TestPrepareInsertStatement (2 subtests): Statement preparation tests
- TestInsertReading (4 subtests): Insert operation tests
- TestComputeDailyRollup (2 subtests): Daily rollup tests
- TestCleanOldReadings (2 subtests): Cleanup function tests

**Bugs Fixed:**
- Weather icon showing moon in daytime (fixed day/night calculation to compare only time components)
- Handler test nilness warnings (fixed with `=` instead of `:=`)
- Ingestor hanging on startup (fixed timeout and retry logic)
- RTL433 version parsing (fixed to use `-V` flag instead of `-v`)

**Files Modified:**
- `core/handlers/handlers.go`: Added system info handlers
- `core/handlers/handlers_test.go`: Added new tests, fixed nilness warnings
- `core/main.go`: Registered new routes
- `core/static/index.html`: Added rtl_433 version display, fixed day/night calculation
- `ingestor/main.go`: Added version sending and logging
- `core/database/database_test.go`: Created comprehensive database tests (NEW FILE)
