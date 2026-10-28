# T5 fix round 1 scoped re-review

## Spec compliance

**❌ Issues found: one Important regression remains.** The four original defects have been addressed on their reported paths, but the new leave failure handling prevents returning to local solo without a working server connection.

**⚠️ Credentialed browser acceptance remains NOT_RUN by user deferral.** This is a code-readiness review of the fix diff, not browser acceptance.

## Strengths / original findings resolved

- **Scoped revocation: resolved.** `apps/server/internal/realtime/hub.go:386` checks relevant account/world/membership generations against the preauthorization revision. Unrelated subjects no longer fail just because the global counter advanced. `apps/server/internal/realtime/hub_test.go:193` gates unrelated revocation during join, validation, and resume; the existing same-subject admission test remains in the reported passing run.
- **Admission cleanup: resolved.** `apps/server/internal/realtime/hub.go:196` checks cancellation under the admission lock and publishes the candidate seat only after runtime join and snapshot succeed. Its error paths remove the candidate avatar/affected same-world old seat and close empty rooms, while a cross-world old seat remains until destination success. `apps/server/internal/realtime/hub_test.go:253` checks six cancellation interleavings for unknown seats and avatar/seat consistency.
- **Received leave and normal switching: resolved on the original paths.** `apps/server/internal/realtime/connection.go:75` records an authenticated transport's valid received leave and processes it after reader shutdown, even if ordinary queue processing or acknowledgement cannot finish. `apps/server/internal/realtime/hub.go:392` binds cleanup to the exact seat/token/world/epoch. `apps/server/internal/realtime/connection_test.go:184` and `:220` exercise immediate close and acknowledged leave followed by another world. `apps/web/src/net/socket.ts:107` waits for the matching acknowledgement, and `apps/web/src/online-ui.ts:24` serializes replacement joins; the new real-controller Node test exercises that ordering. The remaining issue below concerns the newly changed failure path, not the successful acknowledgement path.
- **HTTP drain: resolved.** `apps/server/cmd/server/main.go:71` awaits both Serve and Shutdown, closes remaining connections after the drain deadline, and propagates errors before main tears down the pool. `apps/server/cmd/server/main_test.go:38` covers a blocked handler and drain timeout.

## Issues

### Critical

None found.

### Important

1. **[P2] Returning to solo is now blocked by unconfirmed server release.** `apps/web/src/online-ui.ts:19` returns before `world.setOnline(null)` whenever socket close resolves false. A concrete sequence is: join a private world, lose the transport, then press **Kembali ke solo**. `apps/web/src/net/socket.ts:113` resolves false; `:123`–`:127` closes the transport but retains the seat and enters `leave_unconfirmed`. The controller keeps the dead online adapter, so the existing local movement engine remains disabled and its saved solo position is never restored. Subsequent leave attempts fail again (`ws` is null but `seat` remains), with no automatic recovery when the 30-second server reservation expires. The logout listener at `apps/web/src/online-ui.ts:44` uses the same stop path, so an auth-null transition can also leave a guest stuck in the old online controller. A network/server outage should not make the local solo mode unavailable.

   **Required distinction:** detach from authoritative rendering/input and restore local solo when the user leaves or authentication becomes null, while separately retaining truthful pending-release/recovery state. A new authenticated world join must still wait for confirmed release or follow a valid recovery/explicit takeover path; do not restore the original leave-then-join race. Add synthetic controller regressions for disconnect → return to solo and auth-null while disconnected, asserting `setOnline(null)`/local movement restoration, an honest pending-release indication, and no replacement socket before the release/recovery condition is satisfied.

### Minor

No new Minor finding in the scoped fix. The previous Reset announcement mismatch was explicitly deferred to T15 and was not changed or re-reviewed here.

## Verification and scope

- Read the original T5 brief, the appended fix-round evidence, and the fix diff. Recovered the truncated portion from a bounded diff read. No broad repository search, source/index/HEAD mutation, subagent, or browser action occurred; only this report was written.
- Reported final evidence: scoped Go race tests for realtime/world/main PASS; real-application integration smoke PASS; focused vet/build PASS; final frontend 8 JS + 17 TS tests and typecheck PASS; lint/build and diff checks PASS. These passing suites were not rerun.
- The remaining issue follows directly from the new early return, retained seat, and unchanged online adapter contract already examined in the original review. The new tests verify successful acknowledgements and serialized joins but do not exercise returning to solo on close failure. No additional test was executed during this read-only re-review.

## Task quality

**Needs fixes.** The original concurrency and shutdown defects are corrected with relevant regressions. One new failure-state regression still blocks code-readiness approval; browser acceptance remains independently deferred.
