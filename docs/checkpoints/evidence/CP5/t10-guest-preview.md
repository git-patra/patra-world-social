# T10 actual guest browser smoke

27 October2026. Root used CUA in existing IAB tab2, http://127.0.0.1:4176/. Frontend T10 worktree; API preview remains accepted T9. Guest only: no account, injected session, official score, client clock mutation, or credential workflow. No CLI Playwright execution.

## Initial capture and correction

Race Park exposed local practice; first run showed countdown3, five participants and bots advancing laps. Native screenshots appeared to cut the right track/controls: `t10-guest-first-view.png` and `t10-guest-controls-clipped.png`. Root initially described this as a layout defect. Read-only DOM measurement then disproved that inference: CSS viewport/root client/visualViewport widths were1280, scale1, and the dialog was x215,width850,right1065, fully inside that viewport. The actual image bytes are JPEG938×720 despite those two historical .png filenames. The IAB capture is narrower than its page layout. These captures **do not prove CSS horizontal overflow**. The implementer added responsive constraints, but this evidence does not claim a reproduced CSS bug fixed by them.

An intervening frontend/build reload reset the modal and visited places. That attempt is **inconclusive for modal persistence**, not a gameplay reset defect. After the implementer reported a stable build and held frontend/contracts/build changes, root explicitly reloaded the page, reopened Race Park and started a fresh race. The freeze was released after the smoke.

## Stable-page executed observations — pre-remainder-carry build

- UI clearly said SOLO LOKAL / DUNIA PRIVAT, local Guest+4BOT and no ranked upload. Countdown displayed3 and0.00s before movement. This is visible countdown evidence, not a precise frame-by-frame timing assertion.
- Random grid in this run: BOT1, BOT2, BOT3, BOT4, Guest. Rendered roster had five kart labels and0/5 completed laps.
- Read-only DOM: viewport1280, dialog x215,width850,right1065; clientWidth848=scrollWidth848. SVG x276,width728,right1004. No horizontal overflow within the actual CSS viewport. This does not establish938px/mobile viewport acceptance.
- Actual keyboard W held2s moved the rendered Guest SVG from x16432,y5600 to x26687.30,y5600. A held Gas pointer input with D turned/moved it to approximately x37862.31,y7031.69,heading135degrees. This establishes basic gas/steering input response, not a full human5-lap handling or novice-bot tuning assessment. Pointer input on desktop is not physical-touch evidence.
- Bots physically progressed through displayed lap counts and finished at92.63s,93.97s,95.87s,97.87s, with best laps18.33s,18.60s,18.93s,19.33s respectively.
- Closed the panel with Escape while the race was still active after the120.34s observation. Reopened through Explore Race Park: the same bot finish times and Guest progress remained, elapsed135.39s with3s left. No new countdown/reset.
- Finish displayed137.63s, exactly displayed first bot92.63s+45s, GuestDNF and position5/5. Root did not drive a complete lap. Result explicitly says local practice and explains time-tie/start-slot and DNF progress ordering. [Result capture](t10-guest-results.jpg).
- Kembali ke Race Park returned to the local start panel with Mulai race solo ·5lap. [Return capture](t10-guest-return.jpg).
- Captured browser console warnings/errors: empty array.

## Limits and follow-up

This is actual guest UI smoke plus read-only DOM geometry, not online multiplayer acceptance. Human5-lap finish,1–5human playtest, beginner-bot difficulty tuning, physical mobile steering/fps, network latency/jitter, full reload persistence, live spectator/reconnect/history UI remain unverified by this smoke. Server tests are recorded separately by the implementer.

Hero still described balapan as being prepared during this smoke; root asked implementer to update availability copy. Race Park detail already had correct available copy. Existing guest lobby indefinite Memuat ketersediaan text remains the prior MinorT15. The screenshot capture/viewport mismatch limits visual evidence; it is not used to claim a site layout regression or mobile PASS.

## Subsequent implementation refinement

After this smoke, implementer self-review identified missing residual integer-division carry in race heading-vector movement, required by contracts/README. Go and TS had matched but both truncated fractional subunits each tick. The implementation is being updated with remainder carry/shared vectors, plus frozen final clock and hero availability copy. Therefore the exact observed92–98s bot times/137.63s cutoff above belong only to the earlier tested worktree build. A focused final-candidate guest recheck is planned; those values must not be presented as final-build timing evidence.

## Focused final-candidate recheck

Implementer reported stable build JS `index-5pNF16M7.js` / CSS `index-CF46jrA-.css` after remainder carry, final clock freeze and hero copy. Root reloaded the real page and used another frontend/contracts/build freeze. This is the final-candidate UI check, separate from the earlier timing values.

- Hero now says race and penalty are available solo/private; Duck/voice are being prepared. Race Park has the available copy.
- Countdown3,5actors. This run's grid: Guest,BOT4,BOT2,BOT3,BOT1.
- W held2s followed immediately by D650ms produced actual rendered movement/steering: Guest started18432,5600,heading0; later SVG showed35555,14760,heading57degrees. Basic controls still respond under amended physics.
- Panel closed during active race after the14.74s observation. Reopened at124.92s with15s remaining: same race/grid, four bot finish times and Guest progress, no reset.
- BOT4,2,3,1 finished93.97s,95.87s,97.87s,99.77s; GuestDNF at138.97s (displayed first93.97+45). [Final result capture](t10-guest-final-results.jpg).
- Final result observation at08:00:19.751UTC and unchanged AX tree at08:00:32.901UTC: displayed elapsed remains138.97s after13seconds, so the final clock does not continue advancing.
- Console warn/error capture remains empty. Preview tab left at final results and marked deliverable. Freeze released after capture.

All earlier acceptance limits still apply. No assertion that exact bot times are constant across random grids/seeds; reported times belong to this run only. This does not exercise authenticated Go race through the browser (API preview remains T9), human5-lap finishing, physical-touch performance or novice difficulty tuning.

## Independent-review follow-up: confirmed reopen input defect

Reviewer identified that the component input counter resets on remount while the persistent guest engine retains its last accepted sequence. Root reproduced the user-visible effect on the unchanged staged candidate:

1. Start a fresh race without control input. At37.13s, Guest rendered at18432,5600,heading0.
2. Escape then reopen at52.46s. Hold W for2seconds; at54.66s the rendered Guest remains18432,5600,heading0 despite running state. The race/bots continue normally.
3. [Failure capture](t10-guest-reopen-input-failure.jpg) at about91.5s still shows the stationary Guest at the start. Root had not applied more gas between steps2and3.
4. Later W2s at136.46s moves Guest to28423.24,5600 with the same close-button focus. This confirms eventual recovery and argues against focus alone causing the earlier failure. [Recovered capture](t10-guest-reopen-input-recovered.jpg). The replay counter explanation comes from reviewer code inspection, not hidden state injection.

Earlier smoke established visual/time continuity only; it did **not** establish immediate driveability after remount. This new evidence is a real guest defect and must be fixed/retested before T10 code acceptance. New captures were1280px wide; earlier938px capture limitation remains historical, not a proved CSS bug.

## R1 fix actual-browser retest

Implementer reported R1 candidate JS `index-DCicAL2o.js`, with session-owned input sequence and unchanged simulator monotonic rejection. Root explicitly reloaded, started a new guest race, and used a stable frontend window:

- At31.42s before close, idle Guest position18432,5600,heading0; same five-actor grid.
- Escape/reopen at38.84s. The **first** W2seconds followed byD650ms moved/turned Guest to31421.93,8396.94,heading57 by41.69s. No old-counter catch-up wait; race continued with same bot grid/progress.
- [Fixed-control capture](t10-guest-reopen-input-fixed.jpg) taken about52.79s after coast; position is farther along than the immediate41.69s DOM measurement, as expected from elapsed time. Console warnings/errors empty.
- This confirms the reported remount control defect is fixed in actual guest UI. It does not claim online/device playtest or all T10 review findings resolved; R2 backend batch ordering remains a separate review fix. Root released frontend freeze afterward.
