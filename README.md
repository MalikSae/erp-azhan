# ERP Azhan — Sistem ERP & Microsite Umroh Multi-Brand

Backend API dan Frontend Dashboard untuk ekosistem ERP Umroh Azhan Grup. Dibangun dengan Go (net/http + chi router) native, MySQL 8+, serta React + Vite untuk dashboard admin dan portal layanan.

---

## Tech Stack

### Backend
| Layer | Tech |
|---|---|
| Bahasa | Go 1.22+ |
| Router | chi v5 + CORS + Logger |
| Database | MySQL 8+ (Laragon lokal / Docker production) |
| DB Driver | `go-sql-driver/mysql` |
| Auth | JWT (`golang-jwt/jwt/v5`, HS256) + `bcrypt` |
| Config | `godotenv` (.env) |

### Frontend Dashboards
| Layer | Tech |
|---|---|
| Framework | React 18+ (Vite) |
| Styling | Tailwind CSS (Semantic Design System & Tokens) |
| Routing | `react-router-dom` |
| HTTP Client | `axios` |

---

## Struktur Project

```
erp-azhan/
├── cmd/
│   └── api/
│       └── main.go          # Entry point backend API server
├── internal/
│   ├── addon/               # Modul Add-Ons (perlengkapan/layanan tambahan master paket)
│   ├── adminuser/           # Modul Manajemen Admin/Staff & Reset Password (Super Admin)
│   ├── airline/             # Modul Master Maskapai Penerbangan
│   ├── booking/             # Modul Pemesanan (Bookings, Addons, Diskon, Progress, Distribusi Perlengkapan)
│   ├── brand/               # Modul Brand & Tenant Multi-Brand, Domain Resolution
│   ├── dokumen/             # Modul Manajemen Dokumen Jamaah (Paspor, KTP, KK, Buku Nikah, dll)
│   ├── hotel/               # Modul Master Hotel (Makkah, Madinah, Transit)
│   ├── identity/            # Modul Autentikasi & Otorisasi (JWT Admin Staff & Portal Jamaah)
│   ├── itinerary/           # Modul Master Itinerary & Itinerary Days
│   ├── jamaah/              # Modul Data Master Jamaah & ID Jamaah Terstruktur
│   ├── media/               # Modul Upload Media (Foto Hotel, Logo Maskapai, Dokumen, Bukti Bayar)
│   ├── paket/               # Modul / Helper struktur paket
│   ├── payment/             # Modul Pembayaran, Bukti Transfer & Verifikasi Transaksi
│   ├── perlengkapan/        # Modul Inventory Perlengkapan (Item Master, Stok per Brand, Set Template)
│   ├── portal/              # Modul Self-Service Portal Jamaah
│   ├── schedule/            # Modul Jadwal/Paket Keberangkatan & Seat Quota
│   └── shared/              # Shared config & DB connection helper
├── frontend/
│   ├── master-dashboard/    # Dashboard Super Admin (Multi-Brand, Master Data, Inventory Global, User Mgmt)
│   └── travel-dashboard/    # Dashboard Travel Partner (Katalog Paket, Jamaah, Booking, Progress, Stok Cabang)
├── migrations/              # Script dan file SQL migrasi database (001 - 028)
├── uploads/                 # Storage lokal file upload media, foto, & dokumen
├── .env.example             # Template konfigurasi environment
├── go.mod
└── go.sum
```

---

## Setup & Instalasi Lokal

### 1. Prasyarat

- **Go**: v1.22 atau lebih baru
- **Node.js**: v18+ & **npm**
- **MySQL 8+**: via Laragon (Windows native)
- **Git**

---

### 2. Setup Backend API

1. **Clone repositori dan download dependensi Go:**
   ```powershell
   git clone <repo-url> erp-azhan
   cd erp-azhan
   go mod tidy
   ```

2. **Buat file `.env`:**
   Salin `.env.example` menjadi `.env`:
   ```powershell
   cp .env.example .env
   ```
   Sesuaikan konfigurasi database dan port:
   ```env
   DB_HOST=127.0.0.1
   DB_PORT=3306
   DB_USER=root
   DB_PASSWORD=
   DB_NAME=erp_azhan_dev
   APP_PORT=9090

   JWT_SECRET=your_super_secret_jwt_key_here_minimum_32_chars
   JWT_ACCESS_TTL_MINUTES=15
   JWT_REFRESH_TTL_DAYS=7
   ```

3. **Buat Database MySQL:**
   - Database name: `erp_azhan_dev`
   - Charset: `utf8mb4`, Collation: `utf8mb4_unicode_ci`

4. **Jalankan Migrasi Database:**
   ```powershell
   go run migrations/run.go
   ```

5. **Jalankan Backend Server:**
   ```powershell
   go run cmd/api/main.go
   ```
   API akan aktif di **`http://localhost:9090`**.

---

### 3. Setup Frontend Dashboards

#### A. Master Dashboard (Super Admin)
```powershell
cd frontend/master-dashboard
npm install
npm run dev
```
> Berjalan di **`http://localhost:5173`**

#### B. Travel Dashboard (Brand / Tenant)
```powershell
cd frontend/travel-dashboard
npm install
npm run dev
```
> Berjalan di **`http://localhost:5174`**

### 4. Membuat / Mereset Akun Super Admin

Gunakan command seed berikut. Jangan menyimpan password produksi di README atau source code.

```powershell
go run cmd/seed-admin/main.go admin@example.com "password-minimal-8-karakter"
```

Super Admin Grup (`brand_id = null`) masuk melalui Master Dashboard. Admin yang terikat ke brand masuk melalui Travel Dashboard.

### 5. Verifikasi Build

```powershell
go test ./...

cd frontend/master-dashboard
npm run build

cd ../travel-dashboard
npm run build
```

---

## Fitur Dashboard Terkini

### Master Dashboard (`:5173`)

- Ringkasan jumlah brand, paket terbit, hotel, dan maskapai dari API aktual.
- Daftar keberangkatan terdekat, kapasitas kursi aktif, serta distribusi status paket.
- Line chart transaksi terkonfirmasi selama 30 hari terakhir, dengan garis dan warna berbeda untuk setiap brand.
- Pengelolaan beberapa rekening transfer resmi per brand dan inbox konfirmasi pembayaran lintas brand.
- Pengelolaan brand, pengguna, paket, hotel, maskapai, itinerary, add-on, inventory, dan laporan lintas brand.

### Travel Dashboard (`:5174`)

- Dashboard operasional berisi statistik booking, pembayaran, keberangkatan, tindak lanjut, dan kesiapan perlengkapan.
- Grafik arus pembayaran 30 hari berdasarkan transaksi terkonfirmasi.
- Pengelolaan jamaah, booking, add-on, diskon, pembayaran, progress dokumen, dan distribusi perlengkapan.
- Pembatalan **block seat** tanpa membatalkan data booking.
- Inbox konfirmasi transfer manual yang dapat dikonfirmasi atau ditolak oleh admin brand.
- Form dan modal menggunakan susunan label/input yang konsisten dan responsif.

### Paket dan Penerbangan

- Paket mendukung rute keberangkatan dan kepulangan lengkap, termasuk titik transit.
- Itinerary paket tersedia melalui endpoint publik dan admin.
- Harga Quad, Triple, dan Double disimpan sebagai nilai numerik utuh meskipun input ditampilkan dalam format Rupiah.
- Harga coret adalah harga sebelum promo sehingga wajib lebih besar dari Harga Quad.

---

## Ringkasan Endpoint API

### 1. Public Endpoints (Tanpa Auth)
| Method | Endpoint | Deskripsi |
|---|---|---|
| `GET` | `/api/health` | Status kesehatan API & koneksi database |
| `GET` | `/api/schedules` | List paket/jadwal yang berstatus *published* (publik) |
| `GET` | `/api/itineraries/{id}` | Detail itinerary paket publik |
| `GET` | `/api/public/brand` | Resolusi info brand berdasarkan domain / subdomain |
| `GET` | `/uploads/*` | Static file server untuk media, foto, & dokumen |

---

### 2. Auth Endpoints (Admin & Staff)
| Method | Endpoint | Deskripsi |
|---|---|---|
| `POST` | `/api/auth/login` | Login admin/staff & generate Access Token + Refresh Token |
| `POST` | `/api/auth/refresh` | Refresh Access Token menggunakan Refresh Token |
| `POST` | `/api/auth/logout` | Revoke token / Logout user |

---

### 3. Portal Jamaah Endpoints (`/api/portal/*`)
| Method | Endpoint | Deskripsi |
|---|---|---|
| `POST` | `/api/portal/login` | Login jamaah via NIK & No. WhatsApp |
| `GET` | `/api/portal/me` | Profil data jamaah yang sedang login |
| `GET` | `/api/portal/bookings` | Daftar riwayat booking milik jamaah |
| `GET` | `/api/portal/bookings/{id}` | Detail pemesanan & tracking progress keberangkatan |
| `GET` | `/api/portal/bookings/{id}/payments` | Riwayat pembayaran, invoice, & status verifikasi |
| `GET` | `/api/portal/dokumen` | Daftar dokumen jamaah & status kelengkapan |
| `POST` | `/api/portal/dokumen` | Upload / update dokumen mandiri oleh jamaah |
| `POST` | `/api/portal/media/upload` | Upload media / bukti dari portal jamaah |

---

### 4. Admin Endpoints (`/api/admin/*` — Memerlukan Bearer Token JWT)

#### A. Identitas & Brand
| Method | Endpoint | Deskripsi |
|---|---|---|
| `GET` | `/api/admin/my-brand` | Mendapatkan info brand user yang sedang login |
| `CRUD` | `/api/admin/brands` | *(Super Admin)* Manajemen master data brand & konfigurasi tenant |
| `CRUD` | `/api/admin/users` | *(Super Admin)* Manajemen akun admin/staff tenant |
| `PUT` | `/api/admin/users/{id}/password` | *(Super Admin)* Reset password pengguna admin/staff |

#### B. Master Data Operasional & Paket
| Method | Endpoint | Deskripsi |
|---|---|---|
| `CRUD` | `/api/admin/hotels` | Master data hotel Makkah, Madinah, & Transit |
| `CRUD` | `/api/admin/airlines` | Master data maskapai penerbangan |
| `CRUD` | `/api/admin/addons` | Master perlengkapan & layanan tambahan (Add-Ons) |
| `CRUD` | `/api/admin/itineraries` | Master itinerary perjalanan & detail program harian |
| `CRUD` | `/api/admin/schedules` | Manajemen paket/jadwal keberangkatan |
| `PUT` | `/api/admin/schedules/{id}/status` | Update status publikasi paket (*draft*, *published*, *archived*) |
| `PUT` | `/api/admin/schedules/{id}/seat` | Update kuota total kursi paket |
| `GET` | `/api/admin/analytics/transactions-30-days` | *(Super Admin)* Agregasi transaksi terkonfirmasi 30 hari per brand |
| `CRUD` | `/api/admin/bank-accounts` | *(Super Admin)* Kelola beberapa rekening transfer resmi per brand |

#### C. Data Jamaah & Dokumen
| Method | Endpoint | Deskripsi |
|---|---|---|
| `CRUD` | `/api/admin/jamaah` | Master data jamaah (NIK, Paspor, Kontak, Mahram, ID Terstruktur) |
| `GET` | `/api/admin/jamaah/{jamaah_id}/dokumen` | List dokumen jamaah (Paspor, KTP, KK, Buku Nikah, dll) |
| `POST` | `/api/admin/jamaah/{jamaah_id}/dokumen` | Upsert dokumen jamaah |
| `PUT` | `/api/admin/dokumen/{id}/status` | Verifikasi status dokumen (*pending*, *verified*, *rejected*) |

#### D. Transaksi Pemesanan (Bookings) & Operasional
| Method | Endpoint | Deskripsi |
|---|---|---|
| `CRUD` | `/api/admin/bookings` | Pemesanan paket jamaah & update status order (*draft*, *dp*, *lunas*, *batal*) |
| `DELETE` | `/api/admin/bookings/{id}/seat-block` | Batalkan block seat tanpa menghapus atau membatalkan booking |
| `POST` | `/api/admin/bookings/{id}/addons` | Tambah add-on berbayar ke booking tertentu |
| `DELETE`| `/api/admin/bookings/{id}/addons/{addon_id}`| Hapus add-on dari booking |
| `PUT` | `/api/admin/bookings/{id}/diskon` | Update nominal diskon transaksi booking |
| `PUT` | `/api/admin/bookings/{id}/progress` | Update checklist progress operasional (Paspor, Visa, Siskopatuh, Tiket, Hotel, Perlengkapan) |
| `PUT` | `/api/admin/bookings/{id}/perlengkapan/distribusi` | Catat distribusi perlengkapan & potong stok cabang otomatis |
| `DELETE`| `/api/admin/bookings/{id}/perlengkapan/distribusi`| Batalkan distribusi perlengkapan & kembalikan stok |

#### E. Inventory & Stok Perlengkapan
| Method | Endpoint | Deskripsi |
|---|---|---|
| `CRUD` | `/api/admin/perlengkapan-items` | *(Global)* Master item perlengkapan (Koper, Kain Ihram, Mukena, dll) |
| `GET` | `/api/admin/perlengkapan-stok` | Monitoring stok perlengkapan per brand |
| `PUT` | `/api/admin/perlengkapan-stok/{item_id}` | Penyesuaian / restock kuantitas perlengkapan brand |
| `GET` | `/api/admin/perlengkapan-set-template` | Template standar bundle set perlengkapan jamaah |
| `PUT` | `/api/admin/perlengkapan-set-template` | Update template standar bundle set perlengkapan |

#### F. Pembayaran (Payments) & Media Upload
| Method | Endpoint | Deskripsi |
|---|---|---|
| `GET` | `/api/admin/bookings/{booking_id}/payments` | List riwayat pembayaran suatu booking |
| `GET` | `/api/admin/payments` | Inbox pembayaran; lintas brand untuk pusat dan scoped untuk admin brand |
| `POST` | `/api/admin/bookings/{booking_id}/payments` | Pencatatan pembayaran & upload bukti transfer |
| `PUT` | `/api/admin/payments/{id}/status` | Konfirmasi pembayaran berstatus *pending* menjadi *confirmed* |
| `DELETE`| `/api/admin/payments/{id}` | Hapus catatan pembayaran |
| `POST` | `/api/admin/media/upload` | Upload gambar/dokumen multipart (foto hotel, logo maskapai, dokumen jamaah, bukti bayar) |

---

## Format Response Error Standar

Semua response error mengikuti format standar Bahasa Indonesia:
```json
{
  "error": "pesan error deskriptif dalam Bahasa Indonesia"
}
```
List kosong selalu mengembalikan array `[]`, bukan `null`.

---

## Migrasi Terbaru

| File | Perubahan |
|---|---|
| `026_schedule_flight_routes.sql` | Menambahkan data rute pergi, transit, tujuan, dan rute pulang pada paket |
| `027_booking_seat_block_state.sql` | Menyimpan status pembatalan block seat secara terpisah dari status booking |
| `028_manual_bank_transfers.sql` | Migrasi rekening lama, multi-rekening per brand, bukti transfer portal, dan status penolakan |

Selalu jalankan seluruh migrasi secara berurutan setelah menarik perubahan terbaru:

```powershell
go run migrations/run.go
```
