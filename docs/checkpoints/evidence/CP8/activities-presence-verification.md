# Task 0 presence fix — round 1

## Reproduction

Reviewer P1 reproduced against the real lobby owner and race adapter in
`apps/server/internal/games/race/lobby_presence_cycles_test.go`.

The held owner receives A disconnect, A resume, B leave, A disconnect, then
drains after the second disconnect. The existing first metadata pass installs
only A's first disconnect. Resume clears it. B's departure called
`engine.Tick(clock.Now())` before the second disconnect was installed.

The regression was run before the implementation change and failed in both
cases:

- `grace_expired`: disconnect 4000, resume 5000, B leave 5500, second disconnect
  6000, drain 24000. Actual abandonment happened at 24000 with different frame,
  positions and ranking from chronological replay (which abandoned at 23766
  under the existing bounded catch-up rules).
- `natural_finish`: disconnect 297000, resume 298000, B leave 298500, second
  disconnect 299000, drain 304500. Actual owner installed only one disconnect
  before finalizing at 304000, producing different positions/ranking from the
  complete timeline.

Expected snapshots come from feeding the transport timestamps directly into a
separate real race engine. The test compares the complete snapshot, including
terminal result, to the owner-managed adapter and requires both disconnects.

## Fix

`departAt` now calls `engine.Tick(at)` after delivering timestamped presence.
This retains the existing immediate forfeit/natural-result check before the
lobby can classify all players as departed. It restricts catch-up to the current
departure timestamp. The existing `advance()` consumes `clock.Now()` only after
the ordered batch has installed every transition.

The complete earliest-disconnect metadata pass remains intact, so batch grace
cutoffs and bounded physics catch-up retain their established behavior. The
32-item per-user queue, consecutive duplicate coalescing, original timestamp
retention, and fail-closed overflow policy were not changed by this fix.

This round changes only `presence.go`, the new real-engine regression test, and
this report. No commit or dependency change was made.

## Verification

All commands used the pinned Go 1.26.8 toolchain, `GOTOOLCHAIN=local`, and
`GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build`, from
`apps/server`.

- RED: `go test ./internal/games/race -run '^TestLobbyRaceQueuedLeaveCannotAdvancePastLaterDisconnectCycle$' -count=1`
  failed for both scenarios before the fix.
- GREEN: `go test ./internal/lobby ./internal/games/race ./internal/games/penalty ./internal/games/duck -count=1`
  passed all four packages.
- `go test -race ./internal/lobby ./internal/games/race ./internal/games/penalty ./internal/games/duck -count=1`
  passed all four packages with no race report.
- With approved loopback/database access, `go test -race -tags=integration ./internal/application -run 'Test(Race|Penalty)' -count=1`
  passed (11.190s). This covers actual race/penalty transport, timed returns,
  persistence, rematch and abandoned cleanup paths.
- `git diff --check` passed.

The focused suites include the original whole-disconnect-batch race cases,
activation replay, grace expiry, forfeit/all-gone engine tests, duck delayed
leave results, and penalty's
`TestLobbyNaturalResultBeforeBothPermanentDepartures`.

A single full `go test ./... -count=1` was also launched. Its game/lobby packages
and remaining non-network packages passed; sandbox loopback restrictions blocked
`TestServeUntilSignalWaitsForHTTPDrain` (both subcases),
`TestTransportAdmissionMovementAndRevocation`, and
`TestSDKSerializesExplicitCutoff` with `bind: operation not permitted`.
The affected packages were rerun once with approved loopback access:
`go test ./cmd/server ./internal/realtime ./internal/voice -count=1` passed all
three (1.080s / 8.976s / 0.423s). No unresolved test failure remains from these runs.
