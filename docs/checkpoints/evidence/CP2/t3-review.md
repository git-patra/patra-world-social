# T3 review

## Spec compliance

**Issues found.** The implementation substantially covers T3, but recovery does not invalidate concurrent old-password logins, logout can falsely report successful revocation, and the browser password field does not accept the specified full Unicode length range. These are implementation defects independent of the deferred browser test.

**Full T3 acceptance remains pending.** Browser credential-flow E2E is explicitly NOT_RUN by user direction, following automatic approval review rejection. No browser credential action was attempted during this review. This is an acceptance limitation, not a reason to downgrade implementation defects or demand a browser retry.

## Strengths

- `apps/server/internal/auth/service.go:82` and `:96`: salted Argon2id meets the required baseline; creation and verification share a four-slot concurrency bound. `:40` excludes opaque session tokens from JSON.
- `apps/server/db/queries/auth.sql:10`: session validation and idle refresh are one conditional update, including both absolute and idle expiration. `apps/server/internal/auth/service.go:233` consumes the recovery digest and rotates the password/code atomically; `:240` revokes existing sessions within that transaction.
- `apps/server/internal/http/router.go:145` validates deployment configuration; production cookies use the required flags and host prefix, while HTTP development requires explicit loopback configuration. `:199` enforces exact Origin and a signed matching cookie/header token before each auth mutation. `:111` ignores forwarded client identities unless the peer is explicitly trusted.
- `apps/server/internal/auth/service_integration_test.go:19` isolates PostgreSQL test data; `:55` tests the real lifecycle. `apps/server/sqlc.yaml:4` points at the coordinator-approved canonical migration/query locations.
- `apps/web/src/app.js:105` adds the account mount without replacing the world lifecycle; `apps/web/src/auth-ui.ts:31` displays the recovery code and clears it when acknowledged/closed.

## Issues

### Critical

None found.

### Important

1. **Recovery races with session creation using the old password.** `apps/server/internal/auth/service.go:157`, `:176`, `:233`, `:240`. Authenticate reads credentials and verifies them before an independent session INSERT. A valid interleaving is: login reads the old hash; recovery changes password, revokes all visible sessions and commits; login inserts a fresh session based on the stale hash. The old password has then produced a valid seven-day session after reset. Serialize session creation with recovery using the same user-row lock/transaction and recheck the credential version/hash before insertion. A plain conditional INSERT without coordinating with the recovery transaction is insufficient. Add a deterministic PostgreSQL concurrency test for this interleaving.

2. **Logout silently treats database errors as successful revocation.** `apps/server/internal/auth/service.go:199–201`. RequireSession can return database/connectivity/cancellation errors, but Logout turns all of them into nil. The HTTP logout handler consequently clears the cookie and returns 204 while a copied token remains valid when the database recovers. Treat only ErrInvalid as an idempotent success; propagate other errors, or implement revocation directly with RETURNING for the callback. Add a focused failure-path test.

3. **Browser password length differs from the required Unicode-character contract.** `apps/web/src/auth-ui.ts:19`. HTML minlength/maxlength count UTF-16 code units; the Go service counts Unicode code points. A valid 65-emoji password is 65 code points but 130 UTF-16 units, so the browser cannot enter it in registration/login/recovery despite the 12–128-character contract. Remove these native length limits for passwords and validate with a matching code-point count while retaining required/server validation. Cover supplementary-plane characters and untrimmed whitespace in a focused UI validation test; this does not require running the deferred credential E2E.

4. **Filling limiter storage disables authentication globally for up to an hour.** `apps/server/internal/http/router.go:73–80`. Once 10,001 keys exist, every key is rejected before its own minute window is examined; entries are pruned only after an hour despite attempts expiring after a minute. Many distinct usernames/IPs can therefore cause all registration, login and recovery requests to fail for unrelated users, including existing keys. Expire entries on the actual rate window and preserve checks for existing keys when capacity is full; choose an explicit bounded policy for new entries. Add a capacity test using stale entries and an unrelated existing account.

### Minor

1. **Non-ASCII username input is accepted through Unicode lowercasing.** `apps/server/internal/auth/service.go:53`. For example, U+212A KELVIN SIGN lowercases to ASCII `k`, so `Kevin` passes despite the ASCII-only input requirement and frontend pattern. Validate the original input against `[A-Za-z0-9_]{3,24}` before ASCII normalization and add that boundary case.

2. **The reusable session middleware hides operational errors.** `apps/server/internal/http/router.go:52–54`. All RequireSession failures become 401, whereas `/me` correctly distinguishes invalid sessions from server errors. T4 consumers would misinterpret database outages as expired login. Preserve 401 only for missing/invalid sessions and use a server-error response otherwise.

## Cannot verify / boundaries

- `apps/web/tests/auth.e2e.spec.ts:4`: authored browser lifecycle is not runtime evidence; leave CP2/T3 acceptance pending until the user resumes that test. Existing HTTP and real PostgreSQL test runs are reported PASS, not independently rerun here.
- `apps/server/internal/auth/service.go:47`: revocation callback is exposed, but socket/voice closure and ongoing expiry checks belong to T5/T13 and cannot be verified in T3. Callback currently carries user ID only, so consumers should explicitly decide whether logout closes all connections for that user.
- Dependency versions were reviewed in manifests; omitted lockfile bulk and PNG pixels were not inspected. Runtime log secrecy cannot be proven solely from the diff; changed handlers themselves do not log credentials.

## Assessment and checks

**Code quality: Needs fixes.** Do not approve this version as a safe T3 partial commit or the settled T4 auth foundation until the Important issues are fixed. After those fixes, a partial commit with browser acceptance explicitly pending is reasonable.

Read the binding brief, implementation report, reviewer template, and the entire supplied review diff in bounded passes (the initial combined tool output was truncated). No broad repository crawl or outside-code inspection; no suites rerun, no browser action, and no source/index/Git mutations. Findings follow directly from the changed control flow and contracts; deterministic concurrency, outage, Unicode, and limiter-capacity tests are the recommended focused verification.
