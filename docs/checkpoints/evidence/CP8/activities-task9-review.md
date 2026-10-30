### Spec Compliance

- ✅ Reviewed Task 9 diff against its brief: four exclusive fishing states, Cast/wait/Bite/Reel, bounded cosmetic pool, transient results, guest/private-world projections, strict fenced intents, and cancellation are implemented (`apps/server/internal/world/fishing.go:52`, `:71`, `:125`; `apps/web/src/activities/fishing.ts:5`; `apps/web/src/activities/store.ts:393`). No fishing completion or inventory path is introduced.
- ✅ The fourth anchor remains (53,33), with reachable standing ground (54,33) within 1.25 tiles. This is a sound visual-anchor interpretation of the approved map, documented by root in `progress.md:55`; validation checks reachable bank ground and both owner/client tests exercise all four standing positions (`apps/server/internal/world/activities.go:139`; `apps/server/internal/world/fishing_test.go:38`; `apps/web/tests/fishing.test.ts:56`).
- ⚠️ Source repository immutability and full release acceptance are controller-owned checks. This diff does not alter map art/collision or introduce a second RAF/audio service. Physical mobile, authenticated-browser spectators, audible output, and LiveKit remain explicitly deferred (`docs/checkpoints/evidence/CP8/activities-fishing-browser.md:11`); the responsive proxy is not physical-mobile acceptance.

### Strengths

- Ownership is bounded to four slots, checks runtime incarnation/session epoch/sequence/spot generation, and expires deadlines before processing commands. A long stall releases an expired cast without replaying intermediate events (`apps/server/internal/world/fishing.go:79`, `:134`; focused ordering check `apps/server/internal/world/runtime.go:90`).
- Movement, leave, disconnect, takeover, removal, and real-match entry clear ownership; obsolete client cast projection disappears immediately on disconnection (`apps/server/internal/world/movement.go:21`, `:41`; `apps/server/internal/world/runtime.go:143`, `:163`, `:171`; `apps/server/internal/world/activity.go:62`; `apps/web/src/activities/store.ts:155`).
- No-history evidence preserves preexisting data rather than proving only an empty database: account history bytes and durable table fingerprints are compared after 40 cycles and lifecycle transitions; guest IndexedDB has a seeded record (`apps/server/internal/application/fishing_integration_test.go:91`, `:92`, `:130`, `:202`; `apps/web/tests/fishing.test.ts:12`).
- Wire validators reject outcome/position/axis fields and full resync requires all four unique fishing IDs. Visual phases remain present with muted sound, and actor count stays four (`apps/server/internal/realtime/activity_protocol.go:192`; `contracts/protocol.schema.json:1257`; `apps/web/src/net/socket.ts:168`; `apps/web/tests/fishing.test.ts:30`).

### Issues

#### Critical (Must Fix)

- None found.

#### Important (Should Fix)

- None outstanding. The initial final frontend failure was a deterministic floating-point fixture boundary, now resolved by the separately reviewed comment and two 100→101ms waits in `apps/web/tests/app-shell.test.ts:228`. The change crosses the existing subscriber gate while preserving Kick/Turun/Naik Sepeda assertions and leaving the independent 10Hz cap test intact. It changes no production behavior. The appended implementation report documents the 100.003ms reproduction and focused 25/25 pass; `/tmp/patra-task9-web-fixed.log` records final 4JS +206 bundled tests, 0 failures.

#### Minor (Nice to Have)

- None found.

### Assessment

**Task quality:** Approved.

**Reasoning:** The implementation follows the owner/client boundaries, uses existing scheduling/audio/rendering infrastructure, and supplies meaningful lifecycle and seeded-history coverage. Reviewed the supplied full diff once and the final fixture-only fix diff once; no tests or git commands rerun. Focused out-of-diff checks were limited to named cross-cutting risks: deadline ordering (`runtime.go:90`), fishing directional-input contract (`input-router.ts:2`), world cancellation integration (`controller.ts:94`, `:143`), and voice admission/exclusion (`application/voice.go:23`; `voice/service.go:169`). Deferred acceptance items above remain deferred, not implied by approval of this implementation task.
