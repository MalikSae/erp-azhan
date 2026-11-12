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

**Agen**: akun yang punya kode referral dan berstatus aktif. Agen tidak harus pernah berangkat umroh. Program keagenan disebut **Syiar**.
_Hindari_: memakai "jamaah" untuk menyebut agen; "agen lama" (tidak ada agen lama, sistem belum deploy).

**Calon agen**: akun yang pengajuannya sedang diproses (status agen `pengajuan`).

**Upline**: agen yang mengajak sebuah akun, lewat tautan atau kode referralnya saat akun dibuat, atau agen yang dipasangkan admin. Berlaku untuk semua akun. Bila akun itu menjadi agen, upline-nya menerima bonus pembinaan darinya. Satu tingkat saja. Override admin memindahkan upline dan penerima bonus bersama-sama; bonus yang sudah tercatat tidak ditarik kembali. Disimpan di dua kolom `customer_accounts`: `direkrut_oleh_akun_id` (sejak akun dibuat) dan `upline_akun_id` (disalin saat pengajuan agen disetujui). Keduanya berarti upline.
_Hindari_: agen pengikat, perekrut, pengajak sebagai istilah tersendiri.

**PIC**: pemesan yang ikut berangkat dalam satu booking, dicocokkan lewat NIK dan nama seperti jamaah lain.

## Pemesanan

**Booking**: pemesanan satu jadwal untuk satu atau beberapa pax. Ada dua jalur: didaftarkan agen di portal agen, atau mandiri di web publik lewat link ref.

**Pax**: satu kursi dalam booking (`booking_pax`). Jenis: reguler atau bayi (infant).

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

**Kunci (kaitan)**: ikatan jamaah ke agen yang pertama membawanya. Seumur hidup. Hanya override admin yang memindahkan. Pemilik kunci menerima komisi atas jamaah itu siapa pun yang memesan. Pengecualian (keputusan 12 November 2026): saat admin mengganti upline sebuah akun, jamaah yang booking pertamanya dibuat akun itu dan masih terikat ke upline lama ikut pindah ke upline baru (atau jadi tanpa agen bila upline dilepas). Komisi yang sudah tercatat tetap milik agen lama; kaitan yang dipilih admin per jamaah (jalur3) tidak ikut.

**Override admin**: pemindahan kunci jamaah atau pemasangan upline akun oleh Admin Master (izin `agen.kaitan.edit`), lewat modal "Ganti agen" di detail Jamaah dan "Ganti upline" di detail Akun. Halaman terpisah `/komisi/ganti-kaitan` (layar C4) sudah dihapus.

**Saya sendiri**: pilihan di form booking portal agen untuk jamaah pertama. Menandai booking pribadi agen; agen tidak mendapat komisi dari booking dirinya sendiri, komisinya mengalir ke upline agen.

**Komisi langsung**: nominal flat per pax berangkat untuk agen pemilik kunci jamaah.

**Bonus pembinaan**: nominal flat per pax berangkat untuk upline agen pemilik kunci.

**Komisi bawaan brand**: nominal komisi langsung dan bonus pembinaan yang diatur per brand. Berlaku untuk paket yang nominalnya dikosongkan. Bawaan kosong berarti brand belum ikut program agen.

**Nominal efektif komisi**: nominal yang benar-benar dipakai sebuah paket: nilai paket bila diisi (0 berarti sengaja tanpa komisi), selain itu komisi bawaan brand.

**Snapshot komisi**: salinan nominal efektif komisi ke booking saat booking dibuat. Perubahan paket atau bawaan brand tidak memengaruhi booking lama.

**Saldo tersedia / tertahan**: komisi aktif yang sudah / belum lewat 3 hari kalender dari tanggal berangkat jadwalnya. Saldo tersedia dikurangi pencairan dan boleh negatif.

**Ujroh**: istilah untuk komisi agen di sisi pengguna.

**Keputusan admin**: satu aksi di kartu pengajuan (B1): setujui, bebaskan biaya, minta unggah ulang, tolak, atau setujui dengan bayar menyusul.

## Dinonaktifkan di tahap awal

**Cashback** dan **komisi repeat order** (pemisahan 50/50). Booking ulang dihitung seperti booking biasa. Kodenya tetap ada, dimatikan lewat flag.
