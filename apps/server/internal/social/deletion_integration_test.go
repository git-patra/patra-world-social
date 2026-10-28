//go:build integration

package social_test

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"patra-world-social/server/internal/auth"
	"patra-world-social/server/internal/leaderboard"
	"patra-world-social/server/internal/social"
	"patra-world-social/server/internal/testutil"
	"strings"
	"testing"
)

func TestFriendWritersFenceDeletingActorsAndTargets(t *testing.T) {
	for _, victim := range []int{0, 1} {
		for _, writerFirst := range []bool{true, false} {
			t.Run(fmt.Sprintf("victim%d_writerfirst%v", victim, writerFirst), func(t *testing.T) {
				p := testutil.Database(t)
				u, _ := testutil.Users(t, p, 2)
				ctx := context.Background()
				s := social.New(p)
				a := auth.New(p)
				a.RemoveAccount = leaderboard.RemoveAccount
				writer, deleted := make(chan error, 1), make(chan error, 1)
				if writerFirst {
					barrier, e := p.Begin(ctx)
					if e != nil {
						t.Fatal(e)
					}
					defer barrier.Rollback(ctx)
					if e = social.LockPair(ctx, barrier, u[0], u[1]); e != nil {
						t.Fatal(e)
					}
					var pid int
					if e = barrier.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&pid); e != nil {
						t.Fatal(e)
					}
					go func() { writer <- s.RequestFriend(ctx, u[0], "synthetic_1") }()
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
					go func() { writer <- s.RequestFriend(ctx, u[0], "synthetic_1") }()
					testutil.BlockedPID(t, p, pid)
					close(release)
					if e := <-deleted; e != nil {
						t.Fatal(e)
					}
					if e := <-writer; e == nil {
						t.Fatal("tombstone relation accepted")
					}
				}
				var count int
				if e := p.QueryRow(ctx, `SELECT count(*) FROM friend_requests`).Scan(&count); e != nil || count != 0 {
					t.Fatal("friend relation survived deletion", count, e)
				}
				if e := s.Remove(ctx, u[0], u[1], true); e == nil {
					t.Fatal("tombstone block relation accepted")
				}
			})
		}
	}
}

func TestBlockWriterCanonicalizesTargetAccountFence(t *testing.T) {
	p := testutil.Database(t)
	u, _ := testutil.Users(t, p, 2)
	ctx := context.Background()
	a := auth.New(p)
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
	deleted := make(chan error, 1)
	go func() { deleted <- a.DeleteAccount(ctx, u[1], "synthetic_1") }()
	pid := <-held
	writer := make(chan error, 1)
	go func() {
		writer <- social.New(p).Remove(ctx, u[0], strings.ToUpper(strings.ReplaceAll(u[1], "-", "")), true)
	}()
	testutil.BlockedPID(t, p, pid)
	close(release)
	if e := <-deleted; e != nil {
		t.Fatal(e)
	}
	if e := <-writer; e == nil {
		t.Fatal("variant target bypassed deleted account fence")
	}
	var count int
	if e := p.QueryRow(ctx, `SELECT count(*) FROM user_blocks`).Scan(&count); e != nil || count != 0 {
		t.Fatal("residual block", count, e)
	}
}
