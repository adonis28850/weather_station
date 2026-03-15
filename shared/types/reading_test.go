package types

import (
	"encoding/json"
	"testing"
)

func TestReadingJSONMarshalUnmarshal(t *testing.T) {
	t.Run("CompleteReading", func(t *testing.T) {
		original := Reading{
			Time:         "2026-03-15T10:00:00Z",
			Model:        "Fineoffset_WH90",
			SensorID:     12345,
			TemperatureC: 20.5,
			Humidity:     65,
			UVIndex:      5.5,
			Lux:          50000.0,
			WindSpeedMS:  3.2,
			WindGustMS:   5.8,
			WindDirDeg:   180,
			RainMM:       0.5,
			RainStart:    1,
			BatteryOK:    1.0,
			Firmware:     160,
		}

		// Marshal to JSON
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Failed to marshal reading: %v", err)
		}

		// Unmarshal back to struct
		var unmarshaled Reading
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal reading: %v", err)
		}

		// Verify all fields match
		if unmarshaled.Time != original.Time {
			t.Errorf("Time mismatch: got %s, want %s", unmarshaled.Time, original.Time)
		}
		if unmarshaled.Model != original.Model {
			t.Errorf("Model mismatch: got %s, want %s", unmarshaled.Model, original.Model)
		}
		if unmarshaled.SensorID != original.SensorID {
			t.Errorf("SensorID mismatch: got %d, want %d", unmarshaled.SensorID, original.SensorID)
		}
		if unmarshaled.TemperatureC != original.TemperatureC {
			t.Errorf("TemperatureC mismatch: got %f, want %f", unmarshaled.TemperatureC, original.TemperatureC)
		}
		if unmarshaled.Humidity != original.Humidity {
			t.Errorf("Humidity mismatch: got %d, want %d", unmarshaled.Humidity, original.Humidity)
		}
		if unmarshaled.UVIndex != original.UVIndex {
			t.Errorf("UVIndex mismatch: got %f, want %f", unmarshaled.UVIndex, original.UVIndex)
		}
		if unmarshaled.Lux != original.Lux {
			t.Errorf("Lux mismatch: got %f, want %f", unmarshaled.Lux, original.Lux)
		}
		if unmarshaled.WindSpeedMS != original.WindSpeedMS {
			t.Errorf("WindSpeedMS mismatch: got %f, want %f", unmarshaled.WindSpeedMS, original.WindSpeedMS)
		}
		if unmarshaled.WindGustMS != original.WindGustMS {
			t.Errorf("WindGustMS mismatch: got %f, want %f", unmarshaled.WindGustMS, original.WindGustMS)
		}
		if unmarshaled.WindDirDeg != original.WindDirDeg {
			t.Errorf("WindDirDeg mismatch: got %d, want %d", unmarshaled.WindDirDeg, original.WindDirDeg)
		}
		if unmarshaled.RainMM != original.RainMM {
			t.Errorf("RainMM mismatch: got %f, want %f", unmarshaled.RainMM, original.RainMM)
		}
		if unmarshaled.RainStart != original.RainStart {
			t.Errorf("RainStart mismatch: got %d, want %d", unmarshaled.RainStart, original.RainStart)
		}
		if unmarshaled.BatteryOK != original.BatteryOK {
			t.Errorf("BatteryOK mismatch: got %f, want %f", unmarshaled.BatteryOK, original.BatteryOK)
		}
		if unmarshaled.Firmware != original.Firmware {
			t.Errorf("Firmware mismatch: got %d, want %d", unmarshaled.Firmware, original.Firmware)
		}
	})

	t.Run("MinimalReading", func(t *testing.T) {
		original := Reading{
			Time:         "2026-03-15T10:00:00Z",
			Model:        "TestModel",
			SensorID:     1,
			TemperatureC: 15.0,
			Humidity:     50,
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Failed to marshal minimal reading: %v", err)
		}

		var unmarshaled Reading
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal minimal reading: %v", err)
		}

		if unmarshaled.Time != original.Time {
			t.Errorf("Time mismatch: got %s, want %s", unmarshaled.Time, original.Time)
		}
		if unmarshaled.Model != original.Model {
			t.Errorf("Model mismatch: got %s, want %s", unmarshaled.Model, original.Model)
		}
	})
}

func TestReadingJSONUnmarshalPartialFields(t *testing.T) {
	t.Run("OnlyRequiredFields", func(t *testing.T) {
		jsonStr := `{"time":"2026-03-15T10:00:00Z","model":"TestModel","id":1,"temperature_C":20.5,"humidity":65}`

		var reading Reading
		if err := json.Unmarshal([]byte(jsonStr), &reading); err != nil {
			t.Fatalf("Failed to unmarshal partial JSON: %v", err)
		}

		// Verify specified fields are set
		if reading.Time != "2026-03-15T10:00:00Z" {
			t.Errorf("Time not set correctly: got %s", reading.Time)
		}
		if reading.Model != "TestModel" {
			t.Errorf("Model not set correctly: got %s", reading.Model)
		}
		if reading.SensorID != 1 {
			t.Errorf("SensorID not set correctly: got %d", reading.SensorID)
		}

		// Verify unspecified fields are zero values
		if reading.UVIndex != 0 {
			t.Errorf("UVIndex should be zero, got %f", reading.UVIndex)
		}
		if reading.Lux != 0 {
			t.Errorf("Lux should be zero, got %f", reading.Lux)
		}
		if reading.WindSpeedMS != 0 {
			t.Errorf("WindSpeedMS should be zero, got %f", reading.WindSpeedMS)
		}
		if reading.RainMM != 0 {
			t.Errorf("RainMM should be zero, got %f", reading.RainMM)
		}
	})

	t.Run("MixedFields", func(t *testing.T) {
		jsonStr := `{"time":"2026-03-15T10:00:00Z","model":"TestModel","id":1,"temperature_C":20.5,"humidity":65,"wind_avg_m_s":3.2,"rain_mm":0.5}`

		var reading Reading
		if err := json.Unmarshal([]byte(jsonStr), &reading); err != nil {
			t.Fatalf("Failed to unmarshal mixed JSON: %v", err)
		}

		// Verify specified fields are set
		if reading.WindSpeedMS != 3.2 {
			t.Errorf("WindSpeedMS not set correctly: got %f", reading.WindSpeedMS)
		}
		if reading.RainMM != 0.5 {
			t.Errorf("RainMM not set correctly: got %f", reading.RainMM)
		}

		// Verify unspecified fields are zero values
		if reading.UVIndex != 0 {
			t.Errorf("UVIndex should be zero, got %f", reading.UVIndex)
		}
		if reading.Lux != 0 {
			t.Errorf("Lux should be zero, got %f", reading.Lux)
		}
	})
}

func TestReadingZeroValue(t *testing.T) {
	t.Run("MarshalZeroValue", func(t *testing.T) {
		var reading Reading

		data, err := json.Marshal(reading)
		if err != nil {
			t.Fatalf("Failed to marshal zero value: %v", err)
		}

		// Verify JSON structure
		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		// All fields should be present with zero values
		if result["time"] != "" {
			t.Errorf("Time should be empty string, got %v", result["time"])
		}
		if result["model"] != "" {
			t.Errorf("Model should be empty string, got %v", result["model"])
		}
		if result["id"] != float64(0) {
			t.Errorf("SensorID should be 0, got %v", result["id"])
		}
		if result["temperature_C"] != float64(0) {
			t.Errorf("TemperatureC should be 0, got %v", result["temperature_C"])
		}
		if result["humidity"] != float64(0) {
			t.Errorf("Humidity should be 0, got %v", result["humidity"])
		}
	})

	t.Run("UnmarshalZeroValue", func(t *testing.T) {
		jsonStr := `{"time":"","model":"","id":0,"temperature_C":0,"humidity":0,"uvi":0,"light_lux":0,"wind_avg_m_s":0,"wind_max_m_s":0,"wind_dir_deg":0,"rain_mm":0,"rain_start":0,"battery_ok":0,"firmware":0}`

		var reading Reading
		if err := json.Unmarshal([]byte(jsonStr), &reading); err != nil {
			t.Fatalf("Failed to unmarshal zero value JSON: %v", err)
		}

		// Verify all fields are zero values
		if reading.Time != "" {
			t.Errorf("Time should be empty, got %s", reading.Time)
		}
		if reading.Model != "" {
			t.Errorf("Model should be empty, got %s", reading.Model)
		}
		if reading.SensorID != 0 {
			t.Errorf("SensorID should be 0, got %d", reading.SensorID)
		}
		if reading.TemperatureC != 0 {
			t.Errorf("TemperatureC should be 0, got %f", reading.TemperatureC)
		}
		if reading.Humidity != 0 {
			t.Errorf("Humidity should be 0, got %d", reading.Humidity)
		}
		if reading.UVIndex != 0 {
			t.Errorf("UVIndex should be 0, got %f", reading.UVIndex)
		}
		if reading.Lux != 0 {
			t.Errorf("Lux should be 0, got %f", reading.Lux)
		}
		if reading.WindSpeedMS != 0 {
			t.Errorf("WindSpeedMS should be 0, got %f", reading.WindSpeedMS)
		}
		if reading.WindGustMS != 0 {
			t.Errorf("WindGustMS should be 0, got %f", reading.WindGustMS)
		}
		if reading.WindDirDeg != 0 {
			t.Errorf("WindDirDeg should be 0, got %d", reading.WindDirDeg)
		}
		if reading.RainMM != 0 {
			t.Errorf("RainMM should be 0, got %f", reading.RainMM)
		}
		if reading.RainStart != 0 {
			t.Errorf("RainStart should be 0, got %d", reading.RainStart)
		}
		if reading.BatteryOK != 0 {
			t.Errorf("BatteryOK should be 0, got %f", reading.BatteryOK)
		}
		if reading.Firmware != 0 {
			t.Errorf("Firmware should be 0, got %d", reading.Firmware)
		}
	})
}

func TestReadingJSONTagValidation(t *testing.T) {
	t.Run("CorrectJSONTags", func(t *testing.T) {
		reading := Reading{
			Time:         "2026-03-15T10:00:00Z",
			TemperatureC: 20.5,
			Humidity:     65,
		}

		data, err := json.Marshal(reading)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		// Verify JSON uses correct field names
		jsonStr := string(data)
		expectedFields := []string{
			`"time"`,
			`"model"`,
			`"id"`,
			`"temperature_C"`,
			`"humidity"`,
			`"uvi"`,
			`"light_lux"`,
			`"wind_avg_m_s"`,
			`"wind_max_m_s"`,
			`"wind_dir_deg"`,
			`"rain_mm"`,
			`"rain_start"`,
			`"battery_ok"`,
			`"firmware"`,
		}

		for _, field := range expectedFields {
			if !containsSubstring(jsonStr, field) {
				t.Errorf("JSON should contain field %s", field)
			}
		}
	})

	t.Run("UnmarshalWithCorrectTags", func(t *testing.T) {
		jsonStr := `{
			"time":"2026-03-15T10:00:00Z",
			"model":"TestModel",
			"id":12345,
			"temperature_C":20.5,
			"humidity":65,
			"uvi":5.5,
			"light_lux":50000.0,
			"wind_avg_m_s":3.2,
			"wind_max_m_s":5.8,
			"wind_dir_deg":180,
			"rain_mm":0.5,
			"rain_start":1,
			"battery_ok":1.0,
			"firmware":160
		}`

		var reading Reading
		if err := json.Unmarshal([]byte(jsonStr), &reading); err != nil {
			t.Fatalf("Failed to unmarshal with correct tags: %v", err)
		}

		// Verify all fields are correctly mapped
		if reading.Time != "2026-03-15T10:00:00Z" {
			t.Errorf("Time not mapped correctly")
		}
		if reading.TemperatureC != 20.5 {
			t.Errorf("TemperatureC not mapped correctly")
		}
		if reading.Humidity != 65 {
			t.Errorf("Humidity not mapped correctly")
		}
		if reading.UVIndex != 5.5 {
			t.Errorf("UVIndex not mapped correctly")
		}
		if reading.Lux != 50000.0 {
			t.Errorf("Lux not mapped correctly")
		}
		if reading.WindSpeedMS != 3.2 {
			t.Errorf("WindSpeedMS not mapped correctly")
		}
		if reading.WindGustMS != 5.8 {
			t.Errorf("WindGustMS not mapped correctly")
		}
		if reading.WindDirDeg != 180 {
			t.Errorf("WindDirDeg not mapped correctly")
		}
		if reading.RainMM != 0.5 {
			t.Errorf("RainMM not mapped correctly")
		}
		if reading.RainStart != 1 {
			t.Errorf("RainStart not mapped correctly")
		}
		if reading.BatteryOK != 1.0 {
			t.Errorf("BatteryOK not mapped correctly")
		}
		if reading.Firmware != 160 {
			t.Errorf("Firmware not mapped correctly")
		}
	})
}

func TestReadingFieldTypeValidation(t *testing.T) {
	t.Run("StringFields", func(t *testing.T) {
		jsonStr := `{"time":"2026-03-15T10:00:00Z","model":"TestModel","id":1,"temperature_C":20.5,"humidity":65}`

		var reading Reading
		if err := json.Unmarshal([]byte(jsonStr), &reading); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		// Verify string fields are strings
		if reading.Time != "2026-03-15T10:00:00Z" {
			t.Errorf("Time should be string")
		}
		if reading.Model != "TestModel" {
			t.Errorf("Model should be string")
		}
	})

	t.Run("IntFields", func(t *testing.T) {
		jsonStr := `{"time":"2026-03-15T10:00:00Z","model":"TestModel","id":12345,"temperature_C":20.5,"humidity":65,"wind_dir_deg":270,"rain_start":1,"firmware":160}`

		var reading Reading
		if err := json.Unmarshal([]byte(jsonStr), &reading); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		// Verify int fields are ints
		if reading.SensorID != 12345 {
			t.Errorf("SensorID should be int")
		}
		if reading.Humidity != 65 {
			t.Errorf("Humidity should be int")
		}
		if reading.WindDirDeg != 270 {
			t.Errorf("WindDirDeg should be int")
		}
		if reading.RainStart != 1 {
			t.Errorf("RainStart should be int")
		}
		if reading.Firmware != 160 {
			t.Errorf("Firmware should be int")
		}
	})

	t.Run("FloatFields", func(t *testing.T) {
		jsonStr := `{"time":"2026-03-15T10:00:00Z","model":"TestModel","id":1,"temperature_C":20.5,"humidity":65,"uvi":5.5,"light_lux":50000.0,"wind_avg_m_s":3.2,"wind_max_m_s":5.8,"rain_mm":0.5,"battery_ok":1.0}`

		var reading Reading
		if err := json.Unmarshal([]byte(jsonStr), &reading); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		// Verify float fields are floats
		if reading.TemperatureC != 20.5 {
			t.Errorf("TemperatureC should be float")
		}
		if reading.UVIndex != 5.5 {
			t.Errorf("UVIndex should be float")
		}
		if reading.Lux != 50000.0 {
			t.Errorf("Lux should be float")
		}
		if reading.WindSpeedMS != 3.2 {
			t.Errorf("WindSpeedMS should be float")
		}
		if reading.WindGustMS != 5.8 {
			t.Errorf("WindGustMS should be float")
		}
		if reading.RainMM != 0.5 {
			t.Errorf("RainMM should be float")
		}
		if reading.BatteryOK != 1.0 {
			t.Errorf("BatteryOK should be float")
		}
	})
}

func TestReadingExtremeValues(t *testing.T) {
	t.Run("NegativeTemperature", func(t *testing.T) {
		jsonStr := `{"time":"2026-03-15T10:00:00Z","model":"TestModel","id":1,"temperature_C":-20.5,"humidity":65}`

		var reading Reading
		if err := json.Unmarshal([]byte(jsonStr), &reading); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if reading.TemperatureC != -20.5 {
			t.Errorf("Should handle negative temperature, got %f", reading.TemperatureC)
		}
	})

	t.Run("HighHumidity", func(t *testing.T) {
		jsonStr := `{"time":"2026-03-15T10:00:00Z","model":"TestModel","id":1,"temperature_C":20.5,"humidity":100}`

		var reading Reading
		if err := json.Unmarshal([]byte(jsonStr), &reading); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if reading.Humidity != 100 {
			t.Errorf("Should handle 100%% humidity, got %d", reading.Humidity)
		}
	})

	t.Run("ZeroHumidity", func(t *testing.T) {
		jsonStr := `{"time":"2026-03-15T10:00:00Z","model":"TestModel","id":1,"temperature_C":20.5,"humidity":0}`

		var reading Reading
		if err := json.Unmarshal([]byte(jsonStr), &reading); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if reading.Humidity != 0 {
			t.Errorf("Should handle 0%% humidity, got %d", reading.Humidity)
		}
	})

	t.Run("HighWindSpeed", func(t *testing.T) {
		jsonStr := `{"time":"2026-03-15T10:00:00Z","model":"TestModel","id":1,"temperature_C":20.5,"humidity":65,"wind_avg_m_s":50.0,"wind_max_m_s":75.0}`

		var reading Reading
		if err := json.Unmarshal([]byte(jsonStr), &reading); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if reading.WindSpeedMS != 50.0 {
			t.Errorf("Should handle high wind speed, got %f", reading.WindSpeedMS)
		}
		if reading.WindGustMS != 75.0 {
			t.Errorf("Should handle high wind gust, got %f", reading.WindGustMS)
		}
	})

	t.Run("FullWindDirection", func(t *testing.T) {
		jsonStr := `{"time":"2026-03-15T10:00:00Z","model":"TestModel","id":1,"temperature_C":20.5,"humidity":65,"wind_dir_deg":360}`

		var reading Reading
		if err := json.Unmarshal([]byte(jsonStr), &reading); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if reading.WindDirDeg != 360 {
			t.Errorf("Should handle 360 degree wind direction, got %d", reading.WindDirDeg)
		}
	})
}

func TestReadingRealWorldExample(t *testing.T) {
	t.Run("TypicalWS90Reading", func(t *testing.T) {
		// This is a realistic reading from a WS90 weather station
		jsonStr := `{
			"time":"2026-03-15T10:30:15Z",
			"model":"Fineoffset_WH90",
			"id":12345,
			"temperature_C":18.5,
			"humidity":72,
			"uvi":3.2,
			"light_lux":45000.0,
			"wind_avg_m_s":2.8,
			"wind_max_m_s":4.5,
			"wind_dir_deg":225,
			"rain_mm":0.2,
			"rain_start":0,
			"battery_ok":1.0,
			"firmware":160
		}`

		var reading Reading
		if err := json.Unmarshal([]byte(jsonStr), &reading); err != nil {
			t.Fatalf("Failed to unmarshal realistic reading: %v", err)
		}

		// Verify all fields are correctly parsed
		if reading.Time != "2026-03-15T10:30:15Z" {
			t.Errorf("Time mismatch")
		}
		if reading.Model != "Fineoffset_WH90" {
			t.Errorf("Model mismatch")
		}
		if reading.SensorID != 12345 {
			t.Errorf("SensorID mismatch")
		}
		if reading.TemperatureC != 18.5 {
			t.Errorf("TemperatureC mismatch")
		}
		if reading.Humidity != 72 {
			t.Errorf("Humidity mismatch")
		}
		if reading.UVIndex != 3.2 {
			t.Errorf("UVIndex mismatch")
		}
		if reading.Lux != 45000.0 {
			t.Errorf("Lux mismatch")
		}
		if reading.WindSpeedMS != 2.8 {
			t.Errorf("WindSpeedMS mismatch")
		}
		if reading.WindGustMS != 4.5 {
			t.Errorf("WindGustMS mismatch")
		}
		if reading.WindDirDeg != 225 {
			t.Errorf("WindDirDeg mismatch")
		}
		if reading.RainMM != 0.2 {
			t.Errorf("RainMM mismatch")
		}
		if reading.RainStart != 0 {
			t.Errorf("RainStart mismatch")
		}
		if reading.BatteryOK != 1.0 {
			t.Errorf("BatteryOK mismatch")
		}
		if reading.Firmware != 160 {
			t.Errorf("Firmware mismatch")
		}
	})
}

// Helper function to check if a string contains a substring
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}