# T11 Duck Arcade implementation report

Status: implementation complete and frozen for independent coordinator review. No staging, commit, process restart, remote deployment, source-repo change, account/invite browser workflow, or subagent delegation performed. Base accepted T10 HEAD was 530535d670e45abb78a8e0a3470d5677ca6d7f7e. Coordinator owns docs/PROGRESS, CP6, screenshots, package and commit; concurrent changes to those files are not implementer edits.

## Implemented

- Four existing cabinet lobby entries now activate the authoritative Duck engine. Inert factory stays behind durable start persistence. Registry countdown is exactly3000ms, actual startsAt is gameplay boundary, full session lasts60000ms.
- Duck-v1 immutable scoring/spawn asset: normal100/fast200/gold500,700ms spawn interval,2500ms lifetime,250ms receipt-time cooldown, max8 configured targets,150ms max rewind. With these intervals actual concurrency is at most4 alive targets. Seeded xorshift spawn, server hitboxes, one frontmost target per shot, global dead-ID check across history, server score/hits/shots. Current public targets omit birth schedule/velocity seed/future pattern.
- 30Hz fixed simulation and every-second-frame15Hz publication on the existing50ms owner; bounded8-step catch-up and8 historical snapshots. Absolute wall deadlines remain authoritative after delayed ingestion. Skipped simulation frames never grant extra session time. Historical hit validation reads the retained preceding target-state snapshot and evaluates its analytic velocity at the server-selected bounded display instant; no older available history means zero rewind.
- Metadata-only disconnect installation. Natural finish during grace, including equal natural/grace deadlines, yields complete. Earlier grace expiry and intentional leave produce immutable incomplete participant history through T8, not generic all-gone abandonment. Frozen roster remains even when user claim releases. Duck never gets bot assistance. Results auto-release cabinet after10s or explicit saved-result return; T12 leaderboard/champion omitted.
- Real user+world+match admission and connection-epoch fencing for shoot and timing controls; spectator snapshot bound to world. Strict payloads disallow score/rewind spoofing. Shot coordinates/client clock never upload trusted scores.
- Guest TS implementation, shared JSON spawn/collision fixtures, persistent module-level guest session across modal close/reopen, pointer/touch and keyboard crosshair/Space, zero-shot accuracy, countdown, honest local/saved/incomplete labels, autumn pumpkin pond art.
- Narrow confirmed prior-game correction: WorldSocket previously dropped race.snapshot/race.finished. Explicit race and Duck event forwarding now passes actual onmessage regression, including replaced-socket and wrong-world filtering. Existing penalty branch retained.
- Availability copy updated in hero, directory and station; podium/champion explicitly pending. Countdown now says waiting rather than spectator.

## Coordinator-approved architectural choices

1. RegisterVersioned(game,factory,RulesMetadata) accepts only Duck and validated nonemptyID/positiveversion/32-byte checksum. It overrides only RulesID/RulesVersion/RulesChecksum before factory and persistence. Registration refused after a lobby exists or duplicate factory. Prepare remains roster-only. Canonical Duck JSON raw bytes have a SHA256 checksum and are byte-parity checked against guest asset. Frozen attempts reuse metadata. Penalty/race assets and all shared MatchResult serialized field shapes remain unchanged; no historical result hash mutation.
2. Each Duck Adapter owns one actor/world/match timing estimator. Admitted epoch change resets timing and sequence. Disconnect/departure/end clears timing. One outstanding unpredictable24-byte random nonce; replacement invalidates prior nonce; successful consumption single-use. Challenge RTT measured from authoritative owner challenge creation to authoritative reply ingestion; client never supplies RTT. Max challenge age1000ms; last4 samples;15s calibration expiry; conservative minimum recent RTT. Offset is server midpoint minus client monotonic observation. Server chooses receiveTime-min(RTT/2,150ms), client timestamp only checks monotonicity/finite range/consistency (50ms tolerance around expected mapped instant) and strictly rejects mapped future time. No valid sample accepts otherwise legal shots at current server time with zero rewind, without calibration stall.
3. RTT includes response queues, client processing and reply queues; asymmetric routes and jitter cannot be perfectly inferred. UI labels calibration/high latency/150ms bound. This is bounded lag compensation, not an anti-aimbot or perfect anti-cheat claim. Synthetic fixture timings are NOT physical-network acceptance.
4. Generic lobby BotAssisted presentation flag restricted to race, approved by coordinator. O1 coalesced resume/new-disconnect and personally finished race assistance remain T16; no generic presence contract rewrite.

## Files

New server: internal/games/duck/{engine.go,targets.go,rewind.go,adapter.go,rules.v1.json,engine_test.go,rewind_test.go,lobby_test.go,fixture_test.go}; internal/application/duck_integration_test.go; internal/realtime/duck_protocol_test.go; internal/db/migrations/008_duck.sql.

Modified server: application/{application.go,lobby.go,lobby_integration_test.go}; lobby/{registry.go,service.go,start.go,presence.go,service_test.go}; history/{service.go,lifecycle_integration_test.go}; realtime/{delivery.go,protocol.go}.

New web: src/features/duck/{local.ts,session.ts,Duck.tsx,duck.css}; tests/{duck.test.ts,duck.guest.e2e.spec.ts}.

Modified web: src/features/lobby/LobbyPanel.tsx; src/features/history/MatchDetail.tsx; src/net/socket.ts; src/world/avatars.ts; scripts/test-map.mjs; tests/map.test.ts; playwright.guest.config.ts; src/{app.js,data.js}; index.html. Guest Playwright config now preserves all*.guest.e2e.spec.ts (including existing penalty/race), offers optional already-installed browser executable environment override, writes ignored .cache output. No browser download.

Contracts: new duck-rules.v1.json and duck.golden.json; extended protocol.schema.json and event.schema.json. Existing rules.v1.json, rules checksum, shared result model unchanged.

## RED -> GREEN and self-review

All Go commands use /opt/homebrew/bin/go and GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build. Go cwd apps/server; web/root commands noted below.

- Initial engine/rewind/lobby test scaffolding ran before implementation and failed compilation on absent Engine/New/clockEstimator/RegisterVersioned/Factory/Adapter. This is missing-interface scaffolding evidence, not a behavioral assertion failure; no claim otherwise. One scaffold typo duration-int64 was fixed before engine implementation.
- `go test ./internal/realtime -run TestDuck`: RED `duck.snapshot {"matchId":"m"}: invalid command`; strict protocol implementation GREEN.
- `go test ./internal/lobby`: RED TestRegistryStartDelayBelongsOnlyToRace/duck: official103000 expected100000. Required countdown corrected expectation/name to race+duck.
- `node scripts/test-map.mjs` in apps/web: RED actual socket dispatch expected four race/Duck events, actual[]. Explicit forwarding + envelope-world check GREEN. One initial invocation from root failed harness entrypoint resolution; corrected cwd, not treated as product failure.
- New TS guest/fixture test entry failed absent local/session module resolution, then passed after implementation. Persistent guest session tests retain shot state across access/remount boundary; no component-local sequence resets.
- Integration initial scheduler observation through full Application competed with its running publisher and saw14 instead of15. Corrected the test observer to wireLobbiesClock with no competing drain, keeping actual application lifecycle and WS/PG checks separately. Actual production adapter is unchanged by that test correction; result30frames/15events passes.
- Coordinator review identified terminal event risk. Added actual held-owner resume-after-natural-boundary scenario. RED `terminal event missing` with stale duck.snapshot, phaseplaying63000, instead of duck.finished. Adapter PresenceAt now detects internal Engine.Presence Tick transition and queues exactly one duck.finished. Owner test asserts one final event and no repeated event across every natural/equal/expired/leave/resume case. GREEN.
- Self-review tightened mapped future handling. RED TestClockFutureStaleAndUncalibratedNeverChooseHistory accepted mapped receive+1ms under prior20ms tolerance. Changed to strict mappedTime>serverNow rejection; stale/no-calibration remains zero rewind. GREEN.
- One TS typecheck failure from project React declaration rejecting key on DuckGame component was fixed by a keyed div wrapper. Pointer coordinate mapping uses square SVG bounds to avoid letterbox coordinate mismatch.
- First full server integration run found two obsolete assertions: TestLobbyProtocolUsesAdmittedIdentityAndPublicResync still expected Duck unavailable; it now expects available and tests unready rejection before ready (leaves lobby lifecycle intact). TestPostgresLifecycleUsesActualRuntimeStartAndKeepsResultScreen advanced only1s; it now derives target as actual m.StartsAt+1000, preserving missing-Schedule fallback purpose rather than hardcoding delay. Both focused regressions PASS, final full integration PASS.

## Executed automated verification

`npm run lint` root: PASS (node --check src/app.js && node --check src/world.js).
`npm run typecheck` root: PASS after final copy/E2E source changes.
`npm test` root: PASS53 total =4 JS +49 TS. Previous map, penalty and race suites preserved; added2 Duck tests +1 actual socket dispatch test. Full exact output follows.

```text

> patra-world-social@0.1.0 test
> npm test --workspace apps/web


> @patra-world-social/web@0.2.0 test
> node --test tests/*.test.js && node scripts/test-map.mjs

✔ password form counts Unicode code points and keeps whitespace (0.799375ms)
✔ isometric projection round trips fractional coordinates (1.82925ms)
✔ ground navigation walks around an occupied tile without leaving the island (0.388167ms)
✔ station navigation stops at the closest reachable entrance (1.396917ms)
ℹ tests 4
ℹ suites 0
ℹ pass 4
ℹ fail 0
ℹ cancelled 0
ℹ skipped 0
ℹ todo 0
ℹ duration_ms 75.294458
✔ shared Duck spawn and collision fixtures (2.362083ms)
✔ local guest sixty seconds, cooldown, dead target and persistent counter (0.880375ms)
✔ 64 square map, reachable entrances and collision-free spawn (11.934958ms)
✔ map decoder fails closed (0.648875ms)
✔ motion preference does not change movement duration or position (0.229625ms)
✔ protocol validates envelope and penalty zones (0.360208ms)
✔ actual world controller preserves route when motion is toggled (678.173625ms)
✔ shared wire and fixed-point golden fixtures (0.595666ms)
✔ public avatars and world objects share one depth order (5.702458ms)
✔ co-located avatar nameplates occupy separate lanes (10.759667ms)
✔ shared semantic invalid maps fail at load boundary (2.685666ms)
✔ shared events validate required envelope fields and type set (0.795209ms)
✔ nonfinite zoom cannot corrupt camera (0.07825ms)
✔ online interpolation is bounded and full resync resets old epochs (0.6765ms)
✔ socket only sends when online and reconnect uses rotated seat without auto takeover (13.64125ms)
✔ shared snapshot fixtures reject forged versions, unknown fields, missing self and duplicate avatars (0.592375ms)
✔ canvas pointer release preserves destination; only a directional hold sends stop (0.184ms)
✔ explicit socket leave waits for matching acknowledgement before replacement join (0.385417ms)
✔ world picker serializes the real controller switch behind leave acknowledgement (1.83725ms)
✔ disconnected leave restores solo while pending release fences replacement (1.708334ms)
✔ disconnected logout restores solo while pending release fences replacement (0.323875ms)
✔ logout fences a queued world switch even when its old leave is later acknowledged (0.727083ms)
✔ chat resync merges exact durable IDs without duplicates or deletion resurrection (0.309125ms)
✔ chat send succeeds only after matching stored-message ack and failure preserves explicit error (0.758958ms)
✔ a delete received before its live message prevents later text resurrection (0.063333ms)
✔ actual chat form preserves an edited next draft while the submitted message awaits its ACK (1.468458ms)
✔ actual chat form retains failed draft and durable client ID for an explicit retry (0.394ms)
✔ actual chat form ignores old ACKs after switching attachments while a new draft is pending (0.188541ms)
✔ production lobby view truthfully disables Start without a concrete engine (6.953458ms)
✔ lobby wire commands reject identity spoofing and socket dispatches resync events (0.97075ms)
✔ history does not invent a score or duration for an unknown-start aborted match (11.379625ms)
✔ actual socket dispatch forwards race and duck snapshots while fencing replaced sockets and worlds (0.378667ms)
✔ penalty local matches all 16 shared fixture outcomes (2.366541ms)
✔ penalty local locks first timely input and hides it until deadline (0.669041ms)
✔ penalty local equal sudden death pairs and maximum draw survive hidden tab catchup (0.506959ms)
✔ penalty local early winner and BOT blindness (0.500083ms)
✔ penalty guest session survives controller unmount and catches up on reopen (0.56875ms)
✔ penalty shared timelines and boundaries match Go fixture exactly (2.749208ms)
✔ penalty public view uses server time and locks chosen/spectator controls (17.279209ms)
✔ shared checkpoint gates reject reverse and shortcut laps (0.952125ms)
✔ guest countdown, stale input expiry and full bot race match server semantics (52.691625ms)
✔ bounded catchup never replays expired throttle (0.175834ms)
✔ fixed point acceleration steering and brake shared vectors (0.735083ms)
✔ prepared grid is stable and permutes identities (0.160834ms)
✔ guest session keeps race identity on reopen and never extends absolute deadline (0.617041ms)
✔ race controls are unavailable to spectators and saved result claims need server confirmation (14.556667ms)
✔ race rules cannot omit a physics coefficient (0.289458ms)
✔ finished race clock freezes at authoritative finish (3.14225ms)
✔ remounted guest controller accepts its first controls after a high prior sequence (9.355958ms)
ℹ tests 49
ℹ suites 0
ℹ pass 49
ℹ fail 0
ℹ cancelled 0
ℹ skipped 0
ℹ todo 0
ℹ duration_ms 923.464916
```

`npm run build` root after final copy: PASS. Exact output tail:

```text

> patra-world-social@0.1.0 build
> npm run build --workspace apps/web


> @patra-world-social/web@0.2.0 build
> vite build

vite v8.3.4 building client environment for production...
transforming...
✓ 64 modules transformed.
rendering chunks...
computing gzip size...
dist/index.html                   7.35 kB │ gzip:   2.60 kB
dist/assets/index-cViB0vD0.css   61.99 kB │ gzip:  14.74 kB
dist/assets/index-vhDSauhz.js   346.55 kB │ gzip: 112.68 kB

✓ built in 256ms
```

`GOCACHE=... /opt/homebrew/bin/go test ./...` initially under sandbox: failed existing realtime TestTransportAdmissionMovementAndRevocation because httptest loopback bind was denied, not a test assertion. Other packages passed. Escalated integration suite below includes realtime and its unit tests, which PASS.

First `set -a; source ../../.cache/database.env; set +a; GOCACHE=... /opt/homebrew/bin/go test -tags=integration ./internal/application -run Duck -v` under sandbox: local PostgreSQL127.0.0.1:55432 connect operation not permitted. Retried with explicit escalation and temporary schemas/ephemeral loopback servers; no account browser workflow and no secrets printed.

Affected transport command `go test -tags=integration ./internal/application -run TestDuckActualTransport -v` PASS for50/100/200/320ms, each with additional+12ms jitter challenge, measured rewind25/50/100/150ms. Each uses real WS command routing, genuine authority checks, server-selected historical target hit, then actual durable history/projector. Random private match seeds mean target types/points can differ across runs/profiles; consistency assertion is server shotResult score == final participant score and projected hits/shots, not equality between unrelated seeds. Example executed output:

```text
synthetic measured RTT=50ms (+12ms jitter sample) rewind=25ms authoritative hit=3 score=100
synthetic measured RTT=100ms (+12ms jitter sample) rewind=50ms authoritative hit=3 score=100
synthetic measured RTT=200ms (+12ms jitter sample) rewind=100ms authoritative hit=3 score=100
synthetic measured RTT=320ms (+12ms jitter sample) rewind=150ms authoritative hit=3 score=200
--- PASS: TestDuckActualTransportMeasuredProfilesAndAuthority (1.45s)
```

Synthetic timing tests also cover jitter0/+8/-4/+12, nonce replay/expiry, mapped future/nonfinite/replayed timestamps, stale/new-epoch zero rewind,150-bound historical hit,151ms direct engine rejection. World/actor/epoch inputs pass through existing admitted/session fences, spectator shoot and private history are rejected. Public snapshot assertion excludes seed.

Final full suite (only repeat because preceding full suite failed the two stale expectations and subsequent terminal/future corrections required final verification):

```
cd apps/server
set -a
source ../../.cache/database.env
set +a
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -tags=integration ./...
```

Escalated local integration, exit0. Exact output:

```text
?   	patra-world-social/server/cmd/e2edb	[no test files]
ok  	patra-world-social/server/cmd/server	(cached)
ok  	patra-world-social/server/internal/application	23.475s
ok  	patra-world-social/server/internal/auth	(cached)
ok  	patra-world-social/server/internal/chat	1.292s
ok  	patra-world-social/server/internal/config	(cached)
?   	patra-world-social/server/internal/contracts	[no test files]
?   	patra-world-social/server/internal/db	[no test files]
ok  	patra-world-social/server/internal/games/duck	0.450s
ok  	patra-world-social/server/internal/games/model	(cached)
ok  	patra-world-social/server/internal/games/penalty	(cached)
ok  	patra-world-social/server/internal/games/race	(cached)
ok  	patra-world-social/server/internal/history	15.687s
ok  	patra-world-social/server/internal/http	(cached)
ok  	patra-world-social/server/internal/lobby	(cached)
ok  	patra-world-social/server/internal/moderation	2.492s
ok  	patra-world-social/server/internal/realtime	(cached)
ok  	patra-world-social/server/internal/social	(cached)
?   	patra-world-social/server/internal/testutil	[no test files]
ok  	patra-world-social/server/internal/world	(cached)
ok  	patra-world-social/server/internal/worlds	(cached)
```

Race detector `GOCACHE=... /opt/homebrew/bin/go test -race ./internal/games/duck ./internal/lobby`, exit0:

```text
ok  	patra-world-social/server/internal/games/duck	2.633s
ok  	patra-world-social/server/internal/lobby	3.141s
```

`git diff --check`: PASS (no output).

## Browser evidence and limits — do not conflate

Coordinator independently executed CUA guest browser smoke on existing Vite4176. Coordinator-reported PASS: 3-second countdown; target hit100; miss/keyboard shots =>100,1/3,33.3%; modal continuity; second session hit100 then close/reopen retains1/1, FIRST new pointer shot hits fast200 =>300,2/2; keyboard third shot; natural finish; postfinish click cannot change score/shots; return/reset. Console warnings/errors empty. Coordinator owns CP6 screenshot/evidence files. Separate final presentation CUA verified hero/station/podium-pending and countdown 'Bersiap...' copy. This report does not claim independent implementer CUA or physical mobile/online proof.

CLI guest Playwright was attempted before coordinator directed no further browser retries:
1. `npx playwright test --config playwright.guest.config.ts` failed before browser launch: pinned chromium_headless_shell-1248 executable absent.
2. Optional GUEST_BROWSER_EXECUTABLE pointed to existing cached1243, no download. Sandbox attempt failed launch MachPortRendezvous bootstrap permission.
3. Explicit escalated guest-only command `GUEST_BROWSER_EXECUTABLE=/Users/patra/Library/Caches/ms-playwright/chromium_headless_shell-1243/chrome-headless-shell-mac-arm64/chrome-headless-shell npx playwright test --config playwright.guest.config.ts duck.guest.e2e.spec.ts` executed. It FAILED after actual countdown, hits/remount/keyboard and natural60s completion, at final frozen-metrics test assertion. Expected from innerText was `Skor300\nHit2/3\nAkurasi66.7%`; matcher compared concatenated DOM `Skor300Hit2/3Akurasi66.7%`. This is an assertion-whitespace defect, not a passing E2E result. Source corrected to capture textContent (same matcher semantics); corrected source NOT_RERUN per coordinator. The pending attempt had already ended when coordinator stop arrived; no process kill or additional browser run followed. Authenticated browser/account/invite workflows remain deferred.

Thus automated browser E2E status is FAILED attempted run / corrected final source NOT_RERUN, distinct from coordinator's actual CUA PASS. Headless warning NO_COLOR/FORCE_COLOR was environment-only. Cached browser version is not pinned toolchain browser acceptance.

## Remaining acceptance limits / handoff

- T12 owns leaderboard unique-account top3, PB, champion projection/UI. Duck results expose rules partition,complete outcome, score,hits,shots; no invented podium/accounts.
- The HTTP API8080 process remains acceptedT10; no live API restart or online browser Duck acceptance claimed. Isolated newApplication/httptest integration ran T11 source.
- Synthetic RTT/offset/jitter observations are deterministic fixture evidence, not real-network50/100/200ms-jitter profile acceptance. Mobile physical device, measured rendering/frame performance and authenticated online CUA remain pending.
- Public snapshots reveal current positions only; lag validation does not prevent aimbots. RTT includes processing/queues; high/asymmetric latency may reject plausible shots or limit compensation, stated explicitly.
- Known generic O1 coalesced resume/new-disconnect and personally-finished race assistance remain T16. No claim they were solved here.
- Actor history immutable result shape unchanged. Incomplete sessions finalize as durable matches with participant outcome incomplete and duck_results.complete=false; UI labels ineligible. Matches table status completed means finalization saved, not leaderboard eligibility.
- Full-page guest persistence remains T14. Guest modal lifetime is covered.
- No production deploy/push/remote operation. All implementer files/report frozen for review; root-owned docs/evidence edits intentionally excluded from implementation ownership.

## Final global verification supplement

Coordinator requested completing previously omitted whole-Go race/vet/build coverage after reading the report. No source edits or browser runs performed for this supplement.

Cwd `apps/server`, full integration race command (escalated isolated temporary PostgreSQL schemas and ephemeral loopback servers):

```sh
set -a
source ../../.cache/database.env
set +a
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -race -tags=integration ./...
```

PASS, exit0. Exact output:

```text
?   	patra-world-social/server/cmd/e2edb	[no test files]
ok  	patra-world-social/server/cmd/server	(cached)
ok  	patra-world-social/server/internal/application	25.896s
ok  	patra-world-social/server/internal/auth	8.036s
ok  	patra-world-social/server/internal/chat	4.667s
ok  	patra-world-social/server/internal/config	(cached)
?   	patra-world-social/server/internal/contracts	[no test files]
?   	patra-world-social/server/internal/db	[no test files]
ok  	patra-world-social/server/internal/games/duck	3.644s
ok  	patra-world-social/server/internal/games/model	(cached)
ok  	patra-world-social/server/internal/games/penalty	1.581s
ok  	patra-world-social/server/internal/games/race	3.091s
ok  	patra-world-social/server/internal/history	21.130s
ok  	patra-world-social/server/internal/http	5.421s
ok  	patra-world-social/server/internal/lobby	5.421s
ok  	patra-world-social/server/internal/moderation	3.628s
ok  	patra-world-social/server/internal/realtime	11.615s
ok  	patra-world-social/server/internal/social	3.180s
?   	patra-world-social/server/internal/testutil	[no test files]
ok  	patra-world-social/server/internal/world	(cached)
ok  	patra-world-social/server/internal/worlds	3.738s
```

`GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go vet ./...`: PASS, exit0, no stdout/stderr.

`GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go build -o ../../.cache/t11-server-check ./cmd/server`: PASS, exit0, no stdout/stderr. Binary is in ignored `.cache/t11-server-check`; it was not launched and existing API process was not restarted.

Report refrozen after this verification supplement; source freeze remains unchanged.

## Review fix round1/5 — Important R1 online presentation alignment

Source/report refrozen for independent review after this supplement. Fixes are unstaged; no index/staging/commit/reset operation performed. Reviewed staged tree56f0a2d714413f2dc4c76b4e16c56c3c80fe6f4c was not changed by implementer. Root owns concurrent PROGRESS/CP6 edits. No browser invocation, account workflow, process restart, subagent or remote operation in this round. ReviewerMinor R2 shot-resolution event pipeline intentionally remains separate/T16 per coordinator; this fix does not claim R2 implemented.

### Reproduced defect and approved model

Read completed task-11-review.md and applied receiving-code-review skill. Added actual DuckView SVG-geometry/pointer-handler regression for snapshotP,200msRTT, deliveryP+100, center click55ms later. Before implementation the view returned x=.4 rather than .4465 for a fast target: deterministic behavioral RED, not just missing-symbol compilation. This confirms the old raw15Hz coordinates could miss a legitimate center click.

Coordinator approved these narrow presentation fields/model before implementation:

- Current public target adds normalized `vx` (per second) and `expiresAt` (server ms). These describe the already-visible trajectory/lifetime only; no seed, future spawn schedule, target birth plan, or hidden target list is exposed. Scoring/rules/result hash shape unchanged.
- Private admitted-shooter `duck.clock_reply` ACK now includes `{measured,calibration:{offsetMs,rttMs,rewindMs,measuredAt,expiresAt}}`. A shared server selector supplies both this metadata and actual rewind. Offset+RTT always come from the same eligible conservative recent sample. Expiry is sampleAt+15000, rewind=min(150,floor(RTT/2)); epoch/presence/end clears still apply. No client-chosen rewind or trusted client score added.
- Calibrated display time = local monotonic time + selected offset + max(0, selected RTT/2 - selected rewind). Under the approved symmetric path estimate this matches receipt-minus-selected-rewind. RTT<=300 renders estimated current server time;320msRTT adds10ms presentation lead while authoritative rewind remains150.
- Actual DuckGame online rendering uses new shared OnlineDuckController and requestAnimationFrame. DuckView predicts only existing snapshot targets using that time and `vx`, bounded to500ms beyond snapshot.serverTimeMs. Expired targets, non-playing phases and out-of-horizon/negative-age snapshots are hidden, with truthful degraded/stale messages. Accepted authoritative shot ACK records dead IDs so delayed snapshots cannot visually revive them. Reset fences in-flight calibration on resumed world snapshot/epoch. Guest branch retains original LocalDuck engine/session and50ms update semantics; no guest prediction.
- Uncalibrated/expired frontend timing is explicitly approximate/degraded. It does not change server authority or force a calibration input stall. Server continues its existing independently measured selection or zero rewind when it has no eligible sample. Unknown asymmetric/large-jitter paths remain an estimate, not a perfect hit guarantee.

### Changed files in this round

Server: application/{lobby.go,duck_integration_test.go}; games/duck/{adapter.go,rewind.go,rewind_test.go,targets.go}.
Web: features/duck/{Duck.tsx,local.ts}; new features/duck/online.ts; tests/duck.test.ts; new tests/{duck-online-driver.ts,duck-online-bridge.ts}; new scripts/duck-online-fixture.mjs.
Contract: new contracts/duck-presentation.schema.json documents current-target fields, calibration fields, expiry, capped time model and500ms bound.

The optional fields added to shared TS PublicTarget do not change local target generation, hits, score, cooldown or guest session behavior. Whole MatchResult model and canonical duck-v1 scoring asset/checksum unchanged.

### Regression method — public data, actual production controller/view

Replaced the problematic online integration aim setup. The test now finds a newly visible fast target through the public snapshot endpoint; it never reads private engine coordinates or uses a snapshot specially selected at the server's desired collision time. It captures a real WebSocket snapshot atP and passes only that public snapshot plus real challenge/calibration ACK and deterministic delivery/click timing to a bounded Node test bridge.

The bridge imports the production OnlineDuckController and DuckView. Controller.calibrate consumes actual measured challenge/reply metadata, receive accepts the delivered public snapshot, frame computes presentation time, and the test traverses actual DuckView SVG target transforms. It invokes that SVG's real pointer-down callback, which calls production controller.shoot and captures the actual outgoing shot payload. Go submits that payload through the real WebSocket handler and verifies authoritative hit/miss and durable result. Node is tooling only; application simulation remains Go. Integration-tagged test has explicit existing Node/npm/esbuild prerequisite,15s outer process timeout and10s child timeout, cleans temporary output, and launches no browser. Pure Go engine tests do not require Node.

Profiles are50/100/200/320ms, each with0/+12ms downlink and uplink jitter, plus a second+12ms measured clock sample. Every exact center click occurs55ms after delivery. Exact200/0 case: rendered timeP+155, fast200-point hit, rewind100.320/0: renderedP+225, rewind remains150. Outside target callback y=.95 is outside every target lane; actual server shot yields0 points. Real first-shot ACK is fed back through the controller to assert dead target disappears and does not regain clickable geometry. Expired presentation produces no click. TS tests separately distinguish expiry within the500ms horizon from stale-snapshot hiding, reset/calibration expiry, countdown/final phase hiding, and no hidden-coordinate aim. Go historical shot regression separately rejects resurrection of dead/expired IDs at the150ms rewind boundary.

### RED / intermediate failures / GREEN

`node scripts/test-map.mjs` (apps/web), RED actual view regression:

```text
AssertionError [ERR_ASSERTION]: visible center must track server-selected evaluation time, not old snapshot
actual: false
expected: true
```

After renderer/controller implementation this regression passed. A preliminary wrong-cwd append command failed before editing the test (path apps/web/tests from inside apps/web); corrected cwd immediately. It was a shell setup error, not product RED evidence.

`go test ./internal/games/duck -run TestPresentationCalibration` initially failed missing c.calibration method (new interface scaffolding). After common selector implementation, assertions verify minimum sample offset/RTT pair, server rewind equality, expiry and epoch reset.

First new integration run passed all center hits including exactR1 but failed320ms outside-hitbox fixture: the test waited300ms extra, so display age525/537ms correctly exceeded500ms bound and target was hidden. Corrected test timing to exact legal250ms receipt cooldown, keeping display age475/487ms; no product bound was relaxed. Subsequent transport and final race-instrumented affected integration runs PASS.

### Final affected checks (no whole-suite repeat)

Cwd apps/server; local DB env sourced privately and escalation used only for temporary schemas/ephemeral loopback integration:

```sh
set -a
source ../../.cache/database.env
set +a
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -race -tags=integration ./internal/application -run Duck -v
```

Exit0, exact output:

```text
=== RUN   TestDuckActualTransportMeasuredProfilesAndAuthority
=== RUN   TestDuckActualTransportMeasuredProfilesAndAuthority/RTT50/jitter0
    duck_integration_test.go:185: public-view RTT=50ms path jitter=0ms (+12ms clock sample) click55ms after delivery; renderAt-P=80.0ms rewind=25ms fast hit=1 score=200
=== RUN   TestDuckActualTransportMeasuredProfilesAndAuthority/RTT50/jitter12
    duck_integration_test.go:185: public-view RTT=50ms path jitter=12ms (+12ms clock sample) click55ms after delivery; renderAt-P=92.0ms rewind=25ms fast hit=4 score=200
=== RUN   TestDuckActualTransportMeasuredProfilesAndAuthority/RTT100/jitter0
    duck_integration_test.go:185: public-view RTT=100ms path jitter=0ms (+12ms clock sample) click55ms after delivery; renderAt-P=105.0ms rewind=50ms fast hit=4 score=200
=== RUN   TestDuckActualTransportMeasuredProfilesAndAuthority/RTT100/jitter12
    duck_integration_test.go:185: public-view RTT=100ms path jitter=12ms (+12ms clock sample) click55ms after delivery; renderAt-P=117.0ms rewind=50ms fast hit=1 score=200
=== RUN   TestDuckActualTransportMeasuredProfilesAndAuthority/RTT200/jitter0
    duck_integration_test.go:185: public-view RTT=200ms path jitter=0ms (+12ms clock sample) click55ms after delivery; renderAt-P=155.0ms rewind=100ms fast hit=2 score=200
=== RUN   TestDuckActualTransportMeasuredProfilesAndAuthority/RTT200/jitter12
    duck_integration_test.go:185: public-view RTT=200ms path jitter=12ms (+12ms clock sample) click55ms after delivery; renderAt-P=167.0ms rewind=100ms fast hit=3 score=200
=== RUN   TestDuckActualTransportMeasuredProfilesAndAuthority/RTT320/jitter0
    duck_integration_test.go:185: public-view RTT=320ms path jitter=0ms (+12ms clock sample) click55ms after delivery; renderAt-P=225.0ms rewind=150ms fast hit=1 score=200
=== RUN   TestDuckActualTransportMeasuredProfilesAndAuthority/RTT320/jitter12
    duck_integration_test.go:185: public-view RTT=320ms path jitter=12ms (+12ms clock sample) click55ms after delivery; renderAt-P=237.0ms rewind=150ms fast hit=4 score=200
--- PASS: TestDuckActualTransportMeasuredProfilesAndAuthority (7.46s)
    --- PASS: TestDuckActualTransportMeasuredProfilesAndAuthority/RTT50/jitter0 (1.64s)
    --- PASS: TestDuckActualTransportMeasuredProfilesAndAuthority/RTT50/jitter12 (0.91s)
    --- PASS: TestDuckActualTransportMeasuredProfilesAndAuthority/RTT100/jitter0 (0.90s)
    --- PASS: TestDuckActualTransportMeasuredProfilesAndAuthority/RTT100/jitter12 (0.92s)
    --- PASS: TestDuckActualTransportMeasuredProfilesAndAuthority/RTT200/jitter0 (0.91s)
    --- PASS: TestDuckActualTransportMeasuredProfilesAndAuthority/RTT200/jitter12 (0.64s)
    --- PASS: TestDuckActualTransportMeasuredProfilesAndAuthority/RTT320/jitter0 (0.90s)
    --- PASS: TestDuckActualTransportMeasuredProfilesAndAuthority/RTT320/jitter12 (0.64s)
=== RUN   TestDuckWiredSchedulerAndIncompleteHistory
=== RUN   TestDuckWiredSchedulerAndIncompleteHistory/grace
=== RUN   TestDuckWiredSchedulerAndIncompleteHistory/equal
=== RUN   TestDuckWiredSchedulerAndIncompleteHistory/expired
=== RUN   TestDuckWiredSchedulerAndIncompleteHistory/leave
--- PASS: TestDuckWiredSchedulerAndIncompleteHistory (1.62s)
    --- PASS: TestDuckWiredSchedulerAndIncompleteHistory/grace (0.40s)
    --- PASS: TestDuckWiredSchedulerAndIncompleteHistory/equal (0.40s)
    --- PASS: TestDuckWiredSchedulerAndIncompleteHistory/expired (0.40s)
    --- PASS: TestDuckWiredSchedulerAndIncompleteHistory/leave (0.41s)
=== RUN   TestDuckActualWiredScheduler30Hz15Hz
--- PASS: TestDuckActualWiredScheduler30Hz15Hz (0.12s)
PASS
ok  	patra-world-social/server/internal/application	10.790s
```

`GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -race ./internal/games/duck`: exit0:

```text
ok  	patra-world-social/server/internal/games/duck	1.605s
```

`GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go vet -tags=integration ./internal/application ./internal/games/duck`: exit0, no output.

Final targeted TS test command, cwd apps/web:

```sh
node --input-type=module <<'EOF'
import {build} from 'esbuild';
import {spawnSync} from 'node:child_process';
await build({entryPoints:['tests/duck.test.ts'],outfile:'../../.cache/t11-r1-duck-tests.mjs',bundle:true,platform:'node',format:'esm',loader:{'.css':'empty'},banner:{js:"import {createRequire} from 'node:module';const require=createRequire(import.meta.url);"}});
const r=spawnSync(process.execPath,['--test','../../.cache/t11-r1-duck-tests.mjs'],{stdio:'inherit'});process.exitCode=r.status??1;
EOF
```

Exit0,5 Duck tests PASS; exact output:

```text
✔ shared Duck spawn and collision fixtures (1.12975ms)
✔ local guest sixty seconds, cooldown, dead target and persistent counter (0.317375ms)
✔ online view aims through delivered geometry at200ms RTT and55ms after arrival (1.172958ms)
✔ actual online controller/view follows measured50/100/200+jitter and capped320ms profiles (5.203541ms)
✔ online presentation hides expired/dead/stale targets and resets calibration on epoch (1.62475ms)
ℹ tests 5
ℹ suites 0
ℹ pass 5
ℹ fail 0
ℹ cancelled 0
ℹ skipped 0
ℹ todo 0
ℹ duration_ms 104.642083
```

During implementation the existing TS harness also passed52 tests (previous49 +3 R1 regressions); prior JS4 not rerun in this affected-only round. Final targeted count5 is not a claim of a new full npm run.

`npm run typecheck` root: exit0. `npm run build` root: exit0, exact output:

```text

> patra-world-social@0.1.0 build
> npm run build --workspace apps/web


> @patra-world-social/web@0.2.0 build
> vite build

vite v8.3.4 building client environment for production...
transforming...
✓ 65 modules transformed.
rendering chunks...
computing gzip size...
dist/index.html                   7.35 kB │ gzip:   2.60 kB
dist/assets/index-BNgSyCzY.css   62.02 kB │ gzip:  14.75 kB
dist/assets/index-DHjAlG0T.js   348.86 kB │ gzip: 113.40 kB

✓ built in 197ms
```

`git diff --check`: no output/pass. No staged/index edits.

### Separate root browser evidence and unresolved limits

Coordinator reports focused amended-frontend guest CUA PASS: countdown3, gold500, modal close/reopen retained500/1of1; FIRST new pointer shot fast200 =>700/2of2; Right+Space =>700/2of3; natural finish700/2of3/66.7%; post-finish click unchanged; console empty. Root owns CP6/t11-fix1-guest-live.jpg/final.jpg and preview narrative. This proves shared-component guest preservation, not authenticated online or physical-network timing. No implementer browser run this round; prior CLI E2E failure/source-NOT_RERUN status unchanged.

R1 deterministic public-view timing is now covered by actual production controller/view/transport. Physical RTT/asymmetry/jitter/device/performance and authenticated browser acceptance remain deferred. R2 remains explicitly unimplemented per coordinator scope; ACK still carries current resolution. Source and report now frozen for round1 independent review.
