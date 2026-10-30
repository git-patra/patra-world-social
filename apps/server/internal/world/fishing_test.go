package world

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func fishingState(t *testing.T, r *Runtime, id string) (ActivitySnapshot, map[string]any) {
	t.Helper()
	ss, e := r.ActivitySnapshots(context.Background(), "world")
	if e != nil {
		t.Fatal(e)
	}
	for _, s := range ss {
		if s.Kind == "fishing" && s.AreaID == id {
			raw, _ := json.Marshal(s.Payload)
			var p map[string]map[string]any
			json.Unmarshal(raw, &p)
			return s, p["fishing"]
		}
	}
	t.Fatal("missing fishing spot", id)
	return ActivitySnapshot{}, nil
}
func fishIntent(s ActivitySnapshot, action string) CasualIntent {
	var i CasualIntent
	raw, _ := json.Marshal(map[string]any{"incarnation": s.Incarnation, "generation": s.Generation, "spotId": s.AreaID, "areaGeneration": 0, "action": action})
	json.Unmarshal(raw, &i)
	return i
}
func fishUser(t *testing.T, r *Runtime, id string, x, y float64) {
	r.Join(context.Background(), id, id, 1)
	r.call(context.Background(), func() { r.players[id].X = x; r.players[id].Y = y })
}
func TestFishingFourSpotsOneOwnerEach(t *testing.T) {
	r, _ := testRuntime(t)
	ctx := context.Background()
	for n := 0; n < 4; n++ {
		id := fmt.Sprint("fishing-", n)
		u := fmt.Sprint("u", n)
		x := 53.
		if n == 3 {
			x = 54
		}
		if r.m.Blocked(Point{int(x), 30 + n}) || !r.m.Reachable(r.m.Spawn, Point{int(x), 30 + n}) {
			t.Fatal("unreachable standing tile", n)
		}
		fishUser(t, r, u, x, float64(30+n))
		s, _ := fishingState(t, r, id)
		if e := r.ApplyCasualIntent(ctx, u, 1, 1, fishIntent(s, "fishing.cast")); e != nil {
			t.Fatal(e)
		}
		fishUser(t, r, "other", x, float64(30+n))
		s, p := fishingState(t, r, id)
		if p["ownerId"] != u || p["phase"] != "waiting" {
			t.Fatal(p)
		}
		if r.ApplyCasualIntent(ctx, "other", 1, 1, fishIntent(s, "fishing.cast")) == nil {
			t.Fatal("second owner")
		}
	}
}
func TestFishingCastRequiresRiverProximity(t *testing.T) {
	r, _ := testRuntime(t)
	fishUser(t, r, "u", 32, 32)
	s, _ := fishingState(t, r, "fishing-0")
	if r.ApplyCasualIntent(context.Background(), "u", 1, 1, fishIntent(s, "fishing.cast")) == nil {
		t.Fatal("remote cast")
	}
}
func TestFishingMovementOrLeaveCancelsCast(t *testing.T) {
	for _, event := range []string{"input", "destination", "leave", "idle", "remove", "takeover", "match"} {
		t.Run(event, func(t *testing.T) {
			r, _ := testRuntime(t)
			ctx := context.Background()
			fishUser(t, r, "u", 53, 30)
			s, _ := fishingState(t, r, "fishing-0")
			if e := r.ApplyCasualIntent(ctx, "u", 1, 1, fishIntent(s, "fishing.cast")); e != nil {
				t.Fatal(e)
			}
			s, _ = fishingState(t, r, "fishing-0")
			switch event {
			case "input":
				r.Input(ctx, "u", 1, 2, -1, 0)
			case "destination":
				r.Destination(ctx, "u", 1, 2, 52, 30)
			case "leave":
				r.ApplyCasualIntent(ctx, "u", 1, 2, fishIntent(s, "fishing.leave"))
			case "idle":
				r.Idle(ctx, "u")
			case "remove":
				r.Remove(ctx, "u")
			case "takeover":
				r.Join(ctx, "u", "U", 2)
			case "match":
				r.Activity(ctx, "u", true, nil)
			}
			next, p := fishingState(t, r, "fishing-0")
			if p["ownerId"] != "" || next.Generation <= s.Generation {
				t.Fatal("retained cast", p)
			}
			if r.ApplyCasualIntent(ctx, "u", 1, 3, fishIntent(s, "fishing.reel")) == nil {
				t.Fatal("stale cast revived")
			}
		})
	}
}
func TestFishingBoundedBiteAndTransientCatch(t *testing.T) {
	r, c := testRuntime(t)
	ctx := context.Background()
	fishUser(t, r, "u", 53, 30)
	seq := uint64(0)
	for n := 0; n < 40; n++ {
		s, _ := fishingState(t, r, "fishing-0")
		seq++
		if e := r.ApplyCasualIntent(ctx, "u", 1, seq, fishIntent(s, "fishing.cast")); e != nil {
			t.Fatal(e)
		}
		_, p := fishingState(t, r, "fishing-0")
		bite := int64(p["biteAt"].(float64))
		wait := bite - c.Now().UnixMilli()
		if wait < 1500 || wait > 3500 {
			t.Fatal("unbounded wait", wait)
		}
		c.add(time.Duration(wait) * time.Millisecond)
		s, p = fishingState(t, r, "fishing-0")
		if p["phase"] != "bite" {
			t.Fatal(p)
		}
		seq++
		if e := r.ApplyCasualIntent(ctx, "u", 1, seq, fishIntent(s, "fishing.reel")); e != nil {
			t.Fatal(e)
		}
		s, p = fishingState(t, r, "fishing-0")
		if p["phase"] != "result" || p["catchId"] == "" {
			t.Fatal(p)
		}
		seq++
		if r.ApplyCasualIntent(ctx, "u", 1, seq, fishIntent(s, "fishing.reel")) == nil {
			t.Fatal("duplicate reel")
		}
		c.add(3 * time.Second)
		_, p = fishingState(t, r, "fishing-0")
		if p["phase"] != "idle" || p["catchId"] != "" {
			t.Fatal("persisted cosmetic catch", p)
		}
	}
}
