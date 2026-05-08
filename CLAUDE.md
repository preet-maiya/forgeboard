# Forgeboard — Agent Rules

Rules for Claude Code agents working in this repository.

## Project Purpose

Forgeboard converts vague feature ideas into approved, executable specs via a structured human-in-the-loop workflow. It is NOT a generic project management tool.

## Core Principles

1. **Tasks are durable.** All state lives on disk in `tasks/`. Never rely on in-memory state.
2. **Specs are first-class.** The primary deliverable is `spec.md`, not code.
3. **Human approval is mandatory.** No task transitions from SPEC_DRAFTED to SPEC_APPROVED without an explicit human action.
4. **Simplicity over intelligence.** Prefer explicit, readable code over clever abstractions.

## State Machine

Transitions must follow this sequence — no skipping:

```
IDEA_SUBMITTED → NEEDS_CLARIFICATION → SPEC_DRAFTED → SPEC_APPROVED
→ IMPLEMENTING → IN_REVIEW → READY_TO_MERGE → DONE
```

Transition logic lives in `backend/internal/task/task.go` (`ValidTransitions` map).

## Repository Layout

```
backend/
  main.go                      entry point, wires dependencies
  internal/
    task/task.go               Task struct + State constants + transition map
    task/repository.go         disk-based persistence (read/write/list)
    planner/planner.go         Planner interface + MockPlanner
    spec/generator.go          Generator interface + MockGenerator
    executor/executor.go       Executor interface + MockExecutor
    scheduler/scheduler.go     polling loop (30s interval)
    api/handlers.go            HTTP handlers for all /api/* routes
frontend/
  index.html                   single-file UI (no build step)
tasks/                         runtime task folders (do NOT commit)
docs/
  architecture.md              system design overview
```

## Rules for Agents

### Reading state
- Always read `state.json` via the repository, never parse it manually.
- Check `task.CanTransition(current, next)` before any state change.

### Writing files
- Write task artifacts only through `Repository.WriteFile()`.
- Do not write to `tasks/` directly from outside the `task` package.

### Adding features
- Planner, spec generator, executor: implement the interface, do not modify existing mock.
- New API endpoints: add to `api/handlers.go`, route in `routeTask()`.
- New states: add to `task.go` constants AND `ValidTransitions` map.

### Dependencies
- Standard library only. No external packages unless absolutely necessary.
- No frameworks, ORMs, or event buses.

### Testing
- Test state transitions and repository logic directly.
- Use table-driven tests.
- Run `go vet ./...` and `go build ./...` before declaring done.

## Do Not

- Do not add authentication or multi-user logic.
- Do not build a distributed system.
- Do not add databases — file-based persistence only.
- Do not auto-approve specs. Human approval is always required.
- Do not skip state transitions.
- Do not commit the `tasks/` directory.
