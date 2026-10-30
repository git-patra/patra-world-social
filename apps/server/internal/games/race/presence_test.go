package race

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestDisconnectBatchInstallsWithoutAdvancingAndUsesOriginalBoundaries(t *testing.T) {
	var snapshots []PublicSnapshot
	for _, order := range [][]string{{"a", "b"}, {"b", "a"}} {
		e := New(startFixture(2), DefaultTrack(), DefaultRules())
		before := e.Snapshot("")
		for _, user := range order {
			at := int64(4000)
			if user == "b" {
				at = 5000
			}
			e.Presence(user, "disconnect", time.UnixMilli(at))
			after := e.Snapshot("")
			if after.Frame != before.Frame || after.ServerTimeMs != before.ServerTimeMs || after.Phase != before.Phase || after.Result != nil {
				t.Fatalf("metadata installation advanced race: order=%v before=%+v after=%+v", order, before, after)
			}
			for i, k := range after.Karts {
				if k.X != before.Karts[i].X || k.Y != before.Karts[i].Y {
					t.Fatal("metadata moved kart", order, i)
				}
			}
		}
		e.Tick(time.UnixMilli(5000))
		s := e.Snapshot("")
		if s.Frame != 30 || s.DroppedSteps != 22 {
			t.Fatal("bounded catch-up changed", s)
		}
		// A drives all eight retained frames. B starts takeover only at its own
		// boundary (the last frame), rather than at the earlier batch timestamp.
		if s.Karts[0].X <= before.Karts[0].X+5 || s.Karts[1].X <= before.Karts[1].X || s.Karts[1].X > before.Karts[1].X+5 || s.Karts[1].Speed != 5 {
			t.Fatal("takeover did not use original timestamps", s.Karts[:2])
		}
		snapshots = append(snapshots, s)
	}
	if !reflect.DeepEqual(snapshots[0], snapshots[1]) {
		t.Fatal("map installation order changed physics", snapshots)
	}
}

func TestDisconnectPreservesHeldIntentOnlyBeforeItsTimestamp(t *testing.T) {
	e := New(startFixture(2), DefaultTrack(), DefaultRules())
	if err := e.InputAt("a", 100, Controls{Throttle: 1, Steer: -1}, time.UnixMilli(4000)); err != nil {
		t.Fatal(err)
	}
	e.Presence("a", "disconnect", time.UnixMilli(4200))
	if e.karts[0].controls != (Controls{}) || e.InputAt("a", 101, Controls{Throttle: 1}, time.UnixMilli(4201)) == nil {
		t.Fatal("disconnected active control not neutral/fenced")
	}
	e.Tick(time.UnixMilli(4167))
	k := e.Snapshot("").Karts[0]
	if k.Heading != 345 || k.Speed != 25 || k.X <= e.track.Grid[0].X {
		t.Fatal("valid pre-disconnect intent was erased", k)
	}
	e.Tick(time.UnixMilli(4200))
	k = e.Snapshot("").Karts[0]
	if k.Heading != 348 || k.Speed != 30 || !k.BotAssisted || k.Actor != e.start.Actors[0] {
		t.Fatal("bot did not take over at original boundary", k)
	}
	e.Presence("a", "resume", time.UnixMilli(4300))
	if !e.karts[0].BotAssisted || e.karts[0].controls != (Controls{}) {
		t.Fatal("resume lost assistance history or held control")
	}
	if err := e.InputAt("a", 1, Controls{Brake: 1}, time.UnixMilli(4301)); err != nil {
		t.Fatal("resumed sequence not reset", err)
	}
	if e.InputAt("a", 1, Controls{}, time.UnixMilli(4302)) == nil {
		t.Fatal("monotonic rejection lost")
	}
}

func TestDisconnectAssistanceRespectsGlobalCutoff(t *testing.T) {
	for _, shortened := range []bool{false, true} {
		for _, offset := range []int64{0, 1} {
			for _, prior := range []bool{false, true} {
				name := fmt.Sprintf("shortened=%t/offset=%d/prior=%t", shortened, offset, prior)
				t.Run(name, func(t *testing.T) {
					e, baseline := cutoffFixture(shortened, prior), cutoffFixture(shortened, prior)
					cutoff := e.deadline
					if shortened && cutoff != 55000 {
						t.Fatal("fixture did not establish first-finish cutoff", cutoff)
					}
					e.Tick(time.UnixMilli(cutoff - 50))
					baseline.Tick(time.UnixMilli(cutoff - 50))
					before := e.Snapshot("")
					e.Presence("a", "disconnect", time.UnixMilli(cutoff+offset))
					if e.Snapshot("").Karts[0].BotAssisted != prior || e.frame != before.Frame || e.now != before.ServerTimeMs {
						t.Fatal("metadata changed assistance or advanced the timeline")
					}
					e.Tick(time.UnixMilli(cutoff + 100))
					baseline.Tick(time.UnixMilli(cutoff + 100))
					got, ok := e.Result()
					want, _ := baseline.Result()
					if !ok || got.FinishedAt != cutoff || got.Participants[0].BotAssisted != prior || !reflect.DeepEqual(got, want) {
						t.Fatalf("post-cutoff event changed official result: got=%+v want=%+v", got, want)
					}
					if !reflect.DeepEqual(e.Snapshot("").Karts, baseline.Snapshot("").Karts) {
						t.Fatal("post-cutoff event changed terminal kart positions/times")
					}
				})
			}
		}
	}
}

// The shortened fixture crosses the real final gate on lap five, establishing
// the production first-finish+45s deadline instead of assigning that deadline.
func cutoffFixture(shortened, prior bool) *Engine {
	e := New(startFixture(5), DefaultTrack(), DefaultRules())
	if prior {
		e.Presence("a", "disconnect", time.UnixMilli(4000))
		e.Tick(time.UnixMilli(4100))
		e.Presence("a", "resume", time.UnixMilli(4200))
	}
	if shortened {
		winner := e.karts[1]
		winner.Laps = 4
		winner.LapTimesMs = []int64{1000, 1000, 1000, 1000}
		winner.lapStarted = 8000
		best := int64(1000)
		winner.BestLapMs = &best
		winner.NextCheckpoint = len(e.track.Gates) - 1
		gate := e.track.Gates[winner.NextCheckpoint]
		// Gates in this track's final horizontal straight cross the X axis.
		winner.X, winner.Y = gate.At+1, (gate.Min+gate.Max)/2
		e.cross(winner, Point{gate.At - 1, winner.Y}, 10000)
	}
	return e
}

func TestPreCutoffDisconnectStillRecordsAssistance(t *testing.T) {
	e := cutoffFixture(false, false)
	e.Tick(time.UnixMilli(e.deadline - 100))
	e.Presence("a", "disconnect", time.UnixMilli(e.deadline-40))
	e.Tick(time.UnixMilli(e.deadline + 1))
	result, ok := e.Result()
	if !ok || !result.Participants[0].BotAssisted || e.karts[0].X <= e.track.Grid[0].X {
		t.Fatal("real pre-cutoff takeover lost", result)
	}
}

func TestLaterDisconnectCannotContaminateEarlierAllFinishedResult(t *testing.T) {
	setup := func() *Engine {
		e := New(startFixture(5), DefaultTrack(), DefaultRules())
		e.Tick(time.UnixMilli(10000))
		gate := e.track.Gates[len(e.track.Gates)-1]
		for i, k := range e.karts {
			k.Laps = 4
			k.LapTimesMs = []int64{1000, 1000, 1000, 1000}
			k.lapStarted = 8000
			k.NextCheckpoint = len(e.track.Gates) - 1
			k.X, k.Y = gate.At+1, (gate.Min+gate.Max)/2
			if i < 4 {
				e.cross(k, Point{gate.At - 1, k.Y}, 10000)
			} else {
				k.X = gate.At - 1
				if err := e.InputAt(k.ActorID, 1, Controls{Throttle: 1}, time.UnixMilli(10000)); err != nil {
					t.Fatal(err)
				}
			}
		}
		return e
	}
	e, baseline := setup(), setup()
	e.Presence("a", "disconnect", time.UnixMilli(10040))
	e.Tick(time.UnixMilli(10050))
	baseline.Tick(time.UnixMilli(10050))
	got, ok := e.Result()
	want, _ := baseline.Result()
	if !ok || got.FinishedAt != 10033 || got.Participants[0].BotAssisted || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(e.Snapshot("").Karts, baseline.Snapshot("").Karts) {
		t.Fatal("later metadata contaminated earlier all-finished result", got, want)
	}
}

func TestPersonalFinishCannotGainAssistanceFromLaterDisconnect(t *testing.T) {
	for _, offset := range []int64{0, 1, 1000} {
		e := New(startFixture(2), DefaultTrack(), DefaultRules())
		elapsed := int64(1000)
		k := e.karts[0]
		k.Finished = true
		k.Laps = 5
		k.ElapsedMs = &elapsed
		e.Presence(k.UserID, "disconnect", time.UnixMilli(e.start.StartsAt+elapsed+offset))
		e.recordAssistance(e.start.StartsAt + elapsed + 2000)
		if k.BotAssisted {
			t.Fatalf("personal result gained assistance after finish offset=%d", offset)
		}
	}
	e := New(startFixture(2), DefaultTrack(), DefaultRules())
	elapsed := int64(1000)
	k := e.karts[0]
	k.Finished = true
	k.ElapsedMs = &elapsed
	k.disconnectedAt = e.start.StartsAt + 500
	e.recordAssistance(e.start.StartsAt + 2000)
	if !k.BotAssisted {
		t.Fatal("genuine pre-finish takeover lost")
	}
}
