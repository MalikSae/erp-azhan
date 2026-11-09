# Spesifikasi Keagenan Berbasis Akun (Agen Syiar) — Tahap Awal

Tanggal: 3 November 2026 · Status: disetujui pemilik proyek dalam sesi diskusi 3 Nov 2026; implementasi belum dimulai
Menggantikan sebagian: `agen-azhan.md` (tidak ada di repo; dirujuk oleh komentar kode), `docs/arsip/AUDIT-AGEN-UMROH-2026-09-28.md`
Terkait: `SPEK-WEB-PUBLIK-CMS-2026-10-20.md` (web publik), `SPEK-RBAC-2026-10-02.md` (permission `agen.*`)

## 1. Tujuan dan ruang lingkup

Mengatur ulang keagenan agar sejalan dengan model akun email di web publik `azhan-booking`: agen adalah **akun** yang punya kode referral, bukan baris jamaah. Dokumen ini menyatukan seluruh keputusan agar pekerjaan backend, web publik, dan dashboard admin tidak mengulang pembahasan.

Dalam lingkup tahap awal:
- Pendaftaran agen dari web publik (form gabungan akun + pengajuan).
- Satu keputusan admin atas pengajuan.
- Pengikatan akun dan jamaah ke agen, dan kunci agen atas jamaah.
- Dua jenis komisi: **komisi langsung** dan **bonus pembinaan**.
- Dua jalur booking: portal agen dan web publik lewat link ref.
- Struktur URL web publik untuk area agen.

Di luar lingkup tahap awal (dinonaktifkan, kode tidak dihapus):
- **Cashback** (kredit potongan booking) dan layar "pakai kredit cashback" (B5).
- **Komisi repeat order** dengan pemisahan 50/50. Booking ulang dihitung seperti booking biasa (lihat bagian 5).
- Kolom brand `ujroh_closing`, `ujroh_pembinaan`, dan `ujroh_repeat_booking` tidak dipakai: nominal komisi hanya dari jadwal.
- Verifikasi pelanggan lewat captcha dan verifikasi email untuk alur pendaftaran agen.
- Migrasi data: tidak ada. Sistem belum pernah deploy production, tidak ada agen lama, tidak ada skema PIN.

## 2. Istilah baku

| Istilah | Arti |
|---|---|
| **Akun** | Orang yang punya akun (email + password) dan bisa booking atau memesan. Disebut juga pemesan. |
| **Jamaah** | Orang yang berangkat umroh. Bukan "penumpang". Data identitas milik jamaah, bukan milik akun. |
| **Agen** | Akun yang punya kode referral dan berstatus aktif. Agen tidak harus pernah berangkat umroh. |

Model data mengikuti tiket.com: banyak akun boleh memasukkan jamaah yang sama. Jamaah yang sama dikenali dari identitas (NIK), bukan dari nomor WA atau dari akun. Jamaah belum tentu punya akun, dan akun belum tentu punya jamaah.

Catatan istilah di kode: kode lama menyimpan status agen di tabel `jamaah` dan memakai "jamaah" untuk menyebut agen. Itu yang diubah di bagian 10.

## 3. Ringkasan keputusan

| # | Keputusan |
|---|---|
| K1 | Agen melekat pada akun. Status agen, kode referral, dan upline dipindah dari tabel `jamaah` ke `customer_accounts`. Penerima komisi adalah akun. |
| K2 | Captcha dan verifikasi email ditiadakan untuk alur pendaftaran agen. |
| K3 | Satu form gabungan: buat akun + pengajuan agen. Pemilik akun melewati tahap akun. |
| K4 | Nomor WA harus unik di antara akun: satu WA satu orang. |
| K5 | Admin mengambil satu keputusan per pengajuan dengan lima pilihan (bagian 7). |
| K6 | Agen boleh aktif sebelum membayar biaya pendaftaran, dengan catatan admin wajib. Admin boleh membebaskan biaya sepenuhnya (tagihan dihapus). |
| K7 | Akun terikat ke agen **saat akun dibuat**, dari cookie `azhan_ref` (klik terakhir sebelum daftar). Setelah akun ada, cookie diabaikan. |
| K8 | Akun yang dibuat tanpa agen tetap tanpa agen. Hanya admin yang dapat memasangkannya. |
| K9 | Jamaah terkunci ke agen yang pertama membawanya **selamanya**. Hanya override admin yang dapat memindahkan. |
| K10 | Upline agen baru adalah agen yang mengajak akunnya (upline satu konsep: pengajak sekaligus penerima bonus pembinaan; override admin memindahkan keduanya). Tidak ada kolom "kode agen perekrut" di form. |
| K11 | Komisi tahap awal: komisi langsung dan bonus pembinaan (satu tingkat). Cashback dan komisi repeat dinonaktifkan. |
| K12 | Booking ulang jamaah dihitung seperti booking biasa: komisi langsung dan bonus pembinaan penuh. |
| K13 | Agen tidak mendapat komisi dari booking dirinya sendiri (diubah 2026-11-09; semula "Saya sendiri" memberi komisi atas dirinya). Booking pribadi agen dikenali dari penanda "Saya sendiri" atau dari NIK jamaah yang sama dengan NIK agen (`nik_agen`) pada akun agen aktif atau nonaktif (pemohon yang belum disetujui atau ditolak belum agen), lewat jalur booking mana pun. Rantai bergeser satu tingkat ke atas: komisi langsung ke upline agen, bonus pembinaan ke upline dari upline itu. Agen tanpa upline aktif: tidak ada komisi. Berlaku untuk booking yang lunas sesudah perubahan; komisi yang sudah tercatat tidak diubah. |
| K14 | NIK 16 digit wajib untuk semua pax reguler di semua jalur, termasuk portal agen. Bayi opsional. |
| K15 | Pemesan yang ikut berangkat (PIC) dicocokkan lewat NIK dan nama seperti jamaah lain. |
| K16 | Agen yang memesan jamaah yang sudah terkunci agen lain: booking jalan dengan peringatan, komisi ke pemilik kunci. |
| K17 | Booking agen dan booking pelanggan memakai satu komponen form dan satu inti backend, di dua alamat. |
| K18 | Pax bayi tidak menghasilkan komisi apa pun (langsung maupun pembinaan). |
| K19 | Nominal komisi hanya dari snapshot jadwal. Kolom `ujroh_*` brand tidak dipakai. |
| K20 | Komisi yang sudah tercatat tetap menjadi hak agen walau booking atau pax kemudian dibatalkan. Tidak ada pembalikan komisi. Secara bisnis tidak ada refund penuh, sehingga komisi yang tercatat tidak menjadi beban travel. |
| K21 | "Setujui, bayar menyusul": tagihan tetap tercatat dan dapat dilunasi atau dibebaskan kemudian. |
| K22 | Akun dibuat lewat endpoint atomik tersendiri. Pengajuan agen dikirim lewat panggilan terpisah setelah sesi ada. |
| K23 | Landing `/jadi-agen` menampilkan rentang ujroh dari jadwal tayang, bukan satu angka, karena nominal komisi ada di jadwal. |
| K24 | NIK 16 digit wajib saat mengajukan agen, disimpan di akun (`customer_accounts.nik_agen`, unik per brand). "Saya sendiri" di portal agen hanya diterima bila NIK jamaah pertama sama dengan NIK agen; bila beda, booking ditolak ("Pilihan Saya sendiri hanya untuk data Anda sendiri"). Kartu B1 memperingatkan admin bila NIK pendaftar sudah tercatat atas nama lain atau jamaahnya terkunci ke agen lain. |

## 4. Aturan pengikatan

### 4.1 Akun ke agen

- Cookie `azhan_ref` (httpOnly, 90 hari, klik terakhir menimpa) sudah dipasang oleh `azhan-booking/src/proxy.js:120-124` dari parameter `?ref=KODE`.
- Saat akun dibuat (`/jadi-agen/daftar` tahap 1, atau `/daftar`), kode di cookie diselesaikan menjadi agen aktif pemilik kode di brand yang sama. Bila ada, `direkrut_oleh_akun_id` diisi. Bila kode kosong, tidak dikenal, atau milik agen nonaktif, diabaikan diam-diam (sama seperti `ResolveKodeReferral`).
- Setelah akun ada, cookie tidak pernah dibaca lagi untuk pengikatan akun.
- Akun tanpa cookie saat dibuat tetap tanpa agen. Admin dapat memasangkannya (override, bagian 4.4).

Contoh: hari 1 Z klik link W; hari 2 Z klik link X lalu daftar akun; hari 3 Z klik link Q. Akun Z terikat ke X.

### 4.2 Jamaah ke agen (kunci)

- Kunci disimpan di jamaah (`kaitan_status`, `kaitan_sumber`, penunjuk agen) dan berlaku **seumur hidup**.
- Aturan first-set-wins tetap: hanya jamaah berstatus `belum_ditentukan` yang boleh diikat (`internal/agen/kaitan.go:62`).
- Saat booking dibuat dan jamaah barunya belum terikat, jamaah itu diikat ke agen yang mengikat akun pemesan. Pada booking portal agen, jamaah baru diikat ke agen pembuat booking.
- Jamaah yang sudah terikat agen lain tidak diubah.

Prioritas penentu agen untuk jamaah baru:
1. Booking lewat portal agen: agen pembuat booking.
2. Booking lewat web publik: agen yang mengikat akun pemesan. Cookie tidak dibaca.
3. Tidak ada keduanya: jamaah tetap `belum_ditentukan`.

Catatan: kode sekarang memprioritaskan cookie saat checkout dan hanya mengikat jamaah. Itu berubah (bagian 10).

### 4.3 Mengenali jamaah yang sama

- NIK (16 digit) adalah kunci utama. Pencocokan memakai `cariJamaahByNIK` (`internal/selfbooking/nik.go:42`) dengan kunci unik `(brand_id, nik)`.
- NIK sama dan nama sama (setelah `normalizeName`): jamaah lama dipakai ulang, tanpa menimpa data apa pun.
- NIK sama dan nama berbeda: ditolak dengan pesan generik ("Data Jamaah N belum dapat diproses...") yang tidak membuka nama terdaftar. Percobaan gagal dibatasi per akun.
- Bayi tanpa NIK: dicocokkan lewat nama dan tanggal lahir.
- Nomor WA bukan kunci identitas jamaah. Batas unik nomor HP jamaah per brand (`uq_jamaah_brand_hp`, migrasi 060) dicabut.

### 4.4 Override admin

Admin Master dapat memindahkan jamaah ke agen lain, dan memasangkan atau mengganti agen akun, untuk semua `kaitan_sumber` (bukan hanya `jalur3` seperti `kaitan.go:18`). Bila komisi atas jamaah itu sudah tercatat, harus ada cara memindahkan komisi yang sudah ada (bagian 10, butir 8); aturan lama yang menolak tanpa jalan keluar tidak dipertahankan. Setiap override dicatat di log kaitan yang sudah ada.

## 5. Aturan komisi

### 5.1 Jenis dan penerima

| Jenis | Penerima | Syarat |
|---|---|---|
| Komisi langsung | Agen yang mengunci jamaah (pax) tersebut | Agen berstatus `aktif`, nominal snapshot > 0 |
| Bonus pembinaan | Upline agen itu (satu tingkat) | Upline berstatus `aktif`, nominal snapshot > 0 |

- Komisi dicatat per pax aktif saat booking berstatus `lunas`, di dalam transaksi yang sama (`komisi.OnBookingLunas`), idempoten.
- Nominal flat per pax dari snapshot jadwal (`nominal_komisi_langsung`, `nominal_bonus_pembinaan`), disalin ke booking saat booking dibuat. Snapshot NULL berarti tidak ikut program.
- Pax tanpa agen tidak menghasilkan komisi.
- Pax bayi (`pax_type` infant) tidak menghasilkan komisi, baik langsung maupun pembinaan.
- Komisi yang sudah tercatat tetap menjadi hak agen walau booking atau pax kemudian dibatalkan. Tidak ada pembalikan komisi.
- Booking ulang (`isRepeatOrder`) tidak lagi dibedakan: cabang repeat dan cashback di `internal/komisi/kalkulasi.go` dimatikan lewat satu flag, logikanya tidak dihapus.

### 5.2 Skenario acuan

1. Agen A membawa jamaah X: A mendapat komisi langsung.
2. X mendaftar jadi agen (akun X terikat ke A): upline X adalah A.
3. Agen X membawa jamaah Y: X komisi langsung, A bonus pembinaan.
4. Agen X mendaftarkan dirinya lewat portal agen ("Saya sendiri"): X tidak mendapat komisi (K13). Komisi langsung ke A (upline X), bonus pembinaan ke U (upline A). Hasilnya sama dengan skenario 6.
5. Jamaah Y booking ulang: X komisi langsung, A bonus pembinaan (penuh).
6. Agen X memesan untuk dirinya lewat web publik: dihitung booking ulang. Komisi langsung ke agen yang mengunci jamaah X (A), bonus pembinaan ke upline A.
7. Agen Y mendaftarkan jamaah A yang terkunci ke X: booking jalan dengan peringatan, komisi ke X.

### 5.3 Saldo dan pencairan

Tidak berubah: saldo tersedia setelah `berangkat_tanggal` lewat, saldo tertahan sebelumnya. Pencairan minimal Rp500.000, satu pengajuan pending, disetujui Admin Master (`agen.payout.approve`). Saldo hanya terdiri dari dua jenis komisi di atas. Penerima dan pemilik saldo adalah akun. Komisi dari booking yang dibatalkan setelah lunas tetap menjadi saldo tersedia setelah tanggal berangkat jadwal lewat.

## 6. Alur pendaftaran agen

### 6.1 Pintu masuk

Pengunjung ada di tiga keadaan: belum punya akun, punya akun tapi belum login, atau sudah login. Setelah login, sistem membaca status agen akun dan menampilkan satu dari enam layar di `/akun/agen` (bagian 9.2).

### 6.2 Form gabungan (`/jadi-agen/daftar`)

1. Belum login: tahap 1 (nama, email, WA, password). Bila email atau WA sudah terdaftar sebagai akun, pengunjung diarahkan ke Masuk. Akun dan sesi langsung dibuat, tanpa captcha dan tanpa verifikasi email. Agen akun diikat dari cookie (bagian 4.1).
2. Tahap 2: domisili, foto agen, persetujuan S&K agen (teks dari `brands.syarat_ketentuan_agen`).
3. Kirim pengajuan dalam satu transaksi di server: status agen menjadi `pengajuan`, tagihan dibuat dari `biaya_pendaftaran_agen` brand (snapshot), dan tagihan Rp0 langsung `terverifikasi`.
4. Halaman status: menunggu keputusan admin, atau unggah bukti bayar.

Upload foto membutuhkan sesi login, jadi form berupa dua tahap di satu halaman (akun dulu, lalu data agen). Akun dibuat lewat satu endpoint atomik tersendiri (K22), dan pengajuan agen dikirim lewat panggilan terpisah setelah sesi ada. Bila terputus di antaranya, akun sudah ada dan pengguna melanjutkan dari `/jadi-agen/daftar` tahap 2.

### 6.3 Status

Status agen akun: `tidak_aktif` (belum pernah, atau ditolak), `pengajuan`, `aktif`, `nonaktif`. Status pembayaran pendaftaran: `menunggu_verifikasi`, `terverifikasi`, `ditolak`, ditambah `dibebaskan` (baru). Penolakan pengajuan menutup siklus; pengajuan ulang membuat siklus baru dari awal.

## 7. Keputusan admin (layar B1)

Satu kartu pengajuan menampilkan data pemohon dan bukti transfer. Admin memilih satu:

| Aksi | Syarat | Hasil |
|---|---|---|
| Setujui | Bukti valid, atau biaya Rp0 | Agen aktif, tagihan lunas, kode referral dibuat |
| Bebaskan biaya | Catatan admin wajib | Agen aktif, tagihan dihapus |
| Minta unggah ulang | Bukti kurang jelas | Tetap `pengajuan`, kembali ke antrian setelah diunggah |
| Tolak | Alasan wajib | Kembali `tidak_aktif`, boleh mengajukan ulang |
| Setujui, bayar menyusul | Catatan admin wajib | Agen aktif, tagihan belum lunas; lunas setelah bukti diverifikasi di B2a |

Implementasi: satu aksi gabungan di server yang memanggil verifikasi pembayaran dan persetujuan dalam satu transaksi (`verifikasiPembayaranTx` lalu `setujuiTx`). Endpoint terpisah yang ada tidak dihapus. Admin dapat mengubah agen aktif menjadi `nonaktif` dan sebaliknya (B2a), tidak berubah.

## 8. Booking: dua jalur, satu inti

Dua jalur booking umroh: (1) didaftarkan agen di portal agen, (2) daftar mandiri di web publik lewat link ref. Keduanya memakai request yang sama (`BookingRequest`), validasi yang sama (`validateCheckout`), harga, penahanan kursi 24 jam, cutoff, snapshot komisi, invoice, dan pembayaran.

| | Web publik (`/checkout`) | Portal agen (`/akun/agen/pesan`) |
|---|---|---|
| Pemesan | Akun biasa | Akun berstatus agen aktif |
| Penentu agen komisi | Kunci jamaah, lalu agen akun | Agen pembuat booking, kecuali jamaah sudah terkunci agen lain |
| Memilih jamaah | Jamaah tersimpan akun | "Saya sendiri", "Jamaah Saya", atau isi baru |
| Pemesan ikut berangkat | Pilihan pengguna | Default tidak ikut |
| Perlindungan | Captcha | Tanpa captcha, batas 20 booking per jam per agen |
| Pemberitahuan | Email "Booking berhasil dibuat" | Tidak ada email |

Aturan tambahan portal agen:
- NIK 16 digit wajib untuk pax reguler (K14).
- Pilihan "Saya sendiri" untuk jamaah pertama menandai booking pribadi agen (K13): komisinya mengalir ke upline agen, bukan ke agen. Pilihan ini adalah penanda eksplisit karena akun tidak lagi terhubung ke satu jamaah; NIK yang sama dengan NIK agen juga dikenali tanpa penanda.
- Bila jamaah sudah terkunci agen lain, agen melihat: "Jamaah ini sudah terikat agen lain. Booking ini tidak menghasilkan komisi untuk Anda." Nama pemilik kunci dan data jamaah tidak ditampilkan. Peringatan hanya muncul bila NIK dan nama cocok.
- Booking tetap tercatat sebagai dibuat oleh agen pembuat, sementara komisi mengalir ke pemilik kunci. Dua hal ini dicatat terpisah supaya peringkat agen tidak salah.
- NIK sama dengan nama berbeda ditolak generik.

## 9. Web publik Hana (`azhan-booking`)

### 9.1 Peta URL

| Alamat | Isi | Akses | Indeks |
|---|---|---|---|
| `/jadi-agen` | Landing: manfaat, ujroh, S&K, tombol daftar. Tombol mengikuti status akun. | Publik | index |
| `/jadi-agen/daftar` | Form gabungan (bagian 6.2). Pemilik akun hanya melihat tahap 2. | Publik | noindex |
| `/akun/agen` | Satu pintu; layar sesuai status (9.2) | Login | noindex |
| `/akun/agen/jamaah` | Jamaah yang terkunci ke agen | Agen aktif | noindex |
| `/akun/agen/komisi` | Riwayat komisi langsung dan bonus pembinaan | Agen aktif | noindex |
| `/akun/agen/pencairan` | Saldo dan pengajuan pencairan | Agen aktif | noindex |
| `/akun/agen/pesan` | Form booking mode agen | Agen aktif | noindex |
| `/checkout` | Form booking mode pelanggan (komponen sama) | Login | noindex |

Tidak dibuat: `/akun/agen/daftar`. Tautan referral tetap `?ref=KODE` di halaman mana saja; dashboard menyediakan tombol salin untuk beranda dan tiap halaman paket.

### 9.2 Layar di `/akun/agen` menurut status

| Keadaan akun | Layar |
|---|---|
| Belum pernah mengajukan | Ajakan dan tautan ke `/jadi-agen/daftar` |
| Pernah ditolak | Alasan penolakan dan tautan ajukan ulang |
| Pengajuan diproses | Status: menunggu keputusan, atau unggah ulang bukti |
| Agen aktif, tagihan lunas atau dibebaskan | Dashboard agen |
| Agen aktif, tagihan belum lunas | Dashboard dengan ajakan unggah bukti |
| Agen nonaktif | Halaman tertutup: hubungi admin |

### 9.3 Navigasi

- Tautan "Jadi Agen" di footer dan di `AccountMenu` (`src/components/auth/AccountMenu.jsx`). Item berubah menjadi "Dashboard Agen" setelah akun menjadi agen aktif.
- Tidak masuk menu utama header (Beranda, Umroh, Tentang Hana, Blog).
- Layar Akun di mobile memuat item yang sama.

### 9.4 Tampilan

Pola UI ikut `design-system.md` dan skrip `npm run check:design` di `azhan-booking`. Teks antarmuka mengikuti `docs/copy-guide.md` proyek itu.

## 10. Perubahan backend (`erp-azhan`)

Semua butir di bawah belum dikerjakan. Nomor migrasi: berikutnya setelah migrasi terakhir.

1. **Migrasi agen ke akun.** Tambahkan ke `customer_accounts`: `status_agen`, `kode_referral`, `foto_agen_url`, `domisili`, `menyetujui_syarat_ketentuan_agen_at`, `diajukan_agen_at`, `disetujui_agen_at`, `disetujui_agen_oleh`, `direkrut_oleh_akun_id`, `upline_akun_id` (snapshot saat disetujui). Kolom agen di `jamaah` (`status_agen`, `kode_referral`, `upline_jamaah_id`, `foto_agen_url`, `domisili`) dihapus. `jamaah.direkrut_oleh_jamaah_id` menjadi penunjuk ke akun agen. Tidak ada data untuk dimigrasi.
2. **Tabel terdampak:** `transaksi_komisi.jamaah_penerima_id` menjadi akun penerima; `pembayaran_pendaftaran_agen.jamaah_id` menjadi akun; `pengajuan_pencairan.jamaah_agen_id` menjadi akun. Sesuaikan `komisi.HitungSaldo`.
3. **Pengikatan akun saat dibuat.** Endpoint registrasi akun membaca cookie referral (diteruskan dari route handler web publik, seperti `kode_referral` di `account/[...path]/route.js:40`) dan mengisi `direkrut_oleh_akun_id`.
4. **Prioritas pengikat saat booking.** Ubah `IkatRombongan`: agen akun menang atas cookie; cookie tidak dibaca saat checkout. Pada portal agen, agen pembuat booking mengikat jamaah baru.
5. **`resolveAkunPIC` dan PIC.** Akun tidak lagi terhubung ke satu jamaah (`customer_accounts.jamaah_id` dihapus atau menjadi opsional). PIC yang ikut berangkat dicocokkan seperti anggota (`resolveAnggotaPublik`), tidak lagi dibuat jamaah baru tanpa NIK (`pasangNIKPIC`).
6. **Portal agen booking.** Ubah `resolvePaxAgen` (`selfbooking/agen_booking.go:30`): jamaah milik agen lain atau nomor milik agen lain tidak lagi ditolak (`ErrJamaahBukanMilik`, `ErrNomorMilikLain`), melainkan jalan dengan peringatan. Tambahkan NIK wajib dan pilihan "Saya sendiri". Catat agen pembuat booking terpisah dari pemilik komisi.
7. **Komisi.** Matikan cabang repeat dan cashback di `kalkulasi.go` lewat flag. Tambahkan aturan: jamaah yang dipilih lewat "Saya sendiri" di portal menghasilkan komisi langsung untuk agen itu sendiri. Saring pax bayi (`pax_type` infant) dari perhitungan. Tidak ada pembalikan komisi saat booking atau pax batal; ubah nama dan pesan tes `TestAuditAgenCancelledSourceStillBecomesWithdrawable` (`internal/komisi/audit_agen_test.go`) karena perilakunya kini disengaja.
8. **Override admin (C4).** Longgarkan pembatasan sumber `jalur3`, dan sediakan pemindahan komisi yang sudah tercatat. Atur ulang temuan audit KA-01.
9. **Keputusan admin gabungan.** Endpoint tunggal untuk lima aksi bagian 7, dengan catatan wajib dan status pembayaran `dibebaskan`.
10. **Rute portal agen.** Pasang kembali `/api/portal/agen/*` (komentar `cmd/api/main.go:242`) berbasis akun email, bukan PIN. Handler dan tes yang ada menjadi dasar.
11. **Batas unik HP jamaah.** Cabut `uq_jamaah_brand_hp`. Nomor WA jamaah menjadi data kontak biasa.

## 11. Perubahan web publik (`azhan-booking`)

1. Halaman `/jadi-agen` dan `/jadi-agen/daftar` (form dua tahap). Landing menampilkan rentang ujroh dari jadwal tayang (K23); rancangan tampilannya masih perlu dibuat.
2. Halaman `/akun/agen` dengan enam layar status, serta `/akun/agen/jamaah`, `/komisi`, `/pencairan`, `/pesan`.
3. Komponen form booking dipakai ulang di `/checkout` dan `/akun/agen/pesan` dengan properti mode.
4. Item "Jadi Agen" di footer dan `AccountMenu`; mode "Dashboard Agen" setelah aktif.
5. Registrasi akun meneruskan cookie referral ke backend (K7).
6. Hapus captcha dan verifikasi email dari alur ini, dan pasang pembatas permintaan per IP.

## 12. Perubahan dashboard admin (`master-dashboard`)

1. Layar B1 menjadi satu kartu dengan lima aksi (bagian 7).
2. Hilangkan layar, kolom, dan filter yang terkait cashback dan komisi repeat dari riwayat komisi dan peringkat.
3. Layar Ganti Kaitan Agen (C4) menjangkau semua sumber kaitan dan akun, serta komisi yang sudah ada.
4. Penanda duplikat jamaah: nama dan tanggal lahir sama tetapi NIK berbeda masuk antrian admin sebelum booking lunas.

## 13. Keamanan dan penyalahgunaan

- Tanpa captcha dan verifikasi email pada pendaftaran, antrian B1 bisa dibanjiri pengajuan palsu. Wajib ada batas permintaan per IP dan per perangkat, dan admin menyaring manual.
- Email tidak terverifikasi berarti notifikasi hasil pengajuan bisa salah alamat. Kanal utama untuk kabar keputusan adalah WhatsApp.
- WA unik antar akun membuka kemungkinan menebak nomor yang terdaftar. Batasi percobaan per IP, dan jangan menampilkan siapa pemilik nomor.
- Pesan penolakan NIK tetap generik (tidak menjadi oracle data pihak lain). Peringatan "sudah terikat agen lain" hanya muncul bila NIK dan nama cocok sekaligus.
- Kunci seumur hidup berarti jamaah yang tidak pernah closing tetap terkunci. Satu-satunya jalan keluar adalah override admin. Diputuskan 2026-11-03: tidak ada kunci sementara atau pelepasan otomatis. Risiko agen mengunci massal dinilai rendah karena NIK jamaah sulit didapat sembarang orang; booking palsu ditindak manual oleh admin/travel.

## 14. Keputusan lanjutan dan hal yang masih terbuka

Diputuskan pemilik proyek pada 3 Nov 2026 (sudah tercermin di K18–K22 dan bagian terkait):
- Pax bayi tidak menghasilkan komisi.
- Nominal komisi hanya dari snapshot jadwal.
- Komisi tetap menjadi hak agen walau booking atau pax dibatalkan setelah lunas.
- Akun dibuat lewat endpoint atomik, pengajuan agen lewat panggilan terpisah.
- "Bayar menyusul": tagihan tetap tercatat dan dapat dilunasi atau dibebaskan kemudian.
- `GLOSSARY.md` dibuat di akar repo.
- Landing `/jadi-agen` menampilkan rentang ujroh dari jadwal tayang (K23).
- Secara bisnis tidak ada refund penuh, sehingga komisi yang sudah tercatat aman (K20).

Masih terbuka:
1. Pengiriman pesan WhatsApp untuk hasil keputusan admin: pengirimnya belum diperiksa.
2. Konsekuensi dari K9: agen dapat mendaftarkan jamaah lalu membatalkan hanya untuk mengunci jamaah itu. Satu-satunya penangkal saat ini adalah override admin dan penanda duplikat. Pemilik proyek belum memutuskan apakah ini perlu kontrol tambahan.

## 15. Urutan kerja yang disarankan

1. Migrasi dan refactor backend (bagian 10, butir 1–3, 11), dengan tes `testdb`.
2. Aturan pengikatan dan komisi (butir 4–7), dengan tes untuk tujuh skenario bagian 5.2.
3. Keputusan admin gabungan dan override (butir 8–9).
4. Rute portal agen berbasis akun (butir 10).
5. Halaman web publik (bagian 11), bergantung pada API butir 4.
6. Penyesuaian dashboard admin (bagian 12).

Setiap tahap dapat dirilis terpisah selama tahap 1 dan 2 selesai lebih dulu.

## 16. Catatan implementasi tahap 1 dan 2

Nama yang dipakai di kode (migrasi 114), bila berbeda dari usulan di atas:

- `transaksi_komisi.akun_penerima_id` (usulan: penerima menjadi akun); `pembayaran_pendaftaran_agen.akun_id`; `pengajuan_pencairan.akun_agen_id`; `pemakaian_cashback.akun_id`.
- `jamaah.direkrut_oleh_akun_id` (nama lama `direkrut_oleh_jamaah_id`); `jamaah_kaitan_log.direkrut_oleh_akun_id_lama` dan `_baru`.
- Rekening pencairan (`rekening_bank_agen`, `rekening_nomor_agen`, `rekening_atas_nama_agen`) ikut pindah ke `customer_accounts`.
- `customer_accounts.jamaah_id` dibiarkan opsional dan tidak lagi ditulis.
- Tambahan: `bookings.dibuat_oleh_agen_akun_id` (agen pembuat booking portal, terpisah dari penerima komisi) dan `booking_pax.saya_sendiri_akun_id` (penanda "Saya sendiri").
- Flag komisi: `komisi.KomisiRepeatCashbackAktif` (default `false`).
