# Audit fitur menu "Paket" (dashboard office)

Tanggal: 10 November 2026. Kode yang diaudit: `main` pada merge commit `9feeda5`.

## Cakupan dan metode

Enam item grup sidebar "PAKET" di Master Dashboard: Kelola Paket (daftar, form, detail), Kelola Kategori, Kelola Hotel, Maskapai & Bandara, Kelola Destinasi, Kelola Itinerary. Yang diperiksa: halaman React di `frontend/master-dashboard/src/pages`, handler dan repository Go di `internal/schedule`, `category`, `hotel`, `airline`, `airport`, `destination`, `itinerary`, rute dan izin di `cmd/api/main.go`, serta migrasi terkait.

Metode: pembacaan kode oleh tiga pembaca terpisah (backend paket, layar paket, data master), lalu temuan teratas dibaca ulang. Kolom "Cek":

- "Uji": dibaca ulang dan didukung kueri baca-saja ke database `erp_azhan_dev` atau riwayat git.
- "Ya": baris kodenya dibaca ulang secara terpisah.
- "1x": dibaca sekali oleh satu pembaca, belum dibaca ulang.

Tidak dilakukan: menjalankan skenario lewat API hidup atau browser, tes otomatis baru, dan pemeriksaan web publik di repo `azhan-booking`. Tidak ada data yang diubah.

## Ringkasan

| ID | Tingkat | Area | Temuan | Cek |
|---|---|---|---|---|
| T-01 | Tinggi | Paket | Nominal komisi agen per paket tidak bisa diisi dari form; isiannya hilang sejak 15 Oktober 2026 | Uji |
| T-02 | Tinggi | Kategori | Halaman Kelola Kategori gagal dimuat untuk peran `product_manager`, pemilik izin data master | Uji |
| S-01..S-13 | Sedang | Semua | Lihat bagian Sedang | campuran |
| R-01..R-20 | Rendah | Semua | Lihat bagian Rendah | campuran |

Tidak ada temuan Kritis. Isolasi brand pada paket, penguncian kursi, dan izin rute dinilai kokoh; rinciannya di bagian "Yang sudah berfungsi benar".

## Tinggi

### T-01. Nominal komisi agen per paket tidak bisa diisi dari form

`ScheduleFormPage.jsx` masih menyimpan `nominal_komisi_langsung` dan `nominal_bonus_pembinaan` di state (baris 136-137), memuatnya saat edit (314-315), dan mengirimnya (774-775), tetapi tidak ada isian yang menampilkannya. Tidak ada halaman dashboard lain yang mengubah kolom ini.

Bukti:

- Riwayat git: isian komisi ditambahkan di `6399825` (27 September 2026) dan hilang di `cb9dc58` (15 Oktober 2026, refactor form paket). Pesan commit itu tidak menyebut penghapusan komisi.
- Database dev: 31 dari 53 paket `published` punya `nominal_komisi_langsung` NULL. Semua 40 draft NULL.
- `komisi.SnapshotNominal` (`internal/komisi/hook.go:29-35`) menyalin kolom itu ke booking saat booking dibuat.

Akibat: paket yang dibuat lewat dashboard selalu tanpa nominal komisi, dan booking di paket itu menyalin NULL. Staf tidak punya cara mengisi atau mengoreksinya selain lewat SQL. Belum ditelusuri: berapa komisi yang tercatat saat snapshot NULL (nol, atau tidak ada baris).

Terkait: `minimal_dp` juga tanpa isian. Bila harga Quad diturunkan di bawah DP tersimpan, form menolak dengan "DP harus antara 0 dan harga kamar termurah" (baris 703) dan staf tidak punya isian untuk memperbaikinya (1x).

### T-02. Halaman Kelola Kategori gagal dimuat untuk `product_manager`

`CategoriesPage.jsx:60-63` memuat kategori dan brand dalam satu `Promise.all`. `GET /api/admin/brands` butuh izin `brand.view`. Di database dev, peran `product_manager` punya `masterdata.edit` tetapi tidak punya `brand.view`. Menu tampil (izinnya `masterdata.edit`), halaman hanya menampilkan error.

Hanya dua peran yang punya `masterdata.edit`: `product_manager` dan `super_admin_grup`. Jadi satu-satunya peran non-super-admin yang boleh mengelola kategori tidak bisa membuka layarnya.

Pola sama di Kelola Paket: `SchedulesPage.jsx:110-113` memuat brand dalam `try` yang sama dengan daftar paket. Peran `product_manager`, `cs`, `finance`, dan `admin_cabang` (punya `schedule.view`, tanpa `brand.view`) mendapat banner error di atas daftar paket yang sebenarnya termuat.

## Sedang

| ID | Area | Temuan | Lokasi | Cek |
|---|---|---|---|---|
| S-01 | Paket | Arsip paket dengan booking aktif bisa lolos lewat edit. `bolehArsip` meloloskan bila tanggal berangkat sudah lewat, dan jalur `PUT` memberinya tanggal dari request, bukan yang tersimpan. Status `archived` + tanggal kemarin pada paket bulan depan: cek booking aktif dilewati dan tanggalnya ikut mundur. Jalur `/status` memakai tanggal tersimpan dan aman | `internal/schedule/handler.go:217-219`, `806-809` | Ya |
| S-02 | Paket | Form edit kosong tetap bisa disimpan bila data paket gagal dimuat (selain 404). Yang tampil hanya banner error; form berisi nilai bawaan dengan judul "Edit". Menyimpan menimpa paket: fasilitas, gambar, hotel transit hilang. `expected_seat_total` tidak terkirim, jadi penjaga kapasitas dilewati | `ScheduleFormPage.jsx:343-348` | Ya |
| S-03 | Paket | Form edit mengirim `status` dan backend menerapkannya, padahal komentar kode menyebut status tidak diubah di jalur ini. Admin A membuka edit paket published, admin B menjadikannya draft untuk menghentikan penjualan, A menyimpan: paket terbit lagi. Selain kapasitas, tidak ada penjaga edit bersamaan untuk harga, tanggal, atau status | `ScheduleFormPage.jsx:747`, `handler.go:208-211`, `repository.go:899` | Ya |
| S-04 | Paket | Tanggal berangkat bebas diubah saat paket punya booking atau komisi. Server tidak menolak dan tidak meminta konfirmasi; `komisi_aktif` hanya angka untuk peringatan di form. Mengubah tanggal menggeser kapan komisi tersedia dan batas H-30 web publik | `handler.go:212-216`, `repository.go:890-918` | 1x |
| S-05 | Paket | Harga baru berlaku sebagian pada booking lama. `harga_pax` disalin saat booking dibuat, tetapi ganti tipe kamar membaca harga paket saat ini, sehingga satu booking bisa berisi pax harga lama dan baru. Draft booking difinalisasi dengan harga saat draft dibuat. Edit harga paket tidak memberi peringatan | `internal/booking/repository.go` (ganti kamar, finalisasi) | 1x |
| S-06 | Kategori | Set perlengkapan kategori bisa terhapus saat edit. Modal memuat set lewat `GET`; bila gagal, error ditelan dan daftar dianggap kosong. Simpan selalu mengirim `PUT` ganti-semua dengan isi itu. Pemicu: `GET` gagal, atau Simpan diklik sebelum `GET` selesai. Ketiga kategori di dev punya set. Peran bawaan yang punya `masterdata.edit` juga punya `perlengkapan.view`, jadi jalur 403 tidak terjadi dengan peran sekarang | `CategoriesPage.jsx:98-101`, `179-183` | Uji |
| S-07 | Kategori | Setiap simpan kategori menghapus deskripsinya. Payload tidak memuat `description` dan form tidak punya isiannya; `UPDATE` menulis NULL. Ketiga kategori di dev punya deskripsi | `CategoriesPage.jsx:163-170`, `internal/category/repository.go:220-222` | Uji |
| S-08 | Kategori | Ubah dan hapus kategori tidak memeriksa brand pengguna, berbeda dari baca kategori. `brand_ids` diganti seluruhnya, jadi pengguna brand bisa melepas brand lain. Dampak nyata baru muncul bila pengguna ber-brand diberi `masterdata.edit`; sekarang hanya peran holding. Untuk semua pengguna: melepas brand dari kategori yang masih dipakai paket brand itu tidak dicek, dan edit paket berikutnya ditolak "category_id tidak valid" | `internal/category/handler.go:222-306`, `internal/schedule/repository.go:595-598` | 1x |
| S-09 | Kategori | Status "Nonaktif" kategori hanya ditegakkan di frontend. `CategoryAvailable` tidak memeriksa `is_active`, dan paket published tetap membawa kategori nonaktif ke API publik. Di dev belum ada kategori nonaktif | `internal/schedule/repository.go:595-598` | Ya |
| S-10 | Destinasi | Menonaktifkan destinasi mengunci edit semua paket yang memakainya: backend menolak setiap simpan sampai destinasinya diganti, sedangkan paket tetap tampil publik dengan destinasi itu. Halaman Destinasi tidak memperingatkan walau destinasi masih dipakai paket. Form paket menampilkannya sebagai pilihan "(nonaktif)" tanpa error di isiannya | `internal/schedule/handler.go:612-621`, `ScheduleFormPage.jsx:672-677` | Ya |
| S-11 | Bandara | Mengubah kode bandara memutus paket yang memakainya. Paket menyimpan kode sebagai teks, bukan kunci asing. Setelah kode diubah, edit paket lama ditolak "tidak terdaftar", dan bandara bisa dihapus karena hitungan pemakaiannya nol | `internal/airport/repository.go` (Update), `internal/schedule/handler.go:646-671` | 1x |
| S-12 | Hotel | Nama hotel unik untuk seluruh sistem, kota diabaikan. Hotel bernama sama di Makkah dan Madinah tidak bisa didaftarkan keduanya | `migrations/001_init.sql:13`, `internal/hotel/repository.go:173-175` | Ya |
| S-13 | Paket | Simpan paket tidak atomik. Paket disimpan dulu, lalu set perlengkapan lewat panggilan kedua. Bila panggilan kedua gagal, pesan yang tampil "Gagal menyimpan paket" padahal paket sudah tersimpan; simpan ulang ditolak 409 karena nama dan tanggal sama. Pola sama di simpan kategori (konflik slug) | `ScheduleFormPage.jsx:783-799`, `CategoriesPage.jsx:172-193` | 1x |

## Rendah

| ID | Area | Temuan | Cek |
|---|---|---|---|
| R-01 | Paket | Ubah status lewat `/status` tidak memeriksa kelengkapan (maskapai, hotel); 17 pemeriksaan hanya jalan di buat dan edit. Di dev tidak ada paket yang kolom wajibnya kosong | Uji |
| R-02 | Paket | Ubah status dan arsip tidak dalam transaksi dan tanpa kunci baris; booking yang masuk di antara hitung dan simpan menghasilkan paket arsip dengan booking aktif | 1x |
| R-03 | Paket | "Jadikan Draft", "Arsipkan", dan "Publish" di menu baris langsung jalan tanpa konfirmasi; salah klik menurunkan paket dari penjualan. Tabel dimuat ulang dan pencarian, urutan, serta halaman kembali ke awal (`SchedulesPage.jsx:152-168`) | Ya |
| R-04 | Paket | Isian jam membentuk nilai "HH:MM +1" dan menerima nilai sebagian seperti "9"; backend hanya menerima HH:MM atau HH:MM:SS. Di dev tidak ada jam berformat "+1" | Uji |
| R-05 | Paket | Beralih ke penerbangan langsung tidak mengosongkan data transit; detail paket menampilkan bandara transit di samping lencana "Direct" | 1x |
| R-06 | Paket | Peringatan komisi muncul keliru pada Duplikat: salinan baru membawa `komisi_aktif` dan tanggal awal dari paket sumber. Masuk bersama PR 2 menu Agen | 1x |
| R-07 | Paket | Penjaga perubahan belum tersimpan hanya menangani tutup tab. Tombol kembali dan navigasi dalam aplikasi keluar tanpa peringatan; perubahan teks fasilitas dan perlengkapan tidak dihitung | 1x |
| R-08 | Paket | Ringkasan "Seat Tersisa" di detail tidak ikut berubah setelah penyesuaian kursi | 1x |
| R-09 | Paket | Hitungan di dropdown status mengabaikan filter bulan, sedangkan barisnya mengikuti | 1x |
| R-10 | Paket | Detail paket menelan 403 itinerary dan menampilkan "Belum ada data itinerary" untuk peran tanpa `masterdata.view` | 1x |
| R-11 | Paket | Daftar admin: satu kueri hotel transit per baris, error kueri jumlah booking ditelan menjadi 0, dan batas 10.000 baris memotong tanpa penanda | 1x |
| R-12 | Paket | Tanpa batas atas untuk harga, komisi, kursi, dan panjang URL; nilai di luar kolom menjadi 500 generik. Harga pecahan dibulatkan diam-diam. Nominal komisi tidak dibandingkan dengan harga | 1x |
| R-13 | Paket | Nama dan tanggal kembar hanya dijaga di aplikasi, tanpa indeks unik; dua permintaan bersamaan sama-sama lolos | 1x |
| R-14 | Paket | Hapus paket ikut menghapus riwayat penyesuaian kursi, pengumuman, dan add-on (cascade) | 1x |
| R-15 | Maskapai | Kode IATA boleh kembar dan tidak dicek alfanumerik. Di dev belum ada kode kembar | Uji |
| R-16 | Hotel | Simpan bisa diklik saat foto masih diunggah, sehingga foto baru tidak tersimpan. Hapus tanpa penjaga klik ganda: klik kedua menampilkan error padahal data sudah terhapus. Pola hapus sama di Itinerary | 1x |
| R-17 | Data master | Panjang isian tidak divalidasi di hotel, maskapai, bandara, dan itinerary; input terlalu panjang menjadi "terjadi kesalahan internal" | 1x |
| R-18 | Itinerary | Itinerary berlaku untuk semua brand dan perubahannya langsung tampil di semua paket published. Daftar tidak menunjukkan itinerary mana yang sedang dipakai. Itinerary tanpa hari diterima | 1x |
| R-19 | Menu | Hanya Destinasi yang bisa dibuka dengan `masterdata.view`. Kategori, Hotel, Maskapai, dan Itinerary butuh `masterdata.edit`, sehingga peran lihat-saja tidak bisa membukanya walau API baca mengizinkan | 1x |
| R-20 | Paket | `IncrementViewsPublic` di handler paket tidak terdaftar di router (kode mati) | 1x |

## Yang sudah berfungsi benar

- Isolasi brand paket: daftar, detail, ubah, status, kursi, dan hapus memfilter `brand_id`; pemilik dicek ulang di bawah kunci baris. Pengguna brand tidak bisa mengganti `brand_id`. Ganti brand ditolak bila paket punya booking.
- Kursi: `seat_sisa` dari form diabaikan saat edit dan dihitung ulang dari baris terkunci. Kapasitas di bawah alokasi ditolak 409. Semua jalur booking mengunci baris paket sebelum mengurangi kursi.
- Izin: semua rute ubah paket di bawah `schedule.edit`, baca di bawah `schedule.view`; semua rute ubah data master di bawah `masterdata.edit`. Tombol ubah disembunyikan sesuai izin.
- Validasi dasar paket: harga di atas nol, urutan Quad, Triple, Double, pulang setelah berangkat, DP tidak melebihi harga termurah, kota hotel sesuai slot.
- Komisi dan DP disalin ke booking saat booking dibuat; perubahan nominal paket tidak berlaku surut.
- Publik: draft tidak tampil; paket arsip hanya mengembalikan penanda "gone"; nominal komisi tidak ikut terkirim.
- Hapus data master yang masih dipakai ditolak dengan pesan berbahasa Indonesia (kategori, destinasi, bandara dicek dulu; hotel, maskapai, itinerary lewat galat kunci asing). Tidak ada SQL mentah yang sampai ke klien.
- Input uang hanya menerima digit; tanggal dikirim sebagai teks `YYYY-MM-DD` tanpa pergeseran zona waktu; tombol simpan terkunci selama proses.
- Unggah media memeriksa izin per kategori berkas dan membatasi ukuran 10 MB.

## Batas verifikasi

- Tidak ada skenario yang dijalankan lewat API atau browser. S-01, S-02, dan S-03 disimpulkan dari kode.
- Temuan "1x" belum dibaca ulang; nomor baris dan dampaknya perlu dipastikan sebelum diperbaiki.
- Hak akses diperiksa terhadap database dev. Peran di produksi bisa berbeda.
- T-01: asal 22 paket published yang punya nominal komisi belum ditelusuri (kemungkinan seed atau impor). Jumlah booking yang sudah menyalin NULL belum dihitung.

## Keputusan yang dibutuhkan dari pemilik proyek

1. T-01: isian komisi dan DP dikembalikan ke form paket, atau dipindah ke tempat lain (misalnya bawaan per brand atau per kategori)? Booking yang sudah menyalin NULL perlu diisi susulan atau dibiarkan?
2. S-04: ubah tanggal berangkat pada paket yang punya booking cukup diperingatkan, atau perlu alasan dan catatan riwayat?
3. S-05: ganti tipe kamar memakai harga saat booking dibuat atau harga paket sekarang?
4. S-10 dan S-09: destinasi atau kategori nonaktif yang masih dipakai paket: ditolak dinonaktifkan, atau boleh dan paket lama tetap bisa diedit?
5. S-12: nama hotel unik per kota?
6. R-19: peran lihat-saja boleh membuka Kategori, Hotel, Maskapai, dan Itinerary?

## Keputusan pemilik proyek (10 November 2026)

Komisi (T-01):

- Komisi langsung dan bonus pembinaan punya nilai bawaan per brand; paket boleh menimpanya. Paket kosong ikut brand, paket 0 berarti sengaja tanpa komisi, dan nilai paket selalu menang, juga bila bawaan brand kosong.
- Nilai awal: Hana 500.000 dan 100.000. Alsha, Nava, dan Zahara kosong (belum ikut program agen).
- Paket lama yang kosong langsung ikut bawaan brand.
- Booking `baru` dan `dp` yang snapshot-nya kosong diisi susulan dengan nominal efektif paketnya. Booking `lunas` tidak diubah otomatis; daftarnya dari `scripts/daftar_booking_lunas_tanpa_snapshot_komisi.sql`, diputuskan per booking.
- Perubahan bawaan brand hanya berlaku untuk booking baru. Hak ubahnya sama dengan hak ubah brand (`brand.edit`), dan setiap perubahan tercatat di `brand_komisi_log`.
- Rentang komisi di portal agen memakai nominal efektif.
- DP sudah berpola bawaan brand dengan override paket di backend; isiannya dikembalikan ke form paket.

Paket dan master:

- S-04: tanggal berangkat paket yang punya booking aktif tetap boleh diubah, dengan alasan wajib, jumlah booking terdampak ditampilkan, dan catatan riwayat. Paket yang sudah berangkat tidak boleh diubah tanggalnya; tanggal baru tidak boleh lampau. Tidak ada notifikasi otomatis ke jamaah.
- S-05: pax yang pindah kamar memakai harga paket sekarang, pax lain tidak berubah, dan form menampilkan selisih tagihan. Draft booking memakai harga saat difinalisasi.
- S-09, S-10: kategori, destinasi, hotel, maskapai, dan bandara boleh dinonaktifkan walau masih dipakai. Nonaktif berarti tidak bisa dipilih untuk paket baru atau saat mengganti pilihan; paket lama tetap jalan dan tetap bisa diedit. Ditegakkan di backend.
- S-11: kode bandara dan kode maskapai tidak bisa diubah selama masih dipakai paket.
- S-12: nama hotel unik per kota.
- R-19: peran dengan `masterdata.view` boleh membuka halaman master tanpa tombol ubah.

Rencana pengerjaan, empat PR berurutan:

| PR | Isi | Status |
|---|---|---|
| 1 | Komisi bawaan brand, isian komisi dan DP di form paket, isi susulan booking belum lunas, rentang komisi portal agen | Dikerjakan (migrasi 127) |
| 2 | Akses halaman: T-02 dan halaman master untuk peran lihat-saja | Belum |
| 3 | Pengaman simpan paket dan kategori: S-01, S-02, S-03, S-06, S-07, S-08, S-13, serta aturan ubah tanggal berangkat | Belum |
| 4 | Aturan master nonaktif, kode terkunci, hotel unik per kota, harga ganti kamar dan draft | Belum |

Temuan rendah R-01 sampai R-20 ditunda.

## Urutan perbaikan yang disarankan

1. T-01 dan T-02: menghalangi pekerjaan harian dan komisi.
2. S-02, S-06, S-07: kehilangan data saat simpan.
3. S-01, S-03: status paket berubah tanpa pemeriksaan.
4. S-08 sampai S-12 setelah keputusan di atas.
5. Sisanya menurut kebutuhan.
