package realtime

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	c "patra-world-social/server/internal/contracts"
	"strconv"
	"strings"
)

type Command struct {
	V         int             `json:"v"`
	CommandID string          `json:"commandId"`
	Seq       uint64          `json:"seq"`
	WorldID   string          `json:"worldId"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}
type Event struct {
	V            int             `json:"v"`
	EventID      string          `json:"eventId"`
	ServerTick   uint64          `json:"serverTick"`
	ServerTimeMs int64           `json:"serverTimeMs"`
	Type         string          `json:"type"`
	Payload      json.RawMessage `json:"payload"`
}

func ValidateCommand(raw []byte) (Command, error) {
	var command Command
	v, e := c.Object(raw)
	bad := errors.New("invalid command")
	if e != nil || !c.Keys(v, "v", "commandId", "seq", "worldId", "type", "payload") || len(raw) > 8192 || v["v"] != float64(1) || !c.Text(v["commandId"], 128) || !c.Text(v["worldId"], 128) || !c.Integer(v["seq"], 0, 9007199254740991) {
		return command, bad
	}
	p, ok := v["payload"].(map[string]any)
	if !ok {
		return command, bad
	}
	switch v["type"] {
	case "world.join", "world.leave", "lobby.list", "lobby.leave", "voice.leave", "voice.state", "voice.token", "voice.unmute", "session.resume":
	case "player.destination":
		ok = c.Point(p)
	case "player.input":
		ok = c.Number(p["dx"], -1, 1) && c.Number(p["dy"], -1, 1)
	case "player.interact":
		ok = c.Text(p["targetId"], 128)
	case "lobby.join":
		ok = c.Text(p["arenaId"], 128)
	case "duck.snapshot", "duck.clock", "race.snapshot", "penalty.snapshot", "match.return", "match.rematch":
		ok = c.Text(p["matchId"], 128)
	case "match.start":
		ok = c.Text(p["lobbyId"], 128)
	case "lobby.bot":
		_, ok = p["enabled"].(bool)
		ok = ok && c.Text(p["lobbyId"], 128)
	case "lobby.ready":
		_, ok = p["ready"].(bool)
		ok = ok && c.Text(p["lobbyId"], 128)
	case "race.input":
		ok = c.Text(p["matchId"], 128) && c.Integer(p["steer"], -1, 1) && c.Integer(p["throttle"], 0, 1) && c.Integer(p["brake"], 0, 1)
	case "penalty.choose":
		ok = c.Integer(p["zone"], 0, 3) && c.Text(p["matchId"], 128) && c.Text(p["turnId"], 128)
	case "duck.clock_reply":
		ok = c.Text(p["matchId"], 128) && c.Text(p["nonce"], 128) && c.Number(p["clientTimeMs"], 0, 9007199254740991)
	case "duck.shoot":
		ok = c.Number(p["x"], 0, 1) && c.Number(p["y"], 0, 1) && c.Text(p["matchId"], 128) && c.Number(p["clientTimeMs"], 0, 9007199254740991)
	case "chat.send":
		ok = c.Text(p["text"], 500) && p["channelId"] == "world" && c.Text(p["clientMessageId"], 128)
	case "voice.chat":
		ok = c.Text(p["text"], 500) && voiceMessageID(p["clientMessageId"])
	case "voice.moderate":
		ok = c.Text(p["target"], 128) && (p["action"] == "mute" || p["action"] == "kick")
	case "voice.join":
		ok = c.Integer(p["table"], 0, 1)
	default:
		ok = false
	}
	allowed := map[string][]string{"voice.leave": {}, "voice.state": {}, "voice.token": {}, "voice.unmute": {}, "voice.chat": {"clientMessageId", "text"}, "voice.moderate": {"target", "action"}, "penalty.snapshot": {"matchId"}, "match.return": {"matchId"}, "match.rematch": {"matchId"}, "penalty.choose": {"zone", "matchId", "turnId"}, "player.input": {"dx", "dy"}, "player.destination": {"x", "y"}, "player.interact": {"targetId"}, "lobby.list": {}, "lobby.leave": {}, "lobby.join": {"arenaId"}, "match.start": {"lobbyId"}, "lobby.bot": {"lobbyId", "enabled"}, "lobby.ready": {"lobbyId", "ready"}, "race.snapshot": {"matchId"}, "race.input": {"matchId", "steer", "throttle", "brake"}, "duck.snapshot": {"matchId"}, "duck.clock": {"matchId"}, "duck.clock_reply": {"matchId", "nonce", "clientTimeMs"}, "duck.shoot": {"matchId", "x", "y", "clientTimeMs"}, "chat.send": {"channelId", "clientMessageId", "text"}, "voice.join": {"table"}}
	if fields, known := allowed[func() string { s, _ := v["type"].(string); return s }()]; known && !c.Keys(p, fields...) {
		ok = false
	}
	if !ok {
		return command, bad
	}
	if e = json.Unmarshal(raw, &command); e != nil {
		return command, bad
	}
	return command, nil
}

// ValidateEvent checks the public envelope; detailed game payloads have their own contracts.
func ValidateEvent(raw []byte) (Event, error) {
	var event Event
	v, e := c.Object(raw)
	bad := errors.New("invalid event")
	if e != nil || !c.Keys(v, "v", "eventId", "serverTick", "serverTimeMs", "type", "payload") || v["v"] != float64(1) || !c.Text(v["eventId"], 128) || !c.Integer(v["serverTick"], 0, 9007199254740991) || !c.Integer(v["serverTimeMs"], 0, 9007199254740991) {
		return event, bad
	}
	if p, ok := v["payload"].(map[string]any); !ok || p == nil {
		return event, bad
	}
	switch v["type"] {
	case "world.snapshot", "presence.changed", "world.delta", "command.ack", "lobby.updated", "match.started", "match.result_status", "leaderboard.changed", "race.snapshot", "race.finished", "penalty.snapshot", "penalty.choice_ack", "penalty.turn_opened", "penalty.revealed", "duck.shot_resolved", "duck.snapshot", "duck.finished", "chat.message", "chat.deleted", "command.error", "voice.seat_granted", "voice.seat_revoked":
	default:
		return event, bad
	}
	if v["type"] == "leaderboard.changed" {
		p := v["payload"].(map[string]any)
		if !c.Keys(p, "worldId", "ruleset", "resultId") || !c.Text(p["worldId"], 128) || !c.Text(p["ruleset"], 128) || !c.Text(p["resultId"], 128) {
			return event, bad
		}
	}
	if v["type"] == "match.result_status" {
		p := v["payload"].(map[string]any)
		if !c.Keys(p, "matchId", "status") || !c.Text(p["matchId"], 128) {
			return event, bad
		}
		switch p["status"] {
		case "finalizing", "unconfirmed", "completed", "aborted_storage", "aborted_server", "abandoned", "unavailable":
		default:
			return event, bad
		}
	}
	e = json.Unmarshal(raw, &event)
	return event, e
}

// Voice retries use an admission-bound monotonic JS-safe sequence.
func voiceMessageID(value any) bool {
	id, ok := value.(string)
	if !ok {
		return false
	}
	parts := strings.Split(id, ":")
	if len(parts) != 2 || len(parts[0]) != 48 || strings.ToLower(parts[0]) != parts[0] {
		return false
	}
	if _, err := hex.DecodeString(parts[0]); err != nil {
		return false
	}
	seq, err := strconv.ParseUint(parts[1], 10, 64)
	return err == nil && seq > 0 && seq <= 9007199254740991 && strconv.FormatUint(seq, 10) == parts[1]
}
