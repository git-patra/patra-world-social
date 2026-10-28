package lobby

import (
	"context"
	"patra-world-social/server/internal/games/model"
	"testing"
)

func TestStorageGateCheckedInsideStartReservation(t *testing.T) {
	s := New(&memoryStore{}, &testClock{}, model.Version{})
	defer s.Close()
	register(t, s, "race")
	ctx := context.Background()
	l, e := s.JoinLobby(ctx, JoinContext{UserID: "human", WorldID: "world"}, "race")
	if e != nil {
		t.Fatal(e)
	}
	ready(t, s, "human", l)
	s.SetStorageBlocked(true)
	if _, e = s.StartMatch(ctx, "human", l.ID, "blocked"); e != ErrUnconfirmed {
		t.Fatal(e)
	}
}
