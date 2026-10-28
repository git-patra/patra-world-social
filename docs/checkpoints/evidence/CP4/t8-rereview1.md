# T8 fix round 1 re-review

**Spec compliance: APPROVE T8 code scope.**

**Code quality: APPROVE T8 code scope.**

Both original Important findings are **ADDRESSED**. No new Critical or Important regression was found in this fix diff. No blocking Minor findings.

## Scope

Reviewed the appended `Review fix round 1/5` evidence in `task-8-report.md` and the complete four-file, 17,235-byte `task-8-fix1.diff` (reviewed index to working tree). HEAD remains `deaa1d648a95fa10ae1426b6d257eddf1fcc89cc`. The original brief, binding scope, and `task-8-review.md` R1/R2 remain the acceptance criteria. This is a scoped re-review, not a new repository-wide review.

## R1 / P1 — ADDRESSED

`history/finalizer.go` now treats the initial query solely as candidate enumeration. Each candidate is handled in an explicit READ COMMITTED transaction which acquires the existing `lockMatch` advisory fence before a separate statement reads status and durable-job existence. The absent-job decision therefore uses a fresh snapshot after a settling enqueue has resolved, rather than the stale snapshot that reproduced the original defect.

Only a genuinely absent running/finalizing candidate becomes `aborted_server`. A durable pending job retains its claims for finalization/reconciliation, while terminal old-runtime claims are released by match ID. Per-match transactions may make partial progress if startup recovery fails, but `Application.New` still fails before admission; subsequent startup can safely resume that work.

The permanent real-PostgreSQL `TestPostgresStartupFencesSettlingEnqueueAndRecoversOnce` exercises the original lock/commit interleaving. It observes database blocking through the enqueue backend PID, commits the settling enqueue, and asserts preserved `finalizing` status and claims. It then runs the actual recovery worker and verifies one projector row, completed status, stale-claim cleanup, and reopened admission. Its reported RED result reproduced `aborted_server, claims=0`; the reported GREEN result passes. Existing unknown-start orphan coverage remains included in the covering run.

## R2 / P2 — ADDRESSED

`history/worker.go` now bounds the entire Step with a shared three-second context and attempts at most one ready match per Step. It discovers owner completions and checks all known live deadlines before selecting storage work. The selected attempt, retry bookkeeping, and both database probes share the same operation context, which is further capped by the earliest known live deadline.

Consequently, 21 detached IDs can no longer cause 21 consecutive three-second waits before owner discovery. A completion that appears during storage work is discovered on the next bounded pass; once discovered, its authoritative `Since` caps future operations at its own 60-second boundary. The following pass performs detach/notification within the existing polling allowance. Live work has priority, while ready-time and stable ID ordering avoid arbitrary map-order starvation within a class. New starts remain blocked while unresolved IDs or failed probes remain.

The permanent `TestWorkerDiscoversNewDeadlineDuringSlowDetachedBacklog` preserves the original newly-finished-during-reconciliation trigger. It verifies bounded initial work, retention of the authoritative deadline, detach and unconfirmed notification at that boundary, admission remaining closed, no premature terminal outcome, and eventual absent/durable outcomes after the backlog drains. The initial blocking Step uses real context waits; later deadline checks use the existing injectable Step clock. The report clearly distinguishes this from a 60-second wall-clock run. Both variants reportedly fail against the original implementation and pass after the change.

The fix retains synchronous/joined storage calls, uncertain-outcome reconciliation, match fencing, completed live result screens, and the unchanged generation-fenced lobby return path. No new result fabrication or premature saved notification was introduced by the changed control flow.

## Evidence and review limits

The reported permanent RED/GREEN command covers both regressions. The covering `go test -race -tags integration ./internal/history ./internal/lobby ./internal/application ./internal/realtime -count=1` reportedly passes all four affected packages; `go vet ./...`, server build, and `git diff --check` also reportedly pass. These claims are consistent with the reviewed changes and regression assertions. No frontend file changed in this round, so retaining the earlier frontend evidence is appropriate.

This re-review did not rerun suites: the fix code and supplied focused evidence resolved the original doubts, and no new concrete correctness question required another test. It did not edit source, index, or HEAD, and did not perform browser work or delegate additional review.

**NOT_RUN / not accepted by this verdict:** credentialed browser flows, authenticated History UI acceptance, CP4 acceptance, Playwright/CUA/session injection, LiveKit/device gates, and later concrete game engines/projectors/timers. The user's browser deferral remains in force. T9–T12 and T11's accepted three-second Duck countdown obligation remain later work. This approval closes the two T8 code findings only; it does not imply those separate gates are complete.
