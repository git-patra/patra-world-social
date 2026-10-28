# T14 implementation report

Status: DONE_WITH_CONCERNS. Implementation and required local checks complete; final acceptance and commit are coordinator-owned. No taskdone or CP7 PASS claim. Authenticated browser workflows and the real-browser 100-record boundary remain NOT_RUN. Node tests of the actual IndexedDB transaction code at the 100-record boundary passed. Round 1 repairs below are ready for scoped reviewer recheck; this is not final acceptance.

## Scope and interfaces

- Added `guest/history.ts`: IndexedDB `patra-guest-v1`, `results` keyed by unique local match ID and `settings` stores. `GuestHistory.append/list/reset/settings/saveSettings` use transaction completion before success. Append is idempotent and trims oldest to 100 atomically; reset generations fence pending local appends. No transport or ranked import exists.
- Added `GuestSession.start(gameType, rules): LocalMatch`, shared local completion receipts, and lifecycle cleanup for authentication, world attachment, reset, and page exit. Existing race/penalty/duck local engines remain the source of results. Race, penalty, and Duck own and release their background timers. Finished sessions/settings persist; unfinished monotonic clocks do not survive reload. No online snapshot is converted to a local result.
- Guest entry uses “Main sendiri” / “Main dengan teman”, saved local name/avatar, local world nameplate/avatar colors, device badges, and persistent history. Expanded local history shows Indonesian game names and game-facing outcome, opponents, laps/DNF, hits/shots/accuracy. Version metadata stays in stored records.
- Settings include audio preference, controls, reduced motion, explicit local avatar/preferences copying into account settings, password change, one-time recovery-code rotation, permanent delete confirmation, and link to existing owner transfer/archive UI. Only avatar/preferences can be copied; local scores remain device-only.
- Added account profile/settings routes, password/recovery reauthentication, and `DeleteAccount(ctx,userID,confirmation)`. Routes require session, exact Origin, CSRF, strict JSON fields, and mutation rate limits. Password change revokes sessions; recovery rotation invalidates the old code. Deleted users are explicitly excluded from credential/session/recovery paths and exact username discovery.

## Account tombstone, privacy, and ordering

Operational user tombstones retain the UUID needed by immutable match/job/host FKs, but erase password/recovery material, display/avatar/profile settings, and original username. They cannot authenticate. Deletion rejects live owned worlds until transferred or archived. Sessions and invites are revoked, memberships deactivated, social relations removed, authored chat removed, and participant identity projections anonymized. `voice_revocations` is deliberately untouched, preserving provider cleanup work.

`auth.Service.RemoveAccount` is injected as exactly `leaderboard.RemoveAccount` from application wiring, avoiding the existing `leaderboard -> httpapi -> auth` import cycle. It runs in the same deletion transaction; partition identities are captured before removal and invalidated only after commit. Persistent leaderboard exclusion prevents pending result projection from resurrecting personal bests. Prior champion identities are cleared, and administrative recomputation is not a victory.

Concrete order:
1. Validate username confirmation and ownership preflight.
2. Application takes the exclusive hub admission lease; revoke with the non-reentrant helper, release `hub.mu`, then synchronously drain permanent lobby departure via `g.Sync`.
3. Bounded deletion SQL takes account advisory namespace 12, then the user row, T12 ordered board rows, and participant/membership projections. No SQL executes under `hub.mu` or the lobby owner.
4. Commit, populate the public redaction registry, publish partition invalidations, release admission lease.

Persist takes match namespace 7 then sorted human account namespace 12 locks before checking tombstone/membership and inserting participants. Delete never takes a match lock. CreateWorld and TransferWorld also fence the new owner account before ownership writes. Schedule remains asynchronous metadata; it is not misused as an activation gate. Before-activation leave prevents a sole departed human from activating; after-activation leave reaches the running engine through the existing presence boundary. A failed deletion after admission fencing may disconnect the user, but rolls back durable account/board erasure and leaves credentials usable for explicit reconnect.

Immutable start/result/result-job/hash payloads remain unchanged so pending finalization and retry still bind. History HTTP serialization uses participant projection identities and suppresses erased user IDs. A startup-loaded tombstone registry redacts only known Actor and world-avatar identity shapes at final WebSocket serialization, including stale queued payloads; it does not mutate frozen engine objects or arbitrary chat text. Already serialized/in-flight/client-delivered data cannot be recalled.

## Executed verification

All Go commands use `.cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go`, `GOTOOLCHAIN=local`, and `GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build`. Integration commands source `.cache/database.env` without printing values. Synthetic schemas are created and dropped by testutil; no preview account/browser credentials were used.

- Baseline `npm test`: PASS, 4 JavaScript + 71 TypeScript tests, including existing shared golden fixtures for all three games.
- RED `go test -tags=integration ./internal/auth -run TestAccount -count=1`: expected compile failure for missing new account interfaces.
- First implemented `npm run typecheck`: failed on SettingsPage key prop and misplaced world JSDoc; both repaired and rerun successfully.
- First focused backend run under filesystem/network sandbox: blocked connecting to local PostgreSQL (`operation not permitted`). Authorized isolated backend run with escalated localhost access passed; no credentials printed.
- `go test -tags=integration ./internal/auth -run TestAccount -count=1`: PASS for profile, password/recovery, owner archive prerequisite, erasure, session/credential rejection.
- `go test -race -tags=integration ./internal/application ./internal/auth ./internal/identity -run 'TestDeletedAccount|TestAccount|TestKnownPublic' -count=1`: PASS for actual HTTP/WS identity redaction, immutable pending finalization/hash retry, retained voice cleanup journal, failed-delete rollback and revocation, and known-shape redaction.
- `go test -race -tags=integration ./internal/realtime ./internal/lobby ./internal/application -run 'TestDeletion|TestAccountDeletionFence' -count=1`: PASS for real admission lease ordering, lobby presence on both sides of activation, and blocked concurrent Persist rejecting after deletion.
- `go test -race -tags=integration ./internal/application -run TestAccountDeleteRecomputes -count=1`: PASS for champion recomputation, no false victory, and pending higher score exclusion.
- `go test -race -tags=integration ./internal/http -run TestAccountRoutes -count=1`: PASS for guest/CSRF/password rejection, strict rejection of score-import field, and deleted-session rejection.
- Required full `go vet ./...`: PASS (`.cache/t14-go-vet.log`).
- Required full `go test -race -tags=integration -p 1 -count=1 ./...`: PASS all packages (`.cache/t14-go-race.log`). Serial package scheduling is intentional per known chat fixture timing constraint; no filtered substitute for this full run.
- Required full `go build -o ../../.cache/t14-server-check ./cmd/server`: PASS (`.cache/t14-go-build.log`). Binary built only; runtimes not restarted.

- Required final `npm run lint`, `npm run typecheck`, `npm test`, `npm run build`: PASS (`.cache/t14-web-{lint,types,test,build}.log`). Frontend tests initially totaled 4 JavaScript + 74 TypeScript; after the requested maintained IndexedDB test implementation they total 4 JavaScript + 79 TypeScript. All new guest/settings/duck production sources and guest/account E2E TypeScript files are explicitly included in tsconfig. Final lint/typecheck/build reruns cover the small app.js initial preference-load race guard and added authenticated mode-flow E2E source (`.cache/t14-web-{lint,types,build}-final.log`); these are not browser execution.
- Coordinator-requested transaction test gap closure: `npm view fake-indexeddb version repository.url` verified maintained upstream 6.2.6; `npm install --save-dev --save-exact --workspace apps/web fake-indexeddb@6.2.6 --ignore-scripts` added one dev-only package (0 audit vulnerabilities). `npm test` then passed 4 JavaScript + 79 TypeScript tests, including all five new actual IndexedDB transaction tests (`.cache/t14-indexeddb-tests.log`). Final lint/typecheck/build reran after this test/dependency change; no Go rerun was needed because backend source was unchanged.
- `git diff --check`: PASS.

## Evidence boundaries and remaining review

- `guest.e2e.spec.ts` is executable source for real IndexedDB retention at 100 records, duplicate receipt, isolated test database reload/settings/reset, all three guest games, completion persistence, and absence of WS/score/media-token requests. It has NOT been executed by this implementer; browser execution belongs to the coordinator. Node tests execute the actual GuestHistory transaction code using pinned development-only `fake-indexeddb@6.2.6`: 120 concurrent appends retaining exactly newest 100, duplicate ID immutability, fresh-instance persistence/settings, atomic reset of both stores, pending-open append generation fence, non-cloneable write abort preserving old data, storage-denied errors, failed second-store reset rolling back both stores, and successful retry. Additional unit tests execute retention selection, completion retry/dedup, mode engine cleanup, and all existing game golden fixtures. This is test-implementation evidence, not a real-browser 100-record claim. Do not report the source-authored IndexedDB E2E as an executed 100-record browser test.
- Authenticated browser account lifecycle/guest-login-world-logout acceptance remains deferred. `account-mode.e2e.spec.ts` now authors guest completion → registration/login → empty official history with retained local history → world creation/join → logout → fresh local engine/no audio elements. It is NOT RUN. Existing source-authored authentication suites are not evidence of execution. Backend synthetic service/protocol tests above are independent allowed tests and not equivalent browser acceptance.
- Coordinator reports actual isolated guest UI verification of settings/name/reduced-motion persistence, Penalty 5:4 win with 14 kicks, Duck 0/1 hit, Race four BOT finishers and guest DNF, exactly three completed entries with reload-stable timestamps, and no duplicate Duck receipt when reopening. An unfinished Duck reload preserved only the three completed entries. Console warnings/errors were empty. One initial Duck run was interrupted by development HMR and appropriately discarded; subsequent games ran under a frontend freeze. Root owns the screenshots/narrative in docs/checkpoints/evidence/CP7. After explicit action-time user approval, root clicked the actual reset for the three synthetic records and Guest QA preferences. The UI confirmed deletion; actual reload verified empty History and default Guest/Labu/audio-on/both-controls/reduced-motion-off. This is browser reset evidence for three records, not the 100-record boundary. Real 100-record IDB retention and authenticated browser acceptance are still not executed evidence.
- No stage/commit, source repo changes, runtime restart, remote/push/deploy/provision/purchase, global toolchain/environment changes, or credential browser automation performed. Coordinator-authored CP7 evidence files are untouched.


## Self-review notes

- Rechecked accepted T12/T13 boundaries, immutable hash preservation, transactional rollback, late Persist fencing, and final public serialization. All targeted and required full backend checks passed with the exact commands above.
- Checked every new TS production/test file is covered by tsconfig. Guest completion cannot import into official APIs; account profile strict decoding rejects a supplied score. Reusable tests retain scoped synthetic storage/schema isolation.
- Inspected the old-named `components/PenaltyGame.jsx` before final review: it already renders the current `PenaltyDuel`, so the Arcade alias was correctly left unchanged. Both visible penalty entry paths share the persistent guest session.
- All new source is unstaged and uncommitted. No code/PROGRESS/checkpoint completion claims were written. Root-authored checkpoint evidence was not edited.
- Audio is a persisted preference; no new game soundtrack/effects or microphone access were introduced. Voice retains its existing explicit mute/permission controls. External provider/device acceptance remains governed by T13 evidence boundaries.


## Review repair round 1 — I1–I4 (27 October 2026)

Status: DONE_WITH_CONCERNS / ready for same-reviewer scoped recheck. All four Important findings have implementation and executed regression coverage. All repairs remain unstaged; the coordinator-staged candidate tree `12e80daddd7f66540037528848eceeb44945e18a` is preserved. No runtime restart, browser automation, credential workflow, subagent, stage/commit, or coordinator documentation/evidence edit was performed. The earlier “all new source unstaged” sentence describes the initial handoff; the coordinator subsequently staged that baseline before this round.

### I1 — actual REST writers share the account deletion fence

Added `identity/accounts.go:LockActiveAccounts`: canonicalize UUID spellings through PostgreSQL, sort/deduplicate canonical account IDs, acquire namespace-12 transaction advisory locks for every involved account, then check every account is still live in that transaction. Canonicalization is necessary because uppercase/compact UUID input can otherwise hash differently from the same stored UUID. Invalid UUIDs cannot acquire a usable alternate fence.

`worlds/service.go`, `membership.go`, and `invites.go` call the helper before world rows/pair/invite locks for create, invite issue, both redeem paths, role changes, member removal/ban, transfer, archive, revoke and unban. Redeem discovers immutable world and issuer IDs before locking, then fences redeemer plus issuer. Invite issuance fences actor plus optional target. `social/service.go` fences actor plus target for request, respond, remove/block and unblock, before pair locks; unblock now uses a transaction. No account lock is acquired after a world/pair lock. Deletion retains its existing account → user → ordered board/projection order and takes no world/pair lock. A writer that wins first completes before deletion cleanup; deletion that wins first makes the blocked writer fail before membership/invite/relationship mutation.

Executed tests in `worlds/deletion_integration_test.go`: `TestRedeemAndDeletionSerializeWithoutTombstoneMembership` and `TestInviteIssueFencesDeletingIssuerAndTarget`, both orderings with real blocked production writers, issuer/target cases, no active tombstone membership, no usable residual invite, and no invite consumption when deletion wins. `social/deletion_integration_test.go`: `TestFriendWritersFenceDeletingActorsAndTargets` covers actor/target in both orderings and no remaining friend/block relation; `TestBlockWriterCanonicalizesTargetAccountFence` proves a compact uppercase target cannot bypass the fence. `testutil/blocked.go` observes actual PostgreSQL wait edges rather than treating a sleep as proof of overlap. Existing `TestInviteConcurrencyAndRetry` now observes world-row → shared issuer-account wait edges; its single winner, capacity and same-account retry assertions remain intact.

### I2 — collision-safe reserved tombstones

Deletion now assigns exactly `~deleted:` plus the full UUID. New migration `012_reserved_tombstones.sql` replaces only the username check, first normalizes every already-deleted row, and allows the reserved form only when `deleted_at IS NOT NULL`; live signup remains exactly `^[a-z0-9_]{3,24}$`. Migration 001 and already-applicable 011 are unchanged. A live user holding the old `deleted_<prefix>` spelling remains unrelated and untouched.

Executed `auth/account_integration_test.go`: `TestDeletionCannotBeBlockedByClaimedLegacyTombstoneUsername` reserves the hostile valid old name, deletes its intended victim successfully, rejects reserved-form registration and retains the claimant's login; `TestReservedTombstoneMigrationNormalizesPreviouslyDeletedRows` simulates an existing 011-era deleted row and verifies migration 012 normalizes it while preserving another live account.

### I3 — explicit active preference scope and field-only Motion

New `preferences.ts:ActivePreferences` selects account or device storage from central auth state, publishes only for the captured session generation, and never applies device reset/save events to an active account. `app.js` subscribes to this active store instead of the device event; device events are separately named `patra-device-preferences`. Settings local reset clears device storage but preserves the account form and account effective preferences. Copying device avatar/preferences into an account remains explicit.

Motion uses a narrow `POST /api/v1/account/motion {reducedMotion}` route for accounts. It shares session, Origin, CSRF, rate and strict-decode protection, requires an explicit boolean and changes only that JSONB field atomically in a live user row, returning the fresh profile. Device Motion reads/modifies/writes only that field inside one IndexedDB settings transaction. Pending mutations observe reset generation fencing; normal settings writes also check that generation before writing. Responses from an obsolete auth generation are ignored.

Executed `auth/account_integration_test.go:TestMotionPreservesConcurrentProfileAndRejectsDeletedAccount` blocks a real field update behind a concurrent profile writer, verifies returned/stored name/avatar/audio/controls survive, and rejects a tombstone. `http/account_integration_test.go:TestAccountRoutesRequireSessionCSRFOriginAndPassword` now includes Motion guest/CSRF rejection, unknown/missing-field rejection and fresh preserved profile response. `tests/guest.test.ts` exercises actual fake-indexeddb read/modify/write preservation with concurrent settings save and pending Motion/save reset fencing. `tests/settings.test.ts` exercises production Settings with the lifecycle harness and real AuthState/ActivePreferences, account/device isolation, logout/reload, local reset followed by account save, real singleton reset with actual IndexedDB, fresh Motion profile propagation and stale same-account Motion completion.

### I4 — central session generation and independent HTTP cookie contract

New `auth-state.ts:AuthState` owns the current user and monotonically advancing session generation. Begin/transition advances the generation even for same-account re-login. `auth-ui.ts` uses it for initial session lookup, login, recovery and logout and no longer consumes unversioned `patra-account-updated` events. Settings captures generation, rejects stale profile/null/recovery/preference results, and aborts its requests on unmount or session remount. `api.ts` propagates AbortSignal through both CSRF lookup and mutation. Generation validation remains authoritative when cancellation is too late.

Password change, account deletion, existing logout and recovery keep their existing server-side revocation/reset but emit no unconditional session-cookie expiry. This separately prevents an older response from expiring a cookie installed by a later login. A revoked cookie may remain in the browser until replacement/expiry; it never authorizes a revoked session. Login still installs the normal session cookie. This is not a new token/session architecture, and JavaScript guards are not presented as a substitute for the HTTP fix.

Executed `http/account_integration_test.go` tests `TestAccountRoutesRequireSessionCSRFOriginAndPassword` and `TestLogoutAndRecoveryRevokeOldSessionsWithoutExpiringNewCookie`: no Set-Cookie on these revocations, old sessions rejected, recovery invalidates prior sessions, replacement authentication remains valid. Executed `tests/settings.test.ts`: central stale success/null after same-account re-login; actual Settings delayed profile response for A→guest/B/A with/without unmount; delayed password/delete null cannot end B/new-A; stale recovery code not displayed; aborted signals on unmount; late device-save result does not replace account preferences; actual auth consumer ignores legacy account events and obsolete initial `/me`. These are executed component/state tests, not authenticated browser acceptance.

### Commands, observed outputs, and failures

All Go invocations below run from `apps/server` with the exact local Go 1.26.8 binary and environment setup documented above (`set -a; . .cache/database.env; set +a` occurs at repo root before `cd`; values are never printed). No whole-Go-suite rerun was performed this round.

- RED account collision and cookie regressions, before fixes: `.cache/t14-r1-account-red.log` records `TestDeletionCannotBeBlockedByClaimedLegacyTombstoneUsername` failing with SQLSTATE 23505 `users_username_key`, and `TestAccountRoutesRequireSessionCSRFOriginAndPassword` failing with a password-change 204 response containing session expiry Set-Cookie.
- RED actual writer regressions: `sh .cache/t14-r1-writer-red.sh`, which temporarily restored only staged writer files with an EXIT restore trap and ran `go test -tags=integration -timeout=30s ./internal/worlds ./internal/social -run 'TestRedeemAndDeletion|TestFriendWriters' -count=1`. Expected failure: writer-first missing account wait edge; delete-first “tombstone redeemed invite” / “tombstone relation accepted” (`.cache/t14-r1-writers-red.log`). Fixed files restored; index unchanged.
- RED actual Settings regression: `sh .cache/t14-r1-settings-red.sh`, temporarily restoring staged Settings with an EXIT restore trap, bundling `tests/settings.test.ts` using esbuild and running `node --test --test-name-pattern='production singleton Settings reset' .cache/t14-r1-settings-red.mjs`. Expected failure: `Guest != Account A` (`.cache/t14-r1-settings-red.log`). The first test version asserted before the asynchronous IndexedDB reset completed and falsely passed; it was corrected to wait for the real rendered “Data perangkat dihapus.” status, then demonstrated RED. Fixed Settings restored.
- First migration compatibility fixture run failed because pgx disallows multiple SQL commands in a parameterized prepared statement. Split parameterized data update from fixture DDL; corrected run passed. This was a fixture error, not a production migration failure.
- Focused GREEN: `go test -race -tags=integration -p 1 ./internal/worlds ./internal/social ./internal/auth ./internal/http -run 'TestRedeemAndDeletion|TestFriendWriters|TestDeletionCannotBeBlocked|TestReservedTombstone|TestAccountRoutes|TestLogoutAndRecovery' -count=1` → `ok worlds 2.225s`, `ok social 2.319s`, `ok auth 2.723s`, `ok http 5.183s` (`.cache/t14-r1-backend-focused.log`).
- Affected suites: `go test -race -tags=integration -p 1 -count=1 ./internal/worlds ./internal/social ./internal/auth ./internal/http ./internal/identity` initially failed existing `TestInviteConcurrencyAndRetry`: “both redeemers did not contend on PostgreSQL world row”; other packages passed (`.cache/t14-r1-affected-go.log`). With the approved account-first order the second redeemer waits on the common issuer, so the old test barrier was updated to the actual two-edge chain, retaining outcome assertions.
- Application regression: `go test -race -tags=integration ./internal/application -run 'TestDeletedAccount|TestAccountDelete|TestDeletionFences' -count=1` → `ok .../application 2.458s` (`.cache/t14-r1-application-account.log`).
- Motion RED: `go test -tags=integration ./internal/http -run TestAccountRoutes -count=1` → `guest motion 404` (`.cache/t14-r1-motion-red.log`); `npm test` → `TypeError: h.setMotion is not a function` (`.cache/t14-r1-device-motion-red.log`). Both then implemented and passed.
- Post-Motion affected GREEN: `go test -race -tags=integration -p 1 -count=1 ./internal/worlds ./internal/auth ./internal/http` → `ok worlds 3.643s`, `ok auth 12.017s`, `ok http 5.750s` (`.cache/t14-r1-affected-final-go.log`).
- Canonical-ID RED: `go test -race -tags=integration ./internal/social -run TestBlockWriterCanonicalizes -count=1` → “variant target bypassed deleted account fence” (`.cache/t14-r1-canonical-red.log`). Canonicalization then added. Final affected GREEN: `go test -race -tags=integration -p 1 -count=1 ./internal/worlds ./internal/social ./internal/identity` → `ok worlds 3.620s`, `ok social 2.614s`, `ok identity 1.546s` (`.cache/t14-r1-fence-final-go.log`).
- Final web `npm test` → 4 JavaScript + 90 TypeScript tests passed, zero failures (`.cache/t14-r1-web-final.log`). Earlier scoped round result was 4 + 88 before the two Motion tests (`.cache/t14-r1-web-focused.log`).
- Final `npm run typecheck` → `tsc --noEmit`, exit 0; `npm run lint` → `node --check src/app.js && node --check src/world.js`, exit 0; `npm run build` → Vite success in 246 ms (`.cache/t14-r1-{types,lint,build}-final.log`). The lint task is syntax validation of those JS entry points, not a full TS style linter. New auth-state/preferences production files and settings tests are included in tsconfig; imported component lifecycle code is typechecked transitively.
- Final `go vet ./internal/worlds ./internal/social ./internal/auth ./internal/http ./internal/identity ./internal/application` and `go build -o ../../.cache/t14-r1-server-check ./cmd/server` passed, exit 0, no diagnostics (`.cache/t14-r1-vet-final.log`, `.cache/t14-r1-go-build-final.log`). Built binary only; no runtime restart.
- `git diff --check` and `git diff --cached --quiet 12e80daddd7f66540037528848eceeb44945e18a` pass. No staging performed.

### Remaining boundaries

M1 (local completion timestamp offset including countdown) remains the explicitly ledgered T16 minor issue; it was not changed or claimed fixed. Real-browser 100-record boundary remains NOT_RUN despite executed actual transaction tests under Node. Authenticated browser workflows remain deferred and source-authored E2E is NOT_RUN. Coordinator's actual three-record guest reset/reload PASS remains valid evidence for that tested scope; this implementer performed no browser work. No T14-done or CP7-PASS claim follows from these local checks. Final re-review and any checkpoint disposition belong to the coordinator.

## Coordinator-retained baseline full Go output

```text
?   	patra-world-social/server/cmd/e2edb	[no test files]
ok  	patra-world-social/server/cmd/server	1.754s
ok  	patra-world-social/server/internal/application	32.160s
ok  	patra-world-social/server/internal/auth	9.707s
ok  	patra-world-social/server/internal/chat	1.999s
ok  	patra-world-social/server/internal/config	1.473s
?   	patra-world-social/server/internal/contracts	[no test files]
?   	patra-world-social/server/internal/db	[no test files]
ok  	patra-world-social/server/internal/games/duck	1.939s
ok  	patra-world-social/server/internal/games/model	1.493s
ok  	patra-world-social/server/internal/games/penalty	1.578s
ok  	patra-world-social/server/internal/games/race	2.018s
ok  	patra-world-social/server/internal/history	16.309s
ok  	patra-world-social/server/internal/http	2.652s
ok  	patra-world-social/server/internal/identity	1.477s
ok  	patra-world-social/server/internal/leaderboard	3.488s
ok  	patra-world-social/server/internal/lobby	2.247s
ok  	patra-world-social/server/internal/moderation	1.912s
ok  	patra-world-social/server/internal/realtime	9.957s
ok  	patra-world-social/server/internal/social	1.732s
?   	patra-world-social/server/internal/testutil	[no test files]
ok  	patra-world-social/server/internal/voice	1.727s
ok  	patra-world-social/server/internal/world	1.600s
ok  	patra-world-social/server/internal/worlds	2.393s
```


## Review repair round 2 — R1/R2 Motion lifecycle and request scope

Status: DONE_WITH_CONCERNS / frozen for scoped re-review. Read both Important findings in `task-14-rereview1.md` verbatim. The original I1–I4 fixes are accepted by that scoped review; this round repairs its two new Motion defects. Coordinator-staged round-one tree `5b6a88ea073aabb0c55e65e49d519bc7b2878784` remains unchanged. All round-two changes are unstaged. HEAD is unchanged. No Go source change, backend rerun, stage/commit, runtime restart, browser/network credential action, subagent, or root documentation/evidence edit.

### R1 — profile notifications no longer trigger session teardown

`auth-state.ts` still notifies ordinary subscribers when a same-session profile is refreshed, but the actual singleton `patra-auth-change` bridge now emits only when the session generation changes. `begin` and `transition` still advance that generation and still trigger cleanup, including a new session for the same account. A successful Motion/profile save retains the current generation and does not emit a teardown event.

The existing input/modal callback in `app.js` is moved without changing its behavior into `guest/session.ts:mountGuestUICleanup` and called by the app. It still releases inputs and closes an open game panel or disposes its closed-panel UI on actual auth/world/reset boundaries. This small extraction lets the regression execute the actual callback rather than merely count events. Guest engine lifecycle listeners remain mounted normally.

Executed test `tests/settings.test.ts` **singleton auth bridge keeps an account solo match through Motion and profile updates but clears session transitions** uses the production singleton AuthState, production ActivePreferences, actual mounted GuestSession lifecycle, real penalty engine and actual app cleanup helper. Only account I/O and UI callback effects are synthetic. It verifies Motion and profile updates preserve the running match and invoke no release/close/dispose effects; logout clears the match and releases/disposes; a same-account `begin` boundary clears the new match and closes an open panel. This is production-module lifecycle evidence, not browser acceptance.

### R2 — Motion owns cancellation across CSRF preparation and POST

`ActivePreferences.toggleMotion` creates an AbortController and subscribes it to its captured auth generation before any awaited preference load or mutation preparation. A changed generation synchronously aborts it. The account Motion I/O now accepts and forwards that exact signal to `api.updateMotion`, then through the real CSRF request and mutation fetch. The operation checks generation/signal after preparation, and its listener is removed in `finally` on success, failure or cancellation.

`net/api.ts:post` explicitly awaits CSRF into a local token, checks `signal.throwIfAborted()` after that awaited preparation, and only then dispatches the POST. Synchronous generation invalidation owns that cancellation state, so a delayed CSRF result cannot dispatch an obsolete mutation. The post-response generation/cancellation guard remains: once a POST has already reached the server, cancellation cannot promise to undo it, but its late response cannot update a later session's client state. No new token/session architecture was introduced.

Executed `tests/settings.test.ts` regressions:

- **production Motion CSRF boundary dispatches no obsolete POST for new account or same-account session** uses the production singleton preference adapter and actual API helper, faking only fetch responses. It holds CSRF, transitions A→guest→B and A→guest→new-A, then releases the old response. Both paths return false, abort the CSRF signal and dispatch zero POSTs. The fake intentionally resolves despite cancellation, proving the explicit post-CSRF check rather than relying on fetch implementation cancellation.
- **production Motion forwards cancellation to an already dispatched POST and fences its late response** lets actual helper preparation finish and holds the POST response. It verifies both fetches carry the same signal, a new same-account session aborts it, and the old response cannot change auth or preferences.
- The previous **Motion uses fresh persisted fields and rejects delayed result after same-account relogin** remains and passes, preserving the existing already-dispatched I/O response regression and fresh-profile-field checks.

### Exact executed checks and outputs

Commands below ran from repo root; no environment secret is involved.

1. `npm --prefix apps/web test > .cache/t14-r2-red.log 2>&1` before implementation: expected exit 1. Existing 4 JavaScript tests passed; TypeScript tests reported 90 pass, 2 fail. Exact failures: “Motion must preserve the running game” and “old CSRF completion must not dispatch under the newer session”. These focused added regressions reproduced both review findings through the production paths, not substitutes that only tested an isolated AuthState.
2. After repair, `npm --prefix apps/web test > .cache/t14-r2-web.log 2>&1`: exit 0, 4 JavaScript + 92 TypeScript pass, zero failures. Both original failing regressions passed, including the actual app cleanup helper assertions.
3. After adding the actual already-dispatched POST signal/result case, final affected frontend check sequence: `npm --prefix apps/web test > .cache/t14-r2-web-final.log 2>&1 && npm --prefix apps/web run typecheck > .cache/t14-r2-types.log 2>&1 && npm --prefix apps/web run lint > .cache/t14-r2-lint.log 2>&1 && npm --prefix apps/web run build > .cache/t14-r2-build.log 2>&1`: exit 0. Tests: 4 JavaScript + 93 TypeScript pass, fail 0. Typecheck: `tsc --noEmit`, no diagnostics. Lint: `node --check src/app.js && node --check src/world.js`, no diagnostics. Vite build: success in 247 ms. All modified TypeScript production/tests remain in the existing explicit tsconfig include paths; lint remains JS entry-point syntax validation, not a full TS style linter.
4. `git diff --check && git diff --cached --quiet 5b6a88ea073aabb0c55e65e49d519bc7b2878784`: exit 0. The staged reviewed round-one baseline is preserved. No stage/commit performed.

No unexpected new test/typecheck/lint/build failure occurred this round; the two RED failures above were the intended reproductions. No Go change justified rerunning backend checks. Frontend source is frozen for scoped reviewer recheck.

M1 remains explicitly deferred to T16 and unchanged. Authenticated browser flows and real-browser 100-record boundary remain NOT_RUN. Root's prior three-record guest reset/reload evidence is unchanged; no account browser activity was performed here. Module/fake-fetch checks do not claim a real browser cookie-jar or authenticated acceptance run. No T14-done or CP7-PASS claim.


## Coordinator verification before user-requested pause

27 October 2026: reran `npm --prefix apps/web test`, `run typecheck`, `run lint`, `run build` sequentially with fail-fast execution. All exit 0: 4 JavaScript + 93 TypeScript tests, zero failures; TypeScript no diagnostics; JS entry-point syntax checks; Vite production build in 254 ms. Logs remain ignored under `.cache/t14-pause-{web,types,lint,build}.log`. No backend source changed in round2, and no backend rerun is claimed. Both unstaged and staged diff whitespace checks passed. Round2 has not received independent re-review. User requested stopping and committing current progress; this is a WIP snapshot, not T14 acceptance or CP7 PASS.
