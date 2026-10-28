# Patra World Social Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Menurunkan `patra-world` menjadi repo baru untuk private social world, dengan guest solo, akun username/password, tiga mini-game, history, dan warkop voice/chat.

**Architecture:** Frontend Vite/React/Canvas2D mempertahankan visual existing. Satu backend Go memiliki HTTP API, authoritative WebSocket world/game, bot dan persistence PostgreSQL; WebRTC audio memakai layanan media terpisah. Guest menggunakan simulasi lokal yang diuji dengan fixtures aturan yang sama.

**Tech Stack:** Go, net/http, coder/websocket, pgx, sqlc, PostgreSQL; Vite, React, TypeScript bertahap, Canvas2D, Tailwind/shadcn; LiveKit JS/Go SDK sebagai usulan media; Go testing, Vitest, Playwright.

**Spec:** [Rancangan produk dan arsitektur](../specs/2026-10-26-patra-world-social-design.md).

**Status:** Eksekusi disetujui melalui goal pada 26 Oktober 2026. Repo lokal sudah dibuat; progres dan bukti terbaru ada pada `docs/PROGRESS.md`. Checkbox hanya ditandai setelah verifikasi task, bukan karena perintahnya tercantum.

**Peta eksekusi:** Bagian F–J membagi T1–T16 ke CP1–CP8, menentukan batas commit, bukti kelulusan, dan cara resume. [Progress awal](2026-10-26-patra-world-social-progress.md) mencatat posisi sekarang. [Prompt `/goal`](2026-10-26-patra-world-social-goal.txt) disiapkan untuk dikirim pengguna; penulisan prompt ini tidak menjalankan goal.

## Global Constraints

- Backend aplikasi **Golang**; Node hanya tooling frontend.
- Produk hanya world game/social: tanpa CV, biodata karier, timeline kerja, tautan LinkedIn, PDF resume, showcase perusahaan, atau promosi bisnis. Bangunan lama diberi fungsi/nama lokasi desa; pertahankan art. Fitur yang belum tersedia ditandai jelas, tanpa simulasi seolah multiplayer aktif.
- Multiplayer **dunia privat lewat undangan**; guest hanya solo lokal.
- Repo tujuan usulan `/Users/patra/Project/patra-world-social`; sumber `/Users/patra/Project/patra-world` dipertahankan.
- Basis source yang diperiksa: commit `8d969212d91804e06225f5b1a75eed49e8c25307`. Revalidasi perubahan source saat mulai eksekusi, jangan menimpa perubahan lokal.
- Tema isometrik autumn/Halloween existing dipertahankan.
- Race **5 lap**, **5 slot total termasuk bot**, minimal satu manusia untuk Start.
- Penalti **4 zona**, batas pilihan **3.000 ms**, pilihan kosong random server.
- Duck history, podium **3 akun unik**, dan riwayat pergantian champion.
- World beta **20 online/50 anggota**, dua meja voice **8 peserta/meja**; nilai ini usulan spec.
- Map/rules/protocol version dicatat di setiap hasil. Guest tidak boleh mengunggah skor ke leaderboard resmi.
- UI/background/reduced-motion tidak mengubah jam, posisi sah, atau hasil server.
- Semua keputusan default yang belum dipilih user tetap diberi label usulan dalam spec.

## Review Focus

1. Start, invite redeem, slot join, finalisasi dan update champion serentak tidak boleh menggandakan state; diuji T4/T7/T8/T12.
2. Sesi expired/kicked, socket lama, atau world ID lain tidak boleh menerima data/voice; diuji T3/T5/T6/T13.
3. Client berbohong tentang waktu, posisi, lap atau hit tidak boleh mengubah hasil; diuji T5/T9/T10/T11.
4. Tab background, timeout dan reconnect tidak boleh mengubah batas waktu atau menggandakan avatar; diuji T5/T9/T10/T14.
5. Database/media/server gagal tidak boleh mengklaim history tersimpan atau membiarkan audio aktif tanpa admission; diuji T8/T13/T16.

## A. Peta file dan batas modul

Seluruh path berikut relatif terhadap **repo baru yang belum dibuat**. Detail file adalah rencana, bukan klaim file sudah ada.

```text
patra-world-social/
  apps/web/
    package.json, vite.config.ts, index.html
    public/assets/                         # aset terpilih dari sumber
    src/app/{App,session,routes}.tsx
    src/ui/{theme.css,AppShell.tsx,Dialog.tsx}
    src/world/{renderer,art,camera,map,input,avatars}.ts
    src/net/{api,socket,clock,reconcile}.ts
    src/guest/{session,history}.ts
    src/features/auth/{AuthPage,RecoveryPage}.tsx
    src/features/worlds/{WorldPicker,InvitePage,MembersPanel}.tsx
    src/features/friends/FriendsPanel.tsx
    src/features/chat/ChatPanel.tsx
    src/features/lobby/LobbyPanel.tsx
    src/features/penalty/{PenaltyGame.tsx,local.ts,penalty.css}
    src/features/race/{RaceGame.tsx,renderer.ts,local.ts,controls.ts}
    src/features/duck/{DuckGame.tsx,renderer.ts,local.ts}
    src/features/history/{HistoryPage,MatchDetail,ChampionBoard}.tsx
    src/features/voice/{VoicePanel.tsx,client.ts}
    src/features/settings/SettingsPage.tsx
    tests/                                 # unit/UI tests, mirrored domain names
  apps/server/
    go.mod, go.sum, sqlc.yaml
    cmd/server/main.go
    internal/config/config.go
    internal/http/{router,middleware,errors}.go
    internal/auth/{service,password,session,recovery,http}.go
    internal/social/{friends,blocks,http}.go
    internal/worlds/{service,membership,invites,http}.go
    internal/realtime/{hub,connection,protocol,clock}.go
    internal/world/{runtime,movement,map}.go
    internal/lobby/{service,registry}.go
    internal/games/model/{types,clock,rng}.go
    internal/games/penalty/{engine,bot}.go
    internal/games/race/{engine,track,bot}.go
    internal/games/duck/{engine,targets,rewind}.go
    internal/history/{service,finalizer,worker,http}.go
    internal/leaderboard/{service,champion,http}.go
    internal/chat/{service,http}.go
    internal/voice/{service,livekit,http}.go
    internal/moderation/{service,http}.go
    internal/db/                           # sqlc generated only
    migrations/                            # forward SQL migrations
    queries/                               # typed SQL per domain
    internal/.../*_test.go                  # dekat kode domain
  contracts/
    openapi.yaml, events.schema.json
    maps/{village-v1,race-track-v1}.json
    rules/{hub-v1,race-v1,penalty-v1,duck-v1}.json
    fixtures/{movement,race,penalty,duck,protocol}/
  tests/e2e/{guest,auth,worlds,race,penalty,duck,voice,history}.spec.ts
  tests/integration/                       # DB + WS scenario fixtures
  tools/load/{main.go,README.md}             # scripted clients dan metrik
  deploy/{compose.yaml,Caddyfile,server.Dockerfile,env.example}
  docs/{PROGRESS.md,api.md,operations.md,verification.md}
  docs/superpowers/{specs,plans}/          # salinan spec/plan/prompt/progress awal
  docs/checkpoints/                       # laporan CP1–CP8 dan evidence terkait
  Makefile, package.json, .github/workflows/ci.yml
```

Reusable source: `src/world.js` → fungsi art/renderer; `src/world-math.js` → proyeksi/path primitives; `src/styles.css`/`ui.css` → theme; `PenaltyGame.jsx`/`penalty.css` → presentasi; `public/assets` → aset/font berlisensi. Shell `app.js` tidak disalin sebagai singleton multiplayer. `scripts/deploy.sh` tidak dipakai dengan destination lama. `.idea`, `.env`, node_modules, dist, SSH config dan credentials tidak disalin.

## B. Kontrak bersama sebelum coding paralel

T2 menghasilkan definisi JSON Schema dan tipe Go/TS dengan nama wire yang sama. UUID direpresentasikan string, waktu wire integer Unix milliseconds, durasi integer milliseconds. Rules membaca nilai dalam spec.

| Tipe | Field wajib |
|---|---|
| `Actor` | `actorId`, `actorType: human/bot`, `userId?`, `displayName`, `avatarId` |
| `MapDefinition` | `id`, `version`, `bounds`, `spawn`, `tiles`, `collision`, `stations`, `zones`, `exits` |
| `Command` | `v`, `commandId`, `seq`, `worldId`, `type`, `payload` |
| `PublicEvent` | `v`, `eventId`, `serverTick`, `serverTimeMs`, `type`, `payload` |
| `JoinContext` | `userId`, `sessionId`, `worldId`, `role`, `connectionEpoch` |
| `MatchStart` | `matchId`, `worldId`, `gameType`, `rulesetVersion`, `mapVersion`, `seed`, `actors`, `startsAt` |
| `MatchResult` | `matchId`, `worldId`, `gameType`, `rulesetVersion`, `status`, `startedAt`, `finishedAt`, `participants`, `details` |
| `ParticipantResult` | `actorId`, `userId?`, `rank?`, `score?`, `outcome`, `botAssisted`, `disconnectReason?` |
| `CommandError` | `commandId`, `code`, `message`, `retryable` |

`MatchStart` adalah state internal. Jangan broadcast seed duck atau pilihan rahasia lewat tipe umum; serializer publik per game membatasi field. Outcome `win/loss/draw/completed/dnf/forfeit/abandoned` dibedakan dari status match `running/finalizing/completed/aborted_server/aborted_storage/abandoned`. UI `unconfirmed_storage` adalah status sementara ketika write belum dapat dikonfirmasi, bukan hasil resmi. Persentase akurasi hanya presentasi; data source hits/shots.

Interfaces domain utama:

```go
// Clock dan RNG diinjeksi agar deadline/bot dapat diuji tanpa sleep.
type Clock interface { Now() time.Time }
type RNG interface { IntN(n int) int }

// Setiap package game memiliki New/Apply/Tick/Snapshot/Result sendiri.
// State internal game tidak boleh dipakai sebagai payload public langsung.
type ResultStore interface {
    Enqueue(ctx context.Context, result model.MatchResult) error
    Finalize(ctx context.Context, matchID string) (model.MatchResult, error)
}
```

Root Makefile mendefinisikan `make check`, `make test`, `make test-integration`, `make test-e2e`, `make build`, `make load-smoke`. Frontend memakai npm workspace `apps/web`. Verifikasi seluruh Go dijalankan dari `apps/server` memakai `go test -race ./...`; package-level command pada task juga memakai direktori itu. Jangan menjalankan perintah database destruktif ke produksi.

## C. Backlog implementasi berurutan

Setiap task menghasilkan deliverable yang bisa didemokan dan satu commit scoped setelah pemeriksaan relevan lulus. Untuk fitur, buat tes yang mendemonstrasikan aturan gagal lebih dulu, implementasikan, lalu jalankan lagi. Bootstrap/pemindahan aset cukup diverifikasi build dan visual tanpa tes trivial yang hanya menyalin implementasi.

### T1 — Baseline repo baru dan frontend yang tetap dikenali

**Dependensi:** Persetujuan untuk mulai implementasi setelah review plan.

**Files:** root package/Makefile/README; `apps/web/package.json`, `vite.config.ts`, `index.html`, `src/ui/theme.css`; `apps/server/go.mod`, `cmd/server/main.go`, `internal/config/config.go`; `.github/workflows/ci.yml`; `docs/superpowers/specs/2026-10-26-patra-world-social-design.md`, `docs/superpowers/plans/2026-10-26-patra-world-social.md`, `docs/PROGRESS.md`.

**Interfaces:** Menghasilkan layout workspace, build commands, `GET /health/live`, dan baseline visual sumber yang bisa dibandingkan.

- [x] Revalidasi source commit/status; buat clone lokal independen dengan history dari commit basis yang tercatat, tanpa membawa file untracked/secret dan tanpa hardlink object database. Gunakan branch `feat/social-world` di repo baru. Hapus origin hasil clone **hanya pada repo baru**; pembuatan remote/push berada di luar goal default ini. Jika target sudah ada, periksa lalu resume, jangan overwrite/bootstrap ulang. Catat asal commit dan aset; salin spec, plan, prompt, dan progress awal dengan struktur relatif tetap valid.
- [x] Pindahkan frontend ke `apps/web`, pertahankan art/tema; buat entrypoint Go sederhana dan konfigurasi lewat env. Ganti branding/deskripsi dari portfolio menjadi world sosial tanpa mengganti art.
- [x] Definisikan scripts lint/typecheck/test/build dan CI Go+web. Pin versi dari toolchain aktual yang kompatibel; `.env.example` hanya placeholder.
- [x] Jalankan baseline check/test/build, screenshot desktop/mobile, bandingkan sumber; jangan deploy. Pastikan tidak ada deploy destination portfolio lama di config baru.
- [x] Commit `chore: create isolated social-world baseline`.

**Selesai:** repo baru dapat berjalan lokal; sumber tidak berubah; visual baseline tersedia.

### T2 — Map modular, rules version dan kontrak protocol

**Files:** `contracts/*`; web `world/{map,art,renderer,camera,input,avatars}.ts`; server `world/map.go`, `games/model/{types,clock,rng}.go`, `realtime/protocol.go`; tests `map.test.ts`, `protocol_test.go`.

**Interfaces:** `loadMap(data): MapDefinition`; `drawWorld(canvas, map, publicState, renderOptions)`; `world.LoadMap(data []byte) (MapDefinition,error)`; `ValidateCommand(raw []byte) (Command,error)`.

- [x] Tulis fixtures: round-trip proyeksi; semua entrance reachable; spawn tidak collision; bounds 64×64; schema menolak versi/zone/angka NaN dan field wajib hilang. Assert reduced-motion path memiliki durasi dan posisi sama.
- [x] Jalankan unit fixtures sebelum refactor; kegagalan harus berasal dari kemampuan/bounds baru yang belum tersedia.
- [x] Ekstrak geometri dari hard-code 17×17, art dari singleton player, collision footprint, camera-follow/zoom/minimap; simpan rules JSON dan schema, parser Go/TS. T2 menetapkan unit movement dan fixed-point race untuk parity.
- [x] Jalankan `go test ./internal/world ./internal/realtime` dan `npm run test --workspace apps/web -- map`; screenshot source core, camera zoom dan tiga avatar saling menutupi secara benar.
- [x] Commit `feat: define versioned maps and multiplayer contracts`.

**Selesai:** client dapat menggambar data banyak avatar; map/rules dibaca Go dan TS tanpa beda arti.

### T3 — Akun, cookie session dan recovery

**Files:** server `auth/*.go`, `http/{router,middleware,errors}.go`, migrations `001_identity.sql`, queries `auth.sql`; web auth pages dan `net/api.ts`; `auth/service_test.go`, e2e auth.

**Interfaces:** `Register(ctx, username, password) (User, RecoveryCode,error)`; `Authenticate(ctx, username,password) (Session,error)`; `RequireSession(ctx, token) (Session,error)`; `Recover(ctx,username,code,newPassword) (RecoveryCode,error)`.

- [ ] Tes: `Patra` dan `patra` duplicate; password tidak tersimpan mentah; invalid password/recovery gagal; recovery dipakai ulang gagal; reset/logout/expiry mencabut session; cookie/CSRF/origin enforced; hash concurrency dibatasi.
- [ ] Jalankan `go test ./internal/auth ./internal/http`; amati gagal pada service yang belum ada.
- [ ] Implement Argon2id, cookie opaque, server-side expiry, recovery code hash sekali pakai, login/register throttling, UI tampilkan/simpan recovery. Buat `/api/v1/auth/register|login|logout|recover` dan `GET /api/v1/me`.
- [ ] Jalankan tes unit/integrasi PostgreSQL dan auth E2E; periksa logs tidak mengandung password/token/recovery code.
- [ ] Commit `feat: add username authentication and recovery`.

**Selesai:** akun masuk/keluar/pulih dengan alur nyata; tidak perlu email.

### T4 — Private world, membership, undangan dan role

**Files:** `worlds/{service,membership,invites,http}.go`, migrations `002_worlds.sql`, queries `worlds.sql`; web worlds pages; `worlds/invites_test.go`, e2e worlds.

**Interfaces:** `CreateWorld(ctx,userID,name) (World,error)`; `RedeemInvite(ctx,userID,token) (Membership,error)`; `AuthorizeMember(ctx,userID,worldID) (Membership,error)`; `IssueInvite(ctx,actorID,worldID,targetUserID) (InviteLink,error)`.

- [ ] Tes dua akun redeem token single-use bersamaan → satu membership baru; retry akun sama tetap satu; expired/revoked/target salah/guest/ban ditolak; batas 50 membership; outsider tidak dapat membaca world/history; moderator tidak mengubah owner.
- [ ] Jalankan `go test ./internal/worlds`; gunakan transaksi PostgreSQL asli pada race redeem.
- [ ] Implement API create/list worlds, members, issue/redeem/revoke invite, role change, kick/ban, transfer/archive. UI redirect login mempertahankan tujuan invite tanpa membocorkan token ke log/analytics.
- [ ] Uji invite E2E tiga browser context dan batas 50 membership. T5 memakai `AuthorizeMember` yang sama; pengujian kapasitas online 20 termasuk reconnect dilakukan pada T5/CP3 setelah runtime realtime tersedia.
- [ ] Commit `feat: add private worlds and revocable invites`.

**Selesai:** anggota dapat kembali ke world privat yang sama; owner dapat mencabut akses.

### T5 — Multiplayer movement, isolasi dan reconnect

**Files:** `realtime/{hub,connection,clock}.go`, `world/{runtime,movement}.go`; web `net/{socket,clock,reconcile}.ts`, `world/input.ts`; tests `runtime_test.go`, `connection_test.go`, e2e worlds.

**Interfaces:** `Join(ctx, JoinContext) (PublicSnapshot,error)`; `Submit(ctx,JoinContext,Command) error`; `Resume(ctx,sessionID,epoch,lastEventID) (PublicSnapshot,error)`; `Disconnect(sessionID,reason)`.

- [ ] Tes max20 termasuk reconnect; double socket fenced; world A/B isolated; malformed/out-of-range movement rejected; no speed teleport; idle input expires 250 ms; stale seq ignored; revoked session disconnected; old epoch commands ignored.
- [ ] Jalankan unit dengan fake Clock dan `go test -race ./internal/realtime ./internal/world`; tes concurrent joins tidak perlu sleep arbitrary.
- [ ] Implement single-owner runtime, WS origin/session admission, ping/heartbeat, bounded queues, snapshots/deltas, interpolation, full resync dan status reconnect; pilih baru/takeover secara terlihat saat tab kedua meminta kontrol.
- [ ] Uji 2 browser → berjalan saling terlihat; 20 clients; offline 15 detik/reconnect; tab background; modal terbuka; reduced-motion; baca ulang permission pada revoke. `make load-smoke` dibentuk untuk tes dasar ini.
- [ ] Commit `feat: synchronize private worlds with reconnect support`.

**Selesai:** pemain yang sah terlihat bergerak; user lain/world lain tidak menerima state.

### T6 — Teman, chat, moderation dan world UI

**Files:** `social/*.go`, `chat/*.go`, `moderation/*.go`, migrations `003_social.sql`, queries social/chat; web friends/chat/members panels; tests social/chat + E2E worlds.

**Interfaces:** `RequestFriend(ctx,actorID,targetUsername) error`; `RespondFriend(ctx,actorID,requestID,accept) error`; `SendMessage(ctx,JoinContext,channelID,clientMessageID,text) (Message,error)`; `Moderate(ctx,actorID,worldID,action) error`.

- [ ] Tes request accept/reject/unfriend/block; blocked invite ditolak; chat escaped; 500 karakter; rate limit; duplicate client ID; history pagination; outsider dan muted user ditolak; moderation role enforced.
- [ ] Jalankan `go test ./internal/social ./internal/chat ./internal/moderation`.
- [ ] Implement exact-username search, daftar teman/presence, notifikasi invitation dalam aplikasi, chat world 30 hari dan cleanup, mute/delete/kick; hub event revocation dipanggil ketika hak dicabut.
- [ ] Jalankan multi-context E2E: mengetik tidak bergerak, pesan hanya dalam world, reconnect tidak menggandakan pesan, badge unread dan loading/error jelas.
- [ ] Commit `feat: add friends private chat and moderation`.

**Selesai:** flow undang-teman-main dapat dilakukan tanpa memanggil API manual.

### T7 — Lobby umum, seat dan lifecycle match

**Files:** `lobby/{service,registry}.go`, migrations `004_matches.sql`, queries matches; web `features/lobby/LobbyPanel.tsx`; `lobby/service_test.go`.

**Interfaces:** `JoinLobby(ctx,JoinContext,arenaID) (Lobby,error)`; `SetReady(ctx,userID,lobbyID,ready) error`; `StartMatch(ctx,hostID,lobbyID,commandID) (model.MatchStart,error)`; `ReleaseMatch(matchID,status)`.

- [ ] Tes Start serentak menghasilkan satu match; user busy tidak masuk game lain; race slot6 ditolak; semua ready diperlukan; host pergi berpindah ke terlama; lobby kosong hilang; late join spectator saja; permission spoof gagal.
- [ ] Jalankan `go test -race ./internal/lobby`; fixture satu dunia dengan race/penalty/duck aktif harus tidak deadlock.
- [ ] Implement arena registry per world, roster freeze, seed/ID/start timestamp, persisted running match sebelum countdown, role host, spectators, status busy, explicit leave, return location. Penalty max 2; race max 5; duck 4 kabinet.
- [ ] Jalankan integration dengan DB failure sebelum start → tidak ada countdown/seat yatim; duplicate command mengembalikan match ID sama.
- [ ] Commit `feat: coordinate arena lobbies and match lifecycle`.

**Selesai:** tiga game dapat memakai lifecycle yang sama tanpa berbagi aturan spesifik.

### T8 — History foundation dan finalisasi durable

**Files:** `history/{service,finalizer,worker,http}.go`, migrations `005_results.sql`, queries history/results; web `history/{HistoryPage,MatchDetail}.tsx`; tests finalizer/worker.

**Interfaces:** `Enqueue(ctx, model.MatchResult) error`; `Finalize(ctx,matchID) (model.MatchResult,error)`; `ListHistory(ctx,userID,filter,cursor) (Page,error)`; `RecoverPending(ctx) error`.

- [ ] Tes hasil sama dua kali → satu match/participant/history; crash setelah enqueue sebelum projection → worker lanjut sekali; DB down → finalizing bukan saved; setelah 60 detik arena dilepas tanpa mengklaim gagal/sukses commit; ack hilang direkonsiliasi lewat matchID; startup menandai orphan running aborted; history akun lain dilarang dan history sendiri tetap bisa setelah keluar world.
- [ ] Jalankan `go test ./internal/history` dan integration PostgreSQL pada transaksi/retry.
- [ ] Implement `result_jobs`, unique constraints, immutable result hash, transaction finalization, polling worker dengan retry/backoff, query history/detail. Setelah 60 detik ikuti rekonsiliasi spec: batalkan write outstanding, lepaskan arena, blokir match baru, cek durable job saat DB pulih sebelum memutuskan completed/aborted_storage. Worker status event mengubah UI sementara menjadi tersimpan hanya sesudah commit.
- [ ] Uji failpoint transaksi dan restart; payload kedua dengan matchID sama tetapi hasil berbeda ditolak/diinvestigasi, bukan overwrite.
- [ ] Commit `feat: persist idempotent match results and history`.

**Selesai:** game pertama langsung memakai history nyata, bukan local-only placeholder.

### T9 — Penalti online dan bot

**Files:** `games/penalty/{engine,bot}.go`, `penalty/engine_test.go`, fixtures penalty, migrations `006_penalty.sql`; web `features/penalty/*`, UI tests dan e2e penalty.

**Interfaces:** `New(start model.MatchStart, clock model.Clock, rng model.RNG) *Engine`; `Choose(actorID,turnID string,zone int,receivedAt time.Time) error`; `Tick(now time.Time) []PublicEvent`; `Snapshot(viewerID string) PublicSnapshot`; `Result() (model.MatchResult,bool)`.

- [ ] Tes 16 kombinasi, `2999ms` diterima/`3000ms` ditolak, timeout hanya mengisi slot kosong, pilihan terkunci, bot tidak melihat human choice, early win, sudden-death pair setara, max 5 pairs draw, disconnect 15 detik forfeit, no leak snapshot.
- [ ] Jalankan `go test ./internal/games/penalty`; clocks/RNG deterministic, tidak menunggu tiga detik sungguhan pada unit test.
- [ ] Implement aturan spec dan serializer publik, integrasi lobby2/history, reuse SVG/CSS, pilihan rahasia, timer server, reveal+1500ms, rematch dua persetujuan. `local.ts` mengikuti fixtures untuk guest.
- [ ] Jalankan `npm run test --workspace apps/web -- penalty` dan e2e dua browser; uji tab hidden/clock perangkat bergeser/animasi mati/input lama.
- [ ] Commit `feat: add timed penalty duels and fair bots`.

**Selesai:** manusia-manusia, manusia-bot, dan guest-bot bekerja dengan aturan konsisten.

### T10 — Balap lima lap dengan bot

**Files:** `games/race/{engine,track,bot}.go`, `race/engine_test.go`, fixtures race, map track dan rules race, migrations `007_race.sql`; web race components, controls, local engine; e2e race.

**Interfaces:** `New(start model.MatchStart, track Track, rules Rules) *Engine`; `Input(actorID string,seq uint64,input Controls) error`; `Tick(now time.Time) []PublicEvent`; `Snapshot(viewerID string) PublicSnapshot`; `Result() (model.MatchResult,bool)`.

- [ ] Tes satu manusia+empat bot/5 manusia; semua start satu waktu; no movement countdown; checkpoint skip/reverse finish tidak tambah lap; finis hanya lap 5; 300s/45s limits; host leave; disconnect takeover/DNF; all-human-left abandoned.
- [ ] Jalankan `go test ./internal/games/race` dan fixed-point golden fixtures client/server untuk checkpoint/finish.
- [ ] Implement track tunggal, kart acceleration/brake/steering, wall collision tanpa kart-kart collision, bot input normal, deterministic ordering, countdown 3 detik, spectator view, 10s results, return park; integrasikan T7/T8.
- [ ] Playtest 1–5 manusia, mobile steering, latency/jitter; uji client mengirim posisi/lap/kecepatan fiktif tidak diterima. Catat tuning handling/bot di rules-v1 sebelum rilis.
- [ ] Commit `feat: add five-lap races with bot-filled grids`.

**Selesai:** race tidak bisa start ganda, memotong checkpoint atau mengunci arena selamanya.

### T11 — Duck shooting dengan validasi server

**Files:** `games/duck/{engine,targets,rewind}.go`, `duck/engine_test.go`, fixtures duck, migrations `008_duck.sql`; web duck game/renderer/local; e2e duck.

**Interfaces:** `New(start model.MatchStart,rules Rules,rng model.RNG) *Engine`; `Shoot(actorID string,seq uint64,aim Aim,sample InputTiming) (ShotResult,error)`; `Tick(now time.Time) []PublicEvent`; `Result() (model.MatchResult,bool)`.

- [ ] Tes 60s session, 100/200/500 points, 250ms cooldown, target sekali hit, dead target/duplicate seq ditolak; rewind 150 ms bound, fabricated time rejected; tie target depan; zero-shot accuracy; incomplete session tidak ranked; selesai 60 detik saat grace tetap valid, akhir normal menang pada deadline yang sama, keluar sengaja tetap incomplete.
- [ ] Jalankan `go test ./internal/games/duck`; gunakan fixtures collision/spawn yang sama pada TS guest.
- [ ] Implement authoritative spawn/hit, bounded history buffer, clock offset estimator terukur, UI aim/touch/keyboard, empat kabinet, spectator data tanpa future seed; simpan hits/shots/detail via T8.
- [ ] Jalankan E2E dan profil RTT 50/100/200 ms+jitter; validasi server score konsisten dan maksimum rewind tidak bisa dilampaui melalui timestamp client.
- [ ] Commit `feat: add validated duck arcade sessions`.

**Selesai:** hasil berasal dari server, local practice terpisah dari skor akun.

### T12 — Podium top tiga dan pergantian champion

**Files:** `leaderboard/{service,champion,http}.go`, migrations `009_leaderboards.sql`, queries leaderboard; web ChampionBoard; tests `champion_test.go`, e2e history.

**Interfaces:** `ProjectResult(ctx,tx,result model.MatchResult) error`; `Top(ctx,userID,worldID,ruleset string,limit int) ([]Entry,error)`; `RecentChampionChanges(ctx,userID,worldID string,limit int) ([]ChampionChange,error)`.

- [ ] Tes satu user tidak menempati 3 podium; ranking score→rational accuracy→time→resultID; bot/guest/aborted excluded; PB yang tidak naik tidak replace; dua finalisasi serentak konsisten; pemilik sama naik skor bukan champion change.
- [ ] Jalankan `go test ./internal/leaderboard` dan integration row-lock; initial empty board tidak diberi fake scores.
- [ ] Tambahkan projection dalam transaksi T8; buat podium tiga, personal best, last result, tiga pergantian champion, archive ruleset dan deep link detail terbatas membership.
- [ ] E2E cek new high score muncul sekali pada semua anggota world dan tidak pada world lain; deletion akun memperbarui board dengan audit event administratif terpisah dari kemenangan baru.
- [ ] Commit `feat: add private podiums and champion history`.

**Selesai:** arti “top3” dan “juara terakhir” jelas serta dapat diaudit.

### T13 — Warkop voice dengan admission dan revocation

**Files:** `voice/{service,livekit,http}.go`, `voice/service_test.go`; web `features/voice/*`; warkop zones di map; e2e voice; docs voice runbook.

**Interfaces:** `JoinTable(ctx,JoinContext,tableID) (VoiceAdmission,error)`; `IssueToken(ctx,admissionID) (VoiceToken,error)`; `LeaveTable(ctx,admissionID,reason) error`; `ReconcileParticipants(ctx) error`.

- [ ] Tes seats 8, user kesembilan ditolak, outsider/guest/banned/di luar meja ditolak; hanya satu seat/user; room/world isolation; grants mic-only; logout/kick/leave/game entry mencabut admission; duplicate webhook idempotent.
- [ ] Jalankan `go test ./internal/voice` dengan fake provider; contract test Go SDK harus memeriksa actual token claims, bukan string dummy.
- [ ] Implement dua meja, per-admission opaque media identity, 60s initial token, Go SDK control, signed webhook validation, explicit `revoke_token_ts`, removal retries, room cleanup dan backend reconciliation. Revoke gagal → tandai voice degraded, jangan terbitkan akses meja baru sampai isolasi dipastikan.
- [ ] Hubungkan SDK browser, mute awal, permission hanya tombol mic, listen-only, device selector, PTT/deafen, table chat ephemeral; lakukan uji dua meja dan replay **token awal dan refreshed** setelah kick menggunakan provider staging. Cloud revocation default tidak cukup; ikuti spec.
- [ ] Jalankan real-device desktop/mobile, permission-denied/no-device/network-block tests; commit `feat: add isolated warkop voice tables` setelah bukti tidak ada audio lintas meja.

**Selesai:** voice dapat dipakai, ditinggalkan, dan dicabut tanpa mengganggu game/chat.

### T14 — Guest end-to-end dan account settings

**Files:** web `guest/{session,history}.ts`, SettingsPage, local engines ketiga game, e2e guest; server auth/profile/account deletion endpoints; migrations bila dibutuhkan.

**Interfaces:** `GuestSession.start(gameType, rules): LocalMatch`; `GuestHistory.append(result): Promise<void>`; `GuestHistory.list(): Promise<LocalResult[]>`; `DeleteAccount(ctx,userID,confirmation) error`.

- [ ] Tes IndexedDB 100 records, retention tertua, reset data, reload; guest tidak mengirim WS/authenticated score/media token; golden fixtures tiga game lulus; login hanya migrasi avatar/settings; account delete revokes/recomputes sesuai spec.
- [ ] Jalankan local engine tests dan guest E2E sebelum melengkapi missing routes/UI.
- [ ] Implement entry “Main sendiri”/“Main dengan teman”, local world, seluruh game local, badge “perangkat ini”, setting audio/controls/reduced motion, recovery code rotate, password change, ownership transfer/archive sebelum delete.
- [ ] E2E urutan guest bermain→login→world online→logout; tidak ada timer/listener/media/snapshot tersisa dari mode sebelumnya. Tes account deletion dan anonymization dengan history lawan tetap valid.
- [ ] Commit `feat: complete solo guest play and account settings`.

**Selesai:** guest dan akun punya alur lengkap, bukan sekadar tombol yang menutup fitur.

### T15 — Map district, accessibility dan mobile polish

**Files:** contracts village map, web art/camera/avatars/UI/theme; tests e2e accessibility/map; docs screenshot evidence.

**Interfaces:** Memakai `MapDefinition`, `drawWorld` dan input adapters T2/T5; tidak mengubah aturan networking.

- [ ] Tambah regression checks untuk semua entrance reachable, seat/arena spawn aman, nama/status tidak tertutup panel, keyboard/chat focus, viewport kecil, zoom/pointer mapping.
- [ ] Jalankan checks untuk melihat lokasi/komponen yang belum memenuhi desain, bukan snapshot brittle seluruh canvas.
- [ ] Tata central village, race park, penalty field, duck arcade/podium, warkop dan jalan; cache static chunks/cull viewport; optimalkan label, touch steering/aim, focus, error/loading states.
- [ ] Review screenshot pada 390×844, 768×1024, 1440×900 dan browser nyata; keyboard-only parcours, reduced-motion, font scaling, CPU throttling; catat perangkat/fps.
- [ ] Commit `feat: expand the village and polish accessible play`.

**Selesai:** visual tetap Patra World dan semua lokasi nyaman diakses di desktop/mobile.

### T16 — Reliability, load test dan paket beta terpisah

**Files:** `tools/load/*`, deploy files, health/readiness/drain di server, metrics/logging, docs operations/verification; integration chaos tests.

**Interfaces:** `/health/live`, `/health/ready`, private metrics; `Drain(ctx) error`; startup `RecoverPending`; config origin/DB/media secrets dan rules versions.

- [ ] Uji scripted 60 clients di 3 worlds, tiap world race 5 + penalty 2 + duck 4, tambahan viewer/movement/chat; voice 48 participants diuji terpisah agar resource matrix masuk akal. Simulasikan delay, packet loss/disconnect, slow clients, DB outage, process restart dan provider failure.
- [ ] Jalankan `go test -race ./...`, frontend checks/unit, integrasi, E2E, production build; pastikan seluruh critical acceptance spec punya evidence.
- [ ] Buat container/binary deploy, contoh Caddy origin baru, DB migrations/backup, readiness, drain, release rollback dan secrets configuration; uji pada environment lokal dan staging yang memang sudah tersedia serta diizinkan. Jangan mengubah Caddy/VPS produksi atau mengarang domain/credentials.
- [ ] Jalankan soak 60 menit dan restore backup; catat p95 tick, FPS, CPU/RAM/network, queue/goroutine growth; uji keamanan guest/IDOR/session/voice replay. Jika kapasitas meleset, kurangi target beta secara eksplisit sebelum rilis.
- [ ] Commit `ops: prepare tested isolated beta deployment`; serahkan paket beta beserta hasil validasi dan runbook rollout/rollback. Deploy publik, push remote, provisioning layanan berbayar dan perubahan hosting masuk R1 terpisah setelah target ditentukan pengguna.

**Selesai:** paket beta terukur dan dapat dipulihkan pada environment yang diuji, source/hosting portfolio tidak tertimpa. Hasil device/provider yang belum diuji tetap pending; produksi belum dirilis.

## D. Dependensi dan paralelisasi

```text
T1 → T2 ───────────────→ T15 visual/map polish
       └→ T3 → T4 → T5 → T6
                   └→ T7 → T8
                            ├→ T9 penalty ───┐
                            ├→ T10 race ─────┤
                            └→ T11 duck → T12┤
                    T5+T6 ─→ T13 voice       ├→ T14 → T16
                                              ┘
```

T9/T10/T11 bisa dibagi setelah contracts/lobby/result store stabil. T13 dapat paralel dengan game setelah identity/world/session hooks tersedia. T15 dapat berjalan bertahap sejak T2. Hindari beberapa worker mengubah protocol, migration numbering atau finalizer tanpa owner integrasi. Review kontrak lebih dahulu; berikan tiap worker subset file jelas.

Untuk demo pertama, selesaikan T1–T9 sampai private world + penalty nyata + history; race dan duck menyusul. Ini memberi bukti flow lengkap lebih dini sambil menjaga semua fitur yang diminta tetap masuk beta.

## E. Coverage dan checklist handoff

| Permintaan pengguna | Task |
|---|---|
| Repo baru dari patra-world | T1 |
| Tema sama, map luas, space tambahan | T2, T15 |
| Backend Golang | T1–T13, T16 |
| Guest solo | T9–T11 local adapters, T14 |
| Login username/password | T3, T14 |
| Private multiplayer, undang teman | T4–T6 |
| Race 5 lap/max 5/bot/area start | T7, T10 |
| Penalty 1v1/3 detik/random/bot | T7, T9 |
| Bebek/history/champion top3 | T8, T11, T12 |
| Warkop mic/chat | T6, T13, T15 |
| History lintas game | T8–T12 |
| Testing/hosting/recovery | T16 |

Saat mulai implementasi, copy spec dan plan ini ke docs repo baru, simpan default yang disetujui, dan update checkbox dengan evidence commit/test. Perubahan aturan setelah playtest harus memperbarui ruleset/spec/fixtures bersama. File rencana ini tidak membuktikan fitur telah dibuat atau tes telah dijalankan.

Deliverable review sekarang: dua dokumen lengkap untuk menilai konsep, scope, aturan, arsitektur, biaya kompleksitas, urutan dan cara membuktikan selesai. Tidak ada implementasi dimulai sebelum pengguna meminta lanjut.

## F. Checkpoint eksekusi dan batas commit

**Target prompt default: CP1 sampai CP8.** Checkpoint adalah titik verifikasi dan penyimpanan progres; setelah lulus lanjut otomatis. Tidak meminta konfirmasi pada setiap checkpoint. Jika pengguna hanya meminta sampai checkpoint tertentu pada pesan eksekusi, batas tersebut menjadi target run.

`C01`–`C16` adalah **label rencana commit**, bukan SHA Git. SHA nyata baru dicatat setelah commit dibuat. Semua checkpoint berikut masih `not_started` pada 26 Oktober 2026.

| Checkpoint | Task | Commit fitur terakhir | Hasil yang dapat didemokan | Bukti minimum untuk lulus |
|---|---|---|---|---|
| **CP1 — Fondasi** | T1, T2 | C02 | Repo baru berjalan, tema lama dikenali, map/rules/protocol modular | Build frontend+Go lulus; map fixture parity; screenshot desktop/mobile; source lama tetap utuh |
| **CP2 — Akun dan akses privat** | T3, T4 | C04 | Daftar/login/recovery; buat world; invite yang bisa expired/revoked | Auth dan database integration; concurrent invite redeem; izin member/owner; outsider ditolak |
| **CP3 — Dunia sosial realtime** | T5, T6 | C06 | Dua akun berjalan, berteman, chatting dan reconnect pada world privat | Dua browser context; world isolation; 20 online termasuk reconnect; session revoke memutus socket; chat/moderation |
| **CP4 — Game pertama lengkap** | T7, T8, T9 | C09 | Lobby → penalti manusia/bot → hasil tersimpan → history | Timer tepat 3.000 ms; pilihan tersembunyi; timeout random; finalisasi idempotent; DB failure/recovery |
| **CP5 — Balapan** | T10 | C10 | Race lima peserta/lima lap, bot fill dan hasil | 1 manusia+4 bot dan 5 manusia; double Start; checkpoint anti-cut; DNF; reconnect; history |
| **CP6 — Duck Arcade** | T11, T12 | C12 | Sesi bebek, podium top 3 dan pergantian champion | Server hit validation; tie-break; tiga akun unik; guest/bot excluded; concurrent champion update |
| **CP7 — Warkop dan guest** | T13, T14 | C14 | Voice per meja; semua game guest; history lokal; settings | Audio nyata dua meja terisolasi; replay token lama ditolak; permission denial; guest API rejection; account/guest switch tanpa bocor state |
| **CP8 — Paket beta terverifikasi** | T15, T16 | C16 | Map final, mobile controls, paket deploy terpisah, runbook | Regression; Go race detector; load/soak; restore; bukti browser/device; semua checkpoint sebelumnya lulus |

**CP4 adalah demo multiplayer pertama**, bukan seluruh produk selesai. **CP8 adalah akhir goal default**, bukan deployment publik. R1 (remote/push/public release/provisioning) berada di luar goal ini kecuali pengguna kemudian memperluas scope secara eksplisit.

### Pemeriksaan pada setiap checkpoint

1. Seluruh acceptance task checkpoint lulus; perbaiki defect yang ditemukan sebelum menandai checkpoint selesai.
2. Jalankan pemeriksaan perubahan dan regresi terkait. Pada CP1–CP3, E2E hanya untuk fitur yang sudah dibangun; jangan menciptakan tes placeholder agar suite tampak hijau. CP8 menjalankan seluruh suite yang relevan.
3. Minta reviewer independen menilai hasil checkpoint bila subagent tersedia; temuan yang menghalangi acceptance harus diperbaiki dan diuji ulang. Review otomatis tidak menggantikan bukti runtime.
4. Simpan commit kode beserta hash yang nyata, commands, exit status, waktu, environment, dan lokasi evidence dalam laporan checkpoint.
5. Buat commit dokumentasi `docs: record checkpoint cpN`, lalu laporkan SHA checkpoint tersebut kepada pengguna. Checkpoint report tidak perlu mencantumkan SHA commit dirinya sendiri.
6. Lanjutkan task berikutnya yang dependensinya terpenuhi. Checkpoint tidak otomatis mengakhiri run.

## G. Daftar commit yang direncanakan

Satu task memiliki minimal satu commit fitur utuh setelah tes terkait lulus. Daftar berikut menentukan batas hasil; bukan kewajiban menyatukan task yang terlalu besar dalam satu diff. Bila dibagi, gunakan subtask `T10.1`, `T10.2`, dan seterusnya dengan hasil yang masing-masing bisa diverifikasi; C10 menunjuk commit terakhir yang menuntaskan T10.

| Label | Task | Subject commit fitur | Termasuk checkpoint |
|---|---|---|---|
| C01 | T1 | `chore: create isolated social-world baseline` | CP1 |
| C02 | T2 | `feat: define versioned maps and multiplayer contracts` | CP1 |
| C03 | T3 | `feat: add username authentication and recovery` | CP2 |
| C04 | T4 | `feat: add private worlds and revocable invites` | CP2 |
| C05 | T5 | `feat: synchronize private worlds with reconnect support` | CP3 |
| C06 | T6 | `feat: add friends private chat and moderation` | CP3 |
| C07 | T7 | `feat: coordinate arena lobbies and match lifecycle` | CP4 |
| C08 | T8 | `feat: persist idempotent match results and history` | CP4 |
| C09 | T9 | `feat: add timed penalty duels and fair bots` | CP4 |
| C10 | T10 | `feat: add five-lap races with bot-filled grids` | CP5 |
| C11 | T11 | `feat: add validated duck arcade sessions` | CP6 |
| C12 | T12 | `feat: add private podiums and champion history` | CP6 |
| C13 | T13 | `feat: add isolated warkop voice tables` | CP7 |
| C14 | T14 | `feat: complete solo guest play and account settings` | CP7 |
| C15 | T15 | `feat: expand the village and polish accessible play` | CP8 |
| C16 | T16 | `ops: prepare tested isolated beta deployment` | CP8 |

**Rencana minimum: 16 commit fitur + 8 commit laporan checkpoint.** Perbaikan atau subtask dapat menambah jumlah commit. Jumlah commit bukan ukuran penyelesaian; acceptance yang menentukan.

### Aturan commit

- Stage path yang memang milik task, bukan seluruh workspace. Tinjau staged diff untuk secret, file generated besar, dan perubahan pengguna.
- Commit fitur memuat source, migrasi, tests, docs kontrak yang berubah, dan checkbox task yang sesuai. Jangan commit `node_modules`, `.env`, credentials, rekaman voice atau data pengguna nyata.
- Kode yang lolos tes lokal tetapi menunggu provider/device boleh di-commit sebagai progres dengan subject `feat: implement ... pending external verification`, status `implemented_pending_external_verification`, dan batas validasi yang eksplisit. Itu belum C13/C16 final atau checkpoint lulus. Buat commit penyelesaian setelah bukti eksternal terpenuhi.
- Commit kode lebih dulu. Setelah SHA diketahui, update ledger/laporan pada commit docs checkpoint berikutnya. Ini menghindari referensi SHA diri sendiri dan amend berulang.
- Jika sebuah task sudah selesai sebelum checkpoint, ledger diperbarui di working tree; commit docs checkpoint menyimpan SHA semua task tersebut. Jika harus menyerahkan run sebelum checkpoint, buat commit `docs: record partial progress after Tn` agar progres bisa dilanjutkan.
- Hanya coordinator mengintegrasikan dan membuat commit pada checkout bersama. Subagent diberi ownership file, tidak membuat commit/migrasi yang bertabrakan. Parallel game implementation boleh memakai isolasi checkout bila diperlukan.
- Tidak squash/amend commit terverifikasi hanya demi merapikan history, kecuali pengguna meminta. Bug setelah checkpoint diperbaiki dengan commit baru.
- Jangan menyebut commit dibuat sebelum memeriksa hasil `git log -1`/`git rev-parse HEAD`. Jika commit gagal, status tetap verified-uncommitted dan langkah berikutnya memperbaiki penyebabnya.

## H. Ledger progres dan prosedur resume

Saat T1 dimulai, salin [progress awal](2026-10-26-patra-world-social-progress.md) menjadi `docs/PROGRESS.md` di repo baru. File itu menjadi **sumber status eksekusi utama**. Salinan template di folder plans tetap dokumentasi awal; jangan memperlakukan template sebagai progres terbaru.

Field wajib ledger:

| Field | Arti |
|---|---|
| `source_commit` | Commit basis yang benar-benar dipakai |
| `working_branch` | Branch eksekusi yang benar-benar aktif |
| `highest_contiguous_passed_checkpoint` | CP tertinggi yang lulus beserta seluruh pendahulunya |
| `last_verified_task_commit` | SHA commit task terakhir yang acceptance-nya sudah lulus |
| `latest_code_commit` | SHA progres kode terakhir, mungkin belum tervalidasi eksternal |
| `current_task` / `next_task` | Task yang dikerjakan/berikutnya |
| `task_status` | `not_started`, `in_progress`, `implemented_pending_external_verification`, atau `done` |
| `verification` | Perintah, exit code, environment, evidence path, timestamp |
| `blockers` | Prasyarat yang hilang, task terdampak, pekerjaan independen yang tetap bisa lanjut |
| `uncommitted_changes` | File milik task yang belum tersimpan, tanpa menyertakan secret |

Laporan checkpoint `docs/checkpoints/CPN.md` memuat: tujuan; task dan SHA; acceptance PASS/FAIL/NOT_RUN; commands; hasil reviewer dan resolusi; screenshot/log yang sudah disanitasi; defect tersisa; next task. Hindari klaim “semua lulus” jika ada acceptance wajib `NOT_RUN`.

Saat melanjutkan run:

1. Baca spec, plan, `docs/PROGRESS.md`, lalu laporan checkpoint terakhir.
2. Cocokkan status dengan `git status`, `git log`, branch dan file aktual. SHA pada ledger harus ada di ancestry branch eksekusi. Jangan mengasumsikan task selesai hanya dari checkbox.
3. Jika kode lebih baru dari ledger, periksa diff/commit dan rekonsiliasi ledger; tidak menghapus atau mengulang pekerjaan yang sudah ada.
4. Jika working tree berisi perubahan belum dikenal, pertahankan dan tentukan ownership sebelum mengedit file yang sama.
5. Verifikasi ulang hanya pemeriksaan yang invalid karena perubahan/environment atau evidence hilang; jangan mengulang semua soak setiap resume tanpa alasan.
6. Lanjutkan task pertama yang belum selesai dan dependensinya tersedia. Gunakan source repo hanya sebagai referensi, bukan tempat commit implementasi.

Format update checkpoint kepada pengguna:

```text
Checkpoint: CP4 — PASS
Task selesai: T7, T8, T9
Commit fitur terakhir: <SHA nyata C09>
Commit laporan: <SHA nyata commit CP4>
Verifikasi: <commands dan hasil ringkas>
Demo: <alamat lokal yang benar-benar aktif>
Pending: <jika ada; tidak menyamarkan acceptance wajib yang belum lulus>
Berikutnya: T10 / CP5
```

## I. Prasyarat eksternal dan batas penyelesaian

| Prasyarat | Kapan diperlukan | Jika belum tersedia |
|---|---|---|
| Go, Node/npm dan PostgreSQL lokal yang dapat dijalankan | T1–T4 | Gunakan runtime yang tersedia atau setup dependency lokal yang wajar; catat kendala permission/network secara tepat |
| Project/kredensial LiveKit staging serta secure origin | T13/CP7 | Selesaikan SDK/admission/revocation code dan local tests; tandai pending external; lanjut T14/T15 dan T16 yang independen; jangan membeli layanan otomatis |
| Provider nyata untuk audio isolation dan replay token refreshed | T13/CP7 | Fake provider/token decode tidak menggantikan acceptance ini; CP7 belum PASS |
| Browser/perangkat Android/iOS dan jaringan yang diperlukan spec | T13/T15/CP8 | Catat device/browser yang benar-benar diuji; emulasi bukan bukti perangkat fisik; acceptance yang belum diuji tetap NOT_RUN |
| Resource untuk load60 game clients, voice load terpisah dan soak60menit | T16/CP8 | Gunakan angka yang benar-benar terukur; jangan menaikkan klaim kapasitas atau menurunkan target diam-diam |
| Domain/VPS/remote GitHub/biaya layanan | R1, sesudah goal default | Di luar CP1–CP8; siapkan config/runbook, tidak push/deploy/provision otomatis |

Jika ada prasyarat wajib yang hilang, minta informasi spesifik melalui alur input yang tersedia lalu teruskan pekerjaan independen. Saat tidak ada lagi pekerjaan bermakna tanpa input, laporkan task pending beserta bukti yang sudah ada; jangan menandai goal complete atau CP8 PASS. Status task pending bukan perintah untuk mengubah status goal secara tidak sesuai aturan runtime.

**Kriteria target goal tercapai:** T1–T16 done; CP1–CP8 PASS; seluruh acceptance wajib memiliki bukti; source lama terjaga; seluruh perubahan milik task sudah di-commit; preview lokal dan petunjuk menjalankan benar; paket deploy/rollback tersedia; daftar commit dan laporan akhir lengkap. Deploy publik tidak termasuk.

## J. Handoff `/goal`

Gunakan [prompt siap salin](2026-10-26-patra-world-social-goal.txt). Mengirim prompt tersebut mengotorisasi implementasi repo lokal dan commit sesuai batasnya, sekaligus memilih default rancangan untuk dikerjakan. Menyimpan file prompt saja tidak mengaktifkan goal.

Target pertama yang bisa dilihat adalah CP1; demo end-to-end pertama CP4; target akhir CP8. Eksekutor melanjutkan otomatis setelah checkpoint lulus. Tidak ada token budget atau jadwal otomatis yang ditetapkan oleh dokumen ini.
