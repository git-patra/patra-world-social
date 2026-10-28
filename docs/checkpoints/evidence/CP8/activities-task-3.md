# Task 3 — activity contract and area authority

Implemented on feat/world-activities without a commit. Tasks 0–2 were not redone.

## RED and GREEN

- RED: new world clearing/path and old-generation tests did not compile because ReserveArea/ClearArea/ReleaseArea and ErrAreaClearing did not exist. New lobby start gate test did not compile because SetAreaCoordinator did not exist.
- GREEN owner regressions: clearing blocks existing paths, new destinations and continuously refreshed last-tick movement; stale user epochs do not bypass masks; old generation cannot release rematch; contract/mirror geometry is checked.
- GREEN lobby regressions: Start waits for clear ACK and confirmed durability; definite failure releases while unknown START retains; late result finalization releases spatial occupancy while durable account claims remain.
- GREEN actual world/lobby/hub/application regression: owner CLEARING is observed before advancing three seconds, persisted row is held behind a test barrier, engine stays inert until both ACKs, last-seat removal retains the reservation, ACTIVE follows durability, RESETTING→OPEN is independent of result storage, and actual Hub movement works while durable Busy remains true.
- GREEN realtime regression: empty leased owner survives last-seat removal and re-admission, released empty owner is reclaimed, its replacement uses a new incarnation and rejects old release; protocol-1 handshake explicitly requests reload before admission.

Latest full Go run (pinned Go 1.26.8, local toolchain/cache, isolated database loaded quietly):
`go test -race -tags integration ./internal/world ./internal/lobby ./internal/realtime ./internal/application`
All green: world 7.541s; lobby 7.377s; realtime 10.113s; application 64.205s.

Latest web run: typecheck passed; 148 bundled TS tests passed (plus four existing JS tests). Current command/event/snapshot fixtures migrated to protocol 2; original invalid future-version cases are now version 3, and explicit version-1 rejects were added. Historical match/result fixtures and metadata remain protocol 1. New guest results use protocol 2.

## Authority and lifecycle

- World Runtime owns AreaLease `{WorldID,Incarnation,AreaID,MatchID,Generation}` and all mutations. Reserve closes masks and clears movement atomically; Clear ACK waits three seconds; Activate is acknowledged before Engine.Start. Path search, queued nodes and each movement step check current masks.
- Application coordinator waits outside owners and SQL. Lobby RequestStart acknowledges acceptance immediately, so three-second clearing does not hold a transport admission lock. The existing StartMatch API still waits for completion for internal callers/tests. Persist runs after clear ACK; Activate ACK follows confirmed persist; unknown START retries retain the same lease.
- Lifecycle: OPEN→CLEARING→RESERVED→ACTIVE→RESETTING→OPEN. Reset is two seconds. Releases retry outside SQL/owner loops and survive detached lobby entries. Full lease equality prevents stale generation/incarnation releases.
- Rooms with live leases are retained, including when all seats leave. No empty-room recreation may drop an active reservation. After OPEN, normal empty-room reclamation resumes.
- MovementBusy is distinct from durable Busy: simulation completion permits walking; durable Busy still guards new-match admission/results. Voice exclusion keeps its existing separate semantics. Lobby publication and Hub movement both use MovementBusy.

## Contract and UI seams

`apps/web/src/activities/contracts.ts` exports ActivitySnapshot, AreaPhase, activityContract, validateActivitySnapshot and ActivityIntent. The only accepted foundation intent is exact empty `activity.snapshot`; gameplay action unions are intentionally not implemented before Tasks 5+.

`ActivitySnapshot`:
```
{worldId,areaId,incarnation,generation,serverTimeMs,tick,kind:'area',
 payload:{phase,matchId,startsAt,clearAt,resetAt}}
```

- `activity.lifecycle`: ordered bounded control queue; overflow requires reconnect.
- `activity.snapshot`: all four areas on initial join, then replaceable full states at 5 Hz directly from the owner. No unbounded snapshot mailbox.
- `WorldSocket.activitySnapshots()` requests `{}` and returns validated distinct all-area snapshots from `{activities:[...]}`.
- Socket handshake URL requires `protocolVersion=2`; current envelopes/world snapshots use 2. Existing REST/history version-1 metadata remains readable.
- Server trusted `Hub.WorldActivities(ctx,worldID)` supports coordinator inspection/tests; transport snapshot reads use the admitted seat-bound API.

## Geometry

Map schema stays 1 and grid stays 64×64 with seven districts, no CV. Asset ID is kampung-senja-v2. `scripts/generate-kampung-senja.mjs` regenerates map, voice identity and embedded activity mirror and checks safe anchors. `contracts/activities.v1.json` contains masks, entrances, safe returns, spectator strips, transforms, four vehicle slots, four fishing spots, one claw machine and bounded caps.

- Race mask `(37,37,16,11)`, return `(45,49)`. Uniform race-subunit→art scale 1/64, offset `(1488,1152)` produces outer art bounds `(1520,1184,576,384)`; Task4 scene owner received these coordinates.
- Penalty mask `(24,40,8,11)`, return `(28,52)`.
- Pingpong mask `(43,21,5,3)`, return `(45,25)`.
- Blaster obstacle-free open pad `(40,28,7,5)`, return `(47,33)`, separate from pingpong. Task4 scene owner handles its visible ground treatment; no new collision solids were invented here.
- Registry returns derive from activity safe returns, with Duck return from the map's actual Duck entrance.

## Files and integration notes

New Go files: world/{activities,areas,areas_test}.go + assets/activities.v1.json; realtime/{areas,areas_test,activity_protocol,activity_protocol_test}.go; lobby/{areas,areas_test}.go; application/{areas,areas_test,areas_integration_test}.go.

Modified world runtime/movement/activity; lobby start/service/presence/results/registry; realtime hub/protocol/connection/delivery/snapshot; application lobby; current protocol/map test fixtures; map/voice mirrors and generator. New contracts: activities.v1.json, activity.schema.json, activities.golden.json. Modified current protocol/event/snapshot schemas and goldens. Web: activities/contracts.ts, net/reconcile/socket, world/avatars, guest/completion and current wire tests. Root registered activity-contract.test.ts in runner/tsconfig; this agent did not edit those registration files.

Integration tests now wait for the accepted asynchronous start and advance fake time only after observing the actual CLEARING notice. Revocation races discovered by more frequent activity snapshots now close with explicit policy reason rather than an unexplained EOF.

## Scope and remaining staged work

No free-ride, football, claw/fishing, pingpong/blaster engine, React/controller or new game scene implementation here. They remain Tasks 4–10 owners. No deployment, credential browser workflow or physical-device/voice/account acceptance performed; existing deferred acceptance is unchanged. No claim that CP1 or any later checkpoint changed status. No known failing Task3 tests at handoff.
