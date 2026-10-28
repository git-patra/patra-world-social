package history

import (
	"context"
	"errors"
	"log/slog"
	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/lobby"
	"time"
)

type ResultStore interface {
	Enqueue(context.Context, model.MatchResult) error
	Finalize(context.Context, string) (model.MatchResult, error)
	Reconcile(context.Context, string) (string, error)
	PendingIDs(context.Context) ([]string, error)
	RetryLater(context.Context, string) error
	HasPending(context.Context) (bool, error)
}
type ResultOwner interface {
	Finalizing(context.Context) ([]lobby.FinalizingMatch, error)
	CopyResult(context.Context, string) (model.MatchResult, error)
	SetStorageBlocked(bool) error
	DetachUnconfirmed(string) error
	Transition(context.Context, string, string) error
}
type StatusEvent struct {
	MatchID string `json:"matchId"`
	Status  string `json:"status"`
}
type pendingResult struct {
	ref               lobby.FinalizingMatch
	result            *model.MatchResult
	since, next       time.Time
	attempts          int
	detached, invalid bool
	recovered         bool
}

// Worker has one owner goroutine. Each storage operation is joined before the
// next begins; context cancellation at the deadline cannot leave a retry racing
// reconciliation in this process. The database lock fences server-side work.
type Worker struct {
	Store   ResultStore
	Owner   ResultOwner
	Notify  func(world string, users []string, event StatusEvent)
	pending map[string]*pendingResult
}

func NewWorker(store ResultStore, owner ResultOwner) *Worker {
	return &Worker{Store: store, Owner: owner, pending: map[string]*pendingResult{}}
}
func (w *Worker) notify(p *pendingResult, status string) {
	if w.Notify != nil {
		w.Notify(p.ref.WorldID, p.ref.Users, StatusEvent{p.ref.MatchID, status})
	}
}
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		w.Step(ctx, time.Now())
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Step is serialized by Run; exported to allow deterministic fault/deadline tests.
func (w *Worker) Step(ctx context.Context, now time.Time) {
	stepStarted := time.Now()
	stepCtx, stepCancel := context.WithTimeout(ctx, 3*time.Second)
	defer stepCancel()
	refs, e := w.Owner.Finalizing(stepCtx)
	if e != nil {
		return
	}
	for _, ref := range refs {
		if w.pending[ref.MatchID] != nil {
			continue
		}
		since := ref.Since
		if since.IsZero() {
			since = now
		}
		p := &pendingResult{ref: ref, since: since}
		w.pending[ref.MatchID] = p
		w.Owner.SetStorageBlocked(true)
		r, err := w.Owner.CopyResult(stepCtx, ref.MatchID)
		if err != nil {
			p.invalid = true
			slog.Error("authoritative result unavailable", "match_id", ref.MatchID)
			w.notify(p, "unavailable")
		} else {
			p.result = &r
			w.notify(p, "finalizing")
		}
	}
	failed := false
	for id, p := range w.pending {
		current := now.Add(time.Since(stepStarted))
		if !p.detached && current.Sub(p.since) >= 60*time.Second {
			// All previous Enqueue/Finalize calls returned; drop the complete result only
			// after cancel/join, then retain IDs for reconciliation, never a success claim.
			w.Owner.SetStorageBlocked(true)
			if e = w.Owner.DetachUnconfirmed(id); e != nil {
				failed = true
				continue
			}
			p.detached = true
			p.result = nil
			p.next = current
			w.notify(p, "unconfirmed")
		}

	}
	// Check every live deadline before selecting storage work. At most one ready
	// match is attempted per Step, so a detached backlog cannot hide new owner
	// completions for N*3 seconds. Live results take priority; oldest ready retry
	// (then ID) wins within a class, so map iteration cannot starve old work.
	current := now.Add(time.Since(stepStarted))
	id := w.nextReady(current)
	op, cancel := w.operationContext(stepCtx, current)
	defer cancel()
	if id != "" {
		p := w.pending[id]
		if p.recovered {
			r, err := w.Store.Finalize(op, id)
			e = err
			if e == nil {
				_, e = w.Store.Reconcile(op, id)
			}
			if e == nil {
				p.ref.WorldID = r.WorldID
				for _, a := range r.Participants {
					if a.UserID != "" {
						p.ref.Users = append(p.ref.Users, a.UserID)
					}
				}
				w.notify(p, "completed")
			}
		} else if p.detached {
			var status string
			status, e = w.Store.Reconcile(op, id)
			if e == nil {
				w.notify(p, status)
			}
		} else {
			e = w.Store.Enqueue(op, *p.result)
			if e == nil {
				_, e = w.Store.Finalize(op, id)
			}
			if e == nil {
				e = w.Owner.Transition(op, id, "results")
				if e == nil {
					w.notify(p, "completed")
				}
			}
		}
		if e == nil {
			delete(w.pending, id)
		} else {
			failed = true
			if errors.Is(e, ErrConflict) || errors.Is(e, ErrInvalid) {
				p.invalid = true
				slog.Error("authoritative result rejected; investigation required", "match_id", id)
				w.notify(p, "unavailable")
			}
			p.attempts++
			p.next = current.Add(backoff(p.attempts))
			if p.recovered {
				w.Store.RetryLater(op, id)
			}
			w.Owner.SetStorageBlocked(true)
		}
	}
	// Pending durable jobs include a prior process's enqueued work. Live pending
	// IDs are excluded so their 60s cancellation and detach policy stays authoritative.
	ids, err := w.Store.PendingIDs(op)
	if err != nil {
		failed = true
	} else {
		for _, id := range ids {
			if w.pending[id] != nil {
				continue
			}
			if len(w.pending) >= 50 {
				break
			}
			w.pending[id] = &pendingResult{ref: lobby.FinalizingMatch{MatchID: id}, since: now, detached: true, recovered: true}
		}
	}
	remaining, checkErr := w.Store.HasPending(op)
	w.Owner.SetStorageBlocked(failed || len(w.pending) > 0 || err != nil || checkErr != nil || remaining)
}

// nextReady prioritizes still-attached results, then the least-recently
// scheduled ready attempt. Persistent errors get a future next time/backoff.
func (w *Worker) nextReady(now time.Time) string {
	var selected string
	for id, p := range w.pending {
		if now.Before(p.next) || (p.invalid && !p.detached) {
			continue
		}
		if selected == "" {
			selected = id
			continue
		}
		best := w.pending[selected]
		if (!p.detached && best.detached) || (p.detached == best.detached && (p.next.Before(best.next) || (p.next.Equal(best.next) && id < selected))) {
			selected = id
		}
	}
	return selected
}

// The same context covers the selected attempt, retry bookkeeping and both
// probes. It is bounded by the total Step budget and every known live deadline.
// Unknown owner completions are discovered on the next bounded Step.
func (w *Worker) operationContext(ctx context.Context, now time.Time) (context.Context, context.CancelFunc) {
	timeout := 3 * time.Second
	for _, p := range w.pending {
		if !p.detached {
			left := p.since.Add(60 * time.Second).Sub(now)
			if left < timeout {
				timeout = left
			}
		}
	}
	return context.WithTimeout(ctx, timeout)
}

func backoff(attempt int) time.Duration {
	if attempt > 5 {
		attempt = 5
	}
	return time.Duration(1<<attempt) * time.Second
}
func (s *Service) PendingIDs(ctx context.Context) ([]string, error) {
	rows, e := s.Pool.Query(ctx, `SELECT match_id::text FROM result_jobs WHERE completed_at IS NULL AND next_attempt_at<=now() ORDER BY next_attempt_at,match_id LIMIT 50`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			return nil, e
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
func (s *Service) RetryLater(ctx context.Context, id string) error {
	_, e := s.Pool.Exec(ctx, `UPDATE result_jobs SET attempts=least(attempts+1,30),next_attempt_at=now()+make_interval(secs=>least(32,power(2,least(attempts+1,5)))::double precision) WHERE match_id=$1 AND completed_at IS NULL`, id)
	return e
}

func (s *Service) HasPending(ctx context.Context) (bool, error) {
	var pending bool
	e := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM result_jobs WHERE completed_at IS NULL)`).Scan(&pending)
	return pending, e
}
