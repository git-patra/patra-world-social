# T8 independent review

Reviewed the supplied 37-file / 118,580-byte staged package against `task-8-brief.md`, the binding review context, and implementation report. Base/HEAD: `deaa1d648a95fa10ae1426b6d257eddf1fcc89cc`; no T8 commit. Root-owned documentation and browser evidence are outside this review package.

**Spec compliance: NEEDS CHANGES.** Two Important findings violate startup durable recovery and the 60-second uncertain-storage release requirement.

**Code quality: NEEDS CHANGES.** Both findings are reproducible failure-ordering bugs. The existing passing suites do not exercise these interleavings. No Critical findings; no additional Minor findings requiring action.

## Important findings

### R1 — P1: Fence startup's absent-job decision against a settling enqueue

**Location:** `apps/server/internal/history/finalizer.go:176` (`RecoverPending`; claim cleanup follows at line 180).

The startup UPDATE checks `NOT EXISTS(result_jobs)` in the statement snapshot without acquiring the per-match advisory lock used by enqueue/finalize/runtime reconciliation. If a pre-restart enqueue is still settling in PostgreSQL, recovery can see its job as absent, wait on the match-row lock, then mark the match `aborted_server` after that enqueue commits. The job is now durable, but the absence subquery came from the earlier snapshot. This is relevant to the expressly supported unknown-COMMIT/lost-ack boundary; it does not require admitting a second live game runtime.

**Reproduced interleaving on isolated PostgreSQL:**

1. Persist a running fixture match with NULL `starts_at`.
2. Open an enqueue transaction, acquire `lockMatch`, insert its immutable result job, and update the match to `finalizing`; keep that transaction open.
3. Call `RecoverPending` concurrently. Observe its UPDATE waiting for the old transaction's row lock through `pg_stat_activity`.
4. Commit the enqueue transaction, then await recovery.
5. The match is `aborted_server` despite its durable job. `Finalize` returns `invalid authoritative result`, because its completion UPDATE requires `status='finalizing'`.

**Consequence:** a committed result cannot be projected. Startup also releases its claims as if no job existed. The worker retries the permanently incompatible job and `HasPending` keeps authoritative new Start admission blocked indefinitely. No invented winner is committed, but valid durable work and service availability are lost.

**Required acceptance:** startup must serialize each definitive absent-job decision with the same match lock as enqueue/finalize and perform a fresh job/status read after acquiring that fence. Preserve the distinct `aborted_server` policy for genuinely absent jobs. Add a permanent real-PG regression for the interleaving above, asserting that the durable job survives as finalizable, projects exactly once, and does not leave admission permanently gated. Existing NULL-start orphan behavior must still pass.

### R2 — P2: Reconciliation backlog can hide another match's 60-second deadline

**Location:** `apps/server/internal/history/worker.go:74` (one owner snapshot per Step), `:100` (entire synchronous pending loop), and `:137` (detached reconciliation). `operationContext` at `:199` can only cap deadlines already present in `w.pending`.

Once previous matches have detached, every unresolved reconciliation may consume its full three-second context. `Step` does not resample the owner's newly finalizing matches between those operations. Another match that was already running when the outage began can finish during that loop, but its authoritative `Since` and deadline remain unknown to the worker until the entire backlog completes. Admission gating does not prevent this: it blocks new starts, while already-running matches can still finish at different times across worlds.

**Reproduced interleaving with the real Worker and context deadlines:**

1. Seed 21 already-detached outage IDs in `w.pending`; initial `Owner.Finalizing` returns no new references.
2. In the first `Reconcile`, expose a newly finalizing owner reference whose `Since` is the actual current time.
3. Each old reconciliation waits for and respects its three-second context, then returns that context error. No operation ignores cancellation.
4. After Step returns, **63.035 seconds** have elapsed since the new match finalized, and it has still never been detached. Its ID was never discovered during this Step.

**Consequence:** engine/render state and local arena/player claims remain beyond the required 60 seconds, and the unconfirmed notification is delayed. Larger allowed active-outage backlogs can extend this substantially. The per-operation timeout and existing single-match deadline test do not bound owner-discovery latency.

**Required acceptance:** bound total work between owner/deadline checks or make finalizing/deadline intake independent of slow storage reconciliation. Preserve serialized/joined writes, shared database fencing, uncertain-outcome wording, and the existing generation-fenced return path. Add a regression with a match becoming finalizing while old detached IDs are slow, asserting release and unconfirmed notification at its own 60-second boundary (within the intended polling allowance), continued admission blocking, and eventual durable/absent reconciliation without fabricated outcomes.

## Focused verification added by this review

No source, index, or HEAD edits were made. Both probes were injected through a temporary Go overlay:

- Test source: `/private/tmp/t8-recovery-review-test.go`
- Overlay: `/private/tmp/t8-review-overlay.json`
- Virtual package path: `/Users/patra/Project/patra-world-social/apps/server/internal/history/t8_review_test.go` (not written to the repository)
- Deadline probe log: `/private/tmp/t8-backlog-review.log`

The source contains `TestReviewRecoveryFencesSettlingEnqueue`, using the existing isolated `fixture(t)` and real PostgreSQL, and `TestReviewBacklogDoesNotDelayNewFinalizingDeadline`, using the existing owner fixture and a store whose `Reconcile` waits on `ctx.Done()`. These temporary files are available to the implementer for permanent regression tests. The complete triggering sequences are also recorded above so the report remains useful after temporary-file cleanup.

From the repository root, export the existing ignored `.cache/database.env` without printing it, then run from `apps/server`:

```sh
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -overlay /private/tmp/t8-review-overlay.json -tags integration ./internal/history -run '^TestReviewRecoveryFencesSettlingEnqueue$' -count=1 -v
```

Exit 1:

```text
=== RUN   TestReviewRecoveryFencesSettlingEnqueue
    t8_review_test.go:33: durable job was orphan-aborted: status=aborted_server
    t8_review_test.go:34: durable result no longer finalizable: invalid authoritative result
--- FAIL: TestReviewRecoveryFencesSettlingEnqueue (0.11s)
FAIL
FAIL patra-world-social/server/internal/history 0.853s
```

The following probe needs no database connection (the integration build tag only includes the overlay test file):

```sh
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -overlay /private/tmp/t8-review-overlay.json -tags integration ./internal/history -run '^TestReviewBacklogDoesNotDelayNewFinalizingDeadline$' -count=1 -v
```

Logged run, exit 1:

```text
=== RUN   TestReviewBacklogDoesNotDelayNewFinalizingDeadline
    t8_review_test.go:61: newly finished match still attached after 1m3.035122958s while processing older reconciliation IDs
--- FAIL: TestReviewBacklogDoesNotDelayNewFinalizingDeadline (63.04s)
FAIL
FAIL patra-world-social/server/internal/history 63.321s
```

The deadline probe was launched once before its process-session output was retained, then repeated with explicit log capture; only the captured run supports the output above. No broad suite was repeated.

## Assessment of other required paths and supplied evidence

- Typed immutable payload hashing, persisted T7 metadata/frozen roster binding, actual runtime start-time copying, and the absence of a browser score-write endpoint satisfy the intended authority boundary in the reviewed code.
- Enqueue and normal Finalize use the shared advisory fence. Projection, participant results, match completion and job completion share the finalization transaction. Saved events follow successful finalization or a successful durable lookup; the real-PG rollback, duplicate/conflict and lost-ack tests substantiate those ordinary paths.
- Runtime `Reconcile` acquires the match fence before deciding absence. Completed live result screens are retained for later game-owned release. The detach refactor preserves `releaseClaim`, generation-tagged returns, `WithReturn`, and acknowledgement behavior. R2 concerns when that path is reached, not a replacement of its ownership fence.
- History routes derive identity from SessionMiddleware, and list/detail queries require participation independently of current world membership. The provided guest/other-account/left-world/archive tests support those boundaries. Public result identity is overlaid from mutable participant projections; no result/public-history seed field was introduced.
- History requests use abort cleanup and account-key remounting. The initial account lookup has an epoch guard. Existing online dispatch rejects events from detached sockets before History receives them. Static review found no additional account-history disclosure in the changes.
- The report's race/integration, frontend lint/typecheck/tests/build, sqlc and coordinator vet/build evidence was assessed against the supplied code. This review did not rerun those passing suites or independently reclassify them as browser acceptance.

## Scope and NOT_RUN limits

- **NOT_RUN:** credentialed browser acceptance, session injection, Playwright/CUA, authenticated history UI flows, CP4 acceptance, LiveKit/device gates. The user-deferred browser gate remains deferred.
- Guest-only preview evidence is root-owned and reportedly used the T7 API; it is not proof of T8 authenticated behavior.
- No concrete production race/penalty/duck engine, game outcome computation, PB/champion implementation, or result display timer was required or inferred. Those remain T9–T12 obligations. T11 must still implement the accepted three-second Duck countdown correction.
- Existing passing tests plus fixing R1/R2 can support approval of **T8 code scope only**; they do not close authenticated-browser/CP4 acceptance or later concrete-game work.
