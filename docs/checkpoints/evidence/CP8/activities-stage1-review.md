# Stage 1 scoped review — Tasks 0–2

Reviewed 2026-10-28 against `task-0-brief.md`, `task-1-brief.md`, `task-2-brief.md`, the matching implementation reports, and `docs/checkpoints/evidence/CP8/activities-baseline.md`. Initial review input was `stage1-review.diff` relative to `3079f05`; followed the concrete cross-file risks below. Audio worker's subsequent freshness/cadence changes were reviewed directly in the named production files and final Task 2 report. No production files, index or commits changed by this reviewer; no suites rerun.

## Verdict

**Changes requested for the ordered presence queue and menu-child dialog focus restoration.** Startup PostgreSQL fencing passes the scoped review. The final audio implementation has no additional confirmed important code findings, subject to the acceptance limits below. This is not Stage 1 complete acceptance while R1/R3 remain unresolved.

The intended dynamic-zone seam is present, but it is not an authoritative closure owner or path-collision implementation. Those are later tasks, not missing Task 1 product features. Existing online movement, game rules, durable history, and voice authority are preserved by the reviewed frontend changes.

## Open finding

### R1 — P1: A queued departure advances beyond later queued reconnect transitions

**Location:** `apps/server/internal/lobby/presence.go:49–54`, interacting with `apps/server/internal/lobby/presence.go:117–119` and `apps/server/internal/games/race/engine.go:303–322`.

The first pass installs only the earliest disconnect for each player because `recordDisconnect` skips players already marked disconnected. The second pass clears that metadata on resume and installs later disconnects as it reaches them. However, a different player's intervening leave calls `departAt`, which immediately calls `engine.Tick(s.clock.Now())`. That advances to ingestion time before the rest of the ordered queue has been applied.

Concrete sequence while the owner is stalled: A disconnect at t0, A resume at t1, B leave at t1.5, A disconnect at t2; drain at t20. The first pass installs A's t0 only; A's resume clears it. B's leave then advances the race through t20 with A incorrectly connected. A's t2 disconnect arrives after the simulation has already advanced, potentially after a terminal result. The new queue must not let an intermediate departure advance past pending transitions. Use the notification timestamp for intermediate catch-up (or defer the ingestion-time tick until the complete batch has been applied), and add a real race-owner regression comparing delayed draining with sequential ingestion. Root was notified before prerequisite commit.

Evidence is a direct trace through the production queue and race adapter/engine; this reviewer did not rerun tests. The existing new reconnect test exercises one player's disconnect/resume/disconnect without an intervening departure, so it cannot detect this case.

### R3 — P2: Closing a menu-child dialog loses keyboard focus

**Location:** `apps/web/src/ui/dialogs.ts:2–5`, `apps/web/src/app.js:96` (Menu close-on-button handling), and existing child popup openers such as `apps/web/src/features/settings/SettingsPage.tsx:31`.

Root confirmed in the browser at width 390 that Menu → Settings → Escape leaves `document.activeElement` at BODY. Closing Menu makes the original child opener hidden; connectedness alone is insufficient for restoring focus, and not every child popup uses the helper. Task 1 requires popup focus restoration. Return to a visible Menu button (or equivalent visible parent opener) when a child trigger is hidden, and cover Settings/History/Quick-play. Root dispatched the focused UI repair after confirmation.

## Resolved during review

### R2 — P2: Normal sparse penalty events were classified as stale

Original `apps/web/src/audio/events.ts:15` returned early when the interval since the previous observed event exceeded 3000 ms. Online penalty emits choosing/reveal transitions with a normal 3000 ms choice period; publisher or network jitter makes a legitimate reveal arrive after that threshold. The worker reproduced choosing at 1000 → reveal at 4050 producing zero sounds instead of one.

Final code removes silence duration as a resync signal and resets transition baselines explicitly on visibility/reconnect/account lifecycle changes. The added sparse-cadence regression and final reported web verification pass. Reviewed this focused fix directly; no remaining finding for R2.

## Named risks checked

- **Startup initial-Persist invisibility and lock ordering:** Persist takes schema-scoped shared startup fence before match 7 and sorted account 12 locks. RecoverPending takes the same fence exclusively before enumeration, uses one READ COMMITTED transaction, locks sorted match IDs and rereads durable job state after each match fence. No second pooled connection is needed while the exclusive fence is held. Existing settling-Enqueue policy remains intact. Real commit/rollback, lock-order mutation and cancellation evidence is recorded in the startup report.
- **Presence queue bounds and sequence retention:** per-world/user alternating transitions, identical-state dedup, terminal leave and bounded overflow behavior inspected. Engine disconnect metadata remains installed before catch-up. R1 is the unresolved cross-player catch-up defect.
- **Map coordinate and authority boundaries:** contain-fit letterboxing agrees with CSS `object-fit:contain`; client coordinates exclude DPR/world-camera zoom. Invalid/static-blocked targets are rejected, provided closures redirect to validated safe entrances, and dispatch revalidates. Online destinations go through the existing adapter without writing avatar position.
- **Modal ordering and native behavior:** minimap is a semantic button with contained activation; closeDialogThen waits for the close event before scheduling destination. Native hidden display, modal input guards, Chat IDs/unread semantics, cleanup and connected-trigger restoration inspected. Root browser confirmation of menu-child focus loss is recorded as R3; those buttons become hidden when Menu closes.
- **Audio lifecycle and dedup:** lazy single AudioContext, preference readiness/account generation, pending resume cancellation, hidden/mute drops, bounded ID/revision caches, eight-voice cleanup and mode stop inspected. Persistent local session lifetimes remain unchanged. No microphone/media-track or LiveKit mute ownership added.
- **Duck producer/consumer:** accepted hit and miss each append one resolution, rejects append none; snapshot coalescing retains resolutions. Match/world/monotonic shot count dedup is shared by ACK/event; timestamp and controller generation/freshness fencing consume delayed results silently. Initial/resync snapshots consume the shot highwater. Actual websocket timestamp/identity/negative assertions are in the reported integration run.

## Verification and acceptance limits

Reviewed reported pinned Go integration/race/vet/build results and final web tests/typecheck/build, rather than rerunning them. Final Task 2 report records 4 JS + 123 TypeScript tests and the targeted Duck integration profiles passing. Root separately reported live Map/Chat bounds at 320×568, 390×844, 768×1024, 844×390 and 1440×900, plus minimap no-movement and Escape focus restoration; root is collecting immutable-build browser evidence.

No claim of physical/mobile device acceptance, actual on-screen keyboard behavior, 200% browser zoom acceptance, audible-device quality, authenticated browser workflows, LiveKit staging, or broad T16 completion. The baseline accurately labels the 20-avatar scene synthetic and excludes authenticated multiplayer/network-owner performance acceptance. Explicit timer/listener inventory and retained-heap acceptance are not established by the reported point-in-time heap/callback measurements. Tasks 3–11 remain outside this review.

## Round 1 scoped re-review — final finding status

**All original findings ADDRESSED. No new important defect found in the focused fixes.** This section supersedes the initial changes-requested verdict above. Scope was limited to R1/R2/R3, their repairs and potential breakage caused by those repairs; no suites were rerun and only this review report was edited.

| Finding | Status | Re-review evidence |
| --- | --- | --- |
| R1 — P1 queued departure advances beyond later reconnect transitions | **ADDRESSED** | `apps/server/internal/lobby/presence.go:119–121` now evaluates `Tick(at)` at the departure timestamp. The existing owner loop still calls `advance()` after the complete queue, so ingestion-time catch-up follows all transitions. Immediate result evaluation before all-departed classification remains present. Inspected the real-owner regression in `apps/server/internal/games/race/lobby_presence_cycles_test.go`, including grace-expired and natural-finish cases and comparison of the full snapshot/result with timestamp-ordered real-engine replay. The fix report records behavioral RED for both cases, GREEN, focused package race checks and real race/penalty integration checks. |
| R2 — P2 sparse online penalty reveal suppressed | **ADDRESSED** | `apps/web/src/audio/events.ts:15` no longer treats a normal receipt gap as resync. Explicit visibility transition reset at line 35 complements existing auth/attachment/status/resync resets. `apps/web/tests/audio.test.ts` includes choosing at 1000 → reveal at 4050 and asserts one sound. Re-read final Task 2 verification report; no additional audio change is needed for this finding. |
| R3 — P2 closing menu-child dialog loses focus | **ADDRESSED** | `apps/web/src/ui/dialogs.ts:4–13` retains an overridable close target and avoids parent restoration while another modal remains open. `apps/web/src/app.js:96–101` waits for feature/document-delegated handlers, then sets the opened child's return target to visible Menu. Settings/History/Quick-play regressions model hidden-menu and inert-modal behavior; authored browser coverage checks focus both inside the child and after Escape. Root independently confirmed the immutable-build browser sequence at 390×844 for Settings, History and Duck: focus remains inside the open child and returns to `#menu-button` after Escape, with centering error ≤0.008px. |

The presence repair leaves the initial complete disconnect-metadata pass intact and removes only premature ingestion-time advancement. It preserves the immediate terminal/forfeit check needed by existing all-gone handling. The UI repair adds no movement dispatch and does not alter closeDialogThen destination scheduling. The audio repair preserves bounded dedup, shot highwater and pending-resume fences.

Reported latest UI verification is 4 JS + 126 TypeScript tests, typecheck/build and diff check passing. The presence report distinguishes the initial sandbox-denied loopback tests from their successful authorized reruns; it records no unresolved failures. These are inspected reports, not independent reruns by this reviewer.

**Final scoped spec/quality verdict: PASS for the reviewed Tasks 0–2 implementation and the three repaired findings.** All previously stated acceptance boundaries still apply. Root retains final stage evidence/commit ownership; this verdict does not claim physical/on-screen-keyboard/audio-device acceptance, authenticated/LiveKit acceptance, broader T16 completion, or implementation of Tasks 3–11.
