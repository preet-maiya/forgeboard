package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Repository handles disk-based persistence for tasks.
type Repository struct {
	tasksDir string
}

func NewRepository(tasksDir string) *Repository {
	return &Repository{tasksDir: tasksDir}
}

// Create writes a new task to disk and returns it.
func (r *Repository) Create(title, ideaText string) (*Task, error) {
	id, err := r.nextID()
	if err != nil {
		return nil, fmt.Errorf("generate id: %w", err)
	}

	taskDir := filepath.Join(r.tasksDir, id)
	for _, sub := range []string{"logs", "reviews"} {
		if err := os.MkdirAll(filepath.Join(taskDir, sub), 0755); err != nil {
			return nil, fmt.Errorf("create %s dir: %w", sub, err)
		}
	}

	ideaContent := fmt.Sprintf("# %s\n\n%s\n", title, ideaText)
	if err := os.WriteFile(filepath.Join(taskDir, "idea.md"), []byte(ideaContent), 0644); err != nil {
		return nil, fmt.Errorf("write idea.md: %w", err)
	}

	t := &Task{
		ID:        id,
		Title:     title,
		State:     StateIdeaSubmitted,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := r.saveState(t); err != nil {
		return nil, err
	}
	return t, nil
}

// Get loads a task by ID from disk.
func (r *Repository) Get(id string) (*Task, error) {
	data, err := os.ReadFile(filepath.Join(r.tasksDir, id, "state.json"))
	if err != nil {
		return nil, fmt.Errorf("read state.json: %w", err)
	}
	var t Task
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parse state.json: %w", err)
	}
	return &t, nil
}

// List returns all tasks sorted by creation time.
func (r *Repository) List() ([]*Task, error) {
	entries, err := os.ReadDir(r.tasksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Task{}, nil
		}
		return nil, err
	}

	var tasks []*Task
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "task-") {
			continue
		}
		t, err := r.Get(e.Name())
		if err != nil {
			continue // skip corrupted tasks
		}
		tasks = append(tasks, t)
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})
	return tasks, nil
}

// UpdateState transitions a task to a new state.
func (r *Repository) UpdateState(id string, next State) error {
	t, err := r.Get(id)
	if err != nil {
		return err
	}
	t.State = next
	t.UpdatedAt = time.Now()
	return r.saveState(t)
}

// TaskDir returns the directory path for a given task.
func (r *Repository) TaskDir(id string) string {
	return filepath.Join(r.tasksDir, id)
}

// ReadFile reads a named file from a task directory.
func (r *Repository) ReadFile(id, filename string) (string, error) {
	data, err := os.ReadFile(filepath.Join(r.tasksDir, id, filename))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteFile writes content to a named file in a task directory.
func (r *Repository) WriteFile(id, filename, content string) error {
	return os.WriteFile(filepath.Join(r.tasksDir, id, filename), []byte(content), 0644)
}

func (r *Repository) saveState(t *Task) error {
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(r.tasksDir, t.ID, "state.json"), data, 0644)
}

func (r *Repository) nextID() (string, error) {
	entries, err := os.ReadDir(r.tasksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "task-0001", nil
		}
		return "", err
	}
	max := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		var n int
		fmt.Sscanf(e.Name(), "task-%04d", &n)
		if n > max {
			max = n
		}
	}
	return fmt.Sprintf("task-%04d", max+1), nil
}
