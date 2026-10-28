# T14 fix round 1 — scoped re-review

27 October 2026. Reviewed `task-14-fix1.diff` (1,939 lines, including the eight new files), the appended implementation report, binding I1/I2/I3/I4 rulings, and the original findings in `task-14-review.md`. Scope is the four requested repairs and regressions introduced by their fix diff. M1 is explicitly deferred to T16 and was neither changed nor claimed fixed.

Base reviewed staged tree remains `12e80daddd7f66540037528848eceeb44945e18a`; HEAD remains `658bd09271e31d6b8f86333fb7d407a0d4aaea21`. Read-only Git checks confirmed both. Fixes are unstaged.

## Verdicts

**Spec readiness: CHANGES REQUESTED.** The original I1–I4 defects have corresponding repairs, but the newly introduced account Motion path has two important lifecycle/scope regressions below. They are implementation issues independent of browser acceptance.

**Code readiness: CHANGES REQUESTED.** Backend account fences, reserved-name migration, Settings generation checks and response-cookie changes are acceptable in this scoped review. Repair R1/R2 before accepting the fix round.

**New findings: 0 Critical, 2 Important, 0 Minor.** The earlier M1 remains one carried/deferred Minor, not a new finding. No unrelated observation is promoted into this review.

## Original finding disposition

| Original finding | Verdict | Basis and coverage |
|---|---|---|
| I1 — deletion versus REST writers | **ADDRESSED** | `identity.LockActiveAccounts` canonicalizes UUIDs, sorts/deduplicates, takes all namespace-12 locks, then checks live accounts in the transaction. Social operations take these before pair locks; world/invite/member writers take them before world/invite/relationship locks. Redeem fences issuer and redeemer after discovering immutable IDs. Existing deletion keeps its account-first order without acquiring world/pair locks. `TestRedeemAndDeletionSerializeWithoutTombstoneMembership`, `TestInviteIssueFencesDeletingIssuerAndTarget`, `TestFriendWritersFenceDeletingActorsAndTargets`, and `TestBlockWriterCanonicalizesTargetAccountFence` cover actual writer overlap, both orderings, actor/target/issuer cases and alternate UUID spelling. PostgreSQL wait-edge barriers give stronger overlap evidence than sleeps. |
| I2 — public tombstone-name collision | **ADDRESSED** | Deletion uses `~deleted:` plus full UUID; migration 012 normalizes existing deleted rows before validating a constraint that reserves this form for deleted rows only. Live public-name validation is unchanged. Legacy-name preclaim and upgrade compatibility tests verify successful deletion and preservation of unrelated live accounts. Migration 001/history payloads are not rewritten. |
| I3 — account/device preference mixing | **ADDRESSED for the original two triggers** | ActivePreferences now selects the current account/device store; account local reset preserves the account form/effective preferences. Motion writes only `reducedMotion`, atomically in JSONB or an IndexedDB read/modify/write transaction. Production Settings tests cover reset then account save; actual storage-code tests cover concurrent field preservation and reset fencing. The old unconditional guest-field overwrite is removed. R1/R2 are new regressions in this replacement Motion path and still block readiness. |
| I4 — old Settings callbacks and expiry headers | **ADDRESSED for the original paths** | Settings captures the auth generation, checks it centrally, aborts both CSRF and mutation fetches on unmount/remount, and rejects stale success/null/recovery/preference callbacks, including same-account relogin. The unversioned account-update event consumer is removed. Password/delete/logout/recovery no longer expire the shared session cookie; server revocation remains authoritative. Existing production component/state tests and HTTP cookie/revocation tests cover these changes. The new Motion mutation does not receive the same request cancellation protection; see R2. |

The ADDRESSED entries describe the original defects, not blanket approval of the replacement code or completed external acceptance.

## Important new findings

### R1 — A Motion preference update is emitted as a mode change and destroys solo games

**Changed locations:** `apps/web/src/preferences.ts:23`; `apps/web/src/auth-state.ts:15`.

**Existing consumers involved:** `apps/web/src/guest/session.ts:11`; the `patra-auth-change` cleanup handler in `apps/web/src/app.js`.

**Trigger:** An authenticated user starts a solo game, closes its panel while the match continues, then clicks the main Motion button. `toggleMotion` receives the fresh account profile and calls `auth.update` even though the account/session has not changed. AuthState emits to every subscriber, and its global adapter always dispatches `patra-auth-change`. That event clears all three GuestSession engines and releases/disposes the game UI. A valid visual preference update has therefore become a mode teardown.

**Impact:** The in-progress solo match disappears without a completed history result. This applies to the supported account-plus-local-solo mode; a guest-only Motion smoke cannot catch it. The online socket is not disconnected by this event when the user remains truthy, so this finding specifically does not claim that all online play is disconnected.

**Focused supporting probe:** Bundled the real AuthState, ActivePreferences, GuestSession and mounted guest lifecycle in Node; substituted only account profile/Motion I/O with synthetic data. Started a penalty engine, toggled Motion, then checked the same engine. Actual output:

```json
{"localMatchBefore":true,"localMatchAfter":false,"authUser":"a"}
```

This is an executed production-module/event-path probe, not a browser test. Its temporary bundle was removed; it performed no network requests or credential actions.

**Required repair:** Distinguish session-generation transitions from same-session profile/preference notifications. A Motion update may refresh identity fields returned by the server, but must not emit a teardown signal for the unchanged session. Preserve teardown for real login/logout/switch boundaries. Add a regression using the actual singleton auth event bridge plus mounted GuestSession lifecycle: an account's Motion change must leave its active local match intact; a real session transition must still clear it. Check the app input/modal cleanup listener under the same distinction.

### R2 — An obsolete Motion request can be dispatched using a newer account cookie

**Changed locations:** `apps/web/src/preferences.ts:5`, `:19–23`; `apps/web/src/net/api.ts:20`.

**Trigger:** Account A clicks Motion. `api.updateMotion` first awaits `/auth/csrf`. While that lookup is delayed, A logs out and B logs in. The Motion I/O signature has no AbortSignal and the production adapter calls `api.updateMotion(reduced)` without one. When the old CSRF response finally arrives, `post` still dispatches `/account/motion` with same-origin credentials, now using B's session cookie. The generation check runs only after the mutation response. It correctly refuses to publish the response, but cannot prevent or undo the server update.

**Impact:** A's old visual-setting action can change B's saved `reducedMotion` value, while B's visible preferences remain unchanged until reload. This is not just a stale-render issue. The CSRF guard does not prevent the scenario: the existing CSRF endpoint returns a new signed token and sets its matching cookie independently of account identity; a delayed old CSRF response arriving after B's login can install that valid pair. The normal origin check also passes.

**Focused supporting probe:** Used the production ActivePreferences and API helper with a fake fetch that held only the CSRF response. Changed the central state A→B, then released CSRF. Actual output:

```json
{"accepted":false,"requests":[{"url":"/api/v1/account/motion","authGenerationAtDispatch":2,"userAtDispatch":"b","hasAbortSignal":false}]}
```

The probe confirms that the production request is sent after the generation changes without a signal, although its result is ignored. It does not simulate a browser cookie jar. The wrong-account server effect follows from the inspected same-origin fetch and account-independent CSRF cookie contract (`router.go:267–278`, with validation at `:234–235`). No real HTTP/browser/credential request was made.

**Required repair:** Make Motion requests generation-owned, pass AbortSignal through the account Motion I/O into the existing CSRF and POST helper, and invalidate/abort on session-generation change. Also ensure the generation is still valid before dispatching the mutation after any awaited preparatory work. Keep the post-response generation guard because cancellation can be too late after dispatch. Add a focused deferred-CSRF test against the actual helper that changes account generation and verifies no POST is dispatched for the obsolete operation; retain the existing delayed-result test for the already-dispatched case.

## Verification and evidence limits

Read the complete fix diff once in manageable chunks. The first combined tool response truncated the report portion; the appended report section was then read separately. Supporting source reads were confined to the concrete Motion/event and delayed-CSRF risks, plus the existing component lifecycle harness. No broad repository audit or suite rerun was performed.

Inspected recorded outputs:

- `.cache/t14-r1-fence-final-go.log`: worlds/social/identity passing package results after UUID canonicalization.
- `.cache/t14-r1-affected-final-go.log`: worlds/auth/http passing package results after Motion changes.
- `.cache/t14-r1-application-account.log`: passing application deletion regression run.
- `.cache/t14-r1-web-final.log`: 90 TypeScript tests, zero failures; report separately records four JavaScript tests, 94 total.
- Final typecheck/lint logs contain the invocations with no diagnostics, and Vite build ends successfully. Lint remains the two-JS-file syntax check, not a full TypeScript style linter.
- Final vet/build logs are empty; successful exits are recorded in the implementation report, not independently established by empty text alone.

The reported commands intentionally use affected Go packages and `-race -tags=integration -p 1 -count=1`; no full-suite rerun was requested or inferred. The existing component harness executes production functions/effects using a deterministic Node hook driver, not a DOM/browser renderer. Its tests establish useful callback/state behavior, but the existing Motion tests use an isolated AuthState and did not exercise the production singleton's global teardown event; delayed-result tests also did not hold the real CSRF-to-POST boundary. The two focused probes above address those precise unanswered concerns rather than repeat suites.

No source, index, HEAD, runtime or browser changes; no credentials, provisioning, remote actions or subagents. Only this review report was written persistently. Temporary probe bundles in `/tmp` were deleted.

External authenticated-browser lifecycle and the real-browser 100-record fixture remain **NOT_RUN**. The prior actual three-game/reload/reset guest evidence remains limited to its documented scope. Root's separate guest Motion smoke is not claimed here. M1 remains deferred to T16. Neither the recorded local tests nor this review establishes T14 completion or CP7 PASS.
