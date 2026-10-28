package duck

import (
	"patra-world-social/server/internal/games/model"
	"testing"
	"time"
)

func testStart() model.MatchStart {
	return model.MatchStart{Version: model.Version{RulesID: "duck-v1", RulesVersion: 1}, MatchID: "match", WorldID: "world", ArenaID: "duck-1", GameType: "duck", PreparedAt: 1000, StartsAt: 4000, Actors: []model.Actor{{ActorID: "a", ActorType: "human", UserID: "u"}}}
}
func newTest() *Engine { return New(testStart(), DefaultRules(), NewRNG(1)) }
func shoot(e *Engine, seq uint64, at int64, aim Aim) (ShotResult, error) {
	return e.Shoot("a", seq, aim, InputTiming{ReceivedAt: time.UnixMilli(at)})
}
func TestCountdownAndSixtySecondSession(t *testing.T) {
	e := newTest()
	if _, err := shoot(e, 1, 3999, Aim{}); err == nil {
		t.Fatal("countdown accepted shot")
	}
	e.Tick(time.UnixMilli(63999))
	if _, ok := e.Result(); ok {
		t.Fatal("early finish")
	}
	e.Tick(time.UnixMilli(64000))
	r, ok := e.Result()
	if !ok || r.FinishedAt != 64000 || r.Participants[0].Outcome != "finished" || r.Participants[0].Duck.Shots != 0 {
		t.Fatal(r, ok)
	}
	if _, err := shoot(e, 2, 64000, Aim{}); err == nil {
		t.Fatal("deadline accepted shot")
	}
}
func TestScoreCooldownDuplicateAndFrontmost(t *testing.T) {
	for _, tc := range []struct {
		kind   string
		points int64
	}{{"normal", 100}, {"fast", 200}, {"gold", 500}} {
		t.Run(tc.kind, func(t *testing.T) {
			e := newTest()
			e.Tick(time.UnixMilli(5000))
			e.targets = []target{{ID: 1, Kind: "normal", Born: 4500, X: 500, Y: 500}, {ID: 2, Kind: tc.kind, Born: 4500, X: 500, Y: 500}}
			r, err := shoot(e, 1, 5000, Aim{.5, .5})
			if err != nil || r.Points != tc.points || r.TargetID != 2 {
				t.Fatal(r, err)
			}
			if _, err = shoot(e, 1, 5300, Aim{.5, .5}); err == nil {
				t.Fatal("duplicate accepted")
			}
			if _, err = shoot(e, 2, 5249, Aim{.5, .5}); err == nil {
				t.Fatal("cooldown accepted")
			}
			r, err = shoot(e, 3, 5250, Aim{.5, .5})
			if err != nil || r.TargetID != 1 {
				t.Fatal(r, err)
			}
			r, err = shoot(e, 4, 5500, Aim{.5, .5})
			if err != nil || r.Points != 0 || e.hits != 2 || e.shots != 3 {
				t.Fatal("dead target scored", r, err, e.hits, e.shots)
			}
		})
	}
}
func TestPresenceNaturalDeadlineAndIncomplete(t *testing.T) {
	for _, tc := range []struct {
		name, kind         string
		at, ingest, finish int64
		outcome            string
	}{{"grace", "disconnect", 63000, 65000, 64000, "finished"}, {"equal", "disconnect", 49000, 65000, 64000, "finished"}, {"expired", "disconnect", 44000, 65000, 59000, "incomplete"}, {"exit", "leave", 63000, 65000, 63000, "incomplete"}} {
		t.Run(tc.name, func(t *testing.T) {
			e := newTest()
			before := e.Snapshot()
			e.Presence("u", tc.kind, time.UnixMilli(tc.at))
			if tc.kind == "disconnect" && e.Snapshot().ServerTimeMs != before.ServerTimeMs {
				t.Fatal("metadata advanced time")
			}
			e.Tick(time.UnixMilli(tc.ingest))
			r, ok := e.Result()
			if !ok || r.FinishedAt != tc.finish || r.Participants[0].Outcome != tc.outcome {
				t.Fatal(r, ok)
			}
		})
	}
}
func TestFixedStepsAndBoundedHistory(t *testing.T) {
	e := newTest()
	events := 0
	for at := int64(4000); at <= 5000; at += 50 {
		events += len(e.Tick(time.UnixMilli(at)))
	}
	if e.frame != 30 || events != 15 {
		t.Fatal(e.frame, events)
	}
	e.Tick(time.UnixMilli(63000))
	if len(e.targets) > 8 || len(e.history) > 8 || e.dropped == 0 {
		t.Fatal("unbounded catchup", len(e.targets), len(e.history), e.dropped)
	}
}
