# T12 implementation report

Status: DONE_WITH_CONCERNS. Code and report frozen for independent review. Final serial Go race/integration, Go vet/build and frontend checks passed. The failed default-parallel chat timing test and unexecuted authenticated-browser acceptance remain explicitly recorded. This report is implementation/test evidence, not taskdone or checkpoint PASS.

Base: `ca12faeaba0cb5f5d3c350e5010853bfb6151e74`; accepted T11 `607141a58593f3abf473edb369148dcd59fed289`.

## Scope and architecture decisions

- Implemented Duck private board keyed by `world_id + game_type(duck) + rulesId@rulesVersion`. Current UI defaults to `duck-v1@1`; archived partitions remain independently selectable. No arbitrary result upload API.
- T8 composes `duck.Project` and `leaderboard.ProjectResult` inside its existing transaction before worker/recovery startup. The authoritative argument drives ranking; no dependency on not-yet-updated match result JSON. A receipt, PB, champion and Duck detail commit together. The existing immutable result hash/body contract is unchanged.
- Ranking is score descending, exact rational hits/shots via bounded integer cross-products, FinishedAt ascending, result UUID ascending. Zero shots means zero accuracy. One entry per account/partition enforces unique podium slots. Non-improving results never replace PB. Only finished human Duck results with account, score and Duck details are eligible; incomplete/bot/guest/assisted are excluded.
- Lock order is match advisory lock (namespace7), sorted account advisory locks (namespace12), then board row lock. A board row is inserted before lock so initially empty boards serialize too. Reads use one repeatable-read, read-only transaction for consistent PB/top/champion/last result and active membership authorization.
- Approved privacy boundary: `/api/v1/leaderboards/{world}` returns top3, caller's PB and last stored session, three champion changes, three administrative changes, partitions. `/api/v1/leaderboards/{world}/results/{match}` returns only a complete eligible Duck summary plus map/rules/protocol versions, scoped to that same world and active membership. No other-account private history, seed, hidden input or full result job payload is exposed. Personal `history.Detail` is unchanged and participant-only even after removal/archive. Guest401, outsider/kicked/banned/archived404; successful private reads are `Cache-Control: no-store`.
- First position ownership creates a champion event. Same owner improving does not. Removal recomputes current champion using a separate administrative audit, not a new victory. Old champion identity is anonymized and UI suppresses its result link.
- Approved deletion boundary: T14 owns the actual account-deletion endpoint/lifecycle. T12 provides `RemoveAccount(ctx,tx,user)` to call within that transaction before erasing identity. It takes the account lock before discovering sorted partitions, persists an opaque user-ID exclusion tombstone, removes entries, anonymizes champion references and writes administrative audit. The exclusion serializes late immutable jobs, including previously unseen partitions. Repeated removals are harmless. T14 must commit with deletion and then invalidate relevant worlds/revoke sessions; T12 does not pretend that future orchestration is implemented.
- `leaderboard.changed` contains only worldId/ruleset/resultId. Existing worker confirmed `completed` callback queries committed eligible partition identity and publishes to the world's active connected seats; participant-only `match.result_status` remains unchanged. No partial score/champion claims. Notifications are ephemeral: reconnect/load/focus/manual reload fetch current durable snapshots; lookup failure logs and relies on those refresh paths. UI deduplicates result IDs (bounded256), replaces snapshots, cancels/discards stale fetches and fences attachment/world/rules identity.
- Existing completed T11 Duck history is included, per coordinator clarification. `Backfill` runs before recovery/workers/serving, reads100 IDs per batch, ordered FinishedAt/resultID. It does not re-insert Duck detail, modify immutable history or wipe existing boards. Validated complete Duck detail is required; generic pre-Duck `finished` history without detail and incomplete results only receive skip receipts. Projection receipts make new finalization/import/restart idempotent. Historical champion rows reflect deterministic import order; changedAt records projection/audit time and linked result records the actual achievement time, not a fabricated original commit order.

## Startup / future deletion runbook notes

Backfill shares the existing10-second startup context with RecoverPending. Every result gets a separate atomic transaction; earlier committed receipts remain on cancellation/failure, the current incomplete transaction rolls back, startup fails closed without serving or starting gameplay workers, and a later startup resumes remaining unreceipted results. There is no unbounded single transaction or restart wipe. For a large import, operators may need repeated startups to complete within the existing deadline; no runtime was restarted and no live migration/backfill was run by this implementation agent. Invalid stored historical payloads fail startup visibly rather than manufacturing a score.

The T14 removal call must precede account identity removal in the same transaction, retain the exclusion tombstone to fence pending jobs, and follow commit with revocation/invalidation. Do not turn the administrative audit into a champion victory. Full account-erasure/history anonymization is still T14's work.

## Files changed by implementation agent

Backend:
- `apps/server/internal/db/migrations/009_leaderboards.sql`
- `apps/server/internal/leaderboard/service.go`, `champion.go`, `query.go`, `http.go`, `backfill.go`
- `apps/server/internal/leaderboard/champion_test.go`, `champion_integration_test.go`
- `apps/server/internal/application/application.go`, `leaderboard_integration_test.go`
- `apps/server/internal/realtime/protocol.go`, `leaderboard_protocol_test.go`

Web/contracts:
- `apps/web/src/features/leaderboard/ChampionBoard.tsx`, `controller.ts`, `leaderboard.css`
- `apps/web/src/app.js`, `data.js`, `net/socket.ts`, `world/avatars.ts`
- `apps/web/index.html`, `scripts/test-map.mjs`, `tsconfig.json`
- `apps/web/tests/leaderboard.test.ts`, `leaderboard.e2e.spec.ts`
- `contracts/event.schema.json`

This report is the only implementation-owned process document. Root's concurrent PROGRESS/checkpoint/evidence changes are not mine. No staging, commit, reset, push, deployment, provisioning, purchases, source-repository edits, process restart or live preview credentials/accounts were performed. Tools used existing Go/Node; no application backend was added in Node.

## Behavioral coverage and self-review

- Real isolated PostgreSQL schemas cover first/empty board, unique3 users, same owner improvement, non-improving PB, exact comparison, private reads, removed membership, bans/archive, ruleset separation, excluded actor types, transaction rollback, explicit board-lock blocking, concurrent distinct finalizations and same-result retries.
- Account lock tests cover removal-first/projection-first contention and rolled-back transactions; separate test commits removal before late finalization into an unseen partition. Replay remains excluded; incomplete/valid history persists.
- Backfill tests simulate completed pre009 state (history + detail but no projection), exclude incomplete/deleted/missing-detail rows, fail a receipt insert to prove rollback, restart idempotency, FinishedAt order despite reverse completion insertion, cancellation, and actual application startup twice.
- Actual httptest HTTP server + WebSocket transport + PostgreSQL worker recovery: durable500-point result appears once for owner/member, no widened private result status to spectator, no event/API result leak to other world; summary allowed only in world, private history denied to spectator, replay adds no second victory, load works after missed delivery and kick revokes reads. These are isolated synthetic fixtures, not authenticated browser evidence.
- Four new Node tests exercise the actual controller/view:3 empty slots with no fake winners, world-summary links, rational display formatting, committed-only invalidation, duplicate/world/rules fencing, reload/reconnect fetch, late-request disposal after logout/world replacement and public payload validation. The new E2E source is typechecked, not executed.
- Self-reviewed lock ordering, retry/rollback receipt semantics, read authorization, result summaries vs private history, empty/archived UI, finite bounded ratios, stale async fetch behavior and T14 seam. No open implementation defect found in this review. Root's guest CUA found low contrast; corrected explicit parchment/text/button colors and truthful availability text. Root's corrected guest-only evidence is in `docs/checkpoints/evidence/CP6/t12-guest-preview.md` and `t12-guest-podium-fixed.jpg`.
- Browser web is the deliverable including responsive mobile web. CSS switches podium from3 columns to1 below540px. No native application or physical-device performance evidence is claimed.

## RED / failed attempts / GREEN chronology

Commands ran from repository root unless noted, using `GOCACHE="$PWD/.cache/go-build"` and `/opt/homebrew/bin/go -C apps/server`.

1. Unit API-first RED: `GOCACHE="$PWD/.cache/go-build" /opt/homebrew/bin/go -C apps/server test ./internal/leaderboard` exited1 with `undefined: Entry`, `undefined: better` (new implementation absent). This is a compile RED, not a behavioral failure claim.
2. Integration API-first RED: after privately sourcing `.cache/database.env`, same command with `-tags=integration` exited1 with missing `Service`, `New`, `ProjectResult`, `ErrNotFound`, `lockBoard`, `RemoveAccount`.
3. First implemented focused run used `source .cache/database.env` without export and failed every DB test with `DATABASE_URL required`. Corrected with `set -a; source .cache/database.env; set +a`; no env contents printed.
4. Exported sandbox focused run failed `dial tcp 127.0.0.1:55432: connect: operation not permitted`. Retried with actual loopback escalation, approved; isolated schemas only.
5. Protocol behavioral RED: `GOCACHE="$PWD/.cache/go-build" /opt/homebrew/bin/go -C apps/server test ./internal/realtime -run TestLeaderboardInvalidationContract` exited1, `leaderboard_protocol_test.go:5: invalid event`. GREEN after adding typed public invalidation: same command exited0 (`ok .../internal/realtime 0.683s`).
6. Focused PG GREEN: `... go -C apps/server test -tags=integration ./internal/leaderboard` exited0 (`ok .../internal/leaderboard 2.368s`).
7. Frontend API-first RED: `node --input-type=module -e "import {build} from 'esbuild';await build({entryPoints:['apps/web/tests/leaderboard.test.ts'],outfile:'/tmp/t12-leaderboard-test.mjs',bundle:true,platform:'node',format:'esm'});"` exited1 because new controller/ChampionBoard modules were absent. Compile RED only.
8. Frontend focused GREEN: `node --input-type=module -e "import {build} from 'esbuild';await build({entryPoints:['apps/web/tests/leaderboard.test.ts'],outfile:'/tmp/t12-leaderboard-test.cjs',bundle:true,platform:'node',format:'cjs',loader:{'.css':'empty'}});"` then `node --test /tmp/t12-leaderboard-test.cjs` exited0;4 tests,4 pass,0 fail (156.23125ms).
9. Real transport focused GREEN: exported local env, `... go -C apps/server test -tags=integration ./internal/application -run TestLeaderboardHTTPCommittedWorldTransportAndRecovery -count=1 -v` exited0: test0.81s; package1.448s.
10. `npm run typecheck` initially passed but self-review found new files not included by the existing tsconfig whitelist; added explicit new source/test includes. Corrected check then failed2 real TS issues: unsafe ChampionChange→Entry cast and missing key prop declaration. Fixed by narrowing resultLink to Pick<Entry,'resultId'> and declaring the existing JSX key convention. Following `npm run typecheck` exited0. Final run also includes the authored E2E source.
11. Backfill API-first RED: `... go -C apps/server test -tags=integration ./internal/leaderboard -run TestHistoricalBackfillOrderedRestartAndRollback` exited1: `f.s.Backfill undefined`. After implementation/receipt integration, focused `... go -C apps/server test -tags=integration ./internal/leaderboard -count=1 -v` exited0, package2.814s. Tests passed: ProjectionUniquePBChampionAndPrivateReads, FinalizationRowLockAtomicRollbackAndConcurrentResults, ExcludedAccountsAndRulesetIsolation, AccountRemovalSerializesBothOrdersAndReplay/{true,false}, CommittedRemovalBeforeUnseenPartitionFinalization, BannedAndArchivedWorldReadBoundaries, HistoricalBackfillOrderedRestartAndRollback, RankingUsesExactAccuracyThenTimeAndStableResultID/all7 cases. Further startup/order/cancellation/missing-detail cases were added before the final full run below.
12. Some initial exploratory reads used nonexistent guessed filenames (`duck/projector.go`, `duck/persistence.go`, `duck/project.go`, `duck/result.go`, realtime publish/broadcast.go, model model/start.go, worlds/members.go, web history.test.js) and one zsh unmatched App* glob. Switched to `rg --files`/actual files; no implementation decision relied on those failed reads.

## Final verification commands and exact captured outputs

Environment setup for all DB commands (never printed):
```sh
set -a
source .cache/database.env
set +a
export GOCACHE="$PWD/.cache/go-build"
```

### Full default-parallel Go race/integration — FAILED, preserved

```sh
/opt/homebrew/bin/go -C apps/server test -race -tags=integration ./... > .cache/t12-go-race.log 2>&1
```

```text
?   	patra-world-social/server/cmd/e2edb	[no test files]
ok  	patra-world-social/server/cmd/server	(cached)
ok  	patra-world-social/server/internal/application	38.627s
ok  	patra-world-social/server/internal/auth	18.479s
--- FAIL: TestChatPersistenceModerationIsolationAndResync (2.54s)
    service_integration_test.go:84: rate exceeded accepted <nil>
FAIL
FAIL	patra-world-social/server/internal/chat	4.810s
ok  	patra-world-social/server/internal/config	(cached)
?   	patra-world-social/server/internal/contracts	[no test files]
?   	patra-world-social/server/internal/db	[no test files]
ok  	patra-world-social/server/internal/games/duck	5.390s
ok  	patra-world-social/server/internal/games/model	(cached)
ok  	patra-world-social/server/internal/games/penalty	(cached)
ok  	patra-world-social/server/internal/games/race	(cached)
ok  	patra-world-social/server/internal/history	35.437s
ok  	patra-world-social/server/internal/http	6.117s
ok  	patra-world-social/server/internal/leaderboard	22.427s
ok  	patra-world-social/server/internal/lobby	17.194s
ok  	patra-world-social/server/internal/moderation	7.811s
ok  	patra-world-social/server/internal/realtime	11.814s
ok  	patra-world-social/server/internal/social	7.183s
?   	patra-world-social/server/internal/testutil	[no test files]
ok  	patra-world-social/server/internal/world	(cached)
ok  	patra-world-social/server/internal/worlds	7.112s
FAIL
```

Failure investigation: no chat code/test was changed. Inspection found the existing test's sixth send assumes five prior messages all occur before a token refills; SQL uses `clock_timestamp()` and refills1 token/sec. Default-parallel test elapsed2.54s; a focused diagnostic below elapsed0.70s. That supports a wall-clock-sensitive test/contention hypothesis; per-send timing/token state was not instrumented, so the precise elapsed send interval is not claimed measured. Coordinator approved exactly one focused diagnostic and, if green, one serial full rerun. T16 should stabilize this harness assumption; no production limiter was weakened and no retries were hidden.

### Focused chat diagnostic — PASS

```sh
GOCACHE="$PWD/.cache/go-build" /opt/homebrew/bin/go -C apps/server test -race -tags=integration ./internal/chat -run TestChatPersistenceModerationIsolationAndResync -count=1 -v > .cache/t12-chat-diagnostic.log 2>&1
```

```text
=== RUN   TestChatPersistenceModerationIsolationAndResync
--- PASS: TestChatPersistenceModerationIsolationAndResync (0.70s)
PASS
ok  	patra-world-social/server/internal/chat	2.096s
```

### Full serial-package Go race/integration — PASS (exit0)

```sh
GOCACHE="$PWD/.cache/go-build" /opt/homebrew/bin/go -C apps/server test -race -tags=integration -p 1 -count=1 ./... > .cache/t12-go-race-serial.log 2>&1
```

```text
?   	patra-world-social/server/cmd/e2edb	[no test files]
ok  	patra-world-social/server/cmd/server	1.745s
ok  	patra-world-social/server/internal/application	34.256s
ok  	patra-world-social/server/internal/auth	7.317s
ok  	patra-world-social/server/internal/chat	1.764s
ok  	patra-world-social/server/internal/config	1.467s
?   	patra-world-social/server/internal/contracts	[no test files]
?   	patra-world-social/server/internal/db	[no test files]
ok  	patra-world-social/server/internal/games/duck	1.262s
ok  	patra-world-social/server/internal/games/model	1.467s
ok  	patra-world-social/server/internal/games/penalty	1.505s
ok  	patra-world-social/server/internal/games/race	2.072s
ok  	patra-world-social/server/internal/history	17.223s
ok  	patra-world-social/server/internal/http	1.768s
ok  	patra-world-social/server/internal/leaderboard	4.218s
ok  	patra-world-social/server/internal/lobby	3.103s
ok  	patra-world-social/server/internal/moderation	1.632s
ok  	patra-world-social/server/internal/realtime	9.647s
ok  	patra-world-social/server/internal/social	1.534s
?   	patra-world-social/server/internal/testutil	[no test files]
ok  	patra-world-social/server/internal/world	1.556s
ok  	patra-world-social/server/internal/worlds	2.484s
```

Serial package scheduling is explicit; this is not a claim that the failed default-parallel run passed. The coordinator approved this single rerun after the focused diagnostic passed.

### Focused unit, Go vet and build — PASS (exit0)

```sh
export GOCACHE="$PWD/.cache/go-build"
/opt/homebrew/bin/go -C apps/server test ./internal/leaderboard > .cache/t12-go-unit-vet-build.log 2>&1
/opt/homebrew/bin/go -C apps/server vet ./... >> .cache/t12-go-unit-vet-build.log 2>&1
/opt/homebrew/bin/go -C apps/server build ./... >> .cache/t12-go-unit-vet-build.log 2>&1
```

```text
ok  	patra-world-social/server/internal/leaderboard	1.644s
```

Vet/build emitted no additional output and exited0.

### Frontend lint/typecheck/tests/build — PASS (exit0)

```sh
set -e
npm run lint > .cache/t12-frontend.log 2>&1
npm run typecheck >> .cache/t12-frontend.log 2>&1
npm test >> .cache/t12-frontend.log 2>&1
npm run build >> .cache/t12-frontend.log 2>&1
```

```text

> patra-world-social@0.1.0 lint
> npm run lint --workspace apps/web


> @patra-world-social/web@0.2.0 lint
> node --check src/app.js && node --check src/world.js


> patra-world-social@0.1.0 typecheck
> npm run typecheck --workspace apps/web


> @patra-world-social/web@0.2.0 typecheck
> tsc --noEmit


> patra-world-social@0.1.0 test
> npm test --workspace apps/web


> @patra-world-social/web@0.2.0 test
> node --test tests/*.test.js && node scripts/test-map.mjs

✔ password form counts Unicode code points and keeps whitespace (1.050584ms)
✔ isometric projection round trips fractional coordinates (4.658959ms)
✔ ground navigation walks around an occupied tile without leaving the island (1.018833ms)
✔ station navigation stops at the closest reachable entrance (4.102167ms)
ℹ tests 4
ℹ suites 0
ℹ pass 4
ℹ fail 0
ℹ cancelled 0
ℹ skipped 0
ℹ todo 0
ℹ duration_ms 521.047417
✔ shared Duck spawn and collision fixtures (31.422292ms)
✔ local guest sixty seconds, cooldown, dead target and persistent counter (0.553667ms)
✔ online view aims through delivered geometry at200ms RTT and55ms after arrival (10.24675ms)
✔ actual online controller/view follows measured50/100/200+jitter and capped320ms profiles (71.926125ms)
✔ online presentation hides expired/dead/stale targets and resets calibration on epoch (48.712292ms)
✔ empty board has three empty ranks and no invented winners; summaries link only to world podium (164.395292ms)
✔ committed event invalidates once, wrong world/rules and temporary results cannot promote a score; reconnect reloads (1.192709ms)
✔ late board response cannot restore data after world switch/logout; retry replaces stale requests (0.55025ms)
✔ public invalidation protocol rejects expanded private payload (0.89825ms)
✔ 64 square map, reachable entrances and collision-free spawn (39.225584ms)
✔ map decoder fails closed (1.461209ms)
✔ motion preference does not change movement duration or position (2.780542ms)
✔ protocol validates envelope and penalty zones (0.506792ms)
✔ actual world controller preserves route when motion is toggled (1979.695916ms)
✔ shared wire and fixed-point golden fixtures (1.104041ms)
✔ public avatars and world objects share one depth order (7.168667ms)
✔ co-located avatar nameplates occupy separate lanes (15.027417ms)
✔ shared semantic invalid maps fail at load boundary (3.807958ms)
✔ shared events validate required envelope fields and type set (0.815083ms)
✔ nonfinite zoom cannot corrupt camera (0.116083ms)
✔ online interpolation is bounded and full resync resets old epochs (0.872709ms)
✔ socket only sends when online and reconnect uses rotated seat without auto takeover (22.045708ms)
✔ shared snapshot fixtures reject forged versions, unknown fields, missing self and duplicate avatars (0.516459ms)
✔ canvas pointer release preserves destination; only a directional hold sends stop (0.191958ms)
✔ explicit socket leave waits for matching acknowledgement before replacement join (0.530625ms)
✔ world picker serializes the real controller switch behind leave acknowledgement (1.636ms)
✔ disconnected leave restores solo while pending release fences replacement (1.343958ms)
✔ disconnected logout restores solo while pending release fences replacement (0.329125ms)
✔ logout fences a queued world switch even when its old leave is later acknowledged (1.147208ms)
✔ chat resync merges exact durable IDs without duplicates or deletion resurrection (0.476416ms)
✔ chat send succeeds only after matching stored-message ack and failure preserves explicit error (1.826167ms)
✔ a delete received before its live message prevents later text resurrection (0.190417ms)
✔ actual chat form preserves an edited next draft while the submitted message awaits its ACK (2.946875ms)
✔ actual chat form retains failed draft and durable client ID for an explicit retry (0.414875ms)
✔ actual chat form ignores old ACKs after switching attachments while a new draft is pending (0.344333ms)
✔ production lobby view truthfully disables Start without a concrete engine (9.007792ms)
✔ lobby wire commands reject identity spoofing and socket dispatches resync events (0.735417ms)
✔ history does not invent a score or duration for an unknown-start aborted match (14.095125ms)
✔ actual socket dispatch forwards race and duck snapshots while fencing replaced sockets and worlds (0.512667ms)
✔ penalty local matches all 16 shared fixture outcomes (2.983625ms)
✔ penalty local locks first timely input and hides it until deadline (3.111292ms)
✔ penalty local equal sudden death pairs and maximum draw survive hidden tab catchup (2.499375ms)
✔ penalty local early winner and BOT blindness (0.276833ms)
✔ penalty guest session survives controller unmount and catches up on reopen (1.742875ms)
✔ penalty shared timelines and boundaries match Go fixture exactly (4.717708ms)
✔ penalty public view uses server time and locks chosen/spectator controls (134.609292ms)
✔ shared checkpoint gates reject reverse and shortcut laps (1.798333ms)
✔ guest countdown, stale input expiry and full bot race match server semantics (753.721292ms)
✔ bounded catchup never replays expired throttle (5.88025ms)
✔ fixed point acceleration steering and brake shared vectors (49.860083ms)
✔ prepared grid is stable and permutes identities (0.651584ms)
✔ guest session keeps race identity on reopen and never extends absolute deadline (13.218125ms)
✔ race controls are unavailable to spectators and saved result claims need server confirmation (98.702542ms)
✔ race rules cannot omit a physics coefficient (0.621041ms)
✔ finished race clock freezes at authoritative finish (10.267834ms)
✔ remounted guest controller accepts its first controls after a high prior sequence (10.624667ms)
ℹ tests 56
ℹ suites 0
ℹ pass 56
ℹ fail 0
ℹ cancelled 0
ℹ skipped 0
ℹ todo 0
ℹ duration_ms 2984.031667

> patra-world-social@0.1.0 build
> npm run build --workspace apps/web


> @patra-world-social/web@0.2.0 build
> vite build

vite v8.3.4 building client environment for production...
transforming...
✓ 68 modules transformed.
rendering chunks...
computing gzip size...
dist/index.html                   7.59 kB │ gzip:   2.64 kB
dist/assets/index-CVkBPoVp.css   62.87 kB │ gzip:  14.92 kB
dist/assets/index-fwgFij4K.js   357.07 kB │ gzip: 115.56 kB

✓ built in 865ms
```

`git diff --check` exited0 with no output. The full frontend test command passed4 JavaScript +56 bundled TypeScript behavioral tests. No browser CLI/e2e command was run.

## Unexecuted acceptance and concerns

- Authenticated browser/account/invite/deep-link/reconnect E2E is authored in `leaderboard.e2e.spec.ts` and typechecked, but explicitly not executed because user deferred credential workflows. Isolated HTTP/WebSocket tests are not a substitute claim of browser acceptance. Root's native CUA evidence covers guest gate/copy/contrast only.
- T14 full account deletion UI/API/session/history orchestration is not implemented by T12. The actual transactional leaderboard seam is exercised, including cleanup vs late finalization. No deletion-browser acceptance is claimed.
- Live API8080 remains accepted T11. Root owns migration/restart and integration-preview acceptance. No successful code test means CP6 PASS/taskdone.
- Default-parallel full Go race suite failed the existing wall-clock-sensitive chat limiter test; focused diagnostic passed, serial full result recorded separately. Carry the harness reliability issue into T16.
- No physical mobile browser/performance/network evidence was measured. Responsive browser web is implemented; no native deliverable.

## Freeze

Code and report frozen after self-review and final verification. All required implementation-owned files remain unstaged/uncommitted for coordinator independent review. No further passing suite reruns were performed after the final results.


## Review fix round1/5 — Important R1

Status: DONE_WITH_CONCERNS, R1 candidate frozen for independent rereview. Minor R2 (historical missing-detail representation) is deliberately unchanged per coordinator scope and remains a T16/finalreview obligation. The original implementation evidence above predates this fix; only affected Node tests/typecheck/build were rerun for R1. No taskdone/checkpoint PASS claim.

Reviewed `task-12-review.md` verbatim. Confirmed production had two independent request lifetimes: socket denial only cleared React state; pending callbacks could restore it, and a denied board refresh left loaded detail intact. No architecture expansion was needed. Coordinator approved a deterministic Node hook lifecycle harness using the real exported component/effects/listeners with controlled fetch promises and no controller mock.

### Fix and files

All fixes remain unstaged over the coordinator-preserved reviewed tree `adb420251e5f085ec2a6e763e8bd73b6b322645e`:
- `apps/web/src/features/leaderboard/ChampionBoard.tsx`: exported the actual component for the lifecycle test; consolidated board/detail requests under one effect with a shared access generation; denial disposes board controller, aborts detail request, increments generation, clears both states and settles loading. Successful or failing stale callbacks cannot update a later generation. Focus and invalidation events cannot restart denied requests. Explicit retry, replacement attachment, or a matching-world reconnect transition performs fresh HTTP reads. Repeated online snapshot statuses do not repeatedly recheck denied access. Socket denial checks both socket identity and displayed world, so a different-world deep link is unaffected. Dependency cleanup cancels both reads.
- `apps/web/src/features/leaderboard/controller.ts`: typed `ReadError` retains HTTP status;401/403/404 are access-denial responses. Controller error callback preserves the error object alongside its display message. Transient errors cannot change a denied/obsolete generation.
- `apps/web/tests/leaderboard.test.ts`:4 new production-component regressions (8 total tests in this focused file), using real request parsing/controller/component/effect listeners, not only controller.dispose tests.
- `apps/web/tests/component-lifecycle.ts`: test-only deterministic React hook harness, including state updates, refs, dependency comparison, effect cleanup/replacement and unmount. It inspects actual component output via ReactDOMServer. This is Node component/lifecycle evidence, not DOM/browser/device evidence.
- This report append only. Root documentation/checkpoint/evidence edits remain outside implementation ownership.

Coverage: pending board+detail responses after same-world socket denial stay suppressed through explicit retry; focus stays blocked; loaded detail clears on subsequent401 and404 board refresh; repeated online status cannot reopen a denied lifetime; a reconnect transition fetches both anew; detail404 aborts a pending board read; stale transient503 after a successful retry cannot erase or restore state; different-world socket denial preserves an authorized deep link; route dependency replacement aborts the old detail and clears it until the new response arrives.

Self-review checked cleanup ordering, generation capture, status scoping, HTTP error classification, retry/loading behavior, effect dependency replacement and absence of any backend/R2 edits. No process, browser, credentials, source repo, staging, commit or reset action was performed.

### Exact RED/GREEN commands and output

Initial fixture ran8 tests:5 passed,3 failed (both-read abort, denied-refresh detail clearing, and detail-denial fencing). Strengthened the different-world regression to deny after the detail had actually loaded; the unchanged production effects then failed all4 new tests. Only the component export was added before RED; no production behavior fix was present.

Command used to compile the focused test for both RED and GREEN (exit0):
```sh
node --input-type=module -e "import {build} from 'esbuild';await build({entryPoints:['apps/web/tests/leaderboard.test.ts'],outfile:'/tmp/t12-r1-test.cjs',bundle:true,platform:'node',format:'cjs',loader:{'.css':'empty'}});"
```

RED command, exit1:
```sh
node --test /tmp/t12-r1-test.cjs > .cache/t12-r1-red.log 2>&1
```

```text
✔ empty board has three empty ranks and no invented winners; summaries link only to world podium (28.294583ms)
✔ committed event invalidates once, wrong world/rules and temporary results cannot promote a score; reconnect reloads (0.464917ms)
✔ late board response cannot restore data after world switch/logout; retry replaces stale requests (0.188084ms)
✔ public invalidation protocol rejects expanded private payload (0.445083ms)
✖ production component cancels board and detail together on denial and fences late responses through explicit retry (1.468542ms)
✖ production loaded detail is removed when a board refresh returns401 or404 and only a valid online recheck restores it (23.283125ms)
✖ production detail denial fences pending board and ignores late transient failures after a fresh retry (4.301ms)
✖ production deep link ignores denial from another world and replaces detail request on route change (2.572625ms)
ℹ tests 8
ℹ suites 0
ℹ pass 4
ℹ fail 4
ℹ cancelled 0
ℹ skipped 0
ℹ todo 0
ℹ duration_ms 184.676917

✖ failing tests:

test at ../../../../private/tmp/t12-r1-test.cjs:66020:30
✖ production component cancels board and detail together on denial and fences late responses through explicit retry (1.468542ms)
  AssertionError [ERR_ASSERTION]: both private reads must abort on denial
      at TestContext.<anonymous> (/private/tmp/t12-r1-test.cjs:66026:27)
      at Test.runInAsyncScope (node:async_hooks:213:14)
      at Test.run (node:internal/test_runner/test:1106:25)
      at Test.processPendingSubtests (node:internal/test_runner/test:788:18)
      at Test.postRun (node:internal/test_runner/test:1235:19)
      at Test.run (node:internal/test_runner/test:1163:12)
      at async Test.processPendingSubtests (node:internal/test_runner/test:788:7) {
    generatedMessage: false,
    code: 'ERR_ASSERTION',
    actual: false,
    expected: true,
    operator: '==',
    diff: 'simple'
  }

test at ../../../../private/tmp/t12-r1-test.cjs:66045:30
✖ production loaded detail is removed when a board refresh returns401 or404 and only a valid online recheck restores it (23.283125ms)
  AssertionError [ERR_ASSERTION]: The input was expected to not match the regular expression /Private member|Ringkasan hasil dunia|Memuat podium/. Input:

  '<section class="champion-board"><h2>Podium Duck dunia privat</h2><p>Top 3 akun unik · satu personal best per akun. Urutan skor, akurasi tepat, waktu pencapaian, lalu ID hasil.</p><p role="alert">Masuk untuk melihat podium dunia privat.</p><button type="button">Muat ulang podium</button></section><section><h3>Ringkasan hasil dunia</h3><p>Private member · 300 poin · 1/2 hit · 50.0%</p><p>1/1/1970, 7:00:00 AM · Map m v1 · Aturan duck-v1 v1 · Protokol v1</p><p>Ringkasan ini tersedia untuk anggota aktif dunia. History akun tetap pribadi.</p></section>'

      at TestContext.<anonymous> (/private/tmp/t12-r1-test.cjs:66057:29)
      at async Test.run (node:internal/test_runner/test:1113:7)
      at async Test.processPendingSubtests (node:internal/test_runner/test:788:7) {
    generatedMessage: true,
    code: 'ERR_ASSERTION',
    actual: '<section class="champion-board"><h2>Podium Duck dunia privat</h2><p>Top 3 akun unik · satu personal best per akun. Urutan skor, akurasi tepat, waktu pencapaian, lalu ID hasil.</p><p role="alert">Masuk untuk melihat podium dunia privat.</p><button type="button">Muat ulang podium</button></section><section><h3>Ringkasan hasil dunia</h3><p>Private member · 300 poin · 1/2 hit · 50.0%</p><p>1/1/1970, 7:00:00 AM · Map m v1 · Aturan duck-v1 v1 · Protokol v1</p><p>Ringkasan ini tersedia untuk anggota aktif dunia. History akun tetap pribadi.</p></section>',
    expected: /Private member|Ringkasan hasil dunia|Memuat podium/,
    operator: 'doesNotMatch',
    diff: 'simple'
  }

test at ../../../../private/tmp/t12-r1-test.cjs:66071:30
✖ production detail denial fences pending board and ignores late transient failures after a fresh retry (4.301ms)
  AssertionError [ERR_ASSERTION]: The expression evaluated to a falsy value:

    ok(f.requests[0].signal.aborted)

      at TestContext.<anonymous> (/private/tmp/t12-r1-test.cjs:66076:27)
      at async Test.run (node:internal/test_runner/test:1113:7)
      at async Test.processPendingSubtests (node:internal/test_runner/test:788:7) {
    generatedMessage: true,
    code: 'ERR_ASSERTION',
    actual: false,
    expected: true,
    operator: '==',
    diff: 'simple'
  }

test at ../../../../private/tmp/t12-r1-test.cjs:66091:30
✖ production deep link ignores denial from another world and replaces detail request on route change (2.572625ms)
  AssertionError [ERR_ASSERTION]: unrelated socket denial cannot erase an authorized deep link
      at TestContext.<anonymous> (/private/tmp/t12-r1-test.cjs:66101:27)
      at async Test.run (node:internal/test_runner/test:1113:7)
      at async Test.processPendingSubtests (node:internal/test_runner/test:788:7) {
    generatedMessage: false,
    code: 'ERR_ASSERTION',
    actual: '<section class="champion-board"><h2>Podium Duck dunia privat</h2><p>Top 3 akun unik · satu personal best per akun. Urutan skor, akurasi tepat, waktu pencapaian, lalu ID hasil.</p><p role="alert"></p><button type="button">Muat ulang podium</button></section>',
    expected: /Ringkasan hasil dunia/,
    operator: 'match',
    diff: 'simple'
  }
```

After the shared-lifetime fix, rebuilt with the same compile command. GREEN command, exit0:
```sh
node --test /tmp/t12-r1-test.cjs > .cache/t12-r1-green.log 2>&1
```

```text
✔ empty board has three empty ranks and no invented winners; summaries link only to world podium (25.873583ms)
✔ committed event invalidates once, wrong world/rules and temporary results cannot promote a score; reconnect reloads (0.327291ms)
✔ late board response cannot restore data after world switch/logout; retry replaces stale requests (0.237333ms)
✔ public invalidation protocol rejects expanded private payload (0.385792ms)
✔ production component cancels board and detail together on denial and fences late responses through explicit retry (30.013ms)
✔ production loaded detail is removed when a board refresh returns401 or404 and only a valid online recheck restores it (6.992916ms)
✔ production detail denial fences pending board and ignores late transient failures after a fresh retry (3.622667ms)
✔ production deep link ignores denial from another world and replaces detail request on route change (5.134333ms)
ℹ tests 8
ℹ suites 0
ℹ pass 8
ℹ fail 0
ℹ cancelled 0
ℹ skipped 0
ℹ todo 0
ℹ duration_ms 284.495334
```

Affected typecheck/build commands, exit0:
```sh
set -e
npm run typecheck > .cache/t12-r1-checks.log 2>&1
npm run build >> .cache/t12-r1-checks.log 2>&1
```

```text

> patra-world-social@0.1.0 typecheck
> npm run typecheck --workspace apps/web


> @patra-world-social/web@0.2.0 typecheck
> tsc --noEmit


> patra-world-social@0.1.0 build
> npm run build --workspace apps/web


> @patra-world-social/web@0.2.0 build
> vite build

vite v8.3.4 building client environment for production...
transforming...
✓ 68 modules transformed.
rendering chunks...
computing gzip size...
dist/index.html                   7.59 kB │ gzip:   2.64 kB
dist/assets/index-CVkBPoVp.css   62.87 kB │ gzip:  14.92 kB
dist/assets/index-BZFF9rvk.js   357.61 kB │ gzip: 115.84 kB

✓ built in 262ms
```

`git diff --check` exited0 with no output. No unrelated full Go/frontend suite or browser workflow was rerun. The Node tests/typecheck/build do not remove the existing authenticated-browser acceptance deferral or physical-mobile evidence limits.

Round1 code and report are frozen for coordinator independent rereview; all fixes remain unstaged.
