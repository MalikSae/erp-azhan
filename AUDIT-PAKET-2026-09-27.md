# Audit End-to-End Fitur Paket Azhan

Tanggal: 27 September 2026. Repositori: `erp-azhan` dan `azhan-microsite`, branch `dev-malik`.

## Tindak lanjut perbaikan — 28 September 2026

Seluruh 16 temuan di bawah telah ditangani pada kode lokal kedua repositori. Bagian audit setelah bagian ini adalah rekaman kondisi sebelum perbaikan, termasuk batas pengujian saat audit awal. Perbaikan belum dipush atau dideploy.

| Temuan | Perbaikan |
| --- | --- |
| F01 | Edit mengunci jadwal dalam transaksi dan menghitung sisa kursi dari kapasitas terbaru dikurangi alokasi yang sudah ada. Nilai sisa dari form lama tidak menimpa booking baru. Perubahan kapasitas basi ditolak. Koreksi kursi memerlukan nilai awal dan alasan, serta tidak boleh melebihi kapasitas setelah booking aktif. |
| F02 | Perubahan brand ditolak ketika jadwal memiliki booking; scope brand diperiksa kembali di dalam transaksi. |
| F03 | JSON-LD menggunakan serializer yang meng-escape karakter `<`, termasuk pada layout dan detail paket. |
| F04 | Detail publik wajib menyertakan brand, server microsite memeriksa kepemilikan paket, dan proxy booking memverifikasi brand berdasarkan hostname. Payload brand berbeda ditolak. |
| F05 | Harga infant kosong berarti belum tersedia; harga nol eksplisit berarti gratis. UI dan backend mengikuti aturan yang sama. |
| F06 | API mengembalikan `effective_minimal_dp` dari override paket atau default brand. Wizard tidak membuat nilai DP sendiri dan mempertahankan nilai nol. |
| F07 | Paket dummy di katalog/detail serta hasil sukses booking dummy dihapus. Kegagalan data tidak menghasilkan paket atau transaksi palsu. |
| F08 | Klaim tiket mengikuti data konfirmasi; nomor izin tidak lagi memakai nilai rekaan. |
| F09 | Paket habis menampilkan status kuota penuh dan kontak admin; halaman booking tidak membuka wizard. |
| F10 | Penghematan dan harga coret Quad hanya tampil pada kamar Quad. |
| F11 | Filter kategori memakai ID/slug kategori dari API; pemilihan kategori juga divalidasi terhadap brand di backend. |
| F12 | Progress kursi memakai proporsi aktual tanpa batas minimum visual palsu. |
| F13 | Status pembuatan yang kosong dinormalisasi menjadi `draft`. |
| F14 | DP negatif atau melebihi harga kamar termurah ditolak. Perubahan default DP brand memeriksa paket yang mewarisinya; booking memeriksa ulang DP efektif. |
| F15 | Form memiliki ringkasan error yang menerima fokus, tautan field, asosiasi label, dan pesan error. Modal mengunci serta mengembalikan fokus. Kontrol jumlah jamaah mempunyai label aksesibel. |
| F16 | Kegagalan pemuatan katalog/home memiliki pesan error dan tombol coba lagi, terpisah dari hasil filter kosong. |

### Bukti verifikasi setelah perbaikan

- `ERP_TEST_DB=1 go test ./...`: exit code 0. Semua paket test lulus. Test baru memeriksa pelestarian kursi terbaru, larangan pindah brand, konflik kapasitas, proyeksi DP efektif (default/override/nol), validasi DP, dan penolakan infant tanpa harga. Pengujian integrasi menggunakan transaksi rollback.
- `node --test tests/package-policy.test.mjs`: `tests 7`, `pass 7`, `fail 0`, `skipped 0`, exit code 0. Meliputi JSON-LD, infant/DP nol, promo kamar, identitas kategori, kepemilikan paket, binding brand proxy, dan kegagalan resolver.
- Build frontend ERP: Master `1981 modules transformed`, Travel `1950 modules transformed`, exit code 0.
- Build microsite: `Compiled successfully`, `Generating static pages (31/31)`, exit code 0.
- `git diff --check` kedua repositori: exit code 0; hanya peringatan normalisasi LF/CRLF.

Peringatan nonblocking: ukuran bundle dashboard melebihi 500 kB, konvensi middleware Next.js terdepresiasi, dan Node mendeteksi modul ESM helper saat test. Tidak ada migrasi database.

### Batas verifikasi dan penerapan

Percobaan browser terakhir masih gagal karena koneksi browser timeout. Kelulusan visual 375px/1440px, interaksi keyboard hasil render, serta simpan form lewat sesi admin belum dapat diklaim. Tes unit/integrasi dan build di atas sudah berjalan; proses backend lokal port 9090 belum direstart ke kode baru, sehingga pengujian browser terhadap layanan lama tidak membuktikan kontrak API terbaru.

Rilis backend, dashboard, dan microsite secara terkoordinasi. Kontrak detail publik sekarang `GET /api/schedules/{id}?brand={id}`. Konsumen baru memerlukan `effective_minimal_dp`; ketika field itu belum tersedia, wizard menahan kelanjutan booking. Koreksi manual kursi memerlukan `expected_seat_sisa` dan `reason`. Tidak ada perubahan massal pada data bisnis lama; konfigurasi lama yang tidak valid ditolak saat dipakai atau disimpan.

## Kesimpulan

Fondasi alur paket tersedia, tetapi masih ada masalah integritas kursi, isolasi brand pada tampilan publik, keamanan rendering, serta kesesuaian harga dan informasi penjualan. Ditemukan **16 temuan: 7 P1 dan 9 P2**. P1 perlu diprioritaskan sebelum memperluas transaksi publik. P2 perlu diperbaiki pada iterasi berikutnya.

Audit ini menelusuri skema, handler, repository, dashboard Master/Travel, katalog, detail, compare, dan batas integrasi dengan booking. Pemeriksaan runtime menggunakan API publik dan browser lokal. Tidak ada perubahan kode aplikasi, pengubahan data bisnis, pengiriman booking, atau transaksi pembayaran.

## Cakupan dan batas verifikasi

- API lokal port 9090 aktif dan database terhubung. Dashboard port 5173/5174 merespons HTTP 200.
- Browser dapat membuka katalog Hana, detail paket, dan langkah pertama formulir booking. Dashboard Master mengarah ke login; alur admin diperiksa melalui kode, tanpa akun/sesi admin.
- Screenshot browser berulang kali gagal karena timeout CDP. Override mobile juga tidak menghasilkan ukuran viewport yang dapat diandalkan. Karena itu audit tidak mengklaim kelulusan visual pada 375px/1440px, kontras hasil render, atau ketiadaan overflow.
- `go test -v ./internal/schedule ./internal/selfbooking ./internal/booking` selesai dengan exit code 0. Delapan unit test self-booking lulus. Enam integration test self-booking dan dua integration test booking dilewati karena `ERP_TEST_DB` tidak diaktifkan. Paket `internal/schedule` belum memiliki test.
- Temuan ditandai **runtime** bila diamati pada layanan/browser, atau **kode** bila ditelusuri secara statis. Skenario perubahan data dan eksploit tidak dieksekusi pada database kerja.

## Alur yang dipetakan

1. Master membuat atau mengedit paket, harga, brand, jadwal, hotel, maskapai, fasilitas, promo, dan kuota.
2. API memvalidasi payload kemudian menyimpan `schedules` beserta relasi add-on dan hotel transit dalam transaksi.
3. Daftar publik mengambil paket `published` per brand dengan keberangkatan minimal 14 hari dari hari ini.
4. Microsite menentukan identitas brand berdasarkan hostname, lalu menampilkan katalog dan detail.
5. Wizard booking mengirim pilihan pax/kamar. Backend mengunci jadwal, memeriksa brand/status/cutoff, menghitung harga, dan mengurangi kursi.
6. Travel menangani jamaah, booking, pembayaran, dan kesiapan perjalanan. Brand booking diperoleh melalui relasinya dengan jadwal.

## Temuan P1

### F01 — Edit paket dapat mengembalikan kursi yang sudah terpesan

**Bukti kode:** ERP `frontend/master-dashboard/src/pages/ScheduleFormPage.jsx:225`, `:586`, `:600`; `internal/schedule/repository.go:327`; `internal/schedule/handler.go:328`.

Form menyimpan kuota terisi dari saat halaman dibuka, lalu selalu mengirim nilai absolut `seat_sisa`. Repository menuliskan nilai itu langsung. Jika admin membuka paket dengan 45 kursi, satu booking masuk sehingga sisa menjadi 44, lalu admin hanya mengubah nama dan menyimpan, sisa dapat kembali menjadi 45. Validasi rentang tidak mendeteksi data lama. Payload update yang tidak mengirim `seat_sisa` juga mendapat default `seat_total`.

**Dampak:** overselling dan ketidaksesuaian jumlah booking terhadap persediaan. Endpoint khusus perubahan kursi juga menulis angka absolut tanpa rekonsiliasi booking/hold.

**Perbaikan:** pisahkan perubahan konten dari persediaan. Hitung kapasitas tersedia berdasarkan alokasi aktif dalam transaksi terkunci; gunakan version check untuk konflik edit. Perubahan kapasitas harus menolak nilai di bawah alokasi aktif dan mencatat alasan penyesuaian.

### F02 — Mengganti brand paket memindahkan akses booking lama

**Bukti kode:** ERP `internal/schedule/handler.go:173`, `internal/schedule/repository.go:327`, `internal/booking/repository.go:122`.

Super Admin dapat mengganti `brand_id` tanpa memeriksa keberadaan booking. Akses booking difilter melalui `schedules.brand_id`, sedangkan identitas jamaah tetap terikat ke brand asal. Akibatnya, penggantian brand paket juga mengubah brand yang dapat membaca booking terkait.

**Dampak:** data booking/jamaah dapat muncul kepada admin brand tujuan, hilang dari operasional brand asal, dan tidak selaras dengan pembayaran maupun komisi.

**Perbaikan:** larang penggantian brand setelah ada booking atau transaksi terkait. Jika perpindahan diperlukan, sediakan proses migrasi bisnis khusus dengan validasi dan audit terpisah.

### F03 — JSON-LD detail paket memiliki jalur stored XSS

**Bukti kode:** microsite `src/app/paket/[id]/page.jsx:315`; ERP `internal/schedule/handler.go:308`.

Nama paket dan data lain yang dapat diedit admin masuk ke JSON-LD melalui `dangerouslySetInnerHTML` dengan `JSON.stringify` langsung. JSON.stringify tidak mengubah karakter `<`, sehingga teks yang mengandung penutup elemen script dapat keluar dari konteks JSON-LD saat HTML diparse. Validasi nama hanya memastikan teks tidak kosong. Tidak ditemukan sanitasi pada jalur ini atau CSP di konfigurasi Next.js yang diperiksa.

**Dampak:** admin yang dapat mengedit paket, atau akun admin yang diambil alih, dapat memasukkan markup/script ke halaman publik. CSP dari reverse proxy produksi belum diperiksa; tidak ada payload yang ditanam saat audit.

**Perbaikan:** gunakan serialisasi JSON untuk embedding HTML yang mengubah `<` menjadi escape Unicode sebelum diberikan ke `dangerouslySetInnerHTML`. Tambahkan regression test untuk teks yang mengandung penutup script.

### F04 — Detail dan formulir paket tidak memeriksa brand hostname

**Bukti runtime dan kode:** microsite `src/app/paket/[id]/page.jsx:74`, `src/app/paket/[id]/book/page.jsx:18`; ERP `internal/schedule/handler.go:104`.

Paket **43** berasal dari brand **2 / Alsha**, tetapi `http://hana.azhan.test/paket/43` merespons HTTP 200 dan menampilkan paket tersebut dengan identitas, WhatsApp, dan rekening Hana. Server mengambil detail berdasarkan ID tanpa membandingkan `schedule.brand_id` dengan `x-brand-id`. Metadata mengikuti kesalahan yang sama.

**Dampak:** penawaran bercampur antarbrand dan kontak/rekening salah konteks. Lebih jauh, `src/components/booking/BookingWizard.jsx:715` memprioritaskan `schedule.brand_id` dalam payload, termasuk saat pengecekan nomor telepon pada `:459`. Proxy `src/app/api/public/book/route.js:7` meneruskan brand dari body tanpa mengikatnya ke hostname. Maka wizard pada domain Hana dapat mengirim booking sebagai Alsha. Guard `internal/selfbooking/repository.go:135` memeriksa pasangan brand dan jadwal, tetapi keduanya sudah sama-sama menunjuk Alsha, sehingga guard itu tidak mengatasi ketidakcocokan domain. Ini ditelusuri dari kode; submit booking tidak dilakukan.

**Perbaikan:** validasi pasangan brand dan ID pada endpoint detail dan pada server component sebelum metadata, rekening, atau wizard dirender. Kembalikan 404 untuk ketidakcocokan. Proxy publik harus menetapkan brand dari hostname yang sudah di-resolve secara tepercaya dan menolak brand body yang berbeda.

### F05 — Harga infant di wizard berbeda dari harga yang ditagihkan backend

**Bukti runtime dan kode:** microsite `src/components/booking/BookingWizard.jsx:326`; ERP `internal/selfbooking/repository.go:470`.

API paket **45** mengembalikan `harga_infant: null`. Browser `/paket/45/book` menampilkan **Rp12.000.000** untuk infant. Backend hanya mengisi harga infant jika nilai database valid; jika null, variabel harga tetap **0**.

**Dampak:** estimasi sebelum konfirmasi berbeda Rp12 juta per infant dari perhitungan backend. Harga null belum memiliki arti bisnis yang konsisten.

**Perbaikan:** tentukan apakah null berarti tidak tersedia, perlu konsultasi, atau gratis. Terapkan kebijakan yang sama di API dan UI. Jangan menciptakan harga fallback. Pertahankan nilai 0 secara eksplisit jika gratis memang diizinkan.

### F06 — Wizard membaca field DP yang tidak ada dalam kontrak paket

**Bukti kode:** microsite `src/components/booking/BookingWizard.jsx:327`, `src/app/paket/[id]/book/page.jsx:141`; ERP `internal/selfbooking/repository.go:505`.

Wizard membaca `schedule.dp_amount`, sedangkan API menyediakan `minimal_dp`. Paket diteruskan ke wizard tanpa pemetaan. Estimasi selalu jatuh ke Rp5 juta/pax jika field tersebut tidak ada. Backend menggunakan `schedules.minimal_dp`, kemudian fallback ke `brands.minimal_dp` jika null. Halaman sukses memakai nilai respons backend sehingga angka dapat berubah setelah submit.

**Dampak:** instruksi DP awal salah untuk paket atau brand dengan kebijakan selain Rp5 juta, termasuk DP 0.

**Perbaikan:** expose nilai DP efektif dari backend dan pakai field yang sama di katalog/detail, wizard, invoice, dan validasi pembayaran. Gunakan nullish semantics, bukan `||`, untuk nilai uang yang boleh 0.

### F07 — Detail publik menghidupkan kembali paket dummy ketika API gagal

**Bukti kode:** microsite `src/app/paket/[id]/page.jsx:8`, `:89`.

Jika detail API gagal atau mengembalikan non-200, ID 101 dan 102 diambil dari data dummy. Tidak ada pembatas development. Ini juga berlaku ketika paket asli menjadi draft, diarsipkan, atau dihapus.

**Dampak:** harga, kursi, itinerary, dan status penawaran fiktif tetap muncul beserta metadata publik. Halaman booking tidak memakai fallback yang sama, sehingga alurnya juga dapat berakhir 404.

**Perbaikan:** hapus fallback produksi. Bedakan 404 dengan kegagalan layanan; tampilkan state gagal yang dapat dicoba ulang. Fixture demo harus eksplisit dan terpisah dari domain transaksi.

## Temuan P2

### F08 — Klaim kepastian tiket tidak mengikuti status paket

**Bukti runtime dan kode:** microsite `src/components/paket/PackageDetailClient.jsx:457`, `:1140`; `src/components/booking/BookingWizard.jsx:1077`.

Paket 43 memiliki `is_ticket_confirmed: false`, tetapi browser menampilkan “100% Pasti Berangkat” dan “Tiket terbit resmi”. Teks ditampilkan tanpa kondisi. Terdapat pula fallback nomor izin spesifik pada `src/app/paket/[id]/book/page.jsx:36` saat metadata izin kosong.

**Perbaikan:** tampilkan status faktual sesuai data; klaim jaminan harus memiliki dasar bisnis eksplisit. Sembunyikan nomor izin jika belum tersedia, bukan memakai nomor brand lain/default.

### F09 — Paket habis tetap menawarkan formulir booking

**Bukti kode:** microsite `src/components/paket/PackageDetailClient.jsx:1198`, `src/app/paket/[id]/book/page.jsx:116`, `src/components/booking/BookingWizard.jsx:119`.

CTA hanya mempertimbangkan cutoff tanggal, bukan `seat_sisa`. Wizard menginisialisasi satu pax reguler bahkan ketika sisa kursi 0. Backend memiliki pemeriksaan kapasitas, tetapi pelanggan baru ditolak setelah melewati alur formulir.

**Perbaikan:** gunakan status sold out pada CTA dan entry wizard, tawarkan konsultasi/waitlist, dan tetap validasi ulang kapasitas pada backend saat submit.

### F10 — Penghematan Quad ditampilkan saat kamar lain tidak mendapat diskon

**Bukti kode:** microsite `src/components/paket/PackageDetailClient.jsx:260`.

Jika harga coret lebih rendah dari harga kamar yang dipilih, `savingsText` memakai selisih harga coret dengan harga Quad. Contoh: coret Rp30 juta, Quad Rp28,5 juta, Double Rp33 juta; memilih Double masih dapat menampilkan “Hemat 1,5 Juta”.

**Perbaikan:** batasi promo ke tipe kamar yang memang tercakup, atau simpan harga pembanding masing-masing kamar. Jangan memakai diskon Quad sebagai fallback untuk Double/Triple.

### F11 — Filter kategori mengabaikan kategori paket dari ERP

**Bukti kode:** microsite `src/components/paket/PaketAppClient.jsx:111`, `src/components/home/HomeAppCategories.jsx:5`.

Kategori ditentukan dari kata “turki”, “plus”, dan “haji” pada nama paket, walaupun API menyediakan `category_id` dan objek kategori. Paket “Umroh Plus Dubai” dapat masuk Reguler dan Plus sekaligus; perubahan nama juga dapat mengubah kategori tanpa perubahan master data.

**Perbaikan:** gunakan kategori resmi beserta slug/ID stabil dari API. Pisahkan filter promo dari kategori produk. Pada backend, validasi keterkaitan kategori dengan brand juga diperlukan: `CategoryExists` di `internal/schedule/repository.go:466` hanya memeriksa keberadaan kategori.

### F12 — Progress kursi menampilkan okupansi buatan

**Bukti kode:** microsite `src/components/home/HomeAppCard.jsx:257`, `src/components/SeatProgressBar.jsx:77`.

Kartu memaksa lebar minimal 20%, atau 65% ketika dikategorikan hampir penuh. Detail menampilkan 15% untuk kondisi baru buka. Paket dengan 0/45 kursi terisi tetap terlihat memiliki okupansi; paket kapasitas kecil juga dapat terlihat hampir penuh meskipun persentasenya rendah.

**Perbaikan:** progress harus memakai persentase aktual. Jika ingin indikator dekoratif, jangan tampilkan sebagai progress persediaan. Label kelangkaan harus mengikuti kebijakan yang konsisten.

### F13 — Create tanpa status tidak menggunakan default draft

**Bukti kode:** ERP `internal/schedule/handler.go:314`, `:558`, `internal/schedule/repository.go:270`, `migrations/002_schedules_fix.sql:7`.

Validator menerima status kosong lalu repository memasukkannya secara eksplisit ke kolom ENUM. Default database `draft` tidak dipakai jika string kosong dikirim. Pada SQL mode strict hasilnya error; pada mode permisif dapat tersimpan nilai ENUM tidak valid. Form sekarang mengirim draft sehingga masalah terutama mengenai kontrak API/integrasi.

**Perbaikan:** set status create ke `draft` ketika kosong. Bedakan payload create dan update. Uji create tanpa status dan semua transisi lifecycle yang diizinkan.

### F14 — Minimal DP belum divalidasi secara bisnis

**Bukti kode:** ERP `internal/schedule/handler.go:585`; `internal/selfbooking/repository.go:512`.

`minimal_dp` disalin langsung tanpa menolak nilai negatif atau nilai melebihi harga yang harus dibayar. Backend booking kemudian mengalikan nilai itu dengan jumlah pax reguler. Pembatas UI tidak cukup untuk menjaga kontrak API.

**Perbaikan:** tolak nilai negatif dan tentukan aturan batas DP terhadap harga kamar/tagihan. Terapkan validasi yang sama pada setting DP brand. Arti null dan 0 harus terdokumentasi.

### F15 — Aksesibilitas form dan kontrol belum memadai

**Bukti kode dan browser:** ERP `frontend/master-dashboard/src/pages/ScheduleFormPage.jsx:1599`, `frontend/master-dashboard/src/components/ui/Alert.jsx:14`; microsite `src/components/FacilitiesModal.jsx:34` dan `src/components/booking/BookingWizard.jsx`.

- Error simpan paket ditampilkan dalam alert visual tanpa `role=alert`/live region, pemindahan fokus, dan tautan ke field salah. Form panjang menyulitkan pencarian masalah.
- Modal fasilitas sudah memiliki dialog semantics dan Escape, tetapi belum memindahkan/menahan fokus atau mengembalikan fokus ke pemicu.
- Browser wizard memperlihatkan beberapa tombol bernama identik “+” dan “-”, tanpa nama kamar pada accessible name. Input pencarian katalog juga tidak memiliki label eksplisit.

**Perbaikan:** gunakan error summary yang dapat difokuskan beserta error per field, komponen dialog dengan pengelolaan fokus, dan accessible name seperti “Tambah jamaah kamar Quad”. Verifikasi keyboard dan screen reader setelah perbaikan.

### F16 — Kegagalan katalog tampil sebagai hasil pencarian kosong

**Bukti kode:** microsite `src/app/paket/page.jsx:92`, `src/components/paket/PaketAppClient.jsx:348`.

Kegagalan fetch hanya dicatat ke console dan data tetap `[]`. UI kemudian menyatakan tidak ada paket yang sesuai filter. Pengunjung tidak tahu bahwa layanan sedang gagal dan tidak mendapat tindakan retry yang tepat.

**Perbaikan:** pisahkan state error, katalog kosong, dan hasil filter kosong. Pertahankan filter saat retry. Halaman compare sudah memiliki `errorMessage`; gunakan pola error yang konsisten.

## Kontrol yang sudah tersedia

- Daftar publik mewajibkan parameter brand dan hanya menampilkan published/H-14 ke atas.
- Self-booking memeriksa status dan brand jadwal, cutoff tanggal, kapasitas, serta mengunci jadwal dengan `FOR UPDATE` sebelum reservasi.
- Harga pax dihitung dari database di backend, bukan dipercayakan kepada harga dari browser.
- Relasi add-on dan transit hotel disimpan bersama paket dalam transaksi.
- Foreign key booking menghalangi penghapusan jadwal yang masih direferensikan; handler memetakan pelanggaran FK penghapusan ke HTTP 409.
- Admin brand memiliki pembatas query jadwal berdasarkan brand. Temuan F02 menjelaskan celah konsistensi ketika brand diganti oleh pusat.

## Bukti verifikasi ringkas

```text
GET /api/health
{"status":"ok","database":"connected"}

GET /api/schedules
HTTP 400 {"error":"parameter brand wajib diisi"}

GET /api/schedules/43  [field relevan]
{"id":43,"brand_id":2,"jadwal_nama":"Umroh 10 Hari (Etihad Airways)"}
{"id":43,"is_ticket_confirmed":false,"seat_total":45,"seat_sisa":45}

GET /paket/43 dengan Host: hana.azhan.test
HTTP 200; HTML berisi Hana Tours, paket #43, brand_id 2, dan link /paket/43/book.
Browser menampilkan rekening Hana dan klaim Tiket terbit resmi.

GET /api/schedules?brand=3  [paket 45, field relevan]
{"id":45,"brand_id":3,"harga_infant":null,"minimal_dp":5000000}
Browser /paket/45/book: INFANT (Bayi < 2 Tahun) Rp 12.000.000.

go test -v ./internal/schedule ./internal/selfbooking ./internal/booking
internal/schedule: [no test files]
internal/selfbooking: PASS; 8 unit tests lulus, 6 integration tests SKIP
internal/booking: PASS; 2 integration tests SKIP
```

## Urutan perbaikan dan regression test

1. Tutup F01–F04: update konten bersamaan booking tidak mengubah kursi; pindah brand paket berbooking ditolak; JSON-LD aman; detail/booking lintas domain 404.
2. Satukan kontrak harga dan DP (F05–F06, F14): null, 0, negatif, DP brand, override paket, semua tipe kamar, dan infant harus menghasilkan angka sama sebelum/sesudah booking.
3. Hapus fixture publik dan klaim tidak bersyarat (F07–F08). Uji paket draft, archived, API gagal, dan tiket belum confirmed.
4. Selaraskan sold out, promo kamar, kategori, dan progress kursi (F09–F12). Uji kursi 0, 1, penuh, kapasitas kecil, dan pergantian tipe kamar.
5. Tambahkan test lifecycle schedule (F13), perbaiki error/retry dan aksesibilitas (F15–F16), lalu lakukan pemeriksaan visual aktual pada 375px dan 1440px menggunakan sesi dashboard yang tersedia.

Keputusan bisnis yang perlu ditetapkan sebelum implementasi: arti harga infant kosong/0; batas dan fallback DP; izin perpindahan brand; syarat klaim pasti berangkat; cakupan promo per kamar; dan siapa yang boleh mengubah komisi paket. Audit ini tidak mengubah kebijakan tersebut secara sepihak.
