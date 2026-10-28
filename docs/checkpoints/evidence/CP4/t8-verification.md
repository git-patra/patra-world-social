# T8 — durable match results and private history

Status: DONE. Implementation remains unstaged on `feat/social-world`; no commit, preview restart, source-repository changes, remote, push, or deployment performed by this worker. Root owns `docs/PROGRESS.md` and the guest-preview evidence; those working-tree changes are not from this worker.

## Architecture and source ordering

Root approved the proposal before implementation and confirmed the T7 preview was running before edits began. Implemented the approved `model.MatchResult` → owner-bound `ResultEngine.Result()` copy → durable `result_jobs` → transactional `Finalize` → participant-only status event path.

1. T7 commits the frozen roster, version, seed and prepared timestamp before runtime activation. The optional result capability must return the real `Start(startsAt)` timestamp. `lobby.CopyResult` verifies it against the owner’s actual `StartsAt`, clones the typed result under the owner, and performs no SQL/network. No concrete engine is registered in production by T8.
2. `Enqueue` validates typed fields and binds IDs, game, versions, prepared timestamp and the complete ordered actor roster to the persisted T7 start payload. A persisted schedule must equal the actual result start; NULL schedule is accepted. The immutable hash covers all typed result data, including timing and roster. The seed is absent from the result type and every public history serializer.
3. Enqueue locks the same per-match advisory transaction key used by T7 (`hashtextextended(matchID,7)`), inserts one immutable job, and marks the record finalizing atomically. Identical retry is idempotent. A different result cannot overwrite it; conflict returns an investigation error. A database trigger also forbids changing job identity/hash/body.
4. Finalize locks the job, verifies its hash, applies optional `Projectors[game]` within the same transaction, updates participant/result projections and the real start/end times, and commits completed. Only afterward can the worker transition the lobby to `results` and emit `completed`. Projectors must be registered before the worker starts; T9–T12 implement concrete detail/PB/champion rules and lock their world/ruleset leaderboard row inside this supplied transaction.
5. A completed live match keeps its result screen and claims. T9–T11 own display/return timers and call existing `ReleaseMatch(matchID,"completed")`; T8 does not prematurely erase the screen. The completed guard remains in PGStore.

Supporting files were split by responsibility: `history/query.go`, `games/model/result.go`, `lobby/results.go`, and private delivery support in `realtime/delivery.go`, in addition to the requested service/finalizer/worker/http/migration/query/UI paths. SQLc was regenerated.

## Failure, cancellation, restart and ownership

- One worker owner polls every 250 ms. Storage attempts use a maximum three-second context and are synchronous/joined before another operation starts. Every SQL operation is also capped by the earliest active result’s 60-second deadline, so another pending job cannot hold that deadline open. Retries back off 2/4/8/16/32 seconds.
- The 60-second interval starts from the authoritative lobby finalizing transition, not from a later poll. At expiry the already-cancelled/joined write is no longer running in this process; `DetachUnconfirmed` drops the engine and full result copy, returns local claims through the unchanged generation-fenced `WithReturn`/`AckReturn` path, and emits **Hasil belum dapat dikonfirmasi**. Small match/world/user ID metadata remains for reconciliation, bounded by admitted active matches; new Start reservations are blocked on the lobby owner. Recovery backlog ingestion is separately bounded to 50 pending IDs.
- The original DB active-arena/user claims remain until outcome determination. Reconcile takes the enqueue advisory lock before deciding absence, so a server-side commit with a lost acknowledgement cannot be mistaken for missing work. Durable jobs finish; absent jobs become `aborted_storage`. Claims are deleted only by the resolved old match ID. Admission stays closed while any pending/reconciliation work or database probe error remains, including jobs delayed by retry backoff.
- Missing result capability/extraction failure is observable (`unavailable` event + match-ID-only error log), is copied/attempted only once, and never creates a dummy result/job. Invalid/conflicting result payloads stop repeat enqueue attempts and log an investigation requirement. After the uncertain deadline, normal durable-outcome reconciliation applies.
- `RecoverPending` runs before lobby admission during `Application.New`. Running/finalizing rows without jobs become `aborted_server`, regardless of NULL/non-NULL starts; actor rows remain. Completed/terminal old-runtime claims are released. Fresh worker instances resume durable jobs and keep starts gated until resolution. This follows the application’s existing single authoritative game-process model; it does not introduce multi-process runtime ownership/leases.
- Expected injected faults log only match IDs, not seed, scores, session material or secret choices.

## Personal history and future game/deletion seams

- Authenticated GET `/api/v1/history` and `/api/v1/history/{matchID}` are wrapped in the existing SessionMiddleware and derive identity from SessionFromContext. No official-result write route exists. Own participation authorizes reads after membership revocation/world archive; another account’s detail is 404, guest is 401, caller-supplied `userId` filter is rejected.
- Filters: `game`, `world`, `status`, RFC3339 `from`/`to` over prepared time; stable descending `(prepared_at,id)` cursor; default20/max50. Unknown-start aborts return NULL start and no invented score/duration.
- `HistoryPage`/`MatchDetail` provide filters, paging, private detail, status, local-time timestamps, roster/BOT/assisted flags, scores/positions when actually present, versions, resolved penalty turns, race laps/bestLap, and duck accuracy derived from hits/shots. Status events contain only matchID/status and go only to participating joined accounts, never spectators. The contract enum/schema, Go and TS validators, golden fixture, socket dispatch and lobby labels were updated together.
- Typed details include `RaceResult.LapTimesMs/BestLapMs/ElapsedMs/DNF`, `DuckResult.Hits/Shots`, and resolved penalty-turn data. No winner computation or concrete rule engine was introduced.
- T14 seam: `query.go/readRecord` overlays public result actors from mutable `match_participants` projections. Immutable job JSON is not the sole public identity source. T14 must anonymize those actor projections (and implement its account/tombstone/user-ID policy); public result serialization then substitutes them without modifying competitive result hashes. Tested by changing projected identity while retaining successful identical enqueue.

## Verification and exact commands

All commands ran in `/Users/patra/Project/patra-world-social`, unless stated otherwise. The existing ignored database environment was exported with `set -a; source .cache/database.env; set +a` without printing its contents. Integration tests use `testutil.Database` random schemas with cleanup on the dedicated local PostgreSQL. Network-listening/PG test commands needed the normal tool sandbox escalation; no permission rejection occurred.

Initial meaningful RED:

```
cd apps/server
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -tags integration ./internal/history
--- FAIL: TestResultsMigration
    durable result_jobs missing
FAIL
```

The first sandboxed invocation could not connect (`operation not permitted`); it was rerun with allowed escalation to reach the above real RED. Later API tests initially failed to build because Enqueue/Finalize/history methods did not exist, and the owner-gate test failed to build because SetStorageBlocked did not exist. The result/migration and finalizer/authority/fault tests then passed. This records actual ordering rather than claiming every new test first reached a runtime assertion failure.

Focused implementation checks included:

```
go test -tags integration ./internal/history
ok patra-world-social/server/internal/history 1.073s

go test ./internal/lobby
ok patra-world-social/server/internal/lobby 0.519s

go test -tags integration ./internal/history ./internal/lobby ./internal/application ./internal/realtime
```

That integration run passed lobby (1.545s), application (18.354s) and realtime (10.328s); the new HTTP test initially returned401 because my synthetic token fixture was not the required43 characters. It was corrected to an explicit43-character synthetic token, then:

```
go test -tags integration ./internal/history -run TestPostgresHTTPUsesSessionAndParticipation -v
--- PASS: TestPostgresHTTPUsesSessionAndParticipation (0.11s)
PASS
ok patra-world-social/server/internal/history 0.634s
```

A sandboxed realtime unit run also could not bind httptest’s local port; it was subsequently covered successfully by the escalated full race run. Initial frontend typecheck exposed JSX/type annotations and the missing TS event enum update; those were corrected before final validation.

Final covering Go command (full output also `.cache/t8-go-race.log`):

```
set -o pipefail
set -a
source .cache/database.env
set +a
cd apps/server
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -race -tags integration ./... 2>&1 | tee ../../.cache/t8-go-race.log
```

Exit0. Exact package output:

```
?   patra-world-social/server/cmd/e2edb [no test files]
ok  patra-world-social/server/cmd/server (cached)
ok  patra-world-social/server/internal/application 22.015s
ok  patra-world-social/server/internal/auth 7.579s
ok  patra-world-social/server/internal/chat 3.469s
ok  patra-world-social/server/internal/config (cached)
?   patra-world-social/server/internal/contracts [no test files]
?   patra-world-social/server/internal/db [no test files]
ok  patra-world-social/server/internal/games/model 2.633s
ok  patra-world-social/server/internal/history 6.377s
ok  patra-world-social/server/internal/http 1.800s
ok  patra-world-social/server/internal/lobby 4.986s
ok  patra-world-social/server/internal/moderation 5.609s
ok  patra-world-social/server/internal/realtime 12.353s
ok  patra-world-social/server/internal/social 3.919s
?   patra-world-social/server/internal/testutil [no test files]
ok  patra-world-social/server/internal/world 3.382s
ok  patra-world-social/server/internal/worlds 3.060s
```

Explicit required unit command, from apps/server:

```
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test ./internal/history
ok patra-world-social/server/internal/history 0.536s
```

Self-review added one additional real unavailable-connection-pool test (no production changes). It closes only its isolated pool, restores a new pool with the same schema, and proves both absent-job and durable-job recovery without stopping the shared PostgreSQL server. Focused verification only, no repeated broad suite:

```
go test -race -tags integration ./internal/history -run TestPostgresConnectionOutageRetainsUnconfirmedUntilRecovery -v
=== RUN TestPostgresConnectionOutageRetainsUnconfirmedUntilRecovery/no_job
=== RUN TestPostgresConnectionOutageRetainsUnconfirmedUntilRecovery/durable_job
--- PASS: TestPostgresConnectionOutageRetainsUnconfirmedUntilRecovery (0.31s)
    --- PASS: .../no_job (0.12s)
    --- PASS: .../durable_job (0.19s)
PASS
ok patra-world-social/server/internal/history 1.912s
```

Final frontend command (full output `.cache/t8-frontend.log`):

```
(npm run lint && npm run typecheck && npm test && npm run build) > .cache/t8-frontend.log 2>&1
```

Exit0: lint/typecheck pass; JS tests8/8, bundled TS/controller/contract/render tests29/29 (37 total,0 failures); Vite156 modules, production build551ms, JS341.43kB / gzip110.05kB, CSS57.79kB / gzip13.86kB. Added render test verifies unknown-start abort has neither invented duration nor score, escaped participant identity, actual committed status and derived duck accuracy.

```
/opt/homebrew/bin/sqlc generate -f apps/server/sqlc.yaml
```

Exit0. `git diff --check`: exit0/no output.

## Fault coverage and self-review

Covered real PostgreSQL immutable conflict/trigger rejection;12-way concurrent enqueue/finalize with one projector write; transaction trigger rollback including game projector; fresh-service and fresh-worker resume after enqueue; startup NULL-start orphan; absent/durable reconciliation; both enqueue and final commit lost-ACK wrappers around real commits; real uncommitted-enqueue advisory fencing; real blocked SQL context cancellation before deadline reconciliation; unavailable real pool and restored pool outcomes; retry metadata/backoff/admission gate; schedule projection failure with actual runtime time persisted; completed lobby retained until guarded release; guest/other-account/left-world/archived-world access; filters/cursor; identity overlay; participant-only event delivery; missing engine result capability.

Self-review found and addressed: runtime finalizing time must determine deadline (not poll time); SQL probes must respect the earliest result deadline; delayed durable jobs must keep the admission gate closed; recovered jobs must retain their ID even when final COMMIT succeeds with lost ACK; public immutable actor snapshots need a mutable identity overlay; initial account lookup must not overwrite a newer logout event in history UI. Existing return generations and acknowledgement logic remain unchanged.

## NOT_RUN / handoff

- Worker did not run browser credentials, authenticated browser/online history acceptance, Playwright, CUA, session injection, or read `.cache/auth-e2e.env`. User-deferred browser acceptance remains deferred.
- Root independently performed guest-only History smoke (reported clean console), with root-owned `docs/checkpoints/evidence/CP4/t8-guest-preview.md` and PNG. Root stated preview API still runs accepted T7, so that is not T8 online/API evidence.
- Browser acceptance scenarios to run only when user resumes it: signed-in filter/paging/detail; other account cannot view a known match ID; own detail remains after leaving/archive; temporary/final saved labels; outage after60s shows unconfirmed and prevents Start until recovery; result event visibility restricted to participants; logout clears previous account history.
- T9–T11 implement real Result-capable engines and result return timers; T9–T12 register transactional projectors before worker startup. T11’s approved3-second Duck countdown correction remains with root/T11, not changed by T8. Timing validation uses actual runtime timestamps and makes no hard-coded duck0/race3 assumption.

## Coordinator verification supplement
After implementation report/source freeze, root ran missing Go vet/build checks (no production edits): `GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go vet ./...` from apps/server exited0 with no output. `GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go build -o ../../.cache/server-t8-check ./cmd/server` exited0 with no output. Staged diff check exited0. Root read actual .cache/t8-go-race.log and frontend build log; they match reported output. Existing passing test suites were not repeated.

## Review fix round 1/5 — R1 and R2

Status: ready for re-review. Both Important findings in `task-8-review.md` were reproduced before production fixes and addressed. This supersedes the initial self-review's conclusion that no implementation defect remained. Only four working-tree source/test files changed against the staged review baseline: `history/finalizer.go`, `finalizer_integration_test.go`, `worker.go`, and `worker_test.go`. This worker did not stage, commit, edit PROGRESS, restart previews, or use browser/session flows.

### R1: startup recovery fence and fresh read

The old bulk orphan UPDATE could use an absent-job snapshot taken before a settling enqueue committed. RecoverPending now enumerates candidates only, then starts an explicit READ COMMITTED transaction per match, acquires the existing shared match advisory fence, and performs a **new statement** reading current status and durable-job existence after the fence is held. Only that fresh read can choose `aborted_server` or release terminal claims. Durable pending jobs retain their claims until worker completion/reconciliation. Truly absent NULL-start orphans still become aborted_server; the existing test remains green.

Permanent regression `TestPostgresStartupFencesSettlingEnqueueAndRecoversOnce` opens a real enqueue transaction, inserts the job/updates finalizing while holding the shared fence, starts recovery concurrently, observes actual blocking using `pg_blocking_pids` and the enqueue connection's PID, then commits the enqueue. It checks finalizing/claim preservation, runs the real worker through recovery, verifies one projector row, completed status, zero stale claims, and reopened admission. Unlike the temporary review probe, it detects both row-lock and advisory-lock waiting without depending on the old UPDATE query text.

### R2: bounded discovery and shared Step budget

Worker Step now has one total three-second context covering owner intake, the selected storage attempt, retry bookkeeping, and both storage probes. It checks **every live deadline before choosing work**, then attempts at most one ready match. Live results have priority; retries within a class are ordered by ready time, with stable match-ID tie breaking. Every pending ID remains until resolved. The existing earliest-live-deadline context further shortens the shared budget, so cancellation/join and the next polling pass reach the 60-second detach boundary. A detached backlog can no longer consume N separate three-second waits before the next owner/deadline check.

The permanent `TestWorkerDiscoversNewDeadlineDuringSlowDetachedBacklog` seeds21 detached outage IDs and exposes a newly finalizing reference inside the first blocking Reconcile. It uses real context waits and production Step scheduling, bounds the first Step's work, then advances the injectable Step clock near and across that match's authoritative deadline. Both absent/durable variants verify discovery, release and unconfirmed notification within the250ms polling allowance at the deadline check, continued admission blocking, no earlier terminal outcome, and eventual backlog drainage/correct outcome/admission reopening after storage recovery. It does not stop at asserting that one Step is faster. The 60-second boundary portion uses Step's virtual time; it does not spend60 wall-clock seconds per variant.

Shared enqueue/finalize/reconcile fencing, synchronous/joined writes, existing owner Start gate, unconfirmed wording, completed live result screens, and generation-fenced lobby returns are unchanged.

### RED and GREEN commands/output

From repo root, export the existing ignored database configuration without printing it:

```
set -a
source .cache/database.env
set +a
cd apps/server
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -tags integration ./internal/history -run 'TestPostgresStartupFencesSettlingEnqueueAndRecoversOnce|TestWorkerDiscoversNewDeadlineDuringSlowDetachedBacklog' -count=1 -v
```

Before fixes, exit1:

```
=== RUN   TestPostgresStartupFencesSettlingEnqueueAndRecoversOnce
    finalizer_integration_test.go:651: settling durable job orphan-aborted or claim prematurely released: aborted_server, claims=0
--- FAIL: TestPostgresStartupFencesSettlingEnqueueAndRecoversOnce (0.11s)
=== RUN   TestWorkerDiscoversNewDeadlineDuringSlowDetachedBacklog
=== RUN   TestWorkerDiscoversNewDeadlineDuringSlowDetachedBacklog/absent
    worker_test.go:177: detached backlog monopolized owner discovery: 21 reconciliations, 6.000396667s
=== RUN   TestWorkerDiscoversNewDeadlineDuringSlowDetachedBacklog/durable
    worker_test.go:177: detached backlog monopolized owner discovery: 21 reconciliations, 6.001486292s
--- FAIL: TestWorkerDiscoversNewDeadlineDuringSlowDetachedBacklog (12.00s)
    --- FAIL: TestWorkerDiscoversNewDeadlineDuringSlowDetachedBacklog/absent (6.00s)
    --- FAIL: TestWorkerDiscoversNewDeadlineDuringSlowDetachedBacklog/durable (6.00s)
FAIL
FAIL patra-world-social/server/internal/history 12.693s
```

The same focused command after fixes, exit0:

```
=== RUN   TestPostgresStartupFencesSettlingEnqueueAndRecoversOnce
--- PASS: TestPostgresStartupFencesSettlingEnqueueAndRecoversOnce (0.13s)
=== RUN   TestWorkerDiscoversNewDeadlineDuringSlowDetachedBacklog
=== RUN   TestWorkerDiscoversNewDeadlineDuringSlowDetachedBacklog/absent
=== RUN   TestWorkerDiscoversNewDeadlineDuringSlowDetachedBacklog/durable
--- PASS: TestWorkerDiscoversNewDeadlineDuringSlowDetachedBacklog (12.01s)
    --- PASS: TestWorkerDiscoversNewDeadlineDuringSlowDetachedBacklog/absent (6.00s)
    --- PASS: TestWorkerDiscoversNewDeadlineDuringSlowDetachedBacklog/durable (6.00s)
PASS
ok patra-world-social/server/internal/history 12.851s
```

Covering race/real-PG check from repo root (log `.cache/t8-fix1-race.log`):

```
set -o pipefail
set -a
source .cache/database.env
set +a
cd apps/server
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -race -tags integration ./internal/history ./internal/lobby ./internal/application ./internal/realtime -count=1 2>&1 | tee ../../.cache/t8-fix1-race.log
```

Exit0:

```
ok patra-world-social/server/internal/history 15.863s
ok patra-world-social/server/internal/lobby 3.175s
ok patra-world-social/server/internal/application 20.157s
ok patra-world-social/server/internal/realtime 10.403s
```

From apps/server:

```
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go vet ./... && GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go build -o ../../.cache/t8-fix1-server ./cmd/server
```

Exit0, no output. `git diff --check`: exit0, no output. No frontend changes in this fix round, so the previously passing frontend checks were not repeated. Existing credentialed-browser/CP4 deferrals and the original single authoritative game-process limitation remain; no new scope or claim of browser acceptance was added.
