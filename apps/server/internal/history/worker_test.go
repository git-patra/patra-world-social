package history

import (
	"context"
	"errors"
	"fmt"
	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/lobby"
	"testing"
	"time"
)

type ownerFixture struct {
	refs                     []lobby.FinalizingMatch
	result                   model.MatchResult
	blocked, detached, saved bool
	copyError                error
}

func (o *ownerFixture) Finalizing(context.Context) ([]lobby.FinalizingMatch, error) {
	if o.detached || o.saved {
		return nil, nil
	}
	return o.refs, nil
}
func (o *ownerFixture) CopyResult(context.Context, string) (model.MatchResult, error) {
	return o.result, o.copyError
}
func (o *ownerFixture) SetStorageBlocked(v bool) error                   { o.blocked = v; return nil }
func (o *ownerFixture) DetachUnconfirmed(string) error                   { o.detached = true; return nil }
func (o *ownerFixture) Transition(context.Context, string, string) error { o.saved = true; return nil }

type offlineStore struct {
	offline            bool
	durable, completed bool
	writes             int
}

func (s *offlineStore) Enqueue(context.Context, model.MatchResult) error {
	s.writes++
	if s.offline {
		return errors.New("offline")
	}
	s.durable = true
	return nil
}
func (s *offlineStore) Finalize(context.Context, string) (model.MatchResult, error) {
	if s.offline {
		return model.MatchResult{}, errors.New("offline")
	}
	s.completed = true
	return model.MatchResult{}, nil
}
func (s *offlineStore) Reconcile(context.Context, string) (string, error) {
	if s.offline {
		return "", errors.New("offline")
	}
	if s.durable {
		s.completed = true
		return "completed", nil
	}
	return "aborted_storage", nil
}
func (s *offlineStore) PendingIDs(context.Context) ([]string, error) {
	if s.offline {
		return nil, errors.New("offline")
	}
	return nil, nil
}
func (s *offlineStore) RetryLater(context.Context, string) error { return nil }
func TestWorkerDeadlineReleasesArenaWithoutClaimingCommitOutcome(t *testing.T) {
	for _, durable := range []bool{false, true} {
		t.Run(map[bool]string{false: "absent", true: "ack_lost"}[durable], func(t *testing.T) {
			o := &ownerFixture{refs: []lobby.FinalizingMatch{{MatchID: "match"}}}
			s := &offlineStore{offline: true, durable: durable}
			w := NewWorker(s, o)
			var statuses []string
			w.Notify = func(_ string, _ []string, e StatusEvent) { statuses = append(statuses, e.Status) }
			now := time.Now()
			w.Step(context.Background(), now)
			if !o.blocked || o.saved || o.detached {
				t.Fatal(o)
			}
			w.Step(context.Background(), now.Add(60*time.Second))
			if !o.blocked || !o.detached || o.saved || w.pending["match"].result != nil {
				t.Fatal("deadline fabricated success or retained game", o)
			}
			if statuses[len(statuses)-1] != "unconfirmed" {
				t.Fatal(statuses)
			}
			s.offline = false
			w.Step(context.Background(), now.Add(100*time.Second))
			want := "aborted_storage"
			if durable {
				want = "completed"
			}
			if o.blocked || statuses[len(statuses)-1] != want || len(w.pending) != 0 {
				t.Fatal(statuses, o)
			}
		})
	}
}
func TestMissingResultCapabilityIsObservableAndNeverEnqueued(t *testing.T) {
	o := &ownerFixture{refs: []lobby.FinalizingMatch{{MatchID: "missing"}}, copyError: lobby.ErrUnavailable}
	s := &offlineStore{}
	w := NewWorker(s, o)
	var status string
	w.Notify = func(_ string, _ []string, e StatusEvent) { status = e.Status }
	w.Step(context.Background(), time.Now())
	if s.writes != 0 || status != "unavailable" || !o.blocked {
		t.Fatal(s, status, o)
	}
}

func (s *offlineStore) HasPending(context.Context) (bool, error) {
	if s.offline {
		return false, errors.New("offline")
	}
	return false, nil
}

type slowBacklogStore struct {
	*offlineStore
	first      func()
	reconciles int
	durableNew bool
}

func (s *slowBacklogStore) Reconcile(ctx context.Context, id string) (string, error) {
	s.reconciles++
	if s.first != nil {
		first := s.first
		s.first = nil
		first()
	}
	if s.offline {
		<-ctx.Done()
		return "", ctx.Err()
	}
	if id == "newly-finished" && s.durableNew {
		return "completed", nil
	}
	return "aborted_storage", nil
}
func TestWorkerDiscoversNewDeadlineDuringSlowDetachedBacklog(t *testing.T) {
	for _, durable := range []bool{false, true} {
		t.Run(map[bool]string{false: "absent", true: "durable"}[durable], func(t *testing.T) {
			o := &ownerFixture{}
			store := &slowBacklogStore{offlineStore: &offlineStore{offline: true}, durableNew: durable}
			w := NewWorker(store, o)
			now := time.Now()
			for i := 0; i < 21; i++ {
				id := fmt.Sprintf("old-%02d", i)
				w.pending[id] = &pendingResult{ref: lobby.FinalizingMatch{MatchID: id}, since: now.Add(-time.Minute), detached: true}
			}
			var finished time.Time
			store.first = func() {
				finished = time.Now()
				o.refs = []lobby.FinalizingMatch{{MatchID: "newly-finished", Since: finished}}
			}
			var statuses []string
			var unconfirmedAt time.Time
			w.Notify = func(_ string, _ []string, e StatusEvent) {
				if e.MatchID == "newly-finished" {
					statuses = append(statuses, e.Status)
					if e.Status == "unconfirmed" {
						unconfirmedAt = time.Now()
					}
				}
			}
			// Bound the RED regression to six seconds rather than wait 21*3 seconds.
			// The real store still respects every context and the production Step runs.
			firstCtx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
			w.Step(firstCtx, now)
			cancel()
			if store.reconciles > 1 || time.Since(finished) > 3500*time.Millisecond {
				t.Fatalf("detached backlog monopolized owner discovery: %d reconciliations, %s", store.reconciles, time.Since(finished))
			}
			// Move the injectable Step clock near the newly exposed match's own deadline.
			// Intake must preserve its authoritative Since, not start a new 60s interval.
			w.Step(context.Background(), finished.Add(59980*time.Millisecond))
			if w.pending["newly-finished"] == nil || o.detached || !o.blocked {
				t.Fatal("new finalizing match not tracked/gated", o)
			}
			atBoundary := time.Now()
			w.Step(context.Background(), finished.Add(60*time.Second))
			if !o.detached || !o.blocked || unconfirmedAt.IsZero() || unconfirmedAt.Sub(atBoundary) > 250*time.Millisecond {
				t.Fatal("deadline release or unconfirmed notification waited behind old storage", o, unconfirmedAt.Sub(atBoundary), statuses)
			}
			if len(statuses) != 2 || statuses[0] != "finalizing" || statuses[1] != "unconfirmed" {
				t.Fatal("fabricated outcome before recovery", statuses)
			}
			store.offline = false
			for i := 0; i < 30 && len(w.pending) > 0; i++ {
				w.Step(context.Background(), finished.Add(time.Duration(100+40*i)*time.Second))
			}
			want := "aborted_storage"
			if durable {
				want = "completed"
			}
			if len(w.pending) != 0 || o.blocked || statuses[len(statuses)-1] != want {
				t.Fatal("reconciliation failed to drain/reopen with actual outcome", len(w.pending), o.blocked, statuses)
			}
		})
	}
}
