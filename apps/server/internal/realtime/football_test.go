package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"patra-world-social/server/internal/world"
	"testing"
	"time"
)

func TestFootballWireIsExact(t *testing.T) {
	i := world.CasualIntent{Incarnation: "runtime", Generation: 1, Action: "kick", DY: -1}
	payload, _ := json.Marshal(i)
	c := Command{V: 2, CommandID: "kick", WorldID: "world", Seq: 1, Type: "activity.kick", Payload: payload}
	raw, _ := json.Marshal(c)
	if _, err := ValidateCommand(raw); err != nil {
		t.Fatal("valid kick rejected", err)
	}
	var p map[string]any
	json.Unmarshal(payload, &p)
	for _, key := range []string{"x", "score", "vehicleId"} {
		p[key] = 1
		c.Payload, _ = json.Marshal(p)
		raw, _ = json.Marshal(c)
		if _, err := ValidateCommand(raw); err == nil {
			t.Fatal("extra kick field accepted", key)
		}
		delete(p, key)
	}
	p["dy"] = 0
	c.Payload, _ = json.Marshal(p)
	raw, _ = json.Marshal(c)
	if _, err := ValidateCommand(raw); err == nil {
		t.Fatal("zero impulse accepted")
	}
}

func TestFootballActualHubRejectsBusyAndReservedPitch(t *testing.T) {
	h := newTestHub(t)
	ctx := context.Background()
	s, err := h.Join(ctx, JoinContext{Token: "u", WorldID: "w"})
	if err != nil {
		t.Fatal(err)
	}
	j := JoinContext{Token: "u", WorldID: "w", SessionID: s.SessionID, Epoch: s.Epoch}
	at := world.Point{X: 28, Y: 46}
	h.SetActivity(ctx, "w", "u", false, &at)
	states, _ := h.ActivitySnapshots(ctx, j)
	var i world.CasualIntent
	for _, s := range states {
		if s.Kind == "football" {
			i = world.CasualIntent{Incarnation: s.Incarnation, Generation: s.Generation, Action: "kick", DY: -1}
		}
	}
	payload, _ := json.Marshal(i)
	command := Command{V: 2, CommandID: "kick", WorldID: "w", Seq: 1, Type: "activity.kick", Payload: payload}
	if _, err = h.Execute(ctx, j, command); err != nil {
		t.Fatal("actual hub kick failed", err)
	}
	h.GameBusy = func(string) bool { return true }
	command.Seq++
	if _, err = h.Execute(ctx, j, command); err == nil {
		t.Fatal("durable busy kick accepted")
	}
	h.GameBusy = nil
	if _, err = h.ReserveArea(ctx, "w", "penalty", "match"); err != nil {
		t.Fatal(err)
	}
	command.Seq++
	if _, err = h.Execute(ctx, j, command); err == nil {
		t.Fatal("reserved pitch accepted kick")
	}
}

func TestFootballKicksPreserveActiveDestination(t *testing.T) {
	for _, kind := range []string{"accepted", "cooldown", "stale_generation"} {
		t.Run(kind, func(t *testing.T) {
			clock := &hubClock{now: time.Unix(1000, 0)}
			h := NewHub(&testAuth{revoked: map[string]bool{}}, testMembers{}, world.MapDefinition{Width: 64, Height: 64, Spawn: world.Point{X: 28, Y: 46}}, clock)
			defer h.Close()
			h.GameBusy = func(string) bool { return false }
			ctx := context.Background()
			seat, err := h.Join(ctx, JoinContext{Token: "u", WorldID: "w"})
			if err != nil {
				t.Fatal(err)
			}
			j := JoinContext{Token: "u", WorldID: "w", SessionID: seat.SessionID, Epoch: seat.Epoch}
			states, err := h.ActivitySnapshots(ctx, j)
			if err != nil {
				t.Fatal(err)
			}
			var kick world.CasualIntent
			for _, s := range states {
				if s.Kind == "football" {
					kick = world.CasualIntent{Incarnation: s.Incarnation, Generation: s.Generation, Action: "kick", DY: -1}
				}
			}
			seq := uint64(0)
			execute := func(kind string, payload any) error {
				seq++
				raw, err := json.Marshal(payload)
				if err != nil {
					t.Fatal(err)
				}
				_, err = h.Execute(ctx, j, Command{V: 2, CommandID: fmt.Sprint(seq), WorldID: "w", Seq: seq, Type: kind, Payload: raw})
				return err
			}
			if kind != "accepted" {
				if err = execute("activity.kick", kick); err != nil {
					t.Fatal("initial kick", err)
				}
				if kind == "cooldown" {
					kick.Generation++
				}
			}
			if err = execute("player.destination", map[string]int{"x": 31, "y": 46}); err != nil {
				t.Fatal(err)
			}
			clock.add(50 * time.Millisecond)
			before, err := h.Snapshot(ctx, j)
			if err != nil {
				t.Fatal(err)
			}
			err = execute("activity.kick", kick)
			if kind == "accepted" && err != nil {
				t.Fatal("valid kick", err)
			}
			if kind == "cooldown" && !errors.Is(err, world.ErrMovement) {
				t.Fatal("kick did not reject at cooldown", err)
			}
			if kind == "stale_generation" && !errors.Is(err, world.ErrActivitySuperseded) {
				t.Fatal("kick did not reject at generation fence", err)
			}
			clock.add(200 * time.Millisecond)
			after, err := h.Snapshot(ctx, j)
			if err != nil {
				t.Fatal(err)
			}
			if after.Avatars[0].X <= before.Avatars[0].X {
				t.Fatalf("%s kick canceled destination: before=%+v after=%+v", kind, before.Avatars[0], after.Avatars[0])
			}
		})
	}
}
