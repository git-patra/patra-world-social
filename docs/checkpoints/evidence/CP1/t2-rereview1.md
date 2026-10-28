# Task 2 scoped re-review, fix round 1

Reviewed `task-2-fix1.diff` against the previously reviewed staged baseline and the final appendix in `task-2-report.md`. Scope: Important findings 1–3 and regressions introduced by this fix. No code, index, or commit changes; no suite rerun.

## Verdict

**All three Important findings are ADDRESSED.** I found no new breakage in the fix diff. The reported red/green evidence and final `make check` (17 frontend tests, Go tests/vet, typecheck, web/Go builds) plus uncached world/realtime contract tests support the changes; I did not independently rerun them.

1. **ADDRESSED — directional cancellation.** `apps/web/src/world.js:417` no longer blocks `move()` while a click-to-walk route is active. A legal direction replaces the route with one step. The controller test now starts a route through pointer events and verifies replacement.
2. **ADDRESSED — semantic map validation.** `apps/web/src/world/map.ts:55-65` rejects intersecting district interiors and flood-fills walkable tiles from spawn before accepting entrances. `apps/server/internal/world/map.go:92-103` applies equivalent rectangle and reachability checks. Shared invalid maps cover both cases. Touching rectangle edges remain valid.
3. **ADDRESSED — public event contract.** `contracts/event.schema.json`, `apps/web/src/world/avatars.ts:45-56`, and `apps/server/internal/realtime/protocol.go:72-91` define and validate the versioned event envelope and the 17 enumerated public event types. Shared valid/invalid fixtures exercise both decoders. Detailed event payload contracts remain later engine work, as stated in the fix report.

Previously recorded Minor findings 4/5 remain outside this round and are coordinator-tracked for T15/T5/final review.
