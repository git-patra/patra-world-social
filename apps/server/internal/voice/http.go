package voice

import (
	"encoding/json"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/webhook"
	"net/http"
)

// Webhook signatures authenticate the exact body hash; events are hints only,
// so duplicate/out-of-order deliveries have no admission-changing side effects.
func (s *Service) Handler(p *LiveKit) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/voice/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(map[string]any{"status": s.Status(), "zones": TableZones})
	})
	mux.HandleFunc("POST /api/v1/voice/webhook", func(w http.ResponseWriter, r *http.Request) {
		if p == nil {
			http.Error(w, "voice unavailable", 503)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		_, err := webhook.ReceiveWebhookEvent(r, auth.NewSimpleKeyProvider(p.key, p.secret))
		if err != nil {
			http.Error(w, "invalid signature", 401)
			return
		}
		w.WriteHeader(204)
	})
	return mux
}
