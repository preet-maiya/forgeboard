package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"forgeboard/internal/claudecli"
	"forgeboard/internal/executor"
	ghclient "forgeboard/internal/github"
	"forgeboard/internal/planner"
	"forgeboard/internal/review"
	specpkg "forgeboard/internal/spec"
	"forgeboard/internal/task"
)

// Handler routes all /api/* requests.
type Handler struct {
	repo        *task.Repository
	planner     planner.Planner
	specGen     specpkg.Generator
	executor    executor.Executor
	github      *ghclient.Client // nil if GitHub not configured
	reviewQueue *review.Queue
}

func NewHandler(
	repo *task.Repository,
	p planner.Planner,
	sg specpkg.Generator,
	ex executor.Executor,
	gh *ghclient.Client,
	rq *review.Queue,
) *Handler {
	return &Handler{
		repo:        repo,
		planner:     p,
		specGen:     sg,
		executor:    ex,
		github:      gh,
		reviewQueue: rq,
	}
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
	case action == "ready-for-review" && r.Method == http.MethodPost:
		h.readyForReview(w, r, id)
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

	// Async: create GitHub branch (one-time, skips if already exists)
	if h.github != nil {
		go h.createBranchWithRetry(id)
	}

	taskDir := h.repo.TaskDir(id)
	go func() {
		if err := h.executor.ImplementTask(id, taskDir); err != nil {
			log.Printf("implement failed for %s: %v — rolling back to SPEC_APPROVED", id, err)
			_ = h.repo.UpdateState(id, task.StateSpecApproved)
			return
		}
		_ = h.repo.UpdateState(id, task.StateInReview)
		// Async: push files + open PR
		if h.github != nil {
			go h.pushAndCreatePRWithRetry(id)
		}
	}()
	jsonOK(w, map[string]string{"status": "implementation started", "task_id": id}, http.StatusOK)
}

// POST /api/tasks/:id/review — local Claude review (non-GitHub path)
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

// POST /api/tasks/:id/ready-for-review — triggers GitHub review agent
func (h *Handler) readyForReview(w http.ResponseWriter, r *http.Request, id string) {
	t, err := h.repo.Get(id)
	if err != nil {
		jsonError(w, "task not found", http.StatusNotFound)
		return
	}
	if t.State != task.StateInReview {
		jsonError(w, "task must be in IN_REVIEW state", http.StatusBadRequest)
		return
	}
	if h.github == nil {
		jsonError(w, "GitHub not configured", http.StatusBadRequest)
		return
	}

	if runNow := h.reviewQueue.Enqueue(id); runNow {
		go h.runReviewAgent(id)
	}
	jsonOK(w, map[string]string{"status": "review agent queued", "task_id": id}, http.StatusOK)
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

// ========== GitHub async helpers ==========

// createBranchWithRetry creates the task branch (task/{id}) off main, skipping if it exists.
func (h *Handler) createBranchWithRetry(id string) {
	branch := "task/" + id
	var lastErr error
	delay := time.Second
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			time.Sleep(delay)
			delay *= 2
		}
		exists, err := h.github.BranchExists(branch)
		if err != nil {
			if ghclient.IsRateLimit(err) {
				log.Printf("createBranch: rate limit on attempt %d for %s", attempt+1, id)
				lastErr = err
				continue
			}
			log.Printf("createBranch: check exists error for %s: %v", id, err)
			lastErr = err
			continue
		}
		if exists {
			log.Printf("createBranch: branch %s already exists for %s, skipping", branch, id)
			return
		}
		if err := h.github.CreateBranch(branch, "main"); err != nil {
			if ghclient.IsRateLimit(err) {
				lastErr = err
				continue
			}
			log.Printf("createBranch: attempt %d failed for %s: %v", attempt+1, id, err)
			lastErr = err
			continue
		}
		log.Printf("createBranch: created %s for %s", branch, id)
		return
	}
	log.Printf("createBranch: 5 consecutive failures for %s: %v", id, lastErr)
}

// pushAndCreatePRWithRetry pushes all task files to the task branch and opens/reuses a PR.
// On 5 consecutive failures, transitions the task to ERROR state.
func (h *Handler) pushAndCreatePRWithRetry(id string) {
	branch := "task/" + id
	taskDir := h.repo.TaskDir(id)
	repoRelDir := "tasks/" + id

	var lastErr error
	delay := time.Second
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			time.Sleep(delay)
			delay *= 2
		}

		files, err := ghclient.CollectTaskFiles(taskDir, repoRelDir)
		if err != nil {
			log.Printf("pushAndPR: collect files error for %s: %v", id, err)
			lastErr = err
			continue
		}

		if err := h.github.PushFiles(branch, files); err != nil {
			if ghclient.IsRateLimit(err) {
				log.Printf("pushAndPR: rate limit attempt %d for %s", attempt+1, id)
				lastErr = err
				continue
			}
			log.Printf("pushAndPR: push attempt %d failed for %s: %v", attempt+1, id, err)
			lastErr = err
			continue
		}

		// Find or create PR
		prNum, err := h.github.FindOpenPR(branch)
		if err != nil {
			if ghclient.IsRateLimit(err) {
				lastErr = err
				continue
			}
			log.Printf("pushAndPR: findOpenPR error for %s: %v", id, err)
			lastErr = err
			continue
		}

		if prNum == 0 {
			tsk, _ := h.repo.Get(id)
			title := fmt.Sprintf("[%s] %s", id, tsk.Title)
			specBody, _ := h.repo.ReadFile(id, "spec.md")
			prNum, err = h.github.CreatePR(title, specBody, branch, "main")
			if err != nil {
				if ghclient.IsRateLimit(err) {
					lastErr = err
					continue
				}
				log.Printf("pushAndPR: createPR attempt %d failed for %s: %v", attempt+1, id, err)
				lastErr = err
				continue
			}
		}

		// Success — persist PR number
		tsk, _ := h.repo.Get(id)
		if tsk != nil {
			tsk.PRNumber = prNum
			tsk.PushError = ""
			tsk.Error = ""
			_ = h.repo.UpdateTask(tsk)
		}
		log.Printf("pushAndPR: success for %s, PR #%d", id, prNum)
		return
	}

	// 5 consecutive failures → ERROR state
	tsk, _ := h.repo.Get(id)
	if tsk != nil {
		tsk.State = task.StateError
		tsk.Error = fmt.Sprintf("push failed after 5 attempts: %v", lastErr)
		tsk.PushError = lastErr.Error()
		_ = h.repo.UpdateTask(tsk)
	}
	log.Printf("pushAndPR: 5 consecutive failures for %s → ERROR: %v", id, lastErr)
}

// ========== Review agent ==========

// runReviewAgent runs the review agent loop for taskID.
// It loops as long as the queue signals there is a next run pending.
func (h *Handler) runReviewAgent(id string) {
	for {
		h.doReviewAgentRun(id)
		if !h.reviewQueue.Complete(id) {
			return
		}
	}
}

// doReviewAgentRun runs a single review agent attempt with up to 5 retries.
// On 5 failures, writes review_error to state.json and returns.
func (h *Handler) doReviewAgentRun(id string) {
	var lastErr error
	delay := time.Second
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			time.Sleep(delay)
			delay *= 2
		}
		if err := h.doSingleReviewAttempt(id); err != nil {
			log.Printf("reviewAgent: attempt %d failed for %s: %v", attempt+1, id, err)
			lastErr = err
			continue
		}
		return
	}
	// All 5 failed
	tsk, _ := h.repo.Get(id)
	if tsk != nil {
		tsk.ReviewError = fmt.Sprintf("review agent failed after 5 attempts: %v", lastErr)
		_ = h.repo.UpdateTask(tsk)
	}
	log.Printf("reviewAgent: 5 consecutive failures for %s: %v", id, lastErr)
}

// doSingleReviewAttempt runs Claude against the PR diff + spec + state, then posts a GitHub review.
func (h *Handler) doSingleReviewAttempt(id string) error {
	tsk, err := h.repo.Get(id)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}
	if tsk.PRNumber == 0 {
		return fmt.Errorf("no PR number recorded for task %s — push may still be in progress", id)
	}

	diff, err := h.github.GetPRDiff(tsk.PRNumber)
	if err != nil {
		return fmt.Errorf("get PR diff: %w", err)
	}

	specContent, _ := h.repo.ReadFile(id, "spec.md")
	stateContent, _ := h.repo.ReadFile(id, "state.json")

	prompt := `You are a senior software engineer reviewing a GitHub pull request on behalf of an automated review system.

Review the PR diff against the spec. Evaluate:
1. Code quality (correctness, readability, error handling)
2. Spec compliance — do all acceptance criteria appear to be met?

Respond in EXACTLY this format (no extra text before or after):

## Verdict
APPROVED

or

## Verdict
CHANGES_REQUESTED

## Summary
(2-3 sentence overall assessment)

## Issues
(Numbered list of specific issues, or "None found")

---

Spec:
` + specContent + `

State:
` + stateContent + `

PR Diff:
` + diff

	output, err := claudecli.Run(prompt)
	if err != nil {
		return fmt.Errorf("claude cli: %w", err)
	}

	approved := strings.Contains(output, "## Verdict\nAPPROVED") ||
		strings.Contains(output, "## Verdict\r\nAPPROVED")

	var event, body string
	if approved {
		event = "APPROVE"
		body = "LGTM\n\n" + output
	} else {
		event = "REQUEST_CHANGES"
		body = output
	}

	if err := h.github.PostPRReview(tsk.PRNumber, event, body); err != nil {
		return fmt.Errorf("post PR review: %w", err)
	}

	log.Printf("reviewAgent: posted %s review for %s PR #%d", event, id, tsk.PRNumber)
	return nil
}

// ========== Helpers ==========

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
