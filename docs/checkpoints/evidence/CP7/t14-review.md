# T14 review

Reviewed 27 October 2026 against `task-14-brief.md`, including its binding rulings, and `task-14-review-context.md`.

Base/HEAD: `658bd09271e31d6b8f86333fb7d407a0d4aaea21`.
Candidate staged tree: `12e80daddd7f66540037528848eceeb44945e18a`.
Read-only `git rev-parse HEAD` and `git diff --cached --quiet 12e80daddd7f66540037528848eceeb44945e18a` confirmed these identities; the latter exited 0.

## Verdicts

**Spec compliance: CHANGES REQUESTED.** Most implementation requirements are present, but account deletion does not prevent all post-deletion relationship/membership writes, account/device settings are not consistently isolated, and stale Settings mutations can overwrite a newer client auth state. These are implementation gaps independent of external acceptance. The required real-browser 100-record boundary and authenticated guest → login → online world → logout/account lifecycle acceptance remain NOT_RUN. This review does not mark T14 complete or CP7 PASS.

**Code quality: CHANGES REQUESTED.** The immutable-result/projection approach, account lock around Persist, transactional leaderboard hook, durable voice cleanup retention, bounded admission fence, and atomic IndexedDB writes are appropriate. The important findings below need repair. Existing passing tests do not exercise these interleavings or preference paths.

Counts: **0 Critical, 4 Important, 1 Minor**.

## Critical findings

None identified in the reviewed scope.

## Important findings

### I1 — Fence REST membership/social writers against account deletion

**Changed location:** `apps/server/internal/auth/account.go:170` and `:172` (membership deactivation and friendship deletion).

**Supporting locations:** `apps/server/internal/worlds/service.go:265`, `:312`; `apps/server/internal/social/service.go:54–78`; `apps/server/internal/http/worlds.go:25`, `:125`.

**Trigger:** An invite-redeem request passes session middleware before deletion, then waits for the target world's row lock. DeleteAccount commits while it is waiting. A general invite issued by another account is not revoked by this deletion. When redeem continues, it checks the world, invite, ban and capacity, but never takes the deleting user's account lock or checks `users.deleted_at`. It can insert/reactivate `world_members(active=true)` after the deletion's UPDATE. The retained users row makes the FK valid. A friend request has an analogous window: Search obtains a live target, the operation waits on its pair lock, deletion clears relations, then RequestFriend inserts a new relation without a tombstone recheck. A request already authorized as the deleting actor can also finish after deletion.

**Impact:** A deleted account can reappear as an active world member or friend request, consume one of the 50 membership slots, and consume a single-use invite after deletion reports success. Member/friend listings can expose the operational tombstone identity. Session rejection prevents a later login, but does not undo these in-flight REST writes. The hub admission lease protects realtime operations, not these REST transactions.

**Required repair:** Compose the account deletion fence with the actual relationship/membership writers: acquire appropriate ordered account locks and recheck tombstones inside the writing transaction, with an audited order relative to world/pair locks. Ensure both deleting actors and targets are covered where relevant. Do not rely only on initial middleware or username search. Add deterministic blocked-writer tests for redeem and friendship creation on both sides of deletion; assert no active membership/new relation remains and no invite is consumed by a tombstone.

**Evidence:** Static, concrete transaction interleaving traced through the existing production REST route and writer code. No database probe or browser credential action was performed. The new late-Persist test covers game participant insertion only and does not cover these writers.

### I2 — Keep tombstone usernames outside the public signup namespace

**Changed location:** `apps/server/internal/auth/account.go:178`.

**Supporting locations:** `apps/server/internal/db/migrations/001_identity.sql:3`; the reviewed auth service's public username validation.

**Trigger:** Register the valid 24-character username `deleted_` plus the first 16 hex characters of another account's public UUID before that account is deleted. The deletion UPDATE deterministically chooses exactly that username. Public signup accepts this prefix, and `users.username` remains globally UNIQUE. No conflict fallback or reserved namespace is introduced.

**Impact:** The unique violation rolls back the whole account deletion, including leaderboard and identity erasure. The application fence may already have disconnected the user. Anyone who knows the target's exposed account UUID can reserve this one name and prevent deletion through the UI/API until the conflicting account is changed externally. This is not dependent on a random UUID collision.

**Required repair:** Use a collision-safe operational identity that public registrations cannot claim, with migration/validation rules that also handle existing names. Preserve the unusable-credential/deleted-at checks. Add a deletion test where the currently generated tombstone name is already occupied and verify that deletion still commits without changing the unrelated account.

**Evidence:** The deterministic new UPDATE, existing UNIQUE/CHECK constraint, and permitted signup pattern establish the failure directly. No account was provisioned or modified during review.

### I3 — Keep account and device preference changes in their own scope

**Changed locations:** `apps/web/src/app.js:147`; `apps/web/src/features/settings/SettingsPage.tsx:24`.

**Related location:** `apps/web/src/app.js:140` applies the entire preference object, including identity, controls and audio.

**Trigger A:** Log in with account settings that differ from guest settings, then use the main Motion button. Account profile loading only calls `applyPreferences`; the `preferences()` accessor still contains the device profile. The Motion handler always calls `savePreferences({...preferences(), reducedMotion: motion})`, even when authenticated. Its event reapplies all guest fields to the current UI. For example, an account using touch controls and muted audio can suddenly get the device's keyboard/both controls and audio-on state. The toggle persists to IndexedDB instead of the account, so it is lost from the account view on reload and also changes the guest's saved preference.

**Trigger B:** While the account Settings page is open, choose “Hapus data lokal”. After clearing device storage the handler unconditionally does `setP({...defaults})`; reset also emits the device preference object to the whole application. Thus the account settings form becomes Guest/Labu/default controls and audio even though account settings were not reset on the server. A later ordinary Save writes those defaults over the account profile.

**Impact:** Account/device preferences and identity cross the exact boundary the migration UI is intended to preserve. A visual-motion toggle unexpectedly changes unrelated settings; a local-only reset prepares an unintended account-profile overwrite. The guest-only CUA run cannot detect either case.

**Required repair:** Track the active preference scope explicitly. Route the global toggle to that scope, update only the intended setting, and preserve the account form/effective account preferences when clearing device data. Continue copying local avatar/preferences only through the explicit copy action. Cover distinct guest/account settings, account Motion toggling, logout/reload, and local reset while authenticated with focused frontend state tests; execute the credential browser case only once the existing deferral is lifted.

**Evidence:** Direct event/state trace through the new handlers and `guest/history.ts` preference accessor/save/reset functions; no credential browser workflow was run.

### I4 — Fence Settings mutation callbacks by the current auth generation

**Changed locations:** `apps/web/src/features/settings/SettingsPage.tsx:12`, `:19`, `:22`, `:27`; `apps/web/src/auth-ui.ts:44`; `apps/server/internal/http/account.go:83`.

**Supporting locations:** `apps/web/src/auth-ui.ts:13` (`setUser` broadcasts auth state); `apps/web/src/net/api.ts:19–20` (mutation fetch has no cancellation signal).

**Trigger:** Account A starts a profile save and its successful response is delayed. The user closes Settings, logs out, and logs into B before A's response arrives. Closing Settings unmounts the React root, but does not cancel or invalidate the outstanding `run` closure. After await, it still invokes its captured `onUser({...user,...updated})`. That callback dispatches `patra-account-updated`, whose listener unconditionally replaces the current auth state via `setUser`. The old password-change/delete success handlers similarly dispatch `onUser(null)` after await with no check that their initiating session is still current. Only the initial profile-loading effect has a live guard; the mutations do not.

**Impact:** An obsolete profile response can restore A in the UI after logout, or replace B's client identity while the browser cookie remains B's. An obsolete password/delete callback can clear B's client state and trigger mode teardown. There is also a transport-level problem: successful password/delete responses unconditionally send `Set-Cookie` expiry for the shared session-cookie name. If such an old response arrives after B's login response set B's cookie, the browser expires B's session cookie before any JavaScript generation check can help. B's server-side session may remain valid, but the browser loses its credential. This violates the guest/login/world/logout cleanup boundary and can make displayed and effective account state disagree. `app.js`'s preference-load epoch does not prevent it: the stale mutation itself emits a fresh auth-change event.

**Required repair:** Capture an auth/session generation for each mutation and accept its auth-state update only if that generation still matches at the central auth consumer. Cancel obsolete requests where practical, but do not rely solely on React unmount or a user-ID equality check that misses logout/relogin to the same account. Guard related preference events as well. Address the response-cookie race separately: an obsolete password/delete response must not expire a subsequently established cookie. For example, rely on the existing server-side session revocation for these operations rather than an unconditional cookie-expiration response; a JS/DOM guard alone cannot fix browser cookie ordering. Add a deterministic deferred-response frontend test: begin A's save, switch to guest/B, resolve A's promise, and assert the newer auth state and mode remain intact; include stale null callbacks from password/delete and an HTTP response contract check for cookie behavior. These tests can be authored/executed independently of the deferred real credential browser flow.

**Evidence:** Root requested this specific additional cross-cutting check. Traced the already reviewed Settings mutation closures into the existing auth `setUser` and request helper. The path is unfenced in production code; no browser credential operation or test rerun was performed.

## Minor findings

### M1 — Finished-at timestamps exclude the race/Duck countdown

**Changed location:** `apps/web/src/guest/completion.ts:8`.

**Callers:** `apps/web/src/features/race/session.ts:28`; `apps/web/src/features/duck/session.ts:7`.

**Trigger:** Complete a race or Duck run. Completion records `createdAt` when Start is pressed, before the 3-second countdown. These callers supply gameplay duration measured from `startsAt`, after the countdown. Adding that duration to `createdAt` records a finish timestamp three seconds earlier than the actual scheduled finish.

**Impact:** Reload-stable timestamps are internally consistent but inaccurate; very close cross-game completions may be ordered or retained by the wrong finish time. Penalty starts immediately and does not have this specific offset.

**Suggested repair:** Anchor the wall-clock timestamp to the actual monotonic finish (including the start/countdown offset) or pass an explicit stable finished-at value. Keep retries idempotent and avoid persisting active monotonic clocks. Add a deterministic timestamp assertion for a countdown game.

## Coverage and evidence assessment

Read the complete packaged diff in manageable chunks, including new files and the pinned dev dependency. One tool response cut off the Penalty/Race hunks; only that cutoff interval was re-read to recover the omitted code. New/changed-file line references were mapped from the packaged diff. Root docs and evidence were outside the staged code diff as specified.

Supporting reads were limited to concrete cross-cutting questions: existing social/invite writers and REST session boundary for I1; username schema/validation for I2; auth state/request helpers for the coordinator-requested I4 check; local game clocks and public payload shapes for completion/redaction compatibility; voice callback wiring; and recorded verification outputs. No broad source crawl, suite repeat, live browser interaction, credential action, runtime restart, source/index/HEAD edit, remote action, or subagent was used. The only written file is this review.

| Requirement / risk | Implementation review and named checks | Remaining limits |
|---|---|---|
| Device-only history, dedup, oldest retention, reset | Actual GuestHistory transaction code has same-transaction append/dedup/retention and both-store reset; generation fences pending appends. Reviewed tests: `actual IndexedDB transactions retain newest 100 under concurrent append and reject duplicate replacement`, `actual IndexedDB settings and results survive fresh instance; atomic reset removes both`, `reset generation fences append waiting for IndexedDB open`, `failed IndexedDB writes reject, preserve existing records/settings, and permit retry`, `reset failure aborts both-store transaction without partially deleting history`. | These execute actual app storage code against **fake-indexeddb**, not a real browser. Real-browser 100-record fixture NOT_RUN. |
| Three local games and reload behavior | Reviewed completion/session integration, unchanged local engine rule sources as needed, and `mode cleanup removes all three engines and cannot resume stale monotonic clocks`. Existing golden suites were included in reported npm checks. CUA narrative documents real completed Penalty/Duck/Race results, reopened Duck dedup, reload preservation and interrupted-run abandonment. | Review did not independently run CUA or golden suites. M1 concerns timestamp accuracy, not reload stability. |
| Guest cannot promote scores or obtain online media/socket capability | Completion imports storage/contracts only; explicit account copy sends avatar/preferences, no result. Account strict profile decoding rejects score-import field in `TestAccountRoutesRequireSessionCSRFOriginAndPassword`. | Source-authored network-observation E2E remains unexecuted. The guest CUA narrative does not separately claim an exhaustive network audit. |
| Local reset with scoped user approval | `docs/checkpoints/evidence/CP7/t14-guest-preview.md` documents approved deletion of exactly three synthetic records/preferences and reload showing empty history/default guest settings. | Does not prove account-mode reset semantics (I3) or 100-record browser boundary. |
| Credential/session lifecycle | Reviewed tombstone exclusions on login, session refresh, recovery and credential-change row locks; exact-origin/CSRF/strict decode/rate-limit wiring. Named tests: `TestAccountDeletionAndCredentials`, `TestAccountSettingsPasswordAndRecovery`, `TestAccountRoutesRequireSessionCSRFOriginAndPassword`. | Authenticated browser account changes/deletion remain deferred. I2 prevents deletion for a valid collision input not covered by current tests. |
| Owner prerequisite / game Persist / admission ordering | Ownership preflight is repeated in the account-locked transaction; Create/Transfer fence the new owner; Persist takes sorted account locks. Named tests: `TestDeletionFencesPersistWaitingOnAccountLock`, `TestDeletionPresenceFenceBeforeAndAfterPreparedActivation`, `TestAccountDeletionFenceOrdersAdmissionDrainCommitAndFailure`. | These do not cover concurrent REST relationship/member writers (I1). Schedule is correctly not treated as an activation gate. |
| Immutable results, opponent identity, board recomputation | Reviewed participant projection redaction and final WS serialization, startup-loaded tombstone registry, exact transactional `leaderboard.RemoveAccount` hook, board invalidations after commit. Named tests: `TestDeletedAccountPendingResultHTTPAndLatePersist`, `TestDeletedAccountActualWebsocketSerializationAndRollback`, `TestAccountDeleteRecomputesBoardAndExcludesPendingProjection`, `TestKnownPublicShapesAndNoArbitraryTextRewrite`. | Tests include real HTTP/WS serialization using isolated backend fixtures, not authenticated browser acceptance. Already delivered/in-flight serialized bytes are explicitly outside recall guarantees. |
| Voice cleanup | Deletion intentionally leaves the non-cascading voice revocation journal; existing application seat callback invokes voice revocation. Pending-result integration test asserts a journal row survives deletion. | T13 external provider/device acceptance remains pending; no LiveKit provisioning or native/mobile acceptance inferred. |
| Guest → login → world → logout cleanup | Reviewed event-driven clearing of engines, modal disposal/input release, online snapshot/clock-buffer reset, and authored `account-mode.e2e.spec.ts`. | Account-mode browser source is **NOT_RUN** and does not substitute for the required acceptance. Current unit cleanup test asserts engine clearing, not a complete browser timer/listener/media inventory. I3 and I4 remain. |

## Verification records inspected

- `task-14-report.md` supplies the exact execution commands and known failures/corrections. Backend toolchain is the explicit local Go 1.26.8 binary, with local GOCACHE; system Go/global configuration was not changed.
- `.cache/t14-go-race.log` inspected tail shows passing package results including lobby, realtime, social, voice, world and worlds. Reported full command is `go test -race -tags=integration -p 1 -count=1 ./...`; serial package scheduling is deliberate. This review did not rerun it.
- `.cache/t14-go-vet.log` and `.cache/t14-go-build.log` are empty success-style logs; successful process exits are reported by the implementation report, not independently provable from empty log text alone.
- `.cache/t14-indexeddb-tests.log` inspected summary shows 79 TypeScript tests passed, zero failures. Report also records four JavaScript tests. Earlier `.cache/t14-web-test.log` summary contains 74 TypeScript passes before addition of the five storage tests; it is not the final total.
- Final lint/typecheck logs contain their invocations and no diagnostics; the build log ends in a successful Vite build. The lint script shown is `node --check src/app.js && node --check src/world.js`, so this is syntax validation of those two JS files, not a comprehensive TS/React style linter. Typecheck and test coverage supply separate evidence.
- Read the full CUA evidence narrative `docs/checkpoints/evidence/CP7/t14-guest-preview.md`. It documents guest-only frontend testing against the T14 Vite tree while the API remained accepted T13. Screenshots were listed by that narrative, not independently reinterpreted in this review. No authenticated T14 route execution is inferred from that browser session.

Implementation readiness requires resolving I1–I4 and focused regression coverage. External acceptance remains separately pending even after those repairs; no local test result authorizes a T14-done or CP7-PASS claim.
