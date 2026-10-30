package realtime

import (
	"errors"
	"math"
	c "patra-world-social/server/internal/contracts"
	"patra-world-social/server/internal/world"
)

func validateActivityPayload(raw any) error {
	bad := errors.New("invalid activity snapshot")
	p, ok := raw.(map[string]any)
	if !ok || len(p) != 8 || !c.Keys(p, "worldId", "areaId", "incarnation", "generation", "serverTimeMs", "tick", "kind", "payload") || !c.Text(p["worldId"], 128) || !c.Text(p["incarnation"], 128) || (p["kind"] != "area" && p["kind"] != "vehicles" && p["kind"] != "football" && p["kind"] != "claw") {
		return bad
	}
	id, _ := p["areaId"].(string)
	if _, ok := world.ActivityArea(id); !ok && !(p["kind"] == "claw" && id == "claw-1") {
		return bad
	}
	for _, k := range []string{"generation", "serverTimeMs", "tick"} {
		if !c.Integer(p[k], 0, 9007199254740991) {
			return bad
		}
	}
	if p["kind"] == "claw" {
		return validateClawPayload(p)
	}
	if p["kind"] == "football" {
		return validateFootballPayload(p)
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

func validateKickIntent(p map[string]any) bool {
	return len(p) == 6 && c.Keys(p, "incarnation", "areaGeneration", "generation", "action", "dx", "dy") && c.Text(p["incarnation"], 128) && c.Integer(p["areaGeneration"], 0, 9007199254740991) && c.Integer(p["generation"], 1, 9007199254740991) && p["action"] == "kick" && c.Number(p["dx"], -1, 1) && c.Number(p["dy"], -1, 1) && (p["dx"] != float64(0) || p["dy"] != float64(0))
}
func validateFootballPayload(p map[string]any) error {
	bad := errors.New("invalid football snapshot")
	s, ok := p["payload"].(map[string]any)
	if !ok || len(s) != 1 || !c.Keys(s, "ball") || p["areaId"] != "penalty" || !c.Integer(p["generation"], 1, 9007199254740991) {
		return bad
	}
	b, ok := s["ball"].(map[string]any)
	if !ok || len(b) != 5 || !c.Keys(b, "x", "y", "vx", "vy", "visible") || !c.Number(b["x"], 0, 63) || !c.Number(b["y"], 0, 63) || !c.Number(b["vx"], -8, 8) || !c.Number(b["vy"], -8, 8) {
		return bad
	}
	visible, ok := b["visible"].(bool)
	if !ok || !world.FootballInside(b["x"].(float64), b["y"].(float64)) || math.Hypot(b["vx"].(float64), b["vy"].(float64)) > 8.000000001 || !visible && (b["vx"] != float64(0) || b["vy"] != float64(0)) {
		return bad
	}
	return nil
}

func validateClawIntent(p map[string]any) bool {
	fields := []string{"incarnation", "areaGeneration", "generation", "machineId", "action"}
	if p["action"] == "claw.move" {
		fields = append(fields, "dx", "dy")
	} else if p["action"] != "claw.enter" && p["action"] != "claw.leave" && p["action"] != "claw.drop" {
		return false
	}
	return len(p) == len(fields) && c.Keys(p, fields...) && c.Text(p["incarnation"], 128) && p["areaGeneration"] == float64(0) && c.Integer(p["generation"], 1, 9007199254740991) && p["machineId"] == "claw-1" && (p["action"] != "claw.move" || c.Number(p["dx"], -1, 1) && c.Number(p["dy"], -1, 1))
}
func validateClawPayload(p map[string]any) error {
	bad := errors.New("invalid claw snapshot")
	s, ok := p["payload"].(map[string]any)
	if !ok || len(s) != 1 || !c.Keys(s, "claw") || p["areaId"] != "claw-1" || !c.Integer(p["generation"], 1, 9007199254740991) {
		return bad
	}
	b, ok := s["claw"].(map[string]any)
	if !ok || len(b) != 9 || !c.Keys(b, "operatorId", "operatorName", "leaseUntil", "x", "y", "phase", "phaseAt", "dropId", "prizeId") || !c.Number(b["x"], 0, 1) || !c.Number(b["y"], 0, 1) {
		return bad
	}
	for _, k := range []string{"leaseUntil", "phaseAt", "dropId"} {
		if !c.Integer(b[k], 0, 9007199254740991) {
			return bad
		}
	}
	for _, k := range []string{"operatorId", "operatorName"} {
		v, ok := b[k].(string)
		if !ok || len(v) > 128 {
			return bad
		}
	}
	prize, ok := b["prizeId"].(string)
	if !ok {
		return bad
	}
	validPrize := prize == ""
	for _, v := range world.ClawConfig.Presets {
		if v.ID == prize {
			validPrize = true
		}
	}
	if !validPrize {
		return bad
	}
	switch b["phase"] {
	case "idle":
		if b["operatorId"] != "" || b["operatorName"] != "" || b["leaseUntil"] != float64(0) || prize != "" {
			return bad
		}
	case "aiming", "dropping", "lifting", "releasing":
		if !c.Text(b["operatorId"], 128) || !c.Text(b["operatorName"], 128) || b["leaseUntil"] == float64(0) || b["phase"] == "aiming" && prize != "" {
			return bad
		}
	default:
		return bad
	}
	return nil
}
