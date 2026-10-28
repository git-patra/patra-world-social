//go:build integration

package auth_test

import (
	"context"
	"errors"
	"patra-world-social/server/internal/auth"
	"patra-world-social/server/internal/db"
	"patra-world-social/server/internal/leaderboard"
	"patra-world-social/server/internal/testutil"
	"patra-world-social/server/internal/worlds"
	"strings"
	"testing"
)

func TestAccountDeletionAndCredentials(t *testing.T) {
	p := testutil.Database(t)
	ctx := context.Background()
	s := auth.New(p)
	s.RemoveAccount = leaderboard.RemoveAccount
	u, code, e := s.Register(ctx, "delete_me", "long synthetic password")
	if e != nil {
		t.Fatal(e)
	}
	session, e := s.Authenticate(ctx, u.Username, "long synthetic password")
	if e != nil {
		t.Fatal(e)
	}
	w, e := worlds.New(p).CreateWorld(ctx, u.ID, "Synthetic deletion")
	if e != nil {
		t.Fatal(e)
	}
	if e = s.DeleteAccount(ctx, u.ID, u.Username); !errors.Is(e, auth.ErrOwnsWorld) {
		t.Fatalf("owner prerequisite: %v", e)
	}
	if e = worlds.New(p).ArchiveWorld(ctx, u.ID, w.ID); e != nil {
		t.Fatal(e)
	}
	revoked := false
	s.Revoked = func(context.Context, string) { revoked = true }
	if e = s.DeleteAccount(ctx, u.ID, "wrong"); !errors.Is(e, auth.ErrInput) {
		t.Fatalf("confirmation: %v", e)
	}
	if e = s.DeleteAccount(ctx, u.ID, u.Username); e != nil {
		t.Fatal(e)
	}
	if !revoked {
		t.Fatal("missing revocation")
	}
	if _, e = s.RequireSession(ctx, session.Token); !errors.Is(e, auth.ErrInvalid) {
		t.Fatalf("session survived: %v", e)
	}
	if _, e = s.Authenticate(ctx, u.Username, "long synthetic password"); !errors.Is(e, auth.ErrInvalid) {
		t.Fatalf("password survived: %v", e)
	}
	if _, e = s.Recover(ctx, u.Username, string(code), "another long password"); !errors.Is(e, auth.ErrInvalid) {
		t.Fatalf("recovery survived: %v", e)
	}
	var erased bool
	if e = p.QueryRow(ctx, `SELECT deleted_at IS NOT NULL AND password_hash='' AND octet_length(recovery_hash)=0 AND display_name='Akun dihapus' FROM users WHERE id=$1`, u.ID).Scan(&erased); e != nil || !erased {
		t.Fatalf("tombstone: %v", e)
	}
	if _, e = worlds.New(p).CreateWorld(ctx, u.ID, "No resurrection"); e == nil {
		t.Fatal("deleted owner created world")
	}
}
func TestAccountSettingsPasswordAndRecovery(t *testing.T) {
	p := testutil.Database(t)
	ctx := context.Background()
	s := auth.New(p)
	u, oldCode, e := s.Register(ctx, "settings_user", "long synthetic password")
	if e != nil {
		t.Fatal(e)
	}
	session, e := s.Authenticate(ctx, u.Username, "long synthetic password")
	if e != nil {
		t.Fatal(e)
	}
	profile := auth.Profile{DisplayName: "Autumn", Avatar: "pumpkin", Settings: auth.Settings{Audio: false, Controls: "touch", ReducedMotion: true}}
	if _, e = s.UpdateProfile(ctx, u.ID, profile); e != nil {
		t.Fatal(e)
	}
	if _, e = s.RotateRecovery(ctx, u.ID, "wrong password"); !errors.Is(e, auth.ErrInvalid) {
		t.Fatal(e)
	}
	next, e := s.RotateRecovery(ctx, u.ID, "long synthetic password")
	if e != nil || next == oldCode {
		t.Fatalf("rotate: %v", e)
	}
	if _, e = s.Recover(ctx, u.Username, string(oldCode), "new synthetic password"); !errors.Is(e, auth.ErrInvalid) {
		t.Fatal("old recovery remains valid")
	}
	if e = s.ChangePassword(ctx, u.ID, "long synthetic password", "new synthetic password"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.RequireSession(ctx, session.Token); !errors.Is(e, auth.ErrInvalid) {
		t.Fatal("old session survives password change")
	}
	if _, e = s.Authenticate(ctx, u.Username, "new synthetic password"); e != nil {
		t.Fatal(e)
	}
	got, e := s.Profile(ctx, u.ID)
	if e != nil || got != profile {
		t.Fatalf("profile: %+v %v", got, e)
	}
}

func TestDeletionCannotBeBlockedByClaimedLegacyTombstoneUsername(t *testing.T) {
	p := testutil.Database(t)
	s := auth.New(p)
	s.RemoveAccount = leaderboard.RemoveAccount
	ctx := context.Background()
	u, _, e := s.Register(ctx, "collision_victim", "long synthetic password")
	if e != nil {
		t.Fatal(e)
	}
	reserved := "deleted_" + strings.ReplaceAll(u.ID, "-", "")[:16]
	other, _, e := s.Register(ctx, reserved, "other synthetic password")
	if e != nil {
		t.Fatal(e)
	}
	if e = s.DeleteAccount(ctx, u.ID, u.Username); e != nil {
		t.Fatalf("public name blocked deletion: %v", e)
	}
	var username string
	if e = p.QueryRow(ctx, `SELECT username FROM users WHERE id=$1`, u.ID).Scan(&username); e != nil {
		t.Fatal(e)
	}
	if username != "~deleted:"+u.ID {
		t.Fatal("nonreserved tombstone", username)
	}
	if _, _, e = s.Register(ctx, username, "another synthetic password"); !errors.Is(e, auth.ErrInput) {
		t.Fatal("operational namespace publicly claimable", e)
	}
	if _, e = s.Authenticate(ctx, other.Username, "other synthetic password"); e != nil {
		t.Fatal("unrelated claimant changed", e)
	}
}

func TestReservedTombstoneMigrationNormalizesPreviouslyDeletedRows(t *testing.T) {
	p := testutil.Database(t)
	u, _ := testutil.Users(t, p, 2)
	ctx := context.Background()
	_, e := p.Exec(ctx, `ALTER TABLE users DROP CONSTRAINT users_username_check; ALTER TABLE users ADD CONSTRAINT users_username_check CHECK(username ~ '^[a-z0-9_]{3,24}$'); DELETE FROM schema_migrations WHERE version='012_reserved_tombstones.sql'`)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.Exec(ctx, `UPDATE users SET deleted_at=now(),username='deleted_legacy' WHERE id=$1`, u[0]); e != nil {
		t.Fatal(e)
	}
	if e = db.ApplyMigrations(ctx, p); e != nil {
		t.Fatal(e)
	}
	var name string
	if e = p.QueryRow(ctx, `SELECT username FROM users WHERE id=$1`, u[0]).Scan(&name); e != nil || name != "~deleted:"+u[0] {
		t.Fatal(name, e)
	}
	if e = p.QueryRow(ctx, `SELECT username FROM users WHERE id=$1`, u[1]).Scan(&name); e != nil || name != "synthetic_1" {
		t.Fatal("migration modified live account", name, e)
	}
}

func TestMotionPreservesConcurrentProfileAndRejectsDeletedAccount(t *testing.T) {
	p := testutil.Database(t)
	ctx := context.Background()
	s := auth.New(p)
	s.RemoveAccount = leaderboard.RemoveAccount
	u, _, e := s.Register(ctx, "motion_concurrency", "long synthetic password")
	if e != nil {
		t.Fatal(e)
	}
	tx, e := p.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, `UPDATE users SET display_name='Concurrent edit',avatar='moon',settings='{"audio":false,"controls":"touch","reducedMotion":false}' WHERE id=$1`, u.ID); e != nil {
		t.Fatal(e)
	}
	var pid int
	if e = tx.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&pid); e != nil {
		t.Fatal(e)
	}
	type result struct {
		p auth.Profile
		e error
	}
	out := make(chan result, 1)
	go func() { p, e := s.UpdateMotion(ctx, u.ID, true); out <- result{p, e} }()
	testutil.BlockedPID(t, p, pid)
	if e = tx.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	got := <-out
	expected := auth.Profile{DisplayName: "Concurrent edit", Avatar: "moon", Settings: auth.Settings{Audio: false, Controls: "touch", ReducedMotion: true}}
	if got.e != nil || got.p != expected {
		t.Fatal("motion overwrote latest profile", got)
	}
	if e = s.DeleteAccount(ctx, u.ID, u.Username); e != nil {
		t.Fatal(e)
	}
	if _, e = s.UpdateMotion(ctx, u.ID, false); !errors.Is(e, auth.ErrInvalid) {
		t.Fatal("motion accepted tombstone", e)
	}
}
