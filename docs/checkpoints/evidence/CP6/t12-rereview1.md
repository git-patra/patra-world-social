# T12 R1 fix round 1 — independent rereview

Scope: Important R1 and regressions introduced by `task-12-fix1.diff`, relative to preserved staged tree `adb420251e5f085ec2a6e763e8bd73b6b322645e`. Reviewed the entire fix package, including the appended new `component-lifecycle.ts` file, the implementation report's round-1 section, the binding review context and approved harness ruling, and the current production component/controller wiring. Backend behavior and the unchanged R2 code were not reopened.

**R1: addressed.**

**Spec verdict: PASS for this fix scope.** The previously identified access-revocation/stale-response behavior is corrected without expanding the approved privacy or notification architecture.

**Code verdict: approved for this fix scope.** No new Critical, Important, or Minor regression found in the fix. The original Minor R2 remains open and explicitly deferred to T16/final review by the coordinator; this approval does not resolve it or constitute full checkpoint/browser acceptance.

## Production behavior verified by inspection

- `apps/web/src/features/leaderboard/ChampionBoard.tsx:25`–`:34` now puts board and detail reads in one effect lifetime. Cancellation increments a shared generation, disposes the actual board controller, and aborts the detail request. Denial clears both data states, sets an access error, and settles loading. Both success and failure continuations check the current lifetime/generation, with the detail path also checking its abort signal.
- `apps/web/src/features/leaderboard/controller.ts:4`–`:13` preserves HTTP status in `ReadError` and passes the original cause to the production component. A 401/403/404 from either request invokes the same denial path, so a denied board refresh no longer leaves a loaded detail visible. Treating the deliberately ambiguous 404 as a denial is consistent with the current API boundary.
- `ChampionBoard.tsx:36`–`:49` wires the actual retry button and window listeners to this lifetime. Focus and committed-result events do not restart a denied lifetime. Explicit retry and a matching-world reconnect transition create fresh reads. Repeated `online` snapshots do not repeatedly reopen it. Socket identity plus displayed-world checks prevent an unrelated world's denial from clearing an authorized deep link.
- `ChampionBoard.tsx:50`–`:51` disables the old lifetime, cancels both reads, clears the retry reference, and removes all three listeners on cleanup. World, rules, match, and socket dependency changes replace that lifetime. The existing production mount at `:57`–`:59` still keys account/world/rules identity and unmounts on logout/attachment removal; it mounts the same exported component used by the regression tests.

## Regression evidence assessment

`apps/web/tests/leaderboard.test.ts:24`–`:55` invokes the real `ChampionBoard`, real controller, real response parsing, and actual component window listeners against controlled fetch promises. It covers both reads resolving after socket denial and after a fresh retry; loaded detail cleared by a 401/404 board refresh; detail denial canceling a pending board; stale transient failure after recovery; fresh reconnect reads versus repeated online statuses; unrelated-world denial; and match-route dependency replacement.

The small test-only driver in `apps/web/tests/component-lifecycle.ts:5`–`:11` preserves hook slots, compares dependencies, invokes prior cleanup before replacement effects, and restores React's dispatcher after rendering. It is sufficient for the synchronous effect/state paths exercised here. Its use of React internals is confined to tests and makes it sensitive to React upgrades; it is not a DOM renderer, browser scheduler, StrictMode simulation, or substitute for authenticated browser acceptance. No new dependency or production harness coupling was introduced.

The implementation report records four new regression failures before the fix, then **8/8 focused Node tests PASS**, followed by affected **typecheck and production build PASS**. Those are reported implementation executions, not newly rerun reviewer tests. No passing suite or browser workflow was repeated for this rereview.

## Remaining boundaries

- R2 (missing historical Duck detail rendered as a zero-score incomplete session) remains unchanged and ledgered for T16; it is not silently marked fixed.
- Authenticated account/invite/podium/deep-link/reconnect browser acceptance remains **NOT RUN** under the user's explicit deferral. Guest CUA evidence remains limited to guest gate/copy/contrast. Node lifecycle evidence does not alter either boundary.
- The earlier default-parallel full Go race run remains **FAILED** on the chat timing assertion; the separately recorded serial-package full run passed. This frontend-only fix neither changes that accounting nor fixes the T16 harness obligation.
- Only this rereview report was written. No implementation, staging, commit, process, browser, credential, or fixture changes were made by the reviewer. No new requirement or architecture ambiguity emerged.
