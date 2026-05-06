package cmd

import "testing"

func TestAICmdSilencesUsageOnRuntimeError(t *testing.T) {
	if !aiCmd.SilenceUsage {
		t.Fatal("expected aiCmd.SilenceUsage to be true")
	}
}
