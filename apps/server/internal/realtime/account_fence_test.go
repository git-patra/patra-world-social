package realtime

import (
	"context"
	"testing"
	"time"
)

func TestAccountDeletionFenceOrdersAdmissionDrainCommitAndFailure(t *testing.T) {
	// The real lease boundary, independent of DB/transport adapters: commit must
	// wait for admitted work; new admitted work cannot enter until commit exits.
	h := &Hub{seats: map[string]*seat{}, users: map[string]*seat{}, accountRev: map[string]uint64{}, worldRev: map[string]uint64{}, memberRev: map[[2]string]uint64{}}
	h.lease.RLock()
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	drained := false
	go func() {
		done <- h.DeleteAccount(context.Background(), "deleted", func(context.Context) error { drained = true; return nil }, func(context.Context) error {
			if !drained {
				t.Error("commit before presence drain")
			}
			close(entered)
			<-release
			return context.Canceled
		})
	}()
	select {
	case <-entered:
		t.Fatal("commit bypassed admitted operation")
	case <-time.After(10 * time.Millisecond):
	}
	h.lease.RUnlock()
	<-entered
	next := make(chan struct{})
	go func() { h.lease.RLock(); close(next); h.lease.RUnlock() }()
	select {
	case <-next:
		t.Fatal("admission bypassed deletion transaction")
	case <-time.After(10 * time.Millisecond):
	}
	close(release)
	if e := <-done; e != context.Canceled {
		t.Fatal(e)
	}
	select {
	case <-next:
	case <-time.After(time.Second):
		t.Fatal("failed deletion retained lease")
	}
	if h.accountRev["deleted"] == 0 {
		t.Fatal("revocation fence missing after failed commit")
	}
}
