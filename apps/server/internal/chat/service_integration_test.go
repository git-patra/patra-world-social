//go:build integration

package chat_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"patra-world-social/server/internal/application"
	"patra-world-social/server/internal/chat"
	httpapi "patra-world-social/server/internal/http"
	"patra-world-social/server/internal/moderation"
	"patra-world-social/server/internal/realtime"
	"patra-world-social/server/internal/testutil"
	"testing"
	"time"
)

func TestChatPersistenceModerationIsolationAndResync(t *testing.T) {
	p := testutil.Database(t)
	u, tokens := testutil.Users(t, p, 4)
	ctx := context.Background()
	app, e := application.New(p, httpapi.Options{Origin: "http://127.0.0.1:4176", Addr: "127.0.0.1:0", Development: true})
	if e != nil {
		t.Fatal(e)
	}
	defer app.Close()
	w, e := app.Worlds.CreateWorld(ctx, u[0], "Autumn")
	if e != nil {
		t.Fatal(e)
	}
	for _, id := range u[1:3] {
		inv, e := app.Worlds.IssueInvite(ctx, u[0], w.ID, id)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = app.Worlds.AcceptTargetedInvite(ctx, id, inv.ID); e != nil {
			t.Fatal(e)
		}
	}
	other, _ := app.Worlds.CreateWorld(ctx, u[3], "Other")
	join := func(i int, world string) realtime.JoinContext {
		j := realtime.JoinContext{Token: tokens[i], WorldID: world}
		s, e := app.Hub.Join(ctx, j)
		if e != nil {
			t.Fatal(e)
		}
		j.SessionID = s.SessionID
		j.Epoch = s.Epoch
		return j
	}
	a, b, c := join(0, w.ID), join(1, w.ID), join(3, other.ID)
	if _, e = app.Chat.History(ctx, u[3], w.ID, ""); e == nil {
		t.Fatal("outsider history")
	}
	if _, e = app.Chat.SendMessage(ctx, realtime.JoinContext{Token: tokens[2], WorldID: w.ID}, "world", "unjoined", "x"); e == nil {
		t.Fatal("unjoined sent")
	}
	m, e := app.Chat.SendMessage(ctx, b, "world", "retry", "<script>alert('x')</script>")
	if e != nil {
		t.Fatal(e)
	}
	same, e := app.Chat.SendMessage(ctx, b, "world", "retry", "different retry body")
	if e != nil || same.ID != m.ID {
		t.Fatal("duplicate persisted", e)
	}
	select {
	case <-app.Hub.Events(c):
		t.Fatal("cross-world event")
	default:
	}
	ev := <-app.Hub.Events(a)
	raw, _ := json.Marshal(ev.Payload)
	if ev.Type != "chat.message" || !json.Valid(raw) {
		t.Fatal("live message missing")
	}
	for i := 0; i < 4; i++ {
		if _, e = app.Chat.SendMessage(ctx, b, "world", fmt.Sprint(i), "x"); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = app.Chat.SendMessage(ctx, b, "world", "rate", "x"); !errors.Is(e, chat.ErrRate) {
		t.Fatal("rate exceeded accepted", e)
	}
	if e = app.Moderation.Moderate(ctx, u[2], w.ID, moderation.Action{Kind: "mute", UserID: u[1], Seconds: 900}); e == nil {
		t.Fatal("member moderated")
	}
	if e = app.Moderation.Moderate(ctx, u[0], w.ID, moderation.Action{Kind: "mute", UserID: u[1], Seconds: 59}); e == nil {
		t.Fatal("short mute")
	}
	if e = app.Moderation.Moderate(ctx, u[0], w.ID, moderation.Action{Kind: "mute", UserID: u[1], Seconds: 900}); e != nil {
		t.Fatal(e)
	}
	if _, e = app.Chat.SendMessage(ctx, b, "world", "muted", "x"); !errors.Is(e, chat.ErrMuted) {
		t.Fatal("mute ignored", e)
	}
	if e = app.Moderation.Moderate(ctx, u[0], w.ID, moderation.Action{Kind: "delete", MessageID: m.ID}); e != nil {
		t.Fatal(e)
	}
	history, e := app.Chat.History(ctx, u[1], w.ID, "")
	if e != nil || len(history) != 5 {
		t.Fatal("history", len(history), e)
	}
	for _, v := range history {
		if v.ID == m.ID && (!v.Deleted || v.Text != "") {
			t.Fatal("deleted body leaked")
		}
	}
	// Seed history to exercise actual stable keyset pagination and retention.
	for i := 0; i < 55; i++ {
		_, e = p.Exec(ctx, `INSERT INTO chat_messages(world_id,user_id,client_message_id,text) VALUES($1,$2,$3,'history')`, w.ID, u[0], fmt.Sprint("history-", i))
		if e != nil {
			t.Fatal(e)
		}
	}
	page, e := app.Chat.History(ctx, u[0], w.ID, "")
	if e != nil || len(page) != 50 {
		t.Fatal("page limit", len(page), e)
	}
	older, e := app.Chat.History(ctx, u[0], w.ID, page[49].ID)
	if e != nil || len(older) != 10 || older[0].ID == page[49].ID {
		t.Fatal("pagination", len(older), e)
	}
	_, e = p.Exec(ctx, `UPDATE chat_messages SET created_at=now()-interval '31 days' WHERE id=$1`, m.ID)
	if e != nil {
		t.Fatal(e)
	}
	if e = app.Chat.Cleanup(ctx); e != nil {
		t.Fatal(e)
	}
	var count int
	p.QueryRow(ctx, `SELECT count(*) FROM chat_messages WHERE id=$1`, m.ID).Scan(&count)
	if count != 0 {
		t.Fatal("retention not cleaned")
	}
	// Bounded overflow forces reconnect; persisted history is the resync source.
	for i := 0; i < 70; i++ {
		app.Hub.Publish(w.ID, "chat.deleted", map[string]string{"worldId": w.ID, "id": m.ID})
	}
	select {
	case <-app.Hub.Done(a):
	case <-time.After(time.Second):
		t.Fatal("overflow not fenced")
	}
	resumed, e := app.Hub.Resume(ctx, a, 0)
	if e != nil {
		t.Fatal(e)
	}
	a.SessionID = resumed.SessionID
	a.Epoch = resumed.Epoch
	if _, e = app.Chat.SendMessage(ctx, a, "world", "after-resync", "back"); e != nil {
		t.Fatal(e)
	}
	if e = app.Worlds.KickMember(ctx, u[0], w.ID, u[1]); e != nil {
		t.Fatal(e)
	}
	if _, e = app.Chat.SendMessage(ctx, b, "world", "kicked", "x"); e == nil {
		t.Fatal("kicked sent")
	}
	if _, e = app.Chat.History(ctx, u[1], w.ID, ""); e == nil {
		t.Fatal("kicked history")
	}
	// A deferred failure happens at COMMIT, after a message ID has been allocated.
	// It must still produce neither a stored row nor a live success event.
	if _, e = p.Exec(ctx, `CREATE FUNCTION reject_chat_commit() RETURNS trigger LANGUAGE plpgsql AS 'BEGIN RAISE EXCEPTION ''synthetic commit failure''; END'; CREATE CONSTRAINT TRIGGER reject_chat_commit AFTER INSERT ON chat_messages DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_chat_commit()`); e != nil {
		t.Fatal(e)
	}
	queued := len(app.Hub.Events(a))
	if _, e = app.Chat.SendMessage(ctx, a, "world", "failed-commit", "must not succeed"); e == nil {
		t.Fatal("commit failure reported as success")
	}
	if len(app.Hub.Events(a)) != queued {
		t.Fatal("failed commit broadcast a message")
	}
	if e = p.QueryRow(ctx, `SELECT count(*) FROM chat_messages WHERE client_message_id='failed-commit'`).Scan(&count); e != nil || count != 0 {
		t.Fatal("failed commit persisted", count, e)
	}
	if _, e = p.Exec(ctx, `DROP TRIGGER reject_chat_commit ON chat_messages`); e != nil {
		t.Fatal(e)
	}
	if _, e = app.Chat.SendMessage(ctx, a, "world", "failed-commit", "retry succeeds"); e != nil {
		t.Fatal("retry after commit failure", e)
	}

}
