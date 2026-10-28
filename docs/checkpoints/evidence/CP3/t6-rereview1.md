# T6 scoped rereview — fix round 1

**Spec compliance: PASS for the scoped fixes.** Both blocking T6 review findings are addressed, and the fixes preserve the binding brief's runtime, durable ACK, deduplication and attachment boundaries.

**Code quality: APPROVE for this fix round.** No new Important findings in the four-file change. This is a scoped rereview, not whole-branch or authenticated browser acceptance.

Reviewed `task-6-fix1.diff` against the previously reviewed staged snapshot, the appended round-1 implementation report, prior review, and the binding task brief retained from the initial review. No source/index/HEAD mutation, subagents, browser actions or test reruns. Only this report was written.

## Finding disposition

### P2 shutdown/expiry deadlock — ADDRESSED

`apps/server/internal/realtime/hub.go`, `apps/server/internal/realtime/delivery_test.go`.

`Close` now explicitly releases both `mu` and `lease` before waiting on `done`. The expiry critical section retains lease-before-mu ordering and is shared by the real worker and the focused regression via `expireTick`. A worker that selected expiry before shutdown can acquire the released locks, finish, and subsequently observe the stop signal. Concurrent/repeated `Close` calls see the existing closed state, release their locks and wait on the same worker without repeating shutdown mutation.

The regression holds shutdown inside a controlled room's `Close`, signals a worker committed to executing the real expiry critical section, then releases room shutdown. Under the old deferred lease unlock, that worker and shutdown form the original wait cycle regardless of which blocked goroutine runs next. The test additionally checks concurrent and repeated close completion. This is targeted evidence for the reported defect without adding production test-only hooks.

### P2 edited draft erased by earlier ACK — ADDRESSED

`apps/web/src/chat-ui.ts`, `apps/web/tests/map.test.ts`.

The successful send handler clears the textarea only if it still equals the submitted text. Its consumed draft identity is reset, so an intervening edit remains visible and receives a fresh identity on the next submission. Failure retains the previous draft identity for an explicit unchanged retry. Generation checks still precede transcript/field mutations and pending-state cleanup.

The new tests mount the actual `mountChat` controller with a lightweight DOM and deferred transport. They cover A pending/edit B/A ACK, distinct identity for B, unchanged success clearing, failure/retry retaining the same identity, and an old-world ACK arriving while a new-world submission is pending. The latter checks the new field, disabled send state and transcript isolation. These tests exercise the production form handler rather than a copied clearing function.

### P3 authored keyboard/reconnect acceptance weakness — NOT ADDRESSED, DEFERRED

`apps/web/tests/social.e2e.spec.ts:21–23` is unchanged. Per coordinator direction, this nonblocking source correction is deferred until acceptance/T16. Preserve it as an open item before relying on that acceptance test. Authenticated browser execution remains **AUTH_BROWSER_NOT_RUN** by explicit user instruction; no credential or injected-session workaround was performed.

## Evidence and limits

The appended report records the focused shutdown regression failing under the reviewed locking behavior and passing after explicit unlock; it also records the edited-draft regression failing before the form fix and passing afterward. The reported final realtime package race run passes in 9.909s, and the frontend suite passes 34 tests with typecheck/lint/build passing. The diff's test bodies and production changes support the stated regression coverage. These are implementer-reported execution results; this reviewer did not rerun passing suites or independently recertify their logs.

No broad integration rerun is needed solely for this narrow lock-release/form-clearing change, given the reported affected-package race and frontend checks. No new Important defect was identified. Integration, whole-branch review, and remaining acceptance decisions remain coordinator-owned.
