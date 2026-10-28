package pingpong

import (
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"
)

func TestSharedGolden(t *testing.T) {
	raw, err := os.ReadFile("../../../../../contracts/pingpong.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Started int64
		Cases   []struct {
			Name           string
			Axis, PeerAxis float64
			Steps          int
			Scores         [2]int64
			Paddles        [2]float64
			Ball           Ball
			Phase          string
		}
	}
	if err = json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	for _, v := range f.Cases {
		t.Run(v.Name, func(t *testing.T) {
			e := fixture()
			e.Serve("u", time.UnixMilli(f.Started))
			e.Input("u", 1, v.Axis, time.UnixMilli(f.Started))
			e.Input("v", 1, v.PeerAxis, time.UnixMilli(f.Started))
			for i := 1; i <= v.Steps; i++ {
				e.Tick(time.UnixMilli(f.Started + int64(i)*50))
			}
			s := e.Snapshot()
			if s.Phase != v.Phase || s.Scores != v.Scores {
				t.Fatal(s, v)
			}
			for i, x := range []float64{s.Paddles[0], s.Paddles[1], s.Ball.X, s.Ball.Y, s.Ball.VX, s.Ball.VY} {
				want := []float64{v.Paddles[0], v.Paddles[1], v.Ball.X, v.Ball.Y, v.Ball.VX, v.Ball.VY}[i]
				if math.Abs(x-want) > 1e-9 {
					t.Fatalf("field %d got %v want %v", i, x, want)
				}
			}
		})
	}
	asset, _ := os.ReadFile("../../../../../contracts/pingpong-rules.v1.json")
	if string(asset) != string(rules) {
		t.Fatal("rules drift")
	}
}
func TestWallBounceAndGraceResume(t *testing.T) {
	e := fixture()
	e.playing = true
	e.ball = Ball{X: .5, Y: .975, VX: .6, VY: .3}
	e.Tick(time.UnixMilli(1050))
	if e.ball.Y > .98 || e.ball.VY >= 0 {
		t.Fatal(e.ball)
	}
	e.Presence("u", "disconnect", time.UnixMilli(2000))
	e.Presence("u", "resume", time.UnixMilli(16999))
	e.Tick(time.UnixMilli(17000))
	if e.Snapshot().Result != nil {
		t.Fatal("early resume forfeited")
	}
}
