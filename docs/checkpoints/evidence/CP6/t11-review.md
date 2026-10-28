# T11 independent review

Reviewed candidate: BASE/HEAD `530535d670e45abb78a8e0a3470d5677ca6d7f7e`, packaged staged tree `56f0a2d714413f2dc4c76b4e16c56c3c80fe6f4c`, 44 files. No T11 commit existed at review. This is a scoped T11 review, not T16 branch acceptance.

## Verdicts

**Spec compliance: changes required.** The authoritative engine, durable metadata/result path, countdown, fixed simulation, local practice, and lifecycle boundaries substantially implement the brief. The online display does not align the target actually aimed at with the server-selected evaluation time, and the specified `duck.shot_resolved` event has no producer. Neither the approved timing rationale nor the passing synthetic tests resolves those discrepancies.

**Code quality/readiness: changes required before T11 code acceptance.** One Important correctness finding and one Minor protocol finding are below. No Critical finding identified. Preserve the approved server-selected rewind contract when fixing the presentation defect; do not let the client select arbitrary history or increase the 150 ms bound.

**CP6/user acceptance: pending independently of the code verdict.** Actual guest CUA evidence supports the operations it observed. Authenticated browser acceptance is user-deferred; physical network/device/performance/playtest acceptance was not executed. Resolving these code findings would not by itself make CP6 pass.

## Findings

### Important — R1: align the displayed target with the server's shot evaluation time

Location: `apps/web/src/features/duck/Duck.tsx:11`, `:20`, `:23`; related server selection at `apps/server/internal/games/duck/rewind.go:83` and `apps/server/internal/games/duck/engine.go:163`.

`DuckView` draws `s.targets` directly at the coordinates from the last received snapshot. The online timer updates the clock label but never advances the target positions; there is no interpolation/prediction or other display-time alignment. Shooting sends the pointer coordinates and `performance.now()`. The server correctly applies the approved receive-time-minus-measured-RTT/2 selection. Consequently, the position the user sees can be materially older than the position against which that aim is tested.

Concrete trigger using the shipped constants: let an ordinary snapshot describe a live fast target at server time P. With symmetric measured 200 ms RTT, that snapshot arrives at P+100 ms. Click the center of that still-visible target 55 ms later, before the next snapshot arrives. The shot is ingested at P+255 ms; the approved 100 ms rewind selects P+155 ms. A fast target moves 300 world units/second, so its authoritative center is 46.5 units away from the rendered center, beyond the 45-unit radius. A center click therefore misses despite valid calibration, legal cooldown, no timestamp fabrication, and an unexpired target. The existing publication schedule permits this interval, and positive jitter can widen it. This is a deterministic mismatch, not merely an unexecuted physical-network test.

The transport test does not cover this path: `apps/server/internal/application/duck_integration_test.go:141` sets the synthetic clock to `received - expectedRewind`, reads the engine directly, and at `:150` chooses that freshly read target position as the aim. It proves authoritative historical collision and scoring, but supplies coordinates already aligned to the desired rewind instant instead of the coordinates a delayed browser snapshot actually renders.

Required fix: make the online presentation of currently visible targets follow a defined server-time model consistent with the approved shot time, using bounded presentation prediction/current-target motion data as appropriate. Keep future spawn seeds/private schedules private, retain server authority, receipt-time cooldown and the 150 ms cap, and preserve guest semantics. Do not fix this by trusting a client-selected historical time. Smooth rendering and time alignment must be implemented together rather than simply rendering the same old coordinates more often.

Targeted verification: drive the actual online controller/view from delivered public snapshots with measured challenge/reply timing and deterministic 50/100/200 ms RTT plus jitter. Aim through the displayed target geometry/real shoot callback, not a hidden authoritative-coordinate read. Include the 200 ms/+55 ms center-click example above, a miss outside the displayed hitbox, expired/dead target rejection, and the cap. Keep this synthetic frontend/transport regression separate from deferred authenticated browser and physical-network acceptance.

### Minor — R2: emit and forward the specified shot-resolution event

Location: `apps/server/internal/games/duck/adapter.go:96`–`:104`; integration switch in `apps/server/internal/application/lobby.go`; forwarding at `apps/web/src/net/socket.ts:76`.

The binding protocol maps `duck.shoot` to `duck.shot_resolved` and `duck.snapshot`. The implementation only returns `ShotResult` to the generic `command.ack` path. Adapter events are snapshots/finished events; there is no producer of `duck.shot_resolved`. The new socket branch advertises forwarding it, but the regression injects only race/Duck snapshot and finished events. It therefore does not establish actual shot-resolution event forwarding. `ShotResult` also lacks `worldId`, which the new forwarding branch requires, so merely publishing that struct would still get dropped by the browser.

Impact: the present shooter can read its command acknowledgement and the HUD eventually receives aggregate snapshots, so this is not a loss of authoritative score or a Critical gameplay outage. It is a concrete protocol/spec discrepancy and a missing integration assertion; the named event path must not be reported as implemented based only on an accepted event name.

Required fix: publish the specified resolution event for each accepted shot, with match/world identity and input sequence plus authoritative resolution fields. Preserve the acknowledgement if needed by the current request API. Route the event to the intended admitted recipient(s), retain world/socket fencing, and ensure normal snapshot coalescing cannot replace a queued shot resolution. Avoid double-applying the same resolution when both ACK and event are present.

Targeted verification: an actual `duck.shoot` routed through the application/transport must yield exactly one correctly scoped `duck.shot_resolved` event; exercise a hit and miss, no success event for a rejected shot, real `WorldSocket.onmessage` delivery, and replaced-socket/wrong-world rejection. Assert the payload has the world field the forwarding guard uses. Do not substitute manually injecting the desired event for proof that the server emits it.

## Strengths and compliance evidence

- Go remains the application backend; the new TypeScript engine is explicitly local guest practice. No score-upload API is introduced. The existing admitted-seat and membership boundaries are reused instead of accepting actor/world identity from aim payloads.
- Duck countdown is three seconds in all four cabinet registrations. Actual `StartsAt` is assigned at durable activation, and the engine deadline is that start plus 60,000 ms. The engine rejects pre-start and deadline shots. Guest countdown and absolute session deadlines match that model.
- `RegisterVersioned` changes only the rule identity/version/checksum before factory/persistence. Registration rejects invalid metadata and existing registration/lobbies. Duck's canonical asset is byte-parity checked with the guest asset; the shared result model's serialized shape and other games' rule assets are unchanged.
- Server spawn, target trajectory, radius, points, score, hit count, and shot count are authoritative. Accepted aim is bounded/finite; receipt-time cooldown is 250 ms. Duplicate sequences are rejected; overlap chooses the highest target ID, matching the SVG ordering. Historical candidates also require a current non-dead target ID, preventing a second award through rewind.
- Default rules implement 100/200/500 points, 700 ms spawn interval and 2,500 ms lifetime. These constants naturally keep active targets below eight. Fixed stepping is 30 Hz, publication every second frame, catch-up is capped at eight simulation steps and history at eight snapshots. Skipping work does not extend the absolute deadline.
- The estimator uses unpredictable single-use, expiring challenges and server-observed elapsed time, with four recent samples and 15-second expiry. It chooses the conservative recent minimum RTT and caps rewind at 150 ms. Client timestamps validate monotonicity/consistency rather than select history. Uncalibrated/stale samples use zero rewind. Epoch/disconnect/end reset paths exist, and strict mapped future-time rejection is present. This supports bounded lag compensation, not a perfect anti-cheat claim.
- Metadata-only disconnect installation does not advance Duck. Natural completion wins an equal grace deadline because only an earlier disconnect cutoff changes the outcome; intentional early leave is incomplete. The actual owner tests hold the owner and install original notification timestamps. The engine is finalized before generic all-gone abandonment; the PostgreSQL integration asserts participant history/projection for grace expiry and intentional leave as well as natural completion. Frozen actors remain in results after live claims release. Duck assistance stays false.
- Projector registration precedes recovery. The Duck projector validates rules/roster/details and writes score/hits/shots/complete transactionally through T8. Completed storage status is correctly distinguished from participant `incomplete` eligibility. The UI labels incomplete sessions and zero-shot accuracy truthfully.
- Actual socket-handler coverage for race and Duck snapshots/finished events includes old-socket and wrong-world rejection. Supporting inspection confirms the real race snapshot has `worldId`, so the newly added guard is compatible with its producer.
- Guest engine/session state survives modal lifetime. Shared fixtures check spawn/collision parity, local tests check cooldown/dead targets/finish, and actual CUA separately proves first shot after remount. Full page persistence remains T14. Four cabinets, local labels, autumn/Halloween pond art, and explicit pending podium/champion copy are present. T12 owns podium/PB/champion projection, so their absence is not treated as a T11 defect.

## Verification claims and their limits

Read the complete implementation report, including its final global supplement. Reported executed results are full Go integration PASS, whole-Go integration `-race` PASS, `go vet ./...` PASS, server build PASS, npm tests 53 PASS, typecheck/lint/build PASS, and diff whitespace check PASS. I did not rerun passing suites or a race detector.

The concrete assertions in the packaged tests support the authority, scoring, direct historical hit, metadata, lifecycle, scheduler, and strict-payload statements above. The separate application scheduler test avoids the competing publisher consumer and asserts 30 frames/15 events. The full application tests additionally cover real isolated HTTP/WebSocket/PostgreSQL routing and durable projection. Their profile clocks are deterministic fixtures, not real-network measurements. Different seeded matches correctly compare each server shot score to its own final durable score rather than require unrelated seeds to produce identical totals.

The CLI guest E2E executed and **FAILED** its final whitespace-sensitive frozen-metrics assertion. Its source was corrected and **NOT_RERUN**. Earlier missing pinned browser and sandbox launch failures are retained as failures, not passes. This is distinct from the root's native CUA evidence.

Read `docs/checkpoints/evidence/CP6/t11-guest-preview.md`: three actual guest sessions establish the observed countdown, 100/200 scoring, miss/keyboard input, modal continuation, first pointer shot after remount, finish lock, return/reset, final copy and zero-shot 0% result. The first session's post-expiry Space did not prove remount input, and the evidence correctly uses the second session for that assertion. The third session independently proves zero shots without clock injection. No authenticated browser result is inferred from guest CUA.

## Cannot verify / acceptance still pending

- Authenticated browser account/invite/spectator/reconnect/history UX is user-deferred; no credentials, injected sessions, or alternate browser workflow were used in this review.
- Physical network RTT/jitter, asymmetric routes, mobile/touch devices, measured 60/30 fps rendering, and human tuning remain unexecuted. The renderer timing defect above is a code finding in addition to those acceptance gaps.
- API 8080 was still the accepted T10 process during reported browser checks. Isolated T11 application integration is valid backend evidence but does not imply the live browser used the new online server.
- T12 leaderboard/champion work and T14 full-page guest persistence remain separate tasks. Generic O1/coalesced resume-new-disconnect and personally finished race assistance remain ledgered T16 issues, not silently resolved by T11 tests.
- Source repository untouched/no remote/deploy/provision/purchase are reported constraints. This review did not inspect other repositories or external state to independently certify those operations; no such actions were taken by this reviewer.

## Review scope and checked integration seams

Read the full brief, review context, implementation report/supplement, and the packaged diff in bounded chunks. Recovered only a tool-truncated frontend effect line from the package; did not rerun Git or perform a broad branch review.

Supporting reads were limited to named integration risks:

1. `lobby/presence.go` departure/advance ordering and `lobby/start.go` activation: checked whether Duck incomplete results could be discarded by all-gone abandonment or lose original disconnect times during activation.
2. `lobby/game.go` `ReadEngine`/`InputEngine`, `realtime/delivery.go` admitted-time checks: checked world-bound spectator reads, participant/connection gating, owner receipt time and revocation/expiry authority.
3. `history/service.go` result encoding/binding: checked frozen Version/Actor matching and the effect of adding `incomplete` without changing old result hashes.
4. `lobby/service.go` owner loop: checked whether shot-time stepping bypassed the normal advance/publication path; the owner advances before input callbacks, so no separate blocking finding was raised for the ordinary path.
5. Real race public snapshot producer and the socket world-snapshot handler: checked the actual payload/world guard and reconnect resnapshot seam, rather than relying exclusively on injected handler fixtures.
6. A focused reference search for `duck.shot_resolved`: confirmed there is no server producer behind the newly accepted/forwarded event name.

No test/probe execution was needed to establish the two findings. No source, index, HEAD, browser, or process mutation was performed. Only this requested review report was written.
