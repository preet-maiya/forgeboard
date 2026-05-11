package review

import "sync"

// Queue manages per-task review agent runs.
// At most one run executes per task at a time; at most one additional run can be queued.
// A second signal while a run is in progress and one is already queued is a no-op.
type Queue struct {
	mu    sync.Mutex
	slots map[string]*slot
}

type slot struct {
	running bool
	queued  bool
}

// Enqueue attempts to schedule a review run for taskID.
// Returns runNow=true if the caller should start a goroutine immediately.
// Returns runNow=false if a run is already in progress (queued or no-op).
func (q *Queue) Enqueue(taskID string) (runNow bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	s := q.getOrCreate(taskID)
	if !s.running {
		s.running = true
		return true
	}
	if !s.queued {
		s.queued = true
	}
	return false
}

// Complete marks the current run as finished.
// Returns startNext=true if a queued run is waiting and the caller should start a new goroutine.
func (q *Queue) Complete(taskID string) (startNext bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	s := q.getOrCreate(taskID)
	if s.queued {
		s.queued = false
		// s.running stays true — next run is starting
		return true
	}
	s.running = false
	return false
}

func (q *Queue) getOrCreate(taskID string) *slot {
	if q.slots == nil {
		q.slots = make(map[string]*slot)
	}
	if q.slots[taskID] == nil {
		q.slots[taskID] = &slot{}
	}
	return q.slots[taskID]
}
