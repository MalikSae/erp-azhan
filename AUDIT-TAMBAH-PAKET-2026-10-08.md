# Audit fitur Tambah Paket Baru

Tanggal: 8 Oktober 2026. Branch: `feat/pengguna-organisasi`.
Lingkungan: lokal Laragon, database `erp_azhan_dev`, backend `go run cmd/api/main.go`, Office Dashboard `:5173`.

## Cakupan

`POST /api/admin/schedules` (`internal/schedule/handler.go`, `repository.go`) dan form `frontend/master-dashboard/src/pages/ScheduleFormPage.jsx` untuk mode tambah. Validasi yang dipakai bersama mode edit ikut diperiksa karena `validateScheduleInput` satu untuk keduanya.

Audit sebelumnya: `AUDIT-PAKET-2026-09-27.md` (F01–F16) dan `AUDIT-MASTER-PAKET-2026-10-01.md` (MP-01–09). Perbaikan MP-03, 04, 06, 08, dan 09 terbukti terpasang pada fitur ini.

Metode: baca kode, lalu uji API dengan akun super admin (enam skenario sebelum perbaikan, sembilan sesudahnya). Paket uji dihapus setelah dipakai.

Tidak dicakup: unggah brosur, tampilan mobile, dan uji dengan akun selain super admin.

## Ringkasan

| ID | Tingkat | Temuan | Status |
|---|---|---|---|
| TP-01 | Sedang | `brand_id` tak dikenal memberi 500 (FK), karena brand hanya dicek saat `minimal_dp` kosong | Diperbaiki |
| TP-02 | Sedang | Kode penerbangan > 50 karakter memberi 500 (kolom `VARCHAR(50)`) | Diperbaiki |
| TP-03 | Rendah | Jam tidak valid (`25:99`) memberi 500 | Diperbaiki |
| TP-04 | Sedang | Promo tanpa `harga_coret` dan `promo_until` yang sudah lewat diterima | Diperbaiki |
| TP-05 | Sedang | "Kuota Terisi" di form tambah membuat `seat_sisa` < `seat_total` tanpa booking | Diperbaiki: field dihapus, paket baru selalu penuh |
| TP-06 | Rendah | Nama paket kembar diterima | Diperbaiki (nama + tanggal berangkat + brand) |
| TP-07 | Rendah | Form menerima kota hotel "Mekkah", backend hanya "Makkah" (laten, data sekarang semua "Makkah") | Diperbaiki |
| TP-08 | Rendah | Form: tanggal promo tidak valid hanya menampilkan "Gagal menyimpan paket" | Diperbaiki |
| TP-09 | Rendah | Form: error dari server hanya di atas halaman, tanpa penandaan field | Diperbaiki |
| TP-10 | Rendah | Form: tidak ada peringatan saat menutup halaman dengan isian belum disimpan | Diperbaiki sebagian (tutup tab/muat ulang) |

## Temuan dan perbaikan

### TP-01, TP-02, TP-03 · Error 500 dari database
Handler memvalidasi sebagian besar field, tetapi tidak semua. Nilai yang melewati handler ditolak MySQL (error 1452, 1406, 1292), lalu `handleRepoError` mengubahnya menjadi 500 generik. Pengguna hanya melihat "Gagal menyimpan paket".

Bukti sebelum perbaikan (dari log API):
```text
Error 1452: foreign key constraint fails (... CONSTRAINT schedules_ibfk_5 FOREIGN KEY (brand_id) ...)
Error 1406: Data too long for column 'berangkat_kode_penerbangan'
Error 1292: Incorrect time value: '25:99' for column 'berangkat_jam'
```
Perbaikan: brand selalu diperiksa (`sql.ErrNoRows` menjadi 400, error lain 500); `validateJam` menerima kosong, `HH:MM`, atau `HH:MM:SS`; `validateMaxLen` untuk kode penerbangan (50) dan `transit_bandara` (255). Semua kini 400 dengan pesan yang menyebut nama field.

### TP-04 · Promo
Promo tanpa `harga_coret` dan `promo_until` lampau tersimpan. Tampilan publik sudah menonaktifkan promo yang lewat (`ToPublic`), jadi dampaknya pada data dan label di admin, bukan pada harga publik.

Perbaikan (`validatePromo`): promo aktif wajib punya `harga_coret`. `promo_until` tidak boleh sudah lewat, tetapi hanya dicek untuk paket baru atau saat tanggalnya diubah, supaya paket lama yang promonya sudah berakhir tetap bisa diedit.

### TP-05 · Kuota Terisi
Form tambah menyediakan "Kuota Terisi" dan mengirim `seat_sisa` = total − terisi. Terbukti: paket baru tersimpan dengan sisa 5 dari 10 dan `booking_count` 0. Selanjutnya angka kursi tidak lagi cocok dengan jumlah booking.

**Keputusan akhir (8 Okt 2026):** semua kursi harus terjual lewat booking di ERP, tidak ada penjualan di luar sistem. Field dihapus dari form tambah. Paket baru selalu mulai penuh (sisa = total kuota). Di form edit, "Kuota Terisi" dan "Sisa Kuota" tetap tampil, tetapi hanya baca.

Backend menolak `seat_sisa` yang tidak sama dengan `seat_total` saat membuat paket (400, `seatSisaForCreate`). Pada edit, `seat_sisa` dari klien tetap diabaikan. Koreksi kursi sesudahnya hanya lewat booking atau "Sesuaikan Sisa Seat" yang beralasan dan tercatat riwayatnya.

### TP-06 · Nama kembar
Aturan: paket baru ditolak (409) bila brand, nama, dan tanggal berangkat sama dengan paket yang sudah ada. Nama sama dengan tanggal berbeda tetap boleh, karena itu pola umum (paket yang sama dijual berulang). Pada edit, aturan hanya berlaku bila salah satu dari ketiganya berubah, supaya duplikat lama tidak memblokir edit.

### TP-07 · Ejaan kota hotel
`hotels.city` adalah teks bebas. Dropdown form menerima `MAKKAH` dan `MEKKAH`, sedangkan backend membandingkan dengan "Makkah". `normalizeKota` menyamakan keduanya di backend. Temuan ini dari pembacaan kode; tidak diuji karena semua data hotel saat ini "Makkah" (13), "Madinah" (13), dan satu "Jeddah".

### TP-08, TP-09, TP-10 · Form
- Validasi sisi klien ditambah: urutan harga (quad ≤ triple ≤ double), promo wajib punya harga coret, dan tanggal promo (valid dan tidak lewat).
- Error dari server dipetakan ke field lewat nama field di awal pesan, sehingga muncul di daftar error dan field bisa difokuskan.
- `beforeunload` memperingatkan saat menutup atau memuat ulang halaman dengan isian yang berubah. Perpindahan antarhalaman di dalam aplikasi (mis. tombol kembali) belum dicakup karena routing tidak memakai data router.

## Yang sudah benar
Wajib `schedule.edit`; `brand_id` dari JWT dipakai untuk akun terikat brand; publish dengan tanggal berangkat lampau ditolak; kota hotel, hotel transit, kode bandara, urutan harga, DP, dan harga coret divalidasi; kategori dicek per brand; simpan paket, add-on, dan hotel transit dalam satu transaksi.

## Verifikasi
- `go build ./...`, `go vet ./internal/schedule/`, dan `go test ./...` lulus. Test baru: `TestValidateJam`, `TestValidateMaxLen`, `TestNormalizeKota`, `TestValidatePromo`.
- Uji API setelah perbaikan: kelima kasus 500 dan promo kini 400 dengan pesan jelas; duplikat 409; paket valid 201; nama sama dengan tanggal berbeda 201; edit paket sendiri 200; ganti nama menjadi duplikat 409.
- `npm run build:master` berhasil. Keterangan Kuota Terisi muncul di form tambah.
- Belum diuji di browser: alur klik pada validasi sisi klien dan peringatan `beforeunload`.

## Tidak diubah
Pemeriksaan hak akses akun non-super-admin pada endpoint ini, unggah brosur, dan komisi Syiar (field tetap bisa diisi oleh pemegang `schedule.edit`, yaitu Super Admin dan Product Manager).
