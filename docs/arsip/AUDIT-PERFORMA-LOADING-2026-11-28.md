# Audit Performa Loading — Web, App, API (2026-11-28)

Audit baca-saja atas tiga repo: `azhan-booking` (web Next.js 16), `azhan-mobile` (app Hana, Expo SDK 58), dan `erp-azhan` (API Go). Belum ada pengukuran waktu nyata; angka dampak adalah estimasi. Temuan bertanda ✔ sudah dicek ulang langsung di kode.

## Ringkasan

Penyebab utama lambat, berurutan:

1. **Web: semua halaman dirender dinamis per request** (brand dibaca dari `headers()` di root layout) dan tanpa streaming (`loading.jsx`/`Suspense` = 0). TTFB = waktu render penuh di setiap kunjungan.
2. **App: tanpa cache tersimpan.** Setiap buka app mulai dari kosong, dan katalog Beranda menunggu brand → paket (serial per halaman) → semua itinerary.
3. **Gambar: file asli dari `/uploads` tanpa `Cache-Control`, tanpa thumbnail.** Ada poster 6,5 MB; logo maskapai 70–82 KB untuk tampilan setinggi 16 px.
4. **API: tiap query 3 round-trip ke MySQL** (DSN tanpa `interpolateParams`), detail paket ±12 query berurutan, tanpa kompresi gzip.

## API ERP (`erp-azhan`)

### Dampak tinggi
| # | Temuan | Bukti | Perbaikan |
|---|---|---|---|
| A1 ✔ | DSN tanpa `interpolateParams=true`: tiap query berparameter = prepare+execute+close | `internal/shared/config.go:55` | Tambah `&interpolateParams=true` (aman, utf8mb4) |
| A2 ✔ | `fetchFull` membiarkan `rows` terbuka sambil menjalankan 8 query lanjutan → 2 koneksi per request detail | `internal/schedule/repository.go:544-561` | `rows.Close()` setelah `scanRow`, atau `QueryRowContext` |
| A3 | Detail paket publik ±12 query berurutan; 4–5 di antaranya hitungan admin (booking, komisi) yang dibuang `ToPublic()`; klasifikasi & lokasi dimuat dua kali | `schedule/public_v1.go:330-354`, `repository.go:257-270,536-607` | `GetPublicByID` khusus: selectFull + transit + benefit + lokasi (paralel) |
| A4 ✔ | Tanpa kompresi respons | grep `Compress`/`gzip` = 0; `cmd/api/main.go:109-129` | `middleware.Compress(5, "application/json")` atau `encode zstd gzip` di Caddy |
| A5 ✔ | `/uploads` disajikan `http.ServeFile` tanpa `Cache-Control`; ukuran asli (maks 10 MB), tanpa varian kecil | `internal/media/handler.go:178-188,400` | `Cache-Control: public, max-age=31536000, immutable` (nama file UUID); buat varian ±400 px & ±1200 px WebP saat upload |
| A6 | Worker pengingat jalan tiap menit dan mengulang scan harian (~960×/hari) tanpa indeks `bookings.status`; `INSERT IGNORE…SELECT` mengunci baris tabel booking/payment | `cmd/api/main.go:199`, `notifikasi/pengingat.go:43-151` | Jalankan sekali sehari; tulis ulang `komisiTersedia` agar sargable; tambah indeks status |

### Dampak sedang
- **A7** List publik memuat semua paket brand lalu filter & paginasi di Go (`schedule/publik.go:98`, `public_v1.go:304-310`); sitemap memuat objek penuh. → Filter + `LIMIT` di SQL, query ringan untuk sitemap.
- **A8** Payload list membawa field khusus detail (include/exclude, penerbangan, transit hotel, video, lokasi lengkap), `maxPerPage = 200`. → DTO kartu ramping untuk list.
- **A9** `/api/account/bookings` memakai `OR … IN (subquery)` sehingga indeks tak terpakai, plus 1 query per baris peran jamaah (`customer/orders.go:119-124,237-276`). → `UNION` dua cabang terindeks.
- **A10** Dashboard agen menghitung jumlah jamaah dengan memuat seluruh daftar, `OR` lintas tabel, cek agen dua kali, ±8 query berurutan (`agen/kinerja.go:263-289`, `portal_app.go:123-201`). → `COUNT` khusus, query paralel.
- **A11** `/api/public/brand`, `/categories`, `/itineraries/{id}` tanpa cache; brand membaca 45 kolom termasuk teks panjang (`brand/repository.go:23,115-126`). → Cache brand per domain 60 dtk di memori, pilih kolom publik, `PublicCache(300)`.
- **A12** Indeks komposit yang belum ada (perlu migrasi baru):
  - `schedules (brand_id, status, berangkat_tanggal)`
  - `bookings (status, customer_account_id)`
  - `bookings (schedule_id, status, is_seat_blocked)`
  - `payments (booking_id, status, jumlah)`
  - `booking_pax (jamaah_id, pax_status)`
  - `transaksi_komisi (akun_penerima_id, jenis, dibatalkan_at)`

### Sudah baik
`PublicCache(60)` di `/api/public/v1/*`; list publik memuat relasi dengan `IN (...)` (bukan N+1); pool DB terkonfigurasi; worker lepas seat memakai indeks; notifikasi berindeks + `LIMIT`.

## Web (`azhan-booking`)

### Dampak tinggi
| # | Temuan | Bukti | Perbaikan |
|---|---|---|---|
| W1 ✔ | Semua route dinamis; brand dibaca via `headers()`, consent via `cookies()` di root layout. Build: hanya `/_global-error` yang statis | `src/app/layout.jsx:32,61,109`, `src/proxy.js:85-122` | `cacheComponents` + `'use cache'` dengan brand/host sebagai argumen, atau rewrite ke `/_site/[host]/…` + `revalidate`; bagian per pengunjung (badge akun, consent, konsultan) pindah ke klien |
| W2 ✔ | Tanpa `loading.jsx` & `Suspense` → halaman menunggu await paling lambat; klik kartu paket tanpa umpan balik | `loading.jsx` = 0, `Suspense` = 0 | `loading.jsx` skeleton di `/paket`, `/paket/[...slug]`, `/jadwal-umroh`, `/blog/[slug]`; `Suspense` untuk testimoni/galeri/itinerary |
| W3 ✔ | Itinerary di detail paket `cache: 'no-store'` | `src/lib/api.js:85` | `next: { revalidate: 300, tags: [...] }`; sama untuk `getKonsultan` (`konsultan.js:54`) |
| W4 ✔ | Poster detail `<img>` mentah ukuran penuh, tanpa dimensi/prioritas; poster di `<dialog>` tertutup tetap diunduh; logo tanpa width/height (CLS) | `PaketDetail.jsx:218`, `PosterPaket.jsx:30`, `SiteHeader.jsx:47`, `PackageCard.jsx:65` | `next/image` + `sizes`; poster LCP `fetchPriority="high"`; gambar modal dirender saat dibuka; logo diberi dimensi |

### Dampak sedang
- **W5** `getPackages` tanpa React `cache()` → klasifikasi & merge path diulang 2–3× per request (`api.js:14-36`). Periksa juga respons per halaman tetap < 2 MB agar Data Cache Next mau menyimpannya.
- **W6** Waterfall await berurutan di Beranda (`page.jsx:62-83`) dan `/paket` (`paket/page.jsx:47-58`). → satu `Promise.all`.
- **W7** Hero Hana pakai CSS background (baru ditemukan setelah CSS 106 KB diparse) dan H1 mulai `opacity: 0` (`globals.css:573-618`). → `next/image` `preload` + animasi H1 transform saja.
- **W8** Daftar paket diserialisasi dua kali ke payload RSC (`page.jsx:135,138`, `paket/page.jsx:77`).
- **W9** File `public/hero/*` tanpa `Cache-Control` panjang; logo legal PNG kebesaran (`kemenhaj.png` 162 KB untuk ±64 px).

### Dampak rendah
CSS global memuat tema CMS admin (`globals.css:1644-1959`); `BandingkanModal` bisa `next/dynamic`; `/media` tanpa ETag; ±17 file di `public/` tidak dipakai (±1 MB); `--font-naskh` tidak pernah didefinisikan (teks Arab jatuh ke font sistem).

### Sudah baik
`next/image` + AVIF/WebP di hero non-Hana; Data Cache 5 menit untuk paket/promo; cache CMS 30 dtk dengan dedupe; `next/font` dengan `display: swap`; tracking hanya setelah consent; `qrcode-generator` hanya di server.

## App (`azhan-mobile`, Hana)

### Dampak tinggi
| # | Temuan | Bukti | Perbaikan |
|---|---|---|---|
| M1 ✔ | Katalog menunggu brand → paket (paginasi serial) → **semua** itinerary sebelum tampil | `apps/hana/src/data/paket.ts:304-333`, `packages/core/src/catalog.ts:104-108` | Set `siap` setelah paket diterima; itinerary dimuat belakangan / saat detail dibuka; atau ERP sertakan judul hari di list |
| M2 | Tanpa cache tersimpan: brand, katalog, promo, Haramain mulai kosong tiap cold start; header cache ERP tidak dipakai `fetch` RN | `paket.ts:295`, `lib/brand.ts:6-16`, `lib/muat.ts` | Stale-while-revalidate: simpan JSON ke file/MMKV, render langsung, refresh di background |
| M3 ✔ | `useMuat` state per komponen → request duplikat lintas layar; waterfall 2 tingkat di Detail Booking (1+6 request) dan akun agen; N+1 `BarisKelengkapan` (2 request per kartu); semua diulang tiap tab Booking difokus | `lib/muat.ts:14-41`, `DetailBooking.tsx:34,187-194`, `BarisKelengkapan.tsx:17-18`, `perjalanan.tsx:37-42` | Query cache bersama (mis. `@tanstack/react-query` + persist, `staleTime` ±5 mnt); ringkasan kelengkapan masuk ke daftar pesanan di ERP; refresh fokus hanya bila stale |

### Dampak sedang
- **M4** 6 font dimuat runtime (±400 KB, Noto Naskh Arabic 200 KB) dan render ditahan (`_layout.tsx:24-42`). → Tanam lewat plugin `expo-font`; minimal muat Naskh hanya di layar doa/panduan.
- **M5** Barrel `@azhan/core` mengevaluasi ±197 KB data doa/panduan saat start, ditambah `icon-data.ts` 162 KB. → Subpath export / tree shaking.
- **M6** Ikon via `SvgXml` diparse ulang tiap mount (±40 ikon di Beranda) (`components/Icon.tsx:17-21`). → Komponen per ikon atau cache hasil parse.
- **M7** Tab Paket `ScrollView` + `.map` tanpa virtualisasi, tiap kartu punya animasi tak berujung (`(tabs)/paket.tsx:59,117`, `PaketCard.tsx:32-36`). → `FlatList`/FlashList.
- **M8** Foto Haramain dan banner promo ukuran penuh untuk thumbnail. → Butuh varian thumbnail dari ERP (A5).

### Dampak rendah
SecureStore menyimpan data non-rahasia besar (tabungan s.d. 200 catatan); animasi tak berujung berjalan saat tab tak dilihat; refresh saat fokus tanpa throttle; dependensi tak terpakai (`@expo/ui`, `expo-glass-effect`, `expo-symbols`, `expo-device`, `expo-web-browser`, `react-native-gesture-handler`); foto cadangan JPG 190–300 KB bisa WebP. Bila `expo-updates` ditambah: jangan menunggu jaringan saat start.

### Sudah baik
Splash tidak menunggu jaringan; sesi dipulihkan optimistis; tanpa React Context besar, React Compiler aktif; semua gambar `expo-image`; kartu paket pakai `brosur_thumb_url`; list panjang di Akun/Doa sudah `FlatList`.

## Urutan kerja yang disarankan

| Tahap | Isi | Upaya | Dampak |
|---|---|---|---|
| 1. Cepat (1 hari) | A1, A2, A4, A5 header cache; W3; M1 | Sangat kecil | Tinggi di ketiga sisi |
| 2. Gambar | Varian thumbnail WebP saat upload di ERP (A5) → dipakai web (W4) dan app (M8); kompres poster lama | Sedang | Tinggi (byte terbesar) |
| 3. Web | W2 `loading.jsx`/`Suspense`, W5, W6, W7, W9 | Kecil–sedang | Sedang–tinggi |
| 4. App | M2 + M3 cache bersama tersimpan; M4 font; M7 list | Sedang | Tinggi (kesan "app cepat") |
| 5. API | A3 detail publik, A11 cache brand, A12 indeks (migrasi), A6 worker | Sedang | Sedang, naik seiring data |
| 6. Arsitektur web | W1 render statis/ISR per brand | Besar | Tertinggi untuk TTFB |

Catatan kontrak: A8 (DTO list ramping) dan penambahan judul hari itinerary di list menyentuh `/api/public/v1/packages`. Menambah field aman; mengubah/menghapus field yang ada adalah breaking untuk `azhan-booking` dan app — harus lewat endpoint/parameter baru.

## Baseline terukur (2026-11-28, lokal)

Lingkungan: laptop dev, MySQL Laragon lokal, data dev (Hana `brand=3`, 15 paket). Jaringan lokal tanpa throttle, jadi angka waktu jauh lebih cepat dari HP pelanggan. Yang bisa dibandingkan sebelum/sesudah: **jumlah request, urutan tunggu, byte, dan waktu server**. Skrip: `ukur-api.sh`, `ukur-web.sh` (scratchpad sesi).

### API (15 kali per endpoint, setelah pemanasan, via `127.0.0.1`)
| Endpoint | Byte | Jika gzip | Median | Maks |
|---|---|---|---|---|
| `/api/public/brand?domain=…` | 879 | 555 | 2,5 ms | 3,3 ms |
| `/api/public/v1/packages?brand=3&per_page=200` | 30.992 | 3.121 (−90%) | 8,3 ms | 13,2 ms |
| `/api/public/v1/packages/113?brand=3` | 2.019 | 1.086 | 10,8 ms | 11,9 ms |
| `/api/itineraries/5` | 2.868 | 951 | 3,7 ms | 4,6 ms |
| `/api/public/v1/promosi?brand_id=3` | 308 | 195 | 2,5 ms | 3,0 ms |
| `/api/public/v1/haramain?brand=3` | 3.557 | 1.662 | 2,5 ms | 3,2 ms |
| `/api/public/v1/sitemap-feed?brand=3` | 1.881 | 320 | 8,1 ms | 10,2 ms |
| `/api/public/categories?brand_id=3` | 730 | 346 | 3,5 ms | 5,1 ms |

- Detail paket (10,8 ms) paling lambat walau payload kecil: cocok dengan ±12 query berurutan (A3). Dengan MySQL lokal, RTT DB ±0,1 ms; di server dengan DB terpisah efek A1/A3 berlipat.
- Respons tanpa `Content-Encoding` walau klien kirim `Accept-Encoding: gzip, br` (A4 terkonfirmasi). List paket bisa turun 90%.
- `/uploads/*`: hanya `Last-Modified`, tanpa `Cache-Control` (A5 terkonfirmasi).
- **0 dari 15 paket punya `brosur_thumb_url`**, jadi kartu app jatuh ke poster penuh (1080×1350, 99–182 KB per poster).
- Gambar lain: foto Haramain 98–221 KB, banner promosi 128–134 KB, foto hotel s.d. 139 KB, logo maskapai s.d. 33 KB.
- Temuan dev: `localhost` di Windows menambah **±210 ms per koneksi** (coba IPv6 `::1` dulu; `127.0.0.1` = 1 ms). Hanya lingkungan dev, tapi membuat pengukuran via `localhost` menyesatkan.

### Web (build production `next start`, Hana)
Build: semua route bertanda `ƒ (Dynamic)`; HTML dikirim `Cache-Control: private, no-cache, no-store` (W1 terkonfirmasi).

| Halaman | HTML | HTML gzip | TTFB median (10×) | Request | Total byte | Gambar |
|---|---|---|---|---|---|---|
| `/` | 188 KB | 27 KB | 37 ms | 27 | 1.206 KB | 965 KB (13) |
| `/paket` | 125 KB | 17 KB | 34 ms | 21 | 708 KB | 476 KB (5) |
| `/paket/umroh/12-januari-2027-9-hari` | 154 KB | 24 KB | 73 ms | 29 | 600 KB | 346 KB (5) |
| `/jadwal-umroh` | 147 KB | 17 KB | 38 ms | — | — | — |
| `/tentang-kami` | 79 KB | 14 KB | 15 ms | — | — | — |

- Request pertama setelah server start: 639 ms (cache dingin).
- Detail paket TTFB 2× halaman lain: itinerary `no-store` dari ERP (W3).
- Beranda (375 px): LCP = gambar hero CSS `hero-haram-800.webp`; request gambar baru dimulai 242 ms, setelah TTFB 131 ms (menunggu CSS — W7). Gambar terbesar: poster 161 KB, `kemenhaj.png` 158 KB (tampil ±64 px), hero 1600 156 KB, dua banner 128–134 KB. JS 172 KB, CSS 22 KB, font 20 KB. CLS 0.
- Detail paket: poster 1080×1350 (161 KB) tampil 320 px; **poster kedua 160 KB diunduh di dalam `<dialog>` yang tertutup** (lebar tampil 0) — W4 terkonfirmasi.
- Perkiraan di "Slow 4G" (±1,6 Mbps, RTT ±150 ms): Beranda 1,2 MB ≈ 6 dtk sampai semua gambar selesai; detail ≈ 3 dtk. Belum diukur dengan Lighthouse.

### App Hana (versi web Expo, data lewat API lokal)
Urutan request saat Beranda dibuka (ms sejak navigasi):

| Request | Mulai | Selesai |
|---|---|---|
| `brand` | 853 | 857 |
| `haramain` | 859 | 863 |
| `account/me`, `foto`, `notifikasi/jumlah` | 902–914 | 949–1.429 |
| `packages?page=1&per_page=50` | 954 | 985 |
| `promosi` | 955 | 973 |
| `itineraries/5`, `itineraries/14` | 1.089 | 1.104 |
| Katalog "Keberangkatan Terdekat" tampil | — | **1.265** |
| Foto Haramain, banner, poster | 1.065–1.274 | 1.341–1.455 |

- Rantai tunggu katalog terkonfirmasi 3 tingkat: brand → paket → itinerary → tampil (M1). Di jaringan HP (RTT 200–600 ms) rantai ini saja ≈ 0,6–1,8 dtk sebelum katalog muncul, belum termasuk unduh gambar.
- 16 request di Beranda; 7 di antaranya gambar asli (±100–220 KB per gambar).
- Bundle JS web dev 9,4 MB (dev, tidak dimampatkan; bukan acuan ukuran app native).
- Cold start di HP sungguhan belum diukur: tidak ada HP/emulator tersambung (`adb` tidak tersedia).

## Tahap 1 dikerjakan (2026-11-28)

| # | Perubahan | Repo |
|---|---|---|
| A1 | DSN `interpolateParams=true` | erp-azhan `internal/shared/config.go` |
| A2 | `fetchFull` menutup `rows` sebelum query lanjutan | erp-azhan `internal/schedule/repository.go` |
| A4 | `middleware.Compress(5, "application/json")` | erp-azhan `cmd/api/main.go` |
| A5 | `/uploads` publik: nama UUID `public, max-age=31536000, immutable`; nama lain (logo seed) `public, max-age=86400` | erp-azhan `internal/media/handler.go` + tes |
| W3 | Itinerary detail paket `revalidate: 300` (CMS admin tetap `no-store`) | azhan-booking `src/lib/api.js` |
| M1 | Katalog app tampil setelah paket diterima; itinerary menyusul di background (bagian Itinerary di detail disembunyikan selama belum ada) | azhan-mobile `apps/hana/src/data/paket.ts`, `app/detail-paket/[id].tsx` |

Verifikasi: `go build ./...` dan `go test ./...` lulus (37 paket ok); `npm run build` + `npm test` azhan-booking lulus (955 pass, 0 fail); `npm run typecheck` azhan-mobile lulus.

### Hasil sesudah
| Ukuran | Sebelum | Sesudah |
|---|---|---|
| Transfer list paket (web/app ke ERP, gzip) | 30.992 B | 3.162 B (−90%) |
| Transfer itinerary web → ERP | 2.868 B | 943 B |
| `Cache-Control` poster/foto `/uploads` | tidak ada | 1 tahun, immutable |
| Isi JSON list paket | — | identik byte-per-byte dengan sebelum |
| Panggilan web → ERP `/api/itineraries` untuk 11 kunjungan detail | 11 | 1 |
| Detail paket API, median lokal | 10,8 ms | 7,6 ms (DB lokal; selisih lebih besar bila DB di host lain) |
| App: katalog Beranda tampil sejak request pertama, proxy jeda 300 ms/request (2 kali ukur) | 1.082 / 1.143 ms | 817 / 826 ms (−±300 ms, satu round trip) |

TTFB halaman web hampir sama (lokal 35–70 ms); penghematan nyata ada di byte dan beban ERP. Tahap 1 belum menyentuh gambar ukuran asli (tahap 2) dan render dinamis web (tahap 6).

## Tahap 2 dikerjakan (2026-11-28): varian gambar

**Cara kerja.** `GET /uploads/{kategori}/{berkas}?w=400|600|800|1200` menyajikan salinan selebar itu. Varian dibuat sekali saat pertama diminta (±0,1–0,3 dtk), disimpan di `uploads/_varian/` (tidak bisa diakses langsung), lalu disajikan dari disk (±2 ms) dengan cache 1 tahun. URL di respons API tidak berubah; klien yang menambahkan `?w=`. Tanpa skema DB baru, tanpa dependensi baru (JPEG/PNG dari pustaka Go yang sudah ada; PNG bila gambar transparan).

- Hanya kategori foto/poster: `schedule-brosur`, `banner-promosi`, `haramain`, `ulasan-haramain`, `hotel-photos`. Logo tidak.
- Berkas asli disajikan bila gambar sudah lebih kecil dari lebar diminta, varian tidak ≥15% lebih kecil, format tidak didukung, atau sumber > 40 MP.
- Orientasi EXIF foto ponsel diikuti (varian tidak miring).
- Paling banyak 2 gambar didekode bersamaan; satu varian tidak dibuat dua kali.
- Lebar yang dipakai klien:
  - Web: poster di kartu 600, foto hotel 600, banner 800, poster di detail paket (non-Hana) 800. Poster di jendela "Lihat poster" Hana tetap asli tapi `loading="lazy"` sehingga tidak diunduh sebelum dibuka.
  - App: kartu paket, foto hotel, kartu Haramain 800; hero detail paket & Haramain 1200; thumbnail galeri/ulasan dan ringkasan booking 400; banner 1200; penampil foto penuh dan tautan bagikan poster tetap asli.
- Logo legal statis dikecilkan (tetap PNG, nama sama): `kemenhaj.png` web 158→17 KB (192 px), app 158→13 KB (160 px); `kan.png` dan `nusuk.png` ikut.

### Hasil
Ukuran per lebar (31 gambar data dev):

| | Asli | w400 | w600 | w800 | w1200 |
|---|---|---|---|---|---|
| Total | 4.033 KB | 803 KB | 1.538 KB | 2.384 KB | 3.598 KB |
| Poster 1080×1350 (WebP ±160 KB) | 160 | 38 | 74 | 115 | asli |
| Foto Haramain (±200 KB) | 221 | 18 | 39 | 66 | 138 |
| Banner 1200×600 | 133 | 18 | 33 | 51 | asli |

Halaman web (375 px, gambar setelah gulir ke bawah):

| Halaman | Total sebelum | Total sesudah | Gambar sebelum | Gambar sesudah |
|---|---|---|---|---|
| `/` | 1.206 KB | 788 KB | 965 KB | 536 KB |
| `/paket/umroh/12-januari-2027-9-hari` | 600 KB | 342 KB | 346 KB | 92 KB |

- Detail paket Hana: poster di jendela tertutup (160 KB) tidak lagi diunduh.
- `/paket` sesudah: 315 KB (pembanding sebelum 708 KB diukur tanpa gulir, jadi tidak setara).

App Beranda (12 gambar yang tampil): 1.607 KB → 1.163 KB (−28%). Poster kartu memakai 800 karena kartu ±330 pt (≈990 px di layar 3×); memakai 600 akan menjadi ±870 KB (−46%) dengan poster sedikit kurang tajam di HP berlayar padat.

## Tahap 3 dikerjakan (2026-11-28): web

| # | Perubahan | Berkas (azhan-booking) |
|---|---|---|
| W5 | `getPackages` dibungkus React `cache()`: layout, `generateMetadata`, dan page dalam satu request memakai hasil yang sama | `src/lib/api.js` |
| W6 | Beranda: 7 pemanggilan berurutan (paket, destinasi, konten, testimoni, galeri, banner, origin) jadi satu `Promise.all`; `/paket`: 5 jadi satu | `src/app/page.jsx`, `src/app/paket/(daftar)/page.jsx` |
| W7 | Foto hero Hana di-preload: dikirim di header HTTP `Link` (800 untuk ≤767 px, 1600 untuk ≥768 px, sama dengan CSS) | `src/lib/heroHaram.js`, `HeroHana`, `HeroHalamanHana`, `LayarFokus` |
| W9 | `public/hero/*` dan `hero-makkah.webp`: `Cache-Control: public, max-age=604800, stale-while-revalidate=86400` (sebelumnya `max-age=0`) | `next.config.mjs` |
| W2 | Skeleton `loading.jsx` untuk `/paket` dan `/jadwal-umroh` (komponen `ui/Skeleton` + `paket/DaftarPaketMemuat`) | `src/app/paket/(daftar)/`, `src/app/jadwal-umroh/(daftar)/` |

**Keputusan W2.** `loading.jsx` membuat respons selalu mulai dengan status 200; redirect 301/308 dan 404 sesudahnya hanya jadi meta tag (soft redirect/soft 404). Karena itu:
- Tidak dipasang di `/paket/[...slug]` dan `/blog/[slug]` (redirect slug lama dan 404 harus tetap status HTTP asli).
- Halaman daftar dipindah ke route group `(daftar)` karena `loading.jsx` di folder `/paket` juga membungkus `/paket/[...slug]`. URL tidak berubah.
- Diverifikasi sesudah perubahan: `/paket/113` → 308 ke path paket; `/paket/tidak-ada-xyz` 404; `/blog/tidak-ada-xyz` 404; `/paket` dan `/jadwal-umroh` 200; skeleton hanya ada di stream `/paket`, tidak di detail.
- `/tentang-kami` dan halaman informasi lain tidak diberi skeleton.

Verifikasi: `npm run build`, `npm test` (957 pass, 0 fail), `npm run check:design` OK. Screenshot skeleton di 375 px dan 1440 px, dan halaman `/paket` asli sesudah stream selesai.

### Hasil (A/B lokal, 30 permintaan per halaman, dua putaran)
| Halaman | Sebelum tahap 3 (median byte pertama) | Sesudah tahap 3 |
|---|---|---|
| `/` | 67 / 67 ms | 79 / 76 ms |
| `/paket` | 58 / 54 ms | 59 / 68 ms |
| `/jadwal-umroh` | 75 / 70 ms | 70 / 79 ms |

- **Tidak ada perbaikan TTFB yang terukur di lokal**; selisihnya dalam rentang noise mesin (satu putaran sesudah sempat 36–42 ms saat server sudah lama hangat).
- Sebabnya: data paket sudah di cache 5 menit dan CMS 30 detik, jadi pemanggilan yang diparalelkan hampir semuanya kena cache. Byte pertama ≈ selesai karena layout bersama (brand, akun, nav) selesai lebih dulu sebelum batas skeleton.
- Manfaat yang tersisa dan belum terukur di lokal:
  - Permintaan dengan cache dingin (setiap 5 menit per brand): Beranda menunggu 1 putaran paralel, bukan 7 berurutan.
  - Navigasi klik ke `/paket` dan `/jadwal-umroh` langsung menampilkan skeleton.
  - Foto hero mulai diunduh bersamaan dengan HTML di jaringan lambat.
  - Kunjungan ulang tidak memvalidasi ulang aset `/hero`.

## Tahap 4 dikerjakan (2026-11-28): app Hana

| # | Perubahan | Berkas (azhan-mobile/apps/hana) |
|---|---|---|
| M2 | Salinan disk untuk data publik: brand, katalog (paket + judul hari itinerary, maks 3 hari), banner promosi, daftar Haramain. Cold start menampilkan salinan, lalu menyegarkan di belakang tanpa spinner. Gagal segar tetap menampilkan salinan. Kunci per domain brand. | `src/lib/cacheDisk.ts` (baru), `src/lib/brand.ts`, `src/data/paket.ts`, `src/data/promo.ts`, `src/data/haramain.ts` |
| M1+ | Permintaan paket berangkat bersamaan dengan brand memakai id brand tersimpan (tanpa menunggu respons brand) | `src/data/paket.ts`, `src/data/promo.ts` |
| M3 | `useMuat` jadi cache bersama: satu permintaan per kunci untuk semua layar, hasil < 60 detik dipakai ulang, muat ulang manual / `versi` tetap memaksa ambil | `src/lib/muat.ts` |
| M4 | Font Arab (±200 KB) tidak lagi menahan splash: dimuat di belakang setelah layar pertama | `src/app/_layout.tsx` |
| M7 | Tab Paket memakai `FlatList` (render awal 4 kartu) | `src/app/(tabs)/paket.tsx` |

- Tanpa dependensi baru: memakai `expo-file-system` dan `expo-font` yang sudah ada; di web memakai `localStorage`.
- Token dan data akun tetap hanya di SecureStore; salinan disk hanya data publik.
- Tidak diubah: penyegaran tab Booking setiap fokus (sengaja, agar kartu dan detail selalu sama); N+1 kelengkapan per kartu booking butuh endpoint ringkasan di ERP.
- Font Figtree belum ditanam di build (plugin `expo-font`): nama keluarga font di iOS berbeda dari nama `@expo-google-fonts`, perlu diuji di perangkat dulu.

Verifikasi: `npm run typecheck` lulus. `npm run lint` gagal di `src/data/agen.ts:410` (`set-state-in-effect`), galat yang sama sudah ada sebelum perubahan ini.

### Hasil (app versi web, proxy jeda 300 ms per permintaan)
| Ukuran | Tanpa salinan (start pertama) | Dengan salinan (start berikutnya) |
|---|---|---|
| Katalog Beranda tampil | 868 ms setelah permintaan pertama | sebelum respons jaringan apa pun (166 ms setelah permintaan pertama, sisa waktu render bundle dev) |
| Permintaan paket mulai | setelah respons brand (+319 ms) | bersamaan dengan brand (+3 ms) |

- Tab Paket: pencarian "millen" → 6 paket, hanya 4 kartu dirender awal; fokus kolom cari tetap saat mengetik.
- Belum diukur di HP sungguhan (waktu buka dingin, efek font).

### Bug yang ditemukan saat mengukur
API ERP crash saat start bila MySQL mati, padahal log menyatakan "degraded mode": worker arsip jadwal tetap jalan dengan `db` nil (`internal/schedule/arsip.go:24`, dipanggil dari `cmd/api/main.go:152`).
