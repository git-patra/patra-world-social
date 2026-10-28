package leaderboard

import (
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgtype"
	"net/http"
	httpapi "patra-world-social/server/internal/http"
)

func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()
	handle := func(detail bool) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Content-Type", "application/json")
			session, ok := httpapi.SessionFromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", 401)
				return
			}
			world := r.PathValue("world")
			var uuid pgtype.UUID
			if e := uuid.Scan(world); e != nil {
				writeHTTP(w, nil, ErrInvalid)
				return
			}
			var value any
			var e error
			if detail {
				id := r.PathValue("match")
				if e = uuid.Scan(id); e == nil {
					value, e = s.Result(r.Context(), session.User.ID, world, id)
				} else {
					e = ErrInvalid
				}
			} else {
				rules := r.URL.Query().Get("ruleset")
				if rules == "" {
					rules = CurrentRuleset
				}
				value, e = s.Board(r.Context(), session.User.ID, world, rules)
			}
			writeHTTP(w, value, e)
		}
	}
	mux.HandleFunc("GET /api/v1/leaderboards/{world}", handle(false))
	mux.HandleFunc("GET /api/v1/leaderboards/{world}/results/{match}", handle(true))
	return mux
}
func writeHTTP(w http.ResponseWriter, value any, e error) {
	if e != nil {
		code := 503
		message := "Podium sementara tidak tersedia."
		if errors.Is(e, ErrNotFound) {
			code = 404
			message = "Podium atau hasil tidak ditemukan."
		}
		if errors.Is(e, ErrInvalid) {
			code = 400
			message = "Permintaan podium tidak valid."
		}
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]string{"error": message})
		return
	}
	json.NewEncoder(w).Encode(value)
}
