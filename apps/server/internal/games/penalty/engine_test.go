package penalty

import (
	"encoding/json"
	"os"
	"patra-world-social/server/internal/games/model"
	"strings"
	"testing"
	"time"
)

type clock struct{ at time.Time }

func (c *clock) Now() time.Time { return c.at }

type rng struct {
	values []int
	calls  int
}

func (r *rng) Intn(n int) int {
	v := 0
	if r.calls < len(r.values) {
		v = r.values[r.calls]
	}
	r.calls++
	return v % n
}
func fixture() (*Engine, *clock, *rng) {
	c := &clock{time.UnixMilli(10000)}
	r := &rng{}
	return New(model.MatchStart{MatchID: "m", WorldID: "w", ArenaID: "penalty", GameType: "penalty", PreparedAt: 9000, StartsAt: 10000, Actors: []model.Actor{{ActorID: "a", UserID: "u", ActorType: "human"}, {ActorID: "b", UserID: "v", ActorType: "human"}}}, c, r), c, r
}
func turn(t *testing.T, e *Engine, c *clock, k, z int) {
	t.Helper()
	s := e.Snapshot("")
	if err := e.Choose(s.KickerID, s.TurnID, k, c.at); err != nil {
		t.Fatal(err)
	}
	if err := e.Choose(s.KeeperID, s.TurnID, z, c.at); err != nil {
		t.Fatal(err)
	}
	c.at = time.UnixMilli(s.ChoiceDeadlineAt)
	e.Tick(c.at)
	c.at = c.at.Add(1500 * time.Millisecond)
	e.Tick(c.at)
}
func TestAllZonesFromSharedFixture(t *testing.T) {
	raw, err := os.ReadFile("../../../../../contracts/penalty.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Combinations []struct {
			Kick, Keep int
			Goal       bool
		}
	}
	if err = json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	if len(f.Combinations) != 16 {
		t.Fatal("need 16 combinations")
	}
	for _, v := range f.Combinations {
		e, c, _ := fixture()
		turn(t, e, c, v.Kick, v.Keep)
		s := e.Snapshot("")
		if len(s.Turns) != 1 || s.Turns[0].Goal != v.Goal {
			t.Fatal(v, s)
		}
	}
}
func TestDeadlineLockAndNoLeak(t *testing.T) {
	e, c, r := fixture()
	s := e.Snapshot("")
	if e.Choose("a", s.TurnID, 2, c.at.Add(2999*time.Millisecond)) != nil {
		t.Fatal("2999 rejected")
	}
	if e.Choose("a", s.TurnID, 1, c.at) == nil {
		t.Fatal("choice replaced")
	}
	if e.Choose("b", s.TurnID, 2, c.at.Add(3000*time.Millisecond)) == nil {
		t.Fatal("3000 accepted")
	}
	for _, viewer := range []string{"a", "b", "spectator"} {
		raw, _ := json.Marshal(e.Snapshot(viewer))
		if strings.Contains(string(raw), "kickZone") || strings.Contains(string(raw), "keepZone") {
			t.Fatal(string(raw))
		}
	}
	c.at = c.at.Add(3000 * time.Millisecond)
	e.Tick(c.at)
	s = e.Snapshot("")
	if s.Turns[0].KickZone != 2 || !s.Turns[0].KeepTimeout || s.Turns[0].KickTimeout || r.calls != 2 {
		t.Fatal(s, r.calls)
	}
	e.Tick(c.at)
	if r.calls != 2 {
		t.Fatal("timeout rerolled")
	}
}
func TestTimeoutIndependentAndBotBlind(t *testing.T) {
	e, c, r := fixture()
	r.values = []int{0, 1, 3}
	e.Tick(c.at.Add(3000 * time.Millisecond))
	s := e.Snapshot("")
	if s.Turns[0].KickZone != 1 || s.Turns[0].KeepZone != 3 {
		t.Fatal(s)
	}
	m := e.start
	m.Actors[1].ActorType = "bot"
	m.Actors[1].UserID = ""
	r1, r2 := &rng{values: []int{0, 2}}, &rng{values: []int{0, 2}}
	a, b := New(m, c, r1), New(m, c, r2)
	a.Choose("a", a.Snapshot("").TurnID, 0, c.at)
	b.Choose("a", b.Snapshot("").TurnID, 3, c.at)
	a.Tick(c.at.Add(3000 * time.Millisecond))
	b.Tick(c.at.Add(3000 * time.Millisecond))
	if a.Snapshot("").Turns[0].KeepZone != b.Snapshot("").Turns[0].KeepZone {
		t.Fatal("bot read human")
	}
}
func TestEarlyWinAndSuddenDeathPairs(t *testing.T) {
	e, c, _ := fixture()
	for i := 0; i < 6; i++ {
		if i%2 == 0 {
			turn(t, e, c, 0, 1)
		} else {
			turn(t, e, c, 0, 0)
		}
	}
	r, ok := e.Result()
	if !ok || r.Participants[0].Outcome != "win" || len(r.PenaltyTurns) != 6 {
		t.Fatal(r, ok)
	}
	e, c, _ = fixture()
	for i := 0; i < 10; i++ {
		turn(t, e, c, 0, 0)
	}
	turn(t, e, c, 0, 1)
	if _, ok = e.Result(); ok {
		t.Fatal("sudden death unequal chance")
	}
	turn(t, e, c, 0, 0)
	r, ok = e.Result()
	if !ok || r.Participants[0].Outcome != "win" {
		t.Fatal(r, ok)
	}
}
func TestMaxFiveExtraPairsDraw(t *testing.T) {
	e, c, _ := fixture()
	for i := 0; i < 20; i++ {
		turn(t, e, c, 0, 0)
	}
	r, ok := e.Result()
	if !ok || len(r.PenaltyTurns) != 20 || r.Participants[0].Outcome != "draw" {
		t.Fatal(r, ok)
	}
}
func TestGraceAndNaturalFinishOrdering(t *testing.T) {
	e, c, _ := fixture()
	e.Presence("u", "disconnect", c.at)
	e.Tick(c.at.Add(14999 * time.Millisecond))
	if _, ok := e.Result(); ok {
		t.Fatal("early forfeit")
	}
	e.Tick(c.at.Add(15000 * time.Millisecond))
	r, ok := e.Result()
	if !ok || r.Participants[0].Outcome != "forfeit" || r.Participants[1].Outcome != "win" {
		t.Fatal(r, ok)
	}
	e, c, _ = fixture()
	for i := 0; i < 5; i++ {
		if i%2 == 0 {
			turn(t, e, c, 0, 1)
		} else {
			turn(t, e, c, 0, 0)
		}
	}
	s := e.Snapshot("")
	e.Choose(s.KickerID, s.TurnID, 0, c.at)
	e.Choose(s.KeeperID, s.TurnID, 0, c.at)
	e.Presence("u", "disconnect", c.at)
	e.Presence("v", "disconnect", c.at)
	e.Tick(c.at.Add(20000 * time.Millisecond))
	r, ok = e.Result()
	if !ok || r.Participants[0].Outcome != "win" {
		t.Fatal("natural result overwritten", r, ok)
	}
}
func TestForfeitWinnerIsNotDerivedFromScoreAndBothGoneAbandon(t *testing.T) {
	e, c, _ := fixture()
	turn(t, e, c, 0, 1)
	e.Presence("u", "leave", c.at)
	r, ok := e.Result()
	if !ok || *r.Participants[0].Score != 1 || r.Participants[0].Outcome != "forfeit" || r.Participants[1].Outcome != "win" {
		t.Fatal(r, ok)
	}
	e, c, _ = fixture()
	e.Presence("u", "disconnect", c.at)
	e.Presence("v", "disconnect", c.at)
	e.Tick(c.at.Add(15000 * time.Millisecond))
	r, ok = e.Result()
	if !ok || r.Participants[0].Outcome != "abandoned" || r.Participants[1].Outcome != "abandoned" {
		t.Fatal(r, ok)
	}
}
func TestSharedTimelinesAndBoundaries(t *testing.T) {
	raw, err := os.ReadFile("../../../../../contracts/penalty.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		ChoiceMs, AnimationMs int64
		Boundaries            []struct {
			OffsetMs int64
			Accepted bool
		}
		Scenarios []struct {
			Name     string
			Turns    []struct{ Kick, Keep int }
			Scores   [2]int64
			Outcomes [2]string
		}
	}
	if err = json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	if f.ChoiceMs != ChoiceMs || f.AnimationMs != AnimationMs {
		t.Fatal("rules fixture mismatch")
	}
	for _, b := range f.Boundaries {
		e, c, _ := fixture()
		err = e.Choose("a", e.Snapshot("").TurnID, 2, c.at.Add(time.Duration(b.OffsetMs)*time.Millisecond))
		if (err == nil) != b.Accepted {
			t.Fatal(b, err)
		}
	}
	for _, s := range f.Scenarios {
		t.Run(s.Name, func(t *testing.T) {
			e, c, _ := fixture()
			for _, v := range s.Turns {
				turn(t, e, c, v.Kick, v.Keep)
			}
			r, ok := e.Result()
			if !ok {
				t.Fatal("not finished")
			}
			for i, p := range r.Participants {
				if *p.Score != s.Scores[i] || p.Outcome != s.Outcomes[i] {
					t.Fatal(r, s)
				}
			}
		})
	}
}
func TestRevealIsAtDeadlineAndNextTurnAfterFullAnimation(t *testing.T) {
	e, c, _ := fixture()
	s := e.Snapshot("")
	e.Choose("a", s.TurnID, 0, c.at)
	e.Choose("b", s.TurnID, 1, c.at)
	for _, ev := range e.Tick(c.at.Add(2999 * time.Millisecond)) {
		if ev.Type == "penalty.revealed" {
			t.Fatal("early reveal")
		}
	}
	events := e.Tick(c.at.Add(3000 * time.Millisecond))
	if len(events) != 1 || events[0].Type != "penalty.revealed" || events[0].Snapshot.Phase != "animation" {
		t.Fatal(events)
	}
	e.Tick(c.at.Add(4499 * time.Millisecond))
	if e.Snapshot("").Turn != 1 {
		t.Fatal("animation shortened")
	}
	e.Tick(c.at.Add(4500 * time.Millisecond))
	if e.Snapshot("").Turn != 2 {
		t.Fatal("next turn missing")
	}
}
func TestResumeKeepsOriginalDeadlineAndCannotReviveExpiredGrace(t *testing.T) {
	e, c, _ := fixture()
	before := e.Snapshot("")
	e.Presence("u", "disconnect", c.at)
	e.Presence("u", "resume", c.at.Add(1000*time.Millisecond))
	after := e.Snapshot("")
	if before.TurnID != after.TurnID || before.ChoiceDeadlineAt != after.ChoiceDeadlineAt {
		t.Fatal(before, after)
	}
	e.Presence("u", "disconnect", c.at.Add(1000*time.Millisecond))
	e.Presence("u", "resume", c.at.Add(16000*time.Millisecond))
	r, ok := e.Result()
	if !ok || r.Participants[0].Outcome != "forfeit" {
		t.Fatal(r, ok)
	}
}
