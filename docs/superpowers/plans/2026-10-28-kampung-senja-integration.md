# Kampung Senja Fullscreen Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Apply the approved outdoor Kampung Senja design to the real social game, fullscreen without the old sidebar, preserving all existing games and multiplayer features, and include this plan in local Git commits.

**Architecture:** Replace the isometric presentation with the approved cached top-down canvas art and a compact fullscreen HUD. Keep the existing application mounts, online socket/controller and authoritative movement interfaces. Generate one shared 64×64 navigation contract for the new physical layout, with a new map identity/checksum and matching Go embedded assets and voice-table contract; existing persisted match versions are never rewritten.

**Tech Stack:** Existing Vite/JS/TypeScript/React frontend, procedural Canvas 2D, Go backend, existing Node/Go test runners. No new dependencies.

**Spec:** `design/exploration-demo/DESIGN.md`, approved by user on 28 October 2026 with explicit instruction to implement and preserve games/multiplayer.

## Global Constraints

- Use existing `feat/social-world` checkout; source `/Users/patra/Project/patra-world` unchanged.
- Fullscreen game world, no CV content and no permanent old sidebar.
- User steering: semi-English UI labels (Explore, Map, Friends, Settings, Play solo); Indonesian place names and clear Indonesian explanations/errors remain.
- Retain penalty (including legacy practice), race five laps/five slots, Duck Arcade, podium, history, settings, accounts, friends, invitations, moderation, world/table chat and voice controls.
- Private worlds remain server-authoritative; never synthesize online players or local teleport while connected. Demo NPCs/pretend social replies are not production multiplayer.
- Preserve 64×64 logical coordinates, schema v1, 1024 subunits and current movement speed. New map ID `kampung-senja-v1` distinguishes geometry; map schema version is not an asset revision.
- Project art remains 2560×1920. Uniform 40px logical tiles use `screenX=(x+.5)*40`, `screenY=(y+.5)*40-320`; unused northern/southern logical rows are blocked. Spawn stays `(32,32)` in the open plaza.
- Account browser acceptance is still deferred; no credentials/session injection. LiveKit staging remains unavailable. T14 review, T16 and unpassed checkpoints are not implicitly accepted by this design change.
- Commit plan, accepted design, implementation, tests and evidence locally. Latest user authorization: create a public personal GitHub repository and push the verified tree to `main`. Publishing does not authorize deployment.

## Review Focus

- A modal, text field, blur or released touch must stop directional movement without stopping authoritative snapshots (Task 2 controller regression).
- Arrival at a destination must keep the world visible; only explicit interaction opens a game/voice panel (Task 2 real controller test).
- New map assets and voice zones must agree in JS and Go, remain reachable and reject old snapshot checksum (Task 1 parity/reachability tests).
- Solo → online → solo must preserve local state, clear stale inputs and avoid optimistic online movement or reset teleport (Task 2 actual controller tests).
- Small viewport HUD and overlays must keep every existing feature reachable without clipping game dialogs, and preserve game lifecycle/preferences (Task 3 browser + existing engine/history/settings suites).

### Task 1: Shared navigation and approved art

**Files:** create `contracts/kampung-senja-layout.v1.json`, `scripts/generate-kampung-senja.mjs`, `apps/web/src/world/village-art.ts`, `apps/web/src/world/village.ts`; update `contracts/map.v1.json`, `contracts/voice-tables.v1.json`, embedded Go copies, active snapshot fixtures, map/voice tests.

**Interfaces:** `projectVillage(point)` / `unprojectVillage(point)` convert logical and art coordinates. Layout contains physical solids, seven exploration destinations plus legacy practice access, and two voice approaches. Demo-only NPCs/collectibles are excluded from the production runtime. Map remains accepted by `loadMap`/`world.LoadMap`; legacy game IDs `arcade`, `penalty`, `race`, `duck`, `warkop` retain their behavior.

- [x] Write navigation regression: all seven destinations and both voice approaches reachable from spawn; new map identity/checksum/embedded copies match; top-down fractional coordinates round-trip.
- [x] Run targeted test and observe old map fails new geometry expectation.
- [x] Copy approved procedural art into runtime; store reviewed solid geometry separately and generate collision tiles, activity fixture footprints, map identity/hash and voice positions. Keep open decks and paths walkable; decorative area destinations do not become blocking building footprints.
- [x] Verify JS/Go asset checksums, reachability and shared voice geometry. Adjust tests tied to old scenery rather than weakening collision assertions.
- [x] Include Task 1 with the integrated implementation commit after Tasks 2–3 so no intermediate app ships a mismatched renderer/map.

### Task 2: Real world controller and fullscreen shell

**Files:** replace `apps/web/src/world.js` rendering/controller internals, update `apps/web/index.html`, `apps/web/src/app.js`, `apps/web/src/world-shell.css`, add presentation helper tests in `apps/web/tests/map.test.ts` (or dedicated registered suite).

**Interfaces:** preserve `createWorld(canvas, {stations,onNear,onSelect,onMove})` and all methods consumed by `mountOnline`/preferences: `move`, `stopInput`, `travelTo`, `interact`, `setOnline`, `setOnlineSnapshot`, `setPaused`, `setMotion`, `setLocalIdentity`, `getState`, `reset`, `recenter`, `setZoom`, `destroy`. Online direction uses top-down axes, destinations remain logical integers. `onSelect` only fires for explicit interact. New input handling tracks directions continuously and sends zero on release.

- [x] Add controller regressions for no auto-panel arrival, continuous direction/release, online axis mapping, no reset teleport, paused snapshot updates and mode restoration; run RED.
- [x] Implement cached background, culled/depth-sorted foreground, real avatars with readable nameplates, camera follow/zoom/minimap and destination routing. Honor reduced-motion without affecting movement distance. Production contains no demo NPCs.
- [x] Replace old permanent guide/nav with fullscreen canvas, compact world status, map/destination dialog, context action and dock. Move secondary features into a native menu dialog while retaining unique IDs and all feature mounts.
- [x] Make destination selection route to the location; explicit action opens actual game/voice panels. Keep direct game access in the menu. Keep mobile direction buttons, chat typing guards and semantic controls.
- [x] Run controller/feature suites and typecheck; resolve actual integration regressions.

### Task 3: Verification, documentation and commit

**Files:** relevant existing guest/e2e coverage where selectors changed; `docs/checkpoints/evidence/CP8/kampung-senja-integration.md`; `docs/PROGRESS.md`; `docs/HANDOFF.md`; accepted demo design/README and this plan.

- [x] Run frontend tests, typecheck, lint, production build and Go tests/vet/build with the pinned project Go1.26.8 toolchain; run affected integration suites if project database available, recording exact limitations.
- [x] Start/reuse only the project local frontend/backend as necessary. Browser checks: desktop and 390px/768px responsive layout, click/keyboard/touch movement, context action, every menu entry, all three real guest games and history/settings availability. No credential entry or fake sessions.
- [x] Fresh reviewer checks changes since `b7ae4bd`, with special attention to preserved feature mounts, movement mode transitions, map/server parity and responsive accessibility. Fix important findings with focused regressions.
- [x] Save evidence and update docs honestly: user-approved integration is complete only after checks; T14/T16 and external acceptance remain separate.
- [x] Commit all intended files including this plan and approved design; verify `git show --stat HEAD`, plan membership and clean working tree. Report real SHA, validation and remaining external gates.

## Execution notes

- 28 October: user explicitly approved design and requested implementation + committed plan. Proceeding directly with native execution; no repeat approval gate. The frontend/backend map change and renderer are one atomic integration commit to prevent incompatible intermediate runtime states.
- No new worktree: current branch already isolates this product and the approved demo is in its working tree. No unrelated tracked changes at start; only the demo folder is untracked.


### Publication follow-up

- [x] Resolve intended personal account: user selected `git-patra` on28 October2026. Account existence verified. CLI is currently authenticated as `patra-digital`; in-app GitHub browser has no signed-in session.
- [ ] Create public `patra-world-social`, publish a clean snapshot as `main`, verify remote default branch/tree and repo visibility. Preserve existing local `feat/social-world` history; do not publish old resume/PDF history.

### Verification record

- Frontend103 tests, typecheck, lint and build PASS; Go full race/integration242 top-level tests across20 packages, vet/build PASS.
- Three actual guest games completed; results persisted after reload. Existing account/social/voice gates and menu entries remain reachable. Responsive390/768 layouts checked, with physical-device and authenticated acceptance still deferred.
- Fresh review: Map-close cancellation, semantic direction release and zoom-aware culling fixed and rechecked. See CP8 evidence and review reports.
- Obsolete CV screenshots omitted from current/public tree; inherited history retained locally. Commit result is reported via Git/final response rather than a self-referential SHA.

- Publication preparation: clean local `main` snapshot uses the verified game tree and this plan/evidence, without parents from the inherited CV history. Remote creation/push remains pending authentication for `git-patra`; no repo under another owner is authorized.
