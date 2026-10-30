# Activities resume — 28 October 2026

This is the current handoff; older T14-only summaries and README acceptance wording are historical. User requested a summary and concise prompt for a new conversation while Task8 was in progress. Do not restart completed work or discard the working tree.

## State

Repo `/Users/patra/Project/patra-world-social`, branch `feat/world-activities`, latest committed HEAD `0289a10`. Public repo `git-patra/patra-world-social` already exists; locally tracked `origin/main` is `a71fd85`. New activity commits have not been pushed. User authorized local commits including plan/evidence and eventual push to main; no deployment. Source repo `/Users/patra/Project/patra-world` must stay untouched.

Plan: `docs/superpowers/plans/2026-10-28-world-activities-adjustment.md`; spec: `docs/superpowers/specs/2026-10-28-world-activities-design.md`. Follow existing six-stage order (expanded into Tasks0–11). Ledger: `.superpowers/sdd/2026-10-28-world-activities-adjustment/progress.md`; reports/briefs in same ignored directory. Portable evidence: `docs/checkpoints/evidence/CP8/activities-*.md`.

Completed and reviewed:
- Tasks0–2 `bb1869a`: scoped T14 repairs, reconnect/startup reliability prerequisites, centered dialogs/Chat, clickable route map, bounded shared SFX.
- Tasks3–4 `6f89600`: authoritative area leases/clear-before-start and persistent world scenes, input ownership, passive spectators.
- Task5 `93b940d`: two bikes/two cars, free ride, race sterilization, unchanged5lap/5slot race.
- Task6 `ac521cd`: shared free ball, penalty sterilization and passive spectators, delayed result does not lock pitch.
- Task7 `0289a10`: pingpong1v1/explicitBOT,20Hz,first7/winby2/cap11/120s, typed history and guest completion, real DB/WSBOT+twohuman tests. Frontend4JS+180bundled tests/typecheck/lint/build, Go unit/focusedrace PASS; browser guest natural120s result/history/safe return PASS.

Task8 claw has completed independent review and fix round1 on 30 October 2026. Corrected invisible vertical aim and invalid JSON Schema rejection of discrete/neutral commands. Fresh checks: frontend4JS+193bundled/typecheck/lint/build, fullGo, focusedClaw, actualDB20cycles/nohistory PASS; CUA desktop and390×844responsive proxy enter/directions/drop/expiry/exit/history PASS. Portable evidence: activities-task8.md, activities-task8-review.md, activities-claw-browser.md. Included with Task8 local commit; use git log for its SHA.

Remaining in order:
1. Task9 fishing: four spots, Cast/wait/Reel, random fun catch, no completion/history/inventory.
2. Task10 Blaster:1–4humans solo/co-op vsbots,3waves/180s/3HP,≤8bots/48projectiles,noPvP/nofriendlyBOTroster.
3. Task11 combined regression/optimization:20clientA/B,60clients3worlds,15minuteprofile+20lifecycles,frame/tick/traffic/chunk/heap/counters,fullpinnedGo integration-race/vet/build+frontendchecks,wholebranchreview. Then localcommits/mainpublication/showlatest.

## Runtime and safeguards

- Frontend dev4176, stable production preview4181, old designreference4180. Preview4181 restarted with latest production build on30October; API8080 had no listener at resume: build/start and verify health at final; don't claim live new multiplayer until done.
- Pinned Go `.cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go`, `GOTOOLCHAIN=local`, `GOCACHE=$PWD/.cache/go-build`. Export ignored `.cache/database.env` quietly. Never print/commit credentials.
- DB dedicated Colima profile `patra-world-social`, Docker context `colima-patra-world-social`, container `patra-world-social-postgres`,127.0.0.1:55432. Root recovered broken VM and stale SSH forward preserving volume; hostPGhandshakeN andTask7/8actualtestsPASS. Never touch default/other profiles.
- Browser through CUA only; no shell browser automation or injected authentication. Current4181history contains priorrace5/5,penalty3:5,pingpong0:1; preserveuser/testdata. Viewportoverride nowapplies: Task8verified390×844proxy; physicalmobile remainsdeferred.
- OneRAF/sharedAudioService/boundedentities; noheavyframework. Wire2/mapassetkampung-senja-v2/schema1; historicalprotocol1/checksumsunchanged. CasualactivitiesmustneveruseCompletion.
- Performance risks forTask11: broadcastspectatorengine snapshots currentlyneedmeasurement/ratebudget; minor delayedownrideSFXneeds1sfreshness; Go pingpongJSON/constantscurrentlyagreebutdriftguardmissing. More detail in task-11-context.md.
- OnlyCP1officialPASS. Authenticatedbrowser,LiveKitstaging,physicalmobileacceptanceremaindeferred. Task11's15minprofiledoesnotcompleteolderT16's60minsoak/backuprestore/readiness/voiceacceptance.

This handoff file is new and uncommitted; include it with the next reviewed Task8 documentation commit.
