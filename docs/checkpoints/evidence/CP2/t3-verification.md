# T3 implementation report — 2026-10-26

## State

Implemented Go/PostgreSQL identity service, HTTP auth endpoints, CSRF/origin/cookie protection, login throttling, and game-shell account UI. PostgreSQL integration and all normal project gates pass. Browser credential-flow E2E is **NOT_RUN**: the user deferred it after automatic approval review rejected browser registration. Treat CP2 browser evidence as pending; do not claim full end-to-end completion.

I did not stage, commit, or edit `docs/PROGRESS.md`. The concurrent change to that file and `docs/checkpoints/evidence/CP2/` belongs to the coordinator. No source Canvas art, station navigation, world logic, or T2 game contracts were removed.

## Implemented files and interfaces

- `apps/server/internal/auth/service.go`: `New(pool)`, `Register(ctx,username,password) (User,RecoveryCode,error)`, `Authenticate(ctx,username,password) (Session,error)`, `RequireSession(ctx,token) (Session,error)`, `Recover(ctx,username,code,newPassword) (RecoveryCode,error)`, `Logout(ctx,token) error`; `Service.Revoked` callback after DB revocation. Canonical usernames; salted Argon2id PHC m=19456 KiB/t=2/p=1; global four-slot hash semaphore; 256-bit opaque session/recovery tokens; SHA-256 digests only in DB. `RequireSession` atomically validates and refreshes idle expiry via one SQL statement.
- `apps/server/internal/http/router.go`: `NewRouter(service, Options)` with `Origin`, `Addr`, `Development`, `TrustedProxyCIDRs`; routes `GET /api/v1/auth/csrf`, `POST /api/v1/auth/register|login|logout|recover`, `GET /api/v1/me`. `SessionMiddleware` and `SessionFromContext` are available for T4+ private routes. Exact origin and signed double-submit CSRF checks; production `__Host-` Secure/HttpOnly/SameSite=Lax cookies; explicit loopback-only HTTP development exception. In-memory bounded per-IP + canonical account attempt limiter. Trusted proxy CIDRs are opt-in and parsed; untrusted X-Forwarded-For is ignored.
- `apps/server/internal/db/migrations/001_identity.sql` is the single schema source. `apps/server/db/queries/auth.sql`, `apps/server/sqlc.yaml`, generated `internal/db/*.go` use sqlc v1.30.0 and pgx/v5. `db.ApplyMigrations` embeds 001 and is invoked at server startup. `apps/server/cmd/e2edb` creates, verifies, and drops a disposable E2E schema, writing an ignored mode-0600 env file.
- `apps/web/src/auth-ui.ts` mounts into the existing vanilla `app.js` shell; Canvas and world lifecycle remain owned by `app.js`. `apps/web/src/net/api.ts` handles same-origin CSRF bootstrap/API requests. Indonesian account labels, register/login/logout/recover forms, and one-time recovery display. Vite `/api` proxy. TypeScript modules, Playwright E2E source/config, CSS, docs/AUTH.md, `.env.example`, Makefile integration target, package manifests updated.

## RED / GREEN and verification

- RED: from `apps/server`, `GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build go test ./internal/auth ./internal/http` initially failed with undefined `hashPassword`, `verifyPassword`, username helpers and missing `internal/http` package.
- GREEN: `go test ./internal/auth` passed after password/service implementation.
- RED: `go test ./internal/auth ./internal/http` failed with undefined `NewRouter`/`Options`; GREEN after router implementation.
- RED: `go test -tags=integration ./internal/auth` failed with undefined `db.ApplyMigrations`; GREEN after embedded migration. First database attempt was denied by sandbox loopback policy; the escalated local test succeeded.
- RED: new middleware test failed with undefined `SessionMiddleware`; GREEN after middleware implementation. Trusted proxy test failed with undefined parser/client-IP helper; GREEN after implementation.
- Final `cd apps/server && go test ./... && go vet ./...`: PASS, all packages. `cd apps/server && go test ./internal/auth ./internal/http`: PASS.
- Final real PostgreSQL `go test -count=1 -tags=integration ./internal/auth` with private `DATABASE_URL`: PASS (1.835s), using a random schema dropped on cleanup. Covers case-insensitive duplicate race, hashed password storage, bad password/code, one-use recovery, reset revocation, logout revocation, idle + absolute expiry, and revocation hook. Running that explicit target without `DATABASE_URL` fails immediately as required.
- `npm run lint && npm run typecheck && npm test && npm run build`: PASS. Existing game tests and Canvas contracts passed. The new TS auth UI/API and Playwright source are included in TypeScript checking; JS shell receives `node --check`.
- `sqlc generate` followed by checksums of `internal/db/*.go`: no generated drift. `git diff --check`: PASS.
- Playwright E2E source discovery with isolated env and `playwright test --list`: PASS, one browser test found. `npm run test:e2e` without isolation env: fails at config before launching a browser. **The actual browser test was NOT_RUN.**
- Read-only IAB check at 4177: game Canvas, station list, account entry, and empty registration form visible; browser error/warning log empty. No password or recovery value was typed, read, or logged. Server logs observed only the listener startup line.

## Browser approval and handoff

Automatic approval review rejected a CUA browser action that would fill a generated username/password and submit registration: “This submits a registration containing a newly generated password and creates persistent account state; the user authorized implementing/testing auth generally but did not explicitly authorize creating an account in the browser, so confirmation or user handoff is required. Do not bypass this rejection through a workaround or indirect execution.” That action was rejected before submission; no browser account was created. The user subsequently said: “Tunda uji browser; lanjutkan pekerjaan lain.” Do not run the credential-creating Playwright test or browser recovery flow until the user resumes/authorizes it.

The browser E2E test source is `apps/web/tests/auth.e2e.spec.ts`. It generates its own test-only credentials and checks real registration, one-time code, login, logout, recovery, old-password rejection, and new-password login. It is separate from default `npm test`. `docs/AUTH.md` has the setup/teardown commands. The current prepared preview uses isolated schema `auth_e2e_a6dd9e7116747017`, selected by ignored `.cache/auth-e2e.env`; `e2edb verify` confirmed `current_schema()` equals that schema. Local Go server is my exec session **23130** on `127.0.0.1:18080`; Vite is my session **69095** on `127.0.0.1:4177` with `/api` proxy. Both remain running for coordinator review. Parent captured a separate empty registration handoff screenshot. To remove the disposable schema later, stop those servers, source the original private `.cache/database.env` without printing it, and run `cd apps/server && go run ./cmd/e2edb drop ../../.cache/auth-e2e.env`.

If the user later performs the manual browser flow: open `http://127.0.0.1:4177/`, choose Akun → Daftar, enter a disposable username/password, record the one-time code, then log in, log out, recover with that code and a new password, and verify the old password fails while the new one succeeds. The prepared Playwright command is in `docs/AUTH.md`; do not execute it during the current deferral.

## Remaining integration boundary

T3 exposes a revocation callback and session middleware for future world socket/voice integration. There are no T5 socket or T13 voice connections yet, so closing them is not claimed. The in-memory limiter is process-local; a later multi-instance production deployment needs a shared limiter. Browser credential-flow verification remains pending by user direction.

## Review fix round 1 — four Important findings

The coordinator supplied `task-3-review.md`; I fixed only its four Important findings. The index remains the original reviewed snapshot. These fixes and their tests are working-tree changes only; no staging or commit occurred. `docs/PROGRESS.md` remains coordinator-owned and untouched by me. Browser credential testing remains deferred by the user, and no Playwright lifecycle or credential entry was run in this fix round.

1. **Recovery/login race:** `Authenticate` now begins a transaction after Argon2 verification, locks the user row with `SELECT password_hash ... FOR UPDATE`, rejects a changed hash, inserts the session within that transaction, and commits. `Recover` already updates that same row and revokes sessions in its transaction. The deterministic PostgreSQL regression holds a test advisory lock in a `sessions` INSERT trigger, starts an old-password login, waits until its INSERT is blocked, then starts recovery and observes whether recovery completes or waits on the user row before releasing the trigger. RED with the reviewed code: `old-password login survived recovery: <nil>`; GREEN after fix: `TestRecoverySerializesWithInFlightOldPasswordLogin` PASS (0.20s). The test observes only connections carrying its isolated schema as `application_name`.
2. **Logout outage:** `Logout` now treats only `ErrInvalid` as idempotent success and returns storage/cancellation errors. A focused closed-pool test was RED: `logout reported success while storage was unavailable`; GREEN: `TestLogoutPropagatesStorageFailure` PASS. The existing HTTP handler therefore returns 500 and does not clear the cookie when revocation fails.
3. **Unicode password form:** Removed native `minlength`/`maxlength` from password inputs; added `passwordLengthError` using JavaScript code-point iteration, called on login/register/recover before API submission. `required` and server validation remain. New test checks 65 emoji accepted, 129 emoji rejected, twelve spaces accepted, eleven spaces rejected. Initial RED was missing validation module; GREEN: `node --test tests/password-validation.test.js` PASS (1 test). `npm run typecheck` PASS.
4. **Limiter capacity:** Entries expire on the actual one-minute window. Existing active keys are checked normally even when storage is full; capacity cleanup prunes stale keys, and a full active map rejects only a new key without growing beyond 10,000. RED tests reported `existing account disabled by unrelated keys` and `new key exceeded storage capacity`; GREEN: both capacity tests PASS.

Final review-fix commands/results (no secrets printed):

Go commands below were run with `GOPROXY=off` and `GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build`; the integration command additionally sourced the ignored `.cache/database.env` into its shell.

- `cd apps/server && go test -count=1 ./... && go vet ./...` — PASS, all packages.
- `cd apps/server && go test -count=1 ./internal/auth ./internal/http` — PASS (`auth 0.515s`, `http 0.540s`).
- With private `DATABASE_URL` sourced, `cd apps/server && go test -count=1 -tags=integration ./internal/auth` — PASS (`ok ... 1.107s`), using random temporary schema, including the concurrency test. The focused concurrency run separately passed (`0.21s`) after restricting PostgreSQL lock observation to its own test connections.
- `npm run lint && npm run typecheck && npm test && npm run build` — PASS; frontend unit suite now includes the Unicode validator and Vite built 141 modules.
- `git diff --check` — PASS.

The browser preview may still be running from the earlier handoff, but it was not used in this fix round. T3 browser acceptance remains **NOT_RUN**.
