package httpapi

import (
	"errors"
	"net/http"
	"patra-world-social/server/internal/worlds"
)

func registerWorldRoutes(mux *http.ServeMux, auth Auth, service *worlds.Service, development bool,
	protected func(http.ResponseWriter, *http.Request) bool,
	decode func(http.ResponseWriter, *http.Request, any) bool,
	jsonOut func(http.ResponseWriter, int, any), fail func(http.ResponseWriter, int, string)) {
	handle := func(pattern string, fn func(http.ResponseWriter, *http.Request, string)) {
		method := pattern[:3]
		if len(pattern) >= 4 && pattern[:4] == "POST" {
			method = "POST"
		}
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if method != "GET" && !protected(w, r) {
				return
			}
			session, _ := SessionFromContext(r.Context())
			fn(w, r, session.User.ID)
		})
		mux.Handle(pattern, SessionMiddleware(auth, development, h))
	}
	sendError := func(w http.ResponseWriter, err error) {
		switch {
		case errors.Is(err, worlds.ErrForbidden):
			fail(w, 403, "Akses dunia ditolak.")
		case errors.Is(err, worlds.ErrInviteInvalid):
			fail(w, 404, "Undangan tidak tersedia.")
		case errors.Is(err, worlds.ErrFull):
			fail(w, 409, "Dunia mencapai batas 50 anggota.")
		case errors.Is(err, worlds.ErrInput):
			fail(w, 400, "Data dunia tidak valid.")
		default:
			fail(w, 500, "Kesalahan server.")
		}
	}
	type subject struct {
		UserID string `json:"userId"`
	}
	handle("GET /api/v1/worlds", func(w http.ResponseWriter, r *http.Request, id string) {
		v, e := service.ListWorlds(r.Context(), id)
		if e != nil {
			sendError(w, e)
			return
		}
		jsonOut(w, 200, v)
	})
	handle("POST /api/v1/worlds", func(w http.ResponseWriter, r *http.Request, id string) {
		var q struct {
			Name string `json:"name"`
		}
		if !decode(w, r, &q) {
			return
		}
		v, e := service.CreateWorld(r.Context(), id, q.Name)
		if e != nil {
			sendError(w, e)
			return
		}
		jsonOut(w, 201, v)
	})
	handle("GET /api/v1/worlds/{world}", func(w http.ResponseWriter, r *http.Request, id string) {
		v, e := service.GetWorld(r.Context(), id, r.PathValue("world"))
		if e != nil {
			sendError(w, e)
			return
		}
		jsonOut(w, 200, v)
	})
	handle("GET /api/v1/worlds/{world}/members", func(w http.ResponseWriter, r *http.Request, id string) {
		v, e := service.ListMembers(r.Context(), id, r.PathValue("world"))
		if e != nil {
			sendError(w, e)
			return
		}
		jsonOut(w, 200, v)
	})
	handle("GET /api/v1/worlds/{world}/bans", func(w http.ResponseWriter, r *http.Request, id string) {
		v, e := service.ListBans(r.Context(), id, r.PathValue("world"))
		if e != nil {
			sendError(w, e)
			return
		}
		jsonOut(w, 200, v)
	})
	handle("GET /api/v1/worlds/{world}/invites", func(w http.ResponseWriter, r *http.Request, id string) {
		v, e := service.ListInvites(r.Context(), id, r.PathValue("world"))
		if e != nil {
			sendError(w, e)
			return
		}
		jsonOut(w, 200, v)
	})
	handle("POST /api/v1/worlds/{world}/invites", func(w http.ResponseWriter, r *http.Request, id string) {
		var q struct {
			Username string `json:"username"`
		}
		if !decode(w, r, &q) {
			return
		}
		var v worlds.InviteLink
		var e error
		if q.Username == "" {
			v, e = service.IssueInvite(r.Context(), id, r.PathValue("world"), "")
		} else {
			v, e = service.IssueInviteForUsername(r.Context(), id, r.PathValue("world"), q.Username)
		}
		if e != nil {
			sendError(w, e)
			return
		}
		jsonOut(w, 201, v)
	})
	handle("POST /api/v1/invites/redeem", func(w http.ResponseWriter, r *http.Request, id string) {
		var q struct {
			Token string `json:"token"`
		}
		if !decode(w, r, &q) {
			return
		}
		v, e := service.RedeemInvite(r.Context(), id, q.Token)
		if e != nil {
			sendError(w, e)
			return
		}
		jsonOut(w, 200, v)
	})
	handle("POST /api/v1/worlds/{world}/invites/{invite}/revoke", func(w http.ResponseWriter, r *http.Request, id string) {
		var q struct{}
		if !decode(w, r, &q) {
			return
		}
		e := service.RevokeInvite(r.Context(), id, r.PathValue("world"), r.PathValue("invite"))
		if e != nil {
			sendError(w, e)
			return
		}
		w.WriteHeader(204)
	})
	handle("POST /api/v1/worlds/{world}/members/{member}/role", func(w http.ResponseWriter, r *http.Request, id string) {
		var q struct {
			Role string `json:"role"`
		}
		if !decode(w, r, &q) {
			return
		}
		e := service.ChangeRole(r.Context(), id, r.PathValue("world"), r.PathValue("member"), q.Role)
		if e != nil {
			sendError(w, e)
			return
		}
		w.WriteHeader(204)
	})
	handle("POST /api/v1/worlds/{world}/members/{member}/kick", func(w http.ResponseWriter, r *http.Request, id string) {
		var q struct{}
		if !decode(w, r, &q) {
			return
		}
		if e := service.KickMember(r.Context(), id, r.PathValue("world"), r.PathValue("member")); e != nil {
			sendError(w, e)
			return
		}
		w.WriteHeader(204)
	})
	handle("POST /api/v1/worlds/{world}/members/{member}/ban", func(w http.ResponseWriter, r *http.Request, id string) {
		var q struct{}
		if !decode(w, r, &q) {
			return
		}
		if e := service.BanMember(r.Context(), id, r.PathValue("world"), r.PathValue("member")); e != nil {
			sendError(w, e)
			return
		}
		w.WriteHeader(204)
	})
	handle("POST /api/v1/worlds/{world}/members/{member}/unban", func(w http.ResponseWriter, r *http.Request, id string) {
		var q struct{}
		if !decode(w, r, &q) {
			return
		}
		if e := service.UnbanMember(r.Context(), id, r.PathValue("world"), r.PathValue("member")); e != nil {
			sendError(w, e)
			return
		}
		w.WriteHeader(204)
	})
	handle("POST /api/v1/worlds/{world}/transfer", func(w http.ResponseWriter, r *http.Request, id string) {
		var q subject
		if !decode(w, r, &q) {
			return
		}
		if e := service.TransferWorld(r.Context(), id, r.PathValue("world"), q.UserID); e != nil {
			sendError(w, e)
			return
		}
		w.WriteHeader(204)
	})
	handle("POST /api/v1/worlds/{world}/archive", func(w http.ResponseWriter, r *http.Request, id string) {
		var q struct{}
		if !decode(w, r, &q) {
			return
		}
		if e := service.ArchiveWorld(r.Context(), id, r.PathValue("world")); e != nil {
			sendError(w, e)
			return
		}
		w.WriteHeader(204)
	})
}
