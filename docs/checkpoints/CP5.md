# CP5 — Balapan lima lap

Status: **IN_PROGRESS — bukan PASS**. Kode T10 diterima setelah dua putaran perbaikan dan review. Acceptance manusia/multiplayer/perangkat masih tertunda; checkpoint tertinggi PASS tetap CP1. Commit kode parsial dicatat di docs/PROGRESS.md.

## Scope dan bukti wajib

| Perilaku | Bukti yang harus tersedia | Status |
|---|---|---|
| Lima slot, kosong diisi bot, seluruh peserta start bersamaan | Engine dan actual lobby/application tests untuk1manusia+4bot dan5manusia; browser multiplayer | Backend PASS; browser multiplayer NOT_RUN |
| Grid acak tersimpan sebelum start, tie-break sesuai slot | Preparer permutation tests, real PostgreSQL roster_slot, snapshot/result/tie-break consistency | Backend PASS |
| Countdown3s penuh, simulasi30Hz dan snapshot15Hz | Actual scheduler/clock tests; tidak bergerak sebelum startsAt; bounded catch-up | Backend PASS; guest countdown teramati |
| Kart nyata dan bot memakai aturan gerak/collision sama | Physics/track tests, bot inputs tanpa teleport/speed advantage; playtest handling | Automated PASS; handling/balance human playtest NOT_RUN |
| Tepat5lap, checkpoint berurutan dan arah finish benar | Shared Go/TS fixtures untuk skip/reverse/repeated checkpoint dan finish lap5 | Automated PASS; human5lap browser NOT_RUN |
| Batas300s dan45s setelah finis pertama | Boundary tests termasuk DNF ranking berdasarkan progres terakhir | Backend PASS; guest45s cutoff teramati |
| Disconnect takeover,15s reclaim, assisted/DNF/abandon | Actual lobby+race tests dengan waktu notifikasi asli dan online reconnect browser | Fix regressions PASS dan scoped review accepted; online NOT_RUN |
| Hasil durable,10s results/semua manusia kembali, safe return | PG projector/history tests dan lifecycle return-generation; tidak mengklaim tersimpan sebelum commit | Backend PASS; online UI NOT_RUN |
| Guest solo lokal, kontrol keyboard/touch, penonton | Actual guest browser checks, no official score upload, authenticated UI and physical-device playtest | Guest CUA PASS termasuk first-input after reopen; authenticated/device NOT_RUN |

## Batas evidence

Tes backend/synthetic transport berbeda dari acceptance browser multiplayer. Uji browser akun/multiplayer masih ditunda pengguna. Emulasi viewport berbeda dari perangkat fisik. Tidak ada hasil benchmark, tuning pemula, network-jitter, atau playtest1–5manusia yang boleh dinyatakan lulus tanpa pelaksanaan dan catatan hasilnya.

Known issue lintas task: O1 pada [review T9](evidence/CP4/t9-rereview1.md) mencatat resume/disconnect terkoalesensi yang dapat mempertahankan cutoff lama. Perbaikan dan regresinya menjadi kewajiban T16/final review sebelum beta; belum ada klaim bahwa kasus tersebut benar.

## Berikutnya

Kode T10 diterima, simpan commit C10 parsial dan lanjut T11 sesuai dependensi. Acceptance T10 yang tertunda tetap harus dituntaskan sebelum beta. CP5 hanya PASS setelah seluruh acceptance wajib terverifikasi.

## Guest smoke (belum acceptance checkpoint)

[Actual guest CUA](evidence/CP5/t10-guest-preview.md): input gas/kemudi,5kart, bot finish, modal continuity,45s cutoff/DNF dan return teramati; console bersih. Capture IAB938px terpotong dari CSSviewport1280px, sehingga screenshot awal tidak membuktikan overflow layout. Belum ada playtest manusia menyelesaikan5lap atau perangkat fisik/multiplayer.

## Kandidat siap review

Implementer melaporkan full Go integration dengan race detector PASS,49tes frontend/typecheck/lint/build sertaGo vet/build PASS. Penalty transport fixture diperbaiki agar virtual-clock advance menunggu snapshot nyata pada tiga socket, setelah dua iterasi diagnosis gagal; report menyimpan semua percobaan. Review independen sedang memeriksa staged snapshot53file, termasuk physics, grid, lifecycle, test coverage dan batas acceptance. Review meminta perbaikan R1 input guest setelah remount dan R2 urutan batch disconnect; belum ada commitC10.

## Review dan perbaikan

[Review awal](evidence/CP5/t10-review.md) menemukan dua Important: counter input guest reset saat panel dibuka ulang, dan race memajukan fisika saat metadata disconnect masih dipasang dalam urutan map yang tidak pasti. Root mereproduksi R1 lewat browser, melengkapi smoke sebelumnya yang hanya memeriksa visual continuity. Fix round1 sedang dikerjakan; snapshot review tetap staged.

## Putaran perbaikan kedua

[Report implementasi dan tes](evidence/CP5/t10-verification.md) memuat hasil full-suite awal, diagnosis fixture, kedua putaran perbaikan dan command/output yang dijalankan. [Rereview pertama](evidence/CP5/t10-rereview1.md) menerima R1/R2 namun menemukan label bantuan bot yang tercatat setelah batas hasil global. Perbaikan R3 mencakup cutoff absolut/shortened, exact/+1ms, prior assistance, actual owner dan event terminal; affected race/lobby serta PG result integration dengan race detector PASS. Scoped rereview kedua masih pending.

## Keputusan review akhir kode

[Rereview kedua](evidence/CP5/t10-rereview2.md) menerima koreksi R3 tanpa temuan Critical/Important baru. R1/R2 tetap diterima. Label bantuan bot mengikuti waktu akhir global; input guest setelah remount kembali responsif. Personal-finish assistance edge yang sudah ada sebelumnya dan O1 tetap dibawa ke T16/final review. Laporan tes/evidence tidak berarti CP5 PASS. [Preview dunia](evidence/CP5/t10-world-preview.jpg) menunjukkan shell game saat ini.
