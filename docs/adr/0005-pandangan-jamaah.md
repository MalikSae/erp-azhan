# 0005. Pandangan jamaah (peran jamaah, terbatas)

Tanggal: 18 November 2026
Status: diterima (tahap E4a). Mengikuti ADR 0003 (aktivasi) dan 0004. Tanpa migrasi.

## Konteks

Akun yang lahir dari aktivasi tertaut ke jamaah (`customer_accounts.jamaah_id`). Perjalanannya dibuat orang lain (Ibrahim), jadi `owned()` yang terkunci ke `bookings.customer_account_id` tidak memberi apa pun dan Ahmad mendarat di layar kosong.

## Keputusan

1. **Peran jamaah = baca terbatas, daftar putih.** Rute baru `GET /api/account/perjalanan-jamaah`. `owned()`, `orderSelect`, `sqlAksesPenuh`, dan `dibuat_oleh_akun_id` tidak diubah. Cancel, CreatePayment, Detail, Dokumen, dan semua handler `/bookings/{id}/...` tetap pemilik-saja: akun tertaut mendapat 404.
2. **Cakupan baris:** booking akun lain (`NOT customer_account_id <=> akun`, termasuk booking admin tanpa akun) yang memuat pax aktif jamaah tertaut; bukan draft, bukan batal, dan bukan `baru` yang tahan seat-nya lewat. Pembatalan booking pemilik menghilangkan kartu. Booking milik akun sendiri tetap di `/api/account/bookings`.
3. **Kontrak JSON** (satu-satunya bentuk yang boleh dikirim):

```json
{"perjalanan": [{
  "booking_id": 123,
  "paket": "Umroh Reguler 9 Hari",
  "berangkat": "2027-03-01",
  "status": "baru | dp | lunas",
  "pax": [{"jamaah_id": 45, "nama": "Ahmad", "pax_type": "reguler"}],
  "can_pay": false,
  "can_cancel": false
}]}
```

   `status`: `lunas` mencakup `dokumen_lengkap` dan `siap_berangkat`. `pax` hanya baris milik jamaah tertaut. Tidak ada: kode booking, pax lain, harga, total, DP, sisa, diskon, pembayaran, rekening, kantor. Akun tidak tertaut menerima `{"perjalanan": []}`.
4. **Data Jamaah:** `muatAnggota` menambah baris dari booking tersebut (hanya pax miliknya), ditandai `DariPeran`. Baris booking sendiri didahulukan; satu jamaah satu baris. Akses terbatas (lihat): tanpa ubah data, `kode` booking pada `bookingRingkas` dikosongkan untuk baris peran.
5. **Klaim pembuat:** `GugurkanKlaimPembuatBooking` tidak menggugurkan klaim bila pemesan adalah akun yang `jamaah_id`-nya jamaah itu.
6. **Dokumen, surat, notifikasi dokumen:** dibuka sebagian, lihat ADR 0006 (unduh surat, berkas, dan unggah tetap ditunda). **Ditunda:** manasik, Jelajah Haramain, pengumuman, pengingat. Akses penuh menunggu keputusan faktor kedua.

## Konsekuensi

- M2 dan W2 memakai rute ini untuk kartu jamaah tanpa kode booking dan harga. Ketaatan allowlist dijaga tes yang memeriksa isi body.
- Detail booking per kartu belum ada; `booking_id` hanya pengenal baris, bukan akses.

## Catatan tambahan

- Booking tanpa akun pemesan (`customer_account_id` NULL, dibuat admin atau agen) ikut tampil bila memuat jamaah tertaut. Disengaja: jamaah melihat perjalanannya sendiri.
- `/api/account/me` (`perjalanan_tertaut`) hanya ringkasan profil. Klien M2/W2 memakai `GET /api/account/perjalanan-jamaah` untuk daftar perjalanan peran jamaah.
- Status baris peran di `/account/jamaah` memakai kontrak sama (`baru|dp|lunas`); `kode` kosong berarti baris peran.
- Orang yang sama sebagai Penuh (dibuat akun itu) dan sebagai peran terbatas tetap dua baris, karena hak aksesnya berbeda.
