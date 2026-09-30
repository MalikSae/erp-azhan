# Audit End-to-End Peran Jamaah — 28 September 2026

## Kesimpulan dan batas verifikasi

Ditemukan **24 temuan: 5 P1 dan 19 P2** pada alur akun, pembayaran, dokumen, perjalanan, profil, dan cashback. P1 berarti prioritas tinggi untuk kerahasiaan, pemulihan akun, atau informasi yang dapat mendorong transfer tidak sesuai kondisi reservasi. P2 berarti gangguan fungsi, integritas data, atau UX yang perlu diperbaiki.

Audit mencakup kode backend ERP dan frontend microsite pada branch `dev-malik`, termasuk perubahan lokal dari pekerjaan sebelumnya. Tidak ada kode aplikasi, skema, akun, atau data transaksi permanen yang diubah. Tambahan audit hanya laporan ini dan dua skrip probe menggunakan data sintetis.

**Ini audit alur menyeluruh melalui kode, probe terisolasi, dan pemeriksaan publik; bukan pernyataan bahwa transaksi E2E terautentikasi sudah lulus.** Login akun nyata, aktivasi/reset PIN, unggah dokumen nyata, transfer, persetujuan admin, dan rollback database tidak dieksekusi. Respons 401 tanpa token membuktikan adanya gate autentikasi, bukan kebenaran seluruh pemeriksaan kepemilikan objek. Backend aktif juga tidak sepenuhnya sama dengan sumber: route agen mengembalikan 404 walaupun tersedia dalam kode.

Acuan: aturan repo, model dan route saat ini, `ANALISIS-portal-jamaah.md`, `ANALISIS-portal-jamaah-frontend.md`, serta laporan paket/self-booking/agen sebelumnya. Dua dokumen analisis portal bersifat historis; klaim lamanya diperiksa ulang terhadap kode sekarang. File `analisis-modul-booking-jamaah.md` yang dirujuk aturan repo tidak ditemukan melalui pencarian nama file. Ketentuan visa/vaksin tidak diaudit sebagai kepatuhan regulasi; temuan bisnis di sini menilai konsistensi kontrak aplikasi.

## Peta perjalanan dan hak akses

| Tahap | Perilaku sumber saat ini | Hasil audit |
|---|---|---|
| Pilih paket dan self-booking | Data brand, snapshot checkout, PIN PIC, hold kursi, token invoice | Perbaikan sebelumnya tersedia dalam sumber; portal masih mengabaikan sebagian kontrak checkout |
| Aktivasi anggota/admin | Token acak 32 byte, hash SHA-256, masa berlaku 24 jam, tanggal lahir, PIN bcrypt | Pemakaian token memakai transaksi; penerbitan token dan pencabutan sesi masih bermasalah |
| Login | Brand + WhatsApp/ID jamaah + PIN; limiter IP dan akun | Tidak ada fallback PIN default pada kode sekarang |
| Membaca booking | PIC atau jamaah yang tercatat dalam booking_pax | Anggota bisa membaca; akses tulis pembayaran tetap hanya PIC |
| Pembayaran | Konfirmasi portal pending, rekening sesuai brand, pengecekan tagihan dan hold | Backend memiliki guard; pemilihan booking, hak tombol, deadline, dan status hold di UI belum sesuai |
| Dokumen | Daftar dan upsert hanya untuk ID jamaah dari token | URL file belum dibatasi kepemilikannya; preview dan siklus verifikasi bermasalah |
| Perjalanan | Header booking mengambil progress primary pax; detail memiliki progress per pax | UI menggunakan header sehingga tidak mewakili anggota yang login |
| Profil dan bantuan | Data dibaca; koreksi melalui CS | Mapping paspor salah; DTO admin membocorkan catatan internal |
| Cashback dan pascaperjalanan | Kredit dibentuk backend dan bisa diterapkan admin | Jamaah biasa tidak punya informasi saldo/history kredit |

## Temuan prioritas tinggi

### JM01 — P1 — Catatan internal petugas dikirim melalui profil jamaah

**Bukti kode:** `internal/portal/handler.go:294-311` mengembalikan seluruh hasil `jamaahRepo.GetByID`. `internal/jamaah/model.go` menyertakan `catatan`. `frontend/shared/src/pages/JamaahFormPage.jsx:921-937` secara eksplisit melabeli field sebagai internal dan menyatakan hanya dapat dilihat petugas biro travel.

**Dampak:** jamaah dapat membaca catatan internal tentang dirinya melalui respons API meskipun halaman profil tidak menampilkannya. UI tersembunyi tidak membatasi payload. Ini bukan temuan kebocoran catatan jamaah lain.

**Perbaikan:** gunakan DTO profil portal dengan whitelist field; pisahkan informasi yang memang boleh dibagikan dan catatan petugas.

**Uji penerimaan:** catatan sintetis tersedia pada respons admin berizin, tetapi tidak muncul dalam JSON `/portal/me` maupun payload portal lain.

### JM02 — P1 — Reset PIN tidak mencabut token portal lama

**Bukti kode:** `internal/portal/activation.go:438-445` hanya memperbarui hash PIN dan menandai token aktivasi terpakai. `internal/identity/jwt.go:142-190` menghasilkan token portal 24 jam dan memvalidasi signature/type/expiry tanpa session version atau pemeriksaan pencabutan. `RequirePortalAuth` tidak memeriksa perubahan PIN. Logout frontend hanya membersihkan penyimpanan lokal.

**Dampak:** token yang diperoleh sebelum pemulihan akun tetap dapat digunakan sampai kedaluwarsa, termasuk untuk akses dokumen dan operasi portal yang diizinkan. Mengganti PIN belum menghentikan sesi yang sudah bocor.

**Perbaikan:** persist session version atau sesi portal yang dapat dicabut; reset PIN mencabut seluruh sesi lama secara atomik. Sediakan pencabutan server-side untuk logout sesuai kebijakan sesi.

**Uji penerimaan:** token A valid sebelum reset; setelah reset A mendapat 401, sedangkan login menggunakan PIN baru menghasilkan sesi valid. Jangan mencetak token nyata ke laporan.

### JM03 — P1 — PWA menyimpan invoice pribadi dan menyajikannya saat offline

**Bukti kode dan probe:** `azhan-microsite/public/sw.js:31-39` memasukkan seluruh navigasi same-origin ke Cache Storage tanpa pengecualian invoice atau pemeriksaan `Cache-Control`. Halaman `src/app/invoice/[code]/page.jsx` merender data invoice server-side. Worker didaftarkan pada layout global. Logout `PortalAuthContext.jsx:19-26` tidak membersihkan cache tersebut.

**Reproduksi terisolasi:** jalankan `node audit/jamaah-pwa-probe.cjs`. Skrip menjalankan worker asli dengan respons invoice sintetis `private, no-store`, kemudian membuat fetch sintetis offline. Invoice tetap disimpan dan dikembalikan.

**Dampak:** pada origin yang mendukung service worker, invoice yang pernah dibuka dapat bertahan di profil browser bersama dan tersedia offline. Juga berisiko menampilkan status pembayaran lama. Probe tidak membuktikan adanya invoice nyata di cache pengguna; tidak dilakukan pembacaan cache nyata.

**Perbaikan:** kecualikan invoice, aktivasi, portal, dan konten privat dari caching worker; migrasikan cache versi lama untuk menghapus entri sensitif. Header HTTP saja tidak memperbaiki logika `cache.put` ini.

**Uji penerimaan:** navigasi invoice tidak menghasilkan entri Cache Storage; invoice lama tidak tersedia offline setelah migrasi cache/logout.

### JM04 — P1 — Batas pelunasan portal berbeda dari kontrak checkout

**Bukti kode:** `src/app/portal/page.jsx:228-236` menghitung tanggal keberangkatan minus 30 hari, lalu menampilkan “Batas Pelunasan” pada baris 424. `internal/selfbooking/repository.go:557-561,632,831,913-914` menyimpan dan membaca `due_at`, dengan aturan H-45 atau full payment mengikuti checkout. Model booking portal belum membawa kontrak checkout ini.

**Dampak:** jamaah dapat mengikuti batas yang 15 hari lebih lambat dari invoice. Pemesanan dekat keberangkatan juga tidak mengikuti deadline full payment jika UI menghitung ulang H-30.

**Perbaikan:** kirim deadline dan aturan pembayaran dari snapshot checkout melalui DTO portal. Dashboard, pembayaran, dan invoice menggunakan sumber yang sama.

**Uji penerimaan:** booking jauh/dekat keberangkatan, DP nol, dan perubahan kebijakan brand setelah checkout tetap menampilkan satu deadline yang sama.

### JM05 — P1 — Portal tetap mengajak transfer ketika hold kursi sudah tidak berlaku

**Bukti kode:** `src/app/portal/pembayaran/page.jsx:68-74,340-367` hanya menyaring status batal/draft dan menggunakan `!isLunas` untuk tombol konfirmasi serta WA. Rekening tetap ditampilkan. Tidak ada pemeriksaan `is_seat_blocked`/`seat_hold_expires_at` di halaman ini. `internal/payment/repository.go:185-190` menolak pembayaran portal saat kursi tidak terkunci atau hold kedaluwarsa.

**Dampak:** booking `baru` yang kursinya dilepas masih tampak dapat dibayar. Jamaah bisa melakukan transfer bank nyata sebelum aplikasi menolak pengiriman bukti. Guard backend melindungi pencatatan, tetapi tidak menarik kembali uang yang sudah ditransfer.

**Perbaikan:** gunakan status reservasi yang konsisten dengan invoice; tampilkan kedaluwarsa/perlu konfirmasi CS dan sembunyikan ajakan transfer saat reservasi tidak dapat dibayar. Lakukan validasi ulang sebelum memberi instruksi pembayaran.

**Uji penerimaan:** hold aktif, kedaluwarsa sebelum worker, dilepas worker, permanen, dan booking batal menghasilkan aksi pembayaran yang sesuai.

## Temuan prioritas sedang

### JM06 — P2 — Jamaah tidak dapat membuka preview dokumennya sendiri

**Bukti:** `internal/media/handler.go` mengembalikan `/api/admin/media/dokumen-jamaah/{file}` untuk upload portal. Route tersebut memakai autentikasi admin (`cmd/api/main.go:252,330`). `src/app/portal/perjalanan/page.jsx:705-715` membuka URL langsung melalui `iframe`, `img`, atau tab baru tanpa bearer token. Tidak ada route baca dokumen pribadi di grup portal.

**Dampak:** unggah/metadata dapat berhasil, tetapi preview gagal. Bahkan menambahkan token portal ke route admin tidak menyelesaikan perbedaan tipe token.

**Perbaikan/uji:** endpoint baca berdasarkan ID dokumen yang memeriksa pemilik; frontend fetch sebagai blob. Pemilik dapat membuka JPG/PDF, jamaah lain tetap ditolak. Jangan membuka kembali folder dokumen secara publik.

### JM07 — P2 — Referensi dokumen dan bukti transfer tidak diverifikasi sebagai hasil upload milik pengirim

**Bukti:** `internal/portal/handler.go:485,570-598` memeriksa bukti/URL tidak kosong dan jenis dokumen, tanpa memeriksa keberadaan file, asal upload, pemilik, atau konteks penggunaannya. `dokumen.Repository.Upsert` dan `payment.Repository.Create` menyimpan string tersebut. Upload media belum merekam ownership.

**Dampak:** payload API dapat menyimpan URL palsu, URL eksternal, atau referensi file orang lain yang diketahui. Ini merusak integritas verifikasi; pembayaran tetap pending, bukan otomatis menjadi lunas.

**Perbaikan/uji:** upload ID dengan metadata pemilik/brand/jenis; validasi referensi sebelum menyimpan. Tolak URL eksternal, file tidak ada, dan file pengguna lain. Beririsan dengan AG04 pada audit agen, tetapi endpoint yang diperiksa di sini adalah dokumen dan pembayaran jamaah.

### JM08 — P2 — Pembayaran hanya menampilkan booking pertama

**Bukti:** `src/app/portal/pembayaran/page.jsx:72` memilih `activeBookings[0]`. Tidak ada pemilih booking atau parameter booking. Repository mengurutkan berdasarkan waktu pembuatan (`internal/booking/repository.go:145`). Dashboard memilih keberangkatan terdekat dan menuju `/portal/pembayaran` tanpa ID (`portal/page.jsx:384`).

**Dampak:** pengguna membuka pembayaran untuk perjalanan A tetapi memperoleh booking B yang dibuat lebih baru. Tagihan booking lain tidak dapat dipilih dari halaman pembayaran.

**Perbaikan/uji:** pertahankan ID booking pada navigasi, validasi akses di backend, dan sediakan pemilih booking. Uji dua booking aktif dengan urutan pembuatan berbeda dari urutan keberangkatan.

### JM09 — P2 — Tombol bayar dan invoice ditawarkan kepada anggota non-PIC yang tidak berhak

**Bukti:** pembacaan booking/payments mengizinkan PIC atau anggota (`portal/handler.go:337,406`), sedangkan `CreatePayment:475` hanya PIC. `selfbooking/invoice_link.go:14` hanya PIC atau agen aktif terkait. Halaman pembayaran menampilkan kedua aksi tanpa pemeriksaan peran terhadap booking.

**Dampak:** anggota mengisi form dan mengunggah file sebelum mendapat 404 ketika konfirmasi. Invoice juga berakhir 404; handler UI bahkan membuka modal pembayaran saat pengambilan invoice gagal.

**Perbaikan/uji:** ekspos capability `can_submit_payment` dan `can_view_invoice` atau derive dari kontrak yang jelas; tampilkan mode baca dan kontak PIC. Jangan memperluas hak non-PIC tanpa keputusan bisnis. Uji PIC, anggota, agen terkait, dan akun tidak terkait.

### JM10 — P2 — Progress visa/manasik yang dilihat anggota berasal dari primary pax

**Bukti:** `internal/booking/repository.go:87-108` mengambil progress dari satu pax, dengan prioritas PIC. Dashboard `portal/page.jsx:189-205` dan perjalanan `perjalanan/page.jsx:232-254` memakai progress header tersebut. Detail sebenarnya memiliki progress per pax, tetapi daftar rombongan hanya menampilkan nama/tipe/kamar dan badge “Terdaftar”.

**Dampak:** anggota B dapat melihat visa selesai ketika hanya PIC A yang selesai, atau sebaliknya. Badge perjalanan tidak membedakan progress individu dan kesiapan seluruh booking.

**Perbaikan/uji:** gunakan pax milik jamaah yang login untuk status individu; tampilkan agregat semua pax hanya dengan label rombongan. Uji dua pax dengan progress berlawanan dan infant yang tidak wajib manasik.

### JM11 — P2 — Nomor paspor profil selalu kosong untuk kontrak API sekarang

**Bukti kode dan probe:** `profil/page.jsx:43,115` membaca `nomor_paspor`/`paspor_nomor`; API `jamaah/model.go` mengirim `no_paspor`. Probe menjalankan ekspresi asli dengan `{no_paspor: 'SYNTHETIC-PASSPORT'}` dan memperoleh `-`.

**Dampak:** jamaah mengira data paspornya belum dilengkapi dan menghubungi CS tanpa kebutuhan.

**Perbaikan/uji:** gunakan `no_paspor`; fixture terisi tampil nilainya, NULL tampil keterangan belum tersedia.

### JM12 — P2 — Status perlengkapan membaca field dan nilai yang tidak ada

**Bukti:** `perjalanan/page.jsx:609-629` membaca `booking.perlengkapan_status === 'lengkap'`. Model `internal/booking/model.go` menempatkan status pada `BookingPax`, dengan nilai `belum_diberikan`/`sudah_diberikan`, bukan header booking atau enum `lengkap`.

**Dampak:** perlengkapan tetap tampil “Belum Diambil” meski sudah diserahkan dan dicatat petugas.

**Perbaikan/uji:** gunakan pax yang sesuai dan mapping enum resmi; tampilkan tanggal penyerahan. Uji individu diterima/belum diterima dalam booking yang sama.

### JM13 — P2 — Kegagalan API ditampilkan sebagai data kosong atau tagihan yang belum dibayar

**Bukti:** `pembayaran/page.jsx:64,78-82,201-220` menelan error rekening/payments dan menyamakan `hasError` dengan tidak ada tagihan. Jumlah dibayar dihitung dari array yang bisa kosong akibat request gagal. `perjalanan/page.jsx:114-117` mengubah kegagalan dokumen/booking menjadi `[]`. Dashboard juga menelan error pembayaran. Aktivasi `aktivasi/page.jsx:43-51` menyamakan error jaringan dengan token invalid.

**Dampak:** pengguna bisa melihat “Belum Ada Tagihan Aktif”, dokumen belum diunggah, atau sisa tagihan penuh ketika server gagal; status tersebut tidak faktual. Instruksi meminta link aktivasi baru juga bisa salah.

**Perbaikan/uji:** pisahkan loading/error/empty/success per sumber data; jangan menghitung jumlah uang dari data gagal dimuat. Tambahkan retry. Uji 401, 404, 500, respons non-JSON, dan offline secara terpisah.

### JM14 — P2 — Kelengkapan dokumen selalu berpenyebut tujuh meski dua opsional

**Bukti:** `perjalanan/page.jsx` menandai buku nikah dan akta lahir opsional, tetapi badge baris 357 tetap `/7`. Dashboard `portal/page.jsx:203-218,779-788` juga menghitung tujuh jenis untuk seluruh akun.

**Dampak:** jamaah dewasa yang tidak menikah dapat memenuhi seluruh dokumen wajib tetapi tetap terlihat belum lengkap. Anak juga diberi daftar kewajiban generik tanpa profil applicability.

**Perbaikan/uji:** bedakan dokumen wajib, opsional, dan tidak berlaku berdasarkan kebijakan travel/profil. Jangan menganggap semua aturan dokumen merupakan kewajiban regulasi universal. Uji dewasa belum menikah, pasangan, anak, dan infant.

### JM15 — P2 — Persetujuan dokumen tidak terikat pada versi file yang ditinjau

**Bukti:** `dokumen/repository.go:91-104` mengganti URL pada row yang sama. `UpdateStatus:128-133` hanya menggunakan ID; request `dokumen/model.go:22` hanya berisi status. Tidak ada syarat versi/URL/updated_at dalam approval.

**Reproduksi dari kode:** admin membuka dokumen versi A; jamaah mengganti menjadi B; admin menyetujui ID yang sama. Versi B dapat menjadi approved meski admin meninjau A. Race ini belum dieksekusi pada DB.

**Perbaikan/uji:** simpan versi dokumen atau optimistic concurrency dengan version ID; approval terhadap versi lama mendapat 409. Uji upsert dan approval yang saling bersilangan.

### JM16 — P2 — Penolakan dokumen tidak menyertakan alasan perbaikan

**Bukti:** model dokumen tidak memuat rejection reason; request perubahan status hanya status. UI `perjalanan/page.jsx` menampilkan “Perlu Diperbaiki” tanpa alasan.

**Dampak:** jamaah tidak tahu apakah masalahnya kabur, terpotong, identitas salah, atau masa berlaku. Mengunggah ulang tanpa arahan menambah antrean petugas.

**Perbaikan/uji:** alasan wajib saat rejected, tersedia pada portal; riwayat versi tetap dipertahankan. Alasan terakhir harus dikaitkan dengan versi dokumen, bukan dibawa tanpa konteks ke file pengganti.

### JM17 — P2 — Flag kesiapan paspor mengabaikan status rejected

**Bukti:** `booking/repository.go:1721-1754` menganggap paspor tersedia selama `file_url` tidak kosong. Nilai dipakai dalam perhitungan `siap_berangkat` seluruh pax tanpa syarat dokumen approved.

**Dampak:** dokumen paspor yang ditolak masih memenuhi syarat paspor pada flag kesiapan backend. Ini berbeda dari checklist portal yang menghitung approved. Masalah ini tidak membuktikan petugas benar-benar memberangkatkan jamaah dengan dokumen tidak sah.

**Perbaikan/uji:** bedakan flag “sudah upload” dan “terverifikasi siap”. Gunakan approved untuk syarat kesiapan yang membutuhkan verifikasi. Uji submitted, rejected, approved, dan penggantian dokumen setelah approved.

### JM18 — P2 — Gangguan server menghapus sesi; expiry saat tab terbuka tidak dipulihkan

**Bukti:** `PortalAuthContext.jsx:55-60` memanggil logout untuk seluruh kegagalan `getMe`, termasuk offline/500. `portalApi.js` tidak memiliki penanganan 401 bersama. Pemeriksaan expiry dilakukan saat provider mount, bukan saat setiap respons gagal. Akses `localStorage.getItem/removeItem` juga tidak terlindungi dari storage exception.

**Dampak:** gangguan sementara memaksa login ulang; token yang habis saat tab terbuka meninggalkan UI tampak login dengan request gagal. Dalam kondisi storage diblokir, state loading/logout dapat tidak selesai.

**Perbaikan/uji:** klasifikasi kegagalan auth vs jaringan; arahkan 401 ke login dengan return path, pertahankan sesi saat gangguan sementara, tangani storage tidak tersedia. Uji expiry saat submit, offline saat reload, dan storage exception.

### JM19 — P2 — Penerbitan link aktivasi tidak atomik

**Bukti:** `portal/activation.go:79-113` membatalkan token lama dan membuat token baru melalui dua operasi DB terpisah tanpa transaksi atau lock per jamaah.

**Dampak:** kegagalan insert setelah invalidasi membuat link lama mati tanpa link baru. Dua penerbitan bersamaan dapat sama-sama melewati invalidasi lalu menyimpan dua link aktif. Pemakaian link sudah memakai transaksi, tetapi tidak mencabut sibling token tersebut.

**Perbaikan/uji:** transaksi dengan lock jamaah saat penerbitan; batalkan token sibling ketika aktivasi berhasil. Uji penerbitan bersamaan dan kegagalan insert; tepat satu link terbaru yang berlaku dan rollback tidak mematikan link lama.

### JM20 — P2 — Link aktivasi default memakai HTTP untuk domain tanpa skema

**Bukti:** `portal/activation.go:123-126` menambahkan `http://` ketika kolom domain tidak berisi skema, termasuk domain non-lokal. Tidak ada pemisahan kebijakan domain dev/production.

**Dampak:** tautan pemulihan membawa token pada URL HTTP. Risiko transmisi awal bergantung pada HSTS/infrastruktur browser; konfigurasi production tidak diverifikasi dalam audit ini.

**Perbaikan/uji:** gunakan HTTPS untuk domain publik, HTTP hanya hostname pengembangan yang diizinkan. Uji domain polos production, domain dengan skema, alias brand, serta `.test` lokal.

### JM21 — P2 — File bukti lama tetap terkirim setelah file pengganti ditolak

**Bukti kode dan probe:** `pembayaran/page.jsx:96-110` melakukan return saat ukuran file lebih dari 5 MB tanpa mengosongkan `fileBukti`/preview. Submit baris 170 menggunakan state tersebut, bukan pilihan file terbaru di input.

**Reproduksi:** pilih A yang valid, ganti dengan B sebesar 6 MB. Probe menjalankan handler asli: error tampil, tetapi state masih A. Tombol submit tidak diblokir oleh error file itu; A dapat diunggah ketika pengguna mengira memilih B.

**Perbaikan/uji:** kosongkan state/file input pada pilihan invalid dan batasi submit sampai file valid; render PDF sebagai PDF/nama berkas, bukan selalu `<img>`. Uji penggantian valid-invalid-valid serta pembatalan pemilihan.

### JM22 — P2 — Tanggal pembayaran memakai UTC dan tidak meminta tanggal transfer aktual

**Bukti:** `pembayaran/page.jsx:171-180` mengirim `new Date().toISOString().split('T')[0]`; form tidak menyediakan tanggal transfer.

**Dampak:** di Asia/Jakarta pukul 00.00–06.59, tanggal kiriman adalah hari sebelumnya. Bukti transfer dari hari lain juga diberi tanggal saat submit. Ini mengganggu pencocokan mutasi dan riwayat transaksi.

**Perbaikan/uji:** tanggal transfer diisi pengguna dengan default zona bisnis; timestamp submit dicatat terpisah. Uji pukul 00.30 WIB, transfer kemarin, dan tanggal tidak valid.

### JM23 — P2 — Modal pembayaran/preview belum menyediakan interaksi aksesibel

**Bukti:** modal pembayaran `pembayaran/page.jsx:448` dan preview `perjalanan/page.jsx:690` berupa div overlay tanpa role dialog, aria-modal, pengelolaan fokus, pemulihan fokus, atau Escape. Label nominal/pengirim/bukti di form tidak dihubungkan ke input melalui htmlFor/id. Kontrol tutup memakai karakter silang tanpa nama deskriptif.

**Dampak:** pengguna keyboard/pembaca layar tidak mendapat konteks dialog dan dapat berpindah ke kontrol belakang overlay. Ini berasal dari pemeriksaan kode; uji pembaca layar belum dilakukan. Form login berbeda: dua labelnya terhubung dengan benar pada DOM yang diperiksa.

**Perbaikan/uji:** gunakan pola dialog yang sudah tersedia di repo, label input terhubung, error terkait field dan diumumkan. Uji Tab/Shift+Tab/Escape dan fokus kembali ke pemicu pada layar kecil.

### JM24 — P2 — Cashback jamaah biasa tidak terlihat pada portal

**Bukti:** `internal/komisi/kalkulasi.go` membentuk kredit cashback untuk jamaah repeat. UI saldo berada di `portal/syiar/DashboardAgen.jsx:42-50`; backend dashboard agen mensyaratkan agen aktif (`internal/agen/kinerja.go:163-175`). Profil/dashboard jamaah biasa tidak memiliki saldo atau history cashback.

**Dampak:** jamaah tidak mengetahui hak kredit dan penggunaannya pada booking berikutnya. Penerapan oleh admin sudah tersedia melalui modul cashback; masalahnya bukan seluruh fitur cashback tidak ada.

**Perbaikan/uji:** tampilkan saldo, mutasi, dan cara meminta penerapan kredit bagi pemilik, termasuk jamaah nonagen. Kredit tidak berubah menjadi pencairan uang tanpa keputusan bisnis. **Temuan ini sama dengan AG11 pada audit agen; jangan dihitung dua kali dalam backlog gabungan.**

## Keputusan bisnis dan risiko lanjutan yang tidak dihitung sebagai bug tambahan

1. **Anggota non-PIC:** hak baca booking saat ini luas, tetapi tulis pembayaran hanya PIC. Pertahankan pembatasan sambil memperbaiki affordance JM09, kecuali pemilik bisnis memutuskan delegasi eksplisit. Satu booking tidak otomatis berarti seluruh pihak boleh bertindak atas keuangan pihak lain.
2. **Anak/infant dan jamaah yang dibantu keluarga:** dokumen portal milik akun sendiri. Aktivasi membutuhkan tanggal lahir yang dilengkapi admin. Kebutuhan pendamping/guardian perlu aturan izin dan jejak audit; jangan menghapus isolasi dokumen dengan memberi seluruh PIC akses tanpa batas.
3. **Pembatalan dan refund:** halaman portal menyaring booking batal, sehingga tidak menawarkan riwayat pembatalan/penyelesaian refund. Mekanisme refund dan SLA-nya perlu keputusan bisnis sebelum ditambahkan; laporan ini tidak menyimpulkan refund boleh/harus otomatis.
4. **Komisi dan cashback setelah pembatalan:** keputusan no-reversal yang tercatat pada audit agen tetap merupakan risiko bisnis tersendiri. Jangan diam-diam mengubah ledger saat memperbaiki portal.
5. **Informasi perjalanan operasional:** DTO booking portal belum membawa detail maskapai/hotel yang dicoba dibaca UI; copy generik kadang menyebut “terkonfirmasi” sementara badge masih “Diproses”. Jadwal manasik, titik kumpul, dan akses dokumen perjalanan final membutuhkan penetapan sumber data serta penanggung jawab. Hindari janji layanan yang tidak didukung data.
6. **Identitas tanpa OTP dan reset lewat admin:** ini desain MVP yang sudah ada, bukan alasan untuk memakai PIN default. Akun tanpa hash PIN sudah ditolak pada sumber sekarang. Rate limiter masih in-memory; kebutuhan persistensi lintas restart/multi-instance perlu disesuaikan deployment.

## Kontrol positif yang ditemukan dalam sumber

- Portal dan admin memakai tipe JWT berbeda; portal tidak otomatis memperoleh hak admin.
- Login scope brand, normalisasi nomor, bcrypt, serta limiter IP dan akun tersedia. Tidak ada login PIN default untuk akun tanpa PIN pada kode sekarang.
- Token aktivasi memakai randomness kriptografis dan penyimpanan hash; konsumsi link memakai transaksi dan lock.
- Daftar/detail booking memeriksa relasi jamaah; detail draft ditolak. Pembayaran baru memeriksa PIC dan rekening tujuan sesuai brand.
- Pembayaran portal dipaksa `source=portal` dan masuk pending. Repository melakukan lock booking, menolak booking batal/draft/hold invalid, dan memeriksa jumlah terhadap sisa pembayaran confirmed.
- Dokumen di-upsert atas ID jamaah dari token; pengguna tidak bebas menentukan jamaah_id tujuan melalui payload.
- Format upload diperiksa melalui magic bytes, nama file acak, dan kategori portal dipaksa. Masalah ownership/baca privat tetap dicatat pada JM06–JM07.
- Kontrak checkout dan token invoice berentropi tinggi sudah tersedia dari perbaikan sebelumnya; JM04–JM05 menandai portal yang belum mengikuti kontrak itu.

Kontrol di atas berasal dari inspeksi kode, bukan klaim bahwa seluruhnya telah diuji dengan akun nyata pada proses aktif.

## Bukti pengujian yang benar-benar dijalankan

### HTTP lokal tanpa kredensial

Target `http://localhost:9090`, tanggal audit. Body di bawah adalah body respons yang diterima; newline akhir tidak ditampilkan.

```text
GET /api/health
200
{"status":"ok","database":"connected"}

GET /api/portal/me
401
{"error":"unauthorized"}

GET /api/portal/bookings
401
{"error":"unauthorized"}

GET /api/portal/dokumen
401
{"error":"unauthorized"}

GET /api/portal/bank-accounts
401
{"error":"unauthorized"}

GET /api/portal/agen
404
404 page not found

GET /api/admin/media/dokumen-jamaah/audit-missing.jpg
401
{"error":"unauthorized"}
```

URL media memakai nama file sintetis yang tidak ada. Respons membuktikan request tanpa token dihentikan gate admin, tidak membuktikan file ada atau bahwa suatu dokumen nyata telah dibaca. 404 agen menunjukkan perbedaan sumber/runtime; penyebab proses lama belum dikonfirmasi dengan restart.

### Probe worker dengan data sintetis — exit code 0

```text
node audit/jamaah-pwa-probe.cjs
OBSERVED: invoice navigation cached despite Cache-Control: private, no-store
OBSERVED: cached private invoice returned while offline
Scope: existing service-worker logic, synthetic fixture; no real browser cache, API, or DB modified
```

### Probe ekspresi/handler UI asli — exit code 0

```text
node audit/jamaah-ui-probes.cjs
OBSERVED: populated API field no_paspor renders as "-"
OBSERVED: rejected 6 MB replacement leaves the previous payment proof selected
Scope: current source expressions with synthetic fixtures; no HTTP, upload, account, or DB mutation
```

Probe tersebut sengaja mengonfirmasi perilaku bermasalah saat ini. Exit code 0 **bukan** tanda aplikasi sudah benar; setelah perbaikan, ubah assertion menjadi ekspektasi perilaku yang diinginkan.

### Browser

- `http://hana.azhan.test/portal/login` diperiksa pada viewport 375×812 dan 1440×900 melalui browser; screenshot ditinjau di sesi tool, tidak disimpan sebagai file laporan.
- Pada 375 px, DOM melaporkan `innerWidth=375`, `documentElement.scrollWidth=360`; tidak ditemukan overflow horizontal pada login. Input `identifier` dan `pin` memiliki label terhubung dan atribut required. Screenshot desktop memperlihatkan form terpusat tanpa pemotongan horizontal.
- `/portal/aktivasi` tanpa token menampilkan: `Link aktivasi tidak valid atau sudah kedaluwarsa. Hubungi admin travel Anda untuk mendapatkan link baru.` Tautan kembali ke login tersedia.
- Tidak dilakukan login, perubahan PIN, aktivasi akun, unggah, penerimaan syarat, atau pembayaran. Layout halaman terautentikasi belum diverifikasi langsung; temuan di halaman tersebut memakai inspeksi sumber/probe.
- Viewport sementara sudah dikembalikan. Timeout satu pembacaan DOM desktop dipulihkan menggunakan accessibility tree/screenshot; tidak dihitung sebagai bug aplikasi.

## Urutan perbaikan yang disarankan

Pembaruan 29 September 2026: implementasi dan bukti pengujian perbaikan dicatat pada `PERBAIKAN-PERAN-JAMAAH-2026-09-29.md`. Temuan di atas merupakan keadaan saat audit awal. Probe PWA dan UI telah diubah menjadi pemeriksaan regresi perilaku yang diharapkan. Migrasi permanen 066 dan verifikasi server setelah restart masih belum dilakukan.

1. Tutup paparan data dan pemulihan akun: JM01–JM03, lalu validasi ownership media JM06–JM07.
2. Satukan kontrak tagihan/reservasi: JM04–JM05, JM08–JM09, dan status error finansial JM13.
3. Benahi status individu dan siklus dokumen: JM10–JM12, JM14–JM17.
4. Selesaikan sesi/aktivasi, form pembayaran, dan aksesibilitas: JM18–JM23.
5. Tampilkan cashback jamaah JM24, dengan deduplikasi AG11 pada backlog sebelumnya.

Sebelum menutup audit sebagai E2E lulus, jalankan fixture terisolasi PIC, anggota reguler, infant/pendamping, nonagen penerima cashback, agen terkait, serta akun brand lain. Cakup dua booking aktif, hold kedaluwarsa, pembayaran confirmed/pending/rejected, pergantian dokumen saat approval, reset PIN dengan sesi lama, dan offline PWA. Jalankan melalui backend terbaru yang sesuai sumber, dengan pemeriksaan rollback tidak meninggalkan data parsial.
