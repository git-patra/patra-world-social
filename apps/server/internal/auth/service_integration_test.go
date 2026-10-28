//go:build integration

package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"patra-world-social/server/internal/db"
)

func isolatedDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	raw := os.Getenv("DATABASE_URL")
	if raw == "" {
		t.Fatal("DATABASE_URL required for integration tests")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	var suffix [8]byte
	if _, err = rand.Read(suffix[:]); err != nil {
		t.Fatal(err)
	}
	schema := "auth_test_" + hex.EncodeToString(suffix[:])
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

// A blocked session INSERT reproduces the gap between old-password verification
// and session persistence. Recovery must either wait for that login and revoke
// it, or win the user lock so the login rejects the changed credential hash.
func TestRecoverySerializesWithInFlightOldPasswordLogin(t *testing.T) {
	ctx := context.Background()
	pool := isolatedDB(t)
	service := New(pool)
	_, code, err := service.Register(ctx, "racer", "old password value")
	if err != nil {
		t.Fatal(err)
	}
	const lockID = int64(861223409)
	holder, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Release()
	if _, err = holder.Exec(ctx, "SELECT pg_advisory_lock($1)", lockID); err != nil {
		t.Fatal(err)
	}
	locked := true
	release := func() {
		if locked {
			holder.Exec(ctx, "SELECT pg_advisory_unlock($1)", lockID)
			locked = false
		}
	}
	defer release()
	if _, err = pool.Exec(ctx, `CREATE FUNCTION hold_auth_insert() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_advisory_lock(861223409); PERFORM pg_advisory_unlock(861223409); RETURN NEW; END $$`); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `CREATE TRIGGER hold_auth_insert BEFORE INSERT ON sessions FOR EACH ROW EXECUTE FUNCTION hold_auth_insert()`); err != nil {
		t.Fatal(err)
	}
	type loginResult struct {
		session Session
		err     error
	}
	loginDone := make(chan loginResult, 1)
	go func() {
		session, e := service.Authenticate(ctx, "racer", "old password value")
		loginDone <- loginResult{session, e}
	}()
	waitForPGWait(t, pool, "%INSERT INTO sessions%")
	var testApp string
	if err = pool.QueryRow(ctx, "SELECT current_setting('application_name')").Scan(&testApp); err != nil {
		t.Fatal(err)
	}
	recoverDone := make(chan error, 1)
	go func() { _, e := service.Recover(ctx, "racer", string(code), "new password value"); recoverDone <- e }()
	// Recovery either finishes (the original bug) or blocks on the login's
	// user-row lock. Both outcomes are observed before releasing the INSERT.
	deadline := time.After(5 * time.Second)
	for {
		select {
		case recoveryErr := <-recoverDone:
			// Preserve the recovery result for the assertion below.
			recoverDone <- recoveryErr
			goto releaseInsert
		case <-deadline:
			t.Fatal("recovery neither completed nor waited for the login lock")
		default:
		}
		var waiting bool
		if err = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE application_name=$1 AND pid<>pg_backend_pid() AND wait_event_type='Lock' AND query LIKE '%UPDATE users SET password_hash%')`, testApp).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
releaseInsert:
	release()
	var login loginResult
	select {
	case login = <-loginDone:
	case <-time.After(5 * time.Second):
		t.Fatal("login did not finish")
	}
	select {
	case err = <-recoverDone:
	case <-time.After(5 * time.Second):
		t.Fatal("recovery did not finish")
	}
	if err != nil {
		t.Fatal(err)
	}
	if login.err == nil {
		if _, err = service.RequireSession(ctx, login.session.Token); !errors.Is(err, ErrInvalid) {
			t.Fatalf("old-password login survived recovery: %v", err)
		}
	}
}

func waitForPGWait(t *testing.T, pool *pgxpool.Pool, queryPattern string) {
	t.Helper()
	var testApp string
	if err := pool.QueryRow(context.Background(), "SELECT current_setting('application_name')").Scan(&testApp); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var waiting bool
		if err := pool.QueryRow(context.Background(), `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE application_name=$1 AND pid<>pg_backend_pid() AND wait_event_type='Lock' AND query LIKE $2)`, testApp, queryPattern).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for PostgreSQL lock")
}

func TestRealPostgresIdentityLifecycle(t *testing.T) {
	ctx := context.Background()
	pool := isolatedDB(t)
	s := New(pool)
	var revocations []string
	s.Revoked = func(_ context.Context, userID string) { revocations = append(revocations, userID) }
	user, code, err := s.Register(ctx, "Patra", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "patra" || len(code) < 40 {
		t.Fatal("incorrect registration")
	}
	var stored string
	if err = pool.QueryRow(ctx, "SELECT password_hash FROM users WHERE id=$1", user.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == "correct horse battery" || strings.Contains(stored, "correct horse battery") {
		t.Fatal("plaintext password persisted")
	}
	var wg sync.WaitGroup
	wg.Add(2)
	duplicates := make(chan error, 2)
	for _, name := range []string{"patra", "PAtRa"} {
		go func(v string) {
			defer wg.Done()
			_, _, e := s.Register(ctx, v, "correct horse battery")
			duplicates <- e
		}(name)
	}
	wg.Wait()
	close(duplicates)
	for e := range duplicates {
		if !errors.Is(e, ErrDuplicate) {
			t.Fatalf("duplicate race: %v", e)
		}
	}
	if _, err = s.Authenticate(ctx, "Patra", "wrong password"); !errors.Is(err, ErrInvalid) {
		t.Fatal("invalid password accepted")
	}
	session, err := s.Authenticate(ctx, "Patra", "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.RequireSession(ctx, session.Token); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Recover(ctx, "patra", "invalid", "new correct battery"); !errors.Is(err, ErrInvalid) {
		t.Fatal("invalid recovery accepted")
	}
	newCode, err := s.Recover(ctx, "PAtRa", string(code), "new correct battery")
	if err != nil {
		t.Fatal(err)
	}
	if newCode == code {
		t.Fatal("recovery code reused")
	}
	if len(revocations) != 1 || revocations[0] != user.ID {
		t.Fatal("recovery did not publish revocation")
	}
	if _, err = s.Recover(ctx, "patra", string(code), "another correct battery"); !errors.Is(err, ErrInvalid) {
		t.Fatal("used recovery code accepted")
	}
	if _, err = s.RequireSession(ctx, session.Token); !errors.Is(err, ErrInvalid) {
		t.Fatal("reset left prior session valid")
	}
	fresh, err := s.Authenticate(ctx, "patra", "new correct battery")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Logout(ctx, fresh.Token); err != nil {
		t.Fatal(err)
	}
	if len(revocations) != 2 || revocations[1] != user.ID {
		t.Fatal("logout did not publish revocation")
	}
	if _, err = s.RequireSession(ctx, fresh.Token); !errors.Is(err, ErrInvalid) {
		t.Fatal("logout left session valid")
	}
	expired, err := s.Authenticate(ctx, "patra", "new correct battery")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, "UPDATE sessions SET idle_expires_at=now()-interval '1 second' WHERE token_hash=$1", tokenHash(expired.Token)); err != nil {
		t.Fatal(err)
	}
	if _, err = s.RequireSession(ctx, expired.Token); !errors.Is(err, ErrInvalid) {
		t.Fatal("idle-expired session accepted")
	}
	absolute, err := s.Authenticate(ctx, "patra", "new correct battery")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, "UPDATE sessions SET expires_at=now()-interval '1 second' WHERE token_hash=$1", tokenHash(absolute.Token)); err != nil {
		t.Fatal(err)
	}
	if _, err = s.RequireSession(ctx, absolute.Token); !errors.Is(err, ErrInvalid) {
		t.Fatal("absolute-expired session accepted")
	}
}
