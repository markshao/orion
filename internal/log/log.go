package log

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var logFile *os.File

// Init initializes the global logger to ~/.devswarm.log
func Init() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	logPath := filepath.Join(home, ".devswarm.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	logFile = f
	return nil
}

// Error logs an error message to the log file (if initialized)
func Error(format string, v ...interface{}) {
	if logFile == nil {
		return
	}
	msg := fmt.Sprintf(format, v...)
	timestamp := time.Now().Format(time.RFC3339)
	fmt.Fprintf(logFile, "[%s] ERROR: %s\n", timestamp, msg)
}

// Info logs an info message
func Info(format string, v ...interface{}) {
	if logFile == nil {
		return
	}
	msg := fmt.Sprintf(format, v...)
	timestamp := time.Now().Format(time.RFC3339)
	fmt.Fprintf(logFile, "[%s] INFO: %s\n", timestamp, msg)
}

// Close closes the log file
func Close() {
	if logFile != nil {
		logFile.Close()
	}
}
