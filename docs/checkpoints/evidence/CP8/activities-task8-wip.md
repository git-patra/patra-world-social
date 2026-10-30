# Task 8 handoff snapshot — 2026-10-28

Implementation is frozen for the user's handoff request. No commit, push, service restart, source `patra-world` edit, new dependency, or subagent was performed by this worker. Root owns browser acceptance, docs/plan update and commit. This snapshot does not declare Task 8 accepted or complete.

## Implemented

- One persistent kinematic world cabinet at (41,22), entrance (41,24). Added a Capit Boneka map destination with the same standalone station pattern as pingpong; seven district counting is unchanged. Destination routing has a passing app-shell regression test. Latest production build includes it.
- Server-owned single operator lease of exactly 20,000 ms; entry proximity and epoch/sequence/incarnation/object generation fences; strict `activity.claw` command union with `machineId: claw-1`, `areaGeneration: 0`, control axes only for `claw.move`. No client position, prize or score fields.
- Shared asset constants: normalized grip coordinates [0,1]², speed 1 unit/s with diagonal normalization, tolerance 0.08 Euclidean radius, presets bear (0.28,0.28), bunny (0.72,0.30), fox (0.50,0.75). A center drop misses; six 50 ms up-left samples catch bear. Input expires after 250 ms. Existing 50 ms owner/store timestep and four-step catch-up cap are reused.
- Drop outcome is calculated from the owner's grip. Three 500 ms timestamped stages (dropping, lifting, releasing) show the same preset pickup/release to spectators. The plush returns to the cabinet and the transient prize clears after 1,500 ms. A monotonic dropId and generation fences reject duplicate drops. No payment, inventory, account reward, completion, history or leaderboard pipeline is introduced.
- Release/reset on disconnect/idle, logout/world removal, takeover/rejoin, lease expiry, and match join. Operator cannot walk, ride or kick while holding the cabinet. Existing ResumeCasual behavior is preserved.
- ActivityStore local/online cabinet stream, independent kind/incarnation/generation fences, bounded guest owner, persistent scene adapter on the world's existing RAF, compact HUD and occupied operator name, WASD/arrows + touch directional controls, Drop/Space, E/dock enter/exit. Existing input router handles modal/typing/hidden/blur neutralization. Simulation is owned by ActivityStore/world, not the details popup.
- Shared AudioService grab cue is deduplicated by incarnation/dropId and only plays fresh lifecycle events; full resync and stale/duplicate events are silent.
- Exact activity/protocol schemas and shared goldens extended; full socket resync now requires four area masks, one vehicle pool, one football and one claw (seven total). Existing vehicle scene count test now counts vehicle IDs explicitly.

## RED / GREEN

1. Added owner behavior tests first: `TestClawSingleOperatorTwentySecondLease`, `TestClawDropOutcomeUsesServerGripPosition`, `TestClawDuplicateDropDoesNotDuplicateReward`, `TestClawDisconnectOrMatchJoinResetsOperator`, `TestClawControlsBoundsExpiryAndFences`. Observed RED: missing cabinet snapshot. Implemented owner and observed GREEN for all tests/subcases.
2. Added client behavior/strict contract/audio/input tests in `apps/web/tests/claw.test.ts`. Observed RED: absent cabinet state/methods and unsupported command/snapshot. Implemented client path, then GREEN.
3. Added realtime tests `TestClawWireAndPublicStateAreExact`, `TestClawActualHubBusyAndTakeoverCleanup`. Observed RED: valid claw command rejected. Strict validators/Hub dispatch added, then GREEN.
4. Added `TestClawSharedContractRejectsUnboundedRules` and `TestClawWireUsesMachineIdentifier`. Observed RED for missing bounded-rule validation and dropped machineId during serialization; validation and distinct machineId serialization corrected, then GREEN.
5. Added `claw_map_destination_walks_to_cabinet_entrance_after_close`. Observed RED: map did not list Capit Boneka. Added standalone station/destination and close-safe routing, then GREEN.

## Verification executed

Pinned Go runtime: `.cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go`, `GOTOOLCHAIN=local`, `GOCACHE=$PWD/.cache/go-build`, repo root cwd. Database environment sourced quietly from ignored `.cache/database.env`.

- `npm --prefix apps/web test`: PASS — JavaScript world-math suite and all **185/185** bundled TypeScript/TSX tests, zero failures/skips. Final run log `/tmp/task8-web.log`.
- `npm --prefix apps/web run typecheck`: PASS.
- `npm --prefix apps/web run build`: PASS, 108 modules; final standalone claw scene chunk 1.58 kB / 0.82 kB gzip; latest build includes map destination. Log `/tmp/task8-build.log`.
- `go -C apps/server test ./internal/world ./internal/realtime -run 'TestClaw|TestActivityGolden' -count=1`: PASS.
- `go -C apps/server test ./... -count=1`: PASS, all tested packages. Run with loopback permission because transport tests use httptest listeners. The initial sandbox-only full realtime attempt failed to bind `[::1]:0`; approved rerun passed.
- `go -C apps/server test -tags integration ./internal/application -run '^TestClawActualDatabaseHasNoMatchResultOrHistoryWrites$' -count=1 -v`: **EXECUTED and PASS** against dedicated local PostgreSQL, not merely compiled. Uses a disposable migrated schema and real stored authentication/world membership. Performs 20 successful bear pickup/release cycles, rejects repeated Drop, compares spectator state, clears operator on actual pingpong lobby join and rejects subsequent claw input. Actual SQL counts were zero for matches, match_participants (personal history source), result_jobs, active_match_users, leaderboard_entries and champion_changes. Final passing integration runtime ~0.72 s (package ~1.665 s).
- `git diff --check`: PASS at freeze.

Integration harness setup errors during development were corrected before final PASS: the reused penaltyClock exposes `set`, not `add`; application Options required `Addr:127.0.0.1:0`. An earlier realtime takeover test omitted the required explicit `Takeover:true`; corrected and passed. Earlier vehicle scene assertion counted all non-football actors and therefore counted the new cabinet; narrowed to vehicle IDs and passed.

## Files / ownership

New files: `apps/server/internal/world/claw.go`, `apps/server/internal/world/claw_test.go`, `apps/server/internal/realtime/claw_test.go`, `apps/server/internal/application/claw_integration_test.go`, `apps/web/src/activities/claw.ts`, `apps/web/src/world/scenes/claw.ts`, `apps/web/tests/claw.test.ts`.

Existing integration touched: world activities/areas/activity/runtime/movement/vehicles/football; realtime activity_protocol/protocol/hub/areas; web contracts/store/scene/HUD/socket/avatars/controller/app/data; shared activities asset plus embedded copy, activity/protocol schemas and activity goldens; football/vehicles/app-shell tests.

Root added the new test registration in `apps/web/scripts/test-map.mjs` and `apps/web/tsconfig.json`; worker did not edit those files.

## Remaining acceptance / limits

- Root's real-browser checks are in progress and are not claimed by this worker. Authenticated two-browser acceptance, mobile/touch visual acceptance, real sound playback, and browser evidence remain root-owned. Unit/client tests cover logical spectator equality, neutralization, duplicate audio dispatch and local animation states, not physical device sound/real mobile interaction.
- No dedicated performance profiling or large-load claw benchmark has been run. Bounded simulation, existing timer/RAF ownership and 20 actual authenticated cycles are exercised, but this is not a measured performance acceptance result.
- User-deferred browser authentication/LiveKit acceptance remains deferred; no fabricated session or acceptance was used.
- Task 11's pre-existing own-ride-entry SFX freshness window is untouched (not part of claw changes).
- No unresolved failing automated test remains in this snapshot. Plan/docs review and local commit are pending root work after handoff.
