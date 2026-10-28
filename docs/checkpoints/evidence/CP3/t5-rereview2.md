# T5 fix round 2 scoped re-review

## Spec compliance

**✅ The remaining Important finding is ADDRESSED.** Offline leave and auth-null now restore local solo independently of server acknowledgement, retain truthful pending-release state, and continue fencing replacement authenticated joins.

**⚠️ Credentialed browser acceptance remains NOT_RUN by user deferral.** This approval concerns code readiness in the scoped fix; it does not claim browser acceptance.

## Finding disposition and strengths

- **Round 1 P2, local solo blocked by unconfirmed release: ADDRESSED.** `apps/web/src/online-ui.ts:44` detaches authoritative input/rendering and clears the latest snapshot before awaiting `close()`. A false result retains the release handle and reports pending release at `:53`; `:32` distinguishes locally solo status from server-release status. Returning to local mode no longer requires network availability.
- **Replacement admission remains fenced.** `apps/web/src/online-ui.ts:61` serializes joins and checks authentication plus intent generation both before and after the awaited release. An unconfirmed release aborts replacement creation. `:104` increments the generation on auth-null, preventing an old queued switch from reconnecting after logout.
- **Recovery stays detached.** `apps/web/src/online-ui.ts:70` guards socket identity and routes a detached recovery snapshot into release, rather than authoritative rendering/input. Retry/takeover require authentication at `:90` and `:92`; the rendering loop also requires `attached` at `:112`. A later acknowledged recovery permits a subsequent fresh join without automatic takeover.
- **Relevant regressions are present.** `apps/web/tests/map.test.ts:125` adds disconnected leave/auth-null tests with pending-release and replacement fencing assertions, extends recovery through resume → leave → acknowledgement → later join, and adds logout-during-queued-switch coverage. The prior successful acknowledgement/controller switch tests remain.

## Remaining issues

### Critical / Important

**None in this scoped re-review.** The earlier four backend/normal-switch findings remain resolved as assessed in round 1; no backend changes were part of round 2.

### Minor

- `apps/web/src/online-ui.ts:49` calls `world.setOnline(null)` on every stop, including recovery while already detached (`:75`). The existing controller implementation at `apps/web/src/world.js:509`, inspected in the original T5 diff, clears its current route on each call. Thus recovering a pending server seat while walking locally can cancel that solo route, even though the adapter correctly stays local. Guard the world detach operation with the previous `attached` state (or make redundant detach preserve solo movement). This does not block local solo availability or admission safety.

## Verification and scope

- Reviewed `task-5-fix2.diff`, appended round 2 evidence, the prior remaining finding, and the original brief already available in this review context. No broad re-review, backend commands, browser actions, subagents, source/index/HEAD edits, or test reruns occurred; only this report was written.
- Final reported verification: 28 frontend tests (8 JavaScript + 20 TypeScript), typecheck, lint, build, and diff check all PASS. The tests use the actual mountOnline controller with synthetic Node surfaces; they are distinct from the deferred browser acceptance.

## Task quality

**Approved for the scoped code-readiness gate, with one nonblocking Minor note.** The fix separates local play from server-release accounting and adds the lifecycle fencing needed around asynchronous joins and logout. Browser acceptance remains a separate pending gate.
