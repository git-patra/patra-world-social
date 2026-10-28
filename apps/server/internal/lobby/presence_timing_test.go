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

func TestQueuedReconnectPreservesEveryTransition(t *testing.T) {
	s, _, c := setup(t)
	engine := &timedEngine{}
	s.Register("penalty", func(model.MatchStart) (Engine, error) { return engine, nil })
	l := join(t, s, "u", "penalty")
	s.ChooseBot(context.Background(), "u", l.ID, true)
	ready(t, s, "u", l)
	m, err := s.StartMatch(context.Background(), "u", l.ID, "ordered")
	if err != nil {
		t.Fatal(err)
	}
	origin := c.Now()
	entered, resume, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- s.WithEngine(context.Background(), m.MatchID, "", func(Engine) error { close(entered); <-resume; return nil })
	}()
	<-entered
	s.Notify("disconnect", "world", "u")
	c.advance(time.Second)
	s.Notify("resume", "world", "u")
	c.advance(time.Second)
	s.Notify("disconnect", "world", "u")
	close(resume)
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	s.Sync(context.Background())
	s.control(func() {
		want := []string{"disconnect", "resume", "disconnect"}
		if len(engine.notifications) != len(want) {
			t.Fatalf("lost transitions: %+v", engine.notifications)
		}
		for i, kind := range want {
			if engine.notifications[i].kind != kind || !engine.notifications[i].at.Equal(origin.Add(time.Duration(i)*time.Second)) {
				t.Errorf("transition %d: %+v", i, engine.notifications[i])
			}
		}
	})
	c.advance(13 * time.Second)
	s.Sync(context.Background())
	s.control(func() {
		if len(engine.notifications) != 3 {
			t.Errorf("expired against old disconnect: %+v", engine.notifications)
		}
	})
	c.advance(2 * time.Second)
	s.Sync(context.Background())
	s.control(func() {
		if len(engine.notifications) != 4 || engine.notifications[3].kind != "expired" {
			t.Errorf("new disconnect did not expire: %+v", engine.notifications)
		}
	})
}
