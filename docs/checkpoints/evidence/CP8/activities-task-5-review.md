# Task 5 review

## Spec Compliance

- ✅ Spec compliant for the implementation visible in the frozen review package; no Important/Critical finding. Owner fencing, one rider per vehicle, shared containment, safe returns, race sterilization, local guest integration and continuous-command budgeting are implemented.
- ⚠️ Cannot verify authenticated multiplayer browser acceptance, LiveKit provider behavior, physical multitouch, measured dismount latency under network delay, or final device FPS/heap/network budgets. These remain explicit controller gates; local Chromium evidence at `docs/checkpoints/evidence/CP8/activities-free-ride-browser.md:3` does not claim them.
- ⚠️ The existing race renderer/HUD, five-lap/five-slot rules, BOT behavior, abort/rematch lifecycle, durable history and other activities mostly reside in unchanged code. This task preserves their integration and adds partial-finish coverage (`apps/server/internal/application/race_integration_test.go:149`); the reported affected suites and prior-task evidence remain the gates for those unchanged behaviors. No unnecessary race-scene rewrite is required given task-5-context's explicit statement that its transform/art are already correct.

## Strengths

- `apps/server/internal/world/vehicles.go:102` admits only current owner epoch/sequence/incarnation, current race-area generation, current object generation, OPEN track and a nearby live non-playing player. Mount changes object generation; dismount clears controls and returns to the safe anchor (`:74`). Walking cannot consume the rider's movement sequence (`apps/server/internal/world/movement.go:18`). Runtime join/remove/idle and durable activity entry release ownership (`apps/server/internal/world/runtime.go:129`, `:144`, `:148`; `apps/server/internal/world/activity.go:36`).
- `apps/server/internal/world/areas.go:86` resets riders and objects in the same owner operation that closes the race mask. The real application fixture checks cleared spectators and safe perimeter (`apps/server/internal/application/vehicles_integration_test.go:17`); the actual race fixture now requires a partial finish and proves it does not reopen the track (`apps/server/internal/application/race_integration_test.go:149`).
- `apps/server/internal/world/vehicles.go:168` applies normalized 6/8 tile/s movement, expiry and bounded swept containment. `apps/server/internal/world/runtime.go:115` caps vehicle catch-up at four steps without changing walking/race scheduling. Equivalent local logic lives at `apps/web/src/activities/vehicles.ts:46`; focused cap tests exist at `apps/server/internal/world/vehicles_test.go:205` and `apps/web/tests/vehicles.test.ts:41`.
- `apps/server/internal/realtime/connection.go:130` classifies drive as continuous movement while mount/dismount retain the discrete budget; a real WebSocket fixture sends twenty drive commands in one control window (`apps/server/internal/realtime/vehicles_test.go:143`). The exact JSON union retains neutral axes and rejects client positions/scores.
- `apps/web/src/net/socket.ts:166` requires all four distinct masks plus the vehicle pool. Vehicle revisions and retired incarnations are independent from area masks (`apps/web/src/activities/store.ts:310`); lease loss neutralizes held input. Local race startup clears riders synchronously before lazy import/countdown (`:424`). The same input router, activity clock and world renderer serve rides (`:543`, `:574`, `:578`), with no new activity timer, fake match or persistence pipeline.
- `apps/web/src/activities/scene.ts:57` hides vehicles on closed tracks and replaces only mounted walking actors (`:77`); the controller preserves safe guest returns (`apps/web/src/world/controller.ts:126`). Browser evidence covers keyboard movement, bounds, E dismount, touch buttons and ride-to-race handoff.

## Issues

### Critical

- None found.

### Important

- None found.

### Minor

- `apps/web/src/activities/store.ts:270`: the online entry cue fires whenever the pool revision changes and the current user is still mounted. Another player mounting/dismounting increments that global revision, so the current rider hears a fresh entry cue despite no own ride transition. Compare the previous/current own vehicle ID and object generation before emitting, and key the event to that own mount transition. This does not affect authority or movement.

## Checks and boundaries

- Read the frozen diff in bounded passes; re-read only the validator excerpt that tool output truncated. Used the diff's embedded old/new asset bodies to verify byte-identical track relocation: SHA-256 `4d1c1cee92ab8dc492d0211fc57e2eae5b5852440edfa2567e64034f38a0a12e`. The shared `Driveable` expression preserves the original integer boundary comparisons for race inputs (`apps/server/internal/trackgeom/track.go:16`).
- Focused external-code check: membership admission and voice re-entry risk after adding mount. Checked the surrounding Submit admission, `realtime/voice.go`, application voice wiring and voice table guards; Submit revalidates membership, mount revokes voice, and table proximity remains required. No broader repository crawl.
- Focused external-code check: duplicate-avatar/camera risk and keyboard/touch neutralization. Read the cut-off controller draw context plus the unchanged input router; actor filtering, shared RAF, pointer ownership and owner/modal/blur release are present.
- Did not rerun reported suites. Reviewed the implementer's Go/frontend/build/integration evidence and the controller's additional focused race-detector result: `go test -race -tags=integration ./internal/world ./internal/realtime ./internal/application -run 'TestRide|TestVehicle|TestFreeRider|TestNeutralRide' -count=1`, PASS for world 4.092 s, realtime 1.883 s, application 6.634 s, exit 0. This is reported controller evidence, not an independently repeated run.

## Assessment

**Task quality: Approved.** The authority boundaries and lifecycle separation are coherent and exercised through owner, transport and real application fixtures. The minor audio transition issue is non-blocking; deferred browser/provider/device gates must remain explicit.
