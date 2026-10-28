package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealth(t *testing.T) {
	for _, tc := range []struct {
		method, path string
		status       int
		body         string
	}{
		{"GET", "/health/live", 200, "{\"status\":\"ok\"}\n"},
		{"POST", "/health/live", 405, ""},
		{"GET", "/missing", 404, ""},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			res := httptest.NewRecorder()
			handler().ServeHTTP(res, httptest.NewRequest(tc.method, tc.path, nil))
			if res.Code != tc.status {
				t.Fatalf("status = %d, want %d", res.Code, tc.status)
			}
			if tc.body != "" && (res.Body.String() != tc.body || res.Header().Get("Content-Type") != "application/json") {
				t.Fatalf("invalid health response: %v %q", res.Header(), res.Body.String())
			}
		})
	}
}

func TestServeUntilSignalWaitsForHTTPDrain(t *testing.T) {
	for _, timeoutCase := range []bool{false, true} {
		t.Run(fmt.Sprint(timeoutCase), func(t *testing.T) {
			listener, e := net.Listen("tcp", "127.0.0.1:0")
			if e != nil {
				t.Fatal(e)
			}
			entered, release, draining := make(chan struct{}), make(chan struct{}), make(chan struct{})
			defer close(release)
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				close(entered)
				select {
				case <-release:
					w.Write([]byte("done"))
				case <-r.Context().Done():
				}
			})}
			server.RegisterOnShutdown(func() { close(draining) })
			ctx, signal := context.WithCancel(context.Background())
			defer signal()
			result := make(chan error, 1)
			timeout := time.Second
			if timeoutCase {
				timeout = 25 * time.Millisecond
			}
			go func() { result <- serveUntilSignal(ctx, server, listener, func() {}, timeout) }()
			requestDone := make(chan struct{})
			go func() {
				defer close(requestDone)
				response, e := http.Get("http://" + listener.Addr().String())
				if e == nil {
					io.Copy(io.Discard, response.Body)
					response.Body.Close()
				}
			}()
			<-entered
			signal()
			<-draining
			if timeoutCase {
				if e := <-result; !errors.Is(e, context.DeadlineExceeded) {
					t.Fatalf("drain timeout: %v", e)
				}
			} else {
				select {
				case e := <-result:
					t.Fatalf("returned before handler drained: %v", e)
				case <-time.After(50 * time.Millisecond):
				}
				// Release through a separate channel send, preserving the deferred close.
				release <- struct{}{}
				if e := <-result; e != nil {
					t.Fatal(e)
				}
			}
			<-requestDone
		})
	}
}
