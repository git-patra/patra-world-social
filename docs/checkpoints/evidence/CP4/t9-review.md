# T9 independent review

What this change does: T9 adds the authoritative penalty duel, its lobby/history integration, and matching guest practice rules. It adds hidden simultaneous choices, deadline reveals, semantic results, and return/rematch controls while reusing the existing art.

Reviewed snapshot: `task-9-review-package.diff`, 45 files, BASE/HEAD `e613707a6f64faea291c137504d2de7a1170e9b2`; no T9 commit. Requirements: `task-9-brief.md` and binding `task-9-review-context.md`. The complete diff and implementer report were read; supporting reads were limited to concrete caller, queue, presence, and release questions. Review assumes the approved single authoritative process and world beta limits. Source, index, HEAD, browser, preview, and root evidence were not modified. No subagents or broad test reruns were used.

**Spec compliance verdict: REQUEST CHANGES.** The ordinary rules are substantially implemented, but findings 1–3 violate grace/abandon correctness, independent player input authority, and mutual rematch ownership.

**Quality verdict: REQUEST CHANGES.** Three focused probes reproduce defects missed by the reported passing suites. No Critical finding. All three findings below are Important and must be fixed before approving the T9 code scope.

## Must fix

### 1. Important / P1 — Delayed disconnect ingestion can fabricate a winner

Changed location: `apps/server/internal/games/penalty/adapter.go:46` (the timestamp supplied to the engine), with changed lifecycle ordering at `apps/server/internal/lobby/presence.go:161–174`.

- **What this is:** The lobby retains the original notification time in `p.disconnected`, but the adapter tells the penalty engine that disconnection happened at `a.clock.Now()` when the owner finally processes the notification.
- **Problem:** These become different grace deadlines after any queue delay. The engine may still be running when the lobby expires the original 15-second grace. The lobby then calls `depart`, which forwards an intentional `leave` for the first expired participant and permanently records forfeit/win. With both players disconnected, the required outcome is abandoned. The same disagreement can let a natural result after the true grace boundary take priority because the engine uses a later cutoff.
- **Confirmed trigger:** Hold the lobby owner; notify both players disconnected at `100000`; let the owner ingest at `100100`; advance to `115000`. `CopyResult` returns `win/forfeit`, not `abandoned/abandoned`. This was reproduced through the real lobby plus adapter using a deterministic clock, not only direct `Engine.Tick`.
- **Fix:** Carry the original authoritative disconnect timestamp into the engine, including the earlier `disconnectedAt` preserved during notification coalescing. Keep resume/leave timestamps and grace expiry semantics consistent across the owner and engine; do not reinterpret grace expiration as intentional early departure. The approved owner-ingestion timing for **Choose** does not authorize shifting an already known disconnect time.
- **If we skip it:** A normal short scheduling delay can produce and durably store a false winner when both players leave. Natural/grace priority can also be decided against different deadlines.
- **Required regression:** Add the delayed dual-disconnect scenario above to the actual lobby/adapter suite. Also cover a coalesced disconnect/resume across the original exact 15-second boundary, and a delayed tick whose natural finish falls just before/at/after the original grace cutoff. Assert outcomes and `FinishedAt`, not just lobby terminal state.

### 2. Important / P2 — Failed choices can exhaust the opponent's input budget

Changed location: `apps/server/internal/games/penalty/adapter.go:66–75`.

- **What this is:** The adapter keeps a replay map keyed by user and command, but caps its total size for the whole match at 256. It inserts a key before validating the actor/turn/choice.
- **Problem:** One participant can send distinct, schema-valid commands with stale turn IDs. Each rejected attempt consumes the shared budget. At 256 entries, every future command from the other participant is also rejected. The transport's 10 control commands/second limit does not prevent this in a duel that can last 90 seconds; a client can remain below that rate and exhaust the map during the match.
- **Confirmed trigger:** After 256 rejected stale-turn commands from `u`, `v`'s valid first choice for the current open turn returns `ErrChoice`. A focused adapter probe reproduced this directly. It isolates the stateful bug; it does not claim to be a timed network flood test.
- **Fix:** Bound replay protection without letting one actor's failed commands consume another actor's eligibility. Recording only successfully accepted immutable choices is sufficient for the current first-choice-lock rules and bounds accepted entries to at most 40; otherwise use independently bounded actor state that does not disable future valid choices.
- **If we skip it:** A participant can force the opponent onto random timeout choices for the remainder of the duel, changing the outcome through invalid input.
- **Required regression:** Flood rejected distinct stale-turn/locked-choice commands from one actor, advance into a later turn, and prove the other actor can still submit a timely first choice. Retain the cross-turn duplicate accepted-command rejection test. Exercise the admitted application path with distinct command IDs as well as the adapter.

### 3. Important / P2 — Rematch releases ownership before rebuilding the agreed pair

Changed location: `apps/server/internal/application/lobby.go:76–83`.

- **What this is:** After both users consent, the application calls `ReleaseMatch`, which detaches the old lobby and releases its claims, then calls `JoinLobby` separately for each returning player.
- **Problem:** These are separate owner requests. An admitted third user can join the now-empty penalty arena between release and either rejoin. That user becomes host or fills a seat; the returning pair is only partially rebuilt and the second `JoinLobby` fails with arena full. The old rematch reservation is already gone, and the error path does not restore it. A concurrent join of one participant to another lobby creates a similar partial failure.
- **Confirmed trigger:** Complete the result state in an isolated store stub, collect both rematch approvals, run the same release/rejoin sequence, and interleave a third member's normal `JoinLobby` after release. The first consented participant joins the third member's lobby; the second is rejected with `arena full; spectate after start`. This deterministic interleaving is available between the distinct application calls. The probe isolates ownership after completion and does not stand in for PG durability tests.
- **Fix:** Preserve/reserve the agreed arena and player ownership through durable guarded release, then replace the old lobby with the new pair in one owner operation. Publish a fresh lobby/actor identity with Ready false only after the full replacement succeeds. Handle release failure without losing the old result or exposing a partial new roster; preserve the accepted return-generation fence.
- **If we skip it:** Two users who both approved a rematch may be separated, placed with an unintended third opponent, or left with a failed command after the old result was released.
- **Required regression:** Force a competing third-user join and a participant's other-arena join at the release/rebuild boundary. Verify all-or-none replacement of the exact consented pair, fresh lobby/actor IDs, unchanged durable old result, false Ready flags, and no stale return moving the newly claimed players. Retain the ordinary mutual-consent test and guarded release failure coverage.

## Test assessment and review probes

The implementer reports passing focused engine/application tests, full Go integration and race suites, vet/build, and the configured web tests/typecheck/build/lint. The reviewed sources substantiate meaningful coverage for all 16 combinations; the approved 2999/3000 owner-ingestion boundary; locks and missing-slot RNG; bot blindness; secret-free public views; fixed reveal/animation timing; early wins and equal sudden-death pairs; ordinary grace/natural ordering; typed leading-score forfeit; spectator/world denial; per-turn projection retry; ordinary two-human rematch; bot completion/return; and guest modal lifetime/catch-up. The Go/TS fixtures are actually read by both suites.

Coverage limits matter here:

- The existing integrated natural/grace test processes notifications without delayed ingestion. Direct engine tests use one timestamp source, so neither detects finding 1.
- The duplicate test covers reuse of one command ID, not exhaustion by many rejected IDs, so it does not detect finding 2.
- The two-human rematch integration test serializes both approvals and then inspects the finished roster. It does not interleave a competing claim, so it does not detect finding 3.
- The blocked-auth test proves the admitted callback itself skips storage. The queue-boundary test holds the lobby owner, while the HTTP/WS fixture proves actual command dispatch for the key deadline cases. These are distinct pieces of evidence; they do not promise raw frame-arrival acceptance, and this review does not demand that promise.
- The SSR view test checks server-time text and disabled chosen/spectator buttons. It is not interactive two-browser coverage. The reported web invocation runs the configured full web suite; the trailing `penalty` is not a filter.

Only focused review probes were executed, using Go's `-overlay` to add a temporary test file without editing the repository's source/index. Files are `/private/tmp/t9-review-probes/probe_test.go` and `overlay.json`. Commands from `apps/server`, with the documented GOCACHE and Go executable:

```text
go test -overlay /private/tmp/t9-review-probes/overlay.json ./internal/games/penalty -run '^TestReview' -count=1 -v
FAIL TestReviewFailedChoiceFloodDoesNotLockOpponent
  opponent valid first choice rejected after failed commands
FAIL TestReviewDelayedDualDisconnectRemainsAbandoned
  both disconnected at 100000, ingested 100100, exact grace 115000: got outcomes win/forfeit

go test -overlay /private/tmp/t9-review-probes/overlay.json ./internal/games/penalty -run '^TestReviewRematch' -count=1 -v
FAIL TestReviewRematchReservationSurvivesCompetingJoin
  consented rematch partially rebuilt: player v: arena full; spectate after start
```

The first command ran the first two probes before the third was appended. These expected failures are concrete counterexamples to the required invariants, not failures of a rerun of the implementer's broad suites.

## Acceptance limits and nonblocking notes

- Approval after fixes can cover **T9 code scope only**. This report currently grants no approval.
- Authenticated two-human browser, invite/credential lifecycle, online reconnect, and external device acceptance remain pending/user-deferred. No credential browser flow, session injection, workaround, or browser action was performed by this reviewer.
- Root's reported CUA guest evidence is accepted as the supplied scope context: stable-page modal close/reopen retains the duel and catches up to results. The review did not independently rerun browser work. The interrupted Vite reload is not a failed modal-lifetime retest. Full reload persistence remains T14.
- Authored Playwright sources remain NOT_RUN. The reviewed snapshot contains the guest Playwright source; the two-human online browser criterion is not fulfilled by the backend synthetic HTTP/WS tests.
- The minor leading-zero zone-label overlap remains the already ledgered T15 visual issue, not a new blocker.
- No CP4 PASS, concrete race/duck/voice acceptance, deployment approval, or later-game completion follows from this review. Existing T8 recovery/work-budget and return-generation behavior remains binding; the requested fixes must preserve it.

Verdict: fix 1, 2, and 3 first. Not checked: deferred authenticated browser/device acceptance and later-task features.
