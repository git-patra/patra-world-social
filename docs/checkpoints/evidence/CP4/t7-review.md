# T7 task-scoped review

Reviewed the supplied brief, implementation report, and complete 32-file staged snapshot diff against base/HEAD `ef7ed2dddd738c36392d88c7a525e8af1b849d95`. The truncated socket portion was recovered in a bounded read. No source/index/HEAD changes, browser actions, or test-suite reruns were performed.

## Spec-compliance verdict: changes required

The main T7 lifecycle requirements are implemented: authenticated transport identity, per-world arenas, five race slots with bot fill/all-human readiness, explicit penalty bot choice, four separate duck cabinets, frozen participants, host migration, busy claims, spectator admission, durable preparation before activation, and registry-specific timing. Nullable asynchronous `starts_at` projection follows the coordinator ruling. One Important defect remains in the required safe-return integration, detailed below.

## Task-quality verdict: changes required

The owner/storage split and stable-ID reconciliation are sound foundations, and the tests cover meaningful failure paths. Fix the asynchronous return race before accepting T7. No Critical finding.

## Important finding

**[P2] Fence an arena return against a newer lobby claim.** Location: `apps/server/internal/application/lobby.go:106–111`; related mutation: `apps/server/internal/realtime/delivery.go:128–132`; return payload/drain: `apps/server/internal/lobby/service.go:88–91`, `apps/server/internal/lobby/presence.go:277–287`.

`publishLobbies` checks `g.Busy(user)` and subsequently calls `hub.SetActivity(..., false, &at)` without a shared ownership fence. These operations run on different owners. A concrete production interleaving needs no registered game engine: leave the race lobby, let its return publisher observe `Busy == false`, join the penalty lobby (which claims the user and sets playing), then let the old publisher apply its race return. The latter clears runtime activity and teleports the user to the old race entrance despite the new penalty claim. The later lobby publication/snapshot can restore the playing label, but it does not undo the stale teleport. Database claim cleanup is correctly bound to match/user; that protection is lost in the final world mutation.

Carry an ownership generation/lobby identity through the return and make the conditional application atomic with respect to claiming a newer lobby. A second independent `Busy` check merely narrows the same race. Add a deterministic interleaving test that pauses an old return after eligibility is observed, joins another lobby, and verifies the old return cannot change the new activity or position. Also account for the existing ignored `SetActivity` error when deciding when a return is considered delivered; that error-handling concern is not a separate Important finding because normal Hub serialization limits runtime-queue contention.

## Strengths

- `lobby/start.go` freezes roster/seed/version before storage, shares a start attempt among duplicate callers, and assigns the official race timestamp only after confirmed persistence. Penalty/duck use zero generic delay. Slow persistence and slow metadata projection have targeted tests.
- `lobby/store.go:91–108` takes the same per-match PostgreSQL advisory transaction lock as persistence before reporting absence. This prevents a failed COMMIT acknowledgement from becoming a false-negative lookup. Unconfirmed attempts retain claims and retry their stable identity.
- Individual departure cleanup deletes by match ID plus user ID and checks the current lobby identity before clearing in-memory ownership. Participants remain frozen. Whole-match completed release verifies an already-durable completed record.
- Public lobby serialization excludes the seed. The application derives actor identity through `WithJoined`; strict Go/TypeScript/schema payloads reject identity spoofing. Production factories are absent and the React view truthfully disables Start.
- Engine lifecycle hooks and bounded owner/control/storage queues keep SQL off simulation ticks. Shutdown cancels owned storage work and waits without holding admission locks.

## Verification assessment and scope

The report records passing Go race/integration suites, vet/build, frontend tests/typecheck/lint/build, and diff checks. The diff contains substantive assertions for concurrent starts, rollback, unknown COMMIT, unresolved transaction lookup, grace, natural finalization, command spoofing, owner-queue completion, and unavailable production engines. Those are implementation-reported results, not independently rerun results. The safe-return test in the diff exercises `Runtime.Activity` directly and does not cover the publisher/claim interleaving above.

Named supporting reads outside the changed hunks: `realtime.WithJoined`/`Hub.admit` for the authorization/return locking question; `world.Runtime.call`/`Join`/`Idle` for runtime rejection and activity semantics; `web/src/online-ui.ts` for attachment and cleanup-only recovery isolation. These reads did not reveal another Important defect.

Browser/visual/keyboard/touch acceptance remains **NOT_RUN**, explicitly user-deferred. Actual game engines and game consequences remain T9–T11; durable results/history, startup recovery, and result-storage uncertainty remain T8; voice release remains T13. Their absence is not a T7 defect. Later CP4 acceptance must exercise a concrete penalty lifecycle; production unavailable-engine behavior is not evidence that that future acceptance passed.
