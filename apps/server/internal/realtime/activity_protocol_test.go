package realtime

import (
	"encoding/json"
	"os"
	"testing"
)

func TestActivityGoldens(t *testing.T) {
	raw, e := os.ReadFile("../../../../contracts/activities.golden.json")
	if e != nil {
		t.Fatal(e)
	}
	var g struct {
		Valid   []any
		Invalid []any
	}
	json.Unmarshal(raw, &g)
	for _, v := range g.Valid {
		if e := validateActivityPayload(v); e != nil {
			t.Fatal(e)
		}
	}
	for _, v := range g.Invalid {
		if e := validateActivityPayload(v); e == nil {
			t.Fatal("invalid activity accepted")
		}
	}
}
