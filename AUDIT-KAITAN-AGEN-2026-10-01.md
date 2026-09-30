# Audit Alur Ganti Kaitan Agen (C4) — 2026-10-01

Lingkup: Admin Master mengganti kaitan agen jamaah (`/komisi/ganti-kaitan`).

- **Backend:**
  - `internal/agen/kaitan_ganti.go`
  - `sprint5_handler.go`
  - rute `GET/PUT /api/admin/kaitan-agen`, `GET /api/admin/kaitan-agen/{id}/log`
- **Frontend:** `master-dashboard/src/pages/GantiKaitanAgenPage.jsx`
- **Titik temu dengan pencatatan komisi:** `internal/komisi/kalkulasi.go` (`ProcessBookingLunas`)

Spesifikasi `agen-azhan.md` tidak ada di repo. Aturan acuan diambil dari komentar kode dan `AUDIT-SPRINT-agen-syiar.md`:

- hanya Admin Master;
- hanya kaitan hasil Jalur 3;
- hanya selama jamaah belum pernah menghasilkan komisi;
- alasan wajib dan tercatat di log;
- tidak berdampak ke jamaah lain dan tidak menghitung ulang komisi.

Bukti mentah ada di `audit/evidence-kaitan.log` (label K*, R*, L*, U*). Semua data uji dibuat sementara lalu dipulihkan. Baseline akhir: 38 jamaah `jalur3/tanpa_agen/tidak_aktif`, `jamaah_kaitan_log` = 0, `transaksi_komisi` = 0.

## Yang sudah benar (terverifikasi)

| Aspek | Bukti |
|---|---|
| Admin Travel ditolak di cari, ganti, dan log (403) | K1–K3 |
| Pencarian di bawah 3 huruf mengembalikan kosong | K4 |
| Alasan kurang dari 5 karakter, mode tidak dikenal, dan agen kosong ditolak (400) | K6–K8 |
| Agen ditolak bila dirinya sendiri, dari brand lain, atau nonaktif (400) | K9–K11 |
| Perubahan ke kaitan yang sama ditolak (400) | K12, K20, U (UI) |
| Jamaah tidak ada: 404 | K13 |
| Gate: sudah punya komisi, bukan Jalur 3, jamaah agen aktif, jamaah sedang pengajuan agen (409) | K14–K17 |
| Transisi tanpa → agen A → agen B → tanpa, setiap langkah tercatat di log (lama/baru, admin, alasan) | K19–K25 |
| `kaitan_sumber` tetap `jalur3` setelah diganti | K23 |
| Insert komisi yang belum commit ikut tertangkap gate: ganti menunggu kunci FK, lalu ditolak 409 | R2 (durasi 2,6 detik) |
| UI: cari, pilih, ganti, pesan sukses, log langsung tampil, error "sama" tampil; lebar 375 tanpa scroll horizontal | U1 + screenshot |
| Test `TestLangkah15dan16GantiKaitan` | PASS |

## Temuan

| ID | Tingkat | Temuan | Bukti |
|---|---|---|---|
| KA-01 | Sedang (jarang terjadi) | **Race dengan pencatatan komisi saat lunas.** `ProcessBookingLunas` membaca `direkrut_oleh_jamaah_id` tanpa kunci (`loadPax`). Bila ganti kaitan terjadi setelah pembacaan itu tetapi sebelum komisi di-insert, keduanya sukses. Hasilnya: kaitan pindah ke agen baru, tetapi komisi tercatat untuk agen lama, dan gate "belum pernah menghasilkan komisi" terlewati. | R1: ganti 200, lalu jamaah 6 `agen_sekarang=2`, `penerima_komisi=1` |
| KA-02 | Rendah | **Query kunci C4 ikut mengunci baris `brands` dan baris agen lama.** `SELECT … JOIN brands … FOR UPDATE` mengunci semua tabel dalam join. Baris `brands` adalah kunci counter nomor jamaah (`shared/jamaah.go`, `jamaah/repository.go`, `selfbooking/*`, `crmdeal`), sehingga pembuatan jamaah di brand itu menunggu C4 selesai. Urutan kuncinya (jamaah lalu brands) berlawanan dengan self-booking (brands lalu jamaah PIC), sehingga **berpotensi deadlock**. Potensi deadlock ini belum direproduksi. | L1: `brands` id 1 terkunci (ERROR 3572); kontrol L2/L3 bebas; L4 bebas setelah selesai |
| KA-03 | Keputusan | **Gate hanya mengecek komisi, bukan riwayat lunas, dan pesannya tidak sesuai.** Pesan error berbunyi "booking sudah pernah lunas", padahal jamaah yang bookingnya sudah lunas tanpa komisi (booking non-Syiar, snapshot NULL) tetap bisa diganti. Selain itu, booking yang masih DP/baru saat kaitan diganti akan membayar komisi ke **agen baru** ketika lunas (sumber kode: `loadPax` membaca kaitan saat lunas). Tidak ada peringatan di UI bahwa jamaah punya booking aktif. | K18a/K18: booking #70 lunas, komisi 0, ganti 200 |
| KA-04 | Rendah | Endpoint log untuk jamaah yang tidak ada menjawab `200 []`, bukan 404. | K26 |
| KA-05 | Rendah (UX) | Picker agen ikut menampilkan agen yang sedang terkait; bila dipilih baru ditolak backend (pesan huruf kecil dari backend). Pesan sukses lama tetap tampil saat memulai perubahan berikutnya. Badge "Bisa diganti" terlipat dua baris di daftar hasil. | Screenshot UI |
| KA-06 | Rendah (aturan AGENTS.md) | `GantiKaitanAgenPage.jsx` memakai nilai Tailwind arbitrary `max-h-[480px]` dan `<input>` mentah, bukan komponen `Input`. | Kode baris 131, 150 |

## Rekomendasi

1. **KA-01:** kunci baris jamaah pax sebelum membaca kaitan di `ProcessBookingLunas`, misalnya `loadPax` memakai `… FOR SHARE OF j` atau `FOR UPDATE OF j`.
   - Ganti kaitan yang datang di tengah proses akan menunggu, lalu melihat komisi dan ditolak 409.
   - Tambahkan test dua koneksi (pola R1).
2. **KA-02:** ubah query kunci C4 menjadi `FOR UPDATE OF j`, sehingga hanya baris jamaah target yang dikunci. Tidak ada perubahan perilaku lain.
3. **KA-03:** perlu keputusan:
   - (a) pertahankan aturan "belum ada komisi" dan perbaiki pesan menjadi "sudah pernah menghasilkan komisi", ditambah peringatan di UI bila jamaah punya booking aktif (baru/DP) yang komisinya nanti ikut ke agen baru; atau
   - (b) perketat gate menjadi "belum pernah punya booking lunas" (`pertama_lunas_at`).
   - Saran saya: (a).
4. **KA-04:** kembalikan 404 bila jamaah tidak ada.
5. **KA-05:** sembunyikan atau nonaktifkan agen yang sedang terkait di picker, dan kosongkan pesan sukses saat form diubah.
6. **KA-06:** ganti dengan komponen `Input` dan kelas tinggi dari skala Tailwind.

## Status perbaikan (2026-10-01, KA-03 opsi a)

| ID | Perbaikan | Verifikasi |
|---|---|---|
| KA-01 | `komisi.loadPax` memakai `FOR SHARE OF j`, sehingga kaitan yang dibaca terkunci sampai komisi tercatat. | R1b: ganti menunggu 2,6 detik lalu 409; kaitan tetap agen 1, penerima komisi agen 1, log 0. Test `komisi`, `payment`, `booking` PASS. |
| KA-02 | Query kunci C4 memakai `FOR UPDATE OF j`. | L1b: `brands` bebas; L2b: agen lama bebas; L3b: jamaah target terkunci. |
| KA-03 | Pesan diubah menjadi "sudah pernah menghasilkan komisi". Field baru `booking_belum_lunas` (draft/baru/dp) ditampilkan di UI sebagai baris informasi, disertai peringatan bahwa komisinya akan ikut agen baru. | V1, V2 (=1, booking #10 dp), V3 (=0, booking #70 lunas); test `TestLangkah15dan16GantiKaitan` diperluas; screenshot UI. |
| KA-04 | `listKaitanLog` mengembalikan `ErrNotFound` (404) bila jamaah tidak ada. | V4: 404; V5: jamaah ada, 200 `[]`; test diperluas. |
| KA-05 | `KaitanAgenPicker` mendapat prop `excludeAgenId`, sehingga agen yang sedang terkait disembunyikan. Pesan sukses hilang saat pilihan agen atau alasan diubah. Badge tidak terlipat. | UI: picker hanya menampilkan Siti saat terkait ke Ahmad; pesan sukses hilang setelah mengetik. |
| KA-06 | Diganti dengan komponen `Input` dan `max-h-96`. | Kode + UI. |

Build Master dan Travel OK. `go test ./... -p 1` PASS dua kali berturut-turut. Semua data uji sudah dipulihkan: baseline 38 jamaah, log 0, komisi 0.

**Di luar lingkup:** dua test sesekali gagal di paket yang tidak diubah. `TestPublicScheduleEffectiveDPProjection` gagal saat `go test ./...` berjalan paralel, dan `TestLockNotBatalTx` gagal sekali saat berurutan. Keduanya lulus saat diulang (5× dan 2 run penuh). Pesan error-nya tidak tertangkap. Kemungkinan penyebabnya interferensi DB bersama, baik antar paket maupun dengan worker API yang sedang berjalan, tetapi ini belum dipastikan.
