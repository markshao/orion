package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// SessionExists checks if a tmux session exists.
func SessionExists(sessionName string) bool {
	return exec.Command("tmux", "has-session", "-t", sessionName).Run() == nil
}

// NewSession creates a new detached tmux session.
func NewSession(sessionName, cwd string) error {
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", cwd)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create tmux session: %s: %w", string(output), err)
	}
	return nil
}

// SendKeys sends keys to a tmux session.
func SendKeys(sessionName, keys string) error {
	cmd := exec.Command("tmux", "send-keys", "-t", sessionName, keys, "C-m")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to send keys to session %s: %s: %w", sessionName, string(output), err)
	}
	return nil
}

// AttachSession replaces the current process with tmux attach.
// WARNING: This function does not return if successful!
func AttachSession(sessionName string) error {
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux not found: %w", err)
	}

	// Exec replaces the current process
	args := []string{"tmux", "attach", "-t", sessionName}
	if err := syscall.Exec(tmuxPath, args, os.Environ()); err != nil {
		return fmt.Errorf("failed to attach to session: %w", err)
	}
	
	return nil // Should not reach here
}

// EnsureSession ensures a session exists (creating it if needed) and then attaches to it.
func EnsureAndAttach(sessionName, cwd string) error {
	if !SessionExists(sessionName) {
		fmt.Printf("Creating new tmux session '%s'...\n", sessionName)
		if err := NewSession(sessionName, cwd); err != nil {
			return err
		}
	} else {
		fmt.Printf("Attaching to existing session '%s'...\n", sessionName)
	}

	return AttachSession(sessionName)
}

// IsInsideTmux checks if the current process is running inside tmux.
func IsInsideTmux() bool {
	return os.Getenv("TMUX") != ""
}

// GetCurrentSessionName returns the name of the current tmux session.
func GetCurrentSessionName() (string, error) {
	if !IsInsideTmux() {
		return "", fmt.Errorf("not inside tmux")
	}
	cmd := exec.Command("tmux", "display-message", "-p", "#S")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// SwitchClient switches the current tmux client to another session.
func SwitchClient(sessionName string) error {
	cmd := exec.Command("tmux", "switch-client", "-t", sessionName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// KillSession kills a tmux session.
func KillSession(sessionName string) error {
	// Ignore error if session doesn't exist
	if !SessionExists(sessionName) {
		return nil
	}
	
	cmd := exec.Command("tmux", "kill-session", "-t", sessionName)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Sometimes has-session returns true but kill fails if race condition, so double check
		if strings.Contains(string(output), "no server running on") {
			return nil
		}
		return fmt.Errorf("failed to kill session: %s: %w", string(output), err)
	}
	return nil
}
