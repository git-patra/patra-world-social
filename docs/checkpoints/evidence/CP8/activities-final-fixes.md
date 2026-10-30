# Final whole-branch fix wave — R1–R4

Base `e5909fe`. Resumed the interrupted worker's existing four test-file changes and preserved its RED logs. No commits, browser mutations, service restarts, source `patra-world` access, or edits to root-owned final-review/image evidence. Production source is stable for root-owned final full frontend / Go race-integration / vet / build and scoped review.

## Changes

- R1: `connection.go` classifies only valid `activity.claw` / `claw.move` as continuous movement, under the existing 60/s aggregate movement quota. `ValidateCommand` still runs first. Enter/drop/leave remain under the 10/s control quota. Actual WS regression sends 26 directional/neutral moves over more than 1.3s after entering, requires ACKs, and separately verifies all three discrete actions close with `control rate exceeded` at the existing limit.
- R2: `store.ts` ordinary lobby updates read only missing matches, deduplicate pending requests, and wait at least 1000 monotonic milliseconds after a failed read before retrying that match on subsequent metadata. There is no retry timer or self-scheduling loop. Failure timestamps are pruned when lobby identities disappear and cleared on attachment reset; successful reads clear them. Explicit `reload()` forces every current match read, supersedes pending requests, and bypasses failure cooldown. Existing epoch/socket/resync/read-serial and match-presence guards still fence stale responses. Three unchanged four-match updates now send zero snapshot commands (previously 12); a new race identity sends exactly one; explicit four-match reload sends six commands (metadata + lobbies + four snapshots). Twenty immediate updates after a failure send no extra read; one retry is admitted after 1000ms.
- R3: `hub.go` removes lifecycle cache records by exact snapshot WorldID only after the runtime has passed the existing seat/active-lease checks and is actually closed/deleted. Test cycles 20 distinct worlds through destruction/recreation while a leased world keeps its runtime, 11 cached records and incarnation.
- R4: `app.js` permits E for claw and fishing as advertised, retaining visibility/modifier/text/modal/repeat guards. Tests verify claw exit and fishing reel, including that guarded fishing input preserves its waiting phase.

Changed production files: `apps/server/internal/realtime/{connection.go,hub.go}`, `apps/web/src/{activities/store.ts,app.js}`. Regression files: `apps/server/internal/realtime/{areas_test.go,connection_test.go}`, `apps/web/tests/{activities.test.ts,app-shell.test.ts}`. Existing Go test additions were gofmt-formatted.

## RED evidence

Original worker's exact invocations were not available after interruption; do not infer commands from logs. Preserved `final-fix-go-red.log` records the real WS failure at input 10 (`control rate exceeded`) and area cache 22 instead of 11. Preserved `final-fix-web-red.log` records four failing tests: 12 versus 0 metadata reads, 3 versus 1 pending reads, claw mode not exiting, and fishing phase remaining waiting.

Fresh retry RED used the bundle command below, then `node --test --test-name-pattern='lobby_missing_reads|explicit_resync' /private/tmp/patra-final-tests/activities.test.mjs` from `apps/web`, redirected to `final-fix-retry-red.log`: 3 tests, 2 pass, 1 fail. The retry regression saw 2 reads instead of 1 during the failure storm before the cooldown change. The pattern also matches the metadata/resync test. No earlier RED evidence is relabelled as freshly executed.

## Exact focused GREEN commands and results

From repository root, Go formatting:

```sh
.cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/gofmt -w apps/server/internal/realtime/connection.go apps/server/internal/realtime/hub.go apps/server/internal/realtime/connection_test.go apps/server/internal/realtime/areas_test.go
```

From `apps/web`, focused bundle and execution (output redirected to `../../.superpowers/sdd/2026-10-28-world-activities-adjustment/final-fix-web-green.log`):

```sh
node --input-type=module <<'JS'
import {build} from 'esbuild';
import {spawnSync} from 'node:child_process';
await build({entryPoints:['tests/activities.test.ts','tests/app-shell.test.ts','tests/claw.test.ts','tests/fishing.test.ts'],outdir:'/private/tmp/patra-final-tests',banner:{js:"import {createRequire} from 'node:module'; const require=createRequire(import.meta.url);"},bundle:true,platform:'node',format:'esm',outExtension:{'.js':'.mjs'},define:{'import.meta.env.DEV':'false'}});
const result=spawnSync(process.execPath,['--test',...['activities','app-shell','claw','fishing'].map(name=>`/private/tmp/patra-final-tests/${name}.test.mjs`)],{stdio:'inherit'});process.exitCode=result.status??1;
JS
npm run typecheck
```

PASS: 90 tests, 90 pass, zero failed/skipped/cancelled; 264.107708ms. Typecheck PASS, `tsc --noEmit`, exit 0 (`final-fix-typecheck.log`). The initial attempt imported esbuild from a non-existent workspace-local node_modules path; a subsequent root-cwd invocation caused app-shell fixture ENOENT errors because it reads `src/app.js` relative to cwd. Both were runner setup errors; the corrected bundle uses installed root resolution and runs from apps/web. No product changes were made to address those runner errors.

From repository root, with approved actual loopback listener access:

```sh
GOTOOLCHAIN=local GOCACHE="$PWD/.cache/go-build" .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server test -race ./internal/realtime -run 'TestAreaLifecycle|TestRecreatedRuntime|TestClaw|TestFishing|TestTransport|TestProtocol' -count=1 -v
```

PASS, exit 0, realtime 3.450s, no race diagnostics. Evidence: `final-fix-go-green.log`; sustained WS aiming subtest 1.43s and all discrete boundaries pass. Existing malformed claw wire validation is included by `TestClawWireAndPublicStateAreExact`.

From repository root, private ignored DB environment sourced without printing credentials and approved local DB/WS access:

```sh
set -a
source .cache/database.env
set +a
GOTOOLCHAIN=local GOCACHE="$PWD/.cache/go-build" .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server test -race -tags=integration ./internal/application -run 'TestClawActual|TestFishingActual|TestLobbyProtocol|TestStartWaits' -count=1 -v -timeout=90s
```

BLOCKED / exit 1, application 5.822s. `TestStartWaitsForClearAckAndDurabilityAcrossActualOwners` passed all race/penalty/pingpong subtests. The three DB-backed tests failed during connection setup to dedicated 127.0.0.1:55432: PostgreSQL `FATAL: could not open file "global/pg_filenode.map": Input/output error (SQLSTATE 58030)`. No DB assertion result is claimed. Evidence retained in `final-fix-application-green.log` (the intended output filename does not imply PASS). Root was notified and owns dedicated profile recovery and final actual integration execution; worker did not change services or retry this infrastructure failure.

`git diff --check`: PASS. No duplicate full frontend/Go suites or 15-minute browser load were run by this worker. These fixes do not modify renderer/simulation/wire payloads or the existing paced load scenario; repeated metadata read behavior is measured with the production ActivityStore regression. The previous measured server bandwidth and synthetic browser profiling limitations remain unchanged, not upgraded. Root owns final live runtime, full verification, scoped review and public publication. Existing physical mobile, authenticated browser, LiveKit, post-GC heap, peak bandwidth and T16 gaps remain disclosed; only CP1 remains official PASS.

## Controller final validation after recovery
The earlier PostgreSQL I/O failure was environmental and occurred before assertions. Root recovered only the existing dedicated VM/container and loopback forward, preserving volume. Full pinnedGo -count=1 -race -tags=integration -p1 ./... then passed22packages, including actualDB/WSapplication129.173s and realtime13.148s. Vet/buildpassed. Fullfrontend4JS+239TS/typecheck/lint/buildpassed. Scopedre-review R1–R4 ADDRESSED/no newbreakage; CUAguestEclaw/fishingPASS. Final report: activities-final-review.md.
