package blaster

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestSharedGoldenAndRules(t *testing.T) {
	raw, err := os.ReadFile("../../../../../contracts/blaster.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Started int64
		Cases   []struct {
			Name         string
			Count, Steps int
			Auto         bool
			DX, DY       float64
			Fire         bool
			Expected     json.RawMessage
		}
	}
	if err = json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	for _, c := range f.Cases {
		t.Run(c.Name, func(t *testing.T) {
			e := fixture(c.Count)
			e.Input("a", 1, Controls{DX: c.DX, DY: c.DY, Fire: c.Fire}, time.UnixMilli(f.Started))
			for i := 1; i <= c.Steps; i++ {
				if c.Auto {
					for j, p := range e.players {
						if len(e.bots) == 0 {
							continue
						}
						target := e.bots[0]
						for _, b := range e.bots {
							if math.Hypot(p.X-b.X, p.Y-b.Y) < math.Hypot(p.X-target.X, p.Y-target.Y) {
								target = b
							}
						}
						e.Input(string(rune('a'+j)), uint64(i+1), Controls{Aim: math.Atan2(target.Y-p.Y, target.X-p.X), Fire: true}, time.UnixMilli(f.Started+int64(i-1)*50))
					}
				}
				e.Tick(time.UnixMilli(f.Started + int64(i)*50))
			}
			if c.Steps == 0 {
				e.Tick(time.UnixMilli(181000))
			}
			s := e.Snapshot()
			var result any
			if s.Result != nil {
				result = s.Result.Blaster
			}
			b, _ := json.Marshal(map[string]any{"phase": s.Phase, "frame": s.Frame, "wave": s.Wave, "players": s.Players, "bots": s.Bots, "projectiles": s.Projectiles, "result": result})
			var got, want any
			json.Unmarshal(b, &got)
			json.Unmarshal(c.Expected, &want)
			compareGolden(t, got, want)
		})
	}
	asset, err := os.ReadFile("../../../../../contracts/blaster-rules.v1.json")
	if err != nil || string(asset) != string(rulesJSON) {
		t.Fatal("rules drift", err)
	}
}
func compareGolden(t *testing.T, a, b any) {
	t.Helper()
	switch a := a.(type) {
	case float64:
		v, ok := b.(float64)
		if !ok || math.Abs(a-v) > 1e-9 {
			t.Fatalf("got %v want %v", a, b)
		}
	case []any:
		v, ok := b.([]any)
		if !ok || len(a) != len(v) {
			t.Fatalf("length %v vs %v", a, b)
		}
		for i := range a {
			compareGolden(t, a[i], v[i])
		}
	case map[string]any:
		v, ok := b.(map[string]any)
		if !ok || len(a) != len(v) {
			t.Fatalf("shape %v vs %v", a, b)
		}
		for k, x := range a {
			compareGolden(t, x, v[k])
		}
	default:
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("got %v want %v", a, b)
		}
	}
}
