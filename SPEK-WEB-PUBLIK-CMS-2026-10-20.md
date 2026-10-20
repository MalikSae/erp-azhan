# SPEK — Web Publik, CMS SEO, dan Kontrak Data ERP (Hana & Alsha)

> **Status:** DRAFT untuk review. **Tahap 1 (ERP) dan Tahap 2 (`azhan-booking`, fondasi SEO) sudah dikerjakan, belum di-commit** (lihat §15). Tahap 3 (CMS) dan halaman publik sesuai Figma belum dimulai.
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

1. **Alamat kantor:** Figma Hana menampilkan Depok (halaman Kontak) dan Jakarta Barat (beranda). Alamat Alsha belum ada.
2. **Path detail non-Plus** (`/paket/umroh/{tier}/{tanggal}-{durasi}-hari/`) — spek SEO tidak menuliskannya; perlu persetujuan tim SEO sebelum dikunci.
3. **Format tanggal pada slug** (nama bulan Indonesia, ada/tidaknya nol di depan hari) — spek hanya memberi contoh `20-desember-2026`.
4. **Pemetaan kategori ke tier:** kategori ERP saat ini tampak "Low Cost, Reguler, Plus, VIP" (komentar migrasi 098) sedangkan vocab SEO "Hemat, Reguler, Comfort, Premium"; "Plus" di ERP bisa bentrok dengan program "Plus" di spek.
5. **Domain Hana di spek:** contoh memakai `www.hanatours.com`, produksi adalah `hanatours.co.id`.
6. **Konflik internal spek:** landing lokasi masih tergambar di sebagian mockup tetapi dinyatakan dihapus di v2.3 (mengikuti v2.3).
7. **Rating "4.8 (2200 reviews)"** di Figma: sumbernya belum jelas (editorial atau agregat).
8. **Batas panggilan Figma MCP paket Starter** membatasi pembacaan desain; sebagian halaman Alsha belum dilihat satu per satu.
9. ~~Spek v2.3-B belum dibaca~~ — sudah dicek: isinya sama dengan v2.3-A kecuali varian penyimpanan media (A: tanpa CDN; B: CDN dengan host `media.hanatours.com`). Lihat §14.
10. **Konten legal/regulasi** (nomor PPIU/PIHK, dokumen legalitas) tidak ada di desain; harus dari data brand/legal.

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

### Tahap 1 — ERP (dikerjakan 20 Okt 2026, belum di-commit)

| Butir spek | Hasil | Berkas utama |
|---|---|---|
| §4.1 skema | Migrasi **108** (tabel `destinations`; `schedules.journey_type/program_slug/destination_id`; `package_categories.seo_tier`) dan **109** (izin `cms.view/edit/publish` + pemetaan ke role). Keduanya sudah diterapkan ke DB dev lokal `erp_azhan_dev` (cadangan sebelum migrasi disimpan di scratchpad sesi). | `migrations/108_*.sql`, `migrations/109_*.sql` |
| Master destinasi | CRUD `/api/admin/destinations` (izin `masterdata.view/edit`); slug unik huruf kecil; destinasi yang dipakai paket tidak bisa dihapus (409). | `internal/destination/` |
| Klasifikasi paket | Tersimpan dalam transaksi yang sama dengan paket; dibaca kembali di `GET /api/admin/schedules/{id}`. Aturan: program `plus` wajib `destination_id`; destinasi harus aktif; paket baru default `umroh`; saat edit tanpa `journey_type` klasifikasi tidak diubah. | `internal/schedule/klasifikasi.go` |
| §4.2 status registrasi | `OPEN/ALMOST_FULL/CLOSED/SOLD_OUT` diturunkan (ambang `max(5, 20%)`, batas 30 hari memakai `batasPublikWIB`). | `internal/schedule/registration.go` |
| §4.3 API v1 | `GET /api/public/v1/packages` (filter `periode, destinasi, tier, program, durasi, hotel_bintang, harga_maks, registrasi, updated_since`, paginasi), `/packages/{id}` (`visible` / `gone` untuk arsip atau berangkat < 30 hari / 404 untuk draft), `/destinations`, `/sitemap-feed`. Endpoint lama tidak berubah. | `internal/schedule/public_v1.go`, `cmd/api/main.go` |
| Kategori sebagai tier | `seo_tier` pada kategori (API + form). **Belum diisi:** DB dev berisi kategori "Low Cost, Reguler, Plus, VIP"; pemetaan ke tier SEO menunggu keputusan (§12.4). | `internal/category/` |
| §4.4 izin | `cms.view` (direktur, product manager), `cms.view/edit` (marketing member), `cms.view/edit/publish` (marketing lead, CMO, super admin). | `migrations/109_*.sql` |
| Dashboard ERP | Halaman "Kelola Destinasi"; bagian "Klasifikasi Web Publik" di form dan detail paket; kolom/isian "Tier SEO" di Kategori. | `frontend/master-dashboard/src/...` |

**Pengujian:** unit (registrasi, durasi, filter, paginasi, bentuk JSON v1, normalisasi klasifikasi, slug destinasi, tier kategori), integrasi DB (`TestKlasifikasiDisimpanDanDibaca`, memulihkan data dev), dan uji HTTP manual end-to-end pada instance API sementara (buat paket uji, terbit, filter, arsip, hapus — semua dipulihkan). Seluruh suite `ERP_TEST_DB=1 go test ./...` lulus. Build `vite build` master-dashboard berhasil; belum diuji di browser.

**Yang belum ada dari Tahap 1:** pratinjau URL read-only di form paket (opsional, §14.3); tidak ada pengisian data nyata (destinasi, program, pemetaan tier) — itu keputusan operasional.

**Catatan operasional:** API perlu di-restart agar rute baru aktif; migrasi 108 dan 109 harus dijalankan di setiap lingkungan (`go run migrations/run.go migrations/108_web_publik_atribut.sql`, lalu 109). Izin baru masuk token setelah pengguna masuk ulang atau token diperbarui.

### Tahap 2 — `azhan-booking` (fondasi SEO; dikerjakan 20 Okt 2026, belum di-commit)

Repo `C:\laragon\wwwzhan-booking`. Dikerjakan dalam tiga langkah; sisi ERP: tautan email verifikasi dan reset password kini menunjuk ke `/booking/...` (`internal/customer/handler.go`, konstanta `PathVerifikasiEmail` dan `PathAturUlangPassword`, dengan tes).

| Langkah | Hasil |
|---|---|
| 2A — prefix `/booking` | Masuk, daftar, lupa/atur-ulang password, verifikasi email, akun, checkout, dan pesanan dipindah ke `src/app/booking/` dengan layout `noindex, nofollow`. Redirect permanen 308 dari alamat lama (`/masuk`, `/daftar`, `/lupa-password`, `/atur-ulang-password`, `/verifikasi-email`, `/akun`, `/checkout`, `/pesanan[/:id]`) menjaga email dan bookmark lama. |
| 2B — rute paket | `src/app/paket/[...slug]` menggantikan `/paket/[id]`. Data dari API v1 (`getPackages`, `getPackageV1`, `getDestinations`, revalidate 5 menit); `/api/schedules/{id}` tetap dipakai checkout agar seat dan harga langsung dari ERP. Resolusi: `/paket/{id}` lama → 308 ke slug; paket gone → 308 ke hub terdekat; slug cocok → detail (dua tombol "Pesan sekarang" dan "Tanya via WhatsApp" tetap); hub `/paket/umroh`, `/paket/umroh/plus`, `/paket/umroh/plus/{destinasi}`; selain itu 404 (hub tier dan `/paket/umroh/bulan/...` sengaja 404). Pustaka murni dengan tes: `paketUrl.js` (slug standar, tabrakan diberi akhiran `-{id}` sementara), `paketRoute.js`, `origin.js`. |
| 2C — SEO | `sitemap.xml` dan `robots.txt` dinamis per host (tanpa `/booking`, hub kosong tidak masuk; host `cms.*` disallow semua); JSON-LD: TravelAgency (beranda), Product+Offer dan BreadcrumbList (detail), BreadcrumbList (hub), FAQPage hanya bila data FAQ ada; canonical untuk `/`, `/paket`, detail, dan hub; hub kosong `noindex, follow`. |

**Pengujian:** 46 tes unit Node (`npm test`), `npm run check:design` bersih, `npm run build` berhasil; uji nyata dengan `next start` di port 3100 terhadap API ERP sementara di port 9191 (Host `hana.azhan.test:3100`): sitemap valid (9 URL, 0 `/booking`), robots, JSON-LD dapat di-parse, redirect lama, 404 yang dimaksud, dan meta `noindex` pada `/booking/masuk`. Data uji sementara sudah dihapus; DB dev kembali ke 12 paket terbit, 1 draft, 0 destinasi.

**Belum dikerjakan dari Tahap 2:** halaman publik sesuai desain Figma Hana/Alsha (tampilan beranda, daftar, hub, detail baru, token per brand, layout cetak A4 sederhana untuk `/jadwal-umroh/`, halaman kontak); pembaruan `BottomNav` yang masih menyembunyikan dirinya di `/paket/*`; filter pada hub; `/jadwal-umroh/`; `metadataBase` di root layout; data FAQ (menunggu CMS).

**Risiko dan keputusan terbuka dari Tahap 2:**
1. Kategori ERP bernama "Plus" menghasilkan path detail `/paket/umroh/plus/{tanggal}-{durasi}-hari` yang berbentuk seperti hub program Plus (kategori bukan program). Selama `seo_tier` belum dipetakan, path non-Plus memakai slug kategori. Perlu keputusan pemetaan kategori ke tier (§12.4).
2. Slug masih dibangkitkan di `azhan-booking` dan belum disimpan; ia baru stabil setelah CMS menyimpannya (Tahap 3). Mengubah tanggal atau tier paket sekarang mengubah URL tanpa riwayat redirect.
3. Cache data 5 menit berarti pengalihan paket yang baru diarsipkan bisa tertunda sampai 5 menit.
4. `/paket/...` memberi 500 (bukan 404 palsu) bila API v1 gagal; sitemap juga 500 bila data tidak bisa diambil.
5. Hub destinasi belum diuji dengan data destinasi nyata (DB dev tidak punya destinasi); pada pengujian Tahap 2B hub itu diuji dengan data sementara.
6. Perubahan git di `azhan-booking` terlihat sebagai penghapusan path lama dan berkas baru (`src/app/booking/`) karena belum di-stage.
