package world

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"
)

func TestClearingBlocksExistingPathsAndLastTickIntent(t *testing.T) {
	r, c := testRuntime(t)
	ctx := context.Background()
	r.Join(ctx, "u", "User", 1)
	r.call(ctx, func() { r.players["u"].X = 36; r.players["u"].Y = 42 })
	if err := r.Destination(ctx, "u", 1, 1, 44, 42); err != nil {
		t.Fatal(err)
	}
	lease, err := r.ReserveArea(ctx, "world", "race", "match")
	if err != nil {
		t.Fatal(err)
	}
	r.Input(ctx, "u", 1, 2, 1, 0)
	c.add(250 * time.Millisecond)
	s, _ := r.Snapshot(ctx)
	if s.Avatars[0].X >= 36.5 {
		t.Fatal("last intent entered clearing area")
	}
	if err = r.ClearArea(ctx, lease); err != ErrAreaClearing {
		t.Fatalf("premature clear ACK %v", err)
	}
	c.add(3 * time.Second)
	if err = r.ClearArea(ctx, lease); err != nil {
		t.Fatal(err)
	}
}
func TestOldGenerationCannotReleaseRematch(t *testing.T) {
	r, c := testRuntime(t)
	ctx := context.Background()
	a, _ := r.ReserveArea(ctx, "world", "race", "old")
	c.add(3 * time.Second)
	r.ClearArea(ctx, a)
	r.ReleaseArea(ctx, a)
	c.add(2 * time.Second)
	r.Snapshot(ctx)
	b, err := r.ReserveArea(ctx, "world", "race", "new")
	if err != nil {
		t.Fatal(err)
	}
	if b.Generation <= a.Generation {
		t.Fatal("generation reused")
	}
	if err = r.ReleaseArea(ctx, a); err != ErrActivitySuperseded {
		t.Fatalf("stale release %v", err)
	}
}
func TestActivityGeometryAndMirrors(t *testing.T) {
	m, err := LoadMap(MapAsset)
	if err != nil {
		t.Fatal(err)
	}
	c, err := LoadActivities(ActivitiesAsset, m)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Areas) != 4 {
		t.Fatal("area count")
	}
	raw, err := os.ReadFile("../../../../contracts/activities.v1.json")
	if err != nil || !bytes.Equal(raw, ActivitiesAsset) {
		t.Fatal("activity mirror drift")
	}
	for _, a := range c.Areas {
		if a.ID == "blaster" {
			for x := a.Bounds.X; x < a.Bounds.X+a.Bounds.Width; x++ {
				for y := a.Bounds.Y; y < a.Bounds.Y+a.Bounds.Height; y++ {
					if m.Blocked(Point{x, y}) {
						t.Fatal("blaster pad obstacle")
					}
				}
			}
		}
	}
}

func TestAreaClosureRevalidatesEveryPathAndMovementStep(t *testing.T) {
	r, c := testRuntime(t)
	ctx := context.Background()
	r.Join(ctx, "u", "User", 1)
	r.call(ctx, func() { r.players["u"].X = 36; r.players["u"].Y = 42 })
	if e := r.Destination(ctx, "u", 1, 1, 44, 42); e != nil {
		t.Fatal(e)
	}
	lease, e := r.ReserveArea(ctx, "world", "race", "match")
	if e != nil {
		t.Fatal(e)
	}
	c.add(2 * time.Second)
	s, _ := r.Snapshot(ctx)
	if s.Avatars[0].X != 36 {
		t.Fatal("existing path survived closure")
	}
	if r.Destination(ctx, "u", 1, 2, 44, 42) == nil {
		t.Fatal("new destination crossed mask")
	}
	c.add(time.Second)
	if e = r.ClearArea(ctx, lease); e != nil {
		t.Fatal(e)
	}
	if e = r.ActivateArea(ctx, lease, c.Now().UnixMilli()); e != nil {
		t.Fatal(e)
	}
	for seq := uint64(3); seq < 23; seq++ {
		r.Input(ctx, "u", 1, seq, 1, 0)
		c.add(250 * time.Millisecond)
		s, _ = r.Snapshot(ctx)
		if s.Avatars[0].X >= 36.5 {
			t.Fatal("continuous input crossed active mask")
		}
	}
	r.Join(ctx, "u", "User", 2)
	r.Input(ctx, "u", 1, 99, 1, 0)
	c.add(250 * time.Millisecond)
	s, _ = r.Snapshot(ctx)
	if s.Avatars[0].X >= 36.5 {
		t.Fatal("stale epoch moved into area")
	}
}
