# T9 penalty duel implementation report

Base supplied: `e613707a6f64faea291c137504d2de7a1170e9b2`, branch `feat/social-world`. Work remained in `/Users/patra/Project/patra-world-social`. No staging, commits, source-repo edits, remote activity, credential browser flows, session injection, or preview restarts were performed by this implementer. Root owns integration, ledger and browser evidence. Existing T8 startup recovery, shared Step budget and generation-fenced returns remain in place.

## Implementation

- Pure Go engine exposes the requested `New`, `Choose`, `Tick`, `Snapshot`, `Result` signatures. Four zones, first-choice lock, 3000ms selection, independent missing-slot uniform RNG, reveal at the fixed deadline and 1500ms animation. Alternating five kicks, early mathematical win, equal sudden-death pairs and maximum five extra pairs/draw. Bot RNG runs before human input and its API receives no choices/state. Clock/RNG injected; engine tests never sleep three seconds.
- Inert `penalty.Adapter` activates only at official lobby Start, preserves concrete Tick public events/Result bool via the generic interfaces, copies frozen roster/timing, rejects duplicate command IDs across turns, and derives actor identity from the admitted account. Event buffers drain outside the owner into the existing bounded world delivery queues. Public snapshots expose only chosen flags before reveal; resolved turns carry zones/timeouts afterward. Public current-match reads bind actual match world to the admitted world and work for spectators; private history remains participation-only.
- Authoritative receipt boundary was explicitly approved by root: `receivedAt` means **lobby-owner ingestion after bounded transport/owner queues**, immediately before Choose. It does not mean raw TCP/frame arrival. A queued command ingested at/after the deadline is rejected and cannot alter an already revealed turn. No client timestamp/actor ID is accepted. The latency-sensitive admitted-seat callback uses no SQL, holds the admission/revocation lease, fences world/token/connection epoch, and uses a maximum two-second cached authorization lease clipped by known session/idle expiry. Existing transport revalidation renews it. Expiry is checked again at owner ingestion. Other controls retain full authorization.
- Natural engine ticking now precedes grace departure in the actual lobby lifecycle. The engine limits catch-up to the earliest 15-second grace boundary; a natural result before that boundary survives both players' later permanent departure. Intentional early leave produces forfeit even if the leaver leads on score. Both disconnected humans produce abandoned participant outcomes without a fabricated winner. Reconnect preserves deadline/phase; expired grace cannot revive the duel.
- `006_penalty.sql` stores resolved per-turn detail. `penalty.Project` is registered before startup recovery/worker launch and runs inside the existing finalization transaction. Existing immutable result retry is preserved. Validator/UI narrowly accept semantic penalty win/loss/draw/forfeit/abandoned outcomes alongside earlier valid finished/dnf values; unchanged result structs preserve prior JSON/hash shape. Goal/zone consistency is checked.
- Completed results offer explicit return and rematch. Both humans must consent (one human for an explicit bot duel); rematch creates a new lobby requiring Ready/Start. Running/finalizing/uncertain results cannot return/rematch. Release still calls guarded `ReleaseMatch(completed)`. If every human has permanently departed, the completed penalty arena is released automatically only after durable finalization. Human-bot online natural completion, return, forfeit and post-save cleanup are exercised against isolated PG fixtures.
- TypeScript local engine consumes the same golden combinations, boundaries and complete timelines as Go. Existing goalkeeper/shooter/ball SVG and pitch CSS were reused. Duel UI locks input, uses monotonic server-time estimates online, presents provisional vs saved status, reveals resolved shots, shows opponent BOT status, and offers explicit forfeit confirmation. Guest results never upload. Guest session survives modal unmount/reopen and catches up to current monotonic time, including final results; only explicit play-again starts another duel. Removed obsolete unused streak engine/tests and updated their check/include callers.
- Penalty-specific hero/directory/field availability copy now states private duel availability; race/duck/voice still say upcoming. Internal application constructor accepts the existing world.Clock for deterministic integration tests; production `Application.New` still uses WallClock. No worker Step was driven concurrently with Run.

## Tests first, failures and corrections

1. Engine RED: from apps/server, `GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test ./internal/games/penalty` failed with `undefined: Engine` / `undefined: New` before implementation. After engine implementation, the same command passed. An initial shell invocation used the wrong cwd; temporary nested directories were removed before implementation continued.
2. Admitted hot-path RED: focused realtime test failed because `WithAdmitted` and seat `authorizedUntil` did not exist. Implemented the bounded lease callback, then focused `TestPenaltyAdmitted...` passed. It asserts the blocking auth provider is never called and denies wrong epoch, expired lease and revoked seat.
3. Guest engine RED: `npm run test --workspace apps/web -- penalty` failed to resolve the not-yet-created `features/penalty/local`. Implemented the local engine, after which shared-fixture tests passed.
4. Root CUA discovered a real modal-lifetime defect: closing/reopening practice reset the match despite the UI's continuation copy. Added guest-session regression (RED missing `features/penalty/session`), moved lifetime outside the React modal, then controller tests passed and root's actual CUA reopen/results retest passed. The first root retest was interrupted by Vite full reload and is explicitly inconclusive, not a passing test.
5. Initial local DB/transport tests hit sandbox loopback restrictions (`operation not permitted`). Scoped escalation allowed the authorized isolated fixtures; no preview database was replaced. The first real integration run found a virtual-clock/ordinary-context-deadline comparison error at later turns. Fixed the comparison to use the explicit admitted authorization expiry at owner ingestion rather than treating a real operation-context deadline as simulation time; both integration fixtures now pass.
6. First broad unit run found legacy `TestValidateCommand` still sending zone-only input. Updated it to required match/turn scope; strict malformed/missing/spoofed timestamp/actor fixtures are shared in Go/TS. A temporary TypeScript UUID-default inference issue in the new session test was fixed by declaring its optional ID as string. No known failing test remains in executed suites.

## Executed commands and exact result summaries

Working directory for Go commands: `/Users/patra/Project/patra-world-social/apps/server`.
The ignored PG environment was loaded without printing credentials:

```sh
set -a
source ../../.cache/database.env
set +a
```

Focused engine command:

```sh
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test ./internal/games/penalty
```

Latest output: `ok patra-world-social/server/internal/games/penalty 0.478s` (exit0).

Focused real integration command:

```sh
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -tags=integration ./internal/application -run TestPenalty -count=1 -v
```

```text
=== RUN   TestPenaltyActualTransportHistoryAndMutualRematch
--- PASS: TestPenaltyActualTransportHistoryAndMutualRematch (0.39s)
=== RUN   TestPenaltyOnlineBotNaturalResultReturnAndAbandonedCleanup
--- PASS: TestPenaltyOnlineBotNaturalResultReturnAndAbandonedCleanup (0.59s)
PASS
ok patra-world-social/server/internal/application 1.576s
```

Final full-suite command:

```sh
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -tags=integration ./...
```

```text
?  patra-world-social/server/cmd/e2edb [no test files]
ok patra-world-social/server/cmd/server (cached)
ok patra-world-social/server/internal/application 23.783s
ok patra-world-social/server/internal/auth (cached)
ok patra-world-social/server/internal/chat (cached)
ok patra-world-social/server/internal/config (cached)
?  patra-world-social/server/internal/contracts [no test files]
?  patra-world-social/server/internal/db [no test files]
ok patra-world-social/server/internal/games/model (cached)
ok patra-world-social/server/internal/games/penalty 1.611s
ok patra-world-social/server/internal/history (cached)
ok patra-world-social/server/internal/http (cached)
ok patra-world-social/server/internal/lobby (cached)
ok patra-world-social/server/internal/moderation (cached)
ok patra-world-social/server/internal/realtime (cached)
ok patra-world-social/server/internal/social (cached)
?  patra-world-social/server/internal/testutil [no test files]
ok patra-world-social/server/internal/world (cached)
ok patra-world-social/server/internal/worlds (cached)
```

Race command:

```sh
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -race -tags=integration ./internal/games/penalty ./internal/realtime ./internal/lobby ./internal/application ./internal/history
```

```text
ok patra-world-social/server/internal/games/penalty 1.931s
ok patra-world-social/server/internal/realtime 10.423s
ok patra-world-social/server/internal/lobby 5.443s
ok patra-world-social/server/internal/application 22.156s
ok patra-world-social/server/internal/history 24.310s
```

These also exited0 with no output:

```sh
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go vet ./...
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go build -o ../../.cache/t9-server ./cmd/server
```

Working directory for web/diff commands: `/Users/patra/Project/patra-world-social`.

```sh
npm run test --workspace apps/web -- penalty
npm run typecheck --workspace apps/web
npm run build --workspace apps/web
npm run lint --workspace apps/web
git diff --check
```

Final test output: JS `tests4 / pass4 / fail0`; bundled TS `tests36 / pass36 / fail0 / skipped0`. This is the full configured web suite (the trailing `penalty` argument is not a test-name filter in this repository). Typecheck/lint/diff-check exit0. Vite build: `54 modules transformed`, dist index7.34kB, CSS58.38kB, JS313.32kB; `built in 599ms` (exit0). Build only; preview was not restarted.

## Authored versus executed acceptance

Executed automatically: all16 combinations; exact2999ms accept/3000ms reject; empty-slot-only independent timeout; no pre-reveal snapshots; bot blindness; early win; equal sudden-death pairs/max draw; leading-score forfeit; both-gone abandonment; unchanged reconnect deadlines; full1500ms animation; actual lobby natural-result-before-grace; bounded owner-ingestion late input rejection; no-SQL admitted callback with expiry/epoch/revoke fencing; real synthetic WS commands; spectator/world denial; immutable per-turn projection retries; H-H mutual rematch; H-B natural finish/return; departed-result cleanup; shared Go/TS timelines; guest session reopen/time catch-up; server-clock rendering/disabled chosen and spectator controls. Existing startup recovery/storage uncertainty/return-generation suites also passed unchanged.

Executed browser evidence is **root CUA guest-only**, not implementer Playwright: root reported choice locking and resolved timeout shots, then after fix close/reopen retained active score1:0 with two turns and0.2s timer; later close/reopen showed final Guest4:3 BOT,10 resolved kicks, correct win/loss and play-again. Console warnings/errors empty. Root saved `docs/checkpoints/evidence/CP4/t9-guest-duel.png`, `t9-guest-reopen-before-fix.png`, `t9-guest-reopen-fixed.png`, `t9-guest-results.png`. These files and ledger edits are root-owned, not part of this implementer's authored change list.

Authored **NOT_RUN**: `apps/web/tests/penalty.guest.e2e.spec.ts` and separate credential-free `playwright.guest.config.ts` cover actual modal unmount/reopen, hidden-tab elapsed time, device Date.now shift and reduced-motion toggle. Root explicitly instructed that browser interaction in this session uses CUA, so no CLI Playwright/browser automation was run by this implementer. Two authenticated human browser contexts, credential/invite/session workflows, physical device clock manipulation and online reconnect browser acceptance were **not executed and are not claimed PASS**. Backend synthetic admitted fixtures are not a substitute for that pending external browser acceptance.

## Authored changed files

Go new:
- `apps/server/internal/games/penalty/engine.go`, `bot.go`, `adapter.go`, `engine_test.go`, `adapter_test.go`
- `apps/server/internal/db/migrations/006_penalty.sql`
- `apps/server/internal/lobby/game.go`
- `apps/server/internal/realtime/penalty_admission_test.go`
- `apps/server/internal/application/penalty_integration_test.go`

Go modified:
- `apps/server/internal/application/application.go`, `lobby.go`
- `apps/server/internal/lobby/service.go`, `presence.go`
- `apps/server/internal/realtime/hub.go`, `delivery.go`, `protocol.go`, `protocol_test.go`
- `apps/server/internal/history/service.go`

Web new:
- `apps/web/src/features/penalty/local.ts`, `session.ts`, `Duel.tsx`, `Pitch.jsx`, `penalty.css`
- `apps/web/tests/penalty-duel.test.ts`, `penalty.guest.e2e.spec.ts`
- `apps/web/playwright.guest.config.ts`

Web modified:
- `apps/web/src/components/PenaltyGame.jsx`
- `apps/web/src/features/lobby/LobbyPanel.tsx`, `features/history/MatchDetail.tsx`
- `apps/web/src/net/socket.ts`, `world/avatars.ts`
- `apps/web/src/app.js`, `data.js`, `apps/web/index.html`
- `apps/web/scripts/test-map.mjs`, `tests/map.test.ts`, `tsconfig.json`, `package.json`

Web deleted: unused `apps/web/src/penalty.js`, obsolete `apps/web/tests/penalty.test.js`.

Contracts: new `contracts/penalty.golden.json`; modified `contracts/protocol.schema.json`, `event.schema.json`, `commands.golden.json`, `README.md`.

Report: `.superpowers/sdd/2026-10-26-patra-world-social/task-9-report.md`.

## Limits / remaining review

- Raw socket arrival before a deadline is not an acceptance guarantee; queue delay can produce a correct late rejection. This explicit approved boundary is documented in contracts.
- Guest practice is in-memory for this page session, preserved across modal unmount/reopen. A full page reload starts a new guest session; no official result upload occurs.
- Private authenticated two-browser UI acceptance remains pending; no credential workaround was used.
- Root noted a minor pre-existing-style leading-zero zone-label overlap for T15 visual polish. No unrelated visual redesign was attempted.
- This retains the existing single authoritative game-process deployment assumption; no distributed engine recovery was added. Race/duck/voice remain later tasks.

# T9 fix round 1 — accepted review findings 1–3

All three Important findings in `task-9-review.md` were reproduced with permanent regressions and fixed. Changes in this round remain **unstaged against the staged review baseline**. No index/reset/commit operations, frontend edits, browser work, source-repo edits or process restarts were performed. Root-owned ledger/evidence edits were left untouched.

## Finding 1: original notification time and one grace boundary

Added optional generic `lobby.TimedPresenceEngine.PresenceAt(user,kind,at)`. Penalty implements it; the old Presence method remains compatible. Future concrete race/duck engines can receive original notification times and the explicit `expired` grace boundary. Untimed legacy fixtures retain their old leave fallback.

The lobby now records the entire batch of pending disconnect metadata before any catch-up evaluates it; pending events then apply in notification-time order. The ticker path also drains pending presence before advancement. Original `disconnectedAt` passes through the adapter rather than being restamped at ingestion. Timely resume retains its notification time. A coalesced resume at/after original15seconds remains disconnected for the natural/grace evaluation; it is not rewritten into an intentional early leave. Expiry is delivered as `expired` at `disconnectedAt+15s`. Original intentional-departure time/kind is retained for an engine that activates after start persistence. Engine disconnect metadata installs without an early catch-up, so processing the first player cannot finalize before the second pending disconnect is known. Natural results finish only at/before the true original grace cutoff; an abandoned/forfeit result records that cutoff in FinishedAt. Terminal times cannot precede official StartsAt.

Permanent coverage:

- `apps/server/internal/games/penalty/review_regression_test.go`: `TestDelayedDualDisconnectUsesOriginalGraceAndFinishedAt` holds the real lobby owner, records both disconnects100000, ingests100100, expires115000, and requires abandoned/abandoned plus FinishedAt115000.
- Same file, `TestCoalescedResumeAtOriginalGraceBoundary`: resume114999,115000,115001 with ingestion115100; timely resume remains playable, exact/late resume produces forfeit/win at115000. `TestBothCoalescedResumesCannotReviveOriginalGrace` checks both humans' exact-boundary coalesced resumes remain abandoned at115000.
- Same file, `TestDelayedNaturalFinishBeforeAtAfterOriginalGraceCutoff`: identical deterministic engine RNG in a real lobby, delayed catch-up whose natural finish is one millisecond before, exactly at, or one millisecond after the original grace cutoff. Asserts exact outcomes and FinishedAt; the post-cutoff natural result must lose to abandonment.
- `apps/server/internal/lobby/presence_timing_test.go`: `TestTimedEngineReceivesNotificationAndGraceTimeWithoutIntentionalLeave` proves a generic timed engine receives disconnect at the original notification time and `expired` at its original exact15-second boundary, with no intentional-leave fallback.

## Finding 2: rejected input cannot consume opponent authority

The replay ledger now records only a successfully accepted immutable choice. Rejected actor/turn/zone/locked-choice attempts do not enter it. The duel admits at most40 accepted choices, so this remains bounded without a shared rejection budget. Accepted command reuse remains rejected across later turns.

- `TestRejectedChoiceFloodPreservesBothActorsLaterEligibility` in `review_regression_test.go` accepts one command, rejects512 distinct stale/locked attempts, advances to another turn, rejects cross-turn reuse of the accepted command, then accepts timely first choices from both humans.
- `apps/server/internal/application/penalty_integration_test.go` extends `TestPenaltyActualTransportHistoryAndMutualRematch` with512 distinct schema-valid stale/locked commands through `Hub.Execute` → live admitted-seat authorization → actual application dispatcher → owner input → adapter. Both participants can still choose in later turns. This is an admitted application dispatch regression, **not a claim of running a timed WebSocket flood**; ordinary WS deadline/duplicate coverage remains in the same test.

## Finding 3: atomic consented rematch ownership

`ResultAction` now returns only readiness and retains an internal reservation of the old results entry, arena and both busy claims. Application calls `CompleteResultAction` instead of separate ReleaseMatch/JoinLobby requests. Complete performs the guarded `Terminal(completed)` storage operation outside the owner, then replaces the exact consenting pair in one reserved owner operation. It generates fresh lobby/actor IDs, keeps user order/host, resets Ready to false, and increments claim generations without ever making either busy claim empty. It closes the old engine/removes its old start attempt but emits no arena returns for the rematched pair. Old spectators are not promoted. Generic ReleaseMatch refuses an active result-action reservation. The internal retry helper is no longer exported.

Storage failure or a failed completed-result guard leaves the old results entry/roster/claims intact and permits explicit retry. If a consenting actor leaves/is revoked through the existing leave notification, disconnects, or reaches permanent grace expiry during the external release, completion rechecks current owner state and cancels the entire replacement. After successful durable release it detaches the old entry and returns users; it never rebuilds a departed actor or a partial new pair. Per-user claim-release work is held during the reservation so a departure cannot create an intervening other-arena claim.

Permanent coverage in `apps/server/internal/lobby/rematch_test.go`:

- `TestRematchPreservesAtomicPairClaimsAcrossGuardedDurableRelease`: storage release is gated. A third normal join can only spectate the old entry; a consenting participant's race join remains ErrBusy. A separate generic release cannot bypass the reservation. After gate release, only the exact pair exists, with fresh lobby/actor IDs and Ready false; no old spectator joins it. Third-user join is ErrFull and other-arena join remains ErrBusy. Stale return guards fail both while newly claimed and after subsequently releasing one new claim, proving generation advancement.
- `TestRematchGuardedReleaseFailureRetainsOldResultsAndClaims`: injected storage failure leaves old results/IDs/roster/claims intact; retry succeeds.
- `TestRematchCannotManufactureDurableCompletion`: store's completed-result guard rejects release; old state/claims remain intact.
- `TestRematchDoesNotRestoreDepartedOrRevokedConsentDuringRelease`: leave/revocation-style notification and permanent disconnect expiry during gated storage work cancel all replacement without restoring the departed actor or creating a partial roster.

Those deterministic boundary/failure tests use the isolated store stub to control the exact interleaving, not a claimed real PostgreSQL outage. The real PG application mutual-rematch test remains and now compares the durable old result job hash **and payload bytes** before/after replacement and asserts fresh participant identities. Its ordinary mutual-consent, H-B natural result/return and completed-departure cleanup checks continue to pass.

## RED → GREEN and executed verification

Initial permanent RED command, from apps/server:

```sh
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test ./internal/games/penalty -run 'TestDelayed|TestCoalesced|TestRejected' -count=1 -v
```

Observed expected failures before the fixes:

```text
FAIL TestDelayedDualDisconnectUsesOriginalGraceAndFinishedAt
  original grace115000: got 115000 win/forfeit
FAIL TestCoalescedResumeAtOriginalGraceBoundary/0
  FinishedAt115100 instead of115000
FAIL TestCoalescedResumeAtOriginalGraceBoundary/1
  FinishedAt115100 instead of115000
FAIL TestDelayedNaturalFinishBeforeAtAfterOriginalGraceCutoff/-1
  natural after true cutoff won (FinishedAt140500; cutoff140499)
FAIL TestRejectedChoiceFloodPreservesBothActorsLaterEligibility
  failed flood disabled valid actor v choice closed, locked, stale, or unauthorized
FAIL patra-world-social/server/internal/games/penalty 0.505s
```

The new rematch test was written before its replacement seam; its initial command:

```sh
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test ./internal/lobby -run TestRematch -count=1
```

failed with `s.CompleteResultAction undefined`, then passed after the reserved atomic completion was implemented. The old reviewer rematch probe describes the removed multi-request release/rejoin sequence; the permanent rematch tests now exercise its replacement seam directly.

After the fixes, the complete affected unit packages passed:

```sh
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test ./internal/games/penalty ./internal/lobby -count=1
```

```text
ok patra-world-social/server/internal/games/penalty 0.921s
ok patra-world-social/server/internal/lobby 0.503s
```

Loaded the same ignored isolated PG environment without printing credentials, then ran all affected packages with race detection:

```sh
set -a
source ../../.cache/database.env
set +a
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -race -tags=integration ./internal/games/penalty ./internal/lobby ./internal/application
```

```text
ok patra-world-social/server/internal/games/penalty 1.529s
ok patra-world-social/server/internal/lobby 2.566s
ok patra-world-social/server/internal/application 20.536s
```

These exited0 with no output:

```sh
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go vet ./internal/games/penalty ./internal/lobby ./internal/application
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go build -o ../../.cache/t9-fix1-server ./cmd/server
```

After internal helper/comment/query cleanup, final focused verification:

```sh
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -tags=integration ./internal/games/penalty ./internal/lobby ./internal/application -run 'TestDelayed|TestCoalesced|TestBothCoalesced|TestRejected|TestRematch|TestTimedEngine|TestPenalty' -count=1 -v
```

```text
PASS TestDelayedDualDisconnectUsesOriginalGraceAndFinishedAt
PASS TestCoalescedResumeAtOriginalGraceBoundary (-1,0,1)
PASS TestDelayedNaturalFinishBeforeAtAfterOriginalGraceCutoff (-1,0,1)
PASS TestRejectedChoiceFloodPreservesBothActorsLaterEligibility
PASS TestBothCoalescedResumesCannotReviveOriginalGrace
ok patra-world-social/server/internal/games/penalty 0.497s
PASS TestTimedEngineReceivesNotificationAndGraceTimeWithoutIntentionalLeave
PASS TestRematchPreservesAtomicPairClaimsAcrossGuardedDurableRelease
PASS TestRematchGuardedReleaseFailureRetainsOldResultsAndClaims
PASS TestRematchDoesNotRestoreDepartedOrRevokedConsentDuringRelease (leave,expired-disconnect)
PASS TestRematchCannotManufactureDurableCompletion
PASS TestCoalescedDisconnectResumeReachesEngineAndMarksAssistance
ok patra-world-social/server/internal/lobby 0.906s
PASS TestPenaltyActualTransportHistoryAndMutualRematch (0.36s)
PASS TestPenaltyOnlineBotNaturalResultReturnAndAbandonedCleanup (0.59s)
ok patra-world-social/server/internal/application 2.838s
```

`git diff --check`: exit0, no output. No current failing check is being resolved.

## Round1 file scope and remaining limits

New (untracked, must be explicitly included in root's review package):
- `apps/server/internal/games/penalty/review_regression_test.go`
- `apps/server/internal/lobby/presence_timing_test.go`
- `apps/server/internal/lobby/rematch_test.go`

Modified (unstaged):
- `apps/server/internal/games/penalty/adapter.go`, `engine.go`
- `apps/server/internal/lobby/registry.go`, `service.go`, `presence.go`, `start.go`, `game.go`
- `apps/server/internal/application/lobby.go`, `penalty_integration_test.go`
- this scratch report.

No frontend was amended, so unchanged web suites were not repeated. Authenticated browser/device acceptance, guest Playwright NOT_RUN status, raw-frame arrival limitation, T15 zone-label polish and the single-process assumption remain exactly as previously recorded. No new browser acceptance or CP4 PASS is claimed. T8 worker/startup recovery code was not edited; scoped application integration/race coverage includes its existing lifecycle and return-generation tests.

# T9 fix round 2 — activation regression R1 only

Read `task-9-rereview1.md` fully. Original findings1–3 remain addressed; this round fixes only the blocking R1 fresh-engine activation regression. O1 (processed disconnect followed by coalesced resume/disconnect) was **not modified** and remains explicitly carried to T16/final review per coordinator instruction. No new browser acceptance is claimed.

Baseline supplied for this round: staged tree `2f9a1bae9afa38b1d287d677f6e86ead874a68f6`, HEAD unchanged `e613707a6f64faea291c137504d2de7a1170e9b2`. All round2 edits remain unstaged; no staging/reset/commit/source-repo/browser/process operations occurred. Root-owned docs/evidence were left untouched.

## Change and covering tests

`apps/server/internal/lobby/start.go` now installs **all preserved original disconnect metadata** into the newly activated engine before replaying any saved departure/expiry event. The fresh engine can therefore interpret `expired` against its original disconnect cutoff, rather than treating it as a no-op. After replay, an immediately running engine is ticked and a terminal result is marked finalizing before the start completion returns. The all-gone preactivation branch remains unchanged; no engine/result is fabricated when nobody remains. Existing engine terminal-time clamping preserves `FinishedAt >= StartsAt`.

New untracked test file (include explicitly in root's scoped package):
`apps/server/internal/games/penalty/activation_test.go`.

It uses the actual lobby plus penalty adapter, a held start-persistence store and deterministic clock:

- `TestActivationReplaysOneExpiredGraceBeforeFreshEngineCanContinue`: two humans ready; Persist held; `u` disconnects at100000; exactly `u` expires at115000 while `v` remains connected; persistence completes at115000 or115100. CopyResult is available immediately without an explicit Sync. It requires forfeit/win, unchanged ordered frozen actors, zero resolved timeout turns, actual official StartsAt, and FinishedAt equal to that StartsAt (including clamp from115000 grace to115100 official start).
- `TestActivationPreservesStillInGraceOriginalRemainingTime`: persistence completes at114999. The duel remains choosing with original remaining grace rather than finishing early or granting another15seconds. It forfeits `u` at original115000, one millisecond after official start.
- `TestActivationRetainsAllGoneWithoutFabricatingResult`: both humans expire during held persistence; start is forbidden, no copied result exists, and the controlled store receives abandoned cleanup.
- `TestActivationRetainsOriginalIntentionalLeaveAndClampsFinish`: an intentional leave during held persistence remains forfeit/win, retains frozen roster and clamps the terminal time to official start.

The controlled persistence gate establishes the activation interleaving; it is not a claimed real PostgreSQL outage/reconciliation experiment. Existing isolated PG lobby/application tests were also executed below. No change to O1, timed presence protocol, replay ledger or atomic rematch logic was needed.

## RED → GREEN and exact checks

From `/Users/patra/Project/patra-world-social/apps/server`, before changing production code:

```sh
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test ./internal/games/penalty -run TestActivation -count=1 -v
```

```text
FAIL TestActivationReplaysOneExpiredGraceBeforeFreshEngineCanContinue/115000
  activation did not immediately expose finalizing result lobby action forbidden
FAIL TestActivationReplaysOneExpiredGraceBeforeFreshEngineCanContinue/115100
  activation did not immediately expose finalizing result lobby action forbidden
PASS TestActivationPreservesStillInGraceOriginalRemainingTime
PASS TestActivationRetainsAllGoneWithoutFabricatingResult
PASS TestActivationRetainsOriginalIntentionalLeaveAndClampsFinish
FAIL patra-world-social/server/internal/games/penalty 0.514s
```

After the activation fix, the same command:

```text
PASS TestActivationReplaysOneExpiredGraceBeforeFreshEngineCanContinue
  PASS /115000
  PASS /115100
PASS TestActivationPreservesStillInGraceOriginalRemainingTime
PASS TestActivationRetainsAllGoneWithoutFabricatingResult
PASS TestActivationRetainsOriginalIntentionalLeaveAndClampsFinish
ok patra-world-social/server/internal/games/penalty 0.443s
```

Existing lobby unit suite:

```sh
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test ./internal/lobby -count=1
```

Output: `ok patra-world-social/server/internal/lobby 0.435s` (exit0).

Bounded affected-path verification, using the same ignored isolated PG environment without printing credentials:

```sh
set -a
source ../../.cache/database.env
set +a
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -race -tags=integration ./internal/games/penalty ./internal/lobby
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -tags=integration ./internal/application -run TestPenalty -count=1
```

```text
ok patra-world-social/server/internal/games/penalty 1.482s
ok patra-world-social/server/internal/lobby 2.915s
ok patra-world-social/server/internal/application 1.448s
```

These exited0 without output:

```sh
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go vet ./internal/lobby ./internal/games/penalty
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go build -o ../../.cache/t9-fix2-server ./cmd/server
```

From repository root, `git diff --check`: exit0, no output. No currently failing check is being resolved. Unchanged frontend suites were not repeated. T8 worker/recovery/return-generation code and the deferred external browser/device acceptance limits remain unchanged.
