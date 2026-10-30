package realtime

import (
	"context"
	"fmt"
	"patra-world-social/server/internal/world"
	"sync"
	"testing"
	"time"
)

type areaClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *areaClock) Now() time.Time      { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *areaClock) add(d time.Duration) { c.mu.Lock(); defer c.mu.Unlock(); c.now = c.now.Add(d) }
func TestRecreatedRuntimeRestoresActiveLeasesByRetainingOwner(t *testing.T) {
	c := &areaClock{now: time.Unix(1000, 0)}
	h := NewHub(&testAuth{revoked: map[string]bool{}}, testMembers{}, world.MapDefinition{Width: 64, Height: 64, Spawn: world.Point{X: 32, Y: 32}}, c)
	defer h.Close()
	ctx := context.Background()
	snap, err := h.Join(ctx, JoinContext{Token: "u", WorldID: "w"})
	if err != nil {
		t.Fatal(err)
	}
	lease, err := h.ReserveArea(ctx, "w", "race", "match")
	if err != nil {
		t.Fatal(err)
	}
	c.add(3 * time.Second)
	if err = h.ClearArea(ctx, lease); err != nil {
		t.Fatal(err)
	}
	if err = h.ActivateArea(ctx, lease, c.Now().UnixMilli()); err != nil {
		t.Fatal(err)
	}
	h.Leave(JoinContext{Token: "u", WorldID: "w", SessionID: snap.SessionID, Epoch: snap.Epoch})
	h.mu.Lock()
	retained := h.rooms["w"] != nil
	h.mu.Unlock()
	if !retained {
		t.Fatal("empty leased room destroyed")
	}
	next, err := h.Join(ctx, JoinContext{Token: "new", WorldID: "w"})
	if err != nil {
		t.Fatal(err)
	}
	j := JoinContext{Token: "new", WorldID: "w", SessionID: next.SessionID, Epoch: next.Epoch}
	states, err := h.ActivitySnapshots(ctx, j)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range states {
		if s.Kind == "area" && s.AreaID == "race" && (s.Incarnation != lease.Incarnation || s.Generation != lease.Generation || s.Payload.Phase != "ACTIVE") {
			t.Fatal("admission lost lease", s)
		}
	}
	if err = h.Submit(ctx, j, Command{V: 2, CommandID: "enter", Seq: 1, WorldID: "w", Type: "player.destination", Payload: []byte(`{"x":44,"y":42}`)}); err == nil {
		t.Fatal("rejoined runtime entered active area")
	}
	h.ReleaseArea(ctx, lease)
	c.add(2 * time.Second)
	h.Leave(j)
	h.mu.Lock()
	closed := h.rooms["w"] == nil
	h.mu.Unlock()
	if !closed {
		t.Fatal("open empty room not reclaimed")
	}
	newer, err := h.ReserveArea(ctx, "w", "race", "new-match")
	if err != nil {
		t.Fatal(err)
	}
	if newer.Incarnation == lease.Incarnation {
		t.Fatal("runtime incarnation reused")
	}
	if h.ReleaseArea(ctx, lease) == nil {
		t.Fatal("old runtime release accepted")
	}
}

func TestAreaLifecycleCacheReclaimedOnlyWithDestroyedRuntime(t *testing.T) {
	h := newTestHub(t)
	ctx := context.Background()
	lease, err := h.ReserveArea(ctx, "leased", "race", "match")
	if err != nil {
		t.Fatal(err)
	}
	h.mu.Lock()
	baseline := len(h.areaStates)
	h.closeEmptyRoom("leased")
	retained := h.rooms["leased"] != nil && len(h.areaStates) == baseline
	h.mu.Unlock()
	if baseline == 0 || !retained {
		t.Fatal("active lease lost runtime or lifecycle cache")
	}
	for i := 0; i < 20; i++ {
		id := fmt.Sprintf("transient-%d", i)
		s, err := h.Join(ctx, JoinContext{Token: "u", WorldID: id})
		if err != nil {
			t.Fatal(err)
		}
		j := JoinContext{Token: "u", WorldID: id, SessionID: s.SessionID, Epoch: s.Epoch}
		states, err := h.ActivitySnapshots(ctx, j)
		if err != nil {
			t.Fatal(err)
		}
		old := states[0].Incarnation
		h.Leave(j)
		h.mu.Lock()
		count := len(h.areaStates)
		gone := h.rooms[id] == nil
		h.mu.Unlock()
		if !gone || count != baseline {
			t.Fatalf("destroyed world %s retained cache: got %d want %d", id, count, baseline)
		}
		s, err = h.Join(ctx, JoinContext{Token: "u", WorldID: id})
		if err != nil {
			t.Fatal(err)
		}
		j.SessionID = s.SessionID
		j.Epoch = s.Epoch
		states, err = h.ActivitySnapshots(ctx, j)
		if err != nil {
			t.Fatal(err)
		}
		if states[0].Incarnation == old {
			t.Fatal("recreated runtime reused incarnation")
		}
		h.Leave(j)
	}
	states, err := h.WorldActivities(ctx, "leased")
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range states {
		if state.Kind == "area" && state.AreaID == "race" && state.Incarnation != lease.Incarnation {
			t.Fatal("other teardown changed active lease")
		}
	}
}
