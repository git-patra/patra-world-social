package httpapi

import (
	"context"
	"errors"
	"net/http"
	"patra-world-social/server/internal/auth"
)

type accountService interface {
	Profile(context.Context, string) (auth.Profile, error)
	UpdateProfile(context.Context, string, auth.Profile) (auth.User, error)
	UpdateMotion(context.Context, string, bool) (auth.Profile, error)
	ChangePassword(context.Context, string, string, string) error
	RotateRecovery(context.Context, string, string) (auth.RecoveryCode, error)
	DeleteAccount(context.Context, string, string) error
}

func registerAccountRoutes(mux *http.ServeMux, service Auth, development bool, protected func(http.ResponseWriter, *http.Request) bool, decode func(http.ResponseWriter, *http.Request, any) bool, out func(http.ResponseWriter, int, any), fail func(http.ResponseWriter, int, string), limit func(http.ResponseWriter, *http.Request, string) bool) {
	s, ok := service.(accountService)
	if !ok {
		return
	}
	for _, action := range []string{"profile", "motion", "password", "recovery", "delete"} {
		mux.Handle("POST /api/v1/account/"+action, SessionMiddleware(service, development, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !protected(w, r) {
				return
			}
			session, _ := SessionFromContext(r.Context())
			if !limit(w, r, session.User.ID) {
				return
			}
			var value any
			var e error
			switch action {
			case "profile":
				var p auth.Profile
				if !decode(w, r, &p) {
					return
				}
				value, e = s.UpdateProfile(r.Context(), session.User.ID, p)
			case "motion":
				var p struct {
					ReducedMotion *bool `json:"reducedMotion"`
				}
				if !decode(w, r, &p) {
					return
				}
				if p.ReducedMotion == nil {
					fail(w, 400, "Pengaturan tidak valid.")
					return
				}
				value, e = s.UpdateMotion(r.Context(), session.User.ID, *p.ReducedMotion)
			case "password":
				var p struct {
					Password    string `json:"password"`
					NewPassword string `json:"newPassword"`
				}
				if !decode(w, r, &p) {
					return
				}
				e = s.ChangePassword(r.Context(), session.User.ID, p.Password, p.NewPassword)
			case "recovery":
				var p struct {
					Password string `json:"password"`
				}
				if !decode(w, r, &p) {
					return
				}
				var code auth.RecoveryCode
				code, e = s.RotateRecovery(r.Context(), session.User.ID, p.Password)
				value = map[string]any{"recoveryCode": code}
			case "delete":
				var p struct {
					Confirmation string `json:"confirmation"`
				}
				if !decode(w, r, &p) {
					return
				}
				e = s.DeleteAccount(r.Context(), session.User.ID, p.Confirmation)
			}
			if e != nil {
				switch {
				case errors.Is(e, auth.ErrOwnsWorld):
					fail(w, 409, "Transfer atau arsipkan semua dunia milikmu sebelum menghapus akun.")
				case errors.Is(e, auth.ErrInvalid):
					fail(w, 401, "Kredensial tidak valid.")
				case errors.Is(e, auth.ErrInput):
					fail(w, 400, "Pengaturan atau konfirmasi tidak valid.")
				default:
					fail(w, 500, "Perubahan akun belum dapat dikonfirmasi.")
				}
				return
			}
			// Revocation is authoritative in PostgreSQL. Do not expire a shared cookie:
			// this response may arrive after another login established a newer one.
			if value == nil {
				w.WriteHeader(204)
			} else {
				out(w, 200, value)
			}
		})))
	}
	mux.Handle("GET /api/v1/account/profile", SessionMiddleware(service, development, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := SessionFromContext(r.Context())
		p, e := s.Profile(r.Context(), session.User.ID)
		if e != nil {
			fail(w, 500, "Pengaturan belum tersedia.")
			return
		}
		out(w, 200, p)
	})))
}
