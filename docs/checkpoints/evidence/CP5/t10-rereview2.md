# T10 fix round 2 — scoped rereview

**Spec compliance: Approved for the scoped R3 correction. Code quality: Approved.** R3 is addressed; no new Critical or Important finding in this fix. Accepted R1/R2 findings remain closed. This is code-scope approval, not CP5 acceptance.

Reviewed the complete four-file `task-10-fix2.diff` (15,646 bytes, 439 lines), R3 verbatim in `task-10-rereview1.md`, the appended round-2 report and saved real-owner RED evidence. Baseline staged tree: `cb2355e8e2c7f3c2cf65d82f3fe0b89979c11808`; HEAD remains `f64168fe26be58d8d2a4832f82b2d662a92a7c79`. Review remained read-only apart from this report: no source/index/HEAD changes, browser actions, process changes, subagents or Git commands.

## R3 — ADDRESSED

The metadata-only disconnect branch no longer stamps `BotAssisted`; it retains the original timestamp and prior input without advancing simulation (`apps/server/internal/games/race/engine.go:303–315`). Assistance is recorded after authoritative advancement, and natural finalization records only timestamps strictly before its global finish (`engine.go:261–271`). The helper only sets the flag, so an earlier legitimate takeover remains recorded after resume or a later ignored disconnect. Countdown and abandoned advancement retain effective assistance history (`engine.go:119`, `:156`, `:190`).

The controller selection also prevents a disconnect exactly at the known absolute/shortened deadline from changing that final frame to bot input. It retains the prior human intent, still subject to its original expiry checks (`engine.go:139–151`). This resolves both the post-cutoff history contamination and the exact-cutoff position change exposed by the new tests without reintroducing Tick into unordered metadata installation. No result model, serialization, projector, transport or generic presence contract changed.

The regression coverage answers the original failure and its required boundaries:

- `apps/server/internal/games/race/presence_test.go:79` compares complete final `MatchResult` values and terminal kart state against a no-new-disconnect baseline for eight combinations: absolute/shortened cutoff, exact/+1 ms disconnect, and never-assisted/previously-assisted actor. It checks that installation itself leaves assistance and timeline unchanged. The shortened fixture uses the actual ordered final-gate crossing logic to establish first finish +45 seconds, rather than assigning `deadline` (`presence_test.go:115`). This is focused engine-state setup, not a full race playtest.
- `presence_test.go:138` retains a real pre-cutoff bot takeover and true assistance. `presence_test.go:149` compares an earlier all-finished global result at 10033 with a later disconnect at 10040, including identical final positions and result data.
- `apps/server/internal/games/race/lobby_presence_test.go:277` holds the actual owner on its preceding running tick while a cutoff or cutoff+1 ms notification queues. The real adapter then processes metadata-before-advance and must produce the original result, unchanged kart state, false assistance for the never-assisted actor, one matching `race.finished`, and no repeat. The controlled store isolates owner ordering; it does not itself establish new PG persistence evidence.
- `apps/server/internal/games/race/adapter_test.go:59` verifies the shortened cutoff through the actual adapter, unchanged official result, no metadata-time publication, one matching terminal event and no repeated publication.

These are meaningful regressions with independent outcome/position/flag assertions. The earlier R2 permutation, held-input, grace, activation replay and terminal-event tests remain intact. No frontend change was made, so root's previously accepted real guest remount-control evidence remains applicable.

## Verification evidence

The appended report retains the pre-fix RED results: six of eight direct cutoff cases failed, and both held-owner cutoff cases failed. The saved `/private/tmp/t10-r3-owner-red.log` confirms both actual owner subcases failed with the pre-correction engine; this reviewer inspected that recorded evidence rather than regenerating it.

The report records final affected-package race-detector PASS for race/lobby (2.069s/1.974s), including the new shortened-adapter and all-finished tests; both affected application PG integration tests also passed with race detection (application 2.195s). Those integration cases cover grid/physics/durability/timed return and five-human deadline-before-grace. Formatting and diff checks were clean; the only later edit was a comment clarification. These are implementer-executed results whose assertions and reported output were reviewed, not independently rerun by this reviewer.

No concrete unresolved doubt required a new probe. No passing suite, browser test or unchanged full validation was repeated in this round.

## Separate, unchanged scope and acceptance limits

- Generic O1 remains ledgered for T16/final review. This fix does not change the generic owner coalescing algorithm or claim complete reconnect acceptance.
- The pre-existing personal-finish assistance edge—an individually finished actor disconnecting while other racers remain active—is not resolved here and is not a new blocking condition for R3. This approval concerns the global natural terminal boundary.
- Earlier nonblocking formatting suggestions and later-task guest loading/reload-persistence work remain separate.
- Human five-lap completion, 1–5-human playtests, novice bot tuning, physical mobile steering/FPS, real latency/jitter and live online browser acceptance remain unverified. Credential/account/invite browser workflows remain expressly deferred. Root's guest CUA evidence supports its stated controls/continuation/results observations only; backend fixtures and this approval do not grant CP5 PASS.

**Assessment: Approved for the scoped code correction.** R3 now respects the global terminal timestamp while preserving effective prior assistance and the accepted metadata-only R2 design. No additional code change is requested by this rereview.
