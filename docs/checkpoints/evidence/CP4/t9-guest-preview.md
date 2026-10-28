# T9 guest-only penalty smoke — initial finding

27 October 2026, CUA IAB1280×720, http://127.0.0.1:4176/. Vite serves in-progress T9 frontend; backend preview remains accepted T8. No credentials or session injection.

Started Main penalti → Mulai practice vs BOT. Observed four-zone goal art, Guest/BOT labels, local-only result notice, alternating keeper/kicker role and countdown. One right-top choice was accepted and locked: controls disabled and Guest/BOT both marked chosen before reveal; resolved kick4 later showed zone2/3 without the automatic flag. Other unchosen turns were marked pilihan otomatis. At six resolved kicks score was1:2, with another active choice timer.

[Live screenshot](t9-guest-duel.png).

Initial defect: closed the modal via close-dialog, immediately reopened Main penalti; it showed the initial Mulai practice vs BOT screen rather than the active duel or its result. This contradicted the displayed continuation promise. [Failing reopen screenshot](t9-guest-reopen-before-fix.png). Sent to implement_t9 during initial implementation for a persistent guest controller/catch-up fix and mount/unmount/reopen regression. Retest pending; this document does not claim the defect is fixed.

Captured console warnings/errors were empty. This smoke does not prove exact millisecond boundaries, authenticated online gameplay, or saved account history. Those use separate deterministic/backend tests and the still-deferred browser acceptance.

## Retest after guest controller persistence fix

The first retest was invalidated by a Vite full-page reload (AX page changed and visited locations reset), so it is inconclusive and not classified as a second bug. After requesting a brief frontend edit hold, started a fresh practice, closed the modal, and reopened via Main penalti. It retained the duel: score1:0, two resolved automatic kicks, keeper role and0.2seconds remaining in the next turn. It did not return to the initial Start screen. [Fixed reopen screenshot](t9-guest-reopen-fixed.png). A further closed-panel-to-results check is pending. No browser clock/state injection or credentials were used.

## Closed-panel completion retest — PASS

On the same stable page/session, closed the active panel again and later reopened it. The duel had completed normally: Guest4:3BOT, Guest Menang/BOT Kalah,10 resolved kicks (five per actor), Practice selesai, and explicit Main lagi vs BOT. All turns during the closed interval were correctly marked automatic. Console warnings/errors remained empty. [Final result screenshot](t9-guest-results.png). The earlier fixed-reopen screenshot was taken after the AX observation and therefore already shows a later2:2 reveal; both belong to the same continuing session.

The modal persistence/elapsed catch-up defect is resolved in this executed guest path. Full page reload persistence is T14, not claimed here. No authenticated gameplay, browser hidden-tab/device-clock/reduced-motion automation, or exact millisecond acceptance was executed by this smoke. The authored Playwright scenarios remain NOT_RUN; actual browser work used CUA.

MinorT15: each zone button shows a stray leading0 near its numeric label, visible in screenshot/AX. Cosmetic cleanup remains before final UI acceptance.
