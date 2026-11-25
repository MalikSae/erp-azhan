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
- Tampil: **10 teratas + posisi sendiri**. Bila agen di luar 10 besar, barisnya ditambahkan di bawah daftar.
- Ukuran: **pax closing** (sama dengan admin: jamaah lunas dari komisi langsung + repeat order, komisi aktif, paket berkomisi 0 tidak terhitung).
- Periode: pilihan **Bulan ini** dan **Tahun ini** (WIB, berdasarkan tanggal komisi tercatat). Bawaan Bulan ini.
- Urutan seri: komisi lebih besar, lalu nama (sama dengan admin). Nilai komisi tidak ditampilkan.
- Lingkup: brand agen sendiri.

### Aturan tambahan (disetujui bersama rencana)
- Agen nonaktif tidak masuk daftar agen dan tidak dihitung dalam "dari N agen" (pilihan yang tidak bisa dipilih disembunyikan). Admin tetap melihat badge Nonaktif.
- Nama tampil: nama lengkap dan kota domisili, tanpa kode referral.
- Agen dengan 0 pax closing di periode itu: "Belum masuk peringkat", tanpa nomor.

### Kontrak ERP
`GET /api/portal/agen/peringkat?periode=bulan|tahun` (RequireAgenAktif) →
```json
{
  "periode": "bulan", "dari": "2026-11-01", "sampai": "2026-11-25",
  "jumlah_agen": 37,
  "saya": { "peringkat": 4, "pax_closing": 6 },
  "teratas": [{ "peringkat": 1, "nama": "Hannah", "kota": "Kota Cirebon", "pax_closing": 12, "saya": false }]
}
```
`saya.peringkat` null bila 0 pax. `teratas` maksimal 10, `[]` bila kosong.

### Tampilan
- Web: kartu "Peringkat" di dashboard `/akun` (posisi sendiri) → halaman `/akun/agen/peringkat` (pilihan periode, 10 teratas, baris sendiri disorot).
- App: kartu yang sama di tab Akun → layar `akun/peringkat`.

## 3. Urutan kerja

1. **ERP (dev-malik):** dua endpoint di `internal/agen` (pakai ulang `rentangUjroh` dan `peringkatAgen`), tes integrasi (`ERP_TEST_DB=1`): komisi efektif brand vs override vs 0, jadwal lampau/draft tidak ikut; peringkat bulan vs tahun, agen nonaktif tidak ikut, posisi di luar 10 besar, 0 pax, lintas brand tidak bocor.
2. **Web (azhan-booking):** proxy GET JSON `komisi-paket` dan `peringkat` di `lib/agenProxy.js`; label di detail paket (diambil di browser karena halaman publik di-cache) dan `PilihJadwalAgen`; kartu dan halaman peringkat. Tes murni untuk pemetaan; screenshot 375/1440.
3. **App (azhan-mobile):** `komisiPaketAgen`, `peringkatAgen` di `@azhan/core`; label di Hero detail paket dan ringkas paket form booking; kartu dan layar peringkat. Typecheck + tes; cek Expo web.

Langkah 2 dan 3 bisa berjalan bersamaan setelah kontrak ERP disepakati.
