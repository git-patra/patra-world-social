package duck

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestSharedSpawnAndCollisionFixture(t *testing.T) {
	raw, err := os.ReadFile("../../../../../contracts/duck.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Seed       int64
		Targets    []target
		Overlap    []target
		Collisions []struct {
			Aim      Aim
			TargetID int
		}
	}
	if err = json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	rng := NewRNG(f.Seed)
	for _, want := range f.Targets {
		got := spawn(rng, want.ID, want.Born, DefaultRules())
		if !reflect.DeepEqual(got, want) {
			t.Fatal(got, want)
		}
	}
	for _, c := range f.Collisions {
		e := newTest()
		e.Tick(time.UnixMilli(5000))
		e.targets = append([]target{}, f.Overlap...)
		r, err := shoot(e, 1, 5000, c.Aim)
		if err != nil || r.TargetID != c.TargetID {
			t.Fatal(r, err)
		}
	}
	asset, _ := os.ReadFile("../../../../../contracts/duck-rules.v1.json")
	if string(asset) != string(rulesAsset) {
		t.Fatal("guest/server rules drift")
	}
}
