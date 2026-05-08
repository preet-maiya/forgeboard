package planner

import (
	"forgeboard/internal/claudecli"
)

const plannerPrompt = `You are a senior software architect acting as a requirements planner.

Read the feature idea below and generate a structured set of clarification questions
that will help produce a precise, implementable spec.

Focus your questions on:
- Requirements and must-haves vs nice-to-haves
- Scope boundaries and explicit non-goals
- Edge cases and failure modes
- Constraints (performance, security, compatibility)
- Acceptance criteria — how "done" is defined

Output ONLY a markdown document with this structure:

# Clarification Questions

## Requirements
(numbered questions)

## Scope Boundaries
(numbered questions)

## Edge Cases
(numbered questions)

## Constraints
(numbered questions)

## Acceptance Criteria
(numbered questions)

---

*Add answers below each question, then trigger spec generation.*

Be specific to the idea. Do not ask generic questions that do not apply.

---

Feature idea:

`

// ClaudePlanner uses the claude CLI to generate clarification questions.
type ClaudePlanner struct{}

func NewClaudePlanner() *ClaudePlanner {
	return &ClaudePlanner{}
}

func (p *ClaudePlanner) GenerateClarifications(ideaText string) (string, error) {
	return claudecli.Run(plannerPrompt + ideaText)
}
