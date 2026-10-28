package realtime

import (
	"context"
	"testing"
)

func TestResultStatusOnlyDeliveredToParticipants(t *testing.T) {
	h := newTestHub(t)
	ctx := context.Background()
	a, e := h.Join(ctx, JoinContext{Token: "participant", WorldID: "a"})
	if e != nil {
		t.Fatal(e)
	}
	b, e := h.Join(ctx, JoinContext{Token: "spectator", WorldID: "a"})
	if e != nil {
		t.Fatal(e)
	}
	ac := h.Events(JoinContext{Token: "participant", WorldID: "a", SessionID: a.SessionID, Epoch: a.Epoch})
	bc := h.Events(JoinContext{Token: "spectator", WorldID: "a", SessionID: b.SessionID, Epoch: b.Epoch})
	h.PublishParticipants("a", []string{"participant"}, "match.result_status", map[string]string{"matchId": "private", "status": "completed"})
	select {
	case event := <-ac:
		if event.Type != "match.result_status" {
			t.Fatal(event)
		}
	default:
		t.Fatal("participant missed event")
	}
	select {
	case event := <-bc:
		t.Fatal("spectator saw private status", event)
	default:
	}
}
