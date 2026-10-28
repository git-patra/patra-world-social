# T10 implementer report — five-lap race

Candidate is ready for coordinator staged review. No files were staged, reset, or committed by this agent. No preview process was restarted. No source-repository, remote, deployment, production account, purchase, credentialed browser, session injection, or CLI Playwright workflow was used. Root owns `docs/PROGRESS.md`, CP5 and its evidence; those changes visible in the shared checkout are not implementer edits.

## Implemented behavior

- Authoritative Go kart physics and matching TypeScript guest physics: integer subunits, integer direction vectors, acceleration, brake, steering, coasting drag, signed fractional x/y remainder carry, and outer/inner track walls with axis sliding. There is no kart-to-kart collision. Bots produce ordinary controls through the identical movement function and have no extra speed/collision capability.
- Single Autumn Ring track with three ordered directional checkpoints and a directional finish gate. Initial finish crossing, skipped checkpoints, duplicate checkpoints and reverse finish crossings do not add a lap. Completion requires exactly five laps. Race positions use lap/checkpoint/track progress; finish ties at simulation precision use the persisted grid slot, even when kart overshoot differs.
- Five total slots, one human plus four bots or up to five humans, simultaneous three-second countdown, and no countdown movement. The race factory is inert until the running match has been confirmed durable.
- Actual 30 Hz stepping and 15 Hz race publication from elapsed server time through the existing 50 ms generic lobby scheduler. Catch-up executes at most eight frames per owner invocation and counts discarded old motion opportunities in `droppedSteps`; no skipped interval fabricates distance/laps. Absolute race deadline remains 300 seconds, shortened to first finish plus 45 seconds. Hub physics and publication cadence remain unchanged.
- Inputs are controls only, scoped to an admitted participant and match/world, with integer/range validation, a monotonic sequence, and 250 ms expiry. Client position/lap/speed fields are rejected. Continuous race inputs use the existing bounded admitted-seat authorization lease and the 60/s movement rate classification. Resume resets the engine input sequence only behind the existing fenced transport lifecycle.
- Disconnect clears held input and immediately enables bot driving with persistent assistance flag. Reconnect inside 15 seconds restores that human seat. Expired or intentionally departed humans become DNF with frozen progress. All permanently departed humans abandon the match. Natural race limit at the same grace boundary takes precedence; an earlier grace expiry cannot invent an official five-DNF result. Original notification timestamps are preserved through `TimedPresenceEngine`.
- Spectators receive public snapshots and cannot drive or replace bots. Host leaving after start marks only that human DNF while remaining humans continue. No automatic next-match queue or penalty-style race rematch.
- `007_race.sql` and the registered transactional projector store five race rows with start slot, rank, laps/lap times, best/total time, DNF and assistance. Result retries remain idempotent through T8. Zero-lap DNF uses an empty SQL array, not SQL NULL. The projector is registered before recovery/worker startup.
- Race result return requires every remaining human's consent, or the 10-second timer beginning only after durable results. Existing guarded durable release and generation-fenced Race Park return are retained. Finalizing/storage failure handling remains the existing T8 pipeline.
- Playable local guest race with keyboard and pointer/touch controls, visible track/minimap, interpolation, standings, BOT/assistance labels, countdown, laps, best/total time, DNF and a frozen final race clock. Guest state survives modal closure and never uploads official scores. Online UI reads authoritative snapshots and does not claim saved results until the lobby reaches durable results.
- Race Park/directory/hero availability copy now reflects implemented race code. Responsive width constraints were hardened; root's later geometry inspection showed the initial clipping screenshot was a cropped capture, **not evidence of actual layout overflow**.

## Accepted start-grid binding decision

Coordinator approved optional `lobby.RegisterPrepared(game, factory, prepare)` before implementation of this seam. `Prepare` receives a cloned MatchStart and may return only an ordered actor slice/error. The generic service validates an exact permutation preserving all Actor fields, then copies/freezes it before factory creation and `PGStore.Persist`. Race owns deterministic seed-based xorshift32 Fisher-Yates; existing Register games retain identity order.

The prepared actor order is the order in `matches.start_payload`, `match_participants.roster_slot`, engine slots, public `startSlot`, `race_results.start_slot`, history participant order, and finish tie-breaks. Retrying an accepted start returns the same frozen payload and never reshuffles. No fields were added to MatchStart, Actor, MatchResult or RaceResult, preserving historical immutable result serialization/hash compatibility.

Rules now include proposed beta race handling values and the track checksum. Both lobby result metadata and hub snapshot metadata obtain the checksum from the validated rules asset, replacing the old embedded literal. Snapshot golden fixture metadata and rule/schema copies were updated consistently. The new track's embedded bytes and canonical SHA-256 are tested against the shared contract.

## Files

New core files:

- `apps/server/internal/games/race/{engine,track,bot,adapter}.go`, embedded `track.v1.json`, and engine/adapter/track tests.
- `apps/server/internal/db/migrations/007_race.sql`.
- `apps/server/internal/application/race_integration_test.go`, `race_scheduler_integration_test.go`, `race_transport_integration_test.go`.
- `apps/server/internal/lobby/prepare_test.go`, `race_results_test.go`; realtime race protocol and world race rules tests.
- `apps/web/src/features/race/{local.ts,session.ts,Race.tsx,race.css}`; `apps/web/tests/race.test.ts` and unexecuted `race.guest.e2e.spec.ts` source.
- `contracts/race-track.v1.json`, `race.golden.json`.

Updated seams: application game wiring/projector registration; lobby preparation/results timing; realtime game dispatch, rate classification and rules metadata; Go/TS rules/protocol decoders; shared schemas/fixtures/README; frontend lobby/history integration and copy; TypeScript/test runner inclusion. Existing generic “engine unavailable” application test now uses unimplemented duck instead of newly implemented race.

An approved **test-only** correction to the existing penalty transport fixture is described below. No penalty runtime guards or behavior were changed.

## Test-first and self-review findings

Initial new engine/seam test scaffolds failed compilation because the new interfaces did not yet exist; the standalone race control protocol assertion failed with `invalid command` before integration. Subsequent behavior regressions were observed RED before fixes:

- Different finish overshoot incorrectly overrode a same-frame grid tie (`slot 0` ranked fifth).
- A delayed grace tick moved a disconnected human beyond its original cutoff.
- A resumed transport could not restart input sequence after a previous sequence 500.
- Timed presence could finish naturally but drop its terminal public event.
- Five expired humans could incorrectly produce a completed result instead of abandonment.
- Go and TypeScript both lost direction-vector fractions; hand-checked braking vector expected `(18594,5603)` but got `(18593,5602)`. Remainder carry now passes the shared fixture.
- Both rule decoders accepted an omitted acceleration coefficient; they now reject missing tuning fields.
- Real PostgreSQL projection failed on a zero-lap DNF with `null value in column "lap_times_ms"`; projection now normalizes the omitted JSON empty list to an empty SQL array.

The test that drives five human actors uses ordinary control inputs from a deterministic test driver; it is **not five-human playtest evidence**. Bot completion tests run actual steering/acceleration/collision simulation, not canned positions or fabricated result timers.

## Executed verification

Working directory: `/Users/patra/Project/patra-world-social`. Go executable: `/opt/homebrew/bin/go`; all Go runs use `GOCACHE=$PWD/.cache/go-build`.

1. `go -C apps/server test ./internal/games/race` — PASS. Includes countdown/walls/expiry, real four-bot completion, five-human control-driven completion, exact five-lap finish, shared checkpoint/physics vectors, deterministic grid/ties, sequence reconnect, grace/DNF/abandon, 300/45-second limits, fixed stepping/publication and bounded catch-up.
2. `go -C apps/server test ./internal/lobby ./internal/games/race ./internal/world` — PASS during integration development. Covers exact actor permutation rejection, existing lobby lifecycle tests and race durable return semantics.
3. Exported the existing ignored database environment privately using `set -a; source .cache/database.env; set +a`, without printing secrets. The following backend runs used approved local loopback escalation and temporary `social_test_*` schemas, cleaned by existing test utilities.
4. `go -C apps/server test -tags=integration ./internal/application -run TestRace -count=1 -v` — PASS all four:
   - `TestRaceApplicationGridPhysicsDurabilityAndTimedReturn` (0.43s): persisted/runtime grid identity, duplicate start/idempotent retry, countdown, spectator/world binding, actual owner stepping, four bot finishes, official history and projector retry, 10-second durable result release.
   - `TestRaceApplicationFiveHumanDeadlinePrecedesGraceExpiry` (0.10s): five frozen human actors, assistance/DNF and natural 300-second end at simultaneous grace expiry.
   - `TestRaceWiredSchedulerPublishes15HzAndHostDepartureDoesNotStopPeer` (0.10s): actual application factory registration + generic owner, exactly 30 frames/15 published events over 20 50-ms advances, host DNF while peer races, late spectator cannot replace a bot, all-human departure durably abandoned without a result job.
   - `TestRaceWebSocketContinuousInputUsesMovementBudget` (0.10s): real ephemeral HTTP/WebSocket transport admits 20 continuous controls without hitting the 10/s control budget; accepted controls move the human; public snapshot omits the seed.
5. `go -C apps/server test -tags=integration ./...` — full suite PASS before the later remainder/timed-presence refinements.
6. Final `go -C apps/server test -race -tags=integration ./...` — full suite PASS after final implementation and approved penalty test harness correction. Output: application `ok 21.671s`; all auth/chat/config/model/penalty/race/history/http/lobby/moderation/realtime/social/world/worlds packages `ok` (cached where unchanged); cmd/e2edb/contracts/db/testutil report no test files. **No race detector findings.**
7. Following the full pass, test-only improvements added an embedded track/checksum assertion and corrected the protocol range test to change only the numeric value (without an unrelated unknown field). Targeted `go -C apps/server test ./internal/games/race ./internal/realtime -run 'TestRaceOnly|TestTrackAsset'` — PASS both packages.
8. `npm test` — final PASS: 4 JS tests plus 45 bundled TS tests, 49 total. Includes 9 race tests: shared gates, actual bot completion/expiry, bounded catch-up, fixed-point vectors, shuffle parity, guest modal lifetime, spectator/saved-result rendering, missing coefficients, frozen final clock.
9. `npm run typecheck` — PASS.
10. `npm run lint` — PASS; `go -C apps/server vet ./...` — PASS; `git diff --check` — PASS.
11. `npm run build` — PASS, 59 modules, final candidate `index-5pNF16M7.js` 336.12 kB / gzip109.52 kB and `index-CF46jrA-.css` 61.19 kB / gzip14.57 kB. `go -C apps/server build -o ../../.cache/t10-server-check ./cmd/server` — PASS. Build does not start/restart the API.

Execution-boundary/harness attempts retained for honesty:

- An initial unprivileged realtime suite attempt was denied binding `[::1]:0`; this was a sandbox boundary, not a game assertion failure. Later loopback test commands were explicitly escalated and executed successfully.
- First PG invocation sourced but did not export DATABASE_URL (`DATABASE_URL required`); corrected to private export. New test setup then needed `Addr:127.0.0.1:0`, legitimate memberships for all five synthetic users, and the same admission-validation renewal as the actual transport. These fixture setup corrections are not product failures.
- A standalone esbuild invocation after React rendering tests were added omitted the repository runner's `createRequire` banner and failed dynamic `require('util')`; the normal `npm test` runner executes correctly.
- Full npm test initially found the old shared world-snapshot rules checksum; golden metadata was updated to the validated new rules checksum.

## Penalty transport harness investigation (coordinator approved)

The first final full race-detector run failed `TestPenaltyActualTransportHistoryAndMutualRematch` at the rematch call with `connection fenced`; every other package passed and there was no data race report. An isolated repeat with `-race -tags=integration ./internal/application -run '^TestPenaltyActualTransportHistoryAndMutualRematch$' -count=5 -v` yielded **1 pass / 4 failures**, either late in turn20 or at rematch.

After its last synchronous WebSocket assertion, the original test accelerated about90 virtual seconds through direct admitted commands while no longer reading sockets. Adding active socket readers alone yielded **0 passes / 5 failures**: the accelerated event producer still outran the server writer/authorization path. The production queue is **64 outbound events**, not32 (32 is the inbound command queue). It correctly fenced synthetic clients when publication ran ahead.

The final fixture now has exactly one reader per socket after synchronous `send` assertions, validates all event envelopes, surfaces unexpected termination, and waits for the current turn's actual public snapshot on all three sockets before advancing virtual time again. Reader context is canceled and readers are joined during cleanup. Production admission, expiry, backpressure and rate limits are unchanged; existing slow-client tests remain.

The same isolated race-detector repeat then yielded **5 passes / 0 failures** (0.67–0.91s each; package5.202s), followed by the full race-detector suite PASS reported above. This is a paced test-clock/real-transport barrier, not suppressed fencing or retries around production failures.

## Root-owned browser evidence and remaining acceptance

Root reports final-build actual CUA guest smoke on existing preview4176 (no browser credential workflow): countdown3 and five kart roster; keyboard W then D moves and steers the guest; pointer gas control also exercised in the earlier smoke; closing/reopening the running modal retains the race; final-build bot times93.97/95.87/97.87/99.77s; guest DNF at138.97s equals first93.97+45s; final elapsed138.97s stayed unchanged after another13 seconds; console warnings/errors empty. Root owns `docs/checkpoints/evidence/CP5/t10-guest-final-results.jpg` and the written evidence. Earlier92–98s times refer to the pre-remainder build only. Initial screenshot width938 was a crop of a1280 CSS viewport; root geometry found no horizontal overflow.

Still **not executed or not claimed**:

- Credentialed human browser account/invite workflows and live online browser race acceptance remain explicitly deferred. The running preview API8080 was not restarted and remains the coordinator's prior T9 process; T10 backend evidence comes from isolated compiled application tests.
- Actual1–5 human multiplayer playtests, physical-mobile steering, real latency/jitter, beginner win/bot-balance tuning and device FPS targets. Handling in rules-v1 is proposed beta tuning, not release-certified tuning.
- The new Playwright guest test is source coverage only; no CLI Playwright run was performed. It remains under the repository's isolated browser-test configuration.
- The known T9 O1 coalesced resume/new-disconnect generic presence boundary remains tracked for T16/final review. T10 did not alter that generic presence algorithm and does not claim its race reconnect tests resolve O1.
- No official score upload exists for guest play. No production/deployment/provisioning work was attempted.

## Review fix round 1 — R1 and R2 (2026-10-27)

Changes are **unstaged** relative to the coordinator's reviewed staged tree `fdd7cff0faad511a055313511a187386e85e5a4f`. No staging, reset, commit, preview/API restart, generic presence/O1 change, or root-owned evidence/document edit was performed by this implementer. Review scope is the two Important findings in `task-10-review.md`; CP5 acceptance is still separate.

### R1 — persistent guest input sequence

Moved the sequence counter from the mounted `RaceGame` effect into `GuestRaceSession`, whose lifetime already matches the continuing local engine. Starting/clearing a race resets that counter; closing/reopening its UI does not. The simulator still rejects replayed/nonmonotonic sequences. Reformatted the amended session and input effect into readable blocks. Changed files:

- `apps/web/src/features/race/session.ts`
- `apps/web/src/features/race/Race.tsx`
- `apps/web/tests/race.test.ts`
- `apps/web/tests/race.guest.e2e.spec.ts`

The session regression accumulates 600 accepted neutral inputs during a running race, detaches its controller callback, advances the persistent session, attaches a fresh callback, then requires its **first** throttle+steer input to succeed. The next simulated frames move and turn the same guest; match identity, prior checkpoint/lap progress and absolute deadline are unchanged. A replay of sequence600 remains rejected. The authored browser test now drives after reopening; that source was not executed by CLI. Root's actual CUA test supplies the real component unmount/remount evidence.

RED/GREEN: the initial new regression run before the session API change failed `false !== true`. The review's original-session probe and root's original-build browser reproduction independently established the retained-engine/reset-controller mismatch (see review and root evidence), rather than inferring that cause from the test API transition alone. After the fix, `npm test` passed **4 JS + 46 TS = 50 tests**, including **10 race tests**; `npm run typecheck` passed; `npm run build` passed (59 modules, `index-DCicAL2o.js` 336.09 kB / gzip109.52 kB). The 49-test count earlier in this report describes the pre-review candidate.

Root reports actual fixed-candidate CUA PASS: idle race31.42s at Guest `(18432,5600,0°)`; close/reopen38.84s; the first W2s then D650ms immediately moves/turns to `(31421.93,8396.94,57°)` by41.69s in the same race/grid, without waiting for an old counter. Console warnings/errors empty. Root owns `docs/checkpoints/evidence/CP5/t10-guest-reopen-input-fixed.jpg` and its evidence narrative. Frontend freeze was released afterward.

### R2 — metadata-only disconnect installation and original-time causality

`Engine.Presence(...,"disconnect",at)` now installs metadata and returns without ticking, moving, finalizing or abandoning. The later ordered advance applies bot takeover at the original disconnect timestamp. Installing metadata clears active input/fences new human commands immediately but retains the prior intent and its receipt timestamp for catch-up frames **before** that boundary, with the ordinary 250ms expiry/future-input checks. Resume clears that retained intent, resets the transport sequence as before and retains immutable human identity/assistance history. Ordered resume/departure still advances first so natural finish wins at the exact deadline. No generic owner algorithm, cadence, catch-up budget, result format, database contract or O1 behavior changed.

Changed/added Go files:

- `apps/server/internal/games/race/engine.go`: metadata-only branch and timestamp-aware selection of retained human intent versus bot controls.
- `apps/server/internal/games/race/presence_test.go` (new): both explicit metadata permutations; no frame/time/position/phase/result advance while installing each event; identical authoritative snapshots afterward; frame30 with22 dropped steps and only8 simulated catch-up frames; later participant gets only its first boundary step. Separate nonzero held throttle/steer test checks pre-boundary heading345°/speed25, bot boundary heading348°/speed30, neutral/fenced disconnected control, assistance identity, resume and replay rejection.
- `apps/server/internal/games/race/lobby_presence_test.go` (new): actual lobby owner held in `ReadEngine` while two different timestamp notifications accumulate. An observing wrapper delegates each call immediately to the real adapter, checks every actual metadata installation, and compares the resulting snapshot to the complete chronological batch. It does **not** substitute physics, suppress owner events or reorder the map. Both enqueue permutations run for takeover, one original15s grace expired, all-human abandonment, natural finish exactly at a grace boundary, and ingestion500ms after that deadline (10 subcases). Direct engine coverage deterministically exercises both installation permutations; actual map iteration remains unspecified. The owner tests also assert durable-release intent `abandoned` without an official result and one real `race.finished` event at the natural deadline. A held-persistence activation test verifies all preserved disconnect metadata reaches the fresh adapter before the earlier expired departure, a fresh3s countdown, no countdown movement, B's takeover and B's original grace expiry while C keeps racing.
- `apps/server/internal/games/race/adapter_test.go`: disconnect alone cannot finalize/publish; subsequent owner Tick publishes the natural terminal snapshot. Ordered `resume`, `expired`, and `leave` at the exact grace/deadline boundary preserve one terminal event and do not repeat it on later ticks.

RED/GREEN evidence:

1. Before the production fix, from repository root:
   `GOCACHE="$PWD/.cache/go-build" /opt/homebrew/bin/go -C apps/server test ./internal/games/race -run 'TestDisconnect(Batch|Preserves)' -count=1 -v`
   **FAIL both regressions**: metadata changed time/phase during installation; retained input boundary observed heading342°/speed30 after premature advance instead of the pre-disconnect five-frame state. After the production fix, the held-input test passed; a takeover assertion initially expected +5 X for B's first step but actual direction357° uses integer cosine/remainder and moves +4. Corrected that test to require positive movement no more than5 and speed5, preserving the one-step invariant. Both then passed.
2. `GOCACHE="$PWD/.cache/go-build" /opt/homebrew/bin/go -C apps/server test ./internal/games/race -count=1` — **PASS**, package0.532s.
3. `GOCACHE="$PWD/.cache/go-build" /opt/homebrew/bin/go -C apps/server test ./internal/games/race -run 'TestLobbyRace' -count=1 -v` — **PASS**, all10 held-owner subcases plus activation replay, package0.561s.
4. To verify the new real-owner test detects the reviewed defect without resetting/editing production state, copied the staged engine read-only with `git show :apps/server/internal/games/race/engine.go` to `/private/tmp/t10-fix-round1/engine-before-r2.go`, then mapped it through a Go overlay. Command:
   `GOCACHE="$PWD/.cache/go-build" /opt/homebrew/bin/go -C apps/server test -overlay /private/tmp/t10-fix-round1/r2-red-overlay.json ./internal/games/race -run 'TestLobbyRaceHeldOwnerInstallsWholeDisconnectBatchBeforeAdvance/takeover|TestLobbyRaceActivation' -count=1`
   **FAIL**, both actual-owner takeover enqueue subcases report `owner disconnect metadata advanced race` (including a later B event advancing directly to frame30 before A metadata). Activation replay retains its existing PASS. Full RED output is `/private/tmp/t10-fix-round1/r2-owner-red.log`; the overlay never changed the index or working engine.
5. Final affected-package race detector:
   `GOCACHE="$PWD/.cache/go-build" /opt/homebrew/bin/go -C apps/server test -race ./internal/games/race ./internal/lobby -count=1`
   **PASS**: race2.417s; lobby1.489s; no data race findings.
6. Privately exported the existing `.cache/database.env` with `set -a; source .cache/database.env; set +a`, then ran the approved isolated loopback fixtures:
   `GOCACHE="$PWD/.cache/go-build" /opt/homebrew/bin/go -C apps/server test -race -tags=integration ./internal/application -run '^TestRace' -count=1 -v`
   **PASS all4**: grid/physics/durability/timed-return0.72s; five-human deadline-before-grace0.15s; wired30Hz/15Hz scheduler+host departure0.15s; real WebSocket continuous-input movement-budget0.18s. Package2.790s; no race findings. Existing localPG55432/temporary test schemas and ephemeral HTTP/WS only; no preview process or human browser credentials.
7. Final focused web regression rerun from `apps/web`, using the same existing esbuild settings as `scripts/test-map.mjs`, only `tests/race.test.ts` as entry point, temporary `.mjs` output and `node --test`: **10 tests,10 pass,0 fail**, duration127.70075ms. Exact command below. No unchanged full web/Go suite was rerun after these affected checks.
8. `git diff --check -- apps/server/internal/games/race apps/web/src/features/race apps/web/tests/race.test.ts apps/web/tests/race.guest.e2e.spec.ts` — **PASS**, no output. `/opt/homebrew/bin/gofmt -l` on engine, adapter_test, presence_test and lobby_presence_test — **PASS**, no output.

Focused web command:

```sh
node --input-type=module <<'JS'
import {build} from 'esbuild';
import {spawnSync} from 'node:child_process';
import {mkdtempSync, rmSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
const dir = mkdtempSync(join(tmpdir(), 'patra-r1-'));
try {
 await build({entryPoints:['tests/race.test.ts'],outdir:dir,banner:{js:"import {createRequire} from 'node:module'; const require=createRequire(import.meta.url);"},bundle:true,platform:'node',format:'esm',outExtension:{'.js':'.mjs'},define:{'import.meta.env.DEV':'false'}});
 const result = spawnSync(process.execPath,['--test',join(dir,'race.test.mjs')],{stdio:'inherit'});
 process.exitCode = result.status ?? 1;
} finally { rmSync(dir,{recursive:true,force:true}); }
JS
```

Self-review: active input is neutral/fenced at installation while historical input remains bounded by its original receipt/expiry; both direct order permutations converge; actual owner lifecycle, deadline publication and activation replay pass; guest counter now shares engine lifetime. Existing O1 remains deferred, and minor formatting outside amended blocks was not broadened into this fix. No new acceptance claims for human five-lap completion, novice tuning, physical mobile/FPS, live multiplayer/latency or credentialed online browser flows. R1/R2 fixes are ready for coordinator packaging and re-review; working fix files and report will remain unchanged after handoff until further instruction.

## Review fix round 2 — R3 global cutoff assistance (2026-10-27)

R1/R2 were accepted by the scoped rereview. R3 is corrected in four unstaged race files relative to staged baseline `cb2355e8e2c7f3c2cf65d82f3fe0b89979c11808`:

- `apps/server/internal/games/race/engine.go`: disconnect remains metadata-only and no longer assigns public/history assistance. Accepted authoritative advancement records eligible original disconnect timestamps only after determining whether it reached an earlier natural terminal boundary. `finish(at)` records only timestamps strictly before that global finish; prior assistance is never cleared. Countdown and abandoned advancement still record their already-effective history. A disconnect exactly at the absolute/shortened cutoff cannot select bot controls for that natural final frame, preserving the previous human intent/expiry and natural-finish precedence. No generic owner/transport/storage contract changed.
- `apps/server/internal/games/race/presence_test.go`: eight direct cases compare complete final `MatchResult` and kart positions/times to a no-new-disconnect baseline across absolute versus real first-finish+45s cutoff, exact versus +1ms events, and never-assisted versus genuinely previously-assisted/resumed actor. The shortened fixture crosses the real final checkpoint on lap5 at10000 and asserts cutoff55000; it does not merely set the deadline field. Additional checks retain real pre-cutoff takeover/true assistance and an all-finished global result at10033 that cannot absorb later metadata at10040.
- `apps/server/internal/games/race/lobby_presence_test.go`: two actual held-owner cases at cutoff and cutoff+1ms. Five-human race remains running on tick303950; owner is held while notification304000/304001 is queued, then processes its ordinary metadata-before-advance path. Both require original finish304000, exact result/positions/times equal to the no-event baseline, false assistance, one matching terminal event and no repeated event. Existing activation replay, unordered metadata and grace/deadline tests remain intact.
- `apps/server/internal/games/race/adapter_test.go`: shortened cutoff55000 followed by disconnect55001 produces unchanged official result and one matching `race.finished`, with no metadata-time or repeated publication.

RED evidence before changing the engine:

1. `GOCACHE="$PWD/.cache/go-build" /opt/homebrew/bin/go -C apps/server test ./internal/games/race -run 'TestDisconnectAssistanceRespectsGlobalCutoff|TestPreCutoffDisconnect' -count=1 -v` — **FAIL**, six of the eight cutoff cases: four never-assisted cases contaminated the flag during metadata installation; two previously-assisted exact-cutoff cases changed terminal kart positions via a bot step. Both previously-assisted +1ms cases and the true pre-cutoff takeover control passed. Package0.631s.
2. `GOCACHE="$PWD/.cache/go-build" /opt/homebrew/bin/go -C apps/server test ./internal/games/race -run '^TestLobbyRacePostCutoff' -count=1 -v` — **FAIL both real-owner cases**, `late owner notification contaminated official result`; package0.465s. Full output retained at `/private/tmp/t10-r3-owner-red.log`. The file redirection/readback did not change the tested state or index.

GREEN verification after the production fix:

1. `GOCACHE="$PWD/.cache/go-build" /opt/homebrew/bin/go -C apps/server test ./internal/games/race -run 'TestDisconnect|TestPreCutoff|TestLobbyRace|TestTimedPresence|TestOrderedDeparture' -count=1 -v` — **PASS**, package0.507s. Includes all8 direct cutoff cases, both real-owner cutoff cases, prior metadata/held-input tests, all10 held-owner grace/deadline cases, activation replay, original disconnect lifecycle and three ordered terminal-presence cases.
2. After adding the shortened-adapter and all-finished regressions, final `GOCACHE="$PWD/.cache/go-build" /opt/homebrew/bin/go -C apps/server test -race ./internal/games/race ./internal/lobby -count=1` — **PASS**, race2.069s, lobby1.974s; no race detector findings. All new tests and accepted R2 coverage ran together.
3. Privately exported `.cache/database.env` using `set -a; source .cache/database.env; set +a`, without printing secrets. Approved local isolated PG fixtures: `GOCACHE="$PWD/.cache/go-build" /opt/homebrew/bin/go -C apps/server test -race -tags=integration ./internal/application -run '^TestRaceApplication' -count=1 -v` — **PASS both affected official-result cases**: `TestRaceApplicationGridPhysicsDurabilityAndTimedReturn`0.46s; `TestRaceApplicationFiveHumanDeadlinePrecedesGraceExpiry`0.16s; application package2.195s; no race findings. No browser credential workflow, live API/preview restart, or transport change.
4. `git diff --check -- apps/server/internal/games/race` and `/opt/homebrew/bin/gofmt -l` on the four changed files — **PASS**, no output. Final post-test edit only clarified the retained-intent comment to mention exact-cutoff precedence; no executable changes followed these tests.

No unchanged full-suite/web/browser tests were repeated in this round. Root's R1 actual CUA evidence remains applicable because frontend files did not change. Generic O1 and the pre-existing personal-finish assistance edge remain separately ledgered for T16, not resolved or broadened here. Existing human five-lap, novice tuning, physical-device/FPS and credentialed online acceptance limits remain unchanged. No subagents, staging, reset, commit, root evidence edits or process restarts. R3 changes and this report are ready for narrow coordinator re-review and freeze after handoff.
