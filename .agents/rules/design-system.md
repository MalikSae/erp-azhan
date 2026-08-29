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

Basis warna brand: **Warm Amber Gold** (`#FED853`) dengan aksen UI modern **Altezza Travel**: Dark Charcoal Sidebar (`#181C1F`) + Warm Amber Gold Active Pill (`#FED853`).

### 2.1 Token Sidebar & Navigasi Gelap (Altezza Style)
| Token | Hex | Pemakaian |
|---|---|---|
| `sidebar.bg` | `#181C1F` | Background sidebar utama |
| `sidebar.surface` | `#22272B` | Permukaan card/grup dalam sidebar |
| `sidebar.hover` | `#292F34` | Hover item menu sidebar |
| `sidebar.border` | `#2A3036` | Border pemisah pada sidebar gelap |
| `sidebar.active` | `#FED853` | **Active Nav Pill** (Warm Amber Gold) |
| `sidebar.activeText` | `#14171A` | Teks di atas active gold pill |
| `sidebar.muted` | `#8C95A0` | Teks/label grup menu yang tidak aktif |

### 2.2 Token Brand & Status
| Token | Hex | Pemakaian |
|---|---|---|
| `primary-50` / `primary-soft` | `#FEFDF0` | Background highlight lembut / tint putih-krem |
| `primary-100` / `accent-gold-light` | `#FEF7D6` | Background badge emas muda / soft highlight |
| `primary-500` / `accent-gold` | `#FED853` | **Basis Utama Brand ERP, Primary Button, & Active Pill** |
| `primary-600` / `accent-gold-hover` | `#F5CD3E` | Hover state tombol primary emas |
| `brand-dark` | `#14171A` | Teks kontras di atas tombol/pill emas |
| `neutral-50` | `#FAFAFA` | Background card terang |
| `neutral-100` | `#F4F4F5` | Background badge netral / placeholder |
| `neutral-200` | `#E4E4E7` | Border card & form |
| `neutral-500` | `#71717A` | Teks sekunder / caption |
| `neutral-900` | `#18181B` | Teks utama |
| `page.bg` | `#F6F8FA` | Background halaman dashboard (soft slate) |
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
| `Button` | `primary`, `secondary`, `danger`, `ghost` × size `sm`, `md` | Semua tombol di seluruh app wajib pakai ini, tidak ada `<button>` mentah dengan class manual |
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