# Free football and penalty — browser evidence

2026-10-28, production localhost4181, local guest, Chromium IAB. Desktop1440×900 and mobile390×844 viewport emulation; no physical-device or authenticated multiplayer claim.

- Map→Lapangan Penalti walks to28,46. Free ball appears at the avatar's feet with contextual Kick/E and small Free ball HUD.
- Clicking Kick visibly sends the ball toward the goal. HeldD400ms still walks28,46→30,46; nearby Kick HUD disappears when out of range. Movement is not captured by free football.
- Menu→Duel penalti→Mulai practice vs BOT closes the panel and replaces free-ball controls with four-zone penalty HUD. Kicker/keeper and match ball render in the world; no simultaneous casual Kick action is exposed.
- Ended the unfinished practice through its explicit exit: returned28,52. HeldW1300ms walks to28,46; the reset free ball and Kick action are available again.
- Mobile Kick tap moves the ball. HUD bottom731.5, docktop743.5:12px clearance, HUD height118px within390×844.
- Opened actual device History afterward: exactly the prior completed penalty3:5 and race5/5 entries remain. Free kicks and unfinished practice added no row. No synthetic history/storage injection was used.
- Final build index-CaflXEpR.js retest: dock reads E Kick (single key hint); heldE650ms kicked and left zero open dialogs. Context copy and held-input behavior are also covered by the shell regression.

Automated owner/WS/DB evidence covers authority, simultaneous reservations, late spectator privacy and delayed result storage. These browser checks do not replace those tests or the deferred authenticated-browser/LiveKit acceptance gates.
