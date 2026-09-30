# Audit end-to-end master paket

Tanggal: 1 Oktober 2026. Branch: `dev-malik` (commit `96d72be`).
Lingkungan: lokal Laragon, MySQL 8.4.3, database `erp_azhan_dev` (dibangun dari `migrations/` 001–069), backend `go run cmd/api/main.go`, Master Dashboard `:5173`.

## Cakupan

Menu **Master Paket** di Master Dashboard beserta API-nya: paket (`schedules`), kategori, hotel, maskapai, bandara, itinerary, dan add-on master, termasuk endpoint publik yang dipakai microsite (`/api/schedules`, `/api/schedules/{id}`, `/api/itineraries/{id}`, `/api/public/categories`).

Audit paket sebelumnya (`AUDIT-PAKET-2026-09-27.md`, 16 temuan F01–F16, diperbaiki 28 September) belum menguji sisi admin secara langsung. Audit ini memverifikasi perbaikan tersebut saat dijalankan dan memeriksa master data yang belum tercakup.

Metode: review handler/repository `internal/{schedule,category,hotel,airline,airport,itinerary,addon}`, uji API dengan tiga akun (super admin, admin brand 1 "Travel Pertama", admin brand 2 "Brand Uji Dua"), dan uji UI Master Dashboard. Respons mentah (token disamarkan) di `audit/master-paket-evidence-2026-10-01.log`.

Tidak dicakup: tampilan microsite, Travel Dashboard, dan alur self-booking.

## Ringkasan

| ID | Tingkat | Temuan |
|---|---|---|
| MP-01 | Tinggi | Kategori tidak terisolasi per brand: admin brand lain bisa membaca, mengambil alih, menghapus, dan menambahkan kategori ke brand lain |
| MP-02 | Tinggi | Perubahan `minimal_dp` paket berlaku surut ke booking admin yang sudah ada; booking DP bisa turun ke `baru` |
| MP-03 | Sedang | Hotel Mekkah/Madinah tidak divalidasi kotanya; hotel yang sama bisa dipakai untuk kedua slot |
| MP-04 | Sedang | Paket dengan tanggal berangkat lampau bisa diterbitkan; detail publiknya tetap bisa dibuka lewat URL |
| MP-05 | Sedang | Error SQL mentah dikirim ke klien (500) di modul paket dan kategori |
| MP-06 | Rendah | Bandara yang dipakai paket bisa dihapus; kode bandara paket tidak divalidasi terhadap master bandara |
| MP-07 | Rendah | Pesan saat kapasitas diturunkan di bawah kursi teralokasi menyesatkan |
| MP-08 | Rendah | URL foto/video hotel dan brosur tidak divalidasi |
| MP-09 | Catatan | Urutan harga kamar tidak dicek; paket ber-booking bisa diarsipkan tanpa peringatan; daftar paket tidak menandai paket yang tanggalnya lewat |

## Temuan

### MP-01 · Tinggi · Kategori tidak terisolasi per brand

Rute kategori memakai `r.Post/Put/Delete` biasa (`cmd/api/main.go:294–298`), bukan `superAdmin`, sehingga admin brand boleh mengubah. Namun `GetCategory`, `UpdateCategory`, `DeleteCategory`, dan `CreateCategory` (`internal/category/handler.go`) tidak memeriksa brand pemanggil, dan `brand_ids` diterima apa adanya.

```text
[A1] super admin: POST kategori "Umroh Plus Turki" untuk brand 1 -> 201 (id 1)
[A2] admin brand 2: GET /api/admin/categories/1 -> 200 (data kategori brand 1)
[A4] admin brand 2: PUT /api/admin/categories/1 {"name":"Diambil Brand Dua","brand_ids":[2]} -> 200
[A5] super admin: kategori 1 kini bernama "Diambil Brand Dua", brands=[Brand Uji Dua] (brand 1 terlepas)
[A6] admin brand 1: POST "Kategori Milik TP" -> 201 (id 2)
[A7] admin brand 2: DELETE /api/admin/categories/2 -> 200 {"message":"kategori berhasil dihapus"}
[A8] admin brand 2: POST kategori dengan brand_ids [1,2] -> 201 (muncul di katalog publik brand 1, lihat A10)
```

Dampak: brand lain dapat mengubah atau menghilangkan pengelompokan paket di katalog publik brand sasaran. Daftar kategori (`List`) sudah tersaring per brand.

Rekomendasi: untuk admin brand, batasi `brand_ids` hanya ke brand miliknya dan izinkan ubah/hapus hanya jika kategori tidak dipakai brand lain; atau jadikan pengelolaan kategori khusus super admin, sama seperti master data lain.

### MP-02 · Tinggi · Perubahan DP paket berlaku surut

`shared.RequiredBookingDP` memakai `COALESCE(checkout.dp_per_pax, schedules.minimal_dp, brands.minimal_dp)`. Booking dari self-booking punya snapshot `booking_checkout`, tetapi booking admin tidak, sehingga DP-nya selalu dibaca dari nilai paket saat ini.

```text
[I0] booking #10 (2 pax): status dp, total 60000000, dibayar 10000000, snapshot checkout 0
[I2] PUT /api/admin/schedules/1 minimal_dp 5000000 -> 6000000 -> 200
[I3] booking #10 masih dp (belum ada hitung ulang)
[I4] POST add-on 500000 ke booking #10 (tidak berkaitan dengan DP) -> 201
[I5] booking #10: status baru (butuh 12000000, dibayar 10000000)
```

Dampak: menaikkan DP paket menurunkan status booking lama yang sudah memenuhi DP saat dibuat, pada kejadian apa pun yang memicu hitung ulang (add-on, diskon, pembayaran lain). Status turun ke `baru` memengaruhi laporan dan tampilan portal. Hold kursi tidak dilepas karena perbaikan JB-03.

Rekomendasi: simpan DP per pax pada saat booking dibuat untuk semua jalur (snapshot yang sama dengan checkout self-booking), atau tolak perubahan `minimal_dp` bila paket sudah punya booking aktif.

### MP-03 · Sedang · Hotel tidak dicek kotanya

`validateScheduleInput` hanya memastikan hotel ada (`HotelExists`).

```text
[E1] hotel_mekkah_id=9 (Madinah), hotel_madinah_id=1 (Makkah) -> 201
[E2] hotel_mekkah_id=1, hotel_madinah_id=1 -> 201
```

Rekomendasi: validasi `hotels.city` sesuai slot (Makkah/Madinah) dan tolak hotel yang sama untuk dua slot. Hotel transit tetap bebas kota.

### MP-04 · Sedang · Paket berangkat lampau bisa diterbitkan

```text
[E3] berangkat_tanggal 2025-01-10, status published -> 201
[H1] daftar publik brand 1 tidak memuatnya (sudah difilter)
[H5] GET /api/schedules/10?brand=1 -> 200 (detail tetap bisa dibuka via URL)
```

Daftar paket admin menampilkan paket ini sebagai terbit tanpa penanda sudah lewat. Rekomendasi: tolak tanggal berangkat di masa lalu saat membuat/menerbitkan, dan terapkan filter tanggal yang sama pada detail publik.

### MP-05 · Sedang · Error SQL mentah bocor

`internal/schedule/handler.go:633` dan `internal/category/handler.go:66` menulis `fmt.Sprintf("terjadi kesalahan internal: %v", err)`, pola yang sama dengan JB-05 yang sudah diperbaiki di modul booking.

```text
[E7] POST paket dengan jadwal_nama 300 karakter -> 500 {"error":"terjadi kesalahan internal: schedule.Create: Error 1406 (22001): Data too long for column 'jadwal_nama' at row 1"}
```

Rekomendasi: log di server, balas pesan generik, dan validasi panjang teks (kolom 255) di handler.

### MP-06 · Rendah · Bandara bisa dihapus walau dipakai

Paket menyimpan kode bandara sebagai teks, bukan relasi.

```text
[C6] DELETE /api/admin/airports/1 (CGK, dipakai paket #1) -> 200 {"message":"berhasil dihapus"}
[C7] paket #1 tetap berangkat_bandara_asal=CGK, CGK tidak ada lagi di master (dipulihkan kembali di C8)
[E8] paket dengan berangkat_bandara_asal "XXX" -> 201
```

Rekomendasi: validasi kode bandara paket terhadap master, dan tolak penghapusan bandara yang masih dirujuk paket.

### MP-07 · Rendah · Pesan kapasitas menyesatkan

```text
[F6] PUT paket #13 seat_total 1 (teralokasi 2) -> 400 {"error":"seat_sisa harus antara 0 dan seat_total"}
```

Validasi `seat_sisa` dari body berjalan sebelum pemeriksaan alokasi, sehingga admin tidak diberi tahu bahwa 2 kursi sudah dipesan. Rekomendasi: saat edit, abaikan `seat_sisa` dari form (repository sudah menghitungnya) dan tampilkan pesan alokasi dari `remainingAfterCapacityChange`.

### MP-08 · Rendah · URL media tanpa validasi

```text
[B6] hotel photo_url "javascript:alert(1)", video_url "bukan-url" -> 201
```

Tidak dapat dieksekusi di microsite: foto dirender sebagai `<img src>` dan video hanya diterima jika berupa ID YouTube (`getVideoEmbedUrl`), selain itu tampil "format URL tidak valid". Tetap disarankan validasi skema `http(s)` atau path `/uploads/…` di backend.

### MP-09 · Catatan

- `[E6]` harga Quad 40 jt, Triple 35 jt, Double 30 jt diterima. Perlu keputusan apakah urutan harga wajib.
- `[H7]` paket dengan booking aktif bisa diarsipkan (`200`); booking #142 tetap `baru` di paket yang tidak tampil publik. `UpdateScheduleStatus` tidak punya aturan transisi.
- Daftar paket admin tidak menandai paket `published` yang tanggal berangkatnya sudah lewat.

## Yang sudah berfungsi benar

| Area | Bukti |
|---|---|
| F01: edit dari form basi tidak mengembalikan kursi | `[F3]` sisa 10→8 karena booking; `[F4]` simpan form lama → sisa tetap 8; `[F7]` kapasitas 20 → sisa 18 |
| F02: pindah brand paket ber-booking ditolak | `[G6]` 409 `brand paket yang memiliki booking tidak dapat diubah` |
| F04: detail publik terikat brand | `[H3]` brand salah → 404; `[H4]` brand benar → 200 |
| F11: kategori brand lain ditolak di paket | `[E9]` 400 `category_id tidak valid` |
| F14: DP melebihi harga ditolak | `[E5]` 400 |
| Harga coret promo | `[E4]` 400 `harga_coret harus lebih besar dari harga_quad` |
| Isolasi brand paket | `[G1]`–`[G5]` baca/ubah/arsip/hapus dari brand lain → 404 |
| Hapus data yang masih dipakai | hotel `[B7]`, maskapai `[C3]`, paket ber-booking `[G7]` → 409 |
| Duplikasi case-insensitive | hotel `[B3]`, maskapai `[C2]`, bandara `[C4]`, add-on `[D2]` → 409 |
| Master data khusus super admin | `[B1]` admin brand → 403 |
| Validasi hotel/maskapai/itinerary | `[B4]` bintang 6, `[C1]` kode 3 huruf, `[D4]` hari tanpa aktivitas → 400 |
| Itinerary replace-all dalam transaksi | `[D5]`/`[D6]` 2 hari → 1 hari |
| Itinerary publik hanya jika dipakai paket terbit | `[D7]` 404 sebelum, `[H6]` 200 sesudah |
| Paket arsip hilang dari publik | `[H8]` 404 |
| UI edit paket | Simpan tanpa perubahan pada 1440 px kembali ke daftar; sisa kursi 18/20 tetap |
| UI daftar paket 375 px | `scrollWidth` 375 = `clientWidth` 375 |

## Batas verifikasi

- Form paket pada 375 px tidak terverifikasi: panel browser berubah lebar dua kali sehingga emulasi viewport direset. Halaman lain diperiksa pada 1440 px (10 halaman termuat tanpa pesan error atau overflow).
- Tidak ada perubahan kode pada audit ini.
- Data uji ditambahkan ke database lokal: kategori #1, #3, #4; hotel #17–#19; bandara ZZA (#28); CGK dibuat ulang sebagai #29 setelah terhapus oleh uji C6; add-on #1; itinerary #1; paket #7–#13; booking #142. Booking #10 dipulihkan ke `dp` dan `minimal_dp` paket #1 dikembalikan ke 5 000 000.

## Status perbaikan (1 Oktober 2026)

| ID | Status | Perubahan |
|---|---|---|
| MP-01 | Diperbaiki | Keputusan: kategori khusus super admin. `POST/PUT/DELETE /categories` memakai `superAdmin` (`cmd/api/main.go`); admin brand hanya bisa membaca kategori yang terhubung ke brand-nya; nama maks. 100 karakter |
| MP-02 | Diperbaiki | Keputusan: DP disimpan saat booking dibuat. Migrasi `070_booking_dp_per_pax.sql` (kolom + backfill DP efektif saat migrasi); snapshot diisi di booking admin, finalisasi draft, dan CRM deal; `RequiredBookingDP` membaca `checkout.dp_per_pax` → `bookings.dp_per_pax` → paket → brand |
| MP-03 | Diperbaiki | Hotel slot Mekkah wajib berkota Makkah, slot Madinah wajib Madinah (otomatis menolak hotel sama untuk dua slot) |
| MP-04 | Diperbaiki | Paket bertanggal berangkat lampau tidak bisa diterbitkan (create, update tanggal/status, `PUT /status`); detail publik paket lampau 404. Draft/arsip lampau tetap boleh |
| MP-05 | Diperbaiki | Error internal paket & kategori dicatat di log, dijawab 500 generik; `jadwal_nama` maks. 255 karakter |
| MP-06 | Diperbaiki | Kode bandara paket (asal/tujuan pergi-pulang) wajib terdaftar, disimpan huruf besar; bandara yang dirujuk paket tidak bisa dihapus (409). Rute transit tidak divalidasi (format tersendiri) |
| MP-07 | Diperbaiki | Saat edit, `seat_sisa` dari form diabaikan; kapasitas di bawah alokasi mendapat pesan yang tepat |
| MP-08 | Diperbaiki | `shared.ValidMediaURL`: foto/video hotel, logo maskapai, dan brosur hanya `/uploads/…`, `/api/admin/media/…`, atau URL http(s) |
| MP-09 | Diperbaiki | Keputusan: (1) harga wajib Quad ≤ Triple ≤ Double (sama boleh) dan infant ≤ Quad; (2) paket belum berangkat dengan booking aktif tidak bisa diarsipkan (409, sarankan Draft), setelah tanggal berangkat lewat boleh; (3) daftar paket menampilkan badge "Sudah berangkat" dan tab "Perlu Diarsipkan", tanpa arsip otomatis |
| MP-10 (baru) | Terbuka | Booking admin bisa dibuat pada paket draft atau yang tanggal berangkatnya lewat (`[W11]` booking #192 di paket #14, 10 Jan 2025). `CreateBooking` tidak memeriksa status/tanggal paket |

Bukti (bagian MP4 di log):

```text
[V2]  admin brand 2 baca kategori tanpa brand 2 -> 404 ; [V1] kategori yang terhubung brand 2 -> 200
[V3][V4][V5] admin brand ubah/hapus/buat kategori -> 403 {"error":"hanya Super Admin Grup yang dapat mengakses ini"}
[V9]  hotel Madinah di slot Mekkah -> 400 {"error":"hotel_mekkah_id harus hotel di Makkah (hotel terpilih berada di Madinah)"}
[V11] terbit tanggal 2025 -> 400 {"error":"paket dengan tanggal berangkat yang sudah lewat tidak dapat diterbitkan"} ; [V12] draft lampau -> 201 ; [V13] terbitkan draft lampau -> 400
[V14] detail publik paket lampau -> 404
[V15] nama 300 karakter -> 400 {"error":"jadwal_nama maksimal 255 karakter"}
[V16] bandara "XXX" -> 400 ; [V17] "cgk" -> tersimpan "CGK"
[V18][V19][V21] brosur/foto/logo javascript: atau data: -> 400 ; [V20] /uploads/… dan YouTube -> 201
[V33] hapus CGK yang dipakai -> 409 {"error":"tidak bisa dihapus, masih dipakai oleh 15 paket"} ; [V23] hapus bandara tidak dipakai -> 200
[V24] kapasitas 1, teralokasi 2 -> 409 {"error":"perubahan paket ditolak: kapasitas tidak boleh kurang dari kursi teralokasi"}
[V26] booking #142 dp, dp_per_pax 5000000 ; [V27] DP paket 13 dinaikkan ke 6 jt ; [V28] add-on -> [V29] tetap dp
[V31] booking baru setelah DP naik: dp_per_pax 6000000
```

Bukti MP-09 (bagian MP5):

```text
[W1] triple < quad -> 400 {"error":"harga_triple tidak boleh lebih murah dari harga_quad"}
[W2] double < triple -> 400 ; [W3] infant > quad -> 400 ; [W4] harga seragam -> 201
[W6] arsip paket #13 (berangkat 2027, 2 booking aktif) -> 409 {"error":"paket belum berangkat dan masih punya 2 booking aktif; untuk menghentikan penjualan, ubah status ke Draft"}
[W7] arsip lewat form -> 409 (pesan sama) ; [W8] ubah ke draft -> 200
[W10] arsip paket tanpa booking -> 200 ; [W12] arsip paket lampau ber-booking -> 200
[UI] tab "Perlu Diarsipkan 2": dua paket "Berangkat Lampau" 10 Jan 2025 dengan badge "Sudah berangkat"
```

Tes baru: `TestSyncMemakaiSnapshotDPBooking` dan `TestBackfillDPMigration` (`internal/payment/dp_snapshot_test.go`), `TestValidMediaURL` (`internal/shared/media_url_test.go`). `go test ./...` dan `ERP_TEST_DB=1 go test -p 1 ./internal/payment ./internal/booking ./internal/schedule ./internal/crmdeal ./internal/selfbooking` lulus; jumlah bookings/payments/schedules sama sebelum dan sesudah tes. Migrasi 070 dijalankan dua kali (exit 0). Build Master/Travel Dashboard lulus.

Dampak ke data lama: paket lama yang menyimpan kode bandara di luar master, atau hotel yang kotanya tidak sesuai slot, harus diperbaiki saat diedit berikutnya.

## Urutan perbaikan yang disarankan

1. MP-01 (isolasi kategori) dan MP-02 (DP berlaku surut) — keduanya memerlukan keputusan: siapa yang boleh mengelola kategori, dan apakah DP disimpan per booking atau perubahan DP dikunci.
2. MP-05 dan MP-07 (penanganan error dan pesan) — mekanis, pola JB-05.
3. MP-03, MP-04, MP-06 (validasi data paket).
4. MP-08 dan MP-09 sesuai prioritas.
