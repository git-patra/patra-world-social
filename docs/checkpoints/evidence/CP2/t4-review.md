### Spec Compliance

- ❌ Issues found: active-member invite redemption can remove the owner role (`apps/server/internal/worlds/service.go:284`); moderator kick authority is narrower than the binding table (`apps/server/internal/worlds/membership.go:82`, `apps/web/src/worlds-ui.ts:40`).
- ⚠️ Acceptance remains pending: browser credential/invite lifecycle execution is explicitly deferred; no browser or indirect credential flow was run. The authored E2E also needs the correction below before later execution (`apps/web/tests/worlds.e2e.spec.ts:30`).
- ⚠️ Cross-task requirements remain outside this diff: live disconnect and 20-online capacity belong to T5, friend/block integration to T6, and participant-authorized personal history to T7. T4 supplies a revocation callback and retains membership records rather than deleting them (`apps/server/internal/worlds/membership.go:90`, `apps/server/internal/worlds/membership.go:174`).

### Strengths

- Redemption and membership mutations consistently lock the world before invite/member writes. The PostgreSQL test explicitly observes overlapping lock waits, then asserts one winner and idempotent retry (`apps/server/internal/worlds/service.go:234`, `apps/server/internal/worlds/invites_test.go:66`). Capacity is checked under that lock and covered at the 50th/51st boundary (`apps/server/internal/worlds/service.go:281`, `apps/server/internal/worlds/invites_test.go:226`).
- Issuance uses 32 random bytes, stores only the hash, and sets a 24-hour expiry; fragment links are removed from the address bar before redemption (`apps/server/internal/worlds/service.go:182`, `apps/web/src/worlds-ui.ts:12`).
- World handlers derive identity from SessionMiddleware and reuse the existing mutation protection (`apps/server/internal/http/worlds.go:13`, `apps/server/internal/http/router.go:379`). The T3 ASCII-username and operational middleware error fixes are present with focused tests (`apps/server/internal/auth/service.go:52`, `apps/server/internal/http/router.go:56`).
- Archive preserves records and invokes member revocation after committing; transfer updates both role and world owner within one transaction (`apps/server/internal/worlds/membership.go:103`, `apps/server/internal/worlds/membership.go:138`).

### Issues

#### Critical (Must Fix)

- None.

#### Important (Should Fix)

1. **Preserve active members' roles when redeeming another invite.** `apps/server/internal/worlds/service.go:284` unconditionally sets `role='member'` on conflict, even after identifying an already-active membership. An owner who opens a valid share link can therefore demote themselves, leaving zero active owners while `worlds.owner_id` still names them; no owner-only operation can repair this through the API. Moderators are silently demoted too, and line 295 returns the hard-coded member role. Preserve the current role/joined timestamp for active members, only reactivate inactive members as ordinary members, and return the actual membership. Add isolated backend cases for owner and moderator redemption plus owner-invariant assertions.

2. **Match moderator kick permission to the specified table.** `apps/server/internal/worlds/membership.go:82` denies a moderator from removing another moderator, and `apps/web/src/worlds-ui.ts:40` hides that action. The binding table allows moderator kick except against the owner; it does not exempt fellow moderators. Remove the additional restriction for kick, keep owner protection and owner-only role changes, and cover moderator-to-moderator kick plus moderator-to-owner denial in backend tests. Keep ban semantics explicitly aligned with the intended moderation policy.

3. **Refresh the owner view before the authored E2E kicks the new member.** `apps/web/tests/worlds.e2e.spec.ts:30` clicks a kick button on the owner page after the friend joins, but that page last loaded members when the world contained only its owner. `apps/web/src/worlds-ui.ts:26` only fetches membership on opening detail, and invite issuance refreshes invitations only. Thus no kick button exists and the future authorized test times out before exercising revocation. Reopen/refresh the owner world detail and assert the friend's row before clicking that row's kick action. Correct the test statically now; leave execution NOT_RUN as instructed.

#### Minor (Nice to Have)

- `apps/web/src/worlds-ui.ts:18`–`40` compresses complete renderers and asynchronous action handlers into very long lines, while `apps/server/internal/worlds/membership.go:9` and `apps/server/internal/http/worlds.go:85` similarly compress new Go handlers. Expand ordinary control flow and format Go to make permission and async-state changes reviewable. The UI's unused `selected` state can also be removed.

### Assessment

**Task quality: Needs fixes.** The backend's concurrency and authorization structure is sound in the main paths, but existing-member redemption can destroy the owner invariant. Fix the three Important findings before treating this snapshot as ready for the intended partial-progress commit; browser acceptance must still remain pending.

**Review checks:** Read the full 2,223-line staged review package in bounded passes; did not re-run git diff, mutate source/index/HEAD, spawn agents, or re-run passing suites. No focused test was necessary because the findings follow directly from the control flow. The implementer/coordinator's reported backend integration and make-check results were considered without representing them as reviewer-run evidence.

**Focused context checks:** (1) Migration adoption risk: inspected only `apps/server/internal/db/migrations/001_identity.sql:1`–`19`; its table/index DDL is idempotent, so the new migration ledger can adopt it. (2) Shared mutation-guard risk: the router hunk does not include the reused closure; inspected `apps/server/internal/http/router.go:214`–`237`, confirming signed CSRF validation and exact Origin matching. (3) Invite login continuation risk: the auth hunk cuts off its submit handler; inspected `apps/web/src/auth-ui.ts:22`–`32`, confirming successful login emits the shared state update and closes the account modal. No broader unchanged-code crawl was performed.
