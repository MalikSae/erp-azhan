package shared

// DefaultListCap membatasi jumlah baris yang dikembalikan daftar admin yang belum punya paginasi di klien
// (jamaah, booking, pembayaran, jadwal). Ini katup pengaman terhadap query tanpa batas, bukan batas tampilan:
// jauh di atas ukuran data sekarang. Bila tercapai, daftar terpotong; saat itu paginasi di klien perlu dibuat.
const DefaultListCap = 10000
