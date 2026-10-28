# World Activities Adjustment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Menambah aktivitas sosial yang dimainkan di world, mempertahankan game/multiplayer lama, memperbaiki minimap/popup/SFX, dan membuktikan performa tetap ringan.

**Architecture:** World owner di Go mengatur area dinamis serta aktivitas santai; lobby owner tetap mengatur pertandingan dan durability. Coordinator dengan lease/generation menghubungkan keduanya. Frontend memakai persistent scene store dan satu render loop Canvas2D; input hanya dimiliki satu mode. Game baru dimuat bertahap, tanpa framework physics umum.

**Tech Stack:** Canvas2D, TypeScript/React/Vite, Go 1.26.8, PostgreSQL, WebSocket, Web Audio; test runner proyek yang sudah ada. Tidak ada dependency runtime baru yang diwajibkan.

**Spec:** [World activities design](../specs/2026-10-28-world-activities-design.md).

**Status:** Implementasi diizinkan pengguna pada 28 Oktober 2026, mencakup seluruh enam tahap ringkasan (Task 0–11). Task 0–6 telah diimplementasikan dan direview; gate perangkat fisik/voice/auth tetap terpisah. Base `3079f05`, branch `feat/world-activities`. [Bukti tahap 1](../../checkpoints/evidence/CP8/activities-stage1.md); [area authority dan scene](../../checkpoints/evidence/CP8/activities-stage2.md).

## Ringkasan tahap dan cakupan

| Urutan | Hasil yang dapat direview | Permintaan yang tercakup |
| --- | --- | --- |
| 0 | Baseline profil + pemeriksaan lifecycle yang akan dipakai | Optimasi dan multiplayer tetap aman |
| 1 | Minimap interaktif dan semua popup terpusat | Klik map kanan bawah; popup di tengah |
| 2 | AudioService reusable dan SFX game lama | Sound effect, mute, cleanup |
| 3–4 | Kontrak, lease area, scene/input owner, spectator | Fondasi aktivitas terlihat di world |
| 5 | Free ride + race pada lintasan yang sama | Mobil/sepeda santai; race steril |
| 6 | Bola bebas + penalti di lapangan + penonton | Lapangan steril, pertandingan terlihat |
| 7 | Pingpong solo melawan BOT / 1v1 | Game meja baru |
| 8 | Mesin capit bersama | Capit boneka |
| 9 | Fishing kosmetik tanpa persistence | Mancing pura-pura |
| 10 | Blaster solo/co-op | Top-down bergerak dan menembak |
| 11 | Profil ramai, regression, soak dan review | Bukti ringan, tidak hanya klaim |

Task 1 dan 2 dapat diparalelkan setelah baseline. Task 3→4 membentuk integrasi bersama. Task 5–9 dapat dibagi setelah integrasi stabil dengan kepemilikan file yang jelas. Task 10 dikerjakan terakhir memakai kontrak/lifecycle yang sudah diuji. Root/coordinator mengintegrasikan dan mereview setiap task sebelum task yang bergantung padanya dimulai.

## Global Constraints

- User sudah meminta implementasi seluruh tahap 1–6 pada 28 Oktober 2026. Pertahankan urutan, simpan evidence dan commit lokal bertahap.
- Fullscreen kampung autumn, tanpa CV/sidebar; semi-English labels dan penjelasan Indonesia. Source repo `patra-world` tetap tidak berubah.
- Private invite worlds; guest solo lokal. Race lama tetap 5 lap / 5 slot; penalti tetap 3 s untuk memilih / 1.5 s untuk reveal. Duck/podium/history, accounts, friends, moderation dan voice tetap berfungsi.
- Blaster: usulan 1–4 pemain manusia, 3 wave, 180 s, 3 HP, ≤8 bot / 48 projectile; hanya solo/co-op. Pingpong: usulan 7 poin, selisih 2, batas 11, 120 s. Lihat spec untuk rules lengkap.
- Casual: 4 kendaraan, 1 bola bebas, 1 mesin capit, 4 tempat fishing per world; lease capit 20 s. Fishing/ride/freeball/claw tidak masuk completion/history/inventory. Pingpong dan Blaster memakai pipeline result/history yang sudah ada.
- Sterilization sebelum countdown/Engine.Start; gameplay tidak dimulai sebelum area ACK dan persist confirmed. Uncertain start tetap fenced, hasil lambat tidak menutup ruang tanpa batas. Rematch reserve ulang.
- Semua popup termasuk Chat terpusat; HUD aktivitas tetap kecil. Legacy Duck tetap popup terpusat sebagai exception kompatibilitas; Blaster, pingpong, race/penalty dan casual activities tampil di world.
- Map tetap 64×64 dengan 1024 subunits; asset identity baru `kampung-senja-v2`, map schema 1; wire protocol 2 dengan incompatibility guard. Data historis protocol 1 dan checksum match tidak ditulis ulang.
- Target performa pada spec adalah release gates usulan, bukan hasil terukur. Tidak mengubah authority/aturan game demi angka benchmark.
- No public deployment. User-deferred browser authentication/LiveKit gates tetap eksplisit; jangan memalsukan session/acceptance. Implementasi, review, test evidence dan plan wajib ikut commit lokal per tahap; publikasi mengikuti instruksi user saat eksekusi.

## Review Focus

1. Entry/kick/mount pada milidetik terakhir dan clear/release lama setelah reconnect/rematch tidak menembus arena atau membuka lease baru (Task 3/5/6).
2. World yang dihancurkan/dibuat ulang, pencabutan akun, takeover dan persistence yang belum pasti tidak meninggalkan ghost rider/area terkunci atau meloloskan match tanpa gate (Task 0/3).
3. Modal, pengetikan, kontrol touch dua jempol dan blur tidak mengirim input ke dua mode; menutup view tidak menghentikan deadline atau subscription penting (Task 1/4/10).
4. Penonton dan pemain yang reconnect melihat public state yang benar, tanpa pilihan penalti rahasia, hak input atau baris history; casual/fishing tidak masuk history (Task 4/6/7/9/10).
5. Kapasitas penuh, lag, duplikasi event dan 20 siklus mount tidak menambah timer, traffic, particle/audio pool atau memory tanpa batas (Task 2/8/10/11).

## Interface yang mengikat task

Definisikan kontrak di Task 3 sebelum implementasi yang bergantung padanya. Nama berikut adalah rancangan integrasi, bukan API yang sudah tersedia.

- Go `AreaLease {WorldID, Incarnation, AreaID, MatchID string; Generation uint64}`; `ReserveArea(ctx, worldID, areaID, matchID) (AreaLease,error)`, `ClearArea(ctx, lease) error`, `ActivateArea(ctx, lease, startsAt int64) error`, `ReleaseArea(ctx, lease) error`. ACK berasal dari owner, operasi idempotent; coordinator tidak menunggu SQL di owner loop.
- World `ApplyCasualIntent(userID, epoch, seq, areaGeneration, intent)` hanya menerima kind/input, bukan posisi/score/hit. `intent` adalah union tervalidasi dalam schema per ride/ball/claw/fishing; jenis baru tidak diterima sebagai JSON bebas.
- `ActivitySnapshot`: worldId/areaId/incarnation/generation/serverTimeMs/tick/kind ditambah payload union yang dibatasi berdasarkan kind. Snapshot state replaceable; lifecycle/completion tidak boleh hilang karena coalescing. Full resync saat interest/epoch berubah.
- Client `ActivityStore.attach(source)`, `.apply(snapshot)`, `.leave()`, `.subscribe(listener)`; sumber local/online menghasilkan contract yang sama. `InputRouter.setMode(mode)` + `.release()`; setiap mode punya tepat satu adapter input. `world.setActivityState(state)` hanya untuk render.
- Client `AudioService.unlock()`, `.setEnabled(boolean)`, `.emit({id,kind,worldPosition?,at})`, `.stopScope(scope)`, `.dispose()`; event sudah lewat/ID duplikat tidak diputar ulang. Physics/timing tidak bergantung pada audio.

### Task 0: Baseline dan lifecycle prerequisites

**Files:** baca CP7/T14 reports dan `docs/HANDOFF.md`; buat `docs/checkpoints/evidence/CP8/activities-baseline.md`; buat profiling fixture `apps/web/tests/activities-performance.fixture.ts`; fokus existing tests di `apps/server/internal/lobby/presence_timing_test.go`, `prepare_test.go`, `apps/server/internal/application/lobby_integration_test.go`.

- [ ] Catat commit, build size per chunk, heap/RAF/timer/listener baseline, frame-time, incoming bytes dan owner tick sebelum perubahan; fixture deterministik berisi 20 avatar, tanpa mengklaimnya sebagai browser multiplayer acceptance.
- [x] Selesaikan scoped review T14 putaran 2 yang tertunda. Reproduksi transisi reconnect berurutan serta pemulihan start yang belum pasti yang akan memengaruhi lease. Perbaiki hanya defect yang terbukti dengan RED→GREEN dan fresh review; jangan menganggap T16 seluruhnya selesai.
- [x] Gunakan isolated PostgreSQL test helpers yang sudah ada. Simulasikan persist-before-error versus fail-before-persist dan pembuatan ulang room; identifikasi owner sumber kebenaran sebelum Task 3.
- [x] Jalankan baseline `npm test`, typecheck/build dan affected Go tests dengan pinned toolchain. Simpan hasil aktual dan keterbatasan; commit baseline/evidence dan perbaikan prerequisite terpisah bila ada.


### Task 1: Minimap klik dan popup terpusat

**Files:** create `apps/web/src/world/minimap.ts`, `apps/web/src/ui/dialogs.ts`; modify `apps/web/src/world/controller.ts`, `app.js`, `world-shell.css`, `index.html`, `chat-ui.ts`; test `apps/web/tests/app-shell.test.ts`, `map.test.ts`, new `world-ui.e2e.spec.ts`.

**Interfaces:** `mountMinimap(element, {openMap})`; `mapPoint(clientPoint, canvasRect, contentBounds): Point|null`; `openCenteredDialog(dialog, trigger)` dengan focus restoration. Destination tetap lewat world adapter; tidak menulis posisi online.

- [x] Tambah failing tests `minimap_click_never_routes_ground_behind_it`, `map_click_respects_letterbox_and_dynamic_closed_zone`, `dialog_close_cleanup_precedes_destination`. Assert pemetaan tengah/tepi pada zoom 0.65/1/1.5 dan DPR 1/2; target tidak valid tidak mengirim teleport/destination liar.
- [x] Pisahkan minimap menjadi DOM/canvas surface focusable; tap membuka Map di tengah. Peta besar menerima click-to-path dan destination buttons tetap ada. Pada locked area pilih safe entrance, bukan dalam arena.
- [x] Terapkan explicit native dialog centering, internal scroll/safe-area dan focus restore; konversi Chat menjadi centered dialog sambil mempertahankan IDs/form/mount. Jangan memberi display:grid/flex pada dialog tertutup.
- [ ] Periksa di browser pada 320×568, 390×844, 768×1024, 844×390, desktop, zoom 200%, keyboard/on-screen keyboard. Semua popup terpusat; Tab/Escape dan typing guard tidak menggerakkan world. Jalankan tests/typecheck/build, review, commit.


### Task 2: Sound service ringan

**Files:** create `apps/web/src/audio/service.ts`, `events.ts`; modify `preferences.ts`, `app.js`, existing game session/views, `apps/server/internal/games/duck/adapter.go` bila producer event perlu diperbaiki; tests new `apps/web/tests/audio.test.ts` dan existing Duck transport tests.

**Interfaces:** AudioService di atas; kind untuk UI, paddle, kick, goal/save, countdown/finish, grab, ride, fire/hit, cast/catch. Semua task aktivitas mengirim event ke service yang sama.

- [x] Tambah failing tests `muted_before_gesture_stays_silent`, `duplicate_or_resync_event_never_replays`, `hidden_or_mode_exit_stops_loops`, `eight_voice_limit_and_cleanup`, `account_generation_does_not_restore_old_audio`.
- [x] Implementasi satu AudioContext setelah gesture; bank ≤200 KiB compressed atau synth pendek; cache/dedup dibatasi, maksimum 8 voice, attenuation sederhana. Hubungkan preference audio yang ada, tanpa perubahan mute LiveKit.
- [x] Sambungkan countdown/finish/penalty reveal/Duck hit. Pastikan producer `duck.shot_resolved` benar-benar mengirim satu event accepted hit/miss, nol untuk rejection; test actual transport + ACK dedup sebelum mengandalkannya untuk SFX.
- [x] Uji mute/reload/account-switch, audio suspended, 20 siklus open/close dan feedback visual saat audio mati. Typecheck/test/build + targeted Go bila berubah; review dan commit. Tidak perlu membuat sound per frame.


### Task 3: Kontrak dan area authority

**Files:** create `contracts/activities.v1.json`, `activity.schema.json`, `activities.golden.json`; modify protocol/event/snapshot schemas & goldens, `apps/web/src/world/avatars.ts`, `net/reconcile.ts`, `net/socket.ts`; create `apps/server/internal/world/areas.go`, `activities.go`, `apps/server/internal/application/areas.go`; modify `world/movement.go`, `runtime.go`, `activity.go`, `realtime/{protocol,connection,delivery,hub}.go`, `lobby/{start,registry,presence}.go`, `application/lobby.go`; create mirrored embedded activity asset and tests.

**Produces:** kontrak lease/coordinator/ActivitySnapshot/intent di atas; handshake protocol 2. Data menetapkan masks, boundary/entrance/safe-return/spectator strips, scene transforms, slot kendaraan/fishing. Regenerator tetap `scripts/generate-kampung-senja.mjs`.

- [x] Tambah RED `start_waits_for_clear_ack_and_durability`, `clearing_blocks_existing_paths_and_last_tick_intent`, `old_generation_cannot_release_rematch`, `recreated_runtime_restores_active_leases`, `definite_abort_releases_but_unknown_start_does_not` di owner/application tests; jalankan untuk membuktikan kegagalan baseline.
- [x] Definisikan geometri map baru dalam 64×64, validasi bahwa safe anchor di luar arena tidak berada di obstacle/area lain, lalu perbarui static art/map/voice checksum/Go mirrors bersama. Return registry berasal dari kontrak, bukan angka lama yang hardcoded.
- [x] Implementasi OPEN→CLEARING 3 s→RESERVED→ACTIVE→RESETTING ≤2 s→OPEN; dynamic mask dicek tiap movement step dan path node, bukan hanya saat destination dikirim. Bersihkan input/path/mount/ball ketika menutup.
- [x] Hubungkan async acknowledged lease handshake sebelum Engine.Start/persist activation. Spatial release setelah gameplay selesai terpisah dari durable busy/result claims; rematch acquire ulang. Hydrate lease sebelum admission setelah runtime recreation.
- [ ] Input continuous baru memakai movement budget (maksimum 20 Hz di client, server limiter tetap dibatasi); lifecycle events berurutan, snapshot yang dapat diganti memakai coalescing. Semua command memeriksa world/user epoch/seq/generation, proximity dan membership.
- [x] Uji logout/kick/deletion/takeover/disconnect/kegagalan storage, client protocol 1 ditolak dengan instruksi reload, hasil historis protocol 1 tetap terbaca. Jalankan JS goldens dan Go world/lobby/realtime/application integration+race; review integrasi, commit atomik dengan kompatibilitas asset/protocol.

### Task 4: Scene store, penonton dan input owner

**Files:** create `apps/web/src/activities/{store,input-router,scene}.ts`, `ActivityHUD.tsx`, `apps/web/src/world/scenes/{race,penalty}.ts`; modify `world/controller.ts`, `village-art.ts`, `app.js`, `online-ui.ts`, `features/lobby/LobbyPanel.tsx`, existing Race/Duel controllers.

**Consumes:** kontrak Task 3. **Produces:** persistent store/input modes, world.setActivityState; React HUD hanya projection dengan ≤10 update/s, Canvas memakai satu RAF.

- [x] Tambah RED `closing_menu_does_not_unsubscribe_game`, `one_input_owner_only`, `modal_and_text_neutralize_active_mode`, `passive_spectator_has_no_input_or_history`, `resync_swaps_generation_without_replaying_effects`.
- [x] Ekstrak transport/session lifetime dari React modal ke persistent store. Pisahkan UI render dari fixed-step model. Lazy-load modul aktivitas; cull entities/labels/props, jangan duplikasi Canvas untuk seluruh world.
- [x] Implementasi public scene snapshots/depth order; actor peserta menggantikan avatar berjalan tanpa duplikasi. Penonton pasif di dekat arena melihat state tanpa otomatis lobby.join; tidak memodifikasi voice admission.
- [x] InputRouter menangani walk/ride/race/penalty/pingpong/claw/blaster/fishing/modal; cleanup pointer/key/visibility/disconnect. Menu/hasil tetap centered, tanpa mematikan authoritative snapshots/deadline.
- [ ] Uji local→online→local, dua match di area berbeda, keluar/masuk radius spectator dan reconnect. Daftarkan semua file test/production baru di test-map runner serta tsconfig, jangan mengandalkan import coverage kebetulan. Review dan commit.

### Task 5: Free ride dan race satu lintasan

**Files:** create `apps/server/internal/world/vehicles.go`, `apps/web/src/activities/vehicles.ts`, `world/scenes/vehicles.ts`; modify race scene, village-art, contracts/activities, `games/race/track.go` hanya bila helper pure perlu diekstrak; tests `vehicles_test.go`, new `activities.test.ts`, race transport/scheduler tests.

- [x] RED `single_rider_per_vehicle`, `vehicle_cannot_leave_track_or_tunnel`, `race_clears_riders_before_countdown`, `reconnect_cannot_mount_closed_track`, `first_finisher_does_not_reopen_track`.
- [x] Implementasi pool 2 sepeda + 2 mobil, satu rider per object generation, kecepatan 6/8 tile/s, swept collision/containment, E untuk turun ke anchor aman. Netralkan input saat lease hilang; tanpa damage/collision antaravatar; kendaraan idle masuk sleep.
- [x] Gambar lintasan dari kontrak race dengan uniform world transform. Render kart dan standings di world/HUD, pertahankan 5 lap / 5 slot / BOT / history / race 30 Hz. Jalur casual memakai containment track yang sama, bukan arena match palsu.
- [x] Buktikan start/finish/abort/rematch, latensi turun dari kendaraan, keyboard/touch, rider keluar world, safe returns dan perimeter penonton. Jalankan affected Go/frontend/browser checks, cek budget awal, review dan commit.

### Task 6: Bola bebas, penalti di lapangan, penonton

**Files:** create `apps/server/internal/world/football.go`, `apps/web/src/activities/football.ts`, `world/scenes/football.ts`; modify penalty scene/Duel/Pitch integration dan application areas; tests new football owner tests, penalty integration, scene/app tests.

- [x] RED `kick_requires_proximity_and_cooldown`, `casual_ball_disabled_when_penalty_reserved`, `secret_choice_not_in_spectator_payload`, `result_storage_delay_does_not_lock_pitch_forever`.
- [x] Satu bola dengan impulse yang dibatasi, friction dan swept bounds; Kick intent memeriksa proximity, reset bila keluar/idle. Tidak membuat match/history. Reservasi penalti menyembunyikan/reset bola casual; bola match hanya projection engine.
- [x] Tampilkan kicker/keeper/bola saat reveal dari public snapshot + server time; pilihan pada HUD empat zona tetap private sampai reveal. Penonton pasif berdiri di strip luar, dapat berjalan pergi, tanpa hak input/baris history.
- [x] Uji penonton terlambat/reconnect tepat sebelum reveal, kick/start pada milidetik terakhir, failure/return/rematch dan history yang sudah ada. Simulasikan hasil pending: lapangan tetap dibuka setelah reset tanpa melepas durable claim. Review dan commit.

### Task 7: Pingpong

**Files:** create `contracts/pingpong-rules.v1.json`, `pingpong.golden.json`, `apps/server/internal/games/pingpong/{engine,adapter,bot}.go`, `apps/web/src/features/pingpong/local.ts`, `apps/web/src/features/pingpong/session.ts`, `apps/web/src/features/pingpong/PingpongHUD.tsx`, `world/scenes/pingpong.ts`; extend lobby/protocol/history/result filters, `guest/completion.ts`, `HistoryPage/MatchDetail`; new migration `013_activity_match_types.sql` untuk allowlist pingpong+blaster (engine tersedia hanya bila sudah terdaftar).

- [x] RED golden rally/serve/bounce/scoring: pertama mencapai 7, unggul 2, batas 11 / deadline 120 s; BOT dipilih eksplisit, kontrol hanya peserta, reconnect 15 s→forfeit dan dedup completed result.
- [x] Implementasi physics meja 2D terbatas pada 20 Hz, manusia 1v1 atau manusia melawan BOT; scene terlihat di meja. Input paddle continuous masuk movement budget, model tidak menerima score/hit dari client.
- [x] Extend match types melalui migration baru, typed result metadata, query filters, immutable history dan guest completion. Tidak mengubah migration 004 atau podium Duck; bot bukan pengguna/akun baru.
- [x] Test kesetaraan Go/TS goldens, match completion/history melalui WS+DB aktual, tidak ada baris untuk spectator, pending/retry, touch/keyboard/focus cleanup. Review dan commit.

Evidence: [implementation](../../checkpoints/evidence/CP8/activities-task7.md), [review](../../checkpoints/evidence/CP8/activities-task7-review.md), [browser](../../checkpoints/evidence/CP8/activities-pingpong-browser.md). Minor constant/JSON drift prevention carried to final review.

### Task 8: Capit boneka

**Files:** create `apps/server/internal/world/claw.go`, `apps/web/src/activities/claw.ts`, `world/scenes/claw.ts`; extend activities asset/schema; new claw owner/client tests.

- [ ] RED `single_operator_twenty_second_lease`, `drop_outcome_uses_server_grip_position`, `duplicate_drop_does_not_duplicate_reward`, `disconnect_or_match_join_resets_operator`.
- [ ] Implementasi kabinet kinematik 2D, preset layout boneka, gerakan arah + Drop, grip tolerance dari kontrak, animasi pickup/release yang sama untuk penonton. Occupied state memberi info siapa memakai; tanpa antrean otomatis, pembayaran atau inventory.
- [ ] Uji lease expiry/drop pada tick terakhir, logout/reconnect, batas input, dedup audio event dan query assertion bahwa tidak ada penulisan match/result/history. Review dan commit.

### Task 9: Fishing tanpa history

**Files:** create `apps/server/internal/world/fishing.go`, `apps/web/src/activities/fishing.ts`, `world/scenes/fishing.ts`; extend activity asset; tests world/client/guest history.

- [ ] RED `four_spots_one_owner_each`, `cast_requires_river_proximity`, `movement_or_leave_cancels_cast`, `fishing_never_calls_completion_or_persists_catch`.
- [ ] Implementasi Cast/wait/Reel/emote dengan timer state kecil, pool visual tangkapan, teks maksimum 3 s; animasi bersama di private world dan versi lokal untuk guest. Tanpa simulasi entitas ikan/inventory, XP atau leaderboard.
- [ ] Uji catch/repeat selama beberapa menit, reload/logout/masuk match; row count/hash history akun + guest tetap sama. Uji kontrol mobile, feedback saat sound mati dan cleanup; review dan commit.

### Task 10: Blaster solo/co-op

**Files:** create `contracts/blaster-rules.v1.json`, `blaster.golden.json`, `apps/server/internal/games/blaster/{engine,adapter,bots}.go`, `apps/web/src/features/blaster/local.ts`, `apps/web/src/features/blaster/session.ts`, `apps/web/src/features/blaster/BlasterHUD.tsx`, `world/scenes/blaster.ts`; modify lobby/protocol/rate budget/history details/guest completion, map area + asset generation, activities store/input router; tests new model/transport/DB/touch scenarios.

- [ ] RED kesetaraan fixed tick, speed/fire cooldown/TTL/collision, maksimum 4 manusia / 8 bot / 48 projectile, tanpa damage di luar arena/friendly fire, 3 HP + revive saat wave / kalah bila semua down / timeout 180 s, maksimum 3 wave, seq replay ditolak.
- [ ] Implementasi engine wave terbatas dengan authority server pada 20 Hz; input dx/dy/aim/fire+seq, bukan posisi/hit/kills. AI sederhana dan dibatasi; spawn tidak overlap pemain. Satu senjata dan outcome history, tanpa PvP/progression/toko senjata.
- [ ] Implementasi scene world dan HUD yang terbaca, aim/crosshair/hit flash pendek/effect pool; pointer touch kiri/kanan independen + keyboard-only. Input release ≤250 ms saat disconnect/hidden; tidak ada tembakan tak sengaja setelah modal ditutup.
- [ ] Integrasi batas participant/spectator, activity lease, result serializer/history/guest completion. Duck lama tetap tersedia di popup terpusat dan podiumnya tidak berubah.
- [ ] Test authority/co-op 4 client melalui WS+DB aktual, guest golden, reconnect/all-down/host leave, modal/lost capture, tanpa replay SFX. Benchmark jumlah entitas terburuk sebelum review/commit.

### Task 11: Verifikasi gabungan dan bukti optimasi

**Files:** create `apps/server/internal/application/activities_load_integration_test.go`, `apps/web/tests/activities.e2e.spec.ts`, `docs/checkpoints/evidence/CP8/activities-verification.md`; update PROGRESS/HANDOFF/runbook dan checkbox plan hanya berdasarkan bukti.

- [ ] Jalankan `npm test`, `npm run typecheck`, `npm run lint`, `npm run build`; pastikan runner dan compiler mencakup semua file baru. Go pinned: full `go test -count=1 -race -tags=integration ./...`, `go vet ./...`, build; private DB env tetap ignored.
- [ ] Skenario A, 20 user: race 5, penalty 2, blaster 4, pingpong 2, claw 1, fishing 4, penonton 2. Skenario B mengganti race/penalty dengan 4 free rider dan pengguna bola bebas; jangan menjalankan casual di arena yang semestinya steril. Tambah 60 client / 3 world untuk budget server/network.
- [ ] Rekam frame-time p50/p95/p99, stalls, tick cost, bytes/sec, ukuran chunk gzip, retained heap, jumlah timer/listener/audio/entity. Bandingkan baseline Task 0 dan budget spec; semua angka punya device/build/duration. Profil 15 menit dengan 20 siklus lifecycle; sambungkan ke T16 soak 60 menit sebelum beta, jangan menyebut load smoke sebagai soak.
- [ ] Uji main thread pada desktop + mobile fisik; emulator/throttle dicatat sebagai proxy. Perbaiki bottleneck dengan cache/culling/pools/lazy loading/interpolation, rerun hanya skenario yang dipengaruhi. Jika belum memenuhi target, laporkan gap, jangan diam-diam mengurangi rules/authority atau menaikkan budget.
- [ ] Regression: 5 lap / 5 slot yang sudah ada, pilihan penalti rahasia, Duck compensation/podium/history, isolasi akun/guest, invitations/chat/moderation/voice gates. Authenticated browser/LiveKit acceptance tetap pending bila user belum membuka gate tersebut.
- [ ] Fresh independent review seluruh diff, perbaiki critical/important + focused regression, simpan screenshots/hasil command/keterbatasan. Commit code + plan + evidence, verifikasi status/SHA/tree; publish hanya ketika sesuai instruksi eksekusi. Jangan menaikkan checkpoint ke PASS tanpa seluruh acceptance yang diwajibkan.

## Self-review plan

- Seluruh permintaan dipetakan ke task; pilihan top-down/solo-co-op tercatat, tidak diganti gallery atau PvP.
- Interface area/input/audio ditetapkan sebelum task yang bergantung padanya; physics loop dan kepemilikan scene terpisah, tidak membocorkan pilihan penalti atau authority ke klien.
- Batas entitas/traffic/memory/audio memiliki target dan test gate. Framework berat/3D/worker bukan prasyarat.
- Safe area/pembuatan ulang room/start yang belum pasti/hasil terlambat dan batas spectator memiliki regression eksplisit. Modal/minimap memiliki app-ordering test yang pernah gagal pada integrasi sebelumnya.
- Saat penulisan awal plan belum dieksekusi; hasil tes 103 frontend / 242 backend di atas adalah baseline sebelumnya, bukan bukti fitur baru. Review desain oleh fresh builder menemukan satu klarifikasi: Duck lama sengaja tetap memakai popup terpusat; spec kini menyatakannya eksplisit.

## Verifikasi dokumen — 28 Oktober 2026

- Pemeriksaan lokal: tiga link relatif antara plan/spec/HANDOFF valid; Task 0–11 lengkap dan berurutan; tidak ada checkbox implementasi yang ditandai selesai.
- Scope commit hanya plan, spec dan tautan HANDOFF. Tidak ada perubahan kode, dependency, asset atau database; tes aplikasi tidak dijalankan ulang untuk perubahan dokumentasi ini.
- Pemeriksaan whitespace menggunakan `git diff --cached --check` sebelum commit. Benchmark dan test case di atas masih rencana, bukan hasil yang sudah lulus.

## Handoff

Rekomendasi eksekusi: subagent per task dengan fresh review setelah integrasi/lifecycle dan setiap game, satu coordinator untuk integrasi. Pendekatan ini menangani perubahan lintas Go/TS/DB/protocol secara bertahap. Implementasi sedang berjalan berdasarkan instruksi user; lanjutkan dari task berikut yang belum selesai. Ledger eksekusi ada di `.superpowers/sdd/2026-10-28-world-activities-adjustment/progress.md`, bukti portabel ada di `docs/checkpoints/evidence/CP8/activities-*.md`.
