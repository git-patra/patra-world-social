package pingpong

import (
	"patra-world-social/server/internal/games/model"
	"testing"
	"time"
)

func fixture() *Engine {
	return New(model.MatchStart{MatchID: "m", WorldID: "w", ArenaID: "pingpong", GameType: "pingpong", PreparedAt: 1000, StartsAt: 1000, Actors: []model.Actor{{ActorID: "a", UserID: "u", ActorType: "human"}, {ActorID: "b", UserID: "v", ActorType: "human"}}})
}
func TestServeBudgetBounceAndDeadline(t *testing.T) {
	e := fixture()
	if e.Input("watcher", 1, 1, time.UnixMilli(1000)) == nil {
		t.Fatal("spectator input")
	}
	if e.Serve("v", time.UnixMilli(1000)) == nil {
		t.Fatal("wrong server")
	}
	if e.Serve("u", time.UnixMilli(1000)) != nil {
		t.Fatal("serve")
	}
	for n := uint64(1); n < 50; n++ {
		e.Input("u", n, 1, time.UnixMilli(1000))
	}
	e.Tick(time.UnixMilli(1050))
	s := e.Snapshot()
	if s.Paddles[0] != .55 {
		t.Fatal("input flood moved paddle", s.Paddles)
	}
	e.Tick(time.UnixMilli(121000))
	s = e.Snapshot()
	if s.Result == nil || s.Result.FinishedAt != 121000 {
		t.Fatal("deadline", s)
	}
}
func TestWinByTwoCapAndDisconnect(t *testing.T) {
	for _, v := range [][3]int{{6, 0, 0}, {7, 6, 0}, {7, 5, 1}, {11, 10, 1}} {
		if won(v[0], v[1]) != (v[2] == 1) {
			t.Fatal(v)
		}
	}
	e := fixture()
	e.Presence("u", "disconnect", time.UnixMilli(2000))
	e.Tick(time.UnixMilli(16999))
	if e.Snapshot().Result != nil {
		t.Fatal("early forfeit")
	}
	e.Tick(time.UnixMilli(17000))
	r := e.Snapshot().Result
	if r == nil || r.Participants[0].Outcome != "forfeit" {
		t.Fatal("grace", r)
	}
	e.Presence("u", "resume", time.UnixMilli(18000))
	if e.Snapshot().Result.FinishedAt != 17000 {
		t.Fatal("result changed")
	}
}

func TestNaturalScoreCapExpiryAndResultCopy(t *testing.T) {
	for _, scores := range [][2]int64{{6, 5}, {10, 10}} {
		e := fixture()
		e.scores = scores
		e.playing = true
		e.ball = Ball{X: .99, Y: .1, VX: .6, VY: 0}
		e.Tick(time.UnixMilli(1050))
		s := e.Snapshot()
		if s.Result == nil || s.Scores[0] != scores[0]+1 || s.Result.Pingpong.Reason != "score" {
			t.Fatal("natural goal", s)
		}
		s.Result.Pingpong.Scores[0] = 99
		*s.Result.Participants[0].Score = 99
		if e.Snapshot().Result.Pingpong.Scores[0] == 99 || *e.Snapshot().Result.Participants[0].Score == 99 {
			t.Fatal("mutable result alias")
		}
	}
	e := fixture()
	e.Input("u", 1, 1, time.UnixMilli(1000))
	for at := int64(1050); at <= 1500; at += 50 {
		e.Tick(time.UnixMilli(at))
	}
	if e.Snapshot().Paddles[0] > .751 {
		t.Fatal("expired input kept moving")
	}
	e = fixture()
	e.Presence("u", "disconnect", time.UnixMilli(2000))
	e.Presence("u", "resume", time.UnixMilli(17000))
	if e.Snapshot().Result == nil {
		t.Fatal("late reconnect escaped forfeit")
	}
	if e.Snapshot().Actors[0].ActorType != "human" {
		t.Fatal("human replaced by BOT")
	}
}
