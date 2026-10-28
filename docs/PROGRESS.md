# Patra World Social — Execution Progress

Tanggal pemeriksaan: 28 Oktober 2026, Asia/Jakarta.

Ledger aktif eksekusi. Tabel di bawah adalah status terkini; catatan kronologis menyimpan status pada saat pekerjaan dilakukan. Checkpoint hanya PASS setelah seluruh bukti acceptance tersedia.

| Field | Status sekarang |
|---|---|
| `source_repo` | `/Users/patra/Project/patra-world` |
| `source_commit` | `8d969212d91804e06225f5b1a75eed49e8c25307` |
| `target_repo` | `/Users/patra/Project/patra-world-social` — clone independen, tanpa remote |
| `working_branch` | `feat/social-world` |
| `highest_contiguous_passed_checkpoint` | CP1 PASS — 1/8 |
| `last_verified_task_commit` | d4f60a27bd41048c7ed16a103fbcf4875046ece5 — 2/16 task done |
| `latest_code_commit` | See current Git tip for Kampung Senja integration; prior T14 snapshot6737162 retains its pending review |
| `current_task` | Kampung Senja fullscreen integration verified/reviewed; T14 independent re-review remains pending |
| `next_task` | Authenticate git-patra and publish verified main snapshot; then T14 scoped review and remaining T15/T16 acceptance |
| `blockers` | Uji browser akun ditunda pengguna (T3/CP2); LiveKit staging belum tersedia (CP7). Pekerjaan backend dan fitur lain tetap berjalan. |
| `source_changes` | `.idea/` sudah untracked sebelum perencanaan; jangan dihapus atau dibawa ke repo baru |
| `planning_files` | Snapshot awal di `/Users/patra/Project/docs/superpowers/`; salinan aktif dan ledger tersimpan dalam Git repo baru di `docs/` |

## Task ledger

`—` berarti belum ada SHA/evidence. C01–C16 adalah label yang direncanakan, bukan hash.

| Task | CP | Commit rencana | Status | SHA kode nyata | Evidence |
|---|---|---|---|---|---|
| T1 | CP1 | C01 | done | 6d040e19f4058e714e97545a06e57185a07d6362 | docs/checkpoints/evidence/CP1/t1-verification.md |
| T2 | CP1 | C02 | done | d4f60a27bd41048c7ed16a103fbcf4875046ece5 | docs/checkpoints/evidence/CP1/t2-verification.md; t2-rereview1.md |
| T3 | CP2 | C03 partial | implemented_pending_external_verification | ec154108a6f3f2ec3a83ef1d3b13115359f54990 | docs/checkpoints/evidence/CP2/t3-verification.md; t3-rereview1.md |
| T4 | CP2 | C04 partial | implemented_pending_external_verification | e71d00a2f8285892d036eaaef56504649c205ed3 | docs/checkpoints/evidence/CP2/t4-verification.md; t4-rereview1.md |
| T5 | CP3 | C05 partial | implemented_pending_external_verification | 7f67defe6a1400ea69fdb8b3142f77e3a6f8c244 | docs/checkpoints/evidence/CP3/t5-verification.md; t5-rereview2.md |
| T6 | CP3 | C06 partial | implemented_pending_external_verification | 853d62f3434762c0c5f2b7a58d6b9c9520e819a8 | docs/checkpoints/evidence/CP3/t6-verification.md; t6-rereview1.md |
| T7 | CP4 | C07 partial | implemented_pending_external_verification | 3e37f91ebeafa6f1c7200eee3622330348c56527 | docs/checkpoints/evidence/CP4/t7-verification.md; t7-rereview1.md |
| T8 | CP4 | C08 partial | implemented_pending_external_verification | 5476ede4db27ac8beac150dc3eec2dd993dec8aa | docs/checkpoints/evidence/CP4/t8-verification.md; t8-rereview1.md |
| T9 | CP4 | C09 partial | implemented_pending_external_verification | 5ae6ed657b6b30635fb20135335494aec2893425 | docs/checkpoints/evidence/CP4/t9-verification.md; t9-rereview2.md |
| T10 | CP5 | C10 partial | implemented_pending_external_verification | f9222ccefde585f2c2aba2c02b4015d0d1b503f7 | docs/checkpoints/evidence/CP5/t10-verification.md; t10-rereview2.md |
| T11 | CP6 | C11 partial | implemented_pending_external_verification | 607141a58593f3abf473edb369148dcd59fed289 | docs/checkpoints/evidence/CP6/t11-verification.md; t11-rereview1.md |
| T12 | CP6 | C12 partial | implemented_pending_external_verification | bb1b4e88c7079ad99442e91b98089a0fe286f006 | docs/checkpoints/evidence/CP6/t12-verification.md; t12-rereview1.md |
| T13 | CP7 | C13 partial | implemented_pending_external_verification | b6e74fbf12ff0a2913e09888b81c1e3d767566b6 | docs/checkpoints/evidence/CP7/t13-verification.md; t13-rereview1.md |
| T14 | CP7 | C14 WIP | implemented_pending_review_and_external_verification | 67371621b1d650fcd16533a179993905bd7da79c | docs/checkpoints/evidence/CP7/t14-verification.md; t14-rereview1.md |
| T15 | CP8 | C15 partial | fullscreen_village_reviewed; remaining_acceptance_pending | See Git tip | docs/checkpoints/evidence/CP8/kampung-senja-integration.md |
| T16 | CP8 | C16 | not_started | — | — |

## Checkpoint ledger

| Checkpoint | Status | Commit laporan nyata | Next task setelah lulus |
|---|---|---|---|
| CP1 | PASS | 3d3267c7f336c4ddfbfb0f65ae98ac28a9a3a269 | T3 |
| CP2 | in_progress; browser acceptance deferred | — | T5 |
| CP3 | in_progress — T5/T6 code accepted, browser pending | — | T7 |
| CP4 | in_progress — T7/T8/T9 code accepted; browser pending | — | T10 |
| CP5 | in_progress — T10 code accepted; human/browser/device pending | — | T11 |
| CP6 | in_progress — T11/T12 code accepted; browser/physical acceptance pending | — | T13 |
| CP7 | in_progress — T13 code accepted; T14 and provider acceptance pending | — | T15 |
| CP8 | in_progress — village integration verified; final acceptance pending | — | T16 reliability and beta verification |

## Catatan kronologis

Catatan di bawah mempertahankan hasil tiap tahap; status terkini tetap mengikuti tabel di atas.

### Handoff awal

- Mulai dari pembacaan spec dan plan; revalidasi source commit/status.
- Jangan membuat goal atau repo hanya karena membaca file ini. Jalankan ketika pengguna mengirim instruksi eksekusi, misalnya prompt `/goal` yang disiapkan.
- T1 sudah diuji dan direview. Bukti T2 serta pembersihan konten tercatat pada bagian verifikasi; penerimaan T2/CP1 menunggu review akhir.
- Saat mencatat SHA commit laporan, masukkan pada update sesudah commit itu dibuat atau laporan kepada pengguna; jangan mencoba mencatat hash commit dalam commit dirinya sendiri.

## Baseline evidence
- Clone --no-hardlinks mempertahankan source history; branch feat/social-world; remote dihapus hanya pada clone.
- `npm test`: 7 passed, 0 failed sebelum perubahan.
- `npm run check`: passed sebelum perubahan.
- Toolchain teramati: Go1.26.2, Node25.2.1, npm11.6.2.
- PostgreSQL lokal telah divalidasi melalui container khusus proyek; akses loopback dapat memerlukan eskalasi tool.

## T1 verification
- Independent review: approved; no Important/Critical findings.
- Browser evidence: t1-desktop.png1440×900, t1-mobile.png390×844; art/theme preserved; console errors/warnings empty.
- Scope is baseline only; no multiplayer/auth or checkpoint PASS claimed.

## Local runtime
- Isolated Colima profile `patra-world-social` created with activate=false; existing broken default profile unchanged.
- Dedicated PostgreSQL container `patra-world-social-postgres`, loopback55432; generated credentials under ignored `.cache/`, never committed.

## User-directed scope update — game-only content
- User requested removal of all CV/portfolio content from the new game project.
- Removed profile injection, career data/dialogs, LinkedIn and business promotion, public resume PDF and business images/logos. Replaced destinations with village/activity copy, preserving map IDs and art.
- Added all 11 map destinations to the directory; unavailable multiplayer arenas clearly say “Segera hadir”. Existing local penalty remains playable.
- Changed stylesheet name to world-shell.css; retained T2 camera-control styles.
- Verification: make check passed (14 frontend tests; Go tests/vet; TypeScript; frontend and Go production builds). Browser desktop/mobile smoke verified penalty shot resolution, directory, and Warkop pending status.
- These changes and T2 remain uncommitted; C01 is still the last task commit and CP1 is not PASS. No new checkpoint completion is claimed.
- LiveKit staging is unavailable per user reply. Continue independent work during execution; real audio verification stays pending.

## T2 review gate
- Initial independent review found three Important gaps: directional route cancellation, semantic map validation, and server-event schema/decoder. Fix round1 is underway; CP1 remains not passed.
- Deferred minor work: generic village facade signs in T15; nonfinite zoom input guard in T5. Future ingestion must verify asset checksums. Final branch review must check these items.

## CP1 completion
- T1/T2 accepted after independent review and one scoped fix round. C02 includes user-requested game-only content cleanup.
- Final verification: make check17frontend+Go/vet/typecheck/build PASS; uncached map/protocol tests PASS; browser navigation/zoom/nameplate/motion evidence recorded.
- Prior “uncommitted/review pending” notes above are historical. Current task and checkpoint tables govern status.
- Remaining minor work and external prerequisites remain explicitly assigned to later tasks; no multiplayer or beta completion claimed.

## T3 browser verification prerequisite
- Auto-review rejected browser registration because it would submit a new generated password and create persistent local account state without explicit browser-account authorization. No account was created by that rejected action.
- Coordinator inspected the empty registration form read-only and requested user handoff. User replied: “Tunda uji browser; lanjutkan pekerjaan lain.”
- Browser credential-entry/end-to-end tests remain NOT_RUN; do not retry via another execution path or claim completion from service tests.
- Worker moved preview4177/API18080 to a disposable test schema and authored reusable browser tests without running credential flows. Real PostgreSQL service/HTTP tests are distinct evidence.
- Complete code review and independent world/invite work; CP2 acceptance remains pending until the required browser evidence is available.

## T3 review and fix round 1
- Independent review found four Important defects: recovery/login race, swallowed logout DB errors, browser Unicode password length, and limiter saturation. Original implementer assigned fixes with focused regression tests; no partial commit until scoped re-review passes.
- Task 3: minor (deferred, T4): reject non-ASCII username input before lowercasing; Unicode Kelvin sign currently normalizes to ASCII. Include in final branch review.
- Task 3: minor (deferred, T4): distinguish operational database errors from invalid sessions in reusable session middleware. Include in final branch review.
- Ruling: clean auth implementation can receive a partial progress commit and support T4 while browser acceptance is deferred by the user. T3/CP2 cannot be marked complete from backend tests alone.
- Fix round 1 implemented: deterministic PostgreSQL regression reproduced old-password login surviving recovery, then passed with row locking/hash recheck/transactional insert. Logout outage, Unicode password length, and limiter capacity regressions pass; scoped independent re-review pending.
- Source rechecked during T3: HEAD remains 8d969212d91804e06225f5b1a75eed49e8c25307; only the original untracked .idea/ remains.

- Task 3: fix round 1/5 — all four Important findings addressed, zero open; review approves partial progress commit and T4 foundation. Two deferred Minor findings assigned to T4. Browser lifecycle remains NOT_RUN; T3 is not done and CP2 is not PASS.

## T4 implementation review
- Implemented private-world API/domain/UI and ordered migrations. Worker reports real PostgreSQL concurrent redemption, revoked/expired/target/ban, 50-member limit, role/outsider, transfer/archive tests PASS.
- Coordinator make check PASS: 18 frontend tests, Go tests/vet, TypeScript, frontend/Go production builds; staged diff check PASS.
- Read-only guest UI smoke verified login gate and clean browser console; no credentials submitted. Screenshot: docs/checkpoints/evidence/CP2/t4-guest-world-gate.png. Authenticated browser lifecycle remains NOT_RUN.
- Independent review in progress; no T4 commit or CP2 PASS yet.

- T4 initial independent review requires fixes: active-member invite redemption must retain owner/moderator role; moderator-to-moderator kick must match role table; authored E2E must refresh members before kick. Fix round 1 assigned to original implementer.
- Ruling: moderator kick and ban may target any non-owner; owner protection and owner-only role/transfer/archive operations remain. This follows the moderation capability table consistently; a stricter future ban policy would require an explicit product change.
- Task 4: minor (deferred, T6/T15): expand dense new UI/Go control flow and remove unused world-picker selected state when those modules are next edited. Final branch review must revisit readability.
- T3 deferred minors are implemented and verified in T4 review: original ASCII username validation and operational session middleware errors.

- Task 4: fix round 1/5 — three Important findings addressed, no new breakage; scoped independent re-review approves partial progress commit. Backend authorization/concurrency verified; authenticated browser lifecycle still NOT_RUN.

- T4 partial commit: e71d00a2f8285892d036eaaef56504649c205ed3. T3/T4 code review is accepted; CP2 remains pending browser evidence. T5 can consume these verified backend interfaces under the same partial-progress ruling.
- Main API8080 and isolated preview API18080 restarted on T4 code; both connected to project PostgreSQL and listened successfully. Browser credentials remain untouched.

- Partial-progress docs commit after T4: 9ed7cb4f6ca3429b019bc5ba1d83e2402abf8c62. T5 dispatch base is this commit.

## T5 resume — 27 Oktober
- Eksekusi sebelumnya terinterupsi. Agent lama tidak lagi terdaftar aktif; perubahan awal runtime/hub/movement dan tes dipertahankan. Agent pengganti melanjutkan dari working tree, bukan mengulang task. T5 belum direview, diuji lengkap, atau di-commit.
- T5 milestone dari agent pengganti: uncached `go test -count=1 -race ./internal/realtime ./internal/world` dilaporkan PASS setelah transport RED→GREEN; mencakup origin/cookie denial, authoritative movement, revocation closes. Main wiring, real DB load, dan frontend masih dikerjakan; belum acceptance final.
- Database lokal sempat berhenti setelah interupsi. VM dan container khusus proyek dinyalakan kembali; pg_isready menerima koneksi. Tes T5 dilanjutkan, tanpa mengganti data atau profile default.

- T5 implementation ready for independent review. Reported final full Go integration/race suite, vet/build/mod verify PASS; sustained realDB20-client smoke PASS including logout/ban/archive and ongoing rights checks; frontend23tests/typecheck/lint/build PASS. Browser-specific acceptance remains NOT_RUN. Staged review snapshot preserved.
- Ruling: Resume accepts authenticated JoinContext plus lastEventID rather than bare seat/epoch, retaining cookie/world/seat binding. Logout revocation closes the account’s single online seat while only the requested DB cookie is revoked; other valid sessions can deliberately rejoin. This is consistent with the one-active-world-per-account rule and avoids a weaker unauthenticated resume interface.

- T5 independent review found four Important defects: global revoke fencing unrelated worlds; leave/close ordering retaining seats; canceled admission leaving reservations/avatars; main not awaiting HTTP shutdown. Fix round1/5 assigned to implement_t5_resume with gated regression tests, reviewed index preserved.
- Task5 minor (deferred,T15): reset/brand copy claims returning to plaza in online mode although position correctly remains server-authoritative. Final review must verify truthful feedback.

- T5 fix round1 implementation ready: scoped Go race, real Application20-client smoke, frontend25tests/typecheck/lint/build PASS. Original index preserved; task-5-fix1.diff sent to same reviewer for four-finding scoped re-review. Browser acceptance NOT_RUN.

- Task5 fix round1/5: four original Important findings addressed; one new Important regression remains: failed/unconfirmed leave prevents returning to local solo after disconnect/logout. Fix round2 assigned to original implementer; local solo must be independent while new online joins retain seat-release fencing. Staged index now preserves reviewed round1 snapshot.

- Local preview restored27October: Vite4176 exec33510; API8080 exec48005, healthok/page200. CUA guest-only page/console smoke clean; CP3/t5-guest-preview.png. No credential flows. Old4177/18080 previews remain stopped while browser account testing is deferred.

- T5 fix round2 ready for scoped review: local solo detaches immediately; pending-release socket is separate, recovery releases without reattaching, auth-null invalidates queued joins.28frontend tests/typecheck/lint/build PASS; no backend changes. task-5-fix2.diff submitted to original reviewer.

- Task5 fix round2/5: one Important regression addressed, zero open; scoped review approved code readiness. Browser acceptance NOT_RUN; partial progress commit authorized by existing plan ruling.
- Task5 minor (deferred,T6): redundant world.setOnline(null) during pending-seat recovery can cancel an active solo route; preserve local route when already detached. Carry to T6 controller integration and final branch review. Reset/brand online feedback minor remains T15.

- T5 partial commit 7f67defe6a1400ea69fdb8b3142f77e3a6f8c244. Two scoped fix rounds accepted; no Important issues remain. T5 remains implemented_pending_external_verification, CP3 not PASS. Source HEAD/status rechecked unchanged; target feat/social-world has no remote.

- T6 dispatched to /root/implement_t6 (gpt-6.1-sol high fork none), base10aab4cfd482a71d2ff222835dce8bb982a90eb0. Brief task-6-brief.md; report task-6-report.md. Scope friends/worldchat/moderation/in-app invites; no credentialbrowser tests.

- Follow-up game-only cleanup27October: audit found inherited CV/career docs and legacy screenshots still present outside runtime. Removed CONTENT/PLAN/IMPLEMENTATION/VERIFICATION, obsolete renderer contract, historical source deployment guide and14 inherited screenshots from active tree (Git history retained). README now documents actual Go+PostgreSQL startup, current feature/acceptance limits, HTTP+WS proxy and integration commands. Source repo untouched; no source code changes in this documentation cleanup.

- Independent game-only follow-up audit accepted b448cf4 cleanup with no new actionable findings. README commands matched committed config; no remote/deploydestination. Existing cosmetic CONNECT/P facade remainsT15. Earlier CP1 screenshots are historical baseline evidence, not current product; retained without rewriting provenance. Evidence docs/checkpoints/evidence/CP1/game-only-followup-review.md.

- T6 implementation ready for independent review. Full Go race/integration suite PASS; post-self-review ordering fix scoped race/integration PASS; frontend31tests/typecheck/lint/build PASS. Real HTTP+CSRF/WS durableACK tests and deferred-COMMIT rollback evidence present. Staged snapshot held for review; basebb3eeaa. Global3s lease throughput limitation recorded; browser acceptance NOT_RUN.

- T6 independent review dispatched /root/review_t6 (gpt-6-astra high forknone), staged package196118bytes, basebb3eeaa. Guest-only CUA smoke confirms friend gate, disabled chat send, typed WASD/Right does not move avatar, cleanconsole; evidence CP3/t6-guest-preview.md. Authenticated acceptance remains NOT_RUN.

- Task6 initial review:2Important P2 findings (Hub.Close lease/expiry deadlock; delayed send ACK erases newer draft). Fix round1/5 dispatched to original /root/implement_t6; index preserves reviewed snapshot.
- Task6 minor (deferred,before browser acceptance/T16): social.e2e input.fill does not trigger movement keydown; immediate offline→online can pass without actual disconnect/resume. Replace with keyboard events and observed connection interruption/new snapshot before accepting reconnect/dedup. Source remains AUTH_BROWSER_NOT_RUN; final review must verify correction.

- T6 fix round1 implemented: deterministic selected-expiry shutdown RED→GREEN; actual chat-form deferredACK/edit/retry/worldswitch tests RED→GREEN. Final realtime race packagePASS9.909s and34frontend/typecheck/lint/build PASS. Four-file scopedfix sent to same reviewer; index remains unchanged. Browser acceptance NOT_RUN.

- Task6 fix round1/5:2Important findings addressed,0open; scoped review approved. Deferred P3 browser-source correction preserved beforeacceptance/T16. T6 ready for partial progress commit, implemented_pending_external_verification; CP3 notPASS.

- T6 partial commit 853d62f3434762c0c5f2b7a58d6b9c9520e819a8. Two P2 fixes accepted in one scoped round; global3s lease throughput limit and P3 acceptance-source fix retained. T6 and CP3 await authenticated browser acceptance; T7 independent work continues.

- T7 task-brief extraction attempted; heading style T7 unsupported by script. Manually composed exact task/spec with acceptedT5/T6 interfaces, nofakeengines, game15s/world30s distinction and race lifecycle excerpt. API preview restarted fromT6 on8080 exec94156; healthok and anonymousfriends401. Vite4176 unchanged.

- T7 dispatched /root/implement_t7 (gpt-6.1-sol high forknone), baseef7ed2dddd738c36392d88c7a525e8af1b849d95. Report task-7-report.md. Implement lifecycle/registry/UI and realDB tests; concrete enginesT9–11, no productionfakegame.


Ruling: persist `running` + `prepared_at` + frozen roster/seed/version/command identity with nullable `starts_at`; confirm COMMIT (or reconcile stable ID) before owner assigns official `clock.Now()+3s` and activates/publishes. This guarantees full race countdown after storage preparation. Official starts_at projection may be asynchronous and bounded, idempotent and unable to overwrite a different official timestamp or terminal state. Failure is explicitly observable and does not mean timestamp was stored; authoritative MatchStart retains exact time for T8 final result input. Cancel/wait owned workers safely. A crash with running/no durable result remains aborted_server even with NULLstarts_at; no full-state recovery or winner promise. No generic retry subsystem inT7; T8 durable results will project exact timestamp. Cost: a crash before metadata projection can leave start time unknown for an aborted match, accepted over shortened countdown or fabricated timing. Test slow initial persistence/schedule and duplicate command identity/time.

- CP8 prerequisite inquiry sent asynchronously: availability/model/OS/browser of physical Android/iPhone for later beta performance acceptance. No device availability assumed and no test approval inferred; implementation continues. Authenticatedbrowser deferral and absentLiveKit staging remain unchanged.

- T7 incremental report: bounded lobby owner/registry, PG004activeclaims, frozenroster/secretseed, postcommit3s countdown and uncertainCOMMIT sameID reconciliation implemented. Focused race and realPG12-concurrentStart/uniqueness/rollback/lostACK tests PASS (preliminary, not finaltaskacceptance). ReactUI/contracts/protocolintegration/selfreview pending. Natural service/start/presence/PGstore split approved within lobby package.


Ruling: intentional departure/game grace expiry releases that human's active_match_users claim independently of remaining match duration, while frozen participant stays for DNF/history. Bind DB deletion and async memory completion to userID+matchID so old cleanup cannot clear newer ownership. Retain busy until confirmed; transient/ambiguous failure must have bounded retry/reconciliation after recovery, not permanent busy from one failed call. Concrete game engine owns DNF/forfeit/abandon outcomes; T8 still gates whole-match completed release on durable result.


Ruling: registry start preparation delay is race3seconds, penalty0, duck0 after confirmed persistence. Penalty3second secret-choice deadlines and Duck60second session clock belong concrete engines, not a shared extra countdown. Preserve fullrace3s after slow storage; tests cover allthree start timestamps. This resolves generic Start default without adding an unrequested game delay.

- T7 ready for review: fullGo race/integration PASS; final registrytiming scopedlobby race/PG/vet/build PASS; frontend36tests/typecheck/lint/build PASS. Stagedsnapshot preserved; task7-review-package generated. Preview API remainsT6; no browsercredential actions. Concreteengines/results/voice remain later tasks, not simulated.

- T7 initial review: one Important P2, stale arena return can race a newer lobby claim and teleport/reset activity. Fix round1/5 assigned to original implement_t7 with deterministic interleaving test; staged review snapshot preserved. Return delivery error handling included in fix scope. No other Important findings; browser credential tests remain deferred.

- T7 guest-only CUA smoke: Race Park navigation opens private-lobby gate, no guest join/start, console warnings/errors empty; CP4/t7-guest-preview.md + PNG. Backend preview remains T6, so this is not online integration evidence. Minor deferredT15: disconnected guest should not show indefinite “Memuat ketersediaan…”.

- T7 fix round1 ready: generation-fenced return mutation, delivery ACK/retry, deterministic old-return/new-claim RED→GREEN. Scoped application/lobby realPG and world/lobby/realtime race tests plus vet PASS; original index unchanged. task-7-fix1.diff sent to same reviewer.

- Task7 fix round1/5: one Important P2 addressed, zero open; scoped review approves spec and quality. Generation-fenced return with retained delivery retries accepted. T7 code ready for progress commit; authenticated browser acceptance remains NOT_RUN, CP4 still awaits T8/T9 and end-to-end acceptance.

- T7 partial commit 3e37f91ebeafa6f1c7200eee3622330348c56527. One P2 safe-return fix accepted; backend/test interfaces ready for T8. Authenticated browser acceptance remains NOT_RUN; CP4 not PASS. Guest offline loading copy remains MinorT15.

- T8 dispatched /root/implement_t8 (gpt-6-astra high fork none), base deaa1d648a95fa10ae1426b6d257eddf1fcc89cc. Brief task-8-brief.md restored manually after unsupported-heading extraction; no final-brief truncation. Accepted architecture: typed immutable results, owner-bound extraction/outside-owner PG enqueue/finalize, same-lock uncertain-write reconciliation, startup recovery before admission, bounded IDs after60s detach and authoritative Start gate. Optional engine Result absence is observable and cannot fabricate results; later engines own result display timers. Root T7 API restarted on8080 exec88054, healthok; Vite4176 remains.

Ruling correction (T11 obligation): spec §7 explicitly requires Duck countdown3seconds then60seconds gameplay. Earlier T7 ruling described duck0 as avoiding an unrequested delay; that rationale was mistaken. T7 has no production engine, so no live Duck behavior is affected, but T11 must set authoritative Duck countdown3seconds (prefer registry delay like race), update timing test/rules and make startsAt the gameplay boundary. Do not preserve duck0 as a product requirement or add a second countdown. T8 validation must use actual MatchStart timing rather than hard-code the provisional per-game delay. Cost if missed: shortened/preemptive Duck session; this is required before T11 acceptance, not optional polish.

- T8 implementation milestone (preliminary, not task acceptance): worker reports real-PG idempotency/conflict, transaction trigger rollback, restart jobs, NULL-start orphan, advisory-lock reconciliation and own/other-history authorization PASS; lifecycle Start gate and60s detach tests PASS. HTTP/application wiring implemented; frontend/history UI, deeper ack-loss integration and final affected checks still in progress. Public identity overlay retained for T14 anonymization; typed race lap/bestLap and duck hit/shot detail seam included.

- T8 ready for independent review: full Go race/integration and37frontend/lint/typecheck/build PASS; final isolated-pool outage test scopedrace PASS. Root missing Go vet/build PASS, actual logs matched report. Staged37-file snapshot (118580bytes) preserved; root docs/evidence excluded. Browser acceptance NOT_RUN; guest-only History gate evidence available.

- T8 independent review dispatched /root/review_t8 (gpt-6-astra high fork none), staged package118580bytes/37files, base deaa1d648a95fa10ae1426b6d257eddf1fcc89cc. Root guest History gate/console screenshot verified; authenticated browser remains NOT_RUN. T9 brief includes adapter/timing/result semantics handoff draft, to revalidate after T8 acceptance.

- T8 review preliminary: real-PG overlay regression confirms startup RecoverPending stale NOT EXISTS snapshot can mark a match aborted_server after a concurrent old enqueue commits its durable job; Finalize then rejects it. Reviewer is also testing backlog/discovery60s deadline ordering. No fix dispatched until complete findings package; staged snapshot remains intact.

- T8 initial independent review:2Important findings, R1/P1 startup stale absent-job snapshot can abort committed result; R2/P2 detached reconciliation backlog can hide newly-finalizing match beyond60s. Reviewer proved R1 with realPG overlay and R2 after63.035s respecting3s operation contexts. Fix round1/5 dispatched to original implement_t8; report task-8-review.md includes permanent-regression requirements and temp probe source. Index preserves reviewed snapshot.

- T16 failure-test follow-up (unverified concern, not a T8 finding): test replacement startup while an initial T7 Persist match row is still uncommitted, then commits after recovery enumeration; ensure no ownerless running row/claims survive. Carried to task16 brief and final review, distinct from confirmed T8 settling-result-enqueue bug.

- T8 fix round1 ready: startup decisions use shared per-match fence then fresh read; one ready result per bounded Step with all deadlines checked first. Permanent R1/R2 regressions RED→GREEN; history/lobby/application/realtime realPG race tests, Go vet/build and diff check PASS. Four-file fix package sent for scoped rereview; index unchanged, browser NOT_RUN.

- Task8 fix round1/5: R1/P1 and R2/P2 addressed, zero open; independent scoped review approves spec and quality for T8 code. Root checked actual four-package fix race log. Ready for partial progress commit; authenticated browser and CP4 acceptance remain pending.

- T8 partial commit 5476ede4db27ac8beac150dc3eec2dd993dec8aa; both Important findings accepted after one fix round. T8 remains implemented_pending_external_verification; CP4 not PASS. Source rechecked unchanged at8d96921 with original untracked .idea only; target feat/social-world still no remotes. T9 is next.

- T9 dispatched /root/implement_t9 (gpt-6.1-sol high fork none), base e613707a6f64faea291c137504d2de7a1170e9b2; full task-9-brief.md includes accepted T7/T8 adapters, authority/timing/projector/rematch constraints. API preview restarted from accepted T8 on8080 exec30848, healthok and anonymous history401; Vite4176 unchanged. Authenticated browser testing remains deferred; guest game testing can proceed independently.

Ruling (T9 input timing): receivedAt is the server clock at authoritative lobby-owner ingestion immediately before Choose, after the bounded transport queue; it is not raw TCP/frame arrival or client time. Exactly2999ms before the deadline is eligible,3000ms is rejected at this boundary; revealed turns never change retroactively. A world-bound live-admitted-seat hot path may avoid SQL under the existing admission/revocation lease, while preserving session-expiry bounds, world/seat/epoch fencing, ongoing revalidation and revoke behavior. Cost: congestion can reject a frame that physically arrived earlier but reached the owner after deadline; no raw-arrival guarantee is claimed. Required tests cover delayed queue, blocked auth store, revoke/expired seat, and direct exact deadline. Natural engine Tick before grace departure is approved, retaining intentional-leave semantics and natural-finish precedence.

- T9 milestone (preliminary): worker reports engine/inert adapter/projector/public events/typed outcomes/guest TS UI implemented; focused Go/TS and typecheck PASS. Exact2999/3000, queued-late/reveal, admitted-seat no-SQL/epoch/revoke/expiry coverage added. Actual isolated WS/history/return/rematch integration still in progress. Internal Application constructor with existing world.Clock injection approved; public New remains WallClock, no test endpoints, history worker stays single-owner.

- T9 guest CUA smoke found active-duel reset on modal close/reopen, despite UI promising continuation. Observed actual choice lock, automatic timeouts, six-kick score1:2, then reopened initial Start screen. Evidence CP4/t9-guest-preview.md + live/failing screenshots; sent to implementer during initial work for controller persistence/catch-up regression. No credentials; API stillT8.

- T9 guest modal retest PASS: active duel survives close/reopen, then reaches real4:3 result with10kicks while panel closed; final Guest Menang/BOT Kalah and Main lagi available, console clean. CP4/t9-guest-preview.md records original defect, invalidated HMR attempt and stable-page retest. MinorT15: stray leading0 overlaps zone number labels. Browser credential acceptance remains NOT_RUN.

- T9 implementation milestone: worker reports actual backend human/human WS deadlines, duplicate/stale/world/spectator guards, projection retry and mutual rematch, human/bot return/departed cleanup, forfeit/abandon/precedence tests PASS. Changed-package race/integration and40frontend tests/typecheck/build/lint pass; final full Go/vet/build/report pending. No ready-for-review snapshot staged yet.

- T9 ready for independent review: full Go integration, changed-package race, Go vet/build PASS; web40tests/typecheck/build/lint PASS; staged snapshot preserved. Full report task-9-report.md separates actual backend/guest-CUA evidence from authored NOT_RUN Playwright and deferred authenticated acceptance. Guest persistence defect fixed before review; preview API remains T8.

- T9 independent review dispatched /root/review_t9 (gpt-6-astra high fork none), base e613707a6f64faea291c137504d2de7a1170e9b2, staged45files/167896byte package. Current live inventory confirms reviewer running, implementer completed and available for fixes. Previous goal turn was progress: T9 implementation/test evidence, real guest defect+fixed retest, review dispatch. Not impasse; no restart.

- T9 initial review: 3 Important findings confirmed by focused overlay probes: P1 delayed/coalesced disconnect ingestion shifts engine grace and can fabricate win/forfeit instead of abandoned; P2 rejected-choice shared budget can disable opponent input; P2 separate release/rejoin can split a mutually agreed rematch under competing claims. Fix round1/5 dispatched to original implement_t9, preserving staged review baseline. Required regressions include real owner timing/FinishedAt, admitted invalid-command flood, and atomic rematch ownership/release failure/return fencing. Choose owner-ingestion ruling does not apply to known disconnect timestamps. Authenticated browser remains user-deferred.

- T9 fix round1 milestone: permanent RED regressions reproduced delayed/coalesced original grace timestamp errors (including FinishedAt), natural completion1ms after true cutoff, and rejected-command budget exhaustion. Rematch design retains arena and busy claims through guarded external Terminal(completed), then atomically replaces the exact consented pair with fresh IDs/unready; no free-claim interval or return events. Failure retains old results/claims. Competition/failure/fence regressions added; implementation and covering GREEN tests pending.

- T9 fix round1 ready: original/coalesced presence times and explicit grace expiry, accepted-only replay ledger, and atomic reserved rematch implemented. Three affected Go packages integration/race PASS; final focused regressions, vet/build and diff check PASS. Appended report contains commands/output and distinguishes stub interleavings from real PG/WS. Staged baseline tree unchanged; task-9-fix1.diff includes all three new test files. Sending same reviewer scoped rereview.

- Task9: fix round1/5 (3 original findings addressed, 1 new Important open — start activation replays expired without original disconnect metadata; commits e613707..e613707, uncommitted round1 snapshot tree2f9a1bae9afa38b1d287d677f6e86ead874a68f6). Original implement_t9 dispatched round2 with actual lobby/adapter held-persist regression, single expired vs still-in-grace/all-gone, FinishedAt clamped to official start. Round1 staged only as next review baseline; no T9 commit/acceptance yet.
- T9 scoped-review out-of-scope O1 (confirmed pre-existing, deferred to T16/finalreview): processed disconnect100000, timely resume101000/new disconnect102000 coalesced before owner ingestion102100 loses resume and expires115000 instead117000. Required regression must cover old/new cutoff, later final resume, actually late first resume; preserve ordered transitions without reviving expired authority. Exact probe/source in task-9-rereview1.md. This remains known unresolved behavior before beta, not accepted correctness or silent discard.

- T9 fix round2 ready: activation installs all preserved disconnect metadata before replaying terminal departure, immediately finalizes an already-ended running engine. Permanent held-persist tests cover exact/late activation cutoff, still-in-grace, all-gone no-fabricated-result, intentional leave/clamping. Penalty/lobby race+PG and actual application penalty integration, vet/build/diff PASS; O1 untouched. task-9-fix2.diff includes new activation_test.go; same reviewer scoped rereview pending.

- Task9: fix round2/5 (R1 activation addressed, zero open in scoped fix review; commits e613707..e613707, code not committed yet). Independent spec/quality APPROVE T9 code scope. Original1–3 remain addressed; separate pre-existing O1 retained for T16/finalreview, not resolved. Ready for partial progress commit; browser acceptance/CP4 remain pending.

- T9 partial commit 5ae6ed657b6b30635fb20135335494aec2893425; two fix rounds accepted. Full initial integration/changed-package race and40frontend tests passed; final fixes have covering scoped tests. Browser acceptance remains NOT_RUN, CP4 not PASS, O1 coalesced reconnect bug remains explicit T16/finalreview work. Source unchanged at8d96921 with only original .idea; target feat/social-world has no remotes. T10 next.

- T10 dispatched /root/implement_t10 (gpt-6-astra high, fork none), base f64168fe26be58d8d2a4832f82b2d662a92a7c79. Full task10 brief includes exact race rules, accepted T9 timed presence/owner-input/result seams, persisted randomized grid and30Hz/15Hz scheduling requirements. Worker implements actual physics/bots/guest controls/history, no credential browser workaround. Root preview restarted accepted T9 on8080 exec14703, startup14:26:30 local and healthok; Vite4176 unchanged. T9 O1 remains explicit T16/finalreview defect.

Ruling (T10 randomized grid): approve optional RegisterPrepared(game,factory,prepare) with a preparer receiving a cloned MatchStart and returning only an ordered Actor slice/error. Generic lobby validates an exact permutation preserving all Actor fields/identities, then freezes that ordering before inert factory creation and PG Persist. Race owns seeded shuffle; stored roster_slot is actual grid slot, reused by engine, public snapshot, race_results.start_slot and tie-break. Existing Register games use identity ordering. Do not add MatchStart/Actor JSON fields or mutate other metadata; retries reuse the same prepared roster/seed, no reshuffle after persistence. Failure validates before persist/activation. Cost: a narrow registration seam, justified by durable randomized grid and immutable historical result compatibility. Tests must reject duplicate/missing/altered actor preparations and prove real PG stored order matches runtime/tie-break.

- T10 milestone (preliminary): implementer reports pure Go engine tests PASS for countdown/input expiry/walls, ordered/reverse checkpoint fixtures,30Hz/15Hz under50ms owner calls, bounded catch-up,5-human tie order, disconnect takeover/grace/DNF/abandon, four bots completing5laps and45s cutoff. Prep/persistence/TS/UI integration still underway; no task acceptance yet. Existing sandbox-denied transport attempts are not executed test evidence; isolated integration escalation allowed. Root CP5 pending-evidence matrix created, guest preview tab2 healthy on4176 and marked handoff for later playable race check.

- T10 actual guest CUA smoke: keyboard gas/combined pointer-gas+steering responds,5roster/bots advance and finish, stable modal close/reopen retains elapsed135.39s/results, GuestDNF at137.63s=first92.63+45, return restores local start, console clean. CP5/t10-guest-preview.md and captures saved. Initial clipping inference withdrawn: DOM1280viewport/dialog850fits, nativecapture938wide; no established CSSoverflow. HMR-interrupted attempt explicitly inconclusive. Physical devices/online/human5lap playtest remain NOT_RUN; frontend freeze released.
- T10 backend milestone (preliminary report): realPG application test passes prepared roster/persisted slot/runtime identity, duplicate start, spectator guard, countdown/actual50ms→30Hz, four bots5laps, idempotent result/history and10s release. Implementation caught/fixed DNF empty lap-array SQLnull and tie/grace/resume/event cases. Full verification/report/review still pending.

- T10 self-review follow-up: implementer found race vector division lacked required residual carry despite Go/TS parity; adding carry and shared vectors, final clock freeze and hero copy. Initial fullGo integration reportedly passed before refinement. Earlier actual guest times explicitly labelled pre-carry build; root will do focused final-build check after stable candidate, not claim old timing for amended physics.

- T10 final-candidate actual guest smoke PASS within scope: residual-carry build control response, same-session closed-panel catchup, fourbotsfinish93.97–99.77s, GuestDNF138.97=first+45, final clock unchanged after13s, hero corrected, console clean. CP5/t10-guest-preview.md appended and t10-guest-final-results.jpg saved; no online/device/human5lap acceptance inferred. Backend fullGo-race run currently has one penalty integration connection-fenced failure under diagnosis; not ready for review yet.

Ruling (T10 penalty integration harness): approve test-only continuous drain of each real WebSocket after the fixture's last synchronous send/read assertion and before accelerated direct-dispatch turn loop. Worker reproduced connection-fenced4/5 isolated-race runs; fixture stopped consuming all three sockets while advancing many public events, so bounded slow-client fencing was expected. Preserve single reader per socket, cancel/join drains cleanly, keep explicit slow-client/fencing tests and all admission/result assertions. No production auth/queue limits or guard relaxation. Retain failing runs and5-repeat/full-suite rerun in report. Cost: controlled fixture lifecycle change outside race, justified by maintaining actual transport consumption rather than suppressing a safety invariant.

- T10 penalty harness refinement: continuous readers alone still failed5/5 because accelerated virtual turns outran authorized writer delivery and bounded64-event queue. Final test-only harness now awaits each turn public snapshot on all3single readers before advancing fake time, validates event content, surfaces unexpected close/errors and cancel/joins. Isolated race-detector fixture now5/5PASS (0.67–0.91s each), full-suite rerun pending. No production guard/limit changed; report must retain both failed attempts and corrected64queue explanation.

- T10 ready for independent review: final fullGo integration with race detector PASS,49webtests/typecheck/lint/build andGo vet/build PASS; test-only finalasset/protocol assertions also PASS. Penalty fixture failure investigation retained in report, now paced actual3socket snapshots with5/5isolated andfullPASS. Staged source/contracts snapshot preserved; root docs/CP5evidence excluded. task-10-review-package.diff prepared; noT10commit yet.

- T10 independent review dispatched /root/review_t10 (gpt-6-astra high,forknone), staged53files/207276bytes, basef64168fe26be58d8d2a4832f82b2d662a92a7c79,index treefdd7cff0faad511a055313511a187386e85e5a4f. Implementer complete and available for fixes; root actualguest evidence finalized, APIpreviewstillacceptedT9. This goal turn progressed through CUA verification/correction, final physics recheck, fixture diagnosis/ruling, fullverification and review dispatch; not impasse. T11 brief prepared but awaits accepted T10interfaces.

- T10 review preliminary: guest component seq resets on remount while persistent engine retains prior seq; rootactualCUA confirms W2s afterreopen52.46→54.66s leaves kart18432,5600, laterW2s136.46 moves28423.24 (samefocus). CP5guest evidence appended; prior smoke only proved visual continuity. Reviewer also reproduced race disconnect Presence ticking before full pending metadata batch, making same events map-order dependent; distinct from O1. Await complete report before dispatching fixes; staged snapshot unchanged.

- T10 initial review completed: 2Important findings R1 guest remount seq resets and temporarily disables controls; R2 race Presence ticks during unordered disconnect-metadata pass, causing map-order-dependent motion. RootactualCUA confirmsR1; reviewerfocusedoverlay confirmsR2. Fix round1/5 dispatched original implement_t10, reviewed staged treefdd7cff0faad511a055313511a187386e85e5a4f preserved; noT10commit/acceptance. Required immediate-first-input and actual owner-batch/grace/activation regressions in fullreview.
- Task10: minor (deferred): new race local.ts physics/controller and Race.tsx effects use compressed long lines; format into readable blocks during final code-quality polish/review. Not a third fix-loop blocker. Carry to T16/final review; do not silently discard.

- T10 R1 fix milestone: session-owned sequence +600input remount regression, web50tests/typecheck/build reportedPASS. RootactualCUA retest confirms first W2s+D650ms immediately moves/turns after31.42sclose/38.84sreopen to31421.93,8396.94,57° by41.69s; consoleclean, CP5fixedcapture/evidence saved. R2 backend work still underway, no reviewacceptance yet.

- T10 fix round1 ready for scoped rereview: R1 session-owned sequence and actual CUA first-input retest PASS; R2 metadata-only installation with pre-disconnect held-intent causality, both direct permutations, actual held-owner10scenarios/activation/terminal publication regressions. Race/lobby -race and4application integration -race PASS,50webtests/typecheck/build plus final10focused PASS. Preserved staged tree fdd7cff0faad511a055313511a187386e85e5a4f; task-10-fix1.diff includes both new Go regression files. O1 unchanged, noT10commit yet.

- Task10: fix round1/5 (2 addressed,1newImportant open — R3 post-global-cutoff disconnect stamps BotAssisted into earlier natural result; commits f64168f..f64168f, reviewed round1 snapshot treecb2355e8e2c7f3c2cf65d82f3fe0b89979c11808). Original implementer round2 next; R1/R2 remain addressed.
- T10 out-of-scope existing edge for T16/finalreview: individually finished racer can receive BotAssisted from later disconnect while others still race; prior implementation also permitted it. Assistance history should reflect the actor racing interval; inspect/fix before beta. R3 scoped blocker is post-global-cutoff only.

- T10 fix round2 ready: R3 assistance is recorded on authoritative advancement strictly before actual global terminal time; exact/later cutoff retains original human intent and legitimate prior assistance. Absolute/realshortenedcutoff/allfinished, actual held-owner terminal result+event and adapter regressions PASS; race/lobby -race and2affectedPGresultintegration -race PASS. Four race files only, no frontend/generic/O1 changes; task-10-fix2.diff against staged treecb2355e8e2c7f3c2cf65d82f3fe0b89979c11808. Scoped rereview next.

- Task10: fix round2/5 (R3 addressed, zero open in scoped fix review; commits f64168f..f64168f, code not committed yet). Independent spec/codequality APPROVE T10 code scope after2rounds; R1/R2 remain addressed. Partial code/evidence commit next. O1 and personal-finish assistance edge remain T16/finalreview; full human/mobile/online acceptance pending, CP5 notPASS.

- T10 partial code committed f9222ccefde585f2c2aba2c02b4015d0d1b503f7; two fix rounds accepted. Initial fullGo integration/race and final affected fixes PASS,50frontendtests/typecheck/build, actual guest CUA first-input-after-reopen and results evidence. CP5 remains pending; O1 and personal-finish assistance edge T16. Source unchanged8d96921 with original.idea only; no target remotes. T11 ready after preview restart.

- T11 dispatched /root/implement_t11 (gpt-6-astra high,forknone), base530535d670e45abb78a8e0a3470d5677ca6d7f7e. Brief contains exact Duck rules, corrected3s countdown, authoritative60s/hits/rewind and acceptedT10 lifecycle pointers. duck-v1 metadata seam/measured timing architecture must be settled before implementation. Preview API restarted acceptedT10 on8080 exec18562, startup16:00:33local, healthok; Vite4176 unchanged. Root pureguest tab2 returned toworld and marked handoff. This turn completedT10reviewfixes/commit/runtime and startedT11; progress, not impasse.

Ruling (T11 metadata/timing): approve narrow RegisterVersioned(game,factory,RulesMetadata{ID,Version,Checksum}) applying only validated RulesID/RulesVersion/RulesChecksum before factory/persistence; canonical duck-v1 asset owns checksum. Other game metadata/serialized hash shapes remain unchanged, Prepare roster-only, registration frozen before serving, retries reuse frozen start. Timing uses server-selected receiveTime minus conservative recent measured RTT/2 capped150ms; client timestamp is a monotonic/consistency validator, never an arbitrary history selector. Challenges are unpredictable/single-use/bounded/expiring and bound to actor+world+match+admitted connection generation; reset on reconnect/leave/end. No client-reported RTT. Without valid calibration, legal shots use current authoritative time/zero rewind and truthful calibrating indicator; no forced calibration stall. Cooldown uses server receipt time. Preserve bounded rates and full authority fencing. Test actual challenge-response with deterministic50/100/200ms+jitter, stale/replay/nonce/epoch/future/nonfinite/fabricated-time abuse; distinguish synthetic timing from physical network. Cost/limits: owner queue and client processing contribute to observed roundtrip, asymmetry/jitter constrain accuracy; no perfect anti-cheat claim, no new external dependency required.

Ruling (T11 touched integration corrections): targeted socket read confirmed existing WorldSocket onmessage forwards penalty.* but drops race.snapshot/race.finished; guest CUA did not cover this online defect. T11 must add explicit race and Duck event forwarding on existing socket-scoped dispatch, with actual WorldSocket handler regression and preserved stale-socket/world fencing. Generic lobby recordDisconnect currently marks BotAssisted for all games; restrict this presentation flag to race, assert Duck remains false through disconnect and official result, preserve race takeover flag. These narrow corrections are directly in Duck integration seams; keep known O1/personal-finish edges separate. Actual Duck owner+projector tests must prove intentional leave/grace incomplete still yields participant history despite generic all-gone abandonment behavior. Cost: small correction to prior game dispatch/presentation with explicit regression; no assertion of authenticated browser acceptance.

- T11 milestone (preliminary implementer report): engine/measured-clock tests and actual held-owner notification cases PASS; duck-v1 frozen metadata seam,3s countdown, natural60s-before-grace and immutable incomplete results through finalizing implemented. BotAssisted false forDuck and race-only presentation guard. Application/protocol/frontend/actualPGWS integration still underway; no task review/acceptance yet. Root observed new engine/rewind/lobby tests and migration008; no previewrestart.

- T11 actualguest CUA smoke PASS within scope: countdown3, hit100/fast200, miss/keyboard shots, close/reopen score+timercontinuity, FIRST postremount pointershot accepted, finalinput lock, return/newsession reset, consoleclean. Two realsessions; first near-expiry Space attempt explicitly inconclusive and repeated with ampletime. CP6/t11-guest-preview.md and3JPEGs saved. Stale availability/countdown spectatorcopy sent for correction; offline lobbyloading alreadyT15. Frontendfreeze released, APIstillT10.

- T11 broader verification preliminary: npm checks reportedgreen (await exactJS+TScount); fullGo integration exposed2stale assertions (Duck engine availability and +1s before new3s startsAt), worker updatingactual semantics. Root requested one concrete adapter terminal-event check after inspecting handoff seam. Worker CLIguestbrowser attempts encountered missing pinnedbrowser/cachedbinary sandbox; coordinator instructed no further CLI/browser workaround and retains rootCUA as executed guest evidence, authoredE2E asNOT_RUN. No acceptance inferred.

- T11 browserattempt clarification: worker cachedChromiumCLI run had already completed when stopmessage arrived; actual run FAILED whitespacefinalassertion afterguestflow, sourcecorrectionNOT_RERUN. Earliermissingbrowser/sandboxerrors retained. No furtherCLIbrowserruns; rootCUA is separateexecuted evidence, not a way to label failedE2E PASS.

- T11 self-review milestone: worker reproduced missing duck.finished on internal natural finish during ordered resume; adapter now captures terminal transition once with real held-owner regression. Calibrated mapped-future time now strictly rejected (RED→GREEN). Exact npm count53=4JS+49TS, prior suites retained; final typecheck/build reportedPASS. Hero/station/countdowncopy fixed; affected transport rerun/report pending.

- T11 finalpresentation CUA confirms hero/directory/station availability and countdown waitingcopy corrected. Third real zero-shot session naturally finished0/0accuracy0%, consoleclean; CP6finalcountdown/zeroshotJPEG evidence appended. GameplaypriorCua unchanged, online/provider/device limits intact.

- T11 ready for independent review: fullGo integration then final fullGo-race integration/vet/build PASS;53frontendtests/lint/typecheck/build PASS. ActualguestCUA three sessions and finalcopy evidence recorded separately from failedCLI E2E/correctedNOT_RERUN. Staged source/contracts tree56f0a2d714413f2dc4c76b4e16c56c3c80fe6f4c, base530535d670e45abb78a8e0a3470d5677ca6d7f7e; task-11-review-package.diff prepared. No taskcommit orCP6PASS yet.

- T11 independent review dispatched /root/review_t11 (gpt-6-astra high,forknone),44file132389byte package; staged tree56f0a2d714413f2dc4c76b4e16c56c3c80fe6f4c preserved, base530535d670e45abb78a8e0a3470d5677ca6d7f7e. Original implementer complete/available for fixes. RootCP6evidence excluded from reviewdiff, report copied to evidence. This turn implemented/reran meaningful Duck checks, fixed actual terminal event risk, obtained3guestCUA sessions and finalfullverification, then dispatchedreview; progress,notimpasse.

- T11 review preliminary: reviewer identifies missing duck.shot_resolved emission (ack only) and raw15Hz online rendering inconsistent with server-selected hit instant. Concrete200msRTT +55ms aftersnapshotarrival: server validates target155ms beyond snapshot, fastspeed300moves46.5>radius45, so center click can miss. Existing transport test reads authoritative target atrewoundtime and does not exercise deliveredvisualaim. Await fullreport before originalimplementer fixround1. Preserve150ms serverbound; align presentation and test actualsnapshot/aim pipeline.

- T11 initial independent review completed: one Important R1 online target presentation is stale relative to server-selected hit instant; current synthetic transport tests bypass delivered visual aim. Fixround1/5 will use original implementer; staged tree56f0a2d714413f2dc4c76b4e16c56c3c80fe6f4c remains baseline. NoT11commit/acceptance.
- Task11: minor (deferred): R2 specified duck.shot_resolved has no server producer (ACK only); any future producer needs worldId/matchId/seq, noncoalesced exactly-one accepted hit/miss event, no event for rejected input, actual transport+WorldSocket delivery and no doubleapply. ACK and aggregate snapshots preserve current shooter scoring, so reviewer graded Minor. Carry mandatory protocol completion to T16/finalreview before beta; not claimed implemented. Earlier preliminary note treated both as candidate blockers; fullreview establishes onlyR1 as blocking this fixloop.

Ruling (T11 R1 display-time alignment): approve duck.clock_reply ACK returning selected authoritative estimator offsetMs/rttMs/rewindMs/measuredAt/expiresAt, with offset+RTT from same eligible sample used by server rewind and consistent expiry/epoch reset. Add only current-target normalized vx and expiresAt; no seeds/future spawn schedules. Online display time = performance.now()+offsetMs+max(0,rttMs/2-rewindMs), aligning expected server receipt-minus-rewind even beyond300ms RTT cap. Predict existing current targets no more than500ms from snapshot serverTime; hide expired/stale targets with degraded indication, never revive dead/final/countdown state. rAF renders via shared actual OnlineDuckController; local guest remains engine-driven. Deterministic TS actualview/geometry/pointer callback and integration-tagged Go realtransport→Node controller/view bridge use only public snapshots/calibration to choose aim, with bounded process timeout/npm prerequisite; no browser binary or hidden engine-coordinate targeting. Tests include50/100/200+jitter,200/+55center case,320cap, misses/expiry/dead, sampleexpiry/epoch and prediction bound. Cost: explicit public current-motion/timing fields and bounded presentation predictor; no authority/rewind-cap/result-hash change. R2 remains separateT16 protocol obligation.

- T11 fixround1 milestone: actualDuckView RED at200ms/+55ms rendered0.4instead0.4465; currenttargetmotion/selectedcalibrationACK +OnlineDuckController now basicregressionGREEN/typecheckPASS. Online rAF only; guest keeps50ms timer. Root starts focusedguestregression while worker builds actualpublicsnapshot→viewaim→GoWS bridge and latency/dead/expiry cases. Reviewedindexunchanged, no taskacceptance.

- T11 R1 amendedfrontend actualguest regression PASS: countdown, gold500, remountfirstshotfast200, keyboardthirdshot, final700/2of3/66.7%locked afterpostfinishclick, consoleclean. CP6guest narrative andfix1live/finalJPEG saved. Online delivered-snapshot timing bridge stillunderconstruction; no R1reviewacceptance yet.

- T11 fix round1 ready for scoped rereview: actual public-snapshot + calibration ACK through production OnlineDuckController/DuckView drives8 realWS profiles50/100/200/320ms with0/+12ms jitter, exact200/+55 centerhit and150ms cap. Affected Go Duck/application-race/vet,5focused TS/typecheck/build PASS; actualguest CUA preserved. Reviewedindex56f0a2d714413f2dc4c76b4e16c56c3c80fe6f4c unchanged; task-11-fix1.diff includes all5newfiles. R2 remainsT16; noT11commit/CP6PASS.

- Task11: fix round1/5 (R1 addressed, zero new findings; scoped spec/code readiness accepted). Minor R2 remains T16 protocol obligation. Partial code/evidence commit follows; external acceptance pending, no taskdone/CP6PASS.

- T11 partial code/evidence committed 607141a58593f3abf473edb369148dcd59fed289; accepted after1fixround. API8080 restarted from acceptedT11 at17:03:00local, exec48896; /health/live returned statusok. No authenticated browser acceptance claimed. T12 next; CP1 remains highestPASS.

- T12 dispatched /root/implement_t12 (gpt-6-astra high,forknone), baseca12faeaba0cb5f5d3c350e5010853bfb6151e74. AcceptedT11 transactional Duck projector, participant privacy and participant-only status notification seams in brief. Worker owns podium/PB/champion and tests; root owns docs/runtime/review. API8080 remains acceptedT11 exec48896, Vite4176. T11 implemented_pending_external_verification, CP1 highestPASS.

Ruling (T12 deletion/privacy/notification architecture): approved transactional RemoveAccount(ctx,tx,userID) service seam with persistent leaderboard exclusion, PB removal/recompute and separate administrative audit. Existing users schema has no deleted/status marker; T14 owns full endpoint/session/history lifecycle. Projection and removal must share account-level serialization before ordered board locks/discovery, including a concurrent first result in a previously unseen partition; test both orderings, rollback/replay. Deletion is not a new champion victory, historical champion presentation anonymizes removed account. World-scoped approved Duck summary deep link remains active-membership gated; History.Get remains participant-private. Board key game+rulesID+version. World-only leaderboard.changed after confirmed commit invalidates allmembers, never broadens private result payload; reconnect fetch restores missed events. Cost: narrow durable exclusion/lock seam, T14 must compose it correctly; no new deletion UI/endpoint prematurely.

- T12 preliminary milestone: implementer reports focused realPG leaderboard suitePASS2.368s, uniquePB/top3/rationalties/nonimprovingPB/firstchampion/sameowner, worldprivacy, rowlock/rollback/concurrentfinalization/retry/partitionisolation and accountremoval lock/replay. UI and actual HTTP+WS integration still underway; not final verification/review/acceptance.

Ruling (T12 existing completed results): no undocumented leaderboard launch cutoff. Backfill already-completed eligible Duck history before recovery/worker/serving startup, ordered FinishedAt/resultID, bounded/cancellable and idempotent. Reuse ranking/exclusion/account+board locking projection; do not replay duck.Project duplicate detail INSERT or mutate immutable history. Durable projection receipts/equivalent prevent restart/retry duplicate champion changes and compose with new pending finalizations. Test pre009 completed result→backfill/startup, restart, excluded/incomplete records and rollback. Cost: migration projection path necessary for existing data correctness, no live fixture data or rollout/deploy action.

## User clarification — web including mobile
User replied to device-availability question: "web version aja semua termasuk mobile". Product target is responsive browser web across desktop/mobile; no native app deliverable or native app tooling. Continue responsive viewport/touch/browser testing for the web build. Reply does not identify a physical device/OS/browser or supply measured physical-device evidence; do not claim those measurements. No repeat device question now. Preserve honest distinction between browser viewport checks and measured physical mobile performance.

- T12 final verification attempt: default-parallel fullGo-race integration had T12 leaderboard22.427s/application38.627sPASS but existing chat TestChatPersistenceModerationIsolationAndResync failed atline84 sixth-message ErrRate assertion (test2.54s). Production SQL tokenbucket refills1token/sec; fixture assumes six sequential persisted calls stay below refill. No chat changes authorized. Ruling: one focused diagnostic, then full -race -tags=integration -p1 iffocusedgreen; preservefailedparalleloutput and explicitly label serialPASS ifobtained. Furtherfailure→diagnose, no repeatedretryuntilgreen. T16 harnessreliability obligation to remove implicitwalltimeassumption withoutweakeningproductionlimits.

- T12 ready for independent review: serial fullGo -race integration -p1-count1PASS, Go vet/build and60frontendtests/lint/typecheck/buildPASS. Default-parallel chatfailure andfocuseddiagnostic preserved, T16harnessobligation. DONE_WITH_CONCERNS reviewed: concerns are known externalacceptance/T14orchestration and documentedtesttiming; no unresolvedcodecorrectnessclaim fromworker. Source/contracts staged treeadb420251e5f085ec2a6e763e8bd73b6b322645e, baseca12faeaba0cb5f5d3c350e5010853bfb6151e74; task-12-review-package.diff. NoT12commit/CP6PASS.

- T12 review preliminary: ChampionBoard access_denied clears state without invalidating pending board/detail success handlers; late response can restore private display. A later401/404 refresh also leaves previously loaded detail visible. Await complete reviewer report before fixround1; stagedtreeadb420251e5f085ec2a6e763e8bd73b6b322645e unchanged. Core ranking/transaction/backfill preliminarycompliant; no acceptance yet.

- T12 initial review complete: Important R1 production board/detail access-revocation race; fixround1/5 dispatched originalimplement_t12, reviewedstagedtreeadb420251e5f085ec2a6e763e8bd73b6b322645e preserved. Required realcomponent deferred board/detail denial fences, loaded-detail deniedrefresh and differentworld denial scope. NoT12commit/acceptance.

- Task12: minor (deferred): R2 missing historical Duckdetail is coalesced into score0/hits0/shots0/completefalse by leaderboard/query.go, so latestsession UI fabricates zero-point incomplete facts. Beforebeta T16/finalreview must expose unavailable/unvalidated detail (nullable stats/availability) with no invented outcome, preserve immutable history and boardexclusion; extend existing missingdetailbackfillcase and actualview assertion. See task-12-review.md R2.

Ruling (T12 R1 regression harness): approve small test-only Reacthook lifecycle harness running exported real ChampionBoard component/effects/window listeners and controlled fetch promises, with real controller. Must honor cleanup/dependency replacement and test denial clears board+detail, latecontinuations cannotrestore, explicitretry/validmatchingonline rechecks freshgeneration, unrelatedworldsocketdenial doesnotclear anotherworlddeep link. No DOM/browser dependency install or credential workflow; label deterministicNodecomponent evidence distinctly from browser acceptance.

- T12 fixround1 ready: shared actualcomponent lifetime cancels/clears board+detail on socket/HTTPdenial, generationfenceslatecallbacks, scopedworlddenial and explicitretry/reconnect. Four production lifecycle regressions RED then8focusedtestsGREEN, typecheck/buildPASS. No backend/R2 change. DONE_WITH_CONCERNS retains only ledgeredR2/externalacceptance; fixdiff includes newcomponent-lifecycle.ts, indexunchanged. Scopedrereviewnext.

- Task12: fix round1/5 (R1 addressed, zero newfindings; scoped spec/code approved). R2 remainsT16 beforebeta; partial code/evidence commit next. No taskdone/CP6PASS; browseracceptance stilldeferred.

- T12 partial code/evidence committed bb1b4e88c7079ad99442e91b98089a0fe286f006; R1accepted, R2T16 retained. API8080 restarted acceptedT12 at17:39:54local exec78528; healthok. Cachedwhitespace check exposed preexistingnewCSS trailing space and blank capturedloglines duringcommit; shell continued tocommit. Implementer removed onlyCSStrailing space; root normalized evidenceblanklines, diff--ignore-space-at-eol confirms noCSSbehaviorchange. Cleanup recorded in followupdocscommit; no tests repeated for whitespace. T13next, noCP6PASS.

- T13 dispatched /root/implement_t13 (gpt-6-astra high,forknone), base5ac08783d06aa8202c664f6a01c311bc628a94b1. Brief includes exactvoice/token/revocation/tablechat spec, webmobile-only, noCloudstaging and pendingrealprovideracceptance. Architecture/lifecycle proposal requested beforeedits. Root ownsCUA/runtime/review; API8080 acceptedT12 exec78528, Vite4176. T12 acceptedpartial, CP1highestPASS.

Ruling (T13 admission/provider architecture): use authenticated exact-origin WebSocket voicecommands through existing Hub.Execute/WithJoined/ratevalidation; HTTP only readinessGET and signedproviderwebhook. voice.Service owns ephemeraladmission/tablechat, opaqueperjoinidentity and8seat/oneuser bound. Synchronous localinvalidations compose existing SeatChanged/lobbyactivity with no providerI/O underowner/hub/voicelocks; backgroundworker validates outside locks, generationfencesresults, retries explicitcutoffremovals and reconciles only namespacedmanagedrooms. Pendingremoval globallydegrades newvoiceadmission/token issuance; game/worldchatremainavailable. Mustfence concurrentgameentry vs latevoicejoin/token, not relyonlypolling. Shared independentversioned tablezonecontract linked exactmapID/version/checksum approved atfront8,33 andcorner12,33 radius1.4, samefrontend/backend geometry withbounds/reachability/nonoverlap/parity; no navmap/resultchecksumrewrite. Proposed now+2s revokecutoff conditionalonofficialSDK/docsunits/semantics, actualCloudinitial+refreshreplay remainsunverified.

- T13 dependencyfinding: cachedSDK2.16.1/protocol1.45.x lack RevokeTokenTs; newerSDK2.19 requiresGo1.26.8 vs current1.26.2. Worker investigatingpinnedSDK2.16.1+newprotocol; mustcompileandcapture actualSDKRemoveParticipant wirefield via localfakeHTTP/Twirp, notwrappermockonly. No defaultcutofffallback or systemGo upgrade. Raiseexactminimum/project-scopedoption ifneeded. npm officiallivekit-client2.16.1 installedbyworker; report downloadfailures/scopedapprovals.

Ruling (T13 crash-safe revocation): minimaldurablejournal approved because roomlisting cannotdiscover minted-but-neverconnected token afterrestart. Persistadmissionidentity/managedroom/revocationstate beforetokenmint; never rawtoken/audio/chat. Startuploadsjournalasrevoked and gatesreadiness untilcleanup; keep pendingonproviderfailure. Journalmustsurviveaccountdeletion untilconfirmedremoval. SQLboundedunderadmissionlease butoutsidevoice/hub/ownerlocks, reservegeneration→persist→recheckbeforemint; cancellation duringwrite andcommitunknownfailclosed, no unjournaledtoken. Tests minted-neverconnectedrestart, logout/gameentryduringpersist, DBfailure/unknowncommit/removalretry. Seats/tablechatremain ephemeral. Cost: narrowdurablecleanup state forreplayboundary; notfullmediastaterecovery/Cloudacceptance.

Ruling (T13 Go SDK compatibility): officialSDK2.16.1 and2.18.0 both fail with protocol1.53.1 (SIP TransferSIPParticipant returntype changed). RevokeTokenTs existsin1.53.1, Unixseconds/nbf-before semantics; actualwiretestpendingcompile. Approve task-scoped officialGo1.26.8 runtime underproject.cache withSDK2.19 compatibleofficialprotocolgraph; stopguessingearlierversionpairs. Preserve system/opt/homebrew/bin/go1.26.2 andglobalgoenv/PATH. Verifyofficialchecksum/toolchainmodule, explicitcommands/reproduciblepath/minimumversion; updateprojectgo.mod/tooling/CIonlyasneeded. Scopeddownloadtoolapprovalifneeded, nofurtherhumanconfirmation forroutineprojectdependency. Cost: raisedprojectpatchminimum/cacheartifact; no systemupgrade/providerdeployment. Preservebothfailedcompileattempts inreport.

- T13 preliminaryguestCUA: two8seat tables, unavailable/guestgate, disabledjoin, noobservedmicprompt, cleanconsole. APIstillT12; notT13provider/readinessacceptance. Rawcoordinates/repetitivependingcopy sentforcleanup. CP7initialscreenshot/narrative saved. SDK2.19+protocol1.53.1 nowresolving with projectGo1.26.8; compile/wiretestpending.

- T13 focusedvoice milestone: projectGo1.26.8+SDK2.19.0/protocol1.53.1 compiled, internal/voice testsPASS0.627s includingactualSDKprotobufHTTP RevokeTokenTs Unixsecondscapture andsigned60smic-onlyJWTclaims. Initialcapture incorrectlyassumedJSON thenfixeddecodeprotobuf; firsthttptest sandboxdenial followedscopedapprovedrun. NoCloudreplay/audioacceptance inferred. Fullintegration/UIverification pending.

- T13 actualapplication/PGjournal focusedmilestonePASS0.772s: authoritativewalkout/gameentry/ban/epochresume/logout release, outsider/tokendenial, pre-mintjournal andconfirmedremovaldelete. Fixtureinitiallyraced lobbyreturn→testteleport; separatedsyntheticaccountforresume/logout, noproductionlobbychange. FrontendselfreviewRED→GREEN: audiofailurepreservestablechat; joinACKtimeoutcleanspossiblyallocatedseat. Tablechat5minexpiryreadmissionbeforeworkercycleRED→GREEN. FullserialGo-race/vet/build/frontendcheckspending; noCloud/device/browseracceptance.

- T13 readyreview: fullGo-raceintegration serial-p1PASS withprojectGo1.26.8, vet/buildPASS aftertestliteralstylefix;71webtests/lint/typecheck/buildPASS. ActualSDKprotobufrevokecutoff+60sJWTclaims, actualapplication/PGjournal tests andguestunavailableCUA distinctfromunexecutedCloud/audio/replay/authbrowser/device. LastmicenumregressionRED→GREEN; candidatefrozen. DONE_WITH_CONCERNS reviewed: pendingexternalacceptance anddocumentedoperationalbounds(singleinstance/partialconfigfailfast/4096dedupe), reviewtoassessscope. Source/deps/contracts/runbook staged treeb2f5394c98cd61ecdc6f2d13645798932e6cf88d, base5ac08783d06aa8202c664f6a01c311bc628a94b1; task-13-review-package.diff. NoT13commit/CP7PASS.

- T13 reviewpreliminary: checking spectatorgameentry voicefence (spectatorsnotGameBusy),4096tablechatdedupehardstopuntil5minempty, andfrontendstale mic/chatcompletions. Fullreportpending; nofixdispatch/adjudicationyet, stagedtreeb2f5394c98cd61ecdc6f2d13645798932e6cf88d preserved.

- T13 reviewpreliminaryadditional: unknownmanagedroomparticipant discardedwhenjournal.Save fails (includingcommitunknown); ifparticipantdisappearsbeforenextlist, nextcyclecanmarkhealthywithout explicitremoval/startupLoadone-time. Fullreport/fixrequirements pending; no prematureacceptance orfixdispatch.

- T13 initialreviewcomplete:4Important R1unknownidentityjournalfailureforgetsrevocation, R2spectatorgameentryvoicefence, R3unapproved4096tablechatlifetimequota, R4lateACKerasesedited/reattacheddraft. Fixround1/5 dispatched originalimplement_t13; stagedtreeb2f5394c98cd61ecdc6f2d13645798932e6cf88d preserved. Require actualpersist-then-error journalcase andrealVoicePanel lifecycletests. R3boundedprotocolproposalbeforecontractedit. NoT13acceptance/commit/CP7PASS.

Ruling (T13 R3 continued chat bounded dedupe): replace unapproved4096 lifetimequota with canonical clientMessageId `<currentAdmissionId>:<positive JS-safe integer>`, peradmission highwater and32recentACKs, latest200tablemessages. Exactnextseq creates; recentretry sameACK; evicted/oldseq<=highwater returnsretry-expired withoutrepost; gaps/noncanonical/range/wrongprefixreject. voice.state nextMessageSeq syncmonotonically onlycurrentadmission; stalestate cannotrewindcounter. Singlepending keepsID/textacrossretry whileeditednextdraftsurvives. Retryexpired isterminaloldsubmission+resync/allownewuserintent, neverautorepostoldtext undernewID orrequireemptytable/rejoin. Clear oldadmissionACK/chatpayload onrevocation/5minemptycleanup; durablejournalidentityonly. Testcontinued>4096/recent+evictedretry/gaps/prefix/range/stalestate/recovery. Cost: explicitnewvoice-onlymessageIDsemantic inunreleasedT13contract, boundedmemorywithout availabilitycutoff.

- T13 fixround1 milestone: R1 reportedbehavioralRED→GREEN for precommitfailedsave/persist-then-error, disappearingparticipant andadmission/tokengating untilexplicitcleanup; R2 actualapplicationRED→GREEN voicefirstspectatorrevoke/spectatorfirstadmission+tokendenial, Busyplayer-onlypreserved. R3highwater32ACK andR4actualcomponenttests underway. Indexunchanged; not yetfrozen/rereviewready.

- T13 fixround1 coverage correction: root ran actual tsc --listFilesOnly; prior voice coverage included only zones.ts, not VoicePanel/session/microphone/tests. Worker added explicit production TS/TSX/test/E2E includes and actual typecheckPASS with allvoicefileslisted. EarlierPASS means prior narrowentryset, notfullvoicecoverage. Duckproduction local/session/online/Duck.tsx was independentlyconfirmedalreadyincluded (Ducktestfilesnotlisted). T16 shouldauditentrycoveragefornewfeatures ratherthan assume commandPASS impliesallfiles.
- T13 fixround1 preliminaryfinalchecks: affectedvoice/application/realtime/lobby-raceintegrationPASS, affectedGovatPASS; frontend75tests(4JS+71TS),lint/buildPASS; strengthened initialjournaluncertainwrite bothpersisted/notpersisted focusedracePASS. Reportfreeze pending.

## T13 fix round1 verification
- Unknown-identity journal failures remain pending through explicit cleanup; spectator entry releases/excludes voice; continued chat uses per-admission sequences with32recentACKs/200visiblemessages; lateACK preserves edited drafts and fences obsolete attachments.
- AffectedGo voice/application/realtime/lobby race integration and vetPASS;75frontendtests,lint/buildPASS. Expanded actualvoice production/tests typecheckPASS. Earlier typecheck covered geometry only; the report preserves this correction.
- Scoped independent rereview pending. No T13commit, taskdone orCP7PASS. Cloud/audio/replay/authenticatedbrowser acceptance remains pending.

- T13 scoped rereview: R1–R4 addressed, zero new findings; code approved for integration. Partial C13 commit next, no taskdone/CP7PASS.

## T13 partial commit and preview
- Code/evidence committed `b6e74fbf12ff0a2913e09888b81c1e3d767566b6` after independent review and one fix round. No T13done/CP7PASS.
- API8080 restarted from acceptedT13 at18:52:36local using projectGo1.26.8, dedicatedDB. Initial sandboxDB/loopback restrictions retained; approved scoped retry started successfully. `/health/live` returnedstatusok; `/api/v1/voice/status` returnedunavailable withtwozonecontract, as expected withoutLiveKit. No authenticated/browser/provider acceptance inferred.
- T14 begins next with accepted T12account-removal and T13durablemedia-cleanup seams; webincludingmobile, no nativeapp.

## T14 candidate verification
- Guest IndexedDB history/settings, account lifecycle and identity redaction are implemented and frozen for review. Full serial Go race/integration, vet/build and frontend 83 tests (4 JS + 79 TS), lint/typecheck/build passed in the retained reports.
- Actual GuestHistory transaction tests use pinned dev-only fake-indexeddb: 120 concurrent appends retain newest 100, duplicate immutability, reopen, reset generation and rollback/error paths. This is distinct from real-browser 100-record acceptance.
- Real guest browser: three games finished, history and settings survived reload, reopened Duck did not duplicate, unfinished reload did not invent a result. User approved reset of these three synthetic entries/preferences; actual reset and reload restored empty history/default settings. Console warning/error query empty.
- Independent review pending. Authenticated browser acceptance remains deferred; no T14 commit or CP7 PASS.

## T14 review and fix round 1
- Independent review: 0 Critical, 4 Important, 1 Minor. Important repairs cover REST membership/social writers racing account deletion; a publicly claimable tombstone username blocking deletion; account/device preference mixing; stale Settings callbacks and HTTP cookie expiry overwriting a newer session.
- Original implementer owns fix round1. Reviewed staged baseline preserved for a scoped re-review. No T14 commit or CP7 PASS.
- Minor M1 is assigned to T16 before beta: race/Duck history finish timestamps omit the three-second countdown. Existing browser evidence proves observed persistence, not exact wall-clock finish accuracy.

## T14 scoped review after round 1
- Original I1–I4 addressed, with two new Important Motion regressions: same-session preference updates emit a session teardown event; and a stale Motion operation can POST after its deferred CSRF lookup under a newer account session.
- Original implementer owns round2. Backend repairs accepted within the scoped review; frontend repairs and re-review remain. No T14 commit or CP7 PASS.

## User-requested stop and commit — 27 October 2026

User requested ending this conversation, committing current work, and a handoff for a new conversation. Goal is paused; no T15/T16 dispatch or further feature implementation. T14 round2 is implemented and frontend checks rerun successfully (97 tests, typecheck, lint, build). Independent round2 re-review has NOT_RUN. The WIP commit intentionally preserves this unfinished review gate; it does not approve T14 or CP7. CP1 remains the only PASS; T1/T2 remain the only fully done tasks.

Approved reset of the three synthetic guest results, name and preferences was completed and verified after reload. Account browser acceptance remains deferred; LiveKit staging is unavailable. No remote/push/deployment. See `docs/HANDOFF.md` for restart order and tracked T15/T16 briefs.

- Pause snapshot committed: `67371621b1d650fcd16533a179993905bd7da79c` (`wip: save T14 settings and guest persistence before handoff`). All T14 code, evidence, and portable handoff briefs are tracked. Independent round2 re-review remains pending.


Publication note (28 October 2026): the obsolete `t1-desktop.png` and `t1-mobile.png` portfolio screenshots mentioned above are retained only in local inherited Git history. They are omitted from the game-only public snapshot. Later game-only CP1 screenshots remain available; historical test claims are unchanged.


## Kampung Senja integration — 28 October 2026

User approved the outdoor design and requested fullscreen production integration, semi-English copy, preserved games/multiplayer, local commit with plan/evidence, then a public personal GitHub repo on main. Implemented the shared top-down map, real controller/HUD and retained all feature mounts. Fresh scoped review found three regressions and accepted their repairs.

Verification: 103 frontend tests, typecheck, syntax lint, production build; full Go race/integration across20 packages and242 primary tests, vet/build; actual guest penalty/Race/Duck completions and history after reload;390/768 responsive previews. Full evidence is in `docs/checkpoints/evidence/CP8/kampung-senja-integration.md`. This does not approve T14/T16 or deferred account/LiveKit/physical-device acceptance. CP1 remains the only PASS.

Public main will use a clean snapshot to omit inherited CV/PDF history. User selected git-patra; clean local main snapshot is prepared, but remote creation/push awaits that account’s authentication. Existing CLI credentials belong to patra-digital; in-app GitHub is signed out. No deployment.
