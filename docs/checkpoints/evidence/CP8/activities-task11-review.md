# Independent Task11 review — 30 October 2026

Fresh gpt-6-astra/high reviewer; base c9fdec6 to frozen Task11working tree,243644byte full text package. Read-only, no implementation authorship. Scoped task review; whole-branch review follows separately.

## Initial verdict: Needs fixes

Important: the250ms transport minimum over a100ms publisher produces normal300ms spectator intervals (~3.33Hz). ActivityScene treated gaps>250ms as discontinuities and supplied fraction1, so ordinary spectator motion snapped on packets. A focused production-scene probe returned0.5 halfway for200/250ms gaps but1 for300/400ms. Fix must support normal bounded delayed cadence and preserve stale/identity/phase snaps, with focused regressions.

Minor: report A60's46.582KiB participant complete-second peak beside B20's13.942KiB spectator peak. Average-only claims were accurate, but both peak gaps should be equally visible in prose. Controller updated combined/server reports.

Strengths: reliable terminal delivery separated from replaceable frames; cache compares incarnation/generation/payload; exact Go-generated wire fixture traverses real frontend socket/store/scene; measurements distinguish queue/tick, rawheap/postGC, averages/peaks and actualtransport/syntheticrender. Named cross-call checks confirmed resume resets delivery cache, explicit activity resync bypasses suppression, and command snapshots retain world-bound reads. No full suites rerun.

## Acceptance boundaries

Reviewer accepts honest gap reporting under the plan instruction to disclose missed targets rather than inventPASS. Physicalmobile/60fps, authenticatedbrowser/LiveKit, postGCretention andT16soak remainunverified. Existing five-slot/five-lap, secretpenalty, Duck, account and moderation guarantees also rely on earlier functional reviews and fullregression evidence, not thisdiffalone. Controller owns finalplan/PROGRESS/HANDOFF/runbook/commit updates.

## Fix round1

APPROVED (spec and quality). Normal300/600ms and budget-delayed1600ms updates now interpolate over at most600ms; stale receipt/server gaps and identity/phase/generation/rally/wave changes snap. Focused production-scene tests verify receipt/midpoint/completion and four visible riders in the corrected quick B fixture. Both peak gaps and historical fixture vehicle omission are documented. No Critical/Important/Minor remain within this correction scope. Root then completed4JS+234TS/typecheck/lint/build and20s/4cycle CUA spectator smoke. Whole-branch review follows separately.
