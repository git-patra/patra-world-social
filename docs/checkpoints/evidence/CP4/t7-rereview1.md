# T7 scoped re-review — fix round 1

**Original P2: ADDRESSED.**

**Spec-compliance verdict: approved for T7's authorized scope.** The safe-return requirement now has an ownership fence at the actual world mutation, and delivery failures retain the pending return.

**Task-quality verdict: approved.** No new Critical or Important defect found within the nine-file fix. The original review's task boundaries and deferred acceptance remain in force.

## Resolution evidence

- `lobby/presence.go` now supplies `WithReturn`, checking both an empty busy claim and the return's exact ownership generation under `busyMu.RLock`. `lobby/service.go` increments that generation whenever a new claim is created. Thus a later claim invalidates the return even if that claim has already been released, including rejoining the same lobby UUID.
- `world/activity.go` invokes the guard inside the accepted world-owner operation. The guard holds the ownership fence across movement neutralization, activity status, and coordinates together. The application prefilter is only an optimization; a new claim between that prefilter and world execution is rejected by the second, atomic check.
- Lock ordering is safe for the production callback shown: Hub mutex → world owner → ownership read lock. Claim writers hold the ownership lock only for the claim/generation mutation. The guarded callback directly changes world state and does not call back into Hub/lobby/storage/network. This preserves the no-I/O simulation-owner requirement.
- `DrainChanges` retains returns until `AckReturn`; the acknowledgement removes only the matching world/user/generation. `applyLobbyReturns` acknowledges success or verified supersession, while transient errors leave the return pending. The publisher's 100 ms retry path handles those retained errors without requiring another user action.

## Test assessment

The added PostgreSQL application regression reproduces the original interleaving: pause after eligibility, join penalty through authenticated `Hub.Execute`, resume the old race return, and compare position plus activity. The separate world-owner test directly checks activity because Hub snapshots can repair its visible label. Other focused assertions cover failed delivery retention/retry, same-lobby rejoin, stale return rejection after a newer claim is released, and stale acknowledgement protection.

The appended implementation report records RED failures for the original race and ignored delivery error, followed by passing focused real-PG application tests, affected world/lobby/realtime race suites, PostgreSQL lobby coverage, vet, and diff checks. These are reported execution results evaluated against the supplied tests, not independently rerun results. No unresolved concrete doubt justified another suite run.

## Review scope

Read the appended fix report and complete `task-7-fix1.diff` (reviewed index → working tree, HEAD unchanged at `ef7ed2dddd738c36392d88c7a525e8af1b849d95`). No unrelated source recrawl, tests, browser actions, source/index/HEAD changes, or commits were performed. This re-review writes only this report.

Browser/visual/keyboard/touch acceptance remains **NOT_RUN**, user-deferred. Concrete engines/game consequences (T9–T11), durable results/history/recovery (T8), voice release (T13), and later concrete penalty CP4 acceptance are unchanged future-task obligations, not newly claimed completions.
