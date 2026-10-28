# T3 fix round 1 re-review

## Finding verdicts

1. **Recovery/login concurrency — ADDRESSED.** `apps/server/internal/auth/service.go:182` takes the same user-row lock as recovery and rechecks the stored password hash; session insertion and commit remain inside that transaction (`:189`, `:192`). If recovery wins, the stale hash is rejected; if login wins, recovery waits and subsequently revokes the newly committed session. `apps/server/internal/auth/service_integration_test.go:60`, `TestRecoverySerializesWithInFlightOldPasswordLogin`, exercises the previously vulnerable ordering through a blocked INSERT. Reported RED: `old-password login survived recovery: <nil>`; reported GREEN: PASS, approximately 0.20–0.21s.

2. **Logout storage errors — ADDRESSED.** `apps/server/internal/auth/service.go:217` now treats only ErrInvalid as idempotent success and propagates other errors. `apps/server/internal/auth/service_test.go:28`, `TestLogoutPropagatesStorageFailure`, closes the pool and asserts logout returns an error. Reported RED: `logout reported success while storage was unavailable`; reported GREEN: PASS.

3. **Unicode password length — ADDRESSED.** `apps/web/src/auth-ui.ts:20` removes the password's native UTF-16 length attributes; `:26` applies the shared validator to all three form modes. `apps/web/src/password-validation.js:2` counts code points without trimming, matching the required length semantics. `apps/web/tests/password-validation.test.js:5` covers accepted 65-emoji input, rejected 129-emoji input, and whitespace boundaries. Reported initial RED: missing validation module; reported GREEN: `node --test tests/password-validation.test.js`, 1 test PASS, with typecheck PASS. This verifies validation logic without running the deferred browser credential flow.

4. **Limiter saturation — ADDRESSED.** `apps/server/internal/http/router.go:66` uses the actual minute window; `:86` checks existing active keys independently; `:96` prunes stale keys and `:108` rejects only new keys at the bounded active capacity. `apps/server/internal/http/router_test.go:136`, `TestLimiterCapacityKeepsExistingAccountsAndPrunesMinuteOldKeys`, and `:153`, `TestLimiterRejectsNewKeysAtFullActiveCapacityButKeepsExisting`, cover both failure modes. Reported RED: `existing account disabled by unrelated keys` and `new key exceeded storage capacity`; reported GREEN: both PASS.

## New breakage in the fix diff

None found.

## Out-of-scope observations

The two prior Minor findings remain deferred to the T4 ledger as instructed: original-input ASCII username validation (`apps/server/internal/auth/service.go:53`) and distinguishing storage failures in SessionMiddleware (`apps/server/internal/http/router.go:52`). No new observations outside the fix scope.

## Verification evidence and limits

The appended fix report names the covering tests and records these commands/results:

- `cd apps/server && go test -count=1 ./... && go vet ./...` — reported PASS.
- `cd apps/server && go test -count=1 ./internal/auth ./internal/http` — reported PASS (`auth 0.515s`, `http 0.540s`).
- With private DATABASE_URL, `cd apps/server && go test -count=1 -tags=integration ./internal/auth` — reported PASS (`1.107s`), including the deterministic concurrent login/recovery regression in an isolated schema.
- `node --test tests/password-validation.test.js` — reported PASS, 1 test.
- `npm run lint && npm run typecheck && npm test && npm run build` — reported PASS; validator included in normal frontend tests and typechecking.
- `git diff --check` — reported PASS.

Reviewed the prior findings, binding brief, appended report, and complete fix diff, including the two appended new frontend files. No test reruns, browser operations, source/index/Git changes, or subagents. Reported execution evidence was assessed against the fix; it was not independently regenerated.

## Verdict

**Fix round: All four Important findings addressed; no new Critical/Important breakage.** The code is ready for the authorized partial progress commit and downstream T4 work with the two Minor findings tracked. Full T3/CP2 acceptance remains pending the explicitly deferred browser credential-flow E2E; this approval does not mark that test run or acceptance complete.
