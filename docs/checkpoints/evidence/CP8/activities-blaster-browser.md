# Blaster guest browser verification — 30 October 2026

CUA in-app Chromium, local production preview `http://127.0.0.1:4181/`, default1280×720 and390×844responsive proxy. No injected sessions or app state. This is guest evidence, not authenticated multiplayer, LiveKit, or physical mobile acceptance.

## Observations

- Map destination Blaster co-op naturally walks to47,33; E opens centered solo/co-op panel. Starting solo dismisses panel and projects one Guest inside arena. No duplicated walking avatar.
- Keyboard W600ms moves42,31→42,28; earlier Right aim plus Space produced visible yellow shots. 3HP, Wave1/3,4bots and179s visible. Scene-only autumn perimeter/sign remains inside approved bounds; static map/collision identity unchanged.
- Idle solo run ends all-down after about2s according to typed guest History. Two earlier observations were delayed wall-clock inspections, not31second gameplay runs. Result/history shows0/3waves,0bots; pre-existing race5/5,penalty3:5,pingpong0:1 records remain.
- Menu/detail remains centered and does not pause the simulation. Explicit Return restores47,33; Main lagi starts a fresh run.
- A discovered result HUD countdown continued after completion. Focused regression reproduced it; fixed build freezes at result.finishedAt. Browser observed178s after natural all-down and unchanged on later inspection.
- Initial desktop HUD obscured much of the arena; fixed with Blaster-only compact layout. Final desktop HUD measured320×110px with44px buttons; mobile proxy width360px fits390px viewport, active move/aim pads72px tall, without dock overlap. Final screenshots: [desktop](activities-blaster-desktop.png), [mobile proxy](activities-blaster-mobile-proxy.png).

## Limitations

390×844 is a responsive desktop proxy; actual simultaneous physical two-thumb inputs are not claimed. Independent pointer ownership/lostcapture/modal/hidden behavior is covered by automated production input-router tests. Online4human/spectator/history verification belongs to actual isolated WS+DB tests, not this browser run. OnlyCP1 is an official PASS.

## Post-review build smoke

Final production entry index-Bo1CryzD.js, real CUA heldW + mouseclick600ms: avatar42,31→42,28, enemy count4→3,3HP retained. [Combined input screenshot](activities-blaster-pointer.png). Exact event-order preservation on pointerup/implicitcaptureloss/unrelatedIDs is established by93focused production-controller/router tests; browser smoke does not substitute that deterministic coverage.
