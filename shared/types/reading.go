package types

// Reading represents a weather sensor reading from rtl_433
// This struct matches the JSON format produced by the rtl_433 tool
type Reading struct {
	Time         string  `json:"time"`
	Model        string  `json:"model"`
	SensorID     int     `json:"id"`
	TemperatureC float64 `json:"temperature_C"`
	Humidity     int     `json:"humidity"`
	UVIndex      float64 `json:"uv"`
	Lux          int     `json:"light_lux"`
	WindSpeedMS  float64 `json:"wind_speed_m_s"`
	WindGustMS   float64 `json:"wind_gust_m_s"`
	WindDirDeg   int     `json:"wind_dir_deg"`
	RainMM       float64 `json:"rain_mm"`
	BatteryOK    string  `json:"battery"`
}