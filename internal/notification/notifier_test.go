package notification

import (
	"os/exec"
	"strings"
	"testing"
)

func TestBuildNotificationCommandPrefersTerminalNotifier(t *testing.T) {
	origLookPath := lookPath
	defer func() { lookPath = origLookPath }()

	lookPath = func(file string) (string, error) {
		switch file {
		case "terminal-notifier":
			return "/opt/homebrew/bin/terminal-notifier", nil
		case "osascript":
			return "/usr/bin/osascript", nil
		default:
			return "", exec.ErrNotFound
		}
	}

	cmd, err := buildNotificationCommand("agent-notify-dev", "readme refactor", "Waiting for input")
	if err != nil {
		t.Fatalf("buildNotificationCommand returned error: %v", err)
	}
	if got := cmd.Path; !strings.Contains(got, "terminal-notifier") {
		t.Fatalf("expected terminal-notifier path, got %s", got)
	}
}

func TestBuildNotificationCommandFallsBackToOsaScript(t *testing.T) {
	origLookPath := lookPath
	defer func() { lookPath = origLookPath }()

	lookPath = func(file string) (string, error) {
		if file == "osascript" {
			return "/usr/bin/osascript", nil
		}
		return "", exec.ErrNotFound
	}

	cmd, err := buildNotificationCommand("agent-notify-dev", "readme refactor", "Waiting for input")
	if err != nil {
		t.Fatalf("buildNotificationCommand returned error: %v", err)
	}
	if got := cmd.Path; !strings.Contains(got, "osascript") {
		t.Fatalf("expected osascript path, got %s", got)
	}
}

func TestBuildNotificationCommandIncludesLabelInSubtitle(t *testing.T) {
	origLookPath := lookPath
	defer func() { lookPath = origLookPath }()

	lookPath = func(file string) (string, error) {
		if file == "osascript" {
			return "/usr/bin/osascript", nil
		}
		return "", exec.ErrNotFound
	}

	cmd, err := buildNotificationCommand("agent-notify-dev", "readme refactor", "Waiting for input")
	if err != nil {
		t.Fatalf("buildNotificationCommand returned error: %v", err)
	}
	args := strings.Join(cmd.Args, " ")
	if !strings.Contains(args, "Node agent-notify-dev (readme refactor)") {
		t.Fatalf("expected subtitle to include node name and label, got %q", args)
	}
}
