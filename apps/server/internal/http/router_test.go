package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"patra-world-social/server/internal/auth"
	"patra-world-social/server/internal/worlds"
	"strings"
	"testing"
	"time"
)

type fakeAuth struct{ session auth.Session }
type errorAuth struct{ fakeAuth }

func (errorAuth) RequireSession(context.Context, string) (auth.Session, error) {
	return auth.Session{}, fmt.Errorf("database unavailable")
}

func TestSessionMiddlewarePreservesStorageFailure(t *testing.T) {
	h := SessionMiddleware(errorAuth{}, true, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("handler reached") }))
	r := httptest.NewRequest("GET", "/private", nil)
	r.AddCookie(&http.Cookie{Name: "patra_session", Value: strings.Repeat("x", 43)})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 500 {
		t.Fatalf("storage error mapped to %d", w.Code)
	}
}

func TestWorldMutationSharesOriginAndCSRF(t *testing.T) {
	service := fakeAuth{session: auth.Session{Token: strings.Repeat("x", 43), User: auth.User{ID: "00000000-0000-0000-0000-000000000001"}}}
	h, err := NewRouter(service, Options{Origin: "http://127.0.0.1:4176", Development: true, Addr: "127.0.0.1:8080", Worlds: worlds.New(nil)})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "/api/v1/worlds", strings.NewReader(`{"name":"Grove"}`))
	r.AddCookie(&http.Cookie{Name: "patra_session", Value: strings.Repeat("x", 43)})
	r.Header.Set("Origin", "https://evil.example")
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatalf("world create bypassed origin: %d", w.Code)
	}
	r = httptest.NewRequest("POST", "/api/v1/worlds", strings.NewReader(`{"name":"Grove"}`))
	r.AddCookie(&http.Cookie{Name: "patra_session", Value: strings.Repeat("x", 43)})
	r.Header.Set("Origin", "http://127.0.0.1:4176")
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatalf("world create bypassed CSRF: %d", w.Code)
	}
}

func TestGuestCannotListOrRedeemWorlds(t *testing.T){
 h,err:=NewRouter(fakeAuth{},Options{Origin:"http://127.0.0.1:4176",Development:true,Addr:"127.0.0.1:8080",Worlds:worlds.New(nil)});if err!=nil{t.Fatal(err)}
 for _,tc:=range []struct{method,path string}{{"GET","/api/v1/worlds"},{"POST","/api/v1/invites/redeem"}}{w:=httptest.NewRecorder();h.ServeHTTP(w,httptest.NewRequest(tc.method,tc.path,nil));if w.Code!=401{t.Fatalf("guest %s %s: %d",tc.method,tc.path,w.Code)}}
}

func (fakeAuth) Register(context.Context, string, string) (auth.User, auth.RecoveryCode, error) {
	return auth.User{Username: "patra"}, "recovery", nil
}
func (fakeAuth) Authenticate(context.Context, string, string) (auth.Session, error) {
	return auth.Session{Token: strings.Repeat("x", 43)}, nil
}
func (f fakeAuth) RequireSession(context.Context, string) (auth.Session, error) {
	if f.session.Token != "" {
		return f.session, nil
	}
	return auth.Session{}, auth.ErrInvalid
}
func (fakeAuth) Recover(context.Context, string, string, string) (auth.RecoveryCode, error) {
	return "next", nil
}
func (fakeAuth) Logout(context.Context, string) error { return nil }

func TestMutationRequiresExactOriginAndCSRF(t *testing.T) {
	h, err := NewRouter(fakeAuth{}, Options{Origin: "http://127.0.0.1:4176", Development: true, Addr: "127.0.0.1:8080"})
	if err != nil {
		t.Fatal(err)
	}
	body := `{"username":"Patra","password":"correct horse battery"}`
	for _, tc := range []struct {
		origin, csrf string
		status       int
	}{{"", "", 403}, {"https://evil.example", "", 403}, {"http://127.0.0.1:4176", "", 403}} {
		r := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(body))
		r.Header.Set("Origin", tc.origin)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-CSRF-Token", tc.csrf)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("origin %q: %d", tc.origin, w.Code)
		}
	}
}

func TestProductionCookieIsSecure(t *testing.T) {
	h, err := NewRouter(fakeAuth{}, Options{Origin: "https://world.example", Addr: "0.0.0.0:8080"})
	if err != nil {
		t.Fatal(err)
	}
	csrf := httptest.NewRecorder()
	h.ServeHTTP(csrf, httptest.NewRequest("GET", "/api/v1/auth/csrf", nil))
	token := strings.Trim(strings.Split(strings.Split(csrf.Body.String(), `"token":"`)[1], `"`)[0], " ")
	cookie := csrf.Result().Cookies()[0]
	r := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"username":"patra","password":"correct horse battery"}`))
	r.Header.Set("Origin", "https://world.example")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", token)
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("login %d %s", w.Code, w.Body.String())
	}
	session := w.Result().Cookies()[0]
	if !session.Secure || !session.HttpOnly || session.SameSite != http.SameSiteLaxMode {
		t.Fatalf("unsafe cookie: %#v", session)
	}
}

func TestDevelopmentRequiresLoopback(t *testing.T) {
	if _, err := NewRouter(fakeAuth{}, Options{Origin: "http://example.com", Addr: "0.0.0.0:8080", Development: true}); err == nil {
		t.Fatal("unsafe development accepted")
	}
}

func TestLimiterRejectsEleventhAttempt(t *testing.T) {
	l := &limiter{}
	for i := 0; i < 10; i++ {
		if !l.allow("ip:127.0.0.1") {
			t.Fatalf("rejected attempt %d", i+1)
		}
	}
	if l.allow("ip:127.0.0.1") {
		t.Fatal("accepted eleventh attempt")
	}
	if !l.allow("ip:127.0.0.2") {
		t.Fatal("other IP blocked")
	}
}

func TestSessionMiddlewareExposesAuthenticatedUser(t *testing.T) {
	service := fakeAuth{session: auth.Session{Token: strings.Repeat("x", 43), User: auth.User{Username: "patra"}}}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := SessionFromContext(r.Context())
		if !ok || session.User.Username != "patra" {
			t.Fatal("session missing from context")
		}
		w.WriteHeader(204)
	})
	h := SessionMiddleware(service, true, next)
	r := httptest.NewRequest("GET", "/private", nil)
	r.AddCookie(&http.Cookie{Name: "patra_session", Value: strings.Repeat("x", 43)})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 204 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestClientIPTrustsOnlyConfiguredProxyChain(t *testing.T) {
	trusted, err := parseTrustedProxies("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	if got := clientIP("203.0.113.5:5000", "1.2.3.4", trusted); got != "203.0.113.5" {
		t.Fatalf("trusted spoofed header: %q", got)
	}
	if got := clientIP("10.0.0.2:5000", "1.2.3.4, 198.51.100.8, 10.0.0.3", trusted); got != "198.51.100.8" {
		t.Fatalf("wrong proxy boundary: %q", got)
	}
	if _, err = parseTrustedProxies("anything"); err == nil {
		t.Fatal("invalid proxy CIDR accepted")
	}
}

func TestLimiterCapacityKeepsExistingAccountsAndPrunesMinuteOldKeys(t *testing.T) {
	now := time.Now()
	l := &limiter{hits: map[string][]time.Time{"account:existing": {now}}}
	for i := 0; i < 10001; i++ {
		l.hits[fmt.Sprintf("account:stale-%d", i)] = []time.Time{now.Add(-2 * time.Minute)}
	}
	if !l.allow("account:existing") {
		t.Fatal("existing account disabled by unrelated keys")
	}
	if !l.allow("account:new") {
		t.Fatal("minute-old keys prevented new account")
	}
	if len(l.hits) > 10000 {
		t.Fatalf("limiter remained over capacity: %d", len(l.hits))
	}
}

func TestLimiterRejectsNewKeysAtFullActiveCapacityButKeepsExisting(t *testing.T) {
	now := time.Now()
	l := &limiter{hits: make(map[string][]time.Time)}
	for i := 0; i < 10000; i++ {
		l.hits[fmt.Sprintf("account:active-%d", i)] = []time.Time{now}
	}
	if !l.allow("account:active-0") {
		t.Fatal("existing account rejected at capacity")
	}
	if l.allow("account:new") {
		t.Fatal("new key exceeded storage capacity")
	}
	if len(l.hits) != 10000 {
		t.Fatalf("capacity changed: %d", len(l.hits))
	}
}
