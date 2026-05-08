package executor

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Executor is the abstraction for AI-driven task execution.
// PlanTask, ImplementTask, and ReviewTask will be wired to Claude Code once ready.
type Executor interface {
	PlanTask(taskID, taskDir string) error
	ImplementTask(taskID, taskDir string) error
	ReviewTask(taskID, taskDir string) error
}

// MockExecutor simulates execution by writing log files. No external calls are made.
type MockExecutor struct{}

func NewMockExecutor() *MockExecutor {
	return &MockExecutor{}
}

func (e *MockExecutor) PlanTask(taskID, taskDir string) error {
	msg := fmt.Sprintf("[MOCK] PlanTask: %s\nReading spec.md...\nBuilding execution plan...\nPlan complete.\n", taskID)
	return e.writeLog(taskDir, "plan", msg)
}

func (e *MockExecutor) ImplementTask(taskID, taskDir string) error {
	msg := fmt.Sprintf("[MOCK] ImplementTask: %s\nReading spec.md...\nGenerating code...\nWriting files...\nImplementation complete.\n", taskID)
	return e.writeLog(taskDir, "implement", msg)
}

func (e *MockExecutor) ReviewTask(taskID, taskDir string) error {
	msg := fmt.Sprintf("[MOCK] ReviewTask: %s\nReading implementation...\nChecking against acceptance criteria...\nNo issues found. Ready to merge.\n", taskID)
	return e.writeLog(taskDir, "review", msg)
}

func (e *MockExecutor) writeLog(taskDir, op, content string) error {
	logsDir := filepath.Join(taskDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("create logs dir: %w", err)
	}
	filename := fmt.Sprintf("%s-%s.log", op, time.Now().Format("20060102-150405"))
	path := filepath.Join(logsDir, filename)
	log.Printf("executor: %s -> %s", op, path)
	return os.WriteFile(path, []byte(content), 0644)
}
