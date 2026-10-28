//go:build integration

package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/coder/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"net/http/httptest"
	"os"
	"patra-world-social/server/internal/db"
	httpapi "patra-world-social/server/internal/http"
	"patra-world-social/server/internal/realtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func isolatedDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
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
	schema := "worlds_test_" + hex.EncodeToString(b[:])
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); admin.Close() })
	cfg, err := pgxpool.ParseConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	cfg.ConnConfig.RuntimeParams["application_name"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err = db.ApplyMigrations(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return pool
}

func TestLoadSmoke(t *testing.T) {
	p := isolatedDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	app, e := New(p, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true})
	if e != nil {
		t.Fatal(e)
	}
	defer app.Close()
	srv := httptest.NewServer(app.Handler)
	defer srv.Close()
	var users, tokens []string
	for i := 0; i < 22; i++ {
		var id string
		b := make([]byte, 32)
		rand.Read(b)
		token := base64.RawURLEncoding.EncodeToString(b)
		hash := sha256.Sum256([]byte(token))
		e = p.QueryRow(ctx, `INSERT INTO users(username,display_name,password_hash,recovery_hash) VALUES($1,$1,'unused','\x00') RETURNING id`, fmt.Sprintf("smoke_%d", i)).Scan(&id)
		if e != nil {
			t.Fatal(e)
		}
		_, e = p.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,expires_at,idle_expires_at) VALUES($1,$2,now()+interval '1 day',now()+interval '1 day')`, hash[:], id)
		if e != nil {
			t.Fatal(e)
		}
		users = append(users, id)
		tokens = append(tokens, token)
	}
	w, e := app.Worlds.CreateWorld(ctx, users[0], "smoke-a")
	if e != nil {
		t.Fatal(e)
	}
	for _, id := range users[1:21] {
		inv, e := app.Worlds.IssueInvite(ctx, users[0], w.ID, "")
		if e != nil {
			t.Fatal(e)
		}
		if _, e = app.Worlds.RedeemInvite(ctx, id, inv.Token); e != nil {
			t.Fatal(e)
		}
	}
	other, e := app.Worlds.CreateWorld(ctx, users[21], "smoke-b")
	if e != nil {
		t.Fatal(e)
	}
	dial := func(i int, world, extra string) (*websocket.Conn, realtime.Event) {
		t.Helper()
		c, _, e := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/api/v1/realtime?worldId="+world+extra, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"http://127.0.0.1:4176"}, "Cookie": {"patra_session=" + tokens[i]}}})
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { c.CloseNow() })
		_, raw, e := c.Read(ctx)
		if e != nil {
			t.Fatal(e)
		}
		ev, e := realtime.ValidateEvent(raw)
		if e != nil {
			t.Fatal(e)
		}
		return c, ev
	}
	var sockets []*websocket.Conn
	var first realtime.PublicSnapshot
	for i := 0; i < 20; i++ {
		c, ev := dial(i, w.ID, "")
		if ev.Type != "world.snapshot" {
			t.Fatalf("join %d %s", i, ev.Payload)
		}
		sockets = append(sockets, c)
		if i == 0 {
			json.Unmarshal(ev.Payload, &first)
		}
	}
	_, full := dial(20, w.ID, "")
	if !strings.Contains(string(full.Payload), "world_full") {
		t.Fatal("21st admitted")
	}
	otherSocket, isolated := dial(21, other.ID, "")
	var second realtime.PublicSnapshot
	json.Unmarshal(isolated.Payload, &second)
	if len(second.Avatars) != 1 || second.SelfID != users[21] {
		t.Fatal("world leak")
	}
	// Continuously read so heartbeat pongs and backpressure reflect live clients.
	closed := make([]chan error, 20)
	latest := make([]chan realtime.PublicSnapshot, 20)
	var readers sync.WaitGroup
	for i, c := range sockets {
		closed[i] = make(chan error, 1)
		latest[i] = make(chan realtime.PublicSnapshot, 1)
		readers.Add(1)
		go func(i int, c *websocket.Conn) {
			defer readers.Done()
			for {
				_, raw, err := c.Read(ctx)
				if err != nil {
					closed[i] <- err
					return
				}
				event, err := realtime.ValidateEvent(raw)
				if err != nil {
					t.Error(err)
					return
				}
				if event.Type == "world.delta" {
					snapshot, err := realtime.ValidateSnapshot(event.Payload, w.ID)
					if err != nil {
						t.Error(err)
						return
					}
					select {
					case <-latest[i]:
					default:
					}
					latest[i] <- snapshot
				}
			}
		}(i, c)
	}
	defer func() {
		for _, c := range sockets {
			c.CloseNow()
		}
		readers.Wait()
	}()
	stopInputs := make(chan struct{})
	inputsDone := make(chan struct{})
	go func() {
		defer close(inputsDone)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		seq := 0
		for {
			select {
			case <-stopInputs:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				seq++
				dx := 1
				if seq/20%2 == 1 {
					dx = -1
				}
				for _, c := range sockets[1:] {
					raw, _ := json.Marshal(realtime.Command{V: 1, CommandID: fmt.Sprint(seq), Seq: uint64(seq), WorldID: w.ID, Type: "player.input", Payload: json.RawMessage(fmt.Sprintf(`{"dx":%d,"dy":0}`, dx))})
					if err := c.Write(ctx, websocket.MessageText, raw); err != nil {
						return
					}
				}
			}
		}
	}()
	defer func() {
		select {
		case <-stopInputs:
		default:
			close(stopInputs)
		}
		<-inputsDone
	}()
	otherSocket.CloseNow()
	sockets[0].CloseNow()
	// Exercise the actual documented fifteen-second network interruption.
	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	close(stopInputs)
	<-inputsDone
	for i := 1; i < 20; i++ {
		select {
		case err := <-closed[i]:
			t.Fatalf("load client %d disconnected: %v", i, err)
		default:
		}
		select {
		case snapshot := <-latest[i]:
			if len(snapshot.Avatars) != 20 {
				t.Fatalf("client %d sees %d avatars", i, len(snapshot.Avatars))
			}
			found := false
			for _, a := range snapshot.Avatars {
				if a.ID == users[i] && a.Seq >= 50 {
					found = true
				}
			}
			if !found {
				t.Fatalf("client %d inputs not simulated", i)
			}
		default:
			t.Fatalf("client %d missing snapshots", i)
		}
	}
	resumed, ev := dial(0, w.ID, fmt.Sprintf("&sessionId=%s&epoch=%d", first.SessionID, first.Epoch))
	var resumedSnapshot realtime.PublicSnapshot
	json.Unmarshal(ev.Payload, &resumedSnapshot)
	if ev.Type != "world.snapshot" || len(resumedSnapshot.Avatars) != 20 || resumedSnapshot.Epoch <= first.Epoch {
		t.Fatalf("resume %s", ev.Payload)
	}
	_, full = dial(20, w.ID, "")
	if !strings.Contains(string(full.Payload), "world_full") {
		t.Fatal("resume capacity bypass")
	}
	if e = app.Auth.Logout(ctx, tokens[0]); e != nil {
		t.Fatal(e)
	}
	for {
		_, _, e = resumed.Read(ctx)
		if e != nil {
			break
		}
	}
	if websocket.CloseStatus(e) != websocket.StatusPolicyViolation {
		t.Fatalf("logout did not close %v", e)
	}
	if e = app.Worlds.BanMember(ctx, users[0], w.ID, users[1]); e != nil {
		t.Fatal(e)
	}
	select {
	case e = <-closed[1]:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if websocket.CloseStatus(e) != websocket.StatusPolicyViolation {
		t.Fatalf("ban did not close %v", e)
	}
	denied, response, e := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/api/v1/realtime?worldId="+w.ID, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {"http://127.0.0.1:4176"}, "Cookie": {"patra_session=" + tokens[1]}}})
	if e == nil {
		denied.CloseNow()
		t.Fatal("banned user rejoined")
	}
	if response == nil || response.StatusCode != 401 {
		t.Fatal("ban admission response", e)
	}
	// Bypass callbacks to prove periodic database rereads catch actual expiry.
	hash := sha256.Sum256([]byte(tokens[2]))
	if _, e = p.Exec(ctx, `UPDATE sessions SET expires_at=now()-interval '1 second' WHERE token_hash=$1`, hash[:]); e != nil {
		t.Fatal(e)
	}
	select {
	case e = <-closed[2]:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if websocket.CloseStatus(e) != websocket.StatusPolicyViolation {
		t.Fatalf("expiry did not close %v", e)
	}
	if _, e = p.Exec(ctx, `UPDATE world_members SET active=false WHERE world_id=$1 AND user_id=$2`, w.ID, users[3]); e != nil {
		t.Fatal(e)
	}
	select {
	case e = <-closed[3]:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if websocket.CloseStatus(e) != websocket.StatusPolicyViolation {
		t.Fatalf("membership reread did not close %v", e)
	}
	if e = app.Worlds.ArchiveWorld(ctx, users[0], w.ID); e != nil {
		t.Fatal(e)
	}
	select {
	case e = <-closed[4]:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if websocket.CloseStatus(e) != websocket.StatusPolicyViolation {
		t.Fatalf("archive did not close %v", e)
	}
	t.Log("20 clients, 21st rejection, isolated world, 15s reconnect, epoch/capacity, real logout/ban/archive callbacks and periodic expiry/membership reread: PASS")
}
