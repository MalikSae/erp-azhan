# Utang Teknis

## Query lintas modul (dicatat 2026-11-19, ditunda)

Aturan modular monolith: modul tidak query tabel modul lain. Pelanggaran yang diketahui:

- `internal/agen/repository.go`: `akunJamaah` dan `nikAgenPengajuan` membaca `bookings` dan `jamaah`.
- `internal/agen/handler.go`: `mediaMilikAkun` membaca tabel media.
- `internal/customer/repository.go`: `Create` memanggil `agen.ResolveKodeReferral`, yang membaca `customer_accounts`. Customer dan agen saling bergantung.

Ditunda: bukan celah keamanan dan belum ada rilis production. Perbaikan: antarmuka kecil per modul (mis. `ResolveKodeReferral` dan pengecekan jamaah tertaut disuntikkan sebagai dependensi).
