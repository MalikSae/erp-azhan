---
trigger: always_on
---

# Design System — ERP Azhan Grup (Frontend Internal)

> Berlaku untuk semua dashboard internal (React+Vite+Tailwind). Arah visual: gaya dash.cloudflare.com (hampir monokrom), lihat `DESIGN.md` di root repo (token lengkap, rasio kontras, log keputusan). Sumber kebenaran teknis: `frontend/shared/tailwind-preset.js` dan `frontend/shared/src/corner-radius.css`. Dokumen ini aturan kerjanya.

---

## 1. Prinsip

**Dilarang keras hardcode UI.** Artinya:
- Tidak ada `bg-[#123456]`, `text-[15px]`, `style={{color: '...'}}`, atau nilai Tailwind arbitrary (`w-[327px]`) di JSX manapun
  - Pengecualian nilai arbitrary yang sah (tidak ada padanan di skala Tailwind): tinggi berbasis viewport (`max-h-[70vh]`), `calc(...)`, template grid berkolom tetap (`lg:grid-cols-[minmax(0,1fr)_320px]`), lebar minimum tabel lebar (`min-w-[880px]`), dan posisi presisi titik timeline. Selain itu pakai skala terdekat (`w-8`, `max-w-sm`, `min-w-40`).
  - Inline `style` hanya untuk nilai dari data (warna brand dari database, lebar progress bar) atau posisi hitungan JS (tooltip, menu mengambang).
- Tidak ada warna/spacing/font-size yang ditulis langsung — semua HARUS lewat token di `tailwind.config.js` atau komponen di `src/components/ui/`
- Kalau butuh komponen yang belum ada di design system → **tambahkan dulu ke `components/ui/`**, baru dipakai. Jangan tulis markup styled one-off langsung di halaman

---

## 2. Warna (Color Tokens)

Gaya dash.cloudflare.com: **hampir monokrom**. Latar halaman, sidebar, dan topbar putih; pemisah garis 1px `neutral-200`; teks hitam lembut. Satu-satunya warna yang muncul di layar kerja adalah warna status. Warna brand travel (Zahara, Nava, Alsha, Hana) hanya untuk data (grafik per brand), bukan tombol/navigasi.

**`primary` adalah skala abu-hitam, bukan oranye.** Menu aktif latar abu, fokus abu gelap. Oranye Cloudflare ada di token `accent` dan **hanya** untuk tombol primer (gradien `bg-btn-primary` + `shadow-btn-primary`, lewat komponen `Button`), logo, banner promo, dan seri grafik utama. Teks putih di oranye tombol hanya 2.5–2.9:1 (gagal AA, diterima demi meniru Cloudflare, lihat DESIGN.md Decisions); di luar komponen `Button` jangan taruh teks putih di `accent-*`. Teks oranye di latar putih memakai `accent-700`.

### 2.1 Token Sidebar
| Token | Hex | Pemakaian |
|---|---|---|
| `sidebar.bg` | `#FFFFFF` | Background sidebar (putih, border kanan) |
| `sidebar.hover` | `#F5F5F5` | Hover item menu |
| `sidebar.border` | `#E5E5E5` | Border pemisah |
| `sidebar.active` | `#F0F0F0` | Latar menu aktif (tanpa garis warna) |
| `sidebar.activeText` | `#171717` | Teks menu aktif (weight 500) |
| `sidebar.muted` | `#404040` | Teks menu tidak aktif |
| `sidebar.label` | `#737373` | Label grup menu (`text-sm`, bukan kapital) |

### 2.2 Token Warna & Status
| Token | Hex | Pemakaian |
|---|---|---|
| `primary-50` / `100` | `#F7F7F7` / `#F0F0F0` | Baris terpilih, latar aktif |
| `primary-500` | `#404040` | Ikon aktif, cincin fokus, border field fokus |
| `primary-600` | `#262626` | Tombol gelap (`variant="dark"`), teks putih 15:1 |
| `primary-700` | `#171717` | Hover tombol primer, teks link (bergaris bawah) |
| `accent-400` → `accent-500` | `#FF8130` → `#FF6A00` | Gradien tombol primer (border `accent-600` `#E25A00`); juga logo, promo, grafik |
| `neutral-*` | abu netral murni | `800` teks utama, `600` teks sekunder, `500` label & caption, `200` garis |
| `page.bg` | `#FFFFFF` | Background halaman |
| `success-*` | skala hijau | Badge "published" / "terverifikasi", pesan sukses |
| `warning-*` | skala kuning | Badge "draft" / "perlu review" / "pending" |
| `danger-*` | skala merah | Error, tombol delete, validasi gagal, batal |

---

## 3. Tipografi

Dua font, peran tetap. Dimuat dari Google Fonts di `master-dashboard/index.html`.

| Token | Font | Pemakaian |
|---|---|---|
| `font-heading` | **Inter** (400–700) | Judul halaman, judul section/kartu, angka statistik, tombol, badge |
| `font-body` | **Roboto** (400–700) | Teks isi: body, tabel, field, label form, menu sidebar, caption, tooltip grafik. Default `body`. |
| `font-mono` | ui-monospace | Kode booking, nomor invoice, kode referral (`font-mono text-xs`) |

Aturan:
- Hierarki dari **ukuran dan warna abu**, bukan dari tebal huruf. Weight maksimal `font-semibold` (600); **jangan** `font-bold`/`font-extrabold`.
- Angka di tabel, nominal rupiah, dan stat tile memakai `tabular-nums`.
- Header tabel dan label sentence case, bukan huruf kapital semua (`uppercase` dilarang kecuali kode hex/inisial avatar).
- Hanya skala Tailwind standar. Tidak ada `text-[13px]` atau ukuran arbitrary lain.

| Elemen | Kelas |
|---|---|
| Judul halaman (`PageHeader`) | `font-heading text-lg md:text-xl font-semibold tracking-tight` |
| Judul section / kartu ("Analitik") | `font-heading text-base md:text-lg font-semibold` |
| Judul blok daftar ("Keberangkatan terdekat 50 >") | `font-body text-sm text-neutral-500` |
| Angka statistik | `font-heading text-2xl md:text-3xl font-semibold tracking-tight tabular-nums` |
| Body, tabel, field, item menu | `font-body text-sm` (item menu aktif `font-medium`) |
| Label grup sidebar | `font-body text-sm text-sidebar-label` |
| Header tabel (`<thead>` di `Table`) | `font-body text-xs font-medium text-neutral-500` (`<th>` mewarisi weight dari `<thead>`, lihat `index.css`) |
| Nama baris yang bisa diklik di tabel | `font-body font-medium text-neutral-900 hover:text-primary-600` — isi, bukan judul; jangan `font-heading` |
| Caption, keterangan | `font-body text-xs text-neutral-500` |
| Tombol, badge | `font-heading font-medium` (lewat komponen `Button`/`Badge`) |
| Link teks ("Lihat bukti", "Pilih semua", "Tampilkan semua") | `font-medium text-neutral-900 underline underline-offset-2 decoration-neutral-300 hover:decoration-neutral-900` — garis bawah wajib karena `primary` monokrom; jangan `text-primary-*` untuk link. Nama baris di tabel tetap tanpa garis bawah. |

### 3.1 Scaling per breakpoint (WAJIB — lihat AGENTS.md bagian Responsivitas)

Elemen besar turun satu step di mobile (<768px); teks `text-sm`/`text-xs` sama di semua breakpoint:

| Elemen | Mobile (default) | Desktop (`md:` ke atas) |
|---|---|---|
| Judul halaman | `text-lg` | `md:text-xl` |
| Judul section | `text-base` | `md:text-lg` |
| Angka statistik | `text-2xl` | `md:text-3xl` |
| Body/label form | `text-sm` | tetap `text-sm` |

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

**Layout form tambah/edit (STANDAR — acuan: Tambah/Edit Paket)**: semua form tambah/edit memakai `shared/ui/FormLayout` (`FormLayout`, `FormSection`, `SidebarSection`, `FormAside`), bukan `MetaBox`/grid manual.

- **Kolom kiri**: SATU box putih (`rounded-xl border`), isinya `FormSection` yang dipisah garis tipis. Judul section di atas isi (`text-base font-bold`), tanpa header band, tanpa subtitle, tanpa ikon dekoratif, tanpa kotak bersarang di dalamnya.
- **Kolom kanan (320 px, sticky)**: SATU box berisi `SidebarSection`. Bagian pertama selalu aksi simpan (tombol primer selebar penuh; TIDAK ada tombol "Batal" — kembali lewat panah di header). Sisanya data pendukung (status, brand/kategori, ringkasan, berkas).
- **Tab navigasi**: opsional; pakai prop `tabs` bila form punya ≥ 3 section. Tab mengikuti scroll (scroll-spy) dan klik meluncur ke section.
- **Responsif**: di bawah `lg` sidebar turun di bawah kolom utama.
- **Field**: komponen `Input`/`CustomDropdown`/`AutocompleteInput`; label di atas; tinggi seragam `h-9` (36px), sama dengan `Button` md dan `ActionMenu` supaya baris toolbar rata; teks field `text-sm`, border `neutral-300`; field wajib bertanda `*`; info baca-saja (mis. terisi/sisa) berupa teks keterangan kecil di bawah field, BUKAN field yang tampak bisa diisi. Lebar field mengikuti kolom grid 2 kolom (atau penuh untuk teks panjang); jangan ada field setengah-lebar sendirian dengan ruang kosong di sampingnya.
- **Tombol tambah baris** (transit, hotel transit, aktivitas, hari): tombol putus-putus selebar penuh "+ Tambah …"; baris dihapus lewat tombol sampah di kanan baris.
- **Warna**: hanya abu-hitam dan warna status (oranye `accent` hanya tombol primer, logo, promo, grafik); hindari biru/hijau dekoratif. Status/atribut memakai `Badge`.
- **Setelah simpan**: pindah ke halaman daftar dengan `navigate(path, { state: { successMessage } })`; halaman daftar menampilkan `Alert variant="success"` dan membersihkan state.
- **Scroll**: pindah halaman mereset scroll area utama (`DashboardLayout`).

**Jarak vertikal halaman**: akar setiap halaman `space-y-6` (24px) dan itulah satu-satunya sumber jarak antar blok, termasuk dari `PageHeader` ke konten pertama. `PageHeader` dan komponen blok lain **tidak** boleh punya `mb-*`/`mt-*` sendiri: margin itu melebur dengan elemen block tapi tidak dengan elemen `inline-flex` (mis. `SegmentedTabs`), sehingga jarak jadi beda-beda antar halaman.

**Table**: wrapper `overflow-x-auto` di sekeliling elemen `<table>` sebagai pendekatan default project ini (bukan card-list) — konsisten dipakai di semua halaman Table.

---

## 4. Spacing, Radius, Shadow

- **Spacing**: pakai skala default Tailwind (4px based) — jangan ciptakan skala custom
- **Radius** (dikunci di `corner-radius.css`): `rounded-sm` 4px (badge, chip), `rounded-md`/`lg`/`xl` 6px (tombol, field, item menu), `rounded-2xl`/`3xl` 8px (kartu, tabel, modal, popover, dropdown). `rounded-full` hanya avatar, titik status, dan toggle — bukan badge.
- **Shadow**: kartu, tabel, dan sidebar **datar** (`shadow-card` = `none`); batasnya border 1px `neutral-200`. Lapisan mengambang memakai `shadow-popover` (dropdown, popover, menu) atau `shadow-modal` (modal). Jangan pakai `shadow-sm`/`shadow-lg`/`shadow-xl` bawaan.
- **Gerak**: hover/fokus 100ms; buka dropdown/drawer/modal 200ms dengan `ease-cf`. Tanpa animasi dekoratif.

---

## 5. Komponen Standar (`src/components/ui/`)

| Komponen | Varian | Catatan |
|---|---|---|
| `Button` | `primary`, `secondary`, `dark`, `danger`, `danger-light`, `danger-outline` (hapus baris), `dashed` (+ Tambah …), `ghost` × size `sm`, `md`, `lg` | Semua tombol **aksi** (punya teks perintah atau tampil sebagai tombol berbingkai) wajib pakai ini. `<button>` mentah hanya boleh untuk: nama baris yang bisa diklik di tabel, link teks ("Lihat bukti", "Pilih semua"), tombol ikon tanpa bingkai, item menu/dropdown, tab/segmen pilihan, dan kartu yang bisa diklik. Tombol ikon berbingkai memakai ukuran `h-9 w-9`. |
| `Input` | text, number, date, time | Wrapper konsisten: label + input + pesan error |
| `Select` | — | Dropdown standar, dipakai untuk semua field "pilih dari data master" (hotel, maskapai, itinerary) |
| `Textarea` | — | |
| `Label` | — | Dipakai internal oleh Input/Select, jarang dipanggil langsung |
| `FormField` | — | Wrapper generik: label + children + error message, untuk kasus di luar Input/Select standar |
| `Badge` | nada `success`, `warning`, `danger`, `info`, `neutral`, `promo` | **Satu-satunya gaya status** — lihat §5.2. Juga dipakai untuk atribut (brand, kategori) dengan `PILL_STRONG`. |
| `Card` | — | Container panel dengan padding & shadow standar |
| `PageHeader` | — | Judul halaman + tombol aksi (mis. "+ Tambah Hotel") di kanan, dipakai di SETIAP halaman CRUD |
| `Table` | — | Header, row, empty state bawaan — dipakai untuk semua list data (hotel, maskapai, itinerary, paket) |
| `Alert` | `error`, `success` | Pesan error/sukses global (mis. gagal simpan) |
| `LoadingSpinner` | — | Dipakai saat fetch data, jangan bikin spinner custom per halaman |
| `EmptyState` | — | Tampilan saat data kosong (mis. "Belum ada hotel"), dipakai di dalam `Table` |

---

## 5.1 Aturan Tabel (WAJIB)

**Tabel tidak boleh dibungkus dalam kontainer/box lain.** Kartu putih berbingkai *adalah* tabelnya sendiri.

- **Judul** tabel (dan tombol aksinya, mis. "+ Tambah") berada **di luar** kartu, di atasnya. Jangan taruh judul di dalam kartu yang juga berisi tabel.
- **Dilarang** menaruh `Table`/`DataTable` di dalam `Section`/`Surface`/kartu yang sudah punya padding atau bingkai. Tidak ada "kotak di dalam kotak".
- Di halaman **detail**, pakai `TableCard` (`shared/ui/DetailLayout`): `<TableCard title="…" aside={…}>…<Table/>…</TableCard>`. Keadaan kosong memakai teks biasa dengan padding sendiri (`p-5`), bukan kotak putus-putus.
- Di halaman **daftar**, pakai `DataTable`; pencarian/filter/tab ada di toolbar kartu itu sendiri (`toolbarLeading`, `toolbarActions`, `toolbarPanel`).
- Tabel lebar boleh dibungkus `overflow-x-auto` (bukan kotak berbingkai).
- Pengecualian: kartu **widget dashboard** (judul + ringkasan + tabel mini dalam satu kartu).
**Konfirmasi & pesan**: jangan `window.confirm`, `window.alert`, atau `window.prompt`. Konfirmasi aksi berisiko memakai `Modal` (`size="sm"`, tombol `secondary` "Batal" lalu `danger`/`primary` aksi di footer kanan); pesan gagal tampil sebagai `Alert variant="error"` di dalam modal atau di atas halaman.


## 5.2 Aturan Status (WAJIB)

Semua status di semua halaman tampil dengan **satu gaya**: komponen `Badge` (kotak tint tipis, radius 4px, `text-xs font-medium`, tanpa ikon, tanpa titik). Tidak ada lagi "titik + teks", pil bertitik, atau teks berwarna sebagai status.

Halaman cukup memetakan status ke **nada** `Badge`:

| Nada | Arti | Contoh |
|---|---|---|
| `success` | selesai / baik / aktif | Published, Aktif, Lunas, Terkonfirmasi, Disetujui, Tayang, Selesai |
| `warning` | perlu tindakan / menunggu / setengah jalan | Draft, DP, Menunggu, Mengajukan, Akan tayang, Transit |
| `danger` | gagal / dibatalkan | Batal, Ditolak |
| `info` | informasi berwarna tanpa makna baik-buruk | Baru, Direct, Langsung tersimpan |
| `neutral` | tidak aktif / selesai tanpa makna | Archived, Nonaktif, Berakhir, Dibatalkan (void) |
| `promo` | label promo | Promo |

Larangan:
- Jangan menimpa warna `Badge` lewat `className` (`!bg-…`, `!text-…`). Kalau nada tidak cocok, pilih nada lain.
- Jangan membuat `<span>` status sendiri. Tabel status lama yang memakai kelas titik (`dot: 'bg-success-500'`) boleh diubah lewat `toneFromClass(dot)` dari `Badge.jsx`.
- Status Aktif/Nonaktif memakai `StatusChip` (`penggunaUi`), yang kini merender `Badge`.

## 5.3 Kartu, Empty State, Tab (WAJIB)

**Kartu**
- Kartu = `bg-white border border-neutral-200 rounded-2xl` (8px), tanpa bayangan. Pakai `Surface` + `Section` (DetailLayout) atau `Card`; jangan tulis ulang kelasnya di halaman.
- Kartu berjudul lama `MetaBox` kini tampil sama dengan `Surface` + `Section` (judul di atas isi, tanpa pita abu, tanpa kotak ikon). Halaman baru pakai `Surface` + `Section`.
- Kotak info di dalam modal/kartu (ringkasan, catatan) boleh `rounded-xl` (6px) `bg-neutral-50 border border-neutral-200`.

**Empty state** — komponen `EmptyState`, tiga varian:
| Varian | Kapan | Tampilan |
|---|---|---|
| `list` (default) | tabel/daftar kosong (otomatis di `Table`/`DataTable`) | teks di tengah, `py-12` |
| `section` | satu bagian kartu kosong ("Belum ada pembayaran") | teks rata kiri, `px-5 py-5` |
| `inline` | di dalam wadah yang sudah berpadding (`Section`, `MetaBox`) | teks saja |

Pesan berupa kalimat biasa `text-sm text-neutral-500`, boleh disertai petunjuk aksi. Tanpa ilustrasi, ikon, atau kotak putus-putus. Pesan mikro di dalam baris/dropdown ("Tidak ada pilihan") cukup `text-xs text-neutral-500`.

**Tab** — semua tab pindah daftar memakai `SegmentedTabs`. Jangan membuat `role="tablist"` sendiri.

## 5.4 Header Halaman & Jarak (WAJIB)

**Header** — semua halaman memakai `PageHeader`; tidak ada header buatan sendiri (judul `<h1>` di dalam kartu, teks "← Kembali" di atas kartu).

| Jenis halaman | Pola |
|---|---|
| Daftar | `title` + breadcrumb (otomatis) + tombol aksi utama di kanan (`actionLabel`) |
| Detail | `onBack` (ikon ←, breadcrumb otomatis mati) + `title` = nama/ID dengan `badges` sebaris di kanannya + `meta` di baris kedua + tombol aksi di kanan (`children`). Maksimal dua baris. |
| Form tambah/edit | `onBack` + `title`; tombol simpan ada di `FormLayout`, bukan di header |

- `meta`: array teks pendek (ID/kode, WA, email, brand, tanggal dibuat). Dirender sebagai teks abu `text-sm` dipisah "·". Tanpa pil, tanpa ikon.
- `badges`: status (`Badge`) dan atribut penting dengan tautannya, mis. `<Badge>Upline: <Link>…</Link></Badge>` + tautan "Ganti". Tampil sebaris dengan judul (jarak 12px), turun baris hanya bila layar sempit.
- Tombol aksi selalu di kanan header (Ubah, Edit Data, Cetak / PDF, Nonaktifkan, menu "Aksi"), tidak di dalam kartu. Tanpa foto/avatar di header; foto masuk kartu profil.

**Jarak (semua nilai skala Tailwind, kelipatan 4px)**

| Tempat | Nilai |
|---|---|
| Area konten (`DashboardLayout`) | `px-4 md:px-8`, atas `pt-6 md:pt-8`, bawah `pb-6 md:pb-8` — halaman tidak menambah `pt-*`/`pb-*` sendiri |
| Antar blok halaman (header, tab, kartu, tabel) | akar halaman `space-y-6` (24px); tanpa `mb-*`/`mt-*` di komponen blok (lihat "Jarak vertikal halaman") |
| Grid kartu (kolom kiri/kanan) | `gap-6` (24px); baris stat tile `gap-4` |
| Header: judul + badges → meta | badge sebaris dengan judul (`gap-x-3`), meta `mt-1` (4px), diatur `PageHeader` |
| Padding isi kartu (`Card`, `Section`, `FormSection`, `SidebarSection`, `StatTile`, `MetaBox`, isi `Modal`) | `p-5` (20px), sama di semua breakpoint |
| Judul section → isi | `mb-4` (16px) |
| Antar field / baris data dalam kartu | `space-y-4` atau `gap-y-4` (16px); label → nilai `mt-0.5` |
| Pita toolbar & paginasi `DataTable`, header/footer `Modal` | `p-4` / `px-5 py-4` |
| Sel tabel | `px-4 py-3` |

## 6. Alur Kerja untuk Halaman Baru

1. Cek dulu apakah komponen yang dibutuhkan sudah ada di `components/ui/`
2. Kalau belum ada, buat komponen barunya DULU di `components/ui/` (ikuti pola/props komponen lain yang sudah ada), baru pakai di halaman
3. Halaman (`pages/*.jsx`) hanya berisi *layout* dan *logic* (fetch data, state) — styling detail ada di komponen `ui/`, bukan di halaman