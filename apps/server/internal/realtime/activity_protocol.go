package realtime

import (
	"errors"
	c "patra-world-social/server/internal/contracts"
	"patra-world-social/server/internal/world"
)

func validateActivityPayload(raw any) error {
	bad := errors.New("invalid activity snapshot")
	p, ok := raw.(map[string]any)
	if !ok || len(p) != 8 || !c.Keys(p, "worldId", "areaId", "incarnation", "generation", "serverTimeMs", "tick", "kind", "payload") || !c.Text(p["worldId"], 128) || !c.Text(p["incarnation"], 128) || p["kind"] != "area" {
		return bad
	}
	id, _ := p["areaId"].(string)
	if _, ok := world.ActivityArea(id); !ok {
		return bad
	}
	for _, k := range []string{"generation", "serverTimeMs", "tick"} {
		if !c.Integer(p[k], 0, 9007199254740991) {
			return bad
		}
	}
	s, ok := p["payload"].(map[string]any)
	if !ok || len(s) != 5 || !c.Keys(s, "phase", "matchId", "startsAt", "clearAt", "resetAt") {
		return bad
	}
	for _, k := range []string{"startsAt", "clearAt", "resetAt"} {
		if !c.Integer(s[k], 0, 9007199254740991) {
			return bad
		}
	}
	switch s["phase"] {
	case "OPEN":
		if s["matchId"] != "" {
			return bad
		}
	case "CLEARING", "RESERVED", "ACTIVE", "RESETTING":
		if !c.Text(s["matchId"], 128) || p["generation"] == float64(0) {
			return bad
		}
		if s["phase"] == "CLEARING" && s["clearAt"] == float64(0) || s["phase"] == "ACTIVE" && s["startsAt"] == float64(0) || s["phase"] == "RESETTING" && s["resetAt"] == float64(0) {
			return bad
		}
	default:
		return bad
	}
	return nil
}
