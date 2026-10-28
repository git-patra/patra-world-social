package race

import (
	"context"
	"reflect"
	"testing"
	"time"

	"patra-world-social/server/internal/lobby"
)

func TestLobbyRaceQueuedLeaveCannotAdvancePastLaterDisconnectCycle(t *testing.T) {
	for _, tc := range []struct {
		name                                     string
		disconnect, resume, leave, again, ingest int64
	}{
		{"grace_expired", 4000, 5000, 5500, 6000, 24000},
		{"natural_finish", 297000, 298000, 298500, 299000, 304500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			service, clock, joined, created := presenceLobby(t, &presenceStore{terminal: make(chan string, 1)}, 2)
			start, err := service.StartMatch(ctx, "a", joined.ID, "cycles")
			if err != nil {
				t.Fatal(err)
			}
			observed := <-created
			entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
			go func() {
				done <- service.ReadEngine(ctx, "world", start.MatchID, func(lobby.Engine) error { close(entered); <-release; return nil })
			}()
			<-entered
			for _, n := range []struct {
				user, kind string
				at         int64
			}{
				{"a", "disconnect", tc.disconnect},
				{"a", "resume", tc.resume},
				{"b", "leave", tc.leave},
				{"a", "disconnect", tc.again},
			} {
				clock.set(n.at)
				service.Notify(n.kind, "world", n.user)
			}
			clock.set(tc.ingest)
			close(release)
			if err = <-done; err != nil {
				t.Fatal(err)
			}
			if err = service.Sync(ctx); err != nil {
				t.Fatal(err)
			}
			// Replay the transport timeline directly into the real race engine.
			// No event is allowed to advance past a later queued transition.
			expected := New(start, DefaultTrack(), DefaultRules())
			expected.Presence("a", "disconnect", time.UnixMilli(tc.disconnect))
			expected.Presence("a", "resume", time.UnixMilli(tc.resume))
			expected.Presence("b", "leave", time.UnixMilli(tc.leave))
			expected.Presence("a", "disconnect", time.UnixMilli(tc.again))
			expected.Tick(time.UnixMilli(tc.ingest))
			got, metadata := observed.saved()
			if !reflect.DeepEqual(metadata, []string{"a", "a"}) || !reflect.DeepEqual(got, expected.Snapshot("")) {
				t.Fatalf("queued departure advanced through an uninstalled disconnect: metadata=%v\ngot=%+v\nwant=%+v", metadata, got, expected.Snapshot(""))
			}
		})
	}
}
