// Tanggal hari ini menurut WIB (UTC+7) dalam format YYYY-MM-DD. Backend memvalidasi tanggal
// pembayaran, refund, dan tanggal lahir terhadap WIB; toISOString() saja memakai UTC, sehingga
// antara 00.00 dan 06.59 WIB hasilnya masih tanggal kemarin.
export const hariIniWIB = () => new Date(Date.now() + 7 * 60 * 60 * 1000).toISOString().slice(0, 10);
