# T11 fix round 1 independent rereview

Scope: original Important R1 (online display/shot-time mismatch) and regressions introduced by its fix. Reviewed `task-11-fix1.diff`, including all five new files, the complete appended fix report with executed results, and the appended binding timing/display ruling. Original reviewed staged tree remains identified as `56f0a2d714413f2dc4c76b4e16c56c3c80fe6f4c`; this review concerns the supplied unstaged fix package. No Git operation or independent tree-state mutation/check was performed.

## Verdicts

**R1: addressed.** The production online display now advances disclosed current targets to the approved presentation time, and the revised deterministic regression covers the original 200 ms RTT / click 55 ms after delivery case through actual production view geometry and pointer handling, followed by real WebSocket shot processing.

**Spec readiness for this fix: pass.** The fix follows the approved R1 display/timing model without moving hit, score, cooldown, expiry or rewind authority to the browser. The prior Minor R2 (`duck.shot_resolved` producer) remains explicitly unimplemented and ledgered to T16 by the coordinator. This is not a claim that R2 or all future-task/global requirements are complete.

**Code readiness: pass for the scoped R1 fix; suitable for coordinator acceptance.** No new Critical, Important, or Minor finding identified in this fix package. The original review's R1 blocker is resolved. No passing test suites were rerun.

**CP6 acceptance remains separate and pending.** Guest CUA supports shared guest-component preservation. Authenticated online browser behavior, physical RTT/asymmetry/jitter, device input and rendering performance are not established by the deterministic bridge.

## Why the original defect is resolved

The old view used the last received target coordinates unchanged. The new view calls `presentedTargets` for online snapshots and uses the current frame time from the shared `OnlineDuckController`. `DuckGame` actually instantiates that controller, routes calibration and shots through it, and updates presentation through `requestAnimationFrame`; the tested helper is not a separate implementation disconnected from production.

Server `duck.clock_reply` now returns the selected calibration values after the same admitted `Observe` path. The new `clockEstimator.calibration` method is shared with authoritative `rewind`, so the reported offset and RTT are selected together from the same eligible minimum-RTT sample. The server retains its independent finite/monotonic/consistency checks and capped rewind. The client does not send the returned rewind back as an authority field.

For an eligible sample, the controller computes:

`local monotonic time + offsetMs + max(0, rttMs/2 - rewindMs)`.

For the exact original case, snapshot time P, RTT 200 ms, delivery P+100, and click 55 ms later produce presentation time P+155. The actual view moves the fast target 46.5 world units before the pointer callback chooses its center. The real server then ingests the shot at P+255 and selects P+155 through 100 ms rewind. The displayed aim and server collision instant therefore align under the approved symmetric-path model.

The 320 ms profile similarly advances presentation by the extra 10 ms needed when half-RTT exceeds the 150 ms rewind cap. It does not raise that cap. The +12 ms path-jitter profiles retain a bounded estimation difference rather than claiming exact knowledge of asymmetric or variable paths; the shipped fast-target radius accommodates the tested difference and the tests assert the actual resulting server hit.

## Regression evidence checked against the code

The revised Go integration test no longer reads coordinates at `received - expectedRewind` to manufacture the aim. It finds a suitable target using a public snapshot, performs real WebSocket challenge/reply exchanges, captures the actual calibration ACK, then obtains a public snapshot at publication time P. Only public snapshot/calibration data, target ID and deterministic delivery/click timings go to the Node bridge.

The bridge imports the production `OnlineDuckController` and `DuckView`. It feeds the measured ACK through `controller.calibrate`, accepts the delivered snapshot, calculates a frame, traverses the actual React SVG target transform, and invokes that SVG's actual `onPointerDown` callback. That callback calls production `controller.shoot`; the resulting payload is returned to Go and sent through the real WebSocket handler. No private engine coordinates are available to this view/aim bridge.

The package and reported executed output substantiate:

- 50/100/200/320 ms RTT, each with 0/+12 ms path jitter, and an additional +12 ms measured challenge sample. The conservative selected sample is preserved.
- Exact 200/0 case: `renderAt-P=155.0ms`, rewind 100 ms, authoritative fast-target hit worth 200 points.
- 320/0 case: `renderAt-P=225.0ms`, rewind still 150 ms, authoritative 200-point hit.
- Outside-lane aim produced through the same SVG pointer callback, submitted after the receipt-time cooldown, is a real server miss. Durable history asserts one hit, two shots and the same authoritative score.
- The actual first-shot ACK is fed back to the controller; the acknowledged target disappears and cannot be redisplayed by an older snapshot in the same generation.
- Pure TypeScript tests distinguish known target expiry within the prediction horizon from stale-snapshot hiding beyond 500 ms, calibration expiry/reset, and countdown/final phase hiding.
- Go tests verify selected offset/RTT pairing, sample expiry and epoch reset; authoritative historical collision still cannot rescore dead/expired IDs.

The integration test's snapshot comes through the snapshot command ACK rather than relying on the periodic event stream. This is still the same public snapshot representation passed to production controller/view logic and closes R1's hidden-coordinate targeting gap. It does not establish authenticated browser rendering or transport scheduling on a physical network; those claims are correctly withheld.

## Regression and authority checks

- Only normalized current-target velocity and current-target expiry were added to public target data. No seed, future spawn schedule or undisclosed target list was added. These values describe targets already visible in the authoritative snapshot.
- Prediction is limited to 500 ms from `snapshot.serverTimeMs`. Negative-age or over-horizon snapshots produce no online targets and a degraded message. Expired targets are hidden, and countdown/finished snapshots do not generate target geometry.
- Controller receive filters world/match, ignores older snapshot times, and prevents an already finished controller from accepting a nonfinished snapshot. Acknowledged dead IDs filter later snapshots within the controller generation.
- Calibration and shot-result continuations use a generation check. World resnapshot resets the controller; the server's existing epoch/disconnect/end resets remain in place. The shared selector retains 15-second sample eligibility. No client-chosen collision time or score is introduced.
- The uncalibrated/expired frontend path explicitly reports approximate timing; it does not stall otherwise legal input or claim measured compensation. Server selection remains independent and falls back to zero rewind when it has no eligible sample.
- The canonical scoring rules/checksum, result model/hash shape, server lifecycle and receipt-time cooldown are unchanged by the fix. Existing admission/world/participant fencing remains around both calibration and shot commands.
- Guest target generation, collision, scoring and session lifetime remain engine-driven. Added public-target fields are optional for the shared TypeScript type; the view uses prediction only when online. The guest timer path retains its previous behavior. Root-reported guest CUA separately exercised gold/fast scoring, first input after remount, keyboard input, natural finish and post-finish lock.
- The Node process is integration-test tooling only. The Go application remains the application backend. The bridge has bounded process timeouts, bounded captured output and temporary output cleanup, and requires existing Node/npm tooling rather than downloading a browser.

## Executed-check accounting and acceptance limits

Read the fix report's final affected verification, without rerunning it: affected Duck application integration under `-race` PASS; Duck engine `-race` PASS; targeted application/Duck vet PASS; five targeted Duck TypeScript tests PASS; typecheck/build PASS; diff whitespace check PASS. The intermediate harness pass of 52 TypeScript tests is correctly distinguished from a fresh whole npm run. The original whole-suite evidence remains from the original reviewed candidate; this round adds affected-path evidence.

The report records a genuine initial view-geometry behavioral failure for the exact R1 case. It also records an intermediate 320 ms outside-click fixture failure caused by exceeding the intended 500 ms prediction horizon; changing the test's cooldown wait to the legal 250 ms interval preserves the product bound rather than weakening it.

Root guest CUA is evidence of shared-component guest preservation, not online timing acceptance. The earlier CLI guest E2E remains FAILED with corrected source NOT_RERUN. No new browser invocation was performed by the implementer in this round or by this reviewer. Authenticated online/credential acceptance remains user-deferred, and physical network/device/performance acceptance remains unexecuted.

R2 remains the explicit T16 protocol obligation. No event-production fix was included or inferred here. Existing unrelated T16 presence/race-assistance obligations are outside this rereview and are not claimed resolved.

Only the requested rereview report was written. No source, staging/index, commit, process, browser, remote operation, subagent, or test-suite execution occurred during this rereview.
