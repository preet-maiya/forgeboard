package task

import "time"

type State string

const (
	StateIdeaSubmitted      State = "IDEA_SUBMITTED"
	StateNeedsClarification State = "NEEDS_CLARIFICATION"
	StateSpecDrafted        State = "SPEC_DRAFTED"
	StateSpecApproved       State = "SPEC_APPROVED"
	StateImplementing       State = "IMPLEMENTING"
	StateInReview           State = "IN_REVIEW"
	StateReadyToMerge       State = "READY_TO_MERGE"
	StateDone               State = "DONE"
	StateError              State = "ERROR"
)

// ValidTransitions defines allowed state machine transitions.
var ValidTransitions = map[State][]State{
	StateIdeaSubmitted:      {StateNeedsClarification},
	StateNeedsClarification: {StateSpecDrafted},
	StateSpecDrafted:        {StateSpecApproved},
	StateSpecApproved:       {StateImplementing},
	StateImplementing:       {StateInReview, StateSpecApproved, StateError},
	StateInReview:           {StateReadyToMerge, StateImplementing, StateError},
	StateReadyToMerge:       {StateDone},
	StateDone:               {},
	StateError:              {StateImplementing},
}

// CanTransition returns true if transitioning from current to next is valid.
func CanTransition(current, next State) bool {
	for _, s := range ValidTransitions[current] {
		if s == next {
			return true
		}
	}
	return false
}

type Task struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	State        State     `json:"state"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	PRNumber     int       `json:"pr_number,omitempty"`
	Notification string    `json:"notification,omitempty"`
	Error        string    `json:"error,omitempty"`
	ReviewError  string    `json:"review_error,omitempty"`
	PushError    string    `json:"push_error,omitempty"`
	PRClosed     bool      `json:"pr_closed,omitempty"`
}
