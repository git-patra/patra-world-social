# CP3 — Dunia sosial realtime

Status: **IN_PROGRESS — bukan PASS**. Diperbarui27 Oktober2026. T5/T6 code readiness diterima; acceptance browser akun/multiplayer masih ditunda pengguna. Checkpoint tertinggi PASS tetap CP1.

## Task dan commit

| Task | Status | Commit progres |
|---|---|---|
| T5 realtime movement/reconnect | implemented_pending_external_verification | 7f67defe6a1400ea69fdb8b3142f77e3a6f8c244 |
| T6 teman/chat/moderasi | implemented_pending_external_verification | 853d62f3434762c0c5f2b7a58d6b9c9520e819a8 |

## Bukti T5

[Verification](evidence/CP3/t5-verification.md), [review](evidence/CP3/t5-review.md), [round1](evidence/CP3/t5-rereview1.md), [round2](evidence/CP3/t5-rereview2.md).

Runtime Go satu owner/world, authoritative movement, map/rules/protocol checksum,20seats termasuk reconnect30detik, epoch fencing, takeover eksplisit, cookie/origin/membership dan ongoing revocation. PostgreSQL nyata/random schema dan20direct WebSocket clients menguji15detik outage, pemain21 ditolak, world isolation, logout/ban/archive serta reread sesi/membership. Ini basic smoke, bukan60menit soakT16.

Full Go race/integration awal PASS; scoped fixes realtime/world/cmd-server race PASS; real Application smoke PASS18.000s; frontend28tests/typecheck/lint/build PASS. Empat Important awal (revoke scope, leave/close ordering, admission cleanup, HTTP drain) dan satu regresi local-solo ditangani dalam dua round. Minor redundant detach/solo route telah ditangani dan diuji diT6. Reset/brand announcement masih MinorT15.

## Bukti T6

[Verification](evidence/CP3/t6-verification.md), [review](evidence/CP3/t6-review.md), [review perbaikan](evidence/CP3/t6-rereview1.md).

Friends exact-username/request/accept/reject/unfriend/block, accepted-only availability, targeted invite inbox tanpa token mentah dan bound acceptance, durable world chat500codepoints/5per5detik burst5/30hari/pages50, mute/delete/kick dan audit tersedia. Shared services/roles/CSRF digunakan. Chat COMMIT mendahului live publication dan stored ACK; real deferred-COMMIT failure menghasilkan rollback tanpa sukses palsu. Bounded64queue overflow memicu reconnect/resync.

Full `go test -race -tags=integration ./... -count=1 -timeout=120s` PASS, termasuk application19.648s dengan smoke20client. Setelah final ordering fix: scoped realtime/chat/moderation/application race+integration PASS. Setelah review menemukan shutdown lock cycle dan ACK menghapus draft baru: deterministic RED→GREEN dan affected realtime race PASS9.909s; final34frontend tests/typecheck/lint/build PASS. Semua Important sudah diterima reviewer. Commands dan urutan run lengkap ada di verification, bukan klaim semua suites diulang setelah setiap edit.

Global lease menahan operasi storage maksimal deadline3detik, sementara simulation ticks berjalan; admission/revocation dapat tertunda saat storage macet. Kapasitas/soak produksi tetapT16. Table chat/voice masihT13.

## Guest UI evidence

[Guest T5](evidence/CP3/t5-guest-preview.md), [guest T6](evidence/CP3/t6-guest-preview.md). CUA1280×720 memperlihatkan art desa, friend login gate, disabled chat send, mengetik WASD/Right tanpa gerak guest, dan console bersih. Bukti ini tidak memakai akun/kredensial dan tidak menggantikan multiplayer browser acceptance.

## Acceptance tersisa

- Dua akun/browser bergerak, berteman, mengundang, chat, reconnect, moderation dan isolasi UI: **AUTH_BROWSER_NOT_RUN** sesuai penundaan pengguna.
- Background/modal/reduced-motion/mobile pada sesi online belum diverifikasi melalui browser.
- Sebelum menjalankan authored social.e2e, perbaiki Minor: gunakan keyboard events (bukan fill saja) dan amati disconnect/resume nyata (bukan offline→online instan dengan label lama). Ditugaskan sebelum acceptance/T16.
- LiveKit/provider/media dan physical-device/scale evidence berada pada checkpoint berikutnya; tidak disimpulkan dari Node mocks atau backend smoke.

## Langkah berikutnya

LanjutT7 lobby dan lifecycle pertandingan memakai fondasi yang telah direview. CP3 hanya PASS setelah seluruh acceptanceT5/T6 terpenuhi; scope goalCP1–CP8 tetap utuh.
