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
)

// ValidTransitions defines allowed state machine transitions.
var ValidTransitions = map[State][]State{
	StateIdeaSubmitted:      {StateNeedsClarification},
	StateNeedsClarification: {StateSpecDrafted},
	StateSpecDrafted:        {StateSpecApproved},
	StateSpecApproved:       {StateImplementing},
	StateImplementing:       {StateInReview, StateSpecApproved},
	StateInReview:           {StateReadyToMerge, StateImplementing},
	StateReadyToMerge:       {StateDone},
	StateDone:               {},
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
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	State     State     `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
