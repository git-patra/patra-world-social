package realtime

import (
	"context"
	"patra-world-social/server/internal/world"
	"strings"
	"testing"
	"time"
)

func TestActivitySnapshotCacheKeepsIdentityGenerationAndPayload(t *testing.T) {
	cache := map[string][]byte{}
	a := world.ActivitySnapshot{WorldID: "w", AreaID: "race", Kind: "area", Incarnation: "one", Generation: 1, Tick: 1, ServerTimeMs: 1, Payload: world.AreaPayload{Phase: "OPEN"}}
	if !changedActivity(cache, a) {
		t.Fatal("initial suppressed")
	}
	a.Tick++
	a.ServerTimeMs++
	if changedActivity(cache, a) {
		t.Fatal("clock-only retransmit")
	}
	a.Generation++
	if !changedActivity(cache, a) {
		t.Fatal("generation suppressed")
	}
	a.Incarnation = "two"
	a.Generation = 1
	if !changedActivity(cache, a) {
		t.Fatal("new runtime suppressed")
	}
	a.Payload.Phase = "CLEARING"
	if !changedActivity(cache, a) {
		t.Fatal("lifecycle suppressed")
	}
	a.Kind = "football"
	ball := world.Football{X: 28, Y: 46, Visible: true}
	a.Payload = world.AreaPayload{Ball: &ball}
	if !changedActivity(cache, a) {
		t.Fatal("new kind")
	}
	ball.X++
	if !changedActivity(cache, a) {
		t.Fatal("moving ball suppressed")
	}
}
func TestGameSnapshotInterestRateNewMatchAndReliableEvents(t *testing.T) {
	h := newTestHub(t)
	ctx := context.Background()
	joins := map[string]JoinContext{}
	for _, user := range []string{"player", "near", "explicit", "far", "other"} {
		w := "a"
		if user == "other" {
			w = "b"
		}
		s, e := h.Join(ctx, JoinContext{Token: user, WorldID: w})
		if e != nil {
			t.Fatal(e)
		}
		joins[user] = JoinContext{Token: user, WorldID: w, SessionID: s.SessionID, Epoch: s.Epoch}
	}
	area, _ := world.ActivityArea("pingpong")
	strip := area.SpectatorStrips[0]
	at := world.Point{X: strip.X, Y: strip.Y}
	if e := h.SetActivity(ctx, "a", "near", false, &at); e != nil {
		t.Fatal(e)
	}
	publish := func(match string) {
		h.PublishGameSnapshot(ctx, "a", "pingpong", match, []string{"player"}, []string{"explicit"}, "pingpong.snapshot", map[string]string{"matchId": match})
	}
	publish("one")
	publish("one")
	for user, want := range map[string]int{"player": 2, "near": 1, "explicit": 1, "far": 0, "other": 0} {
		if got := len(h.Events(joins[user])); got != want {
			t.Fatalf("%s got %d want %d", user, got, want)
		}
	}
	publish("two")
	if len(h.Events(joins["near"])) != 2 {
		t.Fatal("new match must immediately establish full state")
	}
	h.Publish("a", "match.result_status", map[string]string{"status": "completed"})
	if len(h.Events(joins["far"])) != 1 || len(h.Events(joins["other"])) != 0 {
		t.Fatal("reliable event filtered or cross-world leaked")
	}
	at = world.Point{X: 32, Y: 32}
	h.SetActivity(ctx, "a", "near", false, &at)
	publish("three")
	if len(h.Events(joins["near"])) != 3 {
		t.Fatal("departed spectator still receives visual state")
	}
	h.mu.Lock()
	for _, s := range h.seats {
		if len(s.gameSnapshots) > 1 {
			t.Fatal("per-match cache growth")
		}
	}
	h.mu.Unlock()
}

func TestVisualBudgetSharesAcrossAreasAndNewMatchBypassesDebt(t *testing.T) {
	h := newTestHub(t)
	ctx := context.Background()
	s, e := h.Join(ctx, JoinContext{Token: "watch", WorldID: "a"})
	if e != nil {
		t.Fatal(e)
	}
	j := JoinContext{Token: "watch", WorldID: "a", SessionID: s.SessionID, Epoch: s.Epoch}
	// Drive diagnostic seat timestamps backwards without changing the real owner
	// clock: equal 250ms refill windows for all competing visual streams.
	payload := map[string]string{"data": strings.Repeat("x", 8000)}
	counts := map[string]int{}
	for n := 0; n < 32; n++ {
		h.mu.Lock()
		for key, state := range h.seats[j.SessionID].gameSnapshots {
			state.at = state.at.Add(-250 * time.Millisecond)
			state.refill = state.refill.Add(-250 * time.Millisecond)
			h.seats[j.SessionID].gameSnapshots[key] = state
		}
		h.mu.Unlock()
		for _, area := range []string{"pingpong", "blaster"} {
			h.PublishGameSnapshot(ctx, "a", area, "one", nil, []string{"watch"}, area+".snapshot", payload)
		}
		for len(h.Events(j)) > 0 {
			counts[(<-h.Events(j)).Type]++
		}
	}
	if counts["pingpong.snapshot"] < 3 || counts["blaster.snapshot"] < 3 {
		t.Fatal("visual stream starved", counts)
	}
	h.mu.Lock()
	seat := h.seats[j.SessionID]
	state := seat.gameSnapshots["blaster"]
	state.tokens = -100000
	seat.gameSnapshots["blaster"] = state
	h.mu.Unlock()
	h.PublishGameSnapshot(ctx, "a", "blaster", "new-generation", nil, []string{"watch"}, "blaster.snapshot", payload)
	if len(h.Events(j)) != 1 {
		t.Fatal("full new match state blocked by budget debt")
	}
}

func TestCasualSnapshotOwnerSelection(t *testing.T) {
	vehicles := [4]world.Vehicle{{RiderID: "rider"}}
	a := world.ActivitySnapshot{Payload: world.AreaPayload{Vehicles: &vehicles}}
	if !ownsActivity(a, "rider") || ownsActivity(a, "watcher") {
		t.Fatal("ride owner cadence")
	}
	a.Payload = world.AreaPayload{Claw: &world.Claw{OperatorID: "operator"}}
	if !ownsActivity(a, "operator") || ownsActivity(a, "watcher") {
		t.Fatal("claw owner cadence")
	}
	a.Payload = world.AreaPayload{Fishing: &world.Fishing{OwnerID: "fisher"}}
	if !ownsActivity(a, "fisher") || ownsActivity(a, "watcher") {
		t.Fatal("fishing owner cadence")
	}
}
