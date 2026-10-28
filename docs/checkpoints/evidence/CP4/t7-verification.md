# T7 implementation report — ready for review

Base: ef7ed2dddd738c36392d88c7a525e8af1b849d95. No commits/staging, preview restarts, root ledger edits, source-repo edits, or browser credential actions.

## Implementation

- Generic race / penalty / four duck cabinet registry. Bounded128 owner request queue, separate storage work, 15second game grace independent of30second world seats. All production factories absent, so lobby available but Start unavailable.
- Internal model.Actor/MatchStart, frozen roster and bot fill. Seed stays internal; public Lobby excludes it.
- PostgreSQL004 matches/participants/active user claims, unique active world-arena and host-command identity. Transaction persists running+prepared_at before countdown; nullable starts_at is a metadata projection per coordinator ruling. Race countdown is a full3seconds beginning at confirmed persistence time; penalty/duck have zero generic prep delay.
- Ambiguous COMMIT keeps reserved state and stable match ID. Same-command retry reconciles before activating; definitive precommit failure restores lobby readiness. T8 owns result finalization/recovery.
- Hub game dispatch and coalesced presence notifications; current authenticated session/seat derives actor, service public resync through lobby.list. World owner playing status neutralizes movement and supports safe entrance returns. Application composition/owned lifecycle publisher wired.

## Focused test evidence to date

1. Initial `GOCACHE=.../.cache/go-build /opt/homebrew/bin/go test -race ./internal/lobby` RED compile: undefined model.MatchStart/Service/New/Lobby (new feature absent). After implementation RED host migration assertion (host remained a after leaving); fixed aliasing of removed slice element. GREEN `ok .../internal/lobby 1.484s`.
2. Added PostgreSQL tests RED compile `undefined: PGStore`. Initial sandbox denied loopback, then approved escalation with dedicated localDB55432 and randomized schemas. Exact command from apps/server: `set -a; source ../../.cache/database.env; set +a; GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -race -tags=integration ./internal/lobby`. GREEN `ok .../internal/lobby 1.561s`. Tests exercise12 concurrent same-command Starts, one persisted match/five actors, active user and arena DB uniqueness, failure rollback, lost COMMIT acknowledgement + unavailable reconciliation, retained claims, and same-ID retry.
3. Strict lobby protocol RED: missing arenaId accepted, spoofed userId/hostId accepted, lobby.list/bot unavailable, ready missing lobbyId accepted. Added validation. World activity RED compile `Runtime.Activity undefined`. GREEN focused: `...go test -race ./internal/lobby ./internal/world ./internal/realtime -run 'TestLobby|TestConcurrentStart|TestStartFailure|TestThreeGame|TestPlaying'`: lobby1.606s, world2.078s, realtime2.529s.
4. Earlier broad realtime attempt under sandbox could not open httptest listener (`bind: operation not permitted`); it is an environment restriction, needs approved escalation during full suite.

## Initial work remaining at first checkpoint (now completed below)

Production React lobby panel/mounting and Go/TS/schema golden synchronization; actual protocol integration tests; expanded start/shutdown edge verification; self-review, relevant full suites and final changed-file list. Browser acceptance remains NOT_RUN as requested. No concrete game engines, saved results, history projections or game consequences are claimed in T7.

## Subsequent edge verification / self-review

- Production React feature is mounted from race/penalty/duck dialogs and exposes six actual registry arenas. No proximity enrollment. Ready/explicit penalty bot choice/host Start/spectator leave and unavailable engine label are visible. WorldSocket adds lobby events and ACK-bound control calls; Start command UUID is retained per lobby across retries/reconnects. Public resync excludes seeds.
- `npm test` GREEN8JS+28bundled TS/React/socket tests. `npm run typecheck --workspace apps/web`, `npm run lint --workspace apps/web`, `npm run build --workspace apps/web` GREEN. Build153modules, JS334.63kB/gzip108.14kB,219ms. Typecheck initially caught UUID template-string inference; annotated retained IDs as string. React SSR bundling needed standard Node createRequire banner in existing test bundler.
- Natural split: service.go owns registry/roster and bounded ownership; start.go handles reserve/persist/reconcile/activate; presence.go handles lifecycle/finalizer hooks; store.go owns PostgreSQL transactions. No framework dependency.
- New actual PostgreSQL participant-write trigger failure rolls back match and claims and restores ready lobby. Unknown COMMIT followed by revocation reconciles same durable ID, never calls engine.Start for an entirely departed roster, and releases arena/claims. Actual WebSocket test verifies admitted actor, non-host denial, production unavailable Start, seed-free six-arena resync, busy movement denial and revoke cleanup. Scoped integration GREEN lobby2.004s/application2.220s.
- Self-review found engine.Start was called before all-departed detection: RED `engine activated with no human`; moved activation behind human check; GREEN. Coalesced disconnect+resume reaches engine in order and marks botAssisted. Definite failed starts apply departures and host migration before restoring lobby.
- Self-review found natural finalizing result could be replaced by abandoned after final human departure: RED `natural result replaced by departure`; now only countdown/running can auto-abandon and engine presence stops once finalizing/results.
- Internal accepted-operation completions wait for queue capacity (rather than dropping on queue-full); public commands still reject when bounded128 queue is full. Added deterministic saturated-queue test.
- Single-user departure claim is deleted by matchID+userID outside owner; busy remains until DB confirms deletion. Frozen participant stays for engine DNF/history. Retry delay1second, bounded3second context. Stale memory completion is bound to lobbyID; it cannot clear a newer lobby claim. Whole completed release is rejected by PGStore until durable match status already completed.
- Full Go integration/race run initially found a timing assumption in own revoke test (`running 0 0`): individual-user release completed before whole-arena terminal transaction. Test now waits for authoritative arena idle before checking durable abandoned. That run otherwise passed application/auth/chat/config/model/http/moderation/realtime/social/world/worlds. Final all-package rerun in progress after this and natural-result correction.

## T8/concrete engine handoff (current API)

- Register(game, Factory) is internal composition only; production registers none. Factory receives a cloned internal MatchStart and prepares inert state. Engine.Start(startsAt) activates after durable running confirmation; Engine.Tick(now) returns true on natural finish. Owner then enters finalizing and emits coalesced Changed -> application lobby.updated. Engines must not call service methods reentrantly from Tick/Presence/Start, nor do network/SQL there.
- WithEngine(ctx,matchID,userID,callback) runs input/result extraction on owner; userID empty is internal finalizer authority, nonempty is checked against connected, nondeparted participant. Finalizer copies immutable result inside callback, then enqueues/finalizes outside owner. Transition(ctx,matchID,"results") follows durable success; ReleaseMatch(matchID,"completed") verifies PG completed status before releasing claims/returning users.
- T8 still owns durable result_jobs, completed projection, startup aborted_server recovery,60second uncertain-storage detachment/reconciliation and saved/unconfirmed UI. T9–T11 own game-specific input, penalties/DNF/bot simulation, race grid/countdown enforcement and result/return timers. T13 owns voice release on game entry. T7 does not assert any of these are complete.
- Initial start uncertainty has bounded1second reconciliation with stable ID; idempotent Persist verifies immutable start_payload and command identity. It never frees an unresolved start merely because COMMIT acknowledgement failed. Schedule is a nullable metadata projection; failures log only generic match ID. Engine/result input retains exact official startsAt for T8.

## Full verification before final registry timing ruling

Exact working directory for Go commands: `/Users/patra/Project/patra-world-social/apps/server`. DedicatedDB is sourced with `set -a` / `source ../../.cache/database.env` / `set +a`; credentials never printed. All integration tests provision randomized `social_test_*` schemas through testutil.Database and register cleanup.

- `/opt/homebrew/bin/sqlc generate` — PASS; generated matches.sql.go and appended match table structs in db/models.go, no drift in existing generated query files.
- `GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -race -tags=integration ./...` — PASS (approved local socket/DB escalation). Final output: cmd/server cached; application18.914s; auth6.531s; chat2.929s; config cached; games/model cached; http3.280s; lobby4.241s; moderation4.812s; realtime cached; social4.299s; world cached; worlds5.856s. cmd/e2edb/contracts/db/testutil have no test files. Zero failing packages.
- `GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go vet ./...` — PASS, no output.
- `GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go build -o ../../.cache/server-t7-check ./cmd/server` — PASS, no output. Does not replace/restart preview executable.
- `npm test` from repo — PASS8JS+28bundled TS/React/controller/socket tests, final bundled duration653.84ms.
- `npm run typecheck --workspace apps/web` — PASS.
- `npm run lint --workspace apps/web` — PASS.
- `npm run build --workspace apps/web` — PASS153modules, JS334.63kB/gzip108.14kB,219ms.
- `git diff --check` — PASS, no output.

Start reconciliation additionally uses a per-match PostgreSQL advisory transaction lock in both Persist and Lookup. A false Lookup cannot run ahead of an unresolved original COMMIT. Real PostgreSQL test holds that lock and verifies Lookup times out rather than incorrectly reporting no row, then reports absence only after rollback releases the lock. No initial-start claims are freed on an unknown acknowledgement. Narrow retry only covers active start/claim cleanup, bounded by active arenas/participants, not a general result retry framework.

## Changed files

New:
- apps/server/internal/lobby/{registry,service,start,presence,store}.go
- apps/server/internal/lobby/{service_test,store_integration_test}.go
- apps/server/internal/db/migrations/004_matches.sql
- apps/server/db/queries/matches.sql
- apps/server/internal/db/matches.sql.go (generated)
- apps/server/internal/application/{lobby,lobby_integration_test}.go
- apps/server/internal/realtime/lobby_protocol_test.go
- apps/server/internal/world/{activity,activity_test}.go
- apps/web/src/features/lobby/{LobbyPanel.tsx,lobby.css}

Modified:
- apps/server/internal/application/application.go
- apps/server/internal/db/models.go (generated)
- apps/server/internal/games/model/types.go
- apps/server/internal/realtime/{delivery,hub,protocol}.go
- apps/server/internal/world/movement.go
- apps/web/src/{app.js,net/socket.ts,world/avatars.ts}
- apps/web/tests/map.test.ts
- apps/web/scripts/test-map.mjs
- apps/web/tsconfig.json
- contracts/{commands.golden,protocol.schema}.json

Coordinator-owned docs/PROGRESS.md was already dirty and has not been edited by this implementer. Scratch task report is the only implementer documentation write. No staging/commits/remotes/push/deploy, source-repo changes or production accounts.

## Remaining acceptance / practical limits

- Browser/visual/keyboard/touch acceptance NOT_RUN, per authorization boundary. Preview API8080 and Vite4176 remain coordinator-owned and were not restarted; running API has not adopted these uncommitted server changes.
- Production starts unavailable until concrete T9–T11 engines register. Test engines exist only in *_test.go. No production fake countdown/play/results, no saved-result claim.
- Actual race simulation/grid, penalty choice3s, duck scores/podium, DNF/forfeit consequences, result presentation timers, durable finalization/history/recovery,60second result-storage uncertainty handling and voice release remain their planned task responsibilities.
- Beta assumes one authoritative server process; DB unique claims protect against duplicate active records, but this change does not introduce distributed simulation failover.


## Final registry timing ruling and verification ordering

Coordinator confirmed race3seconds, penalty/duck zero generic prep delay. Added `TestRegistryStartDelayBelongsOnlyToRace` for all3 game types before code change: RED penalty/duck official start103000 rather than expected100000. Registry now owns unexported countdown duration; race3seconds, other arenas0. Zero-delay matches enter running immediately after confirmed persistence. Penalty3second choices and duck60second sessions remain concrete engine clocks. Existing slow initial store/full3second race and slowSchedule/shutdown regressions remain green.

Final source change touches only lobby registry/start timing and its assertion. After the full Go race/integration pass above, the affected verification was rerun (no unrelated whole-suite repetition):
- `GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -race ./internal/lobby` — PASS1.505s.
- `GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -race -tags=integration ./internal/lobby` — PASS2.106s.
- `GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go vet ./internal/lobby ./internal/application` — PASS, no output.
- `GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go build -o ../../.cache/server-t7-check ./cmd/server` — PASS, no output.
- `git diff --check` — PASS. Frontend unchanged since its green test/typecheck/lint/build runs.

Ready for coordinator review; no known blocking implementation defect. Browser acceptance and concrete game/results/voice integration remain explicitly deferred to their authorized tasks.

## Fix round 1 — P2 stale arena return ownership race

Read the finding verbatim in `task-7-review.md` and applied the receiving-code-review/TDD workflow. Only this Important finding and its return-delivery error handling were changed. Working tree only; the SHA-256 of `git diff --cached --binary` matches the value captured before this round in `task-7-fix1-index.sha256`. No staging/commits, root ledger edits, browser credential actions, preview restarts, subagents or T8 implementation.

### RED reproduction

Extracted the existing return loop into a narrowly typed application helper without changing its behavior, then added real-PG application tests. `TestArenaReturnCannotOverwriteNewerLobbyClaim` pauses the old race return after eligibility has been observed, joins the penalty lobby through the actual authenticated Hub.Execute path, then resumes the old return. RED: before avatar `(32,32,playing)` became `(30,14,playing)` after the stale return. The playing label in the snapshot did not hide the position assertion. `TestArenaReturnRemainsPendingAfterDeliveryError` also failed RED with `delivery error was treated as successful`.

Exact RED command from apps/server (environment sourced/exported without printing credentials):
`GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -count=1 -race -tags=integration ./internal/application -run TestArenaReturn`
Output: both named regressions FAIL; application0.882s.

### Fix / ownership boundary

- Every lobby claim increments a per-user ownership generation. Internal Return carries the generation captured after releasing that claim. Rejoining the same still-existing lobby also gets a new generation.
- `lobby.Service.WithReturn` checks both no current busy claim and exact generation while holding the same ownership read lock that excludes a new claim write. It runs only the direct world-owner state mutation while holding that fence.
- `world.Runtime.ReturnActivity` invokes the guard inside the world owner request, around activity neutralization/status/coordinates together. Hub.ReturnActivity forwards the guarded request. A stale return reports `world.ErrActivitySuperseded` without changing position or activity.
- Lock order remains Hub.mu → world owner → short ownership read lock. Claim writes take only the ownership lock and release it before any cross-service work. The guarded callback performs no hub/service/storage/network call, so it does not introduce lock inversion or SQL in ticks. The earlier eligibility check in the application is merely an optimization; mutation correctness comes from the world-owner guard.
- `DrainChanges` now retains pending returns. `AckReturn` removes only the matching delivered/obsolete generation through the guaranteed internal control queue; an old acknowledgement cannot delete a newer return. Transient delivery errors remain pending and the publisher retries them on its100ms timer. Acknowledgement follows successful mutation or verified supersession, not an ignored error. The queue stays coalesced by world/user.

The deterministic gate exists only as a test implementation of the narrow application return interface. No production pause/test-hook field was added.

### GREEN / covering verification

Approved escalation was used only for dedicated localDB55432 and local test listeners. All PG fixtures use randomized synthetic schemas and cleanup. Exact commands from `/Users/patra/Project/patra-world-social/apps/server`:

1. `GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -count=1 -race -tags=integration ./internal/application -run 'TestArenaReturn|TestLobbyProtocol'` — PASS application1.890s after the fix.
2. `GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -count=1 -race ./internal/world ./internal/lobby ./internal/realtime` — PASS world1.501s, lobby1.626s, realtime10.383s.
3. `GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -count=1 -race -tags=integration ./internal/application ./internal/lobby -run 'TestArenaReturn|TestLobbyProtocol|TestPostgres'` — PASS application1.643s, lobby2.153s.
4. `GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go vet ./internal/application ./internal/lobby ./internal/realtime ./internal/world` — PASS, no output.
5. `git diff --check` — PASS. Index SHA-256 comparison — `Staged snapshot unchanged`.

New focused assertions additionally verify a rejected guard preserves both playing status and coordinates on the actual world owner, a current guard applies the safe return, same-lobby rejoin invalidates the old generation even after the new claim is released, and a stale acknowledgement preserves the newer return.

Changed working-tree files for this round: application/lobby.go and lobby_integration_test.go; lobby/service.go, presence.go and service_test.go; realtime/hub.go and delivery.go; world/activity.go and activity_test.go. No frontend, schema, engine, finalization or unrelated Minor changes. No source edits after the covering checks above.

Status: ready for scoped re-review of P2. Browser acceptance remains NOT_RUN/user-deferred; full-suite reruns were intentionally omitted because covering suites exercised the changed paths and the earlier full suite was already green.
