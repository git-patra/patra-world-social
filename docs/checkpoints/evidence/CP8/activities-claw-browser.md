# Claw browser verification — 30 October 2026

Local production preview `http://127.0.0.1:4181/`, current Task8 working tree based on `0289a10`; CUA in-app Chromium, desktop viewport. The old preview process was absent; production was rebuilt before serving. No authentication or app-state injection was used.

- Village Map → Capit Boneka closed the dialog and walked from (32,32) to (41,24), where the compact cabinet HUD appeared.
- Enter showed operator Guest and 20s lease, direction buttons, Drop and explicit Exit above the dock; cabinet remains in autumn world.
- Drop showed `Turun…`; direction and Drop buttons disabled during animation.
- Natural lease expiry returned to Ready / Capit, without a popup or permanent lock.
- History after playing retained the three previous guest records (race 5/5, penalty 3:5, pingpong 0:1); no claw result appeared.

Independent review found vertical aiming and schema defects; both were fixed with RED→GREEN regressions and the scoped re-review approved them. Post-fix browser verified W moves the grip upward, A moves horizontally, Drop animates with disabled controls, and Exit returns to Ready. The fixed HUD was checked at 390×844 responsive browser size and stayed above the dock; viewport override was then reset. Screenshots: `activities-claw-desktop.png`, `activities-claw-mobile-proxy.png`. These desktop guest observations do not establish authenticated spectator/browser, physical mobile/touch, or audible playback acceptance. Automated actual-DB test separately verifies 20 cycles, spectator equality, duplicate rejection and zero persisted result/history rows.

Fresh root verification: frontend before fix 4 JS +185 bundled, after fix 4 JS +193 bundled; typecheck/lint/build PASS. Full pinned Go `test ./... -count=1` PASS; focused Claw world/realtime PASS; actual PostgreSQL no-history test PASS (20 cycles, package 1.442s). Dedicated Colima/profile/container were started preserving the existing volume.
