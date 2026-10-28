package world

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"
)

func TestRaceRulesRequireEveryPhysicsCoefficient(t *testing.T) {
	var v map[string]any
	json.Unmarshal(RulesAsset, &v)
	r := v["race"].(map[string]any)
	delete(r, "acceleration")
	delete(v, "checksum")
	raw, _ := json.Marshal(v)
	v["checksum"] = fmt.Sprintf("%x", sha256.Sum256(raw))
	raw, _ = json.Marshal(v)
	if _, err := LoadRules(raw); err == nil {
		t.Fatal("missing acceleration accepted")
	}
}
