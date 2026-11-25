# Spek: Komisi per paket dan peringkat agen (2026-11-25)

Status: dibangun 2026-11-25 (ERP dev-malik, web dan app branch agen-fitur). Usulan bawaan peringkat disetujui bersama rencana. Keputusan pemilik 2026-11-25 (brainstorming setelah audit fitur agen POV agen).

## 1. Komisi per paket

### Masalah
Agen tidak tahu berapa komisinya saat memilih paket untuk dibagikan atau dibookingkan. ERP sudah menyimpan nilainya, tetapi tidak dikirim ke agen.

### Keputusan
- Angka yang tampil: **komisi langsung saja**, per jamaah. Bonus pembinaan (hak upline) tidak ditampilkan.
- Nilai efektif sama dengan snapshot saat booking dibuat: `COALESCE(schedules.nominal_komisi_langsung, NULLIF(brands.ujroh_closing, 0))` (migrasi 127).
- Paket berkomisi 0 atau kosong: label tidak tampil.
- Hanya untuk agen aktif. Tidak lewat endpoint publik `/api/public/v1` (kontrak web publik tetap).
- Lokasi:
  - Detail paket web dan app, dekat tombol Bagikan / Pesan.
  - Pemilih jadwal booking agen di web (`PilihJadwalAgen`, baris detail) dan form booking app (ringkas paket di atas form).
    Baris detail pemilih web (keputusan pemilik 2026-11-25): `Rp 45 jt/pax` di kiri, `Komisi Rp 750.000` merah; tanpa sisa seat.
  - Detail paket web memakai label yang sama di brand Hana (`PaketDetailHana`) dan brand lain (`PaketDetail`).
  - Tidak di kartu katalog (web maupun app).
- Copy: `Komisi Rp500.000/jamaah`. Komisi tidak dihitung untuk bayi dan booking diri agen (aturan K13, `internal/komisi/kalkulasi.go`); label tidak menjelaskan ini.

### Kontrak ERP
`GET /api/portal/agen/komisi-paket` (RequireAgenAktif) →
```json
[{ "schedule_id": 93, "komisi_langsung": 500000 }]
```
Hanya jadwal brand agen yang `published` dan belum berangkat, dengan nilai efektif > 0. Array kosong `[]` bila tidak ada.

## 2. Peringkat agen

### Masalah
Admin punya peringkat agen (`peringkatAgen`, kartu 10 agen terbaik); agen tidak tahu posisinya.

### Keputusan
- Tampil: **10 baris pertama + baris sendiri**. Bila baris sendiri di luar 10 baris pertama dan nomornya lebih besar dari nomor baris terakhir, diberi pemisah "···". Tanpa tetangga di atas/bawah agen.
- Ukuran: **pax closing** (sama dengan admin: jamaah lunas dari komisi langsung + repeat order, komisi aktif, paket berkomisi 0 tidak terhitung).
- Periode: pilihan **Bulan ini** dan **Tahun ini** (WIB, berdasarkan tanggal komisi tercatat). Bawaan Bulan ini.
- **Peringkat bersama** (keputusan 2026-11-25, grilling skala): pax sama = nomor sama, nomor berikutnya melompat (1, 2, 2, 4). Komisi lebih besar lalu nama hanya mengatur urutan tampil. Nilai komisi tidak ditampilkan.
- Lingkup: brand agen sendiri. "dari N agen" = agen aktif dengan minimal 1 pax di periode itu.
- Angka memakai titik ribuan ("#1.248 dari 2.140 agen").

### Pesan motivasi
- Baris atas blok posisi: "Butuh N pax lagi untuk naik ke #X". N = selisih ke kelompok tepat di atas; menyamai mereka = ikut memegang #X.
- Peringkat #1: label "Posisi Anda", tanpa pesan. Tanpa pax: "Posisi Anda" + "Butuh 1 pax untuk masuk peringkat".
- Blok posisi selalu dua baris (dilarang tiga baris bertumpuk). Layar sempit: ikon dan kotak pax disembunyikan (pax tampil di baris sendiri pada daftar).
- Kartu dashboard tanpa pesan motivasi.

### Podium
- Tiga baris pertama menurut urutan tampil, disusun kiri-tengah-kanan = baris ke-2, ke-1, ke-3. Tinggi alas per posisi tampil; cincin dan medali per nomor (seri #1 = dua emas).
- Avatar memakai foto profil agen (GET /api/portal/agen/peringkat/{akun_id}/foto, hanya agen aktif sebrand), inisial bila tanpa foto atau gagal dimuat.

### Aturan tambahan (disetujui bersama rencana)
- Agen nonaktif tidak masuk daftar agen dan tidak dihitung (pilihan yang tidak bisa dipilih disembunyikan). Admin tetap melihat badge Nonaktif.
- Nama tampil: nama lengkap dan kota domisili, tanpa kode referral.

### Kontrak ERP
`GET /api/portal/agen/peringkat?periode=bulan|tahun` (RequireAgenAktif) →
```json
{
  "periode": "bulan", "dari": "2026-11-01", "sampai": "2026-11-25",
  "jumlah_agen": 2140,
  "saya": { "peringkat": 1248, "pax_closing": 3, "target": { "peringkat": 1241, "kurang_pax": 2 } },
  "teratas": [{ "peringkat": 1, "akun_id": 7374, "nama": "Hannah", "kota": "Kota Cirebon", "ada_foto": true, "pax_closing": 41, "saya": false }]
}
```
`saya.peringkat` null bila 0 pax; `saya.target` null saat #1 atau tanpa pax. `teratas` = 10 baris pertama + baris sendiri bila di luar, `[]` bila kosong.

### Tampilan
- Web: kartu "Peringkat bulan ini" di dashboard `/akun` → halaman `/akun/agen/peringkat` (pilihan periode, blok posisi, podium, daftar).
- App: kartu yang sama di tab Akun → layar `akun/peringkat`.

## 3. Urutan kerja

1. **ERP (dev-malik):** dua endpoint di `internal/agen` (pakai ulang `rentangUjroh` dan `peringkatAgen`), tes integrasi (`ERP_TEST_DB=1`): komisi efektif brand vs override vs 0, jadwal lampau/draft tidak ikut; peringkat bulan vs tahun, agen nonaktif tidak ikut, posisi di luar 10 besar, 0 pax, lintas brand tidak bocor.
2. **Web (azhan-booking):** proxy GET JSON `komisi-paket` dan `peringkat` di `lib/agenProxy.js`; label di detail paket (diambil di browser karena halaman publik di-cache) dan `PilihJadwalAgen`; kartu dan halaman peringkat. Tes murni untuk pemetaan; screenshot 375/1440.
3. **App (azhan-mobile):** `komisiPaketAgen`, `peringkatAgen` di `@azhan/core`; label di Hero detail paket dan ringkas paket form booking; kartu dan layar peringkat. Typecheck + tes; cek Expo web.

Langkah 2 dan 3 bisa berjalan bersamaan setelah kontrak ERP disepakati.
