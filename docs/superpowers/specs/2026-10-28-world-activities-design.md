# Kampung Senja — aktivitas di dalam world

Status: proposal untuk review, 28 Oktober 2026. Permintaan saat ini adalah **plan dulu**, bukan implementasi. Baseline publik `a71fd85` pada `main`. Angka/rules yang diberi label usulan belum dianggap disetujui pengguna.

## Tujuan dan keputusan pengguna

World menjadi tempat bermain dan nongkrong yang terlihat bersama, bukan sekadar pintu menuju game popup. Tetap fullscreen, tema kampung autumn, teks semi-English, web desktop/mobile, dan ringan.

Permintaan: pingpong, capit boneka, mobil/sepeda santai di lintasan, race tetap tersedia dengan lintasan steril, bola bebas di lapangan yang berhenti saat penalti, pertandingan terlihat di lokasi dan bisa ditonton, shooting yang lebih seru, mancing pura-pura tanpa history, minimap bisa diklik, sound effect, dan semua popup di tengah.

Dua pilihan telah dikonfirmasi pengguna: shooting berupa **top-down arena dengan karakter bergerak sambil menembak**; mode awal **co-op melawan bot, dapat dimainkan solo**. PvP tidak termasuk tahap ini.

## Fakta dari kode saat ini

- Runtime world ada di `apps/web/src/world/controller.ts`, bukan renderer isometrik lama. Art sudah memiliki meja pingpong, mobil parkir, bola dan bangku penonton; semuanya masih dekorasi.
- Race/penalty/Duck sekarang memiliki scene React/SVG terpisah dalam modal. Menutup panel dapat melepas kontrol, sementara engine/deadline tetap berjalan. Subscription view saat ini terikat panel.
- Minimap digambar di canvas utama, tanpa hit-test khusus; tap dapat menjadi tujuan tanah di belakang minimap. Peta besar hanya punya tombol tujuan, belum click-to-path pada gambarnya.
- Preference audio tersimpan, tetapi belum ada service sound effect game. World/game mempunyai beberapa loop render/state sendiri; game modules dimuat eager dari LobbyPanel.
- World server 20 Hz memeriksa collision statis. Race 30 Hz dan lima lap/lima slot sudah ada. Belum ada collision area dinamis untuk sterilisasi lintasan/lapangan.
- Lobby start menggunakan reserve/persist/activate dan bisa mengalami status persistence belum pasti. Penguncian area harus terintegrasi sebelum engine aktif, bukan efek samping broadcast sesudah start.
- Koordinat return lama di registry tidak sama dengan entrance map baru. Bentuk track dekoratif juga berbeda dari kontrak fisika race; keduanya harus diselaraskan.
- Style modal tidak menetapkan centering eksplisit, sementara CSS reset mengatur margin 0. Ini hipotesis penyebab posisi popup yang perlu dibuktikan di browser saat implementasi.

## Pendekatan yang direkomendasikan

1. **World sebagai scene utama + HUD aktivitas kecil** (rekomendasi): satu renderer menampilkan aktivitas, pemain dan penonton; server tetap menentukan aturan/hasil. Lobby, pengaturan, chat dan detail hasil adalah popup terpusat. Membutuhkan pemisahan lifecycle game dari panel, tetapi sesuai tujuan sosial dan menghindari dua render world yang berjalan bersamaan.
2. Mempertahankan semua game di popup: lebih sedikit perubahan, tetapi pertandingan tetap terpisah dari penonton yang ada di lapangan; tidak memenuhi tujuan utama.
3. Mengganti engine dengan 3D/physics framework penuh: memungkinkan simulasi lebih rumit, tetapi menambah download, CPU dan risiko migrasi tanpa kebutuhan yang sepadan. Tidak direkomendasikan.

Proposal ini menggunakan pendekatan pertama, Canvas2D + TypeScript/React + Go yang sudah ada. Tidak mengubah source repo Patra World, akun, undangan, teman/moderasi, voice, history lama, atau podium Duck menjadi fitur lain.

## Pengalaman setiap aktivitas

| Aktivitas | Proposal pengalaman awal | Aturan multiplayer / hasil |
| --- | --- | --- |
| Pingpong | Meja terlihat di halaman arcade; solo vs BOT atau 1v1; gerakkan paddle kiri/kanan, satu tombol serve; penonton melihat rally di meja | Satu meja, dua peserta. Usulan first-to 7, selisih 2, cap 11, batas 120 s; seri jika skor sama pada deadline. Hasil match memakai history yang sudah ada, tanpa leaderboard baru |
| Capit boneka | Kabinet 2D, arahkan capit lalu Drop; posisi dan timing memengaruhi keberhasilan; animasi boneka preset | Satu pemain per mesin, giliran maksimal 20 s; tanpa antrean otomatis tahap awal. Penonton melihat animasi yang sama. Gratis, hadiah kosmetik sementara, tanpa uang/inventory/history |
| Free ride | Ambil sepeda/mobil di parkir, keliling lintasan, turun di titik aman | Usulan pool dua sepeda + dua mobil per dunia, satu pengendara tiap kendaraan; tidak bisa keluar lintasan/akses atau menabrak avatar. Server membatasi kecepatan; tidak menghasilkan history |
| Race | Start tetap dari lintasan, lima slot/lima lap dan BOT tetap ada; kart bergerak pada track yang sama yang dilihat penonton | Area ditutup sebelum countdown, semua pengguna santai keluar ke perimeter aman. Saat peserta pertama finis, track tetap steril sampai seluruh race berakhir |
| Bola bebas | Satu bola di lapangan; mendekat dan Kick/tap untuk menendang, bisa oper santai | Server menentukan posisi/impuls; tanpa skor/history. Bola di-reset dan disembunyikan saat lapangan dipakai penalti |
| Penalti | Penendang, penjaga gawang dan animasi bola tampil di lapangan; pemilihan empat zona lewat HUD kecil | Aturan 3 s pilihan + 1.5 s reveal tetap. Pilihan lawan tidak dibocorkan sebelum reveal. Pengunjung menonton dari luar boundary dan tidak mengendalikan bola pertandingan |
| Blaster co-op | Arena top-down berpagar di sisi halaman arcade; WASD+aim/fire, atau touch dua jempol; solo/teman melawan bot | Usulan 1–4 pemain, tiga wave, batas 180 s; maksimal 8 bot dan 48 proyektil aktif. Satu senjata, tiga HP, friendly fire off. Tidak ada PvP atau damage ke pengunjung. Hasil run masuk history, tanpa leaderboard baru |
| Fishing | Titik di tepi sungai: Cast → tunggu bobber → Reel → hasil lucu acak (ikan/sepatu/kaleng/dll.) | Usulan empat spot, satu pemain per spot. Cukup emote + pesan sementara. **Tidak membuat history, inventory, XP, leaderboard, atau row hasil DB** |

Semua angka di tabel adalah default usulan untuk menjaga scope; dapat diubah saat review plan. Kecepatan ride awal: sepeda 6 tile/s, mobil 8 tile/s, hanya di jalur khusus; sweep collision mencegah menembus pagar pada langkah cepat. Pemain tetap berjalan dengan kecepatan lama 4.5 tile/s.

Blaster: tiga wave masing-masing maksimal 8 bot, spawn jauh dari pemain, tidak ada respawn musuh tanpa batas. Pemain dengan HP 0 menunggu wave berikutnya; jika seluruh tim jatuh, run selesai kalah. Revive otomatis di awal wave berikutnya dengan 3 HP; bila 180 s habis, kalah/timeout. World members dapat menonton dari luar arena; tidak menjadi peserta di tengah run. HP 0/penonton tidak bisa menembak. AI sederhana chase/strafe dengan line-of-sight, tanpa pathfinding ulang per frame.

Pingpong: dua manusia harus ready atau pemain secara eksplisit memilih BOT. Putus koneksi menetralkan paddle, deadline tetap berjalan; reconnect memakai grace 15 s yang sudah ada, lalu forfeit bila tidak kembali. Tidak otomatis mengganti manusia dengan BOT. Penonton pasif tidak mendapat row history.

Claw: grid hadiah preset dan gerakan capit kinematik, bukan tumpukan rigid body. Hasil ditentukan jarak grip terhadap pusat hadiah saat Drop dengan toleransi dalam kontrak; tidak ada monetisasi atau peluang berbayar. Fishing: random visual cukup dari server di world privat atau lokal di solo; tampil maksimal 3 s, tanpa log tangkapan permanen.

Duck lama tetap tersedia lewat kabinet/quick play sebagai **legacy game di popup terpusat**, tidak perlu scene adapter/spectator baru pada tahap ini; ini pengecualian eksplisit dari aktivitas world baru. History/podium/ruleset lama tidak diubah. Blaster menjadi aktivitas shooting utama baru dan terlihat langsung di world. District visitation tetap menghitung tujuh area, terpisah dari jumlah game/mesin.

## Aturan area steril

State spasial terpisah dari status penyimpanan hasil/lobby:

`OPEN → CLEARING → RESERVED → ACTIVE → RESETTING → OPEN`

- OPEN: aktivitas santai boleh. Penonton dapat melewati area yang tidak dibatasi.
- CLEARING: mulai saat permintaan Start diterima; tutup admission casual dan kirim pemberitahuan singkat 3 s. Tolak input kendaraan/bola baru. Server pindahkan pengunjung yang masih di dalam ke safe return di perimeter dan kembalikan kendaraan/bola ke slot.
- RESERVED: world owner memberi ACK bahwa area kosong dan terkunci. Persist/start tetap mengikuti durability gate yang ada; baru jadwalkan countdown/Engine.Start setelah kedua gate terpenuhi. Tidak ada round dimulai hanya karena animation selesai di klien.
- ACTIVE: participant projection saja di arena. Pathfinding, WASD, reconnect dan destination click pengunjung tidak boleh masuk. Penonton berdiri di perimeter; tidak ada teleport atau posisi klien yang dipercaya.
- RESETTING: setelah seluruh gameplay berakhir/abort definitif, tampilkan celebration maksimal 2 s, pindahkan projection peserta ke safe return, reset objek santai, lepas lease generasi yang sama.
- Finalization hasil yang lambat tidak menutup lapangan selamanya. Area dapat dibuka setelah simulation selesai/reset, sementara claim akun/lobby/history tetap mengikuti mekanisme lama. Rematch harus reserve/clear ulang. **Uncertain START persistence tetap terkunci sampai reconciliation menentukan active atau abort**, berbeda dari uncertain result persistence.

Lease mencakup worldId, runtime incarnation, areaId, generation dan matchId. ACK/release lama tidak dapat membuka area milik match baru. Runtime dunia yang dibuat ulang wajib memulihkan desired leases dari owner sebelum menerima movement/snapshot, atau dipertahankan hidup selama lease ada. Penguncian track dan pitch independen; dua pertandingan boleh berlangsung bersamaan jika pemain berbeda.

Safe returns dan area boundaries berasal dari kontrak map bersama, dijamin di luar area steril serta dapat dijangkau. Semua jalur masuk/keluar mempertahankan aturan private-world membership, single active activity, spectator dan voice exclusion. Aksi sesi lama setelah logout, kick, pindah world atau reconnect ditolak.

## Scene, input, UI dan minimap

- Satu persistent activity store/subscription per aplikasi, satu world render loop. Panel React menampilkan HUD/status, tidak memiliki umur engine atau jaringan pertandingan.
- Rendering scene memakai snapshot publik; race memakai transform uniform dari kontrak race ke bounds track. Paint track berasal dari kontrak yang sama agar kart tidak tampak keluar jalan. Collision statis/art solids/voice map identity tetap sinkron.
- Avatar peserta tidak digambar ganda sebagai avatar berjalan dan aktor pertandingan. Penonton biasa dapat melihat permainan dari perimeter tanpa otomatis menjadi registered spectator yang memengaruhi voice; input tetap hanya milik peserta terotorisasi.
- Satu input owner: walk, ride, race, penalty, pingpong, claw, blaster, fishing, modal/text. Pindah mode, blur, hidden tab, lost capture, disconnect, modal atau text focus menetralkan input. Snapshot/deadline server tetap berjalan.
- Shooter desktop: WASD gerak, pointer aim, tahan klik fire. Keyboard-only: WASD gerak, arrows aim, Space fire. Touch: stick kiri gerak, stick kanan aim/fire, pointerId independen. Target tidak kebal karena user membuka menu; tidak ada pause global multiplayer.
- Minimap menjadi surface terpisah yang dapat difokuskan. **Tap/klik pojok minimap membuka peta besar di tengah; klik/tap lokasi pada peta besar membuat jalur berjalan**, bukan teleport. Tombol tujuan tetap tersedia untuk keyboard/screen reader. Marker menunjukkan area sedang dipakai; klik area terkunci mengarah ke entrance/perimeter atau menjelaskan alasannya. Tutup dialog/cleanup dulu, baru kirim destination pada frame berikutnya.
- Semua popup (Map, Menu, akun, Settings, Friends, Chat, History, Podium, voice/lobby dan detail hasil) dipusatkan di viewport. Chat yang sekarang aside dipindah ke modal terpusat dengan ID/form/mount tetap kompatibel. HUD skor/kontrol di world bukan popup, tetap ringkas. Tidak ada sidebar permanen.
- Modal punya batas `100dvh`, safe-area, scroll internal, target sentuh ≥44 px, focus trap/restore dan Escape yang konsisten. Dialog tertutup benar-benar tersembunyi. Keyboard ponsel dan zoom 200% tidak menutupi tombol utama.
- Tutup HUD tidak menghapus match/history atau berhenti berlangganan scene; Exit/Leave tetap aksi eksplisit sesuai aturan game. Hasil ringkas boleh muncul sebagai banner; detail hasil dibuka sebagai popup terpusat.

## Sound effect

Satu AudioService dengan AudioContext yang dibuka oleh gesture nyata. Efek singkat untuk paddle/ball, goal/save, countdown/finish, mesin capit, kendaraan, blaster hit/fire dan fishing splash/catch. Tidak perlu musik latar tahap awal.

Preference Audio yang sudah ada mengendalikan seluruh SFX. Gunakan bank lokal kecil atau synth pendek, cache sekali, maksimum 8 suara bersamaan; ulang suara tembakan dibatasi. Matikan loop saat tab hidden/keluar aktivitas. Deduplikasi eventId/revision agar reconnect tidak memutar ulang efek lama; hit/goal berasal dari hasil server, bukan tebakan klien. Attenuation jarak sederhana untuk penonton; tanpa audio spasial 3D. SFX tidak mengubah mute/mic LiveKit. Semua feedback penting punya visual/text.

## Arsitektur dan kompatibilitas yang diusulkan

- Static map tetap 64×64, subunits 1024. Identitas asset berubah ke `kampung-senja-v2` bila layout baru digabung; schema map tetap v1. Embed Go dan voice references digenerate bersamaan. Hash match lama tetap utuh.
- Rilis aktivitas menggunakan wire envelope **protocol 2**, dengan decoder/golden JS+Go dan handshake yang jelas; klien lama menerima incompatibility/reload, bukan snapshot yang diam-diam salah. URL REST akun/history tetap. Historical result protocol 1 tetap terbaca.
- Kontrak baru `activities.v1.json` menetapkan areaId/bounds, safe returns, vehicle slots, physics caps dan kapasitas. Snapshot aktivitas menjadi event tersendiri, bukan field liar pada world snapshot avatar yang strict.
- Casual simulation milik world owner 20 Hz; match baru pingpong/blaster lewat lobby engine 20 Hz; race tetap 30 Hz. SQL/network tidak berjalan di owner tick. Scheduler/publisher dibatasi, input continuous masuk movement budget, aksi discrete punya rate limit terpisah.
- New match types membutuhkan migration baru untuk allowlist DB, history serializer/query, client completion/history detail dan protocol validators. Jangan edit migration lama 004 atau menimpa metadata hasil lama. Claw/ride/freeball/fishing tidak memakai completion pipeline.
- Guest tetap solo/lokal; tidak membuat kredensial/session tiruan. Local model mengikuti kontrak/golden yang sama. Restore/reset/logout harus melepaskan activity ownership, input, listener, suara dan state visual secara generation-safe.

## Target performa dan cara membuktikannya

**Target awal, bukan hasil benchmark yang sudah terbukti.** Ambil baseline sebelum perubahan dan catat device/OS/browser/resolusi/quality/build. Target mobile diuji di perangkat fisik kelas menengah yang ditentukan sebelum benchmark; desktop throttling hanya proxy, bukan pengganti bukti perangkat.

| Aspek | Budget usulan / acceptance |
| --- | --- |
| Frame | Desktop target 60 fps, mobile minimal 30 fps; p95 kerja main-thread per frame ≤12 ms desktop / ≤25 ms mobile, pada world 20 avatar dan aktivitas ramai |
| Server | Hub tetap 20 Hz, race 30 Hz, aktivitas baru 20 Hz; p95 owner tick ≤25 ms pada 60 client / 3 world; catch-up aktivitas baru maksimal 4 step, tidak memutar simulasi kosong |
| Jaringan | Snapshot peserta 10–15 Hz, penonton maksimal 5 Hz; input maksimal 20 Hz. Tambahan traffic aktivitas ≤40 KiB/s per peserta aktif dan ≤12 KiB/s penonton; total termasuk hub juga dilaporkan |
| Entitas | Per world: 4 free-ride vehicles, 1 bola bebas, 1 meja pingpong, 1 mesin capit, 4 spot fishing, 1 blaster session; Blaster ≤4 manusia / 8 bot / 48 proyektil; particle pool ≤96 |
| Bundle | Tambahan initial JS ≤75 KiB gzip; masing-masing chunk game baru ≤50 KiB gzip. Lazy load ketika aktivitas dibuka/dekat; shared chunks dihitung sekali, bukan menyembunyikan kenaikan total |
| Audio | SFX bank ≤200 KiB compressed, dimuat setelah intent; maksimum 8 voice, tidak autoplay atau membuat AudioContext tiap klik |
| Memory/lifecycle | DPR ≤2, satu cached world background, hindari duplikasi canvas 2560×1920 (~18.75 MiB RGBA). Setelah 20 join/leave/open/close cycles, listener/timer kembali baseline dan retained heap tidak naik terus. Target heap ≤baseline + nilai terbesar antara 10% baseline dan 5 MiB, setelah settle/GC pengukuran |
| Idle/hidden | Tidak ada world render/audio loop aktif saat tab hidden; server authority/reconnect tetap benar. HUD DOM cukup ≤10 update/s, bukan 60 render React/s |

Optimasi utama: culling berbasis bounds, cache static sprites, bounded pools, interpolation, subscription sesuai area/kebutuhan, batched snapshots, dirty minimap/static layers dan asset lazy-load. Metadata status arena tetap tersebar agar semua klien tahu area steril. Saat culling subscription keluar/masuk, selalu minta full state dengan generation baru; tidak memutar ulang event lama.

Jika target gagal, profil dulu, kurangi efek/resolusi/frekuensi visual dan perbaiki hot path. Tidak mengurangi correctness, jumlah lap, authority, atau menaikkan budget diam-diam. Tidak menambah WebGL framework, worker, general physics, asset pack besar atau service baru tanpa bukti bottleneck.

## Batas dan urutan

Pertama UI/minimap/audio foundation dan kontrak/lifecycle, lalu scene + area steril, free ride/race, freeball/penalty, aktivitas kecil, terakhir blaster dan pengukuran ramai. Pingpong/claw/fishing dapat dikerjakan paralel setelah seam stabil; integration tetap satu coordinator.

T14 re-review, ordered reconnect transitions dan uncertain-start recovery yang menyentuh ownership menjadi prasyarat sebelum shared-area release. T16 soak 60 menit/backup-restore dan acceptance akun/LiveKit/perangkat tetap ditandai terpisah; proposal ini tidak mengubah CP1 sebagai satu-satunya checkpoint resmi PASS. Tidak ada implementasi atau deployment pada turn perencanaan ini.
