# SPEK — Manifest Paket (unduh otomatis dari ERP)

> **Status:** Disepakati 25 Nov 2026. Fase 1 dibangun 25 Nov 2026 (migrasi 159, modul `internal/manifest`, tab Manifest di Detail Paket); Fase 2 dan 3 belum.
> **Catatan Fase 1:** PNR dan detail penerbangan disimpan di tabel sendiri `manifest_paket` (satu baris per paket), bukan kolom di `schedules`, supaya modul manifest tidak menulis tabel milik modul paket. Pemeriksaan data ditambah satu: "Tanggal paspor tidak sesuai" (habis tidak lebih lambat dari terbit). Batas usia CHILD Saudia sudah dicek: 2 sampai di bawah 12 tahun.
> **Sumber keputusan:** sesi grilling 25 Nov 2026 (4 babak), berdasarkan file manifest yang selama ini disusun admin secara manual di Excel (contoh: grup 14 Sep 2026, Saudia, 17 pax, 7 sheet).
> **Prinsip:** pekerjaan admin yang sebenarnya adalah mengambil keputusan (PNR, pembagian kamar, catatan seat, siapa ikut layanan tambahan). ERP menyimpan keputusan itu sekali, lalu semua sheet dibuat dari data yang sama.

---

## 1. Masalah

Admin menyalin data jamaah yang sama ke 7 sheet dengan susunan kolom berbeda untuk tiap penerima. Kesalahan yang ditemukan di file contoh:

- Format tanggal campur di kolom yang sama: serial Excel, `21-Dec-1983`, `23 Agst 2024`.
- Satu paspor tercatat dengan tanggal terbit sama dengan tanggal habis berlaku.
- Nama di Roomlist disingkat, tidak sama dengan nama paspor di sheet lain.
- Daftar Telefric kurang 2 pax dibanding Speed Train tanpa catatan.
- Relasi ditulis ulang dengan istilah berbeda di sheet Visa (SUAMI/ISTRI menjadi SINGLE/HUSBAND).

## 2. Isi file manifest

Semua sheet kecuali Roomlist memakai header yang sama:

| Header | Sumber di ERP |
|---|---|
| Judul `MANIFEST GROUP <tanggal berangkat> <nama paket>` | `schedules.berangkat_tanggal`, `schedules.jadwal_nama` (huruf besar) |
| Flight Carrier | maskapai paket |
| PNR Code | isian baru, per paket |
| Date of Departure | `schedules.berangkat_tanggal` |
| Flight Detail | isian baru, teks bebas beberapa baris (ditempel dari maskapai) |
| Pax Number | jumlah pax yang masuk manifest |
| Package (`09 DAYS`) | selisih tanggal berangkat sampai pulang, kedua hari dihitung |

| Sheet | Penerima | Kolom | Fase |
|---|---|---|---|
| Blok Seat | Maskapai | No, nama paspor, L/P, tempat & tanggal lahir, no/terbit/habis/kantor paspor, relation, keterangan seat | 1 (relation & keterangan kosong), 2 (keterangan terisi) |
| Manifest | Maskapai | sama dengan Blok Seat, kolom terakhir jenis tiket | 1 |
| Visa | Provider visa | No, no paspor, nama, gender, mahram, relation, tanggal & tempat lahir, kantor paspor, terbit, habis | 1 (mahram & relation kosong) |
| Asuransi | Asuransi | No, nama paspor, L/P, usia, NIK, tempat & tanggal lahir, data paspor | 1 |
| Roomlist | Hotel | hotel Mekah & Madinah, total kamar per tipe, no, nama, L/P, tipe kamar, nomor kamar, keterangan | 2 |
| Speed Train | Vendor | seperti Blok Seat tanpa keterangan, hanya pax yang ikut | 3 |
| Telefric | Vendor | seperti Speed Train | 3 |

Sheet yang belum fasenya tidak ikut di file (disembunyikan, bukan dikosongkan).

## 3. Pemetaan data

**Sudah ada:** `jamaah.nama_lengkap`, `jenis_kelamin`, `nik`, `tempat_lahir`, `tanggal_lahir`, `no_paspor`, `tanggal_paspor_keluar`, `tempat_paspor_keluar`, `paspor_berlaku_sampai`; `booking_pax.room_type`, `pax_type`, `pax_status`; maskapai, kode penerbangan, tanggal, hotel Mekah/Madinah di `schedules`.

**Perlu ditambah:**

| Data | Tempat | Fase |
|---|---|---|
| Nama paspor | kolom baru di `jamaah`, kosong berarti pakai `nama_lengkap` | 1 |
| PNR, Detail penerbangan | kolom baru per paket | 1 |
| Jejak unduhan (siapa, kapan) | tabel baru | 1 |
| Pembagian kamar dan petugas (muthowif) | tabel baru | 2 |
| Catatan seat per pax | kolom/tabel baru | 2 |
| Ikut layanan per pax | tabel baru, terhubung ke `schedule_add_ons` | 3 |

Bandara di ERP disimpan sebagai nama bebas tanpa kode IATA, dan jam tiba tidak disimpan. Karena itu baris Flight Detail diisi manual dan tidak dibentuk dari data paket.

## 4. Keputusan

### Umum
1. Fitur berupa tab **Manifest** di Detail Paket, bukan menu baru.
2. Pax yang masuk: pax aktif dari booking berstatus `baru`, `dp`, dan `lunas`. Booking berstatus `baru` diberi tanda. Draft dan booking batal tidak masuk.
3. Urutan: per booking menurut waktu dibuat, lalu per pax menurut urutan pengisian.
4. Satu file `.xlsx`, urutan sheet seperti file sekarang. Nama file `<tanggal> <nama paket>.xlsx`.
5. File dibuat dengan dependency Go `excelize` (disetujui).
6. Satu PNR per paket. Rombongan dengan dua PNR menulis keduanya di isian yang sama.
7. Semua tanggal di file berformat teks `14-SEP-2026`.
8. Jenis tiket dihitung dari usia pada tanggal berangkat: di bawah 2 tahun INFANT, 2 sampai 11 tahun CHILD, sisanya ADULT. Batas usia Saudia dicek sebelum dibangun.
9. Usia di sheet Asuransi dihitung pada tanggal berangkat.
10. Izin baru **Kelola manifest**: mengisi PNR, membagi kamar, mengunduh. Admin Brand hanya untuk paket brand-nya sendiri.
11. Jejak unduhan dicatat dan ditampilkan satu baris di tab ("Terakhir diunduh <nama>, <waktu>").
12. Manifest tidak punya status terkirim atau dikunci. Admin bisa mengubah dan mengunduh ulang kapan saja.

### Pemeriksaan data
13. Sebelum unduh, tab menampilkan daftar masalah sebagai peringatan, tanpa mengunci unduhan:
    - paspor kosong;
    - paspor habis kurang dari 6 bulan sebelum tanggal pulang;
    - jenis kelamin atau tanggal lahir kosong.
14. Setiap masalah menautkan ke Detail Jamaah. Perbaikan dilakukan di sana, bukan di tab Manifest.

### Pembagian kamar (Fase 2)
15. Saat tab dibuka pertama kali, ERP membuat usulan: pax dari booking yang sama dengan tipe kamar yang sama dijadikan satu kamar. Sisanya masuk **Belum dapat kamar**.
16. Satu kamar hanya berisi pax dengan tipe kamar yang sama.
17. Pax dipindah lewat pilihan **Kamar** per pax (Kamar 1, Kamar 2, Belum dapat kamar), bukan seret-lepas.
18. Nomor kamar berupa urutan 1, 2, 3, bukan nomor kamar hotel.
19. Satu pembagian kamar dipakai untuk Mekah dan Madinah.
20. Petugas (muthowif, tour leader) bisa ditambahkan ke kamar sebagai nama bebas, tidak dihitung sebagai pax.
21. Kamar boleh belum penuh, dengan tanda peringatan.
22. Tidak ada pemeriksaan campuran laki-laki dan perempuan dalam satu kamar.
23. Pax yang batal otomatis keluar dari kamarnya. Pax baru masuk **Belum dapat kamar**. Kamar lain tidak diubah.
24. Ringkasan jumlah kamar per tipe dihitung otomatis. Tipe kamar ditulis `QUAD`.

### Catatan seat (Fase 2)
25. Setiap pax punya isian pendek **Catatan seat**, disalin ke kolom Keterangan di sheet Blok Seat.

### Layanan tambahan (Fase 3)
26. Setiap pax diberi centang **ikut** untuk setiap layanan dari master add-on paket (`schedule_add_ons`). Centang ini tidak terhubung ke `booking_addons` atau tagihan.
27. Sheet Speed Train dan Telefric hanya berisi pax yang dicentang.

## 5. Tahapan

| Fase | Isi | PR |
|---|---|---|
| 1 | Tab Manifest, PNR & detail penerbangan, nama paspor, pemeriksaan data, unduh 4 sheet, jejak unduhan, izin | sendiri |
| 2 | Pembagian kamar, petugas, catatan seat, sheet Roomlist | sendiri |
| 3 | Centang layanan per pax, sheet Speed Train & Telefric; mahram dan relasi bila masih wajib | sendiri, menunggu jawaban provider visa |

Setiap fase mengikuti aturan verifikasi proyek: `go build ./...`, `go test ./...`, `npm run build`, dan screenshot di lebar 375px dan 1440px.

## 6. Masih terbuka

- **Mahram dan relation di sheet Visa.** Kementerian Haji dan Umrah Saudi menyatakan sejak 2022 wanita boleh umroh tanpa mahram, tapi sumber lain menyebut batas usia 45 tahun. Perlu ditanyakan ke provider visa apakah kolom ini masih wajib. Kalau tidak, kolomnya dihapus dari sheet Visa. Kalau masih wajib, cukup satu isian mahram per pax wanita, dipilih dari pax laki-laki di paket yang sama. Fitur relasi jamaah yang lama sengaja dihapus (migrasi 096) dan tidak dibangun ulang.
- **Kolom Relation di Blok Seat dan Manifest** (SUAMI/ISTRI) diputuskan bersama mahram.
