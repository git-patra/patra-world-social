package realtime

import (
	"encoding/json"
	"os"
	"testing"
)

func TestValidateCommand(t *testing.T) {
	valid := `{"v":1,"commandId":"c1","seq":0,"worldId":"private","type":"penalty.choose","payload":{"zone":3,"matchId":"m","turnId":"m:1"}}`
	if _, e := ValidateCommand([]byte(valid)); e != nil {
		t.Fatal(e)
	}
	for _, s := range []string{`{}`, `{"v":2}`, `{"v":1,"commandId":"c","seq":0,"worldId":"w","type":"penalty.choose","payload":{"zone":4}}`, `{"v":1,"commandId":"c","seq":0,"worldId":"w","type":"penalty.choose","payload":{"zone":null}}`, `{"v":1,"commandId":"c","seq":0,"worldId":"w","type":"player.destination","payload":{"x":1}}`} {
		if _, e := ValidateCommand([]byte(s)); e == nil {
			t.Fatal("accepted", s)
		}
	}
}

func TestGoldenCommands(t *testing.T) {
	data, e := os.ReadFile("../../../../contracts/commands.golden.json")
	if e != nil {
		t.Fatal(e)
	}
	var fixture struct {
		Valid   []json.RawMessage `json:"valid"`
		Invalid []json.RawMessage `json:"invalid"`
	}
	if e = json.Unmarshal(data, &fixture); e != nil {
		t.Fatal(e)
	}
	for _, raw := range fixture.Valid {
		if _, e := ValidateCommand(raw); e != nil {
			t.Fatal(string(raw), e)
		}
	}
	for _, raw := range fixture.Invalid {
		if _, e := ValidateCommand(raw); e == nil {
			t.Fatal("accepted", string(raw))
		}
	}
}

func TestGoldenEvents(t *testing.T) {
	raw, e := os.ReadFile("../../../../contracts/events.golden.json")
	if e != nil {
		t.Fatal(e)
	}
	var f struct {
		Valid   []json.RawMessage `json:"valid"`
		Invalid []json.RawMessage `json:"invalid"`
	}
	if e = json.Unmarshal(raw, &f); e != nil {
		t.Fatal(e)
	}
	for _, data := range f.Valid {
		if _, e = ValidateEvent(data); e != nil {
			t.Error(e)
		}
	}
	for _, data := range f.Invalid {
		if _, e = ValidateEvent(data); e == nil {
			t.Error("accepted", string(data))
		}
	}
}

func TestVoiceMessageSequenceContract(t *testing.T) {
	prefix := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa:"
	for _, id := range []string{prefix + "1", prefix + "9007199254740991"} {
		if !voiceMessageID(id) {
			t.Fatal("valid ID rejected", id)
		}
	}
	for _, id := range []string{"random-id", prefix + "0", prefix + "01", prefix + "+1", prefix + "1e0", prefix + "9007199254740992", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA:1"} {
		raw, _ := json.Marshal(map[string]any{"v": 1, "commandId": "c", "seq": 1, "worldId": "w", "type": "voice.chat", "payload": map[string]string{"clientMessageId": id, "text": "text"}})
		if _, err := ValidateCommand(raw); err == nil {
			t.Fatal("invalid ID accepted", id)
		}
	}
}
