package realtime

import (
	"fmt"
	"testing"
)

func TestDuckClockAndShotStrictPayloads(t *testing.T) {
	for _, tc := range []struct {
		kind, payload string
		valid         bool
	}{{"duck.snapshot", `{"matchId":"m"}`, true}, {"duck.clock", `{"matchId":"m"}`, true}, {"duck.clock_reply", `{"matchId":"m","nonce":"abc","clientTimeMs":4}`, true}, {"duck.shoot", `{"matchId":"m","x":0.5,"y":0.5,"clientTimeMs":4}`, true}, {"duck.shoot", `{"x":0.5,"y":0.5}`, false}, {"duck.shoot", `{"matchId":"m","x":0.5,"y":0.5,"clientTimeMs":4,"score":999}`, false}, {"duck.shoot", `{"matchId":"m","x":0.5,"y":0.5,"clientTimeMs":-1}`, false}} {
		raw := fmt.Sprintf(`{"v":2,"commandId":"c","seq":1,"worldId":"w","type":%q,"payload":%s}`, tc.kind, tc.payload)
		_, err := ValidateCommand([]byte(raw))
		if (err == nil) != tc.valid {
			t.Fatalf("%s %s: %v", tc.kind, tc.payload, err)
		}
	}
}
