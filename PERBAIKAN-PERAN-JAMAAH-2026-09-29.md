# Perbaikan peran jamaah

Tanggal: 28–29 September 2026. Acuan: `AUDIT-PERAN-JAMAAH-2026-09-28.md`.

## Status penerapan

Perbaikan 24 temuan sudah ditulis pada backend ERP, dashboard bersama, dan microsite. Pengujian dilakukan pada kode lokal, fixture database terisolasi, dan browser dengan respons sintetis. Ini bukan konfirmasi bahwa backend yang sedang berjalan sudah memakai kode baru.

Migrasi `066_portal_security_documents.sql` belum dijalankan pada tabel permanen. Backend aktif belum direstart. Tidak ada commit atau push pada pekerjaan ini. Perubahan paket/self-booking sebelumnya tetap dipertahankan.

## Pemetaan temuan

| ID | Perbaikan | Bukti utama |
|---|---|---|
| JM-01 | Profil portal memakai whitelist, tanpa catatan internal. | `internal/portal/view.go`; uji profil |
| JM-02 | JWT terikat pada hash PIN; reset PIN dan logout membatalkan sesi. Login memakai hash yang benar-benar diverifikasi, sehingga reset bersamaan tidak menerbitkan sesi baru yang sah. | `internal/identity/portal_session.go`; uji sesi |
| JM-03 | Service worker mengecualikan invoice, portal, API dan respons private/no-store; cache versi lama dibuang saat aktivasi. | `public/sw.js`; probe PWA |
| JM-04 | Batas pelunasan dan ketentuan DP/pembayaran penuh berasal dari snapshot checkout. Booking tanpa snapshot menampilkan instruksi menghubungi petugas. | DTO booking, beranda dan pembayaran |
| JM-05 | Reservasi kedaluwarsa/tidak terkunci tidak menawarkan rekening atau konfirmasi transfer; status diperiksa ulang sebelum unggah. | DTO booking, halaman pembayaran, uji browser |
| JM-06 | Pratinjau dokumen melalui endpoint portal dengan autentikasi dan kepemilikan akun, ditampilkan melalui blob URL. | `internal/media/ownership.go`; uji HTTP 200/404 |
| JM-07 | Unggahan terlindungi mencatat pemilik; dokumen/bukti portal hanya menerima berkas akun itu yang benar-benar tersedia. URL eksternal, path traversal, dan file akun lain ditolak. | metadata media dan uji kepemilikan |
| JM-08 | Halaman pembayaran memilih booking secara eksplisit dan mempertahankan ID pada tautan. | pemilih booking; uji kebijakan dan browser |
| JM-09 | Kemampuan bayar/invoice diberikan server sesuai PIC; peserta lain mendapat penjelasan akses baca. | DTO booking dan UI pembayaran |
| JM-10 | Status visa, Siskopatuh, manasik dan perlengkapan memakai `personal_pax`. | DTO booking, beranda/perjalanan |
| JM-11 | Profil membaca `no_paspor`. | probe UI dan browser |
| JM-12 | Perlengkapan membaca `perlengkapan_status === 'sudah_diberikan'`. | browser perjalanan |
| JM-13 | Gagal mengambil data ditampilkan sebagai error dengan retry, bukan keadaan kosong atau token aktivasi tidak valid. | halaman pembayaran/perjalanan/aktivasi |
| JM-14 | Progress menghitung dokumen wajib; dokumen opsional tidak mengurangi kelengkapan. Persyaratan umum anak menyesuaikan identitas; petugas tetap mengonfirmasi persyaratan perjalanan. | `portalPolicy.mjs`; uji kebijakan |
| JM-15 | Penggantian dokumen menaikkan versi. Persetujuan mensyaratkan versi yang ditinjau; versi lama ditolak. Versi terdahulu disimpan dalam riwayat. | repository, DocumentReview, uji konflik/rollback |
| JM-16 | Penolakan mensyaratkan alasan; jamaah melihat alasan dan dapat mengganti berkas. | handler, dashboard, perjalanan |
| JM-17 | Progress paspor mensyaratkan dokumen approved. | repository booking |
| JM-18 | Gangguan jaringan tidak menghapus sesi; HTTP 401 memulihkan layar login; kegagalan storage mendapat pesan eksplisit. Logout menunggu pencabutan server. | PortalAuthContext, portalApi |
| JM-19 | Penerbitan ulang link aktivasi memakai transaksi dan lock akun. Aktivasi menutup seluruh token yang belum dipakai pada akun tersebut. | `internal/portal/activation.go` |
| JM-20 | Domain aktivasi tanpa skema memakai HTTPS, kecuali domain lokal `.test`/localhost. Skema eksplisit konfigurasi tetap dihormati. | `internal/portal/activation.go` |
| JM-21 | Memilih bukti pengganti yang tidak valid mengosongkan state file lama dan input. | probe yang menjalankan handler UI |
| JM-22 | Tanggal transfer dapat diisi pengguna, default WIB; backend menolak format salah/tanggal masa depan. | formulir dan handler; uji tanggal WIB |
| JM-23 | Dialog memiliki label, fokus awal, perangkap fokus, Escape dan pemulihan fokus. Input memiliki label; unggah dokumen memiliki indikator fokus keyboard. | PortalDialog, FormField/Label; browser |
| JM-24 | Jamaah biasa dapat melihat saldo dan riwayat kredit cashback serta cara penggunaannya. | endpoint cashback dan panel profil; browser |

JM-24 juga tercatat sebagai AG-11 pada audit agen, sehingga bukan dua perbaikan terpisah.

## Verifikasi

- `go test ./...`: lulus. Integrasi DB pada pemanggilan standar dapat dilewati oleh gate environment.
- `ERP_TEST_DB=1 go test -v ./internal/identity ./internal/dokumen ./internal/media ./internal/portal ./internal/selfbooking ./internal/booking ./internal/payment`: lulus. Skenario portal memakai tabel MySQL sementara pada koneksi khusus; tes booking/pembayaran memakai fixture transaksi dan rollback.
- `node --test C:/laragon/www/azhan-microsite/tests/*.test.mjs`: 15 lulus, 0 gagal, 0 dilewati.
- Uji tambahan `TestBookingViewUsesSnapshotAndPersonalPermissions`: lulus. Ekspektasi awal test sempat gagal karena memperlakukan DATETIME WIB sebagai UTC; diperbaiki agar memeriksa konversi yang benar. Implementasi konversi tidak diubah.
- Probe PWA dan UI: lulus.
- Build master-dashboard dan travel-dashboard: exit 0. Peringatan ukuran bundle tetap ada.
- Build final microsite: exit 0. Percobaan sandbox gagal dengan `Failed to fetch DM Sans from Google Fonts`; pengulangan dengan akses jaringan lulus. Log akhir: `audit/jamaah-microsite-build-2026-09-29.log`. Peringatan konvensi middleware Next.js tetap ada.
- `git diff --check` kedua repo: tidak ada kesalahan whitespace; ada peringatan konversi LF/CRLF.

Bukti mentah uji penting:

```text
GET protected: 401 {"error":"sesi telah berakhir, silakan masuk kembali"}
GET protected: 204
GET protected: 401 {"error":"sesi telah berakhir, silakan masuk kembali"}
GET own-document account=1: 200 %PDF-1.4 synthetic
GET own-document account=2: 404 404 page not found
failed replacement rolled back: history rows=0; original version=1; status=rejected
stale version rejected; replacement remains submitted; prior rejection retained; current version approved
PASS: invoice, portal, activation, and API bypass service-worker caching
PASS: private/no-store response is not cached
PASS: eligible public navigation still caches
PASS: API no_paspor renders its value
PASS: invalid replacement clears the previous payment proof and input
# tests 15
# pass 15
# fail 0
# skipped 0
snapshot due_at=2026-10-10T17:00:00Z (11 Oct 00:00 WIB); non-PIC payment=false invoice=false; expired hold payment=false; personal pax=2 visa=false
--- PASS: TestBookingViewUsesSnapshotAndPersonalPermissions (0.33s)
```

Browser pada `hana.azhan.test:3001` menggunakan akun dan respons API sintetis di tab pengujian. Tidak memakai akun jamaah nyata. Diperiksa pada 375×812 dan 1440×900: pembayaran, pergantian booking, reservasi kedaluwarsa, alasan penolakan dokumen, perlengkapan, nomor paspor, cashback, dan logout. Escape menutup dialog dan mengembalikan fokus ke tombol `Konfirmasi transfer`; desktop tidak overflow horizontal. Tab fixture sudah ditutup, sesi sintetis dihapus melalui logout, viewport dipulihkan, dan server sementara dihentikan.

Bukti gambar:

- `audit/jamaah-payment-mobile.png`
- `audit/jamaah-payment-desktop.png`
- `audit/jamaah-documents-mobile.png`
- `audit/jamaah-documents-desktop.png`

## Urutan penerapan

1. Buat backup database terbaru dan verifikasi integritas backup. Backup pra-065 tidak menggantikan backup baru ini.
2. Pastikan migrasi 065 sudah diterapkan; periksa bahwa 066 belum pernah dijalankan sebagian.
3. Jalankan 066: tiga tabel baru untuk pencabutan sesi, metadata unggahan, riwayat dokumen; dua kolom baru pada dokumen. ALTER pada migrasi ini tidak boleh dijalankan ulang secara buta.
4. Jalankan backend baru, lalu dashboard dan microsite dari build yang sesuai. Frontend baru memerlukan endpoint/field backend baru.
5. Verifikasi login, akses file, unggah, review versi, pembayaran dan cashback pada layanan yang sudah diperbarui.

Semua token portal lama yang belum terikat PIN harus login ulang. Berkas lama yang sudah terhubung ke dokumen/pembayaran tetap dapat dibaca admin sesuai brand. Referensi unggahan baru mensyaratkan metadata pemilik; unggahan lama yang belum terhubung perlu diunggah ulang. Tidak ada backfill yang menebak pemilik file.

Aturan proyek `.agents/rules/AGENTS.md` menyatakan: "Migrasi database ... boleh dieksekusi kalau sudah dijelaskan di prompt secara eksplisit". Instruksi migrasi sebelumnya telah digunakan untuk 065. Migrasi baru 066 masih menunggu instruksi eksplisit.

## Batas verifikasi

Pengujian browser memakai respons sintetis; pengujian backend/DB dilakukan terpisah. Belum dilakukan alur akun nyata terhadap server yang sudah direstart dan skema permanen 066. Penerbitan aktivasi atomik dan skema HTTPS telah diperiksa pada kode, tetapi belum diuji melalui browser admin dengan akun nyata. Tidak ada perubahan kebijakan wali, refund otomatis, OTP, atau pembalikan komisi.
