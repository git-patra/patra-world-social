//go:build integration

package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"patra-world-social/server/internal/auth"
	httpapi "patra-world-social/server/internal/http"
	"patra-world-social/server/internal/leaderboard"
	"patra-world-social/server/internal/testutil"
	"strings"
	"testing"
)

func TestAccountRoutesRequireSessionCSRFOriginAndPassword(t *testing.T) {
	p := testutil.Database(t)
	a := auth.New(p)
	a.RemoveAccount = leaderboard.RemoveAccount
	ctx := context.Background()
	u, _, e := a.Register(ctx, "synthetic_account", "long synthetic password")
	if e != nil {
		t.Fatal(e)
	}
	session, e := a.Authenticate(ctx, u.Username, "long synthetic password")
	if e != nil {
		t.Fatal(e)
	}
	router, e := httpapi.NewRouter(a, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true})
	if e != nil {
		t.Fatal(e)
	}
	csrf := httptest.NewRecorder()
	router.ServeHTTP(csrf, httptest.NewRequest("GET", "/api/v1/auth/csrf", nil))
	var token struct {
		Token string `json:"token"`
	}
	if e = json.Unmarshal(csrf.Body.Bytes(), &token); e != nil {
		t.Fatal(e)
	}
	call := func(path, body string, authenticated, protected bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if authenticated {
			req.AddCookie(&http.Cookie{Name: "patra_session", Value: session.Token})
		}
		if protected {
			req.Header.Set("Origin", "http://127.0.0.1:4176")
			req.Header.Set("X-CSRF-Token", token.Token)
			for _, c := range csrf.Result().Cookies() {
				req.AddCookie(c)
			}
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		return response
	}
	for _, action := range []string{"profile", "motion", "password", "recovery", "delete"} {
		if r := call("/api/v1/account/"+action, `{}`, false, true); r.Code != 401 {
			t.Fatal("guest", action, r.Code)
		}
		if r := call("/api/v1/account/"+action, `{}`, true, false); r.Code != 403 {
			t.Fatal("CSRF", action, r.Code)
		}
	}
	if r := call("/api/v1/account/profile", `{"displayName":"Local","avatar":"moon","settings":{"audio":false,"controls":"touch","reducedMotion":true},"score":999}`, true, true); r.Code != 400 {
		t.Fatal("score import allowed", r.Code)
	}
	if r := call("/api/v1/account/recovery", `{"password":"wrong"}`, true, true); r.Code != 401 {
		t.Fatal("rotation without password", r.Code)
	}
	if r := call("/api/v1/account/password", `{"password":"wrong","newPassword":"another long password"}`, true, true); r.Code != 401 {
		t.Fatal("password reauth", r.Code)
	}
	if r := call("/api/v1/account/profile", `{"displayName":"Local","avatar":"moon","settings":{"audio":false,"controls":"touch","reducedMotion":true}}`, true, true); r.Code != 200 {
		t.Fatal(r.Code, r.Body.String())
	}
	if r := call("/api/v1/account/motion", `{"reducedMotion":false,"displayName":"overwrite"}`, true, true); r.Code != 400 {
		t.Fatal("motion unknown field", r.Code)
	}
	if r := call("/api/v1/account/motion", `{}`, true, true); r.Code != 400 {
		t.Fatal("motion missing field", r.Code)
	}
	r := call("/api/v1/account/motion", `{"reducedMotion":false}`, true, true)
	var fresh auth.Profile
	if r.Code != 200 || json.Unmarshal(r.Body.Bytes(), &fresh) != nil || fresh.DisplayName != "Local" || fresh.Avatar != "moon" || fresh.Settings.Audio || fresh.Settings.Controls != "touch" || fresh.Settings.ReducedMotion {
		t.Fatal("motion must return fresh preserved profile", r.Code, r.Body.String())
	}
	if r := call("/api/v1/account/password", `{"password":"long synthetic password","newPassword":"new long synthetic password"}`, true, true); r.Code != 204 || len(r.Result().Cookies()) != 0 {
		t.Fatal("password response must only revoke server-side", r.Code, r.Header())
	}
	session, e = a.Authenticate(ctx, u.Username, "new long synthetic password")
	if e != nil {
		t.Fatal(e)
	}
	if r := call("/api/v1/account/delete", `{"confirmation":"synthetic_account"}`, true, true); r.Code != 204 || len(r.Result().Cookies()) != 0 {
		t.Fatal(r.Code, r.Body.String(), r.Header())
	}
	if r := call("/api/v1/account/profile", `{}`, true, true); r.Code != 401 {
		t.Fatal("deleted session accepted", r.Code)
	}
}

func TestLogoutAndRecoveryRevokeOldSessionsWithoutExpiringNewCookie(t *testing.T) {
	p := testutil.Database(t)
	a := auth.New(p)
	ctx := context.Background()
	u, code, e := a.Register(ctx, "cookie_contract", "long synthetic password")
	if e != nil {
		t.Fatal(e)
	}
	old, e := a.Authenticate(ctx, u.Username, "long synthetic password")
	if e != nil {
		t.Fatal(e)
	}
	router, e := httpapi.NewRouter(a, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true})
	if e != nil {
		t.Fatal(e)
	}
	csrf := httptest.NewRecorder()
	router.ServeHTTP(csrf, httptest.NewRequest("GET", "/api/v1/auth/csrf", nil))
	var token struct{ Token string }
	if e = json.Unmarshal(csrf.Body.Bytes(), &token); e != nil {
		t.Fatal(e)
	}
	request := func(path, body, session string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", path, strings.NewReader(body))
		r.Header.Set("Origin", "http://127.0.0.1:4176")
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-CSRF-Token", token.Token)
		for _, c := range csrf.Result().Cookies() {
			r.AddCookie(c)
		}
		r.AddCookie(&http.Cookie{Name: "patra_session", Value: session})
		out := httptest.NewRecorder()
		router.ServeHTTP(out, r)
		return out
	}
	r := request("/api/v1/auth/logout", `{}`, old.Token)
	if r.Code != 204 || len(r.Result().Cookies()) != 0 {
		t.Fatal("logout cookie race", r.Code, r.Header())
	}
	if _, e = a.RequireSession(ctx, old.Token); !errors.Is(e, auth.ErrInvalid) {
		t.Fatal("logout weakened revocation", e)
	}
	newer, e := a.Authenticate(ctx, u.Username, "long synthetic password")
	if e != nil {
		t.Fatal(e)
	}
	body, _ := json.Marshal(map[string]string{"username": u.Username, "code": string(code), "newPassword": "another long password"})
	r = request("/api/v1/auth/recover", string(body), old.Token)
	if r.Code != 200 || len(r.Result().Cookies()) != 0 {
		t.Fatal("recovery cookie race", r.Code, r.Header())
	}
	if _, e = a.RequireSession(ctx, newer.Token); !errors.Is(e, auth.ErrInvalid) {
		t.Fatal("recovery weakened session revocation", e)
	}
	latest, e := a.Authenticate(ctx, u.Username, "another long password")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = a.RequireSession(ctx, latest.Token); e != nil {
		t.Fatal("replacement login rejected", e)
	}
}
