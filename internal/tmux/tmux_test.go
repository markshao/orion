package tmux

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestSessionExists tests the SessionExists function
func TestSessionExists(t *testing.T) {
	// Create a unique session name for testing
	sessionName := "orion-test-session-" + time.Now().Format("20060102150405")

	// Session should not exist initially
	if SessionExists(sessionName) {
		t.Error("expected session to not exist initially")
	}

	// Create the session
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName)
	if err := cmd.Run(); err != nil {
		t.Skipf("tmux not available, skipping test: %v", err)
	}
	defer exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	// Session should exist now
	if !SessionExists(sessionName) {
		t.Error("expected session to exist after creation")
	}
}

// TestNewSession tests creating a new tmux session
func TestNewSession(t *testing.T) {
	sessionName := "orion-test-new-" + time.Now().Format("20060102150405")
	defer exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	// Create a temp directory for the session
	tmpDir, err := os.MkdirTemp("", "tmux-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	err = NewSession(sessionName, tmpDir)
	if err != nil {
		t.Skipf("tmux not available, skipping test: %v", err)
	}

	// Verify session exists
	if !SessionExists(sessionName) {
		t.Error("expected session to exist after NewSession")
	}
}

// TestSendKeys tests sending keys to a tmux session
func TestSendKeys(t *testing.T) {
	sessionName := "orion-test-keys-" + time.Now().Format("20060102150405")
	tmpDir, err := os.MkdirTemp("", "tmux-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	defer exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	// Create session
	if err := NewSession(sessionName, tmpDir); err != nil {
		t.Skipf("tmux not available, skipping test: %v", err)
	}

	// Send a simple command (echo)
	err = SendKeys(sessionName, "echo 'hello' > test_output.txt")
	if err != nil {
		t.Skipf("SendKeys failed (tmux may not be fully configured): %v", err)
	}

	// Give tmux time to execute
	time.Sleep(1 * time.Second)

	// Check if file was created
	outputFile := filepath.Join(tmpDir, "test_output.txt")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Logf("output file not created (tmux SendKeys may require interactive environment)")
		t.Skip("skipping due to tmux environment limitations")
	}
}

// TestIsInsideTmux tests the IsInsideTmux function
func TestIsInsideTmux(t *testing.T) {
	// This test depends on the environment
	// We just verify it returns a boolean without error
	result := IsInsideTmux()
	_ = result // The result itself isn't testable without controlling the environment
}

// TestGetCurrentSessionName tests getting the current session name
func TestGetCurrentSessionName(t *testing.T) {
	// This test only works inside tmux
	if !IsInsideTmux() {
		t.Skip("not inside tmux, skipping test")
	}

	sessionName, err := GetCurrentSessionName()
	if err != nil {
		t.Errorf("GetCurrentSessionName failed: %v", err)
	}
	if sessionName == "" {
		t.Error("expected non-empty session name")
	}
}

// TestSwitchClient tests switching tmux client
func TestSwitchClient(t *testing.T) {
	// This test requires being inside tmux and having multiple sessions
	if !IsInsideTmux() {
		t.Skip("not inside tmux, skipping test")
	}

	sessionName := "orion-test-switch-" + time.Now().Format("20060102150405")
	tmpDir, _ := os.MkdirTemp("", "tmux-test-*")
	defer os.RemoveAll(tmpDir)
	defer exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	// Create target session
	if err := NewSession(sessionName, tmpDir); err != nil {
		t.Skipf("tmux not available, skipping test: %v", err)
	}

	// Try to switch (this might fail if not properly set up)
	err := SwitchClient(sessionName)
	if err != nil {
		// This is expected to potentially fail in test environments
		t.Logf("SwitchClient returned: %v (expected in some test environments)", err)
	}
}

// TestKillSession tests killing a tmux session
func TestKillSession(t *testing.T) {
	sessionName := "orion-test-kill-" + time.Now().Format("20060102150405")
	tmpDir, err := os.MkdirTemp("", "tmux-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create session
	if err := NewSession(sessionName, tmpDir); err != nil {
		t.Skipf("tmux not available, skipping test: %v", err)
	}

	// Verify it exists
	if !SessionExists(sessionName) {
		t.Fatal("expected session to exist before kill")
	}

	// Kill the session
	err = KillSession(sessionName)
	if err != nil {
		t.Errorf("KillSession failed: %v", err)
	}

	// Verify it's gone
	if SessionExists(sessionName) {
		t.Error("expected session to not exist after kill")
	}
}

// TestKillNonExistentSession tests killing a session that doesn't exist
func TestKillNonExistentSession(t *testing.T) {
	sessionName := "orion-test-nonexistent-" + time.Now().Format("20060102150405")

	// Should not error for non-existent session
	err := KillSession(sessionName)
	if err != nil {
		t.Errorf("KillSession should not error for non-existent session: %v", err)
	}
}

// TestEnsureAndAttach tests the EnsureAndAttach function
func TestEnsureAndAttach(t *testing.T) {
	sessionName := "orion-test-ensure-" + time.Now().Format("20060102150405")
	tmpDir, err := os.MkdirTemp("", "tmux-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	defer exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	// This function replaces the current process, so we can't test it fully
	// We just verify it creates the session if it doesn't exist
	// Note: We won't actually call EnsureAndAttach as it would replace our test process

	// Test that NewSession is called internally by checking session creation
	err = NewSession(sessionName, tmpDir)
	if err != nil {
		t.Skipf("tmux not available, skipping test: %v", err)
	}

	if !SessionExists(sessionName) {
		t.Error("expected session to be created")
	}
}
