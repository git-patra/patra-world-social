//go:build integration

package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"patra-world-social/server/internal/auth"
	"patra-world-social/server/internal/db"
	"patra-world-social/server/internal/worlds"
)

func TestWorldHTTPScopedBySession(t *testing.T) {
	raw := os.Getenv("DATABASE_URL")
	if raw == "" {
		t.Fatal("DATABASE_URL required")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	var b [8]byte
	if _, err = rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	schema := "world_http_test_" + hex.EncodeToString(b[:])
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); admin.Close() })
	cfg, err := pgxpool.ParseConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err = db.ApplyMigrations(ctx, pool); err != nil {
		t.Fatal(err)
	}
	var owner, outsider string
	for _, row := range []struct {
		name string
		id   *string
	}{{"owner", &owner}, {"outsider", &outsider}} {
		if err = pool.QueryRow(ctx, `INSERT INTO users(username,display_name,password_hash,recovery_hash) VALUES($1,$1,'unused','\x00') RETURNING id`, row.name).Scan(row.id); err != nil {
			t.Fatal(err)
		}
	}
	opt := Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:8080", Development: true, Worlds: worlds.New(pool)}
	ownerHandler, err := NewRouter(fakeAuth{session: auth.Session{User: auth.User{ID: owner}, Token: strings.Repeat("x", 43)}}, opt)
	if err != nil {
		t.Fatal(err)
	}
	outsiderHandler, err := NewRouter(fakeAuth{session: auth.Session{User: auth.User{ID: outsider}, Token: strings.Repeat("x", 43)}}, opt)
	if err != nil {
		t.Fatal(err)
	}
	csrf := httptest.NewRecorder()
	ownerHandler.ServeHTTP(csrf, httptest.NewRequest("GET", "/api/v1/auth/csrf", nil))
	var token struct {
		Token string `json:"token"`
	}
	if err = json.Unmarshal(csrf.Body.Bytes(), &token); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/api/v1/worlds", strings.NewReader(`{"name":"Secret Grove"}`))
	req.Header.Set("Origin", opt.Origin)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", token.Token)
	req.AddCookie(csrf.Result().Cookies()[0])
	req.AddCookie(&http.Cookie{Name: "patra_session", Value: strings.Repeat("x", 43)})
	made := httptest.NewRecorder()
	ownerHandler.ServeHTTP(made, req)
	if made.Code != 201 {
		t.Fatalf("create: %d %s", made.Code, made.Body.String())
	}
	var world worlds.World
	if err = json.Unmarshal(made.Body.Bytes(), &world); err != nil {
		t.Fatal(err)
	}
	read := func(h http.Handler, path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		r.AddCookie(&http.Cookie{Name: "patra_session", Value: strings.Repeat("x", 43)})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := read(ownerHandler, "/api/v1/worlds/"+world.ID); w.Code != 200 {
		t.Fatalf("owner read %d", w.Code)
	}
	if w := read(outsiderHandler, "/api/v1/worlds/"+world.ID); w.Code != 403 {
		t.Fatalf("outsider read %d", w.Code)
	}
	if w := read(outsiderHandler, "/api/v1/worlds/"+world.ID+"/members"); w.Code != 403 {
		t.Fatalf("outsider members %d", w.Code)
	}
	if w := read(outsiderHandler, "/api/v1/worlds"); w.Code != 200 || strings.Contains(w.Body.String(), world.ID) {
		t.Fatalf("outsider listing %d %s", w.Code, w.Body.String())
	}
}
