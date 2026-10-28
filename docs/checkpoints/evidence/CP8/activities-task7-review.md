### Spec Compliance

- ✅ Spec compliant for Task 7's implementation and automated verification: authoritative 20 Hz paddle/ball physics, explicit BOT consent, score/deadline/grace rules, participant-only input, shared goldens, persistent guest completion, world rendering, additive migration, typed history and spectator exclusion are present (`apps/server/internal/games/pingpong/engine.go:59`, `apps/server/internal/lobby/start.go:67`, `apps/server/internal/db/migrations/013_activity_match_types.sql:1`, `apps/web/src/features/pingpong/session.ts:7`, `apps/web/src/world/scenes/pingpong.ts:5`). All named files have corresponding changes; migration 004 and Duck/podium code are not changed by this diff.
- ✅ Actual DB/WS verification is now recorded as passing for BOT and two-human games, superseding the earlier infrastructure failures (`.superpowers/sdd/2026-10-28-world-activities-adjustment/task-7-report.md:114`). The integration test exercises real authenticated transport, forbidden spectator inputs, durable results, filtered private history, participant row counts, immutable retry and rematch (`apps/server/internal/application/pingpong_integration_test.go:23`).
- ⚠️ Authenticated browser multiplayer, LiveKit, physical mobile acceptance and release performance targets remain unverified/deferred; the guest browser record covers natural completion, retained prior history, menu continuity and a desktop HUD gap only (`docs/checkpoints/evidence/CP8/activities-pingpong-browser.md:5`). The controller still owns the Task 7 commit.

### Strengths

- Input is a finite bounded axis rather than client position/score/hit, is movement-budgeted at transport ingress, and is admitted through the existing roster/time fences (`apps/server/internal/realtime/protocol.go:64`, `apps/server/internal/realtime/connection.go:130`, `apps/server/internal/application/lobby.go:65`). Engine input sequences, fixed steps and intent expiry prevent command volume from increasing paddle speed (`apps/server/internal/games/pingpong/engine.go:71`).
- Go and TS share serve, miss/next-serve and paddle-bounce fixtures; additional tests cover score cap, deadline, grace boundaries, expired input and immutable result copies (`apps/server/internal/games/pingpong/golden_test.go:11`, `apps/server/internal/games/pingpong/engine_test.go:38`, `apps/web/tests/pingpong.test.ts:11`).
- The guest session is owned outside menus/HUDs and uses the existing application clock; completion retries retain one receipt identity. The server adapter retains one replaceable snapshot and bounds regular emission to every four physics frames (`apps/web/src/features/pingpong/session.ts:24`, `apps/server/internal/games/pingpong/adapter.go:27`).
- Spectators retain walking, receive no controls, and cannot create account history; stale snapshots cannot rewind a result (`apps/web/src/activities/store.ts:119`, `apps/web/src/activities/store.ts:250`, `apps/web/src/features/pingpong/PingpongHUD.tsx:21`, `apps/server/internal/application/pingpong_integration_test.go:187`).

### Issues

#### Critical (Must Fix)

- None found.

#### Important (Should Fix)

- None found.

#### Minor (Nice to Have)

- `apps/server/internal/games/pingpong/engine.go:79`, `apps/server/internal/games/pingpong/engine.go:103`, `apps/server/internal/games/pingpong/bot.go:3`: Go hashes the rules JSON but repeats the runtime constants as literals, while TS reads several values from JSON. The present values agree, but future edits can change runtime rules without changing the frozen checksum, or change the checksum without changing Go behavior. Parse the embedded rules into a typed immutable configuration, or add a focused contract assertion covering every repeated constant (`contracts/pingpong-rules.v1.json:1`, `apps/server/internal/games/pingpong/golden_test.go:52`).

### Checks Performed

- Read the supplied complete diff, recovering only output ranges omitted by tool truncation; did not regenerate the diff, crawl the repository, run git commands or rerun tests.
- Named risk — a public snapshot during preparation could dereference an unstarted engine: checked `ReadEngine` and the start publication boundary. `Start` precedes assignment to the readable lobby engine; the callback remains bound to its world/match (`apps/server/internal/lobby/game.go:18`, `apps/server/internal/lobby/start.go:247`).
- Named risk — the new continuous control could bypass roster authority or leak held intent across focus/mode changes: checked the existing `InputEngine` and shared input router. Admission requires a connected non-departed roster member; owner changes and modal/text/hidden/blur/pointer release clear intent (`apps/server/internal/lobby/game.go:109`, `apps/web/src/activities/input-router.ts:28`, `apps/web/src/activities/input-router.ts:106`).
- Named risk — new local activity exit could omit its safe return or alter the seven-district count: checked the unchanged `action`/controller transition and visit callback where diff context was incomplete. Local clear restores the activity contract safe return; visitation remains based on `villageLocation` (`apps/web/src/activities/store.ts:642`, `apps/web/src/world/controller.ts:139`, `apps/web/src/app.js:58`).
- Read the implementer's final clean test/build/typecheck/lint evidence and final real DB/WS PASS output; no source/test modifications were reported after the review snapshot (`.superpowers/sdd/2026-10-28-world-activities-adjustment/task-7-report.md:65`, `.superpowers/sdd/2026-10-28-world-activities-adjustment/task-7-report.md:114`).

### Assessment

**Task quality:** Approved.

**Reasoning:** The implementation follows the existing ownership, durability and input paths, with meaningful golden, lifecycle and actual DB/WS tests. No blocking defect was found; the minor rules-source drift risk and explicitly deferred browser/device release gates do not invalidate this task's source approval.
