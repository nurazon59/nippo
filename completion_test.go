package main

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompletion(t *testing.T) {
	bin := buildBinary(t)

	tests := []struct {
		shell string
		want  []byte
	}{
		{"bash", []byte("bash completion for nippo")},
		{"zsh", []byte("#compdef nippo")},
		{"fish", []byte("fish shell completion for nippo")},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			cmd := exec.Command(bin, "completion", tt.shell)
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, string(out))
			require.True(t, bytes.Contains(out, tt.want), "expected %q in output, got: %s", tt.want, out)
		})
	}
}

func TestCompletionInvalidShell(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "completion", "powershell")
	out, err := cmd.CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(out), `must be one of "bash","zsh","fish"`)
}
