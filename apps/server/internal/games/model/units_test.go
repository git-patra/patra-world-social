package model

import (
	"encoding/json"
	"os"
	"testing"
)

func TestSharedMovementUnits(t *testing.T) {
	data, e := os.ReadFile("../../../../../contracts/movement.golden.json")
	if e != nil {
		t.Fatal(e)
	}
	var f struct {
		SubunitsPerTile int `json:"subunitsPerTile"`
		Examples        []struct {
			Tiles    float64 `json:"tiles"`
			Subunits int64   `json:"subunits"`
		} `json:"examples"`
	}
	if e = json.Unmarshal(data, &f); e != nil {
		t.Fatal(e)
	}
	if Subunits(f.SubunitsPerTile) != SubunitsPerTile {
		t.Fatal("unit mismatch")
	}
	for _, example := range f.Examples {
		if int64(example.Tiles*float64(SubunitsPerTile)) != example.Subunits {
			t.Fatal(example)
		}
	}
}
