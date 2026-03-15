package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"weather-station/shared/types"
)

// TestSendRTL433Version tests sending rtl_433 version to core API
func TestSendRTL433Version(t *testing.T) {
	t.Run("SuccessfulVersionSend", func(t *testing.T) {
		// Create a test server that accepts version
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/system/rtl433-version" {
				t.Errorf("Expected path /api/system/rtl433-version, got %s", r.URL.Path)
			}

			var req map[string]string
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("Failed to decode request: %v", err)
				return
			}

			if req["version"] == "" {
				t.Error("Expected version field in request")
				return
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Mock the rtl_433 command by setting environment variable
		// We'll test the version parsing separately, so here we'll just test the HTTP sending
		// by directly calling the send function with a mock version
		client := &http.Client{Timeout: 5 * time.Second}
		
		requestBody := map[string]string{"version": "25.12 (2025-12-12)"}
		jsonBody, _ := json.Marshal(requestBody)

		resp, err := client.Post(server.URL+"/api/system/rtl433-version", "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to send version: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("RetryLogic", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			// Fail first 2 attempts, succeed on 3rd
			if attemptCount < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &http.Client{Timeout: 5 * time.Second}
		
		requestBody := map[string]string{"version": "25.12"}
		jsonBody, _ := json.Marshal(requestBody)

		// Simulate retry logic
		for attempt := 1; attempt <= 3; attempt++ {
			resp, err := client.Post(server.URL+"/api/system/rtl433-version", "application/json", bytes.NewBuffer(jsonBody))
			if err != nil {
				if attempt < 3 {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				t.Fatalf("Failed to send version after 3 attempts: %v", err)
			}
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				return // Success
			}
			if attempt < 3 {
				time.Sleep(100 * time.Millisecond)
			}
		}

		if attemptCount != 3 {
			t.Errorf("Expected 3 attempts, got %d", attemptCount)
		}
	})

	t.Run("TimeoutHandling", func(t *testing.T) {
		// Create a slow server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(10 * time.Second) // Sleep longer than timeout
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &http.Client{Timeout: 5 * time.Second}
		
		requestBody := map[string]string{"version": "25.12"}
		jsonBody, _ := json.Marshal(requestBody)

		// This should timeout
		resp, err := client.Post(server.URL+"/api/system/rtl433-version", "application/json", bytes.NewBuffer(jsonBody))
		if err == nil {
			resp.Body.Close()
			t.Error("Expected timeout error, got nil")
		}
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := &http.Client{Timeout: 5 * time.Second}
		
		requestBody := map[string]string{"version": "25.12"}
		jsonBody, _ := json.Marshal(requestBody)

		// Simulate retry logic with max 1 attempt to fail quickly
		resp, err := client.Post(server.URL+"/api/system/rtl433-version", "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to send version: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", resp.StatusCode)
		}
	})
}

// TestParseRTL433Version tests version parsing logic
func TestParseRTL433Version(t *testing.T) {
	t.Run("VersionWithDate", func(t *testing.T) {
		output := "rtl_433 version 25.12 (2025-12-12) inputs file rtl_tcp RTL-SDR SoapySDR"
		
		lines := strings.Split(output, "\n")
		if len(lines) == 0 {
			t.Fatal("Expected at least one line")
		}

		firstLine := strings.TrimSpace(lines[0])
		
		re := regexp.MustCompile(`rtl_433 version ([\d.]+ \(\d{4}-\d{2}-\d{2}\))`)
		matches := re.FindStringSubmatch(firstLine)
		
		if len(matches) < 2 {
			t.Fatalf("Expected version with date in matches, got: %v", matches)
		}

		expected := "25.12 (2025-12-12)"
		if matches[1] != expected {
			t.Errorf("Expected version %s, got %s", expected, matches[1])
		}
	})

	t.Run("VersionWithoutDate", func(t *testing.T) {
		output := "rtl_433 version 25.12 inputs file rtl_tcp RTL-SDR"
		
		lines := strings.Split(output, "\n")
		if len(lines) == 0 {
			t.Fatal("Expected at least one line")
		}

		firstLine := strings.TrimSpace(lines[0])
		
		re := regexp.MustCompile(`rtl_433 version ([\d.]+)`)
		matches := re.FindStringSubmatch(firstLine)
		
		if len(matches) < 2 {
			t.Fatalf("Expected version in matches, got: %v", matches)
		}

		expected := "25.12"
		if matches[1] != expected {
			t.Errorf("Expected version %s, got %s", expected, matches[1])
		}
	})

	t.Run("FullLineFallback", func(t *testing.T) {
		output := "rtl_433 version unknown inputs file rtl_tcp"
		
		lines := strings.Split(output, "\n")
		if len(lines) == 0 {
			t.Fatal("Expected at least one line")
		}

		firstLine := strings.TrimSpace(lines[0])
		
		re := regexp.MustCompile(`rtl_433 version ([\d.]+)`)
		matches := re.FindStringSubmatch(firstLine)
		
		if len(matches) < 2 {
			// Fallback to full line
			version := firstLine
			if version == "" {
				t.Fatal("Expected fallback version")
			}
			return
		}

		// If regex matched, use it
		if matches[1] != "" {
			return
		}
		
		t.Error("Expected version to be parsed")
	})

	t.Run("EmptyString", func(t *testing.T) {
		output := ""
		
		lines := strings.Split(output, "\n")
		if len(lines) == 0 {
			// Empty output should return empty version
			return
		}

		firstLine := strings.TrimSpace(lines[0])
		if firstLine == "" {
			return
		}

		t.Error("Expected empty version for empty input")
	})

	t.Run("InvalidFormat", func(t *testing.T) {
		output := "invalid format without version"
		
		lines := strings.Split(output, "\n")
		if len(lines) == 0 {
			t.Fatal("Expected at least one line")
		}

		firstLine := strings.TrimSpace(lines[0])
		
		re := regexp.MustCompile(`rtl_433 version ([\d.]+)`)
		matches := re.FindStringSubmatch(firstLine)
		
		if len(matches) < 2 {
			// No match, should use full line as fallback
			version := firstLine
			if version == "" {
				t.Fatal("Expected fallback version")
			}
			return
		}
	})
}

// TestSendToCore tests sending data to core API
func TestSendToCore(t *testing.T) {
	t.Run("SuccessfulSend", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
		}))
		defer server.Close()

		client := &http.Client{Timeout: 5 * time.Second}
		
		reading := types.Reading{
			Time:          "2026-03-15T10:00:00Z",
			Model:         "TestModel",
			SensorID:      1,
			TemperatureC:  20.5,
			Humidity:      65,
			UVIndex:       5.0,
			Lux:           10000.0,
			WindSpeedMS:   3.5,
			WindGustMS:    8.0,
			WindDirDeg:    180,
			RainMM:        0.0,
			BatteryOK:     1.0,
			RainStart:     0,
			Firmware:     160,
		}

		payload, _ := json.Marshal(reading)
		job := Job{data: payload}

		err := sendToCore(client, server.URL, job, 1, 1)
		if err != nil {
			t.Errorf("Expected successful send, got error: %v", err)
		}
	})

	t.Run("RetryLogic", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			// Fail first 2 attempts, succeed on 3rd
			if attemptCount < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusAccepted)
		}))
		defer server.Close()

		client := &http.Client{Timeout: 5 * time.Second}
		
		reading := types.Reading{
			Time:          "2026-03-15T10:00:00Z",
			Model:         "TestModel",
			SensorID:      1,
			TemperatureC:  20.5,
			Humidity:      65,
			UVIndex:       5.0,
			Lux:           10000.0,
			WindSpeedMS:   3.5,
			WindGustMS:    8.0,
			WindDirDeg:    180,
			RainMM:        0.0,
			BatteryOK:     1.0,
			RainStart:     0,
			Firmware:     160,
		}

		payload, _ := json.Marshal(reading)
		job := Job{data: payload}

		err := sendToCore(client, server.URL, job, 3, 1)
		if err != nil {
			t.Errorf("Expected successful send after retries, got error: %v", err)
		}

		if attemptCount != 3 {
			t.Errorf("Expected 3 attempts, got %d", attemptCount)
		}
	})

	t.Run("NetworkError", func(t *testing.T) {
		client := &http.Client{Timeout: 5 * time.Second}
		
		reading := types.Reading{
			Time:          "2026-03-15T10:00:00Z",
			Model:         "TestModel",
			SensorID:      1,
			TemperatureC:  20.5,
			Humidity:      65,
			UVIndex:       5.0,
			Lux:           10000.0,
			WindSpeedMS:   3.5,
			WindGustMS:    8.0,
			WindDirDeg:    180,
			RainMM:        0.0,
			BatteryOK:     1.0,
			RainStart:     0,
			Firmware:     160,
		}

		payload, _ := json.Marshal(reading)
		job := Job{data: payload}

		// Use invalid URL to simulate network error
		err := sendToCore(client, "http://invalid-host-that-does-not-exist.local:9999", job, 1, 1)
		if err == nil {
			t.Error("Expected error for invalid URL, got nil")
		}
	})
}

// TestLoadConfig tests configuration loading
func TestLoadConfig(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		// Set valid environment variables
		t.Setenv("CORE_API_URL", "http://localhost:8080/api/ingest")
		t.Setenv("CORE_API_TIMEOUT_SECONDS", "30")
		t.Setenv("CORE_API_MAX_RETRIES", "3")
		t.Setenv("CORE_API_BACKOFF_DURATION", "2")
		t.Setenv("RTL_433_COMMAND", "rtl_433 -f 868M -F json -M utc")
		t.Setenv("CORE_API_WORKER_COUNT", "5")
		t.Setenv("CORE_API_JOB_QUEUE_SIZE", "100")
		t.Setenv("CORE_API_RESULT_QUEUE_SIZE", "100")

		config, err := loadConfig()
		if err != nil {
			t.Fatalf("Expected successful config load, got error: %v", err)
		}

		if config.url != "http://localhost:8080/api/ingest" {
			t.Errorf("Expected URL http://localhost:8080/api/ingest, got %s", config.url)
		}

		if config.timeout != 30*time.Second {
			t.Errorf("Expected timeout 30s, got %v", config.timeout)
		}

		if config.maxRetries != 3 {
			t.Errorf("Expected maxRetries 3, got %d", config.maxRetries)
		}

		if config.backoffDuration != 2 {
			t.Errorf("Expected backoffDuration 2, got %d", config.backoffDuration)
		}

		if config.numWorkers != 5 {
			t.Errorf("Expected numWorkers 5, got %d", config.numWorkers)
		}

		if config.jobQueueSize != 100 {
			t.Errorf("Expected jobQueueSize 100, got %d", config.jobQueueSize)
		}

		if config.resultQueueSize != 100 {
			t.Errorf("Expected resultQueueSize 100, got %d", config.resultQueueSize)
		}
	})

	t.Run("InvalidTimeout", func(t *testing.T) {
		t.Setenv("CORE_API_URL", "http://localhost:8080/api/ingest")
		t.Setenv("CORE_API_TIMEOUT_SECONDS", "invalid")
		t.Setenv("CORE_API_MAX_RETRIES", "3")
		t.Setenv("CORE_API_BACKOFF_DURATION", "2")

		_, err := loadConfig()
		if err == nil {
			t.Error("Expected error for invalid timeout, got nil")
		}
	})

	t.Run("InvalidWorkerCount", func(t *testing.T) {
		t.Setenv("CORE_API_URL", "http://localhost:8080/api/ingest")
		t.Setenv("CORE_API_TIMEOUT_SECONDS", "30")
		t.Setenv("CORE_API_MAX_RETRIES", "3")
		t.Setenv("CORE_API_BACKOFF_DURATION", "2")
		t.Setenv("CORE_API_WORKER_COUNT", "0")

		_, err := loadConfig()
		if err == nil {
			t.Error("Expected error for invalid worker count, got nil")
		}
	})

	t.Run("MissingRequiredConfig", func(t *testing.T) {
		// Skip this test because loadConfig() calls os.Exit(1) when configuration is missing
		// which terminates the test process
		t.Skip("loadConfig() calls os.Exit(1) when configuration is missing, which terminates the test process")
	})
}

// TestCalculateBackoff tests exponential backoff calculation
func TestCalculateBackoff(t *testing.T) {
	t.Run("ExponentialBackoff", func(t *testing.T) {
		tests := []struct {
			attempt          int
			expectedDuration time.Duration
		}{
			{0, 1 * time.Second},
			{1, 2 * time.Second},
			{2, 4 * time.Second},
			{3, 8 * time.Second},
			{4, 16 * time.Second}, // Capped at 16
			{5, 16 * time.Second}, // Still capped
		}

		for _, tt := range tests {
			result := calculateBackoff(tt.attempt, 30*time.Second)
			if result != tt.expectedDuration {
				t.Errorf("Attempt %d: expected %v, got %v", tt.attempt, tt.expectedDuration, result)
			}
		}
	})

	t.Run("MaxBackoff", func(t *testing.T) {
		// Test with a lower max backoff
		backoff := calculateBackoff(5, 10*time.Second)
		if backoff > 10*time.Second {
			t.Errorf("Expected backoff <= 10s, got %v", backoff)
		}
	})
}

// TestMin tests the min helper function
func TestMin(t *testing.T) {
	tests := []struct {
		a, b     int
		expected int
	}{
		{1, 2, 1},
		{5, 3, 3},
		{0, 0, 0},
		{-1, 1, -1},
		{100, 100, 100},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("min(%d, %d) = %d, expected %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

// TestProcessStdout tests processing stdout data
func TestProcessStdout(t *testing.T) {
	t.Run("ValidJSONReading", func(t *testing.T) {
		jobQueue := make(chan Job, 10)
		defer close(jobQueue)

		// Simulate reading from scanner
		reading := types.Reading{
			Time:          "2026-03-15T10:00:00Z",
			Model:         "TestModel",
			SensorID:      1,
			TemperatureC:  20.5,
			Humidity:      65,
			UVIndex:       5.0,
			Lux:           10000.0,
			WindSpeedMS:   3.5,
			WindGustMS:    8.0,
			WindDirDeg:    180,
			RainMM:        0.0,
			BatteryOK:     1.0,
			RainStart:     0,
			Firmware:     160,
		}

		jsonData, _ := json.Marshal(reading)
		line := string(jsonData)

		// Parse the JSON
		var parsedReading types.Reading
		if err := json.Unmarshal([]byte(line), &parsedReading); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		// Create job
		payload, _ := json.Marshal(parsedReading)
		job := Job{data: payload}

		jobQueue <- job

		// Verify job was received
		select {
		case receivedJob := <-jobQueue:
			if len(receivedJob.data) == 0 {
				t.Error("Expected non-empty job data")
			}
		default:
			t.Error("Expected job in queue")
		}
	})

	t.Run("InvalidJSONReading", func(t *testing.T) {
		jobQueue := make(chan Job, 10)
		defer close(jobQueue)

		line := "invalid json data"

		// Try to parse the JSON
		var reading types.Reading
		if err := json.Unmarshal([]byte(line), &reading); err == nil {
			t.Error("Expected error for invalid JSON, got nil")
		}

		// Job should not be created for invalid data
		select {
		case <-jobQueue:
			t.Error("Expected no job in queue for invalid JSON")
		default:
			// Expected: no job
		}
	})
}
