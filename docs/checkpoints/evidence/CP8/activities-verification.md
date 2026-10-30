# Activities combined verification — 30 October 2026

Approved activities plan, through Task10 commit `c9fdec6` plus Task11 changes. No public deployment. **Only CP1 is an official PASS.** Authenticated browser, LiveKit and physical mobile remain deferred. Backend test sessions and synthetic scenes are not substitutes for those gates.

## Automated regression

Root ran `npm test`, `npm run typecheck`, `npm run lint`, and `npm run build` against the final frontend: **4 JavaScript + 234 bundled TypeScript tests passed**, zero failures/skips; typecheck, syntax lint and production build exited0. Entry bundle: `index-Duq2do7L.js`. Runner/compiler include every new activity unit file, feature modules, combined fixture and authored `activities.e2e.spec.ts`. The Playwright shell command was not run; CUA drove actual guest gameplay and fixture buttons.

Pinned Go1.26.8 full race/integration, vet/build and focused authority/codec/regression evidence is recorded in [server verification](activities-server.md) and the implementation report. Final-source full Go race/integration passed22test-bearing packages; vet and build exited0. The final load fixture passed all3scenarios.

## Bundle budget

Rebuilt immutable `3079f05` with `git archive` into an ignored baseline directory using the same installed Node25.2.1/npm11.6.2/Vite8.3.4. Every JS chunk was compressed with Node `zlib.gzipSync` defaults. Initial means the module scripts and modulepreloads in `index.html`; shared chunks count once. [Exact per-file bytes](activities-bundle-metrics.json) use KiB=1024bytes, separate from older Vite decimal kB estimates.

| Measure | Result | Budget |
|---|---:|---:|
| Baseline initial gzip |134744B|—|
| Candidate initial gzip |142037B|—|
| Added initial JS |7.122KiB|≤75KiB|
| Added all JS, including lazy/shared chunks once |30.450KiB|Reported separately|
| Largest additional noninitial chunk (legacy Duck lazy extraction) |about5.03KiB|New games each≤50KiB|
| Added compressed SFX bank |0B, existing synth|≤200KiB|

LiveKit chunk is unchanged. Production tests enforce the shared8voice cap. Timestamp-derived flashes do not add an accumulating particle queue.

## 15-minute browser profile

Environment: macOS26.5.2(25F84), AppleM1 reported by the local Go benchmark, in-app Chromium156, DPR1,1280×720canvas. Concurrent local build/Go work occurred during the run. The fixture executes production `createWorld`/`ActivityScene`, `ActivityStore` start/subscriptions/dispose, and `AudioService` lifecycle against bounded public synthetic snapshots. A:5race/2penalty/4Blaster/2pingpong/1claw/4fishing/2spectators. B replaces race/penalty with4riders/3freeball users in the synthetic input, but the original fixture omitted OPEN area metadata: the production adapter therefore suppressed the four free vehicles. The historical15-minute run does not prove free-vehicle render cost. A corrected separate spectator smoke covers those vehicles. Render stress uses8bots/48projectiles.

There are no authenticated browser accounts, sockets or LiveKit in this fixture. Network measurements below use separate actual backend transports. The immutable fixture is `activities-combined-DDJHf2IB.js`, SHA256 `b92c1405f42d294fdf8f9dce3027d7d13eacb6f43152ec88d430cde698d194cd`, started approximately15:05Asia/Jakarta. Later wire tuple hydration is tested separately; its decode cost is outside this immutable render measurement.

[Raw15-minute report](activities-profile-15min.json):

| Measure | Result |
|---|---:|
| Elapsed / completed lifecycles |900006ms /20|
| Render callback samples |51351|
| Render CPU p50 / p95 / p99 |1.7 /4.4 /4.8ms|
| Maximum render callback CPU |6.2ms|
| Frame interval p50 / p95 / p99 |6.0 /33.8 /34.4ms|
| Intervals over50ms / hidden time |0 /0ms|
| Average observed callback cadence |57.06fps|
| Lifecycle setup p50 / p95 |14.2 /19.9ms|
| Synthetic snapshot update p50 / p95 |0.3 /0.4ms|

The render callback CPU fits the desktop≤12ms target for this workload. This excludes browser compositing/GPU work and does **not** establish sustained physical60fps or mobile acceptance. Snapshot construction/update and lifecycle setup are measured separately, rather than hidden in the frame statistic.

Every one of20 cleanup rows reports zero RAFs, intervals, timeouts, listeners, AudioContexts and oscillators. During activity, the fixture observed one worldRAF, one ActivityStore interval, six world listeners and one AudioContext. This verifies the listed production lifecycle, not20 authenticated network joins.

Raw used heap fluctuated from about8.68MB to26.48MB; first/last samples were12,903,324/25,772,401B. Numeric measurement arrays and unforced garbage collection affect these samples. **Post-GC retained heap is UNVERIFIED**; raw growth is not a passed5MiB/10% gate.

The initial [20-second smoke](activities-profile-smoke.json) measured p95CPU2ms; its final13listener count included the CUA helper first injected during instrumentation. It is retained as diagnostic evidence, not a clean4cycle acceptance claim. Priming automation before the long run produced all-zero cleanup. A negative first-RAF timestamp initially prevented long-run startup; the fixture clamps elapsed time and restores instrumentation on failure. That failed attempt is not counted as a completed profile.

## Actual server load and optimization

[Server report](activities-server.md) and [raw final metrics](activities-server-final.json) cover actual isolated PostgreSQL and20/60authenticated test WS clients. A20, B20 and A60/3worlds each use an approximately8second steady measurement window after clearing/countdown; startup/full-resync bursts are outside it. Inputs are normally about8Hz, not a maximum20Hz sender stress test. Trusted Hub placement is disclosed setup; admission and game/casual intents travel through WS.

The original observer total was213.783KiB/s. Changes suppress unchanged casual state, separate game frame wakeups from lobby metadata, filter interest, and bound replaceable spectator visuals. Reliable lifecycle/results remain delivered. World and game owner duration hooks run during actual concurrent clients; queue latency is reported separately.

Final measured averages fit≤40KiB/s participant and≤12KiB/s spectator activity traffic; total hub/ACK traffic is also reported. B20 has a **13.942KiB peak complete1second spectator bin**, above12, from reliable casual lifecycle plus visual bursts. A60 also records a **46.582KiB participant complete1second bin**, above40. The average results are not strict peak cap claims. Bytes exclude WS/TCP/TLS overhead. Actual60client world/game owner p95 is well below25ms in the recorded workload.

The exact capped Blaster wire probe uses4UUIDhumans,32HTML-escaped-character names,8bots and48shots. Full floats were78.389KiB/s at10Hz. A visual-only projection gives3490B per actual event,34.082KiB/s at10Hz;0.001coordinate precision has at most0.0005normalized error. Internal physics and historical results are unchanged. A Go-generated golden passes through actual socket validation, store hydration and61scene records, including command/resync paths.

## Review and remaining acceptance

Independent Task11 review and whole-branch review are recorded separately before publication. Guest finish timestamp offset, missing historical Duck details and timing-sensitive chat test follow-ups are explicitly triaged in the implementation/final-review reports; no silent completion is implied.

Authenticated browser, LiveKit staging/media, physical simultaneous touch/mobile performance, post-GC retained heap, and guaranteed physical60fps remain unverified. OlderT16 also requires its separate60-minute soak, backup/restore, readiness/rollback and provider evidence. The15-minute synthetic profile and8-second transport windows do not complete T16.

Read-only source integrity: `patra-world` remains at `8d969212d91804e06225f5b1a75eed49e8c25307`, with only its pre-existing untracked`.idea/`. No source repository edits or public deployment were performed.

## Post-review spectator cadence smoke

The final production scene was measured separately using immutable `activities-combined-CB_e5eYF.js`, SHA256 `42d20250688c922dfe557fb344fbb70420ff42b61277d5d5f8b0e98ae07207d8`. CUA clicked the explicit20s spectator button; no state/auth injection. [Raw report](activities-profile-spectator.json), [screenshot](activities-profile-spectator.png):20021ms,4lifecycles,300/600ms snapshots,603render callbacks, CPU p50/p95/p99=3.1/5.0/5.5ms, maximum51ms; frame interval p95=34.2ms and zero measured intervals>50ms. These are separate metrics; the51ms CPU outlier is retained, not hidden. All four cleanup rows are zero across RAF/timers/listeners/audio. Hidden time0. This quick mode supplies OPEN area metadata and the production-scene test verifies four free-vehicle actors in B. It validates the affected interpolation/vehicle path, not a replacement15-minute or physical60fps claim. The historical100ms fixture defaults and raw artifact remain unchanged.

Final post-fix root fullfrontend4JS+234TS/typecheck/lint/build PASS; the frontend-only fix does not alter the already measured server tree. Focused review regressions went44/51→51/51; quick-fixture vehicle regression51/52→52/52.
