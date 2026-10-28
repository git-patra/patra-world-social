# CP6 — Duck Arcade dan podium privat

Status: **IN_PROGRESS — bukan PASS**. Kode T11 diterima setelah satu putaran perbaikan. Kode T12 diterima setelah satu putaran perbaikan. Acceptance browser multiplayer, jaringan nyata dan perangkat tetap dicatat terpisah.

| Perilaku | Bukti tersedia | Acceptance tersisa |
|---|---|---|
| Empat kabinet, satu pemain per kabinet, spectator terbatas world | Actual application/PG/WS integration | Browser privat |
| Countdown3s lalu gameplay60s,30Hz/15Hz | Engine dan actual owner scheduler; batas input countdown/deadline | Browser online/device |
| Hit server100/200/500, cooldown250ms, satu target per tembakan | Shared Go/TS fixtures, duplicate/dead/frontmost/zero-shot tests | Playtest manusia/touch |
| Rewind maksimal150ms dari offset/RTT terukur |8 deterministic public-view/realWS profiles50/100/200/320ms dengan0/+12ms jitter | Jaringan nyata/asimetri dan browser online |
| Disconnect15s/incomplete versus natural60s | Actual owner delayed presence/activation; natural end menang pada batas sama | Browser reconnect; known O1 tetap T16 |
| Hasil durable dan history privat | PG projector/idempotency/immutable metadata, incomplete history | Browser akun ditunda pengguna |
| Podium tiga akun unik per world/game/ruleset | T12 code accepted | Projection/concurrency/UI/review dan browser privat |
| Tiga pergantian champion | T12 code accepted | First champion, same-owner PB, deletion audit dan browser privat |
| Guest lokal dan pointer/keyboard | Actual CUA countdown,hit/miss,remountfirstshot,finishlock,reset,zero-shot PASS | Physical touch/device/FPS |

## Implementasi dan verifikasi T11

T11 partial code/evidence commit: `607141a58593f3abf473edb369148dcd59fed289`.

[Report verifikasi](evidence/CP6/t11-verification.md) menyimpan whole-Go integration/race,vet/build dan53frontendtests/lint/typecheck/build pada kandidat awal. Setelah perbaikan R1, affected Go race/vet,5targetedTS tests,typecheck/build dijalankan; tidak diklaim sebagai full-suite rerun baru.

[Review awal](evidence/CP6/t11-review.md) menemukan target online tertinggal dari waktu hit server. [Scoped rereview](evidence/CP6/t11-rereview1.md) menerima perbaikannya: production controller/view memakai kalibrasi terukur, prediksi target publik maksimal500ms, dan server tetap membatasi rewind150ms. Tes menggunakan geometri/pointer yang benar-benar dirender sebelum payload masuk ke real WebSocket; kasus200msRTT/click55ms setelah delivery tercakup.

[Guest CUA](evidence/CP6/t11-guest-preview.md) mencatat sesi nyata, termasuk skor100/200/500, keyboard, input pertama sesudah remount dan hasil yang tidak berubah setelah finish. Screenshot serta pemeriksaan console tersimpan dalam evidence. CLIguest E2E pernah gagal pada assertion whitespace terakhir; source dikoreksi tetapi **NOT_RERUN**. CUA adalah pelaksanaan terpisah dan tidak mengubah status E2E tersebut.

## Pekerjaan dan acceptance yang masih terbuka

- T12 podium/PB/champion sudah direview; R1 access-revocation UI sedang diperbaiki, belum di-commit.
- R2 `duck.shot_resolved` belum memiliki server producer; ACK/snapshot menyampaikan hasil saat ini. Penyelesaian kontrak tetap kewajiban T16/final review sebelum beta.
- Authenticated browser akun/invite/spectator/history/reconnect ditunda pengguna.
- Synthetic timing tidak menggantikan pengukuran jaringan nyata, asimetri, mobile/touch, FPS atau playtest manusia.
- Full-page guest persistence milik T14; known O1 coalesced reconnect milik T16. Tidak dinyatakan selesai oleh tes Duck.

## Kandidat T12

[T12 verification](evidence/CP6/t12-verification.md) mencatat projection/PB/champion privat, backfill hasil lama, transaksi removal/admin audit dan notifikasi sesudah commit. Whole-Go race/integration serialpackage serta60frontendtests/vet/build lulus. RunGo paralel sebelumnya gagal pada asumsi waktu tes limiterchat; hasil gagal dan diagnostik tetap tercatat, bukan dianggapPASS.

[Review T12](evidence/CP6/t12-review.md) meminta fence bersama untuk request board/detail setelah akses dicabut (ImportantR1; sedang diperbaiki). MinorR2 mengenai angka nol yang diciptakan untuk historicalmissingdetail dicatat sebagai kewajibanT16 sebelumbeta. [Guest CUA](evidence/CP6/t12-guest-preview.md) memeriksa guestgate/copy/kontras, tidak membuktikan authenticatedboard.

## T12 code accepted — partial C12

[Scoped rereview](evidence/CP6/t12-rereview1.md) menerima R1: board dan detail memakai shared generation, pembatalan/clearing saat denial, serta recheck terkontrol. Empat production lifecycle regressions gagal sebelum fix dan8focusedtests lulus sesudahnya; typecheck/build lulus. Ini deterministicNodecomponent evidence, bukan authenticatedbrowser. R2 historicalmissingdetail tetapT16; fullaccountdeletion orchestrationT14. Tidak ada CP6PASS yang disimpulkan.

T12 partial code/evidence commit: `bb1b4e88c7079ad99442e91b98089a0fe286f006`. API8080 uses acceptedT12; startup migration/backfill and healthok verified at17:39:54local. No authenticated browser acceptance inferred.
