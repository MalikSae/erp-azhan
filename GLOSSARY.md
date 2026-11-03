# Glosarium Azhan ERP

Istilah domain yang dipakai di kode, dokumen, dan percakapan proyek. Satu istilah satu arti. Rujukan lengkap alur keagenan: `docs/spek/SPEK-KEAGENAN-AKUN-2026-11-03.md`.

## Orang

**Akun**: orang yang punya akun (email + password) dan bisa booking atau memesan. Tabel `customer_accounts`. Disebut juga pemesan.
_Hindari_: user, pelanggan (terlalu umum), jamaah (itu orang lain).

**Jamaah**: orang yang berangkat umroh. Tabel `jamaah`. Disebut juga penumpang. Data identitas milik jamaah, bukan milik akun. Banyak akun boleh memasukkan jamaah yang sama; jamaah yang sama dikenali dari NIK. Jamaah belum tentu punya akun.
_Hindari_: peserta, pax untuk orang (pax adalah kursi dalam booking).

**Agen**: akun yang punya kode referral dan berstatus aktif. Agen tidak harus pernah berangkat umroh. Program keagenan disebut **Syiar**.
_Hindari_: memakai "jamaah" untuk menyebut agen; "agen lama" (tidak ada agen lama, sistem belum deploy).

**Calon agen**: akun yang pengajuannya sedang diproses (status agen `pengajuan`).

**Upline**: agen yang mengikat akun seorang agen saat akun itu dibuat. Menerima bonus pembinaan. Satu tingkat saja.

**PIC**: pemesan yang ikut berangkat dalam satu booking, dicocokkan lewat NIK dan nama seperti penumpang lain.

## Pemesanan

**Booking**: pemesanan satu jadwal untuk satu atau beberapa pax. Ada dua jalur: didaftarkan agen di portal agen, atau mandiri di web publik lewat link ref.

**Pax**: satu kursi dalam booking (`booking_pax`). Jenis: reguler atau bayi (infant).

**Lunas**: status booking yang memicu pencatatan komisi.

**Jadwal**: keberangkatan paket pada tanggal tertentu (`schedules`). Sumber nominal komisi.

## Keagenan

**Kode referral**: kode unik agen, dibagikan sebagai `?ref=KODE`. Disimpan di cookie `azhan_ref` (90 hari, klik terakhir menimpa).

**Pengikatan akun**: pencatatan agen pemilik akun saat akun dibuat, dari cookie referral. Setelah akun ada, cookie diabaikan. Akun tanpa agen tetap tanpa agen kecuali admin memasangkannya.

**Kunci (kaitan)**: ikatan jamaah ke agen yang pertama membawanya. Seumur hidup. Hanya override admin yang memindahkan. Pemilik kunci menerima komisi atas jamaah itu siapa pun yang memesan.

**Override admin**: pemindahan kunci jamaah atau pemasangan agen akun oleh Admin Master (layar C4).

**Saya sendiri**: pilihan di form booking portal agen untuk penumpang pertama. Memicu komisi langsung atas dirinya.

**Komisi langsung**: nominal flat per pax berangkat untuk agen pemilik kunci jamaah.

**Bonus pembinaan**: nominal flat per pax berangkat untuk upline agen pemilik kunci.

**Snapshot komisi**: salinan nominal komisi jadwal ke booking saat booking dibuat. Perubahan jadwal tidak memengaruhi booking lama.

**Saldo tersedia / tertahan**: komisi yang jadwal keberangkatannya sudah / belum lewat, dikurangi pencairan.

**Ujroh**: istilah untuk komisi agen di sisi pengguna.

**Keputusan admin**: satu aksi di kartu pengajuan (B1): setujui, bebaskan biaya, minta unggah ulang, tolak, atau setujui dengan bayar menyusul.

## Dinonaktifkan di tahap awal

**Cashback** dan **komisi repeat order** (pemisahan 50/50). Booking ulang dihitung seperti booking biasa. Kodenya tetap ada, dimatikan lewat flag.
