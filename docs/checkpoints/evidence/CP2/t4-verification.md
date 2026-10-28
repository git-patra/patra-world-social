# T4 implementation report — private worlds and invites

## Status

Implemented in `/Users/patra/Project/patra-world-social` on `feat/social-world`, based on `ec154108a6f3f2ec3a83ef1d3b13115359f54990`. No staging, commit, push, deployment, browser credential flow, or source repo edit was performed. Original `/Users/patra/Project/patra-world` remains at `8d969212d91804e06225f5b1a75eed49e8c25307` with its pre-existing untracked `.idea/` only. Coordinator-owned `docs/PROGRESS.md` and `docs/checkpoints/CP2.md` were not edited by this task.

## Implementation and interfaces

- `apps/server/internal/db/migrations/002_worlds.sql`: persistent UUID worlds, one active owner, members, bans, and one-use invites. The raw invite token is never persisted; SHA-256 token hash only. `worlds.sql` and generated SQLC files add world query surface.
- `db.ApplyMigrations`: ordered embedded migrations, version table, one transaction, and transaction-scoped advisory lock for concurrent startup. Existing identity migration is idempotently adopted.
- `internal/worlds/service.go`: `CreateWorld(ctx,userID,name) (World,error)`, `AuthorizeMember(ctx,userID,worldID) (Membership,error)`, `IssueInvite(ctx,actorID,worldID,targetUserID) (InviteLink,error)`, `RedeemInvite(ctx,userID,token) (Membership,error)`. Also list/get world, list members/invites/bans, username-targeted issuance, revoke/unban, change role, kick/ban, transfer owner, and archive.
- Redemption locks the world row before the invite row in one transaction. It checks archive, target, expiry, revocation, ban and 50-member limit, and handles same-account retry idempotently. Transfer demotes the previous owner before promoting the new owner under the world lock. Kick/ban/archive publish `Service.Revoked(ctx, worldID, userID)` after commit for T5 to wire into socket/game/voice teardown; every subsequent `AuthorizeMember` call rejects revoked admission.
- `internal/http/worlds.go` registers scoped `/api/v1/worlds` and `/api/v1/invites/redeem` routes inside the existing auth router. Every world route uses T3 `SessionMiddleware`; every mutation uses the same exact-Origin and signed double-submit CSRF guard as auth. HTTP creates/lists worlds, lists members/invites/bans, issues/redeems/revokes invites, changes roles, kicks/bans/unbans members, transfers ownership, and archives.
- Two T3 minors are fixed: original username must match ASCII before lowercasing (so Kelvin sign rejects), and operational session-store failures return HTTP 500 rather than 401.
- `apps/web/src/worlds-ui.ts` plus CSS adds a private-world picker, creation, member/admin views, issue/copy/revoke invitation, ban list/unban, transfer/archive, and invite acceptance. Auth state changes continue pending invite acceptance after login. Invite tokens stay in URL fragments, are removed from the address bar before network requests, and are submitted only to redeem POST. UI explicitly says multiplayer and voice are not yet available; solo local play is retained.
- `apps/web/tests/worlds.e2e.spec.ts` authors a three-context targeted-invite, acceptance, outsider, and kick flow. It is marked `AUTH_BROWSER_NOT_RUN` and was not executed.

## RED → GREEN evidence

- RED: `GOPROXY=off GOCACHE=$PWD/.cache/go-build /opt/homebrew/bin/go test -tags integration ./internal/worlds` failed with undefined `New`, `ErrInviteInvalid`, and `ErrForbidden` before the domain existed.
- RED: `GOPROXY=off GOCACHE=$PWD/.cache/go-build /opt/homebrew/bin/go test ./internal/auth ./internal/http -run 'TestCanonicalUsername|TestSessionMiddlewarePreservesStorageFailure' -count=1` failed because `Kelvin` was accepted and a DB failure mapped to 401.
- GREEN: `go test -tags integration ./internal/worlds -run TestInviteConcurrencyAndRetry -count=1 -v` passed against isolated PostgreSQL. The test observes two redeem transactions simultaneously blocked on the same world row in `pg_stat_activity`, then confirms exactly one new member and idempotent winner retry.
- GREEN: final `GOPROXY=off GOCACHE=$PWD/.cache/go-build /opt/homebrew/bin/go test -tags integration ./... -count=1` passed all packages, using disposable randomized schemas on loopback PostgreSQL port 55432. World tests cover wrong target, expired/revoked tokens, outsider listing/access, member/moderator permissions, ban/unban, 50th/51st capacity, kick hook/admission denial, transfer and archive. HTTP integration proves owner access and outsider denial through actual routes.
- GREEN: `GOPROXY=off GOCACHE=$PWD/.cache/go-build /opt/homebrew/bin/go test ./...`, `npx tsc --noEmit`, `npm run build`, and `git diff --check` passed.
- Browser account and invite E2E: **NOT_RUN** by explicit user direction after automatic approval rejection. The spec is authored only; no Playwright or CUA credential submission occurred.

## Remaining acceptance and integration

- T5 must wire `Service.Revoked` to close active world/game/voice connections and reuse `AuthorizeMember` at admission and during runtime. T5 owns the 20-online-seat and reconnect-capacity tests. No online count or simulated multiplayer was added in T4.
- T6 must add friend-list invitation and block checks before new username/link issuance; T4 provides registered-username binding and a shareable link.
- Browser three-context lifecycle acceptance remains pending authorized execution. Backend PostgreSQL integration and the 50-member boundary were executed independently.
- T7 world match summary/leaderboard should call `AuthorizeMember`; personal history needs participant authorization even after membership removal or archive.

## Files touched by T4

`apps/server/cmd/server/main.go`; `apps/server/db/queries/worlds.sql`; `apps/server/internal/auth/service.go`, `service_test.go`; `apps/server/internal/db/migrate.go`, `migrations/002_worlds.sql`, generated `models.go` and `worlds.sql.go`; `apps/server/internal/http/router.go`, `router_test.go`, `worlds.go`, `worlds_integration_test.go`; `apps/server/internal/worlds/service.go`, `membership.go`, `invites.go`, `invites_test.go`; `apps/web/index.html`, `playwright.config.ts`, `tsconfig.json`, `src/app.js`, `src/auth-ui.ts`, `src/net/api.ts`, `src/worlds-ui.ts`, `src/worlds.css`, `tests/worlds.e2e.spec.ts`.

## Coordinator integration gate

After implementation report, coordinator ran `make check` on this exact staged implementation: exit0. Frontend18tests passed (8JS +10map/protocol), Go tests/vet, TypeScript, frontend143module production build, and Go production build PASS. `git diff --cached --check` PASS. Browser E2E remains NOT_RUN.

## Read-only guest UI smoke

Coordinator opened localhost4176 as guest and clicked Dunia privat. Actual UI requests login and reveals no world/member data; browser error/warning log empty. No credentials or authenticated world mutations used. Evidence `docs/checkpoints/evidence/CP2/t4-guest-world-gate.png`. This only verifies guest UI rendering; main API8080 has not yet been restarted to the T4 binary and no authenticated browser acceptance is claimed.

## Review round 1 fixes and verification

- Existing active owners and moderators now keep their role and original `joined_at` when redeeming another valid invitation. The returned `Membership` is read from the transaction. Only a previously inactive membership is reactivated as an ordinary member. The new isolated PostgreSQL test verifies both roles and the world owner invariant (`worlds.owner_id` equals the sole active owner).
- Moderators may kick **or ban** any non-owner, including another moderator. Owner protection, owner-only role changes, and owner-only transfer remain in force. Ban follows the same moderation scope as kick and prevents future admission until unbanned. The new test covers peer moderator kick and ban, plus both owner denials.
- The authored three-context E2E now reopens the owner's world detail after the friend joins, asserts the friend's row, and clicks that row's kick button. Browser credentials and invite E2E remain **NOT_RUN**.
- RED command (isolated PostgreSQL): `GOPROXY=off GOCACHE=$PWD/.cache/go-build /opt/homebrew/bin/go test -tags integration ./internal/worlds -run 'TestAdditionalInvitePreservesActiveOwnerAndModerator|TestModeratorMayKickAndBanPeerButNeverOwner' -count=1 -v` — exit 1. Output: `TestAdditionalInvitePreservesActiveOwnerAndModerator: redeeming role = "member", want "owner"`; `TestModeratorMayKickAndBanPeerButNeverOwner: moderator kicked peer: world access denied`.
- GREEN same focused command after fixes — exit 0; both tests PASS, package `ok .../internal/worlds 1.145s`.
- GREEN final `GOPROXY=off GOCACHE=$PWD/.cache/go-build /opt/homebrew/bin/go test -tags integration ./... -count=1` — exit 0; all packages passed, including `internal/http 2.640s` and `internal/worlds 5.243s`, with randomized isolated PostgreSQL schemas.
- GREEN `GOPROXY=off GOCACHE=$PWD/.cache/go-build /opt/homebrew/bin/go test ./...` — exit 0; all packages passed. `npx tsc --noEmit && npm run build` — exit 0; TypeScript passed and Vite built 143 modules in 391ms. `git diff --check` — exit 0.
