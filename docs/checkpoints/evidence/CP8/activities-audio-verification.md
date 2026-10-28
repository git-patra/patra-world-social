# Task 2 — shared lightweight game SFX

Implemented and frozen for Stage 1 review. No commit created by this task.

## Changes and wiring

- New `apps/web/src/audio/service.ts`: one lazy AudioContext after a trusted gesture; short synthesized UI/paddle/kick/goal/save/countdown/finish/grab/ride/fire/hit/cast/catch effects, no media bank or dependencies. Eight simultaneous voices, oldest-voice eviction, oscillator/gain disconnect on completion/stop, bounded 512 event IDs /128 revision scopes. Simple distance attenuation and public `emit({id,kind,at,worldPosition?})`, `setListenerPosition`, `stopScope`, `dispose` APIs; current legacy modal sounds are unpositioned.
- New `apps/web/src/audio/events.ts`: `mountAudio()` owns gesture, preference/auth, attachment/network and visibility listeners independently of React panels. Auth generations mute immediately until that generation's preferences are ready. Mute/hidden/account changes fence asynchronous resume; no sound queue or replay. Initial/reconnect/visibility snapshots establish baselines. GameAudio observes countdown/finish and authoritative penalty reveal only; snapshots never infer a Duck hit.
- `preferences.ts`: `ready(generation)` prevents previous account's audio preference from enabling a new generation.
- Persistent local race/Duck/penalty sessions produce sounds while retaining their original engine/completion lifecycle; clear releases that mode's voices. Online Duck controller uses authoritative ACK results and generation/reset/freshness fencing.
- Duck adapter now queues exactly one `duck.shot_resolved` per accepted hit or miss, zero per rejected input. Reliable shot events survive coalesced state snapshots. Payload includes world/match, monotonic shot count and acceptance timestamp. ACK/event dedup shares world/match/shot count, robust to input sequence reset after epoch rotation. Delayed (>1s) resolutions are consumed silently.
- UI agent wired `mountAudio()` once before `mountOnline()` in app.js and cleanup on nonpersisted pagehide. Root added audio suite to existing runner/typecheck. This task did not edit app.js, controller.ts, runner or tsconfig.

LiveKit/media tracks, existing game rules/history/podium, and existing visual/text feedback are unchanged.

## RED / GREEN evidence

- Initial eight audio behavioral tests failed against a stub (missing output/lifecycle); GREEN after implementation.
- New Duck producer unit regression failed with **0 accepted-shot events, expected2**; GREEN after producer fix.
- Additional regressions failed before fixes: pending resume survived mute/hidden cancellation; resync baseline replayed an old Duck resolution; late ACK played obsolete SFX; normal sparse penalty reveal at3050ms was incorrectly suppressed by a time-gap heuristic.
- Final audio suite has13 passing tests, including20 open/stop cycles, eight-voice cleanup, muted-before-gesture, suspended/hidden drops, auth generation, public stale-event rejection, sparse penalty cadence, initial/resync highwater, and the actual OnlineDuckController ACK/event ordering/reset/late-response path.
- Reviewer P2 sparse penalty cadence is fixed: silence duration is no longer treated as resync; explicit visibility/resync lifecycle resets baselines.

## Final verification

- `npm test`: PASS,4 JS +123 TypeScript tests (including13 audio).
- `npm run typecheck`: PASS.
- `npm run build`: PASS; final shared application chunk434.96 kB raw /139.27 kB gzip, no audio bank.
- Pinned Go1.26.8 with GOTOOLCHAIN=local and project GOCACHE: `go test ./internal/games/duck ./internal/realtime ./internal/lobby -count=1`: PASS.
- Actual websocket/DB `TestDuckActualTransportMeasuredProfilesAndAuthority` with integration tag: PASS across eight RTT50/100/200/320ms + jitter profiles. Test compares real hit/miss event identity and acceptance timestamps to ACKs and checks rejected commands add no resolution.
- Initial local DB/httptest runs were denied by sandbox loopback restrictions; authorized escalated reruns passed. No secrets printed.

## Remaining checks / limits

Root owns final CUA browser acceptance and real audible playback/device checks. This task did not claim physical-device audio, retained-heap measurements, or perceptual loudness validation. The deterministic20-cycle test covers oscillator cleanup; browser listener/heap acceptance is separate. New activities must use the shared service and supply listener/world positions when spatial attenuation is needed. No music, sampled assets, per-frame audio or LiveKit changes were added.
