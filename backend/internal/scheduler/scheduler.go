package scheduler

import (
	"log"
	"time"

	"forgeboard/internal/task"
)

// Scheduler polls tasks on a fixed interval and logs pending work.
// This is intentionally simple: no distributed workers, no queues.
type Scheduler struct {
	repo     *task.Repository
	interval time.Duration
	done     chan struct{}
}

func New(repo *task.Repository, interval time.Duration) *Scheduler {
	return &Scheduler{
		repo:     repo,
		interval: interval,
		done:     make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	log.Printf("scheduler: starting, interval=%s", s.interval)
	go s.loop()
}

func (s *Scheduler) Stop() {
	close(s.done)
}

func (s *Scheduler) loop() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.scan()
		case <-s.done:
			log.Println("scheduler: stopped")
			return
		}
	}
}

func (s *Scheduler) scan() {
	tasks, err := s.repo.List()
	if err != nil {
		log.Printf("scheduler: list tasks error: %v", err)
		return
	}

	pending := 0
	for _, t := range tasks {
		switch t.State {
		case task.StateIdeaSubmitted:
			log.Printf("scheduler: [%s] %s - awaiting clarification", t.ID, t.Title)
			pending++
		case task.StateNeedsClarification:
			log.Printf("scheduler: [%s] %s - awaiting answers + spec generation", t.ID, t.Title)
			pending++
		case task.StateSpecDrafted:
			log.Printf("scheduler: [%s] %s - awaiting human approval", t.ID, t.Title)
			pending++
		case task.StateSpecApproved:
			log.Printf("scheduler: [%s] %s - ready to implement", t.ID, t.Title)
			pending++
		}
	}

	if pending == 0 {
		log.Printf("scheduler: scan complete, no pending tasks (%d total)", len(tasks))
	} else {
		log.Printf("scheduler: scan complete, %d task(s) need attention", pending)
	}
}
