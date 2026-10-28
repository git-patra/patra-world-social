# T12 independent spec and code review

Reviewed base `ca12faeaba0cb5f5d3c350e5010853bfb6151e74`, preserved implementation tree `adb420251e5f085ec2a6e763e8bd73b6b322645e`, and the supplied task-scoped review package. The implementation paths in the working tree match that preserved tree. Root documentation/evidence changes are outside this review.

**Spec verdict: changes required.** The core ranking, transactional projection, removal seam, backfill, and private read boundaries satisfy the supplied T12 architecture rulings. The production UI does not consistently discard private data after learning that access has ended (R1). Historical missing-detail presentation also invents a zero score (R2).

**Code verdict: changes required before approval.** One Important and one Minor finding; no Critical finding. This is a T12 review, not checkpoint acceptance or whole-branch approval.

## Critical

None found in the reviewed scope.

## Important

### R1 — Fence both board and detail reads when access is revoked

**Locations:** `apps/web/src/features/leaderboard/ChampionBoard.tsx:23`, `:25`, `:28`; supporting controller lifetime at `apps/web/src/features/leaderboard/controller.ts:9` and `:16`.

**Trigger:** A member has the podium or a result deep link open. A board/detail request begins while membership is valid, then a kick, ban, archive, or expired session produces `access_denied` before the request's already-authorized response reaches the browser. The status listener clears `board` and `detail`, but neither disposes/invalidates the board controller nor aborts the independent detail request. `online-ui.ts:43` forwards this state without detaching the socket; its normal status path does not trigger the component's effect cleanup. Consequently the outstanding success callbacks can repopulate both states after revocation. A second deterministic path is an already-loaded deep-link detail followed by a focus-triggered board request returning 401/404: the failure callback clears only `board`, leaving the detail rendered beneath the access error indefinitely.

**Impact:** The private world panel continues to present a result as available to an active member after the app knows access has ended. This is a client state/privacy-boundary defect, not a claim that a new unauthorized HTTP request bypasses the backend membership check. The backend's correct read authorization does not fence a previously authorized response already in flight.

**Required fix:** Give board and detail requests a shared access/lifetime fence. On access denial, cancel/invalidate both, clear both displays, settle loading state, and ensure old callbacks cannot restore either. Treat authorization failures on a subsequent read the same way; retain enough error status to distinguish authorization denial from a transient service error if different handling is wanted. Scope socket denial to the displayed world when viewing a deep link for a different world. Restore loads only through a newly valid/rechecked attachment or explicit retry. Add a focused production-component/lifecycle test for deferred board and detail responses resolving after denial, plus a loaded-detail → denied refresh case. The present `dispose()` controller unit test does not execute this status listener or the independent detail effect.

**Evidence:** Static control-flow review of the actual production mount, effects, socket status forwarding, and controller callbacks. No credentialed browser workflow was run to reproduce it.

## Minor

### R2 — Preserve missing historical Duck detail instead of reporting a zero-point incomplete session

**Locations:** `apps/server/internal/leaderboard/query.go:110` and `:161`; rendered by `apps/web/src/features/leaderboard/ChampionBoard.tsx:15`.

**Trigger:** An account's latest completed historical Duck record has no `duck_results` row. This is an explicitly supported pre-Duck/backfill case, constructed in `apps/server/internal/leaderboard/champion_integration_test.go:337`–`:338` and intentionally skipped by `apps/server/internal/leaderboard/backfill.go:59`–`:65`. `LastResult` still selects it, while `summarySelect` converts the absent score/hits/shots to zero and absent `complete` to false.

**Impact:** The UI asserts “0 poin · 0.0% · Tidak lengkap” even though there is no validated Duck detail establishing any of those values; the supplied fixture's immutable history actually contains a 500-point finished outcome. Backfill correctly avoids promoting this record, but the last-session display conflates missing data with a real zero-score incomplete session.

**Required fix:** Represent detail availability explicitly (or nullable score/statistics) and render an unavailable/unvalidated historical detail message without a numeric score or invented completion outcome. Preserve the immutable history and existing exclusion from the official board. Extend the existing missing-detail backfill case to assert the returned last-result representation and its display.

**Evidence:** Query/render path and the existing historical fixture; no additional database test was executed.

## Compliant seams inspected

- Ranking uses descending score, bounded exact integer cross-products for rational accuracy, ascending `FinishedAt`, then canonical result UUID. The per-account partition primary key and `better` check enforce unique podium accounts and prevent a non-improving PB replacement. Zero-shot accuracy is zero; human/account/finished/non-assisted eligibility is checked.
- Application composition runs `duck.Project` then `leaderboard.ProjectResult` in the existing finalizer transaction, before participant/match updates. There is no early commit or separate trusted score-upload API. Projection receipts, PBs, champion rows, Duck detail, and finalization share commit/rollback fate.
- Match advisory locks, sorted account locks, and partition row locks serialize retries and distinct finalizations, including an initially absent partition. `RemoveAccount` takes the same account lock before discovering ordered partitions, persists the exclusion tombstone, removes entries, anonymizes champion references, and records separate administrative changes. Same-owner improvements and administrative promotion do not add champion victories. Full deletion/session/history orchestration remains the expressly approved T14 dependency.
- Startup backfill runs before recovery/workers/serving, uses bounded ID batches and the startup cancellation context, orders historical imports by finish time/result ID, avoids replaying Duck detail insertion, and commits durable per-result receipts. Earlier completed batches survive cancellation and restart. No undocumented launch cutoff was added. The ten-second startup budget/repeated-start operational limitation is disclosed in the implementation report.
- Board and world-result summary reads require active membership, a non-archived world, no ban, and no leaderboard exclusion. The summary exposes approved Duck fields and version metadata, while unchanged personal history remains participant-private and survives membership removal/archive. Reads use a coherent repeatable-read snapshot and `no-store` HTTP responses.
- The worker's completed callback occurs after confirmed finalization/reconciliation. The new world-only event contains partition/result identity; participant-only result status is not widened. Application transport tests exercise same-world owner/member delivery, other-world isolation, durable recovery, and replay. Board snapshots replace state; duplicates are bounded/deduplicated and reconnect/focus/manual reload paths exist.
- The actual app mounts the panel, forwards the new socket event, renders three empty podium slots without fake winners, offers current/archive partitions and world-scoped summary links, and displays PB/last result/champion/admin sections. The access-revocation exception is R1 above.

## Evidence limits and acceptance accounting

The requirements, binding review context, implementation report, full T12 diff, new tests, and concrete predecessor integration seams were inspected. No source, staging, HEAD, server process, or browser/account fixture was changed; only this requested review report was written. Passing suites were not repeated.

The reported full **default-parallel** Go race/integration run failed the existing chat sixth-message rate assertion. The focused chat diagnostic and separately identified full **serial-package** `-race -tags=integration -p 1 -count=1 ./...` run passed. Go unit/vet/build and frontend lint/typecheck/tests/build also passed as recorded. Those are implementation execution evidence, not newly executed reviewer results; the chat timing harness remains a T16 reliability obligation and is not silently relabeled PASS.

Root's native CUA evidence covers guest gate/copy/contrast only. Authenticated board/account/invite/deep-link/reconnect browser E2E is authored and typechecked but **NOT RUN**, under the user's explicit deferral. Isolated HTTP/WebSocket/PostgreSQL tests do not substitute for browser acceptance. No physical mobile or provider acceptance is claimed. The live API remains the accepted T11 runtime until the coordinator performs its separately authorized integration work.

No new product requirement or architecture ruling is needed for these findings: R1 follows the existing active-membership and stale-response boundary; R2 preserves the already accepted missing-detail historical-data case without manufacturing score facts.
