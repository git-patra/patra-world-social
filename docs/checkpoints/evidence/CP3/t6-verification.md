# T6 implementation report — ready for coordinator review

Scope: friends, targeted in-app invitations, persistent world chat, moderation and reachable UI. Source baseline 10aab4c; coordinator docs-only commits through bb3eeaa do not modify application baseline. No staging/commits/deployments/provider purchases or source-repo edits by implementer.

## Evidence so far

- RED `go test ./internal/realtime -run TestChatCommandBoundary -count=1`: 500-codepoint chat rejected by old protocol. Updated Go/TS/schema/golden to exact `{channelId:"world",clientMessageId,text}`, max500 codepoints and strict unknown fields. Focused GREEN, including golden commands.
- RED isolated `go test -tags=integration ./internal/application -run TestSocialRoutesInstalled`: `/api/v1/friends` returned404. GREEN after shared Application/CSRF/session wiring. Initial sandbox loopback denial retried with approved escalation; no secret env values printed.
- GREEN isolated synthetic PostgreSQL tests `go test -tags=integration ./internal/application ./internal/social -run 'TestSocialRoutesInstalled|TestFriendLifecycleAndBlockedInvites|TestChatPersistenceModerationIsolationAndResync' -count=1`: request/reject/accept/unfriend/block; accepted-only presence; blocked exact search/direct invite; bound notification acceptance and duplicate acceptance; public UUID cannot replace token; joined/member/mute checks; durable dedup; five-token global account bucket; 50-message keyset pages; delete tombstone; 30-day cleanup; queue overflow/resume; other-world isolation. Chat integration test subsequently moved into internal/chat; run updated paths before final.
- GREEN focused lease ordering/queue isolation: same-seat leave/revoke waits for active lease; runtime ticks advance while lease held; future send rejected; 64-element queues and overflow terminate affected seat only.
- GREEN `go test ./internal/realtime -count=1 -timeout=40s`: 9.132s including existing T5 tests and WebSocket transport checks.
- RED `npm -w apps/web test`: carried cleanup recovery erased active solo route. GREEN after stop guards attachment: Node/map suite, 22 map/controller/protocol/socket/chat tests (plus prior JS tests). Added durable ACK correlation/error/no-send-after-close tests and exact-ID/deletion/dedup model tests.
- GREEN `npm -w apps/web run typecheck` after initial UI integration.

## Interfaces / lock order

Application owns Auth, Worlds, Hub, Social, Chat, Moderation and hourly/startup retention cleanup. Existing auth/world postcommit revoke callbacks preserved. `Social.RequestFriend`, `RespondFriend`, `Chat.SendMessage(ctx,realtime.JoinContext,channelID,clientID,text)`, `Moderation.Moderate` use shared identity/membership/roles.

`Hub.WithJoined` acquires independent lease read gate, runs current authorization and a short3s storage callback. Join/resume mutation, explicit leave, disconnect, revocation and expiry take write gate before hub mutex. Admission authentication remains outside gate and generation-fenced to preserve T5 revocation semantics. Chat transaction locks world before session, then verifies active member/ban/mute and serializes account token bucket; commits before callback/publish. T4 moderation callbacks execute after transaction commit, without holding DB locks. Runtime ticks never acquire lease/storage locks.

`Hub.Execute` is durable/control command acknowledgement seam (returns stored result); `Hub.Publish` enqueues bounded world-scoped delivery; `Events` exposes per-seat64 queue. Overflow closes with1013/resync_required so socket reconnects; replacement world snapshots invoke persisted history resync. Movement snapshots remain replaceable. Future game command/event handlers can use Execute/Publish without pretending snapshot replay is durable.

Targeted notifications hold invite UUID plus existing target identity, no raw token persistence. `AcceptTargetedInvite` shares transactional redeem implementation, and rejects untargeted UUID acceptance. Social pair advisory lock serializes block, new request and direct invite initiation; block does not remove world membership or mute shared chat.

## Current limits / pending verification

- Browser credential flows explicitly deferred by user. No browser credential entry, injected sessions or workaround; multi-context browser acceptance must be authored and marked NOT_RUN.
- Table chat remains unavailable until actual T13 table-seat authorization; only `world` channel accepted.
- Lease gate is application-global: slow storage can delay admission/revocation by bounded operation duration, while runtime ticks/movement continue. This is a beta correctness boundary, not production throughput evidence.
- Final verification and self-review completed below. No staging, commit or public deployment by implementer.


## Final implementation and review

Reachable UI: nav Teman opens exact username search, friend request accept/reject/unfriend/block/unblock, accepted-friend online availability, targeted-invite notifications and acceptance. Accepted friends have Undang → choose a world where actor is owner/moderator → in-app notification → Terima dan masuk. Existing private-world panel remains the link-invite/create/join/member-role/kick/ban UI and now includes timed chat mute presets900/3600/86400 seconds, visibly labeled product defaults. Owner protection and moderator-peer policy enforced by backend.

Chat nav opens world-only plain-text transcript, unread badge, retry-preserving draft ID, 50-message older-page navigation and Pesan terbaru, loading/error notices and delete controls. The view retains up to500 live entries; every requested older history page remains reachable because navigation replaces the displayed page while preserving live messages arriving during its request. Message text is assigned with textContent; usernames interpolated into social HTML are escaped. Submit success requires a matching durable command ACK containing the actual stored Message. A failed/unconfirmed send retains draft/clientMessageId, allowing explicit retry without a second stored row. Attachment generation fences old-world results and events; pending cleanup sockets never attach chat. Reconnect world.snapshot explicitly reloads persistent history; request generations discard stale history responses. Typing inputs rejects movement keys and focusing a field releases held directional input.

Migration003 adds friend pairs/blocks, world messages and unique durable client IDs, timed mutes, moderation audit and persistent account token buckets. Read queries use sqlc-generated helpers from `db/queries/social.sql` and `chat.sql`; transactional writes stay explicit SQL like T4 to preserve world/session/pair lock ordering. Ran `/opt/homebrew/bin/sqlc generate` (v1.30.0); generated models/social.sql.go/chat.sql.go included. Retention cleanup runs at app startup and hourly with a3s deadline, hides expired history immediately via query, and emits a generic observable error on cleanup failure without logging message bodies or secrets.

Self-review found a delete/publication race after the initial full suite. Final concrete ordering: Chat.WithJoined holds the shared lease through COMMIT and message enqueue (including duplicate retry publication). Mute/delete take Hub.WithControl's exclusive lease before DB work and hold it through their COMMIT/control enqueue. Thus deletion cannot overtake a stored message still awaiting publication; two chat readers can publish different messages out of ID order, and the UI sorts exact decimal IDs. Kick stays outside WithControl and uses existing world postcommit revoke callback, avoiding gate reentry. The UI additionally applies remembered deletion tombstones to history, late live messages and ACK results, so an already-received delete cannot resurrect text. Queue overflow uses1013/resync_required, preserving automatic reconnect rather than treating backpressure as revoked access.

## Final verification evidence

Commands run from repo root unless specified; DATABASE_URL sourced/exported from ignored `.cache/database.env` without printing it. All Go commands used `GOCACHE="$PWD/.cache/go-build" /opt/homebrew/bin/go -C apps/server ...`. Loopback-dependent tests used approved escalation against dedicated55432 and randomized schemas cleaned by test cleanup. No preview accounts used.

1. `go test -race -tags=integration ./... -count=1 -timeout=120s` — PASS all packages. cmd/server1.687s; application19.648s (includes existing T5 real20-client smoke/revocation/reconnect); auth7.525s; chat2.294s; config3.469s; games/model4.451s; http5.087s; moderation4.224s; realtime12.195s; social3.832s; world3.018s; worlds3.409s. This full run preceded final delete/publication ordering refinement; relevant final packages were rerun in item2.
2. Final scoped `go test -race -tags=integration ./internal/realtime ./internal/chat ./internal/moderation ./internal/application -run 'TestJoinedLease|TestWorldEvent|TestDeleteControl|TestChatPersistence|TestOwnerProtection|TestSocial' -count=1 -timeout=40s` — PASS: realtime1.797s; chat2.379s; moderation2.742s; application3.467s. Covers lease leave/revoke/runtime continuity, bounded world-isolated queues, delete/control publication order, actual persistence/rate/mute/delete/page/retention/resume, owner/peer audit rules, actual HTTP CSRF/origin/unknown-field checks and real WS durable result ACK/live delivery/resume/kick.
3. `go test ./internal/social ./internal/chat ./internal/moderation` — PASS; social/moderation DB tests require integration tag and were exercised above. Chat unit validation passes.
4. `go test ./internal/realtime -count=1 -timeout=40s` — PASS9.132s before final ordering refinement, preserving T5 admission/revocation/leave behavior. Final relevant ordering/lease tests covered in item2.
5. `npm test` — PASS31 total (8 existing JS tests +23 bundled map/controller/protocol/socket/chat tests). Final new RED/GREEN: delete-before-live-message test initially failed `deleted !== true`; after tombstone-aware merge it passes for the same source used by history/live/ACK reconciliation. Carried T5 route cleanup RED/GREEN and exact ACK matching remain green.
6. `npm run typecheck`, `npm run lint`, `npm run build` — PASS. Final production build151 modules, JS329.74kB (106.56kB gzip), CSS56.20kB (13.54kB gzip). Build is a local artifact only.
7. `go vet ./...`, `go build -o ../../.cache/t6-server ./cmd/server`, `git diff --check` — PASS. Vet rerun after final backend ordering refinement; final changed packages compiled/executed under race in item2.
8. Browser acceptance `apps/web/tests/social.e2e.spec.ts` — AUTH_BROWSER_NOT_RUN. Source covers three UI contexts, friend request/in-app invitation/join, typing does not move, plain text/no script node, world isolation, unread badge, reconnect deduplication, delete/mute/kick. Existing auth/world browser sources also remain unrun. No CUA/Playwright credential entry, injected browser sessions, alternate credential workaround or browser automation run occurred.

Additional actual failure evidence: deferred PostgreSQL COMMIT trigger raised a synthetic error after message ID allocation. Send returned error, stored row count stayed0 and event queue size did not change; after removing trigger, retry of the same clientMessageId succeeded. New transport emits `result` only when execution succeeded. This exercises real rollback rather than simulating a successful persistence call.

## Remaining limits and concerns

- No unresolved architectural blocker. Coordinator should independently review shared-lease lock ordering and protocol/UI wiring before integration.
- Global lease bounds storage work at3s and keeps simulation ticks/movement independent, but sustained chat/moderation/storage stalls can delay admissions and revocations. This is a single-application beta correctness design; no multi-instance event bus/production-scale throughput claim.
- Presence means an actively connected Hub world seat, shown only to accepted friends, without revealing private world IDs/names. Friends/invite inbox refreshes badges every10s and exposes manual refresh; it is not a persistent DM feature.
- History is retained30days and paginated50. Physical cleanup runs hourly; query filtering prevents expired content exposure between cleanup passes. This does not implement backup erasure policy.
- Table chat authorization/ephemeral retention remains deferred to T13 real table seats; channel validator rejects table channels now.
- Browser credential acceptance remains NOT_RUN by explicit user instruction. No visual browser/keyboard multi-context success claim. Frontend checks are Node/controller/model/socket tests plus build/typecheck; browser source is reviewable only.
- Existing preview Vite4176 and Go8080 processes were not stopped or restarted. No remotes, push, deploy, account/provider purchase, source-repo edits, root docs/PROGRESS/ACCEPTANCE changes, or commits by implementer.

## Changed files

- `apps/server/db/queries/chat.sql`
- `apps/server/db/queries/social.sql`
- `apps/server/internal/application/application.go`
- `apps/server/internal/application/social_integration_test.go`
- `apps/server/internal/application/social_transport_integration_test.go`
- `apps/server/internal/chat/service.go`
- `apps/server/internal/chat/service_integration_test.go`
- `apps/server/internal/chat/service_test.go`
- `apps/server/internal/db/chat.sql.go`
- `apps/server/internal/db/migrations/003_social.sql`
- `apps/server/internal/db/models.go`
- `apps/server/internal/db/social.sql.go`
- `apps/server/internal/http/router.go`
- `apps/server/internal/http/social.go`
- `apps/server/internal/http/worlds.go`
- `apps/server/internal/moderation/service.go`
- `apps/server/internal/moderation/service_integration_test.go`
- `apps/server/internal/realtime/chat_protocol_test.go`
- `apps/server/internal/realtime/connection.go`
- `apps/server/internal/realtime/delivery.go`
- `apps/server/internal/realtime/delivery_test.go`
- `apps/server/internal/realtime/hub.go`
- `apps/server/internal/realtime/protocol.go`
- `apps/server/internal/social/service.go`
- `apps/server/internal/social/service_integration_test.go`
- `apps/server/internal/testutil/database.go`
- `apps/server/internal/worlds/invites.go`
- `apps/server/internal/worlds/membership.go`
- `apps/server/internal/worlds/service.go`
- `apps/web/index.html`
- `apps/web/src/app.js`
- `apps/web/src/chat-model.ts`
- `apps/web/src/chat-ui.ts`
- `apps/web/src/net/api.ts`
- `apps/web/src/net/socket.ts`
- `apps/web/src/online-ui.ts`
- `apps/web/src/social-ui.ts`
- `apps/web/src/social.css`
- `apps/web/src/world/avatars.ts`
- `apps/web/src/worlds-ui.ts`
- `apps/web/tests/map.test.ts`
- `apps/web/tests/social.e2e.spec.ts`
- `contracts/commands.golden.json`
- `contracts/event.schema.json`
- `contracts/protocol.schema.json`

This report itself: `.superpowers/sdd/2026-10-26-patra-world-social/task-6-report.md` (ignored task evidence).

## Review fix round1 — both P2 findings addressed

Read `task-6-review.md`; verified both findings against the reviewed staged source. Used the receiving-code-review/TDD workflow. All fixes remain unstaged; no staging/commits, root-doc changes, preview restarts, browser credential actions or subagents. The SHA-256 of `git diff --cached --binary` before/after this round matches, confirming the reviewed index was preserved.

### P2 shutdown/expiry deadlock

Changed `Hub.Close` to release `h.mu` **and** `h.lease` before waiting on `h.done`. Extracted the unchanged expiry critical section into `expireTick`, shared by the actual ticker worker and regression test. The test uses a room-shutdown channel gate to hold Close at the real lock boundary, then drives the already-selected expiry branch through the actual production `expireTick`. Only after its selected-tick handshake does it release room shutdown. This does not depend on a real ticker/scheduler race or a luck loop. It also verifies a second/repeated Close completes against the same stopped worker.

- RED: `GOCACHE="$PWD/.cache/go-build" /opt/homebrew/bin/go -C apps/server test -race ./internal/realtime -run TestCloseReleasesLeaseBeforeWaitingForSelectedExpiryTick -count=1 -timeout=10s` — FAIL, `Close retained lease while waiting for selected expiry tick` (1.00s boundary timeout; package1.762s). At this point the expiry extraction preserved the reviewed behavior and Close still retained its deferred lease unlock.
- GREEN after explicit unlock: same focused command — PASS1.554s.
- Final affected-package race regression: `GOCACHE="$PWD/.cache/go-build" /opt/homebrew/bin/go -C apps/server test -race ./internal/realtime -count=1 -timeout=40s` — PASS9.909s, including the final deterministic boundary/concurrent-repeat test and existing T5/T6 admission, movement, revocation, lease ordering and loopback transport cases. Approved loopback escalation used; no database/browser credential flow needed.

### P2 preserve edits during pending chat ACK

The successful form submission now clears the textarea only when its current value still equals the submitted text. It resets the successfully consumed draft identity, so edited text remains and gets a new clientMessageId on its next submission. Failure keeps an unchanged draft and its existing clientMessageId for explicit retry. Existing attachment-generation fencing still prevents an old-world completion from changing the new world's field, transcript or pending send state.

Three new tests mount **the actual mountChat/form controller** with lightweight test DOM elements and a deferred transport boundary: A submitted/edit B before ACK; rejected A/retry same durable ID; and A pending/switch to world B/start B/old ACK arrives. They execute the real form handlers; they do not test a duplicate implementation of the clearing logic or invoke a browser/credential workaround.

- RED: `npm test > .cache/t6-review-draft-red.log 2>&1` — the edited-draft test failed `ACK for A must preserve the newly edited draft B`; failure/retry and switched-attachment tests passed under the reviewed code.
- GREEN: `npm test > .cache/t6-review-draft-green.log 2>&1` — PASS34 total (8 JS +26 bundled map/controller/protocol/socket/chat tests), all3 new actual chat-form cases passing.
- Final `npm run typecheck`, `npm run lint`, `npm run build` — PASS. Build151 modules, JS329.76kB (106.57kB gzip), CSS56.20kB (13.54kB gzip); local artifact only.
- `git diff --check` — PASS. Reviewed staged-index digest comparison — unchanged.

### Round1 self-review and limits

Changed files relative to staged snapshot: `apps/server/internal/realtime/hub.go`, `apps/server/internal/realtime/delivery_test.go`, `apps/web/src/chat-ui.ts`, `apps/web/tests/map.test.ts`, plus this ignored report. Shutdown retains lease-before-mu ordering during mutation; once closed, pending admissions still fail the existing closed/revision checks. Both locks are free while waiting for worker completion. No new production test hooks or credentials were introduced. Form success clears an unchanged submitted value, preserves intervening edits, consumes only the successful draft's ID, and keeps generation checks before rendering or altering the form. Deferred failure/retry and attachment switch behavior are covered.

No broad passing integration suite was repeated. P3 authored-E2E keyboard/offline observation weakness remains explicitly coordinator-deferred until acceptance/T16, as instructed for this fix round. Browser acceptance remains AUTH_BROWSER_NOT_RUN; no browser actions were executed. Both blocking P2 findings are ready for coordinator rereview.
