# CP2 — Akun dan akses privat

Status: **IN_PROGRESS — bukan PASS**. Diperbarui 26 Oktober 2026.

## Posisi implementasi

- T3: `implemented_pending_external_verification`. Commit progres `ec154108a6f3f2ec3a83ef1d3b13115359f54990` mengimplementasikan akun, session cookie, recovery, CSRF/origin, throttling, dan UI akun.
- T4: `implemented_pending_external_verification`. Commit progres `e71d00a2f8285892d036eaaef56504649c205ed3` mengimplementasikan private worlds, undangan, membership, role, dan UI pengelolaan.
- Checkpoint tertinggi yang lulus tetap CP1.

## Bukti T3

- Go tests/vet, real PostgreSQL integration dalam schema terisolasi, frontend lint/typecheck/unit tests/build dilaporkan PASS; commands dan hasil lengkap ada di [t3-verification.md](evidence/CP2/t3-verification.md).
- Review awal menemukan empat Important defects; seluruhnya diperbaiki dan diterima melalui [review ulang](evidence/CP2/t3-rereview1.md). Regression concurrency mereproduksi sesi login password lama yang bertahan setelah recovery sebelum perbaikan, lalu lulus setelah session creation diserialkan dengan recovery.
- Dua temuan Minor T3 (validasi username ASCII sebelum normalisasi dan klasifikasi error middleware) sudah diperbaiki dan diterima dalam review T4.

## Acceptance yang belum terpenuhi

- Browser credential lifecycle T3: **NOT_RUN**. Automatic approval review menolak pengisian password baru/pembuatan akun persisten melalui browser; pengguna kemudian secara eksplisit meminta pengujian browser ditunda dan pekerjaan lain dilanjutkan. Tidak ada percobaan melalui jalur lain.
- Browser lifecycle undangan T4: belum dijalankan dan mengikuti penundaan alur akun yang sama. Test source atau backend PASS tidak menggantikan bukti browser.
- Semua temuan Important T4 sudah diperbaiki dan lolos review ulang. Acceptance browser masih menunggu; pemeriksaan backend tidak menggantikannya.

Laporan ini mencatat progres. Status PASS hanya dapat diberikan setelah seluruh acceptance task checkpoint terpenuhi.

## Bukti T4

- Real PostgreSQL tests: concurrent redemption satu pemenang, retry idempotent, expired/revoked/target/ban, outsider denial, 50-member boundary, role preservation, moderator/owner permissions, transfer/archive. Detail di [t4-verification.md](evidence/CP2/t4-verification.md).
- `make check`: 18 frontend tests, Go tests/vet, TypeScript dan production builds PASS sebelum fix; setelah fix, focused regression dan full Go integration/typecheck/build PASS.
- Review awal menemukan tiga Important defects, semuanya diterima setelah perbaikan dalam [review ulang T4](evidence/CP2/t4-rereview1.md).
- Guest UI-only smoke memperlihatkan login gate tanpa data private world, console bersih; [screenshot](evidence/CP2/t4-guest-world-gate.png). Bukan bukti lifecycle akun/invite.
- T5 masih harus menghubungkan callback revocation ke koneksi aktif, menguji 20 online dan reconnect; T6 menangani friend/block checks. Tidak ada multiplayer aktif yang diklaim pada CP2.
