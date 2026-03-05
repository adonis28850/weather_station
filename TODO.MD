# Wind direction in dashboard

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