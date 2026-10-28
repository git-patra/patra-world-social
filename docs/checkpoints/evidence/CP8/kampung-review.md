# Kampung Senja integration review

Reviewed the working changes since `b7ae4bd`, with controller/event lifecycle and shared geometry as the primary scope. Review performed without browser activity or production source edits. At the coordinator's request, added `apps/web/tests/app-shell.test.ts` and registered it in `apps/web/scripts/test-map.mjs` after reproducing the findings.

## Findings and disposition

### P2 — Map dialog close cancels the online destination

**Resolved and independently verified.** Map travel now runs in an animation frame after dialog mutation cleanup, and rechecks paused state before dispatch.

- Location: `apps/web/src/app.js:79`, with `app.js:142`, `apps/web/src/world/controller.ts:64`, and `apps/server/internal/world/movement.go:24` establishing the cancellation path.
- Trigger: join a private world, open Map, select any remote destination.
- The click handler closes the map and immediately sends the destination. The `open` MutationObserver subsequently runs `releaseInputs()`, sending `player.input(0,0)`. The server clears the path for every player input, including zero. The selected route therefore never persists.
- Reproduction with actual app handlers and actual controller in a minimal DOM fixture: commands before the observer are `[["destination",45,45]]`; after it runs they are `[["destination",45,45],["input",0,0]]`.
- Fix recommendation: dispatch the destination after close cleanup, or distinguish modal opening from closing while preserving held-input cleanup.
- Regression: `actual Map selection retains its online destination after dialog close cleanup` checks the resulting command order and verifies no optimistic online movement.

### P2 — Semantic direction-button activation leaves solo movement running

**Resolved and independently verified.** Semantic activation now uses `PointerHold.pulse()` with a 180 ms timeout; normal release also clears the pulse timeout.

- Location: `apps/web/src/app.js:117`, with `app.js:110` and `apps/web/src/world/controller.ts:73` establishing the missing release path.
- Trigger: with the movement pad visible, focus a direction button and activate it using Enter, Space, or an assistive-technology click (`detail === 0`).
- The retained click handler calls `world.move()`, which changed from one-cell stepping to continuous movement. Enter/Space release is ignored by the directional keyup listener, and `PointerHold` owns no hold to release. The solo avatar continues until collision or another action stops it.
- Reproduction: a right-button semantic click followed by Enter keyup moves from x=32 to x=34.475 after 550 ms. The regression fixture likewise continues from x=33.35 to x=35.6 after the release.
- Fix recommendation: make semantic clicks a bounded movement action with explicit cleanup.
- Regressions: separate Enter and Space cases verify movement happens and subsequently stops.

### P2 — Foreground culling removes still-visible wide structures

**Resolved and independently verified.** The renderer now calls `visibleVillageProp()` with a conservative 400-art-pixel margin scaled by zoom, covering the Warkop's largest extent. The focused regression passes at zoom 0.65, 1, and 1.5 while rejecting a truly offscreen prop.

- Location: `apps/web/src/world/controller.ts:45`; Warkop drawing extents are established in `apps/web/src/world/village-art.ts:243` onward.
- Trigger: a 390 px viewport at zoom 1, with the player at logical `(26,19)`. Its art position is `(1060,460)` and camera x is -865.
- The Warkop foreground object is anchored at x=624, yielding screen x=-241; the universal 220 px horizontal culling margin removes it. Its artwork extends to x=974, so the rightmost 109 px of the counter/canopy are still in view. That visible part disappears even though its collision remains. Larger zoom makes the fixed screen margin proportionally smaller.
- The coordinator applied the recommended conservative zoom-scaled bound after this report; the production draw filter calls the tested helper.

## Verified strengths

- The controller reads the online player's location from the authoritative snapshot, accepts snapshots while paused, and retains the solo position across attachment transitions. Reset does not teleport an attached player.
- Arrival updates location/nearby state without calling `onSelect`; explicit interaction is required to open an activity.
- Existing game, auth, account settings, history, friends, invitations/worlds, podium, chat, and voice mounts remain wired to their established feature modules.
- Production rendering uses the local avatar or snapshot avatars; no demo NPC population is inserted.
- All 169 runtime art solids exactly match the checked-in layout. Shared and Go-embedded map/voice JSON files are byte-identical. The canonical checksum is `71814ab8a87be0bd7166cbe2c1049f38e6fa774825ca87230fa0b155b0bb0768`.
- Flood-fill verification found 2,129 reachable cells including all eight destination entries (seven areas plus legacy arcade) and both voice-table approaches.
- Fractional and edge coordinate projection round trips pass.

## Validation

- Before adding event regressions: `node scripts/test-map.mjs` passed 94/94.
- RED after adding regressions, before fixes: the same runner reported 98 tests, 95 passed, 3 failed. Failures correspond exactly to the Map ordering and Enter/Space continuous-input bugs. The new art/projection parity test passed.
- `npm run typecheck` passed with the added tests.
- After the coordinator's input fixes, the focused `app-shell.test.ts` bundle passed all 4/4 tests independently.
- After the culling fix, the focused `large Warkop canopy stays visible at mobile viewport edges at every zoom` test passed 1/1 independently.
- Final small UI changes reviewed: offline lobby availability text now reports offline instead of loading; penalty zone numbering changes only visible text from 01–04 to 1–4 and retains accessible labels and zero-based engine input.

## Deferred / not judged

- Responsive visual layout and real browser focus behavior: coordinator is actively checking these; this review did not use the browser to avoid interference.
- External account/browser authentication acceptance and LiveKit staging: explicitly deferred in the task.
- Pre-existing T14/T16 defects or acceptance gates: excluded unless introduced by this patch; no claim of their completion is made here.

## Assessment

All three findings are resolved with focused regressions passing. No remaining blocking correctness findings were identified in the reviewed integration. Ready from this code-review scope, subject to the coordinator's complete validation and browser checks. External-service acceptance and pre-existing T14/T16 gates retain the limitations above.
