package realtime

import (
	"context"
	"testing"
	"time"
)

func TestJoinedLeaseOrdersLeaveAndRevokeWithoutStoppingTicks(t *testing.T) {
	for _, action := range []string{"leave", "revoke"} {
		t.Run(action, func(t *testing.T) {
			h := newTestHub(t)
			ctx := context.Background()
			s, e := h.Join(ctx, JoinContext{Token: "u", WorldID: "a"})
			if e != nil {
				t.Fatal(e)
			}
			j := JoinContext{Token: "u", WorldID: "a", SessionID: s.SessionID, Epoch: s.Epoch}
			entered, release, finished := make(chan struct{}), make(chan struct{}), make(chan error, 1)
			go func() {
				finished <- h.WithJoined(ctx, j, func(context.Context, string) error { close(entered); <-release; return nil })
			}()
			<-entered
			revoked := make(chan struct{})
			go func() {
				if action == "leave" {
					h.Leave(j)
				} else {
					h.Revoke("a", "u")
				}
				close(revoked)
			}()
			select {
			case <-revoked:
				t.Fatal("seat removed in active storage lease")
			case <-time.After(80 * time.Millisecond):
			}
			next, e := h.Snapshot(ctx, j)
			close(release)
			if e != nil || next.Tick <= s.Tick {
				t.Fatal("storage blocked runtime tick", e)
			}
			if e = <-finished; e != nil {
				t.Fatal(e)
			}
			select {
			case <-revoked:
			case <-time.After(time.Second):
				t.Fatal("seat release stuck")
			}
			if e = h.WithJoined(ctx, j, func(context.Context, string) error { t.Fatal("revoked callback ran"); return nil }); e == nil {
				t.Fatal("revoked lease accepted")
			}
		})
	}
}
func TestWorldEventDeliveryBoundedAndIsolated(t *testing.T) {
	h := newTestHub(t)
	ctx := context.Background()
	a, _ := h.Join(ctx, JoinContext{Token: "a", WorldID: "a"})
	b, _ := h.Join(ctx, JoinContext{Token: "b", WorldID: "b"})
	j := JoinContext{Token: "a", WorldID: "a", SessionID: a.SessionID, Epoch: a.Epoch}
	other := JoinContext{Token: "b", WorldID: "b", SessionID: b.SessionID, Epoch: b.Epoch}
	for i := 0; i < 65; i++ {
		h.Publish("a", "chat.message", map[string]string{"id": "1"})
	}
	if len(h.Events(j)) != 64 {
		t.Fatal("queue unbounded")
	}
	select {
	case <-h.Done(j):
	default:
		t.Fatal("overflow did not terminate")
	}
	select {
	case <-h.Events(other):
		t.Fatal("world leak")
	default:
	}
	if !h.Online("b") || h.Online("a") {
		t.Fatal("presence incorrect")
	}
}

func TestDeleteControlCannotOvertakeStoredChatDelivery(t *testing.T) {
	h := newTestHub(t)
	ctx := context.Background()
	s, _ := h.Join(ctx, JoinContext{Token: "u", WorldID: "a"})
	j := JoinContext{Token: "u", WorldID: "a", SessionID: s.SessionID, Epoch: s.Epoch}
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- h.WithJoined(ctx, j, func(context.Context, string) error {
			close(entered)
			<-release
			h.Publish("a", "chat.message", map[string]string{"id": "1"})
			return nil
		})
	}()
	<-entered
	deleted := make(chan error, 1)
	go func() {
		deleted <- h.WithControl(ctx, func(context.Context) error { h.Publish("a", "chat.deleted", map[string]string{"id": "1"}); return nil })
	}()
	select {
	case <-deleted:
		close(release)
		<-done
		t.Fatal("delete overtook message delivery")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	if e := <-deleted; e != nil {
		t.Fatal(e)
	}
	if event := <-h.Events(j); event.Type != "chat.message" {
		t.Fatal("message publication reordered")
	}
	if event := <-h.Events(j); event.Type != "chat.deleted" {
		t.Fatal("delete publication reordered")
	}
}

type shutdownRoomGate struct {
	room
	entered chan struct{}
	release chan struct{}
}

func (g shutdownRoomGate) Close() { close(g.entered); <-g.release }

func TestCloseReleasesLeaseBeforeWaitingForSelectedExpiryTick(t *testing.T) {
	// The controlled worker has already selected the expiry branch and reaches
	// its real critical section while Close holds both locks in room shutdown.
	entered, release, tickSelected := make(chan struct{}), make(chan struct{}), make(chan struct{})
	h := &Hub{stop: make(chan struct{}), done: make(chan struct{}), rooms: map[string]room{"shutdown": shutdownRoomGate{entered: entered, release: release}}}
	go func() { defer close(h.done); <-entered; close(tickSelected); h.expireTick() }()
	closed := make(chan struct{})
	go func() { h.Close(); close(closed) }()
	<-tickSelected
	secondClosed := make(chan struct{})
	go func() { h.Close(); close(secondClosed) }()
	close(release)
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Close retained lease while waiting for selected expiry tick")
	}
	select {
	case <-secondClosed:
	case <-time.After(time.Second):
		t.Fatal("concurrent Close did not observe completed expiry worker")
	}
	// Repeated shutdown observes the same completed worker.
	h.Close()
}
