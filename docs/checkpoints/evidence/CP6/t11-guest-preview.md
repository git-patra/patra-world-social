# T11 actual guest browser smoke

27 October2026, existing IAB browser3/tab2 at http://127.0.0.1:4176/. Actual native input via CUA and read-only visible SVG geometry. Deliberate reload loaded current candidate; implementer froze frontend during both sessions. Root performed no state/time/credential/session injection or authored E2E execution. API8080 still served acceptedT10; this is local guest frontend evidence, not online Duck acceptance.

## Session1

- Opened Duck Arcade through location navigation. Four private cabinet placeholders, guest practice label and Start60s shown. Station/hero still used upcoming copy; correction requested separately.
- Start visibly displayed `MULAI3`, score0/hits0/shots0/accuracy0. Countdown erroneously displayed spectator copy `Menonton sesi kabinet ini`; minor copy fix requested.
- Screenshot at56s remaining shows actual moving target/pond. Modal needed scrolling to bring pond into viewport; no horizontal overflow claim or physical-device inference.
- Native click on visible target geometry: at27s remaining score100/hits1/shots1/accuracy100%. An intentional click in empty pond then ArrowRight+Space produced score100/hits1/shots3/accuracy33.3% at17s.
- Closed and reopened panel; at6s remaining the same100/1of3 score persisted and timer had continued. Subsequent Space occurred after the session had expired and activated the focused close button. This attempt does not prove remount first-shot behavior; it was repeated with sufficient remaining time in session2, not diagnosed as a game defect.
- Reopened final result: SELESAI,100/1of3/33.3%, local practice/no official upload. Return restored Start. `t11-guest-result.jpg` records this result.

## Session2 — first shot after reopening

- New Start displayed countdown3 and reset0/0of0/0%. At52s remaining a visible target hit gave100/1of1/100%.
- Closed panel, reopened at41s remaining: same100/1of1 and continuing timer.
- After scrolling, the FIRST new pointer shot at28s remaining hit a fast target and immediately gave300/2of2/100%, demonstrating no remount counter blackout.
- ArrowRight then Space while playing at6s remaining gave300/2of3/66.7%; focused close button did not activate during gameplay. Final result retained300/2of3/66.7%.
- A further native click on a remaining visible target after SELESAI left score/shots unchanged. Console warnings/errors empty.
- `t11-guest-reopen.jpg` was captured after natural finish; it shows the final state of the successfully continued session, while AX observations above record the actual live remount/input progression.

## Evidence files and limits

All new captures are native JPEG bytes: `t11-guest-start.jpg`, `t11-guest-result.jpg`, `t11-guest-reopen.jpg`. No synthetic result images.

This proves the observed guest countdown, pointer100/200 scoring, misses, keyboard shooting, continued modal session, immediate first shot after reopening, finished input lock and return/new-session reset. It does not prove every collision/500point/golden fixture boundary, exact fixed simulation cadence, authenticated spectator/multiplayer/history UI, measured real network compensation, physical touch/mobile performance, novice tuning, or CP6PASS. Synthetic Go/TS/PG/WS tests are separate evidence. Podium/champion T12 remains pending; no fake leaderboard was displayed.

Frontend freeze released after these checks. Copy corrections requested; repeat only relevant presentation checks after they land, unless gameplay changes.

## Browser attempt accounting update
Implementer reported a cached-Chromium CLI guest run had already completed when coordinator's stop instruction arrived. It exercised the guest flow but FAILED a whitespace-sensitive assertion at the final result (`innerText` newlines vs concatenated DOM text). Earlier pinned-browser absence and sandbox MachPort failure are retained in the report. Correcting that test source without rerun is NOT_RERUN, not PASS. No further CLI browser runs are authorized in this workflow; root continues browser verification through CUA. Root's separately executed two-session CUA evidence remains scoped and valid; reviewers must not conflate the failed CLI attempt with its actual CUA pass.

## Final presentation retest and zero-shot session

After implementer declared frontend final/frozen, root deliberately reloaded. Hero now lists Duck available; location directory says60seconds; station states four private cabinets/account history/guest practice and explicitly leaves podium/champion pending. Start displayed countdown3 with corrected `Bersiap · sesi dimulai setelah countdown3detik` copy, captured in `t11-guest-countdown-final.jpg`. This is a focused presentation retest, not a repetition of unchanged gameplay.

The resulting third session received no shots and naturally reached SELESAI. Final HUD shows score0, hit0/0, accuracy0%, without NaN or console warnings/errors. `t11-guest-zero-shots.jpg` records the actual zero-shot result. Root did not inject a result or accelerate the clock.

## R1 amended-frontend guest regression (in progress until final check)

After implementer declared the new shared Duck frontend stable, root deliberately reloaded and used the existing guest preview. Countdown3 observed. First target was gold: hit500 at57s remaining. Closed/reopened44s: same500/1of1. FIRST new pointer shot38s hit fast200:700/2of2. ArrowRight+Space at31s produced700/2of3/66.7%. `t11-fix1-guest-live.jpg` records the actual running session. Guest uses its existing50ms engine timer; this check cannot establish the amended online prediction/latency behavior, which requires the delivered-snapshot regression. Natural finish/lock/console follow-up pending at this note.

R1 guest follow-up PASS: same session naturally SELESAI at700/2of3/66.7%; another click on remaining visible target did not change HUD or shots. Console warnings/errors empty. Actual `t11-fix1-guest-final.jpg` records final state. This is the completed focused guest regression on amended shared Duck component, not online timing acceptance.
