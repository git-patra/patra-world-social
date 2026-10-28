# Patra World Social

Dunia game sosial privat dengan tema desa autumn/Halloween. Backend aplikasi memakai Go; Node digunakan untuk tooling frontend. Guest bermain solo lokal. Akun username/password dapat membuat dunia privat, mengundang anggota, dan terhubung ke runtime multiplayer.

## Status implementasi

Kode akun, dunia privat/undangan, realtime movement, teman, chat/moderasi, lobby serta history sudah diimplementasikan dan direview. Duel penalti Go mendukung dua pemain atau bot, pilihan rahasia dengan batas tiga detik, hasil tersimpan, return dan rematch. Balapan Go memiliki grid lima peserta, lima lap, bot untuk slot kosong, kontrol kart dan hasil tersimpan. Duck Arcade memiliki empat kabinet, sesi 60 detik, skor/hit server, kompensasi latensi terbatas dan history. Guest dapat memainkan penalti, balapan dan Duck lokal; pertandingan tetap berjalan saat panel ditutup.

Acceptance browser akun/multiplayer masih ditunda pengguna, sehingga CP1 tetap checkpoint tertinggi yang PASS. Kasus reconnect berurutan dan label bantuan bot setelah finis individu dicatat untuk perbaikan reliability sebelum beta. Podium Duck menampilkan tiga akun unik, personal best, hasil terakhir dan pergantian champion per dunia/ruleset. History lama yang eligible diproyeksikan saat startup. Warkop mic/table chat sudah diimplementasikan; real voice belum diuji karena LiveKit staging belum tersedia. Guest history/settings dan account settings tersedia dalam snapshot T14, dengan review putaran terakhir masih tertunda.

Kampung Senja kini memakai world fullscreen dengan ruang outdoor yang bisa dijelajahi, tanpa sidebar lama. Map, Friends, Chat dan game tersedia lewat dock/menu. Lihat [plan integrasi](docs/superpowers/plans/2026-10-28-kampung-senja-integration.md), [hasil verifikasi](docs/checkpoints/evidence/CP8/kampung-senja-integration.md), dan [handoff](docs/HANDOFF.md).

Lihat [progres aktif](docs/PROGRESS.md), [bukti acceptance](docs/ACCEPTANCE.md), [rancangan](docs/superpowers/specs/2026-10-26-patra-world-social-design.md), dan [task/checkpoint](docs/superpowers/plans/2026-10-26-patra-world-social.md). Tes backend dan controller tidak menggantikan acceptance browser, provider voice, atau perangkat fisik.

## Menjalankan lokal

Toolchain terverifikasi: Node25.2.1, npm11.6.2, Go1.26.8 (toolchain proyek; system Go masih1.26.2), dan PostgreSQL18.6. Jalankan perintah dari root repo. PostgreSQL harus aktif dan database development sudah tersedia; API menerapkan migration berurutan saat startup.

```sh
npm ci
make dev
# Vite: http://127.0.0.1:4176/
```

Di terminal terpisah, muat DATABASE_URL dari file privat. Pada workspace ini konfigurasi database berada di `.cache/database.env` yang diabaikan Git. Untuk setup lain, gunakan database development sendiri; `.env.example` hanya menunjukkan bentuk variabel, bukan kredensial yang bisa langsung dipakai.

```sh
set -a
source .cache/database.env
set +a
export SERVER_ADDR=127.0.0.1:8080
export AUTH_ORIGIN=http://127.0.0.1:4176
export AUTH_DEV_LOOPBACK=1
make server
```

```sh
curl http://127.0.0.1:8080/health/live
```

Vite mem-proxy HTTP dan WebSocket `/api` ke Go8080. Browser harus memakai hostname persis `127.0.0.1` seperti AUTH_ORIGIN. Go membaca environment shell; `.env.local` tidak dimuat otomatis. Jangan memasukkan kredensial ke Git. [Kontrak akun](docs/AUTH.md) menjelaskan session, CSRF dan konfigurasi origin. AUTH_DEV_LOOPBACK hanya untuk development loopback.

## Pemeriksaan

```sh
make check
# lint, TypeScript, frontend/Go unit tests, vet, production builds

# Sesudah memuat DATABASE_URL seperti di atas:
(cd apps/server && GOCACHE="$PWD/../../.cache/go-build" go test -count=1 -race -tags=integration ./...)
make load-smoke
```

Integration tests membuat schema acak terisolasi dan membersihkannya. `make load-smoke` menjalankan aplikasi nyata dengan20 direct WebSocket clients dan PostgreSQL, termasuk reconnect serta world isolation; ini bukan soak/acceptance browser. Target `make test-integration` saat ini khusus paket auth; perintah all-package di atas mencakup paket integration lain.

Source E2E berada di `apps/web/tests/`; credential browser tests tidak dijalankan oleh `npm test` dan masih ditunda. Panduan isolated E2E ada di [AUTH.md](docs/AUTH.md); jangan menggunakan akun development sehari-hari sebagai fixture.

## Struktur dan asal kode

- `apps/web`: Canvas village, UI, game lokal, klien realtime, dan tes frontend.
- `apps/server`: layanan Go, migrations PostgreSQL, auth/private worlds/realtime dan integration tests.
- `contracts`: map/rules/protocol serta fixture lintas Go/TypeScript.
- `docs/checkpoints`: status checkpoint dan evidence proyek baru.
- `docs/superpowers`: rancangan serta rencana implementasi yang disetujui.

Dikembangkan dari Patra World sebagai referensi tema; source asal tetap tidak diubah. Riwayat pengembangan lengkap tetap ada di checkout lokal. Branch publik `main` dimulai dari snapshot game yang telah diverifikasi agar PDF dan screenshot CV dari riwayat lama tidak ikut dipublikasikan. Repo aktif berisi konten world/game. Materi karier, dokumen dan screenshot portfolio warisan, PDF/generator resume, serta aset promosi sudah dihapus dari tree aktif. Art prosedural dan lisensi font dipertahankan.

Pengguna memilih akun personal `git-patra` untuk repository publik `patra-world-social` dan push ke `main`; publikasi masih menunggu autentikasi akun tersebut. Preview aplikasi tetap lokal; belum ada deployment publik. Panduan deploy/rollback beta akan disediakan dan diverifikasi pada T16; konfigurasi hosting proyek sumber tidak digunakan.
