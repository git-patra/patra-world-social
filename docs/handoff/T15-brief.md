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


### T15 — Map district, accessibility dan mobile polish

**Files:** contracts village map, web art/camera/avatars/UI/theme; tests e2e accessibility/map; docs screenshot evidence.

**Interfaces:** Memakai `MapDefinition`, `drawWorld` dan input adapters T2/T5; tidak mengubah aturan networking.

- [ ] Tambah regression checks untuk semua entrance reachable, seat/arena spawn aman, nama/status tidak tertutup panel, keyboard/chat focus, viewport kecil, zoom/pointer mapping.
- [ ] Jalankan checks untuk melihat lokasi/komponen yang belum memenuhi desain, bukan snapshot brittle seluruh canvas.
- [ ] Tata central village, race park, penalty field, duck arcade/podium, warkop dan jalan; cache static chunks/cull viewport; optimalkan label, touch steering/aim, focus, error/loading states.
- [ ] Review screenshot pada 390×844, 768×1024, 1440×900 dan browser nyata; keyboard-only parcours, reduced-motion, font scaling, CPU throttling; catat perangkat/fps.
- [ ] Commit `feat: expand the village and polish accessible play`.

**Selesai:** visual tetap Patra World dan semua lokasi nyaman diakses di desktop/mobile.


## 4. Struktur dunia dan pengalaman utama

### Tata ruang konseptual

```text
                      RACE PARK
                lobby + tribun + portal track
                           |
   WARKOP -------- CENTRAL VILLAGE -------- PENALTY FIELD
 teras, dua meja      spawn, papan arah       lobby + tribun
                           |
                DUCK ARCADE + CHAMPION PLAZA
                           |
                  TAMAN / EKSPANSI BERIKUTNYA
```

Ini diagram hubungan lokasi, bukan gambar final atau koordinat final. Mini-game memiliki scene sendiri; orang yang tidak ikut tetap dapat nongkrong di hub. Satu world tetap satu komunitas privat meskipun beberapa scene aktif sekaligus.

| Space | Isi dan interaksi | Rancangan ukuran awal |
|---|---|---|
| Central Village | Spawn, core bangunan lama, petunjuk, daftar teman, papan lokasi | Area 20×20 |
| Race Park | Zona kumpul lima kursi, host/ready, status arena, tribun, pintu track | Area 18×14; track scene terpisah |
| Penalty Field | Dua slot pemain, tribun, papan skor, rematch | Area 14×12; pitch scene terpisah |
| Duck Arcade | Empat kabinet aktif paralel, panel skor, podium 1–3 | Area 14×12 |
| Warkop | Warung, teko/kopi, meja kayu, kursi plastik, lampu gantung, dua meja percakapan | Area 18×14 |
| Jalur ekspansi | Jembatan, taman, tempat duduk, batas map yang jelas | Sisa map; tidak semua tile harus terisi |

Dimensi area merupakan footprint perencanaan dan harus ditata tanpa overlap pada map 64×64. Collision bangunan menggunakan footprint nyata, entrance, dan interaction zone; tidak lagi satu tile untuk seluruh bangunan.

- Kamera mengikuti karakter, ada zoom terbatas, tombol kembali ke karakter, minimap, dan daftar lokasi.
- Semua avatar diurutkan menurut kedalaman, diberi nameplate dan indikator status: available, playing, AFK, atau voice.
- Pemain hub tidak menghalangi jalur satu sama lain. Collision berlaku pada lingkungan. Tidak ada teleport bebas menuju teman; tombol “temui” memberi arah/path bila scene sama.
- Race scene memakai mobil mini; penalty dan duck mempertahankan art hangat yang selaras. Tidak perlu pindah ke 3D atau engine lain sebelum profiling menunjukkan kebutuhan.
- Seluruh konten CV/portfolio, biodata karier, galeri proyek, tautan promosi, PDF resume, dan generator CV dikeluarkan dari produk. Area bermain menjadi layar utama; art bangunan lama tetap dikenali dengan nama/fungsi desa.
- Ketik di chat tidak menggerakkan avatar; kontrol sentuh, Escape, focus, mute, dan navigasi lokasi tersedia di luar canvas.

### Alur guest

`Buka → Main sendiri → pilih nama/avatar lokal → masuk map lokal → pilih game → bermain vs bot/latihan → history di perangkat`.

Guest mendapat seluruh lokasi dan tiga game. Warkop dapat dijelajahi tanpa terhubung ke manusia, voice, atau chat online. Tombol “Main dengan teman” mengarah ke daftar/login. History guest maksimal 100 sesi di IndexedDB, dapat dihapus; kehilangan data browser berarti kehilangan history. Saat login, avatar/preferensi boleh disalin, tetapi skor guest tidak pernah menjadi skor resmi.

### Alur akun dan undangan

`Daftar/login → pilih world yang diikuti atau buat world → undang teman → teman login → redeem undangan → masuk world → bermain/nongkrong → hasil tersimpan`.

- World memiliki ID persisten; data anggota, history dan papan skor tetap ada setelah semua orang keluar.
- Daftar world hanya menampilkan milik/keanggotaan sendiri; tidak ada public directory pada versi pertama.
- Undangan dibuka melalui link, username yang sudah terdaftar, atau daftar teman. Pengiriman otomatis hanya notifikasi dalam aplikasi; salin link dilakukan pengguna untuk dibagikan.
- Link default berlaku 24 jam dan satu kali redeem. Token acak minimal 128 bit, hash disimpan, token mentah tidak masuk log. Opsi target username mengikat redeem kepada akun tersebut.
- Login tidak menghabiskan undangan. Keanggotaan baru dibuat setelah redeem sukses; duplikasi request dari akun sama idempotent.
- Jika world penuh, redeem boleh menambahkan membership bila batas anggota belum tercapai; join online menampilkan “dunia penuh” tanpa membuat koneksi/seat tambahan.
- Mencabut link mencegah redeem baru. Mengeluarkan anggota mencabut membership, memutus world/game/voice dan menolak koneksi berikutnya. Keduanya tindakan berbeda.
- Owner tetap owner saat offline; peran host pertandingan berpindah terpisah. Owner dapat mengangkat moderator.

### Akun, teman, dan izin

Username case-insensitive, 3–24 karakter `[a-z0-9_]`; display name 1–32 karakter, avatar dari pilihan yang disediakan. Password 12–128 karakter, tidak di-trim diam-diam. Tidak ada email wajib pada beta; saat daftar tampilkan recovery code sekali. Reset memakai username + recovery code, kemudian ganti code dan batalkan seluruh sesi lama. Tanpa password/recovery code tidak ada reset otomatis berbasis pertanyaan pribadi.

Friend request: cari exact username → kirim → terima/tolak; unfriend dan block. Friend tidak otomatis menjadi anggota world. Block menghentikan request/invite baru dan chat langsung dari user itu; bukan pengganti moderation world. Untuk beta tidak ada DM persisten, hanya undangan/notifikasi, world chat, dan table chat.

| Kemampuan | Guest | Member | Moderator | Owner |
|---|---|---|---|---|
| Solo lokal | Ya | Ya | Ya | Ya |
| Join world/game online/history akun | Tidak | Ya | Ya | Ya |
| Membuat world sendiri | Tidak | Ya | Ya | Ya |
| Membuat undangan ke world ini | Tidak | Tidak | Ya | Ya |
| Membuat/memulai lobby game | Lokal | Ya sebagai host lobby | Ya | Ya |
| Kick/mute chat di world | Tidak | Tidak | Ya, kecuali owner | Ya |
| Ubah moderator, transfer/archive world | Tidak | Tidak | Tidak | Ya |

Profil/history pribadi hanya milik user. Ringkasan pertandingan dunia dan leaderboard terlihat oleh anggota aktif world. Guest tidak dapat mengambilnya melalui API.


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
- Address T2 retained facade text CONNECT/P on repurposed Balai Desa, plus remaining T4 readability observations. Use real screenshot evidence for requested viewports; emulation does not prove physical Android/iOS device performance. Request missing device prerequisites only when necessary and continue independent work.

- T5 deferred minor: online Reset and brand action must not announce/label a return to plaza when the server-authoritative player stayed in place. Branch feedback on actual online behavior; preserve no-teleport guarantee.

## T7 guest UI minor
Disconnected guest sees “Memuat ketersediaan…” forever in LobbyPanel despite no world connection. Replace with truthful disconnected/locked copy and verify final responsive dialog; evidence docs/checkpoints/evidence/CP4/t7-guest-preview.md.

## T9 visual minor
Penalty zone buttons render a stray leading0 text node beside/overlapping zone1–4 labels (AX says “0 1 kiri atas”). Clean conditional JSX/text and verify screenshot in final UI pass; evidence CP4/t9-guest-results.png and t9-guest-preview.md.

## User clarification — web including mobile
User replied to device-availability question: "web version aja semua termasuk mobile". Product target is responsive browser web across desktop/mobile; no native app deliverable or native app tooling. Continue responsive viewport/touch/browser testing for the web build. Reply does not identify a physical device/OS/browser or supply measured physical-device evidence; do not claim those measurements. No repeat device question now. Preserve honest distinction between browser viewport checks and measured physical mobile performance.

## New dependency seams to preserve
T13 uses a separate voice-table contract linked to exact map ID/version/checksum and reachable/nonoverlapping geometry. If T15 changes navigation geometry/version/checksum, update and validate both frontend/backend copied contracts together; existing recorded match hashes remain immutable. Inspect hard-coded application MatchStart map version before claiming a version bump. Prefer visual-only district polish where existing navigation already satisfies the approved layout; do not rewrite geometry gratuitously.
T14 adds guest name/avatar/preferences (including controls and reduced motion), actual local history for all three engines, Settings dialog and entry buttons. Preserve these paths and async mode cleanup while polishing layout. Initial T14 root CUA confirms three-game persistence/reload/reset; don't replace game-only UI with portfolio/CV content. Revalidate accepted T14 SHA/interfaces at dispatch.

## Actual T14 browser observations for polish
Guest game history now uses Indonesian labels and gameplay details; keep these in normal UI rather than map/protocol IDs. Native preview was narrow during game checks; Duck/race were reachable by panel scrolling, but offline lobby cabinets still displayed “Memuat ketersediaan…” despite guest mode. That existing issue is already assigned to T15; replace it with an accurate disconnected/guest state, not an endless loading message. Preserve keyboard/touch controls when settings select both; test focus and pointer mapping after viewport/layout changes. Root browser evidence CP7/t14-guest-preview.md records current functional baseline.
