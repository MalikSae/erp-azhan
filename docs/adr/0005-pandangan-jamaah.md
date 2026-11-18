# 0005. Peran jamaah: booking orang lain lewat endpoint booking biasa

Tanggal: 18 November 2026
Status: diterima, direvisi 18 November 2026 setelah melihat tampilan M2/W2. Mengikuti ADR 0003 (aktivasi) dan 0004. Tanpa migrasi. Menggantikan ADR 0006.

## Konteks

Akun yang lahir dari aktivasi tertaut ke jamaah (`customer_accounts.jamaah_id`). Perjalanannya dibuat orang lain (Ibrahim), jadi `owned()` yang terkunci ke `bookings.customer_account_id` tidak memberi apa pun.

Versi pertama memberi pandangan baca terbatas lewat rute khusus (`/perjalanan-jamaah`, tanpa kode booking, harga, atau pax lain). Setelah tampilannya dilihat, versi itu dinilai tidak sesuai motif jamaah: memastikan dirinya terdaftar dan pembayarannya masuk, bersiap berangkat, mengambil surat, dan mengurus dokumennya sendiri. Rute khusus dibuang.

## Keputusan

1. **Endpoint booking biasa.** `orderSelect` (dipakai `GET /account/bookings` dan `owned()`) juga memuat booking yang berisi pax aktif jamaah tertaut ke akun (akun belum dihapus, brand sama dengan akun). Draft tidak tampil; booking batal tetap tampil sebagai riwayat, juga di Data Jamaah, sama dengan pemesan. `orderSummary.peran` = `pemesan` atau `jamaah`.
2. **Tampil utuh.** Kode booking, semua nama jamaah, harga per pax, total, terbayar, sisa, diskon, riwayat pembayaran, rekening, dan kantor sama dengan yang dilihat pemesan.
3. **Pembuat booking.** Untuk peran jamaah, `dibuat_oleh` = `{nama, kontak, whatsapp}`. Kontak `agen` (WhatsApp akun pembuat) bila pembuatnya agen aktif; selain itu `admin` (WhatsApp brand). Booking tanpa akun pembuat memakai nama brand.
4. **Aksi.**
   - Bayar (`POST /bookings/{id}/payments`): boleh, aturan sama dengan pemesan.
   - Batal: tidak boleh (`can_cancel` selalu false; `cancelGuard` tetap memeriksa pemilik).
   - Per jamaah (unggah dokumen, simpan data jamaah, unduh surat, ajukan dan terima perlengkapan, RSVP manasik): hanya untuk jamaah tertaut (`bolehJamaah`), selain itu 403 `Hanya untuk data Anda sendiri.` Daftar surat hanya memuat surat jamaah tertaut.
   - Bacaan dokumen dan data jamaah pax lain memakai akses terbatas yang sudah ada (nama, NIK tersamar, status).
5. **Akses penuh atas diri sendiri.** `sqlAksesPenuh` benar untuk akun pembuat jamaah atau akun yang tertaut ke jamaah itu. Data Jamaah menampilkan baris dirinya dengan kode booking dan boleh diubah lewat aturan pengajuan yang berlaku.
6. **Klaim pembuat:** `GugurkanKlaimPembuatBooking` tidak menggugurkan klaim bila pemesan adalah akun yang `jamaah_id`-nya jamaah itu.
7. **Notifikasi dokumen:** akun tertaut menjadi penerima tambahan (`notifikasi.Dokumen`, cabang UNION).
8. **Lepas tautan** (admin mengosongkan `jamaah_id`) langsung menutup semua akses di atas.

## Konsekuensi

- M2 dan W2 memakai daftar dan detail booking yang sama dengan pemesan, ditambah keterangan "Booking dibuat oleh" dan tombol WhatsApp kontak.
- Faktor kedua untuk unduh surat dan berkas tidak lagi ditunggu: aktivasi (kode booking + NIK) dianggap cukup untuk data atas nama sendiri.
