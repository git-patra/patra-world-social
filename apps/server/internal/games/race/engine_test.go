package race

import (
	"encoding/json"
	"os"
	"patra-world-social/server/internal/games/model"
	"testing"
	"time"
)

func startFixture(humans int) model.MatchStart {
	m := model.MatchStart{MatchID: "match", WorldID: "world", ArenaID: "race", GameType: "race", Seed: 42, PreparedAt: 1000, StartsAt: 4000}
	for i := 0; i < 5; i++ {
		a := model.Actor{ActorID: string(rune('a' + i)), ActorType: "bot", DisplayName: "BOT", AvatarID: "pumpkin"}
		if i < humans {
			a.ActorType = "human"
			a.UserID = a.ActorID
			a.DisplayName = "Human"
		}
		m.Actors = append(m.Actors, a)
	}
	return m
}
func advance(e *Engine, from, to int64) {
	for ms := from; ms <= to; ms += 10 {
		e.Tick(time.UnixMilli(ms))
	}
}
func TestCountdownControlsExpiryAndWall(t *testing.T) {
	e := New(startFixture(1), DefaultTrack(), DefaultRules())
	before := e.Snapshot("").Karts[0]
	if err := e.InputAt("a", 1, Controls{Throttle: 1}, time.UnixMilli(2000)); err != nil {
		t.Fatal(err)
	}
	advance(e, 2000, 4000)
	after := e.Snapshot("").Karts[0]
	if before.X != after.X || before.Y != after.Y {
		t.Fatal("countdown moved")
	}
	e.InputAt("a", 2, Controls{Throttle: 1}, time.UnixMilli(4000))
	advance(e, 4000, 4200)
	if e.Snapshot("").Karts[0].X <= after.X {
		t.Fatal("throttle did not accelerate")
	}
	advance(e, 4210, 10000)
	k := e.Snapshot("").Karts[0]
	if k.Speed != 0 || !e.track.Driveable(k.X, k.Y, e.rules.Radius) {
		t.Fatal("expired input or collision invalid", k)
	}
	if e.Input("a", 2, Controls{}) == nil || e.Input("b", 1, Controls{}) == nil || e.Input("a", 3, Controls{Steer: 2}) == nil {
		t.Fatal("invalid input accepted")
	}
}
func TestBotsActuallyDriveFiveLapsAndHumanTimesOut(t *testing.T) {
	e := New(startFixture(1), DefaultTrack(), DefaultRules())
	advance(e, 4000, 304000)
	r, ok := e.Result()
	if !ok {
		t.Fatal("race locked past 300s")
	}
	finished := 0
	for _, p := range r.Participants {
		if p.ActorType == "bot" && !p.Race.DNF {
			finished++
			if p.Race.Laps != 5 || len(p.Race.LapTimesMs) != 5 {
				t.Fatal(p)
			}
		}
	}
	if finished != 4 {
		t.Fatalf("bots must race physically: %d finished, snapshot %+v", finished, e.Snapshot(""))
	}
	if !r.Participants[0].Race.DNF {
		t.Fatal("idle human finished")
	}
	if r.FinishedAt > 304000 {
		t.Fatal("absolute limit")
	}
	first := int64(300000)
	for _, p := range r.Participants {
		if p.Race.ElapsedMs != nil && *p.Race.ElapsedMs < first {
			first = *p.Race.ElapsedMs
		}
	}
	if r.FinishedAt > 4000+first+45000 {
		t.Fatal("first finish deadline ignored")
	}
}
func TestFiveHumansSameStartAndDeterministicTie(t *testing.T) {
	e := New(startFixture(5), DefaultTrack(), DefaultRules())
	for _, k := range e.karts {
		k.X = 20400
		k.Y = 6144
		k.Heading = 0
		k.Speed = 200
		k.Laps = 4
		k.NextCheckpoint = 3
		k.lapStarted = 100
		k.LapTimesMs = []int64{100, 100, 100, 100}
	}
	e.Tick(time.UnixMilli(4034))
	r, ok := e.Result()
	if !ok {
		t.Fatal("fifth lap crossing did not finish")
	}
	for i, p := range r.Participants {
		if *p.Position != i+1 || *p.Race.ElapsedMs != 33 {
			t.Fatal("same tick tie must follow persisted slot", p)
		}
	}
}
func TestCheckpointFixtures(t *testing.T) {
	raw, err := os.ReadFile("../../../../../contracts/race.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Crossings []struct {
			Name                           string
			From, To                       Point
			Next, Laps, WantNext, WantLaps int
		}
	}
	if err = json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	for _, v := range f.Crossings {
		t.Run(v.Name, func(t *testing.T) {
			k := kart{PublicKart: PublicKart{X: v.To.X, Y: v.To.Y, NextCheckpoint: v.Next, Laps: v.Laps}}
			e := New(startFixture(1), DefaultTrack(), DefaultRules())
			e.cross(&k, v.From, 1000)
			if k.Laps != v.WantLaps || k.NextCheckpoint != v.WantNext {
				t.Fatal(k)
			}
		})
	}
}
func TestDisconnectTakeoverResumeDNFAndAbandon(t *testing.T) {
	e := New(startFixture(2), DefaultTrack(), DefaultRules())
	e.Presence("a", "disconnect", time.UnixMilli(4000))
	advance(e, 4000, 5000)
	if !e.karts[0].BotAssisted || e.karts[0].X == e.track.Grid[0].X {
		t.Fatal("no bot takeover")
	}
	e.Presence("a", "resume", time.UnixMilli(5000))
	if e.karts[0].disconnectedAt != 0 {
		t.Fatal("resume failed")
	}
	e.Presence("a", "disconnect", time.UnixMilli(6000))
	advance(e, 6000, 20990)
	if e.karts[0].DNF {
		t.Fatal("seat expired early")
	}
	e.Tick(time.UnixMilli(21000))
	if !e.karts[0].DNF {
		t.Fatal("seat did not expire")
	}
	e.Presence("b", "leave", time.UnixMilli(21001))
	if e.Snapshot("").Phase != "abandoned" {
		t.Fatal("all permanent human departures must abandon")
	}
	if _, ok := e.Result(); ok {
		t.Fatal("abandoned has no winning result")
	}
}
func TestFixedStepsPublicationAndBoundedCatchUp(t *testing.T) {
	e := New(startFixture(5), DefaultTrack(), DefaultRules())
	events := 0
	for at := int64(4050); at <= 5000; at += 50 {
		events += len(e.Tick(time.UnixMilli(at)))
	}
	if e.frame != 30 || events != 15 {
		t.Fatalf("generic 50ms scheduler must yield 30 steps /15 publications: %d/%d", e.frame, events)
	}
	e.InputAt("a", 1, Controls{Throttle: 1}, time.UnixMilli(5000))
	e.Tick(time.UnixMilli(100000))
	if e.Snapshot("").DroppedSteps == 0 || e.karts[0].Speed != 0 {
		t.Fatal("catchup unbounded or stale input replay")
	}
}

func TestEqualFinishFrameAlwaysUsesGridNotOvershoot(t *testing.T) {
	e := New(startFixture(5), DefaultTrack(), DefaultRules())
	for i, k := range e.karts {
		k.X = 20400 + i*5
		k.Y = 6144
		k.Speed = 200
		k.Laps = 4
		k.NextCheckpoint = 3
		k.LapTimesMs = []int64{1, 1, 1, 1}
	}
	e.Tick(time.UnixMilli(4034))
	r, ok := e.Result()
	if !ok {
		t.Fatal("no result")
	}
	for i, p := range r.Participants {
		if *p.Position != i+1 {
			t.Fatal("overshoot incorrectly beat persisted slot", i, *p.Position)
		}
	}
}

func TestGraceCutoffNeverAdvancesDNFWithDelayedTick(t *testing.T) {
	e := New(startFixture(2), DefaultTrack(), DefaultRules())
	e.Presence("a", "disconnect", time.UnixMilli(4000))
	advance(e, 4000, 18990)
	before := *e.karts[0]
	e.Tick(time.UnixMilli(30000))
	k := e.karts[0]
	if !k.DNF || k.X != before.X || k.Y != before.Y {
		t.Fatal("delayed grace expiry raced past cutoff", before.X, before.Y, k.X, k.Y)
	}
}
func TestResumeAcceptsNewTransportSequence(t *testing.T) {
	e := New(startFixture(1), DefaultTrack(), DefaultRules())
	e.InputAt("a", 500, Controls{Throttle: 1}, time.UnixMilli(4000))
	e.Presence("a", "disconnect", time.UnixMilli(4100))
	e.Presence("a", "resume", time.UnixMilli(4200))
	if err := e.InputAt("a", 1, Controls{Throttle: 1}, time.UnixMilli(4200)); err != nil {
		t.Fatal("new fenced transport cannot resume controls", err)
	}
}
func TestFiveHumansCompleteViaControlsOnly(t *testing.T) {
	e := New(startFixture(5), DefaultTrack(), DefaultRules())
	var seq uint64
	for ms := int64(4000); ms <= 304000; ms += 33 {
		e.Tick(time.UnixMilli(ms))
		if _, ok := e.Result(); ok {
			break
		}
		seq++
		for _, k := range e.karts {
			if !k.Finished {
				if err := e.InputAt(k.ActorID, seq, e.bot(k), time.UnixMilli(ms)); err != nil {
					t.Fatal(err)
				}
			}
		}
		e.Tick(time.UnixMilli(ms))
		if _, ok := e.Result(); ok {
			break
		}
	}
	r, ok := e.Result()
	if !ok {
		t.Fatal("human input race never ended")
	}
	for _, p := range r.Participants {
		if p.Race.DNF || p.Race.Laps != 5 || p.ActorType != "human" {
			t.Fatal("controls did not drive five human laps", p)
		}
	}
}
func TestPhysicsVectors(t *testing.T) {
	raw, err := os.ReadFile("../../../../../contracts/race.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Physics []struct {
			Controls Controls
			Ticks    int
			Want     struct{ X, Y, Heading, Speed int }
		}
	}
	if err = json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	e := New(startFixture(1), DefaultTrack(), DefaultRules())
	k := e.karts[0]
	at := int64(4000)
	for _, v := range f.Physics {
		for i := 0; i < v.Ticks; i++ {
			at += 34
			e.move(k, v.Controls, at)
		}
		if k.X != v.Want.X || k.Y != v.Want.Y || k.Heading != v.Want.Heading || k.Speed != v.Want.Speed {
			t.Fatal(k, v.Want)
		}
	}
}
func TestPreparedSeedParity(t *testing.T) {
	m := startFixture(1)
	got, _ := Prepare(m)
	want := []string{"b", "e", "d", "a", "c"}
	for i, a := range got {
		if a.ActorID != want[i] {
			t.Fatal("grid parity", got)
		}
	}
}
func TestAllFiveHumansGraceExpiryAbandonsBeforeRaceLimit(t *testing.T) {
	e := New(startFixture(5), DefaultTrack(), DefaultRules())
	for _, a := range startFixture(5).Actors {
		e.Presence(a.UserID, "disconnect", time.UnixMilli(4000))
	}
	e.Tick(time.UnixMilli(20000))
	if e.Snapshot("").Phase != "abandoned" {
		t.Fatal("all humans expired yet race produced a result", e.Snapshot("").Phase)
	}
	if _, ok := e.Result(); ok {
		t.Fatal("abandoned race has official result")
	}
}
