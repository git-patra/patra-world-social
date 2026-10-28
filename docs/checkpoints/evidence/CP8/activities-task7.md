# Task 7 — Pingpong

Status: implementation, unit verification and actual WebSocket/PostgreSQL integration complete. BOT and two-human transport/history subtests passed after root repaired the local PostgreSQL forwarding path. Root browser evidence: activities-pingpong-browser.md; independent review recorded separately. Worker did not commit/push, restart live processes, use shell browser automation, create subagents, or edit the source `patra-world` repo.

## Delivered behavior and selected constants

- One authoritative two-player table in the existing match pipeline. Two connected ready humans, or one ready human who explicitly selects BOT. No automatic human-to-BOT replacement, account creation, public-world admission, or new leaderboard.
- Fixed 50 ms / 20 Hz physics, at most four catch-up steps. Long stalls skip excess simulation debt while retaining the absolute 120 s deadline and disconnect grace. One ball and two paddles; no framework, entity pool, private loop, or new runtime dependency.
- Normalized table: x/y in 0..1, ball radius .02, paddle centers at x=.06/.94, paddle half-height .12; paddle center y clamps to .12..88. Human paddle speed 1 unit/s, BOT speed .65, BOT deadzone .04. Serve starts at center with horizontal speed .6 and vertical speed ±.3, alternating the vertical sign by rally. Paddle bounce vertical speed is three times contact offset, clamped to ±.45. Top/bottom walls reflect. A missed paddle scores when the center passes x=0/1.
- First to 7 with a two-point margin, hard cap 11. Equal scores at the 120 s deadline produce a draw; otherwise the greater score wins. The loser serves the next rally. Human serve is explicit and fenced by the current `rallyId`; BOT serves automatically when it owns serve.
- Clients send direction `axis`, never positions, hit results, or scores. Increasing admitted input sequence plus the fixed-step movement budget prevents flooding from moving faster. Held intent expires after 250 ms. Disconnect neutralizes immediately; the deadline continues; reconnect before 15 s preserves the human, otherwise the authoritative result is forfeit. Result copies cannot mutate the engine's completed result.
- The adapter retains at most one replaceable public snapshot and emits at most one regular snapshot every four physics frames (5 Hz for both participants and spectators); final results emit immediately. The existing world's one RAF interpolates the visible table/ball/paddles. This is a conservative broadcast choice; the proposed participant 10–15 Hz tuning and measured traffic targets remain release-performance work.
- Guest session uses the equivalent TS model, owned by `ActivityStore` and its existing 50 ms clock. Menus/HUD subscriptions do not own simulation or completion. Receipt writes use `Completion`, retry at most once/s after a failure, and deduplicate by receipt ID. No extra interval/RAF was added.
- A/D and left/right arrows, keyboard-focused buttons, and independent touch pointers move the paddle; a keyboard-accessible Serve button starts a rally. Modal/text/blur/hidden/disconnect cleanup reuses the shared input router. Registered participants alone get controls. Passive nearby spectators retain walking and see a score/timer without buttons.
- New Map destination and lobby menu: Meja Pingpong at safe return `(45,25)`. Seven-district visitation remains unchanged. The existing decorative table is reused; decorative paddle/ball were removed so the live scene supplies the only moving objects. Explicit exit/return restores the contract safe return.
- Paddle/score/finish cues use the shared AudioService. Initial reads/resync, duplicate/older frames and cues delayed >1 s do not play; distance attenuation and existing bounded audio voices apply.

## Files and API seams

Backend:

- `contracts/pingpong-rules.v1.json` and mirrored `apps/server/internal/games/pingpong/rules.v1.json`: versioned rules identity `pingpong-v1`, version 1; SHA-256 of the exact bytes is frozen into the durable start.
- `contracts/pingpong.golden.json`: shared serve/movement-budget, miss/scoring/next-serve, and paddle-bounce traces.
- `games/pingpong/{engine,bot,adapter}.go`: fixed-step authority, snapshots, immutable results, presence/deadline handling, inert factory and bounded event adapter.
- `games/model/result.go`: optional typed `PingpongResult {scores:[2]int64,rallies,reason}`. Omission for old game results preserves their original serialized fields/order.
- `lobby/{registry,service,start,presence,game}.go`: append pingpong after the unchanged legacy arena ordering; safe return from the activity contract; registered-engine admission, versioned registration, explicit BOT consent, area lease/clear/persist/activate, result return and mutual rematch. New games cannot join without a factory. The upcoming blaster roster is explicitly exempted from human-slot BOT filling.
- `application/lobby.go`, `realtime/{protocol,connection,delivery}.go`: register the engine and admitted gameplay dispatch. Continuous `pingpong.input` uses the existing movement budget; serve remains a discrete control command. Existing membership/epoch/admission/area/durability fences remain in force.
- `db/migrations/013_activity_match_types.sql`: additive allowlist widening to pingpong + blaster; migration 004 is unchanged. No leaderboard migration/projector.
- `history/{service,query}.go`: pingpong/blaster query allowlists; pingpong win/loss/draw/forfeit outcomes; typed pingpong metadata must match both participant scores, score/rally bounds and known termination reasons. Immutable result/history storage remains the existing transaction pipeline.

Wire protocol remains 2:

- `pingpong.input {matchId,axis}`; finite axis in [-1,1], exact keys.
- `pingpong.serve {matchId,rallyId}`; integer rally ID 1..22, exact keys.
- `pingpong.snapshot {matchId}` request; world-scoped public read.
- `pingpong.snapshot` event carries match ID, phase, monotonic frame, server/absolute start/deadline times, rally/server index, roster, paddles, scores, ball, and optional authoritative result.

Frontend/contracts:

- `features/pingpong/{local,session,PingpongHUD,PingpongMenu}`: local model, app-owned session, world HUD and lazy lobby/menu.
- `world/scenes/pingpong.ts`: visible table renderer with public interpolation, no authority; `world/{controller,village-art}.ts` and `data.js/app.js` provide nearby/map entry and remove decorative duplicates.
- `activities/{store,scene,ActivityHUD}`: persistent attachment/resync/game reads, independent latest-frame fence, input ownership, serve, passive spectators, scene loading and fresh-only shared audio. Existing race equal-frame semantics and penalty ordered chosen flags remain unchanged.
- `world/avatars.ts`, `net/socket.ts`, `contracts/{protocol.schema,event.schema,commands.golden}.json`: exact command decoding and event routing. The six-part casual resync (four masks, vehicles, football) is unchanged; pingpong uses its own match snapshot read after lobby resync.
- `guest/{completion,history}.ts`, `history/{HistoryPage,MatchDetail}`: runtime append allowlist, correct `pingpong-v1` receipt identity, account filter and typed result/local detail display.

New test entrypoints: `apps/web/tests/pingpong.test.ts`, Go pingpong engine/golden tests, `lobby/pingpong_test.go`, `history/pingpong_test.go`, `application/pingpong_integration_test.go`. Existing `application/areas_test.go` now runs its real owner clear/durability/pending-result test for pingpong too. Root registered the frontend test and production `features/pingpong/**/*.ts(x)` globs; existing world-scene globs already covered the scene.

## RED evidence and corrections

1. Initial focused Go engine test failed before implementation: undefined `Engine`, `New`, and `won`. The same focused command passes after the authoritative implementation.
2. Real frontend local-control regression failed: `pingpong guest engine survives view subscriptions and controls release with independent pointers`, expected paddle `.45`, got `.5`. The shared store originally deferred changed guest paddle intent until its next tick. Sending the changed local intent immediately at the current store time fixes the delay while retaining fixed-step movement and independent pointer release.
3. Full Go suite caught a compatibility regression because pingpong was initially prepended to the registry. Existing `TestLobbyOwnershipSlotsAndUnavailableEngine`, `TestStartFailureRestoresLobbyAndCanRetry`, and `TestCoalescedDisconnectResumeReachesEngineAndMarksAssistance` failed (the latter indexed the expected race player on the new empty pingpong row). Appending pingpong after the original registry entries preserves legacy ordering; the entire lobby and then entire Go suite pass.
4. Initial TypeScript check rejected an inferred UUID-template parameter for a supplied string ID. Declaring the session ID parameter `string` fixed that type mismatch.
5. Build reported an ineffective dynamic import when menu and HUD shared a module. Splitting `PingpongMenu.tsx` makes the menu lazy and removes the warning.
6. Actual integration initially reported `DATABASE_URL required` because the first compilation/execution command lacked the private environment. Two subsequent approved runs sourced/exported `.cache/database.env` quietly and reached the configured database, but both failed at test setup with `127.0.0.1:55432: connection reset by peer`. No SQL assertion or WebSocket completion claim is made from those runs.

## Verification commands and observed outcomes

All Go commands use the pinned executable from repository root:

```sh
GOTOOLCHAIN=local GOCACHE=$PWD/.cache/go-build .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server ...
```

- `test ./internal/games/pingpong -count=1`: PASS. Shared Go/TS golden trace assertions, finite direction/participant/serve authority, flood budget, top/bottom reflection, input expiry, natural target/cap, deadline, grace boundary/resume, preserved human roster and immutable result copies.
- `test ./internal/history ./internal/lobby -run '^TestPingpong' -count=1`: PASS. Unregistered game admission denial, explicit BOT vs two-ready-human start, BOT account absence, metadata/score binding and filter validity.
- `test ./internal/games/pingpong ./internal/lobby ./internal/history ./internal/application -run 'TestSharedGolden|TestWall|TestServe|TestWin|TestStartWaits' -count=1`: PASS. Includes actual world/lobby owners proving clear ACK and confirmed durability before start, and OPEN/walking restoration while the durable result claim remains pending.
- Final `test ./... -count=1`: PASS all packages, with approved loopback execution for existing HTTP/WebSocket tests. Earlier registry failures are recorded above and fixed.
- `test -race ./internal/games/pingpong ./internal/lobby ./internal/history ./internal/application ./internal/realtime -run 'TestPingpong|TestServeBudget|TestWinBy|TestWallBounce|TestNaturalScore|TestSharedGolden|TestStartWaits|TestCommand|TestProtocol' -count=1`: PASS all five packages.
- `test -tags=integration ./internal/application -run '^$'`: PASS compilation of the integration suite.
- `npm test`: PASS **4 JavaScript tests + 180 bundled tests**, zero failures/skips. Pingpong tests include shared goldens, actor/serve/score authority, fixed-step flood/catchup, grace/deadline, strict inputs, persistent guest lifecycle, independent pointers/modal neutralization, storage retry identity/dedup, latest-frame results, passive walking and rendered spectator HUD, and fresh-only shared audio.
- `npm run typecheck`: PASS.
- `npm run lint`: PASS.
- `npm run build`: PASS. Latest measured main chunk before final receipt pacing/readability-only rebuild: `index-CDO68QAd.js`, **435.77 kB / 137.71 kB gzip**. New local pingpong model is **4.90 kB / 2.21 kB gzip**. These are Vite build sizes, not device FPS/heap/network measurements.
- `git diff --check`: PASS.

Actual DB/WS integration command (environment never printed):

```sh
set -a
source .cache/database.env
set +a
GOTOOLCHAIN=local GOCACHE=$PWD/.cache/go-build .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server test -tags=integration ./internal/application -run '^TestPingpongActual' -count=1
```

Initial outcome: blocked at DB setup as described above; superseded by the successful real DB/WS run recorded at the end of this report. `TestPingpongActualTransportHistoryAndSpectator` contains BOT and two-human subtests with actual authenticated WebSockets, explicit BOT rejection/consent, serve and paddle ACK, spectator input/serve rejection, real deadline completion, private filtered history, no spectator participant row, immutable finalization retry and mutual rematch. Commands are paced within the existing ten-control-command/s transport budget, and all three queues are consumed while virtual time advances. Root subsequently restored the expected local PostgreSQL service and forwarding path; the final run below completes this execution evidence.

## Browser handoff

Production build is available for root's existing local preview. Path: Map → **Meja Pingpong** → walk to `(45,25)` → E / Explore Meja Pingpong → **Mulai practice vs BOT**. Closing the menu reveals the small world HUD and real table. Press **Serve** (also keyboard focus + Enter), use A/D or left/right arrows / independent touch buttons. Menu/detail or E can reopen the panel; the game continues. Explicit **Akhiri practice / kembali berjalan** returns to `(45,25)` without an unfinished result row. Completed play should appear under History → perangkat ini with `Pingpong` and score.

Online: join a private world, open the same menu, choose two Ready humans or check BOT, then Start. The normal three-second area clear and durable-start gate precede the table. Passive spectator strip is `x=42,y=21..24`; it retains walking and shows no controls. Actual authenticated browser multiplayer remains the previously deferred acceptance gate; the automated integration test uses real test fixtures/transport and does not claim browser acceptance.

## Self-review and limits

- Reviewed all new input authorization routes: strict exact-key wire validation, admitted epoch/time, lobby participant checks, increasing engine input sequence, expiry, rally serve fence, no score/position/hit input, no automatic bot replacement.
- Reviewed fixed catch-up and absolute deadlines, result snapshot deep copies, delayed result storage claims, generation-safe area reservation and mutual rematch using the existing owners. Generic uncertain START reconciliation stays fenced.
- Reviewed independent snapshot ordering, six-part casual resync unchanged, spectator walking/no HUD controls/no history authority, two-pointer cleanup, persistent session ownership and no popup clock.
- Reviewed old registry order, version/checksum omission behavior for prior games, map schema 1 / asset `kampung-senja-v2` / protocol 2, and unchanged Duck/podium/race/penalty gameplay. No existing migration was rewritten.
- No live process restart, deployment, fake browser session, LiveKit acceptance, physical-device performance claim, traffic benchmark or after-GC heap claim was made. DB execution is now verified; root browser/review evidence remains root-owned.
- Upcoming blaster can use migration013, versioned registration and area gating; it must add its own engine, arena, typed result/detail semantics and outcomes. Human roster BOT filling is already disabled for `blaster`.

## Database retry after root restored Colima/PostgreSQL

Root reported restoring the dedicated `patra-world-social` Colima profile and starting the existing PostgreSQL container, preserving its data. Worker retried the exact integration command above without changing application code or restarting anything.

- First retry produced no test output and remained blocked for approximately 102 s. Worker interrupted only that test process. Go reported `signal: interrupt` after 101.887 s.
- A bounded diagnostic then ran the same pinned/environment-sourced command with `-run '^TestPingpongActual/BOT$' -count=1 -v -timeout=45s`. It entered the BOT subtest and timed out after 45 s.
- The timeout stack identifies the blocker precisely: `testutil.Database` at `database.go:34`, acquiring its first connection for `CREATE SCHEMA`, with `pgconn.connectOne` waiting in `receiveMessage`. This is the PostgreSQL connection handshake, before migrations, application construction, WebSocket admission or pingpong gameplay. A TCP connection is accepted on the host, but the database handshake does not complete.
- Root was informed that inside-container readiness alone does not establish the host `127.0.0.1:55432` forwarding path. No test assertion, transport budget or game rule was weakened. Actual DB/WS execution remains open pending host connectivity recovery.

The final post-cleanup frontend run remains **4 JavaScript + 180 bundled tests PASS**. Latest clean production build is `index-rALOmB3q.js`, **435.77 kB / 137.71 kB gzip**; pingpong local model `local-gBkYWJkb.js` **4.90 kB / 2.21 kB gzip**. This supersedes the earlier pre-cleanup chunk hash with the same measured sizes.

## Final real DB/WebSocket evidence — PASS

Root identified an orphaned dedicated SSH mux holding port 55432, replaced that profile's stale forwarding connection, and verified a PostgreSQL protocol response from the host loopback port. Worker then reran the unchanged integration test, with bounded connection and overall test timeouts:

```sh
set -a
source .cache/database.env
set +a
PGCONNECT_TIMEOUT=5 GOTOOLCHAIN=local GOCACHE=$PWD/.cache/go-build .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server test -tags=integration ./internal/application -run '^TestPingpongActual' -count=1 -v -timeout=90s
```

Observed output (exit 0):

```text
--- PASS: TestPingpongActualTransportHistoryAndSpectator (9.03s)
    --- PASS: TestPingpongActualTransportHistoryAndSpectator/BOT (4.66s)
    --- PASS: TestPingpongActualTransportHistoryAndSpectator/two_humans (4.37s)
PASS
ok  patra-world-social/server/internal/application 9.664s
```

This closes the database blocker. Both cases execute real authenticated WebSocket commands against the actual application and isolated PostgreSQL schema. They verify explicit BOT authorization/two ready humans; valid serve and paddle ACKs; spectator paddle/serve denial; rejection of early result return; the real 120 s deadline through the engine owner; durable `pingpong-v1` results; `game=pingpong` history filtering; exactly one human participant row for BOT or two for human play, with no spectator row/history access; idempotent finalization preserving the result hash; and a fresh unready rematch lobby (mutual consent for the two-human case).

No product or test source changes were required after root dispatched independent review. Only this report changed during the final database diagnostics/rerun. The separately recorded owner test continues to cover uncertain/pending storage and spatial release before durable claim release. Browser evidence, including mobile acceptance, remains root-owned; this automation does not replace it.
