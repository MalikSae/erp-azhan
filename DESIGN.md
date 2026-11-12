---
version: alpha
name: ERP Azhan Dashboard
description: "Dashboard internal ERP Azhan Grup (master-dashboard + frontend/shared), meniru dash.cloudflare.com: hampir monokrom, latar putih, garis tipis, oranye hanya untuk tombol primer dan logo; Inter untuk judul, Roboto untuk isi."
reference: "dash.cloudflare.com (Account home, Nov 2026), diamati dari screenshot; DesignMD `cloudflare` hanya dipakai untuk nilai oranye merek"

colors:
  # primary sengaja skala abu-hitam: semua kelas primary-* lama ikut monokrom.
  primary-50: "#F7F7F7"    # latar baris terpilih
  primary-100: "#F0F0F0"   # latar menu aktif
  primary-200: "#E3E3E3"
  primary-300: "#CFCFCF"
  primary-400: "#9E9E9E"
  primary-500: "#404040"   # ikon aktif, cincin fokus, border field fokus
  primary-600: "#262626"   # tombol gelap (variant dark), teks putih 15.1:1
  primary-700: "#171717"   # teks link (bergaris bawah)
  on-primary: "#FFFFFF"
  accent-400: "#FF8130"    # atas gradien tombol primer (diukur dari "Watch now")
  accent-500: "#FF6A00"    # bawah gradien tombol primer, logo, promo, grafik
  accent-600: "#E25A00"    # border tombol primer
  accent-700: "#C93F08"    # teks oranye di latar putih (5.00:1)
  neutral-0: "#FFFFFF"     # latar halaman, sidebar, topbar, kartu
  neutral-50: "#FAFAFA"    # header tabel, hover baris
  neutral-100: "#F5F5F5"   # hover menu
  neutral-200: "#E5E5E5"   # garis pemisah, border kartu
  neutral-300: "#D4D4D4"   # border field
  neutral-400: "#A3A3A3"   # placeholder, ikon nonaktif
  neutral-500: "#737373"   # label grup sidebar, judul blok, caption (4.74:1)
  neutral-600: "#595959"   # teks sekunder
  neutral-800: "#262626"   # teks utama
  neutral-900: "#171717"   # judul, teks menu aktif
  page-bg: "#FFFFFF"
  success-600: "#16A34A"
  warning-600: "#D97706"
  danger-600: "#DC2626"

typography:
  heading-family: "Inter, Roboto, system-ui, sans-serif"   # font-heading: judul, angka statistik, tombol, badge
  body-family: "Roboto, system-ui, sans-serif"             # font-body: teks isi, tabel, field, menu (default body)
  mono: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace"
  max-weight: 600   # font-semibold; font-bold tidak dipakai
  page-title: { family: heading, size: "18px → md 20px", weight: 600, tailwind: "font-heading text-lg md:text-xl font-semibold tracking-tight" }
  section-title: { family: heading, size: "16px → md 18px", weight: 600, tailwind: "font-heading text-base md:text-lg font-semibold" }
  block-title: { family: body, size: 14px, weight: 400, tailwind: "font-body text-sm text-neutral-500" }  # "Domains 7 >"
  nav-item: { family: body, size: 14px, weight: 400, active-weight: 500, tailwind: "font-body text-sm" }
  nav-group-label: { family: body, size: 14px, weight: 400, tailwind: "font-body text-sm text-sidebar-label" }
  body: { family: body, size: 14px, weight: 400, tailwind: "font-body text-sm" }
  caption: { family: body, size: 12px, weight: 400, tailwind: "font-body text-xs text-neutral-500" }
  table-header: { family: body, size: 12px, weight: 500, tailwind: "font-body text-xs font-medium text-neutral-500" }
  table-row-name: { family: body, size: 14px, weight: 500, tailwind: "font-body font-medium text-neutral-900" }
  stat-value: { family: heading, size: "24px → md 30px", weight: 600, tailwind: "font-heading text-2xl md:text-3xl font-semibold tracking-tight tabular-nums" }
  control: { family: heading, size: 14px, weight: 500, tailwind: "font-heading text-sm font-medium" }  # Button, Badge

spacing:
  base: 4px
  sidebar-width: 224px   # w-56; nama menu tanpa kata "Kelola" (mis. "Jamaah", bukan "Kelola Jamaah")
  topbar-height: 64px
  nav-item-height: 36px
  control-height: 36px     # h-9: tombol md, field, dropdown, kolom cari, ActionMenu — wajib sama
  list-row-height: 48px

radius:
  sm: 4px      # badge ("New"), chip
  md: 6px      # tombol, field, item menu
  card: 8px    # kartu, tabel, kolom cari besar, modal, popover (rounded-2xl / 3xl)
  full: 9999px # avatar, titik status, toggle

shadows:
  none: "none"   # kartu, tabel, sidebar, topbar
  popover: "0 4px 12px rgba(0, 0, 0, 0.08), 0 1px 2px rgba(0, 0, 0, 0.04)"
  modal: "0 16px 48px rgba(0, 0, 0, 0.12), 0 2px 8px rgba(0, 0, 0, 0.04)"

focus:
  ring: "2px solid #404040"
  offset: 2px

motion:
  duration-fast: 100ms
  duration-base: 200ms
  easing: "cubic-bezier(0.19, 1, 0.22, 1)"
---

# DESIGN.md — Dashboard ERP Azhan (gaya dash.cloudflare.com)

## Status

Revisi 2026-11-11. Versi pertama meniru cloudflare.com (situs marketing, tombol oranye). Revisi ini meniru konsol dash.cloudflare.com, yang hampir monokrom.

Diterapkan ke `frontend/shared/tailwind-preset.js`, `frontend/shared/src/corner-radius.css`, `field-interactions.css`, komponen `shared/src/components/ui`, serta Sidebar, Topbar, dan Login di master-dashboard.
Sumber kebenaran teknis tetap preset dan `corner-radius.css`. Jangan tambah palet di config tiap app.

DashboardHome sudah dirombak ke pola beranda Cloudflare (kolom daftar bergaris + blok Analitik). Pembersihan sisa SmartHR sudah dijalankan: ukuran teks arbitrary diganti skala Tailwind (10–12px jadi `text-xs`, 13px jadi `text-sm`, 15px jadi `text-base`), bayangan bawaan dibuang, border transparan dibuat penuh, label kapital dihapus, hex abu diganti nilai netral. Pengecualian: `DetailAkunPage.jsx` (sedang dikerjakan di branch lain) dan template cetak di `utils/print*.js`.

Tombol aksi mentah (hapus baris, "+ Tambah …", "Cetak / PDF", "Lihat Rincian") sudah dipindah ke `Button`; sisa `<button>` mentah adalah link teks, nama baris, tombol ikon, menu, tab, dan kartu yang sah.

Belum diubah: Pola "daftar bergaris dengan chevron" baru ada di DashboardHome, belum jadi komponen shared.

## Rationale

Konsol Cloudflare dipakai untuk kerja, bukan untuk menjual. Hampir tidak ada warna: putih, abu, hitam. Struktur dibangun dari garis tipis, jarak, dan ukuran teks.
ERP Azhan punya pola kerja yang sama: staf memindai daftar booking, pembayaran, jamaah, dan komisi berjam-jam.
Tanpa warna dekoratif, satu-satunya warna yang muncul adalah warna status (lunas, pending, batal). Status jadi langsung terlihat.

## 1. Suasana Visual

- Latar halaman, sidebar, dan topbar sama-sama putih. Pemisahnya garis 1px `neutral-200`.
- Teks hitam lembut. Label dan judul blok abu `neutral-500`.
- Oranye hanya di tombol primer, logo, banner promo, dan seri grafik utama. Tidak pernah di menu, tab, atau link.
- Warna brand travel (Hana, Alsha, Nava, Zahara) hanya untuk data.
- Tanpa gradien, tanpa bayangan di permukaan diam, tanpa ikon berlatar warna.

## 2. Warna

**Token `primary` adalah skala abu-hitam.** Ini disengaja. Ratusan kelas `primary-*` lama (tombol, link, baris terpilih, fokus) otomatis jadi monokrom tanpa diubah satu per satu.
Oranye pindah ke token `accent`. Pakai `accent-*` hanya untuk tombol primer, logo, promo, dan grafik.

| Peran | Token |
|---|---|
| Tombol primer | gradien `bg-btn-primary` (`accent-400` ke `accent-500`), border `accent-600`, `shadow-btn-primary`, teks putih |
| Menu aktif | latar `sidebar.active` (`#F0F0F0`), teks `neutral-900` weight 500 |
| Link teks | `font-medium text-neutral-900 underline underline-offset-2 decoration-neutral-300`, hover `decoration-neutral-900` (warna saja tidak cukup membedakan) |
| Baris terpilih | `primary-50` |
| Fokus | cincin 2px `primary-500` |
| Logo, promo | `accent-500` |
| Status | `success`, `warning`, `danger` (selalu dengan label teks) |

## 3. Tipografi

Dua font dengan peran tetap, dimuat dari Google Fonts di `master-dashboard/index.html`:

- **Inter** (`font-heading`): judul halaman, judul section, angka statistik, tombol, badge. Bentuknya rapat dan tegas, cocok untuk penanda struktur.
- **Roboto** (`font-body`, default `body`): semua teks isi, yaitu body, tabel, field, menu sidebar, caption, tooltip grafik. Lebih lega dibaca di baris tabel yang padat.

Aturan:
- Hierarki dari ukuran dan warna abu, bukan dari tebal huruf. Weight maksimal `font-semibold`; `font-bold` tidak dipakai.
- Judul halaman `text-lg md:text-xl`, judul section `text-base md:text-lg`, angka statistik `text-2xl md:text-3xl`.
- Judul blok kecil dan abu, seperti "Domains 7 >" di Cloudflare: `text-sm text-neutral-500`, jumlah dalam chip abu, chevron di kanan.
- Item menu `text-sm` weight normal; aktif `font-medium`. Label grup sidebar `text-sm text-sidebar-label`, bukan kapital.
- Angka memakai `tabular-nums`. Kode booking memakai `font-mono text-xs`.
- Hanya skala Tailwind standar, tanpa ukuran arbitrary.

## 4. Komponen

- **Button primer**: seperti tombol "Watch now" Cloudflare. Gradien vertikal `#FF8130` ke `#FF6A00` (`bg-btn-primary`), border 1px `#E25A00`, kilap putih 28% di tepi atas, bayangan oranye lembut (`shadow-btn-primary`), teks putih, radius 8px, tinggi 36px. Hover `bg-btn-primary-hover`.
- **Button gelap** (`variant="dark"`): `neutral-800`, untuk aksi kuat yang bukan aksi utama.
- **Button sekunder**: putih, border `neutral-200`, bayangan tipis `shadow-btn-secondary`, radius 8px (seperti "Last 24 hours").
- **Kolom cari**: putih, border `neutral-200`, radius 8px, petunjuk `Ctrl K` dalam kotak kecil di kanan.
- **Badge**: satu-satunya gaya status. Kotak tint tipis, radius 4px, border tipis, `text-xs font-medium`, tanpa ikon dan tanpa titik. Nada: `success`, `warning`, `danger`, `info`, `neutral`, `promo` (peta lengkap di design-system.md §5.2).
- **Kartu**: putih, border `neutral-200`, radius 8px, tanpa bayangan.
- **Daftar ringkas** (pola Cloudflare "Domains", "Recents"): tanpa kartu. Baris 48px, garis `neutral-200` di antara baris, ikon abu di kiri, chevron `>` di kanan. Pakai untuk widget dashboard dan daftar pendek.
- **DataTable**: tetap kartu berbingkai radius 8px untuk daftar panjang dengan toolbar. Header `neutral-50`, baris dipisah garis.
- **Modal, dropdown, popover**: putih, radius 8px, `shadow-modal` atau `shadow-popover`.

## 5. Layout

- **Sidebar** 224px (`w-56`), putih, border kanan. Item menu 36px, radius 6px, ikon 16px abu. Aktif: latar `#F0F0F0`, tanpa garis warna. Hover: `neutral-100`.
- **Topbar** 64px, putih, border bawah, tanpa bayangan.
- **Konten** di atas putih, padding 16px mobile dan 32px desktop. Blok dipisah jarak, bukan latar berbeda.

## 6. Gerak

Hover 100ms. Dropdown, drawer, modal 200ms dengan `ease-cf`. Tanpa animasi dekoratif. Hormati `prefers-reduced-motion`.

## Aksesibilitas

| Pasangan | Rasio |
|---|---|
| `neutral-800` di putih | 15.1:1 |
| Putih di `primary-600` | 15.1:1 |
| `neutral-500` di putih | 4.74:1 (AA) |
| Putih di `accent-500` `#FF6A00` | 2.87:1 — gagal AA; dipakai di tombol primer karena meniru Cloudflare (lihat Decisions) |
| Putih di `accent-400` `#FF8130` | 2.49:1 — gagal AA |

Cincin fokus 2px wajib di semua elemen interaktif. Target sentuh mobile minimal 44x44px.

## Decisions

- [2026-11-11] (buttons) ~~Tombol primer pakai primary-600 #C93F08 dengan teks putih; oranye terang #FF5E1F hanya aksen~~ — digantikan keputusan monokrom di bawah.
- [2026-11-11] (global) Permukaan datar: kartu dan tabel dibatasi border 1px neutral-200, tanpa shadow; shadow hanya untuk dropdown, popover, modal — Gaya Cloudflare: kedalaman lewat garis, bukan bayangan; tabel padat tetap rapi.
- [2026-11-11] (global) ~~Radius 4px untuk tombol, field, kartu; 6px untuk modal~~ — digantikan: 6px kontrol, 8px kartu dan lapisan mengambang, mengikuti dash.cloudflare.com.
- [2026-11-11] (global) Tiru dash.cloudflare.com, bukan cloudflare.com: hampir monokrom, latar halaman putih, menu aktif latar abu tanpa garis warna — User membandingkan dengan konsol Cloudflare asli; versi marketing terlalu oranye.
- [2026-11-11] (color) Token `primary` adalah skala abu-hitam; oranye hanya lewat `accent` untuk logo, promo, dan grafik — Mengubah satu token membuat semua kelas primary-* lama ikut monokrom.
- [2026-11-11] (buttons) Tombol primer meniru "Watch now" dash.cloudflare.com: gradien oranye #FF8130 ke #FF6A00, border #E25A00, kilap tepi atas, bayangan oranye lembut, radius 8px — User minta tombol sama dengan Cloudflare. Konsekuensi diterima: teks putih di oranye ini 2.5–2.9:1, gagal WCAG AA; kalau jadi masalah, gelapkan gradien ke accent-700 tanpa mengubah bentuk.
- [2026-11-11] (typography) Inter untuk judul, angka statistik, tombol, dan badge (font-heading); Roboto untuk semua teks isi (font-body, default body) — Permintaan user. Weight maksimal font-semibold; font-bold disapu dari seluruh halaman.
- [2026-11-12] (status) Semua status memakai satu gaya: komponen Badge (kotak tint, tanpa ikon, tanpa titik) dengan nada success/warning/danger/info/neutral/promo; override warna lewat className dan span status buatan sendiri dilarang — User melihat status beda-beda antar halaman (Badge kotak, titik + teks, pil bertitik) dan memilih satu gaya.
- [2026-11-12] (links) Link teks memakai garis bawah abu tipis (decoration-neutral-300) yang menebal saat hover, teks neutral-900 font-medium; text-primary-* tidak dipakai untuk link — Setelah primary monokrom, link tidak bisa dibedakan dari teks biasa.
- [2026-11-12] (dialogs) Konfirmasi aksi memakai Modal kecil, bukan window.confirm/alert bawaan browser — Dialog bawaan tidak ikut gaya dan tidak bisa menampilkan galat di tempat.
- [2026-11-12] (cards) Kartu selalu rounded-2xl (8px) datar berbingkai lewat Surface/Section/Card; MetaBox ditampilkan sama dengan Section (tanpa pita header dan kotak ikon); empty state lewat EmptyState varian list/section/inline; tab lewat SegmentedTabs — Sebelumnya radius kartu campur 6/8px, MetaBox bergaya lama, dan pesan kosong ditulis manual dengan padding beda-beda.
- [2026-11-12] (header) Semua halaman memakai PageHeader; halaman detail: ikon kembali tanpa breadcrumb, judul nama, baris meta teks abu, baris badge status/atribut, aksi di kanan; tanpa foto di header — Sebelumnya Detail Akun, Agen, dan Jamaah punya tiga bentuk header berbeda.
- [2026-11-12] (spacing) Jarak baku: konten pt-6/pb-6 (md: 8), antar blok space-y-6, grid kartu gap-6, padding isi kartu p-5 di semua breakpoint, judul section ke isi mb-4, antar field 16px — Padding kartu sebelumnya campur p-4/p-5/p-6 dan jarak bawah halaman tidak sama dengan atas.
- [2026-11-12] (modal/ikon/memuat) Footer modal: Batal secondary lalu aksi utama; ikon 16/14/12/20px per konteks; memuat selalu LoadingSpinner (variant card untuk blok); halaman aktif paginasi berbingkai, bukan hitam — Sebelumnya Batal campur ghost/secondary, ikon 11–22px tanpa pola, 17 kotak loading buatan sendiri, dan paginasi aktif hitam penuh.
- [2026-11-12] (sidebar) Sidebar 224px (w-56) dan nama menu cukup nama bendanya ("Jamaah", "Booking", "Paket"), tanpa kata "Kelola" — Permintaan user: sidebar terlalu lebar dan kata "Kelola" berulang tanpa menambah arti.
