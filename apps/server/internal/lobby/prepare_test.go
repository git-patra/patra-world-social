package lobby

import (
	"context"
	"patra-world-social/server/internal/games/model"
	"testing"
)

func TestPreparedRosterRejectsChangedIdentityAndPersistsPermutation(t *testing.T) {
	for _, mode := range []string{"duplicate", "missing", "altered", "reverse"} {
		t.Run(mode, func(t *testing.T) {
			s, store, _ := setup(t)
			var engineRoster []model.Actor
			err := s.RegisterPrepared("race", func(m model.MatchStart) (Engine, error) { engineRoster = m.Actors; return &testEngine{}, nil }, func(m model.MatchStart) ([]model.Actor, error) {
				a := m.Actors
				switch mode {
				case "duplicate":
					a[0] = a[1]
				case "missing":
					a = a[:4]
				case "altered":
					a[0].DisplayName = "changed"
				case "reverse":
					for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
						a[i], a[j] = a[j], a[i]
					}
				}
				return a, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			l := join(t, s, "human", "race")
			s.SetReady(context.Background(), "human", l.ID, true)
			m, err := s.StartMatch(context.Background(), "human", l.ID, "start")
			if mode != "reverse" {
				if err == nil || len(store.rows) != 0 || engineRoster != nil {
					t.Fatal("invalid preparation crossed persistence boundary")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if m.Actors[4].UserID != "human" || engineRoster[4].UserID != "human" || store.rows[m.MatchID].Actors[4].UserID != "human" {
				t.Fatal("prepared order not frozen")
			}
		})
	}
}
