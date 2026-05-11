package task

import "testing"

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    State
		to      State
		allowed bool
	}{
		// Happy path
		{name: "idea→clarification", from: StateIdeaSubmitted, to: StateNeedsClarification, allowed: true},
		{name: "clarification→spec_drafted", from: StateNeedsClarification, to: StateSpecDrafted, allowed: true},
		{name: "spec_drafted→spec_approved", from: StateSpecDrafted, to: StateSpecApproved, allowed: true},
		{name: "spec_approved→implementing", from: StateSpecApproved, to: StateImplementing, allowed: true},
		{name: "implementing→in_review", from: StateImplementing, to: StateInReview, allowed: true},
		{name: "in_review→ready_to_merge", from: StateInReview, to: StateReadyToMerge, allowed: true},
		{name: "ready_to_merge→done", from: StateReadyToMerge, to: StateDone, allowed: true},

		// Rollback paths
		{name: "implementing→spec_approved (rollback)", from: StateImplementing, to: StateSpecApproved, allowed: true},
		{name: "in_review→implementing (rollback)", from: StateInReview, to: StateImplementing, allowed: true},

		// Error transitions
		{name: "implementing→error", from: StateImplementing, to: StateError, allowed: true},
		{name: "in_review→error", from: StateInReview, to: StateError, allowed: true},
		{name: "error→implementing (reset)", from: StateError, to: StateImplementing, allowed: true},

		// Invalid transitions
		{name: "idea→spec_drafted (skip)", from: StateIdeaSubmitted, to: StateSpecDrafted, allowed: false},
		{name: "idea→implementing (skip)", from: StateIdeaSubmitted, to: StateImplementing, allowed: false},
		{name: "spec_approved→in_review (skip)", from: StateSpecApproved, to: StateInReview, allowed: false},
		{name: "done→implementing", from: StateDone, to: StateImplementing, allowed: false},
		{name: "done→idea", from: StateDone, to: StateIdeaSubmitted, allowed: false},
		{name: "ready_to_merge→in_review", from: StateReadyToMerge, to: StateInReview, allowed: false},
		{name: "error→spec_approved", from: StateError, to: StateSpecApproved, allowed: false},
		{name: "error→done", from: StateError, to: StateDone, allowed: false},
		{name: "implementing→done (skip)", from: StateImplementing, to: StateDone, allowed: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CanTransition(tc.from, tc.to)
			if got != tc.allowed {
				t.Errorf("CanTransition(%q, %q) = %v, want %v", tc.from, tc.to, got, tc.allowed)
			}
		})
	}
}

func TestValidTransitionsCompleteness(t *testing.T) {
	// Every state in ValidTransitions map is a known state constant.
	known := map[State]bool{
		StateIdeaSubmitted:      true,
		StateNeedsClarification: true,
		StateSpecDrafted:        true,
		StateSpecApproved:       true,
		StateImplementing:       true,
		StateInReview:           true,
		StateReadyToMerge:       true,
		StateDone:               true,
		StateError:              true,
	}

	for from, tos := range ValidTransitions {
		if !known[from] {
			t.Errorf("ValidTransitions has unknown source state %q", from)
		}
		for _, to := range tos {
			if !known[to] {
				t.Errorf("ValidTransitions[%q] contains unknown target state %q", from, to)
			}
		}
	}
}

func TestStateConstants(t *testing.T) {
	// Ensure state constants have expected string values (used in JSON serialization).
	cases := []struct {
		s    State
		want string
	}{
		{StateIdeaSubmitted, "IDEA_SUBMITTED"},
		{StateNeedsClarification, "NEEDS_CLARIFICATION"},
		{StateSpecDrafted, "SPEC_DRAFTED"},
		{StateSpecApproved, "SPEC_APPROVED"},
		{StateImplementing, "IMPLEMENTING"},
		{StateInReview, "IN_REVIEW"},
		{StateReadyToMerge, "READY_TO_MERGE"},
		{StateDone, "DONE"},
		{StateError, "ERROR"},
	}
	for _, c := range cases {
		if string(c.s) != c.want {
			t.Errorf("state constant = %q, want %q", c.s, c.want)
		}
	}
}
