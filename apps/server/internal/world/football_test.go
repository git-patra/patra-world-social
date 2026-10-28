package world

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"
)

func footballState(t *testing.T, r *Runtime) (ActivitySnapshot, map[string]any) {
	t.Helper()
	states, err := r.ActivitySnapshots(context.Background(), "world")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range states {
		if s.Kind == "football" {
			raw, _ := json.Marshal(s.Payload)
			var p struct {
				Ball map[string]any `json:"ball"`
			}
			if err := json.Unmarshal(raw, &p); err != nil {
				t.Fatal(err)
			}
			return s, p.Ball
		}
	}
	t.Fatal("world snapshots omit the single casual football")
	return ActivitySnapshot{}, nil
}

func TestKickRequiresProximityAndCooldown(t *testing.T) {
	r, c := testRuntime(t)
	ctx := context.Background()
	r.Join(ctx, "u", "U", 1)
	s, b := footballState(t, r)
	i := CasualIntent{Incarnation: s.Incarnation, Generation: s.Generation, Action: "kick", DX: 1, DY: 1}
	if r.ApplyCasualIntent(ctx, "u", 1, 1, i) == nil {
		t.Fatal("remote kick admitted")
	}
	r.call(ctx, func() { r.players["u"].X = b["x"].(float64); r.players["u"].Y = b["y"].(float64) })
	if err := r.ApplyCasualIntent(ctx, "u", 1, 1, i); err != nil {
		t.Fatal(err)
	}
	s, b = footballState(t, r)
	if math.Hypot(b["vx"].(float64), b["vy"].(float64)) > 8.000001 {
		t.Fatal("unbounded impulse", b)
	}
	i.Generation = s.Generation
	if r.ApplyCasualIntent(ctx, "u", 1, 2, i) == nil {
		t.Fatal("cooldown bypassed")
	}
	c.add(400 * time.Millisecond)
	s, b = footballState(t, r)
	r.call(ctx, func() { r.players["u"].X = b["x"].(float64); r.players["u"].Y = b["y"].(float64) })
	i.Generation = s.Generation
	if err := r.ApplyCasualIntent(ctx, "u", 1, 2, i); err != nil {
		t.Fatal("cooldown never reopened", err)
	}
	if r.ApplyCasualIntent(ctx, "u", 1, 3, i) == nil {
		t.Fatal("old ball generation reused")
	}
}

func TestCasualBallDisabledWhenPenaltyReserved(t *testing.T) {
	r, c := testRuntime(t)
	ctx := context.Background()
	r.Join(ctx, "u", "U", 1)
	s, b := footballState(t, r)
	r.call(ctx, func() { r.players["u"].X = b["x"].(float64); r.players["u"].Y = b["y"].(float64) })
	i := CasualIntent{Incarnation: s.Incarnation, Generation: s.Generation, Action: "kick", DX: 1}
	if err := r.ApplyCasualIntent(ctx, "u", 1, 1, i); err != nil {
		t.Fatal(err)
	}
	lease, err := r.ReserveArea(ctx, "world", "penalty", "match")
	if err != nil {
		t.Fatal(err)
	}
	s, b = footballState(t, r)
	if b["visible"] != false || b["vx"] != float64(0) || b["vy"] != float64(0) {
		t.Fatal("ball not hidden/reset before clear ACK", b)
	}
	i.Generation = s.Generation
	i.AreaGeneration = lease.Generation
	if r.ApplyCasualIntent(ctx, "u", 1, 2, i) == nil {
		t.Fatal("last-millisecond kick entered reserved pitch")
	}
	r.Join(ctx, "u", "U", 2)
	if r.ApplyCasualIntent(ctx, "u", 2, 1, i) == nil {
		t.Fatal("reconnect kicked in closed pitch")
	}
	c.add(2999 * time.Millisecond)
	if r.ApplyCasualIntent(ctx, "u", 2, 1, i) == nil {
		t.Fatal("kick entered one millisecond before clear ACK")
	}
	c.add(time.Millisecond)
	if err = r.ClearArea(ctx, lease); err != nil {
		t.Fatal(err)
	}
	if err = r.ActivateArea(ctx, lease, c.Now().UnixMilli()); err != nil {
		t.Fatal(err)
	}
	if r.ApplyCasualIntent(ctx, "u", 2, 1, i) == nil {
		t.Fatal("active penalty accepted casual kick")
	}
	r.ReleaseArea(ctx, lease)
	c.add(2 * time.Second)
	s, b = footballState(t, r)
	if b["visible"] != true {
		t.Fatal("spatial release did not restore casual football")
	}
	r.call(ctx, func() { r.players["u"].X = b["x"].(float64); r.players["u"].Y = b["y"].(float64) })
	i.Generation = s.Generation
	if err := r.ApplyCasualIntent(ctx, "u", 2, 1, i); err != nil {
		t.Fatal(err)
	}
	next, err := r.ReserveArea(ctx, "world", "penalty", "rematch")
	if err != nil {
		t.Fatal(err)
	}
	if r.ReleaseArea(ctx, lease) == nil {
		t.Fatal("old reset released rematch")
	}
	i.AreaGeneration = next.Generation
	if r.ApplyCasualIntent(ctx, "u", 2, 2, i) == nil {
		t.Fatal("rematch accepted old kick")
	}
}

func TestFootballFrictionBoundsAndIdleReset(t *testing.T) {
	r, c := testRuntime(t)
	ctx := context.Background()
	r.Join(ctx, "u", "U", 1)
	s, b := footballState(t, r)
	x, y := b["x"].(float64), b["y"].(float64)
	r.call(ctx, func() { r.players["u"].X = x; r.players["u"].Y = y })
	if err := r.ApplyCasualIntent(ctx, "u", 1, 1, CasualIntent{Incarnation: s.Incarnation, Generation: s.Generation, Action: "kick", DX: 1}); err != nil {
		t.Fatal(err)
	}
	speed := 8.
	for n := 0; n < 100; n++ {
		c.add(50 * time.Millisecond)
		_, b = footballState(t, r)
		v := math.Hypot(b["vx"].(float64), b["vy"].(float64))
		if v > speed+.000001 {
			t.Fatal("friction increased speed")
		}
		speed = v
		if b["x"].(float64) < 24.15 || b["x"].(float64) > 31.85 || b["y"].(float64) < 40.15 || b["y"].(float64) > 50.85 {
			t.Fatal("ball crossed swept bounds", b)
		}
	}
	c.add(16 * time.Second)
	_, b = footballState(t, r)
	if b["x"] != x || b["y"] != y || b["vx"] != float64(0) {
		t.Fatal("idle football did not reset", b)
	}
}

func TestKickFencesEpochSequenceIncarnationAndRider(t *testing.T) {
	r, _ := testRuntime(t)
	ctx := context.Background()
	r.Join(ctx, "u", "U", 1)
	s, b := footballState(t, r)
	r.call(ctx, func() { r.players["u"].X = b["x"].(float64); r.players["u"].Y = b["y"].(float64) })
	good := CasualIntent{Incarnation: s.Incarnation, Generation: s.Generation, Action: "kick", DY: -1}
	for _, bad := range []struct {
		epoch, seq uint64
		i          CasualIntent
	}{
		{2, 1, good},
		{1, 0, good},
		{1, 1, func() CasualIntent { i := good; i.Incarnation = "retired"; return i }()},
		{1, 1, func() CasualIntent { i := good; i.AreaGeneration = 50; return i }()},
		{1, 1, func() CasualIntent { i := good; i.DX = math.NaN(); return i }()},
	} {
		if r.ApplyCasualIntent(ctx, "u", bad.epoch, bad.seq, bad.i) == nil {
			t.Fatal("kick bypassed authority fence", bad)
		}
	}
	if err := r.ApplyCasualIntent(ctx, "u", 1, 1, good); err != nil {
		t.Fatal("rejections consumed valid sequence", err)
	}
	r.Join(ctx, "u", "U", 2)
	s, _ = footballState(t, r)
	good.Generation = s.Generation
	if r.ApplyCasualIntent(ctx, "u", 2, 1, good) == nil {
		t.Fatal("reconnect bypassed per-user cooldown")
	}
	r.call(ctx, func() { r.vehicles[0].RiderID = "u"; r.players["u"].lastFootballKick = time.Time{} })
	if r.ApplyCasualIntent(ctx, "u", 2, 1, good) == nil {
		t.Fatal("rider acquired overlapping kick authority")
	}
}

func TestFootballCatchUpAndEscapedBallReset(t *testing.T) {
	r, c := testRuntime(t)
	ctx := context.Background()
	r.Join(ctx, "u", "U", 1)
	s, b := footballState(t, r)
	r.call(ctx, func() { r.players["u"].X = b["x"].(float64); r.players["u"].Y = b["y"].(float64) })
	if err := r.ApplyCasualIntent(ctx, "u", 1, 1, CasualIntent{Incarnation: s.Incarnation, Generation: s.Generation, Action: "kick", DX: 1}); err != nil {
		t.Fatal(err)
	}
	c.add(250 * time.Millisecond)
	_, b = footballState(t, r)
	if b["x"].(float64) > 29.3757 {
		t.Fatal("more than four football catch-up steps", b)
	}
	r.call(ctx, func() { r.football.X = 100 })
	c.add(50 * time.Millisecond)
	_, b = footballState(t, r)
	if b["x"] != float64(28) || b["y"] != float64(46) || b["vx"] != float64(0) {
		t.Fatal("escaped ball did not reset", b)
	}
}
