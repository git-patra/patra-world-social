//go:build integration

package worlds

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"patra-world-social/server/internal/db"
	"patra-world-social/server/internal/testutil"
)

func isolatedDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	raw := os.Getenv("DATABASE_URL")
	if raw == "" {
		t.Fatal("DATABASE_URL required")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	var b [8]byte
	if _, err = rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	schema := "worlds_test_" + hex.EncodeToString(b[:])
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); admin.Close() })
	cfg, err := pgxpool.ParseConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	cfg.ConnConfig.RuntimeParams["application_name"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err = db.ApplyMigrations(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return pool
}
func user(t *testing.T, p *pgxpool.Pool, name string) string {
	t.Helper()
	var id string
	err := p.QueryRow(context.Background(), `INSERT INTO users(username,display_name,password_hash,recovery_hash) VALUES($1,$1,'unused','\x00') RETURNING id`, name).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestInviteConcurrencyAndRetry(t *testing.T) {
	p := isolatedDB(t)
	s := New(p)
	ctx := context.Background()
	owner := user(t, p, "owner")
	a := user(t, p, "alice")
	b := user(t, p, "bobby")
	w, err := s.CreateWorld(ctx, owner, "Moon Village")
	if err != nil {
		t.Fatal(err)
	}
	invite, err := s.IssueInvite(ctx, owner, w.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	// Hold the world row while both transactions enter PostgreSQL. The
	// observed lock waits prove this test exercises overlapping redemptions.
	holder, err := p.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Rollback(ctx)
	var lockedID string
	if err = holder.QueryRow(ctx, `SELECT id FROM worlds WHERE id=$1 FOR UPDATE`, w.ID).Scan(&lockedID); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(2)
	out := make(chan error, 2)
	for _, id := range []string{a, b} {
		go func(id string) { defer wg.Done(); _, e := s.RedeemInvite(ctx, id, invite.Token); out <- e }(id)
	}
	// Both redeemers share the issuer's account fence. The first waits on
	// the world row; the second waits on that first transaction's account lock.
	var holderPID int
	if err = holder.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&holderPID); err != nil {
		t.Fatal(err)
	}
	firstPID := testutil.BlockedPID(t, p, holderPID)
	testutil.BlockedPID(t, p, firstPID)
	if err = holder.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	close(out)
	success := 0
	for e := range out {
		if e == nil {
			success++
		} else if !errors.Is(e, ErrInviteInvalid) {
			t.Fatalf("unexpected loser error %v", e)
		}
	}
	if success != 1 {
		t.Fatalf("expected one winner got %d", success)
	}
	var winner string
	if _, err = s.AuthorizeMember(ctx, a, w.ID); err == nil {
		winner = a
	} else {
		winner = b
	}
	if _, err = s.RedeemInvite(ctx, winner, invite.Token); err != nil {
		t.Fatalf("same account retry: %v", err)
	}
	var count int
	if err = p.QueryRow(ctx, "SELECT count(*) FROM world_members WHERE world_id=$1 AND active", w.ID).Scan(&count); err != nil || count != 2 {
		t.Fatalf("member count %d: %v", count, err)
	}
}

func TestInviteBoundariesAndOwnerPermissions(t *testing.T) {
	p := isolatedDB(t)
	s := New(p)
	ctx := context.Background()
	owner := user(t, p, "owner")
	member := user(t, p, "member")
	other := user(t, p, "other")
	w, err := s.CreateWorld(ctx, owner, "Private Grove")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.AuthorizeMember(ctx, other, w.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("outsider %v", err)
	}
	listed, err := s.ListWorlds(ctx, other)
	if err != nil || len(listed) != 0 {
		t.Fatalf("outsider list: %v %v", listed, err)
	}
	targeted, err := s.IssueInvite(ctx, owner, w.ID, member)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.RedeemInvite(ctx, other, targeted.Token); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("wrong target %v", err)
	}
	if _, err = s.RedeemInvite(ctx, member, targeted.Token); err != nil {
		t.Fatal(err)
	}
	if err = s.ChangeRole(ctx, member, w.ID, owner, "member"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("member changed owner %v", err)
	}
	if _, err = s.IssueInvite(ctx, member, w.ID, ""); !errors.Is(err, ErrForbidden) {
		t.Fatalf("member issued invite: %v", err)
	}
	if err = s.ChangeRole(ctx, owner, w.ID, member, "moderator"); err != nil {
		t.Fatal(err)
	}
	if err = s.ChangeRole(ctx, member, w.ID, owner, "member"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("moderator changed owner: %v", err)
	}
	revoked, err := s.IssueInvite(ctx, owner, w.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RevokeInvite(ctx, owner, w.ID, revoked.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.RedeemInvite(ctx, other, revoked.Token); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("revoked %v", err)
	}
	expired, err := s.IssueInvite(ctx, owner, w.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.Exec(ctx, "UPDATE world_invites SET expires_at=now()-interval '1 second' WHERE id=$1", expired.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.RedeemInvite(ctx, other, expired.Token); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("expired %v", err)
	}
	if err = s.BanMember(ctx, owner, w.ID, other); err != nil {
		t.Fatal(err)
	}
	fresh, err := s.IssueInvite(ctx, owner, w.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.RedeemInvite(ctx, other, fresh.Token); !errors.Is(err, ErrForbidden) {
		t.Fatalf("banned %v", err)
	}
	if err = s.UnbanMember(ctx, owner, w.ID, other); err != nil {
		t.Fatal(err)
	}
	if _, err = s.RedeemInvite(ctx, other, fresh.Token); err != nil {
		t.Fatalf("unbanned redeem: %v", err)
	}
}

func TestMemberLimitAndRevocation(t *testing.T) {
	p := isolatedDB(t)
	s := New(p)
	ctx := context.Background()
	owner := user(t, p, "owner")
	last := user(t, p, "last_member")
	extra := user(t, p, "extra_user")
	w, err := s.CreateWorld(ctx, owner, "Full Grove")
	if err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 48; n++ {
		name := fmt.Sprintf("member_%02d", n)
		id := user(t, p, name)
		if _, err = p.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role) VALUES($1,$2,'member')`, w.ID, id); err != nil {
			t.Fatal(err)
		}
	}
	a, err := s.IssueInvite(ctx, owner, w.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.RedeemInvite(ctx, last, a.Token); err != nil {
		t.Fatalf("50th member: %v", err)
	}
	b, err := s.IssueInvite(ctx, owner, w.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.RedeemInvite(ctx, extra, b.Token); !errors.Is(err, ErrFull) {
		t.Fatalf("51st member: %v", err)
	}
	var revoked []string
	s.Revoked = func(_ context.Context, worldID, userID string) {
		if worldID == w.ID {
			revoked = append(revoked, userID)
		}
	}
	if err = s.KickMember(ctx, owner, w.ID, last); err != nil {
		t.Fatal(err)
	}
	if len(revoked) != 1 || revoked[0] != last {
		t.Fatalf("revocation hook: %v", revoked)
	}
	if _, err = s.AuthorizeMember(ctx, last, w.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("kicked admission: %v", err)
	}
	if _, err = s.RedeemInvite(ctx, extra, b.Token); err != nil {
		t.Fatalf("capacity after kick: %v", err)
	}
	if err = s.TransferWorld(ctx, owner, w.ID, extra); err != nil {
		t.Fatal(err)
	}
	if err = s.ArchiveWorld(ctx, owner, w.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("old owner archived: %v", err)
	}
	if err = s.ArchiveWorld(ctx, extra, w.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.AuthorizeMember(ctx, extra, w.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("archived admission: %v", err)
	}
}

func TestAdditionalInvitePreservesActiveOwnerAndModerator(t *testing.T) {
	p := isolatedDB(t)
	s := New(p)
	ctx := context.Background()
	owner := user(t, p, "owner")
	moderator := user(t, p, "moderator")
	w, err := s.CreateWorld(ctx, owner, "Role Grove")
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.IssueInvite(ctx, owner, w.ID, moderator)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.RedeemInvite(ctx, moderator, first.Token); err != nil {
		t.Fatal(err)
	}
	if err = s.ChangeRole(ctx, owner, w.ID, moderator, "moderator"); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ id, role string }{{owner, "owner"}, {moderator, "moderator"}} {
		var joinedBefore, joinedAfter time.Time
		if err = p.QueryRow(ctx, `SELECT joined_at FROM world_members WHERE world_id=$1 AND user_id=$2`, w.ID, tc.id).Scan(&joinedBefore); err != nil {
			t.Fatal(err)
		}
		invite, e := s.IssueInvite(ctx, owner, w.ID, "")
		if e != nil {
			t.Fatal(e)
		}
		membership, e := s.RedeemInvite(ctx, tc.id, invite.Token)
		if e != nil {
			t.Fatal(e)
		}
		if membership.Role != tc.role {
			t.Fatalf("redeeming role = %q, want %q", membership.Role, tc.role)
		}
		if err = p.QueryRow(ctx, `SELECT joined_at FROM world_members WHERE world_id=$1 AND user_id=$2`, w.ID, tc.id).Scan(&joinedAfter); err != nil {
			t.Fatal(err)
		}
		if !joinedAfter.Equal(joinedBefore) {
			t.Fatalf("%s joined_at changed", tc.role)
		}
		actual, e := s.AuthorizeMember(ctx, tc.id, w.ID)
		if e != nil || actual.Role != tc.role {
			t.Fatalf("%s persisted role = %q, %v", tc.role, actual.Role, e)
		}
	}
	var ownerID string
	var count int
	if err = p.QueryRow(ctx, `SELECT owner_id FROM worlds WHERE id=$1`, w.ID).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}
	if err = p.QueryRow(ctx, `SELECT count(*) FROM world_members WHERE world_id=$1 AND role='owner' AND active`, w.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if ownerID != owner || count != 1 {
		t.Fatalf("owner invariant owner_id=%s active owners=%d", ownerID, count)
	}
}

func TestModeratorMayKickAndBanPeerButNeverOwner(t *testing.T) {
	p := isolatedDB(t)
	s := New(p)
	ctx := context.Background()
	owner := user(t, p, "owner")
	modA := user(t, p, "mod_a")
	modB := user(t, p, "mod_b")
	w, err := s.CreateWorld(ctx, owner, "Moderation Grove")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{modA, modB} {
		invite, e := s.IssueInvite(ctx, owner, w.ID, id)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = s.RedeemInvite(ctx, id, invite.Token); e != nil {
			t.Fatal(e)
		}
		if e = s.ChangeRole(ctx, owner, w.ID, id, "moderator"); e != nil {
			t.Fatal(e)
		}
	}
	if err = s.KickMember(ctx, modA, w.ID, owner); !errors.Is(err, ErrForbidden) {
		t.Fatalf("moderator kicked owner: %v", err)
	}
	if err = s.BanMember(ctx, modA, w.ID, owner); !errors.Is(err, ErrForbidden) {
		t.Fatalf("moderator banned owner: %v", err)
	}
	if err = s.KickMember(ctx, modA, w.ID, modB); err != nil {
		t.Fatalf("moderator kicked peer: %v", err)
	}
	if _, err = s.AuthorizeMember(ctx, modB, w.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("kicked peer still admitted: %v", err)
	}
	rejoin, err := s.IssueInvite(ctx, owner, w.ID, modB)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.RedeemInvite(ctx, modB, rejoin.Token); err != nil {
		t.Fatal(err)
	}
	if err = s.ChangeRole(ctx, owner, w.ID, modB, "moderator"); err != nil {
		t.Fatal(err)
	}
	if err = s.BanMember(ctx, modA, w.ID, modB); err != nil {
		t.Fatalf("moderator banned peer: %v", err)
	}
	if _, err = s.AuthorizeMember(ctx, modB, w.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("banned peer still admitted: %v", err)
	}
}
