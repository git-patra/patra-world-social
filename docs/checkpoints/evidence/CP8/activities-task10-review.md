# Independent Task10 review — 30 October 2026

Fresh reviewer (gpt-6-astra/high), base d4e678f to frozen uncommitted Task10, full228618-byte review package including new files. No implementation authorship. Task-scoped spec and quality review; whole-branch review still required inTask11.

## Initial verdict: Needs fixes

Important: keyboard handler overwrites shared fire with Space state on every WASD event; mouse-held firing stops when movement keys change. Canvas pointer-up and implicit capture loss globally release controls, clearing held WASD and rejecting repeat until a fresh press. Focused actual router reproduction confirmed mousehold→W yields y:-1/fire:false; pointerrelease→Wrepeat yields y:0/fire:false. Fix must separate mouse/keyboard fire and release only pointer-owned contribution during ordinary release, retaining global neutralization for interruptions.

Minor: format compressed new TypeScript simulation for Go/TS parity maintenance. Existing expected missing-history diagnostic is noisy; carried toTask11 triage.

Strengths verified: human-only capacity/authority, deterministic bounded20Hz engine, shared Go/TS goldens, actual4human+spectator WS/DB persistence/reconnect/hostleave/rematch tests, shared bounded scene/audio, reset-to-cap worst-entity benchmark. Reviewed logs show216TS and fullGo integrationPASS; timer213/214RED→214/214GREEN. Named cross-call check dismissed nil pre-start snapshot risk because lobby exposes engine only after Start; areaACK/durability ordering remains.

## Evidence limitations and coordinator rulings

- Original inherited feature RED evidence is unavailable. Fresh GREEN and HUD RED→GREEN are recorded honestly; no original RED reconstructed.
- Procedural perimeter/art and timestamp-derived hit flashes satisfy the approved existing-canvas, bounded-entity rendering requirements. No generated raster or separate accumulating particle pool is needed; map bounds were established inTask3/4. This is an implementation choice preserving the approved layout, not a redesign.
- Authenticated browser, physical dual-touch and LiveKit remain deferred. OnlyCP1officialPASS.

## Fix round1

Approved by the independent reviewer. Important input issue ADDRESSED; Minor formatting ADDRESSED; no new Critical/Important issue. Mouse/keyboard/touch fire combine independently, pointer IDs protect normal/unrelated capture termination, global interruptions still neutralize all sources. Focused real production router/controller regressions:4RED failures→93/93GREEN,zero skipped;typecheck/buildPASS. Existing history diagnostic Minor carried toTask11.
