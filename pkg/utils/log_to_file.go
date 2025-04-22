package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LogToFile writes log messages to a file with timestamp
func LogToFile(filePath, message string, toBePrinted bool) error {
	// Create directory if it doesn't exist
	if filePath == "" {
		return nil
	}
	if toBePrinted {
		fmt.Println(message)
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Open log file in append mode
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	// Write timestamped message
	logEntry := fmt.Sprintf("[%s] %s\n", time.Now().Format("15:04:05"), message)
	if _, err := f.WriteString(logEntry); err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	return nil
}
