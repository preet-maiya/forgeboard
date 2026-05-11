package scheduler

import (
	"fmt"
	"log"
	"strings"
	"time"

	ghclient "forgeboard/internal/github"
	"forgeboard/internal/task"
)

// Scheduler polls tasks on a fixed interval.
// When GitHub is configured it also polls IN_REVIEW tasks for merge readiness.
type Scheduler struct {
	repo     *task.Repository
	interval time.Duration
	done     chan struct{}
	github   *ghclient.Client // nil if GitHub not configured
}

func New(repo *task.Repository, interval time.Duration, gh *ghclient.Client) *Scheduler {
	return &Scheduler{
		repo:     repo,
		interval: interval,
		done:     make(chan struct{}),
		github:   gh,
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
		case task.StateInReview:
			pending++
			if s.github != nil && t.PRNumber > 0 {
				go s.checkMergeReadiness(t)
			}
		case task.StateError:
			log.Printf("scheduler: [%s] %s - ERROR: %s", t.ID, t.Title, t.Error)
			pending++
		}
	}

	if pending == 0 {
		log.Printf("scheduler: scan complete, no pending tasks (%d total)", len(tasks))
	} else {
		log.Printf("scheduler: scan complete, %d task(s) need attention", pending)
	}
}

// checkMergeReadiness checks if both bot+human have approved the PR and transitions to READY_TO_MERGE.
func (s *Scheduler) checkMergeReadiness(t *task.Task) {
	open, err := s.github.IsPROpen(t.PRNumber)
	if err != nil {
		if ghclient.IsRateLimit(err) {
			log.Printf("scheduler: rate limit checking PR open for %s, skipping cycle", t.ID)
			return
		}
		log.Printf("scheduler: error checking PR open for %s: %v", t.ID, err)
		return
	}
	if !open {
		tsk, err := s.repo.Get(t.ID)
		if err == nil {
			tsk.PRClosed = true
			_ = s.repo.UpdateTask(tsk)
		}
		log.Printf("scheduler: PR #%d for %s was closed externally — wrote pr_closed warning", t.PRNumber, t.ID)
		return
	}

	reviews, err := s.github.GetPRReviews(t.PRNumber)
	if err != nil {
		if ghclient.IsRateLimit(err) {
			log.Printf("scheduler: rate limit getting reviews for %s, skipping cycle", t.ID)
			return
		}
		log.Printf("scheduler: error getting reviews for %s: %v", t.ID, err)
		return
	}

	// Latest review per user (later reviews supersede earlier ones).
	latest := make(map[string]*ghclient.Review)
	for i := range reviews {
		r := &reviews[i]
		latest[r.User.Login] = r
	}

	botUser := s.github.BotUser()
	botApproved := false
	botHasLGTM := false
	humanApproved := false

	for login, r := range latest {
		if r.State != "APPROVED" {
			continue
		}
		if login == botUser {
			botApproved = true
			if strings.Contains(r.Body, "LGTM") {
				botHasLGTM = true
			}
		} else {
			humanApproved = true
		}
	}

	if !botApproved || !botHasLGTM || !humanApproved {
		return
	}

	// Both conditions met — transition to READY_TO_MERGE.
	tsk, err := s.repo.Get(t.ID)
	if err != nil {
		log.Printf("scheduler: get task %s for READY_TO_MERGE: %v", t.ID, err)
		return
	}
	if tsk.State != task.StateInReview {
		return // already transitioned by another goroutine
	}
	tsk.State = task.StateReadyToMerge
	tsk.Notification = fmt.Sprintf("PR #%d approved. Merge manually.", t.PRNumber)
	if err := s.repo.UpdateTask(tsk); err != nil {
		log.Printf("scheduler: update task %s to READY_TO_MERGE: %v", t.ID, err)
		return
	}
	log.Printf("scheduler: task %s → READY_TO_MERGE (PR #%d)", t.ID, t.PRNumber)
}
