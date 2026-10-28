package realtime

import (
	"strings"
	"testing"
)

func TestRaceOnlyAcceptsBoundedControls(t *testing.T) {
	raw := `{"v":1,"commandId":"c","seq":1,"worldId":"w","type":"race.input","payload":{"matchId":"m","throttle":1,"brake":0,"steer":-1}}`
	if _, err := ValidateCommand([]byte(raw)); err != nil {
		t.Fatal(err)
	}
	for _, extra := range []string{`,"x":100`, `,"laps":5`, `,"speed":999`} {
		if _, err := ValidateCommand([]byte(strings.Replace(raw, `"steer":-1}`, `"steer":-1`+extra+`}`, 1))); err == nil {
			t.Fatal("client authoritative field accepted")
		}
	}
	for field, original := range map[string]string{"throttle": "1", "brake": "0", "steer": "-1"} {
		for _, invalid := range []string{"99", "0.5", "null"} {
			changed := strings.Replace(raw, `"`+field+`":`+original, `"`+field+`":`+invalid, 1)
			if _, err := ValidateCommand([]byte(changed)); err == nil {
				t.Fatal("unbounded or noninteger control", field, invalid)
			}
		}
	}
}
