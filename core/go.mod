module weather-station/core

go 1.25

require (
	github.com/lib/pq v1.11.2
	weather-station/shared v0.0.0-00010101000000-000000000000
)

replace weather-station/shared => ../shared
