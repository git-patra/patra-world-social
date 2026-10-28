package lobby

import (
	"context"
	"errors"
	"fmt"
	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/world"
	"testing"
	"time"
)

type heldAreas struct {
	clear     chan struct{}
	activated chan struct{}
	released  chan struct{}
}

func (a *heldAreas) ReserveArea(context.Context, string, string, string) (world.AreaLease, error) {
	return world.AreaLease{WorldID: "world", AreaID: "race", Incarnation: "one", Generation: 1, MatchID: "m"}, nil
}
func (a *heldAreas) ClearArea(ctx context.Context, l world.AreaLease) error {
	select {
	case <-a.clear:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (a *heldAreas) ActivateArea(context.Context, world.AreaLease, int64) error {
	a.activated <- struct{}{}
	return nil
}
func (a *heldAreas) ReleaseArea(context.Context, world.AreaLease) error {
	a.released <- struct{}{}
	return nil
}
func TestStartWaitsForClearAckAndDurability(t *testing.T) {
	s, store, _ := setup(t)
	register(t, s, "race")
	a := &heldAreas{make(chan struct{}), make(chan struct{}, 1), make(chan struct{}, 1)}
	s.SetAreaCoordinator(a)
	l := join(t, s, "a", "race")
	ready(t, s, "a", l)
	store.entered = make(chan struct{}, 1)
	store.hold = make(chan struct{})
	done := make(chan error, 1)
	go func() { _, e := s.StartMatch(context.Background(), "a", l.ID, "start"); done <- e }()
	select {
	case <-store.entered:
		t.Fatal("persist before clear ACK")
	case <-time.After(30 * time.Millisecond):
	}
	close(a.clear)
	select {
	case <-store.entered:
	case <-time.After(time.Second):
		t.Fatal("missing persist")
	}
	select {
	case <-a.activated:
		t.Fatal("activated before durability")
	default:
	}
	close(store.hold)
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	select {
	case <-a.activated:
	default:
		t.Fatal("no activation ACK")
	}
}

type failedStartStore struct {
	Store
	unknown bool
}

func (s failedStartStore) Persist(context.Context, model.MatchStart, string, string) error {
	if s.unknown {
		return ErrCommitUnknown
	}
	return errors.New("definite abort")
}
func (s failedStartStore) Lookup(context.Context, string) (bool, error) {
	return false, errors.New("unknown lookup")
}
func TestDefiniteAbortReleasesButUnknownStartDoesNot(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		t.Run(fmt.Sprint(unknown), func(t *testing.T) {
			s, store, _ := setup(t)
			s.store = failedStartStore{store, unknown}
			register(t, s, "race")
			a := &heldAreas{make(chan struct{}), make(chan struct{}, 1), make(chan struct{}, 1)}
			close(a.clear)
			s.SetAreaCoordinator(a)
			l := join(t, s, "a", "race")
			ready(t, s, "a", l)
			_, err := s.StartMatch(context.Background(), "a", l.ID, "start")
			if err == nil {
				t.Fatal("failed persistence activated")
			}
			select {
			case <-a.activated:
				t.Fatal("engine activated without durability")
			default:
			}
			select {
			case <-a.released:
				if unknown {
					t.Fatal("unknown START opened area")
				}
			default:
				if !unknown {
					t.Fatal("definite abort retained lease")
				}
			}
		})
	}
}
func TestLateResultPersistenceDoesNotKeepSpatialLock(t *testing.T) {
	s, _, c := setup(t)
	register(t, s, "race")
	a := &heldAreas{make(chan struct{}), make(chan struct{}, 1), make(chan struct{}, 1)}
	close(a.clear)
	s.SetAreaCoordinator(a)
	l := join(t, s, "a", "race")
	ready(t, s, "a", l)
	m, err := s.StartMatch(context.Background(), "a", l.ID, "start")
	if err != nil {
		t.Fatal(err)
	}
	c.advance(4 * time.Second)
	s.Sync(context.Background())
	if err = s.Transition(context.Background(), m.MatchID, "finalizing"); err != nil {
		t.Fatal(err)
	}
	s.Sync(context.Background())
	select {
	case <-a.released:
	case <-time.After(time.Second):
		t.Fatal("result finalization retained spatial lease")
	}
	if !s.Busy("a") {
		t.Fatal("spatial release prematurely released durable claim")
	}
}
