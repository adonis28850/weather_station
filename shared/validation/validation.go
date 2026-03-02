package validation

import (
	"fmt"

	"weather-station/shared/types"
)

// Validation constants for weather readings
// These define the acceptable ranges for each sensor reading based on realistic physical limits
const (
	// Temperature range in Celsius (-50°C to 60°C)
	// Rationale: Covers extreme weather conditions from polar regions (-50°C) to desert heat (60°C)
	// while filtering out sensor errors and impossible values
	MinTempC = -50
	MaxTempC = 60

	// Humidity percentage (0% to 100%)
	// Rationale: Absolute humidity cannot be negative or exceed 100% saturation
	MinHumidity = 0
	MaxHumidity = 100

	// Wind speed in meters per second (0 to 100 m/s)
	// Rationale: Maximum wind speed of 100 m/s (360 km/h) covers the strongest hurricanes ever recorded
	// while filtering out sensor glitches that might report unrealistic values
	MinWindSpeed = 0
	MaxWindSpeed = 100

	// Wind gust speed in meters per second (0 to 200 m/s)
	// Rationale: Gusts can be significantly higher than sustained wind speeds
	// 200 m/s (720 km/h) provides a safety margin for extreme weather events
	MinWindGust = 0
	MaxWindGust = 200

	// Wind direction in degrees (0° to 359°, where 0=N, 90=E, 180=S, 270=W)
	// Rationale: Standard compass notation, 360° is equivalent to 0° (North)
	MinWindDir = 0
	MaxWindDir = 359

	// Rainfall in millimeters (0 to 1000 mm)
	// Rationale: 1000 mm (1 meter) of rain in a single reading period covers extreme rainfall events
	// while filtering out sensor errors that might report impossible accumulation
	MinRain = 0
	MaxRain = 1000
)

// ValidateReading validates a reading against acceptable ranges
func ValidateReading(reading types.Reading) error {
	if reading.Time == "" {
		return fmt.Errorf("missing time field")
	}
	if reading.Model == "" {
		return fmt.Errorf("missing model field")
	}
	if reading.TemperatureC < MinTempC || reading.TemperatureC > MaxTempC {
		return fmt.Errorf("temperature out of range (%.2f°C)", reading.TemperatureC)
	}
	if reading.Humidity < MinHumidity || reading.Humidity > MaxHumidity {
		return fmt.Errorf("humidity out of range (%d%%)", reading.Humidity)
	}
	if reading.WindSpeedMS < MinWindSpeed || reading.WindSpeedMS > MaxWindSpeed {
		return fmt.Errorf("wind speed out of range (%.2f m/s)", reading.WindSpeedMS)
	}
	if reading.WindGustMS < MinWindGust || reading.WindGustMS > MaxWindGust {
		return fmt.Errorf("wind gust out of range (%.2f m/s)", reading.WindGustMS)
	}
	if reading.WindDirDeg < MinWindDir || reading.WindDirDeg > MaxWindDir {
		return fmt.Errorf("wind direction out of range (%d°)", reading.WindDirDeg)
	}
	if reading.RainMM < MinRain || reading.RainMM > MaxRain {
		return fmt.Errorf("rain out of range (%.2f mm)", reading.RainMM)
	}
	if reading.UVIndex < 0 {
		return fmt.Errorf("uv index out of range (%.2f)", reading.UVIndex)
	}
	if reading.Lux < 0 {
		return fmt.Errorf("lux out of range (%.2f)", reading.Lux)
	}
	if reading.BatteryOK < 0 || reading.BatteryOK > 1 {
		return fmt.Errorf("battery out of range (%.2f)", reading.BatteryOK)
	}

	return nil
}