# Proposal eksplorasi Patra World

**Status: arah desain disetujui untuk integrasi pada 28 Oktober 2026.** Pengguna memilih **kampung outdoor + area terbuka**, kemudian menyatakan desain cocok dan meminta penerapan fullscreen ke aplikasi utama sambil mempertahankan fitur game dan multiplayer. Folder ini tetap menyimpan demonstrasi referensi; implementasi produksi mengikuti plan `docs/superpowers/plans/2026-10-28-kampung-senja-integration.md`.

## Maksud dan dasar

Pengguna menginginkan dunia sosial yang bisa dijelajahi seperti Gather: karakter bergerak melalui ruang yang tersambung, menemukan aktivitas, dan punya alasan untuk berhenti bersama. Dunia menjadi pusat layar; tidak ada CV, daftar proyek, atau sidebar yang mendominasi. Tema bersumber dari Patra World: senja autumn, plum gelap, oranye hangat, labu, lentera, pohon berwarna musim gugur, dan suasana desa yang akrab.

Pemeriksaan kode saat ini menunjukkan sidebar sekitar 300 piksel, navigasi yang padat, serta perjalanan ke lokasi yang berakhir dengan dialog dan menghentikan dunia. Map sudah 64×64 dengan kamera mengikuti pemain, tetapi dekorasi terkumpul di pusat; empat arena luar dihubungkan jalan yang relatif kosong. Bentuk bangunan arena belum mencerminkan luas area aktivitasnya. Temuan ini berasal dari pembacaan kode, bukan hasil penerimaan pengalaman pengguna.

## Arah yang didemonstrasikan

Satu dunia tetap **2560×1920 piksel**, lebih luas dari viewport dan tetap berkesinambungan saat kamera bergerak. Ukuran ini milik demo; bukan usulan perubahan langsung pada koordinat map server. Ruang memiliki tepi yang terbaca, jalur masuk yang lebar, penanda tempat, dan detail kecil yang memberi orientasi.

| Ruang | Peran dalam eksplorasi |
| --- | --- |
| Plaza | Titik awal dan pertemuan, dengan penanda arah serta ruang terbuka untuk melihat karakter lain. |
| Warkop | Meja dan kursi sebagai tempat berhenti; kedekatan terasa melalui susunan ruang. |
| Arcade | Kabinet dan area tunggu yang terlihat dari jalan, dengan interaksi demo ringan. |
| Taman autumn | Bangku, pohon, lentera, dan sudut tenang untuk singgah. |
| Lapangan penalti | Lapangan terbuka dan batas penonton yang terbaca sebagai bagian desa. |
| Area balap | Lintasan dan pinggir lintasan untuk berkumpul; landmark yang terlihat saat mendekat. |
| Sungai dan jembatan | Batas alami serta lintasan alternatif yang menghubungkan bagian dunia. |

Jalan utama dan jalan taman membentuk beberapa **loop**, sehingga penjelajah bisa kembali ke plaza melalui arah berbeda. Jembatan, pintu lapangan, dan lorong antarbangunan menjadi penghubung. Bangku, teras, meja, tepi sungai, serta area tunggu memberi tempat untuk tinggal sejenak; ruang tidak hanya menjadi deretan pintu pembuka dialog.

## Interaksi yang dinilai

Desktop memakai WASD/panah untuk gerak kontinu dan klik tanah untuk berjalan mengikuti jalur. Kamera mengikuti karakter. Dock kecil memuat kebutuhan utama; petunjuk interaksi muncul dekat objek atau ketika karakter mendekat. Label lokasi membantu orientasi tanpa menjadi daftar permanen di sisi layar.

Mobile memakai d-pad dan tap untuk berjalan, dengan kontrol yang dapat dijangkau tanpa menutup pusat dunia. Reduced motion mengurangi gerak dekoratif dan efek berulang, sambil mempertahankan gerak karakter yang diperlukan. Demonstrasi perlu dinilai dari keterbacaan jalan, kenyamanan mengikuti karakter, dan ruang yang tersisa setelah kontrol tampil; keberadaan CSS responsif bukan bukti pengujian perangkat fisik.

## Batas kebenaran demo

Karakter pendamping adalah **NPC lokal**, bukan pemain daring. Kedekatan, duduk, emote, dan collectible adalah simulasi lokal untuk memperlihatkan suasana sosial dan alasan menjelajah. Aktivitas yang tampak di arena tidak menyatakan game produksi sudah terintegrasi. Demo tidak membuktikan multiplayer, chat antarperangkat, mikrofon/voice, autentikasi, sinkronisasi server, ataupun acceptance game.

Demo terpisah dari aplikasi utama dan tidak mengubah status T14, T15, T16, maupun checkpoint pada handoff. Repo sumber Patra World tetap tidak diubah. Jalankan dari folder ini dengan `node server.mjs`, lalu buka `http://127.0.0.1:4180/`. Server dan prototipe disiapkan terpisah dari catatan desain ini.

## Keputusan integrasi yang disetujui

Desa dengan banyak interior tidak dipilih untuk demo pertama karena transisi antarruang akan menyulitkan penilaian kesinambungan eksplorasi. Ini keputusan lingkup usulan, bukan penolakan terhadap interior pada produk akhir. Mempertahankan proyeksi isometrik juga tetap menjadi opsi setelah pengguna melihat demonstrasi; tema autumn tidak bergantung pada proyeksi top-down.

Pengguna telah menyetujui arah top-down outdoor dan meminta integrasi dengan semua fitur game/multiplayer tetap tersedia. Plan implementasi dan hasil verifikasi tersimpan di `docs/superpowers/plans/2026-10-28-kampung-senja-integration.md` dan evidence CP8.

Integrasi mempertimbangkan map frontend dan map yang ditanam pada backend, collision dan entrance, koordinat 64×64, identitas/version/checksum map, serta definisi meja voice pada kedua sisi. Mengubah ruang berjalan atau posisi meja membutuhkan pembaruan yang konsisten; mengganti tampilan saja belum tentu membutuhkan perubahan koordinat authoritative. Hash historis pertandingan harus tetap utuh. Rincian kontrak dan hasil pengujian integrasi dicatat pada plan/evidence terpisah.
