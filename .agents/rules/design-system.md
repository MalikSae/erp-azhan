---
trigger: always_on
---

# Design System — ERP Azhan Grup (Frontend Internal)

> Berlaku untuk Master Dashboard, Travel Dashboard, CRM Dashboard (semua dashboard internal React+Vite+Tailwind). Sumber kebenaran teknis ada di `tailwind.config.js` — dokumen ini adalah penjelasannya dalam bahasa manusia.

---

## 1. Prinsip

**Dilarang keras hardcode UI.** Artinya:
- Tidak ada `bg-[#123456]`, `text-[15px]`, `style={{color: '...'}}`, atau nilai Tailwind arbitrary (`w-[327px]`) di JSX manapun
- Tidak ada warna/spacing/font-size yang ditulis langsung — semua HARUS lewat token di `tailwind.config.js` atau komponen di `src/components/ui/`
- Kalau butuh komponen yang belum ada di design system → **tambahkan dulu ke `components/ui/`**, baru dipakai. Jangan tulis markup styled one-off langsung di halaman

---

## 2. Warna (Color Tokens)

**Master Dashboard (holding Azhan Grup)** memakai palet **Onyx & Champagne Gold di atas gading**: tombol utama onyx, emas hanya aksen (menu aktif, logo, garis tipis), latar gading hangat. Warna keempat brand (Zahara, Nava, Alsha, Hana) hanya dipakai untuk data (grafik per brand), bukan tombol/navigasi.

**Travel Dashboard** mengikuti warna tiap brand saat runtime (`applyBrandTheme` → variabel `--brand-primary*`); token bawaan di `travel-dashboard/tailwind.config.js` hanya fallback.

Aturan kontras: teks/ikon di latar terang memakai `primary-600` ke atas (≥ 4.5:1 vs putih); `primary-50`–`500` hanya untuk latar/isi dengan teks onyx di atasnya.

### 2.1 Token Sidebar & Navigasi Gelap (Master)
| Token | Hex | Pemakaian |
|---|---|---|
| `sidebar.bg` | `#16181B` | Background sidebar utama (onyx) |
| `sidebar.surface` | `#1F2226` | Permukaan card/grup dalam sidebar |
| `sidebar.hover` | `#26292E` | Hover item menu sidebar |
| `sidebar.border` | `#2A2D31` | Border pemisah pada sidebar gelap |
| `sidebar.active` | champagne 14% | Latar menu aktif (tint emas tipis) |
| `sidebar.activeText` | `#D9C392` | Teks menu aktif |
| `sidebar.muted` | `#A8A49C` | Teks/label grup menu yang tidak aktif |

### 2.2 Token Brand & Status (Master)
| Token | Hex | Pemakaian |
|---|---|---|
| `primary-50` | `#FAF7EF` | Latar highlight lembut |
| `primary-100` / `accent-gold-light` | `#F3ECDC` | Latar badge/tint champagne |
| `primary-500` / `accent-gold` | `#C4A062` | **Champagne gold** — aksen, logo, menu aktif |
| `primary-600` | `#8C6D2F` | Teks/ikon emas di latar terang (4.8:1) |
| `primary-700` | `#735826` | Emas tua: teks & link (6.7:1) |
| `brand-dark` | `#16181B` | Onyx — tombol utama, teks kontras |
| `brand-ivory` | `#F7F5F0` | Teks di atas tombol onyx |
| `neutral-*` | skala stone | Netral hangat (`neutral-500` `#78716C` untuk caption) |
| `page.bg` | `#F7F5F0` | Background halaman (gading) |
| `success-*` | skala hijau | Badge "published" / "terverifikasi", pesan sukses |
| `warning-*` | skala kuning-emas | Badge "draft" / "perlu review" / "pending" |
| `danger-*` | skala merah | Error, tombol delete, validasi gagal, batal |

---

## 3. Tipografi

| Token | Font | Pemakaian |
|---|---|---|
| `font-heading` | DM Sans | Judul halaman, judul card, label sidebar |
| `font-body` | DM Sans | Body text, input, tabel, paragraf |

Satu keluarga font (DM Sans) dipakai untuk heading maupun body — dibedakan lewat weight (mis. `font-semibold`/`font-bold` untuk heading, `font-normal`/`font-medium` untuk body), bukan font-family berbeda.

Skala ukuran (dipetakan ke Tailwind text-* standar, tidak ada ukuran custom):
- `text-xs` — label kecil, caption
- `text-sm` — body default di tabel/form
- `text-base` — body default halaman
- `text-lg` — subjudul
- `text-2xl` — judul halaman (PageHeader)

### 3.1 Scaling per breakpoint (WAJIB — lihat AGENTS.md bagian Responsivitas)

Ukuran di atas adalah nilai **desktop (≥768px)**. Di mobile, turunkan 1 step Tailwind untuk elemen besar (judul, PageHeader) supaya proporsional — body text kecil (`text-sm`, `text-xs`) boleh tetap sama di semua breakpoint karena sudah cukup kecil:

| Elemen | Mobile (default, <768px) | Desktop (`md:` ke atas) |
|---|---|---|
| Judul halaman (PageHeader) | `text-xl` | `md:text-2xl` |
| Judul MetaBox/Card | `text-sm` | tetap `text-sm` (sudah kecil) |
| Input judul besar (kalau ada) | `text-xl` | `md:text-2xl lg:text-3xl` |
| Body/label form | `text-sm` | tetap `text-sm` |

Contoh pemakaian di komponen: `className="text-xl md:text-2xl font-heading font-semibold"` — bukan `text-2xl` statis.

---

## 3.5 Breakpoint & Layout Responsive

Pakai breakpoint default Tailwind, TIDAK ada breakpoint custom:

| Breakpoint | Lebar | Pemakaian utama |
|---|---|---|
| (default, mobile) | <640px | Base style — sidebar tersembunyi/drawer, layout 1 kolom, tabel scroll horizontal |
| `sm:` | ≥640px | Jarang dipakai eksplisit, biasanya loncat ke `md:` |
| `md:` | ≥768px | Sidebar mulai fixed-visible, layout 2 kolom (mis. ScheduleFormPage) mulai aktif |
| `lg:` | ≥1024px | Lebar maksimal konten, spacing lebih lega |

**Sidebar**: `hidden md:block` untuk versi fixed desktop + komponen drawer/hamburger terpisah yang HANYA render di mobile (`md:hidden`) — dua rendering berbeda, bukan 1 elemen yang di-resize CSS saja.

**Layout 2 kolom (ScheduleFormPage, dst)**: `flex flex-col md:flex-row` — kolom kanan (sidebar MetaBox) otomatis pindah ke BAWAH kolom kiri saat mobile (`order` CSS boleh dipakai kalau urutan visual perlu beda dari urutan DOM), bukan tetap 2 kolom sempit.

**Table**: wrapper `overflow-x-auto` di sekeliling elemen `<table>` sebagai pendekatan default project ini (bukan card-list) — konsisten dipakai di semua halaman Table.

---

## 4. Spacing, Radius, Shadow

- **Spacing**: pakai skala default Tailwind (4px based) — jangan ciptakan skala custom
- **Radius standar**: `rounded-md` (input, button), `rounded-lg` (card, panel), `rounded-full` (badge/pill)
- **Shadow standar**: `shadow-sm` untuk card, `shadow-md` untuk modal/dropdown mengambang

---

## 5. Komponen Standar (`src/components/ui/`)

| Komponen | Varian | Catatan |
|---|---|---|
| `Button` | `primary`, `secondary`, `danger`, `ghost` × size `sm`, `md` | Semua tombol di seluruh app wajib pakai ini, tidak ada `<button>` mentah dengan class manual. Varian `primary` di `frontend/shared` membaca `--brand-primary`/`--brand-primary-text` (diisi `master-dashboard/src/index.css` dan tema brand travel) |
| `Input` | text, number, date, time | Wrapper konsisten: label + input + pesan error |
| `Select` | — | Dropdown standar, dipakai untuk semua field "pilih dari data master" (hotel, maskapai, itinerary) |
| `Textarea` | — | |
| `Label` | — | Dipakai internal oleh Input/Select, jarang dipanggil langsung |
| `FormField` | — | Wrapper generik: label + children + error message, untuk kasus di luar Input/Select standar |
| `Badge` | `draft` (warning), `published` (success), `archived` (neutral), `promo` (warning) | Status paket & promo SELALU pakai ini, bukan span berwarna manual |
| `Card` | — | Container panel dengan padding & shadow standar |
| `PageHeader` | — | Judul halaman + tombol aksi (mis. "+ Tambah Hotel") di kanan, dipakai di SETIAP halaman CRUD |
| `Table` | — | Header, row, empty state bawaan — dipakai untuk semua list data (hotel, maskapai, itinerary, paket) |
| `Alert` | `error`, `success` | Pesan error/sukses global (mis. gagal simpan) |
| `LoadingSpinner` | — | Dipakai saat fetch data, jangan bikin spinner custom per halaman |
| `EmptyState` | — | Tampilan saat data kosong (mis. "Belum ada hotel"), dipakai di dalam `Table` |

---

## 6. Alur Kerja untuk Halaman Baru

1. Cek dulu apakah komponen yang dibutuhkan sudah ada di `components/ui/`
2. Kalau belum ada, buat komponen barunya DULU di `components/ui/` (ikuti pola/props komponen lain yang sudah ada), baru pakai di halaman
3. Halaman (`pages/*.jsx`) hanya berisi *layout* dan *logic* (fetch data, state) — styling detail ada di komponen `ui/`, bukan di halaman