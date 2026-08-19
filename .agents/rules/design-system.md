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

Basis warna: `#CC904A` (amber/bronze hangat).

| Token | Hex | Pemakaian |
|---|---|---|
| `primary-50` | `#FAF2E8` | Background highlight lembut |
| `primary-100` | `#F3E0C9` | |
| `primary-200` | `#E6C29C` | |
| `primary-300` | `#DAA372` | |
| `primary-400` | `#D2954F` | |
| `primary-500` | `#CC904A` | Basis (referensi asli) |
| `primary-600` | `#B87A3A` | **Aksen utama** — tombol, active nav, link |
| `primary-700` | `#96622F` | Hover state |
| `primary-800` | `#744C25` | |
| `primary-900` | `#52371A` | |
| `neutral-50` | `#FAFAFA` | Background halaman |
| `neutral-100` | `#F4F4F5` | |
| `neutral-200` | `#E4E4E7` | Border |
| `neutral-300` | `#D4D4D8` | |
| `neutral-400` | `#A1A1AA` | |
| `neutral-500` | `#71717A` | Teks sekunder |
| `neutral-600` | `#52525B` | |
| `neutral-700` | `#3F3F46` | |
| `neutral-800` | `#27272A` | |
| `neutral-900` | `#18181B` | Teks utama (hitam lembut, bukan `#000000` murni — lebih nyaman dibaca) |
| `pure-white` | `#FFFFFF` | Anchor absolut — background card, teks di atas warna gelap |
| `pure-black` | `#000000` | Anchor absolut — dipakai sangat jarang, BUKAN default teks |
| `success-*` | skala hijau | Badge status "published", pesan sukses |
| `warning-*` | skala **kuning-emas** (hue ~45°, mis. `warning-600: #D97706`) | Badge "draft"/"promo" — SENGAJA digeser dari amber murni supaya tidak mirip dengan `primary` |
| `danger-*` | skala merah | Error, tombol delete, validasi gagal |

Semua didefinisikan sebagai scale 50-900 di `tailwind.config.js theme.extend.colors` — JANGAN pakai warna Tailwind bawaan langsung di komponen, selalu lewat nama semantik (`primary-600`, `success-600`, dst).

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