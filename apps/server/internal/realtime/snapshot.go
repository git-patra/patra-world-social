package realtime

import (
	"encoding/json"
	"errors"
	c "patra-world-social/server/internal/contracts"
	"patra-world-social/server/internal/world"
)

// ValidateSnapshot is the Go half of the shared public wire contract.
func ValidateSnapshot(raw []byte, worldID string) (PublicSnapshot, error) {
	var s PublicSnapshot
	bad := errors.New("invalid snapshot")
	v, e := c.Object(raw)
	if e != nil || !c.Keys(v, "worldId", "selfId", "sessionId", "epoch", "mapId", "mapChecksum", "rulesId", "rulesChecksum", "protocolVersion", "tick", "timeMs", "avatars") || v["worldId"] != worldID || v["protocolVersion"] != float64(2) || !c.Integer(v["epoch"], 1, 9007199254740991) || !c.Integer(v["tick"], 0, 9007199254740991) || !c.Integer(v["timeMs"], 0, 9007199254740991) {
		return s, bad
	}
	for _, k := range []string{"worldId", "selfId", "sessionId", "mapId", "mapChecksum", "rulesId", "rulesChecksum"} {
		if !c.Text(v[k], 128) {
			return s, bad
		}
	}
	var m world.MapDefinition
	var r world.Rules
	json.Unmarshal(world.MapAsset, &m)
	json.Unmarshal(world.RulesAsset, &r)
	if v["mapId"] != m.ID || v["mapChecksum"] != m.Checksum || v["rulesId"] != r.ID || v["rulesChecksum"] != r.Checksum {
		return s, bad
	}
	avatars, ok := v["avatars"].([]any)
	if !ok || len(avatars) < 1 || len(avatars) > 20 {
		return s, bad
	}
	ids := map[string]bool{}
	for _, raw := range avatars {
		a, ok := raw.(map[string]any)
		if !ok || !c.Keys(a, "id", "name", "status", "x", "y", "seq") || !c.Text(a["id"], 128) || !c.Text(a["name"], 40) || !c.Number(a["x"], 0, 63) || !c.Number(a["y"], 0, 63) || !c.Integer(a["seq"], 0, 9007199254740991) {
			return s, bad
		}
		id := a["id"].(string)
		if ids[id] {
			return s, bad
		}
		ids[id] = true
		switch a["status"] {
		case "available", "playing", "AFK", "voice":
		default:
			return s, bad
		}
	}
	if !ids[v["selfId"].(string)] {
		return s, bad
	}
	e = json.Unmarshal(raw, &s)
	return s, e
}
