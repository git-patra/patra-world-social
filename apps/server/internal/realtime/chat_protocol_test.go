package realtime

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestChatCommandBoundary(t *testing.T) {
	command := func(text string) []byte {
		b, _ := json.Marshal(map[string]any{"v": 1, "commandId": "ack-1", "seq": 1, "worldId": "w", "type": "chat.send", "payload": map[string]any{"channelId": "world", "clientMessageId": "retry-1", "text": text}})
		return b
	}
	if _, e := ValidateCommand(command(strings.Repeat("🎃", 500))); e != nil {
		t.Fatalf("500-codepoint world chat rejected: %v", e)
	}
	if _, e := ValidateCommand(command(strings.Repeat("🎃", 501))); e == nil {
		t.Fatal("501 codepoints accepted")
	}
	if _, e := ValidateCommand([]byte(`{"v":1,"commandId":"c","seq":1,"worldId":"w","type":"chat.send","payload":{"text":"missing identity"}}`)); e == nil {
		t.Fatal("legacy message without durable identity accepted")
	}
}
