package pingpong

import (
	"encoding/json"
	"reflect"
	"testing"
)

// The engine intentionally mirrors these fixed constants. Every published field
// must stay pinned until its corresponding implementation and parity tests change.
func TestPublishedRulesMatchEngineConstants(t *testing.T) {
	var got map[string]any
	if err := json.Unmarshal(rules, &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"id": "pingpong-v1", "version": float64(1), "tickMs": float64(50), "maxCatchupSteps": float64(4), "deadlineMs": float64(120000), "disconnectGraceMs": float64(15000), "inputExpiryMs": float64(250), "targetScore": float64(7), "winBy": float64(2), "capScore": float64(11), "paddleSpeed": float64(1), "paddleHalfHeight": .12, "ballRadius": .02, "ballSpeedX": .6, "serveSpeedY": .3, "paddleX": []any{.06, .94}, "botSpeed": .65, "botDeadzone": .04}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("published rules changed without engine parity review: got %v want %v", got, want)
	}
}
