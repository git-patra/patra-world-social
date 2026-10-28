# Free ride — browser evidence

2026-10-28, production localhost4181, local guest, Chromium IAB. Viewports1440×900 and390×844; emulation is not physical-mobile acceptance.

- Minimap→Lintasan Mini walks to45,45. Nearby HUD offers Free ride · Mobil and Naik · E. Mount moves display into the actual car at45,46; no separate walking avatar.
- HeldD650ms moves car to50,46. ContinuedD stops at52,46; W700ms follows right lane to52,40. E dismount returns to safe45,49.
- Walked to39,46; E mounts bicycle. Mobile heldright button500ms moves39,46→42,46. HUD bottom731.5 vs docktop743.5 gives12px clearance; both within390×844.
- While still mounted bicycle, Menu→Balapan→Mulai race solo closes modal and replaces Free ride HUD with race countdown3s and five karts. Ending practice returns45,49; walking north to45,47 offers Naik again.
- Minor UI findings fixed and retested on final build index-B5iKhAvB.js: dock Naik Mobil · E mounts, HUD says Mobil · Cruise, dock changes to Turun · E. Driving to52,40 keeps location Lintasan Mini; dock Turun returns45,49.

Server exclusivity, stale generation/incarnation/sequence, atomic race clearing and no-history guarantees are covered by automated owner/transport/DB tests in Task5 report. This local browser check does not claim authenticated multiplayer, physical multitouch or hardware audio acceptance.
