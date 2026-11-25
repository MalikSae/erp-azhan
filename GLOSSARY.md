# Glosarium Azhan ERP

Istilah domain yang dipakai di kode, dokumen, dan percakapan proyek. Satu istilah satu arti. Rujukan lengkap alur keagenan: `docs/spek/SPEK-KEAGENAN-AKUN-2026-11-03.md`.

## Orang

**Akun**: orang yang punya akun (email + password) dan bisa booking atau memesan. Tabel `customer_accounts`. Disebut juga pemesan.
_Hindari_: user, pelanggan (terlalu umum), jamaah (itu orang lain).

**Jamaah**: orang yang berangkat umroh. Tabel `jamaah`. Bukan "penumpang". Data identitas milik jamaah, bukan milik akun. Banyak akun boleh memasukkan jamaah yang sama; jamaah yang sama dikenali dari NIK. Jamaah belum tentu punya akun.
_Hindari_: peserta, penumpang, pax untuk orang (pax adalah kursi dalam booking).

**Akun pembuat**: akun yang pertama memasukkan seorang jamaah lewat bookingnya. Kolom `jamaah.dibuat_oleh_akun_id`. Hanya akun pembuat yang punya **akses penuh** atas data jamaah itu di portal. Jamaah yang dibuat admin tidak punya akun pembuat. Klaim pembuat gugur bila jamaah dipakai booking pihak lain sebelum akun pembuat punya pembayaran terkonfirmasi.
_Hindari_: pemilik jamaah (data identitas tetap milik jamaah); menyamakan dengan agen perekrut (`direkrut_oleh_akun_id`, itu urusan komisi).

**Akses terbatas**: yang didapat akun atas jamaah yang bukan buatannya: nama, NIK tersamar, jenis pax, dan angka kelengkapan. Tidak bisa melihat atau mengubah isian, tidak bisa mengunduh surat, dan hanya bisa mengunggah jenis dokumen yang belum ada berkasnya.

**Aktivasi Akun Jamaah**: jamaah yang didaftarkan orang lain membuat akunnya sendiri dengan kode booking dan NIK, lewat layar Daftar. Selalu akun baru: akun yang sudah ada tidak bisa ditautkan ke jamaah, dan akun terdaftar tanpa upline tetap tanpa upline sampai pusat mengubahnya (Ganti Upline). Satu galat generik untuk semua sebab gagal. Upline akun baru adalah pemilik kunci jamaah. Rute `/api/public/auth/aktivasi`, dinyalakan flag `AKTIVASI_JAMAAH_AKTIF`. Keputusan: `docs/adr/0003-aktivasi-akun-jamaah.md`. Tanpa kode agen atau kode referral: akun dari jamaah tanpa agen lahir tanpa upline, akun dari jamaah terikat agen ber-upline pemilik kunci (keputusan 24 November 2026).
_Hindari_: klaim akun, verifikasi jamaah, OTP (tidak dipakai).

**Tautan Jamaah**: ikatan satu akun ke satu jamaah, kolom `customer_accounts.jamaah_id` (unik). Dibuat oleh aktivasi; hanya admin yang melepas (**Lepas Tautan Jamaah**, ditolak untuk akun agen aktif atau nonaktif). Memberi akses terbatas (lihat): label paket, tanggal berangkat, data jamaah, dokumen. Tidak mengubah kunci jamaah atau upline.
_Hindari_: memakai "akun pembuat" untuk tautan; akun pembuat (`dibuat_oleh_akun_id`) dan tautan adalah dua hal berbeda.

**Peran Jamaah**: akun tertaut melihat booking orang lain yang memuat dirinya lewat endpoint booking biasa (`peran: "jamaah"`), utuh termasuk harga dan pembayaran, ditambah pembuat booking dan kontaknya. Boleh bayar; tidak boleh batal; aksi per jamaah hanya atas dirinya. Keputusan: `docs/adr/0005-pandangan-jamaah.md`.

**Agen**: akun yang punya kode referral dan berstatus aktif. Agen tidak harus pernah berangkat umroh. Program keagenan disebut **Syiar**.
_Hindari_: memakai "jamaah" untuk menyebut agen; "agen lama" (tidak ada agen lama, sistem belum deploy).

**Calon agen**: akun yang pengajuannya sedang diproses (status agen `pengajuan`).

**Upline**: agen yang mengajak sebuah akun, lewat tautan atau kode referralnya saat akun dibuat, atau agen yang dipasangkan admin. Berlaku untuk semua akun. Bila akun itu menjadi agen, upline-nya menerima bonus pembinaan darinya. Satu tingkat saja. Override admin memindahkan upline dan penerima bonus bersama-sama; bonus yang sudah tercatat tidak ditarik kembali. Disimpan di dua kolom `customer_accounts`: `direkrut_oleh_akun_id` (sejak akun dibuat) dan `upline_akun_id` (disalin saat pengajuan agen disetujui). Keduanya berarti upline.
_Hindari_: agen pengikat, perekrut, pengajak sebagai istilah tersendiri.

**PIC**: pemesan yang ikut berangkat dalam satu booking, dicocokkan lewat NIK dan nama seperti jamaah lain.

## Pengguna dashboard

**Pengguna**: orang yang login ke dashboard ERP. Tabel `admin_users`. Bukan akun (itu pelanggan) dan bukan agen.
_Hindari_: admin (untuk semua pengguna), pegawai, user.

**Super Admin**: jenis akses pengguna yang memegang semua izin. Hanya untuk pengguna Holding. Sistem selalu punya minimal satu yang aktif.

**Tim**: jenis akses pengguna yang izinnya dicentang per pengguna di form Pengguna. Bukan role, bukan divisi.
_Hindari_: role, jabatan, divisi, struktur organisasi (semuanya dihapus 25 November 2026).

**Izin**: kode di tabel `permissions` yang menjaga route (`RequirePermission`). Tim memegangnya lewat `user_permissions`. Izin baru tidak otomatis diberikan ke Tim. Spek: `docs/spek/SPEK-AKSES-PENGGUNA-2026-11-25.md`.

**Templat akses**: pengisi cepat centang izin di form Pengguna (Admin Brand, Keuangan, Konten). Tidak tersimpan pada pengguna dan tidak mengikat.

## Pemesanan

**Booking**: pemesanan satu jadwal untuk satu atau beberapa pax. Ada dua jalur: didaftarkan agen di portal agen, atau mandiri di web publik lewat link ref.

**Pax**: satu kursi dalam booking (`booking_pax`). Jenis: reguler atau bayi (infant).
**Pax DP+**: pax aktif di booking berstatus dp atau lunas. Satuan KPI marketing di Dashboard Utama (keputusan 12 November 2026): pax per channel dan grafik Pax DP+. Pax per channel membagi pax DP+ menjadi "dari agen" (jamaah terikat agen) dan "tanpa agen" (`tanpa_agen`, keadaan normal jamaah web tanpa referral). Seat terisi hanya menghitung pax DP+ reguler, karena infant tidak memakai seat.
**Tahapan umroh**: status pekerjaan travel per jamaah yang dilihat jamaah, agen, dan admin (Detail Booking), urut Perlengkapan, Paspor, SISKOPATUH, Visa, Tiket (internal/tahapan). Perlengkapan tidak berlaku untuk infant; paspor selesai saat dokumen disetujui. Vaksin, hotel, land arrangement, dan manasik bukan tahapan (keputusan 24 November 2026).
**Siap berangkat**: semua tahapan umroh setiap pax aktif selesai, termasuk tiket booking.

**Kuota seat**: jumlah seat yang dijual untuk satu jadwal (`schedules.seat_total`). Padanan "block seat" atau allotment yang travel pesan dari maskapai.

**Sisa seat**: kuota seat dikurangi seat yang sedang di-hold (`schedules.seat_sisa`).

**Hold seat**: seat sebuah booking sudah mengurangi sisa seat jadwal, sehingga tidak bisa dijual ke booking lain (`bookings.is_seat_blocked`). Yang memakai seat hanya pax reguler; bayi tidak.
_Hindari_: block seat, blokir seat, kunci seat (block seat di industri berarti kuota dari maskapai; "kunci" dipakai untuk kunci baris database).

**Batas waktu hold**: saat hold seat berakhir bila booking belum dibayar (`bookings.seat_hold_expires_at`). Booking dari web dan agen: 24 jam. Kosong berarti hold tetap.

**Seat terkonfirmasi**: hold seat pada booking yang sudah dibayar (status dp atau lunas). Tanpa batas waktu, dan tidak bisa dilepas lewat tombol; hanya lepas bila booking atau pax dibatalkan.

**Lepas seat**: seat dikembalikan ke sisa seat jadwal. Terjadi saat booking atau pax dibatalkan, saat batas waktu hold lewat, atau lewat tombol Lepas Seat pada booking yang belum dibayar.

**Lunas**: status booking yang memicu pencatatan komisi. Komisi mengikuti lunas terakhir: booking yang turun dari lunas membatalkan komisinya, dan komisi dihitung ulang saat booking lunas lagi.

**Komisi batal**: baris komisi yang ditandai batal karena bookingnya turun dari lunas. Tidak dihapus dan tidak dihitung di saldo, total, maupun peringkat.

**Jadwal**: keberangkatan paket pada tanggal tertentu (`schedules`). Sumber nominal komisi.

## Keagenan

**Kode referral**: kode unik agen, dibagikan sebagai `?ref=KODE`. Disimpan di cookie `azhan_ref` (90 hari, klik terakhir menimpa).

**Pengikatan akun**: pencatatan upline akun saat akun dibuat, dari cookie referral. Setelah akun ada, cookie diabaikan. Akun tanpa agen tetap tanpa agen kecuali admin memasangkannya.

**Kunci (kaitan)**: ikatan jamaah ke agen yang pertama membawanya. Seumur hidup. Hanya override admin yang memindahkan. Status `jamaah.kaitan_status`: `terikat_agen` atau `tanpa_agen` (`belum_ditentukan` digabung ke `tanpa_agen`, migrasi 158). Hanya jamaah baru yang terikat otomatis: `tanpa_agen`, kaitan belum pernah ditetapkan, dan belum ikut booking non-draft lain. Jamaah `tanpa_agen` lama hanya dikaitkan admin; pengaitan itu berlaku ke depan untuk komisi (booking yang sudah pernah lunas sebelumnya tetap tanpa komisi). (Keputusan 24 November 2026.) Pemilik kunci menerima komisi atas jamaah itu siapa pun yang memesan. Pengecualian (keputusan 12 November 2026): saat admin mengganti upline sebuah akun, jamaah yang booking pertamanya dibuat akun itu dan masih terikat ke upline lama ikut pindah ke upline baru (atau jadi tanpa agen bila upline dilepas). Komisi yang sudah tercatat tetap milik agen lama; kaitan yang dipilih admin per jamaah (jalur3) tidak ikut.

**Klaim agen**: agen yang membawa booking berisi jamaah `tanpa_agen` lama, yang tidak terikat otomatis (`bookings.agen_klaim_akun_id`): akun agen yang memesan di portal agen atau upline akun pemesan web. Booking admin tidak mencatat klaim, begitu pula pax diri agen. Tampil sebagai "Referral" di Detail Booking dengan tautan "Kaitkan ke agen"; admin yang memutuskan.

**Override admin**: pemindahan kunci jamaah atau pemasangan upline akun oleh Admin Master (izin `agen.kaitan.edit`), lewat modal "Ganti agen" di detail Jamaah dan "Ganti upline" di detail Akun. Halaman terpisah `/komisi/ganti-kaitan` (layar C4) sudah dihapus.

**Saya sendiri**: pemesan agen aktif yang ikut berangkat (`pemesan_tidak_ikut` salah), di jalur booking mana pun. Jamaah pertama = diri agen, NIK dari profil agen, ditandai `saya_sendiri_akun_id`. Agen tidak mendapat komisi dari booking dirinya sendiri, komisinya mengalir ke upline agen. Mode agen mengikuti status akun pemesan, bukan endpoint (ADR 0002).

**Diri agen**: pax yang ditandai `saya_sendiri_akun_id` atau jamaahnya tertaut ke akun agen. NIK sama dengan `nik_agen` bukan penentu lagi (ADR 0004). Komisi langsungnya ke pemilik kunci jamaah bila jamaah terikat ke agen lain, selain itu ke upline akun, termasuk bila jamaahnya `tanpa_agen`. Pax bukan diri yang `tanpa_agen` tanpa komisi. Pendaftaran Ahmad oleh Ibrahim tetap ke Ibrahim walau Ahmad menjadi agen.

**Akun jamaah**: akun yang bebas biaya agen dan aktif otomatis saat mengajukan: punya booking sendiri dp/lunas, atau tertaut ke jamaah `terikat_agen` yang upline akunnya sama dengan pemilik kunci dan memuat booking dp/lunas. Jamaah `tanpa_agen` lewat persetujuan admin. `nik_agen` akun tertaut = NIK jamaah tertaut.

**Komisi langsung**: nominal flat per pax berangkat untuk agen pemilik kunci jamaah.

**Bonus pembinaan**: nominal flat per pax berangkat untuk upline agen pemilik kunci.

**Komisi bawaan brand**: nominal komisi langsung dan bonus pembinaan yang diatur per brand. Berlaku untuk paket yang nominalnya dikosongkan. Bawaan kosong berarti brand belum ikut program agen.

**Nominal efektif komisi**: nominal yang benar-benar dipakai sebuah paket: nilai paket bila diisi (0 berarti sengaja tanpa komisi), selain itu komisi bawaan brand.

**Snapshot komisi**: salinan nominal efektif komisi ke booking saat booking dibuat. Perubahan paket atau bawaan brand tidak memengaruhi booking lama.

**Saldo tersedia / tertahan**: komisi aktif yang sudah / belum lewat 3 hari kalender dari tanggal berangkat jadwalnya. Saldo tersedia dikurangi pencairan dan boleh negatif.

**Ujroh**: istilah untuk komisi agen di sisi pengguna.

**Keputusan admin**: satu aksi di kartu pengajuan (B1): setujui, bebaskan biaya, minta unggah ulang, tolak, atau setujui dengan bayar menyusul.

## Tabungan Umroh

**Target Tabungan**: catatan pribadi calon jamaah tentang berapa yang ingin dikumpulkan untuk umroh, berikut bulan berangkat dan DP minimal. Satu per akun. Tabel `tabungan_target`. Hana tidak menyimpan uang; ini hanya catatan.
_Hindari_: saldo, rekening tabungan, tabungan di Hana, cicilan.

**Catatan Tabungan**: satu baris angka yang diisi akun saat menyisihkan uang, dengan tanggal. Tabel `tabungan_catatan`. Mengganti target tidak menghapusnya; menghapus target menghapus semuanya.
_Hindari_: setor, titip, deposit.

## Dinonaktifkan di tahap awal

**Cashback** dan **komisi repeat order** (pemisahan 50/50). Booking ulang dihitung seperti booking biasa. Kodenya tetap ada, dimatikan lewat flag.
