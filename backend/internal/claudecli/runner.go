package claudecli

import (
	"fmt"
	"os/exec"
	"strings"
)

// Run executes `claude -p` with the given prompt piped via stdin.
// Returns the full stdout response as a string.
func Run(prompt string) (string, error) {
	return RunInDir(prompt, "", nil)
}

// RunInDir executes `claude -p` with optional working directory and allowed tools.
// tools is a list of tool names to allow (e.g. ["Edit","Write","Read","Bash"]).
// If dir is empty, the current working directory is used.
func RunInDir(prompt, dir string, tools []string) (string, error) {
	args := []string{"-p"}
	if len(tools) > 0 {
		args = append(args, "--allowedTools", strings.Join(tools, ","))
	}
	cmd := exec.Command("claude", args...)
	cmd.Stdin = strings.NewReader(prompt)
	if dir != "" {
		cmd.Dir = dir
	}

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
