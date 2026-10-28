# T2 browser verification

Date 2026-10-26; local Vite at http://127.0.0.1:4176/.

- Desktop1440×900 and mobile390×844 (emulation): core art retained, location menu/zoom/minimap available. Game-only screenshots from scope cleanup remain applicable.
- Actual Race Park navigation via Walk to location reached `(30,14)` and opened its pending-activity dialog. Camera followed the character. Evidence: t2-race-district.png.
- Starting route from Race Park to Warkop then toggling Motion off left the avatar at `(30,16)` in transit, not at target; integrated controller unit test verifies equal duration/positions on full routes.
- Dev-only `?fixture=avatars`: two explicitly named DEV fixture avatars plus local avatar; sprite depth overlap and general nameplate lanes verified. Evidence: t2-three-avatars.png. These are test fixtures, not multiplayer clients.
- User cleanup browser smoke: penalty shot resolves, directory and Warkop pending status visible; console errors/warnings empty.
- T2 does not implement networking, game engines, final district detail, or physical-device acceptance.
