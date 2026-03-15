package astronomical

import (
	"testing"
	"time"
)

func TestCalculateSunriseSunset(t *testing.T) {
	t.Run("KnownLocationAndDate", func(t *testing.T) {
		// Test with known location: Chicago, IL (41.8781° N, 87.6298° W)
		// Date: March 15, 2026
		date := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
		sunrise, sunset := CalculateSunriseSunset(date, 41.8781, -87.6298, "America/Chicago", func(string, ...interface{}) {})

		// Parse the results
		sunriseTime, err := time.Parse(time.RFC3339, sunrise)
		if err != nil {
			t.Fatalf("Failed to parse sunrise time: %v", err)
		}
		sunsetTime, err := time.Parse(time.RFC3339, sunset)
		if err != nil {
			t.Fatalf("Failed to parse sunset time: %v", err)
		}

		// Basic sanity checks
		if sunriseTime.After(sunsetTime) {
			t.Error("Sunrise should be before sunset")
		}

		// Sunrise should be around 6-8 AM local time in March
		sunriseHour := sunriseTime.In(time.FixedZone("CST", -6*3600)).Hour()
		if sunriseHour < 5 || sunriseHour > 8 {
			t.Errorf("Sunrise hour (%d) seems incorrect for March in Chicago", sunriseHour)
		}

		// Sunset should be around 6-8 PM local time in March
		sunsetHour := sunsetTime.In(time.FixedZone("CST", -6*3600)).Hour()
		if sunsetHour < 17 || sunsetHour > 20 {
			t.Errorf("Sunset hour (%d) seems incorrect for March in Chicago", sunsetHour)
		}
	})

	t.Run("EquinoxCalculation", func(t *testing.T) {
		// March equinox: approximately March 20
		date := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
		sunrise, sunset := CalculateSunriseSunset(date, 0, 0, "UTC", func(string, ...interface{}) {})

		sunriseTime, _ := time.Parse(time.RFC3339, sunrise)
		sunsetTime, _ := time.Parse(time.RFC3339, sunset)

		// At equator on equinox, day and night should be roughly equal
		dayLength := sunsetTime.Sub(sunriseTime)
		expectedLength := 12 * time.Hour
		tolerance := 10 * time.Minute

		if dayLength < expectedLength-tolerance || dayLength > expectedLength+tolerance {
			t.Errorf("Day length at equator on equinox should be ~12 hours, got %v", dayLength)
		}
	})

	t.Run("SolsticeCalculation", func(t *testing.T) {
		// June solstice: approximately June 21
		date := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
		sunrise, sunset := CalculateSunriseSunset(date, 45, 0, "UTC", func(string, ...interface{}) {})

		sunriseTime, _ := time.Parse(time.RFC3339, sunrise)
		sunsetTime, _ := time.Parse(time.RFC3339, sunset)

		// At 45°N on summer solstice, days are much longer than 12 hours
		dayLength := sunsetTime.Sub(sunriseTime)
		if dayLength < 15*time.Hour {
			t.Errorf("Day length at 45°N on summer solstice should be >15 hours, got %v", dayLength)
		}
	})

	t.Run("DifferentLatitudes", func(t *testing.T) {
		date := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)

		// Test at equator
		sunriseEq, sunsetEq := CalculateSunriseSunset(date, 0, 0, "UTC", func(string, ...interface{}) {})
		sunriseEqTime, _ := time.Parse(time.RFC3339, sunriseEq)
		sunsetEqTime, _ := time.Parse(time.RFC3339, sunsetEq)
		dayLengthEq := sunsetEqTime.Sub(sunriseEqTime)

		// Test at 60°N
		sunrise60, sunset60 := CalculateSunriseSunset(date, 60, 0, "UTC", func(string, ...interface{}) {})
		sunrise60Time, _ := time.Parse(time.RFC3339, sunrise60)
		sunset60Time, _ := time.Parse(time.RFC3339, sunset60)
		dayLength60 := sunset60Time.Sub(sunrise60Time)

		// Days should be longer at higher latitudes in summer
		if dayLength60 <= dayLengthEq {
			t.Error("Days should be longer at 60°N than at equator in summer")
		}
	})

	t.Run("InvalidTimezone", func(t *testing.T) {
		date := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
		// Should not panic, should fallback to UTC
		sunrise, sunset := CalculateSunriseSunset(date, 41.8781, -87.6298, "Invalid/Timezone", func(string, ...interface{}) {})

		// Should still return valid times
		if sunrise == "" || sunset == "" {
			t.Error("Should return valid times even with invalid timezone")
		}
	})

	t.Run("PolarRegionMidnightSun", func(t *testing.T) {
		// Test in polar region during summer
		date := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
		sunrise, sunset := CalculateSunriseSunset(date, 80, 0, "UTC", func(string, ...interface{}) {})

		// Should not panic
		sunriseTime, _ := time.Parse(time.RFC3339, sunrise)
		sunsetTime, _ := time.Parse(time.RFC3339, sunset)

		// In polar regions, sunrise/sunset calculations should still work
		if sunriseTime.IsZero() || sunsetTime.IsZero() {
			t.Error("Polar region calculations should return valid times")
		}
	})
}

func TestCalculateMoonPhase(t *testing.T) {
	t.Run("KnownNewMoon", func(t *testing.T) {
		// January 6, 2000 was a new moon (reference date in code)
		date := time.Date(2000, 1, 6, 18, 14, 0, 0, time.UTC)
		phase := CalculateMoonPhase(date)

		// Phase should be very close to 0 or 1 (new moon)
		if phase > 0.1 && phase < 0.9 {
			t.Errorf("Known new moon should have phase near 0 or 1, got %.4f", phase)
		}
	})

	t.Run("KnownFullMoon", func(t *testing.T) {
		// January 21, 2000 was approximately a full moon
		date := time.Date(2000, 1, 21, 5, 40, 0, 0, time.UTC)
		phase := CalculateMoonPhase(date)

		// Phase should be close to 0.5 (full moon)
		if phase < 0.4 || phase > 0.6 {
			t.Errorf("Known full moon should have phase near 0.5, got %.4f", phase)
		}
	})

	t.Run("PhaseRange", func(t *testing.T) {
		date := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
		phase := CalculateMoonPhase(date)

		// Phase should always be between 0 and 1
		if phase < 0 || phase >= 1 {
			t.Errorf("Moon phase should be between 0 and 1, got %.4f", phase)
		}
	})

	t.Run("PhaseProgression", func(t *testing.T) {
		// Test that phase increases over time
		date1 := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
		date2 := time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC)

		phase1 := CalculateMoonPhase(date1)
		phase2 := CalculateMoonPhase(date2)

		// Phase should increase (though may wrap around)
		if phase2 < phase1 {
			// Wrapped around, add 1
			phase2 += 1
		}

		if phase2 <= phase1 {
			t.Errorf("Moon phase should increase over time, got %.4f -> %.4f", phase1, phase2)
		}
	})
}

func TestCalculateMoonIllumination(t *testing.T) {
	t.Run("NewMoonIllumination", func(t *testing.T) {
		// New moon: phase ≈ 0
		illum := CalculateMoonIllumination(0.0)
		if illum > 5 {
			t.Errorf("New moon should have very low illumination, got %.2f%%", illum)
		}
	})

	t.Run("FullMoonIllumination", func(t *testing.T) {
		// Full moon: phase ≈ 0.5
		illum := CalculateMoonIllumination(0.5)
		if illum < 95 {
			t.Errorf("Full moon should have high illumination, got %.2f%%", illum)
		}
	})

	t.Run("QuarterMoonIllumination", func(t *testing.T) {
		// First quarter: phase ≈ 0.25
		illum := CalculateMoonIllumination(0.25)
		if illum < 40 || illum > 60 {
			t.Errorf("Quarter moon should have ~50%% illumination, got %.2f%%", illum)
		}
	})

	t.Run("IlluminationRange", func(t *testing.T) {
		// Test various phases
		for i := 0; i <= 10; i++ {
			phase := float64(i) / 10.0
			illum := CalculateMoonIllumination(phase)

			if illum < 0 || illum > 100 {
				t.Errorf("Illumination should be between 0 and 100, got %.2f%% for phase %.2f", illum, phase)
			}
		}
	})
}

func TestGetMoonTrend(t *testing.T) {
	t.Run("WaxingPhase", func(t *testing.T) {
		trend := GetMoonTrend(0.25)
		if trend != "Waxing" {
			t.Errorf("Phase 0.25 should be Waxing, got %s", trend)
		}
	})

	t.Run("WaningPhase", func(t *testing.T) {
		trend := GetMoonTrend(0.75)
		if trend != "Waning" {
			t.Errorf("Phase 0.75 should be Waning, got %s", trend)
		}
	})

	t.Run("BoundaryPhase", func(t *testing.T) {
		trend := GetMoonTrend(0.5)
		if trend != "Waning" {
			t.Errorf("Phase 0.5 should be Waning, got %s", trend)
		}
	})
}

func TestGetMoonPhaseInfo(t *testing.T) {
	t.Run("NewMoon", func(t *testing.T) {
		icon, desc := GetMoonPhaseInfo(0.0)
		if icon != "bi-moon" {
			t.Errorf("New moon should have 'bi-moon' icon, got %s", icon)
		}
		if desc != "New Moon" {
			t.Errorf("New moon should have 'New Moon' description, got %s", desc)
		}
	})

	t.Run("FullMoon", func(t *testing.T) {
		icon, desc := GetMoonPhaseInfo(0.5)
		if icon != "bi-moon-fill" {
			t.Errorf("Full moon should have 'bi-moon-fill' icon, got %s", icon)
		}
		if desc != "Full Moon" {
			t.Errorf("Full moon should have 'Full Moon' description, got %s", desc)
		}
	})

	t.Run("WaxingCrescent", func(t *testing.T) {
		icon, desc := GetMoonPhaseInfo(0.1)
		if icon != "bi-moon-stars" {
			t.Errorf("Waxing crescent should have 'bi-moon-stars' icon, got %s", icon)
		}
		if desc != "Waxing Crescent" {
			t.Errorf("Waxing crescent should have 'Waxing Crescent' description, got %s", desc)
		}
	})

	t.Run("FirstQuarter", func(t *testing.T) {
		icon, desc := GetMoonPhaseInfo(0.25)
		if icon != "bi-moon" {
			t.Errorf("First quarter should have 'bi-moon' icon, got %s", icon)
		}
		if desc != "First Quarter" {
			t.Errorf("First quarter should have 'First Quarter' description, got %s", desc)
		}
	})

	t.Run("WaxingGibbous", func(t *testing.T) {
		icon, desc := GetMoonPhaseInfo(0.35)
		if icon != "bi-moon" {
			t.Errorf("Waxing gibbous should have 'bi-moon' icon, got %s", icon)
		}
		if desc != "Waxing Gibbous" {
			t.Errorf("Waxing gibbous should have 'Waxing Gibbous' description, got %s", desc)
		}
	})

	t.Run("WaningGibbous", func(t *testing.T) {
		icon, desc := GetMoonPhaseInfo(0.65)
		if icon != "bi-moon" {
			t.Errorf("Waning gibbous should have 'bi-moon' icon, got %s", icon)
		}
		if desc != "Waning Gibbous" {
			t.Errorf("Waning gibbous should have 'Waning Gibbous' description, got %s", desc)
		}
	})

	t.Run("LastQuarter", func(t *testing.T) {
		icon, desc := GetMoonPhaseInfo(0.75)
		if icon != "bi-moon" {
			t.Errorf("Last quarter should have 'bi-moon' icon, got %s", icon)
		}
		if desc != "Last Quarter" {
			t.Errorf("Last quarter should have 'Last Quarter' description, got %s", desc)
		}
	})

	t.Run("WaningCrescent", func(t *testing.T) {
		icon, desc := GetMoonPhaseInfo(0.85)
		if icon != "bi-moon-stars" {
			t.Errorf("Waning crescent should have 'bi-moon-stars' icon, got %s", icon)
		}
		if desc != "Waning Crescent" {
			t.Errorf("Waning crescent should have 'Waning Crescent' description, got %s", desc)
		}
	})

	t.Run("NearNewMoonEnd", func(t *testing.T) {
		icon, desc := GetMoonPhaseInfo(0.97)
		if icon != "bi-moon" {
			t.Errorf("Near new moon should have 'bi-moon' icon, got %s", icon)
		}
		if desc != "New Moon" {
			t.Errorf("Near new moon should have 'New Moon' description, got %s", desc)
		}
	})
}

func TestCalculateData(t *testing.T) {
	t.Run("ValidCalculation", func(t *testing.T) {
		data := CalculateData(41.8781, -87.6298, "America/Chicago", func(string, ...interface{}) {})

		// Check all fields are populated
		if data.Sunrise == "" {
			t.Error("Sunrise should not be empty")
		}
		if data.Sunset == "" {
			t.Error("Sunset should not be empty")
		}
		if data.MoonPhase == "" {
			t.Error("Moon phase should not be empty")
		}
		if data.MoonIcon == "" {
			t.Error("Moon icon should not be empty")
		}
		if data.MoonTrend == "" {
			t.Error("Moon trend should not be empty")
		}
	})

	t.Run("SunriseBeforeSunset", func(t *testing.T) {
		data := CalculateData(41.8781, -87.6298, "America/Chicago", func(string, ...interface{}) {})

		sunriseTime, err := time.Parse(time.RFC3339, data.Sunrise)
		if err != nil {
			t.Fatalf("Failed to parse sunrise: %v", err)
		}
		sunsetTime, err := time.Parse(time.RFC3339, data.Sunset)
		if err != nil {
			t.Fatalf("Failed to parse sunset: %v", err)
		}

		if sunriseTime.After(sunsetTime) {
			t.Error("Sunrise should be before sunset")
		}
	})

	t.Run("MoonIlluminationRange", func(t *testing.T) {
		data := CalculateData(41.8781, -87.6298, "America/Chicago", func(string, ...interface{}) {})

		if data.MoonIllum < 0 || data.MoonIllum > 100 {
			t.Errorf("Moon illumination should be between 0 and 100, got %.2f", data.MoonIllum)
		}
	})

	t.Run("MoonTrendValid", func(t *testing.T) {
		data := CalculateData(41.8781, -87.6298, "America/Chicago", func(string, ...interface{}) {})

		if data.MoonTrend != "Waxing" && data.MoonTrend != "Waning" {
			t.Errorf("Moon trend should be 'Waxing' or 'Waning', got %s", data.MoonTrend)
		}
	})
}

func TestCalculateDataWithDefaultLogger(t *testing.T) {
	t.Run("UsesDefaultLogger", func(t *testing.T) {
		// Should not panic
		data := CalculateDataWithDefaultLogger(41.8781, -87.6298, "America/Chicago")

		// Should return valid data
		if data.Sunrise == "" || data.Sunset == "" {
			t.Error("Should return valid astronomical data")
		}
	})
}