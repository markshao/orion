package log

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInitAndLogging verifies Init, Info, Error and Close work together
// and that log lines are appended to the expected file.
func TestInitAndLogging(t *testing.T) {
	// Use a temporary HOME so we don't touch the real user's home directory.
	tmpHome, err := os.MkdirTemp("", "orion-log-test")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	if err := os.Setenv("HOME", tmpHome); err != nil {
		t.Fatalf("failed to set HOME: %v", err)
	}

	if err := Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	defer Close()

	Info("info message: %s", "hello")
	Error("error message: %s", "boom")

	logPath := filepath.Join(tmpHome, ".orion.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "INFO: info message: hello") {
		t.Errorf("log file missing info entry, got: %s", content)
	}
	if !strings.Contains(content, "ERROR: error message: boom") {
		t.Errorf("log file missing error entry, got: %s", content)
	}
}

