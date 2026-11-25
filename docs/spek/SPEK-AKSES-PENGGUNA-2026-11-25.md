# Spesifikasi Akses Pengguna — ERP Azhan

Tanggal: 25 November 2026 · Status: **disahkan**
Menggantikan: `SPEK-RBAC-2026-10-02.md` dan `SPEK-PENGGUNA-ORGANISASI-2026-10-03.md` (keduanya dihapus; isinya ada di riwayat git).

## 1. Ringkasan

Akses pengguna dashboard ERP disederhanakan menjadi dua jenis:

- **Super Admin**: memegang semua izin. Hanya untuk pengguna Holding.
- **Tim**: memegang izin yang dicentang di form penggunanya. Minimal satu izin.

Tidak ada lagi role per jabatan atau divisi, struktur organisasi, penugasan unit, maupun jabatan. Penjaga route (`RequirePermission`) dan tabel `permissions` tidak berubah; yang berubah hanya sumber izin, dari role ke pengguna.

Alasan: dari 10 role hanya 3 yang punya pengguna, tidak ada pengguna yang memegang lebih dari satu role, dan struktur organisasi tidak dibaca modul mana pun (dibangun untuk Marketing Hub yang tidak pernah ada).

## 2. Aturan

| # | Aturan |
|---|---|
| A1 | Pengguna punya tepat satu jenis akses: `super_admin` atau `tim`. |
| A2 | Super Admin hanya untuk pengguna Holding (`brand_id` kosong). |
| A3 | Tim wajib punya minimal satu izin. |
| A4 | Izin `user.admin`, `brand.edit`, dan `crmdeal.process` hanya dimiliki Super Admin; tidak bisa dicentang untuk Tim. |
| A5 | Pengguna yang terikat brand hanya bisa diberi izin dalam daftar izin brand (§4). |
| A6 | Sistem wajib selalu punya minimal satu Super Admin aktif. Menurunkan, menonaktifkan, atau menghapus Super Admin aktif terakhir ditolak (409). |
| A7 | Mengubah jenis akses, izin, atau akses brand menaikkan `perm_version` dan mencabut semua sesi pengguna itu (login ulang). |
| A8 | Izin baru tidak otomatis diberikan ke Tim. Super Admin langsung memilikinya; Tim dicentang per pengguna. |
| A9 | Semua pengguna boleh mengganti kata sandinya sendiri. |

Batas data antar brand tetap dari `admin_users.brand_id` dan tidak diubah spek ini.

## 3. Skema — migrasi `160_akses_pengguna.sql`

Baru:

- `admin_users.is_super_admin BOOLEAN NOT NULL DEFAULT FALSE`
- `user_permissions (admin_user_id, permission_id)`, kunci gabungan, FK `ON DELETE CASCADE` ke `admin_users` dan `permissions`.

Dihapus:

- Tabel `roles`, `role_permissions`, `user_roles`, `org_unit`, `org_member`.
- Kolom `admin_users.role` dan `admin_users.jabatan`; indeks `idx_admin_users_brand_role_active` diganti `idx_admin_users_brand_active (brand_id, is_active)`.
- 13 izin tanpa route: `marketing.*` (6), `compliance.*` (2), `paket.*` (3), `audit.view`, `report.finance.view`.

Pemindahan data (sebelum tabel lama dihapus): pemegang role `super_admin_grup` menjadi Super Admin; pengguna lain menjadi Tim dengan gabungan izin role lamanya, dikurangi izin A4.

## 4. Daftar centang

Form Pengguna menampilkan satu baris per menu. Mencentang Kelola otomatis mencentang Lihat. Definisi baris ada di `frontend/master-dashboard/src/constants/izinAkses.js`.

| Baris | Lihat | Kelola | Brand |
|---|---|---|---|
| Jamaah | `jamaah.view`, `dokumen.view` | `jamaah.edit`, `dokumen.edit` | ya |
| Booking | `booking.view` | `booking.edit` | ya |
| Pembayaran | `payment.view`, `refund.view` | `payment.edit`, `refund.edit` | ya |
| Akun | `akun.view` | `akun.edit` | ya |
| Agen | `agen.view`, `komisi.view` | `agen.edit` | ya |
| Paket | `schedule.view` | `schedule.edit` | hanya Lihat |
| Manifest | | `manifest.manage` | ya |
| Manasik | `manasik.view` | `manasik.edit` | ya |
| Master data | `masterdata.view` | `masterdata.edit` | hanya Lihat |
| Perlengkapan | `perlengkapan.view` | `perlengkapan.edit` | ya |
| Rekening bank | `bankaccount.view` | `bankaccount.edit` | ya |
| Brand | `brand.view` | | ya |
| Pengumuman | | `pengumuman.manage` | ya |
| Banner Promosi | | `banner.manage` | ya |
| Konten web | `cms.view` | `cms.edit`, `cms.blog.write` | tidak |
| Jelajah Haramain | | `haramain.manage` | tidak |
| Analitik | `report.analytics.view` | | tidak |

Izin khusus (aksi berisiko, dicentang terpisah):

| Izin khusus | Kode | Brand |
|---|---|---|
| Verifikasi pembayaran | `payment.verify` | ya |
| Void pembayaran | `payment.void` | ya |
| Setujui pengajuan agen | `agen.approve` | ya |
| Setujui perubahan data jamaah | `jamaah.perubahan.approve` | ya |
| Diskon besar | `booking.diskon.besar` | tidak |
| Cairkan komisi | `agen.payout.approve` | tidak |
| Ganti kaitan agen | `agen.kaitan.edit` | tidak |
| Terbitkan konten | `cms.publish`, `cms.blog.publish`, `cms.schema.approve`, `cms.tracking.manage` | tidak |

Kolom Brand menandai izin yang boleh dicentang untuk pengguna terikat brand (A5): sama dengan izin yang dimiliki role Admin Brand dan Finance sebelum spek ini. Backend menegakkan daftar yang sama di `internal/akses`.

Templat pengisi cepat (di kode, tidak mengikat, tidak tersimpan pada pengguna): **Admin Brand**, **Keuangan**, **Konten**.

## 5. API

Prefix `/api/admin`, semua di bawah `RequireHoldingPermission("user.admin")` kecuali disebut lain.

| Method | Path | Catatan |
|---|---|---|
| GET | `/users` | Tiap pengguna memuat `akses` dan `permissions`. |
| POST | `/users` | Body: `display_name`, `email`, `no_hp`, `brand_id`, `akses`, `permissions`, `password`. |
| GET, PUT | `/users/{id}` | PUT memakai body yang sama tanpa `password`. |
| PUT | `/users/{id}/status` | Tidak berubah. |
| PUT | `/users/{id}/password` | Tidak berubah. |
| DELETE | `/users/{id}` | Tidak berubah. |
| PUT | `/account/password` | Cukup login (A9). |

Dihapus: `/roles`, `/permissions`, `/users/{id}/roles`, `/org-units/*`.

`akses` bernilai `super_admin` atau `tim`. Untuk `super_admin`, `permissions` diabaikan saat simpan dan dikembalikan kosong.

`GET /api/admin/me` dan respons login/refresh mempertahankan nama field (kontrak dengan azhan-booking): `role` berisi `super_admin` atau `tim`, `roles` berisi satu elemen yang sama, `permissions` berisi kode izin (Super Admin: semua kode).

Middleware `RequireAdminRole` dan `RequireAdminOrCRMAccess` (gerbang khusus akun CS) dihapus. Fitur CS/CRM belum ada; `POST /crm/deals` tetap dijaga `crmdeal.process` yang hanya dimiliki Super Admin sampai sambungan CRM dibangun.

## 6. UI

- Daftar Pengguna: kolom Akses berisi "Super Admin" atau "Tim".
- Form Pengguna: Profil, Akses (Holding atau brand; jenis akses), lalu daftar centang §4 bila Tim. Baris dan izin yang tidak boleh dipilih untuk pengguna brand disembunyikan.
- Menu dan halaman Struktur Organisasi dihapus.

## 7. Di luar cakupan

Halaman kelola templat atau role, riwayat perubahan izin di UI, akses CS/CRM, scope cabang, dan struktur organisasi.
