# Build an MVP autonomous software engineering workflow system

You are building a local-first MVP for an autonomous software engineering workflow assistant.

The goal is NOT to build a production-ready platform.

The goal is to build a minimal but functional system where:

* I can submit feature ideas
* An AI planner asks clarifying questions
* The system generates a draft spec
* Approved specs become tasks
* Claude Code can later implement tasks
* Tasks persist on disk
* The system can recursively improve itself later

This project should prioritize:

* simplicity
* readability
* local-first development
* minimal dependencies
* explicit workflows
* durable file-based state

Avoid:

* microservices
* Kubernetes
* event buses
* complicated orchestration frameworks
* unnecessary abstractions
* multi-agent frameworks
* premature scalability

## Tech Stack

Backend:

* Golang
* SQLite
* standard library preferred where possible

Frontend:

* simple HTML/CSS/JS OR Vue
* minimal styling
* functional over beautiful

Other:

* markdown files for specs
* JSON for state
* Telegram integration should be stubbed but structured cleanly
* GitHub integration can be mocked/stubbed initially

## Core Concept

This is NOT a generic project management board.

This is an AI-oriented task orchestration system.

Tasks are durable execution units for AI agents.

Each task should live on disk.

Example:

/tasks
/task-0001
idea.md
clarification.md
spec.md
state.json
logs/
reviews/

## MVP Features

Implement ONLY these features.

### 1. Create Feature Idea

User can submit:

* title
* freeform idea text

Persist to disk.

Generate:

* task folder
* idea.md
* initial state.json

Initial state:
IDEA_SUBMITTED

---

### 2. Planner Clarification Flow

Implement a planner module.

The planner should:

* read idea.md
* generate clarification questions
* save them to clarification.md

DO NOT call external APIs yet.

For now:

* mock planner responses
  OR
* structure code cleanly for future Claude Code integration

Questions should focus on:

* requirements
* edge cases
* scope boundaries
* constraints
* acceptance criteria

---

### 3. Spec Generation

After clarification answers are added:

* generate spec.md

Spec format:

# Goal

# Requirements

# Non-Goals

# Edge Cases

# Acceptance Criteria

Update task state:
SPEC_DRAFTED

---

### 4. Human Approval Flow

User can approve spec.

State becomes:
SPEC_APPROVED

---

### 5. Task Board UI

Simple UI showing tasks grouped by state.

States:

* IDEA_SUBMITTED
* NEEDS_CLARIFICATION
* SPEC_DRAFTED
* SPEC_APPROVED
* IMPLEMENTING
* IN_REVIEW
* READY_TO_MERGE
* DONE

This should be VERY simple.

No drag-and-drop needed.

---

### 6. Claude Executor Stub

Implement a clean abstraction for future Claude Code execution.

Example:

Executor interface:

* PlanTask()
* ImplementTask()
* ReviewTask()

For now:

* mock execution
* write logs
* simulate status changes

DO NOT build full orchestration yet.

---

### 7. Scheduler

Implement a simple scheduler loop.

Every N seconds:

* scan tasks
* print pending work

Do NOT build distributed workers.

Simple local polling only.

---

## Architecture Requirements

Prioritize:

* clean folder structure
* explicit state transitions
* deterministic workflows
* durable persistence
* composable modules

Avoid:

* giant God objects
* hidden magic
* dynamic runtime behavior
* excessive interfaces

Use:

* clear services
* repository pattern where helpful
* simple state machine logic

---

## Important Design Principles

### 1. Tasks are durable

Everything important should persist on disk.

Do NOT rely on memory.

---

### 2. Specs are first-class

The system exists to convert vague ideas into executable specs.

This is more important than autonomous coding.

---

### 3. Human approval is mandatory

No autonomous implementation should happen without explicit approval.

---

### 4. Simplicity over intelligence

Prefer:

* simple workflows
* understandable code
* explicit logic

Over:

* clever abstractions
* autonomous complexity

---

## Deliverables

Build:

* backend
* minimal frontend
* task persistence
* state machine
* planner module
* spec generator
* scheduler
* mock executor

Also generate:

* README.md
* architecture.md
* CLAUDE.md

CLAUDE.md should contain repository rules for future Claude Code agents.

---

## Folder Structure Suggestion

/backend
/frontend
/tasks
/internal
/docs

---

## Final Notes

This system will later be used to improve itself.

Therefore:

* code clarity matters
* architecture clarity matters
* workflows must be understandable by AI agents

Optimize for future agent readability.

Do not overengineer the first draft.

