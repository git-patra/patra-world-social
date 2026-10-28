# T2 implementation report

Status: implementation ready for coordinator browser review and commit. No commits, subagents, server replacements, ledger updates, or original-source writes were performed by this worker. User's later game-only override is binding: coordinator removed portfolio/CV/gallery content and integrated the map destinations; those changes were preserved.

## Implemented interfaces and files

- `contracts/map.v1.json`: genuine 64×64 map with five non-overlapping district planning rectangles, eleven destinations, explicit rectangular building footprints, reachable entrances, spawn, and original-tree environmental collision coordinates. Core landmarks keep the original relative layout with +24 tile offsets. Tree collision is data shared by Go and TS, not client-only hidden geometry.
- `contracts/rules.v1.json`: version/checksum, 1024 subunits per tile, 4.5 tiles/sec / 4608 subunits/sec, hub 20Hz, race 30Hz, five race laps/slots, four penalty zones and 3000ms deadline. Values remain proposed beta defaults.
- JSON schemas for map, rules and command envelope; shared command and movement-unit golden JSON fixtures. Checksum is SHA-256 of canonical compact key-sorted JSON excluding checksum itself; definition documented in `contracts/README.md`.
- `apps/web/src/world/map.ts`: `loadMap(data): MapDefinition`, `collisionTiles`, `loadRules`; finite/bounded numbers, required fields, supported versions, unique IDs and collision-free spawn/entrance validation.
- `world/input.ts`: explicit `advanceMovement` and integer `toSubunits` seam; visual motion does not control simulation duration.
- `world/art.ts`: actual original procedural ghost drawing extracted to `drawAvatar(ctx, avatar, options, helpers)` with explicit state. Original building/tree/lantern/pumpkin drawing routines remain in compatibility controller, preserving original artwork.
- `world/renderer.ts`: `drawWorld(canvas,map,publicState,renderOptions)` merges explicit avatar states with world objects in one depth order. `drawMinimap`, `layoutNameplates`, `drawNameplates` provide district/spawn navigation overview and general collision-free avatar label lanes with leader lines.
- `world/camera.ts`: following camera, bounded .35–1.6 zoom.
- `apps/web/src/world.js`: backward-compatible `createWorld` API; actual expanded ground/paths, footprint collision, camera follow, zoom/recenter controls and location select. Adds `setPublicState`, `getState`, `setZoom`, `recenter`, `travelToLocation`. Existing travel/move/interact/pause/motion/reset/destroy remain. No reduced-motion teleport shortcut. New district portals navigate; existing game/UI behavior comes from coordinator's game-only integration.
- `apps/server/internal/world/map.go`: `LoadMap([]byte) (MapDefinition,error)`, `LoadRules`, collision and reachability helpers.
- `apps/server/internal/realtime/protocol.go`: exact command/event envelopes; `ValidateCommand` rejects invalid/missing envelope values, unsupported versions/types, nonfinite/invalid JSON, invalid bounded payloads and penalty zone outside 0–3. Known future command names enumerated; detailed engine states remain future tasks. Session/world authorization remains dispatcher work, not inferred from worldId.
- `apps/server/internal/contracts/json.go`: shared Go boundary validation helpers; Unicode text limits use code points consistently with TS.
- `apps/server/internal/games/model/{types,clock,rng}.go`: integer subunits, result version metadata, injectable clock and RNG interfaces. No assertion that JS/Go default random implementations match.
- `apps/web/scripts/test-map.mjs`: actual esbuild TS bundling and Node test execution, temporary files removed. Test command includes inherited suites plus new TS suite; tsconfig includes every new TS world module and the legacy controller.

## Red / green evidence

1. New TS fixtures initially failed with missing map JSON/map/input/avatars capabilities before those implementations existed. Inherited tests were retained. The old out-of-bounds goal changed from tile17 to tile64 because the map bounds intentionally expanded.
2. New Go map/protocol tests initially failed compilation for missing `LoadMap`, `LoadRules`, `ValidateCommand`.
3. Movement test caught `8.999999999999993` versus exact target9 at the two-second arrival boundary. Added epsilon only to arrival comparison; continuous speed remains 4.5 tiles/sec.
4. Actual-controller integration test runs routes through a mocked canvas/RAF and compares every sampled position/remaining route while toggling reduced motion mid-route. Both modes agree.
5. Shared Unicode fixture exposed Go byte-count rejection of a valid 1000-code-point chat payload; `go test -count=1 ./internal/realtime` failed before rune-count fix, then passed.
6. General co-located avatar label fixture failed for missing `layoutNameplates` before implementation, then passed with separate lanes.

## Final validation

- `make check`: PASS after final renderer/nameplate and Unicode changes: JS syntax, Go vet, TypeScript, **15 frontend tests** (7 inherited +8 TS), all Go suites, Vite production build, Go binary build.
- Explicit uncached final `go test -count=1 ./internal/world ./internal/realtime ./internal/games/model`: PASS. `-count=1` ensures external golden JSON is reread even when Go's ordinary test cache would reuse a result.
- Shared map fixtures prove all eleven entrances reachable and spawn unblocked. Projection round trips include fractional and expanded-map coordinates; map/protocol versions, missing fields, null zone, invalid zone and nonfinite TS values are rejected.
- Original source revalidation: `/Users/patra/Project/patra-world` remains HEAD `8d969212d91804e06225f5b1a75eed49e8c25307`, with existing untracked `.idea/` only. Worker wrote exclusively in destination repo and temporary script files.

## Visual review

Coordinator owns screenshots and final browser verification. Existing Vite and Go servers were not replaced.

- Main local view: `http://127.0.0.1:4176/`
- Explicit development-only overlap fixture: `http://127.0.0.1:4176/?fixture=avatars` — two avatars named **DEV fixture A/B** plus the local avatar. Only enabled under `import.meta.env.DEV`; production has no fake online avatars.
- Zoom controls bound camera scale; location selector walks the actual map paths to district entrances. Check core art, Race Park arrival, zoom, mobile HUD, and three-avatar sprite depth/nameplate lanes.
- Coordinator early smoke confirmed expanded ground/core art and sprite depth. Worker addressed reported minimap/top-tools overlap, removed duplicate YOU bubble, wrapped mobile camera controls above location HUD, and added general label lanes. Coordinator should refresh screenshot after latest lane change.

## Exact remaining limitations

- T2 establishes contracts/rendering/navigation seams; authenticated online transport, membership authorization, authoritative world tick ownership, reconnect, engine outcomes, persistent history, voice and social systems are later tasks. Local renderer remains clearly local solo.
- Map districts have real coordinates/footprints/entrances/paths, but expanded portals reuse incremental warm procedural building art. Complete district-specific scenery/game scenes are later art/game tasks.
- Public avatar labels avoid each other with upward lanes; viewport-edge clipping/crowd layout for a dense stack of twenty avatars is not yet optimized.
- Parser validates supplied checksum field shape/presence but does not recompute integrity inside synchronous loadMap/loadRules. Checksum generation definition is documented; asset/network integrity verification remains separate.
- Fixed-point units and shared fixture parity are established; full integer race integration/residual-carry policy belongs race-engine implementation. Hub local rendering simulation uses floating positions at 4.5 tiles/sec.
- Original environmental drawing helpers remain in compatibility controller except extracted avatar art; no wholesale stylistic rewrite was attempted. Camera/minimap/renderer/map/input/avatar TS modules are included in typecheck.
- No coordinator screenshot files were captured by this worker. Final browser signoff and Git commit remain coordinator-owned.

## Review fix round 1 — Important 1–3

All three blocking review findings addressed; changes remain unstaged and uncommitted against coordinator's reviewed index. User game-only cleanup preserved. Minor findings 4/5 remain coordinator-tracked outside this scoped round.

1. **Directional cancellation:** removed the `route.length && !destination` guard from `move()`. A legal directional input now replaces a click-to-walk route with the requested step. Actual-controller fixture dispatches canvas pointerdown/pointerup, proves a multi-tile route exists, then calls `move(-1,0)` and asserts route becomes exactly `[{x:31,y:33}]`.
2. **Semantic map boundaries:** TS loadMap flood-fills reachable cells once from spawn and rejects unreachable building entrances; Go LoadMap uses its existing Reachable helper for every entrance. Both reject intersecting district interiors, while touching district edges are allowed. New `contracts/maps.invalid.golden.json` contains the same disconnected-entrance and overlapping-district maps for both runtime tests. Existing valid map remains accepted and all eleven entrances reachable.
3. **Public event contracts:** new `contracts/event.schema.json`, TS `ServerEvent`/`eventTypes`/`validateEvent`, Go `ValidateEvent`, and `contracts/events.golden.json`. Required wire fields are exactly `{v,eventId,serverTick,serverTimeMs,type,payload}`. Both decoders enforce version1, nonempty bounded event IDs, nonnegative safe integer ticks/timestamps, enumerated public type, object payload, envelope keys and maximum1MiB encoding. Detailed game-event payload validation remains engine-owned. Fixtures include all17 public types, missing required fields, bad version/type, negative tick, null time and array payload; TS additionally tests NaN.

### Commands and observed red/green output

- Initial `npm run test --workspace apps/web -- map`: inherited7 tests passed, TS bundle failed `No matching export ... validateEvent`. Initial `go test -count=1 ./internal/world ./internal/realtime`: `undefined: ValidateEvent`; semantic fixture explicitly reported `accepted disconnectedEntrance` and `accepted overlappingDistricts`.
- After event contract implementation but before behavior fixes, repeated frontend command produced two real failed assertions: `direction replaces click-to-walk` (old22-step route retained versus requested single step) and `Missing expected exception` for semantic invalid map. Event fixtures passed.
- After controller and loader fixes, `make check`: PASS. Output: inherited7 frontend tests +10 TS tests = **17 frontend tests**, all Go suites, Go vet, TypeScript, Vite build and Go binary build pass.
- Final uncached `GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build go test -count=1 ./internal/world ./internal/realtime`: PASS (`world 0.272s`, `realtime 1.175s`). External JSON golden files therefore rerun without cache reuse.

Files changed in this scoped round: world.js, world/map.ts, world/avatars.ts, map.test.ts, server world/map.go +map_test.go, server realtime/protocol.go +protocol_test.go; added semantic invalid maps, event schema, event fixtures; this report appended. No screenshots or browser session replacements performed; coordinator handles visual signoff.
