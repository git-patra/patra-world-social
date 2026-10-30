### Spec Compliance

- ❌ Issues found: kontrol 2D belum tercermin pada gambar grip (`apps/web/src/world/scenes/claw.ts:19`); schema resmi menolak Enter/Leave/Drop dan neutral Move yang valid (`contracts/protocol.schema.json:1206`).
- ✅ Authority/lifecycle: satu operator, lease 20.000 ms, outcome dari posisi owner, dan transient prize diterapkan (`apps/server/internal/world/claw.go:64`, `:78`, `:106`, `:118`). Last-millisecond drop dan expiry tepat batas memiliki test (`apps/server/internal/world/claw_test.go:36`); disconnect/remove/takeover/match cleanup memiliki test (`:138`).
- ⚠️ Cannot verify from diff: autentikasi dua browser, LiveKit staging, physical mobile/touch/sound acceptance, profiling, dan keadaan repo sumber `patra-world`. Semuanya tetap gate/deferred eksplisit; tidak ada klaim session atau acceptance buatan. Empat kendaraan/satu bola/empat fishing spots adalah constraint lintas task; diff ini mempertahankan jalur yang ada, bukan bukti acceptance keseluruhannya.

### Strengths

- Batas input dan konstanta simulasi memakai kontrak bersama; cabinet tetap satu dan catch-up dibatasi empat langkah (`apps/server/internal/world/activities.go:76`, `apps/web/src/activities/claw.ts:11`).
- Reset operator terhubung ke join/remove/idle dan match (`apps/server/internal/world/runtime.go:140`, `:157`, `:164`; `apps/server/internal/world/activity.go:61`).
- Integrasi memakai scene dunia yang ada dan shared AudioService; lifecycle grab memiliki freshness/dropId dedup (`apps/web/src/activities/scene.ts:35`, `apps/web/src/activities/store.ts:314`).
- Test DB memeriksa tabel match/history/result/leaderboard setelah 20 siklus sungguhan, dengan hasil PASS yang dilaporkan tanpa klaim browser acceptance (`apps/server/internal/application/claw_integration_test.go:17`, `:132`; `docs/checkpoints/evidence/CP8/activities-task8-wip.md`).

### Issues

#### Critical (Must Fix)

- Tidak ada.

#### Important (Should Fix)

- **P2 — Vertical aiming tidak terlihat.** `apps/web/src/world/scenes/claw.ts:19`: `gripY=top+24+lift*18` tidak memakai `b.y`, padahal preset digambar dengan koordinat Y dan server menentukan tangkapan dari kedua sumbu (`apps/server/internal/world/claw.go:106`). Menekan Up/Down mengubah hasil Drop tanpa memindahkan grip secara visual; sasaran seperti bear/bunny/fox tidak dapat dibidik sesuai tampilan. Gunakan koordinat Y owner untuk posisi aiming dan endpoint pickup, lalu animasikan turun/angkat dari posisi tersebut. Probe renderer dengan `y=0` versus `y=1` menghasilkan canvas commands identik pada aiming/dropping/lifting/releasing.
- **P2 — Schema menolak command claw yang valid.** `contracts/protocol.schema.json:1206`: unconditional `not` berisi `properties.dx.const=0` dan `properties.dy.const=0`, tanpa `required`. Pada JSON Schema, properti absen tetap memenuhi `properties`, sehingga Enter/Leave/Drop semuanya ditolak; neutral Move `{dx:0,dy:0}` juga ditolak. Validator Go/TS justru menerima command tersebut dan neutral diperlukan untuk cleanup. Hapus assertion ini dari cabang claw, lalu tambah schema-level golden untuk empat action dan neutral Move; pemeriksaan custom validator saja tidak mendeteksi kontradiksi ini.

#### Minor (Nice to Have)

- Tidak ada tambahan.

### Checks

- Membaca full diff dalam beberapa bagian karena output awal terpotong; tidak menjalankan git atau suite yang sudah dilaporkan.
- Named risk: expiry diproses sebelum command owner. Hunk `Runtime.run` terpotong dari diff, sehingga memeriksa hanya fungsi tersebut (`apps/server/internal/world/runtime.go:77`): request memanggil `advance()` sebelum handler; `advanceClaw` mengutamakan `now >= leaseUntil`. Tidak ditemukan celah last-tick command.
- Named risk: modal/blur/pointer cleanup pada kontrol claw. Memeriksa hanya `apps/web/src/activities/input-router.ts`: mode menjadi modal ketika blocked, `release()` menetralkan owner dan membersihkan pointer/keyboard; claw register/unregister dan keydown teardown terpasang (`apps/web/src/activities/store.ts:808`, `:841`).
- Focused renderer probe melalui esbuild in-memory: PASS dalam mereproduksi gambar identik untuk Y berbeda di seluruh phase; tidak menulis source/test baru.
- Focused semantic probe assertion `not` dari schema: mereproduksi penolakan Enter/Leave/Drop/neutral Move, directional Move lolos. Full schema validator tidak tersedia (`python jsonschema` dan Node Ajv tidak terpasang); tidak menginstal dependency atau mengklaim full validation.

### Assessment

**Task quality:** Needs fixes

**Reasoning:** Authority, batas lease, cleanup, dedup, dan pemisahan casual dari match/history cukup jelas dan memiliki coverage terarah. Dua cacat pada visual aiming dan kontrak schema perlu diperbaiki sebelum Task8 dinyatakan selesai; gate browser/LiveKit yang ditunda tetap terpisah.

## Fix round 1 — final scoped review

### Spec Compliance

- ✅ **ADDRESSED — Vertical aiming.** `apps/web/src/world/scenes/claw.ts:19` kini memetakan `b.x` dan `b.y` ke posisi preset yang terlihat; `:21` memakai posisi itu untuk grip, dan `:23` menjaga pickup/lift/release tersambung. Regression memeriksa ketiga preset, kedua sumbu, continuity pickup/release, serta Y pada setiap phase (`apps/web/tests/claw.test.ts:25`, `:37`).
- ✅ **ADDRESSED — Claw command schema.** Assertion `not` yang keliru dihapus dari cabang claw; `allOf` action-specific tetap ada (`contracts/protocol.schema.json:1206`). Test menggunakan cabang schema aktual untuk Enter/Leave/Drop/directional Move/neutral Move, dan menolak extra fields, axes pada non-Move, missing fields, serta bounds yang salah (`apps/web/tests/claw.test.ts:45`, `:65`, `:69`).
- ⚠️ Scope hanya dua finding sebelumnya dan potensi breakage pada fix diff. Browser post-fix tetap milik root; authenticated-browser/LiveKit/physical-mobile gates yang ditunda tidak berubah.

### Strengths

- Fix production kecil: scene dan penghapusan satu assertion schema; tidak mengubah authority, lease, lifecycle, completion/history, dependency, timer atau RAF.
- Regression renderer memeriksa hubungan grip dengan preset dan perpindahan antar phase, bukan sekadar snapshot gambar tetap (`apps/web/tests/claw.test.ts:25`). Schema helper terbatas pada cabang claw dan gagal pada keyword yang tidak didukung (`:45`).

### Issues

- Critical: tidak ada.
- Important: tidak ada.
- Minor: tidak ada tambahan dalam scope ini.

### Checks

- Membaca `task-8-fix-round1.diff` penuh dan report Task8; tidak menjalankan ulang suite atau mengubah source.
- Membaca evidence logs: RED menunjukkan regression gagal sebelum fix; focused GREEN 12/12; full web 193/193, 0 fail/skip; typecheck dan build berhasil (108 modules). Tidak terlihat warning pada bagian hasil yang diperiksa.
- `git diff --check` exit 0 dikonfirmasi controller; tidak dijalankan ulang reviewer.
- Pada saat pembacaan terakhir, `task-8-report.md` masih snapshot lama; evidence fix tersedia pada `task-8-fix-{red,green,web,typecheck,build}.log`. Controller perlu memastikan append laporan fix ikut tersimpan sebelum commit.

### Assessment

**Task quality:** Approved untuk fix scope ini.

**Reasoning:** Kedua finding sebelumnya sudah diperbaiki dan memiliki regression terarah yang dilaporkan lulus. Tidak ditemukan breakage baru pada fix diff; verdict ini tidak mengubah acceptance gates yang masih ditunda.
