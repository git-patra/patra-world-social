package auth

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"testing"
	"time"
)

func TestPasswordHashIsSaltedAndVerifies(t *testing.T) {
	a, err := hashPassword(context.Background(), "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	b, err := hashPassword(context.Background(), "correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if a == b || strings.Contains(a, "correct horse battery") || !strings.HasPrefix(a, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("invalid password storage format")
	}
	if !verifyPassword(context.Background(), a, "correct horse battery") || verifyPassword(context.Background(), a, "incorrect horse battery") {
		t.Fatal("password verification incorrect")
	}
}

func TestLogoutPropagatesStorageFailure(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), "postgres://localhost/test")
	if err != nil {
		t.Fatal(err)
	}
	pool.Close()
	if err = New(pool).Logout(context.Background(), strings.Repeat("x", 43)); err == nil {
		t.Fatal("logout reported success while storage was unavailable")
	}
}

func TestCanonicalUsername(t *testing.T) {
	for _, raw := range []string{"Patra", "patra", "PAtRa"} {
		got, err := canonicalUsername(raw)
		if err != nil || got != "patra" {
			t.Fatalf("%q -> %q, %v", raw, got, err)
		}
	}
	for _, raw := range []string{"ab", "has space", "ábc", "Kelvin", "1234567890123456789012345"} {
		if _, err := canonicalUsername(raw); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
}

func TestPasswordLengthDoesNotTrim(t *testing.T) {
	if err := validatePassword(" 1234567890 "); err != nil {
		t.Fatal(err)
	}
	if err := validatePassword(" 123456789 "); err == nil {
		t.Fatal("accepted 11-character password")
	}
}

func TestHashConcurrencyIsBounded(t *testing.T) {
	for i := 0; i < cap(hashSlots); i++ {
		hashSlots <- struct{}{}
	}
	defer func() {
		for i := 0; i < cap(hashSlots); i++ {
			<-hashSlots
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := hashPassword(ctx, "correct horse battery"); err != context.DeadlineExceeded {
		t.Fatalf("hash should wait for a slot: %v", err)
	}
}
