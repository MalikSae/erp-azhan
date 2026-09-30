# Audit end-to-end Agen Umroh / Syiar — 28 September 2026

## Ringkasan

Audit menghasilkan **14 temuan: 3 P1 dan 11 P2**, serta satu hambatan lingkungan dan risiko bisnis yang dipisahkan dari bug. P1 perlu diselesaikan sebelum fitur dinyatakan siap untuk penggunaan operasional.

Cakupan: pendaftaran akun, pengajuan agen, persetujuan admin, pembayaran pendaftaran, referral dan kaitan jamaah, booking agen, hook komisi, cashback, saldo, pencairan, akses dokumen, dashboard Master/Travel, dan microsite. Acuan bisnis yang tersedia: `AUDIT-SPRINT-agen-syiar.md`, `RELEASE-agen-syiar.md`, serta ketentuan yang benar-benar tampil di microsite. Dokumen `agen-azhan.md`/`screen-agen.md` yang disebut kode tidak ditemukan di kedua repo.

**Tingkat bukti:** temuan implementasi di bawah berasal dari penelusuran kode, bukan klaim eksploitasi atau transaksi yang sudah dilakukan. Pemeriksaan HTTP dan browser yang benar-benar dijalankan dicatat terpisah. Lima probe audit disiapkan, tetapi belum dijalankan karena review persetujuan otomatis gagal akibat batas penggunaan layanan. Tidak ada hasil PASS probe yang diklaim.

Tidak mengubah kode aplikasi, skema, data permanen, status agen, komisi, atau pencairan. Perubahan audit hanya laporan dan dua file probe. Perubahan working tree dari audit paket/self-booking sebelumnya tetap dipertahankan.

## P1 — integritas komisi dan isolasi brand

### AG01 — Klasifikasi repeat dapat berubah dan menambah komisi pada booking yang sama

**Lokasi:** `internal/komisi/kalkulasi.go:60`, `:132–157`; `internal/booking/repository.go:963`.

`isRepeatOrder` membaca `pax_status='aktif'` dari booking historis setiap kali hook dipanggil. Unique key ledger mencakup jenis komisi, sehingga mencegah duplikasi jenis yang sama tetapi tidak perubahan repeat menjadi langsung.

Skenario: booking A pernah lunas; booking B untuk jamaah sama mendapat repeat Rp500.000 + cashback Rp500.000. Pax di A dibatalkan, lalu B turun dari lunas ke DP karena tagihan bertambah dan kembali lunas. B kini diklasifikasikan jamaah baru; komisi langsung Rp1.000.000 dapat ditambahkan tanpa menghapus ledger sebelumnya. Bonus pembinaan juga dapat bertambah bila ada upline.

**Dampak:** total hak komisi booking yang sama dapat berlipat. **Perbaikan:** simpan klasifikasi, penerima, dan keputusan kelayakan saat pemrosesan pertama per pax; replay hanya membaca keputusan tersebut. Jangan menghitung ulang identitas hak dari status historis yang dapat berubah. Probe: `TestAuditAgenRepeatChangesAfterHistoricalPaxCancellation` (belum dijalankan).

### AG02 — Komisi periode nonaktif dapat muncul setelah agen aktif kembali

**Lokasi:** `internal/komisi/kalkulasi.go:67–85`, `:102–112`; ketentuan microsite `src/app/portal/syiar/syarat-ketentuan/page.jsx:16`.

Hook melewati agen/upline nonaktif tanpa menyimpan keputusan final bahwa hak komisi tersebut tidak diberikan. Ketika agen diaktifkan kembali dan booking mengalami pelunasan ulang, status aktif dibaca ulang; baris yang sebelumnya tidak ada kini dapat dibuat.

**Dampak:** bertentangan dengan ketentuan bahwa komisi selama nonaktif tidak dibayarkan kemudian. **Perbaikan:** simpan keputusan eligible/skipped beserta alasan pada pelunasan pertama, termasuk hak bernilai nol/tidak diterbitkan. Probe: `TestAuditAgenSkippedCommissionAppearsAfterReactivation` (belum dijalankan).

### AG03 — Media sensitif tidak dibatasi berdasarkan brand pemilik

**Lokasi:** `internal/media/handler.go:57–69`; `cmd/api/main.go:330`.

`ServeProtected` memeriksa login admin, kategori, dan keamanan nama file, lalu mengirim file. Tidak ada pemeriksaan brand atau pemilik dokumen. Admin Travel yang mengetahui URL file brand lain dapat meminta foto agen atau bukti transfer tersebut. UUID mengurangi kemungkinan menebak, tetapi tidak menggantikan otorisasi.

**Dampak:** batas kerahasiaan antarbrand bergantung pada URL tetap tersembunyi. **Perbaikan:** simpan metadata upload (brand, pengunggah, tujuan), periksa kepemilikan pada setiap akses, dan bedakan izin Master dari Travel. Audit ini tidak mengambil dokumen nyata lintas brand.

## P2 — backend, kontrak frontend, dan UI/UX

### AG04 — URL foto/bukti dianggap sah hanya dari bentuk string

**Lokasi:** `internal/agen/handler.go:28–29,55,84`; `internal/agen/sprint5_handler.go` pada `SetujuiPencairan`; `internal/agen/repository.go:125`.

URL berpola UUID diterima tanpa memastikan file ada dan berasal dari unggahan pengguna/brand yang berhak. Foto rekaan dapat memenuhi syarat pengajuan; bukti transfer keluar yang tidak ada dapat disimpan sebagai bukti pencairan. Kewajiban upload di UI tidak melindungi API langsung.

**Perbaikan:** gunakan ID upload terdaftar, verifikasi eksistensi/tipe/owner dan tujuan pemakaian secara server-side. Probe foto rekaan tersedia, belum dijalankan.

### AG05 — Perubahan biaya pendaftaran tidak meminta persetujuan ulang

**Lokasi:** `internal/agen/model.go:73–77`; `internal/agen/repository.go:111–140`; microsite `src/app/portal/syiar/kelengkapan-agen/page.jsx:69–73,151`.

Frontend menampilkan biaya hasil GET, sedangkan POST hanya mengirim foto, domisili, dan boolean persetujuan. Backend memakai biaya brand terkini. Jika biaya berubah ketika formulir terbuka, tagihan dapat berbeda dari nominal yang disetujui pengguna, termasuk dari gratis menjadi berbayar.

**Perbaikan:** kirim nominal/versi quotation yang disetujui; balas konflik bila berubah, tampilkan biaya baru, dan minta persetujuan ulang sebelum membuat siklus pembayaran.

### AG06 — Persetujuan ketentuan belum punya versi dan isi belum cukup menjelaskan komisi

**Lokasi:** `internal/agen/repository.go:125`; microsite `src/app/portal/syiar/syarat-ketentuan/page.jsx:7–17`.

Hanya waktu persetujuan terakhir yang disimpan pada jamaah; tidak ada versi/hash teks per siklus. Ringkasan yang tampil belum menjelaskan pembagian repeat 50/50, bonus satu tingkat, batasan cashback, dan pengecualian cashback terhadap status agen. Kalimat “komisi jenis apa pun” selama nonaktif tidak mencerminkan pengecualian cashback yang memang ada di mesin komisi. Komentar kode masih menyebut teks sebagai draf.

**Perbaikan:** finalisasi teks operasional dengan pemilik bisnis, simpan versi dan timestamp per siklus, serta tampilkan aturan dan pengecualian yang sesuai implementasi. Ini temuan konsistensi produk, bukan penilaian legal.

### AG07 — Membaca ketentuan dapat menghilangkan isian pengajuan

**Lokasi:** microsite `src/app/portal/syiar/kelengkapan-agen/page.jsx:20–24,162`.

Foto, domisili, dan persetujuan hanya berada di state komponen. Tautan ketentuan berpindah halaman pada tab yang sama tanpa penyimpanan draf. Ketika komponen dimount ulang, pengguna harus memilih file dan mengisi domisili lagi.

**Perbaikan:** buka ketentuan dalam dialog/tab baru, atau pertahankan draf nonsensitif dengan batas waktu. Uji alur isi → baca syarat → kembali tanpa kehilangan pekerjaan.

### AG08 — Akun berhasil dibuat tetapi kegagalan login dilaporkan sebagai kegagalan pendaftaran

**Lokasi:** microsite `src/app/portal/daftar-agen/page.jsx:69–96`; `src/context/PortalAuthContext.jsx` fungsi `login`.

POST pendaftaran dan login berikutnya berbagi blok try/catch. Timeout login atau kegagalan localStorage sesudah akun dibuat menghasilkan pesan kegagalan umum. Pengguna mengulang pendaftaran lalu mendapat konflik nomor sudah terdaftar.

**Perbaikan:** pisahkan status akun berhasil dibuat dari login, sediakan retry login tanpa mendaftar ulang, dan tampilkan petunjuk pemulihan yang jelas.

### AG09 — Dashboard menawarkan PDF, uploader payment-proofs menolaknya

**Lokasi:** `internal/media/handler.go:129`; `frontend/master-dashboard/src/pages/KomisiReferralPage.jsx:101,269`; Travel `PersetujuanAgenPage.jsx` pada upload bukti admin.

PDF hanya diterima uploader untuk kategori `dokumen-jamaah`. Dashboard mengirim bukti pencairan ke `payment-proofs`, sementara input dan validator URL menerima PDF.

**Dampak:** bukti PDF valid tidak dapat menyelesaikan alur persetujuan. **Perbaikan:** selaraskan kategori/MIME backend dan batas frontend, termasuk pesan kesalahan. Tidak ada file pengguna yang diunggah pada audit ini.

### AG10 — Nomor rekening tanpa digit lolos validasi pencairan

**Lokasi:** `internal/agen/pencairan.go:69–79`; microsite `src/app/portal/syiar/pencairan/page.jsx:168`.

Validator mengizinkan digit, spasi, dan tanda hubung, tetapi tidak mewajibkan minimal satu digit. Nilai `---` tidak kosong dan lolos seluruh pemeriksaan karakter. Frontend juga mempertahankan tanda hubung.

**Perbaikan:** normalisasi separator, wajibkan nomor yang berisi digit, dan terapkan panjang operasional yang disepakati. Jangan mengklaim validasi format membuktikan rekening benar-benar ada. Probe tersedia, belum dijalankan.

### AG11 — Jamaah biasa tidak bisa melihat kredit cashback yang menjadi haknya

**Lokasi:** `internal/agen/kinerja.go:163–181`; microsite `src/app/portal/syiar/DashboardAgen.jsx:42–50`; `internal/booking/cashback.go`.

Kredit cashback tampil hanya pada dashboard agen aktif. Penerima cashback bisa jamaah biasa; belum ada ringkasan kredit pada portal biasa atau instruksi yang menghubungkan penerima ke proses penggunaan melalui admin. Pemakaian kredit oleh admin sebenarnya sudah tersedia.

**Perbaikan:** tampilkan saldo/riwayat cashback milik jamaah dan cara meminta pemakaian pada booking berikutnya. Jangan membuat pencairan tunai cashback karena keputusan D4 adalah potongan tagihan.

### AG12 — “Link tersalin” ditampilkan sebelum clipboard berhasil

**Lokasi:** microsite `src/app/portal/syiar/page.jsx:127–130`.

`navigator.clipboard?.writeText` tidak ditunggu dan hasilnya tidak ditangani; `setCopied(true)` selalu dijalankan. Pada HTTP lokal atau izin clipboard ditolak, UI tetap mengatakan berhasil dan tautan lengkap tidak disediakan untuk salin manual.

**Perbaikan:** await hasil clipboard, tampilkan sukses hanya setelah resolve, dan sediakan tautan teks yang dapat dipilih sebagai fallback.

### AG13 — Form rekening pencairan mengandalkan placeholder

**Lokasi:** microsite `src/app/portal/syiar/pencairan/page.jsx:154–180`.

Nama bank, nomor rekening, dan atas nama tidak mempunyai label terasosiasi. Placeholder hilang setelah diisi; pengguna pembaca layar tidak mendapat label yang andal. Pesan kesalahan juga tidak dikaitkan ke field atau diberi fokus.

**Perbaikan:** label yang selalu terlihat, id/htmlFor, aria-describedby untuk error, dan fokus ke field pertama yang tidak valid. Halaman pendaftaran awal sudah memiliki label yang benar; temuan ini khusus form pencairan.

### AG14 — Gangguan API mengunci beberapa halaman tanpa pemulihan yang jelas

**Lokasi:** microsite `src/app/portal/syiar/DashboardAgen.jsx:16–20`; `riwayat-komisi/page.jsx` fungsi `muat`; `pencairan/page.jsx:40–47`; `src/lib/portalApi.js` endpoint agen.

Dashboard berhenti pada teks error tanpa retry. Riwayat kosong yang gagal dimuat tidak menampilkan tombol muat lagi. Beberapa fungsi selalu memanggil `res.json()` sehingga respons HTML/plain text 404/502 berubah menjadi error parsing yang membingungkan. Error lama tidak konsisten dibersihkan setelah permintaan berikutnya berhasil.

**Perbaikan:** helper respons yang aman terhadap non-JSON, status loading/error/empty yang berbeda, retry eksplisit, dan pembersihan error saat pemulihan berhasil.

## Risiko bisnis yang bukan bug implementasi

1. **Komisi booking batal tetap dapat dicairkan ketika tanggal jadwal lewat.** `komisi/saldo.go` menggunakan tanggal keberangkatan jadwal tanpa memeriksa status booking/pax. Ini sesuai kebijakan ledger tanpa reversal yang tertulis di ketentuan. Penundaan sampai tanggal keberangkatan tidak menghapus risiko pembatalan sebelum berangkat. Perlu keputusan bisnis tersendiri bila hendak mengubah eligibility/reversal; jangan diam-diam mengubah aturan yang sudah dijanjikan.
2. Agen boleh disetujui sebelum membayar biaya pendaftaran, bukti pembayaran pendaftaran disarankan (konfirmasi manual tersedia), dan biaya hangus setelah pengajuan ditolak: sudah merupakan keputusan bisnis. Tidak dihitung sebagai bug.
3. Tidak memakai OTP dan pajak komisi di luar sistem adalah batas MVP yang terdokumentasi. Tidak diklaim telah memenuhi persyaratan regulasi tertentu.

## Pemeriksaan aktual dan batas end-to-end

HTTP lokal, tanpa autentikasi:

```text
GET http://localhost:9090/api/health
200 {"status":"ok","database":"connected"}

GET http://localhost:9090/api/portal/agen
404 404 page not found

GET http://localhost:9090/api/portal/agen/dashboard
404 404 page not found

GET http://localhost:9090/api/admin/pencairan
401 {"error":"unauthorized"}
```

**L01 — lingkungan belum selaras dengan sumber:** route portal agen ada di `cmd/api/main.go:232–245` dan middleware portal seharusnya menghasilkan 401 bagi request tanpa token. API lokal justru menghasilkan 404 pada kedua route agen. Ini membuktikan endpoint belum tersedia di proses aktif, bukan bahwa database terputus. Versi binary/proses perlu diselaraskan sebelum acceptance test; tidak direstart selama audit ini.

Browser berhasil membuka `/daftar-agen` yang redirect ke `/portal/daftar-agen`, serta halaman ketentuan. Pada viewport 375: scrollWidth=360; pada 1440: scrollWidth=1425. Tidak ditemukan overflow horizontal pada pendaftaran awal; seluruh empat input punya label dan required. Teks captcha “nonaktif (mode pengembangan)” terlihat; ini observasi lingkungan development, bukan bukti captcha production dapat dilewati. Screenshot kedua ukuran diperiksa melalui alat browser. Tidak melakukan signup, mengisi PIN, menerima ketentuan, atau mengirim transaksi.

Probe yang disiapkan:

- `internal/komisi/audit_agen_test.go`: perubahan repeat setelah pembatalan historis; komisi setelah aktivasi ulang; risiko saldo sumber batal.
- `internal/agen/audit_agen_test.go`: rekening tanpa digit; URL foto rekaan.

Perintah yang belum berhasil dijalankan:

```powershell
$env:ERP_TEST_DB='1'
go test -p 1 -v ./internal/komisi ./internal/agen -run TestAuditAgen -count=1
```

Review persetujuan otomatis gagal akibat batas penggunaan layanan sebelum perintah dijalankan. Ini bukan penilaian bahwa tes tidak aman. Pembatasan tidak dilewati. File probe sudah diformat dengan gofmt, tetapi belum ada klaim kompilasi/lulus tes. Probe sengaja mengharapkan perilaku bermasalah untuk diagnosis; setelah perbaikan, ubah menjadi regression test yang mengharapkan perilaku benar.

## Urutan tindak lanjut

1. Selaraskan backend aktif dengan sumber; pastikan migrasi agen 060–064 dan konfigurasi captcha sesuai lingkungan tanpa mengaktifkan bypass production.
2. Perbaiki AG01–AG03, kemudian jalankan probe serta pengujian lintas brand dan pelunasan ulang.
3. Tutup kontrak upload, quotation biaya, persetujuan versi, dan validasi rekening (AG04–AG06, AG09–AG10).
4. Perbaiki recovery/UI (AG07–AG08, AG11–AG14).
5. Acceptance test dengan akun uji terkontrol: daftar → ajukan → verifikasi/penolakan → aktif/nonaktif → referral → booking → lunas → repeat → cashback → pencairan pending/disetujui/ditolak; cek data parsial setelah rollback dan konkurensi pengajuan/persetujuan.

Audit kode dan permukaan publik selesai. Validasi transaksi database serta browser yang membutuhkan akun agen/admin masih terbuka; jangan menyebut seluruh alur operasional telah lulus.
