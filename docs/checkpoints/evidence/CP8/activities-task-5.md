# Task 5 — Free ride and sterile race track

Status: implemented and frozen for independent review. No commits, push, process restarts, or changes to the original checkout. Root owns the registration edits, browser evidence, and eventual commit.

## Behavior and authority

- Each world owner initializes exactly two bicycles and two cars from `activities.v1.json`. Bicycles move at 6 tile/s, cars at 8 tile/s, with normalized diagonal input and 250 ms intent expiry. Four casual physics steps maximum per owner advance / local tick; the existing walking catch-up and 30 Hz race clocks retain their behavior.
- Idle vehicles have no integration work beyond the fixed pool's sleeping/expired checks. No vehicle timer, RAF, match, completion, history, or inventory record is created.
- `activity.ride` is the exact discriminated mount / drive / dismount command. All actions carry runtime incarnation, race-area generation, vehicle ID, and vehicle generation. Drive alone includes both `dx` and `dy` (including explicit zeros). The envelope sequence and admitted epoch are checked on the owner. Mount also requires a live non-playing member, an unoccupied vehicle, no existing ride, OPEN race area, and distance at most two tiles.
- Driving goes through the existing continuous movement transport budget; mount/dismount retain the separate discrete control budget. Walking input and destinations cannot move a rider or consume its authoritative movement sequence. Membership checks, takeover/reconnect, removal, idle/disconnect, and durable activity entry release rides. Voice is revoked on mount.
- Collision uses the exact race outer/inner geometry under the contract's uniform scene transform, with a 512-subunit casual radius and sweeps of at most 0.05 tile. Race itself retains its configured radius and all original constants/checksum. Avatar collisions/damage are absent.
- Dismount immediately stops held intent and returns the rider to the contract's safe perimeter `(45,49)`. Race reservation closes admission and resets vehicles/riders atomically before the existing three-second clear ACK, persisted start, and countdown. The first finisher does not release the area; the real race integration fixture now witnesses a partial finish and asserts that the race lease stays ACTIVE. Abort/release/rematch use the existing generation-fenced lifecycle.

## Files and seams

Backend:

- `internal/world/vehicles.go`: bounded pool, `Vehicle`, exact `CasualIntent` JSON union, `ApplyCasualIntent`, safe dismount/reset, shared-geometry containment, owner integration.
- `internal/world/runtime.go`, `movement.go`, `activity.go`: initialize/advance/release pool through the existing owner and reject overlapping walking control. The four-step catch-up counter affects vehicles only.
- `internal/world/areas.go`: vehicles join the snapshot cycle; reservation clears them atomically. `AreaPayload.MarshalJSON` emits either the existing five-field area payload or the single-field `{vehicles:[...]}` payload.
- `internal/trackgeom/track.go` and `track.v1.json`: dependency-free asset/containment extraction. The original race package's embedded track asset moved here; `games/race/track.go` reuses it. The byte-for-byte contract mirror/checksum test passes; no historical metadata changed.
- `internal/realtime/{activity_protocol,protocol,hub,areas,connection}.go`: exact validators, admitted command execution, lifecycle keys include kind, and drive-only movement-budget classification. Owner revision changes on mount/dismount/reset; movement stays replaceable state on the existing public snapshot cadence.

Contracts/frontend:

- `contracts/{activity.schema,protocol.schema,activities.golden,commands.golden}.json`: bounded vehicle union and exact ride command, with invalid extras, missing axes, IDs/kinds, duplicate/short pool, generation, and containment cases. Geometry/rider uniqueness are also enforced semantically by both runtime validators.
- `activities/contracts.ts`: area and vehicles discriminated union, strict snapshot and intent validation, shared race geometry for local simulation.
- `net/socket.ts`: full activity resync requires all four distinct area masks plus one vehicle pool. `world/avatars.ts` accepts only the exact ride command.
- `activities/vehicles.ts`: local guest pool and fixed-step movement under the same caps, expiry, containment, and four-step catch-up bound. `activities/store.ts` owns local/online rides, proximity, async input, state revisions, and independent vehicle incarnation ordering without replacing area masks. The production store reload uses the socket's strict full-resync method. Its existing 50 ms activity clock is reused.
- `activities/scene.ts`, `world/scenes/vehicles.ts`: lazy vehicle drawing on the existing world RAF. Riders replace their walking avatars; the shared scene drives the camera. Closed race areas hide casual vehicles. Existing race kart/standings rendering remains intact.
- `world/controller.ts`, `app.js`, `ActivityHUD.tsx`: contextual E/mount/dismount, safe local returns, touch controls preserving independent pointer ownership, dock label synced with the actual action, and Lintasan Mini location during rides. Product text uses “Cruise”, not internal speed units. `world/village-art.ts` removes the old three decorative cars.
- Shared `AudioService` provides deduplicated ride-entry effects; resync does not replay online ride audio.
- `app-shell.test.ts`: VM shell fixture now exposes the real imported activity store's snapshot/subscription and disposes each subscription; shell assertions remain unchanged.
- Root integration-only files: `apps/web/scripts/test-map.mjs` and `apps/web/tsconfig.json` register `tests/vehicles.test.ts`. Worker did not edit these files.

## RED evidence

Observed failures before their corresponding fixes:

1. `TestVehiclePoolIsOwnedByWorld`: `world snapshots omit the four free vehicles`.
2. `TestRideWireIsExact`: valid mount rejected as `invalid command`.
3. Frontend `vehicle_snapshot_is_exact_and_keeps_all_four_area_masks`: `Invalid activity snapshot`; after union support, the store lacked the four-vehicle state. Guest integration test also exposed the missing ride interaction seam before that seam was implemented.
4. `TestNeutralRideIntentSerializesExactDriveUnion`: neutral Go drive serialization omitted `dx`/`dy`, rejected by the exact wire validator. Custom union serialization now preserves explicit zero axes.
5. `TestVehicleCatchUpIsCappedAtFourSteps`: 250 ms backlog moved the car to ~45 instead of at most 44.6. Local `guest_vehicle_catchup_is_capped_at_four_steps` failed the same behavioral assertion. Both pass after adding a vehicle-only four-step cap.

Additional behavioral coverage includes exclusive rider ownership, one-second continuous bicycle/car speed, swept inner/outer containment, walk/ride overlap rejection, old epoch/incarnation/sequence/area generation, reserve-before-countdown clear, reconnect on closed track, actual-map slot reachability, safe dismount, revoked rider cleanup, 20 mount/dismount cycles, snapshot ordering/mask retention, local race handoff and immediate dismount, and replacing only the rider's walking actor.

## Commands and results

Pinned Go executable, from `apps/server`:

```
GOTOOLCHAIN=local GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build ../../.cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go test ./... -count=1
```

PASS across the full unit suite. Loopback listener tests required approved escalation. The initial unrestricted package attempt in the sandbox failed to bind `[::1]:0`; the approved run passed. An existing retained-runtime assertion initially interpreted the new vehicle snapshot as an area because both use `areaId:race`; it now explicitly checks `kind:area`, preserving the original assertion.

Focused Go commands:

```
go test ./internal/world ./internal/games/race -count=1
go test ./internal/realtime -run 'TestRide|TestNeutralRide' -count=1
go test ./internal/world -run 'TestRide|TestVehicle' -count=1
go test ./internal/world -count=1
```

All executed with the pinned executable/env above and PASS. The final world-package run includes the four-step catch-up correction. The realtime tests exercise actual HTTP/WebSocket command acknowledgement and twenty continuous drive intents in one control window, not a separate simulation helper.

Integration database was sourced/exported quietly, without printing credentials:

```
set -a
source ../../.cache/database.env
set +a
GOTOOLCHAIN=local GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build ../../.cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go test -tags=integration ./internal/application -run 'TestFreeRider|TestRace' -count=1
```

PASS (2.684 s). Covers real database membership, actual race start, spectator rider clear, race scheduler/transport/durability/timed return. After adding the explicit partial-finisher assertion, reran `-run TestRaceApplicationGridPhysicsDurabilityAndTimedReturn -count=1`: PASS (1.234 s).

Frontend from repository root:

- `npm run typecheck`: PASS.
- `npm test`: final PASS, 4 JS tests plus 160 bundled tests, zero failures/skips.
- `npm run lint`: PASS.
- `npm run build`: PASS. Final `index-CWwVt79M.js` 423.88 kB / 134.97 kB gzip; lazy vehicle renderer 0.79 kB / 0.48 kB gzip. Task 4 report's initial gzip measurement was 131.54; the approximately 3.43 reported-unit increase is well below the proposed +75 KiB gate. Vite reports kB; these are build measurements, not device FPS claims.
- `git diff --check`: PASS.

The final frontend rerun caught 21 VM shell tests failing with `ReferenceError: activities is not defined` after the dock began reading the store; the VM intentionally strips imports and needed the real imported binding added to its environment. That fixture/cleanup adjustment resolved all 21 failures without weakening assertions. The final package suite is green.

Initial simulation-only budget evidence:

```
go test ./internal/world -run '^$' -bench BenchmarkWorldTwentyAvatarTick -benchtime=1s -count=1
```

Apple M1: 67,276 ns/op, 0 B/op, 0 allocs/op. This was measured before the final four-step counter addition, after vehicle integration; it is not an end-to-end network/render/heap benchmark. Four fixed vehicles add no per-vehicle timers or unbounded pools. Browser memory/physical-device release gates remain broader checkpoint work.

## Browser handoff and evidence

Root ran browser acceptance independently and records it at `docs/checkpoints/evidence/CP8/activities-free-ride-browser.md`.

Observed by root: walk to `(45,45)`; mount car at `(45,46)`; D moves along the bottom lane and stops at the outer boundary; W follows the right lane; E returns `(45,49)`. Bicycle at `(39,46)` mounts with E, mobile 390×844 touch movement works with 12 px HUD/dock separation. Starting solo race while mounted replaces the ride with countdown and five karts; ending practice returns `(45,49)` and nearby rides become available again. Final retest confirmed “Mobil · Cruise”, contextual dock “Naik Mobil · E” / “Turun · E”, and Lintasan Mini retained at `(52,40)`.

Repeat paths for the reviewer: two bicycle slots `(39,46)/(41,46)`, two car slots `(43,46)/(45,46)`; approach within two tiles, E or “Naik · E”; WASD/arrows or four pointer buttons; E / “Turun · E”; Menu → Balapan → solo start for the handoff. The final four-step cap has automated owner/local coverage; no new browser layout changes followed the accepted UI retest.

## Self-review and limitations

- Checked race/vehicle import direction: world depends only on dependency-free track geometry, not the race engine. Shared asset remains byte-identical; five laps, five grid slots, BOT behavior, penalty, Duck, and durable history are unchanged.
- Checked independent kind keys, retired incarnations, area generation, vehicle revision, and same-tick mount/dismount ordering; vehicle state cannot overwrite the four area masks. Lifecycle remains ordered and continuous positions remain replaceable.
- Checked held-input neutralization on lease loss and existing modal/text/blur/disconnect router transitions. The existing two-pointer regression still passes; no input-router ownership logic was replaced.
- Checked safe returns and actual slot reachability against the real map. Idle, disconnect/reconnect, world exit, takeover, activity entry, and reservation remove rider ownership.
- No fake browser authentication/session or LiveKit acceptance was created. Authenticated browser/multiplayer/LiveKit user-deferred gates stay explicit; database/WebSocket integration coverage is real but does not substitute for those browser gates.
- No claim of final physical-device FPS, heap-after-GC, or network-load certification. Those remain performance checkpoint work. No new generic activity framework was introduced.

## Coordinator concurrency verification

After worker freeze, root added race-detector coverage for the newly shared vehicle owner/Hub paths using the pinned Go toolchain, exported private isolated database config, and approved loopback access:

```
go test -race -tags=integration ./internal/world ./internal/realtime ./internal/application -run 'TestRide|TestVehicle|TestFreeRider|TestNeutralRide' -count=1
```

PASS: world4.092s, realtime1.883s, application6.634s; exit0. No source changes followed or preceded this run beyond the already frozen report state. This focused concurrency check supplements the implementation's unit/DB runs; it does not claim authenticated browser acceptance.
