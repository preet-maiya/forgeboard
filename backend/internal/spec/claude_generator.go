package spec

import (
	"forgeboard/internal/claudecli"
)

const specPromptBase = `You are a senior software architect writing a precise, implementable feature specification.

Produce a spec in this EXACT format:

# Goal

(One clear paragraph describing what this feature does and why.)

# Requirements

(Bulleted list of concrete, testable requirements.)

# Non-Goals

(Bulleted list of things explicitly NOT included in this implementation.)

# Edge Cases

(Bulleted list of edge cases the implementation must handle, with expected behavior.)

# Acceptance Criteria

(Checkbox list — implementation is "done" when all boxes can be checked.)

---

Rules:
- Be specific and concrete. Vague requirements are useless.
- Every requirement must be testable.
- Non-goals must be explicit to prevent scope creep.
- Acceptance criteria must be verifiable by a human or automated test.

---

`

// ClaudeGenerator uses the claude CLI to generate a spec document.
type ClaudeGenerator struct{}

func NewClaudeGenerator() *ClaudeGenerator {
	return &ClaudeGenerator{}
}

func (g *ClaudeGenerator) GenerateSpec(ideaText, clarificationText string) (string, error) {
	prompt := specPromptBase + "Feature idea:\n\n" + ideaText

	if clarificationText != "" {
		prompt += "\n\nClarification questions and answers:\n\n" + clarificationText
	}

	return claudecli.Run(prompt)
}
