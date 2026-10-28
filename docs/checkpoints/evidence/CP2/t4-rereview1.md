### Finding Verdicts

- **Preserve active members' roles when redeeming another invite — ADDRESSED.** `apps/server/internal/worlds/service.go:284` now executes the insert/reactivation only for inactive or absent memberships; `service.go:294` reads the actual membership within the transaction before commit. Existing roles and joined timestamps are preserved. `apps/server/internal/worlds/invites_test.go:290` covers owner and moderator roles, unchanged joined timestamps, persisted authorization, and the sole-owner invariant.
- **Match moderator kick permission to the specified table — ADDRESSED.** `apps/server/internal/worlds/membership.go:102` protects the owner while allowing peer moderator removal. `apps/web/src/worlds-ui.ts:40` exposes kick/ban for any other non-owner and retains owner-only role/transfer controls. This matches the coordinator's explicit peer-ban policy. `apps/server/internal/worlds/invites_test.go:351` covers peer kick and ban, immediate admission denial, and both owner denials.
- **Refresh the owner view before the authored E2E kicks the new member — ADDRESSED.** `apps/web/tests/worlds.e2e.spec.ts:30`–`34` returns to the picker, reopens the world, asserts the friend's row, and scopes the kick locator to that row. The stale-owner-member-list defect is removed statically; execution remains NOT_RUN.

### New Breakage in the Fix Diff

- None.

### Out-of-Scope Observations

- None newly identified. The original nonblocking formatting observation remains outside this fix round, except that ListBans was formatted.

### Verdict

**Fix round: All findings addressed, no new Critical/Important breakage.** Ready for the intended partial-progress commit; deferred browser acceptance is still pending.

**Checks:** Read the supplied five-file fix diff and appended fix report; applied the existing task brief and explicit coordinator ban-policy clarification. The report names the two covering PostgreSQL tests with RED failures and GREEN outcomes, followed by passing tagged suites, ordinary Go tests, TypeScript/build, and whitespace checks. Their assertions and the reported fixes agree with the diff. No tests, browser actions, git commands, or source/index/HEAD changes were performed for this rereview.
