package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"forgeboard/internal/api"
	"forgeboard/internal/claudecli"
	"forgeboard/internal/executor"
	"forgeboard/internal/planner"
	"forgeboard/internal/scheduler"
	"forgeboard/internal/spec"
	"forgeboard/internal/task"
)

func main() {
	tasksDir := envOrDefault("TASKS_DIR", "../tasks")
	port := envOrDefault("PORT", "8080")
	frontendDir := envOrDefault("FRONTEND_DIR", "../frontend")

	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		log.Fatalf("failed to create tasks dir: %v", err)
	}

	if !claudecli.Available() {
		log.Fatal("claude CLI not found on PATH — install Claude Code and ensure `claude` is available")
	}
	log.Printf("claude: CLI found")

	absTasksDir, err := filepath.Abs(tasksDir)
	if err != nil {
		log.Fatalf("failed to resolve tasks dir: %v", err)
	}
	projectRoot := filepath.Dir(absTasksDir)

	repo := task.NewRepository(tasksDir)
	p := planner.NewClaudePlanner()
	sg := spec.NewClaudeGenerator()
	ex := executor.NewClaudeCodeExecutor(projectRoot)

	h := api.NewHandler(repo, p, sg, ex)

	sched := scheduler.New(repo, 30*time.Second)
	sched.Start()
	defer sched.Stop()

	mux := http.NewServeMux()
	mux.Handle("/api/", h)
	mux.Handle("/", http.FileServer(http.Dir(frontendDir)))

	log.Printf("forgeboard: listening on :%s", port)
	log.Printf("forgeboard: tasks dir = %s", tasksDir)
	log.Printf("forgeboard: frontend dir = %s", frontendDir)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
