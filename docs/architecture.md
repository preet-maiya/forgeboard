# Architecture

## Overview

Forgeboard is a local-first, file-based task orchestration system. A single Go process serves the API and frontend. All state is persisted to disk. No database, no message queue, no distributed components.

## System Diagram

```
Browser (frontend/index.html)
    │ HTTP (fetch)
    ▼
Go HTTP Server (backend/main.go)
    │
    ├── /api/*  →  api.Handler
    │               ├── task.Repository  (disk I/O)
    │               ├── planner.Planner  (mock → Claude API)
    │               ├── spec.Generator   (mock → Claude API)
    │               └── executor.Executor (mock → Claude Code)
    │
    └── /*      →  http.FileServer (frontend/)

Background:
    scheduler.Scheduler  (goroutine, 30s poll)
        └── task.Repository.List()
```

## Key Components

### task.Repository

Single source of truth for task state. All reads and writes go through this struct.

- `Create()` — generates task ID, writes `idea.md` + `state.json`, creates `logs/` and `reviews/` subdirs.
- `Get()` — reads and parses `state.json`.
- `List()` — scans task dirs, returns sorted slice.
- `UpdateState()` — reads current state, sets new state, writes `state.json`.
- `ReadFile()` / `WriteFile()` — access named files within a task dir.

### State Machine

Defined in `task/task.go`. `ValidTransitions` is an explicit map — no dynamic logic.

```
IDEA_SUBMITTED → NEEDS_CLARIFICATION → SPEC_DRAFTED → SPEC_APPROVED
→ IMPLEMENTING → IN_REVIEW → READY_TO_MERGE → DONE
```

### Planner

Interface: `GenerateClarifications(ideaText string) (string, error)`

Current: `MockPlanner` returns a structured markdown template.
Future: Call Claude API with the idea text, stream questions back.

### Spec Generator

Interface: `GenerateSpec(ideaText, clarificationText string) (string, error)`

Current: `MockGenerator` returns a structured spec template.
Future: Call Claude API with both idea and answered clarifications.

### Executor

Interface: `PlanTask`, `ImplementTask`, `ReviewTask` — all take `(taskID, taskDir string)`.

Current: `MockExecutor` writes log files to `tasks/{id}/logs/`.
Future: Invoke Claude Code with the spec as context.

### Scheduler

Simple goroutine with a `time.Ticker`. Every 30 seconds:
1. Lists all tasks.
2. Logs tasks in actionable states (IDEA_SUBMITTED, NEEDS_CLARIFICATION, SPEC_DRAFTED, SPEC_APPROVED).

No worker pool. No distributed coordination. Extend by adding action logic inside `scan()`.

### API Handler

All routes under `/api/`. Minimal router using `strings.HasPrefix`. No framework.

State transition guards are enforced in each handler — callers get a 400 error if they attempt an invalid transition.

## Data Model

### state.json

```json
{
  "id": "task-0001",
  "title": "Add user authentication",
  "state": "SPEC_DRAFTED",
  "created_at": "2026-05-07T10:00:00Z",
  "updated_at": "2026-05-07T10:05:00Z"
}
```

### Task Directory

```
tasks/task-0001/
  idea.md           written at creation
  clarification.md  written by planner
  spec.md           written by spec generator
  state.json        updated on every state transition
  logs/             executor log files (timestamped)
  reviews/          review artifacts (future)
```

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| File-based state | Durable, inspectable, git-friendly, no DB dependency |
| Single Go binary | Simple to run, no container or orchestrator needed |
| No external deps | Reduces operational complexity, easier for AI agents to understand |
| Interface-based mocks | Clean swap point for real Claude API integration |
| Human approval gate | Prevents uncontrolled autonomous changes |
| Explicit state machine | Makes valid transitions auditable and testable |

## Extension Points

1. **Real planner**: implement `planner.Planner`, wire in `main.go`.
2. **Real spec generator**: implement `spec.Generator`, wire in `main.go`.
3. **Real executor**: implement `executor.Executor` using Claude Code SDK.
4. **Telegram notifications**: add a notifier called from handlers after state transitions.
5. **GitHub integration**: stub `github` package, call from executor on READY_TO_MERGE.
