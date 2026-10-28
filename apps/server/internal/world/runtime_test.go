package world

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time      { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *fakeClock) add(d time.Duration) { c.mu.Lock(); c.now = c.now.Add(d); c.mu.Unlock() }
func testRuntime(t *testing.T) (*Runtime, *fakeClock) {
	t.Helper()
	c := &fakeClock{now: time.Unix(1000, 0)}
	r := NewRuntime(MapDefinition{Width: 64, Height: 64, Spawn: Point{32, 32}, Obstacles: []Point{{34, 32}}}, c)
	t.Cleanup(r.Close)
	return r, c
}
func TestAuthoritativeExpiryAndSequence(t *testing.T) {
	r, c := testRuntime(t)
	ctx := context.Background()
	r.Join(ctx, "u", "User", 1)
	if err := r.Input(ctx, "u", 1, 1, 1, 1); err != nil {
		t.Fatal(err)
	}
	c.add(250 * time.Millisecond)
	s, _ := r.Snapshot(ctx)
	p := s.Avatars[0]
	if math.Abs(math.Hypot(p.X-32, p.Y-32)-1.125) > 1e-8 {
		t.Fatalf("normalized distance: %+v", p)
	}
	c.add(time.Second)
	s, _ = r.Snapshot(ctx)
	if s.Avatars[0].X != p.X {
		t.Fatal("stuck input")
	}
	r.Input(ctx, "u", 1, 1, -1, 0)
	c.add(50 * time.Millisecond)
	s, _ = r.Snapshot(ctx)
	if s.Avatars[0].X != p.X {
		t.Fatal("stale sequence moved")
	}
	r.Join(ctx, "u", "User", 2)
	r.Input(ctx, "u", 1, 999, 1, 0)
	c.add(50 * time.Millisecond)
	s, _ = r.Snapshot(ctx)
	if s.Avatars[0].X != p.X {
		t.Fatal("old epoch moved")
	}
}
func TestCollisionDestinationAndInvalidInput(t *testing.T) {
	r, c := testRuntime(t)
	ctx := context.Background()
	r.Join(ctx, "u", "User", 1)
	for _, n := range []float64{2, math.NaN(), math.Inf(1)} {
		if r.Input(ctx, "u", 1, 1, n, 0) == nil {
			t.Fatal("accepted invalid input")
		}
	}
	if r.Destination(ctx, "u", 1, 1, 34, 32) == nil {
		t.Fatal("accepted blocked goal")
	}
	if err := r.Destination(ctx, "u", 1, 2, 35, 32); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 40; i++ {
		c.add(50 * time.Millisecond)
		s, _ := r.Snapshot(ctx)
		p := s.Avatars[0]
		if math.Round(p.X) == 34 && math.Round(p.Y) == 32 {
			t.Fatal("crossed obstacle")
		}
	}
	s, _ := r.Snapshot(ctx)
	if s.Avatars[0].X != 35 || s.Avatars[0].Y != 32 {
		t.Fatalf("path failed %+v", s)
	}
}
