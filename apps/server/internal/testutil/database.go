//go:build integration

// Package testutil provisions isolated synthetic schemas, never preview accounts.
//
package testutil

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"patra-world-social/server/internal/db"
	"testing"
)

func Database(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	raw := os.Getenv("DATABASE_URL")
	if raw == "" {
		t.Fatal("DATABASE_URL required")
	}
	admin, e := pgxpool.New(ctx, raw)
	if e != nil {
		t.Fatal(e)
	}
	var b [8]byte
	rand.Read(b[:])
	schema := "social_test_" + hex.EncodeToString(b[:])
	if _, e = admin.Exec(ctx, "CREATE SCHEMA "+schema); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); admin.Close() })
	cfg, e := pgxpool.ParseConfig(raw)
	if e != nil {
		t.Fatal(e)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	p, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(p.Close)
	if e = db.ApplyMigrations(ctx, p); e != nil {
		t.Fatal(e)
	}
	return p
}
func Users(t *testing.T, p *pgxpool.Pool, n int) ([]string, []string) {
	t.Helper()
	ctx := context.Background()
	var users, tokens []string
	for i := 0; i < n; i++ {
		var id string
		b := make([]byte, 32)
		rand.Read(b)
		token := base64.RawURLEncoding.EncodeToString(b)
		hash := sha256.Sum256([]byte(token))
		e := p.QueryRow(ctx, `INSERT INTO users(username,display_name,password_hash,recovery_hash) VALUES($1,$1,'unused','\x00') RETURNING id`, fmt.Sprintf("synthetic_%d", i)).Scan(&id)
		if e != nil {
			t.Fatal(e)
		}
		_, e = p.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,expires_at,idle_expires_at) VALUES($1,$2,now()+interval '1 day',now()+interval '1 day')`, hash[:], id)
		if e != nil {
			t.Fatal(e)
		}
		users = append(users, id)
		tokens = append(tokens, token)
	}
	return users, tokens
}
