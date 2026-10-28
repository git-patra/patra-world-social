# CP4 — Game pertama lengkap

Status: **IN_PROGRESS — bukan PASS**. Diperbarui 27 Oktober 2026. T7/T8/T9 code readiness diterima; acceptance browser masih ditunda. Checkpoint tertinggi PASS tetap CP1.

## Scope

Lobby → penalti manusia/bot → hasil tersimpan → history. Acceptance mencakup timer tepat 3.000 ms, pilihan tersembunyi, timeout random server, finalisasi idempotent, dan kegagalan/pemulihan database.

## T7 — Lobby dan lifecycle

Registry per world menyediakan race lima slot, penalti dua slot, serta empat kabinet duck. Identitas berasal dari sesi dan seat yang sah. Roster dibekukan sebelum penyimpanan; seed tidak dikirim ke browser. Host migration, readiness, bot fill, spectator, busy claim, dan return entrance berbagi lifecycle.

PostgreSQL menyimpan running match, roster, seed, version, prepared_at dan command identity sebelum start resmi. Race mendapat countdown penuh tiga detik setelah commit terkonfirmasi; penalti/duck tidak mendapat tambahan countdown generik. starts_at dapat diproyeksikan kemudian; T8 harus membawa waktu runtime sebenarnya dalam hasil. COMMIT yang tidak pasti direkonsiliasi dengan ID stabil dan advisory lock sebelum menyimpulkan row tidak ada.

Pada commit T7, produksi belum mendaftarkan engine game. UI menyatakan Start belum tersedia; engine sintetis hanya fixture tes. Aturan penalti/race/duck tetap milik T9–T11, hasil durable milik T8, dan pelepasan voice milik T13.

Full Go race/integration, vet/build dan 36 frontend tests/typecheck/lint/build dilaporkan PASS. Perubahan timing terakhir mendapat scoped lobby race/PG/vet/build PASS. Review awal menemukan satu Important: return arena lama bisa berlomba dengan klaim lobby baru. Fix round1 diterima: return membawa generation dan mutation world memeriksanya secara atomik terhadap claim baru; delivery gagal tetap pending untuk retry. Scoped real-PG application/lobby dan world/lobby/realtime race tests serta vet PASS. Commit progres T7 `3e37f91ebeafa6f1c7200eee3622330348c56527`; status implemented_pending_external_verification.

## T8 — Hasil durable dan history pribadi

Implementasi dan review perbaikan sudah diterima. Hasil berasal dari engine server, dibekukan dan diikat ke metadata/roster pertandingan, masuk ke result_jobs dengan hash immutable, lalu diproyeksikan melalui satu transaksi. Identical retry tidak menggandakan history; payload berbeda dengan ID sama ditolak. Event “tersimpan” hanya mengikuti konfirmasi durable.

Worker menangani retry, job restart, orphan aborted_server, batas60detik unconfirmed dan rekonsiliasi absent/durable. History list/detail memakai identitas sesi dan keikutsertaan, sehingga tetap tersedia setelah keluar/archive world. Guest/akun lain ditolak. Public identity memiliki overlay agar T14 dapat menganonimkan actor. Projector dalam transaksi tersedia untuk detail game/PB/champion T9–T12; pada commit T8 belum ada engine produksi atau pemenang sintetis.

Full Go race/integration dan37frontend tests/lint/typecheck/build PASS; Go vet/build PASS. Review awal mereproduksi dua defect: startup membaca snapshot absent-job lama setelah enqueue commit, dan backlog rekonsiliasi menyembunyikan deadline hasil baru. Fix round1 membuat keputusan startup di bawah advisory fence dengan fresh read, serta satu ready result per Step dengan total budget3detik dan pemeriksaan semua deadline lebih dahulu. Permanent RED→GREEN dan covering history/lobby/application/realtime race/PG PASS; vet/build PASS. Scoped rereview menyatakan kedua finding ADDRESSED dan tidak menemukan regresi blocking. Commit progres T8 `5476ede4db27ac8beac150dc3eec2dd993dec8aa`; status implemented_pending_external_verification.

Guest [History gate smoke](evidence/CP4/t8-guest-preview.md) dan [screenshot](evidence/CP4/t8-guest-history.png) memakai frontend T8 dengan backend preview T7. Ini hanya bukti gate guest, bukan acceptance history online. Uji browser akun tetap NOT_RUN.

## T9 — Duel penalti (kode diterima, acceptance pending)

Engine Go, duel guest, protokol, proyeksi hasil/history dan rematch sudah diimplementasikan dan direview. Suite awal Go integration, race untuk package yang berubah, vet/build, serta40tes frontend/typecheck/lint/build PASS. Review awal menemukan tiga Important: timestamp disconnect tertunda, budget command gagal, dan klaim rematch yang dapat disela. Putaran pertama memperbaiki ketiganya; review ulang menemukan regresi aktivasi setelah grace habis selama penyimpanan start. Putaran kedua memperbaiki metadata/timing aktivasi, diuji dengan actual lobby+adapter, serta diterima reviewer. Covering integration/race/vet/build PASS. Commit progres T9 `5ae6ed657b6b30635fb20135335494aec2893425`.

[Verification](evidence/CP4/t9-verification.md), [review awal](evidence/CP4/t9-review.md), [review putaran1](evidence/CP4/t9-rereview1.md), [review putaran2](evidence/CP4/t9-rereview2.md). O1 yang sudah ada sebelumnya—disconnect yang sudah diproses lalu resume/disconnect terkoalesensi kehilangan resume—tetap terbuka untuk T16/finalreview. Ini batas correctness yang diketahui sebelum beta, bukan klaim reconnect lengkap.

[Uji guest](evidence/CP4/t9-guest-preview.md) menemukan reset duel saat modal ditutup. Perbaikannya sudah diuji lewat browser: duel bertahan setelah buka ulang, lalu menghasilkan skor4:3 dengan10tendangan dan console bersih. Ini bukti guest saja; backend preview masih T8. Uji akun/multiplayer browser tetap NOT_RUN.

## Guest UI

[Guest smoke](evidence/CP4/t7-guest-preview.md), [screenshot](evidence/CP4/t7-guest-lobby.png): navigasi Race Park membuka gate dunia privat, guest tidak mendapat Join/Start, console bersih. Preview backend masih T6 saat pemeriksaan. Ini bukan acceptance online.

Minor T15: hilangkan teks loading tak berujung pada guest offline. Authenticated browser acceptance tetap **NOT_RUN** sesuai arahan pengguna.

## Acceptance tersisa

- Resolve O1 reconnect terkoalesensi melalui T16/final review sebelum acceptance beta.
- Acceptance browser akun/multiplayer yang masih ditunda.
- Alur browser akun/multiplayer, termasuk lobby dan game, masih ditunda pengguna.

## Berikutnya

Lanjut T10 balapan dengan lifecycle dan hasil T7–T9 yang telah direview. CP4 hanya PASS setelah semua task dan acceptance checkpoint terpenuhi.

## Evidence T7

[Verification](evidence/CP4/t7-verification.md), [review awal](evidence/CP4/t7-review.md), [review perbaikan](evidence/CP4/t7-rereview1.md). Tidak ada Important terbuka pada review T7. Full suite awal dan scoped final-fix checks dicatat terpisah; tidak diklaim semua suite diulang setelah setiap edit.

## Evidence T8

[Verification](evidence/CP4/t8-verification.md), [review awal](evidence/CP4/t8-review.md), [review perbaikan](evidence/CP4/t8-rereview1.md). Full suite awal, focused outage test dan scoped final-fix checks dicatat terpisah. Tidak ada Important terbuka pada review T8. Uji awal-Persist/replace-startup lintas task tercatat sebagai follow-up T16, belum diklaim terverifikasi.
