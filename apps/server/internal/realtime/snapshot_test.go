package realtime

import (
	"encoding/json"
	"os"
	"testing"
)

func TestSharedSnapshotContract(t *testing.T) {
	raw, e := os.ReadFile("../../../../contracts/snapshots.golden.json")
	if e != nil {
		t.Fatal(e)
	}
	var fixtures struct{ Valid, Invalid []json.RawMessage }
	if e = json.Unmarshal(raw, &fixtures); e != nil {
		t.Fatal(e)
	}
	for _, raw := range fixtures.Valid {
		if _, e = ValidateSnapshot(raw, "a"); e != nil {
			t.Fatal(e)
		}
	}
	for _, raw := range fixtures.Invalid {
		if _, e = ValidateSnapshot(raw, "a"); e == nil {
			t.Fatalf("invalid accepted %s", raw)
		}
	}
}
