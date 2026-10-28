# Kampung Senja backend verification

Verified 28 October 2026 against the working tree based on commit `b7ae4bd4cd5765dbae7a597815c288ac2c4738e2`. No application source edits or commits were made by this verification task.

## Result

- PASS: `go vet ./...` (exit 0).
- PASS: `go build -o ../../.cache/kampung-server-verification ./cmd/server` (exit 0).
- PASS: `go test -count=1 -race ./internal/realtime ./internal/world ./internal/voice ./internal/games/...` (exit 0; 7 packages).
- PASS: `go test -count=1 -race -tags=integration -json ./...` (exit 0; 20 tested packages, 242 top-level tests, 397 test/subtest pass events; no failures, no skipped tests; 4 packages contain no tests).
- PASS: post-run database inspection found 0 remaining schemas matching the test helpers (`worlds_test_`, `auth_test_`, `world_http_test_`, `social_test_`).

All Go commands ran from `apps/server` using `GOTOOLCHAIN=local`, `GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build`, and `/Users/patra/Project/patra-world-social/.cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go`. The private ignored `.cache/database.env` was sourced without displaying its contents. The database target was validated as loopback port 55432 before tests.

## Map contract checks

- `contracts/map.v1.json` and `apps/server/internal/world/assets/map.v1.json` are byte-identical.
- `contracts/voice-tables.v1.json` and `apps/server/internal/voice/tables.v1.json` are byte-identical.
- Map `kampung-senja-v1`, version 1, has valid canonical SHA-256 `71814ab8a87be0bd7166cbe2c1049f38e6fa774825ca87230fa0b155b0bb0768`.
- Voice map ID/version/checksum match the map contract. Table centers `(11,23)` and `(20,23)` are reachable from spawn using the generated collision map; 2,129 cells are reachable.

## Selected application integration evidence

| Test | Result | Seconds |
| --- | --- | ---: |
| `TestLoadSmoke` | pass | 21.43 |
| `TestDuckActualTransportMeasuredProfilesAndAuthority` | pass | 18.33 |
| `TestDuckWiredSchedulerAndIncompleteHistory` | pass | 5.26 |
| `TestDuckActualWiredScheduler30Hz15Hz` | pass | 1.01 |
| `TestPenaltyActualTransportHistoryAndMutualRematch` | pass | 1.9 |
| `TestPenaltyOnlineBotNaturalResultReturnAndAbandonedCleanup` | pass | 1.53 |
| `TestRaceApplicationGridPhysicsDurabilityAndTimedReturn` | pass | 1.85 |
| `TestRaceApplicationFiveHumanDeadlinePrecedesGraceExpiry` | pass | 1.16 |
| `TestRaceWiredSchedulerPublishes15HzAndHostDepartureDoesNotStopPeer` | pass | 1.23 |
| `TestRaceWebSocketContinuousInputUsesMovementBudget` | pass | 1.21 |
| `TestVoiceUsesActualAdmissionLifecycleAndJournal` | pass | 1.1 |
| `TestVoiceExcludesRunningMatchSpectatorsInBothOrders` | pass | 1.07 |

## Package outcomes

| Package | Result | Seconds |
| --- | --- | ---: |
| `cmd/server` | pass | 1.641 |
| `internal/config` | pass | 2.777 |
| `internal/games/duck` | pass | 3.034 |
| `internal/games/model` | pass | 3.255 |
| `internal/games/penalty` | pass | 3.586 |
| `internal/identity` | pass | 1.633 |
| `internal/games/race` | pass | 3.532 |
| `internal/chat` | pass | 5.42 |
| `internal/moderation` | pass | 5.508 |
| `internal/lobby` | pass | 7.485 |
| `internal/realtime` | pass | 10.679 |
| `internal/http` | pass | 13.735 |
| `internal/voice` | pass | 4.49 |
| `internal/world` | pass | 8.699 |
| `internal/leaderboard` | pass | 23.996 |
| `internal/social` | pass | 17.523 |
| `internal/auth` | pass | 32.716 |
| `internal/worlds` | pass | 16.682 |
| `internal/history` | pass | 41.039 |
| `internal/application` | pass | 72.658 |

## Environment recovery and initial attempts

1. Initial sandboxed race execution could not bind ephemeral loopback test listeners. Re-running with approved local listener access passed all 7 packages.
2. The dedicated `colima-patra-world-social` Docker daemon was unavailable. Inspection found its `patra-world-social` Colima profile broken, with a stalled host-agent PID and refused host-agent socket. A profile-scoped force stop and start restored the existing VM; no VM, volume, or database was deleted or reset.
3. An orphaned SSH mux from the previous session still owned host port 55432. TCP accepted connections but PostgreSQL handshake timed out; the database had no test connections, and the new host-agent log reported forwarding failure. The first full integration attempt therefore ended with connection errors (295 pass / 102 fail test-subtest events). Its log is preserved separately below as environment failure, not application regression evidence.
4. After confirming the mux belonged to this dedicated profile, it was terminated. The existing project Postgres container was restarted. Removing the old mux also unlinked the reused SSH control socket; the dedicated generated `ssh.config` restored the control socket and only the existing `127.0.0.1:55432` forward was reattached. PostgreSQL protocol handshake returned expected `N` before the fresh passing run.
5. Final runtime: only the dedicated profile/container were operated on; `patra-world-social-postgres` remains running and exposed at `127.0.0.1:55432`. The default and other Colima profiles were untouched.

## Evidence files

- `.cache/kampung-backend-race.log`: passing focused race suite.
- `.cache/kampung-backend-integration.jsonl`: complete passing integration/race events.
- `.cache/kampung-backend-integration-environment-failure.jsonl`: initial connection failures before local runtime repair.
- `.cache/kampung-server-verification`: successfully built server binary; not started by this task.

## Scope limits

These are backend tests with disposable schemas, synthetic test identities, ephemeral HTTP/WebSocket servers, and test voice providers. This verification does not establish browser multiplayer acceptance, microphone/browser permissions, or real LiveKit voice staging. No browser accounts were created or injected, no API server on 8080 was started, and the original `patra-world` repository was not modified.
