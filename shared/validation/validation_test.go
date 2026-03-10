package validation

import (
	"testing"

	"weather-station/shared/types"
)

func TestValidateReading(t *testing.T) {
	// Test 1: Valid reading - all fields within acceptable ranges
	t.Run("ValidReading", func(t *testing.T) {
		reading := types.Reading{
			Time:         "2026-03-10T12:00:00Z",
			Model:        "Fineoffset-WS90",
			SensorID:     12345,
			TemperatureC: 20.5,
			Humidity:     65,
			UVIndex:      5.0,
			Lux:          50000.0,
			WindSpeedMS:  5.2,
			WindGustMS:   8.5,
			WindDirDeg:   180,
			RainMM:       10.5,
			RainStart:    0,
			BatteryOK:    1.0,
			Firmware:     160,
		}

		err := ValidateReading(reading)
		if err != nil {
			t.Errorf("Expected no error for valid reading, got: %v", err)
		}
	})

	// Test 2: Invalid temperature - extreme values (below minimum)
	t.Run("InvalidTemperatureBelowMin", func(t *testing.T) {
		reading := types.Reading{
			Time:         "2026-03-10T12:00:00Z",
			Model:        "Fineoffset-WS90",
			SensorID:     12345,
			TemperatureC: -60.0, // Below MinTempC (-50)
			Humidity:     65,
			UVIndex:      5.0,
			Lux:          50000.0,
			WindSpeedMS:  5.2,
			WindGustMS:   8.5,
			WindDirDeg:   180,
			RainMM:       10.5,
			RainStart:    0,
			BatteryOK:    1.0,
			Firmware:     160,
		}

		err := ValidateReading(reading)
		if err == nil {
			t.Error("Expected error for temperature below minimum, got nil")
		}
		expectedError := "temperature out of range"
		if err != nil && err.Error()[:len(expectedError)] != expectedError {
			t.Errorf("Expected temperature error, got: %v", err)
		}
	})

	// Test 3: Invalid temperature - extreme values (above maximum)
	t.Run("InvalidTemperatureAboveMax", func(t *testing.T) {
		reading := types.Reading{
			Time:         "2026-03-10T12:00:00Z",
			Model:        "Fineoffset-WS90",
			SensorID:     12345,
			TemperatureC: 65.0, // Above MaxTempC (60)
			Humidity:     65,
			UVIndex:      5.0,
			Lux:          50000.0,
			WindSpeedMS:  5.2,
			WindGustMS:   8.5,
			WindDirDeg:   180,
			RainMM:       10.5,
			RainStart:    0,
			BatteryOK:    1.0,
			Firmware:     160,
		}

		err := ValidateReading(reading)
		if err == nil {
			t.Error("Expected error for temperature above maximum, got nil")
		}
		expectedError := "temperature out of range"
		if err != nil && err.Error()[:len(expectedError)] != expectedError {
			t.Errorf("Expected temperature error, got: %v", err)
		}
	})

	// Test 4: Invalid humidity - below minimum (0%)
	t.Run("InvalidHumidityBelowMin", func(t *testing.T) {
		reading := types.Reading{
			Time:         "2026-03-10T12:00:00Z",
			Model:        "Fineoffset-WS90",
			SensorID:     12345,
			TemperatureC: 20.5,
			Humidity:     -5, // Below MinHumidity (0)
			UVIndex:      5.0,
			Lux:          50000.0,
			WindSpeedMS:  5.2,
			WindGustMS:   8.5,
			WindDirDeg:   180,
			RainMM:       10.5,
			RainStart:    0,
			BatteryOK:    1.0,
			Firmware:     160,
		}

		err := ValidateReading(reading)
		if err == nil {
			t.Error("Expected error for humidity below minimum, got nil")
		}
		expectedError := "humidity out of range"
		if err != nil && err.Error()[:len(expectedError)] != expectedError {
			t.Errorf("Expected humidity error, got: %v", err)
		}
	})

	// Test 5: Invalid humidity - above maximum (100%)
	t.Run("InvalidHumidityAboveMax", func(t *testing.T) {
		reading := types.Reading{
			Time:         "2026-03-10T12:00:00Z",
			Model:        "Fineoffset-WS90",
			SensorID:     12345,
			TemperatureC: 20.5,
			Humidity:     105, // Above MaxHumidity (100)
			UVIndex:      5.0,
			Lux:          50000.0,
			WindSpeedMS:  5.2,
			WindGustMS:   8.5,
			WindDirDeg:   180,
			RainMM:       10.5,
			RainStart:    0,
			BatteryOK:    1.0,
			Firmware:     160,
		}

		err := ValidateReading(reading)
		if err == nil {
			t.Error("Expected error for humidity above maximum, got nil")
		}
		expectedError := "humidity out of range"
		if err != nil && err.Error()[:len(expectedError)] != expectedError {
			t.Errorf("Expected humidity error, got: %v", err)
		}
	})

	// Test 6: Invalid wind speed - negative values
	t.Run("InvalidWindSpeedNegative", func(t *testing.T) {
		reading := types.Reading{
			Time:         "2026-03-10T12:00:00Z",
			Model:        "Fineoffset-WS90",
			SensorID:     12345,
			TemperatureC: 20.5,
			Humidity:     65,
			UVIndex:      5.0,
			Lux:          50000.0,
			WindSpeedMS:  -2.5, // Below MinWindSpeed (0)
			WindGustMS:   8.5,
			WindDirDeg:   180,
			RainMM:       10.5,
			RainStart:    0,
			BatteryOK:    1.0,
			Firmware:     160,
		}

		err := ValidateReading(reading)
		if err == nil {
			t.Error("Expected error for negative wind speed, got nil")
		}
		expectedError := "wind speed out of range"
		if err != nil && err.Error()[:len(expectedError)] != expectedError {
			t.Errorf("Expected wind speed error, got: %v", err)
		}
	})

	// Test 7: Invalid wind speed - above maximum
	t.Run("InvalidWindSpeedAboveMax", func(t *testing.T) {
		reading := types.Reading{
			Time:         "2026-03-10T12:00:00Z",
			Model:        "Fineoffset-WS90",
			SensorID:     12345,
			TemperatureC: 20.5,
			Humidity:     65,
			UVIndex:      5.0,
			Lux:          50000.0,
			WindSpeedMS:  150.0, // Above MaxWindSpeed (100)
			WindGustMS:   8.5,
			WindDirDeg:   180,
			RainMM:       10.5,
			RainStart:    0,
			BatteryOK:    1.0,
			Firmware:     160,
		}

		err := ValidateReading(reading)
		if err == nil {
			t.Error("Expected error for wind speed above maximum, got nil")
		}
		expectedError := "wind speed out of range"
		if err != nil && err.Error()[:len(expectedError)] != expectedError {
			t.Errorf("Expected wind speed error, got: %v", err)
		}
	})

	// Test 8: Invalid wind gust - negative values
	t.Run("InvalidWindGustNegative", func(t *testing.T) {
		reading := types.Reading{
			Time:         "2026-03-10T12:00:00Z",
			Model:        "Fineoffset-WS90",
			SensorID:     12345,
			TemperatureC: 20.5,
			Humidity:     65,
			UVIndex:      5.0,
			Lux:          50000.0,
			WindSpeedMS:  5.2,
			WindGustMS:   -1.0, // Below MinWindGust (0)
			WindDirDeg:   180,
			RainMM:       10.5,
			RainStart:    0,
			BatteryOK:    1.0,
			Firmware:     160,
		}

		err := ValidateReading(reading)
		if err == nil {
			t.Error("Expected error for negative wind gust, got nil")
		}
		expectedError := "wind gust out of range"
		if err != nil && err.Error()[:len(expectedError)] != expectedError {
			t.Errorf("Expected wind gust error, got: %v", err)
		}
	})

	// Test 9: Invalid wind gust - above maximum
	t.Run("InvalidWindGustAboveMax", func(t *testing.T) {
		reading := types.Reading{
			Time:         "2026-03-10T12:00:00Z",
			Model:        "Fineoffset-WS90",
			SensorID:     12345,
			TemperatureC: 20.5,
			Humidity:     65,
			UVIndex:      5.0,
			Lux:          50000.0,
			WindSpeedMS:  5.2,
			WindGustMS:   250.0, // Above MaxWindGust (200)
			WindDirDeg:   180,
			RainMM:       10.5,
			RainStart:    0,
			BatteryOK:    1.0,
			Firmware:     160,
		}

		err := ValidateReading(reading)
		if err == nil {
			t.Error("Expected error for wind gust above maximum, got nil")
		}
		expectedError := "wind gust out of range"
		if err != nil && err.Error()[:len(expectedError)] != expectedError {
			t.Errorf("Expected wind gust error, got: %v", err)
		}
	})

	// Test 10: Invalid wind direction - below minimum
	t.Run("InvalidWindDirBelowMin", func(t *testing.T) {
		reading := types.Reading{
			Time:         "2026-03-10T12:00:00Z",
			Model:        "Fineoffset-WS90",
			SensorID:     12345,
			TemperatureC: 20.5,
			Humidity:     65,
			UVIndex:      5.0,
			Lux:          50000.0,
			WindSpeedMS:  5.2,
			WindGustMS:   8.5,
			WindDirDeg:   -10, // Below MinWindDir (0)
			RainMM:       10.5,
			RainStart:    0,
			BatteryOK:    1.0,
			Firmware:     160,
		}

		err := ValidateReading(reading)
		if err == nil {
			t.Error("Expected error for wind direction below minimum, got nil")
		}
		expectedError := "wind direction out of range"
		if err != nil && err.Error()[:len(expectedError)] != expectedError {
			t.Errorf("Expected wind direction error, got: %v", err)
		}
	})

	// Test 11: Invalid wind direction - above maximum
	t.Run("InvalidWindDirAboveMax", func(t *testing.T) {
		reading := types.Reading{
			Time:         "2026-03-10T12:00:00Z",
			Model:        "Fineoffset-WS90",
			SensorID:     12345,
			TemperatureC: 20.5,
			Humidity:     65,
			UVIndex:      5.0,
			Lux:          50000.0,
			WindSpeedMS:  5.2,
			WindGustMS:   8.5,
			WindDirDeg:   400, // Above MaxWindDir (359)
			RainMM:       10.5,
			RainStart:    0,
			BatteryOK:    1.0,
			Firmware:     160,
		}

		err := ValidateReading(reading)
		if err == nil {
			t.Error("Expected error for wind direction above maximum, got nil")
		}
		expectedError := "wind direction out of range"
		if err != nil && err.Error()[:len(expectedError)] != expectedError {
			t.Errorf("Expected wind direction error, got: %v", err)
		}
	})

	// Test 12: Invalid rain - negative values
	t.Run("InvalidRainNegative", func(t *testing.T) {
		reading := types.Reading{
			Time:         "2026-03-10T12:00:00Z",
			Model:        "Fineoffset-WS90",
			SensorID:     12345,
			TemperatureC: 20.5,
			Humidity:     65,
			UVIndex:      5.0,
			Lux:          50000.0,
			WindSpeedMS:  5.2,
			WindGustMS:   8.5,
			WindDirDeg:   180,
			RainMM:       -5.0, // Below MinRain (0)
			RainStart:    0,
			BatteryOK:    1.0,
			Firmware:     160,
		}

		err := ValidateReading(reading)
		if err == nil {
			t.Error("Expected error for negative rain, got nil")
		}
		expectedError := "rain out of range"
		if err != nil && err.Error()[:len(expectedError)] != expectedError {
			t.Errorf("Expected rain error, got: %v", err)
		}
	})

	// Test 13: Invalid rain - above maximum
	t.Run("InvalidRainAboveMax", func(t *testing.T) {
		reading := types.Reading{
			Time:         "2026-03-10T12:00:00Z",
			Model:        "Fineoffset-WS90",
			SensorID:     12345,
			TemperatureC: 20.5,
			Humidity:     65,
			UVIndex:      5.0,
			Lux:          50000.0,
			WindSpeedMS:  5.2,
			WindGustMS:   8.5,
			WindDirDeg:   180,
			RainMM:       1500.0, // Above MaxRain (1000)
			RainStart:    0,
			BatteryOK:    1.0,
			Firmware:     160,
		}

		err := ValidateReading(reading)
		if err == nil {
			t.Error("Expected error for rain above maximum, got nil")
		}
		expectedError := "rain out of range"
		if err != nil && err.Error()[:len(expectedError)] != expectedError {
			t.Errorf("Expected rain error, got: %v", err)
		}
	})

	// Test 14: Invalid UV index - negative values
	t.Run("InvalidUVIndexNegative", func(t *testing.T) {
		reading := types.Reading{
			Time:         "2026-03-10T12:00:00Z",
			Model:        "Fineoffset-WS90",
			SensorID:     12345,
			TemperatureC: 20.5,
			Humidity:     65,
			UVIndex:      -2.0, // Below 0
			Lux:          50000.0,
			WindSpeedMS:  5.2,
			WindGustMS:   8.5,
			WindDirDeg:   180,
			RainMM:       10.5,
			RainStart:    0,
			BatteryOK:    1.0,
			Firmware:     160,
		}

		err := ValidateReading(reading)
		if err == nil {
			t.Error("Expected error for negative UV index, got nil")
		}
		expectedError := "uv index out of range"
		if err != nil && err.Error()[:len(expectedError)] != expectedError {
			t.Errorf("Expected UV index error, got: %v", err)
		}
	})

	// Test 15: Invalid lux - negative values
	t.Run("InvalidLuxNegative", func(t *testing.T) {
		reading := types.Reading{
			Time:         "2026-03-10T12:00:00Z",
			Model:        "Fineoffset-WS90",
			SensorID:     12345,
			TemperatureC: 20.5,
			Humidity:     65,
			UVIndex:      5.0,
			Lux:          -1000.0, // Below 0
			WindSpeedMS:  5.2,
			WindGustMS:   8.5,
			WindDirDeg:   180,
			RainMM:       10.5,
			RainStart:    0,
			BatteryOK:    1.0,
			Firmware:     160,
		}

		err := ValidateReading(reading)
		if err == nil {
			t.Error("Expected error for negative lux, got nil")
		}
		expectedError := "lux out of range"
		if err != nil && err.Error()[:len(expectedError)] != expectedError {
			t.Errorf("Expected lux error, got: %v", err)
		}
	})

	// Test 16: Invalid battery - below minimum
	t.Run("InvalidBatteryBelowMin", func(t *testing.T) {
		reading := types.Reading{
			Time:         "2026-03-10T12:00:00Z",
			Model:        "Fineoffset-WS90",
			SensorID:     12345,
			TemperatureC: 20.5,
			Humidity:     65,
			UVIndex:      5.0,
			Lux:          50000.0,
			WindSpeedMS:  5.2,
			WindGustMS:   8.5,
			WindDirDeg:   180,
			RainMM:       10.5,
			RainStart:    0,
			BatteryOK:    -0.5, // Below 0
			Firmware:     160,
		}

		err := ValidateReading(reading)
		if err == nil {
			t.Error("Expected error for battery below minimum, got nil")
		}
		expectedError := "battery out of range"
		if err != nil && err.Error()[:len(expectedError)] != expectedError {
			t.Errorf("Expected battery error, got: %v", err)
		}
	})

	// Test 17: Invalid battery - above maximum
	t.Run("InvalidBatteryAboveMax", func(t *testing.T) {
		reading := types.Reading{
			Time:         "2026-03-10T12:00:00Z",
			Model:        "Fineoffset-WS90",
			SensorID:     12345,
			TemperatureC: 20.5,
			Humidity:     65,
			UVIndex:      5.0,
			Lux:          50000.0,
			WindSpeedMS:  5.2,
			WindGustMS:   8.5,
			WindDirDeg:   180,
			RainMM:       10.5,
			RainStart:    0,
			BatteryOK:    1.5, // Above 1
			Firmware:     160,
		}

		err := ValidateReading(reading)
		if err == nil {
			t.Error("Expected error for battery above maximum, got nil")
		}
		expectedError := "battery out of range"
		if err != nil && err.Error()[:len(expectedError)] != expectedError {
			t.Errorf("Expected battery error, got: %v", err)
		}
	})

	// Test 18: Missing required field - time
	t.Run("MissingTimeField", func(t *testing.T) {
		reading := types.Reading{
			Time:         "", // Empty string
			Model:        "Fineoffset-WS90",
			SensorID:     12345,
			TemperatureC: 20.5,
			Humidity:     65,
			UVIndex:      5.0,
			Lux:          50000.0,
			WindSpeedMS:  5.2,
			WindGustMS:   8.5,
			WindDirDeg:   180,
			RainMM:       10.5,
			RainStart:    0,
			BatteryOK:    1.0,
			Firmware:     160,
		}

		err := ValidateReading(reading)
		if err == nil {
			t.Error("Expected error for missing time field, got nil")
		}
		expectedError := "missing time field"
		if err != nil && err.Error() != expectedError {
			t.Errorf("Expected 'missing time field' error, got: %v", err)
		}
	})

	// Test 19: Missing required field - model
	t.Run("MissingModelField", func(t *testing.T) {
		reading := types.Reading{
			Time:         "2026-03-10T12:00:00Z",
			Model:        "", // Empty string
			SensorID:     12345,
			TemperatureC: 20.5,
			Humidity:     65,
			UVIndex:      5.0,
			Lux:          50000.0,
			WindSpeedMS:  5.2,
			WindGustMS:   8.5,
			WindDirDeg:   180,
			RainMM:       10.5,
			RainStart:    0,
			BatteryOK:    1.0,
			Firmware:     160,
		}

		err := ValidateReading(reading)
		if err == nil {
			t.Error("Expected error for missing model field, got nil")
		}
		expectedError := "missing model field"
		if err != nil && err.Error() != expectedError {
			t.Errorf("Expected 'missing model field' error, got: %v", err)
		}
	})

	// Test 20: Valid boundary values - test at exact limits
	t.Run("ValidBoundaryValues", func(t *testing.T) {
		reading := types.Reading{
			Time:         "2026-03-10T12:00:00Z",
			Model:        "Fineoffset-WS90",
			SensorID:     12345,
			TemperatureC: -50.0, // MinTempC
			Humidity:     0,      // MinHumidity
			UVIndex:      0.0,    // Min UV
			Lux:          0.0,    // Min Lux
			WindSpeedMS:  0.0,    // MinWindSpeed
			WindGustMS:   0.0,    // MinWindGust
			WindDirDeg:   0,      // MinWindDir
			RainMM:       0.0,    // MinRain
			RainStart:    0,
			BatteryOK:    0.0,    // Min Battery
			Firmware:     160,
		}

		err := ValidateReading(reading)
		if err != nil {
			t.Errorf("Expected no error for valid boundary values, got: %v", err)
		}
	})
}