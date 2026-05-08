package executor

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"forgeboard/internal/claudecli"
)

// ClaudeCodeExecutor invokes the claude CLI to perform planning, implementation, and review.
// All operations run with the task directory as context via explicit file references in the prompt.
type ClaudeCodeExecutor struct{}

func NewClaudeCodeExecutor() *ClaudeCodeExecutor {
	return &ClaudeCodeExecutor{}
}

func (e *ClaudeCodeExecutor) PlanTask(taskID, taskDir string) error {
	specContent, err := os.ReadFile(filepath.Join(taskDir, "spec.md"))
	if err != nil {
		return fmt.Errorf("read spec.md: %w", err)
	}

	prompt := `You are an AI software engineer. Read the spec below and produce a detailed implementation plan.

The plan should include:
- Files to create or modify
- Key functions/types to implement
- Order of implementation steps
- Potential risks or unknowns

Output ONLY the plan in markdown. Do not write any code.

---

` + string(specContent)

	output, err := claudecli.Run(prompt)
	if err != nil {
		return e.writeLog(taskDir, "plan", fmt.Sprintf("ERROR: %v", err))
	}
	if err := os.WriteFile(filepath.Join(taskDir, "plan.md"), []byte(output), 0644); err != nil {
		return fmt.Errorf("write plan.md: %w", err)
	}
	return e.writeLog(taskDir, "plan", output)
}

func (e *ClaudeCodeExecutor) ImplementTask(taskID, taskDir string) error {
	specContent, err := os.ReadFile(filepath.Join(taskDir, "spec.md"))
	if err != nil {
		return fmt.Errorf("read spec.md: %w", err)
	}

	planContent := ""
	if raw, err := os.ReadFile(filepath.Join(taskDir, "plan.md")); err == nil {
		planContent = "\n\nImplementation plan:\n\n" + string(raw)
	}

	prompt := `You are an AI software engineer. Implement the feature described in the spec below.

Rules:
- Write clean, readable code
- Handle all edge cases listed in the spec
- Do not add features beyond what the spec requires
- After implementing, write a brief summary of what you did

Spec:

` + string(specContent) + planContent

	output, err := claudecli.Run(prompt)
	if err != nil {
		return e.writeLog(taskDir, "implement", fmt.Sprintf("ERROR: %v", err))
	}

	summary := "# Implementation Summary\n\n" + output
	summaryPath := filepath.Join(taskDir, "logs", "implementation-summary.md")
	_ = os.WriteFile(summaryPath, []byte(summary), 0644)

	return e.writeLog(taskDir, "implement", output)
}

func (e *ClaudeCodeExecutor) ReviewTask(taskID, taskDir string) error {
	specContent, err := os.ReadFile(filepath.Join(taskDir, "spec.md"))
	if err != nil {
		return fmt.Errorf("read spec.md: %w", err)
	}

	prompt := `You are a senior engineer performing a code review.

Review the implementation against the spec's acceptance criteria below.

Output your review in this format:

# Code Review

## Summary
(Overall assessment)

## Acceptance Criteria Check
(Go through each criterion: PASS or FAIL with notes)

## Issues Found
(Bugs, missing edge cases, spec violations — or "None")

## Verdict
APPROVED or CHANGES_REQUESTED

---

Spec:

` + string(specContent)

	output, err := claudecli.Run(prompt)
	if err != nil {
		return e.writeLog(taskDir, "review", fmt.Sprintf("ERROR: %v", err))
	}

	reviewPath := filepath.Join(taskDir, "reviews", "review.md")
	_ = os.MkdirAll(filepath.Join(taskDir, "reviews"), 0755)
	_ = os.WriteFile(reviewPath, []byte(output), 0644)

	approved := strings.Contains(output, "APPROVED")
	log.Printf("executor: review complete for %s, approved=%v", taskID, approved)
	return e.writeLog(taskDir, "review", output)
}

func (e *ClaudeCodeExecutor) writeLog(taskDir, op, content string) error {
	logsDir := filepath.Join(taskDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("create logs dir: %w", err)
	}
	filename := fmt.Sprintf("%s-%s.log", op, time.Now().Format("20060102-150405"))
	path := filepath.Join(logsDir, filename)
	log.Printf("executor: wrote %s", path)
	return os.WriteFile(path, []byte(content), 0644)
}
