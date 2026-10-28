# World activities baseline — 28 October 2026

Execution authorized by user, covering the six published stages (plan Tasks0–11). Base commit: `3079f05`. Work branch: `feat/world-activities`.

## Frontend baseline

Before product edits: `npm test` passed 4 JS +99 TS tests; `npm run typecheck` and `npm run build` passed. Production entry:425.44 kB /136.12 kB gzip; CSS71.20 kB /16.72 kB gzip; lazy LiveKit398.26 kB /102.58 kB gzip. Vite reports decimal kB, retained here unchanged.

Synthetic fixture: `apps/web/tests/activities-performance.html` with20 deterministic avatars and1280×720 CSS canvas. An immutable tracked source copy of3079f05 was served locally on4179, with only the fixture added. No accounts, credentials, WebSocket, or API requests. CUA in-app Chromium156, macOS user agent, DPR1;15,005ms sample:

| Measure | Observed |
|---|---:|
| RAF samples / observed cadence |2692 /179.34fps|
| Callback CPU p50 / p95 / p99 |0.7 /0.9 /1.3ms|
| JS used heap at finish |9,785,147bytes|

This browser's uncapped cadence is not physical-display/mobile acceptance. CPU samples wrap the world RAF callback (render+simulation), not full browser compositing. Heap is a point sample without forced GC and excludes canvas GPU memory. Static world cache2560×1920 RGBA is approximately18.75MiB. Baseline fixture itself has no sockets; incoming network/owner tick must be measured in the later multiplayer load test. No claim of20 authenticated browser players.

## Prerequisites

- Fresh scoped T14 reviewer found R1/R2 addressed, no actionable new finding; independently ran13 settings tests, all passed. This accepts only these repairs, not CP7/beta/browser/LiveKit acceptance.
- Added `TestQueuedReconnectPreservesEveryTransition`: failed baseline because disconnect/resume/disconnect collapsed to one notification. Ordered per-user mailbox preserves transitions, deduplicates repeated identical states, caps32 entries and fails closed with Leave on overflow. Lobby/race/penalty/Duck packages pass `go test -race` after fix.
- Initial integration attempt lacked exported DATABASE_URL; second was blocked by loopback sandbox. These are setup failures, not product failures. Subsequent approved loopback runs use isolated random schemas; results recorded with final verification.
- Initial Persist/startup enumeration regression is tracked separately in Task0 startup report and will be included in the prerequisite commit after verification.

Authenticated browser workflows and LiveKit staging remain deferred. Physical mobile checks remain NOT_RUN until an actual device is available. CP1 remains the only official PASS.

Simulation-only Go baseline on Apple M1: `BenchmarkWorldTwentyAvatarTick`, 37,644 iterations, 67,057 ns/op, 0 B/op, 0 allocs/op. This is mean CPU cost for 20 avatars at the world tick boundary, excluding network/queue latency; not a p95 load claim.
