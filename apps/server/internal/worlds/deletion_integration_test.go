//go:build integration

package worlds_test

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"patra-world-social/server/internal/auth"
	"patra-world-social/server/internal/leaderboard"
	"patra-world-social/server/internal/testutil"
	"patra-world-social/server/internal/worlds"
	"testing"
)

func TestRedeemAndDeletionSerializeWithoutTombstoneMembership(t *testing.T) {
	for _, writerFirst := range []bool{true, false} {
		t.Run(fmt.Sprint(writerFirst), func(t *testing.T) {
			p := testutil.Database(t)
			users, _ := testutil.Users(t, p, 2)
			ctx := context.Background()
			w := worlds.New(p)
			a := auth.New(p)
			a.RemoveAccount = leaderboard.RemoveAccount
			world, e := w.CreateWorld(ctx, users[0], "Synthetic")
			if e != nil {
				t.Fatal(e)
			}
			link, e := w.IssueInvite(ctx, users[0], world.ID, "")
			if e != nil {
				t.Fatal(e)
			}
			writer, deleted := make(chan error, 1), make(chan error, 1)
			if writerFirst {
				barrier, e := p.Begin(ctx)
				if e != nil {
					t.Fatal(e)
				}
				defer barrier.Rollback(ctx)
				var pid int
				if e = barrier.QueryRow(ctx, `SELECT pg_backend_pid() FROM worlds WHERE id=$1 FOR UPDATE`, world.ID).Scan(&pid); e != nil {
					t.Fatal(e)
				}
				go func() { _, e := w.RedeemInvite(ctx, users[1], link.Token); writer <- e }()
				writerPID := testutil.BlockedPID(t, p, pid)
				go func() { deleted <- a.DeleteAccount(ctx, users[1], "synthetic_1") }()
				testutil.BlockedPID(t, p, writerPID)
				if e = barrier.Commit(ctx); e != nil {
					t.Fatal(e)
				}
				if e = <-writer; e != nil {
					t.Fatal(e)
				}
				if e = <-deleted; e != nil {
					t.Fatal(e)
				}
			} else {
				held := make(chan int, 1)
				release := make(chan struct{})
				defer func() {
					select {
					case <-release:
					default:
						close(release)
					}
				}()
				a.RemoveAccount = func(ctx context.Context, tx pgx.Tx, id string) error {
					var pid int
					if e := tx.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&pid); e != nil {
						return e
					}
					held <- pid
					<-release
					return leaderboard.RemoveAccount(ctx, tx, id)
				}
				go func() { deleted <- a.DeleteAccount(ctx, users[1], "synthetic_1") }()
				pid := <-held
				go func() { _, e := w.RedeemInvite(ctx, users[1], link.Token); writer <- e }()
				testutil.BlockedPID(t, p, pid)
				close(release)
				if e = <-deleted; e != nil {
					t.Fatal(e)
				}
				if e = <-writer; e == nil {
					t.Fatal("tombstone redeemed invite")
				}
			}
			var active, consumed bool
			if e = p.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM world_members WHERE user_id=$1 AND active),(SELECT redeemed_by IS NOT NULL FROM world_invites WHERE id=$2)`, users[1], link.ID).Scan(&active, &consumed); e != nil {
				t.Fatal(e)
			}
			if active || consumed != writerFirst {
				t.Fatal("resurrection or tombstone invite consumption", active, consumed)
			}
		})
	}
}

func TestInviteIssueFencesDeletingIssuerAndTarget(t *testing.T) {
	for _, victim := range []int{1, 2} {
		for _, writerFirst := range []bool{true, false} {
			t.Run(fmt.Sprintf("victim%d_writerfirst%v", victim, writerFirst), func(t *testing.T) {
				p := testutil.Database(t)
				u, _ := testutil.Users(t, p, 3)
				ctx := context.Background()
				w := worlds.New(p)
				a := auth.New(p)
				a.RemoveAccount = leaderboard.RemoveAccount
				world, e := w.CreateWorld(ctx, u[0], "Synthetic")
				if e != nil {
					t.Fatal(e)
				}
				if _, e = p.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role)VALUES($1,$2,'moderator')`, world.ID, u[1]); e != nil {
					t.Fatal(e)
				}
				writer, deleted := make(chan error, 1), make(chan error, 1)
				issue := func() { _, e := w.IssueInvite(ctx, u[1], world.ID, u[2]); writer <- e }
				if writerFirst {
					barrier, e := p.Begin(ctx)
					if e != nil {
						t.Fatal(e)
					}
					defer barrier.Rollback(ctx)
					var pid int
					if e = barrier.QueryRow(ctx, `SELECT pg_backend_pid() FROM worlds WHERE id=$1 FOR UPDATE`, world.ID).Scan(&pid); e != nil {
						t.Fatal(e)
					}
					go issue()
					writerPID := testutil.BlockedPID(t, p, pid)
					go func() { deleted <- a.DeleteAccount(ctx, u[victim], fmt.Sprintf("synthetic_%d", victim)) }()
					testutil.BlockedPID(t, p, writerPID)
					if e = barrier.Commit(ctx); e != nil {
						t.Fatal(e)
					}
					if e = <-writer; e != nil {
						t.Fatal(e)
					}
					if e = <-deleted; e != nil {
						t.Fatal(e)
					}
				} else {
					held := make(chan int, 1)
					release := make(chan struct{})
					defer func() {
						select {
						case <-release:
						default:
							close(release)
						}
					}()
					a.RemoveAccount = func(ctx context.Context, tx pgx.Tx, id string) error {
						var pid int
						if e := tx.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&pid); e != nil {
							return e
						}
						held <- pid
						<-release
						return leaderboard.RemoveAccount(ctx, tx, id)
					}
					go func() { deleted <- a.DeleteAccount(ctx, u[victim], fmt.Sprintf("synthetic_%d", victim)) }()
					pid := <-held
					go issue()
					testutil.BlockedPID(t, p, pid)
					close(release)
					if e = <-deleted; e != nil {
						t.Fatal(e)
					}
					if e = <-writer; e == nil {
						t.Fatal("tombstone invitation created")
					}
				}
				var usable int
				if e = p.QueryRow(ctx, `SELECT count(*) FROM world_invites WHERE revoked_at IS NULL`).Scan(&usable); e != nil || usable != 0 {
					t.Fatal("usable tombstone invitation", usable, e)
				}
			})
		}
	}
}
