package history

import (
	"encoding/json"
	"errors"
	"net/http"
	httpapi "patra-world-social/server/internal/http"
	"strconv"
	"time"
)

// Handler is mounted behind the existing SessionMiddleware. There is no official
// result write endpoint and no caller-selected account ID for personal history.
func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/history", func(w http.ResponseWriter, r *http.Request) {
		session, ok := httpapi.SessionFromContext(r.Context())
		if !ok {
			http.Error(w, "unauthorized", 401)
			return
		}
		q := r.URL.Query()
		f := Filter{Game: q.Get("game"), World: q.Get("world"), Status: q.Get("status")}
		var err error
		if q.Has("userId") {
			writeHTTP(w, nil, ErrInvalid)
			return
		}
		if q.Has("limit") {
			f.Limit, err = strconv.Atoi(q.Get("limit"))
			if err != nil {
				writeHTTP(w, nil, ErrInvalid)
				return
			}
		}
		for name, target := range map[string]**time.Time{"from": &f.From, "to": &f.To} {
			if q.Has(name) {
				v, e := time.Parse(time.RFC3339, q.Get(name))
				if e != nil {
					writeHTTP(w, nil, ErrInvalid)
					return
				}
				*target = &v
			}
		}
		page, e := s.ListHistory(r.Context(), session.User.ID, f, q.Get("cursor"))
		writeHTTP(w, page, e)
	})
	mux.HandleFunc("GET /api/v1/history/{match}", func(w http.ResponseWriter, r *http.Request) {
		session, ok := httpapi.SessionFromContext(r.Context())
		if !ok {
			http.Error(w, "unauthorized", 401)
			return
		}
		record, e := s.Detail(r.Context(), session.User.ID, r.PathValue("match"))
		writeHTTP(w, record, e)
	})
	return mux
}
func writeHTTP(w http.ResponseWriter, value any, e error) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	if e != nil {
		status := 503
		message := "History sementara tidak tersedia."
		if errors.Is(e, ErrNotFound) {
			status = 404
			message = "Hasil tidak ditemukan."
		}
		if errors.Is(e, ErrInvalid) {
			status = 400
			message = "Filter history tidak valid."
		}
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]string{"error": message})
		return
	}
	json.NewEncoder(w).Encode(value)
}
