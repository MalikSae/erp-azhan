# Audit manajemen seat

Tanggal: 8 Oktober 2026. Branch: `feat/pengguna-organisasi`.
Lingkungan: lokal Laragon, database `erp_azhan_dev`.

## Cakupan

Semua kode yang membaca atau mengubah `schedules.seat_total` dan `schedules.seat_sisa`, serta kolom pendukungnya (`bookings.seat_count`, `is_seat_blocked`, `seat_hold_expires_at`, `booking_pax.counts_for_seat`).

| Jalur | Berkas | Operasi pada `seat_sisa` |
|---|---|---|
| Booking admin | `booking/repository.go` `CreateBooking` | kurangi |
| Finalisasi draft | `booking` `FinalizeBooking` | kurangi |
| Blokir ulang kursi | `booking` `BlockSeat` | kurangi |
| Ubah status ke batal | `booking` `UpdateBookingStatus` | kembalikan |
| Lepas kursi manual | `booking` `CancelSeatBlock` | kembalikan |
| Batalkan pax | `booking` `CancelPax` | kembalikan |
| Self-booking | `selfbooking/repository.go` | kurangi |
| Deal CRM dan hold kedaluwarsa | `crmdeal/repository.go` | kurangi dan kembalikan |
| Pembayaran | `payment/repository.go` `SyncBookingStatusTx` | kurangi |
| Edit kapasitas | `schedule/repository.go` `prepareScheduleUpdate` | hitung ulang |
| Penyesuaian manual | `schedule` `UpdateSeat` (`PUT /api/admin/schedules/{id}/seat`) | set |

Metode: baca semua jalur di atas, cek invarian langsung di database, dan satu uji integrasi sementara (dalam transaksi yang di-rollback, file uji sudah dihapus).

Invarian yang dipakai: `seat_sisa = seat_total − Σ seat_count` dari booking dengan `is_seat_blocked=1` dan status bukan batal.

Tidak dicakup: uji beban atau konkurensi nyata, worker hold di luar fungsi yang dibaca, dan tampilan halaman publik.

## Ringkasan

| ID | Tingkat | Temuan | Status |
|---|---|---|---|
| SM-01 | Tinggi | Booking berbayar yang kursinya sudah dilepas manual, lalu naik `dp → lunas`, ditandai memegang kursi tanpa mengurangi `seat_sisa` | Diperbaiki |
| SM-02 | Sedang | 6 dari 24 paket menyimpang dari invarian (sisa < total tanpa booking); tidak ada UI untuk merekonsiliasi | Alat rekonsiliasi dibuat; angka 6 paket belum dikoreksi |
| SM-03 | Sedang | Penyesuaian seat manual hanya tercatat di log server, tanpa pelaku | Diperbaiki (riwayat di database, belum ada tampilan) |
| SM-04 | Rendah | Pengembalian kursi memakai `LEAST(seat_total, …)` sehingga kelebihan pengembalian tertutup diam-diam | Diperbaiki |
| SM-05 | Sedang | Booking admin dan deal CRM tidak memeriksa status paket atau tanggal berangkat | Diperbaiki (kebijakan bawaan, lihat di bawah) |

## Temuan

### SM-01 · Tinggi · Kursi ditandai terpegang tanpa dikurangi

`SyncBookingStatusTx` (`internal/payment/repository.go:429`) hanya mengurangi `seat_sisa` bila `currentStatus == "baru"`. Namun pernyataan `UPDATE bookings` sesudahnya selalu menulis `is_seat_blocked = TRUE` untuk target `dp` atau `lunas`. Akibatnya, untuk booking berstatus `dp` yang kursinya sudah dilepas lewat `CancelSeatBlock` (boleh untuk booking berbayar, UI-02), pembayaran berikutnya yang membawanya ke `lunas` menandai kursi terpegang tanpa menguranginya.

Bukti (uji sementara, rollback):
```text
status=lunas blocked=true seat_sisa sebelum=103 sesudah=103 (seat_count=1)
DRIFT: booking ditandai memegang kursi tetapi seat_sisa tidak berkurang
```

Dampak: `seat_sisa` lebih besar daripada kursi yang benar-benar tersedia, sehingga kursi bisa dijual lebih dari kapasitas. Penyimpangan ini lalu tersembunyi oleh SM-04.

**Perbaikan (8 Okt 2026):** syarat di `internal/payment/repository.go` diubah menjadi "target `dp`/`lunas` dan kursi belum terblokir", tanpa syarat status saat ini. Status `batal` dan `draft` sudah keluar lebih awal di fungsi yang sama. Bila kursi sudah habis terjual, fungsi mengembalikan `ErrSeatUnavailable`, sama seperti jalur `baru → dp`: konfirmasi pembayaran untuk booking itu ditolak sampai admin memutuskan.

Test regresi (`internal/payment/seat_release_test.go`): kursi diambil lagi saat naik status (sisa turun 1); ditolak dengan `ErrSeatUnavailable` bila kursi habis; tidak dikurangi dua kali bila booking masih memegang kursi.

### SM-02 · Sedang · Paket menyimpang dari invarian

Query pada `erp_azhan_dev`: 6 paket (id 18, 19, 20, 21, 22, 24; seluruhnya `published`, kapasitas 45) memiliki `seat_sisa` 8, 32, 40, 20, 12, 35 tanpa satu pun booking. Seharusnya 45. 18 paket lainnya konsisten. Tidak ada booking batal yang masih memegang kursi, dan seluruh booking terblokir memiliki `seat_count` yang cocok dengan pax reguler aktifnya.

Penyebabnya paket hasil impor atau seed dengan `seat_sisa` bawaan (`cmd/seed-packages`), bukan bug alur booking. Catatan di migrasi 068 juga menyebut selisih lama "perlu direkonsiliasi manual oleh admin".

Masalahnya, endpoint `PUT /api/admin/schedules/{id}/seat` sudah ada dan fungsi `updateScheduleSeat` sudah ada di `frontend/master-dashboard/src/api/schedules.js`, tetapi tidak ada halaman yang memakainya. Rekonsiliasi hanya bisa lewat panggilan API langsung.

**Perbaikan (8 Okt 2026):** halaman detail paket (Office Dashboard) kini punya panel "Kuota & Ketersediaan Seat" yang menampilkan sisa, total, kursi dipegang booking, dan kursi terpakai di luar booking (`seat_total − seat_sisa − dialokasikan`). Tombol "Sesuaikan Sisa Seat" membuka dialog dengan batas atas yang dihitung server, tautan "Isi sesuai booking", alasan wajib, dan daftar 20 riwayat penyesuaian terakhir (SM-03). Bila selisih negatif (sisa lebih besar dari seharusnya, gejala SM-01), panel menampilkan peringatan.

Server: `GET /api/admin/schedules/{id}/seat` (izin `schedule.view`, dibatasi brand) mengembalikan ringkasan dan riwayat itu. Penyesuaian tetap lewat `PUT` yang sama dan memerlukan `schedule.edit`.

Angka enam paket impor (id 18, 19, 20, 21, 22, 24) **tidak diubah**: hanya pihak yang tahu penjualan sebenarnya yang bisa menentukan sisa yang benar, dan tiap penyesuaian wajib beralasan.

Catatan: data ini sejalan dengan keputusan mempertahankan "Kuota Terisi" di form tambah (audit tambah paket TP-05), karena memang ada kursi terjual di luar sistem. Namun kursi itu tidak punya jejak booking, jadi tidak terlihat di mana pun selain angka sisa.

### SM-03 · Sedang · Penyesuaian manual tanpa jejak yang layak

`UpdateSeat` mencatat `log.Printf("[AUDIT] schedule_seat_adjustment id=… brand=… before=… after=… reason=…")`. Tidak ada tabel riwayat dan tidak ada ID pelaku, padahal modul lain (`adminuser`, `agen`) mencatat `by=`. Log hilang saat server di-restart atau log dirotasi, dan tidak bisa menjawab "siapa yang mengubah kursi paket ini".

**Perbaikan (8 Okt 2026):** migrasi `084_schedule_seat_adjustments.sql` membuat tabel `schedule_seat_adjustments` (paket, pelaku, sisa sebelum dan sesudah, alasan, waktu). `UpdateSeat` menulis satu baris di transaksi yang sama dengan perubahan kursi, sehingga keduanya berhasil atau gagal bersama. Log `[AUDIT]` kini memuat `by=<id admin>`. Alasan dipotong spasi dan dibatasi 500 karakter (400 bila lebih). `admin_user_id` menjadi NULL bila akun pelaku dihapus; riwayat ikut terhapus bila paketnya dihapus.

Belum ada tampilan atau endpoint untuk membaca riwayat; saat ini hanya lewat query ke tabel itu.

Yang sudah baik sejak awal: alasan wajib, nilai sebelumnya wajib cocok (`expected_seat_sisa`) sehingga tidak menimpa perubahan orang lain, baris dikunci `FOR UPDATE`, dan batas atasnya memperhitungkan kursi yang sudah dialokasikan booking.

### SM-04 · Rendah · `LEAST(seat_total, …)` menutupi kelebihan pengembalian

Pengembalian kursi di `UpdateBookingStatus`, `CancelSeatBlock`, `CancelPax`, dan hold CRM kedaluwarsa memakai `LEAST(seat_total, seat_sisa + n)`. Bila `seat_sisa` sudah terlalu besar (mis. akibat SM-01), pengembalian berikutnya terpotong di batas kapasitas tanpa peringatan atau log, sehingga penyimpangan tidak terdeteksi.

**Perbaikan (8 Okt 2026):** keempat tempat itu kini memanggil satu fungsi, `shared.RestoreSeats` (`internal/shared/seat.go`). Fungsi ini tetap membatasi hasil pada `seat_total`, tetapi mencatat `[WARN] seat_restore_capped schedule=… seat_sisa=… restore=… seat_total=…` di log bila terpotong. Tidak ada lagi `LEAST(seat_total, …)` pada kode aplikasi. Perhitungan batas diuji unit (`TestCapSeatRestore`).

### SM-05 · Sedang · Booking di paket yang sudah tertutup

Self-booking menolak paket yang tidak `published`. Booking admin (`CreateBooking`, `FinalizeBooking`) dan deal CRM (`crmdeal.Process`) tidak memeriksa status paket maupun tanggal berangkat, sehingga kursi bisa diambil di paket yang diarsipkan atau sudah berangkat. Di `erp_azhan_dev` sudah ada 1 booking aktif di paket yang sekarang tertutup; booking yang sudah ada tidak terpengaruh perbaikan ini.

**Perbaikan (8 Okt 2026), kebijakan bawaan:** booking baru ditolak bila paket **diarsipkan** atau **tanggal berangkatnya sudah lewat** (`shared.ScheduleClosedReason`, memakai tanggal WIB; berangkat hari ini masih boleh). Paket **draft tetap boleh dipesan** supaya pra-pendaftaran sebelum terbit bisa dicatat. Pengecekan dilakukan setelah baris paket dikunci, sebelum kursi dihitung:

| Jalur | Hasil bila tertutup |
|---|---|
| `POST /api/admin/bookings` | 400, "Booking tidak dapat dibuat: …" |
| Finalisasi draft booking | 400, "Booking tidak dapat difinalisasi: …" |
| Deal CRM | 409 (`ErrScheduleClosed`) |

Pembuatan draft booking tidak diubah karena tidak mengambil kursi; penolakan terjadi saat difinalisasi.

Kebijakan ini adalah pilihan saya karena belum ada keputusan tertulis. Bila draft juga harus ditolak, ubah satu kondisi di `ScheduleClosedReason` (`internal/shared/seat.go`) dan tabel ujinya.

## Yang sudah benar

- Semua jalur yang mengurangi kursi mengunci baris paket (`FOR UPDATE`) sebelum memeriksa `seat_sisa >= jumlah`, jadi dua booking paralel berurutan.
- Pemeriksaan kecukupan kursi ada di booking admin, finalisasi, blokir ulang, self-booking, deal CRM, dan pembayaran.
- Bayi tidak mengonsumsi kursi (`counts_for_seat`). `seat_count` dijaga sinkron dengan pax reguler aktif saat pax dibatalkan.
- Edit kapasitas tidak bisa di bawah kursi yang sudah teralokasi dan menjaga selisih kursi lama (`remainingAfterCapacityChange`), dengan pengecekan `expected_seat_total` untuk mencegah tabrakan edit.
- Status `batal` bersifat final untuk jalur pembayaran, dan status finansial tidak bisa diubah manual.
- Tidak ada jalur menambah pax ke booking yang sudah ada, sehingga tidak ada penambahan kursi tersembunyi.
- Pada data saat ini: tidak ada booking batal yang masih memegang kursi, dan `seat_count` cocok dengan pax aktif di semua booking terblokir.

## Verifikasi perbaikan

- Uji integrasi baru lulus (`ERP_TEST_DB=1`): tiga uji SM-01 dan `TestUpdateSeatTxRecordsAdjustmentWithActor` (pelaku tercatat; alasan kosong dan nilai sebelumnya basi ditolak tanpa meninggalkan catatan).
- Uji API setelah restart, pada paket sementara (sudah dihapus): tanpa alasan 409, nilai sebelumnya basi 409, alasan 501 karakter 400, dua penyesuaian sah 200. Tabel riwayat berisi dua baris dengan pelaku `admin@azhan.id`, dan baris itu ikut terhapus bersama paketnya.
- SM-02 dan SM-04: `TestSeatOverviewReconcilesWithBookings` (ringkasan cocok dengan perhitungan langsung, dibatasi brand, id tak ada menghasilkan ErrNotFound) dan `TestCapSeatRestore`. Di Office Dashboard, panel pada paket 18 menampilkan 8 dari 45, 0 booking, 37 pax terpakai di luar booking; pada paket sementara, validasi alasan kosong, penyimpanan, dan riwayat (dengan pelaku) berjalan (paket dihapus sesudahnya).
- `testdb.Tx` kini menyerialkan semua uji berbasis transaksi lewat kunci bernama `erp_test_tx_fixture`, karena paket uji paralel saling mengunci baris fixture dan menyebabkan deadlock pada `TestPublicScheduleEffectiveDPProjection`. Tiga kali `go test ./...` berturut-turut dengan `ERP_TEST_DB=1` lulus semua.
- SM-05: `TestScheduleClosedReason` (draft di masa depan boleh; diarsipkan, sudah berangkat, dan draft yang tanggalnya lewat ditolak; berangkat hari ini boleh). Uji API pada tiga paket sementara (sudah dihapus beserta booking ujinya): paket diarsipkan 400, paket berangkat 400 (keduanya tanpa mengubah kursi), paket draft 201 dengan sisa kursi berkurang 1. Jalur finalisasi dan deal CRM diubah dengan pola yang sama tetapi tidak diuji langsung selain lewat suite yang ada.
- Migrasi 084 sudah dijalankan pada `erp_azhan_dev`. Lingkungan lain perlu menjalankannya: `go run migrations/run.go migrations/084_schedule_seat_adjustments.sql`.

## Catatan pengujian

- `go test ./...` tanpa `ERP_TEST_DB=1` hanya menjalankan uji unit; semua uji integrasi dilewati. Dengan `ERP_TEST_DB=1` (transaksi di-rollback), seluruh paket lulus. Sebelum perbaikan isolasi di `testdb.Tx` (lihat bagian Verifikasi perbaikan), `TestPublicScheduleEffectiveDPProjection` kadang terkena deadlock saat dijalankan paralel dengan paket lain.
