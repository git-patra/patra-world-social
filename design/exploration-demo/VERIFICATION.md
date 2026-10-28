# Pemeriksaan demo — 28 Oktober 2026

## Bukti yang dijalankan

- `node --check demo.js`, `node --check art.js`, dan `node --check server.mjs`: lulus setelah penyesuaian terakhir.
- Browser lokal melalui Codex in-app browser: halaman render, input keyboard D selama 600ms memindahkan pemain dari `(1280,1040)` ke `(1394,1040)`; kamera mengikuti. Duduk mengubah posisi ke kursi plaza `(1456,1104)` dan menampilkan status duduk; berdiri mengembalikan posisi pendekatan.
- Peta → Warkop: pemain berjalan melewati map dan mencapai sekitar `(658,656)`. Tidak membuka dialog aktivitas saat tiba. Klik tanah di dekat meja berfungsi; zoom berubah ke 85%.
- Versi akhir: jalan ke Warkop, keyboard ke meja, duduk pada `(448,612)`, dan Sapa menghasilkan balasan NPC Nara. Log browser yang dibaca setelah alur ini: tidak ada error/warning. Screenshot: [Warkop desktop](evidence/warkop-desktop.png).
- Preview iframe berukuran 390 × 844: tampilan dipantau melalui screenshot; tombol arah kanan ditekan dan karakter mendekati bangku plaza, menampilkan interaksi duduk. Dock dan d-pad terlihat pada bagian bawah. Screenshot: [mobile](evidence/mobile-390.png). Ini preview CSS dan input pointer dalam browser desktop, bukan pengujian perangkat mobile fisik.
- Pemeriksaan graph oleh subagent melalui Node dengan canvas mock: 3.651 node grid terjangkau; seluruh 7 destinasi, 5 titik pendekatan tempat duduk, 5 collectible, dan 6 NPC dapat dicapai. Jalur BFS diperiksa setiap 1px dengan radius pemain 11px; tidak ditemukan segmen terhalang. Collider bangku dermaga disesuaikan tanpa mengubah gambar. Pemeriksaan ini memvalidasi geometri, bukan rendering browser.

## Batas bukti

Tidak menjalankan ulang suite aplikasi utama karena perubahan hanya berupa folder demo terpisah. Tidak ada klaim CP tambahan, T14 diterima, acceptance multiplayer, voice, akun, load/soak, atau readiness beta. Semua status handoff tetap berlaku.

Tidak melakukan pengujian lengkap screen reader, semua kombinasi keyboard, browser lintas perangkat, dan penyelesaian collectible melalui UI. Prototype memakai semantic buttons, dialog native, label, live status, serta mode pengurangan animasi; hal tersebut bukan klaim audit aksesibilitas lengkap.

## Temuan yang dibenahi selama demo

Petunjuk sambutan mobile menutupi informasi kedekatan: kedekatan sekarang baru muncul setelah sambutan ditutup. Sambutan yang ditutup dihilangkan juga dari urutan aksesibilitas. Duduk sekarang menempatkan karakter pada furnitur dan berdiri memulihkan posisi aman. Kontrak renderer diselaraskan untuk waktu animasi, arah hadap, serta satu label nama per karakter. Tombol Sapa dan Duduk tidak lagi menggandakan tindakan awal.
