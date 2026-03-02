package types

// Reading represents a weather sensor reading from rtl_433
// This struct matches the JSON format produced by the rtl_433 tool
type Reading struct {
	Time         string  `json:"time"`
	Model        string  `json:"model"`
	SensorID     int     `json:"id"`
	TemperatureC float64 `json:"temperature_C"`
	Humidity     int     `json:"humidity"`
	UVIndex      float64 `json:"uvi"`
	Lux          int     `json:"light_lux"`
	WindSpeedMS  float64 `json:"wind_avg_m_s"`
	WindGustMS   float64 `json:"wind_max_m_s"`
	WindDirDeg   int     `json:"wind_dir_deg"`
	RainMM       float64 `json:"rain_mm"`
	BatteryOK    float64 `json:"battery_ok"`
}