# Handoff — Patra World Social

Updated 28 October 2026. User resumed with an approved Gather-like outdoor village redesign, then authorized implementation, tests, local commit including the plan, and a public personal GitHub repository on `main`. This document records context and remaining acceptance limits.

## Project and non-negotiable scope

- Repo `/Users/patra/Project/patra-world-social`, active branch `main`, tracking `origin/main`. Public repository: https://github.com/git-patra/patra-world-social. Public history starts from clean game snapshot `0adf204`; inherited development/CV history remains only on local `feat/social-world`.
- Source `/Users/patra/Project/patra-world` must remain unchanged, baseline `8d969212d91804e06225f5b1a75eed49e8c25307`; its pre-existing untracked `.idea/` is not ours.
- Go backend; responsive web for desktop and mobile, no native app. Preserve autumn/Halloween world theme. Remove CV/portfolio content; active game tree cleanup already completed.
- Private worlds through invitations; guests only solo; username/password accounts for multiplayer, friends, history. Race five laps/five slots with bots, penalty three-second choice, Duck Arcade/top three, warkop mic/chat.
- Local implementation, commits and public personal GitHub publication on `main` are authorized by the latest user request. No public deployment, production changes, paid services or committed secrets.
- User deferred authenticated browser workflows. Do not retry registration/password/recovery through another route. LiveKit staging not available; continue independent work after explicit resume. User specified web including mobile, not a claim that physical mobile verification has passed.

WIP snapshot commit: `67371621b1d650fcd16533a179993905bd7da79c`. A subsequent documentation-only commit records this SHA; inspect `git log -2 --oneline` for the final tip.

## Latest village integration

- Approved outdoor Kampung Senja, fullscreen canvas without the legacy sidebar, semi-English HUD, Map and activity menu. Existing game, account, social, history and voice mounts remain.
- Shared `kampung-senja-v1` navigation and voice geometry are mirrored into Go; private-world movement stays server-authoritative.
- Plan: [Kampung Senja integration](superpowers/plans/2026-10-28-kampung-senja-integration.md). Evidence/review: [integration verification](checkpoints/evidence/CP8/kampung-senja-integration.md), [fresh review](checkpoints/evidence/CP8/kampung-review.md).
- Latest local frontend is `http://127.0.0.1:4176/`; API `127.0.0.1:8080`, dedicated PostgreSQL `127.0.0.1:55432`. Design-only prototype remains on4180 for reference.
- The following T14 snapshot/acceptance history is retained; the new scoped review does not approve T14/T16 or external browser/LiveKit acceptance.

## Rencana adjustment baru — belum diimplementasikan

28 Oktober 2026, setelah publikasi GitHub: user meminta **plan dulu** untuk pingpong, capit boneka, free ride mobil/sepeda, area steril saat race/penalti, bola bebas dan penonton, Blaster top-down, fishing tanpa history, minimap interaktif, SFX, dan popup terpusat. User memilih shooting solo/co-op melawan bot. Tidak ada implementasi fitur baru pada tahap perencanaan ini.

- [Design proposal](superpowers/specs/2026-10-28-world-activities-design.md)
- [Implementation plan](superpowers/plans/2026-10-28-world-activities-adjustment.md)
- Batas entitas, frame time, network, audio dan memory adalah target yang akan diukur, bukan klaim performa yang sudah lulus. Angka rules baru adalah usulan untuk direview.
- Mulai implementasi hanya setelah user meminta lanjut. Plan baru tidak menghapus gate T14/T15/T16 maupun acceptance yang tertunda di bawah.

## Current implementation and acceptance

Authoritative ledger: [PROGRESS.md](PROGRESS.md). Plan/spec are tracked under `docs/superpowers/`.

- T1/T2 done; **CP1 is the only PASS**. T3–T13 code reviewed and committed as partial progress, external acceptance still pending. T14 is implemented and tested but its latest repair needs independent re-review. T15 fullscreen village integration completed in the current change; remaining T15 acceptance and T16 stay open.
- Implemented: Go/PostgreSQL accounts and recovery, private worlds/invites, authoritative realtime movement, friends/chat/moderation, lobby/match ownership/history, penalty, race, Duck Arcade, Duck leaderboard, warkop admission/table chat and LiveKit integration.
- T13 accepted code commit `b6e74fbf12ff0a2913e09888b81c1e3d767566b6`; prior ledger commit `658bd09271e31d6b8f86333fb7d407a0d4aaea21`.
- T14 snapshot adds actual IndexedDB guest history (dedup, oldest 100 retention), local preferences/reset, account settings/password/recovery rotation/deletion, account deletion fencing and identity redaction. It includes migrations 011/012. This snapshot is WIP pending review, not an accepted C14 checkpoint.
- Guest browser evidence covers real completion of all three games, history/settings persistence after reload, no duplicate finished result, unfinished-game reload, and user-approved reset of the three synthetic results/name/preferences. Reset was completed and verified. Do not repeat permission for that completed action.
- Current coordinator verification before pause: **4 JS + 93 TS tests passed**, typecheck, JS syntax lint and production build passed (254 ms). T14 initial full Go race/integration/vet/build and round1 affected checks are recorded in evidence; round2 changes only frontend and no new backend run is claimed.
- Authenticated browser acceptance, real-browser 100-record boundary, Cloud voice/audio/token replay, and final beta/load/soak acceptance remain NOT_RUN. Node tests use actual GuestHistory transaction code with fake-indexeddb; they do not substitute browser acceptance.

## Pending independent task: finish T14 review

Read:

1. `docs/checkpoints/evidence/CP7/t14-verification.md` (initial and both repairs, commands, outcomes, limits).
2. `docs/checkpoints/evidence/CP7/t14-review.md` and `t14-rereview1.md`.
3. `docs/handoff/t14-round2-review.diff` — only six frontend files changed after reviewed round1.
4. `docs/checkpoints/evidence/CP7/t14-guest-preview.md` for actual browser/reset evidence.

Round1 resolved four original Important findings: concurrent REST writers after deletion, claimable tombstone username, account/device preferences mixing, stale Settings callbacks and response-cookie expiry races. Backend round1 was accepted by the reviewer.

Round1 review found two new Important defects, now repaired but **not independently re-reviewed**:

- R1: profile/Motion updates emitted the singleton auth event and tore down an active solo game. Bridge now emits only on auth generation transitions. Regression uses real singleton, GuestSession, game engine and the actual app cleanup helper.
- R2: Motion waiting for CSRF could POST after account/session switch. Generation-owned AbortController now reaches the real CSRF/POST helper; explicit post-CSRF cancellation check prevents dispatch; stale response guard remains. Tests cover new account, same-account re-login, and already-dispatched POST cancellation/late result.

Old staged round1 tree object: `5b6a88ea073aabb0c55e65e49d519bc7b2878784`; original T14 candidate tree `12e80daddd7f66540037528848eceeb44945e18a`. These may only exist locally; use the tracked round2 diff as portable review input. Current WIP commit contains all repairs; there is no longer a staged/unstaged baseline separation. Never restore these trees over current work.

Prior agents `/root/implement_t14`, `/root/review_t14` completed and are idle; their handles need not exist in a new conversation. Prefer fresh scoped reviewer if unavailable. User's goal authorized subagents/review, one coordinator for integration and commits. No broad implementation restart or repeating already-passed suites without a new reason. If review finds a defect, fix with focused regression and re-review. Preserve the pending external gates. After acceptance, update the ledger before moving on.

## T15 and T16

Full tracked dispatch briefs: [T15](handoff/T15-brief.md), [T16](handoff/T16-brief.md). They were prepared before pause; reconcile stale base SHA/status against current Git and this handoff before dispatch. Their `.superpowers` paths are local scratch references, not required portable dependencies.

T15: map district/art/camera/avatar polish, static cache/culling, accessibility, touch/focus/reduced motion and responsive layouts (390×844, 768×1024, 1440×900). Preserve three guest games/history/preferences. Fix endless offline availability loading, penalty leading-zero label, old CONNECT/P facade, truthful online Reset feedback, panel clipping. Map geometry changes require coordinated frontend/backend map/table identity/version/checksum updates; avoid gratuitous geometry changes. Preserve immutable historical match hashes.

T16: reliability fixes, 60 clients/3 worlds load, actual 60-minute soak evidence, backup/restore, local deployment/rollback configuration/docs, final whole-branch review. No public deployment. Carry these issues explicitly:

- Generic presence coalescing loses timely resume/new-disconnect ordering; preserve ordered transitions and ensure late first resume cannot revive.
- Race personally finished player can later be wrongly labelled bot-assisted when others continue; preserve actual prior assistance.
- Missing `duck.shot_resolved` producer: exactly one accepted hit/miss event, no rejected-shot events; actual transport/WorldSocket and ACK dedup.
- T8 uncertain initial Persist/startup enumeration may leave ownerless running claim; test actual isolated PostgreSQL startup.
- T6 authored browser source uses fill rather than actual key events and weak offline/reconnect assertions; repair before an authorized acceptance run.
- T12 absent historical Duck details appear as fake 0/0/false; use an explicit unavailable projection, immutable stored history unchanged.
- Existing sixth-message chat rate test depends on wall time and can flake under parallel race tests; deterministic fixture, do not weaken production limiter.
- T14 M1 minor: local completion timestamp derived from creation+duration omits race/Duck three-second countdown; use actual monotonic finish offset with deterministic test, never persist active clocks.
- Minor readability/dense Race TS cleanup; ensure production and test TypeScript really included in typecheck.
- Unconfirmed hypothesis for final review: initial `/me` resolving anonymous after a guest game has begun may increment auth generation and clear the game. Reproduce before claiming a bug; not part of R1/R2 review findings.

## Local runtime and secrets

Last verified API8080 is **accepted T13**, not the T14 WIP source. Vite4176 serves working frontend; URLs alone do not prove backend version or acceptance. No restart was performed at pause. Recheck listeners/processes before any action in a new conversation; old exec session IDs are not portable.

- Preview: `http://127.0.0.1:4176/`; Go API loopback8080; Vite proxies HTTP/WS `/api`.
- Dedicated Colima profile/context and PostgreSQL container `patra-world-social` / `patra-world-social-postgres`, PostgreSQL18.6 at loopback55432. Never alter the default profile.
- Database env is private ignored `.cache/database.env`, mode0600. Source quietly, never print. Nothing secret should enter Git or chat.
- Project Go1.26.8 binary: `/Users/patra/Project/patra-world-social/.cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go`. Set `GOTOOLCHAIN=local` and project `.cache/go-build`. System Go remains1.26.2; do not silently use it.
- For API after T14 review: load private DB env; `AUTH_ORIGIN=http://127.0.0.1:4176`, `AUTH_DEV_LOOPBACK=1`, `SERVER_ADDR=127.0.0.1:8080`, run `go run ./cmd/server` from `apps/server` using project toolchain. Stop only the identified project API listener. Startup applies migrations. Check public health and unconfigured voice status without account credentials.
- `npm ci` / `make dev` for frontend; current install/build already exists. Browser work through permitted browser tools; no credential workflows or storage injection to bypass deferred acceptance.

## Prompt for a new conversation

```text
Lanjutkan Patra World Social di /Users/patra/Project/patra-world-social, branch feat/social-world, dari commit dan working tree yang ada; jangan mengulang clone atau overwrite.

Baca docs/HANDOFF.md, docs/PROGRESS.md, spec/plan di docs/superpowers, dan bukti T14 di docs/checkpoints/evidence/CP7. Mulai dengan independent scoped re-review perbaikan T14 round2 berdasarkan docs/handoff/t14-round2-review.diff; perbaiki temuan jika ada, lalu lanjutkan T15 dan T16 sesuai briefs di docs/handoff. Gunakan subagent untuk implementasi/review dengan satu coordinator untuk integrasi dan commit.

Backend Go, produk web desktop/mobile, tema Patra World, private invite worlds, guest solo, tanpa CV. Uji browser akun tetap ditunda; LiveKit staging belum tersedia. Jangan menganggap mock/unit test sebagai acceptance browser/voice. Catat SHA nyata, hasil tes dan batas bukti. Commit lokal per tahap; jangan remote/push/deploy publik atau ubah repo sumber. CP1 saja yang sudah PASS; jangan mengklaim beta selesai sebelum seluruh acceptance wajib terpenuhi.
```
