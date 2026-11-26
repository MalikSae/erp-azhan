# Spesifikasi Tabungan Umroh (Backend)

Tanggal: 17 November 2026 · Status: backend tahap 1 dan 3 dibangun; layar app belum
Terkait: `GLOSSARY.md` (Target Tabungan, Catatan Tabungan), `docs/ux-copy.md` (pesan galat)

## 1. Tujuan dan keputusan

Calon jamaah mencatat target menabung pribadi untuk umroh dan mengisi catatan setiap kali menyisihkan uang. App menampilkan sisa, perkiraan per bulan, dan pengingat.

- **Hana tidak menyimpan uang.** Ini catatan pribadi, bukan rekening. Tidak ada saldo, setoran, atau titipan; tidak ada pergerakan uang lewat sistem. Hindari kata "saldo", "setor", "titip", "tabungan di Hana", dan "cicilan" di teks tampil.
- Satu target per akun. Mengganti target tidak menghapus catatan; menghapus target menghapus semua catatannya.
- Token push disimpan sekarang, tetapi **push belum dikirim** pada tahap ini. Pengingat target (`pengingat_*`) hanya disimpan; pengirimannya tahap berikutnya.
- Hapus akun (anonimkan) menghapus target, catatan, dan token push akun itu.
- Uang bilangan bulat rupiah (BIGINT di DB, number bulat di JSON), tanpa float. "Hari ini" dan bulan berjalan memakai WIB.

## 2. Data

Migrasi `152_tabungan_umroh.sql`.

| Tabel | Isi |
|---|---|
| `tabungan_target` | `akun_id` UNIQUE (FK `customer_accounts`, CASCADE); jenis; `schedule_id` (FK `schedules`, SET NULL); snapshot `nama_paket` dan `dp_minimal`; bulan berangkat; jumlah dewasa/bayi; tipe kamar; `target_total`; pengaturan pengingat |
| `tabungan_catatan` | `akun_id` (FK, CASCADE); `jumlah`; `tanggal`; indeks `(akun_id, tanggal)` |
| `perangkat_push` | `token` UNIQUE; `akun_id` (FK, CASCADE); platform |

Tipe kamar memakai nilai yang sudah dipakai booking: `Quad`, `Triple`, `Double` (huruf awal kapital).

## 3. Kontrak API

Semua rute di grup `/api/account` (butuh sesi akun). Galat: `{"error": "..."}`, pesan Indonesia singkat (masalah, lalu langkah bila ada).

| Rute | Hasil |
|---|---|
| `GET /api/account/tabungan` | 200 `{target, ringkasan, catatan}` |
| `PUT /api/account/tabungan/target` | 200 bentuk GET (upsert) |
| `DELETE /api/account/tabungan/target` | 204 |
| `POST /api/account/tabungan/catatan` | 201 bentuk GET; tanpa target atau catatan sudah 200: 409 |
| `DELETE /api/account/tabungan/catatan/{id}` | 200 bentuk GET; bukan milik akun atau tidak ada 404 |
| `PUT /api/account/perangkat-push` | 204 |
| `DELETE /api/account/perangkat-push` | 204 (hanya token milik sendiri) |

### Target

`jenis` `paket` atau `perkiraan`; `schedule_id` wajib dan harus ada bila `paket`, harus null bila `perkiraan`; `nama_paket` maks 150 (boleh kosong); `bulan_berangkat` `YYYY-MM` dari bulan berjalan sampai 36 bulan ke depan, dan untuk `paket` wajib sama dengan bulan `berangkat_tanggal` jadwal; `jumlah_dewasa` minimal 1, `jumlah_bayi` minimal 0 dan tidak melebihi dewasa, dewasa + bayi maks 9; `tipe_kamar` Quad/Triple/Double; `target_total` 1.000.000-2.000.000.000; `dp_minimal` 0 sampai `target_total` (negatif: "DP tidak valid."); `pengingat_aktif`; `pengingat_tanggal` 1-28; `pengingat_jam` 0-23. Respons menambah `dibuat_at`.

PUT berjalan dalam transaksi dan mengunci target lama (`FOR UPDATE`). Paket harus tayang (published, belum berangkat), kecuali `schedule_id` sama dengan yang tersimpan dan jadwalnya masih ada. Bulan lewat ditolak, kecuali `bulan_berangkat` sama dengan yang tersimpan.

### Catatan

`jumlah` 1.000-1.000.000.000. `tanggal` opsional (default hari ini WIB), tidak boleh lewat hari ini dan tidak boleh sebelum 2020-01-01. Maks 200 catatan per akun (lebih dari itu 409); GET mengembalikan semuanya, terbaru dulu. Tambah catatan dan hapus target mengunci baris target dulu.

### Ringkasan

- `tercatat`: jumlah semua catatan; `sisa` = `target_total - tercatat`, minimal 0.
- `persen`: dibulatkan ke bawah, maks 100.
- `sisa_bulan`: selisih indeks bulan (tahun×12+bulan) dari bulan ini WIB ke bulan akhir, minimal 1. Bulan akhir target `paket` = bulan `batas_pelunasan`, paling awal hari ini; target `perkiraan` = `bulan_berangkat`.
- `batas_pelunasan`: `berangkat_tanggal − brands.batas_pelunasan_hari` (`YYYY-MM-DD`) untuk target `paket`; null untuk `perkiraan`.
- `per_bulan`: `sisa / sisa_bulan` dibulatkan ke atas ke kelipatan 1.000; 0 bila `sisa` 0.
- `tercatat_bulan_ini`: catatan bertanggal di bulan berjalan WIB.
- `per_bulan_ini`: jatah bulan ini = `max(0, target_total − (tercatat − tercatat_bulan_ini)) / sisa_bulan`, pembulatan sama dengan `per_bulan`. `kurang_bulan_ini` = `max(0, per_bulan_ini − tercatat_bulan_ini)`.
- `lewat`: bulan `bulan_berangkat` sebelum bulan ini WIB. `paket_tayang`: target `paket` jadwalnya published dan belum berangkat; `perkiraan` selalu true.
- `dp_tercapai`: `tercatat >= dp_minimal`; `tercapai`: `tercatat >= target_total`.

### Token push

Format `^Expo(nent)?PushToken\[[A-Za-z0-9_-]+\]$`, platform `ios` atau `android`. Upsert menurut token; token milik akun lain pindah ke akun pemanggil. Maks 10 token per akun; yang paling lama diperbarui dihapus.

## 4. Kode

Paket `internal/tabungan` (`model.go`, `tabungan.go`, `handler.go`). Pemeriksaan batas ada di `ValidasiTarget` dan `ValidasiCatatan`; hitungan di `Hitung` (fungsi murni). Penghapusan data akun di `HapusDataAkun`, dipanggil `anonimkanAkun` di `internal/customer/hapus_akun.go`.

## 5. Di luar tahap ini

Layar app, pengiriman push, dan pengingat terjadwal.
