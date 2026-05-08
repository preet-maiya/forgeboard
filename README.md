# Forgeboard

Local-first autonomous software engineering workflow assistant.

Submit feature ideas → generate clarifications → draft specs → approve → implement.

## Quick Start

```bash
cd backend
go run main.go
```

Open http://localhost:8080

## Workflow

```
IDEA_SUBMITTED
    → Generate Clarifications → NEEDS_CLARIFICATION
    → Generate Spec           → SPEC_DRAFTED
    → Approve Spec            → SPEC_APPROVED
    → Start Implementation    → IMPLEMENTING
    → (auto) Run Review       → IN_REVIEW
    → (auto) Review Done      → READY_TO_MERGE
    → Mark Done               → DONE
```

## Configuration

| Env var       | Default       | Description              |
|---------------|---------------|--------------------------|
| `PORT`        | `8080`        | HTTP listen port         |
| `TASKS_DIR`   | `../tasks`    | Task persistence dir     |
| `FRONTEND_DIR`| `../frontend` | Static files dir         |

## Task Structure

Each task lives on disk:

```
tasks/
  task-0001/
    idea.md           # original idea text
    clarification.md  # planner questions
    spec.md           # generated spec
    state.json        # current state + metadata
    logs/             # executor log files
    reviews/          # review artifacts
```

## API

| Method | Path                          | Description               |
|--------|-------------------------------|---------------------------|
| GET    | `/api/tasks`                  | List all tasks            |
| POST   | `/api/tasks`                  | Create task (idea)        |
| GET    | `/api/tasks/:id`              | Get task                  |
| POST   | `/api/tasks/:id/clarify`      | Generate clarifications   |
| POST   | `/api/tasks/:id/spec`         | Generate spec             |
| POST   | `/api/tasks/:id/approve`      | Approve spec              |
| POST   | `/api/tasks/:id/implement`    | Start implementation      |
| POST   | `/api/tasks/:id/review`       | Run review                |
| POST   | `/api/tasks/:id/done`         | Mark done                 |
| GET    | `/api/tasks/:id/files/:name`  | Read task file            |

## Project Structure

```
backend/          Go backend (main.go + internal packages)
frontend/         Single-file HTML/CSS/JS UI
tasks/            Persisted task folders (git-ignored)
docs/             Architecture and design docs
```

## Replacing Mocks

- **Planner**: implement `planner.Planner` interface in `backend/internal/planner/`
- **Spec Generator**: implement `spec.Generator` interface in `backend/internal/spec/`
- **Executor**: implement `executor.Executor` interface in `backend/internal/executor/`

Each interface has a single mock implementation. Swap for real Claude API calls when ready.
