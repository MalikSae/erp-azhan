# Utang Teknis

## Query lintas modul (dicatat 2026-11-19, ditunda)

Aturan modular monolith: modul tidak query tabel modul lain. Pelanggaran yang diketahui:

- `internal/agen/repository.go`: `akunJamaah` dan `nikAgenPengajuan` membaca `bookings` dan `jamaah`.
- `internal/agen/handler.go`: `mediaMilikAkun` membaca tabel media.
- `internal/customer/repository.go`: `Create` memanggil `agen.ResolveKodeReferral`, yang membaca `customer_accounts`. Customer dan agen saling bergantung.

Ditunda: bukan celah keamanan dan belum ada rilis production. Perbaikan: antarmuka kecil per modul (mis. `ResolveKodeReferral` dan pengecekan jamaah tertaut disuntikkan sebagai dependensi).

### Grup menu Paket (dicatat 2026-11-25, review alur Paket)

Sudah diperbaiki (2026-11-25):

- Absensi manasik menulis penanda lewat `booking.CatatProgressManasik`, bukan `UPDATE booking_pax` sendiri.
- Haramain menandai paket berubah lewat `schedule.TandaiDiubahLewatLokasi`, bukan `UPDATE schedules` sendiri.
- Hitungan booking aktif di schedule memakai `booking.HitungAktif`; manifest memakai `booking.SQLStatusAktif`.

Masih membaca tabel modul lain (hanya baca, ditunda):

- `internal/schedule/repository.go`: `transaksi_komisi`, `brands`, cek keberadaan hotel, maskapai, itinerary, bandara. Seat dipegang lewat `shared.SeatDipegangBooking`.
- `internal/manifest`: join `bookings`, `booking_pax`, `jamaah`, `airlines`.
- `internal/manasik`: join `bookings`, `jamaah`, `customer_accounts`, `brands`; aturan `bookingBerlaku` (booking baru dengan hold kedaluwarsa tidak dihitung) masih ditulis di manasik.
- `internal/category`, `internal/hotel`, `internal/airline`, `internal/airport`: menghitung `schedules` sebelum hapus.
- Modul booking sendiri masih menulis literal `'baru','dp','lunas'` di `repository.go`.

## Kode berulang di backend (dicatat 2026-11-25)

- `writeError`/`writeJSON` didefinisikan ulang di hampir tiap paket, dengan nama berbeda (`tulisGalat`, `galatTulis`, `handleRepoError`).
- Blok tangkap MySQL 1451 disalin di hotel, airline, airport, itinerary, category, schedule.
- `internal/schedule/handler.go` dan `repository.go` mencampur CRUD, seat, tanggal, benefit, lokasi, dan publik. Pecah per alasan berubah (mis. `seat.go`, `tanggal.go`).

Perbaikan: helper bersama di `internal/shared`.

## Kode berulang di dashboard (dicatat 2026-11-25)

- Halaman master Hotel, Maskapai, Kategori, dan Itinerary berbentuk sama (muat, modal, hapus 409). Belum dijadikan hook atau komponen bersama.
- `className="!mb-0"` dipakai sekitar 40 kali: margin bawaan `Input`/`CustomDropdown` bertabrakan dengan pola `space-y`.
