package main

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"net"
	"net/http"
	"os/signal"
	"patra-world-social/server/internal/application"
	"patra-world-social/server/internal/config"
	"patra-world-social/server/internal/db"
	httpapi "patra-world-social/server/internal/http"
	"syscall"
	"time"
)

func handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{\"status\":\"ok\"}\n"))
	})
	return mux
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err = pool.Ping(context.Background()); err != nil {
		log.Fatal(err)
	}
	if err = db.ApplyMigrations(context.Background(), pool); err != nil {
		log.Fatal(err)
	}
	app, err := application.New(pool, httpapi.Options{Origin: cfg.AuthOrigin, Addr: cfg.Addr, TrustedProxyCIDRs: cfg.AuthTrustedProxyCIDRs, Development: cfg.AuthDevLoopback})
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/api/", app.Handler)
	mux.Handle("/health/", handler())
	server := &http.Server{Addr: cfg.Addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("Patra World Social listening on %s", cfg.Addr)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		log.Print(err)
		app.Close()
		return
	}
	if err = serveUntilSignal(ctx, server, listener, app.Close, 5*time.Second); err != nil {
		log.Print(err)
	}
}

// Wait for both the listener and the bounded HTTP drain before main closes its pool.
func serveUntilSignal(ctx context.Context, server *http.Server, listener net.Listener, closeRealtime func(), timeout time.Duration) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	drained := make(chan error, 1)
	go func() {
		<-ctx.Done()
		closeRealtime()
		shutdown, done := context.WithTimeout(context.Background(), timeout)
		defer done()
		err := server.Shutdown(shutdown)
		if err != nil {
			server.Close()
		} // Cancel remaining handlers after the drain deadline.
		drained <- err
	}()
	serveErr := server.Serve(listener)
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}
	cancel() // Also release the shutdown worker if Serve fails before a signal.
	return errors.Join(serveErr, <-drained)
}
