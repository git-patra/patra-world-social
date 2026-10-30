# Task 10 — Blaster solo/co-op implementation report

Status: implementation and fresh automated verification complete; uncommitted WIP preserved on `feat/world-activities`, base `d4e678f`. No commit, reset, discard, map relocation, or edit to `/Users/patra/Project/patra-world`. Root owns independent review and browser checkpoint documentation. Only CP1 has official PASS; this report does not promote a later checkpoint.

## Implementation

- Added versioned `contracts/blaster-rules.v1.json`, shared `blaster.golden.json`, identical embedded Go rules, Go engine/adapter/bounded bot AI, and deterministic TypeScript local engine/session.
- Intent-only server input: dx/dy/aim/fire plus envelope sequence. Strict wire validation rejects positions, claimed hits/kills, extra fields and invalid numeric ranges. Participant authority and stale-sequence guards remain server-owned; spectator input is rejected.
- Exact limits: 50 ms ticks (20 Hz), at most four catch-up steps, 250 ms intent expiry, 250 ms fire cooldown, 1500 ms projectile TTL, 180000 ms deadline, 3 HP, waves of 4/6/8 enemies, four human participants and 48 total projectiles. Player/bot/projectile speed is 0.8/0.12/1.2 normalized units per second. Swept collision prevents tunneling; arena bounds discard escaped shots; friendly shots do not hurt humans. Surviving teams revive eligible downed participants on a new wave; all-down, cleared wave 3 and deadline finalize once. A 15 s disconnect grace neutralizes intent immediately and later downs disconnected actors. Resume starts with neutral controls and resets per-session engine sequence.
- Added lobby registration/rules metadata, admission/rate/event delivery, activity ownership/lease integration, cooperative host departure, durable typed result serialization, history filters/details, guest history completion, explicit return and fresh mutual rematch.
- Blaster scene stays inside the approved map projection (1620,820–1900,1020). It uses the existing world canvas/loop with bounded player, bot and projectile draws, aim/crosshair, 120 ms hit flash, and an autumn perimeter/sign. No new rendering framework or accumulating particle effects. Existing walking actor is replaced for an active participant. Duck remains available in the centralized popup; podium/map assets/layout were not changed.
- HUD includes wave, enemy count, HP, countdown, result/storage status and return/rematch. Result countdown now freezes at `result.finishedAt`. Desktop Blaster HUD is limited to 320 px with one 44 px control/menu row; narrow/coarse-pointer layout keeps 72 px pads and a maximum width of 360 px. All controls keep the shared 44 px minimum. This CSS is scoped to Blaster.
- Independent move/aim pointer IDs, capture-loss/cancel release, WASD movement, arrow-key aim, Space fire, blur/hidden/disconnect/modal neutralization and fresh-press rearming use the shared input router. Sound transitions use the shared audio service, suppress resync/duplicate/stale cues, and stop on detach/clear/dispose.

## RED/GREEN evidence and provenance

Most implementation and tests were inherited as existing uncommitted WIP after an interrupted implementer. Their original RED runs are not available in this report and are not claimed as freshly observed. Existing code was preserved and inspected; the resumed work added direct edge, sound and worst-entity checks rather than restarting.

The timer regression was freshly observed RED before the production fix: `node scripts/test-map.mjs` ran 214 tests, 213 passed and `blaster result HUD freezes deadline display and spectator remains passive` failed because the rendered result displayed `0s` instead of `170s`. Evidence: `task-10-hud-red.log`. Changing the countdown reference from the live clock to `result.finishedAt` produced 214/214 GREEN, evidence `task-10-hud-green.log`.

New edge coverage added during continuation asserts exact 4/8/48 bounds, 49 ms versus 50 ms tick behavior, live friendly-shot pass-through, swept hit detection, no damage outside the arena, exact fire cooldown, scene draw bound and fresh-only SFX. These cover existing behavior and passed on first execution; they are verification additions, not claimed original TDD RED evidence. Compact HUD refinement follows an actual root browser screenshot showing the inherited shared flex sizing stretched the pads and obscured the arena. Root performs final visual verification.

## Fresh commands and outcomes

All commands below ran from `/Users/patra/Project/patra-world-social`, except the stated web script command. No environment secrets were printed.

1. `cd apps/web && node scripts/test-map.mjs` — timer RED 213/214, then GREEN 214/214, recorded in the HUD logs above.
2. `npm test` — PASS, 4 JavaScript tests plus 216 TypeScript tests, zero failed/skipped. Final run includes compact HUD JSX/CSS changes. Evidence `task-10-web.log`.
3. `npm run typecheck` — PASS (`tsc --noEmit`).
4. `npm run lint` — PASS (`node --check src/app.js && node --check src/world.js`).
5. `npm run build` — PASS; final build completed in 640 ms. Evidence `task-10-build.log`. Production build frozen after root was notified; root can reload the preview once without racing a rebuild.
6. `GOTOOLCHAIN=local GOCACHE="$PWD/.cache/go-build" "$PWD/.cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go" -C apps/server test ./internal/games/blaster ./internal/history -count=1 -bench BenchmarkWorstEntities -benchmem` — PASS. History emits its existing expected negative-path `authoritative result unavailable match_id=missing` diagnostic while passing.
7. With `.cache/database.env` sourced privately using `set -a`, `GOTOOLCHAIN=local GOCACHE="$PWD/.cache/go-build" "$PWD/.cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go" -C apps/server test -tags=integration ./... -count=1` — PASS across every tested package. This uses actual PostgreSQL schemas and local WebSocket/HTTP listeners, approved escalation, and synthetic users. Application suite completed in 60.163 s. Evidence `task-10-go-integration.log`.
8. `GOTOOLCHAIN=local GOCACHE="$PWD/.cache/go-build" "$PWD/.cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go" -C apps/server test ./internal/games/blaster -count=1 -bench BenchmarkWorstEntities -benchmem` — PASS. Evidence `task-10-benchmark.log`.
9. `GOTOOLCHAIN=local GOCACHE="$PWD/.cache/go-build" "$PWD/.cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go" -C apps/server test ./internal/games/blaster -count=1` — PASS (0.880 s), final focused model verification after adding the explicit friendly-shot/fixed-tick regression.
10. `git diff --check` — PASS. Go implementation/integration files formatted with bundled `gofmt`.

The actual transport test has timeout, natural all-down and host-leave/reconnect subcases. Each uses four human WebSocket clients plus a fifth spectator; asserts the fifth cannot join/input; reads snapshots via the wire; checks premature return rejection, actual session/epoch reconnect, team survival after host leave, durable human-only history, no spectator history/participant row, idempotent result hash, and fresh mutual rematch. Shared Go/TS goldens cover movement/expiry, idle all-down, four-human auto-aim victory across all three waves, and bounded deadline. The four-human victory is deterministic model/golden coverage; the real transport suite covers the other listed terminal paths.

## Worst-entity benchmark

The inherited benchmark allowed its bot/projectile populations to degrade while iterating. It was corrected so every measured iteration resets from exactly 4 players, 8 bots and 48 projectiles before one tick. Reset/copy overhead is included, making this a conservative model-step measure. It measures the engine step, not JSON transport, canvas rasterization, total browser frame time or physical mobile performance.

Apple M1 / darwin arm64 / Go 1.26.8: saved run `146145 iterations, 10579 ns/op, 0 B/op, 0 allocs/op`; an earlier fresh run was `161328 iterations, 6539 ns/op, 0 B/op, 0 allocs/op`. Concurrency/environment affect these microbenchmarks. Both are well below the 50 ms simulation interval. Scene regression asserts exactly 61 draw records at the cap: 4 players + 8 bots + 48 projectiles + 1 arena. No unbounded effects queue is introduced.

## Gaps and concerns

- Independent Task10 review remains a separate gate. Root confirmed final compact HUD browser checks: desktop 320×110 px with 44 px buttons; 390×844 proxy has a fitting 360 px HUD, 72 px active touch pads and no dock overlap. The countdown stays at 178 s after approximately 2 s idle completion. Root browser evidence/checklist is maintained separately; this report does not claim authenticated-browser co-op, physical simultaneous touch, or LiveKit validation.
- Root reported guest WASD/aim/fire/return/history working and idle all-down at approximately 2 s. That elapsed run time must not be confused with the later observation time (31 s); it is a short idle survival duration, not a proven 31 s match.
- Root confirmed final compact HUD geometry on the production build. The responsive proxy cannot establish physical dual-touch behavior.
- The scene renders simple bounded primitives and existing audio cues. There is no generated raster asset addition or separate particle pool; transient hit flash is derived from actor timestamps and projectiles are bounded simulation entities. Review this interpretation against the brief's asset/effect wording.
- Only the timer bug has freshly preserved RED→GREEN evidence. Broader inherited feature tests have fresh GREEN and inspected coverage, not a reconstructed preimplementation RED history.
- No commits created. No additional checkpoint PASS claimed.

## Review fix round 1 — independent mouse/keyboard intent

Reviewer finding: keyboard events replaced mouse fire with the current Space state, and world pointer-up/capture-loss called global release, clearing held keyboard movement. This fix is frontend-only; prior Go/full-suite evidence stands and those suites were not repeated. Existing history diagnostic noise remains deferred to Task11.

Changed `apps/web/src/activities/input-router.ts` to maintain independent mouse and keyboard fire booleans and OR them together with touch-pad fire. `releaseBlasterMouse()` clears only the mouse contribution. Global `release()` still clears mouse, keyboard, touch and remembered pressed keys for modal/blur/hidden/disconnect/owner-change interruption. Changed `apps/web/src/world/controller.ts` to track the active mouse pointer ID and end only that contribution on pointer-up, cancel and capture loss; unrelated pointer IDs and the implicit capture-loss event following a completed pointer-up cannot clear keyboard state. Pausing still globally neutralizes input.

Added regression coverage in `tests/blaster.test.ts` for mouse fire across every WASD down/up event, held Space across mouse release, and global modal interruption. Added production-controller event-path coverage in `tests/app-shell.test.ts` for pointerup, lostpointercapture and pointercancel while W/Space are held, continued held-key repeat, unrelated pointer IDs, implicit capture loss following pointer-up, mouse fire across keyboard changes, and pause neutralization. These execute actual controller listeners and the installed production input router; they do not merely call a mocked callback.

Reformatted `apps/web/src/features/blaster/local.ts` into readable type/class/method blocks using the installed TypeScript printer. A comparison against `task-10-fix-base/apps/web/src/features/blaster/local.ts`, transpiling to ES2022/ESNext with comments removed then normalizing JavaScript whitespace with installed esbuild, printed `PASS: local.ts canonical emitted JavaScript identical to task-10-fix-base`. Simulation logic is unchanged and the shared goldens pass.

Fresh evidence:

- `cd apps/web && node /private/tmp/task10-focused.mjs` — bundles and executes `tests/blaster.test.ts`, `tests/app-shell.test.ts`, `tests/map.test.ts`, `tests/activities.test.ts` using installed esbuild and `node --test`. RED: 93 tests, 89 passed, four failed. Failures were the mouse/WASD assertion (`false !== true`) and each controller release variant losing W (`0 !== -1`). Saved `task-10-fix-red.log`.
- The same command after the fix and final formatting — GREEN: `tests 93`, `pass 93`, `fail 0`, `skipped 0`, duration `503.404125 ms`. Saved `task-10-fix-green.log`.
- `npm run typecheck` — PASS, exit 0. Saved `task-10-fix-typecheck.log`.
- `npm run build` — PASS, exit 0, `built in 327ms`; final entry bundle `index-Bo1CryzD.js`. Saved `task-10-fix-build.log`. Root notified that the production build is stable; no further rebuilds planned before review/browser verification.
- `git diff --check` — PASS.

No commit, checkpoint promotion, backend/history behavior change, or broader test rerun in this fix round. Independent review and root browser verification remain root-owned.
