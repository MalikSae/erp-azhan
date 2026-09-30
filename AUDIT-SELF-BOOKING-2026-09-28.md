# Audit end-to-end self-booking Azhan

Tanggal: 28 September 2026. Repositori: `C:/laragon/www/erp-azhan` dan `C:/laragon/www/azhan-microsite`, branch `dev-malik`.

## Kesimpulan

Ditemukan **21 temuan: 9 P1 dan 12 P2** pada alur self-booking sampai invoice, pembayaran, dan pengelolaan kursi. Lima perilaku bermasalah dibuktikan dengan probe integrasi database yang di-rollback. Temuan lain dibuktikan dari jalur kode, bukan dari eksploit terhadap transaksi pengguna.

Perbaikan audit paket sebelumnya tetap ada di working tree dan menjadi baseline audit ini. Audit ini memperluas pemeriksaan ke invoice, pembayaran, anggota rombongan, pemulihan transaksi, dan aktivasi portal; bukan menyatakan seluruh hasil audit paket sebelumnya telah diverifikasi end-to-end. Pada sesi ini hanya laporan dan probe audit yang ditambahkan, tanpa perbaikan aplikasi atau migrasi.

## Cakupan dan metode

Alur yang diperiksa: katalog/detail → jumlah kamar/pax → identitas PIC/anggota → pemeriksaan nomor/PIN → persetujuan/captcha → transaksi booking publik atau agen → reservasi 24 jam → invoice → login/aktivasi portal → upload dan konfirmasi pembayaran → verifikasi admin → status DP/lunas → expiry dan pembatalan. Referral, pembatasan brand, snapshot harga/komisi, dan pembatasan akses turut ditelusuri.

Sumber: handler/repository selfbooking, payment, booking, crmdeal, portal, shared, migration terkait; BookingWizard, DigitalInvoiceView, halaman invoice/booking agen/pembayaran/login/aktivasi, proxy API, middleware, dan konfigurasi Nginx lokal. Dokumen analisis portal lama dipakai sebagai konteks, lalu dicocokkan ulang dengan kode terkini. Pedoman UI/UX skill dipakai untuk label, fokus, validasi, dan recovery; pencarian dataset skill tidak berjalan karena Python/py tidak tersedia.

Batas penting:

- Browser automation kembali timeout ketika membaca inventaris browser. Tidak ada klaim lulus visual 375px/1440px, kontras hasil render, overflow, atau pengujian keyboard langsung.
- Backend port 9090 sehat, tetapi belum direstart setelah perbaikan audit paket. Hasil runtime tidak dianggap mewakili semua perubahan lokal terbaru.
- Tidak ada booking/pembayaran produksi yang dikirim. Probe memakai fixture transaksi rollback; nomor/kode uji tidak ditampilkan dalam laporan. Sequence auto-increment database dapat bertambah meskipun transaksi rollback.
- Tidak ada brute force invoice/PIN, spam captcha, maupun pengujian beban pada layanan aktif.
- P1 berarti perlu diprioritaskan sebelum memperluas transaksi publik. P2 berarti perlu diperbaiki dalam iterasi berikutnya. Rekomendasi bisnis bukan keputusan kebijakan yang sudah disahkan.

## Temuan P1

### SB01 — Anggota dapat masuk dua booking aktif pada jadwal sama

**Bukti: integrasi rollback.** `internal/selfbooking/repository.go:243` hanya memeriksa duplikasi lintas booking untuk PIC. Pemeriksaan `allJamaahMap` setelah resolusi anggota hanya berlaku di dalam satu booking. Constraint migration 046 juga hanya `(booking_id, jamaah_id)`.

Reproduksi: agen memilih jamaah yang sama sebagai anggota pada dua booking, dengan PIC berbeda, jadwal sama. Kedua transaksi diterima; query menghasilkan dua booking aktif untuk anggota yang sama. Dampak: kursi, tagihan, manifest, dan komisi dapat dihitung dua kali. Perbaikan: periksa semua jamaah hasil resolusi terhadap pax aktif pada jadwal sama di dalam transaksi, dengan pengecualian pembatalan/expiry yang didefinisikan. Uji juga perlombaan dua pemesanan dan anggota yang sudah dibatalkan.

### SB02 — Identitas dewasa dapat memperoleh tarif infant melalui tanggal lahir payload

**Bukti: integrasi rollback.** `internal/selfbooking/infant.go:27` memvalidasi tanggal dari request. `internal/selfbooking/agen_booking.go:32` mengembalikan ID jamaah milik agen tanpa mencocokkan tanggal lahir tersimpan. Jalur reuse anggota publik juga tidak memperbarui atau mencocokkan tanggal lahir.

Reproduksi: identitas tersimpan lahir 1990; agen mengirim identitas tersebut sebagai infant dengan tanggal lahir enam bulan lalu. Booking diterima, menggunakan harga infant dan tidak menghitung kursi. Dampak: salah tarif dan manifest. Perbaikan: validasi usia dari identitas otoritatif untuk jamaah existing; perubahan tanggal lahir membutuhkan alur koreksi identitas. Uji semua jalur reuse, ulang tahun kedua, tanggal masa depan, serta konsistensi tanggal keberangkatan/pulang sesuai kebijakan operator.

### SB03 — DP berbeda antara booking, invoice, dan verifikasi pembayaran

**Bukti: kode dan integrasi rollback.** Booking memakai override termasuk nol, lalu default brand (`selfbooking/repository.go:511`). Invoice memakai fallback tetap Rp5 juta dan mengabaikan override nol (`:748`). Payment juga mengabaikan override nol (`payment/repository.go:341`). DP tidak disnapshot di header booking; invoice/payment membaca konfigurasi terbaru.

Probe membuktikan paket DP=0, brand DP=Rp5 juta, pembayaran terkonfirmasi Rp1 juta tetap berstatus `baru`. Contoh lain: brand DP=Rp7 juta dan paket tanpa override akan ditagih Rp5 juta di invoice. Perubahan DP setelah booking dapat menggeser kewajiban lama. Perbaikan: satu perhitungan DP dengan semantik null/nol yang sama, snapshot ketentuan saat transaksi, dan aturan eksplisit untuk DP nol. Uji invoice, pembayaran, dan expiry dengan matriks null/0/positif/perubahan konfigurasi.

### SB04 — Kursi dapat dilepas ketika bukti transfer masih menunggu admin

**Bukti: kode.** `internal/crmdeal/repository.go:294,335` melepas booking `baru` yang habis 24 jam tanpa memeriksa pembayaran pending. Upload portal membuat payment pending (`payment/repository.go:212`); hold tetap berjalan. Konfirmasi setelah hold habis mencoba memperoleh kursi kembali dan dapat gagal jika habis (`payment/repository.go:391`).

Reproduksi yang perlu diuji terisolasi: upload sebelum deadline, admin verifikasi setelah deadline, kursi telah diambil booking lain. Dampak: jamaah sudah transfer tetapi kursinya hilang. Tetapkan kebijakan bukti pending, grace period terbatas, SLA admin, dan penanganan pembayaran terlambat. Jangan mengunci kursi tanpa batas hanya karena pengguna mengunggah sembarang bukti.

### SB05 — Invoice publik menggunakan kode pendek, tidak terikat brand, dan dapat diindeks

**Bukti: kode.** Kode booking memakai suffix acak empat karakter (`selfbooking/repository.go:410`). `GetInvoiceByCode` hanya memfilter kode (`:590`), mengembalikan nama seluruh pax, jadwal perjalanan, dan nilai transaksi. Halaman `src/app/invoice/[code]/page.jsx` tidak mencocokkan brand hostname. Metadata mengandung nama PIC; layout menetapkan `robots.index=true`, dan `robots.js` tidak mengecualikan invoice.

Dampak: kode berfungsi sebagai satu-satunya kredensial dokumen pribadi; invoice brand lain dapat ditampilkan di host yang salah; tautan yang ditemukan crawler boleh diindeks. Rate limit miss mengurangi percobaan tetapi bukan pengganti token akses berentropi tinggi. Perbaikan: pisahkan kode bisnis dari token invoice rahasia, scope brand, batasi data publik, gunakan noindex/private-no-store sesuai alur berbagi. Tidak dilakukan enumerasi invoice nyata.

### SB06 — Konfirmasi beberapa pembayaran pending dapat melewati total tagihan

**Bukti: kode.** `payment/repository.go:180` memeriksa sisa hanya terhadap payment confirmed saat membuat payment. Dua bukti pending masing-masing sebesar total masih dapat dibuat. `UpdateStatus(:272)` kemudian mengubah keduanya menjadi confirmed tanpa memeriksa kembali plafon pembayaran di dalam transaksi.

Dampak: total confirmed melebihi nilai booking tanpa workflow overpayment/refund yang jelas. Perbaikan: saat konfirmasi, lock booking dan hitung pembayaran lain yang sudah confirmed; tolak atau arahkan ke workflow kelebihan pembayaran yang eksplisit. Uji konfirmasi berurutan dan paralel. Skenario belum dijalankan terhadap database transaksi pengguna.

### SB07 — Status DP tetap ada setelah seluruh pembayaran terkonfirmasi hilang

**Bukti: integrasi rollback.** `payment/repository.go:375-379` tidak menurunkan booking `dp` dengan `is_seat_blocked=true` walaupun akumulasi confirmed menjadi nol/kurang DP. Probe memulai keadaan sesudah koreksi payment: DP, kursi diblokir, confirmed=0; sinkronisasi tetap menghasilkan DP.

Dampak: status pembayaran dan jaminan kursi tidak lagi mencerminkan dana yang valid. Perbaikan: rumus state pembayaran terpusat dan transisi balik yang eksplisit; tentukan apakah hold baru/grace period perlu diberikan. Audit juga dampak pembatalan payment lunas terhadap komisi tanpa otomatis menganggap komisi yang sudah dibayar dapat dibatalkan.

### SB08 — Captcha backend lolos ketika secret kosong, tanpa batas environment

**Bukti: kode/config-dependent.** `internal/selfbooking/turnstile.go:50` mengembalikan true jika secret kosong. Handler hanya menuntut token nonkosong. Guard production di widget frontend tidak melindungi panggilan API langsung.

Dampak: salah konfigurasi deployment menonaktifkan captcha di endpoint yang menahan kursi dan membuat identitas. Perbaikan: fail startup atau fail closed pada environment non-development; bypass lokal harus eksplisit. Verifikasi juga hostname/action yang diharapkan dari hasil captcha. Audit tidak menyatakan secret production saat ini kosong.

### SB09 — Header CF-Connecting-IP yang tidak dibersihkan dapat menghindari rate limit

**Bukti: kode dan konfigurasi Nginx lokal.** `src/lib/forwardClientIp.js` meneruskan header IP masuk. `internal/shared/clientip.go:82` mendahulukan CF-Connecting-IP dari semua peer proxy tepercaya, termasuk loopback. Nginx lokal menimpa X-Real-IP dan menambah X-Forwarded-For, tetapi tidak membuang/menimpa CF-Connecting-IP.

Pada topologi lokal ini, request berheader CF-Connecting-IP buatan dapat diteruskan melalui Next ke backend sebagai IP pembatas. Topologi production perlu diverifikasi tersendiri. Perbaikan: hanya percaya header Cloudflare dari hop Cloudflare tervalidasi; sanitasi di edge dan teruskan satu IP kanonis. Tambahkan tes lintas proxy, IPv6, beberapa hop, dan request langsung. Tidak dilakukan spam untuk membuktikan bypass.

## Temuan P2

### SB10 — Invoice expired/batal tetap dapat mengajak membayar

**Bukti: kode.** Expiry menghapus deadline dan melepas kursi tanpa mengubah status `baru`; query invoice mengganti null deadline dengan created_at (`selfbooking/repository.go:587`). Status expiry hanya ditetapkan untuk `draft` (`:772`). `DigitalInvoiceView.jsx:22,37` menghitung `isExpired` tetapi tidak menggunakannya untuk merender pesan; badge hanya membedakan DP/lunas dan sisanya “Menunggu Pembayaran DP” (`:224-237`). Rekening/CTA tetap tampil. Portal pembayaran juga menganggap setiap pembayaran positif sebagai “DP Terkonfirmasi” (`portal/pembayaran/page.jsx:137`) meskipun ambang DP belum tercapai.

Perbaikan: kirim status pembayaran dan status reservasi yang terpisah, render seluruh state termasuk batal/expired/pending/DP parsial, dan arahkan pemulihan kursi melalui admin. Uji sebelum/sesudah expiry, refresh, dan perubahan status admin saat halaman terbuka.

### SB11 — Invoice masih memakai izin dan akreditasi rekaan

**Bukti: kode.** `DigitalInvoiceView.jsx:201`: fallback PPIU `401/2020` dan akreditasi `A`. Perbaikan paket sebelumnya belum mencakup komponen invoice ini. Dampak: dokumen yang disebut resmi memuat klaim tidak bersumber dari data. Hilangkan fallback, tampilkan hanya nilai tervalidasi atau keterangan belum tersedia. Uji brand tanpa izin/akreditasi.

### SB12 — Persetujuan syarat tidak dapat dibaca atau diaudit

**Bukti: kode.** `BookingWizard.jsx:2197` menautkan Syarat & Ketentuan ke `#`. Request/backend tidak membawa atau menyimpan versi syarat maupun bukti persetujuan. Checkbox hanya guard frontend.

Dampak bisnis: pelanggan menyetujui pembatalan/pelunasan yang tidak bisa diperiksa; tim operasional tidak memiliki rekaman ketentuan transaksi. Sediakan teks kebijakan nyata, versioning, dan waktu persetujuan dari server. Ini temuan desain proses, bukan kesimpulan kepatuhan hukum.

### SB13 — Tidak ada recovery idempoten ketika booking berhasil tetapi respons hilang

**Bukti: kode.** BookingRequest tidak punya idempotency key; handler mengembalikan error duplicate tanpa kode booking lama. `BookingWizard.jsx:749` menulis localStorage sebelum menyimpan hasil/redirect; error storage ikut masuk catch seolah pemesanan gagal. Retry dapat ditolak sebagai duplicate setelah transaksi pertama sebenarnya committed.

Perbaikan: kunci idempotensi per pengajuan, replay respons aman bagi pemilik, dan pemulihan lewat portal/lookup terautentikasi. Kegagalan penyimpanan browser tidak boleh menyembunyikan keberhasilan server. Uji putus koneksi setelah commit, retry token captcha baru, dan browser yang menolak storage.

### SB14 — Validasi wizard sulit diakses dan terlambat

**Bukti: kode UI.** Mayoritas label input `BookingWizard.jsx:1380` dst tidak memiliki htmlFor/id. CustomSelect (`:29`) tidak menyatakan expanded/listbox atau navigasi keyboard yang sesuai. Alert, ganti kamar, dan logout (`:847-939`) tidak memiliki semantik dialog, focus trap, Escape, atau fokus kembali. Validasi PIN baru dilakukan saat submit step 3, lalu mengirim pengguna kembali ke step 2 (`:621`).

Perbaikan: validasi per langkah, error dekat field + ringkasan terfokus, native select atau combobox aksesibel, modal bersama yang konsisten. Uji keyboard/screen reader dan 375px terlebih dahulu. Belum ada bukti visual runtime untuk tingkat kontras atau overflow.

### SB15 — Backend menerima PIN enam huruf yang tidak dapat dimasukkan pada UI login

**Bukti: integrasi rollback dan kode.** Handler/repository booking hanya memeriksa panjang enam (`handler.go:210`, `repository.go:192`), sedangkan UI login membuang nondigit (`portal/login/page.jsx:28`). Probe PIN `abcdef` berhasil membuat booking.

Perbaikan: satu validator PIN numerik di booking, aktivasi, dan login. Tambahkan validasi enum gender, format/panjang nomor dan email, batas panjang nama, dan body size agar input tidak berakhir sebagai error database generik. Uji API langsung, bukan hanya guard frontend.

### SB16 — Pemeriksaan nomor rawan respons basi dan penguncian UX

**Bukti: kode.** `BookingWizard.jsx:441-474` tidak membatalkan request lama atau memeriksa bahwa respons cocok dengan nomor terkini. Setiap blur mengulang check (`:1409`), walaupun sama. Endpoint membatasi sepuluh check/15 menit. Check memerlukan minimal sepuluh digit, validasi langkah berikutnya menyebut sembilan digit.

Reproduksi: nomor A diproses lambat, ubah ke B, respons A tiba belakangan; UI memakai cabang PIN A. Dampak: salah cabang, retry, dan batas check habis oleh penggunaan normal. Backend masih memeriksa identitas sehingga ini bukan bypass autentikasi. Gunakan request sequence/AbortController, cache hasil nomor yang sama, aturan nomor tunggal, serta pesan cooldown dari server.

### SB17 — Jalur tanpa PIN/booking agen belum tersambung ke pembayaran mandiri

**Bukti: kode.** PIC agen baru dibuat tanpa PIN dan tanpa tanggal lahir (`selfbooking/agen_booking.go` + `repository.go:164`). Booking publik cabang tanpa PIN tidak menerbitkan token bagi pengguna anonim (`repository.go:238`). Invoice mengarahkan semuanya ke login. Aktivasi membutuhkan link admin dan tanggal lahir tersimpan (`portal/activation.go:73`).

Dampak: “booking sukses” belum berarti jamaah dapat mengunggah bukti transfer mandiri; admin perlu melengkapi identitas dan menerbitkan aktivasi. Perbaikan: tampilkan onboarding yang sesuai cabang, tugas follow-up admin/agen, pengumpulan data minimum yang dibutuhkan, dan kanal konfirmasi alternatif. Jangan mengurangi verifikasi identitas demi mempermudah login.

### SB18 — Deadline pelunasan H-45 bertentangan dengan penerimaan booking H-14

**Bukti: kode/kebijakan.** Booking menerima keberangkatan minimal 14 hari (`selfbooking/repository.go:145`), tetapi invoice selalu menyatakan jatuh tempo H-45 (`:801`). Booking pada H-30 langsung memiliki deadline masa lalu, tetap ditawari DP dan hold 24 jam.

Perlu keputusan bisnis: keberangkatan dekat harus lunas langsung, masa tenggang khusus, atau melalui admin. Tampilkan sebelum persetujuan dan tegakkan di server. Uji H-45, H-44, H-14, dan H-13, menggunakan zona waktu bisnis yang sama.

### SB19 — Harga dapat berubah setelah pelanggan menyetujui ringkasan

**Bukti: kode.** Wizard memakai harga ketika halaman dimuat. BookingRequest tidak menyertakan versi quotation atau harga yang telah disetujui. Repository mengambil harga terbaru saat mengunci schedule dan langsung membuat booking.

Dampak: pelanggan melihat nominal lama pada konfirmasi, lalu invoice baru tanpa persetujuan ulang. Perbaikan: quote/version token server, deteksi perubahan harga/DP, dan kembalikan ringkasan baru untuk disetujui sebelum commit. Server tetap menjadi sumber harga; jangan mempercayai nominal client sebagai harga final.

### SB20 — Invoice salah menampilkan pax/PIC setelah perubahan administratif

**Bukti: kode.** Query pax invoice tidak memfilter atau mengirim pax_status (`selfbooking/repository.go:693`); nama PIC diambil dari baris pertama (`:713`), bukan `bookings.pic_jamaah_id`. Pembatalan pax dapat memindahkan PIC di `booking/repository.go:1073`. DP invoice menghitung semua pax reguler termasuk yang batal. Invoice juga tidak merinci diskon/add-on ketika total sudah berubah.

Dampak: PIC, jumlah DP, daftar jamaah, dan rincian tidak lagi cocok dengan tagihan. Perbaikan: model invoice mencakup PIC sebenarnya, status pax, item biaya/diskon, dan jejak perubahan. Uji pembatalan PIC, anggota pengganti, diskon, add-on, dan pembatalan parsial.

### SB21 — Draft tidak pulih; hasil lama dapat muncul sebagai sukses baru

**Bukti: kode.** Data step 1–3 hanya useState. SessionStorage menyimpan hasil sukses per schedule (`BookingWizard.jsx:235,762`) tanpa expiry atau identitas pengguna; restore langsung membuka step 4 tanpa mengambil status server. Logout tidak menghapus hasil tersebut. State counts/PIC setelah refresh bukan snapshot transaksi, sehingga rincian fallback dapat berbeda.

Dampak: isi rombongan hilang saat refresh; pada perangkat bersama hasil pemesan lama dapat muncul; booking expired tetap terlihat sukses. Perbaikan: recovery draft minim data sensitif dan berumur terbatas, hasil sukses dimuat dari server dengan akses yang tepat, clear saat logout/pergantian pemesan, serta jangan simpan PIN. Uji refresh tiap langkah dan dua pengguna pada perangkat sama.

## Kontrol yang sudah tepat

- Transaksi serializable, schedule `FOR UPDATE`, pemeriksaan brand/status/cutoff, dan pengurangan kursi reguler berjalan dalam satu transaksi.
- Backend menentukan harga; infant tidak mengurangi kursi, harga infant null ditolak setelah perbaikan paket.
- PIC existing dengan PIN harus lolos bcrypt atau token milik PIC; booking agen tidak menerbitkan token jamaah dan memeriksa kepemilikan jamaah.
- Proxy booking terbaru mengikat brand pada hostname dan tidak memakai brand body sembarangan.
- Captcha yang sudah dikonfigurasi diverifikasi server; respons verifikasi unavailable ditolak. UI mereset token sekali pakai setelah kegagalan.
- Portal pembayaran memeriksa pemilik booking dan rekening aktif dari brand yang sesuai. Payment menyimpan snapshot rekening tujuan.
- Expiry mengunci booking dan memeriksa kondisi ulang; pemulihan kursi saat konfirmasi payment memeriksa sisa, sehingga tidak langsung oversell.
- Komisi disnapshot saat booking dan jalur agen/referral sudah memiliki pemeriksaan tersendiri. Audit ini tidak menguji seluruh proses pencairan komisi.

## Bukti pengujian

Probe diagnostik tersimpan pada `internal/selfbooking/audit_selfbooking_test.go` dan `internal/payment/audit_selfbooking_test.go`. Probe sengaja mengonfirmasi perilaku salah saat ini; PASS di bawah **bukan** berarti fitur bebas bug. Setelah perbaikan, ubah menjadi regression test yang mengharapkan perilaku benar.

```text
ERP_TEST_DB=1 go test -v ./internal/selfbooking -run TestAuditSelfBooking -count=1
CONFIRMED: same member accepted in two active bookings on one schedule
CONFIRMED: six-letter PIN accepted by booking repository
CONFIRMED: stored adult identity accepted as infant using submitted DOB
PASS — exit 0

ERP_TEST_DB=1 go test -v ./internal/payment -run TestAuditSelfBooking -count=1
CONFIRMED: schedule DP=0 falls back to brand DP=5M; paid 1M stays baru
CONFIRMED: zero confirmed payment leaves DP status and permanent seat intact
PASS — exit 0

GET http://localhost:9090/api/health
HTTP 200 {"status":"ok","database":"connected"}
GET http://localhost:9090/api/public/invoice/AUDIT-NONEXISTENT
HTTP 404 {"error":"invoice pendaftaran tidak ditemukan"}
```

Tidak menjalankan ulang build aplikasi karena sesi audit tidak mengubah kode aplikasi. Build dan seluruh suite Go pada akhir perbaikan paket sebelumnya sudah lulus; hasil tersebut bukan pengganti acceptance test terhadap temuan baru ini.

## Prioritas implementasi dan keputusan bisnis

1. Tutup SB01/SB02 untuk integritas manifest/tarif; SB05/SB08/SB09 untuk kontrol akses dan penyalahgunaan.
2. Satukan kontrak quotation/DP/payment/hold (SB03/SB04/SB06/SB07/SB10/SB18/SB19). Definisikan DP nol, late booking, transfer pending, expiry, koreksi payment, dan overpayment sebelum finalisasi transisi status.
3. Perbaiki invoice dan onboarding berdasarkan cabang pengguna (SB11/SB12/SB17/SB20).
4. Tambahkan recovery idempoten, validasi bersama, form aksesibel, dan uji browser (SB13–SB16/SB21).

Acceptance matrix minimum: publik baru / existing PIN / existing tanpa PIN / login / agen; 1–9 pax termasuk infant; null/0/positif DP; kuota terakhir paralel; duplicate seluruh pax; perubahan harga di tengah form; putus koneksi sesudah commit; transfer pending sebelum expiry; verifikasi sesudah expiry ketika kursi habis; pembayaran ditolak/revisi; pembatalan parsial; invoice lintas brand; 375px lalu 1440px dan keyboard penuh.

Metrik bisnis yang perlu dicatat setelah perbaikan: funnel per langkah dan cabang autentikasi, kegagalan check nomor/captcha, booking committed tetapi respons gagal, hold expired dengan payment pending, waktu verifikasi admin, selisih quotation vs invoice, duplikasi manifest, serta aktivasi PIC agen sampai pembayaran. Jangan merekam PIN/token atau identitas sensitif ke analytics.

## Tindak lanjut perbaikan

Implementasi, pemetaan seluruh temuan, hasil regresi, dan batas verifikasi terbaru tersedia pada [laporan perbaikan](PERBAIKAN-SELF-BOOKING-2026-09-28.md). Probe audit di atas merupakan bukti sebelum perbaikan; test sekarang mengharapkan perilaku yang benar. Migrasi 065 dan acceptance test setelah penerapan masih menunggu izin database development.


Pembaruan 28 September 2026, 14:06 WIB: migrasi 065 sudah diterapkan setelah izin eksplisit pengguna. Verifikasi 62 snapshot dari 62 booking lulus; tidak ada snapshot hilang atau token duplikat. Detail backup dan pemeriksaan tercatat di laporan perbaikan.

