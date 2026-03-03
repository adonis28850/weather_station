package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"weather-station/core/astronomical"
	"weather-station/core/config"
	"weather-station/shared/logger"
	"weather-station/shared/types"
	"weather-station/shared/validation"
)

// Database query timeout (5 seconds)
// Rationale: Prevents slow queries from blocking goroutines indefinitely
// 5 seconds is sufficient for most queries while protecting against database overload
const QueryTimeout = 5 * time.Second

// Response struct for current weather with additional data
type CurrentWeatherResponse struct {
	Time          string                        `json:"time"`
	Model         string                        `json:"model"`
	ID            int                           `json:"id"`
	TemperatureC  float64                       `json:"temperature_C"`
	Humidity      int                           `json:"humidity"`
	UVIndex       float64                       `json:"uvi"`
	Lux           float64                       `json:"light_lux"`
	WindSpeedMS   float64                       `json:"wind_avg_m_s"`
	WindGustMS    float64                       `json:"wind_max_m_s"`
	WindDirDeg    int                           `json:"wind_dir_deg"`
	CurrentRainMM float64                       `json:"current_rain_mm"` // Rain from latest reading
	DailyRainMM   float64                       `json:"daily_rain_mm"`   // Total rain for today
	BatteryOK     float64                       `json:"battery"`
	Astronomical  astronomical.AstronomicalData `json:"astronomical"` // Sunrise, sunset, etc.
	Latitude      float64                       `json:"latitude"`     // Station latitude
	Longitude     float64                       `json:"longitude"`    // Station longitude
}

// Struct for daily rollups
type DailyRollup struct {
	DayDate         string
	TempHighC       float64
	TempLowC        float64
	HumidityHigh    int
	HumidityLow     int
	RainMM          float64
	WindMaxGustMS   float64
	WindMeanMS      float64
	WindSampleCount int
	ReadingsCount   int
	FirstReadingTS  string
	LastReadingTS   string
	UVMax           float64
	LightMax        int
}

// Server represents the HTTP server with database and configuration
type Server struct {
	db                 *sql.DB
	config             config.Config
	cachedAstronomical astronomical.AstronomicalData
	astronomicalMutex  sync.RWMutex
}

// NewServer creates a new HTTP server instance
func NewServer(db *sql.DB, cfg config.Config) *Server {
	return &Server{
		db:     db,
		config: cfg,
	}
}

// MethodCheck middleware validates HTTP method
func MethodCheck(allowedMethod string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != allowedMethod {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		next(w, r)
	}
}

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(next http.Handler, enableHTTPLogging bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !enableHTTPLogging {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()

		next.ServeHTTP(w, r)

		logger.Plain("[LOGGER] %s %s %s %v", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start))
	})
}

// DailyRollupReading extends types.Reading with daily rollup fields
type DailyRollupReading struct {
	types.Reading
	TemperatureHighC float64 `json:"temperature_high_c,omitempty"`
	TemperatureLowC  float64 `json:"temperature_low_c,omitempty"`
	HumidityHigh     int     `json:"humidity_high,omitempty"`
	HumidityLow      int     `json:"humidity_low,omitempty"`
}

// sendJSON sends a JSON response with the given status code
func sendJSON(w http.ResponseWriter, data any, status int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		logger.Error("Failed to encode JSON response: %v", err)
		return err
	}

	return nil
}

// sendError sends an error JSON response with the given status code
func sendError(w http.ResponseWriter, message string, status int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(map[string]string{"error": message})
	if err != nil {
		logger.Error("Failed to encode JSON response: %v", err)
		return err
	}

	return nil
}

// IngestHandler handles POST /api/ingest requests
func IngestHandler(ingestionChan chan<- types.Reading) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
			return
		}

		var reading types.Reading
		err := json.NewDecoder(r.Body).Decode(&reading)
		if err != nil {
			logger.Error("Failed to decode JSON: %v", err)
			http.Error(w, "Invalid JSON format", http.StatusBadRequest)
			return
		}

		if err := validation.ValidateReading(reading); err != nil {
			logger.Error("Invalid reading: %v. Skipping.", err)
			http.Error(w, "Invalid JSON data", http.StatusBadRequest)
			return
		}

		select {
		case ingestionChan <- reading:
		default:
			logger.Error("Ingestion channel is full, dropping reading")
			http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
			return
		}

		err = sendJSON(w, map[string]string{"status": "accepted"}, http.StatusAccepted)
		if err != nil {
			logger.Error("Failed to send JSON response: %v", err)
		}
	}
}

// CurrentWeatherHandler handles GET /api/weather/current requests
func (s *Server) CurrentWeatherHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	query := s.db.QueryRowContext(ctx,
		"SELECT sensor_id, timestamp, temperature_c, humidity, uv, light_lux, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model FROM readings ORDER BY timestamp DESC LIMIT 1")

	var reader types.Reading
	var dbSensorID int
	var timestamp time.Time

	err := query.Scan(&dbSensorID, &timestamp, &reader.TemperatureC,
		&reader.Humidity, &reader.UVIndex, &reader.Lux, &reader.WindSpeedMS,
		&reader.WindGustMS, &reader.WindDirDeg, &reader.RainMM, &reader.BatteryOK, &reader.Model)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Info("No data available")
			sendError(w, "No data available", http.StatusNotFound)
			return
		} else if errors.Is(err, context.DeadlineExceeded) {
			logger.Error("Query timeout while fetching current weather: %v", err)
			sendError(w, "Database query timeout", http.StatusGatewayTimeout)
			return
		} else {
			logger.Error("Failed to query DB for most recent reading: %v", err)
			sendError(w, "Failed to query DB for most recent reading", http.StatusInternalServerError)
			return
		}
	}

	ctx, cancel = context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	var dailyRain float64
	startOfDay := time.Now().Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24 * time.Hour)
	rainQuery := s.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(rain_mm), 0) FROM readings WHERE timestamp >= $1 AND timestamp < $2", startOfDay, endOfDay)
	err = rainQuery.Scan(&dailyRain)

	if err != nil {
		logger.Error("Failed to query daily rainfall: %v", err)
		dailyRain = 0
	}

	s.astronomicalMutex.RLock()
	astronomicalData := s.cachedAstronomical
	s.astronomicalMutex.RUnlock()

	response := CurrentWeatherResponse{
		Time:          timestamp.Format(time.RFC3339Nano),
		Model:         reader.Model,
		ID:            dbSensorID,
		TemperatureC:  reader.TemperatureC,
		Humidity:      reader.Humidity,
		UVIndex:       reader.UVIndex,
		Lux:           reader.Lux,
		WindSpeedMS:   reader.WindSpeedMS,
		WindGustMS:    reader.WindGustMS,
		WindDirDeg:    reader.WindDirDeg,
		CurrentRainMM: reader.RainMM,
		DailyRainMM:   dailyRain,
		BatteryOK:     reader.BatteryOK,
		Astronomical:  astronomicalData,
		Latitude:      s.config.Latitude(),
		Longitude:     s.config.Longitude(),
	}

	err = sendJSON(w, response, http.StatusOK)
	if err != nil {
		logger.Error("Failed to send JSON response: %v", err)
	}
}

// HistoryWeatherHandler handles GET /api/weather/history requests
// Returns aggregated daily weather data from daily_weather table
func (s *Server) HistoryWeatherHandler(w http.ResponseWriter, r *http.Request) {
	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")

	var startTime, endTime time.Time

	if start != "" {
		var err error
		startTime, err = time.Parse(time.RFC3339, start)
		if err != nil {
			logger.Error("Failed to parse start date: %v", err)
			http.Error(w, "Invalid start date format", http.StatusBadRequest)
			return
		}
	}
	if end != "" {
		var err error
		endTime, err = time.Parse(time.RFC3339, end)
		if err != nil {
			logger.Error("Failed to parse end date: %v", err)
			http.Error(w, "Invalid end date format", http.StatusBadRequest)
			return
		}
	}

	if startTime.IsZero() {
		startTime = time.Now().AddDate(0, 0, -s.config.RetentionDays())
	}
	if endTime.IsZero() {
		endTime = time.Now()
	}

	// Query daily_weather table for aggregated data
	dailyQuery := `SELECT day_date, temp_high_c, temp_low_c, humidity_high, humidity_low, rain_mm, wind_max_gust_m_s, wind_mean_m_s, wind_sample_count, readings_count, first_reading_ts, last_reading_ts, uv_max, light_max FROM daily_weather WHERE day_date BETWEEN $1 AND $2 ORDER BY day_date ASC`

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()
	dailyRows, err := s.db.QueryContext(ctx, dailyQuery, startTime, endTime)
	if err != nil {
		logger.Error("Failed to query daily_weather: %v", err)
		if errors.Is(err, context.DeadlineExceeded) {
			http.Error(w, "Database query timeout", http.StatusGatewayTimeout)
		} else {
			http.Error(w, "Failed to query DB for history", http.StatusInternalServerError)
		}
		return
	}
	defer dailyRows.Close()

	var readings []DailyRollupReading

	for dailyRows.Next() {
		var rollup DailyRollup
		var dayDate time.Time
		var firstReadingTS time.Time
		var lastReadingTS time.Time

		err := dailyRows.Scan(&dayDate, &rollup.TempHighC, &rollup.TempLowC, &rollup.HumidityHigh, &rollup.HumidityLow, &rollup.RainMM, &rollup.WindMaxGustMS, &rollup.WindMeanMS, &rollup.WindSampleCount, &rollup.ReadingsCount, &firstReadingTS, &lastReadingTS, &rollup.UVMax, &rollup.LightMax)

		if err != nil {
			logger.Error("Failed to scan daily rollup: %v", err)
			continue
		}

		avgTemp := (rollup.TempHighC + rollup.TempLowC) / 2
		avgHumidity := (rollup.HumidityHigh + rollup.HumidityLow) / 2

		reader := DailyRollupReading{
			Reading: types.Reading{
				Time:         dayDate.Format(time.RFC3339Nano),
				Model:        "Daily Rollup",
				SensorID:     1,
				TemperatureC: avgTemp,
				Humidity:     int(avgHumidity),
				UVIndex:      rollup.UVMax,
				Lux:          float64(rollup.LightMax),
				WindSpeedMS:  rollup.WindMeanMS,
				WindGustMS:   rollup.WindMaxGustMS,
				WindDirDeg:   0,
				RainMM:       rollup.RainMM,
				BatteryOK:    1.0,
			},
			TemperatureHighC: rollup.TempHighC,
			TemperatureLowC:  rollup.TempLowC,
			HumidityHigh:     rollup.HumidityHigh,
			HumidityLow:      rollup.HumidityLow,
		}

		readings = append(readings, reader)
	}

	if len(readings) == 0 {
		sendError(w, "No data available for the specified period", http.StatusNotFound)
		return
	}

	if err := sendJSON(w, readings, http.StatusOK); err != nil {
		logger.Error("Failed to send JSON response in history handler: %v", err)
	}
}

// AvailableYearsHandler handles GET /api/weather/years requests
// Returns a list of years that have data in the daily_weather table
func (s *Server) AvailableYearsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	query := `SELECT DISTINCT EXTRACT(YEAR FROM day_date) AS year FROM daily_weather ORDER BY year DESC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		logger.Error("Failed to query available years: %v", err)
		if errors.Is(err, context.DeadlineExceeded) {
			http.Error(w, "Database query timeout", http.StatusGatewayTimeout)
		} else {
			http.Error(w, "Failed to query available years", http.StatusInternalServerError)
		}
		return
	}
	defer rows.Close()

	var years []int
	for rows.Next() {
		var year int
		if err := rows.Scan(&year); err != nil {
			logger.Error("Failed to scan year: %v", err)
			continue
		}
		years = append(years, year)
	}

	if err := sendJSON(w, years, http.StatusOK); err != nil {
		logger.Error("Failed to send JSON response: %v", err)
	}
}

// RecentWeatherHandler handles GET /api/weather/recent requests
// Returns raw readings data from readings table for short time ranges
func (s *Server) RecentWeatherHandler(w http.ResponseWriter, r *http.Request) {
	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")
	limit := r.URL.Query().Get("limit")

	var startTime, endTime time.Time

	if start != "" {
		var err error
		startTime, err = time.Parse(time.RFC3339, start)
		if err != nil {
			logger.Error("Failed to parse start date: %v", err)
			http.Error(w, "Invalid start date format", http.StatusBadRequest)
			return
		}
	}
	if end != "" {
		var err error
		endTime, err = time.Parse(time.RFC3339, end)
		if err != nil {
			logger.Error("Failed to parse end date: %v", err)
			http.Error(w, "Invalid end date format", http.StatusBadRequest)
			return
		}
	}

	if startTime.IsZero() {
		startTime = time.Now().AddDate(0, 0, -1)
	}
	if endTime.IsZero() {
		endTime = time.Now()
	}

	limitInt := 1000
	if limit != "" {
		var err error
		limitInt, err = strconv.Atoi(limit)
		if err != nil {
			logger.Error("Failed to parse limit: %v", err)
			http.Error(w, "Invalid limit value", http.StatusBadRequest)
			return
		}
	}

	// Query readings table for raw data
	query := `SELECT sensor_id, timestamp, temperature_c, humidity, uv, light_lux, wind_speed_m_s, wind_gust_m_s, wind_dir_deg, rain_mm, battery, model FROM readings WHERE timestamp BETWEEN $1 AND $2 ORDER BY timestamp ASC LIMIT $3`

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()
	rows, err := s.db.QueryContext(ctx, query, startTime, endTime, limitInt)
	if err != nil {
		logger.Error("Failed to query DB for recent readings: %v", err)
		if errors.Is(err, context.DeadlineExceeded) {
			http.Error(w, "Database query timeout", http.StatusGatewayTimeout)
		} else {
			http.Error(w, "Failed to query DB for recent readings", http.StatusInternalServerError)
		}
		return
	}
	defer rows.Close()

	var readings []types.Reading

	for rows.Next() {
		var reader types.Reading
		var timestamp time.Time

		err := rows.Scan(&reader.SensorID,
			&timestamp,
			&reader.TemperatureC,
			&reader.Humidity,
			&reader.UVIndex,
			&reader.Lux,
			&reader.WindSpeedMS,
			&reader.WindGustMS,
			&reader.WindDirDeg,
			&reader.RainMM,
			&reader.BatteryOK,
			&reader.Model)

		if err != nil {
			logger.Error("Failed to scan row in recent weather handler: %v", err)
			continue
		}

		reader.Time = timestamp.Format(time.RFC3339Nano)
		readings = append(readings, reader)
	}

	if len(readings) == 0 {
		sendError(w, "No data available for the specified period", http.StatusNotFound)
		return
	}

	if err := sendJSON(w, readings, http.StatusOK); err != nil {
		logger.Error("Failed to send JSON response in recent weather handler: %v", err)
	}
}

// HealthCheckHandler handles GET /health requests
func (s *Server) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	if err := s.db.PingContext(ctx); err != nil {
		logger.Error("Health check failed: database ping error: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "unhealthy",
			"message": "Database connection failed",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"message": "Service is running",
	})
}

// UpdateAstronomicalData updates the cached astronomical data
func (s *Server) UpdateAstronomicalData() {
	data := astronomical.CalculateDataWithDefaultLogger(s.config.Latitude(), s.config.Longitude(), s.config.Timezone())

	s.astronomicalMutex.Lock()
	s.cachedAstronomical = data
	s.astronomicalMutex.Unlock()
}
