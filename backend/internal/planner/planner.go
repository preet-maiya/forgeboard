package planner

import (
	"fmt"
	"strings"
)

// Planner generates clarification questions for a submitted idea.
type Planner interface {
	GenerateClarifications(ideaText string) (string, error)
}

// MockPlanner returns structured clarification questions without calling any external API.
// Replace with a real LLM-backed planner once Claude Code integration is ready.
type MockPlanner struct{}

func NewMockPlanner() *MockPlanner {
	return &MockPlanner{}
}

func (p *MockPlanner) GenerateClarifications(ideaText string) (string, error) {
	title := extractTitle(ideaText)

	return fmt.Sprintf(`# Clarification Questions

The planner reviewed your idea about **%s** and needs answers before generating a spec.

## Requirements

1. Who is the primary user of this feature? Describe the persona.
2. What are the must-have capabilities vs. nice-to-have?
3. Does this need to integrate with any existing systems or APIs?

## Scope Boundaries

4. What is explicitly OUT of scope for this initial version?
5. Is this a one-time operation or a recurring/scheduled one?

## Edge Cases

6. What should happen if the operation fails midway?
7. How should duplicate submissions be handled?

## Constraints

8. Are there performance requirements (response time, throughput)?
9. Are there security or compliance constraints?

## Acceptance Criteria

10. How will you verify this feature is working correctly?
11. What does "done" look like?

---

*Add answers below each question, then trigger spec generation.*
`, title), nil
}

func extractTitle(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return "this feature"
}
