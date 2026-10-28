package realtime

import "testing"

func TestLeaderboardInvalidationContract(t *testing.T) {
	prefix := `{"v":2,"eventId":"fixture","serverTick":1,"serverTimeMs":1,"type":"leaderboard.changed","payload":`
	for _, p := range []string{`{"worldId":"w","ruleset":"duck-v1@1","resultId":"r"}`} {
		if _, e := ValidateEvent([]byte(prefix + p + `}`)); e != nil {
			t.Fatal(e)
		}
	}
	for _, p := range []string{`{"worldId":"w","resultId":"r"}`, `{"worldId":"w","ruleset":"duck-v1@1","resultId":"r","score":100}`, `{"worldId":"","ruleset":"duck-v1@1","resultId":"r"}`} {
		if _, e := ValidateEvent([]byte(prefix + p + `}`)); e == nil {
			t.Fatal("invalid public payload", p)
		}
	}
}
