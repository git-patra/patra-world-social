//go:build integration

package history

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"net/http/httptest"
	"patra-world-social/server/internal/auth"
	"patra-world-social/server/internal/games/model"
	httpapi "patra-world-social/server/internal/http"
	"patra-world-social/server/internal/lobby"
	"patra-world-social/server/internal/testutil"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestResultsMigration(t *testing.T) {
	p := testutil.Database(t)
	var exists bool
	if err := p.QueryRow(context.Background(), `SELECT to_regclass('result_jobs') IS NOT NULL`).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("durable result_jobs missing")
	}
}

func fixture(t *testing.T) (*Service, model.MatchResult, []string) {
	t.Helper()
	p := testutil.Database(t)
	users, _ := testutil.Users(t, p, 2)
	ctx := context.Background()
	var world string
	if e := p.QueryRow(ctx, `INSERT INTO worlds(name,owner_id) VALUES('synthetic',$1) RETURNING id`, users[0]).Scan(&world); e != nil {
		t.Fatal(e)
	}
	if _, e := p.Exec(ctx, `INSERT INTO world_members(world_id,user_id,role) VALUES($1,$2,'owner')`, world, users[0]); e != nil {
		t.Fatal(e)
	}
	m := model.MatchStart{MatchID: "11111111-1111-4111-8111-111111111111", WorldID: world, ArenaID: "duck-1", GameType: "duck", Version: model.Version{MapID: "fixture", MapVersion: 1, RulesID: "fixture", RulesVersion: 1, ProtocolVersion: 1}, PreparedAt: 1000, Actors: []model.Actor{{ActorID: "22222222-2222-4222-8222-222222222222", ActorType: "human", UserID: users[0], DisplayName: "synthetic"}}}
	if e := (&lobby.PGStore{Pool: p}).Persist(ctx, m, users[0], "fixture"); e != nil {
		t.Fatal(e)
	}
	r := model.MatchResult{Version: m.Version, MatchID: m.MatchID, WorldID: m.WorldID, ArenaID: m.ArenaID, GameType: m.GameType, PreparedAt: m.PreparedAt, StartsAt: 4000, FinishedAt: 64000, Participants: []model.ParticipantResult{{Actor: m.Actors[0], Outcome: "finished"}}}
	return New(p), r, users
}
func TestPostgresIdempotentResultAndPrivateHistory(t *testing.T) {
	s, r, users := fixture(t)
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if e := s.Enqueue(ctx, r); e != nil {
			t.Fatal(e)
		}
		if _, e := s.Finalize(ctx, r.MatchID); e != nil {
			t.Fatal(e)
		}
	}
	var jobs, participants int
	var starts int64
	s.Pool.QueryRow(ctx, `SELECT count(*) FROM result_jobs`).Scan(&jobs)
	s.Pool.QueryRow(ctx, `SELECT count(*) FROM match_participants`).Scan(&participants)
	s.Pool.QueryRow(ctx, `SELECT (extract(epoch FROM starts_at)*1000)::bigint FROM matches`).Scan(&starts)
	if jobs != 1 || participants != 1 || starts != r.StartsAt {
		t.Fatal(jobs, participants, starts)
	}
	r.FinishedAt++
	if e := s.Enqueue(ctx, r); !errors.Is(e, ErrConflict) {
		t.Fatal("different result accepted", e)
	}
	s.Pool.Exec(ctx, `UPDATE world_members SET active=false`)
	page, e := s.ListHistory(ctx, users[0], Filter{}, "")
	if e != nil || len(page.Items) != 1 {
		t.Fatal(page, e)
	}
	if _, e = s.Detail(ctx, users[1], r.MatchID); !errors.Is(e, ErrNotFound) {
		t.Fatal("other account history exposed", e)
	}
	detail, e := s.Detail(ctx, users[0], r.MatchID)
	if e != nil || detail.Result == nil {
		t.Fatal(detail, e)
	}
	b, _ := json.Marshal(detail)
	if bytes.Contains(b, []byte(`"seed"`)) {
		t.Fatal("seed leaked")
	}
}
func TestPostgresProjectionRollbackAndRestart(t *testing.T) {
	s, r, _ := fixture(t)
	ctx := context.Background()
	if e := s.Enqueue(ctx, r); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Pool.Exec(ctx, `CREATE TABLE synthetic_projection(match_id uuid PRIMARY KEY); CREATE FUNCTION reject_completion() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.status='completed' THEN RAISE EXCEPTION 'synthetic failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_completion BEFORE UPDATE ON matches FOR EACH ROW EXECUTE FUNCTION reject_completion()`); e != nil {
		t.Fatal(e)
	}
	s.Projectors["duck"] = func(ctx context.Context, tx pgx.Tx, r model.MatchResult) error {
		_, e := tx.Exec(ctx, `INSERT INTO synthetic_projection VALUES($1)`, r.MatchID)
		return e
	}
	if _, e := s.Finalize(ctx, r.MatchID); e == nil {
		t.Fatal("trigger did not abort")
	}
	var n int
	var status string
	s.Pool.QueryRow(ctx, `SELECT count(*) FROM synthetic_projection`).Scan(&n)
	s.Pool.QueryRow(ctx, `SELECT status FROM matches`).Scan(&status)
	if n != 0 || status != "finalizing" {
		t.Fatal(n, status)
	}
	s.Pool.Exec(ctx, `DROP TRIGGER reject_completion ON matches`)
	restarted := New(s.Pool)
	restarted.Projectors = s.Projectors
	if e := restarted.RecoverPending(ctx); e != nil {
		t.Fatal(e)
	}
	if _, e := restarted.Finalize(ctx, r.MatchID); e != nil {
		t.Fatal(e)
	}
	if _, e := restarted.Finalize(ctx, r.MatchID); e != nil {
		t.Fatal(e)
	}
	s.Pool.QueryRow(ctx, `SELECT count(*) FROM synthetic_projection`).Scan(&n)
	if n != 1 {
		t.Fatal(n)
	}
}
func TestPostgresStartupOrphansAndReconcileFence(t *testing.T) {
	s, r, users := fixture(t)
	ctx := context.Background()
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	if e = lockMatch(ctx, tx, r.MatchID); e != nil {
		t.Fatal(e)
	}
	short, cancel := context.WithTimeout(ctx, 40*time.Millisecond)
	defer cancel()
	if _, e = s.Reconcile(short, r.MatchID); e == nil {
		t.Fatal("absence observed before outstanding write settled")
	}
	tx.Rollback(ctx)
	if e = s.RecoverPending(ctx); e != nil {
		t.Fatal(e)
	}
	d, e := s.Detail(ctx, users[0], r.MatchID)
	if e != nil || d.Status != "aborted_server" || d.StartsAt != nil || d.Result != nil {
		t.Fatal(d, e)
	}
	var n int
	s.Pool.QueryRow(ctx, `SELECT count(*) FROM active_match_users`).Scan(&n)
	if n != 0 {
		t.Fatal(n)
	}
	if e = s.Enqueue(ctx, r); e == nil {
		t.Fatal("revived startup orphan")
	}
}
func TestPostgresLostAckReconcileAndAbsentStorage(t *testing.T) {
	s, r, _ := fixture(t)
	ctx := context.Background()
	// Commit succeeds but the caller discards its acknowledgement, as on socket loss.
	if e := s.Enqueue(ctx, r); e != nil {
		t.Fatal(e)
	}
	status, e := s.Reconcile(ctx, r.MatchID)
	if e != nil || status != "completed" {
		t.Fatal(status, e)
	}
	status, e = s.Reconcile(ctx, r.MatchID)
	if e != nil || status != "completed" {
		t.Fatal(status, e)
	}
	s2, r2, _ := fixture(t)
	status, e = s2.Reconcile(ctx, r2.MatchID)
	if e != nil || status != "aborted_storage" {
		t.Fatal(status, e)
	}
}

func TestPostgresResultAuthorityAndIdentityOverlay(t *testing.T) {
	s, r, _ := fixture(t)
	ctx := context.Background()
	bad := r
	bad.RulesVersion++
	if e := s.Enqueue(ctx, bad); !errors.Is(e, ErrInvalid) {
		t.Fatal("version accepted", e)
	}
	bad = r
	bad.Participants = append([]model.ParticipantResult{}, r.Participants...)
	bad.Participants[0].DisplayName = "spoof"
	if e := s.Enqueue(ctx, bad); !errors.Is(e, ErrInvalid) {
		t.Fatal("roster accepted", e)
	}
	if e := s.Enqueue(ctx, r); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Finalize(ctx, r.MatchID); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Pool.Exec(ctx, `UPDATE match_participants SET display_name='Anonim',avatar_id='pumpkin' WHERE match_id=$1`, r.MatchID); e != nil {
		t.Fatal(e)
	}
	d, e := s.Detail(ctx, r.Participants[0].UserID, r.MatchID)
	if e != nil || d.Result.Participants[0].DisplayName != "Anonim" {
		t.Fatal(d, e)
	}
	if e = s.Enqueue(ctx, r); e != nil {
		t.Fatal("public identity overlay mutated immutable result", e)
	}
}
func TestPostgresConcurrentFinalizationOnlyProjectsOnce(t *testing.T) {
	s, r, _ := fixture(t)
	ctx := context.Background()
	if _, e := s.Pool.Exec(ctx, `CREATE TABLE synthetic_projection(match_id uuid PRIMARY KEY)`); e != nil {
		t.Fatal(e)
	}
	s.Projectors["duck"] = func(ctx context.Context, tx pgx.Tx, r model.MatchResult) error {
		_, e := tx.Exec(ctx, `INSERT INTO synthetic_projection VALUES($1)`, r.MatchID)
		return e
	}
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e := s.Enqueue(ctx, r); e != nil {
				t.Error(e)
				return
			}
			if _, e := s.Finalize(ctx, r.MatchID); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	var n int
	s.Pool.QueryRow(ctx, `SELECT count(*) FROM synthetic_projection`).Scan(&n)
	if n != 1 {
		t.Fatal(n)
	}
	if _, e := s.Pool.Exec(ctx, `UPDATE result_jobs SET payload='{}'::jsonb`); e == nil {
		t.Fatal("immutable job update accepted")
	}
}
func TestPostgresRecoveryWorkerRetriesDurableProjection(t *testing.T) {
	s, r, _ := fixture(t)
	ctx := context.Background()
	if e := s.Enqueue(ctx, r); e != nil {
		t.Fatal(e)
	}
	if e := s.RecoverPending(ctx); e != nil {
		t.Fatal(e)
	}
	o := &ownerFixture{}
	w := NewWorker(s, o)
	now := time.Now()
	w.Step(ctx, now)
	if !o.blocked {
		t.Fatal("admitted with pending durable job")
	}
	w.Step(ctx, now.Add(time.Second))
	if o.blocked {
		t.Fatal("recovery did not finish")
	}
	var status string
	var n int
	s.Pool.QueryRow(ctx, `SELECT status FROM matches`).Scan(&status)
	s.Pool.QueryRow(ctx, `SELECT count(*) FROM active_match_users`).Scan(&n)
	if status != "completed" || n != 0 {
		t.Fatal(status, n)
	}
	if e := s.RetryLater(ctx, r.MatchID); e != nil {
		t.Fatal(e)
	}
}

type lostAckStore struct {
	*Service
	fail bool
}

func (s *lostAckStore) Enqueue(ctx context.Context, r model.MatchResult) error {
	e := s.Service.Enqueue(ctx, r)
	if e == nil && s.fail {
		return errors.New("synthetic lost commit acknowledgement")
	}
	return e
}
func TestPostgresWorkerUnconfirmedReconcilesLostAck(t *testing.T) {
	s, r, _ := fixture(t)
	ctx := context.Background()
	fault := &lostAckStore{Service: s, fail: true}
	o := &ownerFixture{refs: []lobby.FinalizingMatch{{MatchID: r.MatchID, WorldID: r.WorldID}}, result: r}
	w := NewWorker(fault, o)
	var statuses []string
	w.Notify = func(_ string, _ []string, e StatusEvent) { statuses = append(statuses, e.Status) }
	now := time.Now()
	w.Step(ctx, now)
	if o.saved || !o.blocked {
		t.Fatal("ack loss claimed saved", o)
	}
	w.Step(ctx, now.Add(60*time.Second))
	if !o.detached || o.blocked {
		t.Fatal(o)
	}
	if len(statuses) != 3 || statuses[0] != "finalizing" || statuses[1] != "unconfirmed" || statuses[2] != "completed" {
		t.Fatal(statuses)
	}
	var n int
	s.Pool.QueryRow(ctx, `SELECT count(*) FROM result_jobs`).Scan(&n)
	if n != 1 {
		t.Fatal(n)
	}
}
func TestPostgresOutstandingEnqueueCannotBeAbortedByReconciliation(t *testing.T) {
	s, r, _ := fixture(t)
	ctx := context.Background()
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	if e = lockMatch(ctx, tx, r.MatchID); e != nil {
		t.Fatal(e)
	}
	payload, hash, e := encodeResult(r)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(ctx, `INSERT INTO result_jobs(match_id,result_hash,payload) VALUES($1,$2,$3)`, r.MatchID, hash[:], payload); e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(ctx, `UPDATE matches SET status='finalizing' WHERE id=$1`, r.MatchID); e != nil {
		t.Fatal(e)
	}
	short, cancel := context.WithTimeout(ctx, 40*time.Millisecond)
	defer cancel()
	if _, e = s.Reconcile(short, r.MatchID); e == nil {
		t.Fatal("raced unresolved enqueue")
	}
	if e = tx.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	if status, e := s.Reconcile(ctx, r.MatchID); e != nil || status != "completed" {
		t.Fatal(status, e)
	}
}
func TestPostgresHTTPUsesSessionAndParticipation(t *testing.T) {
	s, r, users := fixture(t)
	ctx := context.Background()
	if e := s.Enqueue(ctx, r); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Finalize(ctx, r.MatchID); e != nil {
		t.Fatal(e)
	}
	token := strings.Repeat("a", 43)
	hash := sha256.Sum256([]byte(token))
	if _, e := s.Pool.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,expires_at,idle_expires_at)VALUES($1,$2,now()+interval '1 day',now()+interval '1 day')`, hash[:], users[0]); e != nil {
		t.Fatal(e)
	}
	h := httpapi.SessionMiddleware(auth.New(s.Pool), true, s.Handler())
	request := func(path, cookie string, code int) {
		t.Helper()
		req := httptest.NewRequest("GET", path, nil)
		if cookie != "" {
			req.AddCookie(&http.Cookie{Name: "patra_session", Value: cookie})
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != code {
			t.Fatal(path, w.Code, w.Body.String())
		}
	}
	request("/api/v1/history", "", 401)
	request("/api/v1/history?userId="+users[1], token, 400)
	s.Pool.Exec(ctx, `UPDATE world_members SET active=false`)
	s.Pool.Exec(ctx, `UPDATE worlds SET archived_at=now()`)
	request("/api/v1/history/"+r.MatchID, token, 200)
	request("/api/v1/history", token, 200)
	request("/api/v1/history?game=race", token, 200)
	request("/api/v1/history?cursor=bad", token, 400)
	if _, e := s.Pool.Exec(ctx, `UPDATE sessions SET user_id=$2 WHERE token_hash=$1`, hash[:], users[1]); e != nil {
		t.Fatal(e)
	}
	request("/api/v1/history/"+r.MatchID, token, 404)
}

func TestPostgresWorkerCancelsOutstandingWriteBeforeDeadlineReconcile(t *testing.T) {
	s, r, _ := fixture(t)
	ctx := context.Background()
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	if e = lockMatch(ctx, tx, r.MatchID); e != nil {
		t.Fatal(e)
	}
	o := &ownerFixture{refs: []lobby.FinalizingMatch{{MatchID: r.MatchID}}, result: r}
	w := NewWorker(s, o)
	now := time.Now()
	w.pending[r.MatchID] = &pendingResult{ref: o.refs[0], since: now, result: &r}
	w.Step(ctx, now.Add(59980*time.Millisecond))
	if o.saved || o.detached || !o.blocked {
		t.Fatal(o)
	}
	if e = tx.Rollback(ctx); e != nil {
		t.Fatal(e)
	}
	w.Step(ctx, now.Add(60*time.Second))
	if !o.detached || o.saved || o.blocked {
		t.Fatal(o)
	}
	var status string
	var jobs int
	s.Pool.QueryRow(ctx, `SELECT status FROM matches`).Scan(&status)
	s.Pool.QueryRow(ctx, `SELECT count(*) FROM result_jobs`).Scan(&jobs)
	if status != "aborted_storage" || jobs != 0 {
		t.Fatal(status, jobs)
	}
}
func TestPostgresHistoryPaginationAndFilters(t *testing.T) {
	s, r, users := fixture(t)
	ctx := context.Background()
	if e := s.RecoverPending(ctx); e != nil {
		t.Fatal(e)
	}
	var m model.MatchStart
	var b []byte
	s.Pool.QueryRow(ctx, `SELECT start_payload FROM matches`).Scan(&b)
	if e := json.Unmarshal(b, &m); e != nil {
		t.Fatal(e)
	}
	m.MatchID = "33333333-3333-4333-8333-333333333333"
	m.PreparedAt = 2000
	if e := (&lobby.PGStore{Pool: s.Pool}).Persist(ctx, m, users[0], "second"); e != nil {
		t.Fatal(e)
	}
	if e := s.RecoverPending(ctx); e != nil {
		t.Fatal(e)
	}
	p, e := s.ListHistory(ctx, users[0], Filter{Limit: 1}, "")
	if e != nil || len(p.Items) != 1 || p.Items[0].MatchID != m.MatchID || p.NextCursor == "" {
		t.Fatal(p, e)
	}
	next, e := s.ListHistory(ctx, users[0], Filter{Limit: 1}, p.NextCursor)
	if e != nil || len(next.Items) != 1 || next.Items[0].MatchID != r.MatchID || next.NextCursor != "" {
		t.Fatal(next, e)
	}
	p, e = s.ListHistory(ctx, users[0], Filter{Game: "race"}, "")
	if e != nil || len(p.Items) != 0 {
		t.Fatal(p, e)
	}
	from := time.UnixMilli(1500)
	p, e = s.ListHistory(ctx, users[0], Filter{World: r.WorldID, Status: "aborted_server", From: &from}, "")
	if e != nil || len(p.Items) != 1 || p.Items[0].MatchID != m.MatchID {
		t.Fatal(p, e)
	}
}

func TestPostgresRecoveryRetryBackoffKeepsAdmissionClosed(t *testing.T) {
	s, r, _ := fixture(t)
	ctx := context.Background()
	if e := s.Enqueue(ctx, r); e != nil {
		t.Fatal(e)
	}
	s.Projectors["duck"] = func(context.Context, pgx.Tx, model.MatchResult) error {
		return errors.New("synthetic projector unavailable")
	}
	o := &ownerFixture{}
	w := NewWorker(s, o)
	now := time.Now()
	w.Step(ctx, now)
	w.Step(ctx, now.Add(time.Second))
	var attempts int
	var delayed bool
	if e := s.Pool.QueryRow(ctx, `SELECT attempts,next_attempt_at>now() FROM result_jobs`).Scan(&attempts, &delayed); e != nil {
		t.Fatal(e)
	}
	if attempts != 1 || !delayed || !o.blocked {
		t.Fatal(attempts, delayed, o)
	}
	delete(s.Projectors, "duck")
	w.Step(ctx, now.Add(5*time.Second))
	if o.blocked {
		t.Fatal("successful retry did not reopen admission")
	}
}

type finalizeAckStore struct {
	*Service
	lost bool
}

func (s *finalizeAckStore) Finalize(ctx context.Context, id string) (model.MatchResult, error) {
	r, e := s.Service.Finalize(ctx, id)
	if e == nil && !s.lost {
		s.lost = true
		return r, errors.New("synthetic final commit acknowledgement lost")
	}
	return r, e
}
func TestPostgresFinalCommitAckLossCannotPublishPrematureSaved(t *testing.T) {
	s, r, _ := fixture(t)
	o := &ownerFixture{refs: []lobby.FinalizingMatch{{MatchID: r.MatchID}}, result: r}
	w := NewWorker(&finalizeAckStore{Service: s}, o)
	ctx := context.Background()
	now := time.Now()
	w.Step(ctx, now)
	if o.saved || !o.blocked {
		t.Fatal("unknown commit published saved", o)
	}
	var status string
	if e := s.Pool.QueryRow(ctx, `SELECT status FROM matches`).Scan(&status); e != nil || status != "completed" {
		t.Fatal(status, e)
	}
	w.Step(ctx, now.Add(4*time.Second))
	if !o.saved || o.blocked {
		t.Fatal("commit lookup did not reconcile", o)
	}
}

func TestPostgresConnectionOutageRetainsUnconfirmedUntilRecovery(t *testing.T) {
	for _, durable := range []bool{false, true} {
		t.Run(map[bool]string{false: "no_job", true: "durable_job"}[durable], func(t *testing.T) {
			s, r, _ := fixture(t)
			ctx := context.Background()
			if durable {
				if e := s.Enqueue(ctx, r); e != nil {
					t.Fatal(e)
				}
			}
			config := s.Pool.Config().Copy()
			s.Pool.Close() // An unavailable real connection pool, isolated from the shared server.
			o := &ownerFixture{refs: []lobby.FinalizingMatch{{MatchID: r.MatchID}}, result: r}
			w := NewWorker(s, o)
			var statuses []string
			w.Notify = func(_ string, _ []string, e StatusEvent) { statuses = append(statuses, e.Status) }
			now := time.Now()
			w.Step(ctx, now)
			if o.saved || o.detached || !o.blocked {
				t.Fatal(o)
			}
			w.Step(ctx, now.Add(60*time.Second))
			if o.saved || !o.detached || !o.blocked || statuses[len(statuses)-1] != "unconfirmed" {
				t.Fatal(o, statuses)
			}
			restored, e := pgxpool.NewWithConfig(ctx, config)
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(restored.Close)
			s.Pool = restored
			w.Step(ctx, now.Add(100*time.Second))
			expected := "aborted_storage"
			if durable {
				expected = "completed"
			}
			if o.blocked || statuses[len(statuses)-1] != expected {
				t.Fatal(o, statuses)
			}
			var status string
			var claims int
			restored.QueryRow(ctx, `SELECT status FROM matches`).Scan(&status)
			restored.QueryRow(ctx, `SELECT count(*) FROM active_match_users`).Scan(&claims)
			if status != expected || claims != 0 {
				t.Fatal(status, claims)
			}
		})
	}
}

func TestPostgresStartupFencesSettlingEnqueueAndRecoversOnce(t *testing.T) {
	s, r, _ := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if _, e := s.Pool.Exec(ctx, `CREATE TABLE synthetic_projection(match_id uuid PRIMARY KEY)`); e != nil {
		t.Fatal(e)
	}
	s.Projectors["duck"] = func(ctx context.Context, tx pgx.Tx, r model.MatchResult) error {
		_, e := tx.Exec(ctx, `INSERT INTO synthetic_projection VALUES($1)`, r.MatchID)
		return e
	}
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	var blocker int
	if e = tx.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&blocker); e != nil {
		t.Fatal(e)
	}
	if e = lockMatch(ctx, tx, r.MatchID); e != nil {
		t.Fatal(e)
	}
	payload, hash, e := encodeResult(r)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(ctx, `INSERT INTO result_jobs(match_id,result_hash,payload) VALUES($1,$2,$3)`, r.MatchID, hash[:], payload); e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(ctx, `UPDATE matches SET status='finalizing' WHERE id=$1`, r.MatchID); e != nil {
		t.Fatal(e)
	}
	recovered := make(chan error, 1)
	go func() { recovered <- s.RecoverPending(ctx) }()
	// Observe actual database blocking, whether the implementation waits on the
	// old row lock (the bug) or the shared advisory lock (the required fence).
	deadline := time.Now().Add(3 * time.Second)
	for {
		var waiting bool
		if e = s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE $1=ANY(pg_blocking_pids(pid)))`, blocker).Scan(&waiting); e != nil {
			t.Fatal(e)
		}
		if waiting {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("startup did not wait for settling enqueue")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if e = tx.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	if e = <-recovered; e != nil {
		t.Fatal(e)
	}
	var status string
	var claims int
	if e = s.Pool.QueryRow(ctx, `SELECT status FROM matches WHERE id=$1`, r.MatchID).Scan(&status); e != nil {
		t.Fatal(e)
	}
	if e = s.Pool.QueryRow(ctx, `SELECT count(*) FROM active_match_users WHERE match_id=$1`, r.MatchID).Scan(&claims); e != nil {
		t.Fatal(e)
	}
	if status != "finalizing" || claims != 1 {
		t.Fatalf("settling durable job orphan-aborted or claim prematurely released: %s, claims=%d", status, claims)
	}
	o := &ownerFixture{}
	w := NewWorker(s, o)
	now := time.Now()
	w.Step(ctx, now)
	if !o.blocked {
		t.Fatal("startup admitted before durable projection")
	}
	w.Step(ctx, now.Add(time.Second))
	w.Step(ctx, now.Add(2*time.Second))
	var projected int
	if e = s.Pool.QueryRow(ctx, `SELECT count(*) FROM synthetic_projection`).Scan(&projected); e != nil {
		t.Fatal(e)
	}
	if e = s.Pool.QueryRow(ctx, `SELECT count(*) FROM active_match_users`).Scan(&claims); e != nil {
		t.Fatal(e)
	}
	if e = s.Pool.QueryRow(ctx, `SELECT status FROM matches WHERE id=$1`, r.MatchID).Scan(&status); e != nil {
		t.Fatal(e)
	}
	if projected != 1 || status != "completed" || claims != 0 || o.blocked {
		t.Fatal("startup recovery did not complete exactly once/reopen admission", projected, status, claims, o.blocked)
	}
}
