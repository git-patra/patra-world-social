# Pingpong browser evidence — Task7

2026-10-28, production preview http://127.0.0.1:4181/, IAB Chromium, guest mode.

- Map destination Meja Pingpong walks from32,32 to entrance; E opens centered lobby; seven-district count remains3/7 rather than adding a district.
- Mulai practice vs BOT closes popup; world table and small HUD show0:0,120s. Serve starts rally; held D400ms and pointer-left400ms accepted by controls. Menu/detail opens while deadline continues; returning preserves0:1 score.
- Natural120s deadline finishes0:1. Return then History shows exactly one new Pingpong row above the unchanged prior penalty3:5 and race5/5 rows. Expanded detail:120seconds, Guest Kalah/BOT Menang,7points/winby2/cap11.
- At actual1440x900 viewport, HUD bottom793 and dock top805:12px gap. Attempted390x844 viewport override did not change this tab dimensions; no new mobile-layout claim is made here. Responsive regression remains final verification work.
- No authenticated browser, LiveKit or physical mobile acceptance claimed. Automated private-world DB/WS checks are separately documented.

Environment recovery: dedicated Colima profile patra-world-social was Broken; normal stop/start failed on stale hostagent. Forced stop/start of this profile preserved disk; existing PostgreSQL container restarted. Verified orphan same-profile SSH mux19584 held55432; it was terminated and existing loopback forward restored using generated dedicated ssh.config. Host PostgreSQL SSL negotiation returned N within3seconds. Default/other profiles and database volume were untouched. Correct Docker context is colima-patra-world-social.
