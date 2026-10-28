//go:build integration

package moderation_test

import (
	"context"
	"patra-world-social/server/internal/application"
	httpapi "patra-world-social/server/internal/http"
	"patra-world-social/server/internal/moderation"
	"patra-world-social/server/internal/testutil"
	"testing"
)

func TestOwnerProtectionPeerMuteAndAudit(t *testing.T) {
	p := testutil.Database(t)
	u, _ := testutil.Users(t, p, 3)
	ctx := context.Background()
	app, e := application.New(p, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true})
	if e != nil {
		t.Fatal(e)
	}
	defer app.Close()
	w, e := app.Worlds.CreateWorld(ctx, u[0], "Autumn")
	if e != nil {
		t.Fatal(e)
	}
	for _, id := range u[1:] {
		inv, e := app.Worlds.IssueInvite(ctx, u[0], w.ID, id)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = app.Worlds.AcceptTargetedInvite(ctx, id, inv.ID); e != nil {
			t.Fatal(e)
		}
		if e = app.Worlds.ChangeRole(ctx, u[0], w.ID, id, "moderator"); e != nil {
			t.Fatal(e)
		}
	}
	if e = app.Moderation.Moderate(ctx, u[1], w.ID, moderation.Action{Kind: "mute", UserID: u[0], Seconds: 900}); e == nil {
		t.Fatal("owner muted")
	}
	if e = app.Moderation.Moderate(ctx, u[1], w.ID, moderation.Action{Kind: "mute", UserID: u[2], Seconds: 86400}); e != nil {
		t.Fatal("peer policy", e)
	}
	var count int
	if e = p.QueryRow(ctx, `SELECT count(*) FROM moderation_audit WHERE actor_id=$1 AND target_id=$2 AND action='mute' AND expires_at>now()`, u[1], u[2]).Scan(&count); e != nil || count != 1 {
		t.Fatal("missing audit", count, e)
	}
	if e = app.Moderation.Moderate(ctx, u[1], w.ID, moderation.Action{Kind: "kick", UserID: u[2]}); e != nil {
		t.Fatal(e)
	}
	if e = p.QueryRow(ctx, `SELECT count(*) FROM moderation_audit WHERE action='kick'`).Scan(&count); e != nil || count != 1 {
		t.Fatal("kick audit", count, e)
	}
}
