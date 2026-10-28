package lobby

import (
	"context"
	"testing"
)

func TestPingpongRequiresRegistrationAndExplicitBotOrTwoReadyHumans(t *testing.T) {
	for _, bot := range []bool{false, true} {
		t.Run(map[bool]string{false: "humans", true: "bot"}[bot], func(t *testing.T) {
			ctx := context.Background()
			s, _, _ := setup(t)
			if _, err := s.JoinLobby(ctx, JoinContext{UserID: "u", WorldID: "world"}, "pingpong"); err != ErrUnavailable {
				t.Fatal("unregistered game joined", err)
			}
			register(t, s, "pingpong")
			l := join(t, s, "u", "pingpong")
			ready(t, s, "u", l)
			if _, err := s.StartMatch(ctx, "u", l.ID, "implicit"); err != ErrNotReady {
				t.Fatal("implicit bot", err)
			}
			if bot {
				if err := s.ChooseBot(ctx, "u", l.ID, true); err != nil {
					t.Fatal(err)
				}
			} else {
				join(t, s, "v", "pingpong")
				if _, err := s.StartMatch(ctx, "u", l.ID, "unready"); err != ErrNotReady {
					t.Fatal("unready human", err)
				}
				ready(t, s, "v", l)
			}
			m, err := s.StartMatch(ctx, "u", l.ID, "ready")
			if err != nil || len(m.Actors) != 2 {
				t.Fatal(m, err)
			}
			if bot && (m.Actors[1].ActorType != "bot" || m.Actors[1].UserID != "") {
				t.Fatal("BOT account", m.Actors)
			}
			if !bot && m.Actors[1].ActorType != "human" {
				t.Fatal("human replaced", m.Actors)
			}
		})
	}
}
