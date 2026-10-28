# Task 2 review

Base: `6d040e19f4058e714e97545a06e57185a07d6362`. Reviewed the Task 2 brief, implementation report, staged review package, changed implementation and tests, and browser evidence. I did not change code or the Git index and did not rerun the already reported passing suites.

## Spec verdict

**Needs changes before Task 2 signoff.** The 64×64 shared map, original Canvas art, footprint collision, camera/zoom/minimap, multi-avatar rendering seam, rules units, Go/TS command decoders, golden fixtures, and game-only content cleanup are substantially present. The directional cancellation requirement is broken in the working controller. The map loader also accepts semantically invalid maps, and the event side of the stated protocol is only a Go struct.

## Findings

### Important

1. **Directional input cannot cancel click-to-walk** — `apps/web/src/world.js:417`. A canvas click calls `startRoute(path)` without an ID, leaving `destination === null`. While that route has remaining tiles, `move()` returns at line 417, so WASD, arrow keys, and touch-pad directions cannot cancel it. The network spec explicitly says the player can cancel click-to-walk with directional input. Remove this guard and replace the route with the requested legal step, including when a route has no station destination.

2. **Map loaders accept disconnected entrances and overlapping districts** — `apps/web/src/world/map.ts:44-55`, `apps/server/internal/world/map.go:89-97`. Both loaders check only that spawn and entrances are unblocked. Neither checks that each entrance is reachable from spawn or that district rectangles do not overlap. The supplied map passes fixture reachability, but a modified map can be accepted by both runtimes even when it violates the map's required connected paths and nonoverlapping district layout. Add semantic checks at the load boundary, with matching Go/TS invalid fixtures.

3. **Event envelope has no schema or decoder** — `contracts/protocol.schema.json:1-18`, `apps/server/internal/realtime/protocol.go:17-24`. The protocol spec defines an event envelope and enumerates public event types. The JSON schema covers client commands only; `Event` is an unconstrained Go struct, and the browser has neither an event type/validator nor shared event fixtures. This leaves the server-to-browser half of the versioned protocol without a contract and permits future public events to drift. Define an event schema/type set and validate the required envelope fields, even if detailed game event payloads arrive in later tasks.

### Minor

4. **A former business sign remains in the repurposed Balai Desa art** — `apps/web/src/world.js:242` (also line 304). The user requested removal of promotional business showcases while preserving and repurposing original buildings. The Balai Desa still displays `CONNECT` and a prominent `P` coin emblem; the screenshot shows the old sign. Replace those marks with village or game-world details while keeping the building's original shape and palette.

5. **Zoom API accepts nonfinite values** — `apps/web/src/world/camera.ts:2`, `apps/web/src/world.js:499`. `clampZoom(NaN)` returns `NaN`, so the public `setZoom` seam can poison camera transforms. Reject nonfinite input or keep the current finite zoom.

## Strengths and quality verdict

The shared fixtures exercise real TS and Go parsers, including invalid protocol versions, missing values, nonfinite TS values and penalty bounds. The motion integration test uses the actual world controller and compares route state after toggling motion. The renderer merges objects and avatars in depth order; the three-avatar screenshot labels its additional characters as development fixtures. The game-only copy and removal of resume/public business assets align with the user override. **Code quality: good foundation, with the Important behavior and contract gaps above to close.**

## Cannot verify from this review package

- Real desktop/mobile frame rate and dense 20-avatar nameplate behavior; the screenshots show three avatars and mobile emulation, not device benchmarks.
- Runtime network authorization, snapshots, replay/reconnect, race/penalty/duck outcomes and official history, which are outside Task 2 implementation scope.
- Checksum integrity at ingestion: `contracts/README.md:5` explicitly delegates hash verification to a later transport/asset boundary; the current loaders validate presence/shape but do not recompute the hash. Ensure that future ingestion actually verifies it before accepting remote map/rules data.
