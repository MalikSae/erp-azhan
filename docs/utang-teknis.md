# Utang Teknis

## Query lintas modul (dicatat 2026-11-19, ditunda)

Aturan modular monolith: modul tidak query tabel modul lain. Pelanggaran yang diketahui:

- `internal/agen/repository.go`: `akunJamaah` dan `nikAgenPengajuan` membaca `bookings` dan `jamaah`.
- `internal/agen/handler.go`: `mediaMilikAkun` membaca tabel media.
- `internal/customer/repository.go`: `Create` memanggil `agen.ResolveKodeReferral`, yang membaca `customer_accounts`. Customer dan agen saling bergantung.

Ditunda: bukan celah keamanan dan belum ada rilis production. Perbaikan: antarmuka kecil per modul (mis. `ResolveKodeReferral` dan pengecekan jamaah tertaut disuntikkan sebagai dependensi).

### Grup menu Paket (dicatat 2026-11-25, review alur Paket)

Menulis ke tabel modul lain (paling berisiko):

- `internal/manasik/manasik.go`: absensi mengubah `booking_pax.progress_manasik` langsung. Kolom yang sama juga diubah modul booking dari Detail Booking.
- `internal/haramain/haramain.go`: mengubah `schedules.updated_at` lewat `schedule_lokasi`.

Membaca tabel modul lain:

- `internal/schedule/repository.go`: `bookings` (cek booking aktif, seat dipegang), `transaksi_komisi`, `brands`, serta cek keberadaan hotel, maskapai, itinerary, bandara.
- `internal/manifest`: `bookings`, `booking_pax`, `jamaah`, `airlines`.
- `internal/manasik`: `bookings`, `jamaah`, `customer_accounts`, `brands`.
- `internal/category`, `internal/hotel`, `internal/airline`, `internal/airport`: menghitung `schedules` sebelum hapus.

Perbaikan: modul booking menyediakan fungsi untuk status booking aktif dan seat dipegang (sekarang literal `'baru','dp','lunas'` tersebar), dan absensi manasik memanggil fungsi milik booking.

## Kode berulang di backend (dicatat 2026-11-25)

- `writeError`/`writeJSON` didefinisikan ulang di hampir tiap paket, dengan nama berbeda (`tulisGalat`, `galatTulis`, `handleRepoError`).
- Blok tangkap MySQL 1451 disalin di hotel, airline, airport, itinerary, category, schedule.
- `internal/schedule/handler.go` dan `repository.go` mencampur CRUD, seat, tanggal, benefit, lokasi, dan publik. Pecah per alasan berubah (mis. `seat.go`, `tanggal.go`).

Perbaikan: helper bersama di `internal/shared`.

## Kode berulang di dashboard (dicatat 2026-11-25)

- Halaman master Hotel, Maskapai, Kategori, dan Itinerary berbentuk sama (muat, modal, hapus 409). Belum dijadikan hook atau komponen bersama.
- `className="!mb-0"` dipakai sekitar 40 kali: margin bawaan `Input`/`CustomDropdown` bertabrakan dengan pola `space-y`.
