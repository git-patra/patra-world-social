package realtime

import (
	"errors"
	c "patra-world-social/server/internal/contracts"
	"patra-world-social/server/internal/world"
)

func validateActivityPayload(raw any) error {
	bad := errors.New("invalid activity snapshot")
	p, ok := raw.(map[string]any)
	if !ok || len(p) != 8 || !c.Keys(p, "worldId", "areaId", "incarnation", "generation", "serverTimeMs", "tick", "kind", "payload") || !c.Text(p["worldId"], 128) || !c.Text(p["incarnation"], 128) || (p["kind"] != "area" && p["kind"] != "vehicles") {
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
	if p["kind"] == "vehicles" {
		return validateVehiclesPayload(p)
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

func validateRideIntent(p map[string]any) bool {
	fields := []string{"incarnation", "areaGeneration", "vehicleId", "generation", "action"}
	if p["action"] == "drive" {
		fields = append(fields, "dx", "dy")
	} else if p["action"] != "mount" && p["action"] != "dismount" {
		return false
	}
	if len(p) != len(fields) || !c.Keys(p, fields...) || !c.Text(p["incarnation"], 128) || !c.Integer(p["areaGeneration"], 0, 9007199254740991) || !c.Integer(p["generation"], 1, 9007199254740991) {
		return false
	}
	validID := false
	for _, id := range []string{"bicycle-0", "bicycle-1", "car-2", "car-3"} {
		if p["vehicleId"] == id {
			validID = true
		}
	}
	return validID && (p["action"] != "drive" || c.Number(p["dx"], -1, 1) && c.Number(p["dy"], -1, 1))
}
func validateVehiclesPayload(p map[string]any) error {
	bad := errors.New("invalid vehicles snapshot")
	s, ok := p["payload"].(map[string]any)
	if !ok || p["areaId"] != "race" || !c.Keys(s, "vehicles") || len(s) != 1 || !c.Integer(p["generation"], 1, 9007199254740991) {
		return bad
	}
	vehicles, ok := s["vehicles"].([]any)
	if !ok || len(vehicles) != 4 {
		return bad
	}
	seen := map[string]bool{}
	riders := map[string]bool{}
	for _, raw := range vehicles {
		v, ok := raw.(map[string]any)
		if !ok || len(v) != 7 || !c.Keys(v, "id", "kind", "generation", "riderId", "x", "y", "heading") {
			return bad
		}
		id, _ := v["id"].(string)
		kind := "car"
		if id == "bicycle-0" || id == "bicycle-1" {
			kind = "bicycle"
		} else if id != "car-2" && id != "car-3" {
			return bad
		}
		rider, ok := v["riderId"].(string)
		if !ok || len(rider) > 128 || rider != "" && riders[rider] || seen[id] || v["kind"] != kind || !c.Integer(v["generation"], 1, 9007199254740991) || !c.Number(v["x"], 0, 63) || !c.Number(v["y"], 0, 63) || !c.Number(v["heading"], -180, 180) {
			return bad
		}
		if !world.RideDriveable(v["x"].(float64), v["y"].(float64)) {
			return bad
		}
		seen[id] = true
		riders[rider] = true
	}
	return nil
}
