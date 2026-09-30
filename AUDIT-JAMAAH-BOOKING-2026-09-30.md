# Audit end-to-end jamaah dan booking (jalur admin)

Tanggal: 30 September 2026. Branch: `dev-malik` (commit `65c25c4`, sama dengan `main`).
Lingkungan: lokal Laragon, MySQL 8.4.3, database `erp_azhan_dev` yang dibangun dari nol memakai seluruh file `migrations/` (001–066), backend `go run cmd/api/main.go`.

## Cakupan

Jalur admin (Master/Travel Dashboard → API `/api/admin/*`): data jamaah, booking langsung, draft dan finalisasi, kursi (blokir, lepas, batal, worker hold), pembayaran admin, add-on, diskon, batal per pax, progress, perlengkapan, dokumen, dan isolasi antar-brand.

Tidak dicakup: portal jamaah dan self-booking microsite (sudah diaudit 28–29 September), CRM deal, komisi agen, serta walkthrough UI di browser (tertunda, lihat Batas verifikasi).

Metode: review kode `internal/booking`, `internal/payment`, `internal/jamaah`, `internal/dokumen`, dan `internal/crmdeal` (worker hold), lalu uji langsung ke API memakai tiga akun: super admin, admin brand 1 (Travel Pertama), dan admin brand 2 (Brand Uji Dua). Semua respons dicatat mentah di `audit/jamaah-booking-evidence-2026-09-30.log`; token disamarkan.

Paket uji: `schedules.id=1`, 5 kursi, Quad Rp30 jt / Triple Rp33 jt / Double Rp36 jt / infant Rp8 jt, minimal DP Rp5 jt, berangkat 20 Desember 2026.

## Ringkasan

| ID | Tingkat | Temuan |
|---|---|---|
| JB-01 | Kritis | Kolom `booking_pax.perlengkapan_status`/`perlengkapan_tanggal` tidak pernah dibuat migrasi; di database baru detail booking selalu gagal, tetapi perubahannya sudah tersimpan |
| JB-02 | Tinggi | `bookings.seat_count` tidak diisi booking admin dan finalisasi draft, sehingga kursi bisa terjual melebihi kapasitas atau hilang |
| JB-03 | Tinggi | Booking admin yang sudah dibayar dapat kehilangan kursi otomatis setelah statusnya turun ke `baru` |
| JB-04 | Tinggi | Booking admin tidak mencegah jamaah yang sama terdaftar di beberapa booking aktif pada jadwal yang sama |
| JB-05 | Sedang | Pesan error SQL mentah dikirim ke klien, dengan status 400/500 |
| JB-06 | Sedang | Booking berstatus `batal` masih bisa diberi add-on, diskon, dan progress |
| JB-07 | Sedang | Diskon tidak dibatasi; total bisa menjadi 0 walau sudah ada pembayaran, dan status `lunas` sementara bisa terpicu |
| JB-08 | Sedang | Aturan PIC berbeda antara booking langsung dan finalisasi draft |
| JB-09 | Sedang | Validasi data jamaah minim (NIK, tanggal, email, HP, paspor) |
| JB-10 | Rendah | Tanggal pembayaran admin di masa depan diterima |
| JB-11 | Rendah | Dokumen admin menerima `file_url` sembarang |
| JB-12 | Catatan | Batal per pax pada booking `dp`/`lunas` tidak mengubah tagihan (perlu keputusan kebijakan) |
| JB-13 | Catatan | Dokumen acuan `analisis-modul-booking-jamaah.md` yang diwajibkan `AGENTS.md` tidak ada di repo |

## Temuan

### JB-01 · Kritis · Schema drift `booking_pax`

Kode membaca dan menulis `booking_pax.perlengkapan_status` dan `booking_pax.perlengkapan_tanggal` di 15 tempat (`internal/booking/repository.go:235`, `:996`, `:1048`, `:1083`, `:1866`, `:1954`, `:2000`, `:2052`, dan lainnya). Migrasi `048_perlengkapan_pax.sql` hanya menghapus kolom lama di `bookings`, dan tidak ada migrasi yang menambahkannya ke `booking_pax`. Kemungkinan kolom ini dulu ditambahkan manual di database developer, pola yang sama dengan kasus `payments.bukti_url` di `AGENTS.md`.

Dampak di database yang dibangun dari migrasi resmi (environment baru, server baru):

- `GET /api/admin/bookings/{id}` selalu gagal.
- Setiap aksi booking yang mengembalikan detail (buat booking, batal, add-on, diskon, progress, dan seterusnya) **sudah melakukan commit**, lalu gagal saat membaca ulang. Admin melihat "gagal", padahal booking tersimpan dan kursi terpotong. Kalau admin mengulang, terbentuk booking ganda.

Bukti:

```text
[B1-create-2reg-1inf] POST /api/admin/bookings -> 400      (run pertama, body tidak terbaca klien)
[B3-jamaah-sudah-booking-jadwal-sama] POST /api/admin/bookings -> 400 {"error":"booking.ListPax: Error 1054 (42S22): Unknown column 'bp.perlengkapan_status' in 'field list'"}
[C2-detail-booking] GET /api/admin/bookings/1 -> 400 {"error":"booking.ListPax: Error 1054 (42S22): Unknown column 'bp.perlengkapan_status' in 'field list'"}
```

Semua request pembuatan booking di atas dijawab 400, tetapi database berisi:

```text
id  id_booking  pic  pax                                              total_harga
1   TPMG49      1    J1:reguler/Quad,J2:reguler/Quad,J4:infant/-      68000000
2   TPJ769      2    J2:reguler/Double                                36000000
5   TPCZ6A      3    J4:reguler/Quad                                  30000000
6   TP4BWF      2    J2:reguler/Double                                36000000
seat_total 5, seat_sisa 0
```

Tindakan (sudah dilakukan atas persetujuan pengguna): `migrations/067_booking_pax_perlengkapan_status.sql`, idempoten dengan pola yang sama seperti `016`. Dijalankan dua kali di database lokal, keduanya exit 0. Setelah itu `D1-detail-booking-setelah-067 -> 200`. File ini **belum di-commit**. Environment yang sudah punya kolom manual tidak berubah saat migrasi dijalankan.

Rekomendasi tambahan: tambahkan pemeriksaan skema otomatis (misalnya uji integrasi yang membangun database dari `migrations/` lalu memanggil `GET /bookings/{id}`) agar drift seperti ini tertangkap sebelum rilis.

### JB-02 · Tinggi · `seat_count` tidak diisi booking admin

`CreateBooking` (`internal/booking/repository.go:460`) dan `FinalizeBooking` tidak mengisi `bookings.seat_count`, sehingga nilainya default `1`, berapa pun jumlah pax regulernya. Self-booking dan CRM deal mengisinya. Kolom ini dipakai di:

| Pemakai | Lokasi | Akibat |
|---|---|---|
| Blokir ulang kursi | `booking/repository.go:1410–1413` | Hanya mengunci 1 kursi untuk booking 2+ orang, sehingga kursi bisa terjual melebihi kapasitas |
| Worker hold kedaluwarsa | `crmdeal/repository.go:365` | Hanya mengembalikan 1 kursi, sehingga kursi hilang |
| Penyesuaian kuota manual | `schedule/repository.go:444` | Batas `seat_total − SUM(seat_count)` terlalu longgar, sehingga admin bisa menjual melebihi kapasitas |
| Fallback batal/lepas blokir | `booking/repository.go:1269`, `:1354` | Salah jika tidak ada pax reguler aktif |

Bukti blokir ulang pada booking 2 orang (booking #8):

```text
[H1] seat_sisa 2   (booking 2 reguler dibuat; seat_count=1)
[H2] DELETE seat-block -> 200 ; seat_sisa 4   (+2, benar)
[H3] PUT seat-block    -> 200 ; seat_sisa 3   (-1, seharusnya -2)
[H4] PUT status batal  -> 200 ; seat_sisa 5   (+2) — padahal booking #1 masih memegang 1 kursi
```

Bukti penyesuaian kuota manual:

```text
[Q0] seat_total 5 | seat_sisa 2 | terpakai (pax reguler aktif terblokir) 3 | SUM(seat_count) 2
[Q1] PUT /api/admin/schedules/1/seat {"expected_seat_sisa":2,"seat_sisa":3} -> 200 {"id":1,"seat_sisa":3}
```

Hasilnya kapasitas efektif menjadi 6 dari 5 kursi. Bukti worker ada di JB-03.

Rekomendasi: isi `seat_count` dengan jumlah pax reguler aktif di `CreateBooking` dan `FinalizeBooking`, sinkronkan setiap kali pax reguler dibatalkan, lalu siapkan migrasi backfill untuk booking lama (`seat_count = COUNT pax counts_for_seat AND aktif`). Alternatifnya, hitung dari `booking_pax` di semua pemakai di atas.

### JB-03 · Tinggi · Booking admin berbayar kehilangan kursi otomatis

`payment.SyncBookingStatusTx` (`internal/payment/repository.go:443–445`) memberi `seat_hold_expires_at = NOW() + 24 jam` pada setiap booking terblokir yang statusnya turun ke `baru`. Worker `ReleaseExpiredSeatHolds` (`internal/crmdeal/repository.go:294`) kemudian melepas kursi booking `baru` yang hold-nya lewat, tanpa memeriksa pembayaran dan tanpa membedakan jalur booking. Ini bertentangan dengan aturan di `AGENTS.md`: booking admin menahan kursi *tanpa batas waktu*.

Status bisa turun dari `dp`/`lunas` ke `baru` melalui jalur biasa, misalnya koreksi diskon yang salah input atau penambahan add-on.

Bukti (booking #11, 2 pax, dibayar Rp6 jt):

```text
[M2] status baru, hold NULL, total 60000000, paid 6000000
[M3] POST diskon 54 jt -> status lunas, total 6000000
[M4] DELETE diskon     -> status baru, seat_hold_expires_at 2026-10-01 20:43:42
[M5] fixture: hold dimundurkan 1 menit (simulasi 24 jam), tunggu worker
[M6] status baru, is_seat_blocked 0, paid 6000000 ; seat_sisa 1 -> 2 (hanya +1 untuk 2 pax)
```

Rekomendasi: jangan memberi hold berbatas waktu pada booking yang punya pembayaran terkonfirmasi atau booking dari jalur admin; worker sebaiknya melewati booking yang punya pembayaran `confirmed`.

### JB-04 · Tinggi · Duplikasi jamaah lintas booking pada jadwal yang sama

Validasi duplikasi di `CreateBooking` hanya berlaku di dalam satu payload (`repository.go:379`). Perbaikan SB01 hanya diterapkan pada self-booking. Jamaah J2 berhasil terdaftar di tiga booking aktif pada jadwal yang sama (#1, #2, #6), dan masing-masing memotong kursi.

```text
[B3-jamaah-sudah-booking-jadwal-sama] POST /api/admin/bookings -> 400 (tetapi tersimpan; lihat JB-01)
bookings: 1 = J1,J2,J4 ; 2 = J2 ; 6 = J2
```

Rekomendasi: terapkan pemeriksaan yang sama dengan self-booking (anggota aktif pada `schedule_id` yang sama, di dalam transaksi) untuk booking admin dan finalisasi draft.

### JB-05 · Sedang · Error SQL mentah dikirim ke klien

`handleRepoError` di booking (`handler.go:964–965`) mengembalikan `err.Error()` apa adanya dengan status 400. Handler jamaah mengembalikan 500 berisi pesan SQL. `AGENTS.md` mewajibkan error dalam Bahasa Indonesia yang tidak membocorkan SQL mentah.

```text
[C2]  400 {"error":"booking.ListPax: Error 1054 (42S22): Unknown column 'bp.perlengkapan_status' in 'field list'"}
[F4]  POST addons nominal 1e15 -> 400 {"error":"booking.AddAddon insert: Error 1264 (22003): Out of range value for column 'nominal' at row 1"}
[P5]  POST jamaah tanggal_lahir "31-12-1990" -> 500 {"error":"terjadi kesalahan internal: jamaah.Create: Error 1292 (22007): Incorrect date value: '31-12-1990' for column 'tanggal_lahir' at row 1"}
```

Rekomendasi: bedakan error validasi bisnis (400, pesan terkontrol) dari error internal (500, pesan generik + log server). Validasi batas `nominal` (DECIMAL(12,0)) dan format tanggal di handler.

### JB-06 · Sedang · Booking `batal` masih bisa diubah

Tidak ada gate status di add-on, diskon, dan progress header. Pembayaran ke booking batal sudah benar ditolak.

```text
[I1] batal booking #1 -> status batal, total 40000000, paid 6000000 (2 payment confirmed tetap)
[I2] POST addons    -> 201
[I3] POST discounts -> 200
[I4] POST payments  -> 409 {"error":"reservasi tidak aktif; hubungi admin untuk memastikan kursi sebelum pembayaran"}
[I5] PUT progress   -> 200
```

Selain itu tidak ada jejak pengembalian dana untuk Rp6 jt yang sudah terkonfirmasi pada booking yang dibatalkan. Ini sejalan dengan kebijakan "refund ditangani admin", tetapi belum ada penanda di data.

Rekomendasi: tolak perubahan finansial dan progress pada booking `batal` (dan `draft` untuk pembayaran); pertimbangkan status/penanda refund untuk booking batal yang punya pembayaran.

### JB-07 · Sedang · Diskon tidak dibatasi

Diskon dicatat berapa pun nilainya; `total_harga` dijepit ke 0 lewat `GREATEST`. Akibatnya:

```text
[F2] diskon 200 jt -> total_harga 0, paid 6000000, status dp   (bukan lunas; kelebihan bayar Rp6 jt tidak ditandai)
[M3] diskon 54 jt  -> total 6000000 = paid, status lunas -> komisi.OnBookingLunas terpanggil
[M4] diskon dihapus -> status baru (lihat JB-03)
```

`OnBookingLunas` idempoten, tetapi pencatatan komisi dan `pertama_lunas_at` yang terjadi saat `lunas` sementara tidak dibalik ketika status turun lagi (`internal/komisi/hook.go:16–19`).

Rekomendasi: batasi diskon agar `total_harga ≥ total pembayaran terkonfirmasi` (atau minimal ≥ 0 sebelum dijepit), dan minta konfirmasi eksplisit bila diskon menurunkan tagihan di bawah jumlah yang sudah dibayar.

### JB-08 · Sedang · Aturan PIC tidak konsisten

`FinalizeBooking` mewajibkan PIC menjadi pax reguler di booking itu (`handler.go:432`); `CreateBooking` tidak. Booking #5 tersimpan dengan PIC J3 yang tidak ada di daftar pax, dan satu-satunya pax adalah J4 (lahir 1 Maret 2026, usia 9 bulan saat berangkat) sebagai `reguler` Quad dengan harga dewasa, tanpa peringatan.

Rekomendasi: samakan aturan PIC di kedua jalur. Tentukan kebijakan apakah jamaah usia infant boleh didaftarkan sebagai reguler; bila boleh, tampilkan peringatan di UI.

### JB-09 · Sedang · Validasi data jamaah minim

```text
[A9]  nik "123"                        -> 201
[P1]  tanggal_lahir "2030-01-01"       -> 201
[P2]  email "bukan-email"              -> 201
[P3]  no_hp "abc"                      -> 201
[P4]  paspor_berlaku_sampai 2020-01-01 -> 201
[P5]  tanggal_lahir "31-12-1990"       -> 500 (SQL mentah, lihat JB-05)
```

Yang sudah benar: NIK duplikat dalam brand ditolak (`A7b -> 409 {"error":"NIK sudah terdaftar"}`), nama kosong ditolak (`P6 -> 400`).

Rekomendasi: NIK 16 digit angka, tanggal lahir tidak di masa depan, format email/HP, serta peringatan (bukan penolakan) untuk paspor yang kedaluwarsa atau kurang dari 6 bulan sebelum keberangkatan.

### JB-10 · Rendah · Tanggal pembayaran masa depan diterima (admin)

```text
[E4] POST /api/admin/bookings/1/payments {"tanggal":"2030-01-01"} -> 201 ... "status":"confirmed"
```

Perbaikan JM-22 hanya menolak tanggal masa depan di portal. Pembayaran admin langsung berstatus `confirmed`, sehingga salah input tanggal langsung memengaruhi laporan dan analitik 30 hari.

### JB-11 · Rendah · Dokumen admin menerima `file_url` sembarang

```text
[O1] file_url "https://contoh-luar.example/paspor.pdf" -> 200 status submitted
[O2] file_url "/uploads/../../.env"                    -> 200 status submitted
[O3] file_url "/uploads/tidak-ada.pdf"                 -> 200 status submitted
```

Static server aman dari traversal (`[R] GET /uploads/../../.env -> 404`, juga varian `%2F`, `%2e%2e`, `\`). Namun data dokumen bisa menunjuk ke berkas yang tidak ada atau URL luar. Validasi kepemilikan JM-07 hanya berlaku di portal. Rekomendasi: terapkan validasi metadata unggahan yang sama untuk jalur admin.

### JB-12 · Catatan · Batal per pax pada booking `dp`/`lunas`

`CancelPax` hanya meng-nol-kan `harga_pax` bila booking masih `baru`/`draft` (`repository.go:1047–1051`). Pada `dp`/`lunas`, harga pax batal tetap masuk tagihan. Ini kemungkinan disengaja (tidak ada refund otomatis), tetapi belum terdokumentasi dan belum diuji langsung pada status `dp` dalam audit ini. Perlu keputusan kebijakan.

### JB-13 · Catatan · Dokumen acuan hilang

`AGENTS.md` mewajibkan membaca `analisis-modul-booking-jamaah.md` sebelum menyentuh modul booking, tetapi file itu tidak ada di repo (juga `spesifikasi-teknis-sistem-umroh.md`, `form-input-paket-umroh.md`, `arsitektur-modular-erp-azhan-grup.md`, `fase-development-microsite-travel.md`).

## Yang sudah berfungsi benar

| Area | Bukti |
|---|---|
| Isolasi brand (baca dan ubah) | `A10`, `A11`, `B9`, `N1`–`N5`, `N7` → 404; `A12`, `N6` hanya menampilkan data brand sendiri |
| Kunci baris kursi saat paralel | `S2`: 4 booking bersamaan, sisa 2 kursi → 2 × 201, 2 × 400, `seat_sisa` akhir 0 |
| Static `/uploads` terhadap traversal | 4 varian → 404, isi `.env` tidak terbaca |
| Status finansial manual ditolak | `E5 -> 400` |
| Pembayaran melebihi sisa tagihan ditolak | `E3 -> 400 (sisa: Rp 63000000)` |
| Pembayaran ke booking batal ditolak | `I4 -> 409` |
| Duplikasi jamaah dalam satu payload | `B2 -> 400` |
| Validasi infant dan PIC infant | `B5`, `B6 -> 400` |
| Paspor progress manual ditolak | `J1 -> 400` |
| Draft tidak memotong kursi | `K1` seat tetap |
| Hapus jamaah yang punya booking | `L1 -> 409 {"error":"tidak bisa dihapus, masih dipakai oleh data booking"}` |
| Batal booking mengembalikan kursi pax aktif | `D4`: 0 → 3 setelah 3 booking 1-pax dibatalkan |

## Verifikasi dan batasnya

- `go test ./...`: 13 paket lulus, exit 0. Uji integrasi database (`ERP_TEST_DB=1`) belum dijalankan dalam sesi ini karena pemeriksa izin otomatis tidak memberi keputusan.
- Uji tambahan: pembuatan data, pembayaran, diskon, pembatalan, worker, dan paralel memakai API nyata di database lokal. Satu-satunya manipulasi langsung database: `brands.kode_brand='TP'` untuk brand bawaan migrasi, `seat_hold_expires_at` dimundurkan 1 menit (simulasi 24 jam), dan `seat_sisa` disetel 2 untuk uji paralel.
- Uji UI Master Dashboard (375 dan 1440) dilakukan setelah perbaikan; lihat bagian Uji UI. Belum dilakukan: uji UI Travel Dashboard (admin brand), distribusi perlengkapan dengan stok (butuh template set), CRM deal.
- Data uji tertinggal di database lokal: brand 2 "Brand Uji Dua", dua admin brand, 20 jamaah uji, jadwal "Uji Audit Desember", booking #1–#13.

## Status perbaikan (30 September 2026)

| ID | Status | Perubahan |
|---|---|---|
| JB-01 | Diperbaiki | `migrations/067_booking_pax_perlengkapan_status.sql` (idempoten) |
| JB-02 | Diperbaiki | `CreateBooking` dan `FinalizeBooking` mengisi `seat_count` = pax reguler; `CancelPax` menyelaraskannya; backfill `migrations/068_backfill_booking_seat_count.sql` |
| JB-03 | Diperbaiki (jalur penyebab) | `SyncBookingStatusTx` tidak memberi hold 24 jam bila ada pembayaran terkonfirmasi |
| JB-04 | Diperbaiki | `ensureJamaahFreeOnSchedule` di `CreateBooking` dan `FinalizeBooking`, setelah baris jadwal terkunci |
| JB-05 | Diperbaiki | `BusinessError` untuk pesan bisnis (400); error lain di booking/payment/jamaah dicatat di log dan dijawab 500 generik; batas nominal add-on/diskon; validasi format tanggal jamaah |
| JB-06 | Diperbaiki (gate) | `lockNotBatalTx` pada add-on/diskon (tambah & hapus); gate `batal` pada progress header dan ubah tipe kamar. Penanda refund untuk booking batal berbayar belum dibuat (butuh keputusan kebijakan) |
| JB-07 | Diperbaiki | Keputusan: total tagihan tidak boleh di bawah pembayaran terkonfirmasi. `ensureTotalCoversPaidTx` setelah hitung ulang pada tambah diskon, hapus add-on, dan ubah tipe kamar (satu transaksi, termasuk efek komisi). Batal pax tidak dibatasi agar pembatalan tetap bisa dilakukan |
| JB-08 | Diperbaiki | Keputusan: PIC wajib pax reguler; jamaah usia infant wajib didaftarkan sebagai infant. Berlaku pada booking langsung dan finalisasi draft. Form dashboard sudah mengikuti aturan ini |
| JB-12 | Ditetapkan | Keputusan: harga pax batal pada booking DP/Lunas tetap ditagih; refund manual oleh admin. Dialog "Batalkan Pax" menampilkan keterangan ini untuk booking DP/Lunas |
| JB-09 | Diperbaiki | `validateJamaahInput`: NIK dan NIK darurat 16 digit, tanggal lahir/keluar paspor tidak di masa depan (WIB), masa berlaku tidak sebelum tanggal keluar, format email, HP 10–15 digit (aturan self-booking). Paspor kedaluwarsa tetap diterima. Data lama yang tidak valid harus diperbaiki saat diedit berikutnya |
| JB-10 | Diperbaiki | Tanggal pembayaran admin wajib `YYYY-MM-DD` dan tidak di masa depan (WIB), sama dengan JM-22 |
| JB-11 | Diperbaiki | `media.ValidateAdminUpload`: berkas wajib kategori `dokumen-jamaah`, tercatat di `media_uploads` sebagai unggahan brand jamaah/super admin/jamaah itu sendiri, dan ada di disk |
| JB-13 | Diperbaiki | Keputusan: hapus rujukan. Dokumen acuan tidak pernah di-commit (`git log --all` kosong untuk keenamnya); `design-system.md` di root sengaja dihapus di `24d88c2`. `AGENTS.md` kini merujuk `migrations/`, `README.md`, handler, dan file `AUDIT-*`/`PERBAIKAN-*`. Komentar kode yang menyebut `agen-azhan.md` tidak diubah |
| Refund | Dibuat | Keputusan: catatan refund sederhana. Migrasi `069_booking_refunds.sql`; `GET/POST /api/admin/bookings/{id}/refunds` (role admin); field `total_dibayar`, `total_refund`, `perlu_refund`; panel "Pengembalian Dana" dan badge "Perlu refund". Refund hanya untuk booking batal, total tidak melebihi pembayaran terkonfirmasi, tanpa ubah/hapus |

Bukti verifikasi (`audit/jamaah-booking-evidence-2026-09-30.log`, bagian RUN5 dan RUN6):

```text
[V1]  booking 2 reguler -> seat_count 2 ; seat_sisa 7 -> 5
[V2a] DELETE seat-block -> 7 ; [V2b] PUT seat-block -> 5 (-2) ; [V2c] batal -> 7
[V3a] batal 1 dari 3 pax -> seat_count 3 -> 2 ; [V3b] lepas -> 7 ; [V3c] blokir ulang -> 5 (-2)
[V4a] finalisasi draft 2 pax -> seat_count 2
[V5a] POST /bookings -> 400 {"error":"Jamaah Verif V3 sudah terdaftar pada booking TPFRYS di jadwal ini"}
[V5c] POST /bookings/33/finalize -> 400 {"error":"Jamaah Verif V4 sudah terdaftar pada booking TPFRYS di jadwal ini"}
[V5d] jamaah yang booking-nya batal -> 201 ; [V5e] jamaah yang pax-nya batal -> 201
[V6c] booking #36 paid 6000000, diskon dihapus -> status baru, is_seat_blocked 1, seat_hold_expires_at NULL
[V9]  rekonsiliasi: seat_total 16, seat_sisa 3, terpakai 13 (cocok)
```

Bukti JB-05/JB-06 (bagian RUN7):

```text
[X1] POST addons nominal 1e15   -> 400 {"error":"nominal terlalu besar"}                          (dulu 400 berisi Error 1264)
[X3] POST jamaah "31-12-1990"   -> 400 {"error":"tanggal_lahir harus berformat YYYY-MM-DD"}      (dulu 500 berisi Error 1292)
[X4] duplikat dalam payload     -> 400 {"error":"Jamaah Budi Santoso Uji didaftarkan lebih dari satu kali dalam booking ini"}
[X6] addon booking aktif        -> 201
[Y1][Y2][Y4][Y5][Y6][Y7] booking #1 batal: tambah/hapus addon, tambah/hapus diskon, progress, tipe kamar
                                -> 409 {"error":"booking sudah dibatalkan, data tagihan dan progress tidak dapat diubah"}
[Y0]/[Y8] total 40000000, addons 2, diskon 1, progress_hotel 1 — tidak berubah
```

Bukti JB-07/JB-08 (bagian RUN8):

```text
[Z1]  booking #36 total 61500000, paid 6000000; diskon 55500001 -> 400 {"error":"total tagihan tidak boleh lebih kecil dari pembayaran terkonfirmasi (Rp 6000000)"} ; data tidak berubah
[Z2]  diskon 55500000 (total = dibayar) -> 200, status lunas
[Z7]  booking #70 Double lunas 36000000; ubah ke Quad -> 400 (Rp 36000000) ; tetap 36000000 lunas
[Z9]  PIC tidak ikut sebagai pax -> 400 {"error":"Kontak Utama (PIC) harus ikut sebagai pax reguler dalam booking ini"}
[Z12] bayi (lahir 2026-07-01) sebagai reguler -> 400 {"error":"Jamaah Bayi Uji Reguler berusia di bawah 2 tahun pada tanggal keberangkatan, harus didaftarkan sebagai infant"}
[Z14] finalisasi draft dengan bayi reguler -> 400 (pesan sama)
[Z15] bayi sebagai infant -> 201
```

Tes baru JB-07: `TestEnsureTotalCoversPaidTx`.

Bukti JB-09/JB-10/JB-11 (bagian RUN9):

```text
[Q1] nik "123"                              -> 400 {"error":"NIK harus 16 digit angka"}
[Q2] tanggal_lahir 2030-01-01               -> 400 {"error":"tanggal_lahir tidak boleh di masa depan"}
[Q3] email "bukan-email"                    -> 400 {"error":"format email tidak valid"}
[Q4] no_hp "abc"                            -> 400 {"error":"no_hp harus berisi 10–15 digit angka"}
[Q5] berlaku 2023 < keluar 2024             -> 400 {"error":"paspor_berlaku_sampai tidak boleh sebelum tanggal_paspor_keluar"}
[Q6] paspor kedaluwarsa 2020                -> 201 (diterima)
[Q7] data valid, HP "+62 812-3456-0881"     -> 201
[Q8b] simpan ulang jamaah #1 tanpa perubahan -> 200
[R1] pembayaran tanggal 2030-01-01          -> 400 {"error":"tanggal pembayaran tidak valid atau berada di masa depan"}
[R2] tanggal "01-10-2026"                   -> 400 (pesan sama) ; [R3] hari ini -> 201
[S1] URL eksternal / [S2] "/uploads/../../.env" -> 400 {"error":"berkas harus diunggah melalui formulir dokumen jamaah"}
[S3] berkas tidak tercatat                  -> 400 {"error":"berkas bukan unggahan brand jamaah ini"}
[S4] unggah PDF lewat /api/admin/media/upload -> 201 ; [S5] simpan dokumen dari unggahan itu -> 200
```

Tes baru: `TestValidateJamaahInput` (tanpa database), `TestValidateAdminUpload` (berkas brand lain, URL luar, traversal, berkas hilang ditolak; unggahan brand sendiri dan super admin diterima).

Bukti catatan refund (bagian RUN10/RUN11):

```text
[T0]  booking #1 batal: total_dibayar=6000000 total_refund=0 perlu_refund=True
[T2]  refund booking aktif           -> 400 {"error":"pengembalian dana hanya dapat dicatat untuk booking yang sudah dibatalkan"}
[T5]  refund 8 jt dari 7 jt diterima -> 400 {"error":"jumlah pengembalian melebihi dana yang belum dikembalikan (Rp 7000000)"}   (format kini "Rp 7.000.000")
[T6]  tanggal 2999-01-01             -> 400 {"error":"tanggal pengembalian tidak valid atau berada di masa depan"}
[T7]  4 jt -> 201 ; [T8] 3,5 jt dari sisa 3 jt -> 400 ; [T9] 3 jt -> 201
[T10] total_refund=7000000 perlu_refund=False ; [T11] refund lagi -> 400 {"error":"tidak ada dana yang perlu dikembalikan untuk booking ini"}
[T13] token brand 2 ke booking brand 1 -> 404
[V1]  pesan JB-07 kini: "total tagihan tidak boleh lebih kecil dari pembayaran terkonfirmasi (Rp 36.000.000)"
```

Tes baru: `TestCreateRefundValidasiInput`, `TestTrimOptional`, `TestFormatRupiah` (`internal/booking/refund_test.go`).

### Uji UI (Master Dashboard, super admin)

1440 px:

- Daftar booking: badge "Perlu refund" di samping status Batal pada TPMG49.
- Detail booking #1: panel "Pengembalian Dana" (diterima Rp 6.000.000, sudah Rp 0, belum Rp 6.000.000). Modal menolak Rp 7.000.000 di sisi klien; Rp 6.000.000 tersimpan, panel menjadi "Selesai Rp 0", riwayat menampilkan metode dan catatan, badge daftar hilang.
- Modal diskon booking #70 (lunas Rp 36 jt) menampilkan pesan JB-07.
- Dialog "Batalkan Pax" booking #10 (DP) menampilkan keterangan JB-12. Pembatalan tidak dikonfirmasi.
- Form jamaah: NIK "123" menampilkan "NIK harus 16 digit angka", data tidak tersimpan.

375 px: panel refund 1 kolom, `scrollWidth` 375 = `clientWidth` 375; daftar booking tanpa overflow halaman, tabel bergeser di dalam wrapper.

### Temuan baru dari uji UI

| ID | Tingkat | Temuan | Status |
|---|---|---|---|
| UI-01 | Sedang | Halaman di `frontend/shared` memakai `shared/src/components/ui/Button.jsx` dengan inline style `var(--brand-primary, #FED853)`. Di Master Dashboard variabel itu tidak ada, jadi tombol utama tetap kuning lama. `design-system.md` juga masih mendokumentasikan palet kuning | Diperbaiki: `master-dashboard/src/index.css` mengisi `--brand-primary` (onyx) dan `--brand-primary-text` (gading); tombol "Buat Booking Baru" terukur `rgb(22,24,27)` / `rgb(247,245,240)`. Bagian warna `.agents/rules/design-system.md` diperbarui ke palet holding |
| UI-02 | Sedang | "Cancel Block Seat" tersedia pada booking berbayar; kursi yang dilepas tidak bisa dikunci ulang karena `BlockSeat` hanya menerima status `baru` | Keputusan: tetap boleh dengan konfirmasi. Dialog menampilkan peringatan bila `total_dibayar > 0`. `BlockSeat` kini menerima `dp`/`lunas` dengan kunci permanen (`expires_at` diabaikan) |
| UI-03 | Rendah | Error simpan form jamaah tampil di atas form, tombol simpan di bawah. `window.scrollTo` yang ada tidak berefek karena konten menggulir di container dalam | Diperbaiki: gulir ke elemen pesan lewat `ref`; pesan terukur terlihat (top 193 dari tinggi layar 914) setelah simpan dari bawah |
| UI-04 | Catatan | Analitik transaksi 30 hari tidak mengurangi refund | Diperbaiki: nilai bersih pembayaran − refund per tanggal; `count` tetap jumlah pembayaran. Endpoint belum dipakai UI saat ini |

Bukti UI-02/UI-04 (bagian RUN12):

```text
[A0] hitung manual brand 1 hari ini: bayar 64000000, refund 13000000, jumlah_bayar 6
[A1] GET /api/admin/analytics/transactions-30-days -> 200 [{"date":"2026-09-30","brand_id":1,"total_amount":51000000,"count":6}]
[B1] DELETE /bookings/10/seat-block (status dp) -> 200 ; seat_sisa 3 -> 5
[B2] PUT /bookings/10/seat-block + expires_at 2 hari -> 200 ; seat_sisa 5 -> 3, seat_hold_expires_at NULL
[B3] PUT /bookings/1/seat-block (batal) -> 400 {"error":"status tidak valid"}
```

Tes baru JB-05/JB-06: `TestHandleRepoErrorTidakMembocorkanSQL` (tanpa database) dan `TestLockNotBatalTx` (`internal/booking/error_handling_test.go`).

Tes baru: `TestEnsureJamaahFreeOnSchedule`, `TestBackfillSeatCountMigration` (`internal/booking/seat_integrity_test.go`), `TestSyncPaidDowngradeKeepsSeatWithoutExpiry` (`internal/payment/audit_selfbooking_test.go`). Tes JB-03 gagal pada kode lama (`expiry={2026-10-01 20:53:49 +0700 WIB true}`) dan lulus pada kode baru. `go test ./...` dan `ERP_TEST_DB=1 go test -p 1 ./internal/booking ./internal/payment` lulus; jumlah bookings/payments/jamaah/seat_sisa di database sama sebelum dan sesudah tes.

Fixture tes (`internal/testdb/testdb.go`) juga diperbaiki: pada checkout Windows (`core.autocrlf=true`) seluruh tes integrasi booking/payment gagal dengan `Error 1215: Cannot add foreign key constraint` karena penghapusan FK tabel sementara tidak cocok dengan akhir baris CRLF.

Catatan penerapan:

- Jalankan 067 lalu 068 berurutan. Keduanya aman diulang.
- 068 tidak mengubah `schedules.seat_sisa`. Selisih yang sudah terjadi di data lama perlu diperiksa manual dengan query berikut dan dikoreksi melalui penyesuaian kuota:

```sql
SELECT s.id, s.jadwal_nama, s.seat_total, s.seat_sisa,
       (SELECT COUNT(*) FROM booking_pax p JOIN bookings b ON b.id = p.booking_id
        WHERE b.schedule_id = s.id AND b.is_seat_blocked = 1 AND b.status <> 'batal'
          AND p.counts_for_seat = 1 AND p.pax_status = 'aktif') AS terpakai
FROM schedules s
HAVING seat_total - seat_sisa <> terpakai;
```

- Keputusan kebijakan yang masih terbuka dari JB-03: worker tetap melepas hold booking self-booking yang sudah punya pembayaran terkonfirmasi di bawah DP ketika hold awalnya habis. Perbaikan ini hanya menutup jalur hold baru pada status yang turun.

## Urutan perbaikan yang disarankan

1. JB-01: commit dan rilis migrasi 067 bersama tes pembangunan skema dari nol.
2. JB-02 dan JB-03 bersama (keduanya menyentuh kursi): isi dan sinkronkan `seat_count`, backfill booking lama, lalu perbaiki kebijakan hold dan worker.
3. JB-04: duplikasi jamaah lintas booking.
4. JB-05 dan JB-06: gate status dan penanganan error.
5. Sisanya sesuai prioritas bisnis; JB-07, JB-08, dan JB-12 membutuhkan keputusan kebijakan lebih dulu.
