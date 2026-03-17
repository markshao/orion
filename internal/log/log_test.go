package log

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit(t *testing.T) {
	// Save original home
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
	}()

	// Create temp dir to act as home
	tmpDir, err := os.MkdirTemp("", "log-test-home-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.Setenv("HOME", tmpDir)

	// Test Init
	err = Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Close()

	// Verify log file was created
	logPath := filepath.Join(tmpDir, ".orion.log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("expected log file to be created")
	}
}

func TestInitWithInvalidHome(t *testing.T) {
	// Save original home
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
	}()

	// Set invalid home
	os.Setenv("HOME", "/nonexistent/path/that/should/not/exist")

	// Test Init should fail
	err := Init()
	if err == nil {
		t.Error("expected Init to fail with invalid home directory")
	}
}

func TestError(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "log-test-error-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
	}()

	if err := Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Close()

	// Test Error logging
	Error("test error: %s", "something went wrong")

	// Read log file and verify
	logPath := filepath.Join(tmpDir, ".orion.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	logContent := string(data)
	if !strings.Contains(logContent, "ERROR") {
		t.Error("expected log to contain ERROR level")
	}
	if !strings.Contains(logContent, "test error: something went wrong") {
		t.Error("expected log to contain error message")
	}
}

func TestInfo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log-test-info-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
	}()

	if err := Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Close()

	// Test Info logging
	Info("test info: %s", "initializing something")

	// Read log file and verify
	logPath := filepath.Join(tmpDir, ".orion.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	logContent := string(data)
	if !strings.Contains(logContent, "INFO") {
		t.Error("expected log to contain INFO level")
	}
	if !strings.Contains(logContent, "test info: initializing something") {
		t.Error("expected log to contain info message")
	}
}

func TestLogWithoutInit(t *testing.T) {
	// Ensure logFile is nil by not calling Init
	// These should not panic or write anywhere
	Error("this should be ignored")
	Info("this should also be ignored")
	Close() // Should be safe to call when not initialized
}

func TestMultipleLogCalls(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log-test-multi-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
	}()

	if err := Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Close()

	// Log multiple messages
	Error("error 1")
	Info("info 1")
	Error("error 2")
	Info("info 2")

	// Read and verify all messages are present
	logPath := filepath.Join(tmpDir, ".orion.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	logContent := string(data)
	expectedMessages := []string{"error 1", "info 1", "error 2", "info 2"}
	for _, msg := range expectedMessages {
		if !strings.Contains(logContent, msg) {
			t.Errorf("expected log to contain %q", msg)
		}
	}
}

func TestLogTimestamp(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log-test-timestamp-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
	}()

	if err := Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Close()

	Info("timestamp test")

	logPath := filepath.Join(tmpDir, ".orion.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	logContent := string(data)
	// Check for timestamp format (RFC3339 contains year)
	if !strings.Contains(logContent, "20") {
		t.Error("expected log to contain timestamp")
	}
}

func TestClose(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log-test-close-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
	}()

	if err := Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	Info("before close")
	Close()

	// After Close, logging should be silent (no panic)
	Info("after close - should be ignored")
	Error("after close - should also be ignored")

	// Calling Close again should be safe
	Close()
}
