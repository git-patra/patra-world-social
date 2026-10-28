package realtime

import (
	"encoding/json"
	"testing"
)

func TestLobbyStrictPayloads(t *testing.T) {
	for _, v := range []struct {
		kind, payload string
		valid         bool
	}{{"lobby.join", `{"arenaId":"race"}`, true}, {"lobby.join", `{}`, false}, {"lobby.join", `{"arenaId":"race","userId":"spoof"}`, false}, {"lobby.ready", `{"lobbyId":"id","ready":true}`, true}, {"lobby.ready", `{"ready":true}`, false}, {"match.start", `{"lobbyId":"id","hostId":"spoof"}`, false}, {"lobby.list", `{}`, true}, {"lobby.leave", `{"userId":"other"}`, false}, {"lobby.bot", `{"lobbyId":"id","enabled":true}`, true}} {
		raw, _ := json.Marshal(Command{V: 1, CommandID: "c", WorldID: "w", Type: v.kind, Payload: json.RawMessage(v.payload)})
		_, e := ValidateCommand(raw)
		if (e == nil) != v.valid {
			t.Errorf("%s %s: %v", v.kind, v.payload, e)
		}
	}
}
