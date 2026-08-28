# Laporan Audit Menyeluruh: Alur Jamaah, Booking, dan Pembayaran

**Dokumen**: `AUDIT-jamaah-booking-payment.md`  
**Cakupan**: Backend (Golang Chi + MySQL) & Frontend (Shared Package, Master Dashboard, Travel Dashboard)  
**Tipe**: Pure Investigation & Security/Logic Audit (Read-Only)  

---

## 1. Ringkasan Eksekutif

1. **Akses Publik Tanpa Autentikasi pada Dokumen Sensitif Jamaah (Kritis)**:  
   Semua file dokumen pribadi (KTP, Paspor, KK, Buku Nikah) disimpan di direktori `./uploads/dokumen-jamaah/` dan disajikan langsung ke publik via HTTP FileServer `r.Handle("/uploads/*", ...)` di `cmd/api/main.go:151` tanpa enkripsi at-rest, tanpa verifikasi autentikasi/otorisasi, dan tanpa URL sementara bertanda tangan (*signed temporary URLs*). Siapapun yang memiliki/menebak path UUID file dapat mengunduh dokumen sensitif jamaah secara anonim.

2. **Desinkronisasi State `is_seat_blocked` dan Hilangnya Kuota Kursi saat Pembayaran (Kritis)**:  
   Pada saat pembayaran diverifikasi via `payment.UpdateStatus` (`internal/payment/repository.go:293`), transisi status booking dari `baru` ke `dp`/`lunas` mengurangi `schedules.seat_sisa`, namun **TIDAK** mengubah flag `bookings.is_seat_blocked` menjadi `TRUE` (tetap `FALSE`). Akibatnya, jika booking tersebut kemudian dibatalkan via `booking.UpdateBookingStatus`, kursi **TIDAK AKAN PERNAH** dikembalikan ke `schedules.seat_sisa` karena syarat `is_seat_blocked == true` tidak terpenuhi. Selain itu, `schedules` tidak di-lock dengan `FOR UPDATE` di fungsi tersebut sehingga rawan *race condition*.

3. **Ketiadaan Pemisahan Role (CS vs Finance) & Ketiadaan Audit Trail Terpusat (Tinggi)**:  
   Model autentikasi `admin_users` dan token JWT saat ini hanya membedakan Super Admin (`brand_id = nil`) dan Brand Admin (`brand_id != nil`). Tidak ada pemisahan peran struktural antara CS/Staf Operasional dan Finance. Staf manapun yang memiliki akses dashboard brand dapat sekaligus menginput pembayaran dan menyetujui (*verify*) pembayaran sendiri. Tidak ada tabel log audit terpusat (`audit_logs`) untuk melacak perubahan data, nilai *before/after*, maupun riwayat tindakan administratif.

4. **Validasi Scoping Multi-Tenant yang Terlewat pada Super Admin & Bank Account (Sedang)**:  
   - Pada endpoint `POST /api/admin/bookings`, ketika dieksekusi oleh Super Admin (`brand_id = nil`), sistem tidak memvalidasi apakah `jamaah.brand_id` sama dengan `schedules.brand_id`, sehingga memungkinkan Jamaah Brand A didaftarkan ke Jadwal Paket Brand B.
   - Pada `POST /api/admin/bookings/{id}/payments`, sistem tidak memverifikasi apakah `bank_account_id` yang dipilih cocok dengan `brand_id` dari booking terkait (sedangkan endpoint portal jamaah sudah memvalidasinya).

5. **Duplikasi & Divergensi Kode Frontend pada Halaman Konfirmasi Pembayaran (Sedang)**:  
   Modul Jamaah dan Booking telah terpusat rapi di package `frontend/shared`, namun `PaymentConfirmationsPage.jsx` diduplikasi terpisah di `frontend/master-dashboard` dan `frontend/travel-dashboard` dengan implementasi yang mulai divergen (pemanggilan API berbeda, format state filter berbeda, dan penanganan alasan penolakan berbeda).

---

## 2. Alur Jamaah — Temuan Detail

### 2.1 Struktur Tabel `jamaah`
- **Kolom & Tipe**: `id` (BIGINT PK AI), `brand_id` (BIGINT FK -> brands.id), `kode_jamaah` (VARCHAR(10) UNIQUE), `id_jamaah` (VARCHAR(20) UNIQUE), `nama_lengkap` (VARCHAR(150)), `no_paspor` (VARCHAR(30)), `nik` (VARCHAR(20)), `no_hp` (VARCHAR(25)), `tempat_lahir` (VARCHAR(100)), `tanggal_lahir` (DATE), `jenis_kelamin` (ENUM 'L','P'), `status` (ENUM 'aktif','nonaktif'), `created_at`, `updated_at`.
- **Indeks & Constraint**:
  - `UNIQUE KEY unique_id_jamaah (id_jamaah)` (dari migrasi 025).
  - `UNIQUE KEY unique_kode_jamaah (kode_jamaah)` (dari migrasi 013).
  - `INDEX idx_jamaah_brand_id (brand_id)`.

### 2.2 Format & Logika Generate ID Jamaah
- **Format**: `{KodeBrand}-{YYMM}{NoUrut6digit}` (contoh: `AS-2608000001` untuk brand Al-Shafa pada Agustus 2026).
- **Analisis Race Condition**:
  - Diimplementasikan di `internal/jamaah/repository.go:60-95` (`GenerateIDJamaah`).
  - Menggunakan transaksi `tx.BeginTx` dan query penguncian baris `SELECT kode_brand, jamaah_counter FROM brands WHERE id = ? FOR UPDATE`.
  - Melakukan increment `jamaah_counter = counter + 1` lalu menyimpan update ke `brands` di dalam transaksi yang sama.
  - **Kesimpulan**: Alur ini **aman dari race condition / collision** antar request pendaftaran bersamaan untuk brand yang sama karena baris `brands` terkunci secara serial (*row-level lock*).
  - **Catatan Desain**: Counter bersifat sekuensial global per brand dan tidak di-reset per bulan/tahun (ini sesuai kebutuhan agar nomor urut tidak bertabrakan ketika tanggal berganti).

### 2.3 Manajemen Dokumen Jamaah (7 Jenis)
- **Tabel**: `dokumen_jamaah` dengan constraint `UNIQUE KEY unique_jamaah_jenis (jamaah_id, jenis)`.
- **7 Jenis Dokumen Valid**: `pas_foto`, `paspor`, `ktp`, `kk`, `buku_nikah`, `akte_lahir`, `vaksin_meningitis`.
- **Penyimpanan & Upload File**:
  - Dikelola via `internal/media/handler.go:49-130`.
  - Membatasi ukuran maksimal 25MB, memverifikasi magic bytes (`http.DetectContentType`) untuk mime-type gambar dan PDF.
  - Disimpan ke disk lokal `./uploads/{category}/{uuid}.{ext}`.
- **Temuan Kelemahan & Bug**:
  1. *Akses Terbuka Tanpa Autentikasi*: Endpoint `/uploads/*` di `cmd/api/main.go:151` melayani file secara publik tanpa enkripsi at-rest dan tanpa token URL sementara.
  2. *Bug Validasi Dokumen Portal*: Di `internal/portal/handler.go:426-433`, daftar `validJenis` tidak menyertakan `akte_lahir`, sehingga jamaah yang mengunggah Akte Kelahiran melalui portal akan mendapatkan error HTTP 400 `"jenis dokumen tidak valid"`.

### 2.4 Multi-Tenant Scoping Modul Jamaah
- Ekstraksi klaim JWT `brand_id` via `identity.GetBrandID(r.Context())`.
- `ListJamaah`, `GetJamaah`, `CreateJamaah`, `UpdateJamaah`, dan `DeleteJamaah` memverifikasi `brand_id` secara konsisten jika request berasal dari Brand Admin. Jika Super Admin (`brand_id == nil`), scope terbuka ke seluruh brand.

### 2.5 Paritas Frontend (Master vs Travel Dashboard)
- Kedua dashboard menggunakan halaman dari `frontend/shared`:
  - `JamaahPage` (`showBrandColumn={true}` di Master, `showBrandColumn={false}` di Travel).
  - `JamaahDetailPage` & `JamaahFormPage`.
- Paritas UI dan alur CRUD jamaah 100% konsisten antar dashboard.

---

## 3. Alur Booking — Temuan Detail

### 3.1 Struktur Tabel `bookings` & `booking_addons`
- **Tabel `bookings`**: `id`, `id_booking` (VARCHAR(10) UNIQUE), `schedule_id`, `jamaah_id`, `room_type`, `harga_dasar`, `status` (ENUM 'baru','dp','lunas','batal'), `is_seat_blocked` (BOOLEAN DEFAULT FALSE), `total_harga`, `diskon`, `diskon_keterangan`, `progress_visa`, `progress_hotel`, `progress_land_arrangement`, `progress_manasik`, `progress_siskopatuh`, `progress_vaksin_meningitis`, `perlengkapan_status`, `perlengkapan_tanggal`, `perlengkapan_diberikan_oleh`, `created_by`, `created_at`.
- **Tabel `booking_addons`**: `id`, `booking_id`, `nama`, `nominal`, `created_at`.
- **Pemisahan Konsep Add-On**:
  - `add_ons` + `schedule_add_ons`: Master fasilitas paket umroh (misal "Free Al-Baik 1x", "Kereta Cepat Haramain") **tanpa harga**.
  - `booking_addons`: Biaya tambahan transaksi per booking (misal "Upgrade Kamar Double", "Handling Tambahan") **dengan nominal harga**.
  - Backend, skema database, dan UI memisahkan kedua konsep ini dengan tepat dan tidak saling tertukar.

### 3.2 Format & Logika Generate ID Booking
- **Format**: `{KodeBrand 2 karakter}{4 karakter acak}` (contoh: `HN5PLZ` dari charset `23456789ABCDEFGHJKLMNPQRSTUVWXYZ`).
- **Penanganan Kolisi**: Di `internal/booking/repository.go:162-214` (`GenerateIDBooking`), terdapat perulangan hingga 10 percobaan (`for attempt := 0; attempt < 10; attempt++`) dengan query `SELECT COUNT(*) FROM bookings WHERE id_booking = ?`. Jika setelah 10 kali masih bertabrakan, mengembalikan `ErrGenerateIDBookingFailed`.
- **Constraint DB**: Terdapat `UNIQUE KEY unique_id_booking (id_booking)` (migrasi 031).

### 3.3 Progress Keberangkatan (8-Item Checklist)
1. `progress_paspor`: Dihitung dinamis melalui `checkPasporUploaded` yang memeriksa tabel `dokumen_jamaah` (`jenis='paspor'` dan `file_url IS NOT NULL`). Nilai di kolom DB `bookings.progress_paspor` bersifat vestigial.
2. `progress_tiket`: Dihitung dinamis dari `schedules.is_ticket_confirmed`. Nilai di kolom DB `bookings.progress_tiket` bersifat vestigial. Field baru `kode_penerbangan` pada schedule tidak meng-override logika ini.
3. 6 Item Toggle Manual (`visa`, `hotel`, `land_arrangement`, `manasik`, `siskopatuh`, `vaksin_meningitis`): Disimpan di kolom boolean `bookings` dan diubah via `PUT /api/admin/bookings/{id}/progress`.
4. `siap_berangkat`: Dihitung dinamis di memory/response struct sebagai konjungsi boolean (`AND`) dari ke-8 item.
5. `perlengkapan`: Dikelola secara terpisah melalui alur inventori stok (`perlengkapan_status`, `perlengkapan_tanggal`, `perlengkapan_diberikan_oleh`) dengan pemotongan stok otomatis dari `perlengkapan_stok` berdasarkan `perlengkapan_set_template` via `SELECT ... FOR UPDATE`.

### 3.4 Mekanisme `is_seat_blocked` & Kuota Kursi
- **Independensi Blok Kursi**: Flag `is_seat_blocked` mencatat apakah kursi fisik pada jadwal telah dikurangi.
- **Pelepasan Kursi Terpisah**: Endpoint `DELETE /api/admin/bookings/{id}/seat-block` (`CancelSeatBlock`) dapat melepaskan blok kursi (`seat_sisa + 1` dan `is_seat_blocked = false`) tanpa harus mengubah status booking menjadi `batal`.
- **Penguncian Baris (*Row Lock*)**:
  - `UpdateBookingStatus` dan `CancelSeatBlock` menggunakan `SELECT ... FOR UPDATE` untuk mencegah *race condition* saat mengurangi/menambah `seat_sisa`.
- **Celah / Kelemahan State Machine**:
  - `UpdateBookingStatus` tidak memvalidasi transisi state terbalik (misal `lunas` -> `baru`). Jika diubah menjadi `baru`, status berubah namun `is_seat_blocked` tetap `true` dan kursi tidak dilepas.

---

## 4. Alur Pembayaran — Temuan Detail

### 4.1 Struktur Tabel `payments`
- **Kolom**: `id`, `booking_id`, `bank_account_id`, `destination_bank_name`, `destination_account_number`, `destination_account_holder`, `jumlah`, `metode`, `sender_name`, `sender_bank`, `tanggal`, `status` (ENUM 'pending','confirmed','rejected'), `bukti_url`, `notes`, `source` (ENUM 'admin','portal'), `rejection_reason`, `verified_by`, `verified_at`, `created_at`.
- **Snapshot Rekening Tujuan**: Saat pembayaran dibuat dari rekening aktif (`bank_accounts`), data bank, no rekening, dan atas nama langsung di-*snapshot* ke tabel `payments`.

### 4.2 Alur Verifikasi & Sinkronisasi Status Booking
- **Logika Otomatis Sinkronisasi**:
  - Diimplementasikan di `internal/payment/repository.go:253-305` (`syncBookingStatusTx`).
  - Jika total pembayaran terkonfirmasi (`SUM(jumlah)`) >= `total_harga` -> status booking otomatis menjadi `lunas`.
  - Jika total pembayaran terkonfirmasi > 0 dan < `total_harga` -> status booking otomatis menjadi `dp`.
- **Temuan Kritis pada `syncBookingStatusTx`**:
  1. `schedules` tidak di-lock dengan `FOR UPDATE` saat mengeksekusi `UPDATE schedules SET seat_sisa = GREATEST(0, seat_sisa - 1) WHERE id=?`.
  2. Saat status booking berubah dari `baru` ke `dp`/`lunas`, kolom `is_seat_blocked` pada tabel `bookings` **TIDAK DI-UPDATE** menjadi `TRUE`. Hal ini menyebabkan inkonsistensi data: kursi fisik di jadwal sudah berkurang, namun booking berstatus `is_seat_blocked = false`.

### 4.3 Pemisahan Role & Audit Trail
- **Role Permission**: Belum ada role granular di backend. Admin travel manapun dapat menginput pembayaran sekaligus melakukan verifikasi/konfirmasi sendiri.
- **Audit Logging**: Tidak ada tabel log audit terpisah. Jejak verifikasi hanya dicatat pada kolom `payments.verified_by` dan `payments.verified_at`.

### 4.4 Konsistensi Nilai Pembayaran vs Tagihan
- Pada saat pembuatan pembayaran (`payment.Create` dan `portal.CreatePayment`), sistem memvalidasi `req.Jumlah <= sisaTagihan` (`total_harga - totalPaid`).
- Pembayaran berlebih (*overpayment*) ditolak dengan pesan error yang jelas.

---

## 5. Temuan Lintas Modul & Keamanan / Scoping

1. **Cross-Tenant Booking Assignment (Super Admin Scope)**:  
   Di `internal/booking/handler.go:85-153`, ketika Super Admin membuat booking baru, handler memvalidasi keberadaan `schedule_id` dan `jamaah_id`, namun tidak memverifikasi bahwa `jamaah.brand_id == schedule.brand_id`.

2. **Cross-Tenant Bank Account pada Input Pembayaran Admin**:  
   Di `internal/payment/repository.go:166-175`, `req.BankAccountID` hanya dicek `WHERE id=? AND is_active=TRUE` tanpa mencocokkan `brand_id` rekening dengan `brand_id` booking jadwal terkait.

3. **Kebocoran Pesan Error Internal (*Raw Error Leaks*)**:  
   Beberapa handler masih menggunakan format string `%v` yang mengekspos detail error internal Go / database driver ke respon HTTP client:
   - `internal/jamaah/handler.go:181`
   - `internal/dokumen/handler.go:164`
   - `internal/booking/handler.go:451`
   - `internal/payment/handler.go:225`

4. **Duplikasi & Inkonsistensi Frontend `PaymentConfirmationsPage.jsx`**:  
   - Lokasi 1: `frontend/master-dashboard/src/pages/PaymentConfirmationsPage.jsx`
   - Lokasi 2: `frontend/travel-dashboard/src/pages/PaymentConfirmationsPage.jsx`
   - Keduanya memiliki perbedaan implementasi import API (`../api/payments` vs `shared`), struktur state filter, dan format metadata badge status.

---

## 6. Daftar Risiko (Diurutkan dari Kritis → Rendah)

| No | Tingkat Risiko | Deskripsi Masalah | Lokasi File & Baris | Dampak |
|---|---|---|---|---|
| 1 | **KRITIS** | Berkas dokumen sensitif (KTP, Paspor, KK) dapat diakses secara publik tanpa token auth atau URL bertanda tangan (*unauthenticated file server*). | `cmd/api/main.go:150-151`<br>`internal/media/handler.go:95-123` | Kebocoran data pribadi (PII) jamaah jika URL file diketahui atau di-*enumerate*. |
| 2 | **KRITIS** | Verifikasi pembayaran (`syncBookingStatusTx`) mengurangi kuota kursi jadwal tanpa mengubah flag `bookings.is_seat_blocked = TRUE` dan tanpa row-level lock `FOR UPDATE` pada `schedules`. | `internal/payment/repository.go:253-305` | Kuota kursi jadwal hilang permanen saat booking dibatalkan di kemudian hari; rawan *race condition* kuota kursi. |
| 3 | **TINGGI** | Ketiadaan pemisahan hak akses / role (CS vs Finance) pada backend token JWT dan handler pembayaran. | `internal/identity/jwt.go:35-48`<br>`internal/payment/handler.go:145-188` | Staf operasional/CS dapat mengonfirmasi pembayaran sendiri tanpa *maker-checker control*. |
| 4 | **TINGGI** | Tidak ada tabel audit trail terpusat untuk mencatat riwayat perubahan status booking, penghapusan add-on, dan pengubahan diskon. | `internal/booking/repository.go:462-504`<br>`internal/payment/repository.go:220-251` | Kesulitan rekonsiliasi dan investigasi *fraud* atau kesalahan operasional finansial. |
| 5 | **SEDANG** | Super Admin dapat membuat booking dengan memasangkan Jamaah Brand X ke Jadwal Paket Brand Y (*cross-brand mismatch*). | `internal/booking/handler.go:101-127` | Integritas data booking multi-tenant tercampur. |
| 6 | **SEDANG** | Admin dapat memilih `bank_account_id` milik brand lain saat menginput pembayaran manual. | `internal/payment/repository.go:166-175` | Data transfer pembayaran salah tercatat ke rekening bank milik brand berbeda. |
| 7 | **SEDANG** | Jenis dokumen `akte_lahir` tidak terdaftar pada whitelist `validJenis` di portal jamaah. | `internal/portal/handler.go:426-433` | Jamaah gagal mengunggah dokumen Akte Kelahiran melalui portal jamaah. |
| 8 | **SEDANG** | Duplikasi dan divergensi kode `PaymentConfirmationsPage.jsx` antara Master Dashboard dan Travel Dashboard. | `frontend/master-dashboard/src/pages/PaymentConfirmationsPage.jsx`<br>`frontend/travel-dashboard/src/pages/PaymentConfirmationsPage.jsx` | Perubahan fitur atau perbaikan bug pada salah satu dashboard tidak otomatis tercermin di dashboard lain. |
| 9 | **RENDAH** | Ketiadaan validasi transisi state machine pada status booking (misal transisi `lunas` -> `baru`). | `internal/booking/repository.go:271-344` | Data status booking dan status kursi bisa menjadi anomali jika diubah secara tidak wajar. |
| 10 | **RENDAH** | Kebocoran string error internal driver SQL melalui format `%v` pada respon HTTP status 500. | `internal/jamaah/handler.go:181`<br>`internal/dokumen/handler.go:164`<br>`internal/booking/handler.go:451`<br>`internal/payment/handler.go:225` | Informasi teknis internal database terekspos ke klien saat terjadi kegagalan sistem. |

---

## 7. Hal yang SUDAH Benar / Sesuai Desain

1. **Keamanan Generate ID Jamaah**:  
   Penggunaan transaksi database dengan penguncian baris `SELECT ... FOR UPDATE` pada tabel `brands` di `internal/jamaah/repository.go:60-95` menjamin nomor urut ID Jamaah tidak akan mengalami tabrakan (*race-condition safe*).
2. **Pemisahan Konsep Add-On Paket vs Booking Add-On**:  
   Pemisahan tabel `add_ons` (fasilitas paket tanpa nominal) dan `booking_addons` (biaya tambahan per transaksi) diimplementasikan secara konsisten dan terisolasi dengan formula kalkulasi `total_harga = GREATEST(0, harga_dasar + SUM(addons) - diskon)`.
3. **Kalkulasi Dinamis Checklist Siap Berangkat**:  
   Status paspor dan tiket dihitung secara dinamis dari data master riil (`dokumen_jamaah` dan `schedules.is_ticket_confirmed`), mencegah desinkronisasi status dokumen fisik.
4. **Alur Penguncian & Pelepasan Kursi Mandiri**:  
   Fungsi `CancelSeatBlock` (`DELETE /api/admin/bookings/{id}/seat-block`) memungkinkan pelepasan alokasi kursi tanpa membatalkan transaksi booking, dengan penguncian baris `FOR UPDATE`.
5. **Manajemen Stok Perlengkapan Terintegrasi**:  
   Distribusi perlengkapan (`MarkPerlengkapanDiberikan` dan `BatalkanPerlengkapan`) terikat pada template brand dan diverifikasi dengan stok riil secara atomik menggunakan database transaction dan row lock.
6. **Proteksi Overpayment & Rate Limiter Portal**:  
   Sistem secara ketat menolak pembayaran manual yang melebihi sisa tagihan booking, dan portal jamaah dilengkapi dengan proteksi *rate limiter* percobaan login gagal per IP.
7. **Pemanfaatan Package Shared Frontend**:  
   Komponen dan halaman inti Jamaah (`JamaahPage`, `JamaahDetailPage`, `JamaahFormPage`) serta Booking (`BookingsPage`, `BookingDetailPage`, `BookingFormPage`) telah diintegrasikan secara terpusat melalui package `frontend/shared` dengan konfigurasi multi-tenant yang rapi.
