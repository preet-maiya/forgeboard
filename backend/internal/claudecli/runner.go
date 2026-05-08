package claudecli

import (
	"fmt"
	"os/exec"
	"strings"
)

// Run executes `claude -p` with the given prompt piped via stdin.
// Returns the full stdout response as a string.
func Run(prompt string) (string, error) {
	cmd := exec.Command("claude", "-p")
	cmd.Stdin = strings.NewReader(prompt)

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("claude cli: %w\nstderr: %s", err, exitErr.Stderr)
		}
		return "", fmt.Errorf("claude cli: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Available returns true if the claude binary is on PATH.
func Available() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}
