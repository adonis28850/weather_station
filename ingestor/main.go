package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"weather-station/shared/logger"
	"weather-station/shared/types"
	"weather-station/shared/validation"
	"weather-station/shared/workers"
)

const (
	// Maximum backoff delay capped at 16 seconds
	maxBackoffSeconds = 16
)

// Job represents a single HTTP request job to be processed by the worker pool
type Job struct {
	data []byte
}

// Config holds the configuration for the ingestor
type Config struct {
	url             string
	timeout         time.Duration
	maxRetries      int
	backoffDuration int
	rtl433Command   string
	numWorkers      int
	jobQueueSize    int
	resultQueueSize int
}

// Helper function to return the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helper function to calculate exponential backoff delay
func calculateBackoff(attempt int, maxBackoff time.Duration) time.Duration {
	backoffMultiplier := 1 << uint(min(attempt, 4))
	backoff := time.Duration(backoffMultiplier) * time.Second
	if backoff > maxBackoff {
		backoff = maxBackoff
	}
	if backoff > maxBackoffSeconds*time.Second {
		backoff = maxBackoffSeconds * time.Second
	}
	return backoff
}

// loadConfig loads configuration from environment variables
func loadConfig() (Config, error) {
	URL := os.Getenv("CORE_API_URL")

	timeoutSeconds, err := strconv.Atoi(os.Getenv("CORE_API_TIMEOUT_SECONDS"))
	if err != nil {
		return Config{}, fmt.Errorf("Invalid CORE_API_TIMEOUT_SECONDS: %w", err)
	}
	timeout := time.Duration(timeoutSeconds) * time.Second

	maxRetries, err := strconv.Atoi(os.Getenv("CORE_API_MAX_RETRIES"))
	if err != nil {
		return Config{}, fmt.Errorf("Invalid CORE_API_MAX_RETRIES: %w", err)
	}

	backoffDuration, err := strconv.Atoi(os.Getenv("CORE_API_BACKOFF_DURATION"))
	if err != nil {
		return Config{}, fmt.Errorf("Invalid CORE_API_BACKOFF_DURATION: %w", err)
	}

	rtl433Command := os.Getenv("RTL_433_COMMAND")
	if rtl433Command == "" {
		rtl433Command = "rtl_433 -f 868M -F json -M utc"
	}

	numWorkers := 5
	if numWorkersStr := os.Getenv("CORE_API_WORKER_COUNT"); numWorkersStr != "" {
		numWorkers, err = strconv.Atoi(numWorkersStr)
		if err != nil {
			return Config{}, fmt.Errorf("Invalid CORE_API_WORKER_COUNT: %w", err)
		}
		if numWorkers <= 0 {
			return Config{}, fmt.Errorf("CORE_API_WORKER_COUNT must be positive, got: %d", numWorkers)
		}
	}

	jobQueueSize := 100
	if jobQueueSizeStr := os.Getenv("CORE_API_JOB_QUEUE_SIZE"); jobQueueSizeStr != "" {
		jobQueueSize, err = strconv.Atoi(jobQueueSizeStr)
		if err != nil {
			return Config{}, fmt.Errorf("Invalid CORE_API_JOB_QUEUE_SIZE: %w", err)
		}
		if jobQueueSize <= 0 {
			return Config{}, fmt.Errorf("CORE_API_JOB_QUEUE_SIZE must be positive, got: %d", jobQueueSize)
		}
	}

	resultQueueSize := 100
	if resultQueueSizeStr := os.Getenv("CORE_API_RESULT_QUEUE_SIZE"); resultQueueSizeStr != "" {
		resultQueueSize, err = strconv.Atoi(resultQueueSizeStr)
		if err != nil {
			return Config{}, fmt.Errorf("Invalid CORE_API_RESULT_QUEUE_SIZE: %w", err)
		}
		if resultQueueSize <= 0 {
			return Config{}, fmt.Errorf("CORE_API_RESULT_QUEUE_SIZE must be positive, got: %d", resultQueueSize)
		}
	}

	config := Config{
		url:             URL,
		timeout:         timeout,
		maxRetries:      maxRetries,
		backoffDuration: backoffDuration,
		rtl433Command:   rtl433Command,
		numWorkers:      numWorkers,
		jobQueueSize:    jobQueueSize,
		resultQueueSize: resultQueueSize,
	}

	if config.url == "" || config.timeout == 0 || config.maxRetries == 0 || config.backoffDuration == 0 {
		logger.Error("Missing required configuration. Please set CORE_API_URL, CORE_API_TIMEOUT_SECONDS, CORE_API_MAX_RETRIES, and CORE_API_BACKOFF_DURATION")
		os.Exit(1)
	}

	return config, nil
}

// spawnRTL433 spawns the rtl_433 subprocess
func spawnRTL433(command string) (*exec.Cmd, io.ReadCloser, io.ReadCloser, error) {
	// Split the command string into command and arguments
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil, nil, nil, fmt.Errorf("empty command")
	}

	rtl_433 := exec.Command(parts[0], parts[1:]...)

	stdoutPipe, err := rtl_433.StdoutPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := rtl_433.StderrPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := rtl_433.Start(); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to start rtl_433 subprocess: %w", err)
	}

	return rtl_433, stdoutPipe, stderrPipe, nil
}

// sendToCore sends data to Core API with retry logic
func sendToCore(client *http.Client, url string, job Job, maxRetries int, backoffDuration int) error {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err := client.Post(url, "application/json", bytes.NewBuffer(job.data))
		if err != nil {
			logger.Error("Failed to send data to Core API: %v", err)
			backoffMultiplier := 1 << uint(min(attempt-1, 4))
			backoffSeconds := backoffDuration * backoffMultiplier
			if backoffSeconds > maxBackoffSeconds {
				backoffSeconds = maxBackoffSeconds
			}
			logger.Info("Retrying in %d seconds... (attempt %d/%d)", backoffSeconds, attempt, maxRetries)
			time.Sleep(time.Duration(backoffSeconds) * time.Second)
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			logger.Info("Successfully sent data to Core API: %s", resp.Status)
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			return nil
		}

		logger.Error("Core API returned error status: %s (attempt %d/%d)", resp.Status, attempt, maxRetries)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		backoffMultiplier := 1 << uint(min(attempt-1, 4))
		backoffSeconds := backoffDuration * backoffMultiplier
		if backoffSeconds > maxBackoffSeconds {
			backoffSeconds = maxBackoffSeconds
		}
		logger.Info("Retrying in %d seconds...", backoffSeconds)
		time.Sleep(time.Duration(backoffSeconds) * time.Second)
	}

	return fmt.Errorf("failed to send data to Core API after %d attempts", maxRetries)
}

// processStdout reads and processes weather data from rtl_433 stdout
func processStdout(scanner *bufio.Scanner, jobQueue chan<- Job) {
	for scanner.Scan() {
		line := scanner.Text()

		var reading types.Reading
		if err := json.Unmarshal([]byte(line), &reading); err != nil {
			logger.Error("Failed to parse JSON line: %v", err)
			continue
		}

		if err := validation.ValidateReading(reading); err != nil {
			logger.Error("Invalid reading: %v. Skipping.", err)
			continue
		}

		payload, err := json.Marshal(reading)
		if err != nil {
			logger.Error("Failed to marshal reading to JSON: %v", err)
			continue
		}

		jobQueue <- Job{data: payload}
	}
}

// main is the entry point for the ingestor application
func main() {
	config, err := loadConfig()
	if err != nil {
		logger.Error("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	client := &http.Client{
		Timeout: config.timeout,
	}

	jobFunc := func(job Job) error {
		return sendToCore(client, config.url, job, config.maxRetries, config.backoffDuration)
	}

	for restartAttempt := 0; ; restartAttempt++ {
		rtl_433, stdoutPipe, stderrPipe, err := spawnRTL433(config.rtl433Command)
		if err != nil {
			logger.Error("Failed to spawn rtl_433 subprocess: %v", err)
			delay := calculateBackoff(restartAttempt, 30*time.Second)
			logger.Info("Retrying in %v...", delay)
			time.Sleep(delay)
			continue
		}

		go func() {
			scanner := bufio.NewScanner(stderrPipe)
			for scanner.Scan() {
				logger.Error("[rtl_433 stderr] %s", scanner.Text())
			}
			if err := scanner.Err(); err != nil {
				logger.Error("Error reading stderr from rtl_433: %v", err)
			}
		}()

		scanner := bufio.NewScanner(stdoutPipe)

		workerPool := workers.StartWorkerPool(jobFunc, config.numWorkers, config.jobQueueSize, config.resultQueueSize)
		defer workerPool.Shutdown()

		go func() {
			for err := range workerPool.GetResultChan() {
				if err != nil {
					logger.Error("Failed to send data to Core API after retries: %v", err)
				}
			}
		}()

		processStdout(scanner, workerPool.GetJobChan())

		if err := scanner.Err(); err != nil && err != io.EOF {
			logger.Error("Error reading stdout: %v", err)
		}

		if err := rtl_433.Wait(); err != nil {
			logger.Error("rtl_433 subprocess exited with error: %v", err)
			delay := calculateBackoff(restartAttempt, 30*time.Second)
			logger.Info("Restarting rtl_433 subprocess in %v... (attempt %d)", delay, restartAttempt+1)
			time.Sleep(delay)
		} else {
			logger.Info("rtl_433 subprocess exited normally")
			break
		}
	}
}
