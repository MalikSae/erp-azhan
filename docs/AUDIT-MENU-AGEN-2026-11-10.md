# Audit fitur menu "Agen" (dashboard office)

Tanggal: 10 November 2026. Kode yang diaudit: `main` pada merge commit `bef7683`.

## Cakupan dan metode

Tiga item grup sidebar "AGEN" di Master Dashboard: Daftar Agen (beserta Detail Agen), Riwayat Komisi, Pengajuan Agen. Ditambah tiga layar agen yang punya rute tetapi tidak punya item sidebar: Persetujuan Pencairan (`/komisi`), Ganti Kaitan Agen (`/komisi/ganti-kaitan`), dan Peringkat Agen (`/analytics/peringkat-agen`). Yang diperiksa: halaman React, handler dan repository Go di `internal/agen` dan `internal/komisi`, pemanggilnya di `internal/payment`, serta migrasi terkait. Acuan aturan: `docs/spek/SPEK-KEAGENAN-AKUN-2026-11-03.md` (keputusan K1 dan seterusnya) dan `GLOSSARY.md`.

Metode: pembacaan kode. Kolom "Cek" menandai temuan yang baris kodenya dibaca ulang secara terpisah ("Ya"), dijalankan terhadap database ("Uji"), atau dibaca sekali ("1x"). Temuan berlabel "Dugaan" punya bagian yang belum ditelusuri.

Pemeriksaan otomatis:

- `go vet ./internal/agen/... ./internal/komisi/...`: bersih.
- `ERP_TEST_DB=1 go test -count=1 -p 1 ./internal/agen/... ./internal/komisi/...`: lulus (database `erp_azhan_dev`, transaksi di-rollback).
- T-01 direproduksi dengan tes sementara di `internal/komisi` (dihapus setelah dijalankan); hasilnya dikutip di bawah.

Tidak dilakukan: uji lewat API hidup dan browser untuk layar-layar ini, dan pemeriksaan portal agen di repo `azhan-booking`.

## Ringkasan

| ID | Tingkat | Area | Temuan | Cek |
|---|---|---|---|---|
| T-01 | Tinggi | Komisi | Komisi tercatat ganda bila booking keluar lalu masuk lunas lagi setelah penerimanya berubah | Uji |
| T-02 | Tinggi | Pencairan | Layar Persetujuan Pencairan dan Peringkat Agen tidak punya tautan dari mana pun; pengajuan pencairan agen tidak terlihat staf | Ya |
| S-01..S-13 | Sedang | Semua | Lihat bagian Sedang | campuran |
| R-01..R-16 | Rendah | Semua | Lihat bagian Rendah | campuran |

Tidak ada temuan Kritis. Mesin keputusan pengajuan agen (`keputusanTx`), isolasi brand, dan penguncian baris pada pencairan dan override dinilai kokoh; rinciannya di bagian "Yang sudah berfungsi benar".

## Tinggi

### T-01 · Komisi ganda saat booking kembali lunas setelah penerima berubah

`internal/komisi/kalkulasi.go:121-135` menghitung penerima komisi dari keadaan saat booking menjadi lunas, setiap kali. Kunci unik ledger memuat penerima (`UNIQUE (booking_pax_id, jenis, akun_penerima_id)`), dan penyisipan memakai `ON DUPLICATE KEY UPDATE id = id` (`kalkulasi.go:175-179`). Akibatnya pemanggilan ulang hanya aman bila penerimanya sama. Bila penerima berubah di antara dua lunas, baris baru tercatat di samping baris lama.

Booking keluar-masuk lunas adalah operasi biasa: diskon dihapus lalu ditambah, pax dibatalkan, dan sejak audit menu Jamaah & Booking juga pembatalan pembayaran (void) yang dicatat ulang.

Direproduksi di database (tes sementara, transaksi di-rollback):

```text
lunas pertama: [Adam:pembinaan:300000 Nuh:langsung:1000000]
(upline Nuh diganti dari Adam ke Baqir; booking lunas -> dp -> lunas)
lunas kedua : [Adam:pembinaan:300000 Baqir:pembinaan:300000 Nuh:langsung:1000000]
```

Jalur lain dengan akibat yang sama (dibaca, tidak dijalankan):

- Jamaah yang terkunci ke agen A menjadi agen sendiri dengan upline U. Pada lunas berikutnya ia dikenali sebagai "diri agen" lewat NIK (K13), sehingga komisi langsung tercatat untuk U di samping A.
- Jamaah berstatus `belum_ditentukan` saat booking pertama lunas (tanpa komisi), lalu terkunci ke agen B lewat booking kedua. Booking pertama keluar-masuk lunas dan B menerima komisi booking pertama.
- Agen nonaktif yang diaktifkan lagi: komisi yang dulu dilewati muncul saat lunas ulang. Tes `TestAuditAgenSkippedCommissionAppearsAfterReactivation` mengunci ini sebagai perilaku yang diharapkan.

`TestIdempotenSaatLunasBerulang` hanya menguji penerima yang sama.

Perlu keputusan. Dua arah: komisi hanya diproses pada lunas pertama (`pertama_lunas_at` sudah ada), atau pax yang sudah punya baris berjenis sama dilewati.

### T-02 · Persetujuan Pencairan dan Peringkat Agen tidak terjangkau

Rute `/komisi` (Persetujuan Pencairan) dan `/analytics/peringkat-agen` masih ada di `frontend/master-dashboard/src/App.jsx:97,107`, tetapi tidak ada tautan ke keduanya di seluruh `frontend/`. Item sidebar-nya hilang di commit `a572256` ("rapikan layout, sidebar, rute"); sisa logikanya masih terlihat di `Sidebar.jsx` (`end={item.path === '/komisi' ...}`). Ganti Kaitan Agen hanya terjangkau dari tombol "Ganti upline" di Detail Akun.

Skenario: agen mengajukan pencairan dari portal. Tidak ada item menu, angka penanda, atau tautan yang membawa staf ke antrean itu. Pengajuan menunggu sampai seseorang mengetik URL-nya. Database dev belum punya pengajuan pencairan, jadi belum ada yang tertahan.

Terkait:

- Ketiga rute dijaga `holdingOnly`, sedangkan backend memakai izin `agen.payout.approve`, `agen.kaitan.edit`, dan `agen.view`. Pengguna holding tanpa izin itu membuka halaman lalu mendapat 403; admin brand tidak bisa membuka Peringkat Agen walau backend mendukung cakupan brand.
- Hanya peran Super Admin yang memegang `agen.payout.approve` (dicek di database dev). Peran Finance tidak.

Saran: kembalikan item sidebar "Pencairan Komisi" (dengan angka pengajuan yang menunggu) dan "Peringkat Agen", dan samakan penjaga rute dengan izin backend.

## Sedang

| ID | Area | Temuan | Lokasi | Cek |
|---|---|---|---|---|
| S-01 | Pengajuan | Bukti susulan agen aktif tidak bisa ditolak dari UI: "Minta unggah ulang" hanya ditawarkan untuk status `pengajuan`, padahal backend menerimanya untuk agen aktif yang tagihannya belum lunas. Pilihan yang tersisa hanya Setujui (tagihan jadi lunas) atau Bebaskan | `PersetujuanAgenPage.jsx:100`, `internal/agen/keputusan.go:115-119` | Ya |
| S-02 | Pengajuan | Penolakan tidak melepas `nik_agen`. NIK unik per brand, jadi pemilik NIK asli tidak bisa mengajukan; NIK di akun yang ditolak tidak bisa dikosongkan, hanya diganti 16 digit lain | `internal/agen/repository.go:288`, `migrations/116_nik_agen.sql:6` | Ya |
| S-03 | Pengajuan | Agen `nonaktif` dengan tagihan belum lunas terjebak di antrean: tampil dan terhitung di kartu ringkasan, tetapi semua keputusan ditolak karena syaratnya `status == "aktif"` | `internal/agen/keputusan.go:71-73`, `repository.go:180-184` | Ya |
| S-04 | Pengajuan | Kartu "Bukti Perlu Diperiksa" menghitung semua tagihan `menunggu_verifikasi`, termasuk yang belum punya bukti. "Total Pengajuan" ikut menghitung agen aktif yang tinggal tagihan | `PersetujuanAgenPage.jsx:259-261` | Ya |
| S-05 | Pengajuan | Catatan yang diketik untuk satu aksi ikut terkirim saat admin berganti ke aksi lain (misalnya ketik alasan tolak, lalu pilih Setujui). Catatan "bayar menyusul" tertimpa saat pelunasan atau pembebasan; tidak ada riwayat catatan | `PersetujuanAgenPage.jsx:187,440`, `keputusan.go:75-81,107` | 1x |
| S-06 | Pengajuan | `catatan_admin` ikut dikirim ke pemohon lewat `GET /api/portal/agen`, termasuk catatan pembebasan dan bayar menyusul yang tersirat internal. Tampilannya di portal belum diperiksa (Dugaan) | `internal/agen/model.go:40`, `repository.go:101` | 1x |
| S-07 | Pengajuan | Biaya pendaftaran agen yang diverifikasi tidak masuk pencatatan keuangan mana pun. Verifikasi hanya ya atau tidak: tanpa nominal diterima, tanggal transfer, atau rekening tujuan. Spek tidak mengatur; celah kebijakan | `internal/agen/repository.go:326-328` | 1x |
| S-08 | Komisi | Tanggal berangkat jadwal bisa diubah setelah komisinya dicairkan. Saldo tersedia dihitung dari tanggal jadwal saat dibaca, sehingga menjadi negatif; pencairan berikutnya dan pemindahan komisi ikut tertolak | `internal/komisi/saldo.go:28-37`, `internal/schedule` (validasi tanggal tidak memeriksa komisi) | 1x |
| S-09 | Komisi | Booking pribadi agen (K13): komisi mengalir ke upline akun, bukan ke pemilik kunci jamaah. Bila keduanya berbeda, pemilik kunci tidak menerima apa pun; bila agen tanpa upline, tidak ada komisi. K13 dan skenario 5.2 nomor 6 di spek tidak sejalan untuk kasus ini | `internal/komisi/kalkulasi.go:129-131` | Ya |
| S-10 | Komisi | K13 terlewati bila jamaah agen dibuat tanpa NIK lewat booking admin: pencocokan "diri agen" butuh NIK, dan jalur admin tidak mewajibkannya (K14 mewajibkan). Agen menerima komisi atas dirinya sendiri | `kalkulasi.go:130` | Ya (kueri) |
| S-11 | Komisi | Booking dari CRM dengan beberapa pax menyimpan semua kursi atas nama jamaah PIC, sehingga komisi dicatat N kali ke pemilik kunci PIC. Kunci jamaah lain dan aturan bayi tidak bisa berlaku | `internal/crmdeal/repository.go:212-221` | Ya |
| S-12 | Komisi | Riwayat Komisi berhenti di 200 baris terbaru. Total, pencarian, dan urutan hanya atas baris itu; `offset` didukung backend tetapi tidak dipakai. Kartu "Total komisi", "Tersedia", dan "Tertahan" tidak diberi tanda bahwa nilainya terpotong | `RiwayatKomisiPage.jsx:15,54,68-75`, `internal/agen/kinerja.go:72-74` | 1x |
| S-13 | Pencairan | Layar Persetujuan Pencairan memuat semua status tanpa filter, sedangkan backend mengurutkan dari yang terlama dan memotong di 500 baris. Setelah total pengajuan melewati 500, pengajuan terbaru (yang menunggu) tidak tampil | `KomisiReferralPage.jsx:67`, `internal/agen/pencairan.go:256` | Ya |

## Rendah

| ID | Area | Temuan | Cek |
|---|---|---|---|
| R-01 | Daftar Agen | "Closing" menghitung jamaah yang terkunci ke agen (`direkrut_oleh_akun_id`), bukan jamaah yang sudah booking atau lunas. Jamaah yang dikaitkan lewat form Tambah Jamaah ikut terhitung (`internal/agen/kinerja.go:142-146,238`) | Ya |
| R-02 | Detail Agen | Nonaktifkan dan aktifkan agen tidak meminta alasan dan tidak tercatat di database; pelaku hanya ada di log server (`internal/agen/handler.go:191`). Tombolnya tampil tanpa cek izin `agen.edit` | Ya |
| R-03 | Detail Agen | Status tagihan `dibebaskan` tidak punya label di riwayat siklus; tampil sebagai teks mentah (`DetailAgenPage.jsx:40-44`) | Ya |
| R-04 | Daftar Agen | Daftar dimuat penuh tanpa batas, dengan dua subkueri per baris; cari, filter, dan paginasi di browser. Parameter `q` di backend tidak dipakai | Ya |
| R-05 | Pengajuan | Endpoint lama (`/setujui`, `/tolak`, `/pembayaran/{id}/verifikasi`, `/pembayaran/{id}/tolak`) masih terpasang dan melewati aturan kartu keputusan, misalnya mengaktifkan agen dengan tagihan belum lunas tanpa catatan. Tidak terjangkau dari UI; spek meminta endpoint tidak dihapus | 1x |
| R-06 | Pengajuan | Admin yang menolak pengajuan tidak tersimpan di database (persetujuan punya `disetujui_agen_oleh`) | 1x |
| R-07 | Pengajuan, Pencairan | Unggah bukti oleh admin dan bukti transfer keluar hanya dicek polanya, tidak dicek bahwa berkasnya ada dan milik brand yang benar (`handler.go:237`, `sprint5_handler.go:111`) | Ya (pencairan) |
| R-08 | Pengajuan | Keputusan tidak membawa id siklus atau bukti yang dilihat; modal yang sudah basi bisa menyetujui bukti baru yang belum dilihat | 1x |
| R-09 | Pengajuan | Penolakan setelah ada bukti transfer tidak mencatat nasib uangnya; pengajuan ulang membuat tagihan baru penuh | 1x |
| R-10 | Pengajuan | Tidak ada pesan berhasil setelah keputusan; setelah 409 daftar tidak dimuat ulang; tidak ada pemberitahuan ke pemohon (tercatat sebagai hal terbuka di spek) | 1x |
| R-11 | Riwayat Komisi | Tampilan holding tanpa kolom brand; kartu "Tersedia" adalah jumlah bruto, tidak dikurangi pencairan, tetapi labelnya sama dengan saldo tersedia di Detail Agen | 1x |
| R-12 | Riwayat Komisi | Layar dijaga `komisi.view`, API dijaga `agen.view`; `komisi.view` tidak ditegakkan backend di mana pun | 1x |
| R-13 | Komisi | Komisi menjadi tersedia pada hari keberangkatan (`<= CURRENT_DATE`), sedangkan spek dan glosarium menulis "setelah tanggal berangkat lewat" (`saldo.go:32`) | Ya |
| R-14 | Komisi | Galat apa pun saat mencatat komisi membatalkan konfirmasi pembayaran, void, atau perubahan diskon (satu transaksi, sesuai spek). Risiko operasional: lock wait dengan Ganti Kaitan menggagalkan pencatatan uang masuk | 1x |
| R-15 | Ganti Kaitan | `GantiKaitan` dan `LogKaitan` untuk jamaah tidak memeriksa brand admin (versi akun memeriksa); `SetujuiPencairan` dan `TolakPencairan` juga tidak. Aman selama izinnya hanya dipegang Super Admin | Ya |
| R-16 | Ganti Kaitan | Pemindahan komisi bisa bertabrakan dengan kunci unik ledger bila agen tujuan sudah punya baris untuk pax yang sama; hasilnya 500 generik. Jumlah baris yang dihapus tercatat di log, nominalnya tidak (`kaitan_pindah.go:47-63`) | Ya (kode), skenario tidak diuji |

Cakupan tes yang belum ada: lunas ulang setelah penerima berubah (T-01), K13 dengan pemilik kunci berbeda dari upline dan dengan jamaah tanpa NIK, booking CRM multi-pax, agen nonaktif dengan tagihan (S-03), NIK setelah penolakan (S-02), daftar komisi dengan filter dan isolasi brand, serta tes frontend untuk semua layar.

## Yang sudah berfungsi benar

- Keputusan pengajuan: akun lalu siklus tagihan terakhir dikunci dalam satu transaksi; keputusan ganda dijawab 409; transisi terlarang (setujui setelah tolak, tolak setelah aktif, bayar menyusul untuk Rp0) ditolak dan ada tesnya.
- Isolasi brand pada Daftar dan Detail Agen, keputusan, verifikasi, unggah bukti, daftar komisi, dan override akun: brand diambil dari token, dan id milik brand lain dijawab 404.
- Pencairan: baris agen dikunci saat mengajukan dan saat memutuskan; satu pengajuan menunggu per agen; minimal Rp500.000; saldo dicek ulang saat disetujui; pengajuan menunggu dan disetujui sama-sama mengurangi saldo tersedia.
- Override akun: agen tujuan harus aktif dan satu brand, tidak boleh diri sendiri, dan rantai pengikat ditelusuri dengan kunci baris untuk mencegah siklus. Alasan wajib dan tercatat di log.
- Override jamaah: komisi yang sudah tercatat harus dipindahkan secara eksplisit, sesuai spek 4.4, dan ditolak bila membuat saldo agen lama negatif.
- Aturan komisi sesuai spek untuk: pemicu saat lunas, snapshot nominal di keempat jalur pembuatan booking, pax batal dan bayi dilewati, penerima harus aktif, satu tingkat bonus pembinaan, cashback dan repeat order mati di bawah flag.
- Kueri daftar komisi dan peringkat tidak menggandakan baris. Zona waktu sesi database diset ke zona bisnis.
- Data pribadi: NIK agen hanya dikirim 4 digit terakhir; Daftar Agen tidak memuat NIK, email, atau rekening.
- Kode referral: acak kriptografis, unik per brand.

## Batas verifikasi

- Selain T-01, tidak ada temuan yang dijalankan terhadap API atau database.
- Portal agen dan pemohon di repo `azhan-booking` tidak diperiksa, termasuk apakah `catatan_admin` ditampilkan.
- `validateScheduleInput` tidak dibaca baris demi baris (dasar S-08).
- Dua bagian (Daftar dan Detail Agen; pencairan, ganti kaitan, peringkat) diaudit dengan pembacaan langsung, dua lainnya oleh subagent yang hasilnya kemudian dicek ulang pada temuan bertanda "Ya".

## Keputusan pemilik proyek (10 November 2026)

- Komisi mengikuti lunas terakhir: booking yang turun dari `lunas` ke `dp` atau `baru` membatalkan komisinya, dan lunas berikutnya menghitung ulang dengan penerima saat itu. Pembatalan booking atau pax setelah lunas tidak membatalkan komisi. Baris komisi yang batal ditandai, tidak dihapus.
- Komisi tersedia 3 hari kalender setelah tanggal berangkat.
- Komisi yang terlanjur dicairkan lalu batal, atau jadwal yang diundur, membuat saldo agen negatif dan tertutup komisi berikutnya. Koreksi tidak diblokir, hanya diberi peringatan.
- Data lama: komisi booking berstatus `dp` atau `baru` ditandai batal lewat migrasi.
- K13 tetap (booking pribadi agen mengalir ke upline); kalimat skenario 5.2 nomor 6 di spek diselaraskan.
- Booking dari dashboard menolak pax reguler yang jamaahnya belum punya NIK.
- Ditunda: booking CRM dengan beberapa pax (S-11), biaya pendaftaran agen di laporan keuangan (S-07), dan penyesuaian tampilan portal agen di repo `azhan-booking`.

Pekerjaan dibagi tiga PR: (1) menu dan layar Pengajuan Agen, (2) aturan komisi, (3) NIK wajib di booking admin.

## Status perbaikan

PR 1 (menu dan layar Pengajuan Agen):

| ID | Status | Perubahan |
|---|---|---|
| T-02 | Diperbaiki | Item sidebar "Pencairan Komisi" (dengan angka pengajuan yang menunggu) dan "Peringkat Agen" di grup AGEN. Penjaga rute `/komisi`, `/komisi/ganti-kaitan`, dan `/analytics/peringkat-agen` kini memakai izin backend (`agen.payout.approve`, `agen.kaitan.edit`, `agen.view`), bukan `holdingOnly`. Tombol "Ganti upline" di Detail Akun mengikuti izin `agen.kaitan.edit`. Filter brand di Peringkat Agen hanya untuk pengguna holding |
| S-01 | Diperbaiki | "Minta unggah ulang" ditawarkan juga untuk bukti susulan agen yang sudah disetujui |
| S-02 | Diperbaiki | Penolakan mengosongkan `nik_agen`; pelepasannya dicatat di `akun_data_log`. Tes: `TestTolakMelepasNIK` |
| S-03 | Diperbaiki | Keputusan atas tagihan berlaku untuk agen `aktif` maupun `nonaktif`; status agen tidak berubah oleh keputusan tagihan. Tes: `TestTagihanAgenNonaktifDanUnggahUlangSusulan` |
| S-04 | Diperbaiki | Kartu "Bukti Perlu Diperiksa" hanya menghitung tagihan yang sudah punya bukti; kartu pertama diberi label "Total Antrean" |
| S-05, R-06 | Diperbaiki | Tabel baru `keputusan_agen_log`: tiap keputusan tersimpan beserta catatan dan pelakunya, termasuk penolakan, dan tampil sebagai "Riwayat keputusan" di modal tinjau. Catatan dikosongkan saat admin berganti aksi dan hanya dikirim untuk aksi yang memintanya. Tes: `TestRiwayatKeputusanTidakTertimpa` |
| S-06 | Diperbaiki | `catatan_admin` tidak lagi dikirim ke portal pemohon. Pesan untuk pemohon tetap lewat `catatan_penolakan` dan `alasan_ditolak_agen`. Label isian di modal membedakan catatan internal dari pesan untuk pemohon. Tes: `TestCatatanAdminTidakKePortal` |
| R-02 | Diperbaiki | Nonaktifkan dan aktifkan agen wajib beralasan (5 sampai 500 karakter) dan tercatat di tabel baru `agen_status_log`; riwayatnya tampil di Detail Agen. Tes: `TestUbahStatusAgenBeralasanDanTercatat` |
| R-03 | Diperbaiki | Status tagihan `dibebaskan` punya label di riwayat siklus Detail Agen |

Migrasi `125_riwayat_keputusan_dan_status_agen.sql` (dua tabel riwayat) harus dijalankan sebelum kode PR 1 dirilis; aman diulang.

Verifikasi PR 1: `go vet` bersih; tes integrasi `internal/agen`, `internal/komisi`, dan `internal/customer` lulus; build dashboard lulus. Dicek di browser sebagai Super Admin: lima item di grup AGEN, halaman Pencairan Komisi dan Peringkat Agen terbuka dengan item menunya aktif; kartu ringkasan Pengajuan Agen 4 / 4 / 1; label isian berganti sesuai aksi dan teksnya kosong setelah berganti aksi; modal ubah status agen menolak alasan kosong dan alasan 3 karakter. Belum diuji di browser: mengirim keputusan dan perubahan status (hanya lewat tes integrasi, supaya data dev tidak berubah), angka di menu Pencairan Komisi saat lebih dari 0, dan tampilan untuk peran selain Super Admin.

PR 2 (aturan komisi):

| ID | Status | Perubahan |
|---|---|---|
| T-01 | Diperbaiki | Komisi mengikuti lunas terakhir (spek K25). Booking yang turun dari lunas membatalkan komisi aktifnya (`dibatalkan_at`, `sebab_batal`); lunas lagi mencatat baris baru. Kunci unik ledger diganti menjadi satu baris aktif per pax per jenis, sehingga komisi ganda tidak bisa tersimpan. Modal pembatalan pembayaran menjelaskan akibatnya. Tes: `TestKomisiBatalSaatTurunDariLunasLaluDihitungUlang`, `TestBookingBatalSetelahLunasKomisiTetap` |
| S-08 | Diputuskan, diberi peringatan | Saldo tersedia boleh negatif (spek K27); tidak ada pemblokiran. Form jadwal memperingatkan saat tanggal berangkat diubah pada jadwal yang punya komisi aktif. Tes: `TestSaldoHPlus3DanNegatif` |
| S-12 | Diperbaiki | Riwayat Komisi dipaginasi di server (20 baris per halaman); pencarian dan kartu ringkasan dihitung server atas seluruh hasil filter. Filter status baru: komisi aktif, dibatalkan, semua. Tes: `TestDaftarKomisiPaginasiRingkasanDanBatal` |
| S-13 | Diperbaiki | Daftar pencairan mengurutkan pengajuan pending lebih dulu (terlama di atas), lalu sisanya dari yang terbaru, sehingga batas 500 baris tidak memotong pengajuan yang menunggu |
| R-13 | Diperbaiki | Komisi tersedia 3 hari kalender setelah tanggal berangkat (spek K26), satu konstanta di backend. Riwayat Komisi menampilkan tanggal "tersedia mulai" per baris. Spek dan glosarium disamakan |
| R-16 | Sebagian | Kunci unik ledger tidak lagi memuat penerima, jadi pemindahan komisi tidak bertabrakan. Pemindahan hanya menyentuh komisi aktif. Nominal yang dipindahkan belum dicatat di log |

Migrasi `126_komisi_batal_dan_lunas_terakhir.sql` harus dijalankan sebelum kode PR 2 dirilis; aman diulang. Migrasi menandai batal komisi lama dari booking yang kini berstatus `dp` atau `baru`, dan komisi ganda lama (baris terbaru dipertahankan).

Perubahan bentuk API: `GET /api/admin/agen/komisi` kini mengembalikan `{ items, ringkasan }`, bukan larik. Daftar komisi portal agen tetap larik dan tidak memuat baris batal. Detail jadwal mendapat kolom `komisi_aktif`.

Verifikasi PR 2: `go vet` bersih; tes integrasi `internal/agen`, `internal/komisi`, `internal/payment`, `internal/booking`, `internal/schedule`, dan `internal/customer` lulus; build dashboard lulus. Dicek di browser sebagai Super Admin: Riwayat Komisi memuat 4 baris dengan kartu ringkasan dari server, tanggal "tersedia mulai" 3 hari setelah berangkat, filter status "Dibatalkan" (kosong di data dev), dan pencarian "Zakariyya" menyaring menjadi 2 baris dengan ringkasan ikut berubah. Belum diuji di browser: pembatalan pembayaran sampai komisi batal dan peringatan di form jadwal (logikanya hanya lewat tes integrasi, supaya data dev tidak berubah), serta paginasi dengan lebih dari 20 baris.

Belum dikerjakan (di luar PR 2): portal agen di repo `azhan-booking` belum menampilkan aturan H+3 dan saldo negatif; S-07, S-11, dan temuan rendah lain.

## Urutan perbaikan yang disarankan

1. T-02: kembalikan item sidebar Pencairan Komisi dan Peringkat Agen. Perubahan kecil, dan menutup risiko pengajuan pencairan tidak terlihat.
2. T-01: putuskan aturan lunas ulang, lalu perbaiki bersama tesnya. Sebelum itu, hindari mengganti upline agen yang punya booking belum final.
3. S-01, S-03, S-04: perbaikan kecil di layar Pengajuan Agen.
4. S-02: putuskan nasib `nik_agen` saat pengajuan ditolak.
5. S-09, S-10, S-11: butuh keputusan aturan komisi (K13 terhadap kunci jamaah, NIK wajib di booking admin, pax CRM).
6. S-12, S-13: paginasi di server untuk Riwayat Komisi dan antrean pencairan.
