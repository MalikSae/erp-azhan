# Spesifikasi Pengguna & Struktur Organisasi — ERP Azhan

Tanggal: 3 Oktober 2026 · Status: **draf untuk review** (belum ada kode)
Menggantikan: modul "Pegawai & Divisi" (dibatalkan & dihapus 3 Okt 2026)
Bergantung pada: RBAC Fase 1–2 (`SPEK-RBAC-2026-10-02.md`, branch `feat/rbac-fase-1`)

## 0. Keputusan yang sudah disahkan

| # | Keputusan |
|---|---|
| K1 | **Satu orang = satu entitas Pengguna** (tabel `admin_users`). Tidak ada entitas pegawai terpisah. |
| K2 | Satu pengguna **boleh menjadi anggota beberapa tim**. |
| K3 | Satu unit **boleh punya lebih dari satu pemimpin**; semua pemimpin setara. |
| K4 | Format/domain email **bebas**; email tetap unik karena dipakai login. |

## 1. Tujuan & batasan

**Tujuan** (hanya tiga):
1. **Akses** — siapa boleh login dan melakukan apa (sudah ditangani RBAC).
2. **Penugasan** — kerjaan masuk ke tim mana dan siapa pemimpinnya (dipakai Marketing Hub untuk antrean & routing approval).
3. **Offboarding satu aksi** — menonaktifkan pengguna mematikan seluruh aksesnya.

**Bukan tujuan:** data HR (payroll, absensi, kontrak, data pribadi), bagan organisasi visual, tour leader/mutawif (master petugas terpisah di Lapisan 2), agen & jamaah (punya identitas portal sendiri).

## 2. Model konseptual — empat sumbu

| Sumbu | Pertanyaan | Penyimpanan | Dipakai logika? |
|---|---|---|---|
| Akses Brand | Data brand mana? | `admin_users.brand_id` (ada) — NULL = Holding | Ya — scope data |
| Role | Boleh melakukan apa? | `user_roles` (ada) | Ya — **satu-satunya** sumber izin |
| Keanggotaan unit | Kerjaan unit mana, sebagai pemimpin/anggota? | `org_member` (baru) | Ya — antrean & routing, **bukan** izin |
| Jabatan | Ditampilkan sebagai apa? | `admin_users.jabatan` teks bebas (baru) | **Tidak pernah** |

Contoh: Malik → akses Holding · role `cmo` + `super_admin_grup` · pemimpin divisi Marketing & Growth · jabatan "CMO".
Haikal → akses Holding · role `direktur` · pemimpin divisi Manajemen · jabatan "Direktur". Owner kedua → sama, tidak ditolak (K3).

## 3. Skema database — migrasi `081_pengguna_organisasi.sql`

```sql
-- Profil pengguna (display_name sudah ada; dipakai sebagai "Nama")
ALTER TABLE admin_users
  ADD COLUMN no_hp   VARCHAR(32)  NULL AFTER display_name,
  ADD COLUMN jabatan VARCHAR(100) NULL AFTER no_hp;

-- Backfill nama kosong dari bagian lokal email (tampilan saja; bisa diedit)
UPDATE admin_users SET display_name = SUBSTRING_INDEX(email, '@', 1)
WHERE display_name IS NULL OR display_name = '';

-- Struktur organisasi: pohon 2 tingkat (divisi -> tim)
CREATE TABLE IF NOT EXISTS org_unit (
  id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  parent_id  BIGINT UNSIGNED NULL,
  jenis      ENUM('divisi','tim') NOT NULL,
  nama       VARCHAR(120) NOT NULL,
  urutan     INT NOT NULL DEFAULT 0,
  is_active  BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_org_parent FOREIGN KEY (parent_id) REFERENCES org_unit(id) ON DELETE RESTRICT
) ENGINE=InnoDB;

-- Keanggotaan: many-to-many, posisi per keanggotaan
CREATE TABLE IF NOT EXISTS org_member (
  admin_user_id BIGINT UNSIGNED NOT NULL,
  org_unit_id   BIGINT UNSIGNED NOT NULL,
  posisi        ENUM('pemimpin','anggota') NOT NULL DEFAULT 'anggota',
  created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (admin_user_id, org_unit_id),
  CONSTRAINT fk_om_user FOREIGN KEY (admin_user_id) REFERENCES admin_users(id) ON DELETE CASCADE,
  CONSTRAINT fk_om_unit FOREIGN KEY (org_unit_id) REFERENCES org_unit(id) ON DELETE RESTRICT
) ENGINE=InnoDB;
```

**Data awal** (seed sekali, setelahnya sepenuhnya dikelola lewat UI): divisi Manajemen, Marketing & Growth, Keuangan, Operasional, Produk, IT; enam tim M&G sesuai SOP (Brand & Community, Creative Production, SEO, Performance Marketing, Sales & Customer Acquisition, Web Dev & MarTech). Tidak ada metadata unit yang dibaca logika (sebutan, role default) — pelajaran dari percobaan sebelumnya.

Tidak ada permission baru: pengelolaan pengguna **dan** struktur organisasi memakai `user.admin`.

## 4. Aturan bisnis yang ditegakkan

| # | Aturan | Tempat penegakan | Status |
|---|---|---|---|
| R1 | Email unik | DB unique + cek repository | Ada |
| R2 | Minimal satu role per pengguna | Service (transaksi) | Ada |
| R3 | Scope role cocok dengan akses brand — **saat set role DAN saat mengubah akses brand** | Service (transaksi) | Sebagian — celah saat ubah akses brand **masih ada di branch ini**, wajib ditutup |
| R4 | Selalu ada **minimal satu pengguna aktif dengan role Super Admin** — berlaku saat nonaktif, hapus, ubah role, ubah akses | Service | Ganti guard lama yang menghitung `brand_id IS NULL` (tidak lagi tepat setelah RBAC) |
| R5 | Tidak bisa menonaktifkan/menghapus akun sendiri | Service | Hapus: ada; nonaktif: baru |
| R6 | Nonaktif = login ditolak + semua refresh token dicabut + `perm_version` naik | Service (transaksi) | Baru |
| R7 | **Hapus permanen hanya untuk pengguna tanpa riwayat** (lihat §4.1); selain itu 409 "nonaktifkan saja" | Service | Baru |
| R8 | Tim wajib di bawah divisi; divisi tanpa parent; maksimal 2 tingkat | Service | Baru |
| R9 | Unit tidak bisa dihapus bila masih punya sub-unit atau anggota — nonaktifkan; keanggotaan baru hanya ke unit aktif | Service + FK RESTRICT | Baru |
| R10 | Penugasan mengikuti divisi asal role (`roles.divisi_id`, migrasi 083): unit hanya boleh di divisi asal salah satu role pengguna (tim mengikuti divisinya); Super Admin bebas; pengguna akses Brand tidak punya penugasan. Ganti role yang bertentangan dengan penugasan ditolak (409 di `PUT /users/{id}/roles`); divisi asal role tidak bisa dihapus; tim berisi tidak bisa dipindah divisi | Service (`rbac.ValidateUnitsForRoles`) + FK RESTRICT | Tambahan 2026-10-03 |

**Sengaja TIDAK ditegakkan:** jumlah pemimpin per unit (K3), jumlah tim per orang (K2), domain email (K4), kecocokan jabatan dengan role/posisi, pemberian role otomatis dari keanggotaan.

### 4.1 Mengapa hapus permanen dibatasi (temuan dari skema)
Sebelas tabel mereferensikan `admin_users` dengan `ON DELETE NO ACTION` (bookings, booking_refunds, crm_deal_requests, jamaah, jamaah_activation_tokens, jamaah_kaitan_log, pemakaian_cashback, pembayaran_pendaftaran_agen, pengajuan_pencairan, dst.). Menghapus pengguna yang pernah bekerja akan gagal di DB atau menghancurkan jejak audit. Maka offboarding = **nonaktifkan**; hapus hanya untuk akun salah buat tanpa riwayat.

### 4.2 Atomisitas
"Pengguna + role + keanggotaan" adalah **satu agregat**. Create dan update dijalankan dalam **satu transaksi DB** di service `adminuser`. Fungsi RBAC yang dipakai di dalamnya (validasi scope, set role) disediakan dalam bentuk yang menerima transaksi, bukan membuka transaksi sendiri. Tidak ada handler yang merangkai efek lintas tabel.

## 5. API

| Method & path | Keterangan |
|---|---|
| `GET /api/admin/users` | Daftar + `display_name, no_hp, jabatan, is_active, roles[], keanggotaan[{unit_id, unit_nama, jenis, posisi}]` |
| `GET /api/admin/users/{id}` | Detail untuk halaman edit |
| `POST /api/admin/users` | `{display_name, email, no_hp, jabatan, brand_id, password, roles[], keanggotaan[{org_unit_id, posisi}]}` — atomik |
| `PUT /api/admin/users/{id}` | Sama tanpa password — atomik; R2–R4 |
| `PUT /api/admin/users/{id}/status` | `{is_active}` — offboarding/aktivasi ulang; R4–R6 |
| `PUT /api/admin/users/{id}/password` | Ada — reset password (cabut sesi, sudah ada) |
| `DELETE /api/admin/users/{id}` | Hanya tanpa riwayat (R7) |
| `GET /api/admin/org-units` | Pohon divisi→tim + jumlah anggota + daftar pemimpin |
| `POST /api/admin/org-units` · `PUT /{id}` · `PUT /{id}/status` | Kelola unit (R8, R9). Jenis unit tidak bisa diubah; tim boleh dipindah ke divisi lain. Divisi tidak bisa dinonaktifkan selama masih ada tim aktif; tim tidak bisa diaktifkan bila divisinya nonaktif (tanpa efek samping tersembunyi). |
| `DELETE /api/admin/org-units/{id}` | Hanya unit kosong — tanpa tim, tanpa anggota (R9), dan bukan divisi asal role (R10); unit berisi dinonaktifkan |

Catatan R9 (diperjelas saat implementasi): keanggotaan **baru** wajib ke unit aktif, tetapi keanggotaan **lama** di unit yang kemudian dinonaktifkan tetap boleh dipertahankan saat pengguna diedit — tanpa ini pengguna tersebut tidak bisa disimpan sama sekali.

Semua di balik `user.admin`. Endpoint `PUT /users/{id}/roles` (RBAC Fase 1) dipertahankan untuk kompatibilitas tetapi UI memakai form terpadu.

**Dihapus:** `/api/admin/crm/users` + modul `internal/crmuser`. Terverifikasi tidak dipanggil oleh frontend ERP maupun repo CRM (`new-csumroh`); merupakan "pintu kedua" pembuatan akun. Akun CS dibuat lewat halaman Pengguna (role Customer Service + akses brand).

## 6. UI (master-dashboard, gaya SmartHR)

Menu ADMINISTRASI: **Pengguna** (`/users`) · **Struktur Organisasi** (`/organisasi`). Satu halaman per entitas.

**Pengguna — daftar:** kolom Nama (jabatan + email di bawahnya) · Akses Brand · Role · Tim · Status · Aksi (Edit, Reset Password, Nonaktifkan/Aktifkan). Stat: Pengguna Aktif · Akses Holding · Akses Brand · Nonaktif. Filter status (default: aktif).

**Pengguna — form halaman penuh** (`/users/new`, `/users/:id`), empat bagian:
1. **Profil** — Nama*, Email*, No. HP, Jabatan (teks bebas, keterangan "hanya label tampilan").
2. **Akses** — Akses Brand* (Holding / Per Brand + brand) dan daftar Role* tersaring scope, tiap role bisa dibuka melihat permission.
3. **Penugasan** — daftar baris {Unit (divisi/tim aktif), Posisi (Pemimpin/Anggota)}, boleh banyak baris, boleh kosong.
4. **Keamanan** — Password* hanya saat membuat; saat edit diganti tombol Reset Password.

**Struktur Organisasi:** daftar divisi dengan tim di bawahnya; per unit tampil jumlah anggota & nama pemimpin; aksi tambah/ubah/nonaktifkan. Tidak ada urusan akun di halaman ini.

**Konsistensi istilah:** "Pengguna" (bukan user/admin user/pegawai/akun), "Akses Brand", "Role", "Unit/Tim", "Posisi", "Jabatan". Badge "Anda" pada baris diri sendiri.

## 7. Dampak ke modul lain

- **Login:** aturan dashboard tidak berubah (master = akses Holding, travel = akses brand); pengguna nonaktif ditolak (sudah).
- **Marketing Hub (nanti):** boleh approve kerjaan unit X ⇔ punya `marketing.approve.lead` **dan** pemimpin di unit X; eskalasi CMO memakai `marketing.approve.cmo`. Antrean member = kerjaan unit tempat ia menjadi anggota/pemimpin.
- **CRM (nanti):** identitas CS bersumber dari Pengguna ERP (ERP = IdP); nonaktif di ERP menjadi sinyal offboarding akun CS di CRM.
- **travel-dashboard AccountSettings:** tetap memakai `display_name`.

## 8. Data existing

Tiga akun lokal (`admin@azhan.id`, `admin.tp`, `admin.bu`) tetap; nama di-backfill, keanggotaan kosong — dilengkapi manual lewat form. Tidak ada migrasi data lain.

## 9. Rencana implementasi (bertahap, tiap tahap diverifikasi sebelum lanjut)

1. **Migrasi 081** + seed unit → verifikasi skema & seed.
2. **Backend service transaksional** (`adminuser`): create/update/status/delete dengan R1–R7; RBAC versi ber-transaksi; perbaikan R3 & R4 → uji API seluruh skenario §10.
3. **Backend org-units** (R8, R9) → uji API.
4. **Hapus `crmuser`** + route → build & test.
5. **Frontend Pengguna** (daftar + form penuh) menggantikan UserManagementPage/UserFormPage/modal Kelola Role → uji browser.
6. **Frontend Struktur Organisasi** → uji browser.
7. Review menyeluruh, lalu **menunggu perintah commit** (tidak ada commit otomatis).

## 10. Skenario uji wajib

| # | Skenario | Hasil yang diharapkan |
|---|---|---|
| T1 | Buat pengguna lengkap (role + 2 tim, salah satunya pemimpin) | 201; semua tersimpan dalam satu transaksi |
| T2 | Buat pengguna dengan role scope salah | 400; nol baris tercipta di semua tabel |
| T3 | Ubah akses brand sehingga role jadi tak sah | 400; tidak ada yang berubah |
| T4 | Dua pemimpin di satu unit; satu orang di dua tim | Diterima |
| T5 | Nonaktifkan Super Admin terakhir / ubah role-nya | 409 |
| T6 | Nonaktifkan diri sendiri | 409 |
| T7 | Nonaktifkan pengguna | Login 401; refresh token tercabut |
| T8 | Hapus pengguna yang punya booking | 409 "nonaktifkan saja" |
| T9 | Hapus pengguna salah buat tanpa riwayat | 200; role & keanggotaan ikut terhapus |
| T10 | Hapus unit yang masih punya anggota | 409 |
| T11 | Tim tanpa parent divisi / tim di bawah tim | 400 |
| T12 | `go test ./...` | Hijau |

## 11. Di luar cakupan

Data HR, bagan organisasi visual, pemberian role otomatis dari keanggotaan, CRUD role/permission via UI, master petugas (tour leader/mutawif), sinkronisasi akun ke CRM.
