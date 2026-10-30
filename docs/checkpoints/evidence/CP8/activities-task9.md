# Task9 fishing implementation report — 30 October2026

Implementation complete, ready for independent review and root-owned browser/evidence/commit. No commits made. Source repo `/Users/patra/Project/patra-world` untouched. No DB profile/container/volume changes. Test DB schemas are disposable `social_test_*` via the existing testutil.

## Contract and behavior

- Retained the four existing visual anchors in mirrored `contracts/activities.v1.json` and server embedded activity asset: fishing-0(53,30), fishing-1(53,31), fishing-2(53,32), fishing-3(53,33).
- Added shared fishingRules: waitMinMs1500, waitMaxMs3500, biteMs2000, resultMs3000, proximity1.25tiles, catch pool `fish`, `boot`, `can`, `leaf`.
- Four individual public `kind:fishing` snapshots with areaId equal to spot ID and independent spot generation. Exactly one owner each, at most one spot per account. Runtime incarnation, account/session epoch, sequence, and per-spot generation fence all actions.
- Strict discrete intent union `activity.fishing` / `fishing.cast|fishing.reel|fishing.leave` carries incarnation, areaGeneration0, generation, spotId, action only. TS, Go, JSON Schema reject position, score, catch, axes and extra identifiers. Complete resync now requires11unique logical states:4area masks+vehicles+football+claw+4fishing.
- Cast clears held walking/path input, waits for an owner-selected bounded random deadline, exposes Bite!, then accepts Reel. Early reel yields the transient miss feedback; timely reel selects a cosmetic catch from the owner pool. Result ownership and catch clear at3seconds. Missing the2second bite window releases the spot. Long stalls jump directly to the deadline state with no event backlog or catchup loop.
- Movement directions/path intent, explicit Leave, disconnect/Idle, Remove/logout, takeover/Join and entering a real match release the server spot and increment generation. Local directional input, position changes, Leave, reset, attach/logout, match start and disposal clear local ownership. Disconnected client projection hides obsolete cast state immediately.
- No Completion, game history type, inventory, XP, leaderboard or persistence path was added for fishing. Server owner tick does no SQL/network work.
- Local guest simulation uses the existing ActivityStore50ms loop. The lazy scene renders four dock/bobber/rod/result actors on the world's existing RAF. SFX use existing shared AudioService cast/catch sounds and freshness/dedup guards; snapshots and resync do not replay sound. Labels/HUD show all phases even when muted.
- HUD supports touch Cast/Reel/Leave, Space/E and nearby contextual dock. E stays consumed during owned/result/pending phases and cannot fall through to the River menu. River place/map route now goes to(53,30) and copy explains Cast→Bite→Reel, movement cancellation and transient results. Fullscreen/autumn canvas, previous game behavior and district counting remain on their existing paths.

## Geometry ruling

The full Go suite exposed that preexisting anchor fishing-3(53,33) lies on an existing blocked bank tile. Root explicitly approved interpreting fishing positions as visual anchors and requiring reachable standing ground within the shared proximity, preserving approved map/anchors. Validation now checks a reachable nearby standing tile. Tests exercise all four unique selectable standing positions:

| Spot | Reachable standing tile |
|---|---|
| fishing-0 |53,30|
| fishing-1 |53,31|
| fishing-2 |53,32|
| fishing-3 |54,33|

Within overlapping proximity, nearby selection prefers the nearest available spot; if all are occupied it shows the closest occupied spot. No map checksum/art/collision asset was changed.

## TDD evidence

Read/applied superpowers:test-driven-development SKILL.md and writing-good-tests.md.

1. Wrote world behavior tests for four spots/exclusivity, river proximity, movement/leave and7cleanup paths, repeated bounded catches/expiry; wrote client contract/local model/seeded guest history tests. Initial Go RED failed all new tests `missing fishing spot fishing-0`; initial TS RED failed invalid activity snapshot / missing fishing state and action. Implemented owner/store/contract → GREEN.
2. Added socket completeness test with11states. RED `Invalid activity resync` at the valid complete input; updated completeness validation to require4unique fishing IDs → GREEN.
3. Added fishing snapshot area identity regression. RED `Missing expected exception` for fishing bound to areaId race; restricted to real fishing spot IDs → GREEN.
4. Added disconnect projection test. RED `'fishing' !== 'walk'`; suppressed obsolete fishing state immediately on non-online socket → GREEN.
5. Full Go exposed actual geometry validation regression in `TestActivityGeometryAndMirrors` and `TestVehicleSlotsAreReachableOnActualMap`. Corrected anchor/standing-tile interpretation as approved by root, retaining all4positions; final full Go GREEN.
6. Added nearest-free selectable spot regression. RED returned fishing-0 occupied instead of available fishing-1; prefer available spots then distance → GREEN. Four actual standing tiles remain uniquely selectable.
7. Self-review found result-phase E fallthrough. Added behavior regression; RED consumed interaction false during result; owning fishing interaction now always consumes E while Reel remains phase gated → GREEN.

Final focused client file has13tests. Additional characterization covers early reel miss, bite timeout/long stall, exact JSON Schema payloads/missing fences, duplicate/replayed sounds, retired incarnations and stale reads, muted HUD/scene4actors, match cleanup, result label expiry, existing guest IndexedDB record preservation across40cycles and resets.

## Verification commands and outcomes

Pinned Go for all commands:

```
GOTOOLCHAIN=local GOCACHE="$PWD/.cache/go-build" .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server ...
```

- Focused Go `test ./internal/world ./internal/realtime -run Fishing -count=1`: PASS.
- Full Go `test ./...`: final PASS all packages. First sandbox run failed existing localhost HTTP tests (`TestServeUntilSignalWaitsForHTTPDrain`, `TestTransportAdmissionMovementAndRevocation`, `TestSDKSerializesExplicitCutoff`) because bind was prohibited, then permitted localhost rerun exposed the two real geometry failures named above; final permitted run passes all. No failures omitted.
- `npm test`: PASS4JS +204bundled tests at the full-suite checkpoint. Subsequently added two targeted regressions (nearest free spot and consumed result E), each RED→GREEN; final focused fishing13PASS. Full-suite prior count204 includes original11fishing tests, so final runner contains206bundled tests; do not claim an unrun full206suite from this report.
- Final focused `esbuild apps/web/tests/fishing.test.ts --bundle --platform=node --format=esm --outfile=/tmp/patra-fishing-test.mjs` with existing createRequire banner, then `node --test /tmp/patra-fishing-test.mjs`:13PASS.
- `npm run lint`:PASS.
- `npm run typecheck`:PASS (including after final changes).
- `npm run build`: final PASS,110modules, lazy fishing chunk1.18kB /0.70kBgzip. Initially a static HUD label import prevented lazy chunking; labels moved to the existing local fishing module, final build warning-free. Main bundle445.86kB /139.38kBgzip, LiveKit separate398.26kB /102.58kBgzip.
- `git diff --check`:PASS before final small consumed-E change; no whitespace introduced by that one-line replacement.

## Actual DB and account history proof

Quietly exported existing ignored `.cache/database.env`. No credentials printed. Ran:

```
go -C apps/server test -tags=integration ./internal/application -run Fishing -count=1 -v
```

(with pinned environment above and permitted local network access).

Initial no-seed proof PASS40authenticated cycles across3m40.886s simulated: all6durable match/history/result/leaderboard tables count0, MD5 empty hash. Root requested stronger preservation proof, so final test first persists/enqueues/finalizes an actual completed Duck fixture through current PGStore/History service and Duck projector (valid current rules checksum, score300, hits1, shots2), releases fixture's active claim, then takes count/hash plus actual personal ListHistory JSON baseline.

The first seeded attempt failed setup with `invalid duck rules/roster` because an older fixture omitted current Duck rules identity; corrected fixture and final test PASS.

Final seeded proof:40authenticated Cast/Bite/Reel/expiry cycles over3m38.262086s simulated, matching spectator state, rejected duplicate reel, reload/disconnect/Idle, takeover rejoin, explicit logout/Leave+rejoin, and real pingpong lobby join cleanup. Existing account ListHistory JSON byte-for-byte unchanged. Before/after values:

| Table |Count|Identical before/after MD5|
|---|---:|---|
|matches|1|8ddf81cd3537e5d8164d3317f9ca8055|
|match_participants|1|f9d807f56dd162a6ff4d1d79771ad9d4|
|result_jobs|1|e20627264d1d572a074edb97346c2b58|
|active_match_users|0|d41d8cd98f00b204e9800998ecf8427e|
|leaderboard_entries|1|31751beffce545ba060b69a41b8504af|
|champion_changes|1|744de88e6632d3e1234c7bee555e1d9f|
|duck_results|1|5f0f426e2df64a0f5d4325a5da8b252e|

Hashes naturally vary per disposable fixture run because IDs/timestamps vary; equality within the run is asserted. Existing preview/account data never touched.

Guest proof uses actual fake-indexeddb-backed `guestHistory` transactions with a preexisting valid Duck record;40catch cycles over≥180seconds simulated, clearLocal, attach(null) preserve exact serialized list and games stays empty. A separate test verifies starting a match clears fishing ownership without completing fishing.

## Files

New:
- `apps/server/internal/world/fishing.go`, `fishing_test.go`
- `apps/server/internal/realtime/fishing_test.go`
- `apps/server/internal/application/fishing_integration_test.go`
- `apps/web/src/activities/fishing.ts`
- `apps/web/src/world/scenes/fishing.ts`
- `apps/web/tests/fishing.test.ts`

Extended existing owner/protocol paths:
- server world activities/activity/areas/runtime/movement/vehicles/claw and embedded activities asset;
- realtime activity_protocol/areas/hub/protocol;
- mirrored contracts activities and protocol schema;
- web activities contracts/store/scene/HUD, world controller/avatars, net/socket, app/data;
- existing football resync test only to include4fishing states in the expected complete snapshot.

Root alone edited `apps/web/scripts/test-map.mjs` and `apps/web/tsconfig.json` to register the test file. Root owns screenshot/evidence, browser, plan/ledger and commits.

## Remaining review / limits

Implementation tests and builds are complete. Root's independent review/browser gate remains. No authenticated-browser, physical mobile, LiveKit staging or Task11 performance/soak acceptance claim. No new timer/RAF/audio context, particle pool, fish simulation entities, dependency or persistence migration. Fishing actors are a bounded4actor visual pool. Actual WS endpoint transport testing was not added here; actual authenticated Hub+owner+SQL integration, exact wire validators and root CUA guest browser cover this task's implemented boundaries.

## Final app-shell fixture follow-up — root full-suite gate

Root's final `npm test` run exposed one intermittent existing shell-test timing boundary: `contextual_dock_keeps_action_text_separate_from_its_E_keycap` expected Kick but saw Explore Alun-alun. Full failure log `/tmp/patra-task9-final-web.log`. Applied systematic-debugging skill, reproduced the focused25test file (24pass/1fail), traced the actual singleton state and notification gate before changing anything.

Root cause was not absent fishing fixture fields: actual store state was inputModewalk, nearbyFootballtrue, position28,46, listenerCount1. Fixture clearLocal publishes at fractional `performance.now()`, then test advances precisely100ms; binary floating-point subtraction can produce less than100. Deterministic reproduction using a temporary compiled artifact only, overriding performance.now to100.003 while creating the fixture, yielded now200.003, lastNotify100.003, delta99.99999999999999. The store correctly deferred its throttled subscriber notification to its next tick, leaving the dock's previous Explore text at the assertion. Added diagnostic code only in `/tmp`, never in repository production/test code.

Root approved a fixture-only correction: both waits in this docking text test advance101ms to cross the subscriber gate. All real Kick/Turun/Naik Sepeda assertions remain; the separate `HUD_projection_is_limited_to_ten_notifications_per_second` test still tests the notification rate boundary. No production changes or unrelated refactoring.

- Fix-only diff: `/tmp/patra-task9-app-shell-fix.diff` (one file, comment + two100→101 replacements).
- Focused rebuilt production app-shell tests:25/25PASS.
- Deterministic100.003 baseline with the corrected101ms waits:1/1targetedPASS.
- Final `npm test` after this fix:4JS +206bundledPASS,0failed. Full log `/tmp/patra-task9-web-fixed.log`.
- `git diff --check`:PASS.

This final206test run supersedes the earlier204full-suite checkpoint and the initial root206run with its named intermittent fixture failure. No browser/build/runtime code changed in this follow-up; existing root browser/build evidence still targets the current production implementation.
