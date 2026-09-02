# Laporan Investigasi Komprehensif: Alur & Antarmuka Portal Jamaah Frontend (Read-Only)

> **Dokumen Investigasi Teknis Frontend**  
> **Target Analisis:** Pemetaan alur UI/UX, manajemen state, dan penanganan data multi-pax pada Portal Jamaah & Form Self-Booking (`azhan-microsite`).  
> **Status Pemeriksaan:** Read-Only (100% fakta dari source code frontend aktif di `c:\laragon\www\azhan-microsite`).

---

## 1. Ringkasan Eksekutif

1. **Pembuatan PIN Terpusat pada PIC:** Form pendaftaran mandiri (Self-Booking Wizard) hanya meminta 1 buah PIN 6 digit di Step 3 untuk Pendaftar Utama (PIC / Jamaah 1). Anggota rombongan lainnya didaftarkan tanpa input PIN dan tanpa pembuatan akun portal instan.
2. **Tidak Ada Edukasi Hak Akses ke Pemesan:** UI tidak menampilkan disclaimer ataupun penjelasan bahwa PIN tersebut eksklusif hanya untuk pemesan, maupun fakta bahwa anggota rombongan lain tidak langsung mendapatkan akses portal.
3. **Detail Booking Menyembunyikan Pax Lain:** Pada halaman detail booking portal (`/portal/booking/[id]`), antarmuka hanya me-render nama tunggal PIC (`{booking.nama_jamaah}`). Seluruh array data `booking.pax` diabaikan dan tidak ditampilkan di layar.
4. **Isolasi Dokumen Bersifat Pribadi:** Halaman kelengkapan dokumen (`/portal/dokumen`) hanya menampilkan dan mengunggah dokumen milik profil yang sedang login tanpa opsi melihat/mengunggah dokumen anggota keluarga/rombongan lain.
5. **Autentikasi Terisolasi via Token Tunggal:** `PortalAuthContext` menyimpan token JWT di `localStorage` dengan key `'portal_access_token'`, yang mengikat seluruh request portal ke satu identitas `jamaah_id`.

---

## 2. Form Booking Self-Service (BAGIAN 1)

### 2.1 File dan Rute yang Terlibat
Pendaftaran booking publik di `azhan-microsite` melibatkan berkas-berkas berikut:

| Jenis Berkas | Path Berkas | Peran / Deskripsi |
|---|---|---|
| **Next.js Page Route** | [`src/app/paket/[id]/book/page.jsx`](file:///c:/laragon/www/azhan-microsite/src/app/paket/[id]/book/page.jsx#L30) | Server Component untuk rute `/paket/[id]/book`. Memvalidasi schedule, mengambil daftar rekening bank resmi brand, dan me-render `BookingWizard`. |
| **Next.js API Proxy** | [`src/app/api/public/book/route.js`](file:///c:/laragon/www/azhan-microsite/src/app/api/public/book/route.js#L3) | API Route handler internal (`POST /api/public/book`) yang meneruskan payload booking ke backend Go (`/api/public/book`). |
| **Wizard Component Utama** | [`src/components/booking/BookingWizard.jsx`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L95) | Komponen Client 4-step wizard interaktif: Step 1 (Kamar & Pax), Step 2 (Data Jamaah), Step 3 (Review & Buat PIN), Step 4 (Instruksi Transfer & Akses Portal). |
| **Komponen UI Pendukung** | [`src/components/ui/CustomSelect.jsx`](file:///c:/laragon/www/azhan-microsite/src/components/ui/CustomSelect.jsx)<br/>[`src/components/ui/Turnstile.jsx`](file:///c:/laragon/www/azhan-microsite/src/components/ui/Turnstile.jsx)<br/>[`src/components/ui/Button.jsx`](file:///c:/laragon/www/azhan-microsite/src/components/ui/Button.jsx) | Komponen dropdown gender, verifikasi bot Cloudflare Turnstile, dan tombol standar. |
| **Context Provider** | [`src/context/BrandContext.jsx`](file:///c:/laragon/www/azhan-microsite/src/context/BrandContext.jsx) | Menyediakan identitas brand aktif (`brandId`, `brandName`, `brandColor`). |

---

### 2.2 Kapasitas Jamaah & Pembedaan PIC vs Anggota

#### A. Kapasitas Jamaah per Submit
- Jumlah jamaah **bersifat dinamis** dan dihitung otomatis berdasarkan jumlah kamar yang dipilih di Step 1 ([`BookingWizard.jsx:174`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L174)):
  ```javascript
  const totalPax = (counts.quad || 0) + (counts.triple || 0) + (counts.double || 0) + (counts.infant || 0);
  ```
- Pengguna dapat menambah kuantitas kamar Quad, Triple, Double, maupun Infant secara bebas sesuai ketersediaan kursi (`seat_sisa`).

#### B. Logika Penentuan PIC (Pendaftar Utama)
Penetapan PIC dilakukan secara otomatis berdasarkan urutan kamar teratas yang dipilih ([`BookingWizard.jsx:822, 909, 996, 355-360`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L355-L360)):
1. Jika `counts.quad > 0`, maka **Jamaah 1 Quad** (`idx === 0`) otomatis menjadi PIC (`isPic = true`).
2. Jika `counts.quad === 0` dan `counts.triple > 0`, maka **Jamaah 1 Triple** (`idx === 0`) otomatis menjadi PIC.
3. Jika Quad dan Triple sama-sama 0, maka **Jamaah 1 Double** (`idx === 0`) otomatis menjadi PIC.

#### C. Perbedaan Visual & Field PIC vs Anggota di Step 2
| Atribut Form | Pendaftar Utama (PIC / Kontak Utama) | Anggota Rombongan Lain (Pax #2, #3, dst.) | Referensi Baris |
|---|---|---|---|
| **Label / Badge Kartu** | Memiliki badge: `<span className="... border border-slate-200">KONTAK UTAMA</span>` | Hanya label nomor urut: `"Jamaah 2"`, `"Jamaah 3"` | [`BookingWizard.jsx:832-834`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L832-L834) |
| **Nama Lengkap** | Wajib (`Nama Lengkap *`) | Wajib (`Nama Lengkap *`) | [`BookingWizard.jsx:837, 926`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L837) |
| **Jenis Kelamin** | Wajib (`Jenis Kelamin *`) | Wajib (`Jenis Kelamin *`) | [`BookingWizard.jsx:844, 947`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L844) |
| **Nomor WhatsApp** | **Wajib** (`No. WhatsApp *`) | **Opsional** (`No. WhatsApp (Opsional)`) | [`BookingWizard.jsx:861, 964`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L861) |
| **Email** | **Ditampilkan** (`Email (Opsional)`) | **TIDAK DITAMPILKAN** (Disembunyikan) | [`BookingWizard.jsx:881-894`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L881-L894) |
| **Kategori Khusus Infant** | — | Kartu khusus dengan badge `INFANT (< 2 TAHUN)` dan field wajib `Tanggal Lahir *` | [`BookingWizard.jsx:1100-1175`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L1100-L1175) |

---

### 2.3 Alur, Validasi, & Teks PIN (Step 3)

#### A. Langkah Permintaan PIN
PIN diminta pada **Step 3 (Konfirmasi & Buat PIN)** di dalam Card khusus berjudul *"Buat PIN Akun Portal Jamaah"* ([`BookingWizard.jsx:1302-1411`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L1302-L1411)).

#### B. Aturan Validasi PIN
1. **Panjang Karakter:** Wajib tepat **6 digit** (`maxLength={6}` dan `picPin.length !== 6` di [`BookingWizard.jsx:339, 1321`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L339)).
2. **Karakter Angka Murni:** Sanitasi otomatis menghapus non-digit saat diketik `onChange={(e) => setPicPin(e.target.value.replace(/\D/g, ''))}` dan divalidasi via regex `!/^\d{6}$/.test(picPin)` ([`BookingWizard.jsx:339, 1323`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L339)).
3. **Konfirmasi PIN Cocok:** Nilai `picPin` wajib identik dengan `picPinConfirm` ([`BookingWizard.jsx:343`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L343)).

#### C. Kutipan Teks (Copywriting) Persis di UI
- **Judul Box:**  
  `"Buat PIN Akun Portal Jamaah"` ([`BookingWizard.jsx:1306`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L1306))
- **Deskripsi Box:**  
  `"Akses informasi perjalanan, pembayaran, visa, tiket, manasik, dan persiapan umroh Anda dalam satu tempat."` ([`BookingWizard.jsx:1309`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L1309))
- **Label Input 1:**  
  `"Buat 6 Digit PIN *"` ([`BookingWizard.jsx:1316`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L1316))
- **Placeholder Input 1:**  
  `"Contoh: 123456"` ([`BookingWizard.jsx:1324`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L1324))
- **Label Input 2:**  
  `"Konfirmasi 6 Digit PIN *"` ([`BookingWizard.jsx:1348`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L1348))
- **Placeholder Input 2:**  
  `"Ulangi 6 digit PIN"` ([`BookingWizard.jsx:1364`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L1364))
- **Teks Indikator Validasi Realtime:**  
  - Sesuai: `"PIN sesuai"` (Warna hijau, icon checklist) ([`BookingWizard.jsx:1398`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L1398))  
  - Tidak Sesuai: `"Konfirmasi PIN tidak sesuai"` (Warna merah, icon silang) ([`BookingWizard.jsx:1406`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L1406))
- **Pesan Alert Modal/Toast Jika Gagal Submit:**  
  - `"PIN Portal Jamaah wajib 6 digit angka."` ([`BookingWizard.jsx:340`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L340))  
  - `"Konfirmasi PIN tidak sesuai."` ([`BookingWizard.jsx:344`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L344))
- **Teks Petunjuk di Step 4 (Setelah Sukses Booking):**  
  `"Gunakan nomor WhatsApp {picPhone} dan 6 digit PIN yang Anda buat untuk login ke Portal Jamaah (upload bukti transfer & berkas)."` ([`BookingWizard.jsx:1614`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L1614))

---

### 2.4 Penjelasan Hak Akses Jamaah Lain
- **FAKTA KODE:** **TIDAK ADA PENJELASAN APAPUN.**
- Di seluruh alur Step 1, Step 2, Step 3, maupun Step 4, sama sekali **tidak ditemukan** catatan, teks disclaimer, ataupun tooltip yang menginformasikan kepada pemesan bahwa PIN yang dibuat hanya akan aktif untuk dirinya (PIC) dan anggota keluarga/rombongan lainnya belum memiliki akses akun portal.

---

### 2.5 Payload JSON yang Dikirim ke Backend
Struktur data yang dirakit pada fungsi `handleSubmitBooking` ([`BookingWizard.jsx:411-424`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx#L411-L424)):

```json
{
  "brand_id": 3,
  "schedule_id": 6,
  "captcha_token": "0.xxxx",
  "pic": {
    "nama_lengkap": "Budi Santoso",
    "no_hp": "08123456789",
    "email": "budi@example.com",
    "jenis_kelamin": "L",
    "room_type": "Quad",
    "portal_pin": "123456"
  },
  "anggota": [
    {
      "pax_type": "reguler",
      "nama_lengkap": "Siti Aminah",
      "no_hp": "08987654321",
      "jenis_kelamin": "P",
      "room_type": "Quad"
    },
    {
      "pax_type": "infant",
      "nama_lengkap": "Ahmad Kecil",
      "jenis_kelamin": "L",
      "room_type": null,
      "tanggal_lahir": "2025-01-15"
    }
  ]
}
```

---

## 3. Halaman Portal Jamaah (BAGIAN 2)

### 3.1 Daftar Lengkap Rute di Bawah `/portal`

| Rute URL | Berkas Komponen Utama | Deskripsi & Fungsi Layanan |
|---|---|---|
| **Pembungkus** | [`src/app/portal/layout.jsx`](file:///c:/laragon/www/azhan-microsite/src/app/portal/layout.jsx#L22)<br/>[`src/components/PortalMobileShell.jsx`](file:///c:/laragon/www/azhan-microsite/src/components/PortalMobileShell.jsx#L16) | Layout utama yang menyediakan `BrandProvider`, `PortalAuthProvider`, Sidebar navigasi desktop, dan Bottom Tab Bar mobile. |
| **`/portal/login`** | [`src/app/portal/login/page.jsx`](file:///c:/laragon/www/azhan-microsite/src/app/portal/login/page.jsx#L9) | Halaman login portal jamaah mandiri (input ID Jamaah & PIN). |
| **`/portal`** | [`src/app/portal/page.jsx`](file:///c:/laragon/www/azhan-microsite/src/app/portal/page.jsx#L28) | Beranda portal: ringkasan jadwal terdekat (`ProgressTimeline`), card sambutan nama jamaah, dan daftar kartu booking yang dimiliki. |
| **`/portal/booking/[id]`** | [`src/app/portal/booking/[id]/page.jsx`](file:///c:/laragon/www/azhan-microsite/src/app/portal/booking/[id]/page.jsx#L18) | Detail booking spesifik: status invoice, progress kesiapan keberangkatan, status perlengkapan, rincian biaya, dan histori pembayaran. |
| **`/portal/dokumen`** | [`src/app/portal/dokumen/page.jsx`](file:///c:/laragon/www/azhan-microsite/src/app/portal/dokumen/page.jsx#L17) | Halaman checklist kelengkapan dokumen perorangan (Paspor, KTP, KK, Buku Nikah, Pas Foto, Vaksin Meningitis), preview dokumen, dan tombol upload/ganti. |
| **`/portal/pembayaran`** | [`src/app/portal/pembayaran/page.jsx`](file:///c:/laragon/www/azhan-microsite/src/app/portal/pembayaran/page.jsx#L15) | Daftar tagihan aktif seluruh booking dan wizard 3-langkah konfirmasi transfer manual (pilih booking, pilih rekening travel, upload bukti transfer). |
| **`/portal/profil`** | [`src/app/portal/profil/page.jsx`](file:///c:/laragon/www/azhan-microsite/src/app/portal/profil/page.jsx#L10) | Informasi identitas pribadi jamaah (Nama, NIK, No. HP, Gender, TTL, Alamat) dan tombol *"Keluar dari Portal"*. |

---

### 3.2 Field pada Form Login Portal
Didefinisikan pada [`src/app/portal/login/page.jsx:104-132`](file:///c:/laragon/www/azhan-microsite/src/app/portal/login/page.jsx#L104-L132):

1. **`portal_pin` (PIN Portal):**  
   `<input type="password" id="portal_pin" placeholder="Masukkan 6 digit PIN" required />`
2. **`id_jamaah` (ID Jamaah):**  
   `<input type="text" id="id_jamaah" placeholder="mis. AS-2608000001" required />` (Dikonversi otomatis ke format `UPPERCASE`).
3. **`brand_id` (Brand Identifier):**  
   Diinjeksikan secara transparan dari context domain brand microsite (`useBrand().brandId`), tidak diketik manual oleh user.
4. **Teks Bantuan di Bawah Form:**  
   `"ID Jamaah bisa dilihat pada invoice atau ditanyakan ke admin travel Anda."` ([`src/app/portal/login/page.jsx:154`](file:///c:/laragon/www/azhan-microsite/src/app/portal/login/page.jsx#L154))

---

### 3.3 Penampilan Data Pax Lain di Halaman Detail Booking
- **FAKTA KODE:** **DATA PAX LAIN TIDAK DITAMPILKAN SAMA SEKALI.**
- Pada berkas [`src/app/portal/booking/[id]/page.jsx:70-72`](file:///c:/laragon/www/azhan-microsite/src/app/portal/booking/[id]/page.jsx#L70-L72), bagian *Informasi Booking* hanya menampilkan satu string nama jamaah (`{booking.nama_jamaah}`):
  ```jsx
  <CollapsibleSection title="Informasi Booking" description="Informasi paket, kamar, dan jamaah terdaftar." summary={`${tanggal(booking.berangkat_tanggal)} · Kamar ${booking.room_type}`}>
    <dl className="grid gap-3">
      <div className="rounded-xl bg-neutral-50 p-3">
        <dt className="text-xs text-neutral-400">Tanggal berangkat</dt>
        <dd className="mt-1 text-sm font-bold text-neutral-800">{tanggal(booking.berangkat_tanggal)}</dd>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div className="rounded-xl bg-neutral-50 p-3">
          <dt className="text-xs text-neutral-400">Tipe kamar</dt>
          <dd className="mt-1 text-sm font-bold text-neutral-800">Kamar {booking.room_type}</dd>
        </div>
        <div className="rounded-xl bg-neutral-50 p-3">
          <dt className="text-xs text-neutral-400">Jamaah</dt>
          <dd className="mt-1 truncate text-sm font-bold text-neutral-800">{booking.nama_jamaah}</dd>
        </div>
      </div>
    </dl>
  </CollapsibleSection>
  ```
- Array `booking.pax` (yang memuat anggota rombongan) **tidak di-loop ataupun dirender sama sekali** di halaman ini. Informasi seperti nama jamaah lain, tanggal lahir, nomor paspor, NIK, atau status kamar anggota lain sama sekali tidak terlihat.

---

### 3.4 Mekanisme Upload Dokumen
- **FAKTA KODE:** **HANYA UNTUK DIRI SENDIRI.**
- Pada berkas [`src/app/portal/dokumen/page.jsx:43-50`](file:///c:/laragon/www/azhan-microsite/src/app/portal/dokumen/page.jsx#L43-L50), fungsi upload memanggil `uploadMyDokumen(type, file)` dari [`src/lib/portalApi.js:114-153`](file:///c:/laragon/www/azhan-microsite/src/lib/portalApi.js#L114-L153):
  1. File diunggah ke `POST /api/portal/media/upload` dengan header `Authorization: Bearer <token>`.
  2. Data disinkronkan ke `POST /api/portal/dokumen` hanya dengan payload `{"jenis": jenis, "file_url": url}` tanpa ada parameter `jamaah_id` target.
- Tidak ada elemen dropdown/pemilih anggota keluarga di UI. Dokumen yang diunggah otomatis tersimpan murni atas nama `jamaah_id` pemilik token login saat itu.

---

### 3.5 State & Storage pada `PortalAuthContext`
Didefinisikan pada [`src/context/PortalAuthContext.jsx:13-90`](file:///c:/laragon/www/azhan-microsite/src/context/PortalAuthContext.jsx#L13-L90):

1. **State yang Disimpan:**
   - `accessToken` (string): Token JWT yang didapat saat login.
   - `jamaah` (object): Data profil lengkap jamaah hasil request ke `GET /api/portal/me` (`id`, `id_jamaah`, `nama_lengkap`, `no_hp`, `nik`, `jenis_kelamin`, `tempat_lahir`, `tanggal_lahir`, `alamat`, `brand_id`).
   - `isLoading` (boolean): Indikator proses inisialisasi pengecekan token saat aplikasi pertama kali dimuat.
2. **Key `localStorage` yang Digunakan:**
   - **`'portal_access_token'`**  
     - Menyimpan token saat login: `localStorage.setItem('portal_access_token', data.access_token)` ([`PortalAuthContext.jsx:73`](file:///c:/laragon/www/azhan-microsite/src/context/PortalAuthContext.jsx#L73)).
     - Membaca token saat boot: `localStorage.getItem('portal_access_token')` ([`PortalAuthContext.jsx:32`](file:///c:/laragon/www/azhan-microsite/src/context/PortalAuthContext.jsx#L32)).
     - Menghapus token saat logout: `localStorage.removeItem('portal_access_token')` ([`PortalAuthContext.jsx:21`](file:///c:/laragon/www/azhan-microsite/src/context/PortalAuthContext.jsx#L21)).

---

## 4. Daftar Layar & Komponen Terdampak (Multi-Akun Jamaah)

Berikut adalah daftar inventaris layar dan komponen frontend yang akan terdampak apabila anggota rombongan (pax non-PIC) diizinkan memiliki akun portal aktif:

1. **Formulir Pendaftaran & Checkout:**
   - [`src/components/booking/BookingWizard.jsx`](file:///c:/laragon/www/azhan-microsite/src/components/booking/BookingWizard.jsx):
     - *Step 2 (Form Data Jamaah):* Penyesuaian input nomor WhatsApp/kontak tiap anggota pax jika dibutuhkan sebagai sarana aktivasi akun.
     - *Step 3 (Konfirmasi & Buat PIN):* Penyesuaian instruksi PIN (apakah tetap hanya PIN pemesan dengan aktivasi terpisah bagi anggota, atau opsi PIN per pax).
     - *Step 4 (Hasil Booking & CTA Portal):* Teks petunjuk pembagian ID Jamaah kepada masing-masing anggota rombongan.
2. **Tampilan Invoice Digital:**
   - [`src/components/invoice/DigitalInvoiceView.jsx`](file:///c:/laragon/www/azhan-microsite/src/components/invoice/DigitalInvoiceView.jsx):
     - Tabel daftar rombongan jamaah pada nota invoice (penyajian `ID Jamaah` per individu pax agar anggota dapat mengetahui ID login masing-masing).
3. **Autentikasi Portal:**
   - [`src/app/portal/login/page.jsx`](file:///c:/laragon/www/azhan-microsite/src/app/portal/login/page.jsx):
     - Validasi login bagi akun anggota (termasuk penanganan jika akun anggota belum memiliki PIN / butuh aktivasi pertama kali).
4. **Beranda Portal Jamaah:**
   - [`src/app/portal/page.jsx`](file:///c:/laragon/www/azhan-microsite/src/app/portal/page.jsx):
     - Komponen `ProgressTimeline` dan daftar booking (penyesuaian status relasi apakah user bertindak sebagai Pemesan Utama / Anggota Peserta).
5. **Halaman Detail Booking & Tracking:**
   - [`src/app/portal/booking/[id]/page.jsx`](file:///c:/laragon/www/azhan-microsite/src/app/portal/booking/[id]/page.jsx):
     - Section *Informasi Booking:* Perluasan dari menampilkan 1 nama PIC menjadi daftar seluruh anggota rombongan (`booking.pax`).
     - Section *Progress Kesiapan:* Penyesuaian apakah menampilkan checklist progress individu yang sedang login atau progress seluruh pax.
     - Section *Perlengkapan:* Status penyerahan perlengkapan per pax.
6. **Kelola Dokumen Jamaah:**
   - [`src/app/portal/dokumen/page.jsx`](file:///c:/laragon/www/azhan-microsite/src/app/portal/dokumen/page.jsx):
     - Evaluasi model akses: apakah PIC memiliki tab switch untuk melihat status dokumen anggotanya, atau dokumen tetap murni terisolasi 100% per akun login.
7. **Riwayat Tagihan & Pembayaran:**
   - [`src/app/portal/pembayaran/page.jsx`](file:///c:/laragon/www/azhan-microsite/src/app/portal/pembayaran/page.jsx):
     - Penentuan hak akses: apakah anggota non-PIC diizinkan melihat tagihan & mengunggah bukti transfer rombongan, atau hanya bersifat *view-only* / disembunyikan.
