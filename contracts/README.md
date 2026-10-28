# Version 1 contracts

Map coordinates and hub movement use tiles, with 1024 integer subunits per tile on simulation/wire boundaries. Hub speed is 4608 subunits/second (4.5 tiles/second), fixed tick 20 Hz. Race uses integer subunits and a 30 Hz tick; residual integer division must carry across ticks rather than truncate speed each tick. RNG and clock are injected interfaces, not JS/Go default random parity.

Map and rules `checksum` is SHA-256 of compact, recursively key-sorted JSON excluding the checksum field. Loaders validate structure; transport/asset integrity checking is a separate concern. Result records carry map/rules IDs, versions, checksums and protocol version (Go model.Version).

Schema files describe wire structure. Runtime contract-specific TS and Go decoders additionally validate bounds, footprint/entrance collision and payload constraints. Golden JSON files are consumed by both runtimes. JSON cannot encode NaN/Infinity; the TS object boundary also rejects nonfinite values before serialization. World/session authorization is required later at connection dispatch; a valid worldId is not authorization.

District rectangles are planning footprints. Mini-games are separate future scenes. New district portals are navigation landmarks only in T2. Rules are proposed beta tuning values, not a benchmark or a claim of online gameplay.

## T5 private-world transport

`GET /api/v1/realtime?worldId=<id>` upgrades only for an exact configured Origin,
an authenticated HttpOnly session cookie and active membership. Initial join emits
`world.snapshot`; every 100ms `world.delta` carries a complete replaceable snapshot
(the bounded latest-state strategy avoids delta gaps). `world.leave` releases the seat.

Snapshots include worldId, selfId, opaque connection sessionId, epoch, tick/timeMs,
map/rules IDs and checksums, protocolVersion, and at most 20 public avatars with
acknowledged input seq. Connection sessionId is random and unrelated to cookie token
or its hash; it is never shared with other avatars. The browser keeps it in memory.
`sessionId=<opaque>&epoch=<n>` resumes with a new opaque ID and increased epoch;
old IDs/epochs are fenced, including late old-socket cleanup. Resumption always
returns full state; T5 has no durable game/chat events to replay. `takeover=true`
is sent only after the user visibly chooses to take control from another tab.
A duplicate ordinary join emits `command.error` with `takeover_required` and closes.
World capacity includes disconnected seats for 30 seconds. A user owns one seat
across all worlds. Auth/membership are rechecked for commands and every second.

Movement positions are bounded fractional tiles at 4.5 tiles/sec on 20Hz server
steps. Direction intent expires at 250ms; destinations are integer walkable tiles,
server-pathfound and cancelable by direction/zero input. Sequence is scoped to the
connection epoch; old/duplicate sequence commands are acknowledged but do not move.
Snapshots and local rendered state cannot change authority. Remote interpolation
uses two snapshots with a 100ms delay and never extrapolates; self is reconciled
directly to the latest authoritative position. Animation preference is visual only.

Client frames are capped at 16KiB by transport and commands at 8KiB by the shared
decoder (a deliberately stricter existing bound). Per-connection limits are 60
movement and 10 control frames/sec, a 32-command queue, 2-second writes, heartbeat
ping every 5 seconds with a 3-second timeout, and a single writer. Snapshot ticker
coalesces under backpressure; no outbound unbounded queue exists. Slow/failed
transports retain a seat briefly and resync. Logout callback currently closes the
account's online seat, including if another valid cookie session owns it; only the
logged-out cookie is DB-revoked. Recovery revokes all account cookie sessions and
closes the online seat. Kick/ban/archive callbacks run after database commit.

`make load-smoke` uses DATABASE_URL or ignored `.cache/database.env`, creates a
random schema, applies migrations, starts a transient httptest server and synthetic
clients, then closes connections and drops the schema. It never touches the user's
development tables. Browser acceptance remains separate from this backend test.

Penalty uses four zones (0 top-left, 1 top-right, 2 bottom-left, 3 bottom-right).
`penalty.choose` requires payload `{matchId,turnId,zone}` and the envelope
`commandId`. Identity comes from the admitted session, never the payload.
The authoritative `receivedAt` is the lobby owner's ingestion time after the
bounded transport/owner queues. Raw frame arrival and device timestamps do not
prove timely acceptance. The first choice ingested before `choiceDeadlineAt`
locks; input at/after it is rejected. No queued input can revise a reveal.
The admitted-seat input path uses a maximum two-second authorization lease,
clipped by actual session/idle expiry and renewed by ongoing validation. Revoke,
world, token and connection epoch fencing still apply. Other controls use full
authorization. `penalty.snapshot` is a world-bound authorized public read.
All penalty events carry this public snapshot; `chosen` flags are public, and
zone/timeout details appear only in resolved `turns`. `penalty.golden.json` is
shared by the Go online engine and TypeScript guest duel.
`match.return` releases a durable completed result; `match.rematch` requires
both humans (one human for BOT) and creates a new lobby requiring Ready/Start.

## T10 five-lap race

`race-track.v1.json` defines the single Autumn Ring road, ordered directional
checkpoint gates, five start positions, racing line, and integer direction vectors
scaled by 1024. It is bound by `rules.v1.json.race.trackChecksum`. Race handling
uses subunits per 30 Hz step, with signed fractional direction remainders carried
between steps and cleared on wall contact. `race.golden.json` covers hand-checked
acceleration/steer/brake vectors and ordered/reversed/skipped checkpoint crossings
in both Go and TypeScript.

The proposed beta tuning is acceleration 5, brake 12, drag 2, maximum speed 230,
steering 3 degrees/step, kart radius 360 subunits, and normal bot target 155 subunits/step
(with modest slot variation and slower sharp corners). These are candidate
handling values; beginner balance and physical mobile testing remain release
acceptance work. Bots generate only ordinary controls, including during the 15 s
reconnect grace. They have no extra speed or collision privilege.

Before persistence, a bounded race preparation hook shuffles the complete roster
using seed-driven xorshift32 Fisher-Yates. Generic lobby code accepts only an exact
permutation of unchanged Actor values. `start_payload.actors`, `roster_slot`, engine
start positions, public `startSlot`, race_results and history all use this frozen
order. Equal simulation-frame finish times use that order, irrespective of finish
overshoot. Actor/MatchStart JSON contracts did not change.

`race.input` requires `{matchId,throttle,brake,steer}`: throttle/brake are integer 0/1,
steer is integer -1/0/1. Client position/lap/speed fields are rejected. Input sequence
is reset only on a fenced transport resume; intent expires after 250 ms. Race input
uses the existing admitted-seat authorization lease and the 60/s movement budget.
`race.snapshot` is an authorized world-bound public read, available to spectators.
Snapshots contain no seed or internal controls. Online results remain provisional
until the existing durable result worker confirms commit.

The lobby's 50 ms wakeup does not define physics. Elapsed server time schedules
integer 30 Hz frames; snapshots publish on every second frame (15 Hz), coalescing
backpressure. At most 8 frames execute per owner callback; overloaded/suspended
intervals discard old motion opportunities, never fabricate distance or laps,
and expose `droppedSteps`. The absolute 300 s limit and first-finish + 45 s limit still
bound play. Hub 20 Hz physics/10 Hz snapshots are unchanged. Render interpolation,
reduced-motion preferences and client clocks never supply authoritative movement.

After durable results, race returns require every remaining human's consent or
10 seconds. There is no race rematch/automatic queue. The guarded existing arena
return places humans at Race Park's safe entrance. The local guest simulator has
keyboard/touch controls and survives modal closure; it never uploads official scores.
