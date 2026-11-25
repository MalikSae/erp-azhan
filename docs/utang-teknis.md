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

### Grup menu Inventory (dicatat 2026-11-25, review alur Inventory)

Ditunda atas keputusan user; perilaku sudah benar, hanya batas modul:

- `internal/perlengkapan/set.go`: menulis `schedules.perlengkapan_custom`; membaca `schedules`, `package_categories`, `category_brands`.
- `internal/perlengkapan/pengajuan.go`, `mutasi.go`, `repository.go`: join `booking_pax`, `bookings`, `schedules`, `jamaah`, `customer_accounts`, `admin_users`, `brands`.
- `internal/booking/repository.go` (penyerahan): membaca set paket/kategori/global dan mengunci `perlengkapan_stok` langsung; hanya perubahan stok yang lewat `perlengkapan.ApplyBrandDelta`. Urutan set paket → kategori → global ditulis dua kali (booking dan `set.go`).
- `internal/customer/perlengkapan_pelanggan.go`: menulis `booking_pax` (ajukan, terima) dan membaca `brands`.

Perbaikan: `perlengkapan` menyediakan `IsiUntukPax(scheduleID, gender)` dan `KunciStok`; booking menyediakan fungsi ajukan/terima perlengkapan untuk customer.

## Kode berulang di backend (dicatat 2026-11-25)

Sudah diperbaiki (2026-11-25):

- `shared.WriteJSON`/`shared.WriteError` menggantikan salinan lokal di 26 paket.
- `shared.IsFKDipakai` dan `shared.IsMySQLErr` menggantikan blok tangkap 1451 di hotel, airline, airport, itinerary, category, schedule, jamaah, benefit, haramain, brand, adminuser.
- `internal/schedule` dipecah: `seat.go`, `status.go`, `tanggal.go`, `publik.go`, `master_cek.go`, `benefit_paket.go`.

Masih tersisa:

- `booking`, `customer`, `perlengkapan` masih punya `writeJSON`/`writeError` dan blok 1451 sendiri; ditunda karena sedang diubah sesi review Inventory. `customer.writeJSON` menambah `Cache-Control: no-store`, jadi butuh varian sendiri.
- Penerjemah galat per modul (`tulisGalat`, `handleRepoError`) tetap per modul karena isinya memetakan galat sentinel milik modul itu.
- Blok tangkap MySQL 1062 (duplikat) masih ditulis manual di beberapa modul; bisa memakai `shared.IsMySQLErr(err, 1062)`.

## Kode berulang di dashboard (dicatat 2026-11-25)

- Halaman master Hotel, Maskapai, Kategori, dan Itinerary berbentuk sama (muat, modal, hapus 409). Belum dijadikan hook atau komponen bersama.
- `className="!mb-0"` dipakai sekitar 40 kali: margin bawaan `Input`/`CustomDropdown` bertabrakan dengan pola `space-y`.
