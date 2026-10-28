package lobby

import (
	"context"
	"testing"
	"time"
)

func TestRaceResultsRequireAllHumansOrDurableTenSeconds(t *testing.T) {
	for _, mode := range []string{"consent", "timer"} {
		t.Run(mode, func(t *testing.T) {
			s, _, c := setup(t)
			register(t, s, "race")
			l := join(t, s, "u", "race")
			join(t, s, "v", "race")
			ready(t, s, "u", l)
			ready(t, s, "v", l)
			ctx := context.Background()
			m, e := s.StartMatch(ctx, "u", l.ID, "race")
			if e != nil {
				t.Fatal(e)
			}
			c.advance(3 * time.Second)
			s.Sync(ctx)
			s.Transition(ctx, m.MatchID, "finalizing")
			c.advance(time.Minute)
			s.Sync(ctx)
			if !s.Busy("u") {
				t.Fatal("uncommitted finalizing released")
			}
			if _, e = s.ResultAction(ctx, "world", m.MatchID, "u", false); e == nil {
				t.Fatal("return before durable result")
			}
			s.Transition(ctx, m.MatchID, "results")
			if _, e = s.ResultAction(ctx, "world", m.MatchID, "u", true); e == nil {
				t.Fatal("race got penalty rematch")
			}
			r, e := s.ResultAction(ctx, "world", m.MatchID, "u", false)
			if e != nil || r.Ready || !s.Busy("u") {
				t.Fatal("one return released all")
			}
			if mode == "consent" {
				r, e = s.ResultAction(ctx, "world", m.MatchID, "v", false)
				if e != nil || !r.Ready {
					t.Fatal("all returns did not release")
				}
				if e = s.CompleteResultAction(m.MatchID); e != nil {
					t.Fatal(e)
				}
			} else {
				c.advance(9999 * time.Millisecond)
				s.Sync(ctx)
				if !s.Busy("u") {
					t.Fatal("timer early")
				}
				c.advance(time.Millisecond)
				s.Sync(ctx)
				until := time.Now().Add(time.Second)
				for s.Busy("u") && time.Now().Before(until) {
					time.Sleep(time.Millisecond)
				}
			}
			if s.Busy("u") || s.Busy("v") {
				t.Fatal("claims held")
			}
		})
	}
}
