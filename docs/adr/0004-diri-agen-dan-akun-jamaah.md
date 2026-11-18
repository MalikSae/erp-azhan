# 0004. Penentu diri agen di komisi, akun jamaah, dan kunci NIK agen

Tanggal: 18 November 2026
Status: diterima (tahap E3). Mengikuti ADR 0002 (kunci satu jalur) dan 0003 (aktivasi akun jamaah). Tanpa migrasi.

## Konteks

Komisi menghitung booking pribadi agen ("diri agen") dari dua penanda: `booking_pax.saya_sendiri_akun_id` atau NIK jamaah sama dengan `customer_accounts.nik_agen`. `nik_agen` tidak diverifikasi dan ajukan agen tidak memeriksa kepemilikan NIK, jadi akun yang mengetik NIK jamaah orang lain bisa membelokkan komisi jamaah itu ke upline-nya. Setelah aktivasi (ADR 0003) akun punya penentu yang terverifikasi: tautan `customer_accounts.jamaah_id`.

Skenario acuan: Ibrahim (agen aktif) memesankan Ahmad. Ahmad aktivasi, mengajukan agen, lalu memesan untuk dirinya dan istri.

## Keputusan

1. **Diri agen = `saya_sendiri_akun_id` atau jamaah tertaut ke akun.** NIK sama dengan `nik_agen` tidak lagi menentukan, di komisi (`komisi.loadPax`). `selfbooking.jamaahDiriAgen` tetap membaca NIK hanya untuk memilih pemilik kunci (upline, bukan agen); jamaah itu terikat ke upline sehingga penerima komisinya sama.
2. **Penerima komisi langsung untuk pax diri:**
   - jamaah `terikat_agen` dengan pemilik kunci bukan akun itu sendiri: pemilik kunci (`jamaah.direkrut_oleh_akun_id`). Pendaftaran Ahmad oleh Ibrahim tetap ke Ibrahim, juga setelah Ahmad menjadi agen atau upline akunnya diganti admin;
   - selain itu, akun agen aktif atau nonaktif: `upline_akun_id` akun. `belum_ditentukan` ikut cabang ini (perilaku sebelumnya, ditandai di tes);
   - `tanpa_agen`: tanpa komisi.

   Bonus pembinaan ke upline penerima, satu tingkat. Istri Ahmad (terikat ke Ahmad): komisi ke Ahmad, pembinaan ke Ibrahim.
3. **Upline nonaktif satu semantik dengan `UplineAkun`.** Pemilik kunci atau upline nonaktif tetap mengikat jamaah. Komisi langsung ke penerima nonaktif hangus; bonus pembinaan tetap ke upline aktif di atasnya. Tanpa backfill saat agen diaktifkan lagi: komisi dihitung sekali saat lunas.
4. **Akun jamaah untuk biaya agen dipersempit.** `agen.akunJamaah` benar bila akun punya booking sendiri dp/lunas (definisi lama), atau bila jamaah tertaut `terikat_agen` dengan pemilik kunci terisi, upline akun (`direkrut_oleh_akun_id`, dan `upline_akun_id` bila sudah terisi) sama dengan pemilik kunci, dan ada booking dp/lunas yang memuat jamaah itu (pax aktif, bukan bayi). Akun itu bebas biaya dan aktif otomatis. Jamaah `tanpa_agen` dan `belum_ditentukan` tetap lewat persetujuan admin. Dihitung ulang tiap ajukan.
5. **`nik_agen` dikunci di ajukan.** Akun tertaut: `nik_agen` = NIK jamaah tertaut, isian pengguna diabaikan (NIK boleh kosong di permintaan). Akun tidak tertaut: NIK yang sudah tercatat sebagai NIK jamaah di brand yang sama ditolak, kecuali jamaah buatan akun itu sendiri (`dibuat_oleh_akun_id`) yang tidak tertaut ke akun lain, jadi pemesan untuk dirinya tetap lolos dan lewat definisi lama akun jamaah. Percobaan NIK bentrok dibatasi 5 per akun dan 20 per IP per hari (429), karena galat itu membedakan NIK jamaah dari NIK lain. NIK jamaah tertaut yang sudah dipegang akun lain ditolak. Galat tunggal: "NIK tidak dapat dipakai. Hubungi admin." Admin menyelesaikan lewat **Kosongkan NIK Agen** atau Lepas Tautan. Akun tertaut selalu lahir dari aktivasi (akun baru, upline = pemilik kunci); tidak ada jalur menautkan akun yang sudah ada (ADR 0003).
6. **Satu komisi langsung dan satu bonus pembinaan per jamaah per keberangkatan.** Booking kedua jamaah yang sama di jadwal sama ditolak `ErrDuplicate` selama booking pertama aktif. Bila booking pertama batal setelah lunas (komisi tidak ditarik, K25), `komisi.ProcessBookingLunas` melewati jamaah itu bila ada komisi aktif jenis sama di booking lain pada jadwal yang sama. Komisi yang dibatalkan (turun dari lunas) tidak menghalangi. Keberangkatan berbeda dibayar lagi ke pemilik kunci. Repeat order dan cashback tetap mati (`KomisiRepeatCashbackAktif`).

## Konsekuensi

- Booking admin atau CRM untuk pax ber-NIK agen yang tidak tertaut dan tidak bertanda `saya_sendiri` tidak lagi dihitung diri agen: komisinya mengikuti pemilik kunci jamaah. Admin menyelesaikan lewat tautan akun atau Ganti Upline. Tanpa backfill; belum rilis production.
- Pelamar agen yang NIK-nya sudah tercatat sebagai jamaah milik orang lain, atau jamaah buatannya yang tertaut ke akun lain, ditolak "Hubungi admin"; jalannya menautkan akun lewat kode booking dan NIK, lalu mengajukan. Pemesan untuk dirinya sendiri (jamaah buatan akunnya, tidak tertaut ke akun lain) tidak terhalang.
- Akun yang menjadi agen otomatis tidak dicabut bila booking pendukung kemudian batal; admin memantau lewat daftar agen dari tautan (ADR 0003). Komisi yang sudah tercatat tidak ditarik.
- Ganti Upline akun tidak memindahkan jamaah tertaut yang booking pertamanya dibuat agen lain; komisi pendaftaran Ahmad tetap ke Ibrahim.
