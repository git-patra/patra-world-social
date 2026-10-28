# Account and session contract (T3)

## Run locally

The Go API requires PostgreSQL and applies the idempotent identity migration on startup. Set `DATABASE_URL` to a private PostgreSQL URL, `SERVER_ADDR=127.0.0.1:8080`, `AUTH_ORIGIN=http://127.0.0.1:4176`, and `AUTH_DEV_LOOPBACK=1`, then run `make server`. Keep the database URL in the ignored `.cache/database.env` or another private environment source. Vite proxies `/api` to `http://127.0.0.1:8080`; override with `API_PROXY_TARGET` for an isolated local port. The browser and API must use the same hostname spelling (`127.0.0.1` here).

`AUTH_DEV_LOOPBACK=1` permits non-Secure cookies only when both the listening host and configured HTTP origin are loopback. Leave it unset in production. Production requires an HTTPS `AUTH_ORIGIN`; session and CSRF cookies then use `Secure`, `HttpOnly`, `SameSite=Lax`, and `__Host-` names. Terminate TLS at a trusted reverse proxy, and configure its exact public origin. Do not expose the development listener externally.

## API

`GET /api/v1/auth/csrf` returns `{ "token": "…" }` and sets a signed, HttpOnly CSRF cookie. Each mutation sends that token in `X-CSRF-Token` with the exact configured `Origin` and same-origin credentials. `POST /api/v1/auth/register` takes `{username,password}` and returns `{user,recoveryCode}`. Show the recovery code only once. Registration does not create a session. `POST /api/v1/auth/login` takes `{username,password}`, sets the opaque session cookie, and returns `{user,expiresAt,idleExpiresAt}`. `POST /api/v1/auth/logout` revokes the current session and clears its cookie. `POST /api/v1/auth/recover` takes `{username,code,newPassword}`, rotates the recovery code, revokes all old sessions, clears the current cookie, and returns `{recoveryCode}`. `GET /api/v1/me` returns the authenticated user or 401.

Usernames are stored lowercase and accept 3–24 ASCII letters, digits, or underscores. Passwords accept 12–128 Unicode code points without trimming; the browser uses the same code-point count and has no UTF-16-based password length attribute. Passwords are salted Argon2id PHC (`m=19456 KiB,t=2,p=1`); tokens and recovery codes are 256-bit random values and only their SHA-256 digests are stored. Sessions have seven-day absolute expiry and 24-hour idle expiry. Successful `RequireSession` calls extend idle expiry. Login session creation locks the user row and rechecks the password hash, so a concurrent recovery either revokes that session or makes the old password invalid before insertion. Registration, login, and recovery are limited by source IP and canonical username. By default, the IP is the socket peer and forwarded headers are ignored. Behind a reverse proxy, set `AUTH_TRUSTED_PROXY_CIDRS` to the exact trusted proxy CIDRs and configure those proxies to append or replace `X-Forwarded-For`; the server then walks the chain from the trusted peer to the first untrusted client IP. Untrusted peers cannot supply a forged client IP.

The local in-memory limiter allows ten attempts per minute per key and stores at most 10,000 active keys. Minute-old entries are pruned at capacity; if all slots are active, new keys are denied while existing accounts continue under their own rate windows. Multi-instance production deployment needs a shared limiter.

## Go interfaces for later tasks

`auth.New(pool)` returns a service with `Register(ctx,username,password) (User,RecoveryCode,error)`, `Authenticate(ctx,username,password) (Session,error)`, `RequireSession(ctx,token) (Session,error)`, `Recover(ctx,username,code,newPassword) (RecoveryCode,error)`, and `Logout(ctx,token) error`. `Session.Token` is omitted from JSON. `httpapi.SessionMiddleware(service,development,next)` validates the cookie, then `httpapi.SessionFromContext(ctx)` retrieves the session. Apply it to future private world routes. `auth.Service.Revoked` is an optional callback invoked after logout or recovery commits; socket and voice layers can attach their connection closure there. No socket or voice system exists in T3.

## Database and verification

The single schema source is `apps/server/internal/db/migrations/001_identity.sql`. Queries live in `apps/server/db/queries/auth.sql`; `cd apps/server && sqlc generate` reproduces `internal/db/*.sql.go`, `models.go`, and `db.go` from `sqlc.yaml`. Startup calls `db.ApplyMigrations`; integration tests create and drop a random PostgreSQL schema so development rows are preserved. To run integration tests, export `DATABASE_URL` and run `cd apps/server && go test -tags=integration ./internal/auth`. An absent database URL fails the target explicitly.

## Disposable browser E2E setup

`apps/web/tests/auth.e2e.spec.ts` exercises the real registration, one-time code display, login, logout, recovery, old-password rejection, and new-password login through the browser UI. It generates test-only account credentials. The test requires an isolated schema and does **not** run as part of `npm test`. An absent isolation environment fails immediately rather than skipping the test. Run it only when browser credential entry is authorized.

From the repository root, first load the private development `DATABASE_URL` into the shell without printing it. Then:

```sh
cd apps/server
go run ./cmd/e2edb create ../../.cache/auth-e2e.env
set -a; source ../../.cache/auth-e2e.env; set +a
go run ./cmd/e2edb verify
AUTH_ORIGIN=http://127.0.0.1:4177 AUTH_DEV_LOOPBACK=1 SERVER_ADDR=127.0.0.1:18080 go run ./cmd/server
```

In a separate shell, run `cd apps/web && API_PROXY_TARGET=http://127.0.0.1:18080 ../../node_modules/.bin/vite --host 127.0.0.1 --port 4177 --strictPort`. In a third shell, source `.cache/auth-e2e.env` and run `AUTH_E2E_BASE_URL=http://127.0.0.1:4177/ npm run test:e2e --workspace apps/web` from the repository root. Stop the two servers after the test. Restore the original private `DATABASE_URL` in a shell, then run `cd apps/server && go run ./cmd/e2edb drop ../../.cache/auth-e2e.env`. The helper accepts only its generated `auth_e2e_<hex>` schema for deletion and removes the ignored private file after dropping it. Never copy these env files into source control.
