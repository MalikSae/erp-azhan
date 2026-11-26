# Panduan UX copy

Panduan dasar penulisan teks antarmuka untuk seluruh produk Azhan: dashboard ERP, web publik `azhan-booking`, dan aplikasi `azhan-mobile`. Tujuannya satu suara di semua layar.

Istilah domain (akun, jamaah, agen, upline, seat, booking, lunas, dan seterusnya) **tidak** diatur ulang di sini. Sumbernya `GLOSSARY.md` di root repo ini. Panduan khusus per produk melengkapi dokumen ini, tidak menggantikannya:

- `azhan-booking/docs/ux-copy.md` — turunan panduan ini untuk web publik (istilah, kata terlarang, meta/SEO), ditegakkan `src/lib/copy.test.js`.
- `azhan-mobile/docs/ux-copy.md` — turunan panduan ini untuk aplikasi (panjang teks di ponsel, notifikasi, "Segera hadir"); istilah app juga di `azhan-mobile/DESIGN.md` bagian 7.

Bila terjadi konflik, dokumen ini yang menang untuk prinsip dan pola; glosarium yang menang untuk arti istilah.

## Prinsip

1. **Copy sistem, bukan percakapan.** Singkat, netral, tanpa sapaan basa-basi. Satu kalimat satu gagasan.
2. **Tampilkan inti, bukan penjelasan.** Aturan bisnis ditegakkan sistem (validasi, filter, sembunyikan pilihan), bukan dijelaskan di layar. Tanpa blok "Ketentuan" atau helptext yang menerangkan perilaku sistem.
3. **Konsisten.** Satu istilah dan satu kata kerja untuk satu hal di seluruh antarmuka. Ikuti glosarium.
4. **Setiap kata berguna.** Buang kata yang tidak menambah makna. Potong sampai inti sebelum menulisnya.
5. **Jangan menyalahkan pengguna, jangan meminta maaf.** Tanpa "maaf" dan tanpa "!" di pesan sistem.

## Kapitalisasi

- **Title Case** untuk judul halaman, judul section, dan menu/navigasi. Contoh: "Tambah Jamaah", "Dokumen Paspor", "Booking".
- **Menu portal akun (web azhan-booking dan app):** daftar transaksi "Booking", daftar orang "Jamaah", tanpa akhiran "Saya"/"Data" (spek `docs/spek/SPEK-MENU-JAMAAH-ISTILAH-2026-11-25.md`).
- **Sentence case** untuk label field, placeholder, teks bantuan, pesan, dan tombol. Contoh: "Nama lengkap", "Pilih brand", "Simpan".
- `UPPERCASE` dilarang kecuali kode (hex) dan inisial avatar.
- Judul dan label tanpa titik. Kalimat penjelas diakhiri titik.
- Ikuti PUEBI: awalan "di-" disambung, kata depan "di" dipisah ("di akun Anda"); kapital hanya di awal kalimat dan nama diri.

## Sapaan

- Sapa pengguna dengan **Anda** (huruf kapital), bila perlu menyapa. Berlaku di semua produk.
- Hindari "kamu", "lu", dan janji berlebihan.

## Pola

### Tombol
- Kata kerja + objek, sentence case, 1–3 kata, tanpa titik. Contoh: "Simpan", "Buat booking", "Kirim bukti pembayaran".
- Satu aksi utama (`primary`) per layar. Aksi merusak (batalkan, hapus) berupa tautan teks kecil atau tombol sekunder, bukan tombol merah besar.
- Sebut kontrol dengan labelnya saja: "Lanjut", bukan "tombol Lanjut".

### Judul
- Judul halaman: kata benda singkat, Title Case. "Detail Booking", "Tambah Jamaah".
- Judul section: Title Case. "Kontak & Alamat Domisili".

### Label dan placeholder
- Label: sentence case, ringkas, tanpa menjelaskan singkatan yang sudah dipahami petugas. "NIK", bukan "NIK (Nomor Induk Kependudukan)".
- Placeholder: petunjuk isi atau contoh format, bukan mengulang label dan tanpa "Contoh:". "A1234567", "081234567890", "Sesuai dokumen identitas". Jangan nama orang atau data yang bisa dikira sudah terisi. Kosongkan bila tidak menambah informasi.
- Field wajib bertanda `*`.

### Keadaan kosong (empty state)
- Satu kalimat: "Belum ada booking." / "Tidak ditemukan.". Tanpa ilustrasi atau ajakan panjang.
- Tambahkan satu langkah berikutnya hanya bila ada aksi nyata di layar itu.

### Pesan sukses
- Singkat: "Jamaah ditambahkan." / "Booking dibuat." / "Perubahan disimpan.".

### Pesan galat
- Struktur: **masalah**, lalu **langkah berikutnya bila ada aksi nyata**. Satu kalimat per bagian.
  "NIK sudah terdaftar. Periksa NIK dan nama sesuai KTP, atau hubungi admin."
- Tanpa "maaf", tanpa menyalahkan, tanpa menjelaskan aturan sistem, tanpa istilah teknis.
- Jangan sebut data orang lain.
- Letakkan di dekat field yang bermasalah, bukan hanya di atas form. Tampil setelah galat terjadi, bukan sebelum pengguna sempat mengisi.

### Konfirmasi
- Pakai hemat; utamakan aksi yang bisa dibatalkan (undo) daripada dialog.
- Bila perlu dialog: sebut aksinya dan akibat singkatnya, bukan "Yakin?". "Lepas paket X? 3 konfirmasi hadir ikut terhapus."
- Tombol bernama aksi: "Hapus" / "Batal", bukan "Ya" / "Tidak".

## Istilah

Arti dan pilihan kata: `GLOSSARY.md`. Yang paling sering salah:

| Pakai | Hindari |
|---|---|
| jamaah | peserta, penumpang, pax (untuk orang) |
| seat | kursi |
| booking | pesanan, pemesanan, reservasi |
| hold seat, lepas seat, kuota seat | blokir, block seat, kunci seat |
| upline | leader, agen pengikat, perekrut |
| DP, lunas | uang muka, cicilan |
| Kata sandi (teks tampil) | Password (hanya di kode dan rute URL) |

## Menulis

- Teks antarmuka ini pendek dan padat. Catatan, dokumen, pesan ke manusia lain (commit, PR, isu) ditulis normal, bukan dipotong.

## Penegakan

- Dashboard ERP: `npm run test:copy` di `frontend/master-dashboard` memindai teks tampil (`src/lib/copy.test.js`).
- API: `go test ./internal/uxcopy` memindai pesan galat handler (huruf kapital, titik, tanpa nama field teknis dan "silakan").
