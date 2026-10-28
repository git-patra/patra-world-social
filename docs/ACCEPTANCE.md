# Acceptance evidence index

Updated 27 October 2026. This index distinguishes implemented code from verified product behavior. `docs/PROGRESS.md` is the execution ledger; the active spec and implementation plan remain binding. A row marked pending is not acceptance evidence.

| Area | Current evidence | Remaining acceptance |
|---|---|---|
| Isolation and game-only product | CP1 report; source clone/hash check; CV/resume/business showcase cleanup in C02 | Final source/remote/deploy isolation check at handoff; old facade sign cleanup T15 |
| Map, contracts, movement baseline | CP1 map/rules parity, reachable entrances, camera/zoom, reduced-motion route parity, desktop/mobile screenshots | Online browser movement acceptance pending T5; expanded final districts and physical device UX T15 |
| Username/password/session/recovery | T3 PostgreSQL/HTTP tests and review fixes; T4 ASCII/middleware fixes | Browser registration/login/logout/recovery NOT_RUN by user direction; live socket expiry verified T5; actual media expiry T13 pending |
| Private worlds/invites/roles | T4 real PG concurrency, 50-member boundary, target/expiry/revoke/ban, owner invariant, permissions, transfer/archive; review round1 | Authenticated 3-context browser lifecycle NOT_RUN; active connection teardown verified T5 |
| Realtime world | T5 code accepted after two fix rounds; Go race/real PG20-client movement/reconnect/isolation/revoke smoke;28frontend tests; guest page screenshot | Authenticated browser simultaneous play/offline/background/modal/reduced-motion/mobile acceptance NOT_RUN; final scale/soak T16 |
| Friends/chat/moderation | T6 accepted code; real PG/HTTP/WS persistence, authority, dedup, rollback and moderation tests;34frontend tests; guest UI smoke | Authenticated UI acceptance NOT_RUN; fix authored keyboard/reconnect E2E observation before acceptance/T16; production scale covered laterT16 |
| Lobby lifecycle | T7 code accepted; concurrent Start, durable preparation/uncertain COMMIT, readiness/host/busy/slots/spectators, generation-fenced return and PG failure tests; guest gate screenshot | Authenticated browser acceptance NOT_RUN; concrete engine integration T9–T11, result recovery T8, voice release T13; offline loading copy T15 |
| Durable results/history | T8 code accepted after startup/backlog fixes; real PG immutable jobs/atomic projection/ack-loss/restart/outage, private history after removal/archive, shared60s deadline budget;37frontend tests and guest gate | Authenticated history UI NOT_RUN; concrete game/projector integration T9–T12; initial-Persist/startup boundary verification T16 |
| Penalty | T9 code accepted after two fix rounds; initial Go integration/race and40frontend tests, scoped final-fix tests, guest modal continuity/results CUA evidence | Confirmed pre-existing coalesced reconnect issue O1 carried to T16/final review. Authenticated two-human game/history browser remains NOT_RUN |
| Race | T10 code accepted after2fixrounds; Go integration/race,50frontendtests, scoped finalfix tests; real guest CUA controls/remount/results; durable grid/history | O1 and personal-finish assistance edge T16. Human5lap/1–5human handling/tuning, physicalmobile/FPS, realjitter and online browser NOT_RUN; CP5 pending |
| Duck | T11 code accepted after1fixround; full Go integration/race and53frontendtests baseline, affected Go/TS tests afterfix;8 public-view/realWS timing profiles; actual guest CUA | R2 shot-resolution event producer remains T16. Authenticated browser, physical RTT/jitter/mobile/FPS pending; CLI E2E failed, corrected source NOT_RERUN. Podium T12; CP6 pending |
| Podium/champion | Pending T12 | 3 unique accounts, rational tie-break, atomic concurrent projection, bot/guest/incomplete excluded, PB and champion transition semantics, empty board, account deletion recompute |
| Warkop voice/table chat | Pending T13; user has no LiveKit staging project | Actual SDK claims/admission/8seats/2tables, signed webhook/retry/reconcile, microphone permission/mute/deafen/device UX, real audio isolation and initial+refreshed token replay rejection after kick |
| Guest/settings/deletion | Pending T14 | All3games local, IndexedDB100/retention/reload/reset, no ranked upload, safe guest/account transitions, profile/password/recovery settings, backend deletion/anonymization/ownership handling; account browser actions deferred |
| Accessibility/mobile/art | CP1 baseline screenshots only; final T15 pending |390×844/768×1024/1440×900 final scenes, keyboard/focus/font scaling/reducedmotion/touch, physical-device≥30fps evidence, readable20-avatar state |
| Beta reliability/package | Pending T16 |60clients/3worlds,48voice separately, race detector/regression,60min soak with actual metrics, DB outage/restart/providerfailure/slowclients, backuprestore, private metrics/readiness/drain, local deploy and rollback runbook |

## Evidence anchors

- [CP1 PASS](checkpoints/CP1.md), code C01 `6d040e19f4058e714e97545a06e57185a07d6362`, C02 `d4f60a27bd41048c7ed16a103fbcf4875046ece5`.
- [CP2 in progress](checkpoints/CP2.md); T3 partial `ec154108a6f3f2ec3a83ef1d3b13115359f54990`, T4 partial `e71d00a2f8285892d036eaaef56504649c205ed3`.
- [T3 verification](checkpoints/evidence/CP2/t3-verification.md) and [re-review](checkpoints/evidence/CP2/t3-rereview1.md).
- [T4 verification](checkpoints/evidence/CP2/t4-verification.md) and [re-review](checkpoints/evidence/CP2/t4-rereview1.md).

## External acceptance limits

Browser credential workflows were rejected by automatic approval review and then explicitly deferred by the user. No alternate browser execution path is permitted during that deferral. Pure guest UI checks and isolated backend integration remain separate evidence.

LiveKit staging is unavailable; fake-provider and token-claim unit tests cannot establish actual media isolation or provider replay revocation. Browser emulation cannot establish physical Android/iOS device performance. No checkpoint may be marked PASS while one of its required acceptance items remains unverified.

The goal retains the full approved capacity and feature scope. No remote GitHub/push/public deployment/production hosting change or paid provisioning is included.

- [T5 verification](checkpoints/evidence/CP3/t5-verification.md), [round2 accepted review](checkpoints/evidence/CP3/t5-rereview2.md); partial code commit `7f67defe6a1400ea69fdb8b3142f77e3a6f8c244`.

- [T6 verification](checkpoints/evidence/CP3/t6-verification.md), [accepted review fixes](checkpoints/evidence/CP3/t6-rereview1.md); partial code commit `853d62f3434762c0c5f2b7a58d6b9c9520e819a8`.

- [T7 verification](checkpoints/evidence/CP4/t7-verification.md), [accepted safe-return fix](checkpoints/evidence/CP4/t7-rereview1.md); partial code commit `3e37f91ebeafa6f1c7200eee3622330348c56527`. [CP4 in progress](checkpoints/CP4.md).

- [T8 verification](checkpoints/evidence/CP4/t8-verification.md), [accepted startup/deadline fixes](checkpoints/evidence/CP4/t8-rereview1.md); partial code commit `5476ede4db27ac8beac150dc3eec2dd993dec8aa`.

- [T9 verification](checkpoints/evidence/CP4/t9-verification.md), [accepted activation fix](checkpoints/evidence/CP4/t9-rereview2.md), [guest browser evidence](checkpoints/evidence/CP4/t9-guest-preview.md); partial code commit `5ae6ed657b6b30635fb20135335494aec2893425`. [Round1 review](checkpoints/evidence/CP4/t9-rereview1.md) retains known O1 for T16.

- [T10 verification](checkpoints/evidence/CP5/t10-verification.md), [accepted round2 review](checkpoints/evidence/CP5/t10-rereview2.md), [guest browser evidence](checkpoints/evidence/CP5/t10-guest-preview.md); partial code commit `f9222ccefde585f2c2aba2c02b4015d0d1b503f7`. [CP5 in progress](checkpoints/CP5.md); O1/personal-finish assistance remain T16.

- [T11 verification](checkpoints/evidence/CP6/t11-verification.md), [accepted fix review](checkpoints/evidence/CP6/t11-rereview1.md), [guest browser evidence](checkpoints/evidence/CP6/t11-guest-preview.md). [CP6 in progress](checkpoints/CP6.md); no authenticated browser or physical-network acceptance inferred.

T11 partial code/evidence commit: `607141a58593f3abf473edb369148dcd59fed289`; API8080 restarted and health verified.

- T12 code accepted after1fixround: private top3/PB/champion, transactional deletion seam/admin audit, ordered resumable historical backfill and committed world invalidation. [Verification](checkpoints/evidence/CP6/t12-verification.md), [accepted scoped review](checkpoints/evidence/CP6/t12-rereview1.md), [guest gate/contrast CUA](checkpoints/evidence/CP6/t12-guest-preview.md). Full serialpackage Go race/integration and60frontendtests baseline passed; defaultparallel chatfailure retained. Frontend fix8focusedtests/typecheck/build passed. Authenticated browser NOT_RUN; T14 deletion lifecycle pending; R2 missingdetail presentation and chat timing harness remain T16. CP6 notPASS.

T12 partial code/evidence commit `bb1b4e88c7079ad99442e91b98089a0fe286f006`; preview API8080 restarted, migration/backfill startup and healthok verified.
