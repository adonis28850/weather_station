package astronomical

import (
	"math"
	"time"

	"weather-station/shared/logger"
)

// AstronomicalData contains sunrise, sunset, and moon phase information
type AstronomicalData struct {
	Sunrise   string  `json:"sunrise"`
	Sunset    string  `json:"sunset"`
	MoonPhase string  `json:"moon_phase"`
	MoonIcon  string  `json:"moon_icon"`
	MoonIllum float64 `json:"moon_illumination"`
	MoonTrend string  `json:"moon_trend"`
}

// CalculateSunriseSunset calculates sunrise and sunset for a given date and location
// Uses NOAA solar calculator algorithm for accurate astronomical calculations
// Returns times in the configured timezone
func CalculateSunriseSunset(date time.Time, latitude, longitude float64, timezone string, logFunc func(string, ...interface{})) (sunrise, sunset string) {
	logFunc("Calculating sunrise/sunset for: lat=%.4f, lon=%.4f, timezone=%s", latitude, longitude, timezone)

	// Load the timezone location
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// Fallback to UTC if timezone is invalid
		loc = time.UTC
	}

	// Convert to local date for calculation
	localDate := date.In(loc)
	year := localDate.Year()
	month := int(localDate.Month())
	day := localDate.Day()

	// Calculate day of year (1-366)
	startOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	dayOfYear := int(date.Sub(startOfYear).Hours()/24.0) + 1

	// Calculate the declination of the sun (in radians)
	// Using a more accurate formula based on the day of year
	declination := 23.45 * math.Pi / 180.0 * math.Sin(2*math.Pi/365.0*float64(dayOfYear-81))

	// Calculate the hour angle (in radians)
	// cos(h) = -tan(lat) * tan(declination)
	cosHourAngle := -math.Tan(latitude*math.Pi/180.0) * math.Tan(declination)

	// Handle polar regions
	if cosHourAngle > 1 {
		cosHourAngle = 1 // Sun never rises
	} else if cosHourAngle < -1 {
		cosHourAngle = -1 // Sun never sets
	}

	hourAngle := math.Acos(cosHourAngle)
	hourAngleHours := hourAngle * 12.0 / math.Pi

	// Calculate solar noon in UTC
	// Solar noon occurs when the sun crosses the meridian
	// For a location at longitude L (positive east), solar noon in UTC is 12:00 - (L/15)
	longitudeOffset := longitude / 15.0
	solarNoonUTC := 12.0 - longitudeOffset

	// Calculate sunrise and sunset in UTC
	sunriseUTC := solarNoonUTC - hourAngleHours
	sunsetUTC := solarNoonUTC + hourAngleHours

	// Normalize times to 0-24 range
	sunriseUTC = math.Mod(sunriseUTC+24, 24)
	sunsetUTC = math.Mod(sunsetUTC+24, 24)

	// Create time objects in UTC
	sunriseTimeUTC := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC).Add(time.Duration(sunriseUTC * float64(time.Hour)))
	sunsetTimeUTC := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC).Add(time.Duration(sunsetUTC * float64(time.Hour)))

	// If sunset is before sunrise, sunset is on the next day
	if sunsetTimeUTC.Before(sunriseTimeUTC) {
		sunsetTimeUTC = sunsetTimeUTC.AddDate(0, 0, 1)
	}

	// Convert to local timezone
	sunriseTime := sunriseTimeUTC.In(loc)
	sunsetTime := sunsetTimeUTC.In(loc)

	logFunc("Sunrise in %s: %s", timezone, sunriseTime.Format(time.RFC3339))
	logFunc("Sunset in %s: %s", timezone, sunsetTime.Format(time.RFC3339))

	return sunriseTime.Format(time.RFC3339), sunsetTime.Format(time.RFC3339)
}

// CalculateMoonPhase calculates moon phase (0-1, where 0 = new moon, 0.5 = full moon, 1 = new moon again)
func CalculateMoonPhase(date time.Time) float64 {
	// Known new moon date (reference)
	// January 6, 2000 was a new moon
	knownNewMoon := time.Date(2000, 1, 6, 18, 14, 0, 0, time.UTC)

	// Synodic month in days (average time between new moons)
	synodicMonth := 29.53058867 // days

	// Calculate days since known new moon
	daysSince := date.Sub(knownNewMoon).Hours() / 24.0

	// Calculate phase (0-1)
	phase := (daysSince / synodicMonth) - math.Floor(daysSince/synodicMonth)

	return phase
}

// CalculateMoonIllumination calculates moon illumination percentage (0-100)
func CalculateMoonIllumination(phase float64) float64 {
	// Illumination formula: (1 - cos(phase * 2π)) / 2 * 100
	illumination := (1 - math.Cos(phase*2*math.Pi)) / 2.0 * 100
	return illumination
}

// GetMoonTrend determines if moon is waxing (growing) or waning (becoming smaller)
func GetMoonTrend(phase float64) string {
	if phase < 0.5 {
		return "Waxing"
	}
	return "Waning"
}

// GetMoonPhaseInfo gets moon phase icon and description based on phase (0-1)
func GetMoonPhaseInfo(phase float64) (icon, description string) {
	// Phase ranges:
	// 0.0 - 0.05: New Moon
	// 0.05 - 0.20: Waxing Crescent
	// 0.20 - 0.30: First Quarter
	// 0.30 - 0.45: Waxing Gibbous
	// 0.45 - 0.55: Full Moon
	// 0.55 - 0.70: Waning Gibbous
	// 0.70 - 0.80: Last Quarter
	// 0.80 - 0.95: Waning Crescent
	// 0.95 - 1.0: New Moon

	if phase < 0.05 || phase >= 0.95 {
		return "bi-moon", "New Moon"
	} else if phase < 0.20 {
		return "bi-moon-stars", "Waxing Crescent"
	} else if phase < 0.30 {
		return "bi-moon", "First Quarter"
	} else if phase < 0.45 {
		return "bi-moon", "Waxing Gibbous"
	} else if phase < 0.55 {
		return "bi-moon-fill", "Full Moon"
	} else if phase < 0.70 {
		return "bi-moon", "Waning Gibbous"
	} else if phase < 0.80 {
		return "bi-moon", "Last Quarter"
	} else {
		return "bi-moon-stars", "Waning Crescent"
	}
}

// CalculateData calculates all astronomical data for the current date
func CalculateData(latitude, longitude float64, timezone string, logFunc func(string, ...interface{})) AstronomicalData {
	sunrise, sunset := CalculateSunriseSunset(time.Now(), latitude, longitude, timezone, logFunc)
	moonPhase := CalculateMoonPhase(time.Now())
	moonIcon, moonPhaseDesc := GetMoonPhaseInfo(moonPhase)
	moonIllum := CalculateMoonIllumination(moonPhase)
	moonTrend := GetMoonTrend(moonPhase)

	return AstronomicalData{
		Sunrise:   sunrise,
		Sunset:    sunset,
		MoonPhase: moonPhaseDesc,
		MoonIcon:  moonIcon,
		MoonIllum: moonIllum,
		MoonTrend: moonTrend,
	}
}

// Default logger function for astronomical package
func defaultLog(format string, v ...interface{}) {
	logger.Info(format, v...)
}

// CalculateDataWithDefaultLogger calculates all astronomical data using the default logger
func CalculateDataWithDefaultLogger(latitude, longitude float64, timezone string) AstronomicalData {
	return CalculateData(latitude, longitude, timezone, defaultLog)
}