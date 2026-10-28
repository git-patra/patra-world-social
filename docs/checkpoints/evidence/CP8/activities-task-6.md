# Task 6 — Free football, pitch penalty, passive spectators

Status: implemented and frozen for independent review. Worker did not commit, push, restart any process, run shell browser automation, spawn agents, or change the source `patra-world` repository. Root owns registration, browser evidence, review, and the eventual commit.

## Delivered behavior and chosen defaults

- One football lives on each world owner. Guest mode has the same single-ball state through the existing activity clock; neither path creates a match, completion, history, inventory item, timer, or RAF for casual football.
- Spawn/reset is `(28,46)`. Kick proximity is **1.5 tiles**, distinct from the **0.15-tile physical/drawn radius**. Each accepted kick normalizes a finite nonzero direction to at most **8 tiles/s**, replaces the previous impulse, and starts a **400 ms per-user cooldown**. Friction multiplies velocity by **0.9 per 50 ms**; speed below **0.05 tiles/s** sleeps. The ball resets after **15 s without a kick**, or if it escapes bounds.
- Collision sweeps in intervals no larger than **0.05 tile** against the existing penalty-area rectangle inset by the physical radius. A boundary hit stops the ball at the last contained sample. Fixed-step catch-up remains capped at four casual steps; walking/race rules and the existing 250 ms continuous-input expiry are unchanged.
- Kick is a discrete control command, separate from movement transport budgets. The owner checks admitted epoch, increasing envelope sequence, runtime incarnation, penalty-area generation, football generation, player status, rider exclusion, actual player-to-ball distance, and per-user cooldown. Rejecting a kick does not consume its movement sequence. Reconnecting does not clear the retained user's cooldown.
- Penalty reservation atomically hides and resets the casual ball before clear ACK/countdown. It stays unavailable through CLEARING, RESERVED, ACTIVE, and RESETTING. OPEN restores a stationary ball. Old leases/kicks cannot affect a new rematch reservation.
- Guest penalty startup hides/resets the casual ball before the lazy session import completes. Explicit exit/return resets it and retains the existing safe return `(28,52)`. Existing four-zone private choices, 3 s choice deadline, 1.5 s reveal, rules, completion, and saved-result behavior remain in their existing engines.
- Existing public penalty scene renders kicker, keeper, and match ball using public snapshot/server time. Casual football has its own lazy renderer on the world's existing RAF. Removed the decorative football, so match and casual representations cannot coexist as a second static ball.
- Near the casual ball, E / Kick tap sends a kick away from the avatar's side of the ball (standing at its center defaults goalward). Walking remains the input owner. The small HUD explains the interaction; the dock uses plain Kick / Naik / Turun labels with its existing E keycap. Held E does not repeat discrete actions, and a rejected cooldown kick keeps ball context instead of opening the penalty menu.
- A visitor standing in the contract's outside spectator strip sees a passive penalty score/timer HUD. They retain walking controls and can walk away. They have no choice buttons, match input authority, or history rights.
- Scoped Task 5 carry-over fixed: ride-entry sound now keys off the self rider's vehicle ID/generation transition, rather than any global pool revision. A second rider mounting/dismounting, resync, and stale lifecycle replay do not repeat the current rider's cue. Football sound likewise only plays for a fresh live kick transition, with distance attenuation; duplicate, resync, and >1 s delayed cues are dropped.

## API seams and files

Backend:

- `apps/server/internal/world/football.go`: one `Football`, kick admission, chosen constants, swept contained integration, idle/escape reset. Uses only world owner state; imports no game/history/completion service.
- `world/{runtime,vehicles,areas}.go`: initialize/advance the ball; `ApplyCasualIntent` dispatches its exact `action:kick` branch; `CasualIntent.MarshalJSON` omits vehicle ID for kicks and retains explicit zero axes; area reservation and timed OPEN reset/hide the ball.
- Snapshot union adds `kind:football`, `areaId:penalty`, independently increasing football generation, and exact `payload:{ball:{x,y,vx,vy,visible}}`. Four `kind:area` masks and the existing `kind:vehicles` pool remain independent.
- `realtime/{activity_protocol,protocol,hub,areas}.go`: exact `activity.kick` validation, admitted owner execution, bounded snapshot validation, and kind-specific lifecycle keys. No kick was added to the continuous-drive classifier.
- The actual-owner pending-result regression exposed a stale world `playing` status after spatial release. Casual Hub commands now use the existing walking re-admission seam only after `MovementBusy` is false. `Busy` remains true for durable results, so this does not release the account/history claim or allow another match.

Contracts/frontend:

- `contracts/{activity.schema,protocol.schema,activities.golden,commands.golden}.json`: exact bounded football snapshot/kick variants and malformed/extra-field/zero-direction cases. Runtime validators also check total velocity magnitude and geometry, as with vehicle geometry semantic validation.
- `activities/contracts.ts`: Football/FootballSnapshot/KickIntent types and strict validation; chosen local constants and containment.
- `activities/football.ts`: local guest physics/proximity/reset using the existing 50 ms store clock and four-step catch-up cap.
- `activities/store.ts`: independent football incarnation/revision ordering and retired-runtime fences; guest/online interaction; local-start hiding; passive nearby spectator state; focused ride-SFX ownership fix.
- `net/socket.ts`: strict full resync now requires **six** snapshots: four distinct area masks, one vehicle pool, one football. `world/avatars.ts` recognizes the exact kick command.
- `activities/{scene,ActivityHUD}.tsx/ts`, `world/scenes/football.ts`, `world/{controller,village-art}.ts`, `app.js`: shared-world projection, small context HUD/dock, E input behavior, and removal of the decorative ball. No second animation loop or input owner was introduced.

New test entrypoints: `apps/web/tests/football.test.ts`, `apps/server/internal/world/football_test.go`, `apps/server/internal/realtime/football_test.go`. Root registered the frontend entrypoint in `apps/web/scripts/test-map.mjs` and `apps/web/tsconfig.json`; worker did not edit those registration files.

## RED evidence

Observed before the corresponding implementation/fix:

1. New owner tests `TestKickRequiresProximityAndCooldown`, `TestCasualBallDisabledWhenPenaltyReserved`, and `TestFootballFrictionBoundsAndIdleReset`: `world snapshots omit the single casual football`.
2. Realtime `TestFootballWireIsExact`: `valid kick rejected invalid command`; actual Hub test also rejected the missing command.
3. Frontend football contracts rejected `Invalid activity snapshot`; guest store test reported `guest world is missing its one football`. Scene test observed zero casual balls instead of one, and nearby spectator state was absent.
4. Focused two-rider sound regression: `another rider repeats the self entry cue`, observed **2 vs 1** calls. It now also covers the peer dismount, resync, and stale replay.
5. Actual-owner result-delay test, verified by reverting only the re-admission fix and restoring it: `spatial release did not restore football while durable result pending activity ownership superseded`. Corrected run passes for race and penalty and retains `g.Busy("user")`.
6. Browser-requested dock regression: `Kick · E` vs `Kick`; held-E regression: **2 vs 1** world interactions. Both pass after the focused shell change.
7. Actual shell cooldown regression: second interaction opened the penalty dialog (`true vs false`). It now remains in the football context.
8. Football live-audio regression: `a delayed kick cue should be dropped`, **2 vs 1** calls. The age fence passes without changing physics or snapshot acceptance.

The first shell-copy assertion needed the fixture's actual 100 ms notification cadence; after advancing that cadence, it failed at the intended duplicated label. A temporary missing `nearFootball` import caused the cooldown shell and existing controller route tests to fail; adding the real import restored both. Typecheck initially found the audio boundary test stub returned `void` rather than the actual boolean; the stub now preserves its signature. None remain failing.

## Verification commands and outcomes

All Go commands use the pinned executable from repository root:

```sh
GOTOOLCHAIN=local GOCACHE=$PWD/.cache/go-build .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server ...
```

- Focused owner/strict Hub commands: `test ./internal/world -run 'TestKick|TestCasualBall|TestFootball' -count=1`, `test ./internal/realtime -run TestFootball -count=1`: PASS after their RED runs.
- Actual owner/lobby durability: `test ./internal/application -run TestStartWaits -count=1`: PASS, including the new penalty subtest.
- Final full unit suite: `test ./... -count=1`: **PASS all packages**. The first sandbox run failed to bind loopback listeners in `TestServeUntilSignalWaitsForHTTPDrain`, `TestTransportAdmissionMovementAndRevocation`, and `TestSDKSerializesExplicitCutoff`. Approved local-listener execution passed; no test assertion was weakened.
- Focused race detector: `test -race ./internal/world ./internal/realtime ./internal/application -run 'TestKick|TestCasualBall|TestFootball|TestStartWaits|TestRide|TestVehicle' -count=1`: **PASS**, package outputs world 4.778 s, realtime 2.069 s, application 8.481 s.
- Latest owner-only race run after adding explicit 2999 ms clear-ACK, four-step catch-up, and escaped-ball assertions: `test -race ./internal/world -run 'TestKick|TestCasualBall|TestFootball' -count=1`: **PASS** (1.556 s).

Actual database/WebSocket integration, environment sourced/exported quietly without printing credentials:

```sh
set -a
source .cache/database.env
set +a
GOTOOLCHAIN=local GOCACHE=$PWD/.cache/go-build .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server test -tags=integration ./internal/application -run 'TestPenalty|TestFreeRider' -count=1
```

**PASS** (2.465 s), using approved loopback/database access. Extended the existing `TestPenaltyActualTransportHistoryAndMutualRematch`, rather than writing a duplicate privacy/history harness:

- Successful casual kick over the actual WebSocket ACK path asserts **zero persisted matches** in the new world before penalty starts.
- Starting penalty asserts the casual ball is hidden/stationary.
- Real spectator transport re-admission at `choiceDeadlineAt-1` retains the public chosen flag with no current `kickZone`/completed turns; its casual kick is rejected.
- Existing real deadline, spectator choice rejection, cross-world denial, completed turn projection, retry idempotency, spectator history denial, mutual rematch, BOT result/return, and abandoned cleanup assertions remain and pass.
- Existing actual-owner area test proves RESETTING→OPEN restores football/walking while the durable Busy claim remains. This uses actual Hub/world/lobby owners, with a deterministic pending-storage boundary; it does not pretend a transient result is saved.

Frontend final checks, from repository root:

- `npm test`: **PASS 4 JavaScript tests + 173 bundled tests**, zero failures/skips.
- `npm run typecheck`: **PASS**.
- `npm run lint`: **PASS**.
- `npm run build`: **PASS**. Final main `index-CaflXEpR.js` **430.06 kB / 136.50 kB gzip**; lazy football renderer **0.35 kB / 0.25 kB gzip**. Vite reports kB. This is a build measurement, not a device FPS/network/heap claim.
- `git diff --check`: **PASS**.

Frontend coverage additionally checks strict six-part resync (missing/duplicate kinds/masks rejected), independent state and retired-runtime fences, guest physics bounds/friction/idle/catch-up, hide-before-lazy-penalty projection with exactly one match ball, passive nearby reconnect state with walking escape, rendered spectator HUD without buttons, and scoped audio/input regressions. Existing four-zone privacy and guessed-ball-before-reveal tests remain unchanged and pass.

## Browser handoff and evidence

Root independently ran and owns `docs/checkpoints/evidence/CP8/activities-football-browser.md` at production localhost 4181, Chromium IAB, guest desktop 1440×900 and mobile 390×844.

Reported observations: Map → Lapangan Penalti walks to `(28,46)`; Kick visibly sends the ball goalward; D for 400 ms walks to `(30,46)` and removes nearby Kick; guest penalty replaces casual controls with four-zone HUD and kicker/keeper/match ball; explicit unfinished-practice exit returns `(28,52)`, walking north restores reset ball/Kick. Mobile HUD/dock separation is 12 px. Actual device History retains only its two prior completed entries; casual kicks and unfinished practice add none. The dock copy follow-up requested by this check is implemented and has a focused shell regression.

Repeat browser path: Map → Lapangan Penalti; stand within 1.5 tiles of `(28,46)`; press E / tap Kick. Approach from the desired passing side. Menu → Duel penalti → Mulai practice vs BOT; use the small four-zone HUD. Menu → Duel penalti → explicit practice exit; return from `(28,52)` to the ball. Online passive strip is `x=23, y=40..50`, outside the sterile pitch; actual authenticated browser/multiplayer remains a separately deferred gate.

## Self-review and limitations

- Reviewed owner serialization, per-actor cooldown retention, epoch/sequence/incarnation/area/ball generations, last-millisecond clear ACK, rematch old release, and four-step catch-up. Rejection cannot authorize a client position, score, or history write.
- Reviewed mask/vehicle/football state keys and independent resync fences. Unknown/missing area state keeps casual football hidden until the strict resync has the OPEN mask. An old runtime or older generation cannot restore a hidden ball or old rider.
- Reviewed reserved/reveal rendering: the casual ball hides/reset before penalty; penalty motion continues to use public engine turns and server time. No current secret choice was added to any public payload.
- Reviewed spatial vs durable release with the actual owner regression. `MovementBusy` can end so casual play resumes while `Busy` still fences match/account/history operations.
- Reviewed no completion/history imports or writer calls in the football owner/local/store paths. Actual successful-wire SQL count and browser History observations support this property.
- Reviewed one world RAF, reused store interval, bounded entity count, independent pointer ownership, modal/text/disconnect controls, and late/resync audio suppression. No unbounded particle/audio/entity pool was introduced.
- No deployment or process restart occurred. No fake session, browser auth, LiveKit acceptance, physical-device FPS, after-GC heap, or end-to-end network-load certification was created. These previously deferred gates remain explicit.
- Shared game rules, map schema/identity, protocol 2 incompatibility guard, historical protocol-1 data, race checksum/five laps/five grid slots, Duck, podium/history/accounts/friends/moderation/voice were not rewritten. Their relevant existing suites remain green.

## Review fix round 1 — Preserve walking during casual re-admission

The Important review finding was valid: the initial pending-result fix called the general `Activity(ctx,user,false,nil)` before every casual command. That method deliberately neutralizes held movement and destination paths. Consequently, both accepted and rejected HUD kicks canceled an otherwise valid click-to-walk route. This section supersedes the earlier description of using the general walking re-admission call.

Fix:

- Added the narrowly scoped world-owner `ResumeCasual(ctx,user,epoch)` operation in `world/activity.go`. It checks context and the current player epoch and only changes stale `playing` presence to `available`. It never changes movement axes, expiry, destination path, position, rider ownership, sequence, or durable claims. General `Activity`/`ReturnActivity` semantics are unchanged.
- The Hub keeps its `GameBusy`/`MovementBusy` rejection first, then uses the new owner operation before casual intent execution. Existing kick/ride owner admission and generation checks remain in place. An already available walker is left untouched, including when its kick is rejected at cooldown or the ball-generation fence.
- Added `TestFootballKicksPreserveActiveDestination` to the actual Hub regression file. Its accepted, cooldown, and stale-generation cases establish a real `player.destination`, execute the actual admitted kick, advance the owner clock without any replacement movement command, and assert that walking continues. Rejected cases assert the specific cooldown/generation error, rather than accepting arbitrary failure.
- Added the owner test `TestResumeCasualPreservesWalkingAndFencesEpoch`, proving a wrong epoch cannot restore game presence and an already available player's held movement survives re-admission.
- The existing actual-owner pending-result regression remains green for race and penalty, including football availability after RESETTING→OPEN and the retained durable `Busy` claim.

Exact RED command from repository root:

```sh
GOTOOLCHAIN=local GOCACHE=$PWD/.cache/go-build .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server test ./internal/realtime -run '^TestFootballKicksPreserveActiveDestination$' -count=1
```

Observed before the fix (exit 1):

```text
--- FAIL: TestFootballKicksPreserveActiveDestination (0.00s)
    --- FAIL: TestFootballKicksPreserveActiveDestination/accepted (0.00s)
        football_test.go:138: accepted kick canceled destination: before={ID:u Name:u X:28 Y:46 Status:available Seq:1} after={ID:u Name:u X:28 Y:46 Status:available Seq:2}
    --- FAIL: TestFootballKicksPreserveActiveDestination/cooldown (0.00s)
        football_test.go:138: cooldown kick canceled destination: before={ID:u Name:u X:28 Y:46 Status:available Seq:2} after={ID:u Name:u X:28 Y:46 Status:available Seq:2}
    --- FAIL: TestFootballKicksPreserveActiveDestination/stale_generation (0.00s)
        football_test.go:138: stale_generation kick canceled destination: before={ID:u Name:u X:28 Y:46 Status:available Seq:2} after={ID:u Name:u X:28 Y:46 Status:available Seq:2}
FAIL
FAIL patra-world-social/server/internal/realtime 0.568s
FAIL
```

The same command after the fix (exit 0):

```text
ok patra-world-social/server/internal/realtime 0.548s
```

Pending-result regression command after the fix:

```sh
GOTOOLCHAIN=local GOCACHE=$PWD/.cache/go-build .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server test ./internal/application -run '^TestStartWaitsForClearAckAndDurabilityAcrossActualOwners$' -count=1
```

Output (exit 0):

```text
ok patra-world-social/server/internal/application 0.843s
```

Final focused affected-scope race-detector command, with approved loopback listener permission for the included existing WebSocket ride test:

```sh
GOTOOLCHAIN=local GOCACHE=$PWD/.cache/go-build .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server test -race ./internal/world ./internal/realtime ./internal/application -run 'TestResumeCasual|TestFootball|TestKick|TestCasualBall|TestRide|TestVehicle|TestStartWaits' -count=1
```

Output (exit 0):

```text
ok patra-world-social/server/internal/world 3.734s
ok patra-world-social/server/internal/realtime 1.915s
ok patra-world-social/server/internal/application 6.290s
```

`git diff --check` also passed with no output. This round changed only `world/activity.go`, `world/activity_test.go`, `realtime/hub.go`, `realtime/football_test.go`, and this report. No unrelated full suites were repeated, and there were no frontend changes, commits, process restarts, or subagents. The separately recorded minor delayed ride cue remains deferred to the shared audio pass as directed; it is not part of this fix.

Fix round 1 is frozen for re-review.
