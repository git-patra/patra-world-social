package voice

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"github.com/livekit/protocol/auth"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWebhookSignatureTamperAndDuplicateAreSafe(t *testing.T) {
	s, _, _, _ := setup(t)
	p := newLiveKit("wss://test.livekit.cloud", "key", "secret")
	handler := s.Handler(p)
	body := []byte(`{"event":"participant_left","id":"same-event","room":{"name":"test-room"},"participant":{"identity":"unknown"}}`)
	hash := sha256.Sum256(body)
	token, err := auth.NewAccessToken("key", "secret").SetValidFor(time.Minute).SetSha256(base64.StdEncoding.EncodeToString(hash[:])).ToJWT()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		body  []byte
		token string
		want  int
	}{{body, "", 401}, {[]byte(`{}`), token, 401}, {body, token, 204}, {body, token, 204}} {
		r := httptest.NewRequest("POST", "/api/v1/voice/webhook", bytes.NewReader(test.body))
		r.Header.Set("Authorization", test.token)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != test.want {
			t.Fatalf("status %d want %d", w.Code, test.want)
		}
	}
	if s.Status() != "ready" {
		t.Fatal("duplicate event changed admission state")
	}
}
