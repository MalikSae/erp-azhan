# 0006. Dokumen, surat, dan notifikasi dokumen untuk peran jamaah

Tanggal: 18 November 2026
Status: diterima (tahap E4b). Lanjutan ADR 0005. Tanpa migrasi.

## Konteks

Tautan akun ke jamaah hanya dibuktikan kode booking dan NIK, tanpa faktor kedua. Cakupan tahap 1 menyebut "dokumen (lihat), surat, notifikasi dokumen", sedangkan keputusan keamanan menyatakan berkas KTP/paspor dan ubah data belum boleh untuk akun tertaut. Tafsiran aman: peran jamaah hanya melihat status, tidak pernah isi berkas.

## Keputusan per endpoint

| Endpoint | Peran jamaah | Alasan |
|---|---|---|
| `GET /account/perjalanan-jamaah/{id}/dokumen` (baru) | DIBUKA | Status per jenis dokumen jamaah tertaut saja. Tanpa `file_url`, tanpa alasan penolakan. |
| `GET /account/perjalanan-jamaah/{id}/surat` (baru) | DIBUKA (daftar) | Hanya surat yang sudah terbit: label, nomor, tanggal terbit. Tanpa unduhan. |
| `GET /account/bookings/{id}/dokumen` | DITOLAK (404, `owned`) | Memuat pax lain dan jalur pemilik. |
| `POST /account/bookings/{id}/dokumen` (KirimDokumen) | DITOLAK (404, `owned`) | Unggah/ubah berkas identitas belum boleh. |
| `GET /account/bookings/{id}/surat` (SuratDaftar) | DITOLAK (404) | Memuat pax lain dan DP booking. Diganti daftar baru di atas. |
| `GET /account/bookings/{id}/surat/{jenis}` (SuratUnduh) | DITOLAK (404) | PDF memuat tempat dan tanggal lahir serta kota jamaah (data identitas) dan menerbitkan nomor urut baru. Unduh DITUNDA sampai faktor kedua diputuskan. |
| Notifikasi dokumen (`notifikasi.Dokumen`) | DIBUKA | Akun tertaut menjadi penerima tambahan. |

Gerbang dua rute baru: akun tertaut hidup (`customer_accounts.jamaah_id`, dibaca tiap permintaan) dan booking memenuhi `sqlBookingPeran` (bukan draft, bukan batal, memuat pax aktif jamaah itu, bukan milik akun sendiri). Selain itu 404. `owned()`, `ownedJamaah`, `sqlAksesPenuh` tidak diubah.

## Notifikasi

`Dokumen()` menambah cabang UNION: akun yang `jamaah_id`-nya jamaah itu, belum dihapus, dan jamaah itu punya booking aktif (bukan draft/batal). UNION mencegah dobel bila akun tertaut sama dengan pembuat atau pemesan. Cabang itu memakai syarat booking yang sama dengan `sqlBookingPeran` (bukan draft, bukan batal, bukan baru yang tahannya lewat), supaya penerima tidak mendapat notifikasi untuk perjalanan yang tidak tampil di kartunya. Isi tetap "label dokumen · nama jamaah".

Tujuan ketuk: notifikasi memakai `ref_tipe=jamaah`, `ref_id=jamaah.id`. Bagi akun tertaut, M2/W2 membuka layar dokumen pada kartu perjalanan jamaah itu (cari lewat `GET /account/perjalanan-jamaah`), bukan `GET /account/jamaah/{id}`, yang memakai aturan pemilik dan 404 bagi peran jamaah.

## Kontrak JSON (M2/W2)

```json
GET /api/account/perjalanan-jamaah/{booking_id}/dokumen
{"jamaah": {"jamaah_id": 45, "nama": "Ahmad", "pax_type": "reguler",
  "dokumen": [{"jenis": "ktp", "label": "KTP", "wajib": true,
               "status": "belum_upload | submitted | approved | rejected",
               "rejection_reason": null}]}}

GET /api/account/perjalanan-jamaah/{booking_id}/surat
{"surat": [{"slug": "keterangan-umroh", "label": "Surat Keterangan Umroh",
            "nomor": "001/...", "terbit": "2027-02-01"}]}
```

`rejection_reason` selalu `null`. Booking tidak memenuhi gerbang: 404 `{"error": "Perjalanan tidak ditemukan."}`.

## Ditunda

- Unduh surat untuk peran jamaah, dan penerbitan surat baru.
- Berkas identitas, unggah, ubah data.
Semuanya menunggu faktor kedua (ADR 0005 butir 6).
