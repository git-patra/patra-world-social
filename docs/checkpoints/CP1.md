# CP1 — Fondasi: PASS

Verified 26 October 2026, Asia/Jakarta. Scope: T1 baseline and T2 shared map/rules/protocol/render foundation, including the user-requested removal of CV/portfolio product content.

## Commits

| Task | Commit | Result |
|---|---|---|
| T1 / C01 | 6d040e19f4058e714e97545a06e57185a07d6362 | Isolated baseline, build/CI, Go health, original art |
| T2 / C02 | d4f60a27bd41048c7ed16a103fbcf4875046ece5 | Expanded map, Go/TS contracts, camera and avatars, game-only UI |

## Acceptance evidence

| Requirement | Outcome | Evidence |
|---|---|---|
| Separate repo, original history and source preserved | PASS | T1 independent review; source HEAD8d96921, tracked diff empty, only preexisting.idea untracked |
| Frontend and Go build/checks | PASS | t1-verification.md and t2-verification.md; make check exit0 |
| Shared64×64 map and rules | PASS | Go/TS fixtures;11 entrances reachable; spawn unblocked; district overlap/disconnected maps rejected |
| Versioned command/event contracts | PASS | Shared golden fixtures,17 event types, malformed versions/fields rejected in Go/TS |
| Core theme and desktop/mobile layout | PASS | t1-desktop/mobile; game-only-desktop/mobile; mobile is emulation390×844 |
| Camera/zoom/minimap and expanded navigation | PASS | Actual RacePark arrival30,14; t2-browser-verification.md and t2-race-district.png |
| Multiple avatars in correct depth and readable names | PASS | Renderer tests; t2-three-avatars.png with labeled dev fixtures |
| Motion preference does not teleport/change route | PASS | Actual controller route parity test and browser mid-route toggle |
| Game-only content | PASS | game-only-verification.md; source/public/build scan; resume+generator and promotional assets removed |

Evidence directory: [evidence/CP1](evidence/CP1). Final frontend count17 tests; all Go suites/vet, TypeScript and production builds pass. Uncached `go test -count=1 ./internal/world ./internal/realtime` confirms external JSON fixture execution.

## Review and resolution

T1 approved independently. T2 initial review found directional cancellation, semantic map validation and incomplete event contracts. Implementer added regression fixtures and fixes; scoped reviewer marked all three ADDRESSED with no new breakage. See t2-review.md and t2-rereview1.md.

## Explicit limits and next work

This checkpoint proves a local foundation, not online multiplayer. District-specific art, actual game engines, auth, persistent results and voice follow later tasks. General dense20-avatar/physical-device performance remains T15/T16. Static bundled checksums are recorded; future remote ingestion must verify integrity in T5. Minor nonfinitezoom guard belongs T5; village facade signs belong T15. No deployment/push or source changes performed.

Next: T3 username/password authentication, PostgreSQL sessions and one-use recovery. LiveKit staging remains unavailable per user reply and does not block CP2.


Publication note (28 October 2026): the obsolete `t1-desktop.png` and `t1-mobile.png` portfolio screenshots mentioned above are retained only in local inherited Git history. They are omitted from the game-only public snapshot. Later game-only CP1 screenshots remain available; historical test claims are unchanged.
