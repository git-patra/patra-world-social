# Whole-branch review and publication — 30 October 2026

Range: `a71fd85..e5909fe`, ten activity commits. Fresh independent gpt-6-astra/high reviewer reads the full1.55MB extended-context package, plan/spec and named surrounding paths. No implementation authorship, mutation or delegated review. Initial verdict: **With fixes**. No Critical; three Important and one Minor. The single scoped re-review approves all four fixes; final regression and main publication are complete.

## Final local runtime

Root rebuilt `apps/server/cmd/server` using pinned Go1.26.8 and started `.cache/server-activities` against the dedicated local PostgreSQL on55432. Private ignored environment was sourced without printing credentials. API bound only to127.0.0.1:8080; `GET /health/live` returned200 and `{"status":"ok"}`. There is no `/health/ready` route; the attempted request returned404 and is not readiness evidence. Older T16 readiness remains open.

Production preview on127.0.0.1:4181 was reloaded through CUA with the final entry `index-minbxZXK.js`. The full-screen autumn village, walking controls, Map, Friends, Chat, Menu, private-world and account entry points are present. [Latest screenshot](activities-latest-world.png). No authenticated browser or LiveKit workflow was attempted.

Source repository read-only check: `8d969212d91804e06225f5b1a75eed49e8c25307`, only pre-existing untracked`.idea/`; unchanged. Public destination verified `git-patra/patra-world-social`, remote main still`a71fd85b534984ba162d367bcfc45e98836f25a6`. No force push or public deployment is authorized or used.

## Independent findings

- **R1 Important:** normal claw.move20Hz was charged to the10/s discrete control quota, disconnecting online operators. Classify only move as continuous; retain enter/drop/leave discrete. Actual WebSocket sustained-input regression required because Hub.Execute bypasses the limiter.
- **R2 Important:** every lobby.updated reloaded all four game snapshots even with unchanged match IDs. Three ordinary lobby changes could cause12commands/sec per browser and disconnect observers. Read only missing/new identities, deduplicate in-flight reads, retain explicit reconnect/resync full state and bound retries. Test production store feedback, absent from simple load clients.
- **R3 Important:** h.areaStates kept11records per visited world after runtime destruction. Remove on actual empty-world teardown, preserve while leases retain runtime, verify repeated distinct worlds and recreation.
- **R4 Minor:** E handler allowed walk/ride only while claw/fishing HUD advertised E. Honor these activity interactions without weakening modal/text/repeat guards.

One combined fix wave covers R1–R4, followed by one scoped re-review. No separate per-finding fix agents or repeat whole-branch review.

## Explicit carried triage

Eligible to defer from this activities publication, retained forT16/before-beta:

- Guest race/Duck finish time omits3s countdown; future deterministic correction must preserve idempotent retry timestamps.
- Missing historical Duck details currently become zero/false. Before beta, show detail unavailable honestly.
- Chat limiter integration test depends on calls finishing before wall-clock token refill; serial PASS does not resolve reliability.

Finished-racer later-disconnect bot assistance, ride SFX freshness, Pingpong rules drift guard, ordered presence/startup fencing and Duck shot producer have fixes in this branch.

## Declined-to-judge and scope decisions

Controller accepts these boundaries, with no acceptance upgrade: authenticated browser/LiveKit explicitly deferred; physical mobile/simultaneous touch/60fps lack device evidence; post-GC heap lacks forcedGC; full15-minute crowded vehicle rendering was not measured (historical fixture omission, short corrected evidence only); olderT16 soak/backup/restore/readiness/rollback remains separate; production capacity/deployment not tested or authorized. Broad legacy race formatting is polish outside necessary changes. Final bookkeeping remains controller-owned.

Measured spectator13.942KiB and participant46.582KiB complete-second peaks exceed12/40 budgets. Plan allows disclosed gaps, so average compliance is not relabelled peak compliance or a new hidden blocker.

## Scoped correction review — APPROVED

Original reviewer was unavailable after interruption; a fresh read-only gpt-6-astra/high reviewer reviewed only the30,835-byte correction diff, original findings, brief/report and relevant source context. No new whole-branch review or duplicate test run.

- R1 ADDRESSED: only validated claw.move joins movement quota; enter/drop/leave retain discrete limits. Actual WS test sustains20Hz and checks all discrete boundaries.
- R2 ADDRESSED: existing/pending identities skip reads; missing failed identity retries at most once per second; explicit resync forces fresh reads and old response fences remain. Bounded retry metadata follows current lobbies and clears with attachment.
- R3 ADDRESSED: exact-world cache purge after actual destruction; active leases preserve runtime/cache; repeated distinct-world teardown/recreation covered.
- R4 ADDRESSED: E claw/fishing enabled with existing modal/text/repeat/visibility guards.

New Critical/Important/Minor breakage: none. Out-of-scope observations: none. Reviewer personally read90focusedfrontend/typecheck/realtime-race GREEN logs; root owns final full-suite and browser verification. [Fix report](activities-final-fixes.md).

Root final frontend4JS+239TS, typecheck, lint and productionbuild passed. Pinned Go vet/build exited0; latest API liveness200 verified after final rebuild. Guest E checks passed and screenshots recorded in combined evidence. Final full Go race/integration completed with exit0 after database recovery. All22test-bearing packages passed; application129.173s, realtime13.148s, world31.687s. The opt-in load suite was not redundantly repeated: earlier actual A20/B20/A60 results remain separately recorded, while the newly failed seams have dedicated regressions.

## Final command evidence

Pinned Go1.26.8, private environment exported quietly, GOTOOLCHAIN=local, project-local GOCACHE:

```text
go -C apps/server test -count=1 -race -tags=integration -p 1 ./...
exit0 —22test-bearing packages PASS (full output below)
go -C apps/server vet ./...
exit0
go -C apps/server build ./...
exit0
npm test
4 JavaScript +239 bundled TypeScript PASS, zero failures/skips
npm run typecheck && npm run lint && npm run build
exit0 — entry index-minbxZXK.js
```

```text
?   	patra-world-social/server/cmd/e2edb	[no test files]
ok  	patra-world-social/server/cmd/server	7.343s
ok  	patra-world-social/server/internal/application	129.173s
ok  	patra-world-social/server/internal/auth	29.732s
ok  	patra-world-social/server/internal/chat	7.630s
ok  	patra-world-social/server/internal/config	1.473s
?   	patra-world-social/server/internal/contracts	[no test files]
?   	patra-world-social/server/internal/db	[no test files]
ok  	patra-world-social/server/internal/games/blaster	5.953s
ok  	patra-world-social/server/internal/games/duck	6.109s
ok  	patra-world-social/server/internal/games/model	1.579s
ok  	patra-world-social/server/internal/games/penalty	6.227s
ok  	patra-world-social/server/internal/games/pingpong	5.711s
ok  	patra-world-social/server/internal/games/race	10.153s
ok  	patra-world-social/server/internal/history	28.999s
ok  	patra-world-social/server/internal/http	8.061s
ok  	patra-world-social/server/internal/identity	1.551s
ok  	patra-world-social/server/internal/leaderboard	11.160s
ok  	patra-world-social/server/internal/lobby	7.511s
ok  	patra-world-social/server/internal/moderation	7.901s
ok  	patra-world-social/server/internal/realtime	13.148s
ok  	patra-world-social/server/internal/social	6.554s
?   	patra-world-social/server/internal/testutil	[no test files]
?   	patra-world-social/server/internal/trackgeom	[no test files]
ok  	patra-world-social/server/internal/voice	3.067s
ok  	patra-world-social/server/internal/world	31.687s
ok  	patra-world-social/server/internal/worlds	7.822s
```

Final integration fast-forwarded local main and public origin/main to reviewed code commit `4c91a4e3521b7bad2b6bd79342df8ecec96f6f3f`. Post-push git ls-remote returned that exact SHA; local main and retained feat/world-activities had identical tree `8d0d151d208436641a9a06fe1eba9a376d75d42f`, with clean status. This documentation receipt changes no code. The final receipt commit is also pushed and its SHA reported to the user. No public deployment. OnlyCP1officialPASS; the explicitly open external/performance/T16 gates are unchanged. Raw ignored logs remain available for continuity.

Post-publication checks: API liveness200/statusok; preview HTML references `index-minbxZXK.js`; latest CUA village tab marked as deliverable and shown. Source still`8d969212d91804e06225f5b1a75eed49e8c25307`, only pre-existing untracked`.idea/`. No force push, branch deletion or public deployment.
