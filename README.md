# ERP Azhan — Sistem ERP Umroh Multi-Brand

Backend API dan Frontend Dashboard untuk ekosistem ERP Umroh Azhan Grup. Dibangun dengan Go (net/http + chi router) native, MySQL 8+, serta React + Vite untuk dashboard internal (Office Dashboard). Web publik, booking pelanggan, dan CMS SEO ada di repositori terpisah `azhan-booking`.

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
│   ├── api/                 # Entry point backend API server
│   └── seed-*/              # Seeder admin, hotel, maskapai, dan paket
├── internal/
│   ├── adminuser/           # Pengguna admin/staff dan reset password
│   ├── agen/                # Agen Syiar: pengajuan, persetujuan, pencairan, ganti kaitan (portal agen belum dipasang)
│   ├── airline/ airport/ hotel/ destination/ category/ itinerary/   # Master data
│   ├── bankaccount/         # Rekening transfer resmi per brand
│   ├── booking/             # Booking (admin): status, diskon, progress, refund, cashback, perlengkapan
│   ├── brand/               # Brand multi-brand dan resolusi domain
│   ├── crmdeal/             # Pemrosesan deal dari CRM
│   ├── customer/            # Akun pelanggan (email + password) dan pesanan pelanggan
│   ├── dokumen/             # Dokumen jamaah (paspor, KTP, KK, dll)
│   ├── identity/            # Login admin, JWT, dan pemeriksaan izin
│   ├── jamaah/              # Data master jamaah
│   ├── komisi/              # Perhitungan komisi, cashback, dan saldo agen
│   ├── mailer/              # Email transaksional (verifikasi, reset password, booking)
│   ├── media/               # Upload dan akses berkas terlindungi
│   ├── orgunit/             # Struktur organisasi
│   ├── payment/             # Pembayaran, bukti transfer, verifikasi
│   ├── perlengkapan/        # Inventori perlengkapan dan template set
│   ├── rbac/                # Role dan permission
│   ├── schedule/            # Paket/jadwal keberangkatan, seat, dan API publik v1
│   ├── selfbooking/         # Checkout booking mandiri pelanggan
│   └── shared/              # Konfigurasi, middleware, dan helper bersama
├── frontend/
│   ├── master-dashboard/    # Office Dashboard (holding dan admin brand, menu mengikuti izin)
│   └── shared/              # Komponen dan token UI bersama
├── migrations/              # Migrasi SQL berurutan (jalankan lewat migrations/run.go)
├── docs/
│   ├── spek/                # Spek yang masih berlaku (web publik dan CMS, RBAC, pengguna dan organisasi)
│   └── arsip/               # Audit lama yang masih jadi acuan
├── scripts/                 # Skrip bantu
├── uploads/                 # Penyimpanan lokal berkas unggahan
├── .env.example             # Template konfigurasi environment
├── go.mod
└── go.sum
```

Dokumentasi rancangan ada di `docs/spek/`. Audit yang masih dipakai sebagai acuan ada di `docs/arsip/`; audit lain tersimpan di riwayat git.

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
   APP_ENV=development

   # WAJIB diganti: API menolak start bila kosong, < 32 karakter, atau masih nilai contoh.
   # Buat dengan: openssl rand -hex 32
   JWT_SECRET=
   JWT_ACCESS_TTL_MINUTES=15
   JWT_REFRESH_TTL_DAYS=7

   # Captcha fail closed: tanpa TURNSTILE_SECRET_KEY, daftar akun dan booking mandiri ditolak.
   # Untuk pengembangan lokal saja, lewati captcha secara eksplisit:
   ALLOW_DEV_CAPTCHA=true
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

#### A. Office Dashboard (Super Admin)
```powershell
cd frontend/master-dashboard
npm install
npm run dev
```
> Berjalan di **`http://localhost:5173`**

### 4. Membuat / Mereset Akun Super Admin

Gunakan command seed berikut. Jangan menyimpan password produksi di README atau source code.

```powershell
go run cmd/seed-admin/main.go admin@example.com "password-minimal-8-karakter"
```

Super Admin Grup (`brand_id = null`) masuk melalui Office Dashboard. Admin yang terikat ke brand juga masuk melalui Office Dashboard; menu dan data otomatis dibatasi ke brand-nya.

### 5. Verifikasi Build

```powershell
go test ./...

cd frontend/master-dashboard
npm run build

```

### 6. Seeder Hotel dan Maskapai

Isi master data dengan nama hotel Mekkah/Madinah dan maskapai nyata yang umum digunakan perjalanan umrah:

```bash
go run -buildvcs=false cmd/seed-hotels-airlines/main.go
```

Seeder ini aman dijalankan ulang. Data yang sudah ada tidak diduplikasi dan
logo maskapai yang masih kosong akan dilengkapi dari aset di
`uploads/airline-logos`. Logo yang telah diatur admin tidak akan ditimpa.

Jika data hotel dan maskapai sudah pernah di-seed sebelum dukungan logo tersedia,
jalankan migration berikut setelah menarik versi terbaru:

```bash
go run -buildvcs=false migrations/run.go migrations/030_seed_airline_logo_urls.sql
go run -buildvcs=false migrations/run.go migrations/031_seed_hotel_logo_urls.sql
```

Jarak hotel merupakan estimasi operasional dan dapat dikoreksi melalui dashboard.
Foto hotel dan logo maskapai yang belum disediakan tetap dapat diunggah melalui
dashboard menggunakan aset resmi yang memiliki izin penggunaan.

---

## Fitur Dashboard Terkini

### Office Dashboard (`:5173`)

- Ringkasan jumlah brand, paket terbit, hotel, dan maskapai dari API aktual.
- Daftar keberangkatan terdekat, kapasitas kursi aktif, serta distribusi status paket.
- Line chart transaksi terkonfirmasi selama 30 hari terakhir, dengan garis dan warna berbeda untuk setiap brand.
- Pengelolaan beberapa rekening transfer resmi per brand dan inbox konfirmasi pembayaran lintas brand.
- Pengelolaan brand, pengguna, paket, hotel, maskapai, itinerary, inventory perlengkapan, dan laporan lintas brand.

### Admin Brand di Office Dashboard (`:5173`)

- Dashboard operasional berisi statistik booking, pembayaran, keberangkatan, tindak lanjut, dan kesiapan perlengkapan.
- Grafik arus pembayaran 30 hari berdasarkan transaksi terkonfirmasi.
- Pengelolaan jamaah, booking, diskon, pembayaran, progress dokumen, dan distribusi perlengkapan.
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

Daftar lengkap ada di `cmd/api/main.go`. Rute `/api/admin/*` dibatasi per izin (RBAC) dan per brand pengguna.

### 1. Publik (tanpa auth)
| Method | Endpoint | Deskripsi |
|---|---|---|
| `GET` | `/api/health` | Status API dan koneksi database |
| `GET` | `/api/public/v1/packages`, `/packages/{id}` | Paket untuk web publik (read-only, stabil, cache 60 detik) |
| `GET` | `/api/public/v1/destinations`, `/sitemap-feed` | Destinasi dan data sitemap |
| `GET` | `/api/schedules`, `/api/schedules/{id}` | Jadwal published |
| `GET` | `/api/itineraries/{id}` | Itinerary paket |
| `GET` | `/api/public/brand` | Identitas brand berdasarkan domain |
| `GET` | `/api/public/categories`, `/bank-accounts` | Kategori paket dan rekening resmi brand |
| `GET` | `/uploads/{category}/{filename}` | Berkas publik |

### 2. Akun pelanggan (email dan password)
| Method | Endpoint | Deskripsi |
|---|---|---|
| `POST` | `/api/public/auth/register`, `/login`, `/verify-email` | Daftar, masuk, verifikasi email |
| `POST` | `/api/public/auth/forgot-password`, `/reset-password` | Pemulihan password |
| `GET` | `/api/account/me`, `/bank-accounts`, `/travelers` | Profil, rekening, dan data peserta tersimpan |
| `GET` `POST` | `/api/account/bookings` | Daftar pesanan dan checkout booking baru |
| `GET` | `/api/account/bookings/{id}` | Detail pesanan |
| `POST` | `/api/account/bookings/{id}/cancel` | Batalkan pesanan |
| `GET` `POST` | `/api/account/bookings/{id}/payments` | Riwayat dan konfirmasi pembayaran |
| `POST` | `/api/account/media/upload` | Unggah bukti pembayaran |

Rute portal agen `/api/portal/agen/*` belum dipasang; akan dibangun ulang di atas akun email.

### 3. Admin dan staff
| Method | Endpoint | Deskripsi |
|---|---|---|
| `POST` | `/api/auth/login`, `/refresh`, `/logout` | Login admin/staff dan token |
| `GET` | `/api/admin/me`, `/my-brand` | Identitas, brand, dan izin pengguna |

Kelompok rute di bawah `/api/admin/`:

| Kelompok | Rute |
|---|---|
| Identitas dan organisasi | `users`, `roles`, `permissions`, `org-units`, `brands`, `bank-accounts` |
| Master data | `hotels`, `airlines`, `airports`, `destinations`, `categories`, `itineraries` |
| Paket | `schedules` (termasuk `/status`, `/seat`, `/perlengkapan`) |
| Jamaah dan dokumen | `jamaah`, `jamaah/{id}/dokumen`, `dokumen/{id}/status` |
| Booking | `bookings` (draft, finalize, status, seat-block, diskon, cashback, refund, progress, pax) |
| Pembayaran | `payments`, `bookings/{id}/payments`, `payments/{id}/status` |
| Perlengkapan | `perlengkapan-items`, `-stok`, `-mutasi`, `-set-template` |
| Agen Syiar | `agen`, `agen/pencairan`, `kaitan-agen` |
| CRM | `crm/deals` |
| Laporan | `analytics/transactions-30-days`, `analytics/pax-30-days` |
| Media | `media/upload`, `media/{category}/{filename}` (berkas terlindungi) |

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

## Migrasi

File di `migrations/` berurutan menurut nomor. Isi tiap file dijelaskan di komentar awalnya.

Selalu jalankan seluruh migrasi secara berurutan setelah menarik perubahan terbaru:

```powershell
go run migrations/run.go
```

---

## Deployment VPS dengan aaPanel (Dari Awal Sampai Online)

Contoh domain pada panduan ini:

- API: `api.example.com`
- Office Dashboard: `office.example.com`
- Web publik dan CMS dipasang dari repository `azhan-booking` secara terpisah (lihat `docs/deploy.md` di repo itu).

Ganti seluruh domain, password, dan path sesuai server produksi. Jangan memakai kredensial contoh secara langsung.

### 1. Siapkan server dan aaPanel

1. Gunakan Ubuntu 22.04/24.04 dengan minimal 2 vCPU dan RAM 4 GB.
2. Arahkan DNS A record ketiga domain ke IP VPS.
3. Instal aaPanel dari dokumentasi resminya, kemudian pasang **Nginx**, **MySQL 8**, dan **Node.js Manager** dari App Store.
4. Melalui terminal aaPanel, instal Git, build tools, dan Go 1.25+:

```bash
sudo apt update
sudo apt install -y git curl build-essential
cd /tmp
curl -LO https://go.dev/dl/go1.25.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.25.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/go.sh
source /etc/profile.d/go.sh
go version
```

### 2. Clone dan siapkan database

```bash
cd /www/wwwroot
git clone https://github.com/MalikSae/erp-azhan.git
cd erp-azhan
go mod download
```

Di aaPanel buat database dan user khusus, misalnya `erp_azhan`, menggunakan charset `utf8mb4`. Jangan gunakan user `root` untuk aplikasi. Salin konfigurasi:

```bash
cp .env.example .env
nano .env
```

Contoh `.env` produksi:

```env
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=erp_azhan_app
DB_PASSWORD=PASSWORD_DATABASE_YANG_KUAT
DB_NAME=erp_azhan
APP_PORT=9090
# WAJIB di produksi: mematikan origin CORS pengembangan (localhost, *.test), mewajibkan
# TURNSTILE_SECRET_KEY saat start, dan mencegah isi email (token) ditulis ke log.
APP_ENV=production
CORS_ALLOWED_ORIGINS=https://office.example.com,https://*.example.com
# Acak, minimal 32 karakter (disarankan 64 heks: openssl rand -hex 32). Placeholder ditolak saat start.
JWT_SECRET=RANDOM_MINIMAL_64_KARAKTER
# Maksimal 60 (nilai lebih besar dibatasi 60 menit).
JWT_ACCESS_TTL_MINUTES=15
JWT_REFRESH_TTL_DAYS=7
# Wajib di produksi (API gagal start bila kosong). Jangan set ALLOW_DEV_CAPTCHA di produksi.
TURNSTILE_SECRET_KEY=SECRET_TURNSTILE_DARI_CLOUDFLARE
# Wajib agar email verifikasi/reset/booking terkirim; tanpa SMTP di produksi email TIDAK dikirim.
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=noreply@example.com
SMTP_PASSWORD=PASSWORD_SMTP
SMTP_FROM=noreply@example.com
```

Jalankan seluruh migrasi **sekali pada database baru**, berurutan berdasarkan nama file:

```bash
for file in migrations/*.sql; do
  echo "Menjalankan $file"
  go run migrations/run.go "$file" || exit 1
done
```

Migration bukan script reset dan tidak boleh dijalankan ulang sembarangan pada database berisi data. Untuk update berikutnya, jalankan hanya file migration baru yang belum diterapkan. Buat admin pertama tanpa menulis password ke README:

```bash
go run cmd/seed-admin/main.go admin@example.com 'PASSWORD_ADMIN_YANG_KUAT'
```

### 3. Build dan jalankan API

```bash
mkdir -p bin uploads
go build -o bin/erp-azhan-api ./cmd/api
chmod 750 bin/erp-azhan-api
```

Buat service `/etc/systemd/system/erp-azhan-api.service`:

```ini
[Unit]
Description=ERP Azhan API
After=network.target mysql.service

[Service]
Type=simple
User=www
Group=www
WorkingDirectory=/www/wwwroot/erp-azhan
ExecStart=/www/wwwroot/erp-azhan/bin/erp-azhan-api
Restart=always
RestartSec=5
EnvironmentFile=/www/wwwroot/erp-azhan/.env
# Jaga-jaga bila .env tidak memuat APP_ENV (nilai di EnvironmentFile tetap menang; pastikan .env juga production).
Environment=APP_ENV=production

[Install]
WantedBy=multi-user.target
```

```bash
sudo chown -R www:www /www/wwwroot/erp-azhan
sudo chmod 640 /www/wwwroot/erp-azhan/.env
sudo systemctl daemon-reload
sudo systemctl enable --now erp-azhan-api
sudo systemctl status erp-azhan-api
curl http://127.0.0.1:9090/api/health
```

### 4. Build dashboard

Pilih Node.js 20/22 LTS di Node.js Manager. API produksi harus ditentukan **sebelum** build karena Vite menanam environment ke bundle.

```bash
cd /www/wwwroot/erp-azhan/frontend/master-dashboard
printf 'VITE_API_BASE_URL=https://api.example.com\n' > .env.production
npm ci
npm run build
```

### 5. Buat website dan reverse proxy aaPanel

1. Buat website `api.example.com`; hapus index bawaan lalu atur **Reverse Proxy** ke `http://127.0.0.1:9090`.
2. Buat website `office.example.com` dengan document root `/www/wwwroot/erp-azhan/frontend/master-dashboard/dist`.
3. Untuk SPA dashboard, tambahkan pada konfigurasi Nginx di dalam blok `server`:

```nginx
location / {
    try_files $uri $uri/ /index.html;
}
```

5. Aktifkan SSL Let's Encrypt untuk ketiga domain dan centang **Force HTTPS**.
6. Pastikan seluruh origin HTTPS produksi tercantum pada `CORS_ALLOWED_ORIGINS` (dipisahkan koma). Batasi port publik ke `80`, `443`, dan port SSH; port `9090` cukup diakses localhost.

### 6. Permission, backup, dan verifikasi

Direktori `uploads` wajib persisten dan dapat ditulis oleh user service:

```bash
sudo chown -R www:www /www/wwwroot/erp-azhan/uploads
sudo find /www/wwwroot/erp-azhan/uploads -type d -exec chmod 750 {} \;
```

Atur Cron aaPanel untuk backup database dan folder `uploads` setiap hari. Uji deployment:

```bash
curl -i https://api.example.com/api/health
curl -I https://office.example.com
journalctl -u erp-azhan-api -n 100 --no-pager
```

### 7. Prosedur update

```bash
cd /www/wwwroot/erp-azhan
git pull --ff-only origin main
go mod download
# Jalankan hanya migration baru yang belum pernah diterapkan.
go test ./...
go build -o bin/erp-azhan-api ./cmd/api
sudo systemctl restart erp-azhan-api

cd frontend/master-dashboard && npm ci && npm run build
```

Jangan menyimpan `.env`, dump database, private key SSL, atau isi `uploads` ke Git.
