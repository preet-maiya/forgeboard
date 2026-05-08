package spec

import (
	"fmt"
	"strings"
	"time"
)

// Generator produces a spec document from idea and clarification content.
type Generator interface {
	GenerateSpec(ideaText, clarificationText string) (string, error)
}

// MockGenerator returns a structured spec template without calling any external API.
// Replace with a real LLM-backed generator once Claude Code integration is ready.
type MockGenerator struct{}

func NewMockGenerator() *MockGenerator {
	return &MockGenerator{}
}

func (g *MockGenerator) GenerateSpec(ideaText, clarificationText string) (string, error) {
	title := extractTitle(ideaText)

	return fmt.Sprintf(`# Goal

Implement **%s** as described in idea.md. This spec was generated from the submitted idea and clarification answers.

# Requirements

- Core functionality described in idea.md must be implemented
- Feature must be accessible via the defined interface (API/UI/CLI)
- All inputs must be validated before processing
- Results must be persisted durably to disk
- Errors must be logged with enough context to debug

# Non-Goals

- Real-time streaming or push notifications
- Multi-user concurrent editing
- External API integrations beyond stubs
- Authentication or authorization changes
- Mobile-specific optimizations

# Edge Cases

- Empty or malformed input: return a clear error, do not crash
- Duplicate submissions: detect and reject, or handle idempotently
- Partial failure: log the failure state, allow retry
- Large payloads: reject gracefully with a descriptive error message

# Acceptance Criteria

- [ ] Feature can be triggered via UI and API
- [ ] Task state transitions correctly through the state machine
- [ ] All artifacts are persisted to the correct task directory
- [ ] Logs are written for each significant operation
- [ ] Manual test: submit idea → clarify → generate spec → approve → implement flow works end to end

---

*Generated: %s*
*Source: idea.md + clarification.md*
`, title, time.Now().Format("2006-01-02 15:04:05")), nil
}

func extractTitle(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return "Feature"
}
