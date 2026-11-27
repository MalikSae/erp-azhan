# SPEK — Layanan Tambahan Booking

> **Status:** Disepakati dan dibangun 27 Nov 2026 (migrasi 165; ERP, `azhan-booking`, `azhan-mobile`).
> **Catatan pembangunan:** pesan "Total tidak boleh di bawah pembayaran Rp …" juga dipakai diskon, ganti kamar, dan batal pax (dulu "total tagihan tidak boleh lebih kecil dari pembayaran terkonfirmasi (…)"). Ringkasan Pembayaran di Detail Booking mendapat baris "Layanan Tambahan", dan baris "Harga Paket Dasar" menjadi "Harga Paket" berisi jumlah harga semua pax supaya ringkasan menjumlah ke total. Rincian akun menjawab 500 bila layanan atau diskon gagal dibaca, bukan diam-diam mengosongkan barisnya. Halaman Detail Booking tidak punya pola pesan sukses, jadi daftar yang diperbarui menjadi tanda berhasil, sama dengan diskon.
> **Sumber keputusan:** sesi grilling 27 Nov 2026 (3 babak, 18 keputusan), atas permintaan pemilik bisnis: admin bisa menambahkan biaya layanan ke sebuah booking, misalnya kereta cepat, dengan isian sederhana nama + nominal.
> **Prinsip:** layanan adalah kebalikan diskon. Bentuk, gate, dan alur hitung ulangnya mengikuti `booking_discounts`, yang sudah melewati audit.

---

## 1. Latar

Fitur add-on pernah ada: master `add_ons`, pilihan per paket `schedule_add_ons`, dan biaya per booking `booking_addons` (migrasi 015). Ketiganya dihapus 15 Oktober 2026 (migrasi 086) saat menu Paket dirapikan, dengan alasan tidak lagi dipakai. Konsepnya sendiri tidak bermasalah.

Masalah yang pernah ditemukan audit, dan sudah diperbaiki untuk diskon, sehingga cukup diikuti:

| Temuan | Masalah | Perbaikan yang diikuti |
|---|---|---|
| JB-03 | Status turun dari DP/lunas saat hitung ulang, lalu worker melepas seat | Booking berbayar memegang seat tanpa batas waktu (`SyncBookingStatusTx`) |
| JB-05 | Nominal di luar kolom bocor sebagai error SQL | Batas nominal di handler |
| JB-06 | Add-on bisa ditambah ke booking batal | `lockNotBatalTx` |
| JB-07 | Total bisa turun di bawah pembayaran | `ensureTotalCoversPaidTx` |
| MP-02 | Kenaikan DP paket menurunkan status booking lama | Snapshot `dp_per_pax` |

## 2. Keputusan

| # | Topik | Keputusan |
|---|---|---|
| 1 | Bentuk | Per booking: nama (teks bebas) + nominal total. Tanpa jumlah × harga, tanpa pilihan per jamaah |
| 2 | Nilai | Nominal bilangan bulat rupiah > 0. Pengurangan harga hanya lewat diskon |
| 3 | Ubah | Tambah dan hapus di draft, baru, DP, lunas. Tanpa edit: koreksi = hapus lalu tambah |
| 4 | Booking lunas | Boleh ditambah; booking turun ke DP dan komisi batal (aturan komisi lunas terakhir, K25), dicatat ulang saat lunas lagi. ERP meminta konfirmasi singkat |
| 5 | Pelanggan | Tampil sebagai baris di rincian booking web dan app serta invoice |
| 6 | Izin | `booking.edit`, tanpa izin baru |
| 7 | Jejak | Simpan admin penambah dan waktu; tampil di ERP. Layanan yang dihapus tidak disimpan |
| 8 | Master | Belum ada; teks bebas |
| 9 | Batas diskon 10% | Tetap dihitung dari jumlah harga pax saja, layanan tidak ikut |
| 10 | UI ERP | Kartu "Layanan Tambahan" di Detail Booking (bagian 5) |
| 11 | Baris rincian | Nama layanan saja + nominal, tanpa awalan dan tanpa tanda +, sesudah baris jamaah dan sebelum diskon |
| 12 | Notifikasi | Tidak ada notifikasi ke pelanggan saat layanan ditambah atau dihapus |
| 13 | Pelaku | Hanya admin ERP. Web, app, dan self-booking hanya menampilkan |
| 14 | Rilis | ERP, `azhan-booking`, `azhan-mobile` dikerjakan dan di-merge bersamaan, satu PR per repo |
| 15 | Manifest | Tidak dikaitkan. Layanan per jamaah di manifest (Fase 3) diputuskan terpisah |
| 16 | Jamaah batal | Layanan tidak berubah otomatis; admin menyesuaikan sendiri |
| 17 | Booking batal | Layanan tetap tercatat, terkunci, ikut di total. Refund tetap dari pembayaran terkonfirmasi |
| 18 | Draft | Boleh diberi layanan; layanan terbawa saat finalisasi, tidak dihitung ulang |

## 3. Data

Migrasi `165_booking_layanan.sql`:

```sql
CREATE TABLE booking_layanan (
  id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  booking_id  BIGINT UNSIGNED NOT NULL,
  nama        VARCHAR(255) NOT NULL,
  nominal     DECIMAL(12,0) NOT NULL,
  dibuat_oleh BIGINT UNSIGNED NULL,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_booking_layanan_booking FOREIGN KEY (booking_id) REFERENCES bookings (id) ON DELETE CASCADE,
  CONSTRAINT fk_booking_layanan_dibuat_oleh FOREIGN KEY (dibuat_oleh) REFERENCES admin_users (id) ON DELETE SET NULL
) ENGINE=InnoDB;
```

Rumus total menjadi:

```
bookings.total_harga = GREATEST(0, SUM(booking_pax.harga_pax) + SUM(booking_layanan.nominal) - SUM(booking_discounts.nominal))
```

- Satu-satunya tempat rumus: `recalculateTotalTx` (`internal/booking/repository.go`). Rumus di `AGENTS.md` dan `GLOSSARY.md` ikut diperbarui.
- DP wajib tidak berubah (`shared.RequiredBookingDP` = DP per pax × jamaah reguler aktif). Layanan hanya menaikkan target lunas.
- Komisi tidak berubah nilainya (nominal tetap per pax, `komisi/kalkulasi.go`).
- Jatuh tempo pelunasan, pengingat sisa tagihan, dan daftar booking agen memakai `total_harga`, jadi ikut otomatis.

## 4. API

### Admin (`bookingEdit`, izin `booking.edit`)

`POST /api/admin/bookings/{id}/layanan` — body `{"nama": "...", "nominal": 3000000}`

Urutan gate (berhenti di gate pertama yang gagal):

1. Nama tidak kosong setelah trim → `400 {"error": "Nama layanan wajib diisi."}`
2. Nama ≤ 255 karakter → `400 {"error": "Nama layanan maksimal 255 karakter."}`
3. Nominal > 0 → `400 {"error": "Nominal harus lebih dari 0."}`
4. Nominal bilangan bulat → `400 {"error": "Nominal harus bilangan bulat rupiah."}`
5. Nominal ≤ batas kolom (`maxNominal`) → `400 {"error": "Nominal terlalu besar."}`
6. Booking ada dan milik brand pengguna → `404`
7. Di transaksi: `lockNotBatalTx` → INSERT (dengan `dibuat_oleh`) → `recalculateTotalTx` dengan sebab komisi `layanan_ditambah` → COMMIT

Respons: detail booking terbaru (pola sama dengan diskon).

`DELETE /api/admin/bookings/{id}/layanan/{layananID}`

Di transaksi: `lockNotBatalTx` → DELETE (tidak ada → `404`) → `recalculateTotalTx` → `wajibTotalDiAtasNolTx` → `ensureTotalCoversPaidTx` → COMMIT. Penolakan membatalkan seluruh perubahan.

Pesan penolakan:
- total jadi Rp 0 → `"Total booking tidak boleh Rp 0."`
- total di bawah pembayaran → `"Total tidak boleh di bawah pembayaran Rp 6.000.000."`

`GET /api/admin/bookings/{id}` mendapat field baru:

```json
"layanan": [{"id": 1, "nama": "Kereta cepat Madinah–Makkah", "nominal": 3000000, "dibuat_oleh_nama": "Rina", "created_at": "..."}]
```

Daftar kosong = `[]`.

Sebab batal komisi baru di `internal/komisi/batal.go`: `SebabLayananDitambah = "layanan_ditambah"`.

### Akun pelanggan (kontrak dengan `azhan-booking` dan `azhan-mobile`)

`GET /api/account/bookings/{id}` (`internal/customer/orders.go`) mendapat field baru `"layanan": [{"nama": "...", "nominal": 3000000}]` di samping `discounts`. Hanya menambah field; struktur lama tidak berubah.

## 5. UI dan copy

Mengikuti `docs/ux-copy.md` dan `.agents/rules/design-system.md`.

### Detail Booking ERP (`frontend/shared/src/pages/BookingDetailPage.jsx`)

- Kartu **"Layanan Tambahan"** di atas kartu "Program Diskon".
- Tombol kartu: "Tambah layanan" (sekunder, kecil, satu-satunya tombol di kartu). Disembunyikan di booking batal.
- Kosong: "Belum ada layanan."
- Baris: nama layanan; di bawahnya tanggal dan nama admin ("27 Nov 2026 · Rina"); nominal di kanan, warna netral. Hapus lewat menu ikon ⋯.
- Modal **"Tambah Layanan"**:
  - Nama layanan * — placeholder "Kereta cepat"
  - Nominal * — placeholder "Total untuk semua jamaah"
  - Tombol "Simpan"
- Sukses: "Layanan ditambahkan." / "Layanan dihapus."
- Konfirmasi hapus: "Hapus {nama}? Total turun {nominal}." — tombol "Hapus" / "Batal".
- Konfirmasi di booking lunas: "Tambah {nama}? Booking jadi belum lunas." — tombol "Tambah" / "Batal".
- Galat tampil di dekat field (nama, nominal) atau di modal (galat dari server).
- Ringkasan atas tidak berubah: "Total tagihan" sudah termasuk layanan.

### Invoice dan rincian pelanggan

Baris layanan: nama + nominal, tanpa awalan dan tanpa tanda +, sesudah baris jamaah dan sebelum baris diskon.

| Tempat | Berkas |
|---|---|
| Invoice ERP | `frontend/shared/src/utils/printInvoice.js` |
| Web rincian & invoice | `azhan-booking/src/components/pesanan/TabelInvoice.jsx`, `src/app/(pelanggan)/akun/booking/[id]/page.jsx`, `.../invoice/page.jsx` |
| App | `azhan-mobile/apps/hana/src/components/perjalanan/DetailBooking.tsx`, tipe `packages/core/src/perjalanan.ts` |

Web dan app membaca `layanan` dengan nilai bawaan `[]`, supaya tetap jalan bila ERP belum mengirim field itu.

## 6. Pengujian

Uji Go (pakai `internal/testdb`):

- tambah layanan → `total_harga` naik sebesar nominal; DP wajib tetap
- booking DP + layanan sampai bayar < total → status tetap DP; booking lunas + layanan → status DP, komisi aktif batal dengan sebab `layanan_ditambah`, seat tetap terpegang tanpa batas waktu
- hapus layanan → total turun; lunas kembali bila bayar ≥ total, komisi dicatat ulang
- hapus yang membuat total < pembayaran terkonfirmasi → ditolak, layanan masih ada, total tidak berubah
- hapus yang membuat total Rp 0 → ditolak, tanpa data parsial
- booking batal: tambah dan hapus ditolak, data tidak berubah
- draft + layanan → finalisasi → layanan tetap, total = harga pax baru + layanan − diskon
- batas diskon 10% tidak berubah oleh layanan
- `GET /api/account/bookings/{id}` memuat `layanan` (`[]` bila kosong)
- `go test ./internal/uxcopy` lulus untuk pesan galat handler baru (pesan `bizErr` di repository tidak ikut dipindai pemindai itu)

Frontend: `npm run build` di dashboard yang diubah, `npm run test:copy`, screenshot 375px dan 1440px kartu Layanan Tambahan, modal, dan invoice.

## 7. Di luar cakupan

- Master daftar layanan dan laporan per layanan.
- Layanan per jamaah dan kaitannya ke manifest (Fase 3 manifest).
- Notifikasi ke pelanggan.
- Pelanggan atau agen memilih layanan sendiri.
