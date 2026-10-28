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


### T16 — Reliability, load test dan paket beta terpisah

**Files:** `tools/load/*`, deploy files, health/readiness/drain di server, metrics/logging, docs operations/verification; integration chaos tests.

**Interfaces:** `/health/live`, `/health/ready`, private metrics; `Drain(ctx) error`; startup `RecoverPending`; config origin/DB/media secrets dan rules versions.

- [ ] Uji scripted 60 clients di 3 worlds, tiap world race 5 + penalty 2 + duck 4, tambahan viewer/movement/chat; voice 48 participants diuji terpisah agar resource matrix masuk akal. Simulasikan delay, packet loss/disconnect, slow clients, DB outage, process restart dan provider failure.
- [ ] Jalankan `go test -race ./...`, frontend checks/unit, integrasi, E2E, production build; pastikan seluruh critical acceptance spec punya evidence.
- [ ] Buat container/binary deploy, contoh Caddy origin baru, DB migrations/backup, readiness, drain, release rollback dan secrets configuration; uji pada environment lokal dan staging yang memang sudah tersedia serta diizinkan. Jangan mengubah Caddy/VPS produksi atau mengarang domain/credentials.
- [ ] Jalankan soak 60 menit dan restore backup; catat p95 tick, FPS, CPU/RAM/network, queue/goroutine growth; uji keamanan guest/IDOR/session/voice replay. Jika kapasitas meleset, kurangi target beta secara eksplisit sebelum rilis.
- [ ] Commit `ops: prepare tested isolated beta deployment`; serahkan paket beta beserta hasil validasi dan runbook rollout/rollback. Deploy publik, push remote, provisioning layanan berbayar dan perubahan hosting masuk R1 terpisah setelah target ditentukan pengguna.

**Selesai:** paket beta terukur dan dapat dipulihkan pada environment yang diuji, source/hosting portfolio tidak tertimpa. Hasil device/provider yang belum diuji tetap pending; produksi belum dirilis.

## 10. Data, history dan finalisasi

| Tabel | Isi inti |
|---|---|
| `users`, `profiles` | Username canonical unik, password hash, display/avatar, status |
| `sessions`, `recovery_codes` | Hash token, user, expiry/revocation, recovery sekali pakai |
| `friendships`, `user_blocks` | Request/accept, pasangan user unik, block |
| `worlds`, `world_memberships` | Owner, map version, role, membership status |
| `world_invites` | Hash token, target, expiry, redeemed user, revoked time |
| `world_bans` | User yang dilarang join/redeem kembali ke world |
| `matches`, `match_participants` | Game/ruleset/map, start/end/status, user/bot, skor/posisi/flags |
| `penalty_turns` | Hasil pilihan per turn setelah resolve, timeout flags |
| `race_results`, `duck_results` | Lap/time/DNF; score/hits/shots/accuracy |
| `leaderboard_entries`, `champion_changes` | Best resmi dan histori pergantian #1 |
| `chat_messages`, `moderation_actions` | World chat, deletion/mute/kick audit |
| `result_jobs` | Hasil akhir immutable yang sudah durable dan perlu diproyeksikan |

UUID dipakai untuk ID publik; timestamp disimpan UTC dan ditampilkan sesuai timezone pengguna. Bot bukan akun user; peserta memiliki `actorType=human|bot`, `userId` nullable untuk bot.

- History akun: game, waktu, world, lawan/anggota, status, skor/posisi, durasi, aturan yang dipakai, BOT/assisted, detail hasil. Filter game/world/tanggal/status dan pagination.
- Histori per-user bersifat privat. Leaderboard world menampilkan ringkasan yang disepakati dan tidak membuka history akun di world lain.
- Match record dibuat `running` sebelum start resmi. Setelah selesai, hasil masuk `finalizing`; tulis `result_jobs` durable, lalu transaksi memfinalisasi match, detail, personal best dan champion. ID match unik menjamin retry tidak menggandakan hasil.
- Serialisasi pembaruan leaderboard per world/ruleset memakai row lock/transaction agar dua skor bersamaan menghasilkan urutan champion konsisten.
- Bila PostgreSQL tidak tersedia, pertandingan baru tidak dimulai. Match yang sudah berjalan boleh selesai dengan status “hasil belum tersimpan”; finalisasi menunggu/retry sampai 60 detik tanpa mengklaim tersimpan atau masuk leaderboard.
- Setelah 60 detik, arena dilepas dan UI menunjukkan **“hasil belum dapat dikonfirmasi”**. Batalkan write yang masih berjalan dengan context dan simpan daftar ID yang perlu direkonsiliasi, terbatas pada match aktif ketika gangguan dimulai; tetap tolak match baru sampai database pulih dan rekonsiliasi selesai. Jangan langsung menyatakan aborted karena write timeout dapat berarti commit sukses tetapi ack hilang. Saat database pulih: jika job/hasil durable ada, lanjutkan finalisasi; jika tidak ada, tandai `aborted_storage`, tanpa leaderboard. Full state/render memory boleh dilepas setelah fase ini. Jika proses crash sebelumnya, startup memakai aturan `aborted_server` untuk running tanpa job; tidak menjanjikan hasil yang belum durable dapat dipulihkan.
- Tidak ada janji recovery state game penuh pada beta. Setelah process crash, match `running` tanpa hasil durable ditandai `aborted_server`; tidak ada pemenang/rekor palsu. Job hasil yang sudah durable diproses ulang saat restart.
- Hasil resmi baru ditampilkan sebagai tersimpan setelah commit sukses. UI dapat menampilkan hasil sementara selama finalizing dengan label jelas.
- Match/account history disimpan sampai user menghapus akun/owner mengarsipkan world sesuai aturan produk; tidak ada purge diam-diam. Archive world menghentikan join baru dan mempertahankan history personal. Penghapusan akun mencabut sesi, menganonimkan actor pada history lawan dan menghapus profil/leaderboard miliknya; jika owner, harus transfer atau archive world lebih dulu.
- Telemetry tidak berisi password, recovery code, bearer token, pilihan rahasia sebelum reveal, isi voice atau isi table chat.


## 11. Keamanan dan batas akses yang relevan

- Password memakai Argon2id dan salt acak. Baseline minimum `m=19456 KiB, t=2, p=1`; benchmark dan tingkatkan sesuai CPU/RAM server. Jangan gunakan SHA-256 untuk password. [OWASP Password Storage](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html).
- Session berupa opaque token acak 256 bit dalam cookie `HttpOnly; Secure; SameSite=Lax`; database menyimpan hash token. Absolute expiry tujuh hari, idle expiry 24 jam; aktivitas socket yang sah ikut memperbarui idle, bukan setiap render.
- Login/reg/reset mempunyai rate limit per IP dan akun. Username tersedia boleh diketahui saat daftar, tetapi respons login/reset tidak membocorkan apakah password/recovery salah pada akun tertentu. Batasi hashing concurrent agar endpoint tidak menghabiskan RAM.
- REST mutasi memakai CSRF token dan validasi origin. WSS memeriksa exact origin allowlist, cookie session, membership dan status ban. Logout/session expiry menutup socket dan voice; pemeriksaan hak bukan hanya saat handshake. [OWASP WebSocket Security](https://cheatsheetseries.owasp.org/cheatsheets/WebSocket_Security_Cheat_Sheet.html).
- Rate limit frame/input, ukuran payload maksimum 16 KiB untuk command client, validator enum/range/string. Batas awal 60 movement commands/detik dan 10 control commands/detik; chat memakai limit sendiri. Server snapshots dapat lebih besar sesuai limit outbound terukur.
- Guest ditolak untuk world join, invite redeem, history akun, score resmi, dan token voice walaupun memanggil API langsung.
- Operasi world, ringkasan match dunia, leaderboard, world chat, dan media memeriksa membership aktif. **History pribadi** memeriksa bahwa user adalah peserta hasil tersebut, sehingga tetap bisa dibaca setelah keluar/dikeluarkan dari world atau world diarsipkan. Detail pribadi menampilkan data pertandingan yang memang diikuti, bukan chat/history peserta di match lain. UUID yang sulit ditebak tidak menggantikan otorisasi.
- Tidak ada currency, taruhan, pembayaran, atau hadiah uang dalam scope.


## 12. Deployment dan operasi

Repo baru memiliki frontend static, satu Go binary, PostgreSQL, dan konfigurasi media service. Domain/subdomain dan direktori deployment berbeda dari portfolio lama. Jangan menyalin script deploy lama dengan destination yang sama.

- Untuk development: Go + PostgreSQL lokal/Compose + Vite. Seed akun/world hanya lingkungan development, tidak membawa password tetap ke production.
- Untuk beta: Caddy melayani frontend dan reverse-proxy `/api` serta `/ws` ke Go pada origin yang sama. Database tidak dibuka ke internet. HTTPS untuk game dan voice.
- LiveKit managed mengurangi pekerjaan operasi media. Jika dipilih self-hosted, SFU/TURN/firewall/UDP/sertifikat dan uji jaringan seluler menjadi pekerjaan tambahan tersendiri; bukan sekadar menambah route Caddy.
- Health endpoint dipisahkan readiness database dan liveness proses. Ukur active worlds/connections/matches, tick duration, command queue, disconnect, reconnect, write failures, voice join failures dan bandwidth.
- Structured logs memakai request/match/world ID tanpa secret. Backup database harian, target awal RPO 24 jam dan RTO empat jam; lakukan uji restore sebelum beta.
- Deployment game melakukan drain: tolak match baru, tunggu match berjalan maksimal batasnya, kemudian ganti proses. Frontend/backend protocol version harus cocok; client lama diminta refresh dengan jelas.
- Migrasi database kompatibel dengan release sebelumnya. Rollback binary tidak boleh menghapus history baru atau menjalankan down migration destruktif otomatis.
- Kapasitas beta yang harus dibuktikan: **3 world × 20 user = 60 user game terkoneksi**, masing-masing satu race, satu penalty, empat duck sessions; voice diuji terpisah sampai dua meja × delapan peserta per world. Ini target validasi, bukan klaim VPS sekarang sanggup.
- Tidak ada angka biaya hosting dalam plan karena spesifikasi mesin, region, provider dan durasi voice belum dipilih. Estimasi media memakai total participant-minutes, bandwidth, peak concurrent, kemudian pricing resmi pada saat pemilihan.


## 13. Pengujian dan kriteria selesai

| Area | Skenario penerimaan wajib |
|---|---|
| Tema/map | Core world dikenali, empat distrik terhubung, tidak ada spawn terperangkap, kamera/zoom readable di mobile |
| Guest | Semua game lokal jalan; API online menolak guest; skor lokal tidak dapat diupload sebagai ranked |
| Auth/invite | Username case-insensitive; token expiry/revoke; dua redeem satu token; world penuh; reset membatalkan sesi |
| Multiplayer | Dua tab tidak menguasai satu avatar; reconnect resync; world A tidak menerima data B; reduced motion tidak teleport |
| Race | 1 manusia+4 bot; 5 manusia; pemain keenam; double Start; tepat 5 lap; checkpoint skip; DNF; host keluar |
| Penalty | 16 kombinasi zona; timeout salah satu/keduanya; deadline boundary; pilihan tersembunyi; five kicks dan sudden death adil |
| Duck | Hit ganda, target mati, tembakan terlalu cepat, replay input, rewind di luar batas, tie-break leaderboard konsisten |
| Voice | Izin ditolak, mic mute awal, lintas meja tak terdengar, keluar/kick menghentikan audio dan rejoin token lama ditolak |
| History | Finalisasi dua kali hanya satu hasil; database down; restart server; champion update serentak |
| UX | Keyboard/touch, focus dialog, chat tidak menggerakkan avatar, tab background tidak menghentikan match |

Pengukuran target: p95 processing tick di bawah interval tick; frame ≥30 fps pada perangkat mobile uji yang dicatat; soak 60 menit tanpa pertumbuhan goroutine/queue/memori tak terkendali; browser Chrome, Firefox, Safari desktop serta Android Chrome/iOS Safari diuji. Hasil nyata dan hardware dicatat, bukan diasumsikan dari tes unit.


## Coordinator preflight — finalize interfaces before dispatch
- Requirements above remain binding; insert actual accepted previous-task interfaces before implementation. Preserve original source repo and game-only autumn theme. No fake online players/results or claimed PASS from mocked provider/device evidence.
- Authenticated browser account/invite/credential lifecycle is explicitly deferred by user after automatic approval rejection. Do not run CUA/Playwright credential workflows or bypass through injected sessions. Isolated backend integration/protocol tests and pure guest browser gameplay checks are independent allowed work. Clearly separate test source from executed evidence.
- No subagents/staging/commits/docsPROGRESS edits; coordinator reviews and commits. No remote/push/public deploy/provision/purchase or production hosting changes.
- Preserve approved 20-online/world and 60-client/3-world load targets; do not silently lower acceptance to pass. If measured capacity misses target, report and fix; any scope reduction needs explicit user decision. Soak60minutes and restore require actual recorded runtime evidence. Provider/device prerequisites may keep CP8 pending even after all local code work.

## Carried T6 acceptance-source finding
- Before any authorized authenticated acceptance run, fix social.e2e.spec.ts typing and reconnect scenario: fill alone is not keyboard movement evidence; immediate offline→online plus existing Online text is not reconnect evidence. Use actual key events and observe interruption/new successful connection or snapshot, then assert deduplication. See T6 task review P3. Do not run credential flows while user deferral persists.

## Startup boundary to verify at final failure testing
T8 fix R1 covers settling result enqueue for already-visible match IDs. Also explicitly test a T7 initial Persist transaction whose new running match row is uncommitted when a replacement process starts recovery, then commits after recovery's candidate enumeration. Verify no running record/user/arena claim is left without an owning engine or recovery classification. This is a specific unverified cross-task boundary, not a confirmed defect or acceptance claim; initial-start lost ACK is in the existing contract. Use real isolated PG and actual startup path. Do not silently extend T8 scoped review with a broad new lease system; final branch review must resolve this before beta.

## Confirmed T9 review O1 — must resolve before beta
Read docs/checkpoints/evidence/CP4/t9-rereview1.md O1. Running match: processed disconnect100000; hold owner; timely resume101000 then new disconnect102000 coalesced; ingest102100. Current generic presence loses the timely resume and forfeits115000 rather than117000. Preserve sufficient ordered transitions; test no result at old cutoff and correct new cutoff, later final resume, and actually late first resume (cannot revive expired authority). Reviewer probe /private/tmp/t9-rereview1-probes/ was real lobby+penalty; preserve or reconstruct permanent regression if tmp gone. This pre-existing defect was outside fix-round diff, hence deferred here; it is not optional polish and must be resolved for beta/reconnect acceptance.

## Deferred T10 code-quality observation
T10 initial review Minor: new web/features/race/local.ts and Race.tsx effects are compressed into very long lines. Review/readability polish should format them into ordinary blocks; no physics change implied. Check whether subsequent necessary edits already resolved this before changing. Carried to final whole-branch review.

## Existing race assistance history edge
T10 scoped rereview1 distinguishes a pre-existing case: a personally finished racer may get BotAssisted=true on later disconnect while peers continue. Verify assistance reflects actual takeover during that actor racing interval; preserve true prior assistance. See CP5/t10-rereview1.md, personal-finish distinction. Carry into final review; not silently discarded.

## T11 deferred Minor protocol completion
`duck.shot_resolved` is listed/forwarded but lacks server producer; current shots return command.ack and snapshots only. Full finding R2 in CP6/t11-review.md: emit exactlyone accepted-shot resolution scoped byworld/match/seq, preserve ACK, avoid snapshot coalescing/doubleapply, hit/miss/rejection assertions through actual transport then WorldSocket fencing. Complete named contract before beta; no claimed current event implementation. T11review treats current scoring preserved, not an urgent gameplay outage. Final review must explicitly reconcile this item.

## User clarification — web including mobile
User replied to device-availability question: "web version aja semua termasuk mobile". Product target is responsive browser web across desktop/mobile; no native app deliverable or native app tooling. Continue responsive viewport/touch/browser testing for the web build. Reply does not identify a physical device/OS/browser or supply measured physical-device evidence; do not claim those measurements. No repeat device question now. Preserve honest distinction between browser viewport checks and measured physical mobile performance.

- T12 final verification attempt: default-parallel fullGo-race integration had T12 leaderboard22.427s/application38.627sPASS but existing chat TestChatPersistenceModerationIsolationAndResync failed atline84 sixth-message ErrRate assertion (test2.54s). Production SQL tokenbucket refills1token/sec; fixture assumes six sequential persisted calls stay below refill. No chat changes authorized. Ruling: one focused diagnostic, then full -race -tags=integration -p1 iffocusedgreen; preservefailedparalleloutput and explicitly label serialPASS ifobtained. Furtherfailure→diagnose, no repeatedretryuntilgreen. T16 harnessreliability obligation to remove implicitwalltimeassumption withoutweakeningproductionlimits.

- Task12: minor (deferred): R2 missing historical Duckdetail is coalesced into score0/hits0/shots0/completefalse by leaderboard/query.go, so latestsession UI fabricates zero-point incomplete facts. Beforebeta T16/finalreview must expose unavailable/unvalidated detail (nullable stats/availability) with no invented outcome, preserve immutable history and boardexclusion; extend existing missingdetailbackfillcase and actualview assertion. See task-12-review.md R2.

Verificationcoverage audit: tsconfig uses explicit include whitelist. T13 initially only voice/zones.ts was covered despite typecheckPASS; fixedvoiceprod/tests/E2E includes inround1. Rootchecked Duckproduction iscovered butDucktestfilesnotlisted. Confirm intendedfullTypeScriptproduction/testentrycoverage with actualtsc --listFilesOnly beforefinaltypecheckclaim; do not equate esbuildtranspile/testsPASS with typechecking.

- Task14: minor (deferred): M1 guest completion createdAt precedes race/Duck3s countdown but suppliedduration startsaftercountdown; finishedAt therefore3s early, potentiallymisorderscloselyfinishedresults/retention. T16 mustanchorwalltime toactualmonotonicfinish includingoffset, preserveidempotentretries/noactiveclockpersistence, deterministiccountdowntest. See CP7/t14-review.md M1.
