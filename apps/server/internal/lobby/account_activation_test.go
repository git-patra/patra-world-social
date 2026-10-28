package lobby

import (
	"context"
	"patra-world-social/server/internal/games/model"
	"testing"
)

func TestDeletionPresenceFenceBeforeAndAfterPreparedActivation(t *testing.T) {
	for _, before := range []bool{true, false} {
		t.Run(map[bool]string{true: "before_activation", false: "after_activation"}[before], func(t *testing.T) {
			s, store, _ := setup(t)
			engine := &testEngine{}
			s.Register("race", func(model.MatchStart) (Engine, error) { return engine, nil })
			l := join(t, s, "erased", "race")
			ready(t, s, "erased", l)
			store.hold = make(chan struct{})
			store.entered = make(chan struct{}, 1)
			finished := make(chan error, 1)
			go func() { _, e := s.StartMatch(context.Background(), "erased", l.ID, "prepared"); finished <- e }()
			<-store.entered
			if before {
				s.Notify("leave", "world", "erased")
				if e := s.Sync(context.Background()); e != nil {
					t.Fatal(e)
				}
			}
			close(store.hold)
			<-finished
			if !before {
				s.Notify("leave", "world", "erased")
				if e := s.Sync(context.Background()); e != nil {
					t.Fatal(e)
				}
			}
			all, e := s.List(context.Background(), "world")
			if e != nil {
				t.Fatal(e)
			}
			if before && engine.started != 0 {
				t.Fatal("deleted sole human activated")
			}
			for _, l := range all {
				if l.State == "running" || l.State == "countdown" {
					t.Fatal("departed account still running", l.State)
				}
			}
		})
	}
}
