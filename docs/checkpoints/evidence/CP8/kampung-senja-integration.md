# Kampung Senja integration verification

28 October 2026. Scope: user-approved outdoor/fullscreen redesign of `patra-world-social`, based on `b7ae4bd`. Semi-English HUD, no legacy sidebar, existing game and multiplayer interfaces retained. This is scoped implementation evidence, not a new checkpoint PASS.

## Automated checks

| Check | Observed result |
| --- | --- |
| `npm test` | PASS: 4 JS + 99 bundled TypeScript tests = **103**, zero failures |
| `npm run typecheck` | PASS, including world/controller and app-shell regression tests |
| `npm run lint` | PASS (existing JS syntax checks) |
| `npm run build` | PASS, 83 modules, final build 325 ms |
| `go test -count=1 ./...` | PASS with pinned Go1.26.8 |
| `go vet ./...`, server build | PASS |
| Focused Go race suite | PASS: realtime, world, voice and all game packages |
| `go test -count=1 -race -tags=integration -json ./...` | PASS: 20 tested packages, 242 top-level tests / 397 tests and subtests; no failures or skips |
| Shared geometry | Map and voice contract copies byte-identical; canonical checksum valid; 169 art solids match; 2,129 reachable cells include all eight destinations and both tables |

[Backend report](kampung-backend-verification.md) records actual PostgreSQL, WebSocket/game/history, world-isolation/load-smoke and voice-admission lifecycle coverage. Private database settings and raw logs remain in ignored `.cache/`. The dedicated Colima/PostgreSQL runtime required recovery of stale host-agent/SSH forwarding processes; the fresh full suite passed afterward, with disposable schemas removed. No unrelated Docker profiles or source repository were changed.

[Fresh review](kampung-review.md) found three bugs, all repaired and rechecked: dialog cleanup canceled online Map destinations; semantic directional clicks never stopped; large foreground props vanished early near mobile viewport edges. Actual app-event regressions went RED before fixes, then passed. The culling regression checks three zoom levels. Local/online mode restoration, authoritative snapshots while paused, no online reset teleport, explicit-only activity entry, directional release and art/projection parity are covered.

## Actual browser checks

Ran the current app at `http://127.0.0.1:4176/` with the updated Go API on8080. No session injection or credential entry. Initial exploratory sessions during edits are separate from the final frozen-source game run.

| Flow | Observed result |
| --- | --- |
| Fullscreen desktop | Canvas fills viewport; no old CV/sidebar; Map/Friends/Chat/context/Menu and camera controls visible |
| Map route and arrival | Selected Warkop from Map; real avatar walked from32,32 to16,24; arrival kept world visible; E explicitly opened Warkop |
| Continuous movement | Keyboard direction/release and semantic direction activation work; a mobile direction activation stopped at33,32 and remained there on the later observation |
| Text guard | Typed `wasd` into chat; avatar stayed32,32; draft cleared without submission |
| Penalty | Chose first kick; button locked. Closed/reopened panel; final **4–1**, seven kicks, persisted locally |
| Race | Guest + four BOT, countdown, keyboard throttle; closed/reopened panel; BOT finished five laps, Guest DNF at normal137.30s cutoff; five-slot standings and result persisted |
| Duck | 60-second session, one real Space input; completed score0, hits0/1; result persisted |
| History | Final three results present after reload. Two earlier exploratory Penalty/Duck results are also retained, total five. No local data reset was performed |
| Other menu entries | Settings, Play with friends/account gate, Friends/invitations gate, Chat, Podium gate, Warkop mic/table chat and legacy practice mount remain reachable |
| Warkop availability | Truthful LiveKit-unavailable message, two shared table definitions, guest admission controls disabled |
| Responsive layout | Actual390×844 and768×1024 iframe viewports inspected, including top HUD and bottom dock/d-pad; no viewport-edge clipping of controls. Wrappers are layout checks, not physical-device acceptance |
| Browser console | No warning/error entries on the final app check |

Final source was frozen for the three-game completion run. One subsequent CSS-only adjustment separates initial status/welcome/dock at tablet height; final typecheck/lint/build passed and responsive layout was rechecked. No game, storage or input logic changed after completion.

## Evidence images

- [Desktop village](village-desktop.png), [shared world map](world-map.png).
- [Mobile world](mobile-world-390.png), [mobile controls](mobile-dock-390.png), [tablet world](tablet-world-768.png), [tablet controls](tablet-dock-768.png).
- [Penalty result](penalty-result.png), [Race result](race-result.png), [Duck result](duck-result.png), [History after reload](history-after-reload.png).

The prototype on4180 remains a design reference. Production has no simulated multiplayer NPCs, fake social replies or collectible progress; online avatars come from server snapshots. Existing backend game engines and protocols remain in use.

## Acceptance limits and publication

- Authenticated multi-browser/account acceptance remains deferred by the user; backend integration and app-event fixtures do not substitute for it.
- LiveKit Cloud audio, token replay and physical mobile testing remain NOT_RUN. T14 final review, remaining T15 acceptance, T16 load/60-minute soak/backup-restore and beta verification stay open. **CP1 remains the only official PASS.**
- New map identity is `kampung-senja-v1`, checksum `71814ab8a87be0bd7166cbe2c1049f38e6fa774825ca87230fa0b155b0bb0768`; schema v1/subunits/speed unchanged. Historical match hashes were not rewritten.
- User explicitly requested local commits with plan/evidence, then a public personal GitHub repo on `main`. A fresh public snapshot preserves the verified game tree without publishing inherited CV/PDF history. Local feature-branch history remains intact. Two obsolete CV screenshots were removed from the current tree; historical narratives retain that qualification.
- Published on28 October2026 to [git-patra/patra-world-social](https://github.com/git-patra/patra-world-social): PUBLIC visibility, default branch `main`, clean root snapshot `0adf204d59ddfd47eeaaa3ca70d8d1ffc0fbf6f9` verified remotely. Plan/evidence included; inherited CV history was not pushed. See [publication verification](github-publication.md). No deployment was performed.
