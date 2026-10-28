package realtime

import (
	"context"
	"patra-world-social/server/internal/auth"
	"testing"
	"time"
)

type blockedAuth struct{ called chan struct{} }

func (a blockedAuth) RequireSession(ctx context.Context, _ string) (auth.Session, error) {
	select {
	case a.called <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return auth.Session{}, ctx.Err()
}
func TestPenaltyAdmittedCallbackSkipsBlockingStorageAndFencesRevokedExpiredSeat(t *testing.T) {
	h := newTestHub(t)
	ctx := context.Background()
	s, err := h.Join(ctx, JoinContext{Token: "u", WorldID: "w"})
	if err != nil {
		t.Fatal(err)
	}
	j := JoinContext{Token: "u", WorldID: "w", SessionID: s.SessionID, Epoch: s.Epoch}
	calls := make(chan struct{}, 1)
	h.auth = blockedAuth{calls}
	called := false
	if err = h.WithAdmitted(ctx, j, func(_ context.Context, user string) error { called = user == "u"; return nil }); err != nil || !called {
		t.Fatal(err)
	}
	select {
	case <-calls:
		t.Fatal("choice attempted SQL authorization")
	default:
	}
	bad := j
	bad.Epoch++
	if h.WithAdmitted(ctx, bad, func(context.Context, string) error { t.Fatal("old epoch admitted"); return nil }) == nil {
		t.Fatal("epoch bypass")
	}
	h.mu.Lock()
	h.seats[j.SessionID].authorizedUntil = time.Now().Add(-time.Millisecond)
	h.mu.Unlock()
	if h.WithAdmitted(ctx, j, func(context.Context, string) error { return nil }) == nil {
		t.Fatal("expired seat admitted")
	}
	h.mu.Lock()
	h.seats[j.SessionID].authorizedUntil = time.Now().Add(time.Second)
	h.mu.Unlock()
	h.Revoke("w", "u")
	if h.WithAdmitted(ctx, j, func(context.Context, string) error { return nil }) == nil {
		t.Fatal("revoked seat admitted")
	}
}
