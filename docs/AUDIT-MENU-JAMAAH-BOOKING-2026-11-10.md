# Audit fitur menu "Jamaah & Booking" (dashboard office)

Tanggal: 10 November 2026. Branch: `fix/komisi-pribadi-nik-status` (commit `149f165`).

## Cakupan dan metode

Enam item grup sidebar "JAMAAH & BOOKING" di Master Dashboard: Kelola Jamaah, Akun Pelanggan, Persetujuan Data Jamaah, Pengumuman, Kelola Booking, Pembayaran. Yang diperiksa: halaman React (`frontend/shared`, `frontend/master-dashboard`), handler dan repository Go di belakangnya, serta migrasi terkait.

Metode: pembacaan kode. Tidak ada uji ke API nyata, browser, atau database. Kolom "Cek" menandai temuan yang baris kodenya dibaca ulang secara terpisah setelah audit awal ("Ya"), atau hanya dibaca sekali ("1x"). Temuan berlabel "Dugaan" punya bagian yang belum ditelusuri; keterangannya ada di baris temuan.

Pemeriksaan otomatis:

- `go vet ./...`: bersih.
- `go test ./...`: semua paket lulus. Tes yang butuh database (`ERP_TEST_DB=1`) dilewati, jadi logika berbasis SQL tidak tervalidasi di sesi ini.
- `vite build` Master Dashboard: lulus (satu chunk 1,29 MB, peringatan ukuran).
- `oxlint`: tidak ada error, hanya peringatan (impor dan variabel tidak terpakai, antara lain `handleUpdatePaymentStatus` di `BookingDetailPage.jsx:660`).

Audit sebelumnya (`docs/arsip/AUDIT-JAMAAH-BOOKING-2026-09-30.md`): perbaikan JB-01 sampai JB-13 dan UI-02, UI-04 masih terpasang. Sisa: JB-05 belum tuntas di `internal/dokumen` (R-03), JB-06 belum menutup progress per pax dan perlengkapan (T-06), UI-03 belum diterapkan di form booking (S-13). Catatan kebijakan worker hold dari JB-03 masih terbuka.

## Ringkasan

| ID | Tingkat | Area | Temuan | Cek |
|---|---|---|---|---|
| K-01 | Kritis | Booking | "Simpan tanggal lahir" di form booking mengosongkan hampir seluruh profil jamaah | Ya |
| T-01 | Tinggi | Jamaah | Form edit yang gagal memuat tetap bisa disimpan dan mengosongkan data jamaah | Ya |
| T-02 | Tinggi | Booking | Status lunas bisa dicapai lewat diskon; komisi tercatat permanen tanpa uang masuk | Ya |
| T-03 | Tinggi | Pembayaran | Pembayaran terkonfirmasi tidak bisa dikoreksi atau dibatalkan | Ya |
| T-04 | Tinggi | Persetujuan | Persetujuan menimpa nilai terkini tanpa membandingkan nilai lama | Ya |
| T-05 | Tinggi | Booking | Detail booking tidak bisa dibuka tanpa `payment.view` (peran `admin_cabang`, `cmo`) | Ya |
| T-06 | Tinggi | Booking | Booking batal bisa tetap punya pax aktif; perlengkapan masih bisa diserahkan, stok bocor pada draft | Ya (sebagian) |
| T-07 | Tinggi | Jamaah | Jamaah yang dibuat dari Kelola Jamaah selalu dikunci "tanpa agen" | Ya |
| S-01..S-17 | Sedang | Semua | Lihat bagian Sedang | campuran |
| R-01..R-12 | Rendah | Semua | Lihat bagian Rendah | 1x |

## Kritis

### K-01 · "Simpan tanggal lahir" inline mengosongkan profil jamaah

`frontend/shared/src/pages/BookingFormPage.jsx:854-861` mengirim `PUT /api/admin/jamaah/{id}` dengan lima field saja (`nama_lengkap`, `tanggal_lahir`, `nik`, `no_hp`, `status`). `internal/jamaah/repository.go:331-344` menulis semua kolom dari isi permintaan, sehingga field yang tidak dikirim menjadi NULL: jenis kelamin, nama ayah, tempat lahir, data paspor, email, pekerjaan, alamat, kota, catatan, dan kontak darurat.

Skenario: jamaah punya paspor dan alamat tetapi tanggal lahirnya kosong. Staf memilihnya di form booking, mengisi tanggal lahir di kotak kuning, lalu menekan Simpan. Data lain terhapus tanpa peringatan. Jenis kelamin yang hilang ikut memengaruhi penyaringan set perlengkapan.

Saran: backend hanya memperbarui kolom yang dikirim (semantik PATCH), atau frontend mengambil data lengkap lebih dulu lalu mengirim ulang semuanya. Perbaikan di backend juga menutup T-01 dan S-01.

## Tinggi

### T-01 · Form edit jamaah yang gagal memuat tetap bisa disimpan

`JamaahFormPage.jsx:82-86`: bila `getJamaah` gagal selain 404, hanya pesan error yang dipasang dan form tetap dirender dengan nilai awal kosong. `sanitizePayload` (`:115-123`) mengubah string kosong menjadi `null`, lalu `updateJamaah` menimpa semua kolom.

Skenario: jaringan putus atau server menjawab 500 saat membuka `/jamaah/12/edit`. Staf mengetik nama lalu menyimpan; NIK, paspor, HP, alamat, dan kontak darurat jamaah 12 menjadi NULL.

Saran: jangan render form bila data gagal dimuat; tampilkan tombol coba lagi.

### T-02 · Lunas lewat diskon mencatat komisi permanen

`ensureTotalCoversPaidTx` hanya menolak total di bawah jumlah dibayar. Diskon yang membuat total sama dengan jumlah dibayar lolos, lalu `SyncBookingStatusTx` (`internal/payment/repository.go:439-440`, `:482-485`) menetapkan `lunas` dan memanggil `komisi.OnBookingLunas`. Komisi tidak pernah dibalik (keputusan K20 di `docs/spek/SPEK-KEAGENAN-AKUN-2026-11-03.md:61`).

Skenario: booking Rp30 juta, DP Rp5 juta. Admin menambah diskon Rp25 juta karena salah ketik. Status menjadi lunas dan komisi tercatat. Diskon dihapus, status kembali dp, komisi tetap ada dan bisa dicairkan setelah tanggal berangkat.

K20 mengandaikan lunas berarti uang penuh sudah masuk. Jalur diskon mematahkan andaian itu. `TestRecalculateTotalTransitions` saat ini mengunci perilaku ini sebagai yang diharapkan.

Varian (S-08): batal pax pada booking `baru` yang sudah menerima pembayaran bisa menurunkan total di bawah jumlah dibayar dan memicu lunas.

Perlu keputusan: batas diskon (misalnya persentase maksimum, atau persetujuan kedua), atau komisi hanya dicatat bila lunas dicapai lewat pembayaran.

### T-03 · Pembayaran terkonfirmasi tidak bisa dikoreksi

Pembayaran yang dicatat admin langsung berstatus `confirmed`. `internal/payment/handler.go:252-256` menolak perubahan status selain dari `pending`; penghapusan hanya untuk `pending` (`repository.go:497`, dibaca sekali); refund hanya untuk booking batal.

Skenario: admin mengetik Rp25.000.000 padahal uang masuk Rp2.500.000, atau mencatat di booking yang salah. Booking naik ke dp atau lunas, kursi terkunci, komisi bisa tercatat. Satu-satunya jalan keluar adalah membatalkan booking atau mengubah database. Tombol "Pembayaran Diterima" di layar Pembayaran juga tanpa dialog konfirmasi.

Saran: aksi pembatalan pembayaran (void) dengan alasan wajib, izin tersendiri, dan jejak pelaku; status booking dihitung ulang setelahnya.

### T-04 · Persetujuan data jamaah menimpa nilai terkini

`internal/customer/perubahan_jamaah_admin.go:219-226` menerapkan `it.Baru` tanpa membandingkan `it.Lama` dengan isi jamaah saat ini dan tanpa validasi ulang. Modal menampilkan nilai "Sebelumnya" dari saat pengajuan dibuat.

Skenario: pelanggan mengajukan nomor HP A menjadi B. Besoknya staf mengubah nomor itu menjadi C karena jamaah menelepon. Staf lain menyetujui pengajuan; nomor kembali ke B dan riwayat mencatat A ke B.

Terkait: persetujuan diterapkan ke semua baris yang dianggap orang yang sama berdasarkan nama dan tanggal lahir, tanpa NIK (`perubahan_jamaah.go:131-136`), dan kegagalan pencarian grup ditelan (`perubahan_jamaah_admin.go:212`). Dampak penuhnya belum ditelusuri (Dugaan).

Saran: saat menyetujui, bandingkan nilai lama dengan nilai terkini; bila berbeda, tampilkan nilai terkini dan minta keputusan ulang.

### T-05 · Detail booking gagal total tanpa `payment.view`

`BookingDetailPage.jsx:181-184` memuat booking dan pembayaran lewat `Promise.all`. Satu jawaban 403 menjatuhkan keduanya, dan halaman menampilkan "Booking tidak ditemukan". Peran bawaan `admin_cabang` punya `booking.view` dan `booking.edit` tanpa `payment.view` (`migrations/071_rbac.sql:170-172`); `cmo` punya `booking.view` saja (`:131-132`). Isi tabel peran di database berjalan tidak diperiksa.

Skenario: admin cabang membuat booking, diarahkan ke halaman detail, lalu melihat "Booking tidak ditemukan" padahal booking tersimpan dan kursi sudah terpotong. Risiko: booking dibuat ulang.

Pola yang sama ada di `JamaahDetailPage.jsx:146-150` (jamaah, dokumen, booking dalam satu `Promise.all`).

### T-06 · Booking batal dengan pax aktif; perlengkapan tanpa cek status booking

`internal/booking/repository.go:1480-1495`: pax hanya dijadikan batal di dalam cabang `newStatus == "batal" && isSeatBlocked`. Booking yang dibatalkan setelah kursinya dilepas (manual atau oleh worker hold) tetap punya pax `aktif`.

`MarkPerlengkapanDiberikan` (`:2059-2107`) dan `UpdatePaxProgress` hanya memeriksa status pax, bukan status booking. Akibatnya perlengkapan masih bisa diserahkan untuk booking batal dan stok brand berkurang.

Belum dibaca ulang (1x): distribusi pada draft lalu simpan ulang draft menghapus log lewat `ON DELETE CASCADE` sehingga stok tidak pernah kembali; batal seluruh booking tidak mengembalikan stok perlengkapan, berbeda dengan batal per pax pada status `baru`.

Saran: batalkan pax pada setiap pembatalan booking; tambahkan gate status booking pada distribusi perlengkapan dan progress per pax.

### T-07 · Jamaah dari Kelola Jamaah selalu "tanpa agen"

`JamaahFormPage.jsx:152` selalu mengirim `kaitan_agen: { mode: "tanpa_agen" }`. Kunci hanya terisi otomatis bila status masih `belum_ditentukan` (`internal/agen/kaitan.go:66-68`, dibaca sekali). Panel pemilih kaitan dihapus dari form ini di commit `2fa01c8`.

Skenario: staf mendaftarkan jamaah yang dibawa agen X lewat Kelola Jamaah. Booking berikutnya oleh X tidak mengikat kunci, sehingga X tidak menerima komisi kecuali ada override admin.

Perlu keputusan: apakah ini disengaja. Bila tidak, kembalikan pemilih kaitan atau kirim `belum_ditentukan`.

Terkait (S-12): default "Sama seperti PIC" pada jamaah baru di form booking tidak pernah aktif karena membaca field yang salah.

## Sedang

| ID | Area | Temuan | Lokasi | Cek |
|---|---|---|---|---|
| S-01 | Jamaah | Edit admin menulis seluruh baris tanpa versi; perubahan lain yang masuk di antaranya hilang | `internal/jamaah/repository.go:320-354` | Ya |
| S-02 | Jamaah | Persentase dokumen dibagi 6 padahal ada 7 jenis; 7 dokumen menjadi 117% dan terhitung "belum lengkap" | `internal/jamaah/repository.go:138` | Ya |
| S-03 | Booking | `siap_berangkat` selalu false di daftar karena pax tidak dimuat; dashboard brand menghitung semua booking dp/lunas sebagai "progress belum lengkap" | `internal/booking/repository.go:228`, `BrandDashboardPage.jsx:190` | Ya |
| S-04 | Semua | Tombol aksi tidak digating izin di halaman bersama (tidak ada `hasPermission` di `frontend/shared/src`); pengguna `*.view` melihat tombol yang berakhir 403 | `BookingDetailPage.jsx`, `JamaahPage.jsx`, `PaymentConfirmationsPage.jsx:413-440` | 1x |
| S-05 | Jamaah | Hapus jamaah ditolak dengan pesan "masih dipakai data booking" untuk semua pelanggaran FK, termasuk log perubahan dan dokumen (disimpulkan dari skema, tidak diuji) | `internal/jamaah/repository.go:386-392` | 1x |
| S-06 | Jamaah | Log perubahan tidak lengkap (catatan, status, pindah brand, buat, hapus, keputusan dokumen tanpa pelaku) dan tidak ada layar riwayat per jamaah; nilai dipotong 255 karakter | `internal/jamaah/log_perubahan.go` | 1x |
| S-07 | Jamaah | Panjang isian tidak divalidasi sesuai kolom; pengajuan pelanggan bisa tersimpan lalu tidak bisa disetujui (500 berulang) | `internal/customer/data_jamaah.go:251-252`, `migrations/013` | 1x |
| S-08 | Booking | Batal pax pada booking `baru` berbayar bisa memicu lunas dengan kelebihan bayar tidak ditandai | `internal/booking/repository.go:1221-1222`, `:1306-1309` | 1x |
| S-09 | Booking | Finalisasi draft memakai harga saat draft disimpan; infant tanpa harga lolos dengan Rp0 | `internal/booking/repository.go:789-809`, `:961-1133` | 1x |
| S-10 | Booking | Booking batal dengan potongan pembatalan selamanya berlabel "perlu refund"; pax batal pada dp/lunas tidak punya jalur refund | `internal/booking/repository.go:2016` | 1x |
| S-11 | Booking | Booking dengan total Rp0 (diskon 100%) tidak pernah bisa lunas | `internal/payment/repository.go:438-443` | Ya |
| S-12 | Booking | Default kaitan "Sama seperti PIC" membaca `direkrut_oleh_jamaah_id`; API mengirim `direkrut_oleh_akun_id` | `BookingFormPage.jsx:767` | 1x |
| S-13 | Booking | Form booking: jamaah infant yang baru dibuat atau diisi tanggal lahirnya membuat baris macet; pesan error muncul di luar layar (sebab yang sama dengan UI-03) | `BookingFormPage.jsx:727-745`, `:805-826`, `:890` | 1x |
| S-14 | Booking | "Harga Paket Dasar" hanya harga satu pax; invoice tidak menjumlah benar bila ada pax batal pada dp/lunas | `BookingDetailPage.jsx:363`, `:407` | 1x |
| S-15 | Booking | Tanggal pembayaran bawaan memakai tanggal UTC; antara 00.00 dan 06.59 WIB tercatat kemarin | `BookingDetailPage.jsx:122`, `:646` | 1x |
| S-16 | Pembayaran | Bukti PDF dirender sebagai gambar rusak; gagal muat bukti ditulis "dicatat tanpa unggahan bukti" | `PaymentConfirmationsPage.jsx:562-589` | 1x |
| S-17 | Pengumuman | Pengumuman bertarget jadwal hanya sampai ke akun pemesan mandiri; booking dari admin dan CRM tidak punya `customer_account_id` | `internal/pengumuman/pengumuman.go:219-220` | Ya (kueri) |

Catatan lintas area:

- Daftar jamaah, booking, dan pembayaran dimuat penuh lalu disaring dan dipaginasi di browser, dengan batas server 10.000 baris tanpa peringatan. Daftar jamaah dan booking menjalankan 4 sampai 5 subkueri per baris. Daftar akun sudah paging di server.
- Batas hold kursi tidak terlihat dan tidak bisa diatur dari UI walau API mendukung `expires_at`.
- Dialog "Batalkan Booking" tidak memberi peringatan untuk booking berbayar, tombolnya tidak dikunci selama proses, dan berlaku juga untuk draft. Draft yang dibatalkan tidak bisa dihapus lagi.

## Rendah

| ID | Area | Temuan |
|---|---|---|
| R-01 | Persetujuan | Setujui tanpa dialog konfirmasi; badge sidebar berhenti di 200 dan tidak diperbarui setelah keputusan |
| R-02 | Jamaah | Blok "Jamaah ini juga agen" membaca `status_agen` yang tidak ada di API; hapus jamaah bisa terkirim dua kali |
| R-03 | Jamaah | `internal/dokumen/handler.go:192` masih mengirim teks error mentah ke klien (JB-05 belum tuntas) |
| R-04 | Booking | `CancelPax` mengabaikan error di cabang "tidak ada pax reguler tersisa" (`repository.go:1256-1288`), bisa panic |
| R-05 | Booking | Pesan error memuat awalan fungsi internal; beberapa 500 tanpa log |
| R-06 | Booking | Diskon: nominal pecahan dan nama di atas 255 karakter tidak divalidasi; kirim ganda membuat dua diskon |
| R-07 | Booking | Endpoint cashback admin tetap terdaftar dan tidak membaca flag; praktis mati karena kredit selalu 0 |
| R-08 | Booking | Semua mutasi melakukan commit lalu membaca ulang; kegagalan baca ulang menjawab 500 untuk perubahan yang sudah tersimpan |
| R-09 | Booking | Kode mati di detail booking (modal preview bukti, `booking.schedule.*`, `deletePayment`, `verifyPayment`); halaman menjadi spinner penuh setiap selesai mutasi |
| R-10 | Pembayaran | `departure_date` tidak pernah di-SELECT; verifikator dan waktu verifikasi tidak ditampilkan; alasan tolak tanpa trim dan batas panjang |
| R-11 | Pengumuman | Tidak ada tarik tayang; ubah dan hapus tanpa jejak; daftar holding tanpa nama brand; tanggal lampau diterima |
| R-12 | Akun | "Ganti upline" dan tautan agen/jamaah tampil tanpa cek izin layar tujuan; aturan "Ubah NIK" di UI lebih sempit dari backend |

Cakupan tes yang belum ada: `CancelPax`, pembatalan booking, blokir dan lepas kursi, finalisasi draft, perlengkapan, `payment.UpdateStatus`, handler dan isolasi brand per endpoint, persetujuan setelah jamaah diedit, serta tes frontend untuk semua layar.

## Yang sudah berfungsi benar

- Isolasi brand di semua handler yang dibaca: brand diambil dari klaim token, sub-resource (pax, diskon, refund, dokumen, pembayaran, pengumuman, akun) diverifikasi terhadap induknya.
- Akuntansi kursi: semua jalur berjalan dalam transaksi dengan baris jadwal terkunci; tidak ditemukan jalur pelepasan ganda.
- Mesin status booking: admin hanya bisa mengirim `batal`; dp dan lunas hanya dari sinkronisasi pembayaran; batal bersifat final.
- Pembayaran: konfirmasi ganda tertahan, kelebihan bayar ditolak, konfirmasi ke booking batal ditolak.
- Persetujuan data jamaah: setujui atau tolak ganda dijawab 409; NIK, nama, dan email tidak bisa diubah lewat jalur ini.
- Data pribadi akun: daftar tidak memuat hash kata sandi atau token; NIK agen disamarkan.
- Pengumuman dirender sebagai teks di sisi pelanggan; batas panjang ditegakkan di server.
- Izin rute frontend selaras dengan backend untuk keenam item; selisih hanya pada tombol di dalam layar (S-04).

## Batas verifikasi

- Tidak ada uji jalan. Temuan yang bergantung pada perilaku MySQL (FK, mode strict, pembulatan DECIMAL) disimpulkan dari migrasi.
- Tes integrasi ber-database tidak dijalankan.
- Tidak diperiksa: `internal/selfbooking`, portal pelanggan selain yang disebut, alur pencairan komisi, `perlengkapan.ApplyBrandDelta`, komponen bersama (`DataTable`, `Modal`), isi tabel peran di database berjalan.

## Status perbaikan (10 November 2026)

| ID | Status | Perubahan |
|---|---|---|
| K-01 | Diperbaiki, belum diuji jalan | Endpoint baru `PUT /api/admin/jamaah/{id}/tanggal-lahir` (izin `jamaah.edit`) yang hanya mengubah kolom `tanggal_lahir` dan mencatatnya di riwayat perubahan. Form booking memakai endpoint ini; `updateJamaah` tidak lagi dipanggil dari form booking. Tes baru: `TestValidasiTanggalLahir` |
| T-01 | Diperbaiki, belum diuji jalan | `JamaahFormPage.jsx`: bila data gagal dimuat, form tidak dirender; yang tampil pesan galat dan tombol "Coba Lagi" |

Verifikasi: `go vet` dan `go test` untuk `internal/jamaah` dan `cmd/api` lulus, build dashboard lulus. Belum diuji terhadap API dan database nyata maupun di browser. S-01 (edit penuh tanpa versi) belum dikerjakan.

## Urutan perbaikan yang disarankan

1. K-01 dan T-01 bersama S-01: ubah pembaruan jamaah agar tidak menimpa kolom yang tidak dikirim, dan cegah form kosong tersimpan.
2. T-05: pisahkan pemuatan pembayaran dari pemuatan booking di halaman detail.
3. T-06: batalkan pax pada setiap pembatalan booking; gate status pada perlengkapan dan progress.
4. T-04: bandingkan nilai lama saat menyetujui perubahan.
5. T-02, T-03, T-07: butuh keputusan kebijakan lebih dulu (batas diskon dan pemicu komisi, pembatalan pembayaran, kaitan agen di form jamaah).
6. S-02, S-03, S-12, S-15: perbaikan kecil satu sampai beberapa baris.
