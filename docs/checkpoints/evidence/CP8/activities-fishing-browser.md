# Fishing browser verification — 30 October 2026

Production preview `http://127.0.0.1:4181/`, Task9 working tree based on `4aba8fc`. CUA in-app browser, real UI input only; no injected state or authentication.

- Map → Tepi Sungai closes and walks to (53,30). Compact Fishing Ready/Cast HUD appears at river/bridge; four spot labels are visible in the world.
- Desktop Cast → Waiting → observed Bite message → Reel produced `Daun autumn!`; Reel disabled during transient result. Result expired automatically to Ready.
- Cast followed by held A immediately returned to Ready, cancelling the cast.
- Repeated at 390×844 responsive browser size: Cast/Bite/Reel produced `Ikan senja!`; HUD buttons remained readable above dock. Screenshot `activities-fishing-mobile-proxy.png`; desktop screenshot `activities-fishing-desktop.png`. Viewport reset afterwards.
- History still contains exactly the three previous guest records: pingpong0:1, penalty3:5, race5/5. No fishing row.

Physical touch/mobile, authenticated browser/spectator, audible output, and LiveKit acceptance remain deferred. Automated owner/transport/DB/client tests provide separate evidence; they do not substitute for those gates. Final rebuilt version verified Cast/Bite/Reel yielding `Sepatu nyasar!`; pressing E during Catch kept the world HUD visible and opened no River dialog.

Final automated gate: root initially reproduced one floating-point timing fixture failure (205/206). Focused diagnosis showed actual nearbyFootball state correct; test-only wait crosses100ms at101ms. Focused25app-shell and final4JS+206bundled PASS; no product behavior changed. Full Go/typecheck/lint/build and seeded actual DB evidence are in activities-task9.md.
