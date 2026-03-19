package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// QwenProvider implements the Provider interface for Qwen.
// Ideally this would use an SDK or HTTP client, but for now we might wrap a CLI or just stub it.
// To support "yolo mode", we assume the agent tool itself handles the loop.
type QwenProvider struct {
	Config Config
}

func NewQwenProvider(cfg Config) *QwenProvider {
	return &QwenProvider{Config: cfg}
}

func (p *QwenProvider) Name() string {
	return "qwen"
}

func (p *QwenProvider) Run(ctx context.Context, prompt string, workdir string, env []string) (string, error) {
	// In a real implementation, this might call a python script or a binary
	// that implements the agentic loop (Reason -> Act -> Observe).
	// For this prototype, we'll simulate it or assume a 'qwen-agent' CLI exists.

	// Example: echo prompt to a file and run a command
	// cmd := exec.CommandContext(ctx, "qwen-agent", "--prompt", prompt, "--yolo")

	// Since we don't have the actual CLI, we'll implement a mock behavior
	// that writes a file to simulate "coding".

	fmt.Printf("[Qwen] Executing in %s with model %s\n", workdir, p.Config.Model)

	// Create a dummy file to prove it ran
	f, err := os.Create(workdir + "/agent_output.txt")
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = f.WriteString("Agent executed with prompt:\n" + prompt)
	if err != nil {
		return "", err
	}

	return "Agent execution completed successfully (simulated).", nil
}

// KimiProvider implements the Provider interface for Kimi CLI.
// Uses the 'kimi' command line tool with -y (yolo) and -p (prompt) flags.
type KimiProvider struct {
	Config Config
}

func NewKimiProvider(cfg Config) *KimiProvider {
	return &KimiProvider{Config: cfg}
}

func (p *KimiProvider) Name() string {
	return "kimi"
}

func (p *KimiProvider) Run(ctx context.Context, prompt string, workdir string, env []string) (string, error) {
	fmt.Printf("[Kimi] Executing in %s\n", workdir)

	// Execute kimi with -y (yolo/auto-approve) and -p (prompt) flags
	cmd := exec.CommandContext(ctx, "kimi", "-y", "-p", prompt)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(), env...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("kimi execution failed: %w", err)
	}

	return string(output), nil
}

// Factory to create providers
func NewProvider(cfg Config) (Provider, error) {
	switch cfg.Provider {
	case "qwen":
		return NewQwenProvider(cfg), nil
	case "kimi":
		return NewKimiProvider(cfg), nil
	case "trae":
		// return NewTraeProvider(cfg), nil
		return nil, fmt.Errorf("trae provider not yet implemented")
	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
	}
}
