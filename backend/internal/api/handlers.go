package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"forgeboard/internal/executor"
	"forgeboard/internal/planner"
	specpkg "forgeboard/internal/spec"
	"forgeboard/internal/task"
)

// Handler routes all /api/* requests.
type Handler struct {
	repo     *task.Repository
	planner  planner.Planner
	specGen  specpkg.Generator
	executor executor.Executor
}

func NewHandler(
	repo *task.Repository,
	p planner.Planner,
	sg specpkg.Generator,
	ex executor.Executor,
) *Handler {
	return &Handler{repo: repo, planner: p, specGen: sg, executor: ex}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	case path == "/api/tasks" && r.Method == http.MethodGet:
		h.listTasks(w, r)
	case path == "/api/tasks" && r.Method == http.MethodPost:
		h.createTask(w, r)
	case strings.HasPrefix(path, "/api/tasks/"):
		h.routeTask(w, r, path)
	default:
		jsonError(w, "not found", http.StatusNotFound)
	}
}

func (h *Handler) routeTask(w http.ResponseWriter, r *http.Request, path string) {
	// path: /api/tasks/{id}[/{action}[/{filename}]]
	rest := strings.TrimPrefix(path, "/api/tasks/")
	parts := strings.SplitN(rest, "/", 3)
	id := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch {
	case action == "" && r.Method == http.MethodGet:
		h.getTask(w, r, id)
	case action == "clarify" && r.Method == http.MethodPost:
		h.clarifyTask(w, r, id)
	case action == "spec" && r.Method == http.MethodPost:
		h.generateSpec(w, r, id)
	case action == "approve" && r.Method == http.MethodPost:
		h.approveSpec(w, r, id)
	case action == "implement" && r.Method == http.MethodPost:
		h.implementTask(w, r, id)
	case action == "review" && r.Method == http.MethodPost:
		h.reviewTask(w, r, id)
	case action == "done" && r.Method == http.MethodPost:
		h.markDone(w, r, id)
	case action == "files" && len(parts) == 3:
		h.getFile(w, r, id, parts[2])
	default:
		jsonError(w, "not found", http.StatusNotFound)
	}
}

// POST /api/tasks
type createTaskRequest struct {
	Title string `json:"title"`
	Idea  string `json:"idea"`
}

func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Idea) == "" {
		jsonError(w, "title and idea are required", http.StatusBadRequest)
		return
	}
	t, err := h.repo.Create(req.Title, req.Idea)
	if err != nil {
		jsonError(w, "failed to create task: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, t, http.StatusCreated)
}

// GET /api/tasks
func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.repo.List()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, tasks, http.StatusOK)
}

// GET /api/tasks/:id
func (h *Handler) getTask(w http.ResponseWriter, r *http.Request, id string) {
	t, err := h.repo.Get(id)
	if err != nil {
		jsonError(w, "task not found", http.StatusNotFound)
		return
	}
	jsonOK(w, t, http.StatusOK)
}

// POST /api/tasks/:id/clarify
func (h *Handler) clarifyTask(w http.ResponseWriter, r *http.Request, id string) {
	t, err := h.repo.Get(id)
	if err != nil {
		jsonError(w, "task not found", http.StatusNotFound)
		return
	}
	if t.State != task.StateIdeaSubmitted {
		jsonError(w, "task must be in IDEA_SUBMITTED state", http.StatusBadRequest)
		return
	}
	ideaText, err := h.repo.ReadFile(id, "idea.md")
	if err != nil {
		jsonError(w, "idea.md not found", http.StatusNotFound)
		return
	}
	questions, err := h.planner.GenerateClarifications(ideaText)
	if err != nil {
		jsonError(w, "planner error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.repo.WriteFile(id, "clarification.md", questions); err != nil {
		jsonError(w, "failed to write clarification.md", http.StatusInternalServerError)
		return
	}
	if err := h.repo.UpdateState(id, task.StateNeedsClarification); err != nil {
		jsonError(w, "failed to update state", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "clarification generated", "task_id": id}, http.StatusOK)
}

// POST /api/tasks/:id/spec
func (h *Handler) generateSpec(w http.ResponseWriter, r *http.Request, id string) {
	t, err := h.repo.Get(id)
	if err != nil {
		jsonError(w, "task not found", http.StatusNotFound)
		return
	}
	if t.State != task.StateNeedsClarification {
		jsonError(w, "task must be in NEEDS_CLARIFICATION state", http.StatusBadRequest)
		return
	}
	ideaText, err := h.repo.ReadFile(id, "idea.md")
	if err != nil {
		jsonError(w, "idea.md not found", http.StatusNotFound)
		return
	}
	clarText, _ := h.repo.ReadFile(id, "clarification.md") // ok if missing

	specContent, err := h.specGen.GenerateSpec(ideaText, clarText)
	if err != nil {
		jsonError(w, "spec generation error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.repo.WriteFile(id, "spec.md", specContent); err != nil {
		jsonError(w, "failed to write spec.md", http.StatusInternalServerError)
		return
	}
	if err := h.repo.UpdateState(id, task.StateSpecDrafted); err != nil {
		jsonError(w, "failed to update state", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "spec generated", "task_id": id}, http.StatusOK)
}

// POST /api/tasks/:id/approve
func (h *Handler) approveSpec(w http.ResponseWriter, r *http.Request, id string) {
	t, err := h.repo.Get(id)
	if err != nil {
		jsonError(w, "task not found", http.StatusNotFound)
		return
	}
	if t.State != task.StateSpecDrafted {
		jsonError(w, "task must be in SPEC_DRAFTED state to approve", http.StatusBadRequest)
		return
	}
	if err := h.repo.UpdateState(id, task.StateSpecApproved); err != nil {
		jsonError(w, "failed to update state", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "spec approved", "task_id": id}, http.StatusOK)
}

// POST /api/tasks/:id/implement
func (h *Handler) implementTask(w http.ResponseWriter, r *http.Request, id string) {
	t, err := h.repo.Get(id)
	if err != nil {
		jsonError(w, "task not found", http.StatusNotFound)
		return
	}
	if t.State != task.StateSpecApproved {
		jsonError(w, "task must be in SPEC_APPROVED state", http.StatusBadRequest)
		return
	}
	if err := h.repo.UpdateState(id, task.StateImplementing); err != nil {
		jsonError(w, "failed to update state", http.StatusInternalServerError)
		return
	}
	taskDir := h.repo.TaskDir(id)
	go func() {
		if err := h.executor.ImplementTask(id, taskDir); err != nil {
			log.Printf("implement failed for %s: %v — rolling back to SPEC_APPROVED", id, err)
			_ = h.repo.UpdateState(id, task.StateSpecApproved)
			return
		}
		_ = h.repo.UpdateState(id, task.StateInReview)
	}()
	jsonOK(w, map[string]string{"status": "implementation started", "task_id": id}, http.StatusOK)
}

// POST /api/tasks/:id/review
func (h *Handler) reviewTask(w http.ResponseWriter, r *http.Request, id string) {
	t, err := h.repo.Get(id)
	if err != nil {
		jsonError(w, "task not found", http.StatusNotFound)
		return
	}
	if t.State != task.StateInReview {
		jsonError(w, "task must be in IN_REVIEW state", http.StatusBadRequest)
		return
	}
	taskDir := h.repo.TaskDir(id)
	go func() {
		approved, err := h.executor.ReviewTask(id, taskDir)
		if err != nil {
			log.Printf("review failed for %s: %v — rolling back to IMPLEMENTING", id, err)
			_ = h.repo.UpdateState(id, task.StateImplementing)
			return
		}
		if approved {
			_ = h.repo.UpdateState(id, task.StateReadyToMerge)
		} else {
			log.Printf("review: CHANGES_REQUESTED for %s — rolling back to IMPLEMENTING", id)
			_ = h.repo.UpdateState(id, task.StateImplementing)
		}
	}()
	jsonOK(w, map[string]string{"status": "review started", "task_id": id}, http.StatusOK)
}

// POST /api/tasks/:id/done
func (h *Handler) markDone(w http.ResponseWriter, r *http.Request, id string) {
	t, err := h.repo.Get(id)
	if err != nil {
		jsonError(w, "task not found", http.StatusNotFound)
		return
	}
	if t.State != task.StateReadyToMerge {
		jsonError(w, "task must be in READY_TO_MERGE state", http.StatusBadRequest)
		return
	}
	if err := h.repo.UpdateState(id, task.StateDone); err != nil {
		jsonError(w, "failed to update state", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "done", "task_id": id}, http.StatusOK)
}

// GET /api/tasks/:id/files/:filename
func (h *Handler) getFile(w http.ResponseWriter, r *http.Request, id, filename string) {
	allowed := map[string]bool{
		"idea.md":          true,
		"clarification.md": true,
		"spec.md":          true,
		"state.json":       true,
	}
	if !allowed[filename] {
		jsonError(w, "file not accessible", http.StatusForbidden)
		return
	}
	content, err := h.repo.ReadFile(id, filename)
	if err != nil {
		jsonError(w, "file not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(content))
}

func jsonOK(w http.ResponseWriter, v interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
