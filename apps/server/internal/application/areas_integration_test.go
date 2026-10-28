//go:build integration

package application

import (
	"context"
	"patra-world-social/server/internal/games/model"
	"patra-world-social/server/internal/lobby"
	"patra-world-social/server/internal/realtime"
	"testing"
	"time"
)

// Advance only after observing the owner's notice, preserving the real three-second gate.
func startClearedMatch(t *testing.T, ctx context.Context, g *lobby.Service, h *realtime.Hub, clock *penaltyClock, host, lobbyID, command string) (model.MatchStart, error) {
	t.Helper()
	var worldID string // derive world from caller's accepted start using the returned preparation
	type result struct {
		m model.MatchStart
		e error
	}
	done := make(chan result, 1)
	// RequestStart establishes the accepted attempt without holding admission across clear.
	m, e := g.RequestStart(ctx, host, lobbyID, command)
	if e != nil {
		return m, e
	}
	worldID = m.WorldID
	go func() { m, e := g.StartMatch(ctx, host, lobbyID, command); done <- result{m, e} }()
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case r := <-done:
			if r.e == lobby.ErrUnconfirmed {
				go func() { m, e := g.StartMatch(ctx, host, lobbyID, command); done <- result{m, e} }()
				continue
			}
			return r.m, r.e
		case <-ctx.Done():
			return model.MatchStart{}, ctx.Err()
		case <-timer.C:
			t.Fatal("area handshake did not complete")
		case <-tick.C:
			states, _ := h.WorldActivities(ctx, worldID)
			for _, a := range states {
				if a.Payload.Phase == "CLEARING" && a.Payload.ClearAt > clock.Now().UnixMilli() {
					clock.set(a.Payload.ClearAt)
				}
				if a.Payload.Phase == "RESETTING" && a.Payload.ResetAt > clock.Now().UnixMilli() {
					clock.set(a.Payload.ResetAt)
				}
			}
		}
	}
}
func awaitClearedStart(t *testing.T, ctx context.Context, g *lobby.Service, h *realtime.Hub, clock *penaltyClock, worldID, lobbyID string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		states, _ := h.WorldActivities(ctx, worldID)
		for _, a := range states {
			if a.Payload.Phase == "CLEARING" && a.Payload.ClearAt > clock.Now().UnixMilli() {
				clock.set(a.Payload.ClearAt)
			}
		}
		all, _ := g.List(ctx, worldID)
		for _, l := range all {
			if l.ID == lobbyID && l.MatchID != "" && l.StartsAt > 0 {
				return
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("async start did not reach clear/durable activation")
}
