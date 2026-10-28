package realtime

import (
	"context"
	"errors"
	"fmt"
	"patra-world-social/server/internal/auth"
	"patra-world-social/server/internal/world"
	"patra-world-social/server/internal/worlds"
	"sync"
	"testing"
	"time"
)

type testAuth struct {
	mu      sync.Mutex
	revoked map[string]bool
}

func (a *testAuth) RequireSession(ctx context.Context, token string) (auth.Session, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.revoked[token] {
		return auth.Session{}, auth.ErrInvalid
	}
	return auth.Session{User: auth.User{ID: token, DisplayName: token}, Token: token}, nil
}

type testMembers struct{}

func (testMembers) AuthorizeMember(ctx context.Context, u, w string) (worlds.Membership, error) {
	if w == "denied" {
		return worlds.Membership{}, worlds.ErrForbidden
	}
	return worlds.Membership{UserID: u, WorldID: w}, nil
}
func newTestHub(t *testing.T) *Hub {
	h := NewHub(&testAuth{revoked: map[string]bool{}}, testMembers{}, world.MapDefinition{Width: 64, Height: 64, Spawn: world.Point{X: 32, Y: 32}}, world.WallClock{})
	t.Cleanup(h.Close)
	return h
}
func TestConcurrentSeatsReconnectAndIsolation(t *testing.T) {
	h := newTestHub(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	results := make(chan PublicSnapshot, 30)
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s, e := h.Join(ctx, JoinContext{Token: fmt.Sprint(i), WorldID: "a"})
			if e == nil {
				results <- s
			} else if !errors.Is(e, ErrFull) {
				t.Error(e)
			}
		}(i)
	}
	wg.Wait()
	close(results)
	if len(results) != 20 {
		t.Fatalf("seats %d", len(results))
	}
	s := <-results
	j := JoinContext{Token: s.SelfID, WorldID: "a", SessionID: s.SessionID, Epoch: s.Epoch}
	h.Disconnect(s.SessionID, "network")
	next, e := h.Resume(ctx, j, 0)
	if e != nil || next.Epoch <= s.Epoch || len(next.Avatars) != 20 {
		t.Fatalf("resume %+v %v", next, e)
	}
	if _, e = h.Resume(ctx, j, 0); !errors.Is(e, ErrFenced) {
		t.Fatalf("old epoch resumed %v", e)
	}
	if _, e = h.Join(ctx, JoinContext{Token: s.SelfID, WorldID: "a"}); !errors.Is(e, ErrTakeover) {
		t.Fatal(e)
	}
	b, e := h.Join(ctx, JoinContext{Token: "other", WorldID: "b"})
	if e != nil || len(b.Avatars) != 1 || b.Avatars[0].ID != "other" {
		t.Fatalf("isolation %+v %v", b, e)
	}
	h.Revoke("a", s.SelfID)
	if _, e = h.Resume(ctx, JoinContext{Token: s.SelfID, WorldID: "a", SessionID: next.SessionID, Epoch: next.Epoch}, 0); e == nil {
		t.Fatal("revoked seat resumed")
	}
}
func TestTakeoverFencesOldCommandsAndDisconnect(t *testing.T) {
	h := newTestHub(t)
	ctx := context.Background()
	old, _ := h.Join(ctx, JoinContext{Token: "u", WorldID: "a"})
	next, e := h.Join(ctx, JoinContext{Token: "u", WorldID: "b", Takeover: true})
	if e != nil {
		t.Fatal(e)
	}
	j := JoinContext{Token: "u", WorldID: "a", SessionID: old.SessionID, Epoch: old.Epoch}
	if e = h.Submit(ctx, j, Command{V: 2, CommandID: "x", Seq: 1, WorldID: "a", Type: "player.input", Payload: []byte(`{"dx":1,"dy":0}`)}); !errors.Is(e, ErrFenced) {
		t.Fatal(e)
	}
	h.Disconnect(old.SessionID, "old socket")
	s, e := h.Snapshot(ctx, JoinContext{Token: "u", WorldID: "b", SessionID: next.SessionID, Epoch: next.Epoch})
	if e != nil || s.Avatars[0].Status != "available" {
		t.Fatalf("old disconnect affected new %+v %v", s, e)
	}
}
func TestRevocationDuringAdmissionCannotResurrect(t *testing.T) {
	h := newTestHub(t)
	gate := make(chan struct{})
	entered := make(chan struct{})
	h.members = blockingMembers{entered, gate}
	result := make(chan error, 1)
	go func() { _, e := h.Join(context.Background(), JoinContext{Token: "u", WorldID: "a"}); result <- e }()
	<-entered
	h.Revoke("a", "u")
	close(gate)
	select {
	case e := <-result:
		if e == nil {
			t.Fatal("stale authorization admitted")
		}
	case <-time.After(time.Second):
		t.Fatal("blocked owner")
	}
}

type blockingMembers struct{ entered, gate chan struct{} }

func (b blockingMembers) AuthorizeMember(context.Context, string, string) (worlds.Membership, error) {
	close(b.entered)
	<-b.gate
	return worlds.Membership{}, nil
}

type hubClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *hubClock) Now() time.Time      { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *hubClock) add(d time.Duration) { c.mu.Lock(); c.now = c.now.Add(d); c.mu.Unlock() }
func TestDisconnectedSeatExpiresAndRevokedSessionCannotResume(t *testing.T) {
	c := &hubClock{now: time.Unix(1000, 0)}
	a := &testAuth{revoked: map[string]bool{}}
	h := NewHub(a, testMembers{}, world.MapDefinition{Width: 64, Height: 64, Spawn: world.Point{X: 32, Y: 32}}, c)
	defer h.Close()
	ctx := context.Background()
	s, e := h.Join(ctx, JoinContext{Token: "u", WorldID: "a"})
	if e != nil {
		t.Fatal(e)
	}
	j := JoinContext{Token: "u", WorldID: "a", SessionID: s.SessionID, Epoch: s.Epoch}
	h.Disconnect(s.SessionID, "offline")
	c.add(15 * time.Second)
	next, e := h.Resume(ctx, j, 0)
	if e != nil {
		t.Fatal(e)
	}
	j.SessionID = next.SessionID
	j.Epoch = next.Epoch
	a.mu.Lock()
	a.revoked["u"] = true
	a.mu.Unlock()
	if e = h.Validate(ctx, j); !errors.Is(e, ErrUnauthorized) {
		t.Fatal(e)
	}
	if _, e = h.Resume(ctx, j, 0); e == nil {
		t.Fatal("revoked session resumed")
	}
	s, _ = h.Join(ctx, JoinContext{Token: "v", WorldID: "b"})
	h.Disconnect(s.SessionID, "offline")
	c.add(30 * time.Second)
	if _, e = h.Resume(ctx, JoinContext{Token: "v", WorldID: "b", SessionID: s.SessionID, Epoch: s.Epoch}, 0); !errors.Is(e, ErrFenced) {
		t.Fatal("expired seat resumed", e)
	}
	if _, e = h.Join(ctx, JoinContext{Token: "v", WorldID: "b"}); e != nil {
		t.Fatal("expired seat blocked new join", e)
	}
}
func TestCrossWorldAndMalformedInputsNeverMutate(t *testing.T) {
	h := newTestHub(t)
	ctx := context.Background()
	s, _ := h.Join(ctx, JoinContext{Token: "u", WorldID: "a"})
	j := JoinContext{Token: "u", WorldID: "a", SessionID: s.SessionID, Epoch: s.Epoch}
	for _, cmd := range []Command{{V: 2, CommandID: "x", WorldID: "b", Type: "player.input", Seq: 1, Payload: []byte(`{"dx":1,"dy":0}`)}, {V: 2, CommandID: "x", WorldID: "a", Type: "player.input", Seq: 1, Payload: []byte(`{"dx":100,"dy":0}`)}, {V: 2, CommandID: "x", WorldID: "a", Type: "player.destination", Seq: 1, Payload: []byte(`{"x":64,"y":32}`)}} {
		if e := h.Submit(ctx, j, cmd); e == nil {
			t.Fatal("invalid movement accepted")
		}
	}
	next, _ := h.Snapshot(ctx, j)
	if next.Avatars[0].X != s.Avatars[0].X {
		t.Fatal("invalid movement mutated")
	}
}

func TestUnrelatedRevocationDoesNotFenceAuthorization(t *testing.T) {
	for _, operation := range []string{"join", "validate", "resume"} {
		t.Run(operation, func(t *testing.T) {
			h := newTestHub(t)
			ctx := context.Background()
			j := JoinContext{Token: "b", WorldID: "b"}
			if operation != "join" {
				s, e := h.Join(ctx, j)
				if e != nil {
					t.Fatal(e)
				}
				j.SessionID = s.SessionID
				j.Epoch = s.Epoch
				if operation == "resume" {
					h.Disconnect(s.SessionID, "network")
				}
			}
			entered, release := make(chan struct{}), make(chan struct{})
			h.members = blockingMembers{entered, release}
			result := make(chan error, 1)
			go func() {
				var e error
				switch operation {
				case "join":
					_, e = h.Join(ctx, j)
				case "resume":
					_, e = h.Resume(ctx, j, 0)
				default:
					e = h.Validate(ctx, j)
				}
				result <- e
			}()
			<-entered
			h.Revoke("a", "a")
			close(release)
			if e := <-result; e != nil {
				t.Fatalf("unrelated revoke fenced %s: %v", operation, e)
			}
		})
	}
}

type admissionGate struct {
	room
	cancel context.CancelFunc
	stage  string
}

func (g admissionGate) Join(ctx context.Context, id, name string, epoch uint64) error {
	if g.stage == "join" {
		g.cancel()
	}
	return g.room.Join(ctx, id, name, epoch)
}
func (g admissionGate) Snapshot(ctx context.Context) (world.Snapshot, error) {
	if g.stage == "snapshot" {
		g.cancel()
	}
	return g.room.Snapshot(ctx)
}
func TestCancelledAdmissionLeavesNoUnknownSeatOrGhost(t *testing.T) {
	for _, mode := range []string{"join", "takeover", "resume"} {
		for _, stage := range []string{"join", "snapshot"} {
			t.Run(mode+"/"+stage, func(t *testing.T) {
				h := newTestHub(t)
				base := context.Background()
				anchor, _ := h.Join(base, JoinContext{Token: "anchor", WorldID: "a"})
				known := map[string]bool{anchor.SessionID: true}
				j := JoinContext{Token: "u", WorldID: "a"}
				if mode != "join" {
					s, e := h.Join(base, j)
					if e != nil {
						t.Fatal(e)
					}
					known[s.SessionID] = true
					j.SessionID = s.SessionID
					j.Epoch = s.Epoch
					if mode == "resume" {
						h.Disconnect(s.SessionID, "network")
					} else {
						j.Takeover = true
					}
				}
				ctx, cancel := context.WithCancel(base)
				defer cancel()
				original := h.rooms["a"]
				h.rooms["a"] = admissionGate{room: original, cancel: cancel, stage: stage}
				var e error
				if mode == "resume" {
					_, e = h.Resume(ctx, j, 0)
				} else {
					_, e = h.Join(ctx, j)
				}
				if e == nil {
					t.Fatal("canceled admission unexpectedly succeeded")
				}
				h.mu.Lock()
				defer h.mu.Unlock()
				for id := range h.seats {
					if !known[id] {
						t.Errorf("failed admission published unknown seat %s", id)
					}
				}
				snapshot, e := original.Snapshot(base)
				if e != nil {
					t.Fatal(e)
				}
				for _, a := range snapshot.Avatars {
					if h.users[a.ID] == nil {
						t.Errorf("uncounted ghost avatar %s", a.ID)
					}
				}
				if len(snapshot.Avatars) != len(h.seats) {
					t.Errorf("avatars=%d seats=%d", len(snapshot.Avatars), len(h.seats))
				}
			})
		}
	}
}
