package realtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/coder/websocket"
	"net/http"
	"patra-world-social/server/internal/world"
	"strconv"
	"strings"
	"time"
)

// NewHandler uses exact Origin matching before upgrade. Authentication and membership
// checks run outside the simulation goroutine, including ongoing idle validation.
func NewHandler(h *Hub, origin string, development bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.Header.Get("Origin") != origin {
			http.Error(w, "origin denied", 403)
			return
		}
		name := "__Host-patra_session"
		if development {
			name = "patra_session"
		}
		cookie, e := r.Cookie(name)
		if e != nil {
			http.Error(w, "session required", 401)
			return
		}
		j := JoinContext{Token: cookie.Value, WorldID: r.URL.Query().Get("worldId"), SessionID: r.URL.Query().Get("sessionId"), Takeover: r.URL.Query().Get("takeover") == "true"}
		j.Epoch, _ = strconv.ParseUint(r.URL.Query().Get("epoch"), 10, 64)
		if len(j.WorldID) == 0 || len(j.WorldID) > 128 || len(j.SessionID) > 128 {
			http.Error(w, "invalid world", 400)
			return
		}
		if _, _, e = h.authorize(r.Context(), j); e != nil {
			http.Error(w, "access denied", 401)
			return
		}
		c, e := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if e != nil {
			return
		}
		defer c.CloseNow()
		if r.URL.Query().Get("protocolVersion") != "2" {
			h.writeEvent(r.Context(), c, "command.error", 0, map[string]string{"code": "incompatible", "message": "World updated. Reload to use protocol 2."})
			c.Close(websocket.StatusPolicyViolation, "incompatible: reload required")
			return
		}
		c.SetReadLimit(16 * 1024)
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()
		var snap PublicSnapshot
		if j.SessionID != "" {
			snap, e = h.Resume(ctx, j, 0)
		} else {
			snap, e = h.Join(ctx, j)
		}
		if e != nil {
			code := "access_denied"
			if errors.Is(e, ErrTakeover) {
				code = "takeover_required"
			} else if errors.Is(e, ErrFull) {
				code = "world_full"
			} else if errors.Is(e, ErrFenced) {
				code = "resume_expired"
			}
			h.writeEvent(ctx, c, "command.error", 0, map[string]string{"code": code})
			c.Close(websocket.StatusPolicyViolation, code)
			return
		}
		j.SessionID = snap.SessionID
		j.Epoch = snap.Epoch
		defer h.Disconnect(j.SessionID, "transport closed")
		if e = h.writeEvent(ctx, c, "world.snapshot", snap.Tick, snap); e != nil {
			return
		}
		if activities, err := h.ActivitySnapshots(ctx, j); err == nil {
			for _, a := range activities {
				if h.writeEvent(ctx, c, "activity.snapshot", a.Tick, a) != nil {
					return
				}
			}
		}
		// One reader and one writer. A bounded queue never grows with slow readers.
		out := make(chan Command, 32)
		receivedLeave := make(chan struct{}, 1)
		// The reader may observe a peer close before the writer drains commands.
		// Release a received, valid leave even when the last ack cannot be written.
		defer func() {
			select {
			case <-receivedLeave:
				h.Leave(j)
			default:
			}
		}()
		readDone := make(chan struct{})
		reason := ""
		go func() {
			defer close(readDone)
			window := time.Now()
			movement, control, blasterInputs := 0, 0, 0
			for {
				kind, raw, e := c.Read(ctx)
				if e != nil {
					return
				}
				if kind != websocket.MessageText {
					reason = "text commands required"
					return
				}
				cmd, e := ValidateCommand(raw)
				if e != nil {
					reason = "malformed command"
					return
				}
				if cmd.Type == "world.leave" && cmd.WorldID == j.WorldID {
					select {
					case receivedLeave <- struct{}{}:
					default:
					}
				}
				now := time.Now()
				if now.Sub(window) >= time.Second {
					window = now
					movement = 0
					blasterInputs = 0
					control = 0
				}
				if cmd.Type == "blaster.input" {
					blasterInputs++
					if blasterInputs > 20 {
						reason = "blaster input rate exceeded"
						return
					}
				}
				if cmd.Type == "player.input" || cmd.Type == "player.destination" || cmd.Type == "race.input" || cmd.Type == "blaster.input" || cmd.Type == "pingpong.input" || rideDriveCommand(cmd) || clawMoveCommand(cmd) {
					movement++
					if movement > 60 {
						reason = "movement rate exceeded"
						return
					}
				} else {
					control++
					if control > 10 {
						reason = "control rate exceeded"
						return
					}
				}
				select {
				case out <- cmd:
				default:
					reason = "command queue full"
					return
				}
			}
		}()
		defer func() { cancel(); c.CloseNow(); <-readDone }()
		activityCache := map[string][]byte{}
		activitySent := map[string]time.Time{}
		activityTick := time.NewTicker(100 * time.Millisecond)
		defer activityTick.Stop()
		snapshotTick := time.NewTicker(100 * time.Millisecond)
		defer snapshotTick.Stop()
		validationTick := time.NewTicker(time.Second)
		defer validationTick.Stop()
		pingTick := time.NewTicker(5 * time.Second)
		defer pingTick.Stop()
		deliveries := h.Events(j)
		for {
			select {
			case delivery := <-deliveries:
				if h.Validate(ctx, j) != nil {
					c.Close(websocket.StatusPolicyViolation, "access revoked")
					return
				}
				if h.writeEvent(ctx, c, delivery.Type, 0, delivery.Payload) != nil {
					return
				}
			case <-readDone:
				if reason != "" {
					c.Close(websocket.StatusPolicyViolation, reason)
				}
				return
			case <-h.Done(j):
				if h.NeedsResync(j) {
					c.Close(websocket.StatusTryAgainLater, "resync_required")
				} else {
					c.Close(websocket.StatusPolicyViolation, "access revoked or control transferred")
				}
				return
			case <-ctx.Done():
				return
			case cmd := <-out:
				result, err := h.Execute(ctx, j, cmd)
				if errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrFenced) {
					c.Close(websocket.StatusPolicyViolation, "access revoked")
					return
				}
				kind := "command.ack"
				payload := map[string]any{"commandId": cmd.CommandID, "seq": cmd.Seq}
				if result != nil && err == nil {
					payload["result"] = result
				}
				if err != nil {
					kind = "command.error"
					payload["code"] = "invalid_or_unavailable"
					if strings.HasPrefix(cmd.Type, "voice.") {
						payload["message"] = err.Error()
						var coded interface{ CommandCode() string }
						if errors.As(err, &coded) {
							payload["code"] = coded.CommandCode()
						}
					}
					if cmd.Type == "chat.send" {
						payload["message"] = "Pesan ditolak: periksa mute atau batas 5 pesan per 5 detik."
					}
				}
				if h.writeEvent(ctx, c, kind, 0, payload) != nil {
					return
				}
			case <-validationTick.C:
				if h.Validate(ctx, j) != nil {
					c.Close(websocket.StatusPolicyViolation, "session or membership expired")
					return
				}
			case <-pingTick.C:
				pctx, pcancel := context.WithTimeout(ctx, 3*time.Second)
				err := c.Ping(pctx)
				pcancel()
				if err != nil {
					return
				}
			case <-activityTick.C:
				activities, err := h.ActivitySnapshots(ctx, j)
				if err != nil {
					c.Close(websocket.StatusPolicyViolation, "access revoked")
					return
				}
				for _, a := range activities {
					key := a.AreaID + ":" + a.Kind
					if !ownsActivity(a, snap.SelfID) && time.Since(activitySent[key]) < 200*time.Millisecond {
						continue
					}
					if !changedActivity(activityCache, a) {
						continue
					}
					activitySent[key] = time.Now()
					if h.writeEvent(ctx, c, "activity.snapshot", a.Tick, a) != nil {
						return
					}
				}
			case <-snapshotTick.C:
				s, e := h.Snapshot(ctx, j)
				if e != nil {
					c.Close(websocket.StatusPolicyViolation, "access revoked")
					return
				}
				if h.writeEvent(ctx, c, "world.delta", s.Tick, s) != nil {
					return
				}
			}
		}
	})
}
func writeEvent(ctx context.Context, c *websocket.Conn, kind string, tick uint64, payload any) error {
	return (&Hub{}).writeEvent(ctx, c, kind, tick, payload)
}
func (h *Hub) writeEvent(ctx context.Context, c *websocket.Conn, kind string, tick uint64, payload any) error {
	p, e := json.Marshal(payload)
	if e != nil {
		return e
	}
	if h.RedactPayload != nil {
		p, e = h.RedactPayload(p)
		if e != nil {
			return e
		}
	}
	raw, e := json.Marshal(Event{V: 2, EventID: randomID(), ServerTick: tick, ServerTimeMs: time.Now().UnixMilli(), Type: kind, Payload: p})
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return c.Write(ctx, websocket.MessageText, raw)
}

func rideDriveCommand(cmd Command) bool {
	if cmd.Type != "activity.ride" {
		return false
	}
	var p struct {
		Action string `json:"action"`
	}
	return json.Unmarshal(cmd.Payload, &p) == nil && p.Action == "drive"
}

func clawMoveCommand(cmd Command) bool {
	if cmd.Type != "activity.claw" {
		return false
	}
	var p struct {
		Action string `json:"action"`
	}
	return json.Unmarshal(cmd.Payload, &p) == nil && p.Action == "claw.move"
}

// The cache is scoped to one connection (eleven fixed public activity slots).
// Clock/tick-only changes are replaceable; identity, generation and payload are not.
func changedActivity(cache map[string][]byte, a world.ActivitySnapshot) bool {
	key := a.AreaID + ":" + a.Kind
	copy := a
	copy.Tick = 0
	copy.ServerTimeMs = 0
	raw, err := json.Marshal(copy)
	if err != nil {
		return true
	}
	if bytes.Equal(cache[key], raw) {
		return false
	}
	cache[key] = raw
	return true
}

func ownsActivity(a world.ActivitySnapshot, user string) bool {
	if a.Payload.Claw != nil {
		return a.Payload.Claw.OperatorID == user
	}
	if a.Payload.Fishing != nil {
		return a.Payload.Fishing.OwnerID == user
	}
	if a.Payload.Vehicles != nil {
		for _, v := range *a.Payload.Vehicles {
			if v.RiderID == user {
				return true
			}
		}
	}
	return false
}
