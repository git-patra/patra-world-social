# T9 scoped rereview — fix round 1

What this change does: The fixes carry original presence timestamps into the engine, record only accepted choices for replay protection, and reserve rematch ownership through durable release before replacing the pair atomically.

Scope: complete `task-9-fix1.diff` (46,286 bytes, including all three new test files), appended fix section of `task-9-report.md`, and the unchanged binding brief/context. Base/HEAD remains `e613707a6f64faea291c137504d2de7a1170e9b2`; fixes were reviewed against original index tree `daf117ee2cf32e84985d4bba61bc227dabefb6ff`. No source/index/HEAD edits, browser actions, credentials, restarts, subagents, or broad suite reruns. Supporting reads and two focused overlay probes addressed concrete unanswered lifecycle questions.

**Spec verdict: REQUEST CHANGES. Quality verdict: REQUEST CHANGES.** Original findings 1–3 are addressed as scoped below, but fix-round regression R1 is Important and blocking. No Critical finding. The separate pre-existing observation O1 is nonblocking for this scoped rereview and should be carried to T16/final review, per coordinator direction.

## Original finding status

| Finding | Status | Evidence and limit |
| --- | --- | --- |
| 1. Delayed disconnect creates a false winner | **ADDRESSED** for the original delayed/coalesced notification cases | `TimedPresenceEngine` carries the notification timestamp. The lobby installs the batch's disconnect metadata before catch-up, drains presence before ticker advancement, and emits `expired` at the original grace boundary. `TestDelayedDualDisconnectUsesOriginalGraceAndFinishedAt` reproduces the exact original 100000/100100/115000 case and asserts abandoned/abandoned and FinishedAt115000. Coalesced resume tests cover just-before, exact, and just-after grace; both exact-boundary resumes; natural finish before/at/after the original cutoff; and the generic timestamp/kind seam. These are meaningful assertions against the original defect. R1 below affects a different activation path introduced by this fix. |
| 2. Failed-command replay budget disables the opponent | **ADDRESSED** | `adapter.go:71–82` removes the shared rejection ceiling and records an ID only after a successful immutable choice. Accepted choices are inherently bounded to 40. The permanent adapter test rejects 512 distinct stale/locked commands, rejects cross-turn reuse of an accepted command, and then admits both actors. The real application fixture adds the same flood through live admitted authorization and owner dispatch. This is correctly reported as direct admitted application dispatch, not a timed WS flood. |
| 3. Release/rejoin splits a consented rematch | **ADDRESSED** | Application now calls `CompleteResultAction`; `game.go:121–177` retains the old entry and busy claims during guarded storage release, checks the reserved entry again, then replaces the complete pair in one owner operation. Fresh actor/lobby IDs and false Ready flags are built together; claims receive new generations without becoming empty; no rematch return is emitted. Gated tests interleave a third member's join, a participant's other-arena join, and generic release; test storage failure/undurable guard/retry; and cancel replacement after leave or permanent disconnect expiry. Stale return guards are checked both during the fresh claim and after one new claim is released. The PG fixture also compares the old result hash and payload bytes before/after rematch. |

The rematch failure tests use a controlled store stub to prove owner ordering and rollback behavior. The PG fixture supplies ordinary integration evidence. The report keeps these distinctions explicit. No new finding was identified in the replay or atomic rematch fix paths.

## Must fix — new regression in this fix diff

### R1. Important / P2 — A grace-expired player is ignored when the engine activates

**Changed location:** `apps/server/internal/lobby/start.go:194–198`, specifically line196. Related receiving code: `apps/server/internal/games/penalty/engine.go:191–212`.

- **What this is:** When start persistence finishes, the lobby activates a fresh engine and replays a previously departed player's saved `departureKind`. This round changed the old unconditional `leave` to `expired` for a player whose 15-second grace ended during the start attempt.
- **Problem:** The fresh penalty engine has never received that player's original disconnect. It handles `expired` only indirectly by `Tick(at)` consulting its disconnect array; there is no explicit expired branch. With an empty disconnect array, the replay is a no-op. Because the other player is still present, the all-gone guard does not prevent activation. The departed actor remains in a running duel and is filled by normal timeout choices instead of losing by forfeit.
- **Confirmed trigger:** Start a two-human duel with the store's start write held. Notify `u` disconnected at100000 and let the lobby ingest it. Advance to115000 so the lobby marks `u` permanently departed with kind `expired`. Complete the held start write while `v` remains present. The engine starts, receives `expired` without the disconnect metadata, and still has no result after `Sync`; `CopyResult` returns `lobby action forbidden` because the duel remains running. The controlled clock/store probe executes this interleaving without waiting15 real seconds. In production it corresponds to unresolved start persistence/reconciliation lasting beyond grace.
- **Fix:** When activating, install all preserved original disconnect metadata before replaying any grace expiration or other event that can finalize. Alternatively provide an explicit terminal expiry operation with enough original timing/state to preserve the same semantics. Preserve the all-gone path, original intentional-leave semantics, frozen roster, and the rule that FinishedAt cannot precede official StartsAt.
- **If we skip it:** A player who has already permanently departed can continue as an ordinary participant and receive a normal result, including a possible win, rather than forfeit.
- **Required regression:** Hold start persistence with two humans; expire exactly one player's original15-second grace before activation; then complete persistence. Require forfeit/win, the exact appropriate FinishedAt clamped to official StartsAt, and immediate finalizing/result availability. Include a still-in-grace disconnected participant to prove that normal remaining grace is preserved, and retain the existing all-gone/no-fabricated-result behavior. Exercise the actual lobby plus penalty adapter, not just a direct fresh-engine call.

## Separately recorded, pre-existing and nonblocking in this scoped rereview

### O1. Processed disconnect followed by coalesced resume/disconnect loses the timely resume

Locations: `apps/server/internal/lobby/presence.go:16–19` and `:404–411`. This behavior was also possible before fix round1; it is not treated as a new fix regression or an additional condition for closing original findings1–3. Coordinator explicitly requested carrying it to T16/final review.

Reproduction: In a running duel, process `u`'s disconnect at100000. Hold the owner, queue a timely resume at101000 and a new disconnect at102000, then ingest at102100. The pending record now represents the later disconnect, but `recordDisconnect` skips it because the player already has the first disconnect recorded; the coalesced resume never reaches the engine. At115000 the result is forfeit/win with FinishedAt115000, although the new continuous disconnection should expire at117000.

A focused probe confirmed that result. Required future regression: process the first disconnect, coalesce resume then disconnect while the owner is held, assert no forfeit at the old cutoff and correct expiry at the new cutoff. Also test a later final resume and an actually late first resume so coalescing cannot either revive expired authority or retain a grace period that a timely resume ended. Preserve sufficient ordered transition information rather than only the final notification when an earlier disconnect is already installed.

## Evidence assessed and focused probes

The appended report records passing affected penalty/lobby unit packages, affected application integration with race detection, scoped vet/build, and final targeted regressions. I read the assertions and implementation and did not rerun those suites. They substantiate the original fixes, but none covers grace expiry before fresh-engine activation or O1's processed-first-disconnect sequence.

Only the two concrete unanswered cases above were probed using a temporary Go overlay, without repository source changes. Probe source and mapping:

- `/private/tmp/t9-rereview1-probes/probe_test.go`
- `/private/tmp/t9-rereview1-probes/overlay.json`

Executed from `apps/server`, using the documented Go executable and GOCACHE:

```text
go test -overlay /private/tmp/t9-rereview1-probes/overlay.json ./internal/games/penalty -run '^TestReview1' -count=1 -v
FAIL TestReview1ProcessedDisconnectThenCoalescedResumeDisconnect
  timely resume discarded: result at 115000: forfeit/win; new grace expires 117000
FAIL TestReview1ExpiredParticipantBeforeActivationStillForfeits
  expired participant's departure ignored on activation: result remains pending: lobby action forbidden
```

These failures are isolated counterexamples, not broad-suite reruns. R1 is the blocking one.

## Acceptance limits

The eventual approval can cover T9 code scope only. Authenticated two-human browser, credential/invite/session workflows, online reconnect/device acceptance, and CP4 remain pending. Authored Playwright sources remain NOT_RUN. No browser/session workaround was used. Root's supplied guest CUA modal-lifetime/results evidence and the distinction from the interrupted Vite reload remain unchanged; full reload persistence remains T14. The T15 zone-label polish remains nonblocking. No race/duck/voice or later-game acceptance is implied. Preserve the accepted T8 startup recovery/work-budget and return-generation guarantees.

Verdict: fix R1 before approval. Original findings1–3 are addressed; carry O1 separately to T16/final review. Not checked: deferred authenticated browser/device acceptance and later-task features.
