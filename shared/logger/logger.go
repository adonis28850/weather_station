package logger

import (
	"fmt"
	"log"
)

// ANSI color codes for colored logging
const (
	colorReset = "\033[0m"
	colorRed   = "\033[31m"
	colorGreen = "\033[32m"
	colorWhite = "\033[37m"
)

// Info logs an informational message with green color
func Info(format string, v ...interface{}) {
	log.Printf("%s[INFO]%s %s", colorGreen, colorReset, fmt.Sprintf(format, v...))
}

// Error logs an error message with red color
func Error(format string, v ...interface{}) {
	log.Printf("%s[ERROR]%s %s", colorRed, colorReset, fmt.Sprintf(format, v...))
}

// Plain logs a message without any color or tag
func Plain(format string, v ...interface{}) {
	log.Printf("%s%s%s", colorWhite, fmt.Sprintf(format, v...), colorReset)
}