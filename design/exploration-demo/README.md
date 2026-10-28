# Kampung Senja — demo desain eksplorasi

Prototipe lokal sementara, dibuat 28 Oktober 2026 untuk permintaan desain dan demonstrasi world outdoor yang bisa dijelajahi. Prototipe ini tetap terpisah. Arah desainnya kemudian disetujui dan diterapkan ke aplikasi utama pada4176; lihat plan integrasi di docs/superpowers/plans/2026-10-28-kampung-senja-integration.md.

```sh
cd design/exploration-demo
node server.mjs
```

- Demo: <http://127.0.0.1:4180/>
- Preview layout ponsel 390 × 844: <http://127.0.0.1:4180/mobile.html>
- Rancangan yang disetujui: [DESIGN.md](DESIGN.md)
- Pemeriksaan dan batas bukti: [VERIFICATION.md](VERIFICATION.md)

WASD/panah untuk berjalan, Shift berlari, klik/tap tanah untuk mengikuti jalur, E duduk/berdiri atau melihat konsep aktivitas, Q menyapa, M membuka peta. Di layar kecil tersedia tombol arah. Lima daun emas dapat dikumpulkan dengan mendekat.

Tujuh area saling menyambung: alun-alun, Warkop, halaman arcade, kebun lentera, lapangan penalti, lintasan mini, dan tepi sungai. Tidak ada sidebar CV. Tidak ada jaringan, mikrofon, akun atau penyimpanan progres; reload mengulang sesi demo. NPC dan respons sapa adalah simulasi yang diberi label.

Sumber art procedural: `art.js`; gerak/kamera/interaksi: `demo.js`; HUD responsif: `index.html` dan `style.css`. Font DM Sans disalin dari aset proyek; lisensi disertakan. Server hanya bind loopback dan melayani folder demo, tanpa dependensi baru.

Responsive preview untuk aplikasi utama yang sudah berjalan pada4176: `verify-mobile.html` (390×844) dan `verify-tablet.html` (768×1024). Keduanya hanya membungkus aplikasi aktual dengan iframe untuk pemeriksaan layout.
