# T9 scoped rereview — fix round 2

What this change does: The fresh engine receives preserved disconnect metadata before saved departure events are replayed. If that replay ends an immediately running duel, activation exposes it as finalizing before the start call completes.

**R1: ADDRESSED. Spec verdict: APPROVE T9 CODE SCOPE. Quality verdict: APPROVE T9 CODE SCOPE.** No new Critical or Important finding in the round2 diff. Original findings1–3 remain addressed under the prior review. O1 remains separately ledgered to T16/final review and was not reopened or included in this fix's acceptance gate.

## Scope and evidence

Read the complete `task-9-fix2.diff` (7,231 bytes), including the new `activation_test.go`, and the appended round2 section of `task-9-report.md`. The original brief, binding review context, and `task-9-rereview1.md` supply the unchanged requirements. The patch is relative to staged round1 tree `2f9a1bae9afa38b1d287d677f6e86ead874a68f6`; base/HEAD remains `e613707a6f64faea291c137504d2de7a1170e9b2`.

The production change is confined to `apps/server/internal/lobby/start.go`:

- After activation, it installs every preserved nonzero `p.disconnected` timestamp in a first pass. No expiry/leave catch-up can run against only part of the frozen roster's disconnect metadata.
- The second pass replays saved departures with their original kind and time. An `expired` event can now find the original grace cutoff in the fresh penalty engine, closing the R1 no-op path.
- An immediate `Tick` for a running engine promotes a terminal result to finalizing before closing the start attempt. Countdown engines retain their existing later advancement path.
- The all-gone branch is unchanged. Existing engine time clamping keeps FinishedAt at or after official StartsAt, including when the original grace cutoff precedes activation.

The permanent tests use the actual lobby and penalty adapter with a held persistence operation and deterministic clock:

| Test | Required invariant established by its assertions |
| --- | --- |
| `TestActivationReplaysOneExpiredGraceBeforeFreshEngineCanContinue` | One actor expires while persistence is held; activation at115000 and115100 immediately exposes forfeit/win, preserves the ordered frozen actors, produces no resolved timeout turns, and uses the actual official StartsAt with FinishedAt equal to it. |
| `TestActivationPreservesStillInGraceOriginalRemainingTime` | Activation at114999 remains choosing, preserves the original cutoff, and forfeits at115000 rather than ending early or granting another15seconds. |
| `TestActivationRetainsAllGoneWithoutFabricatingResult` | Both actors expire before activation; start is forbidden, no game result is copied, and cleanup is abandoned. |
| `TestActivationRetainsOriginalIntentionalLeaveAndClampsFinish` | Intentional leave remains forfeit/win, frozen roster is preserved, and FinishedAt is clamped to official start. |

The report records the new expiry-before-activation cases failing before the production fix and passing afterward, while the adjacent preservation cases already passed. It also records passing lobby unit tests, penalty/lobby integration with race detection, focused application penalty integration, scoped vet/build, and diff-check. Those commands and outputs are consistent with the inspected implementation and test coverage.

The controlled store establishes the precise activation interleaving; it does not claim a real PostgreSQL outage experiment. The report keeps that distinction explicit. No concrete unanswered doubt justified a further probe, so this rereview ran no tests or suites. No source/index/HEAD changes, subagents, browser activity, credentials, process restarts, or preview changes were performed; only this review report was written.

## Acceptance limits

This approval is for **T9 code scope**, not full external acceptance or deployment. Authenticated two-human browser, credential/invite/session workflows, online reconnect/device acceptance, and CP4 remain pending. Authored Playwright sources remain NOT_RUN. Root's supplied guest CUA evidence and the interrupted-reload distinction are unchanged; full reload persistence remains T14. The T15 zone-label visual issue remains nonblocking. No race/duck/voice or later-task completion is implied.

The pre-existing O1 processed-disconnect → coalesced timely resume → new disconnect issue remains recorded in `task-9-rereview1.md` for T16/final review. This fix neither changes nor resolves it. Accepted T8 recovery/work-budget and return-generation guarantees remain binding.

Verdict: ship the reviewed T9 code scope. Not checked: pending authenticated browser/device acceptance and later-task features.
