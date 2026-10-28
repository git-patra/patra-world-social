//go:build integration

package application

import (
	"context"
	"net/http/httptest"
	httpapi "patra-world-social/server/internal/http"
	"testing"
)

func TestSocialRoutesInstalled(t *testing.T) {
	p := isolatedDB(t)
	a, e := New(p, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true})
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	for _, path := range []string{"/api/v1/friends", "/api/v1/notifications", "/api/v1/worlds/00000000-0000-0000-0000-000000000001/chat"} {
		r := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		a.Handler.ServeHTTP(w, r)
		if w.Code != 401 {
			t.Fatalf("%s: expected authenticated route, got %d", path, w.Code)
		}
	}
	_ = context.Background()
}
