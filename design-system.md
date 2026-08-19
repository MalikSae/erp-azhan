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

Tema warna Emas/Coklat (Tema Kiswah), konsisten dengan desain premium dan elegan.

| Token | Hex (contoh) | Pemakaian |
|---|---|---|
| `primary-50` s.d. `primary-900` | skala emas | `primary-600` = warna aksen utama (tombol, active nav, link). `primary-700` = hover state. `primary-50` = background highlight lembut |
| `neutral-50` s.d. `neutral-900` | skala abu | Teks, border, background netral. `neutral-900` = teks utama, `neutral-500` = teks sekunder, `neutral-200` = border |
| `success-*` | hijau | Badge status "published", pesan sukses |
| `warning-*` | oranye/amber | Badge "draft", promo |
| `danger-*` | merah | Error, tombol delete, validasi gagal |

Semua didefinisikan sebagai scale 50-900 di `tailwind.config.js theme.extend.colors` — JANGAN pakai `yellow-600` atau warna bawaan Tailwind langsung di komponen, selalu lewat nama semantik (`primary-600`, `success-600`, dst) supaya kalau brand color berubah nanti, cukup ubah di 1 tempat.

---

## 3. Tipografi

| Token | Font | Pemakaian |
|---|---|---|
| `font-heading` | DM Sans | Judul halaman, judul card, label sidebar |
| `font-body` | DM Sans | Body text, input, tabel, paragraf |

Skala ukuran (dipetakan ke Tailwind text-* standar, tidak ada ukuran custom):
- `text-xs` — label kecil, caption
- `text-sm` — body default di tabel/form
- `text-base` — body default halaman
- `text-lg` — subjudul
- `text-2xl` — judul halaman (PageHeader)

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
| `Select` | — | Dropdown HTML standar |
| `CustomDropdown` | — | Dropdown kustom dengan styling yang lebih kaya |
| `Textarea` | — | |
| `Label` | — | Dipakai internal oleh form input, jarang dipanggil langsung |
| `FormField` | — | Wrapper generik: label + children + error message |
| `Badge` | `draft` (warning), `published` (success), `archived` (neutral), `promo` (primary) | Status paket & promo SELALU pakai ini |
| `Pill` | `default`, `primary`, `success` | Tag filter atau kategori, bisa dihapus (`onRemove`) |
| `Card` | — | Container panel dengan padding & shadow standar |
| `PageHeader` | - | Judul halaman + tombol aksi di kanan (bisa tambah `onBack` untuk memunculkan tombol kembali) |
| `Table` | — | Komponen tabel dasar |
| `DataTable` | — | Tabel yang lebih advanced (Header, row, empty state bawaan) dengan prop columns & data |
| `Alert` | `error`, `success`, `warning`, `info` | Pesan notifikasi global |
| `LoadingSpinner` | — | Dipakai saat fetch data, jangan bikin spinner custom per halaman |
| `EmptyState` | — | Tampilan saat data kosong, biasa dipakai di dalam tabel atau list |
| `Modal` | ✅ | Dialog/Popup konfirmasi dengan header, body, dan footer standar. Mendukung prop `size` ("sm", "md", "lg", "xl", "2xl") default "md". |
| `MetaBox` | — | Kontainer mirip WordPress meta box, memiliki header abu-abu kecil dan konten. Dipakai di layout form kompleks. |

---

## 6. Alur Kerja untuk Halaman Baru

1. Cek dulu apakah komponen yang dibutuhkan sudah ada di `components/ui/`
2. Kalau belum ada, buat komponen barunya DULU di `components/ui/` (ikuti pola/props komponen lain yang sudah ada), baru pakai di halaman
3. Halaman (`pages/*.jsx`) hanya berisi *layout* dan *logic* (fetch data, state) — styling detail ada di komponen `ui/`, bukan di halaman
