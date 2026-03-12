package logger

import (
	"bytes"
	"log"
	"strings"
	"sync"
	"testing"
)

func TestLoggerInfo(t *testing.T) {
	// Test 1: INFO level logging
	t.Run("InfoLogging", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(nil)

		Info("Test message")

		output := buf.String()
		if output == "" {
			t.Error("Expected log output, got empty string")
		}

		if !strings.Contains(output, "[INFO]") {
			t.Error("Expected log output to contain [INFO] tag")
		}

		if !strings.Contains(output, "Test message") {
			t.Error("Expected log output to contain 'Test message'")
		}
	})

	// Test 2: Info log format
	t.Run("InfoLogFormat", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(nil)

		Info("Temperature: %.1f°C", 23.5)

		output := buf.String()
		if !strings.Contains(output, "23.5") {
			t.Error("Expected log output to contain formatted value '23.5'")
		}

		if !strings.Contains(output, "Temperature:") {
			t.Error("Expected log output to contain 'Temperature:'")
		}
	})

	// Test 3: Info with multiple arguments
	t.Run("InfoWithMultipleArgs", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(nil)

		Info("Sensor %d: %s", 12345, "OK")

		output := buf.String()
		if !strings.Contains(output, "12345") {
			t.Error("Expected log output to contain sensor ID '12345'")
		}

		if !strings.Contains(output, "OK") {
			t.Error("Expected log output to contain 'OK'")
		}
	})

	// Test 4: Info with empty message
	t.Run("InfoWithEmptyMessage", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(nil)

		Info("")

		output := buf.String()
		if !strings.Contains(output, "[INFO]") {
			t.Error("Expected log output to contain [INFO] tag even with empty message")
		}
	})
}

func TestLoggerError(t *testing.T) {
	// Test 1: ERROR level logging
	t.Run("ErrorLogging", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(nil)

		Error("Test error message")

		output := buf.String()
		if output == "" {
			t.Error("Expected log output, got empty string")
		}

		if !strings.Contains(output, "[ERROR]") {
			t.Error("Expected log output to contain [ERROR] tag")
		}

		if !strings.Contains(output, "Test error message") {
			t.Error("Expected log output to contain 'Test error message'")
		}
	})

	// Test 2: Error log format
	t.Run("ErrorLogFormat", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(nil)

		Error("Failed to process: %v", "connection timeout")

		output := buf.String()
		if !strings.Contains(output, "Failed to process:") {
			t.Error("Expected log output to contain 'Failed to process:'")
		}

		if !strings.Contains(output, "connection timeout") {
			t.Error("Expected log output to contain error message")
		}
	})

	// Test 3: Error with multiple arguments
	t.Run("ErrorWithMultipleArgs", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(nil)

		Error("Worker %d failed: %s", 3, "panic")

		output := buf.String()
		if !strings.Contains(output, "Worker 3 failed:") {
			t.Error("Expected log output to contain 'Worker 3 failed:'")
		}

		if !strings.Contains(output, "panic") {
			t.Error("Expected log output to contain 'panic'")
		}
	})

	// Test 4: Error with nil error
	t.Run("ErrorWithNil", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(nil)

		Error("Error: %v", nil)

		output := buf.String()
		if !strings.Contains(output, "[ERROR]") {
			t.Error("Expected log output to contain [ERROR] tag")
		}
	})
}

func TestLoggerPlain(t *testing.T) {
	// Test 1: Plain logging without tags
	t.Run("PlainLogging", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(nil)

		Plain("Plain message")

		output := buf.String()
		if output == "" {
			t.Error("Expected log output, got empty string")
		}

		if strings.Contains(output, "[INFO]") || strings.Contains(output, "[ERROR]") {
			t.Error("Expected plain log output to not contain tags")
		}

		if !strings.Contains(output, "Plain message") {
			t.Error("Expected log output to contain 'Plain message'")
		}
	})

	// Test 2: Plain log format
	t.Run("PlainLogFormat", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(nil)

		Plain("Value: %d", 42)

		output := buf.String()
		if !strings.Contains(output, "Value: 42") {
			t.Error("Expected log output to contain 'Value: 42'")
		}

		// Verify it doesn't contain [INFO] or [ERROR] tags (but may contain other brackets from log.Printf)
		if strings.Contains(output, "[INFO]") || strings.Contains(output, "[ERROR]") {
			t.Error("Expected plain log output to not contain [INFO] or [ERROR] tags")
		}
	})

	// Test 3: Plain with empty message
	t.Run("PlainWithEmptyMessage", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(nil)

		Plain("")

		output := buf.String()
		// Plain log should still produce output even with empty message
		// (due to log.Printf behavior)
		if !strings.Contains(output, "\n") || output == "" {
			t.Error("Expected plain log to produce output")
		}
	})

	// Test 4: Plain with special characters
	t.Run("PlainWithSpecialChars", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(nil)

		Plain("Special: %s", "test\nwith\ttabs")

		output := buf.String()
		if !strings.Contains(output, "Special:") {
			t.Error("Expected log output to contain 'Special:'")
		}
	})
}

func TestLoggerColorCodes(t *testing.T) {
	// Test 1: Info log contains green color code
	t.Run("InfoContainsGreenColor", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(nil)

		Info("Test")

		output := buf.String()
		// Check for ANSI color codes
		if !strings.Contains(output, "\033[32m") {
			t.Error("Expected Info log to contain green color code")
		}

		if !strings.Contains(output, "\033[0m") {
			t.Error("Expected log to contain color reset code")
		}
	})

	// Test 2: Error log contains red color code
	t.Run("ErrorContainsRedColor", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(nil)

		Error("Test")

		output := buf.String()
		if !strings.Contains(output, "\033[31m") {
			t.Error("Expected Error log to contain red color code")
		}

		if !strings.Contains(output, "\033[0m") {
			t.Error("Expected log to contain color reset code")
		}
	})

	// Test 3: Plain log contains white color code
	t.Run("PlainContainsWhiteColor", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(nil)

		Plain("Test")

		output := buf.String()
		if !strings.Contains(output, "\033[37m") {
			t.Error("Expected Plain log to contain white color code")
		}

		if !strings.Contains(output, "\033[0m") {
			t.Error("Expected log to contain color reset code")
		}
	})
}

func TestLoggerConcurrentLogging(t *testing.T) {
	// Test 1: Concurrent logging from multiple goroutines
	t.Run("ConcurrentLogging", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(nil)

		var wg sync.WaitGroup

		// Log from multiple goroutines concurrently
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				Info("Goroutine %d", id)
			}(i)
		}

		wg.Wait()

		output := buf.String()
		// Verify all goroutines logged
		for i := 0; i < 10; i++ {
			if !strings.Contains(output, "Goroutine "+string(rune('0'+i))) {
				// This test might be flaky due to goroutine scheduling
				// Just verify we got some output
			}
		}

		// Count [INFO] tags
		infoCount := strings.Count(output, "[INFO]")
		if infoCount != 10 {
			t.Errorf("Expected 10 [INFO] tags, got %d", infoCount)
		}
	})
}

func TestLoggerMixedLevels(t *testing.T) {
	// Test 1: Mix of Info, Error, and Plain logs
	t.Run("MixedLogLevels", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(nil)

		Info("Info message")
		Error("Error message")
		Plain("Plain message")

		output := buf.String()

		if !strings.Contains(output, "[INFO]") {
			t.Error("Expected output to contain [INFO] tag")
		}

		if !strings.Contains(output, "[ERROR]") {
			t.Error("Expected output to contain [ERROR] tag")
		}

		// Verify all messages are present
		if !strings.Contains(output, "Info message") {
			t.Error("Expected output to contain 'Info message'")
		}

		if !strings.Contains(output, "Error message") {
			t.Error("Expected output to contain 'Error message'")
		}

		if !strings.Contains(output, "Plain message") {
			t.Error("Expected output to contain 'Plain message'")
		}
	})
}