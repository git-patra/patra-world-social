# Activities stage 1 — UI, minimap, SFX and prerequisites

28 October 2026. User authorized execution of all six stages; this records the first stage only. Plan: [activities adjustment](../../../superpowers/plans/2026-10-28-world-activities-adjustment.md).

## Changes

- Focusable DOM minimap opens centered full Map. Full-map clicks route through the existing local/authoritative destination adapter; no position writes or teleport. Coordinate mapping respects contain-fit letterboxing; closed-area resolver seam is ready for Task3.
- Centered native dialogs, safe-area/internal scroll/VisualViewport sizing, Chat converted to dialog. Closing a popup waits for close cleanup before sending a route. Nested Menu children restore focus to visible Menu; closing parent does not steal focus from child.
- Shared synthesized Web Audio SFX: one context after gesture, maximum8 voices, bounded event IDs, mute/visibility/auth-generation cleanup, initial/resync snapshots silent. Race countdown/finish, penalty reveal, Duck accepted hit/miss hooks are connected to persistent sessions/events. LiveKit mic/mute unchanged.
- Duck server now emits exactly one accepted shot result and preserves reliable shot events during snapshot coalescing. ACK/event share shot identity; delayed/replayed/resync results do not replay sounds.
- Prerequisites: ordered bounded presence mailbox, event-time departure tick, shared/exclusive PostgreSQL startup fence. T14 R1/R2 independently accepted, without changing external acceptance status.

## Review and regressions

Fresh review identified two defects beyond the initial implementations: a queued departure could tick past a later reconnect transition, and nested popup focus could land on body. Both reproduced before fixes. Real race-owner tests cover grace-expired and natural-finish ordering; shell tests cover Settings/History/Quick-play focus. Audio review also reproduced sparse online penalty snapshots dropping a normal reveal; explicit resync/visibility resets replace the timing heuristic.

Actual PostgreSQL regressions cover initial Persist commit-before-ack-error and rollback, startup-before-match lock ordering and cancellation. Actual Duck websocket transport covers eight RTT/jitter profiles and compares accepted event to ACK; rejected shots emit no event.

## Browser evidence

CUA Chromium, guest-only, no credential workflows. Initial live-preview checks were interrupted by HMR; final interaction checks used production preview4181.

- Map and Chat at320×568,390×844,768×1024,844×390,1440×900: all dialog bounds inside viewport. Map center error0px; Chat≤0.008px.
- Clicking corner minimap opens Map with avatar unchanged at32,32. Escape returns focus to minimap.
- Actual W/A/S/D keystrokes in Chat produce `wasd`, avatar stays32,32; Escape returns Chat focus. Draft cleared without submitting.
- After focus repair, Menu→Settings/History/Duck→Escape: child owns focus while open, then visible `menu-button` owns focus. Dialog center error≤0.008px at390×844.
- Responsive dimensions are desktop emulation, not physical mobile acceptance. Physical on-screen keyboard, audio output listening, authenticated browser and LiveKit checks remain NOT_RUN. Audio behavior is covered by context/oscillator lifecycle tests and real transport, not a claim of hardware listening.

Automated command totals and local commit recorded after final coordinator verification. No public deployment; CP1 remains the only official checkpoint PASS.

Final coordinator `npm test`: 4 JS + 126 TS tests passed; typecheck, lint, build and whitespace checks passed. Entry JS 435.29 kB / 139.40 kB gzip; no new runtime dependency. Full Go integration/race evidence and focused post-fix application checks are in [startup verification](activities-startup-verification.md) and [presence verification](activities-presence-verification.md). Fresh review and scoped fix re-review accept all three findings; see [review](activities-stage1-review.md).
