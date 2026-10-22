# SPEK — Web Publik, CMS SEO, dan Kontrak Data ERP (Hana & Alsha)

> **Status:** Rancangan awal 20 Okt 2026, kini sebagian besar terimplementasi dan ter-merge (Tahap 1, Tahap 2, Tahap 3 CMS, dan Fase 2 sampai gelombang 2E-1). Keadaan terbaru dan hal yang belum ada: §15; keputusan yang masih terbuka: §12.
> **Tanggal:** 20 Oktober 2026
> **Sumber keputusan:** sesi diskusi 20 Okt 2026 (grilling Q1–Q35), dokumen SEO di Drive (Spec CMS v2.3-A, Launch Plan v2.3, Mockup Reference Pack v2.3, Kebijakan Partner v1.0), desain Figma Hana & Alsha, memori keputusan arsitektur 2 Okt 2026.
> **Prinsip utama:** *CMS memiliki kata-kata, ERP memiliki angka.* Dokumen SEO adalah acuan struktur web; ERP/booking dibangun relevan dengannya.

---

## 1. Tujuan dan batas

Sistem booking yang sedang dibangun harus selaras dengan dokumentasi SEO tim Ganesha: struktur URL, model data paket, status registrasi, dan pemisahan fakta paket (ERP) dari konten/SEO (CMS).

**Dalam lingkup:** hanya Umroh; brand Hana dan Alsha; Fase 1 (lihat §9).
**Di luar lingkup:** migrasi dari situs lama (tidak ada migrasi: tidak ada redirect registry, disposisi URL lama, atau Change of Address), Nava, Haji dan Wisata Halal, Partner/Affiliate/komisi portal, form lead dan integrasi CRM, payment gateway.

> Bagian dokumen SEO yang gugur karena "tidak ada migrasi": seluruh isi Launch Plan v2.3 tentang `hanahajiumroh.com`, MIGRATE/RETIRE, pilot artikel, dan cutover. Yang tetap berlaku: struktur situs, model data, aturan URL/indeks, dan isi mockup CMS.

## 2. Peta sistem

| Host | Fungsi | Pemilik data |
|---|---|---|
| `master.azhan.id` | Dashboard holding: pusat data, pengguna, Marketing Hub (belum dibangun) | ERP |
| `office.azhan.id` | Dashboard admin brand berbasis izin | ERP |
| `crm.azhan.id` | CRM terpisah (tidak berubah) | CRM |
| `hanatours.co.id`, `alsha.travel` | Web publik + booking + akun pelanggan (`azhan-booking`) | ERP (angka) + CMS (kata-kata) |
| `cms.hanatours.co.id`, `cms.alsha.travel` | Admin CMS tim SEO, satu per brand | CMS |
| `hana.azhan.id` / `alsha.azhan.id` | Staging opsional (protected + noindex), bukan bagian rancangan inti | — |

Catatan: riwayat 2 Okt menyebut master-dashboard dipasang di `office.azhan.id` dan hanya dua dashboard internal. Daftar host di atas adalah koreksi terbaru dari pemilik proyek.

```
 ERP (Go, MySQL)  ──API publik v1──►  azhan-booking (Next.js)  ◄──── CMS (database per brand)
 angka: harga, tanggal, seat,          halaman publik, booking, akun,   kata-kata: slug, SEO, FAQ,
 status, atribut klasifikasi           sitemap, JSON-LD                 landing bulan, redirect
        ▲                                                                       │
        └────────── login & izin (JWT ERP) ◄────────────────────────────────────┘ (cms.<domain brand>)
```

### Keputusan arsitektur yang berubah dari 2 Okt 2026
| Keputusan lama | Menjadi | Alasan |
|---|---|---|
| "Proxy `/booking` ke microsite" | Dibatalkan. Situs publik dan booking satu aplikasi (`azhan-booking`). | CMS dibangun di repo yang sama, satu origin. |
| "CMS terpisah per brand, 1 codebase di-deploy per brand, DB terpisah" | **Dipertahankan.** CMS di dalam `azhan-booking`, dideploy per brand, database sendiri. | Sesuai keputusan awal. |
| `azhan-booking` tanpa database, brand dari host | Tetap resolusi brand dari host; menambah database CMS per brand. | Konten SEO butuh penyimpanan. |

## 3. Pembagian kepemilikan data

| Data | Pemilik | Catatan |
|---|---|---|
| Harga (quad/triple/double/infant), tanggal berangkat/pulang, seat total/sisa, status, maskapai, hotel, itinerary, include/exclude, brosur | **ERP** | Tidak pernah disalin ke CMS. |
| Tier (dari kategori), destinasi, program, `journey_type`, durasi (dihitung) | **ERP** | Atribut klasifikasi (§4). |
| Slug paket, meta title/description, canonical, robots, OG, fokus keyword, schema override, FAQ, H1/intro/slot konten | **CMS** | Slug dimiliki tim SEO. |
| Landing bulan, alias pencarian, redirect, riwayat URL | **CMS** | |
| Identitas brand: nama PT, alamat, telepon, WhatsApp, email, jam buka, sosial, NAP, logo, warna | **ERP** (profil brand) | Satu sumber; sudah ada field SEO/NAP (migrasi 050). |
| Teks marketing: statistik, visi misi, deskripsi footer, testimoni, tour leader | **CMS** (Fase 2) | |
| Akun pelanggan, booking, pembayaran | **ERP** | Tidak ada "permanent customer ownership" (sesuai spek SEO). |
| Media paket | **ERP** (URL) | CMS punya media library sendiri untuk aset editorial. |

## 4. Perubahan ERP (Tahap 1)

### 4.1 Skema (migrasi baru)
Usulan; nomor migrasi menyusul setelah 107.

1. `schedules` — tambah kolom:
   - `journey_type` ENUM('umroh','haji','wisata_halal') NOT NULL DEFAULT 'umroh'.
   - `program_slug` VARCHAR(50) NULL (mis. `plus`, `ramadhan`; NULL = paket tanpa program khusus).
   - `destination_id` BIGINT UNSIGNED NULL → `destinations`.
2. `destinations` (master baru): `id`, `name`, `slug` (unik), `is_active`. Dikelola dari dashboard ERP; tidak ada kombinasi destinasi→tier yang di-hardcode (aturan spek).
3. `package_categories` — tambah `seo_tier` ENUM('hemat','reguler','comfort','premium') NULL sebagai pemetaan stabil dari kategori ke vocab tier SEO. Kategori tetap pemilik klasifikasi tier (tidak ada field `tier` terpisah).
4. **Durasi tidak menjadi kolom.** Dihitung dari `berangkat_tanggal` dan `pulang_tanggal` (sudah ada dan sudah dipakai `durasiHari` di `azhan-booking`).
5. **Slug tidak disimpan di ERP.**

### 4.2 Status registrasi (turunan, tidak disimpan)
Dihitung di API dari `seat_total`, `seat_sisa`, status, dan tanggal:

| Status | Aturan |
|---|---|
| `SOLD_OUT` | `seat_sisa` = 0 |
| `CLOSED` | ditutup manual, atau berangkat kurang dari 30 hari (aturan yang sudah ada: `bookingClosed`) |
| `ALMOST_FULL` | `seat_sisa` ≤ max(5, 20% × `seat_total`) |
| `OPEN` | selain di atas |

Ambang 5 seat dan 20% adalah konfigurasi (konstanta kode dulu, pengaturan per brand menyusul).

### 4.3 API publik v1 (read-only, stabil)
Endpoint lama (`/api/schedules`, `/api/schedules/{id}`, `/api/public/*`) tetap berjalan.

| Endpoint | Fungsi |
|---|---|
| `GET /api/public/v1/packages` | Daftar paket brand. Filter: `periode` (YYYY-MM), `destinasi`, `tier`, `program`, `durasi`, `hotel_bintang`, `harga_max`, `registrasi`; `updated_since`; paginasi. |
| `GET /api/public/v1/packages/{id}` | Detail: fakta paket, itinerary, include/exclude, hotel, maskapai, harga per tipe kamar, `registration_status`, atribut klasifikasi (`journey_type`, `program`, `destination`, `tier`, `duration_days`). |
| `GET /api/public/v1/destinations` | Destinasi aktif beserta jumlah paket yang dijual (untuk hub). |
| `GET /api/public/v1/sitemap-feed` | ID + `updated_at` paket yang layak masuk sitemap. |
| `GET /api/public/brand` | Sudah ada; diperluas bila perlu (nama PT, jam buka, sosial). |

Aturan penting:
- Paket yang diarsipkan atau ditutup tidak dihapus dari jawaban: detail mengembalikan `visibility: "gone"` (bukan 404) agar penerima memutuskan 301 atau 410 (§6.4).
- Cache: respons dapat di-cache pendek; seat dan harga untuk checkout tidak boleh dari cache halaman.
- Brand ditentukan dari `domain` atau parameter `brand`, seperti pola yang sudah ada.

### 4.4 Izin CMS (RBAC)
Tambah permission data (pola migrasi 071–083): `cms.view`, `cms.edit`, `cms.publish`. Marketing Member: view + edit draft. Marketing Lead: + publish. Cakupan brand lewat Akses Brand yang sudah ada. Tidak ada role baru ("SEO adalah tim, bukan role" — migrasi 078).

## 5. CMS (Tahap 3)

### 5.1 Penempatan dan login
Rute admin di dalam `azhan-booking`, dilayani pada host `cms.<domain brand>`, dilindungi login dan `noindex`. Login memakai `POST /api/auth/login` ERP; CMS membaca permission dari JWT. Menu "Users & Permissions" di mockup tidak dibangun (pengguna dikelola di ERP).

### 5.2 Database CMS (per brand, MySQL)
Usulan tabel Fase 1:
- `package_seo`: `schedule_id` (unik), `slug` (unik per brand), `meta_title`, `meta_description`, `canonical_mode/url`, `robots_index`, `robots_follow`, `in_sitemap`, `og_title/description/image`, `focus_keyword`, `schema_custom`, `faq` (JSON), `review_status` (`auto` | `needs_review` | `reviewed`), `updated_by`, timestamps.
- `slug_history`: `schedule_id`, `old_slug`, `new_slug`, `changed_at`, `changed_by` (sumber 301 satu hop, tanpa chain).
- `pages` + `page_slots`: halaman statis dan hub (Beranda, `/paket/`, `/paket/umroh/`, hub tier, hub program, hub destinasi, `/jadwal-umroh/`) dengan slot editorial.
- `seo_landings`: tipe PERIOD saja; nilai periode, status Draft → Review → Approved, indeks/sitemap.
- `search_aliases`: alias yang hanya memetakan intent ke entitas kanonik, tidak membuat URL.
- `redirects`: manual (301/302) dan otomatis.
- `audit_log`.

### 5.3 Sinkronisasi dari ERP
Job berkala (beberapa menit) memanggil `GET /api/public/v1/packages?updated_since=` dan membuat catatan `package_seo` untuk paket baru: slug awal dari pola standar, meta awal dari atribut paket, `review_status = needs_review`. Halaman paket langsung tampil dan boleh diindeks dengan data otomatis itu; daftar "perlu review SEO" muncul di dashboard CMS.

### 5.4 Menu dashboard
Mengikuti Mockup Reference Pack v2.3 (tema marun Hana; tema brand untuk Alsha). Fase 1 aktif: Dashboard, Paket (hanya tab SEO Settings: Search Appearance, Indexing & Sitemap, Social/OG, Schema, Redirect & URL History), Halaman (Page & SEO Manager + editor slot), SEO Landing, Search Router, Redirects, URL History, Sitemap & Indexing. Menu Fase 2 tampil dengan penanda "Segera" dan tidak bisa dibuka.

Aturan CMS dari spek yang wajib diikuti:
- SEO hanya mengubah H1–H4 dan blok editorial pada slot yang disediakan; urutan system section dan layout terkunci; tanpa page builder bebas dan tanpa CSS/JS mentah.
- Slug berubah setelah publish → masuk URL History dan 301 satu hop ke URL final; hindari chain/loop; validasi keunikan sebelum publish; tanpa suffix acak.
- Landing hanya `PERIOD`; hanya yang Approved + Published masuk router dan sitemap; landing tak terdaftar → 404.
- Data Product schema (nama, harga, ketersediaan, gambar) dikunci dan bersumber dari ERP.
- Published + Index → sitemap otomatis; Noindex → keluar dari sitemap; SEO Manager bisa menimpa.

## 6. `azhan-booking` (Tahap 2)

### 6.1 Struktur URL publik
Mengikuti spek SEO (§5 Spec v2.3-A). Hanya Umroh pada Fase 1.

```
/                                   Beranda
/paket/                             Hub paket
/paket/umroh/                       Hub umroh
/paket/umroh/{hemat|reguler|comfort|premium}/   Hub tier — OPSIONAL: hanya bila dibuat sebagai landing kurasi (TBD-012); tidak otomatis dari inventori
/paket/umroh/plus/                  Hub program Plus
/paket/umroh/plus/{destinasi}/      Hub destinasi (stabil walau tier berubah)
/paket/umroh/plus/{destinasi}/{tier}-{tanggal}-{durasi}-hari/   Detail (contoh: reguler-20-desember-2026-12-hari)
/paket/umroh/{tier}/{tanggal}-{durasi}-hari/    Detail non-Plus  ← ASUMSI, perlu konfirmasi tim SEO
/paket/umroh/bulan/{periode}/       Landing bulan (Approved saja)
/jadwal-umroh/                      Jadwal; filter bulan tanpa URL baru
/kontak/
/booking/...                        Booking, checkout, pesanan, akun, masuk/daftar (noindex)
```

Aturan: filter dan pencarian tidak membuat URL (tanpa `?q=`, `?bulan=`); tidak ada landing lokasi; tidak ada kombinasi bulan+lokasi. Router pencarian: paket persis → periode → hub → no match (URL tidak berubah). Rute lama `/paket/{id}` numerik dialihkan 301 ke slug.

### 6.2 Halaman Fase 1
Beranda, daftar paket umroh, jadwal (dengan layout cetak A4 sederhana), detail paket, hub Umroh Plus per destinasi, kontak, header dan footer. Tab/tautan Haji dan Wisata Muslim disembunyikan.

### 6.3 Perilaku halaman
- **Detail paket:** tata letak Figma (kartu booking sticky di desktop, bar sticky bawah di mobile; itinerary, hotel, accordion termasuk/tidak termasuk/syarat) + breadcrumb, FAQ dari CMS, dan paket terkait. Dua tombol tetap: **"Pesan sekarang"** (nonaktif bila penuh/ditutup) dan **"Tanya via WhatsApp"** (pesan terisi). Pemilih kamar yang ada dipertahankan.
- **Kartu paket:** semua teks dari ERP (nama, tanggal, durasi, maskapai, hotel, harga "mulai dari", bilah seat selalu tampil, badge kategori/status). Poster hanya gambar; harga di poster tidak dianggap sumber harga.
- **Konsultasi:** tombol "Konsultasi" dan form di beranda/kontak menyusun pesan WhatsApp terisi (nama + pertanyaan). Tanpa endpoint baru, tanpa CRM.
- **Hasil pencarian:** state UI di halaman yang sama, bukan halaman/URL.

### 6.4 URL paket yang tidak dijual lagi
Paket berangkat kurang dari 30 hari atau diarsipkan: 301 ke hub relevan terdekat (tier/destinasi atau `/jadwal-umroh/`); 410 bila tidak ada hub yang relevan. Tidak ada redirect massal ke beranda dan tidak ada halaman kosong berstatus 200.

### 6.5 SEO teknis
- `generateMetadata` per halaman dari `package_seo`/`pages` (fallback ke metadata brand), canonical, Open Graph/Twitter.
- JSON-LD: `Product`+`Offer` (harga dan ketersediaan dari ERP), `BreadcrumbList`, `FAQPage` (dari FAQ CMS), `TravelAgency`/`Organization` (dari profil brand).
- `sitemap.xml` dan `robots.txt` per brand dibuat dari status indeks CMS + `sitemap-feed` ERP. `cms.*` selalu `noindex` dan di-disallow.
- Rendering: ISR dengan revalidate sekitar 5 menit untuk daftar/detail/hub; checkout dan akun `no-store`. Webhook revalidasi menyusul bila diperlukan.
- Brand diresolusi dari host (`src/proxy.js`), tema lewat token per brand.

### 6.6 Visual
Satu set komponen (`src/components/ui`, `design-system.md`) dengan token per brand: Hana mengikuti Figma Hana (marun), Alsha mengikuti Figma Alsha (navy, heading serif, footer gelap). Alur booking dan akun tetap gaya `azhan-booking` yang ada. Teks placeholder di Figma (lorem ipsum, "PT. Lorem ipsum", sisa nama "Alsha Tour" di halaman Hana, typo, label Inggris) tidak boleh dibawa ke produksi.

## 7. Akun pelanggan (di luar lingkup spek SEO)

Tetap ada sebagai bagian sistem booking (email + password per brand, `customer_accounts`, migrasi 104–107). Spek SEO menyatakan tidak ada akun/login pelanggan dan tidak ada "permanent customer ownership"; akun di sini hanya untuk booking dan tidak mengubah aturan atribusi partner.

## 8. Urutan kerja

| Tahap | Repo | Isi | Selesai bila |
|---|---|---|---|
| 1 | `erp-azhan` | Migrasi atribut (§4.1), master destinasi + halaman kelola di dashboard, status registrasi, API v1, izin `cms.*` | API v1 lulus tes (filter, status registrasi, `gone`); data paket yang ada terisi atribut |
| 2 | `azhan-booking` | Rute `/paket/...` bergaya spek, prefix `/booking`, metadata, sitemap, robots, canonical, JSON-LD, halaman Fase 1 sesuai Figma | Halaman dirender dari API v1 dengan slug sementara dari pola standar |
| 3 | `azhan-booking` | Database CMS, login ERP, sinkronisasi paket, halaman admin Fase 1 | Tim SEO dapat mengubah slug/meta, redirect otomatis berjalan |

Tahap 2 dan 3 berjalan beriringan. Sebelum CMS ada, slug dibangkitkan dari pola standar tanpa penyimpanan; setelah CMS berjalan slug dibaca dari `package_seo`.

## 9. Fase

**Fase 1:** semua di atas.
**Fase 2:** perbandingan paket; tentang kami, legalitas, tour leader, testimoni; blog (artikel, tag terkelola, primary cluster, alat regex GSC); tracking & custom code; Global SEO; template A4 khusus; Haji dan Wisata Halal; override slug oleh SEO bila diperlukan di luar alur normal; halaman tambahan Alsha (Haji Plus, Kalkulator Umroh, Biaya Umroh, Cabang Alsha); integrasi form lead/CRM; Marketing Hub.

## 10. Pengujian

- ERP: tes API v1 (filter, paginasi, `registration_status` di batas ambang, `visibility: gone`), migrasi atribut, izin `cms.*`.
- `azhan-booking`: slug → paket, 301/410, sitemap hanya berisi URL yang layak, canonical, JSON-LD valid, `noindex` pada `/booking/*` dan `cms.*`.
- CMS: keunikan slug, riwayat slug tanpa chain, landing tak terdaftar → 404, auto-create paket baru.
- Aturan repo: komit hanya atas perintah pemilik proyek.

## 11. Risiko

| Risiko | Mitigasi |
|---|---|
| Halaman paket bergantung pada CMS untuk slug | Fallback ke slug standar dari atribut bila catatan CMS belum ada atau CMS tidak dapat dijangkau |
| Kategori ERP tidak cocok dengan vocab tier | Pemetaan `seo_tier` per kategori; periksa data nyata sebelum Tahap 1 selesai |
| Poster brosur memuat harga basi | Harga hanya dari ERP; poster hanya gambar |
| Satu aplikasi memuat situs publik dan admin CMS | Pisahkan rute dan middleware; `cms.*` selalu terproteksi |
| Penumpukan keputusan yang belum dikonfirmasi tim SEO | Ditandai di §12 |

## 12. Hal yang masih perlu konfirmasi

Status per 22 Okt 2026. "Sementara" berarti diputuskan pemilik proyek tanpa menunggu tim SEO dan dapat dikoreksi kemudian; perubahan slug lewat CMS menulis URL History dan redirect satu hop, jadi koreksi tidak memutus URL yang sudah terindeks (rujukan: `azhan-booking/docs/deploy.md` bagian 7b).

| # | Hal | Status |
|---|---|---|
| 1 | **Alamat kantor.** Figma Hana menampilkan Depok (halaman Kontak) dan Jakarta Barat (beranda). | **Terbuka.** Alamat dibaca dari profil brand ERP; pemilik proyek harus memastikan alamat Hana. Alamat Alsha juga belum ada. |
| 2 | **Path detail non-Plus** `/paket/umroh/{tier}/{tanggal}-{durasi}-hari`. | **Diputuskan sementara** oleh pemilik proyek: indexable, mengikuti pola Plus. Tim SEO dapat meninjau setelah live. |
| 3 | **Format tanggal pada slug.** | **Diputuskan sementara:** hari tanpa nol di depan, bulan Indonesia, tahun (`9-desember-2026`); jangan diubah lagi. |
| 4 | **Pemetaan kategori ke tier SEO.** | **Diputuskan sementara** (Low Cost: hemat, Reguler: reguler, Plus: comfort, VIP: premium), **tetapi belum diisi di data**: DB dev belum memiliki nilai `seo_tier`. Skrip `UPDATE` ada di `azhan-booking/docs/deploy.md` bagian 7b; harus dijalankan di ERP produksi sebelum paket terlihat mesin pencari. Kategori "Plus" tetap hanya label harga; program Plus ditentukan per paket. |
| 5 | **Domain Hana.** | **Selesai.** Produksi `hanatours.co.id`; `www.hanatours.com` di spek hanya contoh. Domain Alsha dalam panduan deploy: `alsha.travel`. |
| 6 | **Konflik internal spek** (landing lokasi). | **Selesai:** mengikuti v2.3, tanpa landing lokasi. |
| 7 | **Rating "4.8 (2200 reviews)"** di Figma. | **Terbuka:** sumbernya belum jelas; tidak dipakai sebagai data. |
| 8 | **Pembacaan Figma terbatas** (paket Starter, resolusi rendah). | **Dicatat:** hex warna dan font Alsha adalah perkiraan (lihat §15.2). Butuh aset resmi dari tim desain. |
| 9 | Spek v2.3-B. | **Selesai** (lihat §14). |
| 10 | **Konten legal/regulasi** (nomor PPIU/PIHK, dokumen legalitas, teks Kebijakan Privasi). | **Terbuka.** Nomor PPIU/PIHK dari profil brand ERP; dokumen legalitas dan teks Kebijakan Privasi harus datang dari tim legal. Tidak ada teks bawaan. |
| 11 | **Konfirmasi tim SEO** atas keputusan sementara no. 2, 3, dan 4. | **Terbuka.** |
| 12 | **Profil Alsha** (tentang, tim, legalitas) dan aset desain resmi. | **Terbuka.** |

## 13. Referensi

- Spesifikasi SEO: Hana_Tours_Website_CMS_Business_Technical_Specification_v2.3-A (Drive: Migration/Documentations/Revisi IV).
- Mockup CMS: Hana_Tours_Website_CMS_Mockup_Reference_Pack_v2.3 (65 halaman).
- Desain publik: Figma "Azhan (Copy)", halaman Hana dan Alsha.
- Kode terkait: `internal/schedule/*`, `internal/category/*`, `internal/brand/*`, `internal/customer/*`, `cmd/api/main.go` (rute publik), `azhan-booking/src/proxy.js`, `src/lib/backend.js`, `src/app/paket/[id]/page.jsx`, `src/components/RoomPicker.jsx`.
- Memori keputusan: Arsitektur Terintegrasi Azhan Grup (2 Okt 2026), arah UI ERP, aturan "jangan commit tanpa perintah".

## 14. Kesesuaian dengan Spec v2.3-A/B FINAL (dicek 20 Okt 2026)

Sumber: `Hana_Tours_Website_CMS_Business_Technical_Specification_v2.3-A_FINAL.pdf` dan `...v2.3-B_FINAL.pdf`. Keduanya berstatus "Implementation review"; isi A dan B sama, kecuali **varian penyimpanan media** (A: shared object storage tanpa CDN; B: dengan CDN dan host media terpisah, `STOR-B-*`). Daftar TBD (TBD-001 s/d TBD-021) tidak berubah dari draft yang saya baca sebelumnya.

### 14.1 Sesuai
| Aturan spek | Penerapan di dokumen ini |
|---|---|
| ERP/Central adalah sumber fakta paket; CMS tidak membuat field harga/hotel/flight/itinerary alternatif (CMS-ARCH-001) | §3 |
| Satu keberangkatan = satu record paket, independen (AC-PKG-V2-02) | `schedules` sudah per keberangkatan |
| URL detail dari hierarki stabil + slug keberangkatan unik; tier masuk slug Umroh Plus; uji keunikan; tanpa suffix acak; slug berubah → URL History + 301 tanpa chain (SEO-URL-001..005) | §5, §6.1 |
| Contoh slug `reguler-20-desember-2026-12-hari` (AC-PKG-V2-01) | §6.1 |
| Landing hanya PERIOD, Approved saja, rute tak terdaftar → 404; tanpa landing lokasi; query lokasi saja = no match tanpa URL | §5.4, §6.1 |
| Pencarian dan filter tidak membuat URL; router: paket persis → periode → hub → no match | §6.1, §6.3 |
| `/jadwal-umroh/` otomatis dari paket Published, filter bulan tanpa URL baru (SCHED-001/002, AC-SCHED-V2-01) | §6.2 |
| Published + Index → sitemap otomatis; Noindex → keluar; SEO Manager dapat menimpa | §5.4 |
| Status registrasi `OPEN/ALMOST_FULL/CLOSED/SOLD_OUT` ("or approved enum"), seat dari sumber booking | §4.2 |
| SEO hanya mengedit slot yang disediakan; urutan system section terkunci; tanpa page builder bebas | §5.4 |
| Data Product schema dari Central | §6.5 |

### 14.2 Menyimpang secara sadar (keputusan pemilik proyek) — perlu dicatat
| # | Spek | Keputusan kita | Dampak |
|---|---|---|---|
| D1 | `PKG-MEDIA-001`, `MEDIA-*`, `STOR-A/B`: satu Shared Media Service dengan Asset ID yang sama untuk Central dan Dedicated CMS; file tidak disalin ke storage terpisah | CMS punya media library sendiri; foto paket dibaca dari ERP lewat URL (Q20). **Diputuskan: dipertahankan.** | Acceptance `AC-MEDIA-V2-01` (aset tersedia di keduanya) tidak terpenuhi; "Used In" lintas sistem tidak ada. Dapat disatukan nanti. |
| D2 | `PRINT-001` (MUST): `/jadwal-umroh/` punya layout A4 khusus, bukan sekadar print layout web | **Diputuskan:** layout cetak A4 khusus yang sederhana (stylesheet/rute cetak) masuk Fase 1; template final dan field default menunggu TBD-008 dan masuk Fase 2 (Q21 direvisi) | Memenuhi `PRINT-001` secara minimal; template final menyusul. |
| D3 | Spek mengasumsikan transaksi dilakukan offline lewat WhatsApp/Agent/CS; web adalah sarana lead | Ada booking online di `azhan-booking` dan akun pelanggan | Sesuai keputusan 2 Okt ("booking tandingan di spec dibatalkan"). Halaman booking `noindex`. Spek SEO tidak mengatur alur ini. |
| D4 | Appendix F (baseline, final = TBD-002): Super Admin, Developer, SEO Manager, Editor, Writer, Ops/Admin | Izin ERP `cms.view/edit/publish`; tanpa role SEO baru (SEO = tim) | Pemetaan awal: SEO Manager ≈ Marketing Lead (publish); Editor/Writer ≈ Marketing Member (edit draft). Appendix F memisahkan aksi halus (mis. Custom Schema "Approved", Tracking Code "Manage approved"); perlu izin tambahan saat modul Fase 2 dibangun. Publish Package tetap di ERP. |
| D5 | Fase 1 hanya sebagian modul (tanpa Blog, Tracking, Testimoni, Author, Global SEO) | Fase bertahap (Q9, Q29) | Banyak requirement berstatus MUST di spek akan baru terpenuhi di Fase 2. |

### 14.3 Koreksi pada rancangan awal setelah pencocokan
- **Hub tier (`/paket/umroh/{tier}/`) tidak otomatis ada.** Pohon §5.1 spek mencantumkannya, tetapi `SEO-URL-002` dan `TBD-012` menyatakan tier tidak otomatis membuat tier hub; hub tier hanya landing kurasi opsional. §6.1 sudah diperbaiki. Akibatnya path detail non-Plus (`/paket/umroh/{tier}/{tanggal}-{durasi}-hari/`) menjadi lebih tidak pasti, karena segmen `{tier}` bisa tidak punya halaman induk. Perlu keputusan tim SEO (butir §12.2).
- **Durasi:** spek memasukkan "Duration" sebagai isian editor paket, sedangkan rancangan ini menghitungnya dari tanggal berangkat/pulang (`pulang − berangkat + 1`, sama dengan `durasiHari` di `azhan-booking`). Verifikasi dulu pada data nyata bahwa hasilnya sama dengan durasi yang dimaksud ops (mis. paket "9 Hari").
- **Generated URL Preview:** editor paket di spek/mockup (Central) menampilkan pratinjau URL hasil pola. Opsional di ERP: tampilkan pratinjau read-only dari atribut klasifikasi di form paket (slug aktual tetap di CMS).
- **Domain contoh spek:** `www.hanatours.com` (dan `media.hanatours.com` di varian B) hanya contoh; domain produksi adalah `hanatours.co.id`.
- **Seat:** spek menghendaki seat terpesan diturunkan dari sumber booking dan menghindari field manual ganda. ERP menyimpan `seat_total` dan `seat_sisa` yang diperbarui oleh transaksi booking dan penyesuaian seat; `registration_status` diturunkan dari keduanya. Dapat diterima, tetapi perlu dicatat bahwa `seat_sisa` adalah kolom tersimpan.

### 14.4 TBD spek yang menyentuh rancangan ini
`TBD-001` (lifecycle dan perilaku SEO setelah Departed/Archived — ditutup sementara oleh §6.4), `TBD-002` (matriks izin), `TBD-003` (vendor storage), `TBD-006` (integrasi booking untuk seat — dipenuhi oleh API v1 §4.3), `TBD-008` (template A4), `TBD-010` (field Haji), `TBD-012` (landing tier kurasi), `TBD-013` (definisi slot per template).

## 15. Status implementasi

Per 22 Okt 2026. Yang disebut "ter-merge" sudah ada di cabang `main` masing-masing repo. Hal yang belum ada dicatat di §15.7.

### 15.1 ERP — Tahap 1 (repo `erp-azhan`, ter-merge)

| Butir spek | Hasil | Berkas utama |
|---|---|---|
| §4.1 skema | Migrasi **108** (tabel `destinations`; `schedules.journey_type/program_slug/destination_id`; `package_categories.seo_tier`). | `migrations/108_*.sql` |
| §4.4 izin | Migrasi **109** (`cms.view/edit/publish` dan pemetaan ke role) dan **110** (izin Fase 2: `cms.blog.write`, `cms.blog.publish`, `cms.schema.approve`, `cms.tracking.manage`; PR #10). Izin baru masuk token setelah pengguna masuk ulang. Semua role `cms.*` saat ini berstatus Holding. | `migrations/109_*.sql`, `migrations/110_*.sql` |
| Master destinasi | CRUD `/api/admin/destinations` (izin `masterdata.view/edit`); slug unik; destinasi yang dipakai paket tidak bisa dihapus (409). | `internal/destination/` |
| Klasifikasi paket | Disimpan dalam transaksi yang sama dengan paket. Program `plus` wajib `destination_id`; paket baru default `umroh`. | `internal/schedule/klasifikasi.go` |
| §4.2 status registrasi | `OPEN/ALMOST_FULL/CLOSED/SOLD_OUT` diturunkan (ambang `max(5, 20%)`, batas 30 hari). | `internal/schedule/registration.go` |
| §4.3 API publik v1 | `GET /api/public/v1/packages` (filter dan paginasi), `/packages/{id}` (`visible` / `gone` / 404 untuk draft), `/destinations`, `/sitemap-feed`; `GET /api/admin/me` untuk identitas dan izin (PR #7); `ketentuan_pembatalan` di profil publik brand (PR #8). | `internal/schedule/public_v1.go`, `internal/identity/`, `internal/brand/` |
| Tier kategori | Kolom dan isian `seo_tier` ada di API dan form Kategori. **Belum diisi di data dev** (§12.4). | `internal/category/` |
| Dashboard ERP | Halaman "Kelola Destinasi"; "Klasifikasi Web Publik" di form dan detail paket; isian "Tier SEO" di Kategori. | `frontend/master-dashboard/` |
| Istilah | Seragam "booking" dan "seat" di email, galat API, dan dashboard (PR #9; panduan lengkap di `azhan-booking/docs/copy-guide.md`). | `internal/mailer/`, dashboard |
| Email booking | Email "Booking berhasil dibuat" dikirim setelah booking mandiri pelanggan baru dibuat (asinkron; kegagalan kirim hanya dicatat; tidak dikirim untuk retry idempoten maupun booking agen). Sebelumnya fungsi templatenya tidak dipanggil di mana pun. Belum ter-merge saat dokumen ini ditulis (cabang `fix/email-booking-dibuat`). | `internal/selfbooking/notify.go` |

PR ERP ter-merge yang relevan: #7 (API v1 dan `/api/admin/me`), #8 (ketentuan pembatalan brand), #9 (istilah booking/seat), #10 (izin CMS Fase 2). Pratinjau URL read-only di form paket (opsional, §14.3) belum dibuat.

### 15.2 `azhan-booking` — Tahap 2: halaman publik Fase 1 (ter-merge)

- Prefix `/booking` untuk masuk, daftar, akun, checkout, dan detail booking (`noindex`); alamat lama dialihkan 308. Rute detail booking tetap `/booking/pesanan/{id}` (hanya teks tampilan yang memakai istilah "booking").
- Rute paket `/paket/...` bergaya spek dari API v1, hub program dan destinasi, `sitemap.xml` dan `robots.txt` dinamis per host, JSON-LD (TravelAgency, Product+Offer, BreadcrumbList, FAQPage bila ada data).
- Halaman publik sesuai Figma: beranda, daftar paket, hub, detail paket, kontak, dan `/jadwal-umroh`; tema per brand: **Hana marun, Alsha navy**. Figma terbaca dalam resolusi rendah, jadi **hex warna dan font Alsha adalah perkiraan** (judul Alsha memakai Lora; pita footer Alsha polos) dan harus diganti dengan aset resmi di `src/lib/theme.js` dan `src/app/globals.css`.
- Keputusan URL dan SEO untuk go-live dirangkum di `azhan-booking/docs/deploy.md` bagian 7b (lastmod jujur, robots, meta unik, `priceValidUntil`).

### 15.3 `azhan-booking` — Tahap 3: CMS (ter-merge)

Admin CMS tinggal di aplikasi yang sama pada host `cms.<domain brand>`, dengan database MySQL per brand (hanya kata-kata: slug, SEO, halaman, redirect) dan login memakai akun ERP. Dokumentasi teknis: `azhan-booking/docs/cms.md`.

| Tahap | Isi |
|---|---|
| 3A | Fondasi: database per brand dengan penjaga brand, sinkronisasi paket dari ERP (cron dengan kunci), login ERP, admin shell. |
| 3B | Editor paket (lima tab): slug, meta, canonical, robots, Open Graph; daftar paket dengan pencarian. |
| 3C | Redirects, URL History (tanpa chain), monitor sitemap dan indexing. |
| 3D | Manajer halaman dan editor slot konten (SEO hanya mengedit slot yang disediakan, tanpa page builder bebas). |
| 3E | SEO Landing bulan dan Search Router. |

### 15.4 `azhan-booking` — Fase 2 (ter-merge; migrasi CMS 001–010)

| Gelombang | Isi |
|---|---|
| 2A | Global SEO dan Tracking terstruktur (GTM, GA4, Meta Pixel; ID saja, tanpa kode bebas); kode verifikasi Search Console dari profil brand ERP. |
| 2B | Media library, Tentang kami, Legalitas, Tour Leader, Testimoni (tayang hanya bila Published dan terisi). |
| 2C | Blog: artikel, tag terkelola, primary cluster, alat regex dan ekspor GSC; izin terpisah `cms.blog.write` dan `cms.blog.publish`. |
| 2D-1 | Perbandingan paket (`/bandingkan`, utilitas noindex), template A4 untuk `/jadwal-umroh/cetak` (`PRINT-001`), override slug terkontrol (butuh alasan; slug lama 308 satu hop). |
| 2D-2 | Haji Plus, Kalkulator Umroh, Biaya Umroh, Cabang (halaman tambahan Alsha), dan layar Pengaturan berisi saklar fitur per brand (semua mati secara bawaan; halaman 404 bila saklar mati atau konten belum Published). |
| 2E-1 | Persetujuan cookie (skrip Tracking tidak dimuat sebelum izin; pilihan Terima semua, Tolak, Atur setara), halaman Kebijakan Privasi dari CMS (tanpa teks bawaan), hapus dan redirect item, halaman daftar terkelola. |

Migrasi CMS: 001 inti, 002 manajer halaman, 003 landing dan router, 004 meta otomatis, 005 Global SEO dan Tracking, 006 media/Tour Leader/Testimoni/halaman info, 007 Blog, 008 Template A4 dan override slug, 009 saklar fitur dan Cabang, 010 pengerasan 2E-1.

### 15.5 Pengujian

- ERP: unit dan integrasi DB (`ERP_TEST_DB=1 go test ./...`) untuk registrasi, filter dan paginasi API v1, klasifikasi, tier kategori, izin, serta alur akun dan booking mandiri (termasuk idempotensi checkout dan email booking).
- `azhan-booking`: tes unit Node (`npm test`), `npm run check:design`, dan `npm run build`.
- **Uji e2e penuh terhadap ERP asli (22 Okt 2026, setelah Fase 1 dan 2 selesai): 504 pemeriksaan, 492 lulus, 11 gagal, 1 tidak teruji.** Dari 11 yang gagal, 3 adalah temuan produk dan 8 artefak uji atau perilaku yang sudah didokumentasikan. Cakupan: crawl 71 URL Hana dan 58 URL Alsha (uji) tanpa 404/5xx tak disengaja, canonical dan JSON-LD valid tanpa Review, tidak ada overflow di 375 px, sitemap sama dengan Monitor di empat keadaan data, 34 layar CMS, sembilan siklus hidup konten, matriks izin sekitar 170 sel dengan akun role nyata (aksi tulis tanpa izin ditolak server), XSS di 15 halaman, ERP mati/hang dan DB CMS mati (degradasi anggun), alur booking pelanggan sampai Booking Saya, dan dry-run `docs/deploy.md` dari database kosong (migrasi 001-010).
- **Temuan dan penanganannya:** (a) Next.js 16.3.1 punya kerentanan kritis (`npm audit`), dinaikkan ke 16.4.2 dengan 0 kerentanan tersisa; (b) email "Booking berhasil dibuat" tidak pernah terkirim, diperbaiki di ERP (§15.1); (c) `cms.blog.write` dapat menyetel jadwal terbit, ditolak di server; (d) URL ber-encoding rusak menjawab 500, kini 404; (e) editor paket memakai data basi untuk pola standar, kini data fresh; (f) tidak ada `prefers-reduced-motion`, urutan heading, meta judul menerima tag, `CMS_MEDIA_DIR` dibuat diam-diam, label "Password": semuanya diperbaiki; (g) dokumen deploy dilengkapi (env saat build, `SITE_URL_TEMPLATE`/`SMTP_*`, kode 502 sinkronisasi, header keamanan reverse proxy). **Belum ditangani (bukan kode):** deskripsi meta identik di beberapa hub (diisi tim SEO di menu Halaman), header HSTS/CSP/Referrer-Policy (reverse proxy), `x-middleware-rewrite` terlihat di host cms (dihapus di reverse proxy), `/api/cms/sync` terjangkau dari host publik (batasi ke IP cron bila perlu), `/promo` yatim.
- Belum diuji: tampilan di perangkat nyata dan semua peramban, pengiriman SMTP sungguhan, banner cookie dengan CDN di depannya, dan beban.

### 15.6 Keputusan sadar dan penyimpangan

D1–D5 (§14.2) tetap berlaku, dengan pembaruan: D2 kini terpenuhi (template A4 khusus, 2D-1); D4 diperluas dengan izin halus Fase 2 (migrasi 110); D5 sebagian terpenuhi (Blog, Tracking, Testimoni, Tour Leader, Global SEO sudah ada, §15.4); D1 dipertahankan (butir 6). Penyimpangan baru:

1. **Cabang disimpan di CMS.** ERP tidak punya model cabang, sehingga cabang adalah satu-satunya data fakta baru di database CMS (dicadangkan bersama database).
2. **Redirect memakai 308**, bukan 301 maupun 410. Google menyamakan 308 dengan 301; paket kedaluwarsa dialihkan ke hub terdekat yang berisi paket (terakhir `/jadwal-umroh`), bukan 410.
3. **"ISR" sebenarnya dinamis.** Halaman dirender dinamis dengan cache data 5 menit untuk API ERP (30 detik untuk data CMS, per proses), bukan ISR statis; perubahan di ERP bisa tertunda sampai 5 menit.
4. **Tombol "Booking sekarang"** menggantikan "Pesan sekarang" (panduan istilah: Booking, seat, jamaah; bukan pesanan/pemesanan/kursi/reservasi).
5. **Tracking terstruktur tanpa kode bebas.** Hanya ID GTM/GA4/Meta Pixel yang dapat diisi.
6. **Media di disk** (`CMS_MEDIA_DIR`, WebP hasil proses ulang, disk persisten dan dicadangkan bersama database), bukan Shared Media Service; konsisten dengan D1.
7. **Path non-Plus, format tanggal, dan pemetaan tier** adalah keputusan sementara pemilik proyek (§12).

### 15.7 Belum ada atau di luar rilis ini

- Produk **Wisata Halal** dan **Haji** (Haji Plus sebagai halaman informasi sudah ada di 2D-2; produk Haji dan template khusus belum, menunggu TBD-010).
- Form lead dan integrasi CRM; Marketing Hub.
- Pemetaan `seo_tier` di data ERP (skrip siap, belum dijalankan).
- Konfirmasi tim SEO atas path non-Plus dan format tanggal.
- Alamat Hana yang pasti, profil dan aset desain resmi Alsha (hex dan font), serta teks Kebijakan Privasi dan dokumen legal dari tim legal.
- Header keamanan di reverse proxy (HSTS, CSP, dan sejenisnya) belum diatur; itu bagian konfigurasi deploy.
- Pratinjau URL di form paket ERP, halaman paket evergreen (Fase 1.5), dan Shared Media Service (D1).
- Banner cookie sudah ada (2E-1); yang belum adalah teks Kebijakan Privasi yang ditautkannya.

### 15.8 Urutan peluncuran dan prasyarat deploy

Rincian langkah dan perintah ada di `azhan-booking/docs/deploy.md`; ringkasannya:

1. **ERP:** terapkan migrasi 108, 109, 110; deploy kode ERP terbaru; isi profil brand (domain persis sama dengan host publik, kontak, PPIU/PIHK, ketentuan pembatalan); isi master destinasi, klasifikasi paket, dan `seo_tier` kategori; beri role `cms.*` ke tim SEO (mereka harus masuk ulang). Atur `SITE_URL_TEMPLATE` dan SMTP agar tautan dan email transaksional (verifikasi, reset password, booking) benar.
2. **Database CMS per brand:** buat, lalu `npm run db:migrate` (001–010). Siapkan direktori media persisten (`CMS_MEDIA_DIR`).
3. **Deploy `azhan-booking`:** variabel lingkungan, build, reverse proxy (teruskan `Host` dan `X-Forwarded-Host`, `client_max_body_size` minimal 4 MB untuk host `cms.*`, tanpa cache HTML publik lintas pengunjung), DNS, TLS.
4. **Cron sinkronisasi** paket tiap beberapa menit, lalu verifikasi pasca-deploy (bagian 7 panduan deploy, termasuk uji ERP mati).
5. **Konten dan legal:** tim legal mengisi dan menerbitkan Kebijakan Privasi, periksa banner cookie di staging, baru publikasikan Tracking.
6. **Go-live ke mesin pencari** hanya setelah butir di atas lulus dan §12 butir 1, 4, dan 10 beres; daftarkan `sitemap.xml` di Search Console. Jaga domain tidak terindeks lebih awal selama persiapan.
