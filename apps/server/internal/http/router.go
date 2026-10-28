package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"patra-world-social/server/internal/auth"
	"patra-world-social/server/internal/chat"
	"patra-world-social/server/internal/moderation"
	"patra-world-social/server/internal/social"
	"patra-world-social/server/internal/worlds"
)

type Auth interface {
	Register(context.Context, string, string) (auth.User, auth.RecoveryCode, error)
	Authenticate(context.Context, string, string) (auth.Session, error)
	RequireSession(context.Context, string) (auth.Session, error)
	Recover(context.Context, string, string, string) (auth.RecoveryCode, error)
	Logout(context.Context, string) error
}
type Options struct {
	Origin, Addr, TrustedProxyCIDRs string
	Development                     bool
	Worlds                          *worlds.Service
	Social                          *social.Service
	Chat                            *chat.Service
	Moderation                      *moderation.Service
}
type contextKey struct{}

func SessionFromContext(ctx context.Context) (auth.Session, bool) {
	s, ok := ctx.Value(contextKey{}).(auth.Session)
	return s, ok
}

// SessionMiddleware authenticates a route and adds the session for downstream world handlers.
func SessionMiddleware(service Auth, development bool, next http.Handler) http.Handler {
	name := "__Host-patra_session"
	if development {
		name = "patra_session"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(name)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		session, err := service.RequireSession(r.Context(), cookie.Value)
		if err != nil {
			if errors.Is(err, auth.ErrInvalid) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
			} else {
				http.Error(w, "server error", http.StatusInternalServerError)
			}
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey{}, session)))
	})
}

type limiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

const limiterWindow = time.Minute
const limiterMaxKeys = 10000

func recentAttempts(hits []time.Time, now time.Time) []time.Time {
	recent := hits[:0]
	for _, at := range hits {
		if now.Sub(at) < limiterWindow {
			recent = append(recent, at)
		}
	}
	return recent
}

func (l *limiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if l.hits == nil {
		l.hits = make(map[string][]time.Time)
	}
	v := recentAttempts(l.hits[key], now)
	if len(v) > 0 {
		if len(v) >= 10 {
			l.hits[key] = v
			return false
		}
		l.hits[key] = append(v, now)
		return true
	}
	delete(l.hits, key)
	if len(l.hits) >= limiterMaxKeys {
		for account, hits := range l.hits {
			recent := recentAttempts(hits, now)
			if len(recent) == 0 {
				delete(l.hits, account)
			} else {
				l.hits[account] = recent
			}
		}
	}
	// At capacity, reject only a new key. Existing active accounts remain
	// independently rate-limited; never grow the map past this bound.
	if len(l.hits) >= limiterMaxKeys {
		return false
	}
	l.hits[key] = []time.Time{now}
	return true
}

func parseTrustedProxies(raw string) ([]*net.IPNet, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var trusted []*net.IPNet
	for _, part := range strings.Split(raw, ",") {
		_, network, err := net.ParseCIDR(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		trusted = append(trusted, network)
	}
	return trusted, nil
}
func clientIP(remoteAddr, forwarded string, trusted []*net.IPNet) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	peer := net.ParseIP(host)
	if peer == nil {
		return host
	}
	isTrusted := func(ip net.IP) bool {
		for _, network := range trusted {
			if network.Contains(ip) {
				return true
			}
		}
		return false
	}
	if !isTrusted(peer) || forwarded == "" {
		return peer.String()
	}
	parts := strings.Split(forwarded, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		ip := net.ParseIP(strings.TrimSpace(parts[i]))
		if ip == nil {
			return peer.String()
		}
		if !isTrusted(ip) {
			return ip.String()
		}
	}
	return peer.String()
}

func NewRouter(service Auth, opt Options) (http.Handler, error) {
	origin, err := url.Parse(opt.Origin)
	if err != nil || origin.Host == "" || origin.User != nil || origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" {
		return nil, errors.New("invalid origin")
	}
	if opt.Development {
		host, _, err := net.SplitHostPort(opt.Addr)
		if err != nil {
			return nil, err
		}
		if origin.Scheme != "http" || !loopback(host) || !loopback(origin.Hostname()) {
			return nil, errors.New("development requires loopback HTTP")
		}
	} else if origin.Scheme != "https" {
		return nil, errors.New("production requires HTTPS origin")
	}
	trusted, err := parseTrustedProxies(opt.TrustedProxyCIDRs)
	if err != nil {
		return nil, err
	}
	secret := make([]byte, 32)
	if _, err = rand.Read(secret); err != nil {
		return nil, err
	}
	secure := !opt.Development
	cookieName := "__Host-patra_session"
	csrfName := "__Host-patra_csrf"
	if opt.Development {
		cookieName = "patra_session"
		csrfName = "patra_csrf"
	}
	mutLimiter := &limiter{}
	mux := http.NewServeMux()
	cookie := func(name, value string, maxAge int) *http.Cookie {
		return &http.Cookie{Name: name, Value: value, Path: "/", Secure: secure, HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: maxAge}
	}
	jsonOut := func(w http.ResponseWriter, status int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(v)
	}
	fail := func(w http.ResponseWriter, status int, msg string) {
		jsonOut(w, status, map[string]string{"error": msg})
	}
	csrfValid := func(token string) bool {
		parts := strings.Split(token, ".")
		if len(parts) != 2 || len(parts[0]) != 43 || len(parts[1]) != 43 {
			return false
		}
		mac := hmac.New(sha256.New, secret)
		mac.Write([]byte(parts[0]))
		sig, err := base64.RawURLEncoding.DecodeString(parts[1])
		return err == nil && hmac.Equal(sig, mac.Sum(nil))
	}
	protected := func(w http.ResponseWriter, r *http.Request) bool {
		if r.Header.Get("Origin") != opt.Origin {
			fail(w, 403, "Asal permintaan tidak diizinkan.")
			return false
		}
		c, err := r.Cookie(csrfName)
		if err != nil || !csrfValid(c.Value) || r.Header.Get("X-CSRF-Token") != c.Value {
			fail(w, 403, "Token keamanan tidak cocok.")
			return false
		}
		return true
	}
	decode := func(w http.ResponseWriter, r *http.Request, v any) bool {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			fail(w, 415, "JSON diperlukan.")
			return false
		}
		d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048))
		d.DisallowUnknownFields()
		if err := d.Decode(v); err != nil {
			fail(w, 400, "Permintaan tidak valid.")
			return false
		}
		if err := d.Decode(new(any)); err != io.EOF {
			fail(w, 400, "Permintaan tidak valid.")
			return false
		}
		return true
	}
	limit := func(w http.ResponseWriter, r *http.Request, username string) bool {
		ip := clientIP(r.RemoteAddr, r.Header.Get("X-Forwarded-For"), trusted)
		user := strings.ToLower(username)
		if !mutLimiter.allow("ip:"+ip) || !mutLimiter.allow("account:"+user) {
			fail(w, 429, "Terlalu banyak percobaan. Coba lagi nanti.")
			return false
		}
		return true
	}
	mux.HandleFunc("GET /api/v1/auth/csrf", func(w http.ResponseWriter, r *http.Request) {
		random := make([]byte, 32)
		if _, err := rand.Read(random); err != nil {
			fail(w, 500, "Kesalahan server.")
			return
		}
		nonce := base64.RawURLEncoding.EncodeToString(random)
		mac := hmac.New(sha256.New, secret)
		mac.Write([]byte(nonce))
		token := nonce + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
		http.SetCookie(w, cookie(csrfName, token, 3600))
		jsonOut(w, 200, map[string]string{"token": token})
	})
	mux.HandleFunc("POST /api/v1/auth/register", func(w http.ResponseWriter, r *http.Request) {
		if !protected(w, r) {
			return
		}
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if !decode(w, r, &req) || !limit(w, r, req.Username) {
			return
		}
		user, code, err := service.Register(r.Context(), req.Username, req.Password)
		if err != nil {
			switch {
			case errors.Is(err, auth.ErrDuplicate):
				fail(w, 409, "Nama pengguna sudah dipakai.")
			case errors.Is(err, auth.ErrInput):
				fail(w, 400, "Nama pengguna atau kata sandi tidak valid.")
			default:
				fail(w, 500, "Kesalahan server.")
			}
			return
		}
		jsonOut(w, 201, map[string]any{"user": user, "recoveryCode": code})
	})
	mux.HandleFunc("POST /api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if !protected(w, r) {
			return
		}
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if !decode(w, r, &req) || !limit(w, r, req.Username) {
			return
		}
		session, err := service.Authenticate(r.Context(), req.Username, req.Password)
		if err != nil {
			if errors.Is(err, auth.ErrInvalid) {
				fail(w, 401, "Nama pengguna atau kata sandi salah.")
			} else {
				fail(w, 500, "Kesalahan server.")
			}
			return
		}
		http.SetCookie(w, cookie(cookieName, session.Token, 7*24*3600))
		jsonOut(w, 200, session)
	})
	mux.HandleFunc("POST /api/v1/auth/recover", func(w http.ResponseWriter, r *http.Request) {
		if !protected(w, r) {
			return
		}
		var req struct {
			Username    string `json:"username"`
			Code        string `json:"code"`
			NewPassword string `json:"newPassword"`
		}
		if !decode(w, r, &req) || !limit(w, r, req.Username) {
			return
		}
		code, err := service.Recover(r.Context(), req.Username, req.Code, req.NewPassword)
		if err != nil {
			if errors.Is(err, auth.ErrInput) {
				fail(w, 400, "Kata sandi baru tidak valid.")
			} else if errors.Is(err, auth.ErrInvalid) {
				fail(w, 401, "Kode pemulihan tidak valid.")
			} else {
				fail(w, 500, "Kesalahan server.")
			}
			return
		}
		// Revoke server-side only; a delayed response must not expire a newer login cookie.
		jsonOut(w, 200, map[string]any{"recoveryCode": code})
	})
	mux.HandleFunc("POST /api/v1/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		if !protected(w, r) {
			return
		}
		if c, err := r.Cookie(cookieName); err == nil {
			if err = service.Logout(r.Context(), c.Value); err != nil {
				fail(w, 500, "Kesalahan server.")
				return
			}
		}
		// Revoke server-side only; a delayed response must not expire a newer login cookie.
		w.WriteHeader(204)
	})
	mux.HandleFunc("GET /api/v1/me", func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err != nil {
			fail(w, 401, "Silakan masuk.")
			return
		}
		session, err := service.RequireSession(r.Context(), c.Value)
		if err != nil {
			if errors.Is(err, auth.ErrInvalid) {
				fail(w, 401, "Silakan masuk.")
			} else {
				fail(w, 500, "Kesalahan server.")
			}
			return
		}
		jsonOut(w, 200, session.User)
	})
	registerAccountRoutes(mux, service, opt.Development, protected, decode, jsonOut, fail, limit)
	if opt.Worlds != nil {
		registerWorldRoutes(mux, service, opt.Worlds, opt.Development, protected, decode, jsonOut, fail)
	}
	if opt.Social != nil {
		registerSocialRoutes(mux, service, opt, protected, decode, jsonOut, fail)
	}
	return mux, nil
}
func loopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
