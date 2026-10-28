# T6 task-scoped spec and code-quality review

Reviewed staged snapshot against base/HEAD `bb3eeaa6e456e2a5bcec6a26631a7b3f2718557c`, using the binding task-6 brief, implementation report, and full task-6 review-package diff in bounded passes (including recovery of truncated sections). No source/index/HEAD changes, tests rerun, browser actions, or subagents. This report is the only written file.

**Spec compliance: substantially implemented, changes required before approval.** The requested friends/invitation/world-chat/moderation flows and authority boundaries are present. Authenticated multi-context browser acceptance remains **AUTH_BROWSER_NOT_RUN**, explicitly deferred by the user; this is unavailable acceptance evidence, not a defect or authorization to work around that deferral.

**Code quality: request changes.** Two implementation findings and one nonblocking acceptance-source finding follow.

## Findings

### P2 — Release the lease before waiting for the expiry worker to stop

`apps/server/internal/realtime/hub.go:112–127` (new lease acquisition at 113; expiry worker acquisition at 102).

`Close` holds `h.lease` until its deferred unlock, but waits on `h.done` before returning. The expiry worker can already have selected `ticker.C` and be waiting for the same lease when `Close` wins acquisition (or be scheduled into that lock after `Close` acquires it). Closing `h.stop` cannot wake a goroutine blocked in `Lock`: shutdown waits for the worker, and the worker waits for shutdown to release the lease. `Application.Close` therefore can hang indefinitely. Release both hub locks before waiting for `h.done`, or otherwise arrange worker shutdown without holding a lock it requires. Add a deterministic regression that puts shutdown and the expiry path at this boundary; the current lease tests cover leave/revoke and publication, not shutdown.

### P2 — Preserve text edited while an earlier message is awaiting its ACK

`apps/web/src/chat-ui.ts:130–142`, especially the unconditional `input.value = ''` at 141.

The textarea remains editable while `sendChat` is pending; only the send button is disabled. If the user submits A and types B while the network/storage request is pending, A's successful ACK clears the current textarea and drops B. The operation can legitimately wait several seconds. Clear the field only if it still contains the submitted text (and preserve/reset draft identity consistently), or deliberately disable editing until the request settles. Add a focused controller test with a deferred ACK and an intervening textarea edit. Generation checks protect world switches but do not protect edits in the same attachment.

### P3 — Make the authored acceptance actually trigger keyboard input and a completed reconnect

`apps/web/tests/social.e2e.spec.ts:21–23`.

The typing assertion uses `input.fill`, which does not exercise the application's movement `keydown` path; it can pass even if typing WASD moves the avatar. The reconnect assertion toggles offline immediately back online and then accepts the already-present `Online` label and existing single message. It never establishes that the socket disconnected or resumed, so it can pass without testing resync. Use real keyboard events, and require an observable connection interruption followed by a new successful connection/snapshot before asserting deduplication. This is a defect in the future acceptance source, separate from the explicitly authorized NOT_RUN status; do not run it until the user authorizes the credential flow.

## Strengths and specification evidence

- Application wiring reuses existing Auth, Worlds and Hub instances, preserves auth/world postcommit revocation callbacks, and introduces no second identity/role system. No DM, public directory, or fake table membership is added.
- Friendship pair locks cover requests, response/block, and direct invitation initiation. Both-direction blocks are checked. Presence is populated only for accepted relations and contains no private world identity. Targeted invitation acceptance is bound to the target and shares the existing redeem transaction; an untargeted invite UUID cannot substitute for its secret token.
- Chat validation is consistently world-only and 500 Unicode codepoints across Go, TypeScript and contract changes. SQL uniqueness gives durable per-world/user/client-message deduplication. The account bucket refills one token/second with capacity five. History has stable descending-ID pagination capped at 50 and 30-day filtering, with startup/hourly physical cleanup.
- Storage completion precedes live publication and successful result ACK. The integration test injects a real deferred COMMIT failure and asserts no row and no event, then retries the same client ID. Duplicate lookup returns the stored message rather than substituting retry text.
- Shared leases cover chat authorization/storage/publication; exclusive control leases cover mute/delete storage/publication. Kick retains postcommit revocation outside that control gate. Runtime ownership code contains no new database/network calls. Delivery queues cap at 64 per seat, overflow fences that seat and uses reconnect/resync; other worlds are filtered. The shutdown finding is an exception to otherwise consistent lease-before-hub-mutex ordering.
- Moderation checks active actor roles, protects the owner, permits moderator peers under the recorded policy, persists mute expiry and audits, and broadcasts delete tombstones. Plain-text rendering uses `textContent`. Client merge uses exact decimal IDs and deletion tombstones for history/live/ACK reconciliation.
- Chat attachment follows actual local world attachment, not the retained cleanup socket. The carried T5 redundant-detach/solo-route issue is guarded and has a focused route-continuity assertion.

## Verification evidence and limitations

The report's claimed commands were evaluated against the staged test source, not rerun or independently recertified as successful:

- Reported full `go test -race -tags=integration ./... -count=1 -timeout=120s` predates the last ordering refinement, correctly disclosed. The reported final scoped race command covers the changed lease/control/publication, chat persistence, moderation and application transport test names.
- The staged backend tests substantiate friend accept/reject/unfriend/block and targeted invite behavior, 500-codepoint boundaries, outsider/unjoined/muted denial, durable duplicate ID, five-token exhaustion, history pagination/cleanup, rollback/no publication, owner/peer moderation, queue isolation, resume and transport ACK/kick handling. The bucket test itself exercises one account/world; the global scope follows from the account-keyed SQL and lock, not a cross-world exhaustion assertion.
- The three unit-package command, realtime suite, Node suite, typecheck/lint/build, vet/build and diff check are reported passes. Source review confirms the relevant Node additions cover exact message IDs/tombstones, correlated ACK/error results, no send after close, and solo-route continuity. They do not mount the full chat form or exercise an edit during a pending send.
- Authenticated E2E source is authored but NOT_RUN. Coordinator separately reports guest-only preview evidence; this review neither inspected nor claims to have executed that evidence. It cannot substitute for authenticated multiplayer acceptance.
- No additional focused test was needed to establish the two implementation findings: both follow directly from the lock wait cycle and asynchronous form state transitions. No broad suites were rerun.

Additional unchanged context was read only for named risks: the remainder of `Hub.Close`/`Validate` to resolve the lease shutdown/removal paths; auth `RequireSession`/`Logout`/`Recover` to check session locking and postcommit callbacks; realtime reader control-rate/leave handling to check transport boundaries; runtime request/close implementation to check publication/shutdown interactions; and a search confirming the existing `data-position` instrumentation used by acceptance. No further architectural defect was established from those reads.

Recommended disposition: fix P2 findings and address the small acceptance-source correction, then perform a focused T6 rereview. Preserve AUTH_BROWSER_NOT_RUN until explicitly authorized. Whole-branch review and integration remain coordinator-owned.
