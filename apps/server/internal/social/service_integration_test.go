//go:build integration

package social_test

import (
	"context"
	"patra-world-social/server/internal/social"
	"patra-world-social/server/internal/testutil"
	"patra-world-social/server/internal/worlds"
	"testing"
)

func TestFriendLifecycleAndBlockedInvites(t *testing.T) {
	p := testutil.Database(t)
	u, _ := testutil.Users(t, p, 3)
	s := social.New(p)
	ctx := context.Background()
	s.Online = func(string) bool { return true }
	if _, e := s.Search(ctx, u[0], "synthetic"); e == nil {
		t.Fatal("prefix search accepted")
	}
	if e := s.RequestFriend(ctx, u[0], "SYNTHETIC_1"); e != nil {
		t.Fatal(e)
	}
	pending, e := s.List(ctx, u[1])
	if e != nil || len(pending) != 1 || !pending[0].Incoming || pending[0].Online {
		t.Fatalf("pending privacy: %v %v", pending, e)
	}
	if e = s.RespondFriend(ctx, u[2], pending[0].RequestID, true); e == nil {
		t.Fatal("outsider responded")
	}
	if e = s.RespondFriend(ctx, u[1], pending[0].RequestID, false); e != nil {
		t.Fatal(e)
	}
	if e = s.RequestFriend(ctx, u[0], "synthetic_1"); e != nil {
		t.Fatal(e)
	}
	pending, _ = s.List(ctx, u[1])
	if e = s.RespondFriend(ctx, u[1], pending[0].RequestID, true); e != nil {
		t.Fatal(e)
	}
	friends, _ := s.List(ctx, u[0])
	if len(friends) != 1 || !friends[0].Accepted || !friends[0].Online {
		t.Fatal("accepted presence missing")
	}
	if e = s.Remove(ctx, u[0], u[1], false); e != nil {
		t.Fatal(e)
	}
	friends, _ = s.List(ctx, u[0])
	if len(friends) != 0 {
		t.Fatal("unfriend failed")
	}
	if e = s.Remove(ctx, u[1], u[0], true); e != nil {
		t.Fatal(e)
	}
	if e = s.RequestFriend(ctx, u[0], "synthetic_1"); e == nil {
		t.Fatal("block allowed request")
	}
	if _, e = s.Search(ctx, u[0], "synthetic_1"); e == nil {
		t.Fatal("block leaked search")
	}
	w := worlds.New(p)
	world, e := w.CreateWorld(ctx, u[0], "Autumn")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = w.IssueInvite(ctx, u[0], world.ID, u[1]); e == nil {
		t.Fatal("block allowed invite")
	}
	if e = s.Unblock(ctx, u[1], u[0]); e != nil {
		t.Fatal(e)
	}
	inv, e := w.IssueInvite(ctx, u[0], world.ID, u[1])
	if e != nil {
		t.Fatal(e)
	}
	notes, e := w.Notifications(ctx, u[1])
	if e != nil || len(notes) != 1 || notes[0].ID != inv.ID {
		t.Fatal("notification missing", e)
	}
	if _, e = w.AcceptTargetedInvite(ctx, u[2], inv.ID); e == nil {
		t.Fatal("outsider accepted notification")
	}
	if _, e = w.AcceptTargetedInvite(ctx, u[1], inv.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = w.AcceptTargetedInvite(ctx, u[1], inv.ID); e != nil {
		t.Fatal("retry not idempotent", e)
	}
	link, e := w.IssueInvite(ctx, u[0], world.ID, "")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = w.AcceptTargetedInvite(ctx, u[2], link.ID); e == nil {
		t.Fatal("public invite UUID bypassed token")
	}
}
