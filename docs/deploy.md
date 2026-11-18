# Catatan deploy

Pelengkap bagian deploy di `README.md`. Hanya konfigurasi yang butuh keputusan saat rilis. Daftar lengkap variabel ada di `.env.example`.

## Aktivasi Akun Jamaah

Jamaah yang dibuatkan booking oleh orang lain (misalnya agen) mengaktifkan akunnya dengan kode booking dan NIK. Dua rute: `POST /api/public/auth/aktivasi` (akun belum ada) dan `POST /api/account/jamaah/tautkan` (akun sudah masuk). Keputusan lengkap: `docs/adr/0003-aktivasi-akun-jamaah.md`.

| Variabel | Bawaan | Keterangan |
|---|---|---|
| `AKTIVASI_JAMAAH_AKTIF` | `false` | `true` atau `1` menyalakan kedua rute. Selain itu keduanya menjawab 404. |
| `AKTIVASI_PEPPER` | kosong | Wajib saat flag menyala, minimal 32 byte. Acak, simpan sebagai rahasia: `openssl rand -hex 32`. |

Aturan saat start:

- Flag menyala dan pepper kosong atau kurang dari 32 byte: server **gagal start** (`konfigurasi aktivasi akun jamaah`). Tidak ada mode setengah jalan.
- Flag mati: pepper tidak dibaca. Rute 404, tabel `jamaah_aktivasi_*` tetap ada.

Urutan menyalakan:

1. Migrasi 153 dan 154 sudah diterapkan.
2. Tahap komisi (akun jamaah, kunci `nik_agen`) dan pandangan jamaah sudah rilis. Tanpa itu aktivasi menautkan akun tetapi komisi dan tampilan belum mengikuti.
3. Isi `AKTIVASI_PEPPER` di staging, nyalakan flag, uji skenario agen mendaftarkan jamaah lalu jamaah aktivasi.
4. Ulangi di production.

Mengganti pepper: penghitung percobaan lama tidak lagi cocok (hash berubah), jadi batas yang sedang berjalan dimulai dari nol. Log lama tetap tersimpan tetapi hash-nya tidak bisa dicocokkan dengan kode atau NIK baru. Ganti hanya bila pepper bocor.

Pembersihan: proses API menjalankan pembersih tiap 6 jam (juga saat flag mati). Penghitung dibuang setelah kedaluwarsa, log kegagalan setelah 90 hari, jejak aktivasi, tautan, dan lepas tautan setelah 2 tahun.

Jangan catat body permintaan di proxy atau log akses untuk kedua rute: body memuat kode booking dan NIK.
