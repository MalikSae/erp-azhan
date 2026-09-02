# Laporan Investigasi Komprehensif: Arsitektur & Skema Portal Jamaah (Read-Only)

> **Dokumen Investigasi Teknis**  
> **Target Analisis:** Kesiapan arsitektur untuk multi-akun aktif per booking pada Portal Jamaah.  
> **Status Pemeriksaan:** Read-Only (100% fakta dari live database MySQL `erp_azhan_dev` dan codebase `erp-azhan`).

---

## 1. Ringkasan Eksekutif

1. **Skema Database Perorangan Sudah Siap:** Kolom autentikasi `portal_pin_hash` melekat pada tabel perorangan `jamaah` (bukan di tabel `bookings`), dan relasi booking-jamaah murni menggunakan tabel perantara `booking_pax`. Secara skema database, setiap individu jamaah sudah memiliki entitas yang independen dan siap menampung PIN masing-masing.
2. **59% Data Riil Adalah Multi-Pax:** Dari 39 booking di database aktif saat ini, **23 booking (59.0%) memiliki lebih dari 1 pax** (20 booking dengan 2 pax, 3 booking dengan 3 pax), namun saat ini 100% pax anggota (#2, #3) memiliki `portal_pin_hash = NULL`.
3. **Endpoint List Bookings Sudah Siap Multi-Pax:** Query pada `GET /api/portal/bookings` (`internal/booking/repository.go:128`) telah menggunakan filter `b.pic_jamaah_id = ? OR EXISTS (SELECT 1 FROM booking_pax bp2 WHERE bp2.booking_id = b.id AND bp2.jamaah_id = ?)`. Jika pax #2 login, daftar booking akan otomatis muncul.
4. **Penghalang Utama Ada di Level Gate Otorisasi Handler Go (Hardcoded PIC Check):** Tiga handler utama di `internal/portal/handler.go` (`GetBookingByID:266`, `ListPayments:296`, `CreatePayment:359`) melakukan validasi hardcoded `*b.JamaahID != jamaahID` (di mana `b.JamaahID` adalah alias dari `bookings.pic_jamaah_id`). Akibatnya, jika pax #2 login dan membuka detail booking/pembayaran, server langsung mengembalikan `404 Not Found`.
5. **Belum Ada Alur Pembuatan/Aktivasi PIN Non-PIC & Admin Tooling:** Payload self-service booking saat ini hanya menyediakan field PIN untuk PIC (`req.PIC.PortalPIN`). Di seluruh backend dan admin dashboard, belum ada endpoint ataupun antarmuka untuk aktivasi PIN mandiri bagi pax anggota maupun reset PIN oleh admin travel.

---

## 2. Skema Database Portal (BAGIAN 1)

Hasil eksekusi riil `SHOW CREATE TABLE` dan `SHOW TABLES` pada database aktif `erp_azhan_dev`:

### 2.1 `SHOW CREATE TABLE jamaah`
```sql
CREATE TABLE `jamaah` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `brand_id` bigint unsigned NOT NULL,
  `nama_lengkap` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `nama_ayah_kandung` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `jenis_kelamin` enum('L','P') COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `nik` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `tempat_lahir` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `tanggal_lahir` date DEFAULT NULL,
  `no_paspor` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `tempat_paspor_keluar` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `tanggal_paspor_keluar` date DEFAULT NULL,
  `paspor_berlaku_sampai` date DEFAULT NULL,
  `no_hp` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `email` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `pekerjaan` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `pendidikan_terakhir` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `penjamin_kesehatan` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `no_asuransi_bpjs` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `alamat` text COLLATE utf8mb4_unicode_ci,
  `kota` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `emergency_nama` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `emergency_nik` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `emergency_hp` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `emergency_hubungan` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `emergency_alamat` text COLLATE utf8mb4_unicode_ci,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `kode_jamaah` varchar(6) COLLATE utf8mb4_unicode_ci NOT NULL,
  `id_jamaah` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  `catatan` text COLLATE utf8mb4_unicode_ci,
  `status` enum('draft','aktif') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'aktif',
  `portal_pin_hash` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_kode_jamaah` (`kode_jamaah`),
  UNIQUE KEY `unique_id_jamaah` (`id_jamaah`),
  UNIQUE KEY `nik` (`nik`),
  UNIQUE KEY `uq_jamaah_brand_phone` (`brand_id`,`no_hp`),
  CONSTRAINT `jamaah_ibfk_1` FOREIGN KEY (`brand_id`) REFERENCES `brands` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=97 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
```

### 2.2 `SHOW CREATE TABLE bookings`
```sql
CREATE TABLE `bookings` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `id_booking` varchar(6) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `schedule_id` bigint unsigned NOT NULL,
  `pic_jamaah_id` bigint unsigned DEFAULT NULL,
  `status` enum('draft','baru','dp','lunas','batal') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'baru',
  `is_seat_blocked` tinyint(1) NOT NULL DEFAULT '0',
  `seat_hold_expires_at` datetime DEFAULT NULL,
  `seat_hold_key` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `total_harga` decimal(12,0) DEFAULT NULL,
  `created_by` bigint unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `progress_hotel` tinyint(1) NOT NULL DEFAULT '0',
  `progress_land_arrangement` tinyint(1) NOT NULL DEFAULT '0',
  `seat_count` smallint unsigned NOT NULL DEFAULT '1',
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_id_booking` (`id_booking`),
  KEY `schedule_id` (`schedule_id`),
  KEY `created_by` (`created_by`),
  KEY `fk_bookings_pic_jamaah` (`pic_jamaah_id`),
  CONSTRAINT `bookings_ibfk_1` FOREIGN KEY (`schedule_id`) REFERENCES `schedules` (`id`),
  CONSTRAINT `bookings_ibfk_3` FOREIGN KEY (`created_by`) REFERENCES `admin_users` (`id`),
  CONSTRAINT `fk_bookings_pic_jamaah` FOREIGN KEY (`pic_jamaah_id`) REFERENCES `jamaah` (`id`) ON DELETE RESTRICT
) ENGINE=InnoDB AUTO_INCREMENT=111 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
```

### 2.3 `SHOW CREATE TABLE booking_pax`
```sql
CREATE TABLE `booking_pax` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `booking_id` bigint unsigned NOT NULL,
  `jamaah_id` bigint unsigned NOT NULL,
  `pax_type` enum('reguler','infant') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'reguler',
  `room_type` enum('Quad','Triple','Double') COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `harga_pax` decimal(12,0) NOT NULL DEFAULT '0',
  `counts_for_seat` tinyint(1) NOT NULL DEFAULT '1',
  `pax_status` enum('aktif','batal') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'aktif',
  `progress_visa` tinyint(1) NOT NULL DEFAULT '0',
  `progress_siskopatuh` tinyint(1) NOT NULL DEFAULT '0',
  `progress_manasik` tinyint(1) NOT NULL DEFAULT '0',
  `progress_vaksin_meningitis` tinyint(1) NOT NULL DEFAULT '0',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `perlengkapan_status` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'belum_diberikan' COMMENT 'belum_diberikan, sudah_diberikan',
  `perlengkapan_tanggal` date DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_booking_jamaah` (`booking_id`,`jamaah_id`),
  KEY `jamaah_id` (`jamaah_id`),
  CONSTRAINT `booking_pax_ibfk_1` FOREIGN KEY (`booking_id`) REFERENCES `bookings` (`id`) ON DELETE CASCADE,
  CONSTRAINT `booking_pax_ibfk_2` FOREIGN KEY (`jamaah_id`) REFERENCES `jamaah` (`id`) ON DELETE RESTRICT
) ENGINE=InnoDB AUTO_INCREMENT=172 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
```

### 2.4 `SHOW CREATE TABLE dokumen_jamaah`
```sql
CREATE TABLE `dokumen_jamaah` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `jamaah_id` bigint unsigned NOT NULL,
  `jenis` enum('pas_foto','paspor','ktp','kk','buku_nikah','akte_lahir','vaksin_meningitis') COLLATE utf8mb4_unicode_ci NOT NULL,
  `file_url` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` enum('belum_upload','submitted','approved','rejected') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'belum_upload',
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_jamaah_jenis` (`jamaah_id`,`jenis`),
  CONSTRAINT `dokumen_jamaah_ibfk_1` FOREIGN KEY (`jamaah_id`) REFERENCES `jamaah` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=8 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
```

### 2.5 `SHOW CREATE TABLE payments`
```sql
CREATE TABLE `payments` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `booking_id` bigint unsigned NOT NULL,
  `bank_account_id` bigint unsigned DEFAULT NULL,
  `destination_bank_name` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `destination_account_number` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `destination_account_holder` varchar(150) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `jumlah` decimal(12,0) NOT NULL,
  `metode` varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `sender_name` varchar(150) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `sender_bank` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `tanggal` date DEFAULT NULL,
  `status` enum('pending','confirmed','rejected') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
  `bukti_url` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `notes` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `source` enum('admin','portal','crm') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'admin',
  `rejection_reason` varchar(500) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `verified_by` bigint unsigned DEFAULT NULL,
  `verified_at` datetime DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `booking_id` (`booking_id`),
  KEY `fk_payment_bank_account` (`bank_account_id`),
  KEY `fk_payment_verified_by` (`verified_by`),
  CONSTRAINT `fk_payment_bank_account` FOREIGN KEY (`bank_account_id`) REFERENCES `bank_accounts` (`id`) ON DELETE SET NULL,
  CONSTRAINT `fk_payment_verified_by` FOREIGN KEY (`verified_by`) REFERENCES `admin_users` (`id`) ON DELETE SET NULL,
  CONSTRAINT `payments_ibfk_1` FOREIGN KEY (`booking_id`) REFERENCES `bookings` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=24 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
```

### 2.6 `SHOW TABLES` (27 Tabel Lengkap)
1. `add_ons`
2. `admin_users`
3. `airlines`
4. `airports`
5. `bank_accounts`
6. `booking_addons`
7. `booking_discounts`
8. `booking_pax`
9. `booking_pax_perlengkapan_logs`
10. `bookings`
11. `brand_domains`
12. `brands`
13. `category_brands`
14. `crm_deal_requests`
15. `dokumen_jamaah`
16. `hotels`
17. `itineraries`
18. `itinerary_days`
19. `jamaah`
20. `jamaah_relasi`
21. `package_categories`
22. `payments`
23. `perlengkapan_items`
24. `perlengkapan_set_template`
25. `perlengkapan_stok`
26. `schedule_add_ons`
27. `schedule_transit_hotels`
28. `schedules`

> **Temuan Tabel Tambahan:** Tidak ada tabel khusus autentikasi portal (seperti `portal_users`, `portal_sessions`, atau `portal_tokens`). Autentikasi portal terintegrasi langsung pada tabel `jamaah`. Ada tabel `jamaah_relasi` (untuk hubungan mahram/keluarga di admin), namun tidak digunakan dalam autentikasi portal saat ini.

---

### 2.7 Detail Analisis Skema Tabel `jamaah`
- **Kolom Autentikasi Portal:**
  - `id_jamaah VARCHAR(20) NOT NULL`: Berfungsi sebagai *username / Public Identifier* (contoh format: `HN-2609000034`).
  - `brand_id BIGINT UNSIGNED NOT NULL`: Penentu scope travel/brand pemilik akun.
  - `portal_pin_hash VARCHAR(100) NULL`: Hash PIN 6 digit.
  - `status ENUM('draft','aktif') NOT NULL DEFAULT 'aktif'`: Status operasional jamaah (belum dievaluasi dalam query login).
  - *Catatan:* **Tidak ada** kolom `portal_active` (boolean flag), `portal_last_login` (timestamp), `portal_failed_attempts` (counter), maupun `portal_activated_at`.
- **Format Penyimpanan PIN (Hash vs Plaintext):**
  - **PIN disimpan sebagai Bcrypt Hash** (string 60 karakter dengan prefix `$2a$10$...`).
  - *Bukti Kode Penulisan:* `internal/selfbooking/repository.go:98`: `pinHash, err := bcrypt.GenerateFromPassword([]byte(req.PIC.PortalPIN), bcrypt.DefaultCost)`.
  - *Bukti Kode Verifikasi:* `internal/portal/handler.go:183`: `bcrypt.CompareHashAndPassword([]byte(pinHash.String), []byte(req.PortalPIN))`.
  - *Bukti Sampel Database Riil:*
    - ID 95: `$2a$10$...qCXC` (panjang 60 karakter)
    - ID 92: `$2a$10$...niqK` (panjang 60 karakter)
    - ID 89: `$2a$10$...Jqeu` (panjang 60 karakter)
- **Batasan Unik (Unique Constraints):**
  - `UNIQUE KEY unique_id_jamaah (id_jamaah)`: **GLOBAL** lintas brand.
  - `UNIQUE KEY unique_kode_jamaah (kode_jamaah)`: **GLOBAL** lintas brand.
  - `UNIQUE KEY uq_jamaah_brand_phone (brand_id, no_hp)`: **PER BRAND** (nomor HP yang sama boleh dipakai di brand berbeda).
  - `UNIQUE KEY nik (nik)`: **GLOBAL** (nullable).
- **Relasi Booking ↔ Jamaah:**
  - Kolom lama `bookings.jamaah_id` **SUDAH TIDAK ADA** di skema fisik tabel `bookings` (`old_jamaah_id_col_exists = false`).
  - Tabel `bookings` hanya memiliki kolom `pic_jamaah_id BIGINT UNSIGNED DEFAULT NULL` (terisi pada seluruh 39 baris booking aktif).
  - Relasi seluruh jamaah ke booking disimpan murni di tabel perantara `booking_pax` (`booking_id`, `jamaah_id`) dengan constraint `UNIQUE KEY uq_booking_jamaah (booking_id, jamaah_id)`.
- **Penanda PIC / Pemesan:**
  - Kolom penanda tunggal berada pada `bookings.pic_jamaah_id` (foreign key ke `jamaah.id`).
  - Pada tabel `booking_pax`, **tidak ada kolom `is_pic`**. Status PIC ditentukan secara komputasi runtime: `bp.jamaah_id == b.pic_jamaah_id`.
  - Kolom `pic_jamaah_id` dijamin **tepat 1 per booking** karena bertindak sebagai kolom skalar di tabel `bookings`.

---

## 3. Alur Pembuatan PIN (BAGIAN 2)

### 3.1 Endpoint & Handler Self-Booking
- **Endpoint:** `POST /api/public/booking` (atau `POST /api/selfbooking`)
- **Handler:** `selfbooking.Handler.CreateBooking` di [`internal/selfbooking/handler.go:44`](file:///c:/laragon/www/erp-azhan/internal/selfbooking/handler.go#L44)
- **Repository:** `selfbooking.Repository.CreateBooking` di [`internal/selfbooking/repository.go:28`](file:///c:/laragon/www/erp-azhan/internal/selfbooking/repository.go#L28)

### 3.2 Struktur Payload Request
Didefinisikan di [`internal/selfbooking/model.go:24-30`](file:///c:/laragon/www/erp-azhan/internal/selfbooking/model.go#L24-L30):
```go
type BookingRequest struct {
    BrandID      int64          `json:"brand_id"`
    ScheduleID   int64          `json:"schedule_id"`
    CaptchaToken string         `json:"captcha_token"`
    PIC          PICInput       `json:"pic"`
    Anggota      []AnggotaInput `json:"anggota"`
}
```
- **Jumlah Jamaah:** Mampu menerima 1 PIC + N Anggota (array `AnggotaInput`).
- **Penerima PIN:** PIN (`portal_pin`, 6 digit) dikirim **hanya di dalam objek `PIC`** (`PICInput.PortalPIN`). Objek `AnggotaInput` tidak memiliki field `portal_pin`.

### 3.3 Titik Penyimpanan PIN
Di [`internal/selfbooking/repository.go:98-105`](file:///c:/laragon/www/erp-azhan/internal/selfbooking/repository.go#L98-L105):
```go
// 5. Update PIN PIC
pinHash, err := bcrypt.GenerateFromPassword([]byte(req.PIC.PortalPIN), bcrypt.DefaultCost)
if err != nil {
    return nil, fmt.Errorf("hash pin: %w", err)
}
_, err = tx.ExecContext(ctx, `UPDATE jamaah SET portal_pin_hash=? WHERE id=?`, string(pinHash), picJamaahID)
```
Eksekusi UPDATE PIN secara eksplisit **hanya menargetkan `picJamaahID`**.

### 3.4 Kondisi Data Jamaah Lain (Pax #2, #3, dst.)
Di [`internal/selfbooking/repository.go:134-142`](file:///c:/laragon/www/erp-azhan/internal/selfbooking/repository.go#L134-L142):
```go
res, err := tx.ExecContext(ctx, `
    INSERT INTO jamaah (brand_id, id_jamaah, kode_jamaah, nama_lengkap, no_hp, jenis_kelamin)
    VALUES (?, ?, ?, ?, ?, ?)
`, brandID, idJamaah, kodeJamaah, a.NamaLengkap, a.NoHP, a.JenisKelamin)
```
- Baris jamaah anggota dibuat dengan kolom `portal_pin_hash = NULL` (karena kolom di-omit saat INSERT).
- Kolom `status` terisi default `'aktif'`.
- *Bukti Data Riil Database (Sample Booking Multi-Pax #109 / Kode `HNQ4YQ`):*
  - Pax #1 (PIC - Jamaah ID 92, `HN-2609000034`): `portal_pin_hash` = `$2a$10$...` (`has_pin: true`)
  - Pax #2 (Anggota - Jamaah ID 93, `HN-2609000035`): `portal_pin_hash` = `NULL` (`has_pin: false`)
  - Pax #3 (Infant - Jamaah ID 94, `HN-2609000036`): `portal_pin_hash` = `NULL` (`has_pin: false`)

### 3.5 Mutasi PIN di Tempat Lain
- **TIDAK ADA.** Berdasarkan pencarian menyeluruh kata kunci `portal_pin_hash` di seluruh file `.go` backend, tidak ada endpoint reset PIN oleh admin, tidak ada endpoint penggantian PIN mandiri oleh jamaah, dan tidak ada fungsi lain yang mengubah kolom tersebut.

---

## 4. Autentikasi & Otorisasi Portal (BAGIAN 3)

### 4.1 Endpoint Login Portal
- **Path:** `POST /api/portal/login` ([`internal/portal/handler.go:131`](file:///c:/laragon/www/erp-azhan/internal/portal/handler.go#L131))
- **Kredensial Diminta:**
  1. `brand_id` (int64, wajib)
  2. `id_jamaah` (string, wajib, case-insensitive)
  3. `portal_pin` (string, wajib)
- **Query SQL Pencocokan:**
  ```sql
  SELECT id, COALESCE(id_jamaah, ''), nama_lengkap, brand_id, portal_pin_hash 
  FROM jamaah 
  WHERE brand_id = ? AND UPPER(id_jamaah) = UPPER(?)
  ```
- **Rate Limit:**
  - In-memory rate limiter per IP client (`failedLogins map[string]*loginAttempt`).
  - Maksimal 5 kali kegagalan per IP dalam kurun waktu 15 menit ([`internal/portal/handler.go:57-72`](file:///c:/laragon/www/erp-azhan/internal/portal/handler.go#L57-L72)).
  - Jika limit terlewati, handler merespons `429 Too Many Requests: {"error": "terlalu banyak percobaan login yang gagal, coba lagi dalam 15 menit"}`.
- **Catatan Khusus PIN Kosong:**
  Di [`internal/portal/handler.go:173-180`](file:///c:/laragon/www/erp-azhan/internal/portal/handler.go#L173-L180), terdapat logika fallback untuk akun tanpa PIN:
  ```go
  if !pinHash.Valid || pinHash.String == "" {
      if req.PortalPIN != "123456" {
          h.recordFailedLogin(clientIP)
          writeError(w, http.StatusUnauthorized, "ID jamaah atau PIN tidak cocok")
          return
      }
  }
  ```
  *(Akun yang `portal_pin_hash`-nya NULL dapat login jika memasukkan PIN default `"123456"`)*.

### 4.2 Struktur JWT Token Portal
Dihasilkan oleh `identity.GeneratePortalToken(jamaahID)` di [`internal/identity/jwt.go:107-118`](file:///c:/laragon/www/erp-azhan/internal/identity/jwt.go#L107-L118):
- **Claims:**
  - `sub`: `jamaahID` (int64, ID database jamaah yang login)
  - `type`: `"portal"` (string)
  - `jti`: UUID v4 acak string unik
  - `exp`: `now + 24 Jam` (Unix timestamp)
  - `iat`: `now` (Unix timestamp)
- *Catatan:* Token portal **tidak memuat** claim `brand_id` maupun `role`.

### 4.3 Middleware `RequirePortalAuth`
Didefinisikan di [`internal/identity/middleware.go:76-94`](file:///c:/laragon/www/erp-azhan/internal/identity/middleware.go#L76-L94):
- Memeriksa header `Authorization: Bearer <token>`.
- Memvalidasi signature HMAC-SHA256 menggunakan `JWT_SECRET`.
- Memastikan token belum kedaluwarsa (`exp`) dan `type == "portal"`.
- Menyuntikkan ID jamaah ke context: `context.WithValue(r.Context(), PortalJamaahIDKey, jamaahID)`.

### 4.4 Daftar Rute Lengkap `/api/portal/*`
Terdaftar di [`cmd/api/main.go:191-207`](file:///c:/laragon/www/erp-azhan/cmd/api/main.go#L191-L207):
| Method | Endpoint Path | Middleware | Handler |
|---|---|---|---|
| `POST` | `/api/portal/login` | *Public* | `portalHandler.Login` |
| `GET` | `/api/portal/me` | `RequirePortalAuth` | `portalHandler.GetMe` |
| `GET` | `/api/portal/bookings` | `RequirePortalAuth` | `portalHandler.ListBookings` |
| `GET` | `/api/portal/bookings/{id}` | `RequirePortalAuth` | `portalHandler.GetBookingByID` |
| `GET` | `/api/portal/bookings/{id}/payments` | `RequirePortalAuth` | `portalHandler.ListPayments` |
| `GET` | `/api/portal/bank-accounts` | `RequirePortalAuth` | `portalHandler.ListBankAccounts` |
| `POST` | `/api/portal/bookings/{id}/payments` | `RequirePortalAuth` | `portalHandler.CreatePayment` |
| `GET` | `/api/portal/dokumen` | `RequirePortalAuth` | `portalHandler.ListDokumen` |
| `POST` | `/api/portal/dokumen` | `RequirePortalAuth` | `portalHandler.UploadDokumen` |
| `POST` | `/api/portal/media/upload` | `RequirePortalAuth` | `mediaHandler.UploadPortalMedia` |

---

## 5. Matriks Kepemilikan Data per Endpoint (BAGIAN 4)

| Endpoint | Jalur Penentuan Kepemilikan Data | Apakah Pax #2 Otomatis Dapat Data Ini Jika Diberi PIN? | Referensi File:Baris |
|---|---|---|---|
| **`GET /api/portal/me`** | Berdasarkan ID Jamaah dari token (`j.id = jamaahID`). | **YA** (Mengembalikan profil pribadi Pax #2). | [`internal/portal/handler.go:214`](file:///c:/laragon/www/erp-azhan/internal/portal/handler.go#L214) |
| **`GET /api/portal/bookings`** | Query SQL: `b.pic_jamaah_id = ? OR EXISTS (SELECT 1 FROM booking_pax bp2 WHERE bp2.booking_id = b.id AND bp2.jamaah_id = ?)`. | **YA** (Booking muncul di list karena Pax #2 terdaftar di `booking_pax`). | [`internal/booking/repository.go:128`](file:///c:/laragon/www/erp-azhan/internal/booking/repository.go#L128) |
| **`GET /api/portal/bookings/{id}`** | Pengecekan di Go handler: `if b.JamaahID == nil \|\| *b.JamaahID != jamaahID` (di mana `b.JamaahID` bernilai `bookings.pic_jamaah_id`). | **TIDAK (Blocker / 404 Not Found)**. Gate memeriksa apakah user adalah PIC. Pax #2 ditolak. | [`internal/portal/handler.go:266`](file:///c:/laragon/www/erp-azhan/internal/portal/handler.go#L266) |
| **`GET /api/portal/bookings/{id}/payments`** | Pengecekan di Go handler: `if b.JamaahID == nil \|\| *b.JamaahID != jamaahID`. | **TIDAK (Blocker / 404 Not Found)**. Gate memeriksa apakah user adalah PIC. Pax #2 ditolak. | [`internal/portal/handler.go:296`](file:///c:/laragon/www/erp-azhan/internal/portal/handler.go#L296) |
| **`GET /api/portal/bank-accounts`** | Berdasarkan Brand ID milik jamaah yang login: `JOIN jamaah j ON j.brand_id = a.brand_id WHERE j.id = ?`. | **YA** (Mengembalikan rekening resmi milik brand). | [`internal/portal/handler.go:325`](file:///c:/laragon/www/erp-azhan/internal/portal/handler.go#L325) |
| **`POST /api/portal/bookings/{id}/payments`** | Pengecekan di Go handler: `if b.JamaahID == nil \|\| *b.JamaahID != jamaahID`. | **TIDAK (Blocker / 404 Not Found)**. Hanya PIC yang diizinkan mengunggah bukti bayar. | [`internal/portal/handler.go:359`](file:///c:/laragon/www/erp-azhan/internal/portal/handler.go#L359) |
| **`GET /api/portal/dokumen`** | Query SQL: `SELECT ... FROM dokumen_jamaah WHERE jamaah_id = ?`. | **YA** (Mengembalikan daftar dokumen pribadi milik Pax #2 sendiri). | [`internal/dokumen/repository.go:34`](file:///c:/laragon/www/erp-azhan/internal/dokumen/repository.go#L34) |
| **`POST /api/portal/dokumen`** | SQL: `INSERT INTO dokumen_jamaah (jamaah_id, ...) VALUES (?, ...)`. | **YA** (Pax #2 dapat mengunggah dokumen atas namanya sendiri). | [`internal/dokumen/repository.go:96`](file:///c:/laragon/www/erp-azhan/internal/dokumen/repository.go#L96) |
| **`POST /api/portal/media/upload`** | Mengunggah file gambar umum/bukti transfer, menghasilkan URL static. | **YA** (Berlaku untuk seluruh token portal yang valid). | [`internal/media/handler.go:88`](file:///c:/laragon/www/erp-azhan/internal/media/handler.go#L88) |

### 5.1 Rincian Analisis Data per Endpoint

#### A. Data Jamaah Lain yang Terkirim di `GET /api/portal/bookings/{id}`
Jika gate kepemilikan berhasil dilewati, detail booking mengembalikan array `b.Pax` via [`internal/booking/repository.go:227-260`](file:///c:/laragon/www/erp-azhan/internal/booking/repository.go#L227-L260):
- **Field yang terkirim:** `id`, `booking_id`, `jamaah_id`, `nama_jamaah`, `pax_type`, `room_type`, `harga_pax`, `counts_for_seat`, `pax_status`, `progress_visa`, `progress_siskopatuh`, `progress_manasik`, `progress_vaksin_meningitis`, `perlengkapan_status`, `perlengkapan_tanggal`.
- **Field yang TIDAK terkirim:** `nik`, `no_paspor`, `tanggal_lahir`, `no_hp`, `alamat` jamaah lain tidak ikut terkirim (aman secara privasi).

#### B. Isolasi Dokumen Jamaah
- `GET /api/portal/dokumen`: Disaring murni dengan `WHERE jamaah_id = ?` (ID dari token). Jamaah yang login **TIDAK BISA** melihat dokumen pax lain.
- `POST /api/portal/dokumen`: Disimpan murni untuk `jamaah_id` dari token. Jamaah **TIDAK BISA** mengunggah atas nama pax lain karena tidak ada parameter target jamaah ID.

#### C. Isolasi Pembayaran
- Tabel `payments` dilekatkan pada header booking (`payments.booking_id`).
- Pembayaran bersifat terpusat per rombongan booking (bukan split bill per pax).
- Siapa yang bisa melihat saat ini: **Hanya PIC**.

#### D. Progress Keberangkatan
- **Tingkat Booking (Header):** `progress_hotel`, `progress_land_arrangement`, `is_ticket_confirmed`.
- **Tingkat Pax (Individu):** `progress_paspor`, `progress_visa`, `progress_siskopatuh`, `progress_manasik`, `progress_vaksin_meningitis`, `perlengkapan_status`.
- Di level response header `b`, progress per-pax mengambil sampel dari pax pertama/PIC (`LIMIT 1`). Namun di dalam array `b.Pax`, masing-masing item memuat status progress perorangan setiap jamaah.

---

## 6. Sisi Admin (BAGIAN 5)

1. **Antarmuka & Endpoint Admin:**
   - **TIDAK ADA** antarmuka ataupun endpoint di Travel Dashboard maupun Master Dashboard untuk melihat status akun portal, membuat PIN, mereset PIN, atau memblokir akun portal seorang jamaah.
2. **Jamaah Hasil Input Admin Manual:**
   - Saat admin mendaftarkan jamaah via dashboard (`POST /api/admin/jamaah`), kolom `portal_pin_hash` tidak diisi (bernilai `NULL`).
   - *Statistik Data Riil Database (Per Brand):*
     - **Brand ID 2 (Altezza):** Total 3 jamaah — **0 dengan PIN (0%)**, 3 tanpa PIN (100%).
     - **Brand ID 3 (Hana Tours):** Total 24 jamaah — **10 dengan PIN (41.7%)**, 14 tanpa PIN (58.3%).

---

## 7. Kondisi Data Riil (BAGIAN 6)

### 7.1 Distribusi Pax per Booking (Total 39 Booking)
- **1 Pax:** 16 booking (41.0%)
- **2 Pax:** 20 booking (51.3%)
- **3 Pax:** 3 booking (7.7%)
- **Total Booking Multi-Pax:** **23 booking (59.0%)**

### 7.2 Sampel Riil Booking Multi-Pax
Cuplikan data dari tabel `bookings`, `booking_pax`, dan `jamaah`:
| Booking Code | Booking ID | BP ID | Jamaah ID | ID Jamaah | Nama | Status PIC | Pax Type | Memiliki PIN Portal |
|---|---|---|---|---|---|---|---|---|
| `HNPV5Z` | 110 | 170 | 95 | `HN-2609000037` | S*** | **YA (PIC)** | reguler | **YA** (`$2a$10$...`) |
| `HNPV5Z` | 110 | 171 | 96 | `HN-2609000038` | U*** | TIDAK | infant | **TIDAK** (`NULL`) |
| `HNQ4YQ` | 109 | 167 | 92 | `HN-2609000034` | H*** | **YA (PIC)** | reguler | **YA** (`$2a$10$...`) |
| `HNQ4YQ` | 109 | 168 | 93 | `HN-2609000035` | H*** | TIDAK | reguler | **TIDAK** (`NULL`) |
| `HNQ4YQ` | 109 | 169 | 94 | `HN-2609000036` | Y*** | TIDAK | infant | **TIDAK** (`NULL`) |
| `HNVERY` | 108 | 164 | 89 | `HN-2609000031` | I*** | **YA (PIC)** | reguler | **YA** (`$2a$10$...`) |
| `HNVERY` | 108 | 165 | 90 | `HN-2609000032` | H*** | TIDAK | reguler | **TIDAK** (`NULL`) |
| `HNVERY` | 108 | 166 | 91 | `HN-2609000033` | I*** | TIDAK | infant | **TIDAK** (`NULL`) |

### 7.3 Kasus Satu Jamaah Muncul di Lebih dari Satu Booking
- **DITEMUKAN FAKTA:** Ada 5 ID jamaah yang terdaftar di lebih dari 1 booking:
  - **Jamaah ID 1:** terdaftar di **22 booking** berbeda.
  - **Jamaah ID 2:** terdaftar di **15 booking** berbeda.
  - **Jamaah ID 3:** terdaftar di **5 booking** berbeda.
  - **Jamaah ID 74:** terdaftar di **2 booking** berbeda.
  - **Jamaah ID 77:** terdaftar di **2 booking** berbeda.
- **Kesimpulan:** Model kepemilikan data 1 jamaah ke N booking (*repeat order*) sudah terbentuk secara natural di data riil.

---

## 8. Daftar Penghalang (Blockers) untuk Skenario Multi-Aktivasi

Daftar diurutkan dari yang paling struktural ke kosmetik:

### 1. Hardcoded PIC Ownership Check di Handler Portal Booking ([Struktural - Kritis])
- **Lokasi:** [`internal/portal/handler.go:266`](file:///c:/laragon/www/erp-azhan/internal/portal/handler.go#L266)
- **Kode:**
  ```go
  if errors.Is(err, booking.ErrNotFound) || (b != nil && (b.JamaahID == nil || *b.JamaahID != jamaahID || b.Status == "draft"))
  ```
- **Masalah:** `b.JamaahID` memetakan langsung ke `bookings.pic_jamaah_id`.
- **Dampak:** Pax #2 yang sudah memiliki akun aktif dan berhasil login akan melihat card booking di halaman daftar (`/bookings`), namun saat membuka detail booking (`/bookings/{id}`), server merespons `404 Not Found`.

### 2. Hardcoded PIC Ownership Check di Handler Pembayaran Portal ([Struktural - Kritis])
- **Lokasi:** [`internal/portal/handler.go:296`](file:///c:/laragon/www/erp-azhan/internal/portal/handler.go#L296) & [`internal/portal/handler.go:359`](file:///c:/laragon/www/erp-azhan/internal/portal/handler.go#L359)
- **Kode:**
  ```go
  if errors.Is(err, booking.ErrNotFound) || (b != nil && (b.JamaahID == nil || *b.JamaahID != jamaahID))
  ```
- **Masalah:** Verifikasi kepemilikan pembayaran memeriksa apakah user adalah PIC.
- **Dampak:** Pax #2 tidak bisa melihat histori pembayaran booking (`/payments`) dan tidak bisa mengunggah bukti transfer.

### 3. Ketiadaan Mekanisme Input PIN Anggota pada Self-Booking ([Fungsional])
- **Lokasi:** [`internal/selfbooking/model.go:14-21`](file:///c:/laragon/www/erp-azhan/internal/selfbooking/model.go#L14-L21) & [`internal/selfbooking/repository.go:134-142`](file:///c:/laragon/www/erp-azhan/internal/selfbooking/repository.go#L134-L142)
- **Masalah:** Struct `AnggotaInput` tidak memiliki field PIN, dan proses INSERT ke tabel `jamaah` untuk anggota tidak menyertakan `portal_pin_hash`.
- **Dampak:** Seluruh pax anggota yang didaftarkan lewat self-booking otomatis memiliki `portal_pin_hash = NULL` dan tidak dapat login (kecuali dengan PIN fallback default `"123456"`).

### 4. Ketiadaan Endpoint Aktivasi Akun / Set PIN Mandiri bagi Jamaah ([Fungsional])
- **Lokasi:** [`cmd/api/main.go:191-207`](file:///c:/laragon/www/erp-azhan/cmd/api/main.go#L191-L207)
- **Masalah:** Tidak ada rute/flow publik untuk `POST /api/portal/activate` atau verifikasi nomor HP/identitas guna membuat PIN pertama kali bagi anggota rombongan.
- **Dampak:** Anggota rombongan yang baru ingin mengakses portal setelah booking selesai dibuat tidak memiliki pintu masuk untuk mengaktifkan kredensialnya.

### 5. Ketiadaan Tooling Admin untuk Portal Management ([Operasional])
- **Lokasi:** `internal/jamaah/handler.go` & `frontend/master-dashboard/src/pages/`
- **Masalah:** Tidak ada endpoint dan UI admin untuk mereset PIN jamaah yang lupa atau mengaktifkan akun jamaah hasil input manual admin.
- **Dampak:** Admin travel tidak bisa membantu jamaah yang terkendala akses portal.

---

## 9. Hal yang SUDAH Mendukung Skenario Multi-Aktivasi

1. **Skema Database `jamaah` Perorangan:** Kolom `id_jamaah` dan `portal_pin_hash` sudah berada di level baris individu tabel `jamaah`, bukan di level tabel `bookings`. Setiap orang siap memiliki ID dan PIN masing-masing tanpa perlu perubahan skema DDL tabel.
2. **Relasi M:N via `booking_pax`:** Struktur data rombongan tersimpan bersih di tabel perantara `booking_pax (booking_id, jamaah_id)`.
3. **Query List Booking Sudah Mendukung Anggota:** Query di [`internal/booking/repository.go:128`](file:///c:/laragon/www/erp-azhan/internal/booking/repository.go#L128) sudah mencari keterlibatan jamaah di `booking_pax`.
4. **Token Portal Berbasis Individu:** JWT portal memuat `sub = jamaahID` murni tanpa terikat role PIC, sehingga identitas token tetap valid untuk siapapun pax yang login.
5. **Isolasi Dokumen Jamaah Sudah Perorangan:** Tabel `dokumen_jamaah` dan endpoint dokumen (`GET/POST /api/portal/dokumen`) sudah terisolasi murni berdasarkan `jamaah_id` masing-masing.
