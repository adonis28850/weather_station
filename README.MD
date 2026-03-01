# Weather Station

A self-hosted weather station system that captures RF signals from an Ecowitt WS90 weather sensor using an RTL-SDR dongle, stores data in PostgreSQL, and provides a web dashboard for visualization.

## Overview

The system consists of two Go applications:

- **Ingestor**: Captures RF signals from the weather sensor and sends data to the core service
- **Core**: Receives data, stores it in PostgreSQL, and provides a REST API and web dashboard

## Architecture

```
┌──────────────────┐
│  Web Dashboard   │
└────────┬─────────┘
         │ HTTP
         ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Core (Go)                                      │
│  • HTTP server with REST API                                                │
│  • PostgreSQL database interface                                            │
│  • Embedded web dashboard (HTML/CSS/JS)                                     │
│  • Data validation and storage                                              │
└────────────────────┬────────────────────────────────────────────────────────┘
            ▲       SQL
            │        ▼
            │    ┌───────────────────────┐
            │    │    PostgreSQL         │
            │    └───────────────────────┘
            │
            │ HTTP POST
┌───────────┴─────────────────────────────────────────────────────────────────┐
│                           Ingestor (Go)                                     │
│  • Spawns rtl_433 subprocess to capture RF signals                          │
│  • Parses JSON output from rtl_433                                          │
│  • Validates and sends data to Core via HTTP                                │
│  • Worker pool for concurrent data processing                               │
└────────────────────┬────────────────────────────────────────────────────────┘
                     │ USB
                     ▼
                ┌───────────────────────┐
                │     RTL-SDR Dongle    │
                └───────────────────────┘
                     │ RF Signal
                     ▼
                ┌───────────────────────┐
                │     Ecowitt WS90      │
                │    915/433 MHz        │
                └───────────────────────┘
```

## Components

### Ingestor

The ingestor is responsible for capturing weather data from the RF sensor:

- Spawns `rtl_433` as a subprocess to decode RF signals
- Reads JSON output line-by-line from stdout
- Validates sensor readings
- Uses a worker pool for concurrent HTTP POST requests to the Core service
- Implements automatic restart with exponential backoff on failure
- Captures stderr for debugging

**Key Features:**
- Configurable number of workers for concurrent processing
- Retry logic with exponential backoff for failed HTTP requests
- Data validation before sending to Core service
- Robust process management with auto-restart

### Core

The core service provides the main application logic:

- HTTP server with REST API endpoints
- PostgreSQL database interface with connection pooling
- Embedded web dashboard for data visualization
- Data validation and storage
- Daily rollup aggregation for efficient historical queries
- Automatic data retention management

**API Endpoints:**
- `POST /api/ingest`: Receive weather data from ingestor
- `GET /api/weather/current`: Get current/latest weather reading
- `GET /api/weather/recent`: Get recent weather readings (≤24 hours)
- `GET /api/weather/history`: Get historical weather data (aggregated)
- `GET /api/weather/years`: Get available years with data
- `GET /health`: Health check endpoint
- `GET /`: Serve web dashboard

### Web Dashboard

The embedded dashboard provides:

- **Current Weather Tab**: Real-time weather data with interactive charts
  - Temperature, humidity, UV index, light levels
  - Wind speed and gust
  - Rainfall
  - Statistics summary cards
  - Time range selection (1 hour, 6 hours, 24 hours)
  - Tooltips on all charts

- **Historical Analytics Tab**: Long-term data analysis
  - Year and month filtering
  - Temperature charts (high, low, average)
  - Humidity charts (high, low, average)
  - Rainfall chart
  - Wind charts (max gust, mean speed)
  - Sortable data table
  - Year boundary markers on charts
  - Interactive tooltips

**Dashboard Features:**
- Responsive design with dark theme
- Chart.js for interactive visualizations
- Automatic data sorting
- Table sorting by any column
- Dynamic year dropdown populated from database
- Tooltips on all charts for detailed data inspection

## Database Schema

The system uses PostgreSQL with two main tables:

### `readings` Table
Stores raw sensor readings (retained for 30 days by default):
- `sensor_id`: Unique sensor identifier
- `timestamp`: Reading timestamp
- `temperature_c`: Temperature in Celsius
- `humidity`: Relative humidity percentage
- `uv`: UV index
- `light_lux`: Light level in lux
- `wind_speed_m_s`: Wind speed in m/s
- `wind_gust_m_s`: Wind gust in m/s
- `wind_dir_deg`: Wind direction in degrees
- `rain_mm`: Rainfall in mm
- `battery`: Battery level
- `model`: Sensor model

### `daily_weather` Table
Stores daily aggregated data (retained indefinitely):
- `day_date`: Date of the aggregation
- `temp_high_c`: Daily high temperature
- `temp_low_c`: Daily low temperature
- `humidity_high`: Daily high humidity
- `humidity_low`: Daily low humidity
- `rain_mm`: Total rainfall
- `wind_max_gust_m_s`: Maximum wind gust
- `wind_mean_m_s`: Mean wind speed
- `wind_sample_count`: Number of wind samples
- `readings_count`: Total readings for the day
- `first_reading_ts`: First reading timestamp
- `last_reading_ts`: Last reading timestamp
- `uv_max`: Maximum UV index
- `light_max`: Maximum light level

## Data Flow

1. **Signal Capture**: Ecowitt WS90 transmits weather data via RF (915/433 MHz)
2. **Signal Decoding**: RTL-SDR dongle captures RF signals, `rtl_433` decodes them to JSON
3. **Data Ingestion**: Ingestor reads JSON output, validates data, and sends to Core via HTTP POST
4. **Data Storage**: Core receives data, validates again, and stores in `readings` table
5. **Daily Aggregation**: Scheduled job aggregates daily data into `daily_weather` table
6. **Data Access**: Dashboard queries appropriate table based on time range requested
7. **Data Retention**: Scheduled job deletes old raw readings (default: 30 days)

## Building

### Prerequisites
- Go 1.21 or later
- PostgreSQL 12 or later
- RTL-SDR hardware and drivers

### Build Core
```bash
cd core
go build -o core
```

### Build Ingestor
```bash
cd ingestor
go build -o ingestor
```

## Running

### Environment Variables

**Core (Mandatory):**
- `WEATHER_DB_HOST`: PostgreSQL host
- `WEATHER_DB_PORT`: PostgreSQL port (default: 5432)
- `WEATHER_DB_NAME`: Database name
- `WEATHER_DB_USER`: Database user
- `WEATHER_DB_PASSWORD`: Database password

**Core (Optional):**
- `WEATHER_SERVER_PORT`: HTTP server port (default: 8080)
- `WEATHER_INGESTION_CHAN_SIZE`: Ingestion channel size (default: 1000)
- `WEATHER_JOB_QUEUE_SIZE`: Job queue size (default: 100)
- `WEATHER_WORKER_COUNT`: Number of worker goroutines (default: 5)
- `WEATHER_RESULT_QUEUE_SIZE`: Result queue size (default: 100)
- `WEATHER_AGGREGATION_SCHEDULE`: Cron schedule for aggregation (default: "0 0 * * *")
- `WEATHER_RETENTION_DAYS`: Raw data retention in days (default: 30)
- `WEATHER_LATITUDE`: Geographic latitude (default: 41.8262)
- `WEATHER_LONGITUDE`: Geographic longitude (default: -87.6841)
- `WEATHER_TIMEZONE`: Timezone (default: "UTC")
- `WEATHER_DB_MAX_OPEN_CONNS`: Max open DB connections (default: 25)
- `WEATHER_DB_MAX_IDLE_CONNS`: Max idle DB connections (default: 5)
- `WEATHER_DB_CONN_MAX_LIFETIME_SECONDS`: DB connection lifetime (default: 300)
- `WEATHER_DB_MAX_RETRIES`: Max DB connection retries (default: 5)
- `WEATHER_DB_RETRY_INTERVAL_SECONDS`: DB retry interval (default: 5)
- `WEATHER_ENABLE_HTTP_LOGGING`: Enable HTTP request logging (default: true)

**Ingestor (Mandatory):**
- `CORE_API_URL`: Core service URL (e.g., `http://localhost:8080`)
- `CORE_API_TIMEOUT_SECONDS`: HTTP request timeout in seconds
- `CORE_API_MAX_RETRIES`: Maximum retry attempts
- `CORE_API_BACKOFF_DURATION`: Backoff duration in seconds

**Ingestor (Optional):**
- `RTL_433_COMMAND`: rtl_433 command (default: `rtl_433 -f 868M -F json -M utc`)
- `CORE_API_WORKER_COUNT`: Number of worker goroutines (default: 5)
- `CORE_API_JOB_QUEUE_SIZE`: Job queue size (default: 100)
- `CORE_API_RESULT_QUEUE_SIZE`: Result queue size (default: 100)

### Database Setup

Run migrations in order:
```bash
# Create the database first (run against postgres database)
psql -U postgres -f migrations/000_create_database.sql

# Run remaining migrations against the weather_station database
psql -U postgres -d weather_station -f migrations/001_init_schema.sql
psql -U postgres -d weather_station -f migrations/002_create_retention_function.sql
psql -U postgres -d weather_station -f migrations/003_create_daily_aggregation_function.sql
```

### Start Services

1. Start PostgreSQL
2. Start Core:
```bash
./core
```

3. Start Ingestor:
```bash
./ingestor
```

4. Access dashboard at `http://localhost:8080`

## Development

### Project Structure
```
weather_station/
├── core/             # Core service
│   ├── main.go       # Main application
│   ├── handlers/     # HTTP handlers
│   ├── config/       # Configuration management
│   ├── database/     # Database operations
│   ├── astronomical/ # Astronomical calculations
│   └── static/       # Web dashboard (HTML/CSS/JS)
├── ingestor/         # Ingestor service
│   └── main.go       # Main application
├── shared/           # Shared code
│   ├── logger/       # Logging utilities
│   ├── types/        # Common types
│   ├── validation/   # Data validation
│   └── workers/      # Worker pool implementation
└── migrations/       # Database migrations
```

### Testing

For testing without RTL-SDR hardware, use the mock script:
```bash
python3 mock_rtl_433.py | ./ingestor
```

## Data Volume

The WS90 transmits approximately every 5 seconds:
- **Readings per day**: ~17,280
- **Raw data per day**: ~3.5 MB
- **30-day retention**: ~104 MB
- **Aggregated data**: Minimal storage (one row per day)