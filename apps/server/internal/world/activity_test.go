package world

import (
	"context"
	"testing"
	"time"
)

func TestPlayingNeutralizesInputAndReturnsToSafeEntrance(t *testing.T) {
	r, c := testRuntime(t)
	ctx := context.Background()
	r.Join(ctx, "u", "User", 1)
	r.Input(ctx, "u", 1, 1, 1, 0)
	if e := r.Activity(ctx, "u", true, nil); e != nil {
		t.Fatal(e)
	}
	r.Input(ctx, "u", 1, 2, 1, 0)
	c.add(100 * time.Millisecond)
	snap, _ := r.Snapshot(ctx)
	if snap.Avatars[0].X != 32 || snap.Avatars[0].Status != "playing" {
		t.Fatal(snap)
	}
	at := Point{30, 14}
	if e := r.Activity(ctx, "u", false, &at); e != nil {
		t.Fatal(e)
	}
	snap, _ = r.Snapshot(ctx)
	if snap.Avatars[0].X != 30 || snap.Avatars[0].Y != 14 || snap.Avatars[0].Status != "available" {
		t.Fatal(snap)
	}
}

func TestArenaReturnGuardFencesPositionAndActivityTogether(t *testing.T) {
	r, _ := testRuntime(t)
	ctx := context.Background()
	r.Join(ctx, "u", "User", 1)
	r.Activity(ctx, "u", true, nil)
	at := Point{30, 14}
	if e := r.ReturnActivity(ctx, "u", at, func(apply func()) bool { return false }); e != ErrActivitySuperseded {
		t.Fatal(e)
	}
	snap, _ := r.Snapshot(ctx)
	if p := snap.Avatars[0]; p.X != 32 || p.Y != 32 || p.Status != "playing" {
		t.Fatal("rejected return mutated owner state", p)
	}
	if e := r.ReturnActivity(ctx, "u", at, func(apply func()) bool { apply(); return true }); e != nil {
		t.Fatal(e)
	}
	snap, _ = r.Snapshot(ctx)
	if p := snap.Avatars[0]; p.X != 30 || p.Y != 14 || p.Status != "available" {
		t.Fatal("current return not applied", p)
	}
}
