package race

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/lobby"
)

type presenceClock struct{ millis atomic.Int64 }

func (c *presenceClock) Now() time.Time { return time.UnixMilli(c.millis.Load()) }
func (c *presenceClock) set(at int64)   { c.millis.Store(at) }

type presenceStore struct {
	entered  chan struct{}
	release  chan struct{}
	terminal chan string
}

func (s *presenceStore) Persist(ctx context.Context, _ model.MatchStart, _, _ string) error {
	if s.entered != nil {
		close(s.entered)
		select {
		case <-s.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
func (*presenceStore) Lookup(context.Context, string) (bool, error)  { return true, nil }
func (*presenceStore) Schedule(context.Context, string, int64) error { return nil }
func (s *presenceStore) Terminal(_ context.Context, _, status string) error {
	s.terminal <- status
	return nil
}
func (*presenceStore) ReleaseUser(context.Context, string, string) error { return nil }

// The observer delegates every operation immediately to the real adapter. It
// checks every actual owner metadata call, without choosing the map's order.
// Snapshot copies survive asynchronous abandoned-arena cleanup.
type observedRace struct {
	*Adapter
	t        *testing.T
	mu       sync.Mutex
	last     PublicSnapshot
	metadata []string
}

func (a *observedRace) remember() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.last = a.Snapshot()
}
func (a *observedRace) saved() (PublicSnapshot, []string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.last, append([]string{}, a.metadata...)
}
func (a *observedRace) Start(at int64) { a.Adapter.Start(at); a.remember() }
func (a *observedRace) Tick(at time.Time) bool {
	finished := a.Adapter.Tick(at)
	a.remember()
	return finished
}
func (a *observedRace) PresenceAt(user, kind string, at time.Time) {
	before := a.Snapshot()
	a.Adapter.PresenceAt(user, kind, at)
	after := a.Snapshot()
	if kind == "disconnect" {
		if before.Frame != after.Frame || before.ServerTimeMs != after.ServerTimeMs || before.Phase != after.Phase || before.FinishedAt != after.FinishedAt {
			a.t.Errorf("owner disconnect metadata advanced race: user=%s at=%d before=%+v after=%+v", user, at.UnixMilli(), before, after)
		}
		for i, k := range after.Karts {
			if k.X != before.Karts[i].X || k.Y != before.Karts[i].Y {
				a.t.Error("owner metadata moved kart", user, i)
			}
		}
		a.mu.Lock()
		a.metadata = append(a.metadata, user)
		a.mu.Unlock()
	}
	a.remember()
}
func (a *observedRace) Close() { a.remember(); a.Adapter.Close() }

func presenceLobby(t *testing.T, store *presenceStore, humans int) (*lobby.Service, *presenceClock, lobby.Lobby, <-chan *observedRace) {
	t.Helper()
	clock := &presenceClock{}
	clock.set(1000)
	service := lobby.New(store, clock, model.Version{})
	t.Cleanup(service.Close)
	created := make(chan *observedRace, 1)
	err := service.Register("race", func(m model.MatchStart) (lobby.Engine, error) {
		concrete, err := Factory(clock)(m)
		if err != nil {
			return nil, err
		}
		observed := &observedRace{Adapter: concrete.(*Adapter), t: t}
		created <- observed
		return observed, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var joined lobby.Lobby
	for i := 0; i < humans; i++ {
		user := string(rune('a' + i))
		joined, err = service.JoinLobby(context.Background(), lobby.JoinContext{UserID: user, WorldID: "world", DisplayName: user}, "race")
		if err != nil {
			t.Fatal(err)
		}
		if err = service.SetReady(context.Background(), user, joined.ID, true); err != nil {
			t.Fatal(err)
		}
	}
	return service, clock, joined, created
}

func TestLobbyRaceHeldOwnerInstallsWholeDisconnectBatchBeforeAdvance(t *testing.T) {
	cases := []struct {
		name                             string
		disconnectA, disconnectB, ingest int64
		phase                            string
	}{
		{"takeover", 4000, 5000, 5000, "racing"},
		{"first_grace_expired", 4000, 5000, 19500, "racing"},
		{"all_humans_abandon", 4000, 5000, 20500, "abandoned"},
		{"natural_at_grace", 289000, 290000, 304000, "finished"},
		{"natural_before_late_ingestion", 289000, 290000, 304500, "finished"},
	}
	for _, tc := range cases {
		for _, order := range [][]string{{"a", "b"}, {"b", "a"}} {
			t.Run(tc.name+"/"+order[0]+order[1], func(t *testing.T) {
				ctx := context.Background()
				store := &presenceStore{terminal: make(chan string, 1)}
				service, clock, joined, created := presenceLobby(t, store, 2)
				start, err := service.StartMatch(ctx, "a", joined.ID, "batch")
				if err != nil {
					t.Fatal(err)
				}
				observed := <-created
				entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
				go func() {
					done <- service.ReadEngine(ctx, "world", start.MatchID, func(lobby.Engine) error { close(entered); <-release; return nil })
				}()
				<-entered
				for _, user := range order {
					at := tc.disconnectA
					if user == "b" {
						at = tc.disconnectB
					}
					// Only the held owner could consume time; reversing enqueue order tests
					// the same timestamped batch, not a backwards simulation clock.
					clock.set(at)
					service.Notify("disconnect", "world", user)
				}
				clock.set(tc.ingest)
				close(release)
				if err = <-done; err != nil {
					t.Fatal(err)
				}
				if err = service.Sync(ctx); err != nil {
					t.Fatal(err)
				}
				got, metadata := observed.saved()
				if len(metadata) != 2 {
					t.Fatal("owner did not install both disconnects", metadata)
				}
				expected := New(start, DefaultTrack(), DefaultRules())
				expected.Presence("a", "disconnect", time.UnixMilli(tc.disconnectA))
				expected.Presence("b", "disconnect", time.UnixMilli(tc.disconnectB))
				expected.Tick(time.UnixMilli(tc.ingest))
				if got.Phase != tc.phase || !reflect.DeepEqual(got, expected.Snapshot("")) {
					t.Fatalf("actual owner differs from complete chronological batch: installation=%v\ngot=%+v\nwant=%+v", metadata, got, expected.Snapshot(""))
				}
				if tc.name == "first_grace_expired" && (!got.Karts[0].DNF || got.Karts[0].X != DefaultTrack().Grid[0].X || got.Karts[1].DNF || got.Karts[1].Speed == 0) {
					t.Fatal("original 15s grace not honored independently", got.Karts[:2])
				}
				if tc.phase == "finished" {
					changes, err := service.DrainChanges(ctx)
					if err != nil {
						t.Fatal(err)
					}
					if len(changes.Events) != 1 || changes.Events[0].Type != "race.finished" || got.Result == nil || got.FinishedAt != 304000 {
						t.Fatal("natural terminal result/event missing", changes, got)
					}
				}
				if tc.phase == "abandoned" {
					if got.Result != nil {
						t.Fatal("abandon invented official result")
					}
					select {
					case status := <-store.terminal:
						if status != "abandoned" {
							t.Fatal(status)
						}
					case <-time.After(time.Second):
						t.Fatal("owner did not release abandoned match")
					}
				}
			})
		}
	}
}

func TestLobbyRaceActivationReplaysOriginalDisconnectsBeforeDepartures(t *testing.T) {
	ctx := context.Background()
	store := &presenceStore{entered: make(chan struct{}), release: make(chan struct{}), terminal: make(chan string, 1)}
	service, clock, joined, created := presenceLobby(t, store, 3)
	type startResult struct {
		start model.MatchStart
		err   error
	}
	done := make(chan startResult, 1)
	go func() {
		start, err := service.StartMatch(ctx, "a", joined.ID, "activation")
		done <- startResult{start, err}
	}()
	<-store.entered
	observed := <-created
	clock.set(2000)
	service.Notify("disconnect", "world", "a")
	if err := service.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	clock.set(16000)
	service.Notify("disconnect", "world", "b")
	if err := service.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	// A expires while persistence is outstanding; B still owns its grace.
	clock.set(18000)
	if err := service.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	close(store.release)
	result := <-done
	if result.err != nil {
		t.Fatal(result.err)
	}
	if result.start.StartsAt != 21000 {
		t.Fatal("activation did not retain fresh countdown", result.start.StartsAt)
	}
	if err := service.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	snapshot, metadata := observed.saved()
	if len(metadata) != 2 || snapshot.Frame != 0 || snapshot.Phase != "countdown" || !snapshot.Karts[0].DNF || snapshot.Karts[1].DNF || !snapshot.Karts[0].BotAssisted || !snapshot.Karts[1].BotAssisted {
		t.Fatal("activation lost original metadata/departure ordering", snapshot, metadata)
	}
	clock.set(21050)
	if err := service.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	snapshot, _ = observed.saved()
	if snapshot.Karts[0].X != DefaultTrack().Grid[0].X || snapshot.Karts[1].Speed == 0 {
		t.Fatal("activation grace takeover incorrect", snapshot)
	}
	clock.set(31000)
	if err := service.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	snapshot, _ = observed.saved()
	if !snapshot.Karts[1].DNF || snapshot.Karts[2].DNF || snapshot.Phase != "racing" || snapshot.Result != nil {
		t.Fatal("activation restamped original 15s grace", snapshot)
	}
}

func TestLobbyRacePostCutoffDisconnectCannotChangeOfficialResult(t *testing.T) {
	for _, offset := range []int64{0, 1} {
		t.Run(fmt.Sprintf("offset=%d", offset), func(t *testing.T) {
			ctx := context.Background()
			service, clock, joined, created := presenceLobby(t, &presenceStore{terminal: make(chan string, 1)}, 5)
			start, err := service.StartMatch(ctx, "a", joined.ID, "post-cutoff")
			if err != nil {
				t.Fatal(err)
			}
			observed := <-created
			cutoff := start.StartsAt + 300000
			clock.set(cutoff - 50)
			if err = service.Sync(ctx); err != nil {
				t.Fatal(err)
			}
			before, _ := observed.saved()
			if before.Phase != "racing" {
				t.Fatal("test did not hold preceding running tick", before)
			}
			entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
			go func() {
				done <- service.ReadEngine(ctx, "world", start.MatchID, func(lobby.Engine) error { close(entered); <-release; return nil })
			}()
			<-entered
			clock.set(cutoff + offset)
			service.Notify("disconnect", "world", "a")
			close(release)
			if err = <-done; err != nil {
				t.Fatal(err)
			}
			if err = service.Sync(ctx); err != nil {
				t.Fatal(err)
			}
			got, metadata := observed.saved()
			baseline := New(start, DefaultTrack(), DefaultRules())
			baseline.Tick(time.UnixMilli(cutoff - 50))
			baseline.Tick(time.UnixMilli(cutoff + offset))
			want, _ := baseline.Result()
			if len(metadata) != 1 || got.Result == nil || got.FinishedAt != cutoff || got.Result.Participants[0].BotAssisted || !reflect.DeepEqual(*got.Result, want) {
				t.Fatalf("late owner notification contaminated official result: got=%+v want=%+v", got.Result, want)
			}
			if !reflect.DeepEqual(got.Karts, baseline.Snapshot("").Karts) {
				t.Fatal("natural terminal positions/times changed")
			}
			changes, err := service.DrainChanges(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if len(changes.Events) != 1 || changes.Events[0].Type != "race.finished" {
				t.Fatal("terminal event missing", changes)
			}
			event := changes.Events[0].Payload.(PublicSnapshot)
			if !reflect.DeepEqual(event.Result, got.Result) {
				t.Fatal("published result differs from owner result")
			}
			clock.set(cutoff + 100)
			changes, err = service.DrainChanges(ctx)
			if err != nil || len(changes.Events) != 0 {
				t.Fatal("terminal event repeated", changes, err)
			}
		})
	}
}
