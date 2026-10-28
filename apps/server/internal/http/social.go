package httpapi

import (
	"errors"
	"net/http"
	"patra-world-social/server/internal/chat"
	"patra-world-social/server/internal/moderation"
	"patra-world-social/server/internal/social"
	"patra-world-social/server/internal/worlds"
)

func registerSocialRoutes(mux *http.ServeMux, auth Auth, opt Options, protected func(http.ResponseWriter, *http.Request) bool, decode func(http.ResponseWriter, *http.Request, any) bool, jsonOut func(http.ResponseWriter, int, any), fail func(http.ResponseWriter, int, string)) {
	handle := func(pattern string, fn func(*http.Request, string) (any, error)) {
		mux.Handle(pattern, SessionMiddleware(auth, opt.Development, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" && !protected(w, r) {
				return
			}
			session, _ := SessionFromContext(r.Context())
			value, e := fn(r, session.User.ID)
			if e != nil {
				status, msg := 500, "Layanan sosial belum tersedia."
				switch {
				case errors.Is(e, social.ErrForbidden), errors.Is(e, worlds.ErrForbidden):
					status, msg = 403, "Tindakan tidak diizinkan."
				case errors.Is(e, worlds.ErrInviteInvalid):
					status, msg = 404, "Undangan tidak tersedia."
				case errors.Is(e, worlds.ErrFull):
					status, msg = 409, "Dunia mencapai batas 50 anggota."
				case errors.Is(e, social.ErrInput), errors.Is(e, worlds.ErrInput), errors.Is(e, chat.ErrInput):
					status, msg = 400, "Data tidak valid."
				}
				fail(w, status, msg)
				return
			}
			if value == nil {
				w.WriteHeader(204)
				return
			}
			jsonOut(w, 200, value)
		})))
	}
	// Decode through the same bounded strict JSON parser, buffering only its error response.
	read := func(r *http.Request, v any) error {
		w := &decodeSink{}
		if !decode(w, r, v) {
			return social.ErrInput
		}
		return nil
	}
	s := opt.Social
	handle("GET /api/v1/friends", func(r *http.Request, u string) (any, error) { return s.List(r.Context(), u) })
	handle("GET /api/v1/friends/search", func(r *http.Request, u string) (any, error) {
		return s.Search(r.Context(), u, r.URL.Query().Get("username"))
	})
	handle("GET /api/v1/blocks", func(r *http.Request, u string) (any, error) { return s.Blocks(r.Context(), u) })
	handle("POST /api/v1/friends/request", func(r *http.Request, u string) (any, error) {
		var q struct {
			Username string `json:"username"`
		}
		if e := read(r, &q); e != nil {
			return nil, e
		}
		return nil, s.RequestFriend(r.Context(), u, q.Username)
	})
	handle("POST /api/v1/friends/{request}/respond", func(r *http.Request, u string) (any, error) {
		var q struct {
			Accept bool `json:"accept"`
		}
		if e := read(r, &q); e != nil {
			return nil, e
		}
		return nil, s.RespondFriend(r.Context(), u, r.PathValue("request"), q.Accept)
	})
	for _, action := range []string{"unfriend", "block", "unblock"} {
		handle("POST /api/v1/friends/{user}/"+action, func(r *http.Request, u string) (any, error) {
			var q struct{}
			if e := read(r, &q); e != nil {
				return nil, e
			}
			if action == "unblock" {
				return nil, s.Unblock(r.Context(), u, r.PathValue("user"))
			}
			return nil, s.Remove(r.Context(), u, r.PathValue("user"), action == "block")
		})
	}
	handle("GET /api/v1/notifications", func(r *http.Request, u string) (any, error) { return opt.Worlds.Notifications(r.Context(), u) })
	handle("POST /api/v1/notifications/{invite}/accept", func(r *http.Request, u string) (any, error) {
		var q struct{}
		if e := read(r, &q); e != nil {
			return nil, e
		}
		return opt.Worlds.AcceptTargetedInvite(r.Context(), u, r.PathValue("invite"))
	})
	handle("GET /api/v1/worlds/{world}/chat", func(r *http.Request, u string) (any, error) {
		return opt.Chat.History(r.Context(), u, r.PathValue("world"), r.URL.Query().Get("before"))
	})
	handle("POST /api/v1/worlds/{world}/moderation", func(r *http.Request, u string) (any, error) {
		var q moderation.Action
		if e := read(r, &q); e != nil {
			return nil, e
		}
		return nil, opt.Moderation.Moderate(r.Context(), u, r.PathValue("world"), q)
	})
}

type decodeSink struct{}

func (*decodeSink) Header() http.Header         { return http.Header{} }
func (*decodeSink) Write(b []byte) (int, error) { return len(b), nil }
func (*decodeSink) WriteHeader(int)             {}
