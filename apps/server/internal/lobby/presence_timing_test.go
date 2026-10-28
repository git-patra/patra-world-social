package lobby

import (
	"context"
	"patra-world-social/server/internal/games/model"
	"testing"
	"time"
)

type timedEngine struct {
	testEngine
	notifications []struct {
		kind string
		at   time.Time
	}
}

func (e *timedEngine) PresenceAt(user, kind string, at time.Time) {
	e.notifications = append(e.notifications, struct {
		kind string
		at   time.Time
	}{kind, at})
}
func TestTimedEngineReceivesNotificationAndGraceTimeWithoutIntentionalLeave(t *testing.T) {
	s, _, c := setup(t)
	engine := &timedEngine{}
	s.Register("penalty", func(model.MatchStart) (Engine, error) { return engine, nil })
	l := join(t, s, "u", "penalty")
	s.ChooseBot(context.Background(), "u", l.ID, true)
	ready(t, s, "u", l)
	m, e := s.StartMatch(context.Background(), "u", l.ID, "timed")
	if e != nil {
		t.Fatal(e)
	}
	origin := c.Now()
	entered, resume, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- s.WithEngine(context.Background(), m.MatchID, "", func(Engine) error { close(entered); <-resume; return nil })
	}()
	<-entered
	s.Notify("disconnect", "world", "u")
	c.advance(100 * time.Millisecond)
	close(resume)
	if e = <-done; e != nil {
		t.Fatal(e)
	}
	s.Sync(context.Background())
	c.advance(14900 * time.Millisecond)
	s.Sync(context.Background())
	s.control(func() {
		if len(engine.notifications) != 2 || engine.notifications[0].kind != "disconnect" || !engine.notifications[0].at.Equal(origin) || engine.notifications[1].kind != "expired" || !engine.notifications[1].at.Equal(origin.Add(15*time.Second)) {
			t.Error(engine.notifications)
		}
		if len(engine.events) != 0 {
			t.Error("timed engine received ingestion-time/intentional fallback", engine.events)
		}
	})
}
