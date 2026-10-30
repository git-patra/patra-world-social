# Activities resume — 30 October 2026

This is the current handoff; older T14-only summaries and README acceptance wording are historical. Task8 WIP was preserved and completed, followed by Tasks9–11. Do not restart completed work or discard the working tree.

## State

Repo `/Users/patra/Project/patra-world-social`, active branch `main`; implementation branch `feat/world-activities` retained, Task8 `4aba8fc`, Task9 `d4e678f`, Task10 `c9fdec6`; Task11 `e5909fe`; final review fixes are included with this handoff (inspect Git tip). Public repo `git-patra/patra-world-social` already exists; the activity range starts at `a71fd85`. Reviewed code `4c91a4e` was fast-forwarded and pushed to public main, with remote SHA verified. This final documentation receipt is included in main; inspect Git for its tip. User authorized local commits including plan/evidence and eventual push to main; no deployment. Source repo `/Users/patra/Project/patra-world` must stay untouched.

Plan: `docs/superpowers/plans/2026-10-28-world-activities-adjustment.md`; spec: `docs/superpowers/specs/2026-10-28-world-activities-design.md`. Follow existing six-stage order (expanded into Tasks0–11). Ledger: `.superpowers/sdd/2026-10-28-world-activities-adjustment/progress.md`; reports/briefs in same ignored directory. Portable evidence: `docs/checkpoints/evidence/CP8/activities-*.md`.

Completed and reviewed:
- Tasks0–2 `bb1869a`: scoped T14 repairs, reconnect/startup reliability prerequisites, centered dialogs/Chat, clickable route map, bounded shared SFX.
- Tasks3–4 `6f89600`: authoritative area leases/clear-before-start and persistent world scenes, input ownership, passive spectators.
- Task5 `93b940d`: two bikes/two cars, free ride, race sterilization, unchanged5lap/5slot race.
- Task6 `ac521cd`: shared free ball, penalty sterilization and passive spectators, delayed result does not lock pitch.
- Task7 `0289a10`: pingpong1v1/explicitBOT,20Hz,first7/winby2/cap11/120s, typed history and guest completion, real DB/WSBOT+twohuman tests. Frontend4JS+180bundled tests/typecheck/lint/build, Go unit/focusedrace PASS; browser guest natural120s result/history/safe return PASS.

Task8 claw has completed independent review and fix round1 on 30 October 2026. Corrected invisible vertical aim and invalid JSON Schema rejection of discrete/neutral commands. Fresh checks: frontend4JS+193bundled/typecheck/lint/build, fullGo, focusedClaw, actualDB20cycles/nohistory PASS; CUA desktop and390×844responsive proxy enter/directions/drop/expiry/exit/history PASS. Portable evidence: activities-task8.md, activities-task8-review.md, activities-claw-browser.md. Included with Task8 local commit; use git log for its SHA.

Task9 fishing complete after independent review: four exclusive visual spots, Cast/wait/Bite/Reel, transient3s result, no history/inventory. Existing anchor53,33 remains; standing54,33 is reachable within1.25. Automated4JS+206frontend/typecheck/lint/build/fullGo PASS; actualDB40cycles preserves seededDuck/history7tables and accountListHistory. Browserdesktop/390×844proxy Cast/Bite/Reel/expiry/movementcancel/Eresult/historyunchanged PASS. See activities-task9.md, activities-task9-review.md and activities-fishing-browser.md.

Task10 Blaster complete after independent fix-round review:1–4humans solo/co-op vsbots,3waves/180s/3HP,4/8/48caps, typed history. FourJS+216TS baseline; final frontend-only fix93focusedGREEN after4RED, typecheck/buildPASS. FullGo integration actualDB/WS4humans+spectatorPASS; exactcapstep10.579µs/0allocs. Browser desktop/390×844proxy gameplay/result/history/return and combinedW+mousefirePASS. See activities-task10.md, activities-task10-review.md and activities-blaster-browser.md. Included in current commit; use git log for SHA.

Task11 implemented: final full Go race/integration (22 test-bearing packages), vet/build, frontend regression, actual WS+DB A20/B20/A60, capped Blaster wire, owner timing hooks and transport coalescing/interest budgets. A 15-minute synthetic browser profile completed20 lifecycles with zero tracked resources after cleanup. Render p95=4.4ms; raw heap is not post-GC acceptance. Historical B render omitted free vehicles; corrected quick spectator fixture covers them. Normal spectator cadence interpolation corrected after independent review. See [combined evidence](../checkpoints/evidence/CP8/activities-verification.md), [server](../checkpoints/evidence/CP8/activities-server.md), [runbook](../runbooks/activities-verification.md).

Tasks0–11 implementation and independent reviews are complete. Whole-branch review found three Important issues (claw transport quota, reactive snapshot bursts, world cache cleanup) and one Minor E-control issue; one combined correction wave and scoped re-review resolved all four. Final frontend4JS+239TS/typecheck/lint/build pass; full Go race/integration, runtime and publication evidence are in activities-final-review.md. No public deployment. External acceptance stays pending.

## Runtime and safeguards

- Frontend dev4176, stable production preview4181, old designreference4180. Preview4181 restarted with latest production build on30October; API8080 rebuilt from final fixes and healthPASS. After interruption, the dedicated VM/SSH forward were recovered preserving PostgreSQL volume; no other profile was changed.
- Pinned Go `.cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go`, `GOTOOLCHAIN=local`, `GOCACHE=$PWD/.cache/go-build`. Export ignored `.cache/database.env` quietly. Never print/commit credentials.
- DB dedicated Colima profile `patra-world-social`, Docker context `colima-patra-world-social`, container `patra-world-social-postgres`,127.0.0.1:55432. Root recovered broken VM and stale SSH forward preserving volume; hostPGhandshakeN andTask7/8actualtestsPASS. Never touch default/other profiles.
- Browser through CUA only; no shell browser automation or injected authentication. Current4181history contains priorrace5/5,penalty3:5,pingpong0:1; preserveuser/testdata. Viewportoverride nowapplies: Task8verified390×844proxy; physicalmobile remainsdeferred.
- OneRAF/sharedAudioService/boundedentities; noheavyframework. Wire2/mapassetkampung-senja-v2/schema1; historicalprotocol1/checksumsunchanged. CasualactivitiesmustneveruseCompletion.
- Task11 resolves delayed ride SFX, Pingpong rule drift guard, finished-racer bot attribution and capped Blaster wire. Average traffic budgets pass in measured windows; complete-second peaks exceed participant40KiB and spectator12KiB. Physical60fps and post-GC retained heap remain unverified.
- OnlyCP1officialPASS. Authenticatedbrowser,LiveKitstaging,physicalmobileacceptanceremaindeferred. Task11's15minprofiledoesnotcompleteolderT16's60minsoak/backuprestore/readiness/voiceacceptance.

Next work is the explicitly open acceptance/T16 list, not a restart of activities implementation. See activities-final-review.md for the carried guest timestamp, unavailable Duck detail and chat-test reliability items.
